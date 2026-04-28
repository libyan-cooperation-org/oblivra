/**
 * OBLIVRA — Multi-Tenant Store (Svelte 5 runes)
 *
 * Orchestrates cross-tenant management and platform-wide metrics.
 */
import { IS_BROWSER } from '@lib/context';
import { apiFetch } from '@lib/apiClient';
import type { TacticalMessage } from './collaboration.svelte';

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
  tenants = $state<Tenant[]>([]);
  messages = $state<TacticalMessage[]>([]);
  
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
            const res = await apiFetch('/api/v1/platform/metrics');
            if (res.ok) {
                const metrics = await res.json();
                this.platformMetrics = {
                    totalAgents: metrics.totalAgents,
                    totalIncidents: metrics.activeIncidents,
                    platformEps: metrics.platformEps,
                    activeTenants: metrics.activeTenants
                };
            }
        } else {
            // Desktop mode — derive platform-level metrics from the live
            // stores rather than calling a dedicated RPC. The agent +
            // alert + diagnostic stores already poll the backend; we just
            // aggregate. Closes the audit gap where desktop operators saw
            // a perpetually-empty MultiTenantAdmin page.
            try {
                const [{ agentStore }, { alertStore }, { diagnosticsStore }] = await Promise.all([
                    import('./agent.svelte'),
                    import('./alerts.svelte'),
                    import('./diagnostics.svelte'),
                ]);
                const agents = (agentStore as any).agents ?? [];
                const alerts = (alertStore as any).alerts ?? [];
                const eps = (diagnosticsStore as any).eps ?? 0;
                this.platformMetrics = {
                    totalAgents: agents.length,
                    totalIncidents: alerts.filter((a: any) => a.status === 'open').length,
                    platformEps: String(eps),
                    activeTenants: 1, // Desktop deployments are single-tenant by design.
                };
            } catch (err) {
                console.warn('[TenantStore] Desktop metric aggregation failed:', err);
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
