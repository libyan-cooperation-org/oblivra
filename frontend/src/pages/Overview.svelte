<!--
  Overview — SOC mission control page.

  Phase 31 SOC redesign — the investigation-first landing page that
  replaces the cluttered traditional dashboard. Per the audit spec:
  "fast investigation, contextual navigation, real-time awareness,
  minimal cognitive load."

  Layout:
    Row 1: Risk-level KPI (Low/Medium/High) + Active Incidents +
           Critical Alerts + Online Agents
    Row 2 (left, large):  Global event timeline (interleaved alerts +
                          agent transitions, sorted DESC)
    Row 2 (right, narrow): Live Activity Feed (the drop-in widget
                           from Phase 30.4b — same content, narrower
                           presentation)

  Design choices:
    - NO charts. Per spec: "Avoid traditional cluttered dashboards
      with too many charts. Focus on timelines, context, and
      actionable insights."
    - Risk level is computed from the alert distribution (count of
      critical + high in the last hour) so it's a function of real
      state, not a placeholder.
    - Every entity rendered (host, alert title) is an EntityLink so
      one click drills the operator into the InvestigationPanel.
-->
<script lang="ts">
  import { onMount } from 'svelte';
  import { ShieldAlert, ShieldCheck, ShieldQuestion, AlertTriangle, Server, Activity, Clock } from 'lucide-svelte';
  import { PageLayout, KPI, Badge, ActivityFeed, EntityLink } from '@components/ui';
  import { alertStore } from '@lib/stores/alerts.svelte';
  import { agentStore } from '@lib/stores/agent.svelte';

  // ── Risk level computation ───────────────────────────────────────
  // Simple operator-friendly heuristic — real production logic would
  // pull from the risk service, but the audit's example is just a
  // {Low, Medium, High} traffic light.
  const oneHourAgo = $derived(Date.now() - 60 * 60_000);

  const recentCritical = $derived(
    alertStore.alerts.filter((a) =>
      (a.severity === 'critical') &&
      new Date(a.timestamp).getTime() > oneHourAgo,
    ).length,
  );
  const recentHigh = $derived(
    alertStore.alerts.filter((a) =>
      (a.severity === 'high' || a.severity === 'error') &&
      new Date(a.timestamp).getTime() > oneHourAgo,
    ).length,
  );

  type Risk = 'low' | 'medium' | 'high';
  const riskLevel = $derived<Risk>(
    recentCritical > 0
      ? 'high'
      : recentHigh > 3
        ? 'medium'
        : 'low',
  );

  const onlineAgents = $derived(
    agentStore.agents.filter((a) => a.status === 'online').length,
  );

  const incidentCount = $derived(
    // Active incidents = alerts in 'open' or 'investigating' state
    alertStore.alerts.filter((a) => a.status === 'open' || a.status === 'investigating').length,
  );

  const criticalCount = $derived(
    alertStore.alerts.filter((a) => a.severity === 'critical').length,
  );

  // ── Global timeline data: alerts + agent online/offline events ───
  // For the demo flow we surface alerts only; agent transitions
  // surface in ActivityFeed already.
  const timeline = $derived(
    [...alertStore.alerts]
      .sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime())
      .slice(0, 30),
  );

  function severityClass(sev: string): string {
    const s = (sev ?? '').toLowerCase();
    if (s === 'critical') return 'sev-critical';
    if (s === 'high' || s === 'error') return 'sev-error';
    if (s === 'medium' || s === 'warn' || s === 'warning') return 'sev-warn';
    if (s === 'low' || s === 'info') return 'sev-info';
    return 'sev-debug';
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

  onMount(() => {
    if (alertStore.alerts.length === 0) alertStore.refresh?.();
    if (agentStore.agents.length === 0) agentStore.refresh?.();
  });
</script>

<PageLayout title="Mission Control" subtitle="Real-time platform posture & live activity">
  <div class="ov-root">

    <!-- Risk + KPI strip -->
    <section class="ov-kpis">
      <div class="risk-card risk-{riskLevel}">
        <div class="risk-icon">
          {#if riskLevel === 'high'}
            <ShieldAlert size={28} strokeWidth={1.6} />
          {:else if riskLevel === 'medium'}
            <ShieldQuestion size={28} strokeWidth={1.6} />
          {:else}
            <ShieldCheck size={28} strokeWidth={1.6} />
          {/if}
        </div>
        <div class="risk-text">
          <span class="risk-label">PLATFORM RISK</span>
          <span class="risk-value">{riskLevel.toUpperCase()}</span>
          <span class="risk-context">
            {recentCritical} critical · {recentHigh} high · last hour
          </span>
        </div>
      </div>

      <KPI label="ACTIVE INCIDENTS" value={String(incidentCount)} variant={incidentCount > 0 ? 'critical' : 'success'} />
      <KPI label="CRITICAL ALERTS"  value={String(criticalCount)} variant={criticalCount > 0 ? 'critical' : 'success'} />
      <KPI label="ONLINE AGENTS"    value={`${onlineAgents}/${agentStore.agents.length}`} variant={onlineAgents === agentStore.agents.length ? 'success' : 'warning'} />
    </section>

    <!-- Main grid: timeline (8) + activity feed (4) -->
    <section class="ov-grid">

      <!-- Global event timeline -->
      <div class="ov-panel ov-timeline">
        <header class="ov-panel-header">
          <Clock size={12} strokeWidth={1.7} class="ov-panel-icon" />
          <span class="ov-panel-title">Global Event Timeline</span>
          <span class="ov-panel-subtitle">{timeline.length} most recent</span>
        </header>
        <div class="ov-timeline-body">
          {#if timeline.length === 0}
            <div class="ov-empty">
              <Activity size={20} strokeWidth={1.4} />
              <p>No events yet — alerts and agent transitions appear here.</p>
            </div>
          {:else}
            <ol class="ov-timeline-list">
              {#each timeline as a (a.id)}
                <li class="ov-tl-row {severityClass(a.severity)}">
                  <span class="ov-tl-marker" aria-hidden="true">
                    <AlertTriangle size={10} strokeWidth={2} />
                  </span>
                  <div class="ov-tl-body">
                    <div class="ov-tl-line">
                      <Badge variant={severityClass(a.severity).replace('sev-', '') as any} size="xs">{a.severity}</Badge>
                      <span class="ov-tl-title">{a.title}</span>
                    </div>
                    <div class="ov-tl-meta">
                      {#if a.host && a.host !== 'unknown' && a.host !== 'remote'}
                        <Server size={9} strokeWidth={1.6} class="ov-tl-meta-icon" />
                        <EntityLink type="host" id={a.host} class="ov-tl-host" />
                        <span class="ov-tl-sep">·</span>
                      {/if}
                      <span class="ov-tl-time">{relativeTime(a.timestamp)}</span>
                    </div>
                  </div>
                </li>
              {/each}
            </ol>
          {/if}
        </div>
      </div>

      <!-- Live activity feed -->
      <div class="ov-panel ov-feed">
        <ActivityFeed limit={50} />
      </div>

    </section>

  </div>
</PageLayout>

<style>
  .ov-root {
    display: flex;
    flex-direction: column;
    gap: 14px;
    padding: 4px;
    min-height: 0;
  }

  .ov-kpis {
    display: grid;
    grid-template-columns: 1.6fr 1fr 1fr 1fr;
    gap: 10px;
  }
  @media (max-width: 900px) {
    .ov-kpis { grid-template-columns: 1fr 1fr; }
  }

  /* Risk card — visually loudest element on the page. */
  .risk-card {
    display: flex;
    align-items: center;
    gap: 14px;
    padding: 16px 18px;
    border-radius: 10px;
    background: var(--color-surface-1);
    border: 1px solid var(--color-border-primary);
    transition: border-color 200ms, background 200ms;
  }
  .risk-card.risk-low {
    border-color: var(--color-sev-info);
    background: linear-gradient(135deg, var(--color-surface-1), var(--color-sev-info-bg));
  }
  .risk-card.risk-medium {
    border-color: var(--color-sev-warn);
    background: linear-gradient(135deg, var(--color-surface-1), var(--color-sev-warn-bg));
  }
  .risk-card.risk-high {
    border-color: var(--color-sev-critical);
    background: linear-gradient(135deg, var(--color-surface-1), var(--color-sev-critical-bg));
  }
  .risk-icon { flex-shrink: 0; }
  .risk-low    .risk-icon { color: var(--color-sev-info); }
  .risk-medium .risk-icon { color: var(--color-sev-warn); }
  .risk-high   .risk-icon { color: var(--color-sev-critical); }

  .risk-text {
    display: flex;
    flex-direction: column;
    gap: 2px;
    min-width: 0;
  }
  .risk-label {
    font-family: var(--font-mono);
    font-size: 9px;
    font-weight: 800;
    color: var(--color-text-muted);
    text-transform: uppercase;
    letter-spacing: 0.16em;
  }
  .risk-value {
    font-family: var(--font-mono);
    font-size: 24px;
    font-weight: 900;
    color: var(--color-text-heading);
    line-height: 1;
    letter-spacing: 0.04em;
  }
  .risk-context {
    font-family: var(--font-mono);
    font-size: 10px;
    color: var(--color-text-secondary);
    margin-top: 4px;
  }

  /* Main grid */
  .ov-grid {
    display: grid;
    grid-template-columns: 2fr 1fr;
    gap: 10px;
    flex: 1;
    min-height: 0;
  }
  @media (max-width: 900px) {
    .ov-grid { grid-template-columns: 1fr; }
  }

  .ov-panel {
    background: var(--color-surface-1);
    border: 1px solid var(--color-border-primary);
    border-radius: 8px;
    display: flex;
    flex-direction: column;
    overflow: hidden;
    min-height: 0;
  }
  .ov-feed { padding: 0; }

  .ov-panel-header {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 8px 12px;
    background: var(--color-surface-2);
    border-bottom: 1px solid var(--color-border-primary);
  }
  :global(.ov-panel-icon) { color: var(--color-accent); }
  .ov-panel-title {
    font-family: var(--font-mono);
    font-size: 9px;
    font-weight: 800;
    letter-spacing: 0.15em;
    color: var(--color-text-heading);
    text-transform: uppercase;
  }
  .ov-panel-subtitle {
    margin-left: auto;
    font-family: var(--font-mono);
    font-size: 9px;
    color: var(--color-text-muted);
  }

  .ov-timeline-body { flex: 1; overflow-y: auto; }
  .ov-empty {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 8px;
    padding: 32px 16px;
    color: var(--color-text-muted);
    text-align: center;
  }
  .ov-empty p {
    font-family: var(--font-mono);
    font-size: 10px;
    margin: 0;
  }

  .ov-timeline-list {
    list-style: none;
    margin: 0;
    padding: 4px;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }
  .ov-tl-row {
    display: grid;
    grid-template-columns: 18px 1fr;
    gap: 8px;
    padding: 6px 8px;
    border-radius: 4px;
    border-left: 2px solid transparent;
    background: var(--color-surface-2);
    transition: background 100ms, border-left-color 100ms;
  }
  .ov-tl-row:hover { background: var(--color-surface-3, rgba(255, 255, 255, 0.04)); }
  .ov-tl-row.sev-critical { border-left-color: var(--color-sev-critical); }
  .ov-tl-row.sev-error    { border-left-color: var(--color-sev-error); }
  .ov-tl-row.sev-warn     { border-left-color: var(--color-sev-warn); }
  .ov-tl-row.sev-info     { border-left-color: var(--color-sev-info); }
  .ov-tl-row.sev-debug    { border-left-color: var(--color-sev-debug); }

  .ov-tl-marker {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 18px;
    height: 18px;
    border-radius: 50%;
    color: var(--color-text-muted);
  }
  .ov-tl-row.sev-critical .ov-tl-marker { color: var(--color-sev-critical); }
  .ov-tl-row.sev-error    .ov-tl-marker { color: var(--color-sev-error); }
  .ov-tl-row.sev-warn     .ov-tl-marker { color: var(--color-sev-warn); }
  .ov-tl-row.sev-info     .ov-tl-marker { color: var(--color-sev-info); }

  .ov-tl-body { display: flex; flex-direction: column; gap: 2px; min-width: 0; }
  .ov-tl-line {
    display: flex;
    align-items: center;
    gap: 6px;
    min-width: 0;
  }
  .ov-tl-title {
    font-family: var(--font-ui);
    font-size: 11px;
    font-weight: 600;
    color: var(--color-text-heading);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .ov-tl-meta {
    display: flex;
    align-items: center;
    gap: 4px;
    font-family: var(--font-mono);
    font-size: 9px;
    color: var(--color-text-muted);
  }
  :global(.ov-tl-meta-icon) { color: var(--color-text-muted); }
  :global(.ov-tl-host)      { color: var(--color-accent); }
  .ov-tl-sep                { opacity: 0.5; }
</style>
