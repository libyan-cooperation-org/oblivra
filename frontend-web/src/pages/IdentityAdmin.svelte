<!-- OBLIVRA Web — Identity Administration (Svelte 5) -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { PageLayout, Badge, Button, Spinner, DataTable } from '@components/ui';
  import { 
    Users, 
    Shield, 
    Key, 
    Search, 
    UserPlus, 
    Settings, 
    Activity, 
    Lock, 
    CheckCircle, 
    XCircle,
    UserCheck,
    Globe
  } from 'lucide-svelte';
  import { request } from '../services/api';

  // -- Types --
  interface User { 
    id: string; 
    email: string; 
    name: string; 
    role_id: string; 
    role_name: string; 
    tenant_id: string; 
    created_at: string; 
    last_login?: string; 
    mfa_enabled?: boolean; 
  }
  interface Role { 
    id: string; 
    name: string; 
    permissions: string[]; 
  }

  // -- State --
  type Tab = 'users' | 'roles' | 'providers';
  let tab       = $state<Tab>('users');
  let users     = $state<User[]>([]);
  let roles     = $state<Role[]>([]);
  let loading   = $state(true);
  let userSearch = $state('');

  // -- Actions --
  async function fetchData() {
    loading = true;
    try {
      const [u, r] = await Promise.all([
        request<User[]>('/users'),
        request<Role[]>('/roles')
      ]);
      users = u ?? [];
      roles = r ?? [];
    } catch {
      users = [];
      roles = [];
    } finally {
      loading = false;
    }
  }

  onMount(fetchData);

  const filteredUsers = $derived(users.filter(u =>
    u.email.toLowerCase().includes(userSearch.toLowerCase()) ||
    u.name.toLowerCase().includes(userSearch.toLowerCase())
  ));

  const providers = [
    { name: 'OIDC / OAuth2', desc: 'Google Workspace, Okta, Azure AD', status: 'configured', icon: Globe },
    { name: 'SAML 2.0',      desc: 'Okta SAML, ADFS, PingIdentity',   status: 'configured', icon: Shield },
  ];
</script>

<PageLayout title="Identity Command" subtitle="Principal lifecycle management, RBAC sharding, and federated trust orchestration">
  {#snippet toolbar()}
    <div class="flex items-center gap-2">
      <Button variant="secondary" size="sm" onclick={fetchData}>RE-SYNC</Button>
      <Button variant="primary" size="sm" icon={UserPlus}>PROVISION_USER</Button>
    </div>
  {/snippet}

  <div class="flex flex-col h-full gap-0 -m-6 overflow-hidden">
    <!-- SUB-NAV -->
    <div class="bg-surface-1 border-b border-border-primary px-6 flex items-center gap-8 shrink-0">
      {#each (['users', 'roles', 'providers'] as Tab[]) as t}
        <button 
          class="py-3 text-[10px] font-black uppercase tracking-widest border-b-2 transition-all
            {tab === t ? 'border-accent-primary text-text-heading' : 'border-transparent text-text-muted hover:text-text-secondary'}"
          onclick={() => tab = t}
        >
          {t}
        </button>
      {/each}
    </div>

    <!-- MAIN BODY -->
    <div class="flex-1 overflow-auto bg-surface-0 p-6">
      {#if loading}
        <div class="h-full flex items-center justify-center"><Spinner /></div>
      {:else if tab === 'users'}
        <div class="space-y-4">
          <div class="flex justify-between items-center bg-surface-1 p-3 border border-border-primary rounded-sm shadow-premium">
            <div class="flex items-center gap-3 bg-surface-2 border border-border-subtle rounded-sm px-3 py-1.5 focus-within:border-accent-primary transition-colors group">
              <Search size={14} class="text-text-muted group-focus-within:text-accent-primary" />
              <input bind:value={userSearch} placeholder="Filter principals..." class="bg-transparent border-none outline-none text-xs font-mono text-text-secondary w-64" />
            </div>
            <div class="text-[9px] font-mono text-text-muted uppercase tracking-widest italic">
              Displaying {filteredUsers.length} active principals
            </div>
          </div>

          <DataTable 
            data={filteredUsers} 
            columns={[
              { key: 'name', label: 'PRINCIPAL_IDENTITY' },
              { key: 'role', label: 'AUTH_Z_ROLE', width: '180px' },
              { key: 'tenant', label: 'TENANT_SHARD', width: '150px' },
              { key: 'mfa', label: 'MFA_STATE', width: '120px' },
              { key: 'last_login', label: 'LAST_ACTIVITY', width: '150px' }
            ]} 
            compact
            rowKey="id"
          >
            {#snippet cell({ column, row })}
              {#if column.key === 'name'}
                <div class="flex flex-col">
                  <span class="text-[11px] font-bold text-text-heading uppercase tracking-tighter">{row.name}</span>
                  <span class="text-[9px] font-mono text-text-muted lowercase opacity-60">{row.email}</span>
                </div>
              {:else if column.key === 'role'}
                <Badge variant="secondary" size="xs" class="font-black italic">{(row.role_name || row.role_id).toUpperCase()}</Badge>
              {:else if column.key === 'tenant'}
                <span class="text-[10px] font-mono text-accent-primary font-bold uppercase">{row.tenant_id || 'GLOBAL_ROOT'}</span>
              {:else if column.key === 'mfa'}
                <div class="flex items-center gap-2">
                  {#if row.mfa_enabled}
                    <UserCheck size={12} class="text-status-online" />
                    <span class="text-[9px] font-mono text-status-online font-black uppercase">Secured</span>
                  {:else}
                    <XCircle size={12} class="text-alert-critical" />
                    <span class="text-[9px] font-mono text-alert-critical font-black uppercase">Exposed</span>
                  {/if}
                </div>
              {:else if column.key === 'last_login'}
                <span class="text-[10px] font-mono text-text-muted uppercase">{row.last_login ? new Date(row.last_login).toLocaleDateString() : 'NEVER'}</span>
              {/if}
            {/snippet}
          </DataTable>
        </div>

      {:else if tab === 'roles'}
        <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {#each roles as role}
            <div class="bg-surface-1 border border-border-primary rounded-sm overflow-hidden flex flex-col shadow-premium group hover:border-accent-primary transition-colors">
              <div class="bg-surface-2 border-b border-border-primary p-4 flex justify-between items-center">
                <div class="flex items-center gap-3">
                  <Shield size={16} class="text-accent-primary" />
                  <span class="text-sm font-black text-text-heading uppercase tracking-tighter italic">{role.name}</span>
                </div>
                <Button variant="ghost" size="xs" icon={Settings}>EDIT</Button>
              </div>
              <div class="p-4 flex-1 space-y-4">
                 <div class="text-[9px] font-mono text-text-muted uppercase tracking-widest border-b border-border-subtle pb-1">Permission Matrix</div>
                 <div class="flex flex-wrap gap-1.5">
                    {#each role.permissions as p}
                      <span class="px-2 py-0.5 bg-surface-2 border border-border-subtle rounded-xs text-[8px] font-mono text-text-secondary uppercase">{p}</span>
                    {/each}
                 </div>
              </div>
            </div>
          {/each}
        </div>

      {:else}
        <div class="grid grid-cols-1 md:grid-cols-2 gap-6 max-w-4xl mx-auto py-12">
          {#each providers as p}
            <div class="bg-surface-1 border border-border-primary p-6 rounded-sm space-y-6 shadow-premium relative overflow-hidden group hover:border-accent-primary transition-colors">
               <div class="absolute top-0 right-0 p-8 opacity-5 -mr-4 -mt-4 group-hover:opacity-10 transition-opacity">
                 <p.icon size={120} />
               </div>
               
               <div class="space-y-2">
                 <div class="flex items-center gap-3">
                   <p.icon size={20} class="text-accent-primary" />
                   <h3 class="text-lg font-black text-text-heading uppercase tracking-tighter italic">{p.name}</h3>
                 </div>
                 <p class="text-xs text-text-muted font-mono leading-relaxed">{p.desc}</p>
               </div>

               <div class="flex items-center justify-between pt-4 border-t border-border-subtle">
                 <div class="flex items-center gap-2">
                   <div class="w-2 h-2 rounded-full bg-status-online animate-pulse"></div>
                   <span class="text-[10px] font-mono text-status-online font-black uppercase tracking-widest">Active Sync</span>
                 </div>
                 <Button variant="secondary" size="sm">TEST CONNECTION</Button>
               </div>
            </div>
          {/each}
        </div>
      {/if}
    </div>

    <!-- FOOTER STATUS -->
    <div class="bg-surface-2 border-t border-border-primary px-3 py-1 flex items-center gap-4 text-[8px] font-mono text-text-muted shrink-0 uppercase tracking-widest">
      <div class="flex items-center gap-1.5">
        <div class="w-1 h-1 rounded-full bg-status-online"></div>
        <span>IAM_PLANE: Optimized</span>
      </div>
      <span class="text-border-primary opacity-30">|</span>
      <div class="flex items-center gap-1.5">
        <span>Auth_Protocol: OIDC/SAML2</span>
      </div>
      <div class="ml-auto opacity-40">OBLIVRA_IDENTITY_SUBSYSTEM v2.0.1</div>
    </div>
  </div>
</PageLayout>
