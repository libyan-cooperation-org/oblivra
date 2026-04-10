<!--
  OBLIVRA — Snippet Manager (Svelte 5)
  Library of reusable commands and automation scripts.
-->
<script lang="ts">
  import { KPI, Badge, DataTable, PageLayout, Button, EmptyState, SearchBar, Tabs } from '@components/ui';

  const mockSnippets = [
    { id: 's1', name: 'Docker Cleanup', content: 'docker system prune -af', lang: 'bash', tags: ['ops', 'docker'] },
    { id: 's2', name: 'Retrieve Cloud Meta', content: 'curl -s http://169.254.169.254/latest/meta-data/', lang: 'bash', tags: ['cloud', 'aws'] },
    { id: 's3', name: 'Update SIEM Config', content: './oblivra-cli sync --force', lang: 'bash', tags: ['system'] },
  ];

  let searchQuery = $state('');
  let activeTab = $state('all');
  
  const filtered = $derived(mockSnippets.filter(s => s.name.toLowerCase().includes(searchQuery.toLowerCase())));

  const columns = [
    { key: 'name', label: 'Snippet Name', width: '250px' },
    { key: 'content', label: 'Command' },
    { key: 'tags', label: 'Tags', width: '150px' },
  ];
</script>

<PageLayout title="Snippet Vault" subtitle="Execute atomic operations across your fleet with pre-validated commands">
  {#snippet toolbar()}
    <SearchBar bind:value={searchQuery} placeholder="Filter snippets..." compact />
    <Button variant="primary" size="sm" icon="+">Create Snippet</Button>
  {/snippet}

  <div class="flex flex-col h-full gap-5">
    <div class="grid grid-cols-1 md:grid-cols-3 gap-4 shrink-0">
      <KPI title="Saved Snippets" value={mockSnippets.length} trend="Validated" />
      <KPI title="Team Shared" value="12" trend="Synced" variant="accent" />
      <KPI title="Personal" value="3" trend="Private" />
    </div>

    {#if mockSnippets.length > 0}
      <DataTable data={filtered} {columns} striped>
        {#snippet render({ value, col, row })}
          {#if col.key === 'content'}
            <code class="text-[10px] bg-surface-2 px-2 py-1 rounded border border-border-primary font-mono text-accent block truncate max-w-[400px]">
              {value}
            </code>
          {:else if col.key === 'tags'}
            <div class="flex flex-wrap gap-1">
              {#each value as tag}
                <Badge variant="muted">{tag}</Badge>
              {/each}
            </div>
          {:else if col.key === 'name'}
            <span class="font-bold text-text-heading">{value}</span>
          {:else}
            {value}
          {/if}
        {/snippet}
      </DataTable>
    {:else}
      <EmptyState title="No snippets found" description="Save your most frequently used terminal commands for rapid execution." icon="📝" />
    {/if}
  </div>
</PageLayout>
