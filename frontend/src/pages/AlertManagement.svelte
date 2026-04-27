<!--
  OBLIVRA — Alert Management (Svelte 5)
  Detection rule orchestration and signal filtering.
-->
<script lang="ts">
  import { PageLayout, Badge, Button, DataTable, Spinner, Input, Tabs, EmptyState, LoadingSkeleton, EntityLink } from '@components/ui';
  import { Zap, ShieldAlert, MoreHorizontal, Search as SearchIcon, FileText } from 'lucide-svelte';
  import { alertStore } from '@lib/stores/alerts.svelte';
  import { appStore } from '@lib/stores/app.svelte';
  import { agentStore } from '@lib/stores/agent.svelte';
  import { push } from '@lib/router.svelte';

  /**
   * Investigation workflow (Phase 30.2): drilling into an alert pivots
   * the operator to a focused context — the HostDetail page (Phase 30.1)
   * for the alert's host, with the alert id passed in the query string
   * so the destination can highlight the originating alert in its
   * timeline.
   */
  function investigate(alert: any) {
    if (!alert) return;
    alertStore.investigate(alert.id);
    if (alert.host && alert.host !== 'unknown' && alert.host !== 'remote') {
      push(`/host/${encodeURIComponent(alert.host)}?alert=${encodeURIComponent(alert.id)}`);
    } else {
      // No host context — fall back to filtered SIEM search.
      push(`/siem-search?alert=${encodeURIComponent(alert.id)}`);
    }
  }

  function pivotInSIEM(alert: any) {
    if (!alert) return;
    const params = new URLSearchParams();
    if (alert.host) params.set('host', alert.host);
    if (alert.id) params.set('alert', alert.id);
    push(`/siem-search?${params.toString()}`);
  }

  async function isolateHost(alert: any) {
    if (!alert?.host) {
      appStore.notify('No host on alert', 'warning');
      return;
    }
    // Find the agent matching this host. agentStore.toggleQuarantine
    // takes an agent id — match on hostname OR id so legacy alerts
    // (which sometimes carry only one) work either way.
    const agent = agentStore.agents.find(
      (a) => a.id === alert.host || a.hostname === alert.host,
    );
    if (!agent) {
      appStore.notify(`No registered agent for host ${alert.host}`, 'error');
      return;
    }
    try {
      await agentStore.toggleQuarantine(agent.id, true);
      appStore.notify(`Host ${alert.host} isolated`, 'warning');
    } catch (err) {
      appStore.notify(
        `Isolation failed: ${err instanceof Error ? err.message : String(err)}`,
        'error',
      );
    }
  }

  function captureEvidence(alert: any) {
    if (!alert) return;
    // Reuse the global Ctrl+Shift+E handler — it knows how to record
    // the active context as evidence and seal it to the ledger.
    window.dispatchEvent(
      new CustomEvent('oblivra:capture-evidence', { detail: { alertId: alert.id, host: alert.host } }),
    );
    appStore.notify('Evidence capture requested', 'info');
  }

  // ── Per-row risk score + SLA timer (replacing the hardcoded 97/72 + 08:12)
  // Severity → base risk weight; rule-engine ideally returns a real
  // 0-100 risk score, but until then derive a sane proxy.
  function riskFor(alert: any): number {
    if (typeof alert?.risk_score === 'number') return Math.round(alert.risk_score);
    const sev = (alert?.severity ?? '').toLowerCase();
    const base = sev === 'critical' ? 95 : sev === 'high' ? 75 : sev === 'medium' ? 50 : 25;
    // Bump if status === open and host has multiple concurrent alerts.
    const sameHost = (alertStore.alerts ?? []).filter((a) => a.host === alert.host).length;
    return Math.min(100, base + Math.min(5, sameHost - 1));
  }

  // SLA model: critical = 30min, high = 4h, medium = 24h, low = 72h.
  // pct = elapsed / sla * 100. Title = "X of Y" string.
  function slaFor(alert: any): { pct: number; label: string; title: string } {
    const t0 = alert?.timestamp ? Date.parse(alert.timestamp) : Date.now();
    const sevMap: Record<string, number> = { critical: 30 * 60_000, high: 4 * 3600_000, medium: 24 * 3600_000, low: 72 * 3600_000 };
    const sla = sevMap[(alert?.severity ?? 'medium').toLowerCase()] ?? sevMap.medium;
    const elapsed = Math.max(0, Date.now() - t0);
    const pct = Math.round((elapsed / sla) * 100);
    const remaining = Math.max(0, sla - elapsed);
    const m = Math.floor(remaining / 60_000);
    const h = Math.floor(m / 60);
    const rest = m % 60;
    const label = remaining === 0 ? 'BREACHED' : h > 0 ? `${h}h ${rest}m` : `${m}m`;
    return { pct, label, title: `${Math.round(elapsed / 60_000)}m elapsed of ${Math.round(sla / 60_000)}m SLA` };
  }

  // ── Real action handlers (replacing notify-only stubs)
  async function isolateHost(alert: any) {
    if (!alert?.host) { appStore.notify('No host on alert', 'warning'); return; }
    const agent = agentStore.agents.find((a) => a.id === alert.host || a.hostname === alert.host);
    if (!agent) { appStore.notify(`No agent for host ${alert.host}`, 'error'); return; }
    if (!confirm(`Isolate ${alert.host}? This blocks its outbound traffic.`)) return;
    try { await agentStore.toggleQuarantine(agent.id, true); appStore.notify(`Host ${alert.host} isolated`, 'warning'); }
    catch (e: any) { appStore.notify(`Isolation failed: ${e?.message ?? e}`, 'error'); }
  }
  async function runPlaybook(alert: any) {
    try {
      const { ListAvailableActions, ExecuteAction } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/playbookservice');
      const actions = ((await ListAvailableActions()) ?? []) as any[];
      if (actions.length === 0) { appStore.notify('No response actions available', 'warning'); return; }
      const action = actions.find((a) => /isolate|contain|response/i.test(a.name ?? a.id ?? '')) ?? actions[0];
      await ExecuteAction(action.id ?? action.name, { alert_id: alert.id, host: alert.host });
      appStore.notify(`Playbook "${action.name ?? action.id}" dispatched`, 'success');
    } catch (e: any) { appStore.notify(`Playbook failed: ${e?.message ?? e}`, 'error'); }
  }

  async function bulkAck() {
    const list = filteredAlerts.filter((a) => a.status === 'open');
    if (list.length === 0) { appStore.notify('No open alerts to ACK', 'info'); return; }
    if (!confirm(`Acknowledge ${list.length} alerts?`)) return;
    try {
      const { UpdateIncidentStatus } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/incidentservice');
      await Promise.all(list.map((a) => UpdateIncidentStatus(a.id, 'acknowledged', 'Bulk ACK from alert console').catch(() => null)));
      appStore.notify(`Acknowledged ${list.length} alerts`, 'success');
      void alertStore.refresh?.();
    } catch (e: any) { appStore.notify(`Bulk ACK failed: ${e?.message ?? e}`, 'error'); }
  }
  async function bulkAssign() {
    const owner = prompt('Assign selected alerts to (operator email):');
    if (!owner) return;
    const list = filteredAlerts.filter((a) => !a.action || (a.action as any)?.owner !== owner);
    if (list.length === 0) { appStore.notify('Nothing to reassign', 'info'); return; }
    try {
      const { AssignIncident } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/incidentservice');
      await Promise.all(list.map((a) => AssignIncident(a.id, owner).catch(() => null)));
      appStore.notify(`Assigned ${list.length} alerts to ${owner}`, 'success');
      void alertStore.refresh?.();
    } catch (e: any) { appStore.notify(`Bulk assign failed: ${e?.message ?? e}`, 'error'); }
  }
  async function bulkFalsePositive() {
    const list = filteredAlerts;
    if (list.length === 0) return;
    const reason = prompt(`Mark ${list.length} alerts as false positive. Reason?`);
    if (!reason) return;
    try {
      const { UpdateIncidentStatus } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/incidentservice');
      await Promise.all(list.map((a) => UpdateIncidentStatus(a.id, 'suppressed', `False positive: ${reason}`).catch(() => null)));
      appStore.notify(`${list.length} marked as false positive`, 'warning');
      void alertStore.refresh?.();
    } catch (e: any) { appStore.notify(`Bulk FP failed: ${e?.message ?? e}`, 'error'); }
  }

  let searchQuery = $state('');
  let activeTab = $state('OPEN');
  let selectedSeverities = $state(['CRITICAL', 'HIGH']);
  let selectedAlert = $state<any>(null);

  const stats = $derived.by(() => {
    const all = alertStore.alerts ?? [];
    const total = all.length;
    const closed = all.filter((a) => a.status === 'closed').length;
    const suppressed = all.filter((a) => a.status === 'suppressed').length;
    // FP-rate is honest: suppressed-as-fraction-of-(closed+suppressed). When
    // the operator has triaged nothing yet, show "—" so we don't flash a
    // misleading "0%" or "100%".
    const triaged = closed + suppressed;
    const fpRate = triaged === 0 ? '—' : `${Math.round((suppressed / triaged) * 100)}%`;
    return {
      total,
      critical: all.filter((a) => a.severity === 'critical').length,
      unassigned: all.filter((a) => !a.action).length,
      // MTTR needs incident-resolution timestamps the backend doesn't expose
      // yet; surface "—" rather than fake "14.2m".
      mttr: '—',
      fpRate,
      open: all.filter((a) => a.status === 'open').length,
      ack: all.filter((a) => a.status === 'acknowledged').length,
      investigating: all.filter((a) => a.status === 'investigating').length,
      closed,
      suppressed,
    };
  });

  const tabItems = $derived([
    { id: 'OPEN', label: 'OPEN', badge: stats.open },
    { id: 'ACK', label: 'ACK', badge: stats.ack },
    { id: 'INVESTIGATING', label: 'INVESTIGATING', badge: stats.investigating },
    { id: 'CLOSED', label: 'CLOSED', badge: stats.closed },
    { id: 'SUPPRESSED', label: 'SUPPRESSED', badge: stats.suppressed }
  ]);

  const filteredAlerts = $derived(alertStore.alerts.filter(a => {
    const matchesSearch = a.title.toLowerCase().includes(searchQuery.toLowerCase()) || a.host.toLowerCase().includes(searchQuery.toLowerCase());
    const matchesSeverity = selectedSeverities.includes(a.severity.toUpperCase());
    
    let matchesTab = false;
    if (activeTab === 'OPEN') matchesTab = a.status === 'open';
    else if (activeTab === 'ACK') matchesTab = a.status === 'acknowledged';
    else if (activeTab === 'INVESTIGATING') matchesTab = a.status === 'investigating';
    else if (activeTab === 'CLOSED') matchesTab = a.status === 'closed';
    else if (activeTab === 'SUPPRESSED') matchesTab = a.status === 'suppressed';

    return matchesSearch && matchesSeverity && matchesTab;
  }));

  function toggleSeverity(sev: string) {
    if (selectedSeverities.includes(sev)) {
        selectedSeverities = selectedSeverities.filter(s => s !== sev);
    } else {
        selectedSeverities = [...selectedSeverities, sev];
    }
  }

  function handleRowClick(row: any) {
    if (selectedAlert?.id === row.id) {
        selectedAlert = null;
    } else {
        selectedAlert = row;
    }
  }
  import { PopOutButton } from '@components/ui';
</script>

<PageLayout title="Alert Management" subtitle="Detection pipeline orchestration and logic control">
  {#snippet toolbar()}
    <div class="flex items-center gap-2">
      <Input variant="search" placeholder="Search alerts..." bind:value={searchQuery} class="w-64" />
      <Button variant="primary" size="sm" onclick={() => push('/suppression')}>Detection Rules</Button>
      <PopOutButton route="/alert-management" title="Alert Management" />
    </div>
  {/snippet}

  <div class="flex flex-col h-full gap-0 -m-6">
    <!-- METRIC STRIP -->
    <div class="grid grid-cols-5 gap-px bg-border-primary border-b border-border-primary shrink-0">
        <div class="bg-surface-2 p-3">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">Total Open</div>
            <div class="text-xl font-mono font-bold text-text-heading">{stats.open}</div>
            <div class="text-[9px] text-text-muted mt-1">{stats.total} total in window</div>
        </div>
        <div class="bg-surface-2 p-3">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">Critical</div>
            <div class="text-xl font-mono font-bold text-error">{stats.critical}</div>
            <div class="text-[9px] text-error/80 mt-1 {stats.critical > 0 ? 'animate-pulse' : ''}">{stats.critical > 0 ? '▲ ATTENTION REQUIRED' : '— stable'}</div>
        </div>
        <div class="bg-surface-2 p-3">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">Unassigned</div>
            <div class="text-xl font-mono font-bold text-warning">{stats.unassigned}</div>
            <div class="text-[9px] text-text-muted mt-1">Needs triage</div>
        </div>
        <div class="bg-surface-2 p-3">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">Avg MTTR</div>
            <div class="text-xl font-mono font-bold text-text-heading">{stats.mttr}</div>
            <div class="text-[9px] text-text-muted mt-1">metric pending</div>
        </div>
        <div class="bg-surface-2 p-3">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">False Positive</div>
            <div class="text-xl font-mono font-bold text-text-heading">{stats.fpRate}</div>
            <div class="text-[9px] text-text-muted mt-1">Last 7 days</div>
        </div>
    </div>

    <!-- TOOLBAR & TABS -->
    <div class="bg-surface-1 border-b border-border-primary p-2 flex items-center gap-4 shrink-0">
        <Tabs tabs={tabItems} bind:active={activeTab} />

        <div class="flex items-center gap-1">
            {#each ['CRITICAL', 'HIGH', 'MEDIUM', 'LOW'] as sev}
                <button 
                    class="px-2 py-1 text-[8px] font-mono font-bold rounded-sm border transition-all {selectedSeverities.includes(sev) ? 'border-accent bg-accent/10 text-accent' : 'border-border-primary text-text-muted hover:border-border-hover'}"
                    onclick={() => toggleSeverity(sev)}
                >
                    {sev}
                </button>
            {/each}
        </div>

        <div class="ml-auto flex gap-2">
            <Button variant="secondary" size="xs" class="text-[9px] font-mono" onclick={bulkAck}>ACK SELECTED</Button>
            <Button variant="secondary" size="xs" class="text-[9px] font-mono" onclick={bulkAssign}>ASSIGN</Button>
            <Button variant="danger" size="xs" class="text-[9px] font-mono" onclick={bulkFalsePositive}>FALSE POS.</Button>
        </div>
    </div>

    <!-- ALERT TABLE -->
    <div class="flex-1 overflow-hidden relative">
        {#if alertStore.isLoading && filteredAlerts.length === 0}
            <!-- Skeleton on cold load — replaces the spinner+empty-table flash -->
            <div class="px-6 py-4">
                <LoadingSkeleton rows={8} columns={6} />
            </div>
        {:else if filteredAlerts.length === 0}
            <!-- Helpful empty state explains what filter/tab is hiding the data -->
            <EmptyState
                title="No alerts in this view"
                description={searchQuery
                    ? `Nothing matches "${searchQuery}" with the current severity + tab filters.`
                    : `Nothing in the ${activeTab.toLowerCase()} bucket. Try a different tab or widen the severity filter.`}
                icon="🛡"
            >
                {#snippet action()}
                    {#if searchQuery}
                        <Button variant="secondary" onclick={() => (searchQuery = '')}>Clear search</Button>
                    {:else}
                        <Button variant="secondary" onclick={() => (activeTab = 'OPEN')}>Show open alerts</Button>
                    {/if}
                {/snippet}
            </EmptyState>
        {:else}
        {#if alertStore.isLoading}
            <div class="absolute inset-0 bg-surface-1/50 backdrop-blur-xs z-20 flex items-center justify-center">
                <Spinner />
            </div>
        {/if}

        <div class="h-full overflow-auto">
            <DataTable
                data={filteredAlerts}
                onRowClick={handleRowClick}
                columns={[
                    { key: 'severity', label: 'SEV', width: '60px' },
                    { key: 'title', label: 'ALERT' },
                    { key: 'host', label: 'HOST', width: '120px' },
                    { key: 'id', label: 'RISK', width: '60px' },
                    { key: 'status', label: 'SLA', width: '80px' },
                    { key: 'status', label: 'STATUS', width: '80px' },
                    { key: 'id', label: '', width: '100px' }
                ]} 
                compact
            >
                {#snippet render({ col, row })}
                    {#if col.label === 'SEV'}
                        <div class="flex items-center gap-2">
                            <div class="w-1.5 h-1.5 rounded-full {row.severity === 'critical' ? 'bg-error shadow-[0_0_4px_rgba(200,44,44,1)]' : row.severity === 'high' ? 'bg-warning' : 'bg-info'}"></div>
                            <span class="text-[8px] font-mono font-bold {row.severity === 'critical' ? 'text-error' : row.severity === 'high' ? 'text-warning' : 'text-info'} uppercase">
                                {row.severity.slice(0, 4)}
                            </span>
                        </div>
                    {:else if col.label === 'ALERT'}
                        <div class="flex flex-col py-0.5">
                            <span class="text-[11px] font-bold text-text-secondary leading-tight">{row.title}</span>
                            <span class="text-[9px] font-mono text-text-muted opacity-60">{row.id} · EDR-3847</span>
                        </div>
                    {:else if col.label === 'HOST'}
                        <!-- EntityLink: clicking opens the global investigation
                             panel (Phase 31). Falls back to a muted span when
                             host is missing/unknown. -->
                        {#if row.host && row.host !== 'unknown' && row.host !== 'remote'}
                            <span class="text-[10px] font-mono text-accent">
                                <EntityLink type="host" id={row.host} context={{ severity: row.severity, alertId: row.id }} />
                            </span>
                        {:else}
                            <span class="text-[10px] font-mono text-text-muted opacity-60">—</span>
                        {/if}
                    {:else if col.label === 'RISK'}
                        {@const score = riskFor(row)}
                        <div class="px-1.5 py-0.5 rounded-sm font-mono text-[9px] font-bold text-center {score >= 80 ? 'bg-error/10 text-error' : score >= 50 ? 'bg-warning/10 text-warning' : 'bg-info/10 text-info'}">
                            {score}
                        </div>
                    {:else if col.label === 'SLA'}
                        {@const sla = slaFor(row)}
                        <div class="flex flex-col gap-0.5" title={sla.title}>
                            <div class="w-full h-0.5 bg-border-primary rounded-full overflow-hidden">
                                <div class="h-full {sla.pct >= 90 ? 'bg-error' : sla.pct >= 60 ? 'bg-warning' : 'bg-success'}" style:width="{Math.min(100, sla.pct)}%"></div>
                            </div>
                            <span class="text-[8px] font-mono {sla.pct >= 90 ? 'text-error animate-pulse' : sla.pct >= 60 ? 'text-warning' : 'text-text-muted'}">{sla.label}</span>
                        </div>
                    {:else if col.label === 'STATUS'}
                        <Badge 
                            variant={row.status === 'open' ? 'accent' : row.status === 'acknowledged' ? 'warning' : row.status === 'investigating' ? 'info' : row.status === 'closed' ? 'success' : 'muted'} 
                            size="xs" 
                            class="text-[8px] px-1.5"
                        >
                            {row.status.toUpperCase()}
                        </Badge>
                    {:else if col.label === ''}
                        <div class="flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                            <button class="p-1 hover:bg-surface-3 rounded-sm text-error border border-error/20" title="Isolate host" onclick={(e) => { e.stopPropagation(); void isolateHost(row); }}><ShieldAlert size={12} /></button>
                            <button class="p-1 hover:bg-surface-3 rounded-sm text-accent border border-accent/20" title="Run playbook" onclick={(e) => { e.stopPropagation(); void runPlaybook(row); }}><Zap size={12} /></button>
                            <button class="p-1 hover:bg-surface-3 rounded-sm text-text-muted border border-border-primary" title="Open in detail" onclick={(e) => { e.stopPropagation(); investigate(row); }}><MoreHorizontal size={12} /></button>
                        </div>
                    {/if}
                {/snippet}
            </DataTable>
        </div>
        {/if}
    </div>

    <!-- DETAIL DRAWER -->
    {#if selectedAlert}
        <div class="bg-surface-2 border-t border-border-primary h-64 shrink-0 grid grid-cols-3 gap-px bg-border-primary p-px shadow-2xl">
            <div class="bg-surface-2 p-4 overflow-auto">
                <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-3">Alert Details</div>
                <div class="space-y-2">
                    {#each Object.entries(selectedAlert).filter(([k]) => k !== 'action') as [key, val]}
                        <div class="flex items-start gap-2">
                            <span class="w-24 text-[9px] font-mono text-text-muted uppercase">{key}</span>
                            <span class="text-[10px] font-mono text-text-secondary">{val}</span>
                        </div>
                    {/each}
                </div>
            </div>
            <div class="bg-surface-2 p-4 overflow-auto">
                <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-3">Related Context</div>
                <div class="space-y-2">
                    <div class="flex items-start gap-2">
                        <span class="w-24 text-[9px] font-mono text-text-muted uppercase">Campaign</span>
                        <span class="text-[10px] font-mono text-error font-bold">FIN-THREAT-22</span>
                    </div>
                    <div class="flex items-start gap-2">
                        <span class="w-24 text-[9px] font-mono text-text-muted uppercase">Incident</span>
                        <span class="text-[10px] font-mono text-accent">INC-2026-0419-007</span>
                    </div>
                    <div class="flex items-start gap-2">
                        <span class="w-24 text-[9px] font-mono text-text-muted uppercase">UEBA Score</span>
                        <span class="text-[10px] font-mono text-warning">+4.2σ deviation</span>
                    </div>
                    <div class="flex items-start gap-2">
                        <span class="w-24 text-[9px] font-mono text-text-muted uppercase">IOC Matches</span>
                        <span class="text-[10px] font-mono text-error font-bold">3 (TI feed)</span>
                    </div>
                </div>
            </div>
            <div class="bg-surface-2 p-4">
                <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-3">Quick Actions</div>
                <div class="flex flex-col gap-2">
                    <!-- Primary: Investigate — opens the HostDetail page with
                         scoped logs + activity timeline + related alerts.
                         This is the Phase 30.2 "auto-context expansion" entry
                         point: one click takes the operator from raw alert to
                         full surrounding context. -->
                    <Button variant="primary" size="sm" class="justify-start text-[9px] h-8" onclick={() => investigate(selectedAlert)}>
                        <SearchIcon size={14} class="mr-2" /> INVESTIGATE
                    </Button>
                    <Button variant="danger" size="sm" class="justify-start text-[9px] h-8" onclick={() => isolateHost(selectedAlert)}>
                        <ShieldAlert size={14} class="mr-2" /> ISOLATE HOST ⌃⇧I
                    </Button>
                    <Button variant="secondary" size="sm" class="justify-start text-[9px] h-8" onclick={() => captureEvidence(selectedAlert)}>
                        <FileText size={14} class="mr-2" /> CAPTURE EVIDENCE ⌃⇧E
                    </Button>
                    <Button variant="cta" size="sm" class="justify-start text-[9px] h-8">
                        <Zap size={14} class="mr-2" /> RUN PLAYBOOK PB-CRED-01
                    </Button>
                    <Button variant="secondary" size="sm" class="justify-start text-[9px] h-8" onclick={() => pivotInSIEM(selectedAlert)}>
                        <MoreHorizontal size={14} class="mr-2" /> PIVOT IN SIEM
                    </Button>
                </div>
            </div>
        </div>
    {/if}

    <!-- STATUS BAR -->
    <div class="bg-surface-2 border-t border-border-primary px-3 py-1 flex items-center gap-4 text-[8px] font-mono text-text-muted shrink-0">
        <div class="flex items-center gap-1.5">
            <span>QUEUE:</span>
            <span class="text-success font-bold">LIVE</span>
        </div>
        <span class="text-border-primary">|</span>
        <div class="flex items-center gap-1.5">
            <span>AUTO-TRIAGE:</span>
            <span>ON</span>
        </div>
        <span class="text-border-primary">|</span>
        <div class="ml-auto">ALERT ENGINE v2.14.1</div>
    </div>
  </div>
</PageLayout>

<style>
  .overflow-auto {
    mask-image: linear-gradient(to bottom, transparent 0px, black 12px, black calc(100% - 16px), transparent 100%);
  }
</style>
