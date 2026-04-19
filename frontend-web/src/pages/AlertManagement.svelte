<!-- OBLIVRA Web — Alert Management (Svelte 5) -->
<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { PageLayout, Badge, Button, Spinner } from '@components/ui';
  import { 
    Bell, 
    RefreshCw, 
    Filter, 
    ShieldAlert, 
    ShieldCheck, 
    Search, 
    Clock, 
    Activity, 
    Terminal, 
    User, 
    Globe,
    AlertTriangle,
    CheckCircle,
    Eye
  } from 'lucide-svelte';
  import { request } from '../services/api';
  import { wsStream } from '../lib/stores/websocket.svelte';

  // -- Types --
  interface Alert { 
    id: number; 
    tenant_id: string; 
    host_id: string; 
    timestamp: string; 
    event_type: string; 
    source_ip: string; 
    user: string; 
    raw_log: string; 
    status?: Status; 
  }
  type Status = 'new' | 'investigating' | 'acknowledged' | 'closed';

  // -- Constants --
  const SEV: Record<string, { label: string; variant: any }> = {
    security_alert:   { label: 'CRITICAL', variant: 'danger' },
    failed_login:     { label: 'HIGH',     variant: 'warning' },
    sudo_exec:        { label: 'MEDIUM',   variant: 'warning' },
    successful_login: { label: 'INFO',     variant: 'success' },
  };

  const statusColor: Record<Status, string> = {
    new:            'var(--alert-critical)',
    investigating:  'var(--alert-medium)',
    acknowledged:   'var(--status-online)',
    closed:         'var(--text-muted)'
  };

  // -- State --
  let alerts     = $state<Alert[]>([]);
  let loading    = $state(true);
  let localSt    = $state<Record<number, Status>>({});
  let filter     = $state<Status | 'all'>('all');
  let selected   = $state<Alert | null>(null);
  let liveCount  = $state(0);

  // -- Helpers --
  const getSev = (t: string) => SEV[t] ?? { label: 'LOW', variant: 'secondary' };
  const fmtDate = (iso: string) => new Date(iso).toLocaleString();
  
  const displayed = $derived.by(() => {
    const base = alerts.map(a => ({ ...a, status: localSt[a.id] ?? a.status ?? 'new' }));
    return filter === 'all' ? base : base.filter(a => a.status === filter);
  });

  // -- Actions --
  async function fetchAlerts() {
    loading = true;
    try {
      const res = await request<{ alerts: Alert[] }>('/alerts');
      alerts = (res.alerts ?? []).map(a => ({ ...a, status: 'new' as Status }));
    } catch { 
      alerts = []; 
    } finally {
      loading = false;
    }
  }

  function setStatus(id: number, s: Status) {
    localSt = { ...localSt, [id]: s };
    if (selected?.id === id) selected = { ...selected, status: s };
  }

  onMount(() => {
    fetchAlerts();
    // In a real app, we'd subscribe to wsStream messages here
  });
</script>

<PageLayout title="Alert Command" subtitle="Centralized event triage, real-time threat ingestion, and response state orchestration">
  {#snippet toolbar()}
    <div class="flex items-center gap-2">
      <div class="flex items-center gap-1.5 px-3 py-1 bg-surface-2 border border-border-primary rounded-sm mr-2">
        <Activity size={12} class="text-accent-primary animate-pulse" />
        <span class="text-[9px] font-mono font-bold uppercase tracking-widest text-text-muted">Stream: Active</span>
      </div>
      <Button variant="secondary" size="sm" onclick={fetchAlerts}>
        <RefreshCw size={14} class="mr-2" />
        RE-SYNC
      </Button>
    </div>
  {/snippet}

  <div class="flex h-full gap-0 -m-6 overflow-hidden">
    <!-- LEFT: ALERT LIST -->
    <div class="w-[450px] flex flex-col border-r border-border-primary bg-surface-1 shrink-0 overflow-hidden">
      <!-- Search & Filters -->
      <div class="p-4 border-b border-border-primary space-y-4 shrink-0">
        <div class="flex items-center gap-2 bg-surface-2 border border-border-subtle rounded-sm px-3 py-1.5 group focus-within:border-accent-primary transition-colors">
          <Search size={14} class="text-text-muted group-focus-within:text-accent-primary transition-colors" />
          <input 
            type="text" 
            placeholder="Search alerts..." 
            class="bg-transparent border-none outline-none text-xs font-mono text-text-secondary w-full"
          />
        </div>

        <div class="flex gap-1">
          {#each ['all', 'new', 'investigating', 'acknowledged', 'closed'] as f}
            <button 
              class="px-2 py-1 text-[9px] font-black uppercase tracking-tighter border transition-all rounded-xs flex-1
                {filter === f 
                  ? 'bg-accent-primary border-accent-primary text-black' 
                  : 'bg-surface-0 border-border-subtle text-text-muted hover:border-text-secondary'}"
              onclick={() => filter = f as any}
            >
              {f}
            </button>
          {/each}
        </div>
      </div>

      <!-- Scrollable List -->
      <div class="flex-1 overflow-y-auto">
        {#if loading}
          <div class="h-full flex items-center justify-center p-12">
            <Spinner />
          </div>
        {:else if displayed.length === 0}
          <div class="h-full flex flex-col items-center justify-center p-12 text-center opacity-40 gap-3">
            <ShieldCheck size={48} class="text-status-online" />
            <p class="text-[10px] font-mono uppercase tracking-widest text-status-online">Operational Calm: No alerts pending</p>
          </div>
        {:else}
          {#each displayed as a (a.id)}
            {@const s = getSev(a.event_type)}
            {@const status = localSt[a.id] ?? a.status ?? 'new'}
            <button 
              class="w-full text-left p-4 border-b border-border-subtle hover:bg-surface-2 transition-all relative group overflow-hidden
                {selected?.id === a.id ? 'bg-surface-2 border-l-2' : 'bg-transparent border-l-2 border-l-transparent'}"
              style="border-left-color: {selected?.id === a.id ? 'var(--accent-primary)' : 'transparent'}"
              onclick={() => selected = a}
            >
              <div class="flex justify-between items-start mb-2">
                <Badge variant={s.variant} size="xs" class="font-black italic">{s.label}</Badge>
                <div class="flex items-center gap-1.5">
                  <div class="w-1.5 h-1.5 rounded-full" style="background: {statusColor[status]}"></div>
                  <span class="text-[9px] font-mono font-bold uppercase tracking-tighter" style="color: {statusColor[status]}">{status}</span>
                </div>
              </div>

              <div class="text-[11px] font-black text-text-heading uppercase tracking-tighter mb-1 line-clamp-1">{a.event_type.replace(/_/g, ' ')}</div>
              
              <div class="flex items-center gap-4 text-[9px] font-mono text-text-muted uppercase tracking-widest opacity-60">
                <span class="flex items-center gap-1"><Terminal size={10} /> {a.host_id || 'LOCAL'}</span>
                <span class="flex items-center gap-1"><Clock size={10} /> {a.timestamp.split('T')[1].slice(0, 5)}</span>
              </div>
            </button>
          {/each}
        {/if}
      </div>
    </div>

    <!-- RIGHT: ALERT DETAIL -->
    <div class="flex-1 bg-surface-0 flex flex-col overflow-hidden">
      {#if !selected}
        <div class="h-full flex flex-col items-center justify-center text-center opacity-20 gap-4">
          <Bell size={64} />
          <p class="text-xs font-mono uppercase tracking-[0.3em]">SELECT_EVENT_FOR_ANALYSIS</p>
        </div>
      {:else}
        {@const s = getSev(selected.event_type)}
        {@const status = localSt[selected.id] ?? selected.status ?? 'new'}
        
        <!-- Detail Header -->
        <div class="p-8 border-b border-border-primary bg-surface-1 shrink-0 space-y-6">
          <div class="flex justify-between items-start">
            <div class="space-y-1">
              <div class="flex items-center gap-3">
                <span class="text-[10px] font-black text-text-muted uppercase tracking-widest italic">{selected.id}</span>
                <Badge variant={s.variant} size="xs" dot>{s.label}</Badge>
              </div>
              <h2 class="text-2xl font-black text-text-heading uppercase tracking-tighter italic">{selected.event_type.replace(/_/g, ' ')}</h2>
            </div>
            
            <div class="flex items-center gap-2">
              <Button variant="secondary" size="sm" icon={Eye}>VIEW_LOG</Button>
              <Button variant="primary" size="sm" icon={ShieldAlert}>PIVOT_INVESTIGATION</Button>
            </div>
          </div>

          <div class="grid grid-cols-4 gap-8">
            {#each [
              { label: 'Host', val: selected.host_id, icon: Terminal },
              { label: 'Source', val: selected.source_ip, icon: Globe },
              { label: 'Identity', val: selected.user, icon: User },
              { label: 'Timestamp', val: fmtDate(selected.timestamp), icon: Clock }
            ] as item}
              <div class="space-y-1.5">
                <div class="flex items-center gap-2 text-[9px] font-mono text-text-muted uppercase tracking-widest">
                  <item.icon size={10} />
                  {item.label}
                </div>
                <div class="text-[11px] font-bold text-text-secondary uppercase">{item.val || '—'}</div>
              </div>
            {/each}
          </div>
        </div>

        <!-- Detail Body -->
        <div class="flex-1 overflow-y-auto p-8 space-y-8">
          <div class="space-y-4">
            <div class="flex items-center justify-between border-b border-border-subtle pb-2">
              <span class="text-[10px] font-black text-text-heading uppercase tracking-widest">Telemetry Evidence</span>
              <span class="text-[9px] font-mono text-text-muted uppercase">Topic: security.events.l7</span>
            </div>
            <pre class="p-6 bg-surface-2 border border-border-primary rounded-sm text-[11px] font-mono text-accent-primary leading-relaxed whitespace-pre-wrap overflow-x-auto shadow-inner">
              {selected.raw_log}
            </pre>
          </div>

          <!-- Actions -->
          <div class="space-y-4 pt-4 border-t border-border-subtle">
             <span class="text-[10px] font-black text-text-heading uppercase tracking-widest">Tactical Response</span>
             <div class="grid grid-cols-3 gap-4">
               <Button 
                variant={status === 'investigating' ? 'warning' : 'secondary'} 
                size="md" 
                class="font-black italic tracking-tighter"
                onclick={() => setStatus(selected!.id, 'investigating')}
               >
                 INVESTIGATE
               </Button>
               <Button 
                variant={status === 'acknowledged' ? 'success' : 'secondary'} 
                size="md" 
                class="font-black italic tracking-tighter"
                onclick={() => setStatus(selected!.id, 'acknowledged')}
               >
                 ACKNOWLEDGE
               </Button>
               <Button 
                variant={status === 'closed' ? 'secondary' : 'secondary'} 
                size="md" 
                class="font-black italic tracking-tighter"
                onclick={() => setStatus(selected!.id, 'closed')}
               >
                 CLOSE_CASE
               </Button>
             </div>
          </div>
        </div>

        <!-- Status Bar -->
        <div class="bg-surface-2 border-t border-border-primary px-4 py-2 flex items-center justify-between text-[9px] font-mono uppercase tracking-widest text-text-muted shrink-0">
          <div class="flex items-center gap-4">
            <span>Tenant: {selected.tenant_id || 'GLOBAL'}</span>
            <span>|</span>
            <span>Policy: DEFAULT_INGEST_v1</span>
          </div>
          <div class="flex items-center gap-2">
            <div class="w-1.5 h-1.5 rounded-full" style="background: {statusColor[status]}"></div>
            <span style="color: {statusColor[status]}">Current State: {status}</span>
          </div>
        </div>
      {/if}
    </div>
  </div>
</PageLayout>

<style>
  :global(.flex-1::-webkit-scrollbar) {
    width: 4px;
  }
  :global(.flex-1::-webkit-scrollbar-track) {
    background: transparent;
  }
  :global(.flex-1::-webkit-scrollbar-thumb) {
    background: var(--border-primary);
    border-radius: 2px;
  }
</style>
