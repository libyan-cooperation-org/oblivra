// notificationStore — persistent history of toasts + system events.
//
// The existing toastStore is ephemeral (toasts auto-dismiss after a few
// seconds). This store keeps the operator's full notification log so they
// can scroll back through the morning's alerts, dismissed warnings, and
// system status changes — the same way a desktop OS' Action Center works.
//
// Backed by localStorage so it survives a tab/window restart. Capped at
// MAX_ENTRIES so an alert flood doesn't grow the store unboundedly.

export type NotificationLevel = 'info' | 'warning' | 'error' | 'critical' | 'success';

export interface NotificationEntry {
  id: string;
  level: NotificationLevel;
  title: string;
  message?: string;
  /** ISO 8601 timestamp at notification creation. */
  ts: string;
  /** True once the operator has clicked through / dismissed. */
  read: boolean;
  /** Optional action that should be taken when the user clicks the entry. */
  action?: { label: string; route?: string; href?: string };
}

const STORAGE_KEY = 'oblivra:notifications:v1';
const MAX_ENTRIES = 200;
// Debounce window for localStorage persistence — coalesces an alert flood
// (e.g. a SIEM panic burst pushing 50 notifications/sec) into one disk
// write per quarter-second instead of N writes that thrash localStorage.
const PERSIST_DEBOUNCE_MS = 250;

class NotificationStore {
  // Reactive state visible to Svelte components via $state runes.
  entries: NotificationEntry[] = $state([]);
  drawerOpen = $state(false);
  // Cache the unread count instead of re-filtering on every getter access —
  // SOC dashboards re-evaluate this dozens of times per render. The cached
  // value is invalidated by every mutation method below.
  private unreadCountCache = $state(0);
  private criticalUnreadCache = $state(0);
  private persistTimer: ReturnType<typeof setTimeout> | null = null;

  get unreadCount(): number { return this.unreadCountCache; }
  get criticalUnread(): number { return this.criticalUnreadCache; }

  constructor() {
    this.load();
    this.recomputeCounts();
  }

  /** Load from localStorage. Silently no-ops on parse error / SSR. */
  private load() {
    if (typeof localStorage === 'undefined') return;
    try {
      const raw = localStorage.getItem(STORAGE_KEY);
      if (!raw) return;
      const parsed = JSON.parse(raw);
      if (Array.isArray(parsed)) {
        this.entries = parsed.slice(0, MAX_ENTRIES);
      }
    } catch {
      /* corrupt storage — start fresh */
    }
  }

  private recomputeCounts() {
    let unread = 0;
    let critical = 0;
    for (const e of this.entries) {
      if (e.read) continue;
      unread++;
      if (e.level === 'critical' || e.level === 'error') critical++;
    }
    this.unreadCountCache = unread;
    this.criticalUnreadCache = critical;
  }

  /**
   * Debounced persist — coalesces multiple mutations within
   * PERSIST_DEBOUNCE_MS into a single localStorage write. Page-unload is
   * handled separately via persistImmediately() so we never lose the most
   * recent batch.
   */
  private persist() {
    if (typeof localStorage === 'undefined') return;
    if (this.persistTimer) return; // a flush is already pending
    this.persistTimer = setTimeout(() => {
      this.persistTimer = null;
      this.persistImmediately();
    }, PERSIST_DEBOUNCE_MS);
  }

  private persistImmediately() {
    if (typeof localStorage === 'undefined') return;
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(this.entries));
    } catch (err) {
      // Quota exceeded — drop oldest 50% and retry. Surface the truncation
      // to the operator (via console; the bell drawer would itself be the
      // place to surface this but recursing into push() during a quota
      // crisis is the wrong move).
      console.warn(
        '[notifications] localStorage quota exceeded; truncating history by 50%',
        err,
      );
      try {
        this.entries = this.entries.slice(0, Math.floor(MAX_ENTRIES / 2));
        localStorage.setItem(STORAGE_KEY, JSON.stringify(this.entries));
      } catch (err2) {
        console.error('[notifications] persist still failing after truncate:', err2);
      }
    }
  }

  /** Force-flush any pending debounce. Useful before unload / navigation. */
  flush() {
    if (this.persistTimer) {
      clearTimeout(this.persistTimer);
      this.persistTimer = null;
    }
    this.persistImmediately();
  }

  push(level: NotificationLevel, title: string, message?: string, action?: NotificationEntry['action']) {
    const entry: NotificationEntry = {
      id: crypto.randomUUID(),
      level,
      title,
      message,
      ts: new Date().toISOString(),
      read: false,
      action,
    };
    this.entries = [entry, ...this.entries].slice(0, MAX_ENTRIES);
    this.unreadCountCache++;
    if (entry.level === 'critical' || entry.level === 'error') this.criticalUnreadCache++;
    this.persist();
  }

  markRead(id: string) {
    let mutated = false;
    const next = this.entries.map((e) => {
      if (e.id === id && !e.read) {
        mutated = true;
        return { ...e, read: true };
      }
      return e;
    });
    if (!mutated) return;
    this.entries = next;
    this.recomputeCounts();
    this.persist();
  }

  markAllRead() {
    // Fast path: bail out when nothing is unread (audit MED-7).
    if (this.unreadCountCache === 0) return;
    this.entries = this.entries.map((e) => (e.read ? e : { ...e, read: true }));
    this.unreadCountCache = 0;
    this.criticalUnreadCache = 0;
    this.persist();
  }

  remove(id: string) {
    const next = this.entries.filter((e) => e.id !== id);
    if (next.length === this.entries.length) return;
    this.entries = next;
    this.recomputeCounts();
    this.persist();
  }

  clearAll() {
    if (this.entries.length === 0) return;
    this.entries = [];
    this.unreadCountCache = 0;
    this.criticalUnreadCache = 0;
    this.persist();
  }

  toggleDrawer() {
    this.drawerOpen = !this.drawerOpen;
  }
}

export const notificationStore = new NotificationStore();

// Audit fix High-5: wire 4 backend events that were previously
// published but had no frontend listener. Without these subscriptions
// the operator never sees real-time confirmation of:
//   - setup completion
//   - playbook execution result
//   - host network isolation (ransomware shield response)
//   - tenant deletion (GDPR audit)
//
// Imported lazily inside an IIFE so this module stays free of circular
// imports and tree-shakeable when the bridge isn't loaded.
(async () => {
  if (typeof window === 'undefined') return;
  try {
    const { subscribe } = await import('@lib/bridge');

    subscribe('setup:initialized', () => {
      notificationStore.add({
        level: 'success',
        title: 'Setup complete',
        message: 'Bootstrap admin created and platform initialised.',
      });
    });

    subscribe('playbook:executed', (data: { name?: string; status?: string; incident_id?: string }) => {
      notificationStore.add({
        level: data.status === 'failed' ? 'error' : 'success',
        title: `Playbook ${data.status ?? 'executed'}`,
        message: `${data.name ?? 'unnamed'} on incident ${data.incident_id ?? '—'}`,
        action: data.incident_id ? { label: 'Open incident', route: `/cases?incident=${data.incident_id}` } : undefined,
      });
    });

    subscribe('ransomware:host_isolated', (data: { host?: string; reason?: string }) => {
      notificationStore.add({
        level: 'critical',
        title: 'Host isolated by ransomware shield',
        message: `${data.host ?? 'unknown host'} — ${data.reason ?? 'policy match'}`,
        action: data.host ? { label: 'Open host', route: `/host/${encodeURIComponent(data.host)}` } : undefined,
      });
    });

    subscribe('tenant:deleted', (data: { tenant_id?: string; deleted_by?: string }) => {
      notificationStore.add({
        level: 'warning',
        title: 'Tenant wiped',
        message: `${data.tenant_id ?? 'tenant'} cryptographically wiped by ${data.deleted_by ?? 'system'}.`,
      });
    });
  } catch {
    // Bridge not available (e.g. test environment) — silently skip.
  }
})();

// Force-flush pending debounced writes when the user is leaving the page,
// so a bug or crash doesn't lose the most recent ~250ms of notifications.
if (typeof window !== 'undefined') {
  window.addEventListener('beforeunload', () => notificationStore.flush());
}
