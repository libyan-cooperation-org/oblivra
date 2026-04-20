<!-- OBLIVRA Web — Fleet Management (Svelte 5) -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { 
    PageLayout, 
    Badge, 
    Button, 
    Spinner, 
    DataTable, 
    KPI,
    Tabs
  } from '@components/ui';
  import type { TabItem } from '@components/ui/Tabs.svelte';
  import { 
    RefreshCw, 
    Search, 
    ShieldCheck, 
    Clock,
    Terminal,
    Globe,
    Cpu,
    ArrowUpCircle,
    Zap,
    Shield
  } from 'lucide-svelte';
  import { request } from '../services/api';

  // -- Types --
  interface Agent { 
    id: string; 
    hostname: string; 
    ip: string;
    os: 'win' | 'lin' | 'mac'; 
    arch: string; 
    version: string; 
    collectors: string[]; 
    last_seen: string; 
    status: 'online' | 'offline' | 'degraded' | 'isolated'; 
    risk_score: number;
    group: string;
    tenant_id?: string; 
  }

  const STATUS_TABS: TabItem[] = [
    { id: 'all', label: 'ALL HOSTS' },
    { id: 'win', label: 'WINDOWS' },
    { id: 'lin', label: 'LINUX' },
    { id: 'mac', label: 'MACOS' },
    { id: 'isolated', label: 'ISOLATED' },
  ];

  // -- State --
  // -- State --
  let agents = $state<Agent[]>([]);
  let loading = $state(true);
  let activeTab = $state('all');
  let searchQuery = $state('');
  let selectedAgent = $state<Agent | null>(null);

  // -- Actions --
  async function fetchAgents() {
    loading = true;
    try {
      const res = await request<Agent[]>('/agents');
      // Mocking missing fields for high-fidelity view
      agents = (res ?? []).map(a => ({
        ...a,
        ip: a.ip || `10.18.44.${Math.floor(Math.random() * 254)}`,
        risk_score: Math.floor(Math.random() * 100),
        group: a.group || (['DC-CORE', 'FIN-SRV', 'WS-PROD'][Math.floor(Math.random() * 3)]),
        version: a.version || '3.4.1'
      }));
      if (agents.length > 0) selectedAgent = agents[0];
    } catch {
      agents = [];
    } finally {
      loading = false;
    }
  }

  onMount(fetchAgents);

  // -- Derived --
  const filtered = $derived.by(() => {
    return agents.filter(a => {
      const matchesTab = activeTab === 'all' || a.os === activeTab || (activeTab === 'isolated' && a.status === 'isolated');
      const matchesSearch = a.hostname?.toLowerCase().includes(searchQuery.toLowerCase()) ||
                           a.ip?.toLowerCase().includes(searchQuery.toLowerCase()) ||
                           a.group?.toLowerCase().includes(searchQuery.toLowerCase());
      return matchesTab && matchesSearch;
    });
  });

  const stats = $derived.by(() => {
    const total = agents.length;
    const online = agents.filter(a => a.status === 'online').length;
    const updatePending = agents.filter(a => a.version !== '3.4.1').length;
    const isolated = agents.filter(a => a.status === 'isolated').length;
    return { total, online, updatePending, isolated };
  });

  // -- Table Columns --
  const columns = [
    { key: 'hostname', label: 'HOST / IP', sortable: true },
    { key: 'os', label: 'OS', width: '80px', sortable: true },
    { key: 'version', label: 'VERSION', width: '100px', sortable: true },
    { key: 'status', label: 'STATE', width: '120px', sortable: true },
    { key: 'risk_score', label: 'RISK', width: '80px', sortable: true },
    { key: 'last_seen', label: 'LAST SEEN', width: '140px', sortable: true },
  ];

  function timeSince(iso: string): string {
    const diff = Date.now() - new Date(iso).getTime();
    const m = Math.floor(diff / 60000);
    if (m < 1) return 'JUST NOW';
    if (m < 60) return `${m}M AGO`;
    return `${Math.floor(m/60)}H AGO`;
  }
</script>

<PageLayout 
  title="Fleet Management" 
  subtitle="Agent telemetry orchestration and deployment state validation"
>
  {#snippet toolbar()}
    <div class="flex items-center gap-2">
      <Button variant="secondary" size="sm" onclick={fetchAgents} icon={RefreshCw}>
        RE-SYNC
      </Button>
      <Button variant="primary" size="sm" icon={ShieldCheck}>PROVISION_NODE</Button>
    </div>
  {/snippet}

  <div class="flex flex-col h-full gap-4 overflow-hidden">
    <!-- METRICS -->
    <div class="grid grid-cols-4 gap-3 shrink-0">
      <KPI label="Total Nodes" value={stats.total} variant="accent" trend="up" trendValue="+12" />
      <KPI label="Operational" value={stats.online} variant="success" sublabel="Live Heartbeat" />
      <KPI label="Isolated" value={stats.isolated} variant="critical" sublabel="Network Restricted" progress={stats.isolated > 0 ? 100 : 0} />
      <KPI label="Update Pending" value={stats.updatePending} variant="warning" sublabel="v3.3.x → v3.4.1" />
    </div>

    <!-- MAIN BODY SPLIT -->
    <div class="flex-1 flex gap-4 min-h-0">
      <!-- LEFT: AGENT LIST -->
      <div class="flex-1 flex flex-col bg-surface-1 rounded-sm border border-border-primary overflow-hidden shadow-lg">
        <div class="flex items-center gap-4 bg-surface-2 border-b border-border-primary p-2 shrink-0">
          <Tabs tabs={STATUS_TABS} bind:active={activeTab} variant="pills" />
          
          <div class="relative flex-1 max-w-md ml-auto">
            <Search size={14} class="absolute left-3 top-1/2 -translate-y-1/2 text-text-muted" />
            <input 
              type="text" 
              bind:value={searchQuery}
              placeholder="Search host, IP..." 
              class="w-full bg-surface-0 border border-border-primary rounded-sm pl-9 pr-3 py-1.5 text-xs font-mono text-text-secondary focus:border-accent focus:outline-none transition-all"
            />
          </div>
        </div>

        <div class="flex-1 overflow-auto">
          {#if loading}
            <div class="h-full flex items-center justify-center"><Spinner size="lg" /></div>
          {:else}
            <DataTable 
              data={filtered} 
              columns={columns} 
              compact
              rowKey="id"
              onRowClick={(row) => selectedAgent = row}
            >
              {#snippet cell({ value, column, row })}
                {#if column.key === 'hostname'}
                  <div class="flex flex-col">
                    <span class="font-bold uppercase tracking-tighter {selectedAgent?.id === row.id ? 'text-accent-primary' : 'text-text-heading'} italic">{value}</span>
                    <span class="text-[9px] font-mono text-text-muted">{row.ip}</span>
                  </div>
                {:else if column.key === 'os'}
                  <Badge variant="secondary" size="xs" class="uppercase font-mono">{value}</Badge>
                {:else if column.key === 'version'}
                  <span class={value === '3.4.1' ? 'text-status-online font-mono' : 'text-alert-high font-mono'}>v{value}</span>
                {:else if column.key === 'status'}
                  <Badge 
                    variant={value === 'online' ? 'success' : value === 'isolated' ? 'danger' : 'secondary'} 
                    size="xs" 
                    dot 
                    class="font-black italic uppercase"
                  >
                    {value}
                  </Badge>
                {:else if column.key === 'risk_score'}
                  <div class="flex items-center gap-2">
                    <div class="w-1.5 h-1.5 rounded-full {value > 70 ? 'bg-alert-critical' : value > 40 ? 'bg-alert-high' : 'bg-status-online'}"></div>
                    <span class="font-mono text-[10px]">{value}</span>
                  </div>
                {:else if column.key === 'last_seen'}
                  <div class="flex items-center gap-2 text-[9px] font-mono text-text-muted uppercase tracking-tighter">
                    <Clock size={10} />
                    {timeSince(value)}
                  </div>
                {/if}
              {/snippet}
            </DataTable>
          {/if}
        </div>
      </div>

      <!-- RIGHT: AGENT DETAIL SIDEBAR -->
      <aside class="w-96 bg-surface-1 border border-border-primary rounded-sm flex flex-col shrink-0 overflow-hidden shadow-premium">
        {#if selectedAgent}
          <div class="p-4 bg-surface-2 border-b border-border-primary shrink-0 flex justify-between items-center">
            <span class="text-[10px] font-mono text-text-muted uppercase tracking-widest">Node Intelligence</span>
            <Badge variant={selectedAgent.status === 'online' ? 'success' : 'danger'} size="xs" dot>{selectedAgent.status.toUpperCase()}</Badge>
          </div>
          
          <div class="flex-1 overflow-y-auto">
            <div class="p-6 space-y-8">
              <!-- Node Header -->
              <div class="flex items-center gap-4">
                <div class="w-14 h-14 rounded-full bg-accent-primary/10 border border-accent-primary/30 flex items-center justify-center text-accent-primary">
                  <Globe size={28} />
                </div>
                <div>
                  <h3 class="text-xl font-black text-text-heading uppercase tracking-tighter italic">{selectedAgent.hostname}</h3>
                  <div class="flex items-center gap-2 mt-1">
                    <span class="text-[10px] font-mono text-text-muted">{selectedAgent.ip}</span>
                    <span class="text-text-muted opacity-30">•</span>
                    <span class="text-[10px] font-mono text-accent-primary font-bold">{selectedAgent.group}</span>
                  </div>
                </div>
              </div>

              <!-- Telemetry Stats Grid -->
              <div class="grid grid-cols-2 gap-px bg-border-primary border border-border-primary rounded-sm overflow-hidden">
                <div class="bg-surface-2 p-3 space-y-1">
                  <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest">CPU Usage</div>
                  <div class="text-lg font-mono font-bold text-text-heading">14%</div>
                </div>
                <div class="bg-surface-2 p-3 space-y-1">
                  <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest">Memory</div>
                  <div class="text-lg font-mono font-bold text-text-heading">2.4/16 GB</div>
                </div>
                <div class="bg-surface-2 p-3 space-y-1">
                  <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest">Uptime</div>
                  <div class="text-lg font-mono font-bold text-text-heading">14d 2h</div>
                </div>
                <div class="bg-surface-2 p-3 space-y-1">
                  <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest">Latency</div>
                  <div class="text-lg font-mono font-bold text-status-online">12ms</div>
                </div>
              </div>

              <!-- Collectors -->
              <div class="space-y-4">
                <div class="flex items-center gap-2 text-[10px] font-black text-text-muted uppercase tracking-widest">
                  <Zap size={14} class="text-accent-primary" />
                  Active Telemetry Shards
                </div>
                <div class="space-y-2">
                  {#each [
                    { id: 'proc', label: 'Process Execution', status: 'active' },
                    { id: 'net', label: 'Network Flow', status: 'active' },
                    { id: 'fim', label: 'File Integrity', status: 'standby' },
                    { id: 'logs', label: 'Syslog Forwarder', status: 'active' }
                  ] as collector}
                    <div class="flex justify-between items-center p-2 bg-surface-2 border border-border-subtle rounded-xs">
                      <span class="text-[10px] font-mono text-text-secondary uppercase">{collector.label}</span>
                      <Badge variant={collector.status === 'active' ? 'success' : 'secondary'} size="xs">{collector.status.toUpperCase()}</Badge>
                    </div>
                  {/each}
                </div>
              </div>

              <!-- Hardware Summary -->
              <div class="space-y-3">
                <div class="flex items-center gap-2 text-[10px] font-black text-text-muted uppercase tracking-widest">
                  <Cpu size={14} class="text-text-muted" />
                  Hardware Signature
                </div>
                <div class="text-[10px] font-mono text-text-muted space-y-1 leading-relaxed">
                  <div>CPU: AMD EPYC 7742 64-Core Processor</div>
                  <div>KERNEL: 5.15.0-76-generic (x86_64)</div>
                  <div>SERIAL: OBLV-8829-FF01-X9</div>
                </div>
              </div>
            </div>
          </div>

          <!-- Actions -->
          <div class="p-4 bg-surface-2 border-t border-border-primary grid grid-cols-2 gap-2 shrink-0">
             <Button variant="secondary" size="sm" icon={Terminal} class="text-[10px]">OPEN_TERMINAL</Button>
             <Button variant="secondary" size="sm" icon={Shield} class="text-[10px]">POLICIES</Button>
             <Button variant="primary" size="sm" class="col-span-2 text-[10px]" icon={ArrowUpCircle}>FORCE_UPDATE_AGENT</Button>
             <Button variant="secondary" size="sm" class="col-span-2 text-alert-critical border-alert-critical/30 text-[10px]">ISOLATE_NODE</Button>
          </div>
        {:else}
          <div class="flex-1 flex flex-col items-center justify-center p-12 text-center opacity-30 gap-4">
             <Shield size={48} />
             <span class="text-xs font-mono uppercase tracking-widest">Select a node to inspect telemetry</span>
          </div>
        {/if}
      </aside>
    </div>
  </div>

  <!-- STATUS BAR -->
  <div class="bg-surface-2 border-t border-border-primary px-3 py-1 flex items-center gap-4 text-[8px] font-mono text-text-muted shrink-0 uppercase tracking-widest mt-4">
    <div class="flex items-center gap-1.5">
      <div class="w-1 h-1 rounded-full bg-success"></div>
      <span>FLEET_PLANE:</span>
      <span class="text-success font-bold italic">OPTIMIZED</span>
    </div>
    <span class="text-border-primary opacity-30">|</span>
    <div class="flex items-center gap-1.5">
      <span>HEARTBEAT_INTERVAL:</span>
      <span class="text-success font-bold italic">60s</span>
    </div>
    <span class="text-border-primary opacity-30">|</span>
    <div class="flex items-center gap-1.5">
      <span>AGENT_AUTO_UPDATE:</span>
      <span class="text-accent font-bold italic">ENABLED</span>
    </div>
    <div class="ml-auto opacity-40">OBLIVRA_FLEET_ORCHESTRATOR v3.2.1</div>
  </div>
</PageLayout>
