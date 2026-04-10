<!--
  OBLIVRA — Tunnel Manager (Svelte 5)
  SSH Port Forwarding and Reverse Tunnels orchestration.
-->
<script lang="ts">
  import { appStore } from '@lib/stores/app.svelte';
  import { KPI, Badge, DataTable, PageLayout, Button, EmptyState, SearchBar } from '@components/ui';

  // Mock data for tunnels
  const mockTunnels = [
    { id: 't1', label: 'Local DB Proxy', local_port: 5432, remote_host: 'localhost', remote_port: 5432, target: 'prod-db-master', status: 'active', speed: '128 KB/s' },
    { id: 't2', label: 'Admin Web Console', local_port: 8080, remote_host: '127.0.0.1', remote_port: 80, target: 'edge-gw-01', status: 'paused', speed: '0 B/s' },
    { id: 't3', label: 'Internal API', local_port: 3000, remote_host: 'staging-api', remote_port: 3000, target: 'staging-app-01', status: 'error', speed: 'NaN' },
  ];

  let searchQuery = $state('');
  const filteredTunnels = $derived(
    mockTunnels.filter(t => t.label.toLowerCase().includes(searchQuery.toLowerCase()) || t.target.toLowerCase().includes(searchQuery.toLowerCase()))
  );

  const columns = [
    { key: 'label', label: 'Label', sortable: true },
    { key: 'local_port', label: 'Local Port', width: '100px' },
    { key: 'target', label: 'Exit Node', width: '150px' },
    { key: 'remote_port', label: 'Remote', width: '120px' },
    { key: 'speed', label: 'Throughput', width: '100px' },
    { key: 'status', label: 'Status', width: '100px' },
  ];

  function toggleTunnel(id: string) {
    appStore.notify(`Tunnel ${id} state updated`, 'info');
  }
</script>

<PageLayout title="Tunnel Manager" subtitle="Encrypted port forwarding and reverse proxy orchestration">
  {#snippet toolbar()}
    <SearchBar bind:value={searchQuery} placeholder="Filter tunnels..." compact />
    <Button variant="primary" size="sm" icon="+">New Tunnel</Button>
  {/snippet}

  <div class="flex flex-col h-full gap-5">
    <!-- Tunnel Stats -->
    <div class="grid grid-cols-1 md:grid-cols-3 gap-4 shrink-0">
      <KPI title="Active Tunnels" value={mockTunnels.filter(t => t.status === 'active').length} trend="Encrypted" variant="success" />
      <KPI title="Total Bandwidth" value="1.2 GB/s" trend="+12%" variant="accent" />
      <KPI title="Exposed Ports" value="8" trend="Firewalled" />
    </div>

    {#if mockTunnels.length > 0}
      <DataTable data={filteredTunnels} {columns} striped>
        {#snippet render({ value, col, row })}
          {#if col.key === 'status'}
            <Badge variant={value === 'active' ? 'success' : value === 'paused' ? 'warning' : 'critical'} dot>
              {value}
            </Badge>
          {:else if col.key === 'local_port'}
            <span class="font-mono text-accent">:{value}</span>
          {:else if col.key === 'remote_port'}
            <span class="text-text-muted text-[10px] font-mono">{row.remote_host}:{value}</span>
          {:else if col.key === 'speed'}
            <span class="text-[10px] font-mono text-text-muted">{value}</span>
          {:else if col.key === 'label'}
            <div class="flex flex-col">
              <span class="font-bold text-text-heading">{value}</span>
              <span class="text-[9px] text-text-muted uppercase tracking-tighter">ID: {row.id}</span>
            </div>
          {:else}
            {value}
          {/if}
        {/snippet}
      </DataTable>
    {:else}
      <EmptyState title="No tunnels configured" description="Create an SSH tunnel to securely access remote services on your local machine." icon="🚇" />
    {/if}
  </div>
</PageLayout>
