<!--
  OBLIVRA — SIEM Search (Svelte 5)
  OQL-driven query interface for sovereign telemetry ingestion.
-->
<script lang="ts">
  import { PageLayout, Badge, Button, DataTable } from '@components/ui';
  import { Search, History, Download, Play, Save, Filter, ChevronRight, BarChart3 } from 'lucide-svelte';
  import { siemStore } from '@lib/stores/siem.svelte';

  let query = $state('select * from events limit 100');
  const results = $derived(siemStore.results);
  const isExecuting = $derived(siemStore.loading);

  async function executeQuery() {
    if (!query.trim()) return;
    await siemStore.executeQuery(query);
  }

  const queryHistory = [
    { q: 'select count(*) from logs where status = 404', time: '2m ago' },
    { q: 'events | where host.name == "SRV-PROD-01"', time: '14m ago' },
    { q: 'select intel.actor from events group by intel.actor', time: '1h ago' }
  ];
</script>

<PageLayout title="SIEM Search" subtitle="Execute OQL queries against the sovereign data lake">
  {#snippet toolbar()}
    <div class="flex items-center gap-2">
      <Button variant="secondary" size="sm" icon={History}>HISTORY</Button>
      <Button variant="secondary" size="sm" icon={Save}>SAVE QUERY</Button>
      <Button variant="primary" size="sm" icon={Download}>EXPORT RESULTS</Button>
    </div>
  {/snippet}

  <div class="flex flex-col h-full gap-4">
    <!-- QUERY EDITOR -->
    <div class="bg-surface-2 border border-border-primary rounded-sm flex flex-col shrink-0 overflow-hidden shadow-premium">
        <div class="flex items-center justify-between px-4 py-2 bg-surface-3 border-b border-border-primary">
            <div class="flex items-center gap-2">
                <Search size={14} class="text-accent" />
                <span class="text-[9px] font-mono font-bold uppercase tracking-widest text-text-heading">OQL Query Editor v1.4</span>
            </div>
            <div class="flex items-center gap-4">
                <div class="flex items-center gap-2 text-[9px] font-mono text-text-muted">
                    <span class="w-1.5 h-1.5 rounded-full bg-success"></span>
                    <span>ENGINE_ONLINE</span>
                </div>
                <div class="h-4 w-px bg-border-primary"></div>
                <span class="text-[9px] font-mono text-text-muted uppercase">Latency: 14ms</span>
            </div>
        </div>
        <div class="p-4 flex gap-4 bg-black/20">
            <div class="flex-1 font-mono text-sm">
                <textarea 
                    bind:value={query}
                    class="w-full h-24 bg-transparent text-text-secondary outline-none resize-none caret-accent placeholder:text-text-muted/30"
                    placeholder="Enter OQL query..."
                ></textarea>
            </div>
            <div class="flex flex-col gap-2">
                <Button variant="cta" class="h-full px-6 flex flex-col gap-2 font-black tracking-widest uppercase text-xs" onclick={executeQuery} loading={isExecuting}>
                    <Play size={20} fill="currentColor" />
                    RUN
                </Button>
            </div>
        </div>
        <div class="px-4 py-2 bg-surface-3 border-t border-border-primary flex items-center justify-between">
            <div class="flex items-center gap-4">
                <div class="flex items-center gap-1 text-[8px] font-mono text-text-muted hover:text-accent cursor-pointer transition-colors">
                    <Filter size={10} />
                    <span>ADD FILTER</span>
                </div>
                <div class="flex items-center gap-1 text-[8px] font-mono text-text-muted hover:text-accent cursor-pointer transition-colors">
                    <BarChart3 size={10} />
                    <span>VISUALIZE</span>
                </div>
            </div>
            <div class="text-[8px] font-mono text-text-muted uppercase">
                Ready to execute against 1.4 TB of telemetry
            </div>
        </div>
    </div>

    <!-- MAIN VIEW -->
    <div class="flex-1 flex gap-4 min-h-0">
        <!-- RESULTS -->
        <div class="flex-1 bg-surface-1 border border-border-primary rounded-sm flex flex-col min-w-0">
            <div class="flex items-center justify-between p-3 border-b border-border-primary bg-surface-2 shrink-0">
                <div class="flex items-center gap-2">
                    <span class="text-[10px] font-bold text-text-heading uppercase tracking-widest">Query Results</span>
                    <Badge variant="info" size="xs">2,412 EVENTS</Badge>
                </div>
                <div class="flex gap-2">
                    <button class="text-[9px] font-mono text-text-muted hover:text-text-secondary transition-colors">COMPACT VIEW</button>
                    <span class="text-border-primary opacity-30">|</span>
                    <button class="text-[9px] font-mono text-text-muted hover:text-text-secondary transition-colors">JSON</button>
                </div>
            </div>
            <div class="flex-1 overflow-auto mask-fade-bottom">
                <DataTable 
                    data={results} 
                    columns={[
                        { key: 'timestamp', label: 'TIMESTAMP', width: '140px' },
                        { key: 'host', label: 'HOST', width: '120px' },
                        { key: 'severity', label: 'SEV', width: '80px' },
                        { key: 'message', label: 'EVENT_DESCRIPTION' },
                        { key: 'source', label: 'SOURCE', width: '100px' }
                    ]} 
                    compact
                >
                    {#snippet render({ col, row })}
                        {#if col.key === 'timestamp'}
                            <span class="text-[9px] font-mono text-text-muted tabular-nums">{new Date(row.timestamp).toLocaleString()}</span>
                        {:else if col.key === 'host'}
                            <span class="text-[9px] font-mono text-accent font-bold">{row.host}</span>
                        {:else if col.key === 'severity'}
                            <Badge variant={row.severity === 'critical' || row.severity === 'high' ? 'critical' : row.severity === 'medium' ? 'warning' : 'info'} size="xs" class="w-full justify-center">
                                {row.severity}
                            </Badge>
                        {:else if col.key === 'message'}
                            <span class="text-[10px] font-bold text-text-secondary line-clamp-1">{row.message}</span>
                        {:else if col.key === 'source'}
                            <Badge variant="muted" size="xs" dot>{row.source}</Badge>
                        {/if}
                    {/snippet}
                </DataTable>
            </div>
        </div>

        <!-- SIDEBAR: HISTORY / FACETS -->
        <div class="w-64 flex flex-col gap-4 shrink-0">
            <div class="bg-surface-2 border border-border-primary rounded-sm p-4 space-y-4">
                <div class="flex items-center justify-between border-b border-border-primary pb-2">
                    <span class="text-[9px] font-mono font-bold text-text-muted uppercase tracking-widest">Recent Queries</span>
                    <History size={12} class="text-text-muted" />
                </div>
                <div class="space-y-3">
                    {#each queryHistory as item}
                        <div class="space-y-1 group cursor-pointer">
                            <div class="text-[9px] font-mono text-text-muted group-hover:text-accent transition-colors line-clamp-2 leading-tight">
                                {item.q}
                            </div>
                            <div class="text-[7px] font-mono text-text-muted opacity-40 uppercase">{item.time}</div>
                        </div>
                    {/each}
                </div>
            </div>

            <div class="bg-surface-2 border border-border-primary rounded-sm p-4 space-y-4 flex-1">
                <div class="flex items-center justify-between border-b border-border-primary pb-2">
                    <span class="text-[9px] font-mono font-bold text-text-muted uppercase tracking-widest">Field Facets</span>
                    <Filter size={12} class="text-text-muted" />
                </div>
                <div class="space-y-3">
                    {#each ['severity', 'host', 'event_type', 'intel_actor', 'destination_ip'] as field}
                        <div class="flex items-center justify-between group cursor-pointer">
                            <div class="flex items-center gap-2">
                                <ChevronRight size={10} class="text-text-muted group-hover:text-accent transition-transform" />
                                <span class="text-[9px] font-mono text-text-secondary uppercase">{field}</span>
                            </div>
                            <span class="text-[8px] font-mono text-text-muted opacity-40">5+</span>
                        </div>
                    {/each}
                </div>
            </div>
        </div>
    </div>
  </div>
</PageLayout>
