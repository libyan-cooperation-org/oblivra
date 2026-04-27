<!--
  Tasks — schedule + monitor automation jobs. No dedicated TasksService
  exists; we surface PlaybookService.ListAvailableActions as the closest
  proxy and let the operator launch them on demand.
-->
<script lang="ts">
  import { onMount } from 'svelte';
  import { PageLayout, KPI, Button, Badge, DataTable, PopOutButton } from '@components/ui';
  import { Activity, Play, RefreshCw } from 'lucide-svelte';
  import { IS_BROWSER } from '@lib/context';
  import { appStore } from '@lib/stores/app.svelte';

  type Action = { id?: string; name?: string; description?: string; kind?: string };
  let actions = $state<Action[]>([]);
  let loading = $state(false);

  async function refresh() {
    loading = true;
    try {
      if (IS_BROWSER) return;
      const { ListAvailableActions } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/playbookservice');
      actions = ((await ListAvailableActions()) ?? []) as Action[];
    } finally { loading = false; }
  }

  async function run(id: string, name: string) {
    try {
      const { ExecuteAction } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/playbookservice');
      await ExecuteAction(id, {});
      appStore.notify(`Task ${name} ran`, 'success');
    } catch (e: any) {
      appStore.notify(`Run failed: ${e?.message ?? e}`, 'error');
    }
  }

  onMount(refresh);
</script>

<PageLayout title="Tasks & Automation" subtitle="Available SOAR actions and on-demand runs">
  {#snippet toolbar()}
    <Button variant="secondary" size="sm" icon={RefreshCw} onclick={refresh}>{loading ? 'Loading…' : 'Refresh'}</Button>
    <PopOutButton route="/tasks" title="Tasks" />
  {/snippet}

  <div class="flex flex-col h-full gap-4">
    <div class="grid grid-cols-1 md:grid-cols-3 gap-3">
      <KPI label="Available Tasks" value={actions.length.toString()} variant="accent" />
      <KPI label="Last Refresh" value={new Date().toLocaleTimeString()} variant="muted" />
      <KPI label="Mode" value={IS_BROWSER ? 'Browser' : 'Desktop'} variant="muted" />
    </div>

    <div class="flex-1 bg-surface-1 border border-border-primary rounded-md overflow-hidden">
      <DataTable data={actions} columns={[
        { key: 'name',        label: 'Task' },
        { key: 'kind',        label: 'Kind',  width: '120px' },
        { key: 'description', label: 'Description' },
        { key: 'run',         label: '',      width: '90px' },
      ]} compact>
        {#snippet render({ col, row })}
          {#if col.key === 'kind'}<Badge variant="info" size="xs">{row.kind ?? 'misc'}</Badge>
          {:else if col.key === 'run'}<Button variant="ghost" size="xs" onclick={() => run(row.id ?? row.name, row.name ?? row.id)}><Play size={10} /></Button>
          {:else}<span class="text-[11px]">{row[col.key] ?? '—'}</span>{/if}
        {/snippet}
      </DataTable>
    </div>
  </div>
</PageLayout>
