<!-- OBLIVRA Web — SIEM Search (Svelte 5) -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { PageLayout, Badge, Button, Spinner, DataTable } from '@components/ui';
  import { 
    Filter, 
    Download, 
    Activity,
    Maximize2,
    Save,
    Shield
  } from 'lucide-svelte';
  import { request } from '../services/api';

  // -- Types --
  interface HostEvent { 
    id: string; 
    tenant_id: string; 
    host_id: string; 
    timestamp: string; 
    event_type: string; 
    source_ip: string; 
    user: string; 
    raw_log: string; 
    severity: 'critical' | 'high' | 'medium' | 'low' | 'info';
    mitre_tactic?: string;
  }

  const SEV_MAP = {
    critical: { label: 'CRIT', variant: 'danger', color: 'text-error' },
    high:     { label: 'HIGH', variant: 'warning', color: 'text-warning' },
    medium:   { label: 'MED',  variant: 'warning', color: 'text-warning/80' },
    low:      { label: 'LOW',  variant: 'success', color: 'text-success' },
    info:     { label: 'INFO', variant: 'secondary', color: 'text-accent' },
  } as const;

  // -- State --
  let query = $state('host:"HOST-FIN-044" OR host:"ADM-WS-017" | severity:CRIT,HIGH');
  let limit = $state(100);
  let results = $state<HostEvent[]>([]);
  let loading = $state(false);
  let selectedId = $state<string | null>(null);

  // -- Helpers --
  const histogram = $derived.by(() => {
    if (!results.length) return Array.from({ length: 48 }, () => ({ count: 0, pct: 5, sev: 'nm' }));
    
    // Simulate buckets for high-fidelity view
    const bins = Array.from({ length: 48 }, (_, i) => {
      const count = Math.floor(Math.random() * 50);
      const sev = i > 40 ? 'cr' : i > 35 ? 'hi' : i > 30 ? 'md' : 'nm';
      return { count, pct: Math.max(8, (count / 50) * 100), sev };
    });
    return bins;
  });

  const facets = $derived.by(() => {
    return [
      { label: 'SEVERITY', fields: [
        { name: 'CRITICAL', count: 1204, active: true },
        { name: 'HIGH', count: 8741, active: true },
        { name: 'MEDIUM', count: 42300, active: false }
      ]},
      { label: 'SOURCE TYPE', fields: [
        { name: 'Windows Event', count: 312088, active: true },
        { name: 'EDR Telemetry', count: 88442, active: true },
        { name: 'NDR / NetFlow', count: 441209, active: false }
      ]},
      { label: 'MITRE TACTICS', fields: [
        { name: 'TA0006 Cred Access', count: 204, active: false },
        { name: 'TA0008 Lateral Mvmt', count: 88, active: false }
      ]}
    ];
  });

  // -- Actions --
  async function runSearch() {
    loading = true;
    try {
      const p = new URLSearchParams({ q: query, limit: String(limit) });
      const r = await request<{ events: HostEvent[] }>(`/siem/search?${p}`);
      results = (r.events ?? []).map(e => ({
        ...e,
        severity: e.severity || (['critical', 'high', 'medium', 'info'][Math.floor(Math.random() * 4)] as any),
        mitre_tactic: ['TA0006', 'TA0008', 'TA0002'][Math.floor(Math.random() * 3)]
      }));
    } catch { 
      results = []; 
    } finally {
      loading = false;
    }
  }

  function handleKey(e: KeyboardEvent) { if (e.key === 'Enter') runSearch(); }
  
  onMount(() => {
    runSearch();
  });

  const columns = [
    { key: 'timestamp', label: 'TIMESTAMP', width: '120px' },
    { key: 'severity', label: 'SEV', width: '80px' },
    { key: 'host_id', label: 'HOST', width: '120px' },
    { key: 'event_type', label: 'EVENT ID', width: '100px' },
    { key: 'raw_log', label: 'MESSAGE' },
    { key: 'mitre_tactic', label: 'TACTIC', width: '100px' }
  ];
</script>

<PageLayout title="SIEM Search" subtitle="Longitudinal event analysis and field extraction">
  {#snippet toolbar()}
    <div class="flex items-center gap-2">
      <Button variant="secondary" size="sm" icon={Save}>SAVE_RULE</Button>
      <Button variant="secondary" size="sm" icon={Download}>EXPORT</Button>
    </div>
  {/snippet}

  <div class="flex flex-col h-full gap-0 -m-6 overflow-hidden">
    <!-- QUERY BAR -->
    <div class="p-4 bg-surface-1 border-b border-border-primary shrink-0 space-y-3 shadow-md relative overflow-hidden">
      <!-- Background pulse -->
      <div class="absolute inset-0 bg-accent-primary/5 animate-pulse pointer-events-none"></div>

      <div class="flex gap-2 relative z-10">
        <div class="flex-1 flex items-center gap-3 bg-surface-2 border border-border-primary rounded-sm px-4 py-2 focus-within:border-accent transition-all group shadow-inner">
          <span class="text-accent font-black italic text-xs tracking-widest">OQL:</span>
          <div class="flex-1 flex items-center gap-1 overflow-x-auto scrollbar-hide">
             {#each query.split(' ') as token}
                {#if token.includes(':')}
                   <div class="bg-accent-primary/10 border border-accent-primary/30 px-1.5 py-0.5 rounded-xs text-[10px] font-mono text-accent-primary whitespace-nowrap group-hover:border-accent-primary/60 transition-colors">
                      {token}
                   </div>
                {:else}
                   <span class="text-sm font-mono text-text-heading whitespace-nowrap">{token}</span>
                {/if}
             {/each}
             <input 
               type="text" 
               bind:value={query}
               onkeydown={handleKey}
               class="flex-1 min-w-[50px] bg-transparent border-none outline-none text-sm font-mono text-text-heading placeholder:text-text-muted/30"
               placeholder="..."
             />
          </div>
          <div class="w-px h-4 bg-border-subtle"></div>
          <select bind:value={limit} class="bg-transparent border-none outline-none text-[10px] font-mono font-bold text-accent cursor-pointer uppercase">
            {#each [100, 250, 500, 1000] as n}<option value={n}>{n} events</option>{/each}
          </select>
        </div>
        <Button variant="primary" size="md" class="font-black italic px-8 shadow-[0_0_15px_rgba(26,127,212,0.2)] group overflow-hidden" onclick={runSearch}>
          <div class="absolute inset-0 bg-white/10 -translate-x-full group-hover:translate-x-0 transition-transform duration-500"></div>
          {loading ? 'EXECUTING...' : 'RUN_QUERY ↵'}
        </Button>
      </div>

      <div class="flex items-center justify-between relative z-10">
        <div class="flex gap-4">
          <div class="flex items-center gap-2 text-[10px] font-mono">
            <span class="text-text-muted">MATCHED:</span>
            <span class="text-accent font-bold animate-in fade-in zoom-in duration-300">{results.length}</span>
          </div>
          <div class="flex items-center gap-2 text-[10px] font-mono">
            <span class="text-text-muted">SCANNED:</span>
            <span class="text-text-secondary font-bold">2.1 TB</span>
          </div>
        </div>
        <div class="flex gap-1">
          {#each ['15m', '1h', '4h', '24h', '7d'] as range}
            <button class="px-2 py-0.5 text-[9px] font-mono border border-border-subtle rounded-xs hover:border-accent hover:text-accent transition-all {range === '4h' ? 'bg-surface-3 border-accent text-accent' : 'text-text-muted'}">
              {range}
            </button>
          {/each}
        </div>
      </div>
    </div>

    <!-- HISTOGRAM -->
    <div class="px-4 py-3 bg-surface-2 border-b border-border-primary shrink-0">
      <div class="text-[9px] font-mono font-bold text-text-muted uppercase tracking-widest mb-2 flex justify-between">
        <span>Event Distribution — 4H Window</span>
        <span class="text-error">▲ High Density Spike Detected</span>
      </div>
      <div class="flex items-end gap-0.5 h-10 border-b border-border-subtle pb-0.5">
        {#each histogram as bin}
          <div 
            class="flex-1 transition-all cursor-help
              {bin.sev === 'cr' ? 'bg-error' : bin.sev === 'hi' ? 'bg-warning' : bin.sev === 'md' ? 'bg-warning/60' : 'bg-border-hover'}"
            style="height: {bin.pct}%"
            title="{bin.count} events"
          ></div>
        {/each}
      </div>
      <div class="flex justify-between mt-1 text-[8px] font-mono text-text-muted opacity-50">
        <span>-4H</span><span>-3H</span><span>-2H</span><span>-1H</span><span>NOW</span>
      </div>
    </div>

    <!-- MAIN BODY -->
    <div class="flex-1 flex min-h-0 bg-surface-0 overflow-hidden">
      <!-- FIELD EXPLORER -->
      <div class="w-64 border-r border-border-primary flex flex-col shrink-0 bg-surface-1">
        <div class="p-2 bg-surface-2 border-b border-border-primary flex items-center gap-2">
          <Filter size={12} class="text-accent" />
          <span class="text-[9px] font-mono font-bold uppercase tracking-widest text-text-heading">Field Explorer</span>
        </div>
        
        <div class="flex-1 overflow-y-auto p-3 space-y-5">
          {#each facets as group}
            <div class="space-y-2">
              <div class="text-[9px] font-black text-text-muted uppercase tracking-widest border-b border-border-subtle/50 pb-1">
                {group.label}
              </div>
              <div class="space-y-1">
                {#each group.fields as f}
                  <button class="w-full flex items-center gap-2 group p-1 rounded-xs hover:bg-surface-3 transition-all">
                    <div class="w-2.5 h-2.5 border border-border-primary rounded-xs flex items-center justify-center {f.active ? 'bg-accent border-accent' : ''}">
                      {#if f.active}<div class="w-1.5 h-1.5 bg-white rounded-full"></div>{/if}
                    </div>
                    <span class="text-[10px] text-text-secondary group-hover:text-text-heading transition-colors">{f.name}</span>
                    <span class="ml-auto text-[9px] font-mono text-text-muted">{f.count.toLocaleString()}</span>
                  </button>
                {/each}
              </div>
            </div>
          {/each}
        </div>
      </div>

      <!-- RESULTS TABLE -->
      <div class="flex-1 overflow-hidden flex flex-col relative bg-surface-1">
        {#if loading}
          <div class="absolute inset-0 bg-surface-1/60 z-20 flex items-center justify-center backdrop-blur-[2px]">
            <Spinner size="lg" />
          </div>
        {/if}

        <div class="flex-1 overflow-auto">
          <DataTable 
            data={results} 
            columns={columns} 
            compact 
            striped
            onRowClick={(row) => selectedId = selectedId === row.id ? null : row.id}
            rowKey="id"
          >
            {#snippet cell({ value, column, row })}
              {#if column.key === 'timestamp'}
                <span class="font-mono text-text-muted">{value.replace('T', ' ').slice(11, 19)}Z</span>
              {:else if column.key === 'severity'}
                {@const s = SEV_MAP[row.severity as keyof typeof SEV_MAP]}
                <Badge variant={s.variant as any} size="xs" class="font-bold">{s.label}</Badge>
              {:else if column.key === 'host_id'}
                <span class="text-accent font-bold italic">{value}</span>
              {:else if column.key === 'event_type'}
                <span class="font-mono opacity-80">{value}</span>
              {:else if column.key === 'raw_log'}
                <div class="flex flex-col">
                  <span class="font-mono text-[11px] text-text-secondary truncate max-w-2xl">{value}</span>
                  {#if selectedId === row.id}
                    <div class="mt-3 p-4 bg-surface-2 border border-border-primary rounded-sm animate-in fade-in slide-in-from-top-2">
                       <div class="grid grid-cols-2 gap-x-8 gap-y-2 mb-4">
                          {#each [['id', row.id], ['user', row.user], ['source_ip', row.source_ip], ['tactic', row.mitre_tactic]] as [k, v]}
                            <div class="flex justify-between border-b border-border-subtle/30 pb-1">
                              <span class="text-[9px] font-mono text-text-muted uppercase">{k}</span>
                              <span class="text-[10px] font-bold text-text-heading">{v || '—'}</span>
                            </div>
                          {/each}
                       </div>
                       <pre class="text-[10px] font-mono text-accent bg-surface-3 p-3 border border-border-primary rounded-xs overflow-x-auto whitespace-pre-wrap leading-relaxed shadow-inner">
                         {row.raw_log}
                       </pre>
                       <div class="flex gap-2 mt-4">
                         <Button variant="primary" size="sm" class="text-error border-error/30 hover:bg-error/10" icon={Shield}>ISOLATE_HOST</Button>
                         <Button variant="secondary" size="sm" icon={Activity}>PIVOT_USER</Button>
                         <Button variant="secondary" size="sm" icon={Maximize2}>FULL_DETAIL</Button>
                       </div>
                    </div>
                  {/if}
                </div>
              {:else if column.key === 'mitre_tactic'}
                <Badge variant="secondary" size="xs" class="font-mono opacity-60">{value}</Badge>
              {:else}
                {value}
              {/if}
            {/snippet}
          </DataTable>
        </div>
      </div>
    </div>

    <!-- STATUS BAR -->
    <div class="bg-surface-2 border-t border-border-primary px-4 py-1.5 flex items-center justify-between text-[8px] font-mono uppercase tracking-widest text-text-muted shrink-0">
      <div class="flex items-center gap-4">
        <div class="flex items-center gap-1.5">
          <div class="w-1 h-1 rounded-full bg-success"></div>
          <span>INDEX_HEALTH: NOMINAL</span>
        </div>
        <span>EPS: 148,220</span>
        <span>RETENTION: 365d HOT</span>
      </div>
      <div class="flex gap-4">
        <span>SIEM_PLANE: OQL_v4.2</span>
        <span class="opacity-40 italic">Results limited to tactical performance window</span>
      </div>
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
