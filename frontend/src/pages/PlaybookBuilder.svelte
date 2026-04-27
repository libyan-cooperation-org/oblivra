<!--
  Playbook Builder — list available actions, run a playbook against an
  incident, view recent runs. Bound to PlaybookService.
-->
<script lang="ts">
  import { onMount } from 'svelte';
  import { PageLayout, KPI, Badge, Button, DataTable, PopOutButton } from '@components/ui';
  import { Zap, Play, RefreshCw } from 'lucide-svelte';
  import { IS_BROWSER } from '@lib/context';
  import { appStore } from '@lib/stores/app.svelte';

  type Action = { id?: string; name?: string; description?: string; kind?: string };
  let actions = $state<Action[]>([]);
  let loading = $state(false);
  let executing = $state<string | null>(null);

  async function refresh() {
    loading = true;
    try {
      if (IS_BROWSER) { actions = []; return; }
      const { ListAvailableActions } = await import(
        '@wailsjs/github.com/kingknull/oblivrashell/internal/services/playbookservice'
      );
      actions = ((await ListAvailableActions()) ?? []) as Action[];
    } catch (e: any) {
      appStore.notify(`Action list failed: ${e?.message ?? e}`, 'error');
    } finally { loading = false; }
  }

  async function execute(id: string, name: string) {
    if (!confirm(`Execute action "${name}"?`)) return;
    executing = id;
    try {
      const { ExecuteAction } = await import(
        '@wailsjs/github.com/kingknull/oblivrashell/internal/services/playbookservice'
      );
      await ExecuteAction(id, {});
      appStore.notify(`Executed ${name}`, 'success');
    } catch (e: any) {
      appStore.notify(`Execute failed: ${e?.message ?? e}`, 'error');
    } finally { executing = null; }
  }

  async function runPlaybook() {
    const playbookID = prompt('Playbook ID:');
    if (!playbookID) return;
    const incidentID = prompt('Incident ID (optional):') ?? '';
    try {
      const { RunPlaybook } = await import(
        '@wailsjs/github.com/kingknull/oblivrashell/internal/services/playbookservice'
      );
      await RunPlaybook(playbookID, incidentID);
      appStore.notify(`Playbook ${playbookID} dispatched`, 'success');
    } catch (e: any) {
      appStore.notify(`Run failed: ${e?.message ?? e}`, 'error');
    }
  }

  onMount(refresh);

  let stats = $derived({
    total: actions.length,
    kinds: new Set(actions.map((a) => a.kind ?? 'misc')).size,
  });
</script>

<PageLayout title="Playbook Builder" subtitle="Available SOAR actions and execution control">
  {#snippet toolbar()}
    <Button variant="secondary" size="sm" icon={RefreshCw} onclick={refresh}>{loading ? 'Loading…' : 'Refresh'}</Button>
    <Button variant="primary" size="sm" onclick={runPlaybook}>Run Playbook</Button>
    <PopOutButton route="/playbook-builder" title="Playbook Builder" />
  {/snippet}

  <div class="flex flex-col h-full gap-4">
    <div class="grid grid-cols-1 md:grid-cols-3 gap-3 shrink-0">
      <KPI label="Available Actions" value={stats.total.toString()} variant="accent" />
      <KPI label="Action Kinds"      value={stats.kinds.toString()} variant="muted" />
      <KPI label="Mode"              value={IS_BROWSER ? 'Browser' : 'Desktop'} variant="muted" />
    </div>

    <div class="flex-1 min-h-0 bg-surface-1 border border-border-primary rounded-md overflow-hidden">
      <div class="flex items-center gap-2 p-3 border-b border-border-primary">
        <Zap size={14} class="text-accent" />
        <span class="text-[10px] uppercase tracking-widest font-bold">Action Library</span>
      </div>
      {#if actions.length === 0}
        <div class="p-12 text-center text-sm text-text-muted">{loading ? 'Loading…' : 'No actions registered. Add Sigma response handlers under sigma/responses/.'}</div>
      {:else}
        <DataTable data={actions} columns={[
          { key: 'name',        label: 'Action' },
          { key: 'kind',        label: 'Kind',  width: '120px' },
          { key: 'description', label: 'Description' },
          { key: 'execute',     label: '',      width: '100px' },
        ]} compact>
          {#snippet render({ col, row })}
            {#if col.key === 'name'}
              <span class="font-mono text-[11px] text-accent">{row.name ?? row.id ?? '—'}</span>
            {:else if col.key === 'kind'}
              <Badge variant="info" size="xs">{row.kind ?? 'misc'}</Badge>
            {:else if col.key === 'execute'}
              <Button variant="ghost" size="xs" onclick={() => execute(row.id ?? row.name, row.name ?? row.id)} disabled={executing === (row.id ?? row.name)}>
                <Play size={10} class="mr-1" />{executing === (row.id ?? row.name) ? '…' : 'Run'}
              </Button>
            {:else}
              <span class="text-[11px] text-text-muted">{row[col.key] ?? '—'}</span>
            {/if}
          {/snippet}
        </DataTable>
      {/if}
    </div>
  </div>
</PageLayout>
