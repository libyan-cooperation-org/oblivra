<!-- OBLIVRA Web — Fleet Management (Svelte 5) -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { PageLayout, Badge, Button, Spinner, DataTable, KPI } from '@components/ui';
  import { 
    RefreshCw, 
    Search, 
    ShieldCheck, 
    Clock
  } from 'lucide-svelte';
  import { request } from '../services/api';

  // -- Types --
  interface Agent { 
    id: string; 
    hostname: string; 
    os: string; 
    arch: string; 
    version: string; 
    collectors: string[]; 
    last_seen: string; 
    status: 'online' | 'offline' | 'degraded'; 
    tenant_id?: string; 
  }

  // -- State --
  let agents  = $state<Agent[]>([]);
  let loading = $state(true);
  let search  = $state('');

  // -- Actions --
  async function fetchAgents() {
    loading = true;
    try {
      agents = await request<Agent[]>('/agents');
    } catch {
      agents = [];
    } finally {
      loading = false;
    }
  }

  onMount(fetchAgents);

  // -- Derived --
  const filtered = $derived(agents.filter(a =>
    a.hostname?.toLowerCase().includes(search.toLowerCase()) ||
    (a.tenant_id ?? '').toLowerCase().includes(search.toLowerCase())
  ));

  const stats = $derived({
    total: agents.length,
    online: agents.filter(a => a.status === 'online').length,
    offline: agents.filter(a => a.status === 'offline').length,
    degraded: agents.filter(a => a.status === 'degraded').length
  });

  const statusMap: Record<string, { label: string; variant: any }> = {
    online:   { label: 'ONLINE',   variant: 'success' },
    offline:  { label: 'OFFLINE',  variant: 'secondary' },
    degraded: { label: 'DEGRADED', variant: 'warning' },
  };

  function timeSince(iso: string): string {
    const diff = Date.now() - new Date(iso).getTime();
    const m = Math.floor(diff / 60000);
    if (m < 1) return 'JUST NOW';
    if (m < 60) return `${m}M AGO`;
    return `${Math.floor(m/60)}H AGO`;
  }
</script>

<PageLayout title="Fleet Command" subtitle="Agent telemetry orchestration, multi-tenant shard monitoring, and deployment state validation">
  {#snippet toolbar()}
    <div class="flex items-center gap-2">
      <Button variant="secondary" size="sm" onclick={fetchAgents}>
        <RefreshCw size={14} class="mr-2" />
        RE-SYNC
      </Button>
      <Button variant="primary" size="sm" icon={ShieldCheck}>PROVISION_NODE</Button>
    </div>
  {/snippet}

  <div class="flex flex-col h-full gap-6">
    <!-- METRIC STRIP -->
    <div class="grid grid-cols-1 md:grid-cols-4 gap-4 shrink-0">
      <KPI label="Total Nodes" value={stats.total.toString()} sublabel="Global Fleet" variant="accent" />
      <KPI label="Operational" value={stats.online.toString()} sublabel="Active Heartbeat" variant="success" />
      <KPI label="Degraded" value={stats.degraded.toString()} sublabel="Action Required" variant="warning" />
      <KPI label="Offline" value={stats.offline.toString()} sublabel="Silent Shards" variant="critical" />
    </div>

    <!-- MAIN CONTENT -->
    <div class="flex-1 flex flex-col bg-surface-1 border border-border-primary rounded-sm overflow-hidden shadow-premium">
      <div class="p-3 bg-surface-2 border-b border-border-primary flex justify-between items-center shrink-0">
        <div class="flex items-center gap-3 bg-surface-0 border border-border-primary rounded-sm px-3 py-1.5 focus-within:border-accent-primary transition-colors group">
          <Search size={14} class="text-text-muted group-focus-within:text-accent-primary" />
          <input bind:value={search} placeholder="Filter by host or tenant..." class="bg-transparent border-none outline-none text-xs font-mono text-text-secondary w-64" />
        </div>
        <div class="text-[9px] font-mono text-text-muted uppercase tracking-widest italic">
          Sharding active across {new Set(agents.map(a => a.tenant_id)).size} tenants
        </div>
      </div>

      <div class="flex-1 overflow-auto">
        {#if loading}
          <div class="h-full flex items-center justify-center"><Spinner /></div>
        {:else}
          <DataTable 
            data={filtered} 
            columns={[
              { key: 'status', label: 'NODE_STATE', width: '120px' },
              { key: 'hostname', label: 'IDENTITY_ENDPOINT' },
              { key: 'tenant', label: 'TENANT_SHARD', width: '150px' },
              { key: 'platform', label: 'ARCHITECTURE', width: '180px' },
              { key: 'collectors', label: 'TELEMETRY_PIPELINE' },
              { key: 'version', label: 'BUILD', width: '100px' },
              { key: 'last_seen', label: 'HEARTBEAT', width: '150px' }
            ]} 
            compact
            rowKey="id"
          >
            {#snippet cell({ column, row })}
              {#if column.key === 'status'}
                {@const s = statusMap[row.status] ?? statusMap.offline}
                <Badge variant={s.variant} size="xs" dot class="font-black italic">{s.label}</Badge>
              {:else if column.key === 'hostname'}
                <div class="flex flex-col py-1">
                  <span class="text-[11px] font-bold text-text-heading uppercase tracking-tighter italic">{row.hostname}</span>
                  <span class="text-[9px] font-mono text-text-muted opacity-40 uppercase tracking-tighter">ID: {row.id}</span>
                </div>
              {:else if column.key === 'tenant'}
                <span class="text-[10px] font-mono text-accent-primary font-bold uppercase">{row.tenant_id || 'GLOBAL_ROOT'}</span>
              {:else if column.key === 'platform'}
                <div class="flex items-center gap-2">
                  <Badge variant="secondary" size="xs">{row.os?.toUpperCase()}</Badge>
                  <span class="text-[9px] font-mono text-text-muted opacity-60 uppercase">{row.arch}</span>
                </div>
              {:else if column.key === 'collectors'}
                <div class="flex flex-wrap gap-1">
                  {#each row.collectors as c}
                    <span class="px-1.5 py-0.5 bg-surface-2 border border-border-subtle rounded-xs text-[8px] font-mono text-text-secondary uppercase">{c}</span>
                  {/each}
                </div>
              {:else if column.key === 'version'}
                <span class="text-[10px] font-mono text-text-muted">v{row.version}</span>
              {:else if column.key === 'last_seen'}
                <div class="flex items-center gap-2 text-[9px] font-mono text-text-muted uppercase tracking-tighter">
                  <Clock size={10} />
                  {timeSince(row.last_seen)}
                </div>
              {/if}
            {/snippet}
          </DataTable>
        {/if}
      </div>
    </div>
  </div>

  <!-- STATUS BAR -->
  <div class="bg-surface-2 border-t border-border-primary px-3 py-1 flex items-center gap-4 text-[8px] font-mono text-text-muted shrink-0 uppercase tracking-widest mt-6">
    <div class="flex items-center gap-1.5">
      <div class="w-1 h-1 rounded-full bg-status-online"></div>
      <span>FLEET_PLANE:</span>
      <span class="text-status-online font-bold italic">OPTIMIZED</span>
    </div>
    <span class="text-border-primary opacity-30">|</span>
    <div class="flex items-center gap-1.5">
      <span>HEARTBEAT_INTERVAL:</span>
      <span class="text-status-online font-bold italic">60s</span>
    </div>
    <span class="text-border-primary opacity-30">|</span>
    <div class="flex items-center gap-1.5">
      <span>AGENT_AUTO_UPDATE:</span>
      <span class="text-accent-primary font-bold italic">ENABLED</span>
    </div>
    <div class="ml-auto opacity-40">OBLIVRA_FLEET_ORCHESTRATOR v3.2.1</div>
  </div>
</PageLayout>
