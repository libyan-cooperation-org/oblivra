<!--
  OBLIVRA — SSH Bookmarks (Svelte 5)
  Secure storage for remote connection profiles.
-->
<script lang="ts">
  import { KPI, Badge, PageLayout, Button, DataTable, SearchBar } from '@components/ui';
  import { appStore } from '@lib/stores/app.svelte';

  let searchQuery = $state('');

  const bookmarks = [
    { id: '1', label: 'edge-gateway-01', host: '10.0.4.1', user: 'admin', tags: ['edge', 'prod'] },
    { id: '2', label: 'honeypot-dmz', host: '192.168.50.22', user: 'root', tags: ['sandbox'] },
    { id: '3', label: 'vault-controller', host: '10.0.9.5', user: 'security', tags: ['critical'] },
  ];

  const columns = [
    { key: 'label', label: 'Identity', sortable: true },
    { key: 'host', label: 'Endpoint', sortable: true },
    { key: 'user', label: 'Principal' },
    { key: 'tags', label: 'Class' },
    { key: 'actions', label: '' },
  ];

  function connect(id: string) {
    appStore.notify('Connecting...', 'info', `Initiating encrypted tunnel to ${id}`);
    // Real logic would be: appStore.connectToHost(id)
  }
</script>

<PageLayout title="Boundaries" subtitle="Managed SSH Bookmarks & Credentials">
  {#snippet toolbar()}
    <div class="flex items-center gap-2">
      <Button variant="primary" size="sm">+ New Bookmark</Button>
    </div>
  {/snippet}

  <div class="space-y-6">
    <div class="w-full max-w-md">
      <SearchBar bind:value={searchQuery} placeholder="Search identities..." />
    </div>

    <div class="bg-surface-1 border border-border-primary rounded-sm overflow-hidden">
      <DataTable {columns} data={bookmarks}>
        {#snippet cell({ column, row })}
          {#if column.key === 'label'}
            <div class="flex flex-col">
              <span class="font-bold text-text-primary">{row.label}</span>
              <span class="text-[10px] text-text-muted font-mono">{row.id}</span>
            </div>
          {:else if column.key === 'tags'}
            <div class="flex gap-1">
              {#each row.tags as tag}
                <Badge variant={tag === 'critical' ? 'error' : 'info'}>{tag}</Badge>
              {/each}
            </div>
          {:else if column.key === 'actions'}
            <div class="flex justify-end">
              <Button variant="secondary" size="xs" onclick={() => connect(row.id)}>Connect</Button>
            </div>
          {:else}
            <span class={column.key === 'host' ? 'font-mono text-[11px]' : ''}>
              {row[column.key]}
            </span>
          {/if}
        {/snippet}
      </DataTable>
    </div>

    <!-- Stats -->
    <div class="grid grid-cols-1 md:grid-cols-3 gap-4">
      <KPI title="Managed Hosts" value="412" trend="+5" />
      <KPI title="Active Tunnels" value="28" trend="-2" />
      <KPI title="Key Health" value="100%" trend="stable" />
    </div>
  </div>
</PageLayout>
