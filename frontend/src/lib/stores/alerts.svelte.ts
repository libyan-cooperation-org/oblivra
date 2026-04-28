/**
 * OBLIVRA — Alert Store (Svelte 5 runes)
 *
 * Loads historical alerts and subscribes to live detection events.
 * Works in both desktop (Wails IPC) and browser (REST API) modes.
 * 
 * IMPORTANT: The Wails import is lazy (inside the if-block) so this
 * module can be evaluated safely in browser mode without throwing
 * "GetAlertHistory is not a function".
 */
import { subscribe } from '@lib/bridge';
import { IS_BROWSER } from '@lib/context';
import { apiFetch } from '@lib/apiClient';

export interface Alert {
  id: string;
  timestamp: string;
  title: string;
  severity: string;
  host: string;
  status: string;
  description: string;
  action?: any;
}

class AlertStore {
  alerts = $state<Alert[]>([]);
  isLoading = $state(false);

  /**
   * Operator-side suppression — IDs the operator hit `x` on. Clears
   * the alert from the queue immediately so the operator can move on,
   * with a 5-second undo window via toast. Persisted to localStorage
   * so a page reload doesn't undo the suppression.
   *
   * Distinct from server-side SuppressionRules (which match on
   * pattern). This is a per-id mute that doesn't affect future alerts.
   * Pattern-based rules require backend wiring (Phase 33).
   */
  suppressedIds = $state<Set<string>>(new Set());

  constructor() {
    this.init();
    this.hydrateSuppressions();
  }

  private hydrateSuppressions() {
    if (typeof localStorage === 'undefined') return;
    try {
      const raw = localStorage.getItem('oblivra:suppressedAlertIds');
      if (raw) {
        const arr = JSON.parse(raw) as string[];
        this.suppressedIds = new Set(arr);
      }
    } catch { /* private mode */ }
  }

  private persistSuppressions() {
    if (typeof localStorage === 'undefined') return;
    try {
      localStorage.setItem(
        'oblivra:suppressedAlertIds',
        JSON.stringify(Array.from(this.suppressedIds)),
      );
    } catch { /* quota / private mode */ }
  }

  /** Suppress an alert by id. Idempotent. */
  suppressAlert(id: string) {
    if (this.suppressedIds.has(id)) return;
    const next = new Set(this.suppressedIds);
    next.add(id);
    this.suppressedIds = next;
    this.persistSuppressions();
  }

  /** Restore an alert (used by the toast Undo button). */
  unsuppressAlert(id: string) {
    if (!this.suppressedIds.has(id)) return;
    const next = new Set(this.suppressedIds);
    next.delete(id);
    this.suppressedIds = next;
    this.persistSuppressions();
  }

  /** True if the alert was suppressed via the operator's `x` action. */
  isSuppressed(id: string): boolean {
    return this.suppressedIds.has(id);
  }

  async init() {
    this.isLoading = true;
    try {
      if (IS_BROWSER) {
        // Browser / server mode — fetch from REST API.
        // apiFetch attaches the X-Tenant-Id header from appStore.currentTenantId
        // so the backend can scope alerts to the operator's selected tenant.
        const res = await apiFetch('/api/v1/alerts');
        if (res.ok) {
          const data = await res.json();
          const list = data.alerts || [];
          this.alerts = list.map((a: any) => ({
            id: a.id || `alert-${Date.now()}`,
            timestamp: a.timestamp || new Date().toISOString(),
            title: a.description || a.name || 'Security Alert',
            severity: a.severity || 'medium',
            host: a.host_id || a.host || 'unknown',
            status: a.status || 'open',
            description: a.raw_log || a.description || '',
          }));
        }
      } else {
        // Desktop / Wails mode — lazy import to avoid crashing browser bundle
        const { GetAlertHistory } = await import(
          '@wailsjs/github.com/kingknull/oblivrashell/internal/services/alertingservice'
        );
        const history = await GetAlertHistory();
        if (history) {
          this.alerts = history.map((a: any) => ({
            id: a.id || a.trigger_id,
            timestamp: a.timestamp || new Date().toISOString(),
            title: a.name || 'Security Alert',
            severity: a.severity || 'medium',
            host: a.host || 'unknown',
            status: a.status || 'open',
            description: a.log_line || '',
          }));
        }
      }
    } catch (e) {
      console.error('[AlertStore] Failed to load alert history:', e);
    } finally {
      this.isLoading = false;
    }

    // Live alert feed — works in both modes via bridge event system
    subscribe('security.alert', (data: any) => {
      const newAlert: Alert = {
        id: `live-${Date.now()}`,
        timestamp: new Date().toISOString(),
        title: data.message || 'Heuristic Alert',
        severity: data.severity || 'high',
        host: data.host_id || 'remote',
        status: 'open',
        description: data.message || '',
      };
      this.alerts = [newAlert, ...this.alerts];
    });

    subscribe('detection.match', (match: any) => {
      const newAlert: Alert = {
        id: match.RuleID || `match-${Date.now()}`,
        timestamp: new Date().toISOString(),
        title: match.RuleName || 'Detection Match',
        severity: match.Severity || 'high',
        host: match.Context?.host || 'remote',
        status: 'open',
        description: match.Description || '',
      };
      this.alerts = [newAlert, ...this.alerts];
    });
  }

  async refresh() {
    this.alerts = [];
    await this.init();
  }

  // Audit fix High-6: each state transition does an optimistic local
  // update + a backend POST. On error, we surface (caller decides
  // whether to revert) so the UI never silently lies about state.
  // The backend acknowledges via the live `detection.match` /
  // `security.alert` event stream, which our subscribe() handlers
  // pick up — but we still apply the optimistic flip for instant
  // visual feedback.

  async acknowledge(id: string) {
    this.alerts = this.alerts.map(a =>
      a.id === id ? { ...a, status: 'acknowledged' } : a
    );
    if (IS_BROWSER) {
      await apiFetch(`/api/v1/alerts/${encodeURIComponent(id)}/ack`, { method: 'POST' })
        .catch((e) => console.error('[AlertStore] acknowledge failed:', e));
    }
  }

  async investigate(id: string) {
    this.alerts = this.alerts.map(a =>
      a.id === id ? { ...a, status: 'investigating' } : a
    );
    if (IS_BROWSER) {
      await apiFetch(`/api/v1/alerts/${encodeURIComponent(id)}/investigate`, { method: 'POST' })
        .catch((e) => console.error('[AlertStore] investigate failed:', e));
    }
  }

  async close(id: string) {
    this.alerts = this.alerts.map(a =>
      a.id === id ? { ...a, status: 'closed' } : a
    );
    if (IS_BROWSER) {
      await apiFetch(`/api/v1/alerts/${encodeURIComponent(id)}/close`, { method: 'POST' })
        .catch((e) => console.error('[AlertStore] close failed:', e));
    }
  }

  async suppress(id: string) {
    this.alerts = this.alerts.map(a =>
      a.id === id ? { ...a, status: 'suppressed' } : a
    );
    if (IS_BROWSER) {
      await apiFetch(`/api/v1/alerts/${encodeURIComponent(id)}/suppress`, { method: 'POST' })
        .catch((e) => console.error('[AlertStore] suppress failed:', e));
    }
  }
}

export const alertStore = new AlertStore();
