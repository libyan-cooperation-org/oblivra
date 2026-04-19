<!-- OBLIVRA Web — Ransomware Center (Svelte 5) -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { KPI, Badge, Button, DataTable, PageLayout, Spinner, ProgressBar } from '@components/ui';
  import { ShieldAlert, ShieldCheck, Activity, Zap, Lock, Unlock, HardDrive, History, Flame, AlertCircle } from 'lucide-svelte';
  import { request } from '../services/api';

  // -- Types --
  interface RansomwareEvent {
    id: string;
    timestamp: string;
    host_id: string;
    type: 'entropy_spike' | 'canary_triggered' | 'shadow_copy_deleted' | 'mass_rename' | 'ransom_note';
    severity: 'critical' | 'high' | 'medium';
    details: string;
    isolated: boolean;
  }
  interface HostDefenseStatus {
    host_id: string;
    hostname: string;
    status: 'protected' | 'at_risk' | 'isolated' | 'active_threat';
    canary_count: number;
    last_scan: string;
    entropy_score?: number;
  }

  // -- State --
  let tab           = $state<'overview' | 'events' | 'hosts'>('overview');
  let events        = $state<RansomwareEvent[]>([]);
  let hosts         = $state<HostDefenseStatus[]>([]);
  let stats         = $state<Record<string, number>>({});
  let loading       = $state(true);
  let isolatingHost = $state<string | null>(null);
  let actionResult  = $state('');

  // -- Helpers --
  const sevColor = (sev: string) => {
    if (sev === 'critical') return 'var(--alert-critical)';
    if (sev === 'high') return 'var(--alert-high)';
    return 'var(--alert-medium)';
  };
  const hostColor = (status: string) => {
    if (status === 'protected') return 'var(--status-online)';
    if (status === 'at_risk') return 'var(--alert-medium)';
    if (status === 'isolated') return 'var(--alert-high)';
    return 'var(--alert-critical)';
  };

  const activeThreats = $derived(events.filter(e => !e.isolated).length);
  const atRiskHosts   = $derived(hosts.filter(h => h.status === 'at_risk' || h.status === 'active_threat').length);

  // -- Actions --
  async function fetchData() {
    loading = true;
    try {
      const [e, h, s] = await Promise.all([
        request<RansomwareEvent[]>('/ransomware/events?limit=50'),
        request<HostDefenseStatus[]>('/ransomware/hosts'),
        request<Record<string, number>>('/ransomware/stats')
      ]);
      events = e ?? [];
      hosts = h ?? [];
      stats = s ?? {};
    } catch (err) {
      console.error('Ransomware Data fetch failed', err);
    } finally {
      loading = false;
    }
  }

  async function isolateHost(hostId: string) {
    isolatingHost = hostId;
    try {
      await request('/ransomware/isolate', { method: 'POST', body: JSON.stringify({ host_id: hostId }) });
      actionResult = `✓ Host ${hostId} isolated successfully.`;
      fetchData();
    } catch (e: any) {
      actionResult = `✗ Error: ${e?.message ?? e}`;
    } finally {
      isolatingHost = null;
      setTimeout(() => actionResult = '', 5000);
    }
  }

  onMount(() => {
    fetchData();
  });
</script>

<PageLayout title="Ransomware Defusal" subtitle="Fleet-wide active countermeasures and automated data recovery orchestration">
  {#snippet toolbar()}
    <div class="flex items-center gap-2">
      <Button variant="danger" size="sm" icon={Flame} class="font-black italic tracking-tighter">WAR MODE</Button>
      <Button variant="secondary" size="sm" onclick={fetchData}>
        <History size={14} class="mr-2" />
        REFRESH
      </Button>
    </div>
  {/snippet}

  <div class="flex flex-col h-full gap-0 -m-6">
    <!-- CRITICAL ALERT BANNER (If threats exist) -->
    {#if activeThreats > 0}
      <div class="bg-alert-critical/20 border-b border-alert-critical p-4 flex items-center justify-between animate-pulse">
        <div class="flex items-center gap-4">
          <ShieldAlert size={24} class="text-alert-critical" />
          <div class="flex flex-col">
            <span class="text-xs font-black uppercase tracking-widest text-alert-critical">Active Encryption Threats Detected</span>
            <span class="text-[10px] font-mono text-text-secondary">{activeThreats} unmitigated events across {atRiskHosts} endpoints. Immediate isolation recommended.</span>
          </div>
        </div>
        <Button variant="danger" size="sm">GLOBAL FLEET ISOLATION</Button>
      </div>
    {/if}

    <!-- METRIC STRIP -->
    <div class="grid grid-cols-4 gap-px bg-border-primary border-b border-border-primary shrink-0">
        <div class="bg-surface-2 p-3">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">Active Threats</div>
            <div class="text-xl font-mono font-bold {activeThreats > 0 ? 'text-alert-critical' : 'text-status-online'}">{activeThreats}</div>
            <div class="text-[9px] mt-1 {activeThreats > 0 ? 'text-alert-critical italic' : 'text-status-online'}">
              {activeThreats > 0 ? '▲ Resolution required' : '✓ No active events'}
            </div>
        </div>
        <div class="bg-surface-2 p-3">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">At-Risk Nodes</div>
            <div class="text-xl font-mono font-bold {atRiskHosts > 0 ? 'text-alert-high' : 'text-text-heading'}">{atRiskHosts}</div>
            <div class="text-[9px] text-text-muted mt-1">Behavioral anomaly flag</div>
        </div>
        <div class="bg-surface-2 p-3">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">Entropy Ingest</div>
            <div class="text-xl font-mono font-bold text-accent-primary">1.4 TB/h</div>
            <div class="text-[9px] text-status-online mt-1">L7 Logic Depth Active</div>
        </div>
        <div class="bg-surface-2 p-3">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">Protected Hosts</div>
            <div class="text-xl font-mono font-bold text-status-online">{hosts.filter(h => h.status === 'protected').length}</div>
            <div class="text-[9px] text-status-online mt-1">Canary Tripwires Armed</div>
        </div>
    </div>

    <!-- MAIN BODY -->
    <div class="flex-1 flex min-h-0 bg-surface-0">
        <!-- LEFT: MAIN CONTENT -->
        <div class="flex-1 flex flex-col min-w-0 border-r border-border-primary overflow-hidden">
            <div class="bg-surface-1 border-b border-border-primary p-3 flex items-center justify-between shrink-0">
                <div class="flex items-center gap-4">
                    <div class="flex items-center gap-2">
                        <Zap size={14} class="text-alert-critical" />
                        <span class="text-[10px] font-mono font-bold uppercase tracking-widest text-text-heading">Ransomware Response Substrate</span>
                    </div>
                    
                    <div class="flex border border-border-primary rounded-sm overflow-hidden">
                      {#each ['overview', 'events', 'hosts'] as t}
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
                
                {#if actionResult}
                  <div class="text-[10px] font-mono font-bold {actionResult.startsWith('✓') ? 'text-status-online' : 'text-alert-critical'} animate-pulse">
                    {actionResult}
                  </div>
                {/if}
            </div>

            <div class="flex-1 overflow-auto">
              {#if loading}
                <div class="h-full flex items-center justify-center">
                  <Spinner />
                </div>
              {:else if tab === 'overview'}
                <div class="p-6 space-y-8">
                  <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
                    {#each [
                      { name: 'Entropy Monitor', status: 'active', icon: Activity, detail: 'Real-time file write analysis via L7 logic' },
                      { name: 'Canary Tripwires', status: 'active', icon: Zap, detail: 'Hidden decoy file clusters deployed' },
                      { name: 'Shadow Copy Guard', status: 'active', icon: HardDrive, detail: 'Kernel-level VSS deletion prevention' },
                      { name: 'Network Isolation', status: 'ready', icon: Lock, detail: 'Immediate L3/L4 sharding triggers' },
                      { name: 'Process Terminator', status: 'ready', icon: Flame, detail: 'Auto-kill logic for encryption threads' },
                      { name: 'Backup Snapshot', status: 'active', icon: History, detail: 'Immutable off-site snapshot triggers' },
                    ] as module}
                      <div class="bg-surface-2 border border-border-primary p-4 rounded-sm flex flex-col gap-3 group hover:border-accent-primary transition-colors">
                        <div class="flex justify-between items-start">
                          <div class="p-2 bg-surface-1 border border-border-subtle rounded-sm">
                            <module.icon size={16} class={module.status === 'active' ? 'text-status-online' : 'text-accent-primary'} />
                          </div>
                          <Badge variant={module.status === 'active' ? 'success' : 'info'} size="xs" dot>{module.status.toUpperCase()}</Badge>
                        </div>
                        <div class="flex flex-col">
                          <span class="text-[11px] font-black text-text-heading uppercase tracking-widest">{module.name}</span>
                          <span class="text-[10px] text-text-muted font-mono mt-1 opacity-60 leading-tight">{module.detail}</span>
                        </div>
                      </div>
                    {/each}
                  </div>

                  {#if Object.keys(stats).length > 0}
                    <div class="bg-surface-1 border border-border-primary rounded-sm overflow-hidden shadow-premium">
                      <div class="p-3 bg-surface-2 border-b border-border-primary text-[10px] font-bold uppercase tracking-widest text-text-muted">
                        Fleet-wide Statistical Shards
                      </div>
                      <div class="p-6 grid grid-cols-2 md:grid-cols-4 gap-6">
                        {#each Object.entries(stats) as [key, val]}
                          <div class="flex flex-col gap-1">
                            <span class="text-[20px] font-mono font-bold text-alert-critical italic">{val.toLocaleString()}</span>
                            <span class="text-[9px] font-mono text-text-muted uppercase tracking-tighter border-t border-border-subtle pt-1">{key.replace(/_/g, ' ')}</span>
                          </div>
                        {/each}
                      </div>
                    </div>
                  {/if}
                </div>
              {:else if tab === 'events'}
                <div class="p-4 space-y-3">
                  {#each events as evt}
                    <div class="bg-surface-1 border border-border-primary border-l-2 p-3 rounded-sm relative overflow-hidden group" style="border-left-color: {sevColor(evt.severity)}">
                      <div class="absolute -right-2 -bottom-2 opacity-[0.03] grayscale group-hover:scale-110 transition-transform duration-700">
                        <AlertCircle size={64} />
                      </div>
                      <div class="flex justify-between items-start mb-2 relative z-10">
                        <div class="flex flex-col gap-0.5">
                          <div class="flex items-center gap-2">
                             <span class="text-[10px] font-black uppercase tracking-widest" style="color: {sevColor(evt.severity)}">{evt.type.replace(/_/g, ' ')}</span>
                             <Badge variant="secondary" size="xs">{evt.severity.toUpperCase()}</Badge>
                          </div>
                          <div class="text-[9px] font-mono text-text-muted">HOST_ID: {evt.host_id}</div>
                        </div>
                        <span class="text-[9px] font-mono text-text-muted opacity-50">{new Date(evt.timestamp).toLocaleString()}</span>
                      </div>
                      <p class="text-[11px] text-text-secondary mb-3 relative z-10">{evt.details}</p>
                      
                      <div class="flex items-center gap-3 relative z-10">
                        {#if evt.isolated}
                          <Badge variant="warning" size="xs" icon={Lock}>HOST_ISOLATED</Badge>
                        {:else}
                          <Button variant="danger" size="xs" icon={Lock} onclick={() => isolateHost(evt.host_id)}>ISOLATE NOW</Button>
                        {/if}
                      </div>
                    </div>
                  {:else}
                    <div class="h-full flex flex-col items-center justify-center text-text-muted gap-4 opacity-40 py-20">
                      <ShieldCheck size={48} />
                      <p class="font-mono text-[10px] uppercase tracking-widest">No active ransomware threat signatures detected</p>
                    </div>
                  {/each}
                </div>
              {:else if tab === 'hosts'}
                <DataTable 
                  data={hosts} 
                  columns={[
                    { key: 'status', label: 'STATE', width: '120px' },
                    { key: 'hostname', label: 'ENTITY_IDENTIFIER' },
                    { key: 'canary_count', label: 'TRIPWIRES', width: '100px' },
                    { key: 'entropy_score', label: 'ENTROPY', width: '140px' },
                    { key: 'last_scan', label: 'L7_SCAN_TIME', width: '160px' },
                    { key: 'actions', label: 'REMEDIATION', width: '120px' }
                  ]} 
                  compact
                  rowKey="host_id"
                >
                  {#snippet cell({ column, row })}
                    {#if column.key === 'status'}
                      <div class="flex items-center gap-2">
                         <div class="w-1.5 h-1.5 rounded-full" style="background: {hostColor(row.status)}"></div>
                         <span class="text-[10px] font-black uppercase tracking-widest" style="color: {hostColor(row.status)}">{row.status.replace(/_/g, ' ')}</span>
                      </div>
                    {:else if column.key === 'hostname'}
                      <span class="text-[11px] font-bold text-text-heading">{row.hostname || row.host_id}</span>
                    {:else if column.key === 'canary_count'}
                      <span class="text-[10px] font-mono text-text-muted">{row.canary_count ?? '—'}</span>
                    {:else if column.key === 'entropy_score'}
                      {#if row.entropy_score !== undefined}
                        <div class="flex items-center gap-3 w-full">
                           <ProgressBar value={row.entropy_score * 10} color={row.entropy_score > 7 ? 'var(--alert-critical)' : 'var(--status-online)'} height="3px" />
                           <span class="text-[10px] font-mono font-bold w-10 text-right">{row.entropy_score.toFixed(2)}</span>
                        </div>
                      {:else}
                        <span class="text-[9px] font-mono text-text-muted opacity-20 uppercase tracking-widest">Calibrating</span>
                      {/if}
                    {:else if column.key === 'last_scan'}
                      <span class="text-[9px] font-mono text-text-muted uppercase tracking-tighter opacity-60">{row.last_scan ? new Date(row.last_scan).toLocaleString() : 'NEVER'}</span>
                    {:else if column.key === 'actions'}
                      {#if row.status !== 'isolated'}
                        <Button 
                          variant="ghost" 
                          size="xs" 
                          class="h-6 px-3 border border-alert-critical/20 text-alert-critical hover:bg-alert-critical/10" 
                          onclick={() => isolateHost(row.host_id)}
                          disabled={isolatingHost === row.host_id}
                        >
                          {isolatingHost === row.host_id ? '...' : 'ISOLATE'}
                        </Button>
                      {:else}
                        <Badge variant="secondary" size="xs">OFFLINE</Badge>
                      {/if}
                    {/if}
                  {/snippet}
                </DataTable>
              {/if}
            </div>
        </div>

        <!-- RIGHT: ADVISORY SIDEBAR -->
        <div class="w-80 bg-surface-1 flex flex-col shrink-0">
            <div class="px-3 py-2 bg-surface-2 border-b border-border-primary flex items-center gap-2">
                <ShieldCheck size={14} class="text-status-online" />
                <span class="text-[9px] font-mono font-bold uppercase tracking-widest text-text-heading">Defense Readiness</span>
            </div>
            
            <div class="p-4 space-y-6">
                <div class="space-y-4">
                  <div class="text-[9px] font-mono font-bold text-text-muted uppercase tracking-widest border-b border-border-subtle pb-2">Active Policies</div>
                  {#each [
                    { name: 'AUTO_ISOLATE_HIGH', val: 'ENABLED', color: 'status-online' },
                    { name: 'VSS_LOCKDOWN', val: 'ARMED', color: 'status-online' },
                    { name: 'ENTROPY_THRESHOLD', val: '7.82', color: 'accent-primary' },
                    { name: 'HONEYPOT_DENSITY', val: 'HIGH', color: 'status-online' }
                  ] as policy}
                    <div class="flex justify-between items-center text-[10px] font-mono">
                      <span class="text-text-muted uppercase tracking-tight">{policy.name}</span>
                      <span class="font-bold text-{policy.color} italic">{policy.val}</span>
                    </div>
                  {/each}
                </div>

                <div class="pt-4 border-t border-border-primary space-y-4">
                    <span class="text-[9px] font-mono font-bold text-text-muted uppercase tracking-widest">Remediation Pipelines</span>
                    <div class="bg-surface-2 border border-border-primary p-3 rounded-sm space-y-2 group hover:border-accent-primary cursor-pointer transition-colors">
                        <div class="flex items-center gap-2">
                            <History size={14} class="text-status-online" />
                            <span class="text-[10px] font-bold text-text-heading uppercase">Snapshot Vault</span>
                        </div>
                        <p class="text-[8px] text-text-muted font-mono leading-relaxed opacity-60">
                            Managed immutable backup shards available for immediate restoration. Last sync: 14m ago.
                        </p>
                    </div>
                    <div class="bg-surface-2 border border-border-primary p-3 rounded-sm space-y-2 group hover:border-alert-high cursor-pointer transition-colors opacity-50">
                        <div class="flex items-center gap-2">
                            <Unlock size={14} class="text-text-muted" />
                            <span class="text-[10px] font-bold text-text-heading uppercase">Decryption Bot</span>
                        </div>
                        <p class="text-[8px] text-text-muted font-mono leading-relaxed opacity-60">
                            Offline logic for known variant key negotiation. Currently dormant (No active variants).
                        </p>
                    </div>
                </div>
            </div>

            <div class="mt-auto border-t border-border-primary p-4 bg-surface-2">
                 <div class="flex items-center justify-between mb-2">
                    <span class="text-[9px] font-mono font-bold text-text-muted uppercase tracking-widest">Isolation Latency</span>
                    <Badge variant="accent" size="xs">142 ms</Badge>
                 </div>
                 <div class="text-[8px] font-mono text-text-muted space-y-1 opacity-60">
                    <div>Fleet Status: HOMOGENEOUS</div>
                    <div>Canaries Alive: 100%</div>
                    <div>Detections (24h): 0</div>
                 </div>
            </div>
        </div>
    </div>

    <!-- STATUS BAR -->
    <div class="bg-surface-2 border-t border-border-primary px-3 py-1 flex items-center gap-4 text-[8px] font-mono text-text-muted shrink-0 uppercase tracking-widest">
        <div class="flex items-center gap-1.5">
            <div class="w-1 h-1 rounded-full bg-status-online"></div>
            <span>ENTROPY_ANALYSIS:</span>
            <span class="text-status-online font-bold italic">OPTIMIZED</span>
        </div>
        <span class="text-border-primary opacity-30">|</span>
        <div class="flex items-center gap-1.5">
            <span>ISOLATION_FABRIC:</span>
            <span class="text-status-online font-bold italic">STANDBY</span>
        </div>
        <span class="text-border-primary opacity-30">|</span>
        <div class="flex items-center gap-1.5">
            <span>TPM_ATTESTATION:</span>
            <span class="text-status-online font-bold italic">SIGNED</span>
        </div>
        <div class="ml-auto opacity-40">OBLIVRA_RANSOM_SHIELD v2.1.0</div>
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
