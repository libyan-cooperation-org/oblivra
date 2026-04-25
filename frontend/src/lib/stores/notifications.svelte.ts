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

class NotificationStore {
  // Reactive state visible to Svelte components via $state runes.
  entries: NotificationEntry[] = $state([]);
  drawerOpen = $state(false);

  // Derived getters
  get unreadCount(): number {
    return this.entries.filter((e) => !e.read).length;
  }

  // Per-level counts power the bell-icon badge colour.
  get criticalUnread(): number {
    return this.entries.filter((e) => !e.read && (e.level === 'critical' || e.level === 'error')).length;
  }

  constructor() {
    this.load();
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

  /** Persist on every mutation. Silent on quota / SSR errors. */
  private persist() {
    if (typeof localStorage === 'undefined') return;
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(this.entries));
    } catch {
      /* quota exceeded — drop oldest and retry once */
      try {
        this.entries = this.entries.slice(0, Math.floor(MAX_ENTRIES / 2));
        localStorage.setItem(STORAGE_KEY, JSON.stringify(this.entries));
      } catch { /* give up */ }
    }
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
    this.persist();
  }

  markRead(id: string) {
    const next = this.entries.map((e) => (e.id === id ? { ...e, read: true } : e));
    if (next.some((e, i) => e !== this.entries[i])) {
      this.entries = next;
      this.persist();
    }
  }

  markAllRead() {
    if (this.entries.every((e) => e.read)) return;
    this.entries = this.entries.map((e) => ({ ...e, read: true }));
    this.persist();
  }

  remove(id: string) {
    const next = this.entries.filter((e) => e.id !== id);
    if (next.length !== this.entries.length) {
      this.entries = next;
      this.persist();
    }
  }

  clearAll() {
    if (this.entries.length === 0) return;
    this.entries = [];
    this.persist();
  }

  toggleDrawer() {
    this.drawerOpen = !this.drawerOpen;
  }
}

export const notificationStore = new NotificationStore();
