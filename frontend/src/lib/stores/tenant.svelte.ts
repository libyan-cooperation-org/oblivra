/**
 * OBLIVRA — Multi-Tenant Store (Svelte 5 runes)
 *
 * Orchestrates cross-tenant management and platform-wide metrics.
 */
import { IS_BROWSER } from '@lib/context';

export interface Tenant {
  id: string;
  name: string;
  abbr: string;
  color: string;
  mode: 'SOVEREIGN' | 'CLOUD';
  tier: string;
  agents: number;
  eps: string;
  incidents: number;
  storage: string;
  health: number;
}

export class MultiTenantStore {
  tenants = $state<Tenant[]>([
    { id: 'GOV-FIN-001', name: 'GovFin Authority', abbr: 'GF', color: '#5aaef0', mode: 'SOVEREIGN', tier: 'T1', agents: 4900, eps: '148K', incidents: 2, storage: '37%', health: 98.4 },
    { id: 'GOV-DEF-002', name: 'Defence Ops CERT', abbr: 'DO', color: '#9878e0', mode: 'SOVEREIGN', tier: 'T1', agents: 3800, eps: '180K', incidents: 1, storage: '52%', health: 99.1 },
    { id: 'CORP-BANK-03', name: 'Global Finance S.A.', abbr: 'GF', color: '#1aaa60', mode: 'CLOUD', tier: 'T2', agents: 2100, eps: '64K', incidents: 0, storage: '12%', health: 99.8 },
    { id: 'CORP-ENER-04', name: 'Grid Control Unit', abbr: 'GC', color: '#e08020', mode: 'SOVEREIGN', tier: 'T1', agents: 1640, eps: '49K', incidents: 0, storage: '28%', health: 97.2 }
  ]);
  
  platformMetrics = $state({
    totalAgents: 0,
    totalIncidents: 0,
    platformEps: '0',
    activeTenants: 0
  });

  loading = $state(false);
  async refresh() {
    this.loading = true;
    try {
        if (IS_BROWSER) {
            const res = await fetch('/api/v1/platform/metrics', { credentials: 'include' });
            if (res.ok) {
                const metrics = await res.json();
                this.platformMetrics = {
                    totalAgents: metrics.totalAgents,
                    totalIncidents: metrics.activeIncidents,
                    platformEps: metrics.platformEps,
                    activeTenants: metrics.activeTenants
                };
            }
        }
    } catch (e) {
        console.error('[TenantStore] Refresh failed:', e);
    } finally {
        this.loading = false;
    }
  }

  async provisionTenant(name: string) {
    // Future: Call provisioning service
    console.log(`Provisioning new tenant: ${name}`);
  }
}

export const tenantStore = new MultiTenantStore();
