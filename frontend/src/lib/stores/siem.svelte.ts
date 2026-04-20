/**
 * OBLIVRA — SIEM Store (Svelte 5 runes)
 *
 * Handles OQL execution and search result persistence.
 */
import { IS_BROWSER } from '@lib/context';

export interface SIEMEvent {
  timestamp: string;
  host: string;
  source: string;
  event_id: string;
  message: string;
  severity: string;
  raw?: any;
}

export class SIEMStore {
  results = $state<SIEMEvent[]>([]);
  stats = $state<any>(null);
  loading = $state(false);
  error = $state<string | null>(null);

  async executeQuery(oql: string) {
    this.loading = true;
    this.error = null;
    try {
      if (IS_BROWSER) {
        const res = await fetch(`/api/v1/events?q=${encodeURIComponent(oql)}`, { credentials: 'include' });
        if (res.ok) {
            const data = await res.json();
            this.results = (data.events || []).map((e: any) => ({
                timestamp: e.Timestamp || e.timestamp || new Date().toISOString(),
                host: e.HostID || e.host || 'unknown',
                source: e.EventType || e.source || 'parsed',
                event_id: e.ID || e.event_id || '0',
                message: e.RawLog || e.message || '',
                severity: e.Severity || e.severity || 'info',
                raw: e
            }));
        } else {
            this.error = await res.text() || 'Failed to fetch events';
        }
      } else {
        const { ExecuteOQL } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/siemservice');
        const res = await ExecuteOQL(oql);
        if (res) {
          this.results = (res.Rows || []).map((e: any) => ({
            timestamp: e.timestamp || new Date().toISOString(),
            host: e.host || 'unknown',
            source: e.source || 'parsed',
            event_id: e.event_id || '0',
            message: e.message || e.raw_log || '',
            severity: e.severity || 'info',
            raw: e
          }));
        }
      }
    } catch (err: any) {
      this.error = err.message || 'OQL Execution failed';
      console.error('[SIEMStore] Error:', err);
    } finally {
      this.loading = false;
    }
  }

  async refreshStats() {
    try {
      if (IS_BROWSER) {
        const res = await fetch('/api/v1/platform/metrics', { credentials: 'include' });
        if (res.ok) {
          const metrics = await res.json();
          // Map backend metrics to store format
          this.stats = {
            TotalEvents: metrics.total_events || 0,
            EventsPerSecond: metrics.eps || 0,
            ActiveAgents: metrics.active_agents || 0,
            StorageUsed: metrics.storage_bytes || 0,
            ThreatLevel: metrics.threat_level || 'LOW'
          };
        }
      } else {
        const { GetPlatformStats } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/siemservice');
        this.stats = await GetPlatformStats();
      }
    } catch (e) {
      console.error('[SIEMStore] Stats refresh failed', e);
    }
  }
}

export const siemStore = new SIEMStore();
