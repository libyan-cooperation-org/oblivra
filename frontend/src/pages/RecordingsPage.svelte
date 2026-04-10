<!--
  OBLIVRA — Recordings (Svelte 5)
  Audit trail of session replays and command history logs.
-->
<script lang="ts">
  import { KPI, Badge, DataTable, PageLayout, Button, EmptyState, SearchBar } from '@components/ui';

  const mockRecordings = [
    { id: 'r1', name: 'Root investigation - Web-01', host: 'prod-web-01', user: 'root', duration: '12:05', size: '2.4 MB', date: '2026-04-09' },
    { id: 'r2', name: 'Database Schema Update', host: 'staging-db-02', user: 'postgres', duration: '05:22', size: '840 KB', date: '2026-04-08' },
    { id: 'r3', name: 'Incident #412 Replay', host: 'dev-workstation-k', user: 'maverick', duration: '45:10', size: '12.1 MB', date: '2026-04-07' },
  ];

  let searchQuery = $state('');
  const filtered = $derived(mockRecordings.filter(r => r.name.toLowerCase().includes(searchQuery.toLowerCase()) || r.host.toLowerCase().includes(searchQuery.toLowerCase())));

  const columns = [
    { key: 'date', label: 'Date', width: '120px' },
    { key: 'name', label: 'Session Name', sortable: true },
    { key: 'host', label: 'Host', width: '150px' },
    { key: 'user', label: 'User', width: '100px' },
    { key: 'duration', label: 'Duration', width: '100px' },
    { key: 'size', label: 'Size', width: '100px' },
  ];
</script>

<PageLayout title="Session Recordings" subtitle="Comprehensive audit replays and immutable terminal history">
  {#snippet toolbar()}
    <SearchBar bind:value={searchQuery} placeholder="Search recordings..." compact />
    <Button variant="secondary" size="sm">Export All</Button>
  {/snippet}

  <div class="flex flex-col h-full gap-5">
    <div class="grid grid-cols-1 md:grid-cols-3 gap-4 shrink-0">
      <KPI title="Total Recordings" value={mockRecordings.length} trend="Library" />
      <KPI title="Storage Used" value="15.3 MB" trend="Nominal" />
      <KPI title="Retention Policy" value="90 Days" trend="Auto-Purge" variant="accent" />
    </div>

    {#if mockRecordings.length > 0}
      <DataTable data={filtered} {columns} striped onRowClick={(r) => console.log('Replay', r.id)}>
        {#snippet render({ value, col, row })}
          {#if col.key === 'name'}
            <div class="flex items-center gap-2">
              <span class="text-accent">▶</span>
              <span class="font-bold text-text-heading">{value}</span>
            </div>
          {:else if col.key === 'duration' || col.key === 'size'}
            <span class="text-[10px] font-mono text-text-muted">{value}</span>
          {:else if col.key === 'date'}
            <span class="text-[11px] font-mono opacity-60 tabular-nums">{value}</span>
          {:else}
            {value}
          {/if}
        {/snippet}
      </DataTable>
    {:else}
      <EmptyState title="No recordings found" description="Enable session recording in Settings to start capturing terminal activity." icon="🎥" />
    {/if}
  </div>
</PageLayout>
