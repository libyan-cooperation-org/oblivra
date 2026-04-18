<!--
  OBLIVRA — Enrichment Viewer (Svelte 5)
  Deep context inspector: Enriching raw logs with threat intel, identity, and geo data.
-->
<script lang="ts">
  import { KPI, Badge, PageLayout, Button, DataTable } from '@components/ui';
  import { Search, Globe, User, Shield, ExternalLink, Database, Activity, Target, Zap } from 'lucide-svelte';
  import { appStore } from '@lib/stores/app.svelte';

  const enrichedLogs = [
    { id: 'L1', time: '14:22:05', raw: 'SSH Login Success', host: 'web-prod-01', intel: 'KNOWN VPN EXIT', geo: 'NL' },
    { id: 'L2', time: '14:21:40', raw: 'Sudo attempt iceman', host: 'db-prod-02', intel: 'INTERNAL SHIFT', geo: 'LAN' },
    { id: 'L3', time: '14:18:22', raw: 'Binary Execution /tmp/sh', host: 'edge-gw', intel: 'MALICIOUS_HASH', geo: 'RU' },
    { id: 'L4', time: '14:15:10', raw: 'Inbound 443 Connection', host: 'mail-01', intel: 'REPUTATION_LOW', geo: 'US' },
  ];
</script>

<PageLayout title="Enrichment Core" subtitle="Unified context explorer: Raw telemetry combined with federated intelligence and identity mapping">
  {#snippet toolbar()}
    <div class="flex items-center gap-2">
      <Button variant="secondary" size="sm" icon="🔄">Intel Refresh</Button>
      <Button variant="primary" size="sm" icon="📦">Export Enriched Bundle</Button>
    </div>
  {/snippet}

  <div class="flex flex-col h-full gap-6">
    <!-- Pulse Stats -->
    <div class="grid grid-cols-1 md:grid-cols-4 gap-4 shrink-0">
      <KPI title="Enriched Events" value="1.4M" trend="+12k/s" variant="accent" />
      <KPI title="Intel Hits" value="142" trend="Active Scan" variant="warning" />
      <KPI title="Geo Attribution" value="Global" trend="Synced" variant="success" />
      <KPI title="Logic Accuracy" value="99.2%" trend="Verified" variant="success" />
    </div>

    <div class="flex-1 min-h-0 bg-surface-1 border border-border-primary rounded-md overflow-hidden flex flex-col shadow-premium relative group">
       <!-- Decoration -->
       <div class="absolute inset-0 pointer-events-none opacity-[0.02] flex items-center justify-center grayscale group-hover:scale-105 transition-transform duration-1000">
          <Database size={500} />
       </div>

       <div class="p-3 bg-surface-2 border-b border-border-primary flex justify-between items-center text-[10px] font-bold uppercase tracking-widest text-text-muted font-mono">
          Federated Context Stream (Live Enrichment)
          <Badge variant="success" size="xs">RESOLVER ACTIVE</Badge>
       </div>
       
       <div class="flex-1 overflow-auto relative z-10">
          <DataTable data={enrichedLogs} columns={[
            { key: 'time', label: 'Telemetry Time', width: '120px' },
            { key: 'host', label: 'Enriched Entity', width: '150px' },
            { key: 'raw', label: 'Tactical Event Logic' },
            { key: 'intel', label: 'Intelligence Context', width: '200px' },
            { key: 'geo', label: 'Attrib.', width: '80px' },
            { key: 'action', label: '', width: '60px' }
          ]} density="compact">
            {#snippet cell({ column, row })}
              {#if column.key === 'intel'}
                 <div class="flex items-center gap-2">
                    <Shield size={12} class={row.intel.includes('MALICIOUS') ? 'text-error' : row.intel.includes('LOW') ? 'text-warning' : 'text-accent'} />
                    <span class="text-[10px] font-bold uppercase tracking-widest {row.intel.includes('MALICIOUS') ? 'text-error' : row.intel.includes('LOW') ? 'text-warning' : 'text-text-secondary'}">
                       {row.intel}
                    </span>
                 </div>
              {:else if column.key === 'geo'}
                 <Badge variant={row.geo === 'LAN' ? 'accent' : 'secondary'} size="xs" class="font-bold">{row.geo}</Badge>
              {:else if column.key === 'time'}
                 <span class="text-[10px] font-mono font-bold text-text-muted opacity-60 tracking-widest">{row.time}</span>
              {:else if column.key === 'raw'}
                 <div class="flex items-center gap-2">
                    <Zap size={10} class="text-accent opacity-40" />
                    <code class="text-[11px] font-bold text-text-heading group-hover:text-accent transition-colors">{row.raw}</code>
                 </div>
              {:else if column.key === 'host'}
                 <div class="flex items-center gap-2">
                    <User size={12} class="text-text-muted" />
                    <span class="text-[11px] font-bold text-text-secondary uppercase tracking-tighter">{row.host}</span>
                 </div>
              {:else if column.key === 'action'}
                 <Button variant="ghost" size="xs"><ExternalLink size={14} /></Button>
              {:else}
                <span class="text-[11px] text-text-secondary">{row[column.key]}</span>
              {/if}
            {/snippet}
          </DataTable>
       </div>
    </div>
  </div>
</PageLayout>
