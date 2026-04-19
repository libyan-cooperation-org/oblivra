<!-- OBLIVRA Web — SIEM Search (Svelte 5) -->
<script lang="ts">
  import { PageLayout, Badge, Button, Spinner } from '@components/ui';
  import { 
    Search, 
    History, 
    Terminal, 
    Filter, 
    Download, 

    ChevronRight,
    ChevronDown,
    Activity
  } from 'lucide-svelte';
  import { request } from '../services/api';

  // -- Types --
  interface HostEvent { 
    id: number; 
    tenant_id: string; 
    host_id: string; 
    timestamp: string; 
    event_type: string; 
    source_ip: string; 
    user: string; 
    raw_log: string; 
  }

  const SMAP: Record<string, { label: string; variant: any }> = {
    failed_login:     { label: 'CRIT', variant: 'danger' },
    security_alert:   { label: 'HIGH', variant: 'warning' },
    sudo_exec:        { label: 'MED',  variant: 'warning' },
    successful_login: { label: 'INFO', variant: 'success' },
  };

  // -- State --
  let query     = $state('');
  let limit     = $state(100);
  let lastQuery = $state('');
  let results   = $state<HostEvent[]>([]);
  let loading   = $state(false);
  let expanded  = $state<Set<number>>(new Set());

  // -- Helpers --
  const getSev = (t: string) => SMAP[t] ?? { label: 'LOW', variant: 'secondary' };
  
  const histogram = $derived.by(() => {
    if (!results.length) return [];
    const times = results.map(e => new Date(e.timestamp).getTime());
    const minT = Math.min(...times), maxT = Math.max(...times);
    const bins = Array.from({ length: 40 }, () => 0);
    const span = maxT - minT || 1;
    results.forEach(e => { 
      let i = Math.floor(((new Date(e.timestamp).getTime() - minT) / span) * 40); 
      if (i >= 40) i = 39; 
      bins[i]++; 
    });
    const mx = Math.max(...bins, 1);
    return bins.map((count, i) => ({ 
      count, 
      pct: (count / mx) * 100, 
      label: new Date(minT + (i / 40) * span).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }) 
    }));
  });

  const fields = $derived.by(() => {
    const fc: Record<string, Record<string, number>> = {};
    results.forEach(e => {
      (['host_id', 'event_type', 'source_ip', 'user'] as const).forEach(k => {
        const v = String((e as any)[k] || '');
        if (v && v !== 'undefined') { 
          if (!fc[k]) fc[k] = {}; 
          fc[k][v] = (fc[k][v] || 0) + 1; 
        }
      });
    });
    return Object.entries(fc).map(([field, vm]) => ({
      field, 
      vals: Object.entries(vm).sort((a, b) => b[1] - a[1]).slice(0, 5)
    }));
  });

  // -- Actions --
  async function runSearch() {
    if (!query.trim()) return;
    lastQuery = query;
    loading = true;
    expanded = new Set();
    try {
      const p = new URLSearchParams({ q: query, limit: String(limit) });
      const r = await request<{ events: HostEvent[] }>(`/siem/search?${p}`);
      results = r.events ?? [];
    } catch { 
      results = []; 
    } finally {
      loading = false;
    }
  }

  function handleKey(e: KeyboardEvent) { if (e.key === 'Enter') runSearch(); }
  function toggleRow(id: number) { 
    const s = new Set(expanded); 
    s.has(id) ? s.delete(id) : s.add(id); 
    expanded = s; 
  }
  function appendFilter(field: string, val: string) {
    const cur = query.trim();
    query = cur ? `${cur} AND ${field}:"${val}"` : `${field}:"${val}"`;
    runSearch();
  }
</script>

<PageLayout title="Tactical Search" subtitle="High-performance event analysis, field extraction, and longitudinal timeline mapping">
  {#snippet toolbar()}
    <div class="flex items-center gap-2">
      <Button variant="secondary" size="sm" icon={Download}>EXPORT_JSON</Button>
      <Button variant="secondary" size="sm" onclick={() => query = ''}>CLEAR</Button>
    </div>
  {/snippet}

  <div class="flex flex-col h-full gap-0 -m-6 overflow-hidden">
    <!-- SEARCH BAR AREA -->
    <div class="p-6 bg-surface-1 border-b border-border-primary shrink-0 space-y-4 shadow-premium">
      <div class="flex items-center gap-3">
        <div class="flex-1 flex items-center gap-3 bg-surface-2 border border-border-primary rounded-sm px-4 py-2 focus-within:border-accent-primary transition-colors group">
          <span class="text-accent-primary font-black italic">OQL:</span>
          <input 
            type="text" 
            bind:value={query}
            onkeydown={handleKey}
            placeholder="e.g. event_type:failed_login OR source_ip:10.0.0.*"
            class="flex-1 bg-transparent border-none outline-none text-sm font-mono text-text-heading placeholder:text-text-muted/40"
          />
          <div class="w-px h-4 bg-border-subtle"></div>
          <select 
            bind:value={limit}
            class="bg-transparent border-none outline-none text-[10px] font-mono font-bold text-accent-primary cursor-pointer uppercase tracking-tighter"
          >
            {#each [100, 250, 500, 1000] as n}<option value={n}>{n} events</option>{/each}
          </select>
        </div>
        <Button variant="primary" size="md" class="font-black italic px-8" onclick={runSearch}>
          {loading ? 'EXECUTING...' : 'RUN_QUERY'}
        </Button>
      </div>

      {#if results.length > 0}
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-6">
            <div class="flex items-center gap-2">
              <span class="text-[9px] font-mono text-text-muted uppercase tracking-widest">Matched:</span>
              <span class="text-xs font-mono font-black text-accent-primary">{results.length}</span>
            </div>
            <div class="flex items-center gap-2">
              <span class="text-[9px] font-mono text-text-muted uppercase tracking-widest">Execution:</span>
              <span class="text-xs font-mono font-black text-status-online">12ms</span>
            </div>
          </div>
          
          <!-- Mini Histogram -->
          <div class="flex items-end gap-0.5 h-6 w-64 border-b border-border-subtle pb-0.5">
            {#each histogram as bin}
              <div 
                class="flex-1 bg-accent-primary/40 hover:bg-accent-primary transition-colors cursor-help"
                style="height: {bin.pct}%"
                title="{bin.label}: {bin.count}"
              ></div>
            {/each}
          </div>
        </div>
      {/if}
    </div>

    <!-- MAIN BODY -->
    <div class="flex-1 flex min-h-0 bg-surface-0 overflow-hidden">
      <!-- SIDEBAR: INTERESTING FIELDS -->
      <div class="w-64 border-r border-border-primary flex flex-col shrink-0 bg-surface-1">
        <div class="p-3 bg-surface-2 border-b border-border-primary flex items-center gap-2">
          <Filter size={14} class="text-accent-primary" />
          <span class="text-[9px] font-mono font-bold uppercase tracking-widest text-text-heading">Extracted Fields</span>
        </div>
        
        <div class="flex-1 overflow-y-auto p-4 space-y-6">
          {#if fields.length === 0}
            <div class="text-center py-12 opacity-20 space-y-2">
              <Activity size={32} class="mx-auto" />
              <p class="text-[8px] font-mono uppercase tracking-widest">No fields extracted</p>
            </div>
          {:else}
            {#each fields as group}
              <div class="space-y-2">
                <div class="text-[10px] font-black text-text-heading uppercase tracking-tighter border-b border-border-subtle pb-1">
                  {group.field.replace('_', ' ')}
                </div>
                <div class="space-y-1">
                  {#each group.vals as [val, count]}
                    <button 
                      class="w-full flex justify-between items-center px-2 py-1.5 rounded-xs hover:bg-surface-2 group transition-all"
                      onclick={() => appendFilter(group.field, val as string)}
                    >
                      <span class="text-[10px] font-mono text-text-secondary truncate text-left flex-1 group-hover:text-accent-primary">{val}</span>
                      <span class="text-[9px] font-mono text-text-muted bg-surface-0 px-1.5 rounded-full border border-border-subtle ml-2">{count}</span>
                    </button>
                  {/each}
                </div>
              </div>
            {/each}
          {/if}
        </div>
      </div>

      <!-- RESULTS AREA -->
      <div class="flex-1 overflow-auto relative">
        {#if loading}
          <div class="absolute inset-0 bg-surface-0/60 z-20 flex items-center justify-center backdrop-blur-[2px]">
            <Spinner />
          </div>
        {/if}

        <table class="w-full border-collapse text-left">
          <thead class="sticky top-0 z-10 bg-surface-2 border-b border-border-primary shadow-sm">
            <tr class="text-[10px] font-mono font-bold text-text-muted uppercase tracking-widest">
              <th class="p-3 w-10"></th>
              <th class="p-3 w-48">Timestamp</th>
              <th class="p-3 w-40">Host</th>
              <th class="p-3 w-40">Event Type</th>
              <th class="p-3">Raw Log Snippet</th>
            </tr>
          </thead>
          <tbody class="divide-y divide-border-subtle">
            {#if results.length === 0 && !loading}
              <tr>
                <td colspan="5" class="py-32 text-center opacity-20 space-y-4">
                  <Search size={64} class="mx-auto" />
                  <p class="text-xs font-mono uppercase tracking-[0.4em]">AWAITING_QUERY_INPUT</p>
                </td>
              </tr>
            {:else}
              {#each results as evt (evt.id)}
                {@const isExpanded = expanded.has(evt.id)}
                {@const sev = getSev(evt.event_type)}
                <tr 
                  class="group hover:bg-surface-1 cursor-pointer transition-colors {isExpanded ? 'bg-surface-1' : ''}"
                  onclick={() => toggleRow(evt.id)}
                >
                  <td class="p-3 text-center">
                    {#if isExpanded}
                      <ChevronDown size={14} class="text-accent-primary" />
                    {:else}
                      <ChevronRight size={14} class="text-text-muted group-hover:text-accent-primary transition-colors" />
                    {/if}
                  </td>
                  <td class="p-3 text-[11px] font-mono text-text-muted">{evt.timestamp.replace('T', ' ').slice(0, 19)}</td>
                  <td class="p-3">
                    <span class="text-[11px] font-bold text-text-secondary uppercase">{evt.host_id}</span>
                  </td>
                  <td class="p-3">
                    <Badge variant={sev.variant} size="xs" class="font-black italic">{sev.label}</Badge>
                  </td>
                  <td class="p-3">
                    <div class="text-[11px] font-mono text-text-muted line-clamp-1 opacity-60 group-hover:opacity-100 transition-opacity">
                      {evt.raw_log}
                    </div>
                  </td>
                </tr>
                {#if isExpanded}
                  <tr class="bg-surface-2 shadow-inner">
                    <td colspan="5" class="p-6">
                      <div class="grid grid-cols-12 gap-8">
                        <!-- Metadata -->
                        <div class="col-span-4 space-y-4 border-r border-border-primary pr-8">
                          <div class="text-[10px] font-black text-text-heading uppercase tracking-widest border-b border-border-subtle pb-1 flex items-center gap-2">
                             <History size={12} class="text-accent-primary" />
                             Event Metadata
                          </div>
                          <div class="space-y-3">
                            {#each [['ID', evt.id], ['Tenant', evt.tenant_id || 'GLOBAL'], ['Source', evt.source_ip || '—'], ['User', evt.user || '—']] as [k, v]}
                              <div class="flex justify-between items-center text-[11px] font-mono">
                                <span class="text-text-muted uppercase tracking-tighter">{k}</span>
                                <span class="text-text-secondary font-bold">{v}</span>
                              </div>
                            {/each}
                          </div>
                        </div>
                        
                        <!-- Raw Log Expanded -->
                        <div class="col-span-8 space-y-4">
                          <div class="text-[10px] font-black text-text-heading uppercase tracking-widest border-b border-border-subtle pb-1 flex items-center gap-2">
                             <Terminal size={12} class="text-accent-primary" />
                             Raw Event Structure
                          </div>
                          <pre class="p-4 bg-surface-0 border border-border-primary rounded-sm text-[11px] font-mono text-accent-primary leading-relaxed whitespace-pre-wrap overflow-x-auto">
                            {evt.raw_log}
                          </pre>
                        </div>
                      </div>
                    </td>
                  </tr>
                {/if}
              {/each}
            {/if}
          </tbody>
        </table>
      </div>
    </div>

    <!-- FOOTER STATUS -->
    <div class="bg-surface-2 border-t border-border-primary px-4 py-1.5 flex items-center justify-between text-[8px] font-mono uppercase tracking-widest text-text-muted shrink-0">
      <div class="flex items-center gap-4">
        <div class="flex items-center gap-1.5">
          <div class="w-1 h-1 rounded-full bg-status-online"></div>
          <span>Indexing: Nominal</span>
        </div>
        <span>Plane: OQL_v4.2</span>
      </div>
      <div class="opacity-40 italic">Results limited to top {limit} events for tactical performance</div>
    </div>
  </div>
</PageLayout>

<style>
  :global(.overflow-auto::-webkit-scrollbar) {
    width: 6px;
    height: 6px;
  }
  :global(.overflow-auto::-webkit-scrollbar-track) {
    background: transparent;
  }
  :global(.overflow-auto::-webkit-scrollbar-thumb) {
    background: var(--border-primary);
    border-radius: 3px;
  }
</style>
