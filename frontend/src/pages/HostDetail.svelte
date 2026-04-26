<!--
  HostDetail — single-pane-of-glass per machine.

  Phase 30.1: closes the "no host-centric view" gap from the operator UX
  audit. Routed at `/host/:id` (hash router param). Every host cell
  across SIEM tables, alerts, agent fleet, etc. drills down here.

  Sections:
    - Status banner (online/offline, last heartbeat, OS, version, trust)
    - Telemetry KPIs (CPU/RAM/Disk — best-effort from agent metadata)
    - Security posture (collectors, watchdog, quarantine state)
    - Scoped logs (most recent SIEM events for this host)
    - Scoped alerts (active alerts where alert.host == this id)
    - Activity timeline (interleaved logs + alerts, sorted DESC)

  "What happened on this machine?" workflow:
    The Activity Timeline IS that workflow. A single chronologically-
    ordered stream of every event touching this host — login, process,
    network, alert — derived from the local event store + alert store.
-->
<script lang="ts">
  import { onMount } from 'svelte';
  import {
    Server, Cpu, Shield, ShieldOff,
    AlertTriangle, Activity, Clock, ArrowLeft, RefreshCw, Power,
    Bug, Search as SearchIcon,
  } from 'lucide-svelte';
  import { agentStore, type AgentDTO } from '@lib/stores/agent.svelte';
  import { alertStore, type Alert } from '@lib/stores/alerts.svelte';
  import { siemStore } from '@lib/stores/siem.svelte';
  import { push } from '@lib/router.svelte';
  import { apiPostJSON } from '@lib/apiClient';

  /**
   * sendAgentAction enqueues a PendingAction for the agent. The
   * server side endpoint (`POST /api/v1/agent/action`) writes the
   * action to the pending queue; the agent reads it on its next
   * heartbeat (~10s) and executes locally. The agent's
   * acknowledgement event surfaces in the activity timeline below.
   */
  async function sendAgentAction(type: string, payload: Record<string, string>) {
    if (!agent) throw new Error('No agent in scope');
    await apiPostJSON('/api/v1/agent/action', {
      agent_id: agent.id,
      type,
      payload,
    });
  }
  import PageLayout from '@components/ui/PageLayout.svelte';
  import Badge from '@components/ui/Badge.svelte';
  import Button from '@components/ui/Button.svelte';
  import KPI from '@components/ui/KPI.svelte';
  import EmptyState from '@components/ui/EmptyState.svelte';

  // Route param spread by RouterView. The `id` arrives as a string
  // (the hostname / agent id from the URL `/host/:id`).
  interface Props {
    id?: string;
  }
  let { id = '' }: Props = $props();

  // ── Derived: agent matching this host id ─────────────────────────
  // Match on agent.id OR agent.hostname so links from anywhere work.
  const agent = $derived<AgentDTO | undefined>(
    agentStore.agents.find((a) => a.id === id || a.hostname === id),
  );

  const isOnline = $derived.by(() => {
    if (!agent) return false;
    if (agent.status === 'offline' || agent.status === 'down') return false;
    const last = new Date(agent.last_seen).getTime();
    if (!isFinite(last)) return false;
    // Generous heartbeat window — agent ticks every ~10s, mark offline at 60s
    return Date.now() - last < 60_000;
  });

  // ── Derived: alerts scoped to this host ──────────────────────────
  const scopedAlerts = $derived<Alert[]>(
    alertStore.alerts.filter((a) => a.host === id || a.host === agent?.hostname),
  );

  const criticalAlerts = $derived(
    scopedAlerts.filter((a) => a.severity === 'critical' || a.severity === 'high'),
  );

  // ── Derived: events scoped to this host (siemStore.results is the
  // last query result set; we filter client-side to the host of
  // interest). For deeper historical search the operator clicks
  // through to /siem-search?host=<id> via pivotToSIEM(). ──────────
  const scopedEvents = $derived(
    ((siemStore as any).results ?? []).filter((e: any) => {
      const h = e.host ?? e.Host ?? e.hostname;
      return h === id || h === agent?.hostname;
    }),
  );

  // ── Live telemetry: pull the most-recent `metrics` event for this
  // host and expose CPU / RAM / disk / queue. The agent's
  // MetricsCollector emits these every `Interval` (default 30s) and
  // they ride the same ingest channel as everything else.
  // Phase 31 close-out: Pass F surfaces this in the KPI strip below.
  const latestMetrics = $derived.by<Record<string, any>>(() => {
    const found = scopedEvents.find((e: any) => {
      const t = e.event_type ?? e.EventType ?? e.type ?? '';
      return t === 'metrics' || t === 'metric' || t === 'system.metrics';
    });
    return (found?.data ?? found?.Data ?? found ?? {}) as Record<string, any>;
  });

  const cpuPct = $derived<string>(formatPct(latestMetrics?.cpu_percent ?? latestMetrics?.cpu));
  const ramPct = $derived<string>(formatPct(latestMetrics?.memory_percent ?? latestMetrics?.ram_percent ?? latestMetrics?.ram));
  const diskPct = $derived<string>(formatPct(latestMetrics?.disk_percent ?? latestMetrics?.disk));

  function formatPct(v: any): string {
    if (v === undefined || v === null) return '—';
    const n = typeof v === 'number' ? v : parseFloat(String(v));
    if (!isFinite(n)) return '—';
    return n.toFixed(0) + '%';
  }

  // ── Activity timeline: interleave alerts + events, sort DESC ─────
  type TimelineEntry = {
    ts: string;
    kind: 'alert' | 'event';
    severity?: string;
    title: string;
    detail?: string;
    raw?: any;
  };

  const timeline = $derived.by<TimelineEntry[]>(() => {
    const out: TimelineEntry[] = [];
    for (const a of scopedAlerts) {
      out.push({
        ts: a.timestamp,
        kind: 'alert',
        severity: a.severity,
        title: a.title,
        detail: a.description,
        raw: a,
      });
    }
    for (const e of scopedEvents) {
      const ts = (e as any).timestamp ?? (e as any).ts ?? new Date().toISOString();
      const evType = (e as any).event_type ?? (e as any).EventType ?? 'event';
      out.push({
        ts,
        kind: 'event',
        severity: (e as any).severity ?? 'info',
        title: evType,
        detail: (e as any).message ?? (e as any).RawLine ?? '',
        raw: e,
      });
    }
    out.sort((a, b) => new Date(b.ts).getTime() - new Date(a.ts).getTime());
    return out.slice(0, 100); // cap at 100 most recent for render perf
  });

  // ── Actions ──────────────────────────────────────────────────────
  let actionInFlight = $state<string | null>(null);
  let actionMessage = $state<string | null>(null);

  async function runAction(name: string, fn: () => Promise<void>) {
    actionInFlight = name;
    actionMessage = null;
    try {
      await fn();
      actionMessage = `${name} succeeded`;
    } catch (err) {
      actionMessage = `${name} failed: ${err instanceof Error ? err.message : String(err)}`;
    } finally {
      actionInFlight = null;
      setTimeout(() => { actionMessage = null; }, 4000);
    }
  }

  function refresh() {
    agentStore.refresh();
    alertStore.refresh();
  }

  function quarantine() {
    if (!agent) return;
    runAction('Isolate Host', async () => {
      await agentStore.toggleQuarantine(agent.id, true);
    });
  }

  function unquarantine() {
    if (!agent) return;
    runAction('Restore Host', async () => {
      await agentStore.toggleQuarantine(agent.id, false);
    });
  }

  function pivotToSIEM() {
    push(`/siem-search?host=${encodeURIComponent(id)}`);
  }

  function pivotToAlerts() {
    push(`/alert-management?host=${encodeURIComponent(id)}`);
  }

  function severityClass(sev?: string): string {
    switch ((sev ?? '').toLowerCase()) {
      case 'critical': return 'sev-critical';
      case 'error':
      case 'high':     return 'sev-error';
      case 'warning':
      case 'warn':
      case 'medium':   return 'sev-warn';
      case 'info':     return 'sev-info';
      case 'debug':    return 'sev-debug';
      default:         return 'sev-info';
    }
  }

  function relativeTime(iso: string): string {
    const t = new Date(iso).getTime();
    if (!isFinite(t)) return iso;
    const diff = (Date.now() - t) / 1000;
    if (diff < 60) return `${Math.floor(diff)}s ago`;
    if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
    if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
    return new Date(iso).toLocaleString();
  }

  // Refresh on mount so we have the latest agent + alert state.
  onMount(() => {
    if (agentStore.agents.length === 0) agentStore.refresh();
  });
</script>

<PageLayout title="Host: {id}" subtitle={agent?.hostname ?? 'Unknown host'}>
  {#snippet toolbar()}
    <div class="flex items-center gap-2">
      <Button variant="ghost" onclick={() => push('/fleet')}>
        <ArrowLeft class="w-3 h-3 mr-1" /> Fleet
      </Button>
      <Button variant="ghost" onclick={refresh}>
        <RefreshCw class="w-3 h-3 mr-1" /> Refresh
      </Button>
    </div>
  {/snippet}

  {#if !agent}
    <EmptyState
      title="Host not found"
      description="No agent matching '{id}' is currently registered. The host may be offline or the link may be stale."
      icon="📡"
    >
      {#snippet action()}
        <Button variant="primary" onclick={() => push('/fleet')}>Open Fleet</Button>
      {/snippet}
    </EmptyState>
  {:else}
    <div class="grid grid-cols-1 gap-3">

      <!-- Status banner -->
      <section class="status-banner" class:online={isOnline} class:offline={!isOnline}>
        <div class="flex items-center gap-3">
          <Server class="w-5 h-5 shrink-0" />
          <div class="flex-1 min-w-0">
            <div class="flex items-center gap-2 text-sm font-bold">
              <span class="status-dot" aria-hidden="true"></span>
              <span>{isOnline ? 'ONLINE' : 'OFFLINE'}</span>
              <span class="text-text-muted font-normal">— last heartbeat {relativeTime(agent.last_seen)}</span>
            </div>
            <div class="flex flex-wrap gap-3 mt-1 text-[10px] font-mono text-text-muted">
              <span>OS: {agent.os ?? 'unknown'}</span>
              <span>ARCH: {agent.arch ?? '—'}</span>
              <span>VERSION: {agent.version}</span>
              <span>TRUST: {agent.trust_level ?? 'unverified'}</span>
              {#if agent.tenant_id}<span>TENANT: {agent.tenant_id}</span>{/if}
            </div>
          </div>
          <div class="flex gap-2 shrink-0">
            <Button variant="ghost" onclick={pivotToSIEM} title="Open SIEM Search filtered to this host">
              <SearchIcon class="w-3 h-3 mr-1" /> Logs
            </Button>
            <Button variant="ghost" onclick={pivotToAlerts} title="Open alerts filtered to this host">
              <AlertTriangle class="w-3 h-3 mr-1" /> Alerts ({scopedAlerts.length})
            </Button>
          </div>
        </div>
      </section>

      <!-- KPI strip — alert posture + live telemetry from the agent's
           MetricsCollector. CPU/RAM/Disk fall back to "—" until the
           first metrics event arrives (typically within Interval seconds
           of agent start; default Interval is 30s). -->
      <div class="grid grid-cols-2 md:grid-cols-4 lg:grid-cols-7 gap-2">
        <KPI label="ALERTS"     value={String(scopedAlerts.length)} />
        <KPI label="CRITICAL"   value={String(criticalAlerts.length)} variant={criticalAlerts.length > 0 ? 'critical' : 'success'} />
        <KPI label="EVENTS"     value={String(scopedEvents.length)} sublabel="live tail" />
        <KPI label="COLLECTORS" value={String(agent.collectors?.length ?? 0)} />
        <KPI label="CPU"        value={cpuPct}  sublabel="agent host" />
        <KPI label="RAM"        value={ramPct}  sublabel="agent host" />
        <KPI label="DISK"       value={diskPct} sublabel="agent host" />
      </div>

      <!-- Agent control panel (Phase 30.2 piece — remote actions) -->
      <section class="panel">
        <header class="panel-header">
          <Cpu class="w-3.5 h-3.5" />
          <span class="panel-title">Agent Control</span>
        </header>
        <div class="panel-body">
          <div class="flex flex-wrap gap-2">
            <!-- Remote-action RPCs landed in Phase 31 close-out.
                 Each button enqueues a PendingAction on the agent's
                 next heartbeat; the agent executes locally and emits
                 an acknowledgement event the operator sees in the
                 timeline below. -->
            <Button
              variant="ghost"
              onclick={() => runAction('Trigger Scan', () => sendAgentAction('trigger_scan', {}))}
              title="Request agent run an on-demand scan"
              disabled={!agent || actionInFlight !== null}
            >
              <RefreshCw class="w-3 h-3 mr-1" /> Trigger Scan
            </Button>

            <Button
              variant="ghost"
              onclick={() => runAction('Toggle Debug', () => sendAgentAction('toggle_debug', { state: 'on' }))}
              title="Toggle agent debug-level logging"
              disabled={!agent || actionInFlight !== null}
            >
              <Bug class="w-3 h-3 mr-1" /> Toggle Debug
            </Button>

            <Button
              variant="ghost"
              onclick={() => runAction('Restart Agent', () => sendAgentAction('restart_agent', {}))}
              title="Request agent process restart (the OS service manager auto-respawns)"
              disabled={!agent || actionInFlight !== null}
            >
              <Power class="w-3 h-3 mr-1" /> Restart Agent
            </Button>

            {#if agent.watchdog_active}
              <Button
                variant="danger"
                onclick={unquarantine}
                title="Restore network connectivity for this host"
              >
                <ShieldOff class="w-3 h-3 mr-1" /> Restore Host
              </Button>
            {:else}
              <Button
                variant="danger"
                onclick={quarantine}
                title="Cut network connectivity (network-isolation playbook)"
              >
                <Shield class="w-3 h-3 mr-1" /> Isolate Host
              </Button>
            {/if}
          </div>

          {#if actionInFlight}
            <div class="mt-2 text-[10px] font-mono text-text-muted">
              <span class="inline-block animate-pulse">●</span> {actionInFlight} in flight…
            </div>
          {:else if actionMessage}
            <div class="mt-2 text-[10px] font-mono" class:text-error={actionMessage.includes('failed')}>
              {actionMessage}
            </div>
          {/if}

          <div class="mt-3 text-[10px] font-mono text-text-muted">
            <span class="font-bold">Collectors:</span>
            {#if agent.collectors && agent.collectors.length > 0}
              {agent.collectors.join(', ')}
            {:else}
              <span class="opacity-60">none registered</span>
            {/if}
          </div>
        </div>
      </section>

      <!-- Activity Timeline ("What happened on this machine?") -->
      <section class="panel">
        <header class="panel-header">
          <Activity class="w-3.5 h-3.5" />
          <span class="panel-title">Activity Timeline</span>
          <span class="panel-subtitle">— what happened on this machine</span>
        </header>
        <div class="panel-body">
          {#if timeline.length === 0}
            <div class="text-[10px] font-mono text-text-muted opacity-60 py-4 text-center">
              No alerts or events for this host yet. The timeline populates as logs arrive.
            </div>
          {:else}
            <ol class="timeline" aria-label="Host activity timeline">
              {#each timeline as entry, i (entry.ts + i)}
                <li class="t-entry {severityClass(entry.severity)}" class:t-alert={entry.kind === 'alert'}>
                  <div class="t-marker" aria-hidden="true">
                    {#if entry.kind === 'alert'}
                      <AlertTriangle size={10} strokeWidth={2.2} />
                    {:else}
                      <Clock size={10} strokeWidth={1.6} />
                    {/if}
                  </div>
                  <div class="t-body">
                    <div class="t-head">
                      <span class="t-kind">{entry.kind}</span>
                      {#if entry.severity}
                        <Badge variant={severityClass(entry.severity).replace('sev-', '') as any}>
                          {entry.severity}
                        </Badge>
                      {/if}
                      <span class="t-title">{entry.title}</span>
                      <span class="t-time">{relativeTime(entry.ts)}</span>
                    </div>
                    {#if entry.detail}
                      <div class="t-detail">{entry.detail}</div>
                    {/if}
                  </div>
                </li>
              {/each}
            </ol>
          {/if}
        </div>
      </section>

    </div>
  {/if}
</PageLayout>

<style>
  .status-banner {
    display: flex;
    align-items: center;
    gap: 12px;
    padding: 12px 16px;
    background: var(--s1);
    border: 1px solid var(--b1);
    border-left: 3px solid var(--tx3);
    border-radius: 6px;
    color: var(--tx);
  }
  .status-banner.online { border-left-color: var(--success, #4ade80); }
  .status-banner.offline { border-left-color: var(--error, #ef4444); }
  .status-dot {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: var(--tx3);
    box-shadow: 0 0 0 2px transparent;
    flex-shrink: 0;
  }
  .status-banner.online .status-dot {
    background: var(--success, #4ade80);
    box-shadow: 0 0 6px var(--success, #4ade80);
    animation: pulse 2s ease-in-out infinite;
  }
  .status-banner.offline .status-dot { background: var(--error, #ef4444); }
  @keyframes pulse {
    0%, 100% { opacity: 1; }
    50%      { opacity: 0.4; }
  }

  .panel {
    background: var(--s1);
    border: 1px solid var(--b1);
    border-radius: 6px;
    overflow: hidden;
  }
  .panel-header {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 8px 12px;
    background: var(--s2);
    border-bottom: 1px solid var(--b1);
    color: var(--ac2);
  }
  .panel-title {
    font-family: var(--mn);
    font-size: 10px;
    font-weight: 700;
    color: var(--tx);
    text-transform: uppercase;
    letter-spacing: 0.1em;
  }
  .panel-subtitle {
    font-family: var(--sn);
    font-size: 10px;
    color: var(--tx3);
  }
  .panel-body { padding: 10px 12px; }

  /* ── Timeline ─────────────────────────────────────────── */
  .timeline {
    list-style: none;
    padding: 0;
    margin: 0;
    display: flex;
    flex-direction: column;
    gap: 6px;
  }
  .t-entry {
    display: grid;
    grid-template-columns: 18px 1fr;
    gap: 8px;
    padding: 6px 8px;
    border-left: 2px solid var(--b1);
    background: var(--s2);
    border-radius: 4px;
    transition: background 100ms;
  }
  .t-entry:hover { background: var(--s3, rgba(255, 255, 255, 0.03)); }
  .t-entry.t-alert { border-left-color: var(--ac); }
  .t-entry.sev-critical { border-left-color: var(--error, #ef4444); }
  .t-entry.sev-error    { border-left-color: var(--error, #ef4444); }
  .t-entry.sev-warn     { border-left-color: var(--warning, #f59e0b); }
  .t-entry.sev-info     { border-left-color: var(--accent, #0099e0); }
  .t-entry.sev-debug    { border-left-color: var(--tx3); }

  .t-marker {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 18px;
    height: 18px;
    color: var(--tx2);
  }
  .t-body { min-width: 0; }
  .t-head {
    display: flex;
    align-items: center;
    gap: 6px;
    flex-wrap: wrap;
  }
  .t-kind {
    font-family: var(--mn);
    font-size: 8px;
    font-weight: 800;
    color: var(--tx3);
    text-transform: uppercase;
    letter-spacing: 0.1em;
  }
  .t-title {
    font-family: var(--sn);
    font-size: 11px;
    font-weight: 600;
    color: var(--tx);
    flex: 1;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .t-time {
    font-family: var(--mn);
    font-size: 9px;
    color: var(--tx3);
    flex-shrink: 0;
  }
  .t-detail {
    font-family: var(--mn);
    font-size: 10px;
    color: var(--tx2);
    margin-top: 2px;
    line-height: 1.4;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
</style>
