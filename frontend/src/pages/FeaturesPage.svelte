<!--
  OBLIVRA — Features & Capabilities (Svelte 5)
  Managing platform-wide feature flags and modular capability blocks.
-->
<script lang="ts">
  import { KPI, PageLayout, Badge, Button, DataTable } from '@components/ui';
  import { Zap, Activity, Eye, Settings } from 'lucide-svelte';

  const features: Record<string, any>[] = [
    { id: 'F-01', name: 'Adaptive Egress Filtering', state: 'enabled', tier: 'Sovereign' },
    { id: 'F-02', name: 'Cognitive AI Shell V2', state: 'experimental', tier: 'Enterprise' },
    { id: 'F-03', name: 'P2P Tunnel Mesh', state: 'enabled', tier: 'Sovereign' },
    { id: 'F-04', name: 'Hardware Trust Root', state: 'enabled', tier: 'Platform' },
  ];
</script>

<PageLayout title="Platform Features" subtitle="Capability orchestration: Managing platform-wide feature toggles, experimental modules and sovereign-grade entitlements">
  {#snippet toolbar()}
     <Button variant="secondary" size="sm">Reset to Baseline</Button>
     <Button variant="primary" size="sm" icon="⭐">Feature Marketplace</Button>
  {/snippet}

  <div class="flex flex-col h-full gap-6">
    <div class="grid grid-cols-1 md:grid-cols-4 gap-4">
      <KPI label="Active Flags" value="12" trend="stable" trendValue="Nominal" />
      <KPI label="Release Channel" value="STABLE" trend="stable" trendValue="Hardened" variant="success" />
      <KPI label="Entitlement Tier" value="SOVEREIGN" trend="stable" trendValue="Verified" variant="success" />
      <KPI label="System Agility" value="High" trend="stable" trendValue="Optimized" variant="success" />
    </div>

    <div class="flex-1 min-h-0 grid grid-cols-1 lg:grid-cols-3 gap-6">
      <!-- Feature Registry -->
      <div class="lg:col-span-2 bg-surface-1 border border-border-primary rounded-md overflow-hidden flex flex-col shadow-premium">
         <div class="p-3 bg-surface-2 border-b border-border-primary flex justify-between items-center text-[10px] font-bold uppercase tracking-widest text-text-muted">
            Entitlement & Feature Registry
         </div>
         <div class="flex-1 overflow-auto">
            <DataTable data={features} columns={[
              { key: 'name', label: 'Feature Identity' },
              { key: 'tier', label: 'Domain', width: '120px' },
              { key: 'state', label: 'Runtime Status', width: '120px' },
              { key: 'action', label: '', width: '60px' }
            ]} compact>
              {#snippet render({ col: column, row })}
                {#if column.key === 'state'}
                   <Badge variant={row.state === 'enabled' ? 'success' : row.state === 'experimental' ? 'accent' : 'muted'}>{row.state.toUpperCase()}</Badge>
                {:else if column.key === 'name'}
                   <div class="flex items-center gap-2">
                      <Zap size={14} class="text-accent opacity-70" />
                      <span class="text-[11px] font-bold text-text-heading">{row.name}</span>
                   </div>
                {:else if column.key === 'tier'}
                   <span class="text-[10px] font-bold text-text-muted uppercase tracking-widest">{row.tier}</span>
                {:else if column.key === 'action'}
                   <Button variant="ghost" size="xs"><Settings size={12} /></Button>
                {:else}
                  <span class="text-[11px] text-text-secondary">{row[column.key]}</span>
                {/if}
              {/snippet}
            </DataTable>
         </div>
      </div>

      <!-- Feature Insights -->
      <div class="flex flex-col gap-6">
         <div class="bg-surface-1 border border-border-primary rounded-md p-6 flex flex-col items-center justify-center text-center gap-4 relative overflow-hidden group border-dashed shadow-sm">
            <Eye size={48} class="text-accent opacity-40 animate-pulse" />
            <div class="relative z-10">
               <h4 class="text-xs font-bold text-text-heading uppercase tracking-widest">Logic Agility</h4>
               <p class="text-[10px] text-text-muted mt-2 max-w-[150px]">OBLIVRA allows for the dynamic hot-loading of new capability blocks without platform downtime.</p>
            </div>
         </div>

         <div class="flex-1 bg-surface-1 border border-border-primary rounded-md p-4 space-y-4">
            <div class="text-[10px] font-bold text-text-muted uppercase tracking-widest border-b border-border-primary pb-2 flex items-center gap-2">
               <Activity size={12} />
               Release Entropy
            </div>
            <div class="flex-1 h-32 flex items-end justify-between px-2 gap-1 font-mono">
               {#each Array(10) as _}
                  <div class="flex-1 bg-accent/20 rounded-t-sm border-x border-t border-accent/10" style="height: {30 + Math.random() * 60}%"></div>
               {/each}
            </div>
         </div>
      </div>
    </div>
  </div>
</PageLayout>
