<!--
  OBLIVRA — Alert Dashboard (Svelte 5)
  Real-time view of security alerts with filtering and investigation tools.
-->
<script lang="ts">
  import { onMount } from 'svelte';
  import { alertStore } from '@lib/stores/alerts.svelte';
  import { appStore } from '@lib/stores/app.svelte';
  import { KPI, Badge, DataTable, PageLayout, Button, Input, PopOutButton } from '@components/ui';
  import { push } from '@lib/router.svelte';
  import { IS_BROWSER } from '@lib/context';

  let searchQuery = $state('');
  // Track new-alert count since the page mounted so the "today" trend is real.
  let baselineCount = $state(0);

  const filteredAlerts = $derived(
    alertStore.alerts.filter((a) =>
      a.title.toLowerCase().includes(searchQuery.toLowerCase()) ||
      a.host.toLowerCase().includes(searchQuery.toLowerCase())
    ),
  );

  const stats = $derived.by(() => ({
    total:    alertStore.alerts.length,
    critical: alertStore.alerts.filter((a) => a.severity === 'critical').length,
    high:     alertStore.alerts.filter((a) => a.severity === 'high').length,
    open:     alertStore.alerts.filter((a) => a.status === 'open').length,
    delta:    Math.max(0, alertStore.alerts.length - baselineCount),
  }));

  // Last-24h alert volume — replaces the old "Mean Time To Detect" tile
  // (which we shipped as "—" because we have no detection-latency
  // telemetry). Surfacing a real, useful number instead of a permanent
  // placeholder.
  const last24h = $derived(
    alertStore.alerts.filter((a) => {
      const t = Date.parse(a.timestamp ?? '');
      return Number.isFinite(t) && Date.now() - t < 24 * 60 * 60 * 1000;
    }).length,
  );

  // Phase 36.9: resolveAlert removed — depended on IncidentService which was
  // deleted with the broad scope cut. Operators close alerts via the live
  // /api/v1/alerts/{id}/suppress endpoint (per-row suppress button); pair
  // with an external SOAR for incident-status workflows.

  function investigate(alertID: string, hostID: string) {
    // Pivot to incident timeline if we can; otherwise drop into a host-scoped
    // SIEM search.
    push(`/timeline/${encodeURIComponent(alertID)}/alert/${Date.now()}`);
  }

  onMount(() => {
    if (typeof alertStore.init === 'function') alertStore.init();
    // Snapshot baseline count after first paint so deltas read sensibly.
    setTimeout(() => (baselineCount = alertStore.alerts.length), 50);
  });

  const columns = [
    { key: 'timestamp', label: 'Timestamp', width: '140px' },
    { key: 'title', label: 'Alert Title', sortable: true },
    { key: 'host', label: 'Host', width: '150px' },
    { key: 'severity', label: 'Severity', width: '100px' },
    { key: 'status', label: 'Status', width: '120px' },
    { key: 'action', label: '', width: '160px' },
  ] as any;
</script>

<PageLayout title="Security Alerts" subtitle="Real-time incident detection and management">
  {#snippet toolbar()}
    <div class="flex items-center gap-3">
      <Input variant="search" placeholder="Filter incidents..." bind:value={searchQuery} class="w-64" />
      <Button variant="secondary" size="sm" onclick={() => alertStore.refresh?.()}>Refresh</Button>
      <!-- Phase 36.9: Bulk Resolve removed (depended on IncidentService). -->
      <PopOutButton route="/alerts" title="Security Alerts" />
    </div>
  {/snippet}

  <div class="flex flex-col h-full gap-6">
    <!-- KPI Overview -->
    <div class="grid grid-cols-1 md:grid-cols-4 gap-4 shrink-0">
      <KPI label="Total Detected" value={stats.total} trend={stats.delta > 0 ? 'up' : 'stable'} trendValue={stats.delta > 0 ? `+${stats.delta} since open` : 'no change'} />
      <KPI label="Critical Payload" value={stats.critical} variant={stats.critical > 0 ? 'critical' : 'muted'} trend={stats.critical > 0 ? 'up' : 'stable'} trendValue={stats.critical > 0 ? 'Active' : 'Quiet'} />
      <KPI label="Last 24h" value={last24h} variant={last24h > 0 ? 'accent' : 'muted'} trendPolarity="up-bad" />
      <KPI label="Pending Analysis" value={stats.open} variant={stats.open > 0 ? 'accent' : 'muted'} trend="stable" trendValue={stats.open > 0 ? 'Awaiting triage' : 'Clear'} />
    </div>

    <!-- Alerts Table -->
    <div class="flex-1 min-h-0 bg-surface-1 border border-border-primary rounded-md overflow-hidden shadow-card">
      <DataTable data={filteredAlerts} {columns} compact>
        {#snippet render({ col, row, value })}
          {#if col.key === 'severity'}
            <Badge variant={row.severity === 'critical' ? 'critical' : row.severity === 'high' ? 'warning' : 'info'}>
              {row.severity}
            </Badge>
          {:else if col.key === 'status'}
            <div class="flex items-center gap-2">
              <span class="w-1.5 h-1.5 rounded-full {row.status === 'open' ? 'bg-accent animate-pulse' : 'bg-text-muted'}"></span>
              <span class="text-[10px] font-bold uppercase tracking-wider text-text-secondary">{row.status}</span>
            </div>
          {:else if col.key === 'title'}
            <div class="flex flex-col">
              <span class="text-[11px] font-bold text-text-heading">{row.title}</span>
              <span class="text-[9px] text-text-muted font-mono opacity-50">ID: {row.id}</span>
            </div>
          {:else if col.key === 'timestamp'}
            <span class="text-[10px] text-text-muted font-mono tabular-nums">{String(value).split(' ')[1]}</span>
          {:else if col.key === 'action'}
            <!-- Phase 36.9: Resolve button removed (depended on IncidentService). -->
            <div class="flex items-center gap-1 justify-end">
              <Button variant="ghost" size="sm" onclick={() => investigate(row.id, row.host)}>Investigate</Button>
            </div>
          {:else}
            <span class="text-[11px] text-text-secondary">{value}</span>
          {/if}
        {/snippet}
      </DataTable>
    </div>
  </div>
</PageLayout>
