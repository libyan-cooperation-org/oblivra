import { IS_BROWSER } from '@lib/context';
import { diagnosticsStore } from './diagnostics.svelte';

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
  TotalEvents?: number;
}

export class DashboardStore {
  health = $state<HealthStats>({ AlertCount: 0, Status: 'Active' });
  siemStats = $state<SIEMStats>({ StorageUsage: '0', EPS: '0' });
  loading = $state(false);
  lastRefreshed = $state<Date | null>(null);

  constructor() {
    if (!IS_BROWSER) {
      // Delay first poll by 2s so Wails services finish Start() before we query
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

      const [h] = await Promise.all([
        GetAllHealth().catch(() => null),
      ]);

      const diag = diagnosticsStore.snapshot;

      this.health = (h as HealthStats) || { AlertCount: 0, Status: 'Active' };

      // Merge EPS from diagnostics snapshot into siemStats
      const eps = diag ? String(diag.ingest.current_eps ?? '0') : '0';
      this.siemStats = {
        StorageUsage: '0', // Storage usage tracking moved to Diagnostics in next phase
        EPS: eps,
        TotalEvents: diag?.ingest.dropped_total ?? 0, // Mocking TotalEvents as dropped for now or omit
      };

      this.lastRefreshed = new Date();
    } catch (err: any) {
      console.error('[DashboardStore] Refresh failed:', err);
      // Keep last known good status — don't flash raw error text to the user
      if (this.health.Status === 'Active' || this.health.Status === 'Operational') {
        this.health = { ...this.health, Status: 'Degraded' };
      }
    } finally {
      this.loading = false;
    }
  }
}

export const dashboardStore = new DashboardStore();
