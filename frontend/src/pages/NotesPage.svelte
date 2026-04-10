<!--
  OBLIVRA — Notes (Svelte 5)
  Secure, per-host or global markdown notes for SOC analysts.
-->
<script lang="ts">
  import { KPI, PageLayout, Button, EmptyState, SearchBar } from '@components/ui';

  const mockNotes = [
    { id: 'n1', title: 'Prod-Web-01 Patch History', updated: '1h ago', category: 'Maintenance' },
    { id: 'n2', title: 'Incident #492 Root Cause', updated: '2d ago', category: 'IR' },
    { id: 'n3', title: 'Standard Credential Rotation', updated: '1w ago', category: 'Policy' },
  ];

  let searchQuery = $state('');
  const filtered = $derived(mockNotes.filter(n => n.title.toLowerCase().includes(searchQuery.toLowerCase())));
</script>

<PageLayout title="Knowledge Base & Notes" subtitle="Internal documentation and investigation notes">
  {#snippet toolbar()}
    <SearchBar bind:value={searchQuery} placeholder="Search notes..." compact />
    <Button variant="primary" size="sm" icon="+">New Note</Button>
  {/snippet}

  <div class="flex flex-col h-full gap-6">
    <div class="grid grid-cols-1 md:grid-cols-4 gap-4 shrink-0">
      <KPI title="Total Entries" value={mockNotes.length} trend="Library" />
      <KPI title="Active Drafts" value="4" trend="Manual Audit" variant="accent" />
      <KPI title="Team Shared" value="12" trend="Synced" />
      <KPI title="System Capacity" value="98%" trend="Zero-Lag" variant="success" />
    </div>

    {#if mockNotes.length > 0}
      <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-4">
        {#each filtered as note}
          <div class="bg-surface-1 border border-border-primary rounded-md p-4 cursor-pointer transition-all duration-fast hover:border-accent hover:shadow-glow-sm hover:translate-y-[-2px] group">
            <div class="flex justify-between items-start mb-2">
              <span class="text-[9px] font-bold text-accent uppercase tracking-widest">{note.category}</span>
              <span class="text-[9px] text-text-muted opacity-50">{note.updated}</span>
            </div>
            <h3 class="text-sm font-bold text-text-heading group-hover:text-accent transition-colors">{note.title}</h3>
            <p class="text-[11px] text-text-muted mt-2 line-clamp-3 leading-relaxed">
              No content summary available. Click to open the secure markdown editor and continue documenting...
            </p>
          </div>
        {/each}
      </div>
    {:else}
      <EmptyState title="No notes found" description="Create investigation notes or internal playbooks to share with your team." icon="🗒️" />
    {/if}
  </div>
</PageLayout>
