/**
 * OBLIVRA — Compliance Store (Svelte 5 runes)
 *
 * Orchestrates regulatory control monitoring and evidence validation.
 */
import { IS_BROWSER } from '@lib/context';

export interface ComplianceControl {
  id: string;
  framework: string;
  control: string;
  status: 'compliant' | 'warning' | 'critical';
  coverage: number;
  last_audit: string;
}

export interface ComplianceStats {
  global_score: number;
  active_breaches: number;
  controls_monitored: number;
  audit_readiness: string;
}

export class ComplianceStore {
  controls = $state<ComplianceControl[]>([]);
  stats = $state<ComplianceStats>({
    global_score: 0,
    active_breaches: 0,
    controls_monitored: 0,
    audit_readiness: 'LEVEL 1'
  });
  loading = $state(false);

  async refresh() {
    this.loading = true;
    try {
        if (IS_BROWSER) {
            const res = await fetch('/api/v1/compliance/status', { credentials: 'include' });
            if (res.ok) {
                const data = await res.json();
                this.controls = data.controls;
                this.stats = data.stats;
            }
        }
    } catch (e) {
        console.error('[ComplianceStore] Refresh failed:', e);
    } finally {
        this.loading = false;
    }
  }

  async validateAll() {
      // Future: Trigger re-evaluation
      await this.refresh();
  }
}

export const complianceStore = new ComplianceStore();
