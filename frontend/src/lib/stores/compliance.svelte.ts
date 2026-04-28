/**
 * OBLIVRA — Compliance Store (Svelte 5 runes)
 *
 * Orchestrates regulatory control monitoring and evidence validation.
 *
 * Audit fix — `refresh()` previously gated the fetch on `IS_BROWSER`,
 * which left desktop operators staring at an empty CompliancePage
 * (no controls, all KPIs zero) because the desktop branch silently
 * fell through. apiFetch retargets `/api/*` to the in-process REST
 * listener on desktop (apiClient.ts:50), so we can drop the gate
 * entirely and rely on a single transport.
 */
import { apiFetch } from '@lib/apiClient';

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
      const res = await apiFetch('/api/v1/compliance/status');
      if (res.ok) {
        const data = await res.json();
        // Defensive: backend may return null for either field if the
        // compliance evaluator hasn't run a pass yet. Don't blow away
        // existing populated state with `undefined`.
        if (Array.isArray(data.controls)) this.controls = data.controls;
        if (data.stats && typeof data.stats === 'object') this.stats = data.stats;
      } else {
        console.warn('[ComplianceStore] /api/v1/compliance/status returned', res.status);
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
