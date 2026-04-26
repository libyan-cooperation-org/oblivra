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
