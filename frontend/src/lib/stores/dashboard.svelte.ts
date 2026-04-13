import { IS_BROWSER } from '@lib/context';

interface HealthStats {
  AlertCount: number;
  Status: string;
  hosts?: any;
  services?: Record<string, string>;
  compute?: { usage: string; status: string };
  network?: { latency: string; status: string };
  storage?: { throughput: string; status: string };
}

interface SIEMStats {
  StorageUsage: string;
  EPS: string;
  total_failed_logins?: number;
}

export class DashboardStore {
  health = $state<HealthStats>({ AlertCount: 0, Status: 'Active' });
  siemStats = $state<SIEMStats>({ StorageUsage: '0', EPS: '0' });
  loading = $state(false);
  lastRefreshed = $state<Date | null>(null);

  constructor() {
    if (!IS_BROWSER) {
      // Delay first poll by 2s to let Wails services finish Start() before we query
      setTimeout(() => {
        this.refresh();
        setInterval(() => this.refresh(), 5000);
      }, 2000);
    }
  }

  async refresh() {
    if (IS_BROWSER) return;
    this.loading = true;
    try {
      const { GetAllHealth } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/healthservice');
      const { GetGlobalThreatStats } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/siemservice');

      const [h, s] = await Promise.all([
        GetAllHealth(),
        GetGlobalThreatStats()
      ]);

      this.health = (h as HealthStats) || { AlertCount: 0, Status: 'Unknown' };
      this.siemStats = (s as SIEMStats) || { StorageUsage: '0', EPS: '0' };
      this.lastRefreshed = new Date();
    } catch (err: any) {
      console.error('[DashboardStore] Refresh failed:', err);
      // Don't overwrite Status with raw error text — keep last known good value
      // or show a compact degraded indicator
      if (this.health.Status === 'Active' || this.health.Status === 'Operational') {
        this.health.Status = 'Degraded';
      }
    } finally {
      this.loading = false;
    }
  }
}

export const dashboardStore = new DashboardStore();
