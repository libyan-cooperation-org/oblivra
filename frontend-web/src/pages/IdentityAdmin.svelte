<!-- OBLIVRA Web — Identity Administration (Svelte 5) -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { PageLayout, Badge, Button, Spinner, DataTable } from '@components/ui';
  import { 
    Shield, 
    Search, 
    UserPlus, 
    Settings, 
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
  // -- State --
  type Tab = 'users' | 'roles' | 'providers';
  let tab       = $state<Tab>('users');
  let users     = $state<User[]>([]);
  let roles     = $state<Role[]>([]);
  let loading   = $state(true);
  let userSearch = $state('');
  let selectedUser = $state<User | null>(null);

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
      if (users.length > 0) selectedUser = users[0];
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
    <div class="flex-1 flex min-h-0 bg-surface-0">
      {#if loading}
        <div class="flex-1 flex items-center justify-center"><Spinner /></div>
      {:else if tab === 'users'}
        <!-- USER LIST MAIN AREA -->
        <div class="flex-1 flex flex-col border-r border-border-primary overflow-hidden">
          <div class="p-3 bg-surface-1 border-b border-border-primary flex justify-between items-center shrink-0">
            <div class="flex items-center gap-3 bg-surface-2 border border-border-subtle rounded-sm px-3 py-1.5 focus-within:border-accent-primary transition-colors group">
              <Search size={14} class="text-text-muted group-focus-within:text-accent-primary" />
              <input bind:value={userSearch} placeholder="Filter principals..." class="bg-transparent border-none outline-none text-xs font-mono text-text-secondary w-64" />
            </div>
            <div class="text-[9px] font-mono text-text-muted uppercase tracking-widest italic">
              Displaying {filteredUsers.length} active principals
            </div>
          </div>

          <div class="flex-1 overflow-auto">
            <DataTable 
              data={filteredUsers} 
              columns={[
                { key: 'name', label: 'PRINCIPAL_IDENTITY' },
                { key: 'role', label: 'AUTH_Z_ROLE', width: '150px' },
                { key: 'mfa', label: 'MFA_STATE', width: '100px' },
                { key: 'last_login', label: 'LAST_ACTIVITY', width: '120px' }
              ]} 
              compact
              rowKey="id"
              onRowClick={(row) => selectedUser = row}
            >
              {#snippet cell({ column, row })}
                {#if column.key === 'name'}
                  <div class="flex flex-col">
                    <span class="text-[11px] font-bold uppercase tracking-tighter {selectedUser?.id === row.id ? 'text-accent-primary' : 'text-text-heading'}">{row.name}</span>
                    <span class="text-[9px] font-mono text-text-muted lowercase opacity-60">{row.email}</span>
                  </div>
                {:else if column.key === 'role'}
                  <Badge variant="secondary" size="xs" class="font-black italic">{(row.role_name || row.role_id).toUpperCase()}</Badge>
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
        </div>

        <!-- USER DETAIL SIDEBAR (Refined from prototype) -->
        <aside class="w-[320px] bg-surface-1 flex flex-col shrink-0 overflow-hidden">
          {#if selectedUser}
            <div class="p-4 bg-surface-2 border-b border-border-primary shrink-0">
              <span class="text-[10px] font-mono text-text-muted uppercase tracking-widest">Principal Detail</span>
            </div>
            
            <div class="flex-1 overflow-y-auto">
              <div class="p-6 space-y-6">
                <!-- Profile Header -->
                <div class="flex items-center gap-4">
                  <div class="w-12 h-12 rounded-full bg-accent-primary/10 border border-accent-primary/30 flex items-center justify-center text-accent-primary font-black text-xl italic">
                    {selectedUser.name.split(',')[0][0]}
                  </div>
                  <div>
                    <h3 class="text-lg font-black text-text-heading uppercase tracking-tighter italic">{selectedUser.name}</h3>
                    <Badge variant="secondary" size="xs" class="mt-1 font-mono uppercase opacity-70">{(selectedUser.role_name || selectedUser.role_id)}</Badge>
                  </div>
                </div>

                <!-- Fields -->
                <div class="space-y-4">
                  <div class="space-y-1">
                    <div class="text-[9px] font-mono text-text-muted uppercase">Primary Email</div>
                    <div class="text-xs font-bold text-text-secondary">{selectedUser.email}</div>
                  </div>
                  <div class="space-y-1">
                    <div class="text-[9px] font-mono text-text-muted uppercase">Department</div>
                    <div class="text-xs font-bold text-text-secondary">SOC_OPERATIONS</div>
                  </div>
                  <div class="space-y-1">
                    <div class="text-[9px] font-mono text-text-muted uppercase">Assigned Tenant</div>
                    <div class="text-xs font-bold text-accent-primary">{selectedUser.tenant_id || 'GLOBAL_ROOT'}</div>
                  </div>
                </div>

                <!-- Permissions -->
                <div class="pt-4 border-t border-border-subtle space-y-3">
                  <div class="text-[9px] font-black text-text-muted uppercase tracking-widest">Module Permissions</div>
                  <div class="space-y-1">
                    {#each [
                      { mod: 'SIEM_SEARCH', acc: 'READ/WRITE' },
                      { mod: 'FLEET_MGMT', acc: 'READ_ONLY' },
                      { mod: 'ISOLATION', acc: 'APPROVE_REQ' }
                    ] as perm}
                      <div class="flex justify-between items-center text-[10px]">
                        <span class="text-text-muted font-mono">{perm.mod}</span>
                        <span class="font-bold text-status-online">{perm.acc}</span>
                      </div>
                    {/each}
                  </div>
                </div>

                <!-- Recent Activity -->
                <div class="pt-4 border-t border-border-subtle space-y-3">
                  <div class="text-[9px] font-black text-text-muted uppercase tracking-widest">Recent Activity</div>
                  <div class="space-y-2">
                    <div class="flex gap-3 text-[10px]">
                      <span class="text-text-muted font-mono opacity-50 shrink-0">00:54</span>
                      <span class="text-text-secondary">Evidence sealed EVD-044</span>
                    </div>
                    <div class="flex gap-3 text-[10px]">
                      <span class="text-text-muted font-mono opacity-50 shrink-0">00:52</span>
                      <span class="text-text-secondary">SIEM query executed</span>
                    </div>
                  </div>
                </div>
              </div>
            </div>

            <!-- Sidebar Actions -->
            <div class="p-4 border-t border-border-primary bg-surface-2 space-y-2 shrink-0">
               <Button variant="primary" class="w-full text-[10px]" size="sm">EDIT_PERMISSIONS</Button>
               <Button variant="secondary" class="w-full text-alert-critical border-alert-critical/30 text-[10px]" size="sm">TERMINATE_SESSION</Button>
            </div>
          {:else}
            <div class="flex-1 flex items-center justify-center p-12 text-center opacity-30">
               <span class="text-xs font-mono uppercase tracking-widest">Select a principal to view profile</span>
            </div>
          {/if}
        </aside>
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
