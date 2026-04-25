<!--
  OBLIVRA — Plugin Manager (Svelte 5)
  Dynamic integration of third-party security modules and extensions.
-->
<script lang="ts">
  import { KPI, Badge, DataTable, PageLayout, Button, EmptyState, SearchBar, PopOutButton} from '@components/ui';

  const mockPlugins = [
    { id: 'p1', name: 'Suricata Bridge', version: '2.1.0', author: 'OBLIVRA Core', status: 'active', type: 'Ingestion' },
    { id: 'p2', name: 'Slack Notifications', version: '1.0.5', author: 'Community', status: 'active', type: 'Notification' },
    { id: 'p3', name: 'VirusTotal Enricher', version: '3.4.2', author: 'OBLIVRA Core', status: 'disabled', type: 'Intel' },
    { id: 'p4', name: 'DarkTrace Connector', version: '0.9.0', author: 'DarkTrace', status: 'error', type: 'NDR' },
  ];

  let searchQuery = $state('');
  const filtered = $derived(mockPlugins.filter(p => p.name.toLowerCase().includes(searchQuery.toLowerCase())));

  const columns = [
    { key: 'name', label: 'Plugin Name' },
    { key: 'type', label: 'Type', width: '120px' },
    { key: 'version', label: 'Version', width: '100px' },
    { key: 'author', label: 'Author', width: '150px' },
    { key: 'status', label: 'Status', width: '120px' },
  ];
</script>

<PageLayout title="Extensibility & Plugins" subtitle="Manage external integrations and proprietary security modules">
  {#snippet toolbar()}
    <SearchBar bind:value={searchQuery} placeholder="Search plugins..." compact />
    <Button variant="primary" size="sm" icon="+">Install Plugin</Button>
      <PopOutButton route="/plugins" title="Plugin Manager" />
    {/snippet}

  <div class="flex flex-col h-full gap-5">
    <div class="grid grid-cols-1 md:grid-cols-3 gap-4 shrink-0">
      <KPI title="Installed" value={mockPlugins.length} trend="Library" />
      <KPI title="Active Hooks" value="42" trend="Synced" variant="accent" />
      <KPI title="Registry Status" value="Online" trend="Authenticated" variant="success" />
    </div>

    {#if mockPlugins.length > 0}
      <DataTable data={filtered} {columns} striped>
        {#snippet render({ value, col })}
          {#if col.key === 'status'}
            <Badge variant={value === 'active' ? 'success' : value === 'disabled' ? 'muted' : 'critical'} dot>
              {value}
            </Badge>
          {:else if col.key === 'name'}
            <div class="flex flex-col">
              <span class="font-bold text-text-heading">{value}</span>
              <span class="text-[9px] text-text-muted">Compatible with Wails v3</span>
            </div>
          {:else if col.key === 'type'}
            <span class="text-[10px] uppercase font-bold text-text-muted">{value}</span>
          {:else}
            {value}
          {/if}
        {/snippet}
      </DataTable>
    {:else}
      <EmptyState title="No plugins installed" description="Browse the OBLIVRA Marketplace to find integrations for your security stack." icon="🧩" />
    {/if}
  </div>
</PageLayout>
