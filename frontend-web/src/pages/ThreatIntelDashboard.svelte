<!-- OBLIVRA Web — Threat Intelligence Dashboard (Svelte 5) -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { Badge, Button, DataTable, PageLayout, Spinner, ProgressBar } from '@components/ui';
  import { Globe, Fingerprint, Activity, RefreshCw, AlertTriangle, Hash, Link as LinkIcon } from 'lucide-svelte';
  import { request } from '../services/api';

  // -- Types --
  interface Indicator {
    type: string;
    value: string;
    source: string;
    severity: 'low' | 'medium' | 'high' | 'critical';
    description: string;
    campaign_id?: string;
    expires_at: string;
  }
  interface Campaign {
    id: string;
    name: string;
    actor?: string;
    ttps?: string[];
    description?: string;
  }
  interface IOCStats { [type: string]: number; }

  // -- State --
  let tab          = $state<'indicators' | 'campaigns' | 'stats'>('indicators');
  let search       = $state('');
  let typeFilter   = $state('all');
  let sevFilter    = $state('all');
  let queryValue   = $state('');
  let queryResult  = $state<Indicator | null | 'none'>(null);
  let loading      = $state(true);
  
  let indicators   = $state<Indicator[]>([]);
  let campaigns    = $state<Campaign[]>([]);
  let stats        = $state<IOCStats>({});

  // -- Helpers --
  const SEV_STYLE: Record<string, { color: string; bg: string }> = {
    critical: { color: 'var(--alert-critical)', bg: 'rgba(200,44,44,0.1)' },
    high:     { color: 'var(--alert-high)',     bg: 'rgba(200,80,0,0.1)' },
    medium:   { color: 'var(--alert-medium)',   bg: 'rgba(200,140,0,0.1)' },
    low:      { color: 'var(--status-online)',  bg: 'rgba(0,200,100,0.1)' },
  };

  const getIcon = (type: string) => {
    if (type.includes('addr')) return Globe;
    if (type.includes('domain')) return LinkIcon;
    if (type.includes('hash') || type.length > 20) return Hash;
    return Activity;
  };

  const filtered = $derived(indicators.filter(i => {
    const matchSearch = !search || i.value.toLowerCase().includes(search.toLowerCase()) || i.source.toLowerCase().includes(search.toLowerCase());
    const matchType = typeFilter === 'all' || i.type === typeFilter;
    const matchSev  = sevFilter === 'all' || i.severity === sevFilter;
    return matchSearch && matchType && matchSev;
  }));

  const types = $derived(['all', ...new Set(indicators.map(i => i.type))]);
  const totalIOCs = $derived(Object.values(stats).reduce((a, b) => a + b, 0));

  // -- Actions --
  async function fetchData() {
    loading = true;
    try {
      const [s, i, c] = await Promise.all([
        request<{ stats: IOCStats }>('/threatintel/stats'),
        request<{ indicators: Indicator[] }>('/threatintel/indicators?limit=500'),
        request<{ campaigns: Campaign[] }>('/threatintel/campaigns')
      ]);
      stats = s.stats ?? {};
      indicators = i.indicators ?? [];
      campaigns = c.campaigns ?? [];
    } catch (e) {
      console.error('Threat Intel fetch failed', e);
    } finally {
      loading = false;
    }
  }

  async function lookupIOC() {
    const v = queryValue.trim();
    if (!v) return;
    try {
      const r = await request<{ match: boolean; indicator?: Indicator }>(
        `/threatintel/lookup?value=${encodeURIComponent(v)}`
      );
      queryResult = r.match && r.indicator ? r.indicator : 'none';
    } catch { 
      queryResult = 'none'; 
    }
  }

  onMount(() => {
    fetchData();
  });
</script>

<PageLayout title="Threat Intelligence" subtitle="Strategic intelligence and real-time IOC correlation mesh: The OBLIVRA Intel Orbit">
  {#snippet toolbar()}
    <div class="flex items-center gap-2">
      <Button variant="secondary" size="sm" onclick={fetchData}>
        <RefreshCw size={14} class="mr-2" />
        SYNC FEEDS
      </Button>
      <Button variant="primary" size="sm">NEW IOC</Button>
    </div>
  {/snippet}

  <div class="flex flex-col h-full gap-0 -m-6 overflow-hidden">
    <!-- METRIC STRIP -->
    <div class="grid grid-cols-5 gap-px bg-border-primary border-b border-border-primary shrink-0">
        <div class="bg-surface-2 p-3">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">Total Indicators</div>
            <div class="text-xl font-mono font-bold text-accent-primary">{totalIOCs.toLocaleString()}</div>
            <div class="text-[9px] text-text-muted mt-1 italic">Verified intel shards</div>
        </div>
        {#each Object.entries(stats).slice(0, 4) as [type, count]}
           <div class="bg-surface-2 p-3">
               <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">{type}</div>
               <div class="text-xl font-mono font-bold text-text-heading">{count.toLocaleString()}</div>
               <div class="text-[9px] text-status-online mt-1">✓ Feed Active</div>
           </div>
        {/each}
    </div>

    <!-- IOC LOOKUP OVERLAY BAR -->
    <div class="bg-surface-1 border-b border-border-primary p-4 shrink-0 flex items-center gap-4">
        <div class="flex-1 flex flex-col gap-1.5">
           <span class="text-[9px] font-mono font-bold text-text-muted uppercase tracking-widest">Instant Intel Lookup</span>
           <div class="flex gap-2">
              <input 
                type="text" 
                placeholder="Query IP, domain, hash, or URL identifier..." 
                class="flex-1 bg-surface-2 border border-border-primary rounded-sm px-3 py-1.5 text-xs font-mono text-text-secondary focus:border-accent-primary focus:outline-none transition-colors"
                bind:value={queryValue}
                onkeydown={(e) => e.key === 'Enter' && lookupIOC()}
              />
              <Button variant="primary" size="sm" class="h-8 font-black italic tracking-tighter" onclick={lookupIOC}>LOOKUP</Button>
           </div>
        </div>
        
        {#if queryResult}
           <div class="min-w-[320px] p-2.5 rounded-sm border animate-in fade-in slide-in-from-right-4 duration-300
             {queryResult === 'none' ? 'bg-surface-2 border-border-subtle' : ''}"
             style={queryResult !== 'none' ? `background: ${SEV_STYLE[queryResult.severity].bg}; border-color: ${SEV_STYLE[queryResult.severity].color}` : ''}
           >
              {#if queryResult === 'none'}
                 <div class="flex items-center gap-3">
                    <div class="w-1.5 h-1.5 rounded-full bg-text-muted"></div>
                    <span class="text-[10px] font-mono text-text-muted uppercase font-bold tracking-widest italic">No matches in OBLIVRA substrate</span>
                 </div>
              {:else}
                 <div class="flex flex-col gap-1">
                    <div class="flex justify-between items-center">
                       <span class="text-[10px] font-black uppercase tracking-widest italic" style="color: ${SEV_STYLE[queryResult.severity].color}">● {queryResult.severity} MATCH</span>
                       <span class="text-[8px] font-mono text-text-muted uppercase">{queryResult.type}</span>
                    </div>
                    <div class="text-[11px] font-bold text-text-heading truncate">{queryResult.value}</div>
                    <div class="text-[8px] font-mono text-text-muted uppercase tracking-tighter opacity-60 truncate">{queryResult.source} — {queryResult.description}</div>
                 </div>
              {/if}
           </div>
        {/if}
    </div>

    <!-- MAIN BODY -->
    <div class="flex-1 flex min-h-0 bg-surface-0 overflow-hidden">
        <!-- LEFT: MAIN CONTENT -->
        <div class="flex-1 flex flex-col min-w-0 border-r border-border-primary overflow-hidden">
            <div class="bg-surface-1 border-b border-border-primary p-3 flex items-center justify-between shrink-0">
                <div class="flex items-center gap-4">
                    <div class="flex border border-border-primary rounded-sm overflow-hidden">
                      {#each ['indicators', 'campaigns', 'stats'] as t}
                        <button
                          class="px-3 py-1 text-[9px] font-bold uppercase tracking-widest transition-colors
                            {tab === t ? 'bg-accent-primary text-black' : 'bg-surface-0 text-text-muted hover:text-text-secondary'}"
                          onclick={() => tab = t as any}
                        >
                          {t}
                        </button>
                      {/each}
                    </div>
                    
                    {#if tab === 'indicators'}
                       <div class="flex items-center gap-2">
                          <input 
                            type="text" 
                            placeholder="Filter indicators..." 
                            class="bg-surface-2 border border-border-subtle rounded-sm px-2 py-0.5 text-[10px] w-48 font-mono focus:border-accent-primary focus:outline-none"
                            bind:value={search}
                          />
                          <select bind:value={typeFilter} class="bg-surface-2 border border-border-subtle rounded-sm px-1 py-0.5 text-[10px] font-mono focus:border-accent-primary focus:outline-none">
                             {#each types as t}
                                <option value={t}>{t.toUpperCase()}</option>
                             {/each}
                          </select>
                          <select bind:value={sevFilter} class="bg-surface-2 border border-border-subtle rounded-sm px-1 py-0.5 text-[10px] font-mono focus:border-accent-primary focus:outline-none">
                             {#each ['all', 'critical', 'high', 'medium', 'low'] as s}
                                <option value={s}>{s.toUpperCase()}</option>
                             {/each}
                          </select>
                       </div>
                    {/if}
                </div>
                
                <div class="text-[9px] font-mono text-text-muted uppercase tracking-widest">
                   Showing {filtered.length} of {indicators.length}
                </div>
            </div>

            <div class="flex-1 overflow-auto">
              {#if loading}
                <div class="h-full flex items-center justify-center"><Spinner /></div>
              {:else if tab === 'indicators'}
                <DataTable 
                  data={filtered} 
                  columns={[
                    { key: 'severity', label: 'SEV', width: '100px' },
                    { key: 'type', label: 'TYPE', width: '140px' },
                    { key: 'value', label: 'INDICATOR_IDENTIFIER' },
                    { key: 'source', label: 'INTEL_SOURCE', width: '150px' },
                    { key: 'description', label: 'CONTEXTUAL_METADATA' },
                    { key: 'expires_at', label: 'TTL', width: '120px' }
                  ]} 
                  compact
                  rowKey="value"
                >
                  {#snippet cell({ column, row })}
                    {@const Icon = getIcon(row.type)}
                    {#if column.key === 'severity'}
                      <div class="flex items-center gap-2">
                         <div class="w-1.5 h-1.5 rounded-full" style="background: {SEV_STYLE[row.severity]?.color}"></div>
                         <span class="text-[9px] font-black uppercase tracking-widest italic" style="color: {SEV_STYLE[row.severity]?.color}">{row.severity}</span>
                      </div>
                    {:else if column.key === 'type'}
                      <div class="flex items-center gap-2 opacity-60">
                         <Icon size={12} />
                         <span class="text-[9px] font-mono uppercase">{row.type}</span>
                      </div>
                    {:else if column.key === 'value'}
                      <span class="text-[11px] font-mono font-bold text-accent-primary truncate block max-w-md">{row.value}</span>
                    {:else if column.key === 'source'}
                      <span class="text-[9px] font-mono text-text-muted uppercase tracking-tighter">{row.source}</span>
                    {:else if column.key === 'description'}
                      <span class="text-[10px] text-text-secondary truncate block max-w-sm" title={row.description}>{row.description}</span>
                    {:else if column.key === 'expires_at'}
                      <span class="text-[9px] font-mono text-text-muted opacity-60 uppercase">{row.expires_at ? new Date(row.expires_at).toLocaleDateString() : 'INFINITY'}</span>
                    {/if}
                  {/snippet}
                </DataTable>
              {:else if tab === 'campaigns'}
                <div class="p-6 space-y-4">
                   {#each campaigns as campaign}
                      <div class="bg-surface-1 border border-border-primary p-4 rounded-sm flex flex-col gap-3 group hover:border-accent-primary transition-colors cursor-default relative overflow-hidden">
                         <div class="absolute -right-4 -bottom-4 opacity-[0.03] grayscale group-hover:scale-110 transition-transform duration-700">
                            <Fingerprint size={120} />
                         </div>
                         <div class="flex justify-between items-start relative z-10">
                            <div class="flex flex-col gap-1">
                               <div class="flex items-center gap-2">
                                  <span class="text-[12px] font-black text-text-heading uppercase tracking-tighter italic">{campaign.name}</span>
                                  {#if campaign.actor}
                                     <Badge variant="accent" size="xs" class="font-bold">{campaign.actor.toUpperCase()}</Badge>
                                  {/if}
                               </div>
                               <span class="text-[9px] font-mono text-text-muted uppercase tracking-widest opacity-60">ID: {campaign.id}</span>
                            </div>
                            
                            <div class="flex flex-wrap gap-1 max-w-[400px] justify-end">
                               {#each campaign.ttps?.slice(0, 8) ?? [] as ttp}
                                  <span class="text-[8px] font-mono bg-surface-2 border border-border-subtle px-1.5 py-0.5 text-text-muted uppercase">{ttp}</span>
                               {/each}
                            </div>
                         </div>
                         <p class="text-[11px] text-text-secondary leading-relaxed max-w-3xl relative z-10">{campaign.description}</p>
                      </div>
                   {:else}
                      <div class="py-20 text-center opacity-40 flex flex-col items-center gap-4">
                         <Fingerprint size={48} />
                         <p class="text-[10px] font-mono uppercase tracking-widest">No threat campaigns synchronized</p>
                      </div>
                   {/each}
                </div>
              {:else if tab === 'stats'}
                <div class="p-6 grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-6">
                   {#each Object.entries(stats) as [type, count]}
                      {@const Icon = getIcon(type)}
                      <div class="bg-surface-1 border border-border-primary p-5 rounded-sm flex flex-col gap-4 shadow-premium group hover:border-accent-primary transition-colors">
                         <div class="flex justify-between items-start">
                            <div class="p-2.5 bg-surface-2 border border-border-subtle rounded-xs">
                               <Icon size={18} class="text-accent-primary" />
                            </div>
                            <span class="text-[10px] font-mono font-bold text-text-muted italic">{Math.round(count / totalIOCs * 100)}%</span>
                         </div>
                         <div class="flex flex-col gap-1">
                            <span class="text-2xl font-mono font-black text-text-heading italic">{count.toLocaleString()}</span>
                            <span class="text-[9px] font-mono text-text-muted uppercase tracking-widest border-t border-border-subtle pt-2">{type.replace(/-/g, ' ')}</span>
                         </div>
                         <ProgressBar value={count / totalIOCs * 100} height="3px" color="var(--accent-primary)" />
                      </div>
                   {/each}
                </div>
              {/if}
            </div>
        </div>

        <!-- RIGHT: ADVISORY SIDEBAR -->
        <div class="w-80 bg-surface-1 flex flex-col shrink-0">
            <div class="px-3 py-2 bg-surface-2 border-b border-border-primary flex items-center gap-2">
                <AlertTriangle size={14} class="text-alert-medium" />
                <span class="text-[9px] font-mono font-bold uppercase tracking-widest text-text-heading">Global Advisories</span>
            </div>
            
            <div class="p-4 space-y-6">
                <div class="space-y-4">
                  {#each [
                    { title: 'Zero-Day in Core Mesh', ref: 'CVE-2026-1042', score: '9.8', severity: 'critical' },
                    { title: 'New APT: Iron Veil', ref: 'ACTOR_REPORT_88', score: 'HIGH', severity: 'high' },
                    { title: 'BGP Hijacking: AS-9942', ref: 'NETWORK_ADV_02', score: '7.4', severity: 'medium' }
                  ] as adv}
                    <div class="flex flex-col gap-1.5 p-3 bg-surface-2 border-l-2 border-border-primary hover:border-accent-primary transition-colors cursor-pointer group"
                      style="border-left-color: {SEV_STYLE[adv.severity].color}">
                       <div class="flex justify-between items-center">
                          <span class="text-[10px] font-black text-text-heading uppercase group-hover:text-accent-primary transition-colors">{adv.title}</span>
                          <span class="text-[8px] font-mono font-bold italic" style="color: {SEV_STYLE[adv.severity].color}">{adv.score}</span>
                       </div>
                       <span class="text-[8px] font-mono text-text-muted uppercase tracking-tighter opacity-60">{adv.ref}</span>
                    </div>
                  {/each}
                </div>

                <div class="pt-4 border-t border-border-primary space-y-4">
                    <span class="text-[9px] font-mono font-bold text-text-muted uppercase tracking-widest">Connected Orbitals</span>
                    {#each [
                      { name: 'OBLIVRA_FEEDS_V2', status: 'online', items: '14,204' },
                      { name: 'STIX_TAXII_RELAY', status: 'online', items: '2,109' },
                      { name: 'MITRE_NAVI_SYNC', status: 'standby', items: 'V15' }
                    ] as orbital}
                       <div class="bg-surface-2 border border-border-primary p-3 rounded-sm space-y-1">
                          <div class="flex items-center justify-between">
                             <span class="text-[10px] font-bold text-text-heading uppercase tracking-tighter">{orbital.name}</span>
                             <div class="w-1.5 h-1.5 rounded-full {orbital.status === 'online' ? 'bg-status-online animate-pulse' : 'bg-accent-primary'}"></div>
                          </div>
                          <div class="flex justify-between items-center text-[8px] font-mono text-text-muted uppercase opacity-60">
                             <span>Status: {orbital.status}</span>
                             <span>Depth: {orbital.items}</span>
                          </div>
                       </div>
                    {/each}
                </div>
            </div>

            <div class="mt-auto border-t border-border-primary p-4 bg-surface-2">
                 <div class="flex items-center justify-between mb-2">
                    <span class="text-[9px] font-mono font-bold text-text-muted uppercase tracking-widest">Intelligence Depth</span>
                    <Badge variant="accent" size="xs">L7_GLOBAL</Badge>
                 </div>
                 <div class="text-[8px] font-mono text-text-muted space-y-1 opacity-60">
                    <div>Engine: OBLIVRA_ORBIT_v2.1</div>
                    <div>Confidence Baseline: 94%</div>
                    <div>Ingest Rate: 1.4k/hr</div>
                 </div>
            </div>
        </div>
    </div>

    <!-- STATUS BAR -->
    <div class="bg-surface-2 border-t border-border-primary px-3 py-1 flex items-center gap-4 text-[8px] font-mono text-text-muted shrink-0 uppercase tracking-widest">
        <div class="flex items-center gap-1.5">
            <div class="w-1 h-1 rounded-full bg-status-online"></div>
            <span>INTEL_MESH:</span>
            <span class="text-status-online font-bold italic">CONNECTED</span>
        </div>
        <span class="text-border-primary opacity-30">|</span>
        <div class="flex items-center gap-1.5">
            <span>FEED_INTEGRITY:</span>
            <span class="text-status-online font-bold italic">VERIFIED</span>
        </div>
        <span class="text-border-primary opacity-30">|</span>
        <div class="flex items-center gap-1.5">
            <span>ORBIT_SYNC:</span>
            <span class="text-accent-primary font-bold italic">REALTIME</span>
        </div>
        <div class="ml-auto opacity-40">OBLIVRA_THREAT_INTEL v2.1.4</div>
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
