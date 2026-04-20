/**
 * OBLIVRA — Identity Store (Svelte 5 runes)
 *
 * Orchestrates operator identities, RBAC roles, and session gravity.
 */
import { IS_BROWSER } from '@lib/context';

export interface Identity {
  id: string;
  name: string;
  role: string;
  mfa: 'enabled' | 'disabled';
  status: 'active' | 'suspended';
  lastLogin: string;
  tenant_id?: string;
}

export interface Role {
  id: string;
  name: string;
  permissions: string[];
}

export class IdentityStore {
  identities = $state<Identity[]>([]);
  roles = $state<Role[]>([]);
  loading = $state(false);

  constructor() {
    this.refresh();
  }

  async refresh() {
    this.loading = true;
    try {
        if (IS_BROWSER) {
            const res = await fetch('/api/v1/identities', { credentials: 'include' });
            if (res.ok) {
                const data = await res.json();
                this.identities = data.identities.map((u: any) => ({
                    id: u.id,
                    name: u.username || u.name,
                    role: u.role || 'Operator',
                    mfa: u.mfa_enabled ? 'enabled' : 'disabled',
                    status: u.suspended ? 'suspended' : 'active',
                    lastLogin: u.last_login || 'N/A',
                    tenant_id: u.tenant_id
                }));
            }

            const rolesRes = await fetch('/api/v1/identities/roles', { credentials: 'include' });
            if (rolesRes.ok) {
                const data = await rolesRes.json();
                this.roles = data.roles;
            }
        } else {
            // Desktop fallback or mock
            this.identities = [
                { id: 'U-01', name: 'maverick', role: 'Super Admin', mfa: 'enabled', status: 'active', lastLogin: '2m ago' },
                { id: 'U-02', name: 'iceman', role: 'Security Analyst', mfa: 'enabled', status: 'active', lastLogin: '14m ago' }
            ];
        }
    } catch (e) {
        console.error('[IdentityStore] Refresh failed:', e);
    } finally {
        this.loading = false;
    }
  }
}

export const identityStore = new IdentityStore();
