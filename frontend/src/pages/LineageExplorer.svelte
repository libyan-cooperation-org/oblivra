<!-- Lineage Explorer — bound to LineageService. -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { PageLayout, KPI, Button, DataTable, PopOutButton } from '@components/ui';
  import { Workflow, RefreshCw } from 'lucide-svelte';
  import { IS_BROWSER } from '@lib/context';
  import { appStore } from '@lib/stores/app.svelte';

  let recent = $state<any[]>([]);
  let stats = $state<any>(null);
  let loading = $state(false);

  async function refresh() {
    loading = true;
    try {
      if (IS_BROWSER) return;
      const svc = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/lineageservice');
      recent = ((await svc.GetRecentLineage(200)) ?? []) as any[];
      stats = await svc.GetStats();
    } catch (e: any) {
      appStore.notify(`Lineage load failed: ${e?.message ?? e}`, 'error');
    } finally { loading = false; }
  }
  onMount(refresh);
</script>

<PageLayout title="Data Lineage" subtitle="Provenance for ingested telemetry">
  {#snippet toolbar()}
    <Button variant="secondary" size="sm" icon={RefreshCw} onclick={refresh}>{loading ? 'Loading…' : 'Refresh'}</Button>
    <PopOutButton route="/lineage" title="Lineage" />
  {/snippet}
  <div class="flex flex-col h-full gap-4">
    <div class="grid grid-cols-1 md:grid-cols-3 gap-3">
      <KPI label="Records Tracked" value={(stats?.total_records ?? recent.length).toString()} variant="accent" />
      <KPI label="Unique Sources"  value={(stats?.unique_sources ?? new Set(recent.map((r) => r.source ?? r.origin)).size).toString()} variant="muted" />
      <KPI label="Last 24h"         value={(stats?.recent_24h ?? recent.length).toString()} variant="muted" />
    </div>
    <div class="flex-1 bg-surface-1 border border-border-primary rounded-md overflow-hidden">
      <div class="flex items-center gap-2 p-3 border-b border-border-primary">
        <Workflow size={14} class="text-accent" />
        <span class="text-[10px] uppercase tracking-widest font-bold">Recent Lineage</span>
      </div>
      <DataTable data={recent} columns={[
        { key: 'timestamp', label: 'When', width: '180px' },
        { key: 'source',    label: 'Source', width: '160px' },
        { key: 'entity_id', label: 'Entity' },
        { key: 'operation', label: 'Op',     width: '120px' },
      ]} compact>
        {#snippet render({ col, row })}
          {#if col.key === 'timestamp'}
            <span class="font-mono text-[10px] text-text-muted">{(row.timestamp ?? '').slice(0, 19)}</span>
          {:else}<span class="text-[11px] font-mono">{row[col.key] ?? '—'}</span>{/if}
        {/snippet}
      </DataTable>
      {#if recent.length === 0}
        <div class="p-8 text-center text-sm text-text-muted">{loading ? 'Loading…' : 'No lineage records yet.'}</div>
      {/if}
    </div>
  </div>
</PageLayout>
