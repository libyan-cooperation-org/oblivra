/**
 * OBLIVRA — UEBA Store (Svelte 5 runes)
 *
 * Orchestrates behavioral analytics and anomaly feed from the backend.
 */
import { IS_BROWSER } from '@lib/context';

export interface EntityProfile {
  id: string;
  type: string;
  risk_score: number;
  peer_group_id: string;
  last_active: string;
}

export interface Anomaly {
  entity_id: string;
  entity_type: string;
  risk_score: number;
  peer_group_id: string;
  timestamp: string;
  evidence: Array<{ key: string; value: number; description: string }>;
}

export class UEBAStore {
  profiles = $state<EntityProfile[]>([]);
  anomalies = $state<Anomaly[]>([]);
  stats = $state({
    total_entities: 0,
    high_risk_entities: 0,
    anomalies_24h: 0,
    baselines_active: 0
  });
  loading = $state(false);

  async refresh() {
    this.loading = true;
    try {
        if (IS_BROWSER) {
            const [profilesRes, anomaliesRes, statsRes] = await Promise.all([
                fetch('/api/v1/ueba/profiles', { credentials: 'include' }),
                fetch('/api/v1/ueba/anomalies', { credentials: 'include' }),
                fetch('/api/v1/ueba/stats', { credentials: 'include' })
            ]);

            if (profilesRes.ok) {
                this.profiles = await profilesRes.json();
            }
            if (anomaliesRes.ok) {
                this.anomalies = await anomaliesRes.json();
            }
            if (statsRes.ok) {
                this.stats = await statsRes.json();
            }
        }
    } catch (e) {
        console.error('[UEBAStore] Refresh failed:', e);
    } finally {
        this.loading = false;
    }
  }
}

export const uebaStore = new UEBAStore();
