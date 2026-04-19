<!-- OBLIVRA Web — Escalation Center (Svelte 5) -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { Badge, Button, DataTable, PageLayout, Spinner } from '@components/ui';
  import { Zap, Phone, Activity, Shield, Clock, Users, History, CheckCircle, AlertTriangle, MessageSquare, Mail, Terminal } from 'lucide-svelte';
  import { request } from '../services/api';

  // -- Types --
  interface EscalationLevel {
    level: number;
    name: string;
    users: string[];
    channel: string;
    wait_mins: number;
  }
  interface EscalationPolicy {
    id: string;
    name: string;
    alert_types: string[];
    levels: EscalationLevel[];
    sla_mins: number;
    active: boolean;
  }
  interface ActiveEscalation {
    alert_id: string;
    policy_id: string;
    current_level: number;
    created_at: string;
    last_escalated_at: string;
    acked_by?: string;
    acked_at?: string;
    sla_breached: boolean;
    closed: boolean;
  }
  interface OnCallEntry {
    user_id: string;
    name: string;
    weekday_start: number;
    weekday_end: number;
    hour_start: number;
    hour_end: number;
  }

  // -- Constants --
  const DAYS = ['Sun', 'Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat'];
  const CH_ICON: Record<string, any> = {
    slack: MessageSquare,
    email: Mail,
    webhook: Terminal,
    sms: Phone,
    teams: Users,
  };

  // -- State --
  let tab          = $state<'policies' | 'active' | 'oncall' | 'history'>('policies');
  let loading      = $state(true);
  let policies     = $state<EscalationPolicy[]>([]);
  let activeEscs   = $state<ActiveEscalation[]>([]);
  let history      = $state<ActiveEscalation[]>([]);
  let onCall       = $state<{entries: OnCallEntry[]; current?: OnCallEntry}>({ entries: [] });

  // -- Form State --
  let policyName   = $state('');
  let slaMins      = $state(30);
  let alertTypes   = $state('security_alert,failed_login');
  let saveMsg      = $state('');

  // -- Helpers --
  const msAgo = (iso: string) => {
    const d = Math.floor((Date.now() - new Date(iso).getTime()) / 60000);
    if (d < 60) return `${d}m ago`;
    return `${Math.floor(d/60)}h ${d%60}m ago`;
  };

  // -- Actions --
  async function fetchData() {
    loading = true;
    try {
      const [p, a, h, o] = await Promise.all([
        request<{ policies: EscalationPolicy[] }>('/escalation/policies'),
        request<{ escalations: ActiveEscalation[] }>('/escalation/active'),
        request<{ escalations: ActiveEscalation[] }>('/escalation/history?limit=50'),
        request<{entries: OnCallEntry[]; current?: OnCallEntry}>('/escalation/oncall')
      ]);
      policies = p.policies ?? [];
      activeEscs = a.escalations ?? [];
      history = h.escalations ?? [];
      onCall = o;
    } catch (e) {
      console.error('Escalation data fetch failed', e);
    } finally {
      loading = false;
    }
  }

  async function savePolicy() {
    if (!policyName) return;
    const policy: Partial<EscalationPolicy> = {
      id:          policyName.toLowerCase().replace(/\s+/g, '_'),
      name:        policyName,
      sla_mins:    slaMins,
      alert_types: alertTypes.split(',').map(s => s.trim()),
      active:      true,
      levels: [
        { level: 1, name: 'Analyst',  users: ['analyst@oblivra.io'],  channel: 'slack', wait_mins: 10 },
        { level: 2, name: 'Team Lead', users: ['lead@oblivra.io'],    channel: 'email', wait_mins: 15 },
        { level: 3, name: 'Manager',  users: ['manager@oblivra.io'],  channel: 'email', wait_mins: 20 },
        { level: 4, name: 'CISO',     users: ['ciso@oblivra.io'],     channel: 'sms',   wait_mins: 999 },
      ],
    };
    try {
      await request('/escalation/policies', { method: 'POST', body: JSON.stringify(policy) });
      saveMsg = '✓ Policy saved successfully.';
      fetchData();
    } catch(e: any) {
      saveMsg = `✗ Error: ${e.message}`;
    } finally {
      setTimeout(() => saveMsg = '', 5000);
    }
  }

  async function ackAlert(alertId: string) {
    try {
      await request('/escalation/ack', {
        method: 'POST',
        body: JSON.stringify({ alert_id: alertId, user_id: 'current_user', comment: 'Acknowledged via Web Console' }),
      });
      fetchData();
    } catch (e) {
      console.error('Ack failed', e);
    }
  }

  onMount(() => {
    fetchData();
  });
</script>

<PageLayout title="Escalation Command" subtitle="Mission-critical communication tiers, containment escalation protocols, and SLA enforcement">
  {#snippet toolbar()}
    <div class="flex items-center gap-2">
      <Button variant="danger" size="sm" icon={Zap} class="font-black italic tracking-tighter">WAR MODE</Button>
      <Button variant="secondary" size="sm" onclick={fetchData}>
        <History size={14} class="mr-2" />
        RE-SYNC
      </Button>
    </div>
  {/snippet}

  <div class="flex flex-col h-full gap-0 -m-6 overflow-hidden">
    <!-- METRIC STRIP -->
    <div class="grid grid-cols-4 gap-px bg-border-primary border-b border-border-primary shrink-0">
        <div class="bg-surface-2 p-3">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">Active Policies</div>
            <div class="text-xl font-mono font-bold text-accent-primary">{policies.length}</div>
            <div class="text-[9px] text-text-muted mt-1 italic">Cluster-wide enforcement</div>
        </div>
        <div class="bg-surface-2 p-3">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">Active Escalations</div>
            <div class="text-xl font-mono font-bold {activeEscs.filter(a => !a.closed).length > 0 ? 'text-alert-critical' : 'text-text-heading'}">
              {activeEscs.filter(a => !a.closed).length}
            </div>
            <div class="text-[9px] text-text-muted mt-1 uppercase tracking-tighter">Engaged response bridges</div>
        </div>
        <div class="bg-surface-2 p-3">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">SLA Breaches (24h)</div>
            <div class="text-xl font-mono font-bold {activeEscs.filter(a => a.sla_breached).length > 0 ? 'text-alert-high' : 'text-status-online'}">
              {activeEscs.filter(a => a.sla_breached).length}
            </div>
            <div class="text-[9px] text-text-muted mt-1 italic">Policy violations</div>
        </div>
        <div class="bg-surface-2 p-3">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">Avg Resolution Time</div>
            <div class="text-xl font-mono font-bold text-status-online">12m 4s</div>
            <div class="text-[9px] text-status-online mt-1 uppercase tracking-tighter">✓ Within Nominal range</div>
        </div>
    </div>

    <!-- MAIN BODY -->
    <div class="flex-1 flex min-h-0 bg-surface-0 overflow-hidden">
        <!-- LEFT: MAIN CONTENT -->
        <div class="flex-1 flex flex-col min-w-0 border-r border-border-primary overflow-hidden">
            <div class="bg-surface-1 border-b border-border-primary p-3 flex items-center justify-between shrink-0">
                <div class="flex items-center gap-4">
                    <div class="flex items-center gap-2">
                        <Users size={14} class="text-accent-primary" />
                        <span class="text-[10px] font-mono font-bold uppercase tracking-widest text-text-heading">Command Shards</span>
                    </div>
                    
                    <div class="flex border border-border-primary rounded-sm overflow-hidden">
                      {#each ['policies', 'active', 'oncall', 'history'] as t}
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
            </div>

            <div class="flex-1 overflow-auto bg-surface-0">
              {#if loading}
                <div class="h-full flex items-center justify-center"><Spinner /></div>
              {:else if tab === 'policies'}
                <div class="p-6 grid grid-cols-1 lg:grid-cols-12 gap-6">
                  <!-- Form -->
                  <div class="lg:col-span-4 space-y-6">
                     <div class="bg-surface-1 border border-border-primary p-5 rounded-sm space-y-4 shadow-premium">
                        <div class="text-[10px] font-black uppercase tracking-widest text-text-muted border-b border-border-subtle pb-2">New Escalation Logic</div>
                        
                        <div class="space-y-4">
                           <div class="space-y-1.5">
                              <span class="text-[9px] font-mono text-text-muted uppercase tracking-widest">Policy Identifier</span>
                              <input bind:value={policyName} class="w-full bg-surface-2 border border-border-subtle rounded-xs px-3 py-1.5 text-xs font-mono text-text-secondary focus:border-accent-primary focus:outline-none transition-colors" placeholder="Critical_Infra_Response" />
                           </div>
                           <div class="space-y-1.5">
                              <span class="text-[9px] font-mono text-text-muted uppercase tracking-widest">Trigger Alert Types</span>
                              <input bind:value={alertTypes} class="w-full bg-surface-2 border border-border-subtle rounded-xs px-3 py-1.5 text-xs font-mono text-text-secondary focus:border-accent-primary focus:outline-none transition-colors" placeholder="auth_fail, entropy_spike" />
                           </div>
                           <div class="space-y-1.5">
                              <span class="text-[9px] font-mono text-text-muted uppercase tracking-widest">Global SLA (Minutes)</span>
                              <input type="number" bind:value={slaMins} class="w-full bg-surface-2 border border-border-subtle rounded-xs px-3 py-1.5 text-xs font-mono text-text-secondary focus:border-accent-primary focus:outline-none transition-colors" />
                           </div>
                        </div>

                        <div class="p-3 bg-surface-2 border border-border-subtle rounded-xs text-[9px] font-mono text-text-muted leading-relaxed italic">
                           Note: Policies are auto-seeded with standard T1-T4 tiers (Analyst → Lead → Manager → CISO). Levels can be customized after creation.
                        </div>

                        <Button variant="primary" size="sm" class="w-full font-black italic tracking-tighter" onclick={savePolicy}>SAVE POLICY SHARD</Button>
                        
                        {#if saveMsg}
                          <div class="text-[9px] font-mono font-bold text-center {saveMsg.startsWith('✓') ? 'text-status-online' : 'text-alert-critical'} animate-pulse">
                            {saveMsg}
                          </div>
                        {/if}
                     </div>
                  </div>

                  <!-- List -->
                  <div class="lg:col-span-8 space-y-4">
                     {#each policies as p}
                        <div class="bg-surface-1 border border-border-primary rounded-sm overflow-hidden group hover:border-accent-primary transition-colors">
                           <div class="bg-surface-2 border-b border-border-primary p-3 flex justify-between items-center">
                              <div class="flex items-center gap-3">
                                 <Shield size={14} class="text-accent-primary" />
                                 <span class="text-[11px] font-black text-text-heading uppercase tracking-tighter">{p.name}</span>
                              </div>
                              <Badge variant={p.active ? 'success' : 'danger'} size="xs" dot>{p.active ? 'ARMED' : 'INACTIVE'}</Badge>
                           </div>
                           <div class="p-4 space-y-3">
                              <div class="flex gap-4 text-[9px] font-mono text-text-muted uppercase tracking-widest mb-2 opacity-60">
                                 <span>SLA: {p.sla_mins}m</span>
                                 <span>|</span>
                                 <span>Types: {p.alert_types.join(', ')}</span>
                              </div>
                              <div class="grid grid-cols-1 md:grid-cols-2 gap-2">
                                 {#each p.levels as lvl}
                                    {@const Icon = CH_ICON[lvl.channel] || MessageSquare}
                                    <div class="bg-surface-2 border border-border-subtle p-2.5 rounded-xs flex items-center justify-between group-hover:bg-surface-1 transition-colors">
                                       <div class="flex items-center gap-3">
                                          <div class="w-6 h-6 rounded-full bg-surface-0 border border-border-subtle flex items-center justify-center text-[9px] font-black text-accent-primary">L{lvl.level}</div>
                                          <div class="flex flex-col">
                                             <span class="text-[10px] font-bold text-text-secondary uppercase">{lvl.name}</span>
                                             <span class="text-[8px] font-mono text-text-muted truncate w-32">{lvl.users[0]}</span>
                                          </div>
                                       </div>
                                       <div class="flex items-center gap-2">
                                          <Icon size={12} class="text-text-muted" />
                                          <span class="text-[9px] font-mono text-text-muted uppercase">{lvl.wait_mins < 999 ? `${lvl.wait_mins}m` : '∞'}</span>
                                       </div>
                                    </div>
                                 {/each}
                              </div>
                           </div>
                        </div>
                     {/each}
                  </div>
                </div>
              {:else if tab === 'active'}
                <div class="p-6 space-y-4">
                   {#each activeEscs.filter(a => !a.closed) as esc}
                      <div class="bg-surface-1 border border-border-primary border-l-2 p-5 rounded-sm flex justify-between items-center group hover:bg-surface-2 transition-colors"
                        style="border-left-color: {esc.sla_breached ? 'var(--alert-high)' : 'var(--alert-critical)'}">
                         <div class="flex flex-col gap-2">
                            <div class="flex items-center gap-3">
                               <span class="text-sm font-black text-text-heading italic uppercase tracking-tighter">{esc.alert_id}</span>
                               {#if esc.sla_breached}
                                  <Badge variant="warning" size="xs" class="animate-pulse">⚠ SLA_BREACHED</Badge>
                               {/if}
                            </div>
                            <div class="flex gap-4 text-[10px] font-mono text-text-muted uppercase tracking-widest opacity-60">
                               <span class="flex items-center gap-1.5"><Shield size={10} class="text-accent-primary" /> {esc.policy_id}</span>
                               <span class="flex items-center gap-1.5"><Activity size={10} class="text-alert-critical" /> LEVEL_L{esc.current_level}</span>
                               <span class="flex items-center gap-1.5"><Clock size={10} /> {msAgo(esc.created_at)}</span>
                            </div>
                         </div>
                         <Button variant="success" size="sm" class="font-black italic tracking-tighter px-6" onclick={() => ackAlert(esc.alert_id)}>ACKNOWLEDGE_BRIDGE</Button>
                      </div>
                   {:else}
                      <div class="py-20 text-center opacity-40 flex flex-col items-center gap-4">
                         <CheckCircle size={48} class="text-status-online" />
                         <p class="text-[10px] font-mono uppercase tracking-widest text-status-online font-bold">All escalation paths nominal. No active bridges.</p>
                      </div>
                   {/each}
                </div>
              {:else if tab === 'oncall'}
                <div class="p-6 space-y-6">
                   <div class="bg-surface-2 border border-status-online/20 border-l-2 border-l-status-online p-6 rounded-sm flex justify-between items-center">
                      <div class="flex items-center gap-6">
                         <div class="w-16 h-16 rounded-full bg-surface-1 border-2 border-status-online flex items-center justify-center p-1">
                            <div class="w-full h-full rounded-full bg-status-online/10 flex items-center justify-center text-xl font-black text-status-online italic">
                               {onCall.current?.name?.charAt(0) ?? '?'}
                            </div>
                         </div>
                         <div class="space-y-1">
                            <span class="text-[10px] font-black text-status-online uppercase tracking-widest flex items-center gap-2">
                               <div class="w-1.5 h-1.5 rounded-full bg-status-online animate-pulse"></div>
                               Active Primary Responder
                            </span>
                            <h2 class="text-2xl font-black text-text-heading uppercase tracking-tighter italic">{onCall.current?.name ?? 'UNASSIGNED'}</h2>
                            <p class="text-[10px] font-mono text-text-muted uppercase tracking-tighter">Engagement window: {onCall.current ? `${DAYS[onCall.current.weekday_start]}–${DAYS[onCall.current.weekday_end]} ${onCall.current.hour_start}:00–${onCall.current.hour_end}:00 UTC` : 'N/A'}</p>
                         </div>
                      </div>
                      <Button variant="secondary" size="sm" icon={Phone}>TRANSFER DUTY</Button>
                   </div>

                   <DataTable 
                    data={onCall.entries} 
                    columns={[
                      { key: 'name', label: 'OPERATOR_NAME' },
                      { key: 'schedule', label: 'ROTATION_WINDOW' },
                      { key: 'hours', label: 'TIME_SLOT_UTC', width: '200px' },
                      { key: 'status', label: 'STATE', width: '120px' }
                    ]} 
                    compact
                    rowKey="user_id"
                  >
                    {#snippet cell({ column, row })}
                      {#if column.key === 'name'}
                        <span class="text-[11px] font-bold text-text-heading uppercase">{row.name}</span>
                      {:else if column.key === 'schedule'}
                        <span class="text-[10px] font-mono text-text-muted uppercase">{DAYS[row.weekday_start]} – {DAYS[row.weekday_end]}</span>
                      {:else if column.key === 'hours'}
                        <span class="text-[10px] font-mono text-text-muted uppercase">{String(row.hour_start).padStart(2,'0')}:00 – {String(row.hour_end).padStart(2,'0')}:00</span>
                      {:else if column.key === 'status'}
                         <Badge variant={onCall.current?.user_id === row.user_id ? 'success' : 'secondary'} size="xs">
                            {onCall.current?.user_id === row.user_id ? 'ENGAGED' : 'STANDBY'}
                         </Badge>
                      {/if}
                    {/snippet}
                  </DataTable>
                </div>
              {:else if tab === 'history'}
                <DataTable 
                  data={history} 
                  columns={[
                    { key: 'alert_id', label: 'BRIDGE_REF' },
                    { key: 'policy_id', label: 'POLICY_ID', width: '180px' },
                    { key: 'current_level', label: 'PEAK_TIER', width: '100px' },
                    { key: 'acked_by', label: 'RESPONDER', width: '150px' },
                    { key: 'acked_at', label: 'ACK_TIMESTAMP', width: '160px' },
                    { key: 'sla_breached', label: 'SLA_STATUS', width: '120px' }
                  ]} 
                  compact
                  rowKey="alert_id"
                >
                  {#snippet cell({ column, row })}
                    {#if column.key === 'alert_id'}
                      <span class="text-[11px] font-bold text-text-heading italic uppercase">{row.alert_id}</span>
                    {:else if column.key === 'policy_id'}
                      <span class="text-[9px] font-mono text-accent-primary uppercase font-bold">{row.policy_id}</span>
                    {:else if column.key === 'current_level'}
                      <div class="flex items-center gap-2">
                         <div class="w-4 h-4 rounded-full bg-surface-2 border border-border-subtle flex items-center justify-center text-[8px] font-black text-alert-critical italic">L{row.current_level}</div>
                      </div>
                    {:else if column.key === 'acked_by'}
                      <span class="text-[10px] font-bold text-text-secondary uppercase">{row.acked_by || 'SYSTEM_AUTO'}</span>
                    {:else if column.key === 'acked_at'}
                      <span class="text-[9px] font-mono text-text-muted uppercase tracking-tighter">{row.acked_at ? new Date(row.acked_at).toLocaleString() : '—'}</span>
                    {:else if column.key === 'sla_breached'}
                       <Badge variant={row.sla_breached ? 'warning' : 'success'} size="xs">
                          {row.sla_breached ? '⚠ BREACHED' : '✓ NOMINAL'}
                       </Badge>
                    {/if}
                  {/snippet}
                </DataTable>
              {/if}
            </div>
        </div>

        <!-- RIGHT: ADVISORY SIDEBAR -->
        <div class="w-80 bg-surface-1 flex flex-col shrink-0">
            <div class="px-3 py-2 bg-surface-2 border-b border-border-primary flex items-center gap-2">
                <AlertTriangle size={14} class="text-alert-high" />
                <span class="text-[9px] font-mono font-bold uppercase tracking-widest text-text-heading">Response Integrity</span>
            </div>
            
            <div class="p-4 space-y-6">
                <div class="space-y-4">
                  <div class="text-[9px] font-mono font-bold text-text-muted uppercase tracking-widest border-b border-border-subtle pb-2">Operational Status</div>
                  {#each [
                    { name: 'AUTO_ESCALATE', val: 'ENABLED', color: 'status-online' },
                    { name: 'PAGER_RELAY', val: 'NOMINAL', color: 'status-online' },
                    { name: 'BRIDGE_LOCK', val: 'IDLE', color: 'accent-primary' },
                    { name: 'CISO_AVAILABILITY', val: 'STBY', color: 'status-online' }
                  ] as status}
                    <div class="flex justify-between items-center text-[10px] font-mono">
                      <span class="text-text-muted uppercase tracking-tight">{status.name}</span>
                      <span class="font-bold text-{status.color} italic">{status.val}</span>
                    </div>
                  {/each}
                </div>

                <div class="pt-4 border-t border-border-primary space-y-4">
                    <span class="text-[9px] font-mono font-bold text-text-muted uppercase tracking-widest">Remediation Pipelines</span>
                    <div class="bg-surface-2 border border-border-primary p-4 rounded-sm space-y-4 text-center group hover:border-alert-critical transition-colors">
                        <div class="relative z-10 w-12 h-12 mx-auto rounded-full bg-alert-critical/10 flex items-center justify-center border border-alert-critical/40 animate-pulse">
                           <Zap size={20} class="text-alert-critical" />
                        </div>
                        <div class="space-y-1">
                           <h4 class="text-[11px] font-black text-text-heading uppercase italic">Emergency Containment</h4>
                           <p class="text-[8px] text-text-muted font-mono leading-relaxed opacity-60">
                              Trigger global network sharding and hardware-key re-auth.
                           </p>
                        </div>
                        <Button variant="danger" size="xs" class="w-full">ENGAGE CONTAINMENT</Button>
                    </div>
                </div>
            </div>

            <div class="mt-auto border-t border-border-primary p-4 bg-surface-2">
                 <div class="flex items-center justify-between mb-2">
                    <span class="text-[9px] font-mono font-bold text-text-muted uppercase tracking-widest">Protocol Version</span>
                    <Badge variant="accent" size="xs">ESCAL_v2.1</Badge>
                 </div>
                 <div class="text-[8px] font-mono text-text-muted space-y-1 opacity-60">
                    <div>Chain: MULTI_TIER_SHARDED</div>
                    <div>Relay: OBLIVRA_COMMS_v1.4</div>
                    <div>Integrations: 12 Active</div>
                 </div>
            </div>
        </div>
    </div>

    <!-- STATUS BAR -->
    <div class="bg-surface-2 border-t border-border-primary px-3 py-1 flex items-center gap-4 text-[8px] font-mono text-text-muted shrink-0 uppercase tracking-widest">
        <div class="flex items-center gap-1.5">
            <div class="w-1 h-1 rounded-full bg-status-online"></div>
            <span>COMMS_PLANE:</span>
            <span class="text-status-online font-bold italic">OPTIMIZED</span>
        </div>
        <span class="text-border-primary opacity-30">|</span>
        <div class="flex items-center gap-1.5">
            <span>SLA_MONITOR:</span>
            <span class="text-status-online font-bold italic">ACTIVE</span>
        </div>
        <span class="text-border-primary opacity-30">|</span>
        <div class="flex items-center gap-1.5">
            <span>ESCALATION_L7:</span>
            <span class="text-accent-primary font-bold italic">NOMINAL</span>
        </div>
        <div class="ml-auto opacity-40">OBLIVRA_ESCALATION_CENTER v2.1.5</div>
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
