import { subscribe } from '@lib/bridge';
import { GetAlertHistory } from '@wailsjs/github.com/kingknull/oblivrashell/internal/services/alertingservice';

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
      const history = await GetAlertHistory();
      if (history) {
        this.alerts = history.map((a: any) => ({
          id: a.id || a.trigger_id,
          timestamp: a.timestamp || new Date().toISOString(),
          title: a.name || 'Security Alert',
          severity: a.severity || 'medium',
          host: a.host || 'unknown',
          status: a.status || 'open',
          description: a.log_line || ''
        }));
      }
    } catch (e) {
      console.error('Failed to load alert history:', e);
    } finally {
      this.isLoading = false;
    }

    // Subscribe to live alerts
    subscribe('security.alert', (data: any) => {
      const newAlert: Alert = {
        id: `live-${Date.now()}`,
        timestamp: new Date().toISOString(),
        title: data.message || 'Heuristic Alert',
        severity: data.severity || 'high',
        host: data.host_id || 'remote',
        status: 'open',
        description: data.message
      };
      this.alerts = [newAlert, ...this.alerts];
    });

    subscribe('detection.match', (match: any) => {
      const newAlert: Alert = {
        id: match.RuleID,
        timestamp: new Date().toISOString(),
        title: match.RuleName || 'Detection Match',
        severity: match.Severity || 'high',
        host: match.Context?.host || 'remote',
        status: 'open',
        description: match.Description
      };
      this.alerts = [newAlert, ...this.alerts];
    });
  }

  async refresh() {
    await this.init();
  }
}

export const alertStore = new AlertStore();
