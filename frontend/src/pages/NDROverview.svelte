<!--
  OBLIVRA — NDR Overview (Svelte 5)
  Deep packet inspection and traffic flow analysis.
-->
<script lang="ts">
  import { PageLayout, Badge, Button, DataTable } from '@components/ui';
  import { Activity, Share2, ArrowRight, Filter, Shield } from 'lucide-svelte';

  const flows = [
    { time: '10:42:15', src: '10.18.2.44', dst: '104.1.2.4', proto: 'HTTPS', app: 'Web-SSL', bytes: '1.2 GB', risk: 94 },
    { time: '10:42:14', src: '10.18.2.12', dst: '8.8.8.8', proto: 'DNS', app: 'GoogleDNS', bytes: '4.2 KB', risk: 12 },
    { time: '10:42:12', src: '10.18.4.1', dst: 'internal-git', proto: 'SSH', app: 'Git-Sync', bytes: '44.8 MB', risk: 5 },
    { time: '10:42:10', src: '10.18.2.44', dst: '10.18.5.1', proto: 'SMB', app: 'File-Share', bytes: '882.1 MB', risk: 78 },
    { time: '10:42:08', src: '192.168.1.1', dst: '10.18.2.44', proto: 'RDP', app: 'Remote-Desktop', bytes: '12.4 MB', risk: 85 }
  ];
</script>

<PageLayout title="Network Operations" subtitle="Deep packet inspection and native NDR orchestration">
  {#snippet toolbar()}
    <div class="flex items-center gap-2">
      <Button variant="secondary" size="sm" icon={Filter}>FLOW FILTERS</Button>
      <Button variant="primary" size="sm">ENGAGE IDS</Button>
    </div>
  {/snippet}

  <div class="flex flex-col h-full gap-0 -m-6">
    <!-- METRIC STRIP -->
    <div class="grid grid-cols-4 gap-px bg-border-primary border-b border-border-primary shrink-0">
        <div class="bg-surface-2 p-3">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">Mesh Throughput</div>
            <div class="text-xl font-mono font-bold text-accent">1.4 GB/s</div>
            <div class="text-[9px] text-success mt-1">▲ Nominal capacity</div>
        </div>
        <div class="bg-surface-2 p-3">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">Active Flows</div>
            <div class="text-xl font-mono font-bold text-text-heading">12,482</div>
            <div class="text-[9px] text-text-muted mt-1">Across 14 shards</div>
        </div>
        <div class="bg-surface-2 p-3">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">Anomalous Flows</div>
            <div class="text-xl font-mono font-bold text-error">42</div>
            <div class="text-[9px] text-error mt-1 animate-pulse">▲ SLA alert threshold</div>
        </div>
        <div class="bg-surface-2 p-3">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">DPI Efficiency</div>
            <div class="text-xl font-mono font-bold text-success">99.8%</div>
            <div class="text-[9px] text-success mt-1">L7 Logic depth active</div>
        </div>
    </div>

    <!-- MAIN BODY -->
    <div class="flex-1 flex min-h-0">
        <!-- LEFT: FLOW LEDGER -->
        <div class="flex-1 flex flex-col min-w-0">
            <div class="bg-surface-1 border-b border-border-primary p-3 flex items-center justify-between shrink-0">
                <div class="flex items-center gap-2">
                    <Activity size={14} class="text-accent" />
                    <span class="text-[10px] font-mono font-bold uppercase tracking-widest text-text-heading">Live Traffic Stream (L7 DPI)</span>
                </div>
                <div class="flex items-center gap-2 px-2 py-0.5 bg-success/10 border border-success/20 rounded-full">
                    <div class="w-1.5 h-1.5 bg-success rounded-full animate-pulse"></div>
                    <span class="text-[8px] font-mono text-success uppercase">Streaming</span>
                </div>
            </div>

            <div class="flex-1 overflow-auto mask-fade-bottom">
                <DataTable 
                    data={flows} 
                    columns={[
                        { key: 'time', label: 'TIME', width: '80px' },
                        { key: 'src', label: 'SOURCE' },
                        { key: 'dst', label: 'DESTINATION' },
                        { key: 'proto', label: 'PROTO', width: '80px' },
                        { key: 'app', label: 'APP', width: '100px' },
                        { key: 'bytes', label: 'BYTES', width: '100px' },
                        { key: 'risk', label: 'RISK', width: '60px' }
                    ]} 
                    compact
                >
                    {#snippet render({ col, row })}
                        {#if col.key === 'time'}
                            <span class="text-[9px] font-mono text-text-muted">{row.time}</span>
                        {:else if col.key === 'src'}
                            <div class="flex items-center gap-2 py-0.5">
                                <div class="w-1 h-1 rounded-full bg-accent"></div>
                                <span class="text-[10px] font-mono text-text-secondary">{row.src}</span>
                            </div>
                        {:else if col.key === 'dst'}
                            <div class="flex items-center gap-2 py-0.5">
                                <ArrowRight size={10} class="text-text-muted" />
                                <span class="text-[10px] font-mono text-text-secondary">{row.dst}</span>
                            </div>
                        {:else if col.key === 'proto'}
                            <Badge variant="info" size="xs" class="text-[8px] font-mono px-1.5">{row.proto}</Badge>
                        {:else if col.key === 'app'}
                            <span class="text-[9px] font-mono text-text-muted uppercase">{row.app}</span>
                        {:else if col.key === 'bytes'}
                            <span class="text-[9px] font-mono text-text-muted tabular-nums">{row.bytes}</span>
                        {:else if col.key === 'risk'}
                            <div class="flex items-center gap-1.5">
                                <div class="w-1.5 h-1.5 rounded-full {row.risk > 80 ? 'bg-error shadow-[0_0_4px_rgba(200,44,44,1)]' : row.risk > 40 ? 'bg-warning' : 'bg-success'}"></div>
                                <span class="text-[9px] font-mono text-text-muted">{row.risk}</span>
                            </div>
                        {/if}
                    {/snippet}
                </DataTable>
            </div>
        </div>

        <!-- RIGHT: PROTOCOL DISTRIBUTION -->
        <div class="w-80 bg-surface-2 border-l border-border-primary flex flex-col shrink-0">
            <div class="px-3 py-2 bg-surface-3 border-b border-border-primary flex items-center gap-2">
                <Share2 size={14} class="text-text-muted" />
                <span class="text-[9px] font-mono font-bold uppercase tracking-widest text-text-heading">Protocol Distribution</span>
            </div>
            
            <div class="p-4 space-y-6">
                {#each [
                    { name: 'HTTPS / TLS', val: 64, color: 'accent' },
                    { name: 'P2P Mesh Sync', val: 22, color: 'success' },
                    { name: 'DNS (Recursive)', val: 8, color: 'info' },
                    { name: 'Entropy / Unknown', val: 6, color: 'warning' }
                ] as proto}
                    <div class="space-y-2">
                        <div class="flex justify-between text-[10px] font-mono">
                            <span class="text-text-muted uppercase tracking-tight">{proto.name}</span>
                            <span class="text-text-heading font-bold">{proto.val}%</span>
                        </div>
                        <div class="h-1 bg-surface-3 rounded-full overflow-hidden">
                            <div class="h-full bg-{proto.color}" style="width: {proto.val}%"></div>
                        </div>
                    </div>
                {/each}

                <div class="pt-4 border-t border-border-primary space-y-4">
                    <span class="text-[9px] font-mono font-bold text-text-muted uppercase tracking-widest">Network Logic Status</span>
                    <div class="bg-surface-1 border border-border-primary p-3 rounded-sm space-y-2 group hover:border-accent cursor-pointer transition-colors">
                        <div class="flex items-center gap-2">
                            <Shield size={14} class="text-success" />
                            <span class="text-[10px] font-bold text-text-heading uppercase">Exfil Defense</span>
                        </div>
                        <p class="text-[8px] text-text-muted font-mono leading-relaxed opacity-60">
                            Automatic egress blocking enabled for all high-entropy UDP traffic to unverified ASNs.
                        </p>
                    </div>
                </div>
            </div>

            <div class="mt-auto border-t border-border-primary p-4 bg-surface-3/30">
                 <div class="flex items-center gap-2 mb-2">
                    <span class="text-[9px] font-mono font-bold text-text-muted uppercase tracking-widest">Global Ingress</span>
                 </div>
                 <div class="text-[8px] font-mono text-text-muted space-y-1">
                    <div>Origin: US-EAST-1 (62%)</div>
                    <div>Peak Rate: 4.2 Gbps</div>
                    <div>Threat Blocks: 1,422/h</div>
                 </div>
            </div>
        </div>
    </div>

    <!-- STATUS BAR -->
    <div class="bg-surface-2 border-t border-border-primary px-3 py-1 flex items-center gap-4 text-[8px] font-mono text-text-muted shrink-0">
        <div class="flex items-center gap-1.5">
            <span>DPI_ENGINE:</span>
            <span class="text-success font-bold">OPTIMIZED</span>
        </div>
        <span class="text-border-primary">|</span>
        <div class="flex items-center gap-1.5">
            <span>MESH_THROUGHPUT:</span>
            <span class="text-accent font-bold">NOMINAL</span>
        </div>
        <span class="text-border-primary">|</span>
        <div class="flex items-center gap-1.5">
            <span>SURICATA_V7:</span>
            <span class="text-success font-bold">ARMED</span>
        </div>
        <div class="ml-auto uppercase tracking-widest opacity-60">NDR_MESH v1.4.1</div>
    </div>
  </div>
</PageLayout>

<style>
  .overflow-auto {
    mask-image: linear-gradient(to bottom, transparent 0px, black 12px, black calc(100% - 16px), transparent 100%);
  }
</style>
