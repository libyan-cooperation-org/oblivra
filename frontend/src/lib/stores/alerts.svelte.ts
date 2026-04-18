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

  constructor() {
    this.init();
  }

  async init() {
    this.isLoading = true;
    try {
      if (IS_BROWSER) {
        // Browser / server mode — fetch from REST API
        const res = await fetch('/api/v1/alerts', {
          headers: { 'Authorization': 'Bearer oblivra-dev-key' }
        });
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

  dismiss(id: string) {
    this.alerts = this.alerts.map(a =>
      a.id === id ? { ...a, status: 'dismissed' } : a
    );
  }

  acknowledge(id: string) {
    this.alerts = this.alerts.map(a =>
      a.id === id ? { ...a, status: 'acknowledged' } : a
    );
  }
}

export const alertStore = new AlertStore();
