<!-- SOAR Panel — playbook + incident orchestration. Bound to PlaybookService + IncidentService. -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { PageLayout, KPI, Badge, Button, DataTable, PopOutButton } from '@components/ui';
  import { Zap, Play, RefreshCw } from 'lucide-svelte';
  import { IS_BROWSER } from '@lib/context';
  import { appStore } from '@lib/stores/app.svelte';
  import { push } from '@lib/router.svelte';

  let actions = $state<any[]>([]);
  let openInc = $state<any[]>([]);
  let loading = $state(false);

  async function refresh() {
    loading = true;
    try {
      if (IS_BROWSER) return;
      const pb = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/playbookservice');
      const ic = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/incidentservice');
      const [a, i] = await Promise.all([pb.ListAvailableActions(), ic.ListIncidents('open', '', 50)]);
      actions = (a ?? []) as any[];
      openInc = (i ?? []) as any[];
    } finally { loading = false; }
  }

  async function dispatch(actionID: string, incidentID: string) {
    try {
      const { RunPlaybook } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/playbookservice');
      await RunPlaybook(actionID, incidentID);
      appStore.notify(`Dispatched ${actionID} on ${incidentID}`, 'success');
    } catch (e: any) { appStore.notify(`Dispatch failed: ${e?.message ?? e}`, 'error'); }
  }
  onMount(refresh);
</script>

<PageLayout title="SOAR — Response" subtitle="Orchestrate response actions across open incidents">
  {#snippet toolbar()}
    <Button variant="secondary" size="sm" icon={RefreshCw} onclick={refresh}>{loading ? 'Loading…' : 'Refresh'}</Button>
    <Button variant="primary" size="sm" onclick={() => push('/playbook-builder')}>Playbook Builder</Button>
    <PopOutButton route="/response" title="SOAR" />
  {/snippet}
  <div class="flex flex-col h-full gap-4">
    <div class="grid grid-cols-1 md:grid-cols-3 gap-3">
      <KPI label="Available Actions" value={actions.length.toString()} variant="accent" />
      <KPI label="Open Incidents" value={openInc.length.toString()} variant={openInc.length > 0 ? 'warning' : 'muted'} />
      <KPI label="Mode" value={IS_BROWSER ? 'Browser' : 'Desktop'} variant="muted" />
    </div>
    <div class="grid grid-cols-1 md:grid-cols-2 gap-3 flex-1 min-h-0">
      <div class="bg-surface-1 border border-border-primary rounded-md flex flex-col min-h-0">
        <div class="flex items-center gap-2 p-3 border-b border-border-primary">
          <Zap size={14} class="text-accent" />
          <span class="text-[10px] uppercase tracking-widest font-bold">Actions</span>
        </div>
        <div class="flex-1 overflow-auto">
          {#each actions as a (a.id ?? a.name)}
            <div class="px-3 py-2 border-b border-border-primary text-[11px]">
              <div class="font-mono text-accent truncate">{a.name ?? a.id}</div>
              {#if a.description}<div class="text-[10px] text-text-muted truncate">{a.description}</div>{/if}
            </div>
          {/each}
        </div>
      </div>
      <div class="bg-surface-1 border border-border-primary rounded-md flex flex-col min-h-0">
        <div class="flex items-center gap-2 p-3 border-b border-border-primary">
          <span class="text-[10px] uppercase tracking-widest font-bold">Open Incidents</span>
        </div>
        <DataTable data={openInc} columns={[
          { key: 'id',       label: 'ID',     width: '120px' },
          { key: 'title',    label: 'Title' },
          { key: 'severity', label: 'Sev',    width: '70px' },
          { key: 'run',      label: '',       width: '70px' },
        ]} compact>
          {#snippet render({ col, row })}
            {#if col.key === 'id'}<span class="font-mono text-[10px] text-accent">{row.id}</span>
            {:else if col.key === 'severity'}<Badge variant={row.severity === 'critical' ? 'critical' : 'warning'} size="xs">{row.severity}</Badge>
            {:else if col.key === 'run'}
              <Button variant="ghost" size="xs" onclick={() => {
                if (actions.length === 0) { appStore.notify('No actions available', 'warning'); return; }
                const id = actions[0].id ?? actions[0].name;
                if (id) void dispatch(id, row.id);
              }}><Play size={10} /></Button>
            {:else}<span class="text-[11px]">{row[col.key] ?? '—'}</span>{/if}
          {/snippet}
        </DataTable>
        {#if openInc.length === 0}<div class="p-8 text-center text-sm text-text-muted">{loading ? 'Loading…' : 'No open incidents.'}</div>{/if}
      </div>
    </div>
  </div>
</PageLayout>
