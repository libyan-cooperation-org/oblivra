<!--
  OBLIVRA — Alert Dashboard (Svelte 5)
  Real-time view of security alerts with filtering and investigation tools.
-->
<script lang="ts">
  import { alertStore } from '@lib/stores/alerts.svelte';
  import { appStore } from '@lib/stores/app.svelte';
  import { KPI, Badge, DataTable, PageLayout, Button, Input } from '@components/ui';

  let searchQuery = $state('');
  
  const filteredAlerts = $derived(
    alertStore.alerts.filter(a =>
      a.title.toLowerCase().includes(searchQuery.toLowerCase()) ||
      a.host.toLowerCase().includes(searchQuery.toLowerCase())
    )
  );

  const stats = $derived.by(() => ({
    total: alertStore.alerts.length,
    critical: alertStore.alerts.filter(a => a.severity === 'critical').length,
    high: alertStore.alerts.filter(a => a.severity === 'high').length,
    open: alertStore.alerts.filter(a => a.status === 'open').length,
  }));

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
      <Button variant="secondary" size="sm" onclick={() => alertStore.refresh()}>Refresh</Button>
      <Button variant="cta" size="sm">Bulk Resolve</Button>
    </div>
  {/snippet}

  <div class="flex flex-col h-full gap-6">
    <!-- KPI Overview -->
    <div class="grid grid-cols-1 md:grid-cols-4 gap-4 shrink-0">
      <KPI label="Total Detected" value={stats.total} trend="stable" trendValue="+12 today" />
      <KPI label="Critical Payload" value={stats.critical} variant="critical" trend="up" trendValue="High Risk" />
      <KPI label="Mean Time to Det." value="4.2s" trend="down" trendValue="-0.5s" variant="success" />
      <KPI label="Pending Analysis" value={stats.open} variant="accent" trend="stable" trendValue="Assigning" />
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
            <div class="flex items-center gap-1 justify-end">
              <Button variant="ghost" size="sm" onclick={() => appStore.notify(`Investigating ${row.id}`, 'info')}>Investigate</Button>
              <Button variant="ghost" size="sm" onclick={() => appStore.notify(`Alert ${row.id} resolved`, 'success')}>Resolve</Button>
            </div>
          {:else}
            <span class="text-[11px] text-text-secondary">{value}</span>
          {/if}
        {/snippet}
      </DataTable>
    </div>
  </div>
</PageLayout>
