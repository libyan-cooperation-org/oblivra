<!-- Escalation Center — bound to IncidentService for assign/escalate. -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { PageLayout, KPI, Badge, Button, DataTable, PopOutButton } from '@components/ui';
  import { Flame, RefreshCw, Send } from 'lucide-svelte';
  import { IS_BROWSER } from '@lib/context';
  import { appStore } from '@lib/stores/app.svelte';

  let incidents = $state<any[]>([]);
  let loading = $state(false);

  async function refresh() {
    loading = true;
    try {
      if (IS_BROWSER) return;
      const { ListIncidents } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/incidentservice');
      incidents = ((await ListIncidents('open', '', 200)) ?? []) as any[];
    } finally { loading = false; }
  }

  async function assign(id: string) {
    const owner = prompt('Assign to (operator email):'); if (!owner) return;
    try {
      const { AssignIncident } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/incidentservice');
      await AssignIncident(id, owner);
      appStore.notify(`Assigned ${id} → ${owner}`, 'success');
      void refresh();
    } catch (e: any) {
      appStore.notify(`Assign failed: ${e?.message ?? e}`, 'error');
    }
  }

  async function escalate(id: string) {
    const reason = prompt('Escalation reason:'); if (!reason) return;
    try {
      const { UpdateIncidentStatus } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/incidentservice');
      await UpdateIncidentStatus(id, 'escalated', reason);
      appStore.notify(`Escalated ${id}`, 'warning');
      void refresh();
    } catch (e: any) {
      appStore.notify(`Escalate failed: ${e?.message ?? e}`, 'error');
    }
  }

  onMount(refresh);

  let unassigned = $derived(incidents.filter((i) => !i.owner));
</script>

<PageLayout title="Escalation Center" subtitle="On-call routing and emergency assignment">
  {#snippet toolbar()}
    <Button variant="secondary" size="sm" icon={RefreshCw} onclick={refresh}>{loading ? 'Loading…' : 'Refresh'}</Button>
    <PopOutButton route="/escalation" title="Escalation" />
  {/snippet}
  <div class="flex flex-col h-full gap-4">
    <div class="grid grid-cols-1 md:grid-cols-3 gap-3">
      <KPI label="Open Incidents" value={incidents.length.toString()} variant={incidents.length > 0 ? 'warning' : 'muted'} />
      <KPI label="Unassigned" value={unassigned.length.toString()} variant={unassigned.length > 0 ? 'critical' : 'muted'} />
      <KPI label="Mode" value={IS_BROWSER ? 'Browser' : 'Desktop'} variant="muted" />
    </div>
    <div class="flex-1 bg-surface-1 border border-border-primary rounded-md overflow-hidden">
      <div class="flex items-center gap-2 p-3 border-b border-border-primary">
        <Flame size={14} class="text-warning" />
        <span class="text-[10px] uppercase tracking-widest font-bold">Open Incidents</span>
      </div>
      <DataTable data={incidents} columns={[
        { key: 'id',       label: 'ID',     width: '120px' },
        { key: 'title',    label: 'Title' },
        { key: 'owner',    label: 'Owner',  width: '160px' },
        { key: 'severity', label: 'Sev',    width: '70px' },
        { key: 'assign',   label: '',       width: '160px' },
      ]} compact>
        {#snippet render({ col, row })}
          {#if col.key === 'severity'}<Badge variant={row.severity === 'critical' ? 'critical' : 'warning'} size="xs">{row.severity}</Badge>
          {:else if col.key === 'owner'}<span class="font-mono text-[10px] {row.owner ? '' : 'text-warning'}">{row.owner ?? 'unassigned'}</span>
          {:else if col.key === 'assign'}
            <div class="flex gap-1 justify-end">
              <Button variant="ghost" size="xs" onclick={() => assign(row.id)}><Send size={10} /></Button>
              <Button variant="ghost" size="xs" onclick={() => escalate(row.id)}>Escalate</Button>
            </div>
          {:else}<span class="text-[11px]">{row[col.key] ?? '—'}</span>{/if}
        {/snippet}
      </DataTable>
      {#if incidents.length === 0}<div class="p-8 text-center text-sm text-text-muted">{loading ? 'Loading…' : 'No open incidents.'}</div>{/if}
    </div>
  </div>
</PageLayout>
