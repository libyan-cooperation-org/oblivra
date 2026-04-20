<!--
  OBLIVRA — Identity Administration (Svelte 5)
  Unified Identity & Access Management for platform operators.
-->
<script lang="ts">
  import { KPI, Badge, DataTable, PageLayout, Button, Input } from '@components/ui';
  import { identityStore } from '@lib/stores/identity.svelte';
  import { User, Settings, Activity, ShieldCheck, Lock, Globe } from 'lucide-svelte';

  const identities = $derived(identityStore.identities);

  let searchQuery = $state('');
  const filteredIdentities = $derived(
    identities.filter(i => i.name.toLowerCase().includes(searchQuery.toLowerCase()))
  );

  const columns = [
    { key: 'name', label: 'Operator Identity' },
    { key: 'role', label: 'Mission Set', width: '150px' },
    { key: 'mfa', label: 'MFA Status', width: '120px' },
    { key: 'status', label: 'State', width: '100px' },
    { key: 'action', label: '', width: '60px' },
  ];
</script>

<PageLayout title="Identity Orbit" subtitle="Zero-trust identity management and granular RBAC orchestration">
  {#snippet toolbar()}
    <div class="flex items-center gap-2">
      <Input variant="search" placeholder="Filter identities..." bind:value={searchQuery} class="w-64" />
      <Button variant="primary" size="sm" icon="+">Create Operator</Button>
    </div>
  {/snippet}

  <div class="flex flex-col h-full gap-6">
    <div class="grid grid-cols-1 md:grid-cols-4 gap-4 shrink-0">
      <KPI label="Active Operators" value={identities.filter(i => i.status === 'active').length} trend="stable" trendValue="Nominal" variant="success" />
      <KPI label="MFA Coverage" value="88%" trend="down" trendValue="-12%" variant="warning" />
      <KPI label="Session Gravity" value="High" trend="stable" trendValue="4 Active" variant="accent" />
      <KPI label="Logic Accuracy" value="99.2%" trend="stable" trendValue="Verified" variant="success" />
    </div>

    <div class="flex-1 min-h-0 grid grid-cols-1 lg:grid-cols-3 gap-6">
       <!-- Identity Inventory -->
       <div class="lg:col-span-2 bg-surface-1 border border-border-primary rounded-md overflow-hidden flex flex-col shadow-card">
          <div class="p-3 bg-surface-2 border-b border-border-primary flex justify-between items-center text-[10px] font-bold uppercase tracking-widest text-text-muted font-mono">
             Sovereign Identity Inventory
             <Badge variant="success" size="xs">RBAC SYNC ACTIVE</Badge>
          </div>
          <div class="flex-1 overflow-auto">
             <DataTable data={filteredIdentities} {columns} compact>
               {#snippet render({ col, row, value })}
                 {#if col.key === 'status'}
                    <Badge variant={row.status === 'active' ? 'success' : 'critical'} dot={row.status === 'active'}>
                       {String(value).toUpperCase()}
                    </Badge>
                 {:else if col.key === 'role'}
                    <span class="text-[11px] font-bold text-text-secondary uppercase tracking-tighter">{value}</span>
                 {:else if col.key === 'mfa'}
                    <div class="flex items-center gap-1.5">
                       <ShieldCheck size={12} class={row.mfa === 'enabled' ? 'text-success' : 'text-warning'} />
                       <span class="text-[10px] font-bold uppercase tracking-widest">{value}</span>
                    </div>
                 {:else if col.key === 'name'}
                    <div class="flex items-center gap-3">
                       <div class="w-7 h-7 rounded-sm bg-surface-3 flex items-center justify-center border border-border-primary">
                          <User size={14} class="text-accent opacity-70" />
                       </div>
                       <div class="flex flex-col">
                          <span class="text-[12px] font-bold text-text-heading">{value}</span>
                          <span class="text-[9px] text-text-muted font-mono uppercase tracking-widest">UID-{row.id}</span>
                       </div>
                    </div>
                 {:else if col.key === 'action'}
                    <Button variant="ghost" size="xs"><Settings size={12} /></Button>
                 {:else}
                   <span class="text-[11px] text-text-secondary">{value}</span>
                 {/if}
               {/snippet}
             </DataTable>
          </div>
       </div>

       <!-- Security Posture -->
       <div class="flex flex-col gap-6">
          <div class="bg-surface-1 border border-border-primary border-dashed rounded-md p-6 flex flex-col items-center justify-center text-center gap-4 relative overflow-hidden group shadow-card hover:border-accent/40 transition-all">
             <div class="absolute inset-x-0 top-0 h-1 bg-accent/20">
                <div class="h-full bg-accent" style="width: 88%"></div>
             </div>
             <Lock size={48} class="text-accent opacity-40 group-hover:scale-110 transition-transform duration-500" />
             <div class="relative z-10">
                <h4 class="text-xs font-bold text-text-heading uppercase tracking-widest">Sovereign Proof</h4>
                <p class="text-[10px] text-text-muted mt-2 max-w-[150px]">OBLIVRA multi-factor verification utilizes hardware trust-roots for all administrator sessions.</p>
             </div>
          </div>

          <div class="flex-1 bg-surface-1 border border-border-primary rounded-md p-5 flex flex-col gap-4 shadow-card">
             <div class="text-[10px] font-bold text-text-muted uppercase tracking-widest border-b border-border-primary pb-3 flex items-center gap-2">
                <Activity size={12} />
                Global Session Entropy
             </div>
             <div class="flex-1 h-32 flex items-end justify-between px-2 gap-1">
                {#each Array(10) as _, i}
                   <div class="flex-1 bg-accent/20 rounded-t-sm border-x border-t border-accent/5"
                     style="height: {30 + Math.sin(i * 0.9) * 25 + 35}%">
                   </div>
                {/each}
             </div>
             <div class="mt-2 flex justify-between text-[9px] font-bold text-text-muted uppercase tracking-widest opacity-60">
                <span>Identity drift monitor active</span>
                <Globe size={10} />
             </div>
          </div>
       </div>
    </div>
  </div>
</PageLayout>
