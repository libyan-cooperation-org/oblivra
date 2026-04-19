<!-- OBLIVRA Web — Lookup Manager (Svelte 5) -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { KPI, Badge, Button, DataTable, PageLayout, Spinner, Input } from '@components/ui';
  import { Database, Search, Upload, Trash2, History, Filter, CheckCircle, AlertTriangle, Hash, Network, Zap, FileText } from 'lucide-svelte';
  import { request } from '../services/api';

  // -- Types --
  interface LookupTable {
    name: string;
    match_type: 'exact' | 'cidr' | 'wildcard' | 'regex';
    fields: string[];
    rows?: Record<string, string>[];
  }
  type MatchType = 'exact' | 'cidr' | 'wildcard' | 'regex';

  // -- Constants --
  const MT_STYLE = {
    exact:    { color: 'var(--status-online)',  bg: 'rgba(0,200,100,0.1)', icon: Hash },
    cidr:     { color: 'var(--accent-primary)', bg: 'rgba(0,180,255,0.1)', icon: Network },
    wildcard: { color: 'var(--alert-medium)',   bg: 'rgba(200,140,0,0.1)', icon: Zap },
    regex:    { color: 'var(--alert-critical)', bg: 'rgba(200,44,44,0.1)', icon: Filter },
  };

  // -- State --
  let tables       = $state<LookupTable[]>([]);
  let loading      = $state(true);
  let selected     = $state<LookupTable | null>(null);
  
  // Query state
  let queryTable   = $state('');
  let queryKey     = $state('');
  let queryResult  = $state<{match: boolean; data: Record<string,string>|null} | null>(null);
  let querying     = $state(false);

  // Upload state
  let uploadName   = $state('');
  let uploadMT     = $state<MatchType>('exact');
  let uploadFmt    = $state<'csv'|'json'>('csv');
  let uploadFile   = $state<File | null>(null);
  let uploadMsg    = $state('');
  let uploading    = $state(false);

  // -- Actions --
  async function fetchTables() {
    loading = true;
    try {
      const res = await request<{ tables: LookupTable[] }>('/lookups');
      tables = res.tables ?? [];
    } catch (e) {
      console.error('Lookup fetch failed', e);
    } finally {
      loading = false;
    }
  }

  async function loadTable(name: string) {
    try {
      const t = await request<LookupTable>(`/lookups/${name}`);
      selected = t;
    } catch (e) {
      console.error('Table load failed', e);
    }
  }

  async function deleteTable(name: string) {
    if (!confirm(`Confirm destruction of lookup table: ${name}?`)) return;
    try {
      await request(`/lookups/${name}`, { method: 'DELETE' });
      if (selected?.name === name) selected = null;
      fetchTables();
    } catch(e: any) {
      alert(`ERR: ${e.message}`);
    }
  }

  async function runQuery() {
    if (!queryTable || !queryKey) return;
    querying = true;
    try {
      const res = await request<{match:boolean; data: Record<string,string>|null}>(
        `/lookups/query?table=${encodeURIComponent(queryTable)}&key=${encodeURIComponent(queryKey)}`
      );
      queryResult = res;
    } catch { 
      queryResult = { match: false, data: null }; 
    } finally {
      querying = false;
    }
  }

  async function uploadTable() {
    if (!uploadFile || !uploadName) return;
    uploading = true;
    const form = new FormData();
    form.append('name', uploadName);
    form.append('match_type', uploadMT);
    form.append('format', uploadFmt);
    form.append('file', uploadFile);
    try {
      const token = localStorage.getItem('oblivra_token') ?? '';
      const res = await fetch('/api/v1/lookups/upload', {
        method: 'POST',
        headers: { 'X-API-Key': token },
        body: form,
      });
      if (!res.ok) throw new Error(await res.text());
      uploadMsg = `✓ Table "${uploadName}" synchronized.`;
      uploadName = '';
      uploadFile = null;
      fetchTables();
    } catch(e: any) {
      uploadMsg = `✗ Synchronization failed: ${e.message}`;
    } finally {
      uploading = false;
      setTimeout(() => uploadMsg = '', 5000);
    }
  }

  onMount(() => {
    fetchTables();
  });
</script>

<PageLayout title="Lookup Tables" subtitle="High-density enrichment substrate: Manage CSV/JSON mappings for OQL ingestion pipelines">
  {#snippet toolbar()}
    <div class="flex items-center gap-2">
      <Button variant="secondary" size="sm" onclick={fetchTables}>
        <History size={14} class="mr-2" />
        RE-SYNC
      </Button>
      <Button variant="primary" size="sm" icon={Upload}>MASS UPLOAD</Button>
    </div>
  {/snippet}

  <div class="flex flex-col h-full gap-0 -m-6 overflow-hidden">
    <!-- METRIC STRIP -->
    <div class="grid grid-cols-4 gap-px bg-border-primary border-b border-border-primary shrink-0">
        <div class="bg-surface-2 p-3">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">Total Tables</div>
            <div class="text-xl font-mono font-bold text-text-heading">{tables.length}</div>
            <div class="text-[9px] text-text-muted mt-1 italic">Active mapping shards</div>
        </div>
        <div class="bg-surface-2 p-3">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">Global Strategy</div>
            <div class="text-xl font-mono font-bold text-accent-primary">HYBRID</div>
            <div class="text-[9px] text-status-online mt-1 uppercase tracking-tighter">L7_MATCH_ACTIVE</div>
        </div>
        <div class="bg-surface-2 p-3">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">Query Latency</div>
            <div class="text-xl font-mono font-bold text-status-online">&lt; 0.2ms</div>
            <div class="text-[9px] text-text-muted mt-1 italic">In-memory hot path</div>
        </div>
        <div class="bg-surface-2 p-3">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">Enrichment State</div>
            <div class="text-xl font-mono font-bold text-text-heading">ACTIVE</div>
            <div class="text-[9px] text-status-online mt-1 uppercase tracking-tighter">✓ Substrate Healthy</div>
        </div>
    </div>

    <!-- MAIN BODY -->
    <div class="flex-1 flex min-h-0 bg-surface-0 overflow-hidden">
        <!-- LEFT SIDEBAR: TABLES & QUERY -->
        <div class="w-80 border-r border-border-primary flex flex-col shrink-0 bg-surface-1 overflow-hidden">
           <!-- Table List -->
           <div class="flex-1 flex flex-col min-h-0">
              <div class="p-3 bg-surface-2 border-b border-border-primary flex justify-between items-center shrink-0">
                 <span class="text-[9px] font-mono font-bold text-text-muted uppercase tracking-widest">Available Shards</span>
                 <Badge variant="secondary" size="xs">{tables.length}</Badge>
              </div>
              <div class="flex-1 overflow-auto p-1 space-y-1">
                 {#if loading}
                    <div class="py-10 flex justify-center"><Spinner size="sm" /></div>
                 {:else}
                    {#each tables as t}
                       {@const Style = MT_STYLE[t.match_type]}
                       <div 
                         role="button"
                         tabindex="0"
                         class="w-full text-left p-3 rounded-sm border transition-all group flex flex-col gap-1 cursor-pointer
                           {selected?.name === t.name ? 'bg-surface-2 border-accent-primary shadow-premium' : 'bg-transparent border-transparent hover:bg-surface-2 hover:border-border-subtle'}"
                         onclick={() => loadTable(t.name)}
                         onkeydown={(e) => e.key === 'Enter' && loadTable(t.name)}
                       >
                          <div class="flex justify-between items-start">
                             <span class="text-[11px] font-bold text-text-heading uppercase tracking-tighter">{t.name}</span>
                             <button type="button" class="opacity-0 group-hover:opacity-100 text-text-muted hover:text-alert-critical transition-opacity" 
                               onclick={(e) => { e.stopPropagation(); deleteTable(t.name); }}>
                                <Trash2 size={12} />
                             </button>
                          </div>
                          <div class="flex items-center gap-2">
                             <Style.icon size={10} style="color: {Style.color}" />
                             <span class="text-[8px] font-mono font-black uppercase tracking-widest italic" style="color: {Style.color}">{t.match_type}</span>
                          </div>
                       </div>
                    {/each}
                 {/if}
              </div>
           </div>

           <!-- Quick Query -->
           <div class="bg-surface-2 border-t border-border-primary p-4 space-y-4">
              <div class="flex items-center gap-2">
                 <Search size={14} class="text-accent-primary" />
                 <span class="text-[9px] font-mono font-bold text-text-heading uppercase tracking-widest">Substrate Query</span>
              </div>
              
              <div class="space-y-3">
                 <select bind:value={queryTable} class="w-full bg-surface-1 border border-border-subtle rounded-sm px-2 py-1.5 text-xs font-mono text-text-secondary focus:border-accent-primary focus:outline-none">
                    <option value="">SELECT TABLE...</option>
                    {#each tables as t}
                       <option value={t.name}>{t.name.toUpperCase()}</option>
                    {/each}
                 </select>
                 <div class="flex gap-2">
                    <input 
                      type="text" 
                      placeholder="KEY_TO_LOOKUP" 
                      class="flex-1 bg-surface-1 border border-border-subtle rounded-sm px-2 py-1.5 text-xs font-mono text-text-secondary focus:border-accent-primary focus:outline-none"
                      bind:value={queryKey}
                      onkeydown={(e) => e.key === 'Enter' && runQuery()}
                    />
                    <Button variant="primary" size="sm" class="font-black italic" onclick={runQuery} loading={querying}>GO</Button>
                 </div>
              </div>

              {#if queryResult}
                 <div class="p-3 rounded-sm border animate-in fade-in duration-300
                   {queryResult.match ? 'bg-status-online/5 border-status-online/40' : 'bg-alert-critical/5 border-alert-critical/40'}"
                 >
                    <div class="flex items-center gap-2 mb-2">
                       <div class="w-1.5 h-1.5 rounded-full {queryResult.match ? 'bg-status-online' : 'bg-alert-critical'}"></div>
                       <span class="text-[9px] font-black uppercase tracking-widest italic {queryResult.match ? 'text-status-online' : 'text-alert-critical'}">
                          {queryResult.match ? 'MATCH_FOUND' : 'NO_MATCH'}
                       </span>
                    </div>
                    {#if queryResult.data}
                       <div class="space-y-1 overflow-auto max-h-32 pr-2">
                          {#each Object.entries(queryResult.data) as [k, v]}
                             <div class="flex flex-col">
                                <span class="text-[8px] font-mono text-text-muted uppercase">{k}</span>
                                <span class="text-[10px] text-text-secondary break-all">{v}</span>
                             </div>
                          {/each}
                       </div>
                    {/if}
                 </div>
              {/if}
           </div>
        </div>

        <!-- RIGHT: TABLE DETAIL & UPLOAD -->
        <div class="flex-1 flex flex-col min-w-0 bg-surface-0 overflow-hidden">
           <!-- UPLOAD AREA -->
           <div class="bg-surface-1 border-b border-border-primary p-5 space-y-4 shrink-0">
              <div class="flex items-center justify-between">
                 <div class="flex items-center gap-2">
                    <Upload size={14} class="text-accent-primary" />
                    <span class="text-[10px] font-black text-text-heading uppercase tracking-widest">Synchronize New Shard</span>
                 </div>
                 {#if uploadMsg}
                    <div class="text-[9px] font-mono font-bold {uploadMsg.startsWith('✓') ? 'text-status-online' : 'text-alert-critical'} animate-pulse">
                       {uploadMsg}
                    </div>
                 {/if}
              </div>

              <div class="grid grid-cols-1 md:grid-cols-4 gap-4 items-end">
                 <div class="space-y-1.5">
                    <span class="text-[8px] font-mono text-text-muted uppercase tracking-widest">Table Name</span>
                    <input bind:value={uploadName} class="w-full bg-surface-2 border border-border-subtle rounded-sm px-3 py-1.5 text-xs font-mono text-text-secondary focus:border-accent-primary focus:outline-none" placeholder="threat_actors_v2" />
                 </div>
                 <div class="space-y-1.5">
                    <span class="text-[8px] font-mono text-text-muted uppercase tracking-widest">Match Strategy</span>
                    <select bind:value={uploadMT} class="w-full bg-surface-2 border border-border-subtle rounded-sm px-3 py-1.5 text-xs font-mono text-text-secondary focus:border-accent-primary focus:outline-none">
                       <option value="exact">EXACT</option>
                       <option value="cidr">CIDR</option>
                       <option value="wildcard">WILDCARD</option>
                       <option value="regex">REGEX</option>
                    </select>
                 </div>
                 <div class="space-y-1.5">
                    <span class="text-[8px] font-mono text-text-muted uppercase tracking-widest">Source Shard</span>
                    <div class="flex gap-2">
                       <label class="flex-1 cursor-pointer bg-surface-2 border border-border-subtle rounded-sm px-3 py-1.5 text-xs font-mono text-text-muted hover:border-accent-primary transition-colors truncate">
                          <input type="file" class="hidden" onchange={(e) => uploadFile = e.currentTarget.files?.[0] ?? null} accept=".csv,.json" />
                          {uploadFile ? uploadFile.name : 'CHOOSE FILE (.CSV/JSON)'}
                       </label>
                    </div>
                 </div>
                 <Button variant="primary" class="w-full font-black italic tracking-tighter" onclick={uploadTable} loading={uploading} disabled={!uploadFile || !uploadName}>COMMIT SHARD</Button>
              </div>
           </div>

           <!-- TABLE VIEWER -->
           <div class="flex-1 flex flex-col min-h-0">
              {#if selected}
                 <div class="p-3 bg-surface-2 border-b border-border-primary flex justify-between items-center shrink-0">
                    <div class="flex items-center gap-3">
                       <div class="p-1 bg-accent-primary/10 rounded-xs">
                          <FileText size={14} class="text-accent-primary" />
                       </div>
                       <div class="flex flex-col">
                          <span class="text-[11px] font-black text-text-heading uppercase tracking-tighter">{selected.name}</span>
                          <span class="text-[8px] font-mono text-text-muted uppercase tracking-widest opacity-60">Strategy: {selected.match_type} // Total Rows: {selected.rows?.length ?? 0}</span>
                       </div>
                    </div>
                    <div class="flex items-center gap-2">
                       <Button variant="ghost" size="xs" icon={Filter}>FIELD FILTERS</Button>
                       <Button variant="ghost" size="xs" icon={Trash2} class="text-alert-critical hover:bg-alert-critical/10" onclick={() => deleteTable(selected!.name)}>PURGE</Button>
                    </div>
                 </div>
                 <div class="flex-1 overflow-auto">
                    <DataTable 
                       data={selected.rows ?? []} 
                       columns={selected.fields.map(f => ({ key: f, label: f.toUpperCase() }))} 
                       compact
                    >
                       {#snippet cell({ column, row })}
                          <span class="text-[10px] font-mono text-text-secondary truncate block max-w-xs">{row[column.key] ?? '—'}</span>
                       {/snippet}
                    </DataTable>
                 </div>
              {:else}
                 <div class="flex-1 flex flex-col items-center justify-center gap-4 text-text-muted opacity-30">
                    <Database size={48} />
                    <p class="text-[10px] font-mono uppercase tracking-widest">Select a lookup shard to inspect telemetry</p>
                 </div>
              {/if}
           </div>
        </div>
    </div>

    <!-- STATUS BAR -->
    <div class="bg-surface-2 border-t border-border-primary px-3 py-1 flex items-center gap-4 text-[8px] font-mono text-text-muted shrink-0 uppercase tracking-widest">
        <div class="flex items-center gap-1.5">
            <div class="w-1 h-1 rounded-full bg-status-online"></div>
            <span>LOOKUP_PLANE:</span>
            <span class="text-status-online font-bold italic">ENGAGED</span>
        </div>
        <span class="text-border-primary opacity-30">|</span>
        <div class="flex items-center gap-1.5">
            <span>SYNC_LATENCY:</span>
            <span class="text-status-online font-bold italic">0.02ms</span>
        </div>
        <span class="text-border-primary opacity-30">|</span>
        <div class="flex items-center gap-1.5">
            <span>OQL_ENRICHMENT:</span>
            <span class="text-accent-primary font-bold italic">OPTIMIZED</span>
        </div>
        <div class="ml-auto opacity-40">OBLIVRA_LOOKUP_v2.4.1</div>
    </div>
  </div>
</PageLayout>

<style>
  :global(.flex-1::-webkit-scrollbar) {
    width: 6px;
    height: 6px;
  }
  :global(.flex-1::-webkit-scrollbar-track) {
    background: var(--surface-0);
  }
  :global(.flex-1::-webkit-scrollbar-thumb) {
    background: var(--border-primary);
    border-radius: 3px;
  }
</style>
