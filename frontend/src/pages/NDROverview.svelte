<!--
  OBLIVRA — NDR Overview (Svelte 5)
  Network Detection and Response: Deep packet inspection and traffic flow analysis.
-->
<script lang="ts">
  import { KPI, PageLayout, Badge, Button, DataTable } from '@components/ui';
  import { Shield, Zap, Globe, Activity, Database, Server, Crosshair, Map, Network } from 'lucide-svelte';
  import { appStore } from '@lib/stores/app.svelte';

  const trafficFlows = [
    { id: 'F1', source: '10.0.8.2', dest: '45.12.1.8', protocol: 'HTTPS/TLS', status: 'monitored', risk: 'low' },
    { id: 'F2', source: 'db-01', dest: 'backup-s3', protocol: 'S3-REST', status: 'encrypted', risk: 'low' },
    { id: 'F3', source: 'web-prod-04', dest: 'cnc.malicious.net', protocol: 'DNS-TUNNEL', status: 'intercepted', risk: 'critical' },
    { id: 'F4', source: 'mesh-node-k', dest: 'mesh-node-z', protocol: 'P2P-SYMM', status: 'peer-verified', risk: 'low' },
  ];
</script>

<PageLayout title="Network Operations" subtitle="Deep packet inspection and protocol-level traffic analysis: Native NDR orchestration">
  {#snippet toolbar()}
    <div class="flex items-center gap-2">
      <Button variant="secondary" size="sm" icon="📡">Capture PCAP</Button>
      <Button variant="primary" size="sm" icon="🛡️">Engage IDS Filter</Button>
    </div>
  {/snippet}

  <div class="flex flex-col h-full gap-6">
    <!-- Pulse Stats -->
    <div class="grid grid-cols-1 md:grid-cols-4 gap-4 shrink-0">
      <KPI title="Mesh Throughput" value="4.2 Gbps" trend="High Density" variant="accent" />
      <KPI title="Analyzed Shards" value="14.2M" trend="Nominal" variant="success" />
      <KPI title="Logic Breaches" value={trafficFlows.filter(f => f.risk === 'critical').length} trend="Active Eviction" variant="error" />
      <KPI title="Encryption Ratio" value="98.2%" trend="Hardened" variant="success" />
    </div>

    <div class="flex-1 min-h-0 grid grid-cols-1 lg:grid-cols-3 gap-6">
      <!-- Traffic Inventory -->
      <div class="lg:col-span-2 bg-surface-1 border border-border-primary rounded-md overflow-hidden flex flex-col shadow-premium">
         <div class="p-3 bg-surface-2 border-b border-border-primary flex justify-between items-center text-[10px] font-bold uppercase tracking-widest text-text-muted font-mono">
            Live Traffic Flow Analytics (L7 Deep Inspection)
            <Badge variant="error" size="xs">DPI ACTIVE</Badge>
         </div>
         <div class="flex-1 overflow-auto">
            <DataTable data={trafficFlows} columns={[
              { key: 'source', label: 'Origin Shard' },
              { key: 'dest', label: 'Exit Node' },
              { key: 'protocol', label: 'Protocol Logic', width: '120px' },
              { key: 'status', label: 'Verification', width: '120px' },
              { key: 'action', label: '', width: '80px' }
            ]} density="compact">
              {#snippet cell({ column, row })}
                {#if column.key === 'status'}
                   <Badge variant={row.risk === 'critical' ? 'error' : row.status === 'monitored' ? 'info' : 'success'} dot={row.risk === 'critical'}>
                      {row.status.toUpperCase()}
                   </Badge>
                {:else if column.key === 'protocol'}
                   <span class="text-[10px] font-bold text-text-secondary uppercase tracking-widest">{row.protocol}</span>
                {:else if column.key === 'source' || column.key === 'dest'}
                   <div class="flex items-center gap-2">
                      <div class="w-1.5 h-1.5 rounded-full {row.risk === 'critical' ? 'bg-error' : 'bg-accent'} opacity-40"></div>
                      <code class="text-[11px] font-bold text-text-heading">{row[column.key]}</code>
                   </div>
                {:else if column.key === 'action'}
                   <div class="flex gap-2">
                      <Button variant="ghost" size="xs"><Activity size={12} /></Button>
                      <Button variant="ghost" size="xs"><Crosshair size={12} /></Button>
                   </div>
                {:else}
                  <span class="text-[11px] text-text-secondary">{row[column.key]}</span>
                {/if}
              {/snippet}
            </DataTable>
         </div>
      </div>

      <!-- Protocol Distribution -->
      <div class="flex flex-col gap-6">
         <div class="bg-surface-1 border border-border-primary rounded-md p-6 flex flex-col gap-5 shadow-sm">
            <div class="text-[10px] font-bold text-text-muted uppercase tracking-widest border-b border-border-primary pb-3 flex items-center justify-between">
               <span>Tactical Distribution</span>
               <Network size={12} />
            </div>
            <div class="space-y-5">
               <div>
                  <div class="flex justify-between text-[10px] mb-1.5 font-bold uppercase tracking-widest">
                     <span class="text-text-secondary">HTTPS/TLS (Sovereign)</span>
                     <span class="text-accent">64%</span>
                  </div>
                  <div class="w-full bg-surface-3 h-1.5 rounded-full overflow-hidden shadow-inner">
                     <div class="bg-accent h-full shadow-glow-accent/20" style="width: 64%"></div>
                  </div>
               </div>
               <div>
                  <div class="flex justify-between text-[10px] mb-1.5 font-bold uppercase tracking-widest">
                     <span class="text-text-secondary">Encrypted P2P Mesh</span>
                     <span class="text-success">22%</span>
                  </div>
                  <div class="w-full bg-surface-3 h-1.5 rounded-full overflow-hidden shadow-inner">
                     <div class="bg-success h-full" style="width: 22%"></div>
                  </div>
               </div>
               <div>
                  <div class="flex justify-between text-[10px] mb-1.5 font-bold uppercase tracking-widest">
                     <span class="text-text-secondary">Unknown/Entropy</span>
                     <span class="text-error">4%</span>
                  </div>
                  <div class="w-full bg-surface-3 h-1.5 rounded-full overflow-hidden shadow-inner">
                     <div class="bg-error h-full" style="width: 4%"></div>
                  </div>
               </div>
            </div>
         </div>

         <div class="flex-1 bg-surface-1 border border-border-primary rounded-md p-8 flex flex-col items-center justify-center text-center gap-4 relative overflow-hidden group shadow-premium border-dashed hover:border-accent/40 transition-all">
            <div class="absolute inset-0 bg-accent/5 group-hover:bg-accent/10 transition-colors pointer-events-none"></div>
            <Zap size={40} class="text-accent relative z-10 opacity-60 group-hover:scale-110 transition-transform duration-500" />
            <div class="relative z-10">
               <h4 class="text-xs font-bold text-text-heading uppercase tracking-widest">Logic Eviction</h4>
               <p class="text-[10px] text-text-muted mt-2 max-w-[180px] leading-relaxed font-bold opacity-60">
                 DPI logic is analyzing 142 distinct protocols across all mesh shards.
               </p>
            </div>
         </div>
      </div>
    </div>
  </div>
</PageLayout>
