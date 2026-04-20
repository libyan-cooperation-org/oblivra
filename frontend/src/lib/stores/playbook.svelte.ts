/**
 * OBLIVRA — Playbook Store (Svelte 5 runes)
 *
 * Orchestrates automated response workflows and execution metrics.
 */
import { IS_BROWSER } from '@lib/context';

export interface Playbook {
  id: string;
  name: string;
  steps: any;
  created_at: string;
  last_run?: string;
}

export interface PlaybookMetrics {
  total_executions: number;
  success_count: number;
  failure_count: number;
  avg_duration_ms: number;
  executions_by_playbook: Record<string, number>;
  recent_executions: any[];
}

export class PlaybookStore {
  playbooks = $state<Playbook[]>([]);
  metrics = $state<PlaybookMetrics>({
    total_executions: 0,
    success_count: 0,
    failure_count: 0,
    avg_duration_ms: 0,
    executions_by_playbook: {},
    recent_executions: []
  });
  actions = $state<string[]>([]);
  loading = $state(false);

  async refresh() {
    this.loading = true;
    try {
        if (IS_BROWSER) {
            const [pbRes, metricsRes, actionsRes] = await Promise.all([
                fetch('/api/v1/playbooks', { credentials: 'include' }),
                fetch('/api/v1/playbooks/metrics', { credentials: 'include' }),
                fetch('/api/v1/playbooks/actions', { credentials: 'include' })
            ]);

            if (pbRes.ok) {
                const data = await pbRes.json();
                this.playbooks = data.playbooks;
            }
            if (metricsRes.ok) {
                this.metrics = await metricsRes.json();
            }
            if (actionsRes.ok) {
                const data = await actionsRes.json();
                this.actions = data.actions;
            }
        }
    } catch (e) {
        console.error('[PlaybookStore] Refresh failed:', e);
    } finally {
        this.loading = false;
    }
  }

  async runPlaybook(name: string, incidentID: string, steps: any[]) {
      if (!IS_BROWSER) return;
      const res = await fetch('/api/v1/playbooks/run', {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ name, incident_id: incidentID, steps }),
          credentials: 'include'
      });
      if (res.ok) {
          await this.refresh();
      }
      return res.ok;
  }
}

export const playbookStore = new PlaybookStore();
