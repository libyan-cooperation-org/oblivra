<!-- Response Replay — list incidents and offer to play their timeline. -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { PageLayout, KPI, Button, DataTable, PopOutButton } from '@components/ui';
  import { Rewind, Play, RefreshCw } from 'lucide-svelte';
  import { IS_BROWSER } from '@lib/context';
  import { push } from '@lib/router.svelte';

  let incidents = $state<any[]>([]);
  let loading = $state(false);

  async function refresh() {
    loading = true;
    try {
      if (IS_BROWSER) return;
      const { ListIncidents } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/incidentservice');
      incidents = ((await ListIncidents('', '', 200)) ?? []) as any[];
    } finally { loading = false; }
  }
  onMount(refresh);
</script>

<PageLayout title="Response Replay" subtitle="Re-run past incident timelines">
  {#snippet toolbar()}
    <Button variant="secondary" size="sm" icon={RefreshCw} onclick={refresh}>{loading ? 'Loading…' : 'Refresh'}</Button>
    <PopOutButton route="/response-replay" title="Response Replay" />
  {/snippet}
  <div class="flex flex-col h-full gap-4">
    <div class="grid grid-cols-1 md:grid-cols-2 gap-3">
      <KPI label="Incidents" value={incidents.length.toString()} variant="accent" />
      <KPI label="Closed" value={incidents.filter((i) => i.status === 'closed' || i.status === 'resolved').length.toString()} variant="muted" />
    </div>
    <div class="flex-1 bg-surface-1 border border-border-primary rounded-md overflow-hidden">
      <div class="flex items-center gap-2 p-3 border-b border-border-primary">
        <Rewind size={14} class="text-accent" />
        <span class="text-[10px] uppercase tracking-widest font-bold">Past Incidents</span>
      </div>
      <DataTable data={incidents} columns={[
        { key: 'id',       label: 'ID',     width: '140px' },
        { key: 'title',    label: 'Incident' },
        { key: 'status',   label: 'Status', width: '100px' },
        { key: 'replay',   label: '',       width: '70px' },
      ]} compact>
        {#snippet render({ col, row })}
          {#if col.key === 'id'}<span class="font-mono text-[10px] text-accent">{row.id}</span>
          {:else if col.key === 'replay'}<Button variant="ghost" size="xs" onclick={() => push(`/timeline/${encodeURIComponent(row.id)}/incident/${Date.now()}`)}><Play size={11} /></Button>
          {:else}<span class="text-[11px]">{row[col.key] ?? '—'}</span>{/if}
        {/snippet}
      </DataTable>
      {#if incidents.length === 0}<div class="p-8 text-center text-sm text-text-muted">{loading ? 'Loading…' : 'No past incidents.'}</div>{/if}
    </div>
  </div>
</PageLayout>
