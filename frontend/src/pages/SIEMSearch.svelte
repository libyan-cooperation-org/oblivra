<!--
  OBLIVRA — SIEM Search (Svelte 5)
  OQL-driven query interface for sovereign telemetry ingestion.
-->
<script lang="ts">
  import { onMount } from 'svelte';
  import { PageLayout, Badge, Button, DataTable, PopOutButton, TimeRangePicker } from '@components/ui';
  import { Search, History, Download, Play, Save, Filter, BarChart3, Pin, X, Clock, ChevronRight } from 'lucide-svelte';
  import { siemStore } from '@lib/stores/siem.svelte';
  import { savedQueriesStore, type SavedQuery } from '@lib/stores/savedQueries.svelte';

  let query = $state('select * from events limit 100');
  let timeRange = $state<{ start: string | null; end: string | null; preset: 'live' | '5m' | '1h' | '24h' | '7d' | '30d' | 'install' | 'custom' }>({
    start: null,
    end: null,
    preset: 'live',
  });
  let showHistory = $state(false);
  let saveDialogOpen = $state(false);
  let saveDialogName = $state('');

  const results = $derived(siemStore.results);
  const isExecuting = $derived(siemStore.loading);

  // Compose the final query: when a time range is bound, append the
  // appropriate WHERE clause so the user's free-form OQL stays intact.
  // OQL-aware merging happens server-side via the parser; this is a
  // simple textual append that's safe for the current grammar.
  function composedQuery(): string {
    const base = query.trim();
    if (timeRange.preset === 'live' || timeRange.preset === 'custom' && !timeRange.start) {
      return base;
    }
    if (!timeRange.start) return base;
    const startClause = `timestamp >= "${timeRange.start}"`;
    const endClause = timeRange.end ? ` AND timestamp <= "${timeRange.end}"` : '';
    // Heuristic insertion: if `where` already present, AND the clause
    // onto it; else append a fresh `where`.
    const lower = base.toLowerCase();
    if (lower.includes('where')) {
      return `${base} AND ${startClause}${endClause}`;
    }
    return `${base} where ${startClause}${endClause}`;
  }

  async function executeQuery() {
    const q = composedQuery();
    if (!q.trim()) return;
    await siemStore.executeQuery(q);
    savedQueriesStore.save('Last run', q);
  }

  function applyTimeRange(range: typeof timeRange) {
    timeRange = range;
  }

  function openSaveDialog() {
    saveDialogName = `Query at ${new Date().toLocaleTimeString()}`;
    saveDialogOpen = true;
  }

  function confirmSave() {
    if (!saveDialogName.trim() || !query.trim()) return;
    savedQueriesStore.save(saveDialogName.trim(), query.trim());
    saveDialogOpen = false;
  }

  function loadSaved(q: SavedQuery) {
    query = q.query;
    savedQueriesStore.bumpUsage(q.id);
    showHistory = false;
  }

  function togglePin(e: MouseEvent, q: SavedQuery) {
    e.stopPropagation();
    savedQueriesStore.togglePin(q.id);
  }

  function removeSaved(e: MouseEvent, q: SavedQuery) {
    e.stopPropagation();
    savedQueriesStore.remove(q.id);
  }

  onMount(() => {
    savedQueriesStore.init();
  });
</script>

<PageLayout title="SIEM Search" subtitle="Execute OQL queries against the sovereign data lake">
  {#snippet toolbar()}
    <div class="flex items-center gap-2">
      <TimeRangePicker value={timeRange} onChange={applyTimeRange} />
      <Button variant="secondary" size="sm" icon={History} onclick={() => (showHistory = !showHistory)}>HISTORY</Button>
      <Button variant="secondary" size="sm" icon={Save} onclick={openSaveDialog}>SAVE QUERY</Button>
      <Button variant="primary" size="sm" icon={Download}>EXPORT RESULTS</Button>
      <PopOutButton route="/siem-search" title="SIEM Search" />
    </div>
  {/snippet}

  <div class="flex flex-col h-full gap-4">
    <!-- PINNED QUERIES STRIP — chips for the operator's pinned queries.
         Phase 30.4c — fast one-click recall for repeat investigations. -->
    {#if savedQueriesStore.pinned.length > 0}
      <div class="flex items-center gap-2 flex-wrap shrink-0">
        <Pin size={11} class="text-text-muted shrink-0" />
        <span class="text-[8px] font-mono font-bold uppercase tracking-widest text-text-muted shrink-0">PINNED</span>
        {#each savedQueriesStore.pinned as q (q.id)}
          <!-- Two sibling buttons inside a chip — never nest <button>
               (Phase 29 lesson). -->
          <span class="group flex items-center gap-1.5 px-2 py-1 bg-surface-2 border border-border-primary hover:border-accent rounded-sm text-[10px] font-mono text-text-secondary hover:text-text-heading transition-colors">
            <button
              type="button"
              class="bg-transparent border-none p-0 cursor-pointer text-inherit hover:text-text-heading"
              onclick={() => loadSaved(q)}
              title={q.query}
            >
              {q.name}
            </button>
            <span class="text-[8px] text-text-muted opacity-60">{q.usageCount}×</span>
            <button
              type="button"
              class="opacity-0 group-hover:opacity-100 text-text-muted hover:text-error bg-transparent border-none p-0 transition-opacity cursor-pointer"
              onclick={(e) => removeSaved(e, q)}
              aria-label="Remove pinned query"
              title="Remove"
            ><X size={10} /></button>
          </span>
        {/each}
      </div>
    {/if}

    <!-- HISTORY DRAWER — shows recent saved queries with pin/load. -->
    {#if showHistory}
      <div class="bg-surface-2 border border-border-primary rounded-sm shrink-0 max-h-48 overflow-y-auto">
        <div class="flex items-center justify-between px-4 py-2 bg-surface-3 border-b border-border-primary sticky top-0">
          <div class="flex items-center gap-2">
            <Clock size={11} class="text-text-muted" />
            <span class="text-[9px] font-mono font-bold uppercase tracking-widest text-text-heading">Recent Queries</span>
            <span class="text-[8px] text-text-muted">{savedQueriesStore.recent.length}</span>
          </div>
          <button class="text-text-muted hover:text-text-heading bg-transparent border-none cursor-pointer" onclick={() => (showHistory = false)} aria-label="Close history">
            <X size={12} />
          </button>
        </div>
        <ol class="divide-y divide-border-primary/50">
          {#each savedQueriesStore.recent as q (q.id)}
            <li class="group flex items-center gap-2 px-4 py-2 hover:bg-surface-3 transition-colors">
              <!-- Two sibling buttons inside a row container — never nest
                   <button> in <button> (Phase 29 lesson). -->
              <button
                type="button"
                class="flex-1 flex flex-col min-w-0 bg-transparent border-none cursor-pointer text-left p-0"
                onclick={() => loadSaved(q)}
              >
                <span class="text-[10px] font-mono text-text-heading truncate">{q.name}</span>
                <span class="text-[9px] font-mono text-text-muted truncate">{q.query}</span>
              </button>
              <span class="text-[8px] font-mono text-text-muted opacity-60 shrink-0">{q.usageCount}×</span>
              <button
                type="button"
                class="text-text-muted hover:text-accent bg-transparent border-none p-0.5 cursor-pointer shrink-0"
                onclick={(e) => togglePin(e, q)}
                aria-label={q.pinned ? 'Unpin query' : 'Pin query'}
                title={q.pinned ? 'Unpin' : 'Pin'}
              >
                <Pin size={10} class={q.pinned ? 'fill-accent text-accent' : ''} />
              </button>
              <button
                type="button"
                class="text-text-muted hover:text-error bg-transparent border-none p-0.5 cursor-pointer shrink-0"
                onclick={(e) => removeSaved(e, q)}
                aria-label="Delete saved query"
                title="Delete"
              >
                <X size={10} />
              </button>
            </li>
          {/each}
          {#if savedQueriesStore.recent.length === 0}
            <li class="px-4 py-3 text-[10px] font-mono text-text-muted opacity-60 text-center">
              No saved queries yet — run a query and click SAVE.
            </li>
          {/if}
        </ol>
      </div>
    {/if}

    <!-- SAVE DIALOG -->
    {#if saveDialogOpen}
      <div class="bg-surface-2 border border-accent rounded-sm shrink-0 p-3 flex items-center gap-2">
        <Save size={12} class="text-accent shrink-0" />
        <input
          type="text"
          bind:value={saveDialogName}
          class="flex-1 bg-surface-1 border border-border-primary rounded-sm px-2 py-1 text-[11px] font-mono text-text-heading outline-none focus:border-accent"
          placeholder="Query name"
          onkeydown={(e) => { if (e.key === 'Enter') confirmSave(); if (e.key === 'Escape') saveDialogOpen = false; }}
        />
        <Button variant="primary" size="sm" onclick={confirmSave} disabled={!saveDialogName.trim()}>SAVE</Button>
        <Button variant="ghost" size="sm" onclick={() => (saveDialogOpen = false)}>Cancel</Button>
      </div>
    {/if}

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
                    {#each savedQueriesStore.recent.slice(0, 6) as item (item.id)}
                        <button
                            type="button"
                            class="w-full block space-y-1 group cursor-pointer bg-transparent border-none text-left p-0"
                            onclick={() => loadSaved(item)}
                            title={item.query}
                        >
                            <div class="text-[9px] font-mono text-text-muted group-hover:text-accent transition-colors line-clamp-2 leading-tight">
                                {item.query}
                            </div>
                            <div class="text-[7px] font-mono text-text-muted opacity-40 uppercase">
                                {item.usageCount}× · {new Date(item.lastUsed).toLocaleTimeString()}
                            </div>
                        </button>
                    {:else}
                        <div class="text-[9px] font-mono text-text-muted opacity-40 italic">
                            No history yet — run a query.
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
