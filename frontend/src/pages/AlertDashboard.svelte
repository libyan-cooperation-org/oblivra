<!--
  OBLIVRA — Alert Dashboard (Svelte 5)
  Real-time view of security alerts with filtering and investigation tools.
-->
<script lang="ts">
  import { appStore } from '@lib/stores/app.svelte';
  import { KPI, Badge, DataTable, PageLayout, Button, Input } from '@components/ui';

  // Mock alerts for demonstration (in production this comes from a store/backend)
  const mockAlerts = [
    { id: '1', timestamp: '2026-04-10 01:20:05', title: 'Suspicious SSH Login', severity: 'high', host: 'prod-web-01', status: 'open' },
    { id: '2', timestamp: '2026-04-10 01:15:32', title: 'Multiple Failed Logins', severity: 'medium', host: 'staging-db-02', status: 'open' },
    { id: '3', timestamp: '2026-04-10 01:05:12', title: 'Outbound Connection to Known Malicious IP', severity: 'critical', host: 'dev-workstation-k', status: 'investigating' },
    { id: '4', timestamp: '2026-04-10 00:55:00', title: 'Unexpected Binary Execution', severity: 'medium', host: 'prod-web-01', status: 'resolved' },
    { id: '5', timestamp: '2026-04-10 00:45:22', title: 'Sensitive File Access', severity: 'low', host: 'file-server-internal', status: 'open' },
  ];

  let searchQuery = $state('');
  const filteredAlerts = $derived(
    mockAlerts.filter(a => 
      a.title.toLowerCase().includes(searchQuery.toLowerCase()) || 
      a.host.toLowerCase().includes(searchQuery.toLowerCase())
    )
  );

  const stats = $derived.by(() => {
    return {
      total: mockAlerts.length,
      critical: mockAlerts.filter(a => a.severity === 'critical').length,
      high: mockAlerts.filter(a => a.severity === 'high').length,
      open: mockAlerts.filter(a => a.status === 'open').length,
    }
  });

  const columns = [
    { key: 'timestamp', label: 'Timestamp', width: '180px' },
    { key: 'title', label: 'Alert Title', sortable: true },
    { key: 'host', label: 'Host', width: '150px' },
    { key: 'severity', label: 'Severity', width: '100px' },
    { key: 'status', label: 'Status', width: '120px' },
  ];
</script>

<PageLayout title="Security Alerts" subtitle="Real-time incident detection and management">
  {#snippet toolbar()}
    <div class="flex items-center gap-3">
      <Input variant="search" placeholder="Filter incidents..." bind:value={searchQuery} class="w-64" />
      <Button variant="secondary" size="sm" onclick={() => appStore.notify('Refreshing alerts...', 'info')}>Refresh</Button>
      <Button variant="cta" size="sm">Bulk Resolve</Button>
    </div>
  {/snippet}

  <!-- KPI Overview -->
  <div class="grid grid-cols-1 md:grid-cols-4 gap-4 mb-6">
    <KPI title="Detected Inbound" value={stats.total} trend="+12%" />
    <KPI title="Critical Payload" value={stats.critical} variant="error" trend="High Risk" />
    <KPI title="Mean Time to Det." value="4.2s" trend="-0.5s" variant="success" />
    <KPI title="Pending Analysis" value={stats.open} variant="accent" trend="Assigning" />
  </div>

  <!-- Alerts Table -->
  <div class="bg-surface-1 border border-border-primary rounded-md overflow-hidden shadow-premium">
    <DataTable data={filteredAlerts} columns={[...columns, { key: 'action', label: '' }]} density="compact">
      {#snippet cell({ column, row })}
        {#if column.key === 'severity'}
          <Badge variant={row.severity === 'critical' ? 'error' : row.severity === 'high' ? 'warning' : 'info'}>
            {row.severity}
          </Badge>
        {:else if column.key === 'status'}
          <div class="flex items-center gap-2">
            <span class="w-1.5 h-1.5 rounded-full {row.status === 'open' ? 'bg-accent animate-pulse' : 'bg-text-muted'}"></span>
            <span class="text-[10px] font-bold uppercase tracking-wider text-text-secondary">
              {row.status}
            </span>
          </div>
        {:else if column.key === 'title'}
          <div class="flex flex-col">
            <span class="text-[11px] font-bold text-text-heading">{row.title}</span>
            <span class="text-[9px] text-text-muted font-mono opacity-50">ID: {row.id}</span>
          </div>
        {:else if column.key === 'timestamp'}
          <span class="text-[10px] text-text-muted font-mono tabular-nums">{row.timestamp.split(' ')[1]}</span>
        {:else if column.key === 'action'}
          <div class="flex items-center gap-1 justify-end">
            <Button variant="ghost" size="xs" onclick={() => appStore.notify(`Investigating ${row.id}`, 'info')}>Investigate</Button>
            <Button variant="ghost" size="xs" class="text-success" onclick={() => appStore.notify(`Alert ${row.id} resolved`, 'success')}>Resolve</Button>
          </div>
        {:else}
          <span class="text-[11px] text-text-secondary">{row[column.key]}</span>
        {/if}
      {/snippet}
    </DataTable>
  </div>
</PageLayout>
