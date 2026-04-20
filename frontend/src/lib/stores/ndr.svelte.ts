/**
 * OBLIVRA — NDR Store (Svelte 5 runes)
 *
 * Orchestrates Network Detection and Response telemetry and flows.
 */
import { IS_BROWSER } from '@lib/context';

export interface NetworkFlow {
  id: string;
  timestamp: string;
  source_ip: string;
  destination_ip?: string;
  protocol?: string;
  event_type: string;
  user: string;
}

export interface NDRAlert {
  id: string;
  timestamp: string;
  event_type: string;
  source_ip: string;
  message?: string;
}

export class NDRStore {
  flows = $state<NetworkFlow[]>([]);
  alerts = $state<NDRAlert[]>([]);
  protocols = $state<Record<string, number>>({});
  loading = $state(false);

  async refresh() {
    this.loading = true;
    try {
        if (IS_BROWSER) {
            const [flowsRes, alertsRes, protoRes] = await Promise.all([
                fetch('/api/v1/ndr/flows?limit=50', { credentials: 'include' }),
                fetch('/api/v1/ndr/alerts?limit=50', { credentials: 'include' }),
                fetch('/api/v1/ndr/protocols', { credentials: 'include' })
            ]);

            if (flowsRes.ok) this.flows = await flowsRes.json();
            if (alertsRes.ok) this.alerts = await alertsRes.json();
            if (protoRes.ok) this.protocols = await protoRes.json();
        }
    } catch (e) {
        console.error('[NDRStore] Refresh failed:', e);
    } finally {
        this.loading = false;
    }
  }
}

export const ndrStore = new NDRStore();
