<!-- OBLIVRA Web — NDR Dashboard (Svelte 5) -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { KPI, Badge, Button, DataTable, PageLayout, Spinner, ProgressBar } from '@components/ui';
  import { Activity, Share2, ArrowRight, Filter, Shield, Zap, Globe, AlertTriangle, ChevronRight } from 'lucide-svelte';
  import { request } from '../services/api';

  // -- Types --
  interface NetFlow {
    id: string;
    timestamp: string;
    src_ip: string;
    dst_ip: string;
    src_port: number;
    dst_port: number;
    protocol: string;
    bytes: number;
    packets: number;
    flags: string;
    ja3?: string;
    anomaly?: boolean;
    anomaly_type?: string;
  }
  interface NDRAlert {
    id: string;
    timestamp: string;
    type: string;
    src_ip: string;
    dst_ip: string;
    confidence: number;
    description: string;
    mitre_technique?: string;
  }

  // -- State --
  let tab         = $state<'flows' | 'alerts' | 'protocols'>('flows');
  let flows       = $state<NetFlow[]>([]);
  let alerts      = $state<NDRAlert[]>([]);
  let protocolMap = $state<Record<string, number>>({});
  let loading     = $state(true);
  
  // -- Filters --
  let protoFilter      = $state('all');
  let showAnomalyOnly  = $state(false);

  // -- Helpers --
  const anomalyColor = (type?: string) => {
    if (!type) return 'var(--text-muted)';
    if (type.includes('lateral')) return 'var(--alert-critical)';
    if (type.includes('c2') || type.includes('beacon')) return 'var(--alert-high)';
    if (type.includes('exfil') || type.includes('dns')) return 'var(--alert-medium)';
    return 'var(--text-primary)';
  };

  const protocols = $derived(['all', ...new Set(flows.map(f => f.protocol).filter(Boolean))]);
  const anomalyCount = $derived(flows.filter(f => f.anomaly).length);
  const totalBytes = $derived(flows.reduce((acc, f) => acc + (f.bytes || 0), 0));

  const filteredFlows = $derived.by(() => {
    let list = flows;
    if (protoFilter !== 'all') list = list.filter(f => f.protocol === protoFilter);
    if (showAnomalyOnly) list = list.filter(f => f.anomaly);
    return list;
  });

  // -- Actions --
  async function fetchData() {
    loading = true;
    try {
      const [f, a, p] = await Promise.all([
        request<NetFlow[]>('/ndr/flows?limit=200'),
        request<NDRAlert[]>('/ndr/alerts?limit=50'),
        request<Record<string, number>>('/ndr/protocols')
      ]);
      flows = f ?? [];
      alerts = a ?? [];
      protocolMap = p ?? {};
    } catch (e) {
      console.error('NDR Data fetch failed', e);
    } finally {
      loading = false;
    }
  }

  onMount(() => {
    fetchData();
  });
</script>

<PageLayout title="Network Operations" subtitle="Deep packet inspection and native NDR orchestration via sovereign mesh nodes">
  {#snippet toolbar()}
    <div class="flex items-center gap-2">
      <Button variant="secondary" size="sm" onclick={fetchData}>
        <Activity size={14} class="mr-2" />
        LIVE REFRESH
      </Button>
    </div>
  {/snippet}

  <div class="flex flex-col h-full gap-0 -m-6">
    <!-- METRIC STRIP -->
    <div class="grid grid-cols-4 gap-px bg-border-primary border-b border-border-primary shrink-0">
        <div class="bg-surface-2 p-3">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">Mesh Throughput</div>
            <div class="text-xl font-mono font-bold text-accent-primary">{(totalBytes / 1024 / 1024).toFixed(1)} MB/s</div>
            <div class="text-[9px] text-status-online mt-1">▲ Nominal capacity</div>
        </div>
        <div class="bg-surface-2 p-3">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">Active Flows</div>
            <div class="text-xl font-mono font-bold text-text-heading">{flows.length.toLocaleString()}</div>
            <div class="text-[9px] text-text-muted mt-1">L7 Ingest Active</div>
        </div>
        <div class="bg-surface-2 p-3">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">Anomalous Flows</div>
            <div class="text-xl font-mono font-bold text-alert-critical">{anomalyCount}</div>
            <div class="text-[9px] text-alert-critical mt-1 {anomalyCount > 0 ? 'animate-pulse' : ''}">
              {anomalyCount > 0 ? '▲ Resolution required' : '✓ Baseline nominal'}
            </div>
        </div>
        <div class="bg-surface-2 p-3">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">DPI Efficiency</div>
            <div class="text-xl font-mono font-bold text-status-online">99.8%</div>
            <div class="text-[9px] text-status-online mt-1">Protocol Depth Synced</div>
        </div>
    </div>

    <!-- MAIN BODY -->
    <div class="flex-1 flex min-h-0 bg-surface-0">
        <!-- LEFT: MAIN CONTENT -->
        <div class="flex-1 flex flex-col min-w-0 border-r border-border-primary">
            <div class="bg-surface-1 border-b border-border-primary p-3 flex items-center justify-between shrink-0">
                <div class="flex items-center gap-4">
                    <div class="flex items-center gap-2">
                        <Zap size={14} class="text-accent-primary" />
                        <span class="text-[10px] font-mono font-bold uppercase tracking-widest text-text-heading">L7 Packet Inspection Shard</span>
                    </div>
                    
                    <div class="flex border border-border-primary rounded-sm overflow-hidden">
                      {#each ['flows', 'alerts', 'protocols'] as t}
                        <button
                          class="px-3 py-1 text-[9px] font-bold uppercase tracking-widest transition-colors
                            {tab === t ? 'bg-accent-primary text-black' : 'bg-surface-0 text-text-muted hover:text-text-secondary'}"
                          onclick={() => tab = t as any}
                        >
                          {t}
                        </button>
                      {/each}
                    </div>
                </div>
                
                {#if tab === 'flows'}
                  <div class="flex items-center gap-3">
                    <select bind:value={protoFilter} class="bg-surface-0 border border-border-primary text-[9px] font-mono text-text-muted px-2 py-0.5 rounded-sm outline-hidden">
                      {#each protocols as p}
                        <option value={p}>{p.toUpperCase()}</option>
                      {/each}
                    </select>
                    <label class="flex items-center gap-1.5 cursor-pointer">
                      <input type="checkbox" bind:checked={showAnomalyOnly} class="sr-only peer" />
                      <div class="w-2.5 h-2.5 border border-border-primary rounded-xs peer-checked:bg-alert-critical peer-checked:border-alert-critical transition-all"></div>
                      <span class="text-[9px] font-mono text-text-muted uppercase">Anomalies</span>
                    </label>
                  </div>
                {/if}
            </div>

            <div class="flex-1 overflow-auto">
              {#if loading}
                <div class="h-full flex items-center justify-center">
                  <Spinner />
                </div>
              {:else if tab === 'flows'}
                <DataTable 
                  data={filteredFlows} 
                  columns={[
                    { key: 'timestamp', label: 'TIME', width: '80px' },
                    { key: 'src', label: 'SOURCE' },
                    { key: 'dst', label: 'DESTINATION' },
                    { key: 'protocol', label: 'PROTO', width: '80px' },
                    { key: 'bytes', label: 'BYTES', width: '100px' },
                    { key: 'flags', label: 'FLAGS', width: '80px' },
                    { key: 'anomaly', label: 'ANALYSIS', width: '140px' }
                  ]} 
                  compact
                  rowKey="id"
                >
                  {#snippet cell({ column, row })}
                    {#if column.key === 'timestamp'}
                      <span class="text-[9px] font-mono text-text-muted">{new Date(row.timestamp).toLocaleTimeString()}</span>
                    {:else if column.key === 'src'}
                      <span class="text-[10px] font-mono text-text-secondary">{row.src_ip}:{row.src_port}</span>
                    {:else if column.key === 'dst'}
                      <div class="flex items-center gap-2">
                        <ArrowRight size={10} class="text-text-muted opacity-40" />
                        <span class="text-[10px] font-mono text-text-secondary">{row.dst_ip}:{row.dst_port}</span>
                      </div>
                    {:else if column.key === 'protocol'}
                      <Badge variant="accent" size="xs" class="text-[8px] font-mono px-1.5">{row.protocol}</Badge>
                    {:else if column.key === 'bytes'}
                      <span class="text-[9px] font-mono text-text-muted tabular-nums">{(row.bytes / 1024).toFixed(1)} KB</span>
                    {:else if column.key === 'flags'}
                      <span class="text-[9px] font-mono text-text-muted opacity-60 uppercase">{row.flags || '—'}</span>
                    {:else if column.key === 'anomaly'}
                      {#if row.anomaly}
                        <div class="flex items-center gap-1.5">
                          <div class="w-1.5 h-1.5 rounded-full animate-pulse" style="background: {anomalyColor(row.anomaly_type)}"></div>
                          <span class="text-[9px] font-bold uppercase tracking-tight" style="color: {anomalyColor(row.anomaly_type)}">
                            {row.anomaly_type?.replace(/_/g, ' ') || 'ANOMALY'}
                          </span>
                        </div>
                      {:else}
                        <span class="text-[9px] font-mono text-text-muted opacity-20 uppercase">Baseline</span>
                      {/if}
                    {/if}
                  {/snippet}
                </DataTable>
              {:else if tab === 'alerts'}
                <div class="p-4 space-y-3">
                  {#each alerts as alert}
                    <div class="bg-surface-1 border border-border-primary border-l-2 p-3 rounded-sm relative overflow-hidden group" style="border-left-color: {anomalyColor(alert.type)}">
                      <div class="absolute -right-2 -bottom-2 opacity-[0.03] grayscale group-hover:scale-110 transition-transform duration-700">
                        <Shield size={64} />
                      </div>
                      <div class="flex justify-between items-start mb-2 relative z-10">
                        <div class="flex flex-col gap-0.5">
                          <div class="flex items-center gap-2">
                             <span class="text-[10px] font-black uppercase tracking-widest" style="color: {anomalyColor(alert.type)}">{alert.type}</span>
                             <Badge variant="secondary" size="xs">{Math.round(alert.confidence * 100)}% CONF</Badge>
                          </div>
                          <div class="text-[9px] font-mono text-text-muted">{alert.src_ip} → {alert.dst_ip}</div>
                        </div>
                        <span class="text-[9px] font-mono text-text-muted opacity-50">{new Date(alert.timestamp).toLocaleTimeString()}</span>
                      </div>
                      <p class="text-[11px] text-text-secondary mb-2 relative z-10">{alert.description}</p>
                      {#if alert.mitre_technique}
                        <div class="inline-flex items-center gap-1.5 px-2 py-0.5 bg-surface-2 border border-border-primary rounded-xs relative z-10">
                          <Target size={10} class="text-text-muted" />
                          <span class="text-[9px] font-mono text-text-muted uppercase tracking-tighter">{alert.mitre_technique}</span>
                        </div>
                      {/if}
                    </div>
                  {:else}
                    <div class="h-full flex flex-col items-center justify-center text-text-muted gap-4 opacity-40 py-20">
                      <Shield size={48} />
                      <p class="font-mono text-[10px] uppercase tracking-widest">No active network threat signatures detected</p>
                    </div>
                  {/each}
                </div>
              {:else if tab === 'protocols'}
                <div class="p-6 grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                  {#each Object.entries(protocolMap).sort((a,b) => b[1] - a[1]) as [proto, count]}
                    {@const total = Object.values(protocolMap).reduce((a,b) => a+b, 0) || 1}
                    {@const pct = Math.round(count / total * 100)}
                    <div class="bg-surface-2 border border-border-primary p-4 rounded-sm flex flex-col gap-2 group hover:border-accent-primary transition-colors cursor-default">
                       <div class="flex justify-between items-center">
                          <span class="text-[10px] font-black text-text-muted uppercase tracking-widest">{proto}</span>
                          <span class="text-[9px] font-mono text-text-muted">{pct}%</span>
                       </div>
                       <div class="text-2xl font-mono font-bold text-text-heading group-hover:text-accent-primary transition-colors">{count.toLocaleString()}</div>
                       <ProgressBar value={pct} height="2px" color="var(--accent-primary)" />
                       <div class="text-[8px] font-mono text-text-muted opacity-60 uppercase tracking-tighter">Observed Inbound/Outbound Flows</div>
                    </div>
                  {/each}
                </div>
              {/if}
            </div>
        </div>

        <!-- RIGHT: DISTRIBUTION SIDEBAR -->
        <div class="w-80 bg-surface-1 flex flex-col shrink-0">
            <div class="px-3 py-2 bg-surface-2 border-b border-border-primary flex items-center gap-2">
                <Share2 size={14} class="text-text-muted" />
                <span class="text-[9px] font-mono font-bold uppercase tracking-widest text-text-heading">Traffic Distribution</span>
            </div>
            
            <div class="p-4 space-y-6">
                <div class="space-y-4">
                  {#each Object.entries(protocolMap).sort((a,b) => b[1] - a[1]).slice(0, 5) as [proto, count]}
                    {@const total = Object.values(protocolMap).reduce((a,b) => a+b, 0) || 1}
                    {@const pct = Math.round(count / total * 100)}
                    <div class="space-y-1.5">
                        <div class="flex justify-between text-[10px] font-mono">
                            <span class="text-text-muted uppercase tracking-tight">{proto}</span>
                            <span class="text-text-heading font-bold">{pct}%</span>
                        </div>
                        <ProgressBar value={pct} height="3px" color={proto === 'HTTPS' ? 'var(--accent-primary)' : 'var(--text-muted)'} />
                    </div>
                  {/each}
                </div>

                <div class="pt-4 border-t border-border-primary space-y-4">
                    <span class="text-[9px] font-mono font-bold text-text-muted uppercase tracking-widest">Network Logic Shards</span>
                    <div class="bg-surface-2 border border-border-primary p-3 rounded-sm space-y-2 group hover:border-accent-primary cursor-pointer transition-colors">
                        <div class="flex items-center gap-2">
                            <Shield size={14} class="text-status-online" />
                            <span class="text-[10px] font-bold text-text-heading uppercase">BGP Hijack Detect</span>
                        </div>
                        <p class="text-[8px] text-text-muted font-mono leading-relaxed opacity-60">
                            Monitoring autonomous system path stability for all critical B2B egress points.
                        </p>
                    </div>
                    <div class="bg-surface-2 border border-border-primary p-3 rounded-sm space-y-2 group hover:border-alert-critical cursor-pointer transition-colors">
                        <div class="flex items-center gap-2">
                            <Globe size={14} class="text-accent-primary" />
                            <span class="text-[10px] font-bold text-text-heading uppercase">Geo-Fencing</span>
                        </div>
                        <p class="text-[8px] text-text-muted font-mono leading-relaxed opacity-60">
                            Automatic drop rules active for all inbound traffic from sanctioned or high-risk geographic sharding zones.
                        </p>
                    </div>
                </div>
            </div>

            <div class="mt-auto border-t border-border-primary p-4 bg-surface-2">
                 <div class="flex items-center justify-between mb-2">
                    <span class="text-[9px] font-mono font-bold text-text-muted uppercase tracking-widest">Global Ingress Peak</span>
                    <Badge variant="accent" size="xs">4.2 Gbps</Badge>
                 </div>
                 <div class="text-[8px] font-mono text-text-muted space-y-1 opacity-60">
                    <div>Origin Substrate: US-EAST-1 (62%)</div>
                    <div>Threat Blocks: 1,422/h</div>
                 </div>
            </div>
        </div>
    </div>

    <!-- STATUS BAR -->
    <div class="bg-surface-2 border-t border-border-primary px-3 py-1 flex items-center gap-4 text-[8px] font-mono text-text-muted shrink-0 uppercase tracking-widest">
        <div class="flex items-center gap-1.5">
            <div class="w-1 h-1 rounded-full bg-status-online"></div>
            <span>DPI_ENGINE:</span>
            <span class="text-status-online font-bold italic">OPTIMIZED</span>
        </div>
        <span class="text-border-primary opacity-30">|</span>
        <div class="flex items-center gap-1.5">
            <span>MESH_THROUGHPUT:</span>
            <span class="text-accent-primary font-bold italic">NOMINAL</span>
        </div>
        <span class="text-border-primary opacity-30">|</span>
        <div class="flex items-center gap-1.5">
            <span>SURICATA_V7:</span>
            <span class="text-status-online font-bold italic">ARMED</span>
        </div>
        <div class="ml-auto opacity-40">OBLIVRA_NDR_MESH v1.4.1</div>
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
