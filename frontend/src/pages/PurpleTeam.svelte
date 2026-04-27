<!-- Purple Team — sim/red-vs-blue dashboard. Bound to PlaybookService.ListAvailableActions filtered by sim tags + alertStore for blue-team detection counts. -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { PageLayout, KPI, Badge, Button, DataTable, PopOutButton } from '@components/ui';
  import { Swords, Play, RefreshCw } from 'lucide-svelte';
  import { IS_BROWSER } from '@lib/context';
  import { appStore } from '@lib/stores/app.svelte';
  import { alertStore } from '@lib/stores/alerts.svelte';
  import { push } from '@lib/router.svelte';

  let scenarios = $state<any[]>([]);
  let loading = $state(false);

  async function refresh() {
    loading = true;
    try {
      if (IS_BROWSER) return;
      const { ListAvailableActions } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/playbookservice');
      const all = ((await ListAvailableActions()) ?? []) as any[];
      scenarios = all.filter((a) => /sim|red.?team|purple|atomic|attack/i.test(`${a.kind ?? ''} ${a.name ?? ''}`));
    } finally { loading = false; }
  }

  async function fire(id: string, name: string) {
    if (!confirm(`Fire purple-team scenario "${name}"?`)) return;
    try {
      const { ExecuteAction } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/playbookservice');
      await ExecuteAction(id, { mode: 'purple-team' });
      appStore.notify(`Scenario "${name}" fired`, 'success');
    } catch (e: any) { appStore.notify(`Fire failed: ${e?.message ?? e}`, 'error'); }
  }

  onMount(() => { void refresh(); if (typeof alertStore.init === 'function') alertStore.init(); });

  let recent = $derived((alertStore.alerts ?? []).slice(0, 20));
  let detected = $derived(recent.filter((a) => (a.title ?? '').toLowerCase().includes('purple') || (a.description ?? '').toLowerCase().includes('purple')).length);
</script>

<PageLayout title="Purple Team" subtitle="Continuous adversary simulation + detection validation">
  {#snippet toolbar()}
    <Button variant="secondary" size="sm" icon={RefreshCw} onclick={refresh}>{loading ? 'Loading…' : 'Refresh'}</Button>
    <Button variant="primary" size="sm" onclick={() => push('/simulation')}>Simulation Library</Button>
    <PopOutButton route="/purple-team" title="Purple Team" />
  {/snippet}
  <div class="flex flex-col h-full gap-4">
    <div class="grid grid-cols-1 md:grid-cols-3 gap-3">
      <KPI label="Available Scenarios" value={scenarios.length.toString()} variant="accent" />
      <KPI label="Recent Detections" value={recent.length.toString()} variant={recent.length > 0 ? 'warning' : 'muted'} />
      <KPI label="Purple-tagged" value={detected.toString()} variant="muted" />
    </div>
    <div class="flex-1 bg-surface-1 border border-border-primary rounded-md overflow-hidden">
      <div class="flex items-center gap-2 p-3 border-b border-border-primary">
        <Swords size={14} class="text-accent" />
        <span class="text-[10px] uppercase tracking-widest font-bold">Scenarios</span>
      </div>
      <DataTable data={scenarios} columns={[
        { key: 'name',        label: 'Name' },
        { key: 'kind',        label: 'Kind', width: '120px' },
        { key: 'description', label: 'Description' },
        { key: 'fire',        label: '',     width: '80px' },
      ]} compact>
        {#snippet render({ col, row })}
          {#if col.key === 'kind'}<Badge variant="info" size="xs">{row.kind ?? 'sim'}</Badge>
          {:else if col.key === 'fire'}<Button variant="ghost" size="xs" onclick={() => fire(row.id ?? row.name, row.name ?? row.id)}><Play size={10} class="mr-1" />Fire</Button>
          {:else}<span class="text-[11px]">{row[col.key] ?? '—'}</span>{/if}
        {/snippet}
      </DataTable>
      {#if scenarios.length === 0}<div class="p-8 text-center text-sm text-text-muted">{loading ? 'Loading…' : 'No simulation actions registered.'}</div>{/if}
    </div>
  </div>
</PageLayout>
