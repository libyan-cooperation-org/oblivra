<script lang="ts">
  import { Shield, Globe, Activity } from 'lucide-svelte';
  import { KPI, Badge, DataTable, PageLayout, Button, Input } from '@components/ui';

  const intelFeed = [
    { id: 'I-442', source: 'AlienVault', indicator: '192.168.1.104', type: 'IP', risk: 'high', firstSeen: '2h ago' },
    { id: 'I-443', source: 'CrowdStrike', indicator: 'libcrypt.so.6', type: 'Hash', risk: 'critical', firstSeen: '10m ago' },
    { id: 'I-444', source: 'SANS-ISC', indicator: 'pwned.onion', type: 'Domain', risk: 'medium', firstSeen: '1d ago' },
    { id: 'I-445', source: 'Internal', indicator: 'admin_backdoor.py', type: 'Filename', risk: 'critical', firstSeen: 'now' },
  ];

  let searchQuery = $state('');
  const filteredFeed = $derived(intelFeed.filter(i => i.indicator.toLowerCase().includes(searchQuery.toLowerCase())));
</script>

<PageLayout title="Intelligence Orbit" subtitle="Federated threat indicators and risk correlation: Synchronizing global adversary signals in real-time">
  {#snippet toolbar()}
    <div class="flex items-center gap-3">
      <Input variant="search" placeholder="Filter indicators..." bind:value={searchQuery} class="w-64" />
      <Button variant="secondary" size="sm">Sync Feeds</Button>
    </div>
  {/snippet}

  <div class="flex flex-col h-full gap-6">
    <div class="grid grid-cols-1 md:grid-cols-4 gap-4 shrink-0">
      <KPI label="Verified Mal-Blocks" value="1.2M" trend="stable" trendValue="Library" variant="success" />
      <KPI label="Global Campaigns" value="42" trend="up" trendValue="+3" variant="critical" />
      <KPI label="Platform Latency" value="0.2s" trend="stable" trendValue="Minimal" variant="success" />
      <KPI label="Intel Reliability" value="High" trend="stable" trendValue="Hardened" variant="success" />
    </div>

    <div class="flex-1 min-h-0 grid grid-cols-1 lg:grid-cols-3 gap-6">
       <!-- Intel Feed -->
       <div class="lg:col-span-2 bg-surface-1 border border-border-primary rounded-md overflow-hidden flex flex-col shadow-premium">
          <div class="p-3 bg-surface-2 border-b border-border-primary flex justify-between items-center text-[10px] font-bold uppercase tracking-widest text-text-muted">
             Global Indicator Ingest Ledger
          </div>
          <div class="flex-1 overflow-auto">
            <DataTable data={filteredFeed} columns={[
              { key: 'source', label: 'Origin Agency' },
              { key: 'indicator', label: 'Indicator / Primitive' },
              { key: 'risk', label: 'Gravity', width: '100px' },
              { key: 'firstSeen', label: 'Detection', width: '100px' }
            ]} compact striped>
              {#snippet render({ value, col })}
                {#if col.key === 'risk'}
                  <Badge variant={value === 'critical' ? 'critical' : value === 'high' ? 'warning' : 'info'}>
                    {value.toUpperCase()}
                  </Badge>
                {:else if col.key === 'source'}
                  <div class="flex items-center gap-2">
                    <Globe size={12} class="text-accent opacity-60" />
                    <span class="text-[11px] font-bold text-text-heading">{value}</span>
                  </div>
                {:else if col.key === 'indicator'}
                   <code class="text-[10px] text-accent font-mono truncate max-w-[200px]">{value}</code>
                {:else if col.key === 'firstSeen'}
                   <span class="text-[10px] text-text-muted font-mono">{value}</span>
                {:else}
                   <span class="text-[11px] text-text-secondary">{value}</span>
                {/if}
              {/snippet}
            </DataTable>
          </div>
       </div>

       <!-- Intel Visualization -->
       <div class="flex flex-col gap-6">
          <div class="bg-surface-1 border border-border-primary rounded-md p-6 flex flex-col items-center justify-center text-center gap-4 relative overflow-hidden group border-dashed shadow-sm">
             <Shield size={48} class="text-accent opacity-40 group-hover:scale-110 transition-transform" />
             <div class="relative z-10">
                <h4 class="text-xs font-bold text-text-heading uppercase tracking-widest">Post-Verify Logic</h4>
                <p class="text-[10px] text-text-muted mt-2 max-w-[150px]">OBLIVRA automatically re-verifies all indicators across private mesh nodes before ingestion.</p>
             </div>
             <div class="absolute inset-x-0 bottom-0 h-1 bg-accent/20">
                <div class="h-full bg-accent animate-pulse" style="width: 70%"></div>
             </div>
          </div>

          <div class="flex-1 bg-surface-1 border border-border-primary rounded-md p-4 space-y-4">
             <div class="text-[10px] font-bold text-text-muted uppercase tracking-widest border-b border-border-primary pb-2 flex items-center gap-2">
                <Activity size={12} />
                Global Ingest Pressure
             </div>
             <div class="space-y-4">
                {#each Array(3) as _, i}
                   <div class="flex flex-col gap-1">
                      <div class="flex justify-between text-[9px] font-mono text-text-muted">
                         <span>FEED CHANNEL {i+1}</span>
                         <span class="text-text-heading font-bold">{Math.floor(80 + Math.random() * 20)}%</span>
                      </div>
                      <div class="h-1 bg-surface-3 rounded-full overflow-hidden">
                         <div class="h-full bg-accent" style="width: {80 + Math.random() * 20}%"></div>
                      </div>
                   </div>
                {/each}
             </div>
          </div>
       </div>
    </div>
  </div>
</PageLayout>
