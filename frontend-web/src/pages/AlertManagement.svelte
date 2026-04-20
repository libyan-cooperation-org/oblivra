<!-- OBLIVRA Web — Alert Management (Svelte 5) -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { 
    PageLayout, 
    Badge, 
    Button, 
    Spinner, 
    KPI, 
    Tabs, 
    DataTable 
  } from '@components/ui';
  import type { TabItem } from '@components/ui/Tabs.svelte';
  import { 
    ShieldAlert, 
    Search, 
    User, 
    Activity,
    Target,
    RefreshCw, 
    Terminal, 
    Globe,
    Eye,
    Zap
  } from 'lucide-svelte';
  import { request } from '../services/api';
  import { push } from '../core/router.svelte';
  import InvestigationCanvas from '../components/ui/InvestigationCanvas.svelte';

  // -- Types --
  interface Alert { 
    id: string; 
    tenant_id: string; 
    host_id: string; 
    timestamp: string; 
    event_type: string; 
    source_ip: string; 
    user: string; 
    raw_log: string; 
    severity: 'critical' | 'high' | 'medium' | 'low';
    status: 'open' | 'investigating' | 'acknowledged' | 'closed' | 'suppressed';
    risk_score: number;
    sla_seconds: number;
    sla_pct: number;
    assigned_to?: string;
  }

  // -- Constants --
  const SEV_MAP = {
    critical: { label: 'CRIT', variant: 'danger', color: 'text-error' },
    high:     { label: 'HIGH', variant: 'warning', color: 'text-warning' },
    medium:   { label: 'MED',  variant: 'warning', color: 'text-warning/80' },
    low:      { label: 'LOW',  variant: 'success', color: 'text-success' },
  } as const;

  const STATUS_TABS: TabItem[] = [
    { id: 'open', label: 'OPEN' },
    { id: 'acknowledged', label: 'ACK' },
    { id: 'investigating', label: 'INVEST.' },
    { id: 'closed', label: 'CLOSED' },
    { id: 'suppressed', label: 'SUPPRESSED' },
  ];

  // -- State --
  let alerts = $state<Alert[]>([]);
  let loading = $state(true);
  let activeTab = $state('open');
  let activeSevs = $state<string[]>(['critical', 'high']);
  let searchQuery = $state('');
  let selected = $state<Alert | null>(null);

  // -- Derived State --
  const filteredAlerts = $derived.by(() => {
    return alerts.filter(a => {
      const matchesTab = (activeTab === 'open' && (a.status === 'open' || a.status === 'acknowledged' || a.status === 'investigating')) || (a.status === activeTab);
      const matchesSev = activeSevs.includes(a.severity);
      const matchesSearch = a.event_type.toLowerCase().includes(searchQuery.toLowerCase()) || 
                           a.host_id.toLowerCase().includes(searchQuery.toLowerCase()) || 
                           a.id.toLowerCase().includes(searchQuery.toLowerCase());
      return matchesTab && matchesSev && matchesSearch;
    });
  });

  const metrics = $derived.by(() => {
    const total = alerts.filter(a => a.status !== 'closed').length;
    const critical = alerts.filter(a => a.severity === 'critical' && a.status !== 'closed').length;
    const unassigned = alerts.filter(a => !a.assigned_to && a.status !== 'closed').length;
    return { total, critical, unassigned };
  });

  // -- Actions --
  async function fetchAlerts() {
    loading = true;
    try {
      const res = await request<{ alerts: Alert[] }>('/alerts');
      // Mocking more detailed fields for the high-density view if they don't exist
      alerts = (res.alerts ?? []).map(a => ({
        ...a,
        severity: a.severity || (['critical', 'high', 'medium', 'low'][Math.floor(Math.random() * 4)] as any),
        status: a.status || (['open', 'investigating', 'acknowledged'][Math.floor(Math.random() * 3)] as any),
        risk_score: Math.floor(Math.random() * 40) + 60,
        sla_seconds: Math.floor(Math.random() * 3600),
        sla_pct: Math.floor(Math.random() * 100),
      }));
    } catch { 
      alerts = []; 
    } finally {
      loading = false;
    }
  }

  function toggleSev(sev: string) {
    if (activeSevs.includes(sev)) {
      activeSevs = activeSevs.filter(s => s !== sev);
    } else {
      activeSevs = [...activeSevs, sev];
    }
  }

  onMount(() => {
    fetchAlerts();
  });

  // -- Table Columns --
  const columns = [
    { key: 'severity', label: 'SEV', width: '80px', sortable: true },
    { key: 'event_type', label: 'ALERT', sortable: true },
    { key: 'host_id', label: 'HOST', width: '120px', sortable: true },
    { key: 'risk_score', label: 'RISK', width: '60px', sortable: true, align: 'center' as const },
    { key: 'sla_seconds', label: 'SLA', width: '100px', sortable: true },
    { key: 'status', label: 'STATUS', width: '100px', sortable: true },
    { key: 'actions', label: 'ACTIONS', width: '120px', sortable: false }
  ];

  function formatSLA(seconds: number) {
    const mins = Math.floor(seconds / 60);
    const secs = seconds % 60;
    return `${String(mins).padStart(2, '0')}:${String(secs).padStart(2, '0')}`;
  }
</script>

<PageLayout 
  title="Alert Management" 
  subtitle="Centralized event triage and response orchestration"
>
  {#snippet toolbar()}
    <div class="flex items-center gap-2">
      <div class="flex items-center gap-1.5 px-3 py-1 bg-surface-2 border border-border-primary rounded-sm mr-2 shadow-inner">
        <Activity size={12} class="text-accent animate-pulse" />
        <span class="text-[9px] font-mono font-bold uppercase tracking-widest text-text-muted">Live Queue: Active</span>
      </div>
      <Button variant="secondary" size="sm" onclick={fetchAlerts} icon={RefreshCw}>
        RE-SYNC
      </Button>
    </div>
  {/snippet}

  <div class="flex flex-col gap-4 h-full overflow-hidden">
    <!-- TOP METRICS -->
    <div class="grid grid-cols-5 gap-3 shrink-0">
      <KPI label="Total Open" value={metrics.total} trend="up" trendValue="+31 vs prev 4h" />
      <KPI label="Critical" value={metrics.critical} variant="critical" sublabel="SLA BREACH RISK" trend="up" trendValue="HIGH" />
      <KPI label="Unassigned" value={metrics.unassigned} variant="high" sublabel="Needs Triage" />
      <KPI label="Avg MTTR" value="14.2m" variant="success" trend="down" trendValue="-2.1m vs SLA" />
      <KPI label="False Positive" value="8.4%" sublabel="Last 7 days" />
    </div>

    <!-- MAIN TOOLBAR -->
    <div class="flex items-center gap-4 bg-surface-1 border border-border-primary p-2 rounded-sm shrink-0 shadow-sm">
      <Tabs tabs={STATUS_TABS} bind:active={activeTab} variant="pills" />
      
      <div class="h-6 w-px bg-border-primary mx-1"></div>

      <div class="flex items-center gap-1">
        {#each ['critical', 'high', 'medium', 'low'] as sev}
          <button 
            onclick={() => toggleSev(sev)}
            class="px-2 py-1 text-[9px] font-bold uppercase border transition-all rounded-xs
              {activeSevs.includes(sev) 
                ? 'bg-surface-3 border-accent text-accent shadow-[0_0_8px_rgba(26,127,212,0.15)]' 
                : 'bg-transparent border-border-subtle text-text-muted hover:border-text-secondary'}"
          >
            {sev.slice(0, 4)}
          </button>
        {/each}
      </div>

      <div class="relative flex-1 max-w-md ml-auto">
        <Search size={14} class="absolute left-3 top-1/2 -translate-y-1/2 text-text-muted" />
        <input 
          type="text" 
          bind:value={searchQuery}
          placeholder="Search host, ID, or alert type..." 
          class="w-full bg-surface-2 border border-border-primary rounded-sm pl-9 pr-3 py-1.5 text-xs font-mono text-text-secondary focus:border-accent focus:outline-none transition-all"
        />
      </div>

      <div class="flex gap-2">
        <Button variant="secondary" size="sm" class="px-4">ACK SELECTED</Button>
        <Button variant="secondary" size="sm" class="px-4 text-error hover:bg-error/10">FALSE POS.</Button>
      </div>
    </div>

    <!-- DATA TABLE -->
    <div class="flex-1 overflow-hidden min-h-0 bg-surface-1 rounded-sm border border-border-primary shadow-lg">
      {#if loading}
        <div class="h-full flex items-center justify-center">
          <Spinner size="lg" />
        </div>
      {:else}
        <DataTable 
          data={filteredAlerts} 
          columns={columns} 
          compact 
          onRowClick={(row: Alert) => selected = row}
          rowKey="id"
        >
          {#snippet cell({ value, column, row }: { value: any, column: any, row: Alert })}
            {#if column.key === 'severity'}
              {@const s = SEV_MAP[row.severity as keyof typeof SEV_MAP]}
              <div class="flex items-center gap-2">
                <div class="w-1.5 h-1.5 rounded-full {s.variant === 'danger' ? 'bg-error shadow-[0_0_6px_rgba(200,44,44,0.5)]' : s.variant === 'warning' ? 'bg-warning' : 'bg-success'}"></div>
                <Badge variant={s.variant as any} size="xs" class="font-bold">{s.label}</Badge>
              </div>
            {:else if column.key === 'event_type'}
              <div class="flex flex-col">
                <span class="font-bold text-text-heading">{value.replace(/_/g, ' ')}</span>
                <span class="text-[9px] text-text-muted font-mono">{row.id} · MITRE T1003.001</span>
              </div>
            {:else if column.key === 'host_id'}
              <span class="font-mono text-accent">{value || 'LOCAL'}</span>
            {:else if column.key === 'risk_score'}
              <div class="font-mono font-bold px-2 py-0.5 rounded-xs inline-block
                {value > 90 ? 'bg-error/15 text-error' : value > 70 ? 'bg-warning/15 text-warning' : 'bg-surface-3 text-text-muted'}">
                {value}
              </div>
            {:else if column.key === 'sla_seconds'}
              {@const slaColor = row.sla_pct > 90 ? 'bg-error' : row.sla_pct > 75 ? 'bg-warning' : row.sla_pct > 50 ? 'bg-accent' : 'bg-success'}
              <div class="flex flex-col gap-1 w-20">
                <div class="h-1 bg-surface-3 rounded-full overflow-hidden">
                  <div class="h-full {slaColor}" style="width: {row.sla_pct}%"></div>
                </div>
                <span class="text-[9px] font-mono {row.sla_pct > 90 ? 'text-error animate-pulse' : row.sla_pct > 75 ? 'text-warning' : 'text-text-muted'}">
                  {formatSLA(value)}
                </span>
              </div>
            {:else if column.key === 'status'}
              <Badge variant="secondary" size="xs" class="uppercase opacity-80">{value}</Badge>
            {:else if column.key === 'actions'}
              <div class="flex gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                <Button variant="secondary" size="xs" class="h-6 px-2 text-[8px] border-error/30 text-error hover:bg-error/10">ISO</Button>
                <Button variant="secondary" size="xs" class="h-6 px-2 text-[8px]">SIEM</Button>
                <Button variant="secondary" size="xs" class="h-6 px-2 text-[8px]">⋯</Button>
              </div>
            {:else}
              {value}
            {/if}
          {/snippet}
        </DataTable>
      {/if}
    </div>

    <!-- DETAIL DRAWER (Overlay when selected) -->
    {#if selected}
      <div class="fixed inset-y-0 right-0 w-[500px] bg-surface-1 border-l border-border-primary shadow-2xl z-50 flex flex-col transform transition-transform duration-300">
        <div class="p-4 border-b border-border-primary bg-surface-2 flex items-center justify-between shrink-0">
          <div class="flex items-center gap-3">
            <ShieldAlert size={16} class="text-error" />
            <h3 class="font-bold text-text-heading uppercase tracking-widest text-sm">Alert Detail</h3>
          </div>
          <Button variant="ghost" size="sm" onclick={() => selected = null}>×</Button>
        </div>
        
        <div class="flex-1 overflow-y-auto p-6 space-y-6">
          <div class="space-y-4">
            <div class="flex justify-between items-start">
              <div>
                <div class="flex items-center gap-2 mb-1">
                  <span class="text-[10px] font-mono text-text-muted">{selected.id}</span>
                  <Badge variant={SEV_MAP[selected.severity].variant} size="xs">{SEV_MAP[selected.severity].label}</Badge>
                </div>
                <h2 class="text-xl font-bold text-text-heading leading-tight">{selected.event_type.replace(/_/g, ' ')}</h2>
              </div>
              <div class="text-right">
                <div class="text-[10px] font-mono text-text-muted uppercase">Risk Score</div>
                <div class="text-2xl font-bold text-error font-mono">{selected.risk_score}</div>
              </div>
            </div>

            <!-- RISK ANATOMY (New from prototype) -->
            <div class="p-4 bg-surface-2 border border-border-primary rounded-sm space-y-3">
              <div class="text-[10px] font-bold text-text-muted uppercase tracking-widest border-b border-border-subtle pb-1">Scoring Factors (ML-Weighted)</div>
              <div class="space-y-2">
                {#each [
                  { label: 'IOC Confidence Match', val: 98, weight: 0.30 },
                  { label: 'MITRE Tactic Severity', val: 90, weight: 0.25 },
                  { label: 'UEBA Entity Score', val: 94, weight: 0.20 },
                  { label: 'Asset Criticality', val: 80, weight: 0.15 }
                ] as factor}
                  <div class="flex items-center gap-3">
                    <span class="text-[9px] text-text-muted flex-1">{factor.label}</span>
                    <div class="w-24 h-1 bg-surface-3 rounded-full overflow-hidden">
                      <div class="h-full bg-error" style="width: {factor.val}%"></div>
                    </div>
                    <span class="text-[9px] font-mono text-error/80 w-8 text-right">×{factor.weight.toFixed(2)}</span>
                  </div>
                {/each}
              </div>
            </div>

            <div class="grid grid-cols-2 gap-4">
              <div class="p-3 bg-surface-2 border border-border-primary rounded-sm space-y-1">
                <div class="text-[9px] font-mono text-text-muted uppercase flex items-center gap-1.5">
                  <Terminal size={10} /> Host Entity
                </div>
                <div class="text-xs font-bold text-accent">{selected.host_id}</div>
              </div>
              <div class="p-3 bg-surface-2 border border-border-primary rounded-sm space-y-1">
                <div class="text-[9px] font-mono text-text-muted uppercase flex items-center gap-1.5">
                  <User size={10} /> Identity
                </div>
                <div class="text-xs font-bold text-text-secondary">{selected.user || 'SYSTEM'}</div>
              </div>
            </div>

            <!-- BLAST RADIUS -->
            <div class="space-y-3">
               <div class="flex items-center justify-between">
                  <div class="flex items-center gap-2 text-[10px] font-black text-text-heading uppercase tracking-widest">
                     <Target size={14} class="text-error" />
                     Blast Radius (Tactical Graph)
                  </div>
                  <Badge variant="secondary" size="xs" class="font-mono">HOPS: 2</Badge>
               </div>
               <div class="h-56 w-full bg-surface-2 border border-border-primary rounded-sm relative overflow-hidden group">
                  <InvestigationCanvas 
                     nodes={[
                       { id: selected.id, type: 'process', label: selected.event_type.split('_')[0], meta: { criticality: 'high' } },
                       { id: selected.host_id, type: 'host', label: selected.host_id },
                       { id: 'user:victim', type: 'user', label: selected.user || 'SYSTEM' },
                       { id: 'ip:dest', type: 'ip', label: 'C2 Destination' }
                     ]}
                     edges={[
                       { from: selected.host_id, to: selected.id, type: 'parent' },
                       { from: 'user:victim', to: selected.host_id, type: 'context' },
                       { from: selected.id, to: 'ip:dest', type: 'outbound' }
                     ]}
                  />
                  <div class="absolute top-2 right-2 opacity-0 group-hover:opacity-100 transition-opacity">
                     <Button variant="secondary" size="xs" class="bg-surface-1/80 px-2" onclick={() => push('/investigation')}>EXPAND_FULL_GRAPH</Button>
                  </div>
               </div>
            </div>
          </div>

          <div class="space-y-2">
            <div class="text-[10px] font-bold text-text-muted uppercase tracking-widest flex items-center justify-between">
              Raw Telemetry
              <span class="text-[8px] font-mono opacity-50">TOPIC: SEC.EVENTS.L7</span>
            </div>
            <div class="bg-surface-0 border border-border-primary p-4 rounded-sm">
              <pre class="text-[11px] font-mono text-text-secondary whitespace-pre-wrap leading-relaxed">
                {JSON.stringify(selected, null, 2)}
              </pre>
            </div>
          </div>

          <div class="space-y-3">
            <div class="text-[10px] font-bold text-text-muted uppercase tracking-widest">Tactical Actions</div>
            <div class="grid grid-cols-2 gap-2">
              <Button variant="primary" class="w-full text-error border-error/30 hover:bg-error/10" icon={ShieldAlert}>ISOLATE HOST</Button>
              <Button variant="secondary" class="w-full" icon={Eye}>FORENSIC CAPTURE</Button>
              <Button variant="secondary" class="w-full" icon={Zap}>RUN PLAYBOOK</Button>
              <Button variant="secondary" class="w-full" icon={Globe}>SIEM PIVOT</Button>
            </div>
          </div>
        </div>

        <div class="p-4 border-t border-border-primary bg-surface-2 shrink-0">
          <div class="flex items-center justify-between mb-4">
             <span class="text-[10px] font-mono text-text-muted uppercase">Change Status</span>
             <Badge variant="secondary" size="xs" class="uppercase">{selected.status}</Badge>
          </div>
          <div class="flex gap-2">
            <Button variant="primary" class="flex-1" onclick={() => selected!.status = 'investigating'}>INVESTIGATE</Button>
            <Button variant="secondary" class="flex-1" onclick={() => selected!.status = 'acknowledged'}>ACKNOWLEDGE</Button>
          </div>
        </div>
      </div>
      <div 
        class="fixed inset-0 bg-black/40 backdrop-blur-sm z-40 transition-opacity" 
        role="presentation"
        onclick={() => selected = null}
        onkeydown={(e) => e.key === 'Escape' && (selected = null)}
      ></div>
    {/if}
  </div>
</PageLayout>

<style>
  :global(.group:hover .group-hover\:opacity-100) {
    opacity: 1;
  }
</style>
