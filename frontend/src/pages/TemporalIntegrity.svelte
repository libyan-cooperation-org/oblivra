<!-- Temporal Integrity — TemporalService.GetFleetDriftReport / GetViolations. -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { PageLayout, KPI, Badge, Button, DataTable, PopOutButton } from '@components/ui';
  import { Clock4, RefreshCw } from 'lucide-svelte';
  import { IS_BROWSER } from '@lib/context';
  import { appStore } from '@lib/stores/app.svelte';

  let report = $state<any>(null);
  let drifts = $state<any[]>([]);
  let violations = $state<any[]>([]);
  let loading = $state(false);

  async function refresh() {
    loading = true;
    try {
      if (IS_BROWSER) return;
      const svc = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/temporalservice');
      [report, drifts, violations] = await Promise.all([
        svc.GetFleetDriftReport(), svc.GetAgentDrift(), svc.GetViolations(),
      ]);
    } catch (e: any) {
      appStore.notify(`Temporal load failed: ${e?.message ?? e}`, 'error');
    } finally { loading = false; }
  }
  onMount(refresh);
</script>

<PageLayout title="Temporal Integrity" subtitle="Fleet clock drift and event-ordering audit">
  {#snippet toolbar()}
    <Button variant="secondary" size="sm" icon={RefreshCw} onclick={refresh}>{loading ? 'Loading…' : 'Refresh'}</Button>
    <PopOutButton route="/temporal-integrity" title="Temporal Integrity" />
  {/snippet}
  <div class="flex flex-col h-full gap-4">
    <div class="grid grid-cols-1 md:grid-cols-4 gap-3">
      <KPI label="Avg Drift (ms)" value={(report?.avg_drift_ms ?? '—').toString()} variant="accent" />
      <KPI label="Max Drift (ms)" value={(report?.max_drift_ms ?? '—').toString()} variant={(report?.max_drift_ms ?? 0) > 1000 ? 'warning' : 'muted'} />
      <KPI label="Out-of-spec Hosts" value={(drifts.filter((d) => Math.abs(d.drift_ms ?? 0) > 1000)).length.toString()} variant="muted" />
      <KPI label="Sequence Violations" value={violations.length.toString()} variant={violations.length > 0 ? 'critical' : 'muted'} />
    </div>
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-3 flex-1 min-h-0">
      <div class="bg-surface-1 border border-border-primary rounded-md overflow-hidden flex flex-col">
        <div class="flex items-center gap-2 p-3 border-b border-border-primary">
          <Clock4 size={14} class="text-accent" />
          <span class="text-[10px] uppercase tracking-widest font-bold">Per-agent Drift</span>
        </div>
        <DataTable data={drifts} columns={[
          { key: 'agent_id', label: 'Agent' },
          { key: 'drift_ms', label: 'Drift (ms)', width: '100px' },
          { key: 'last_sync', label: 'Last sync',  width: '160px' },
        ]} compact>
          {#snippet render({ col, row })}
            {#if col.key === 'drift_ms'}<span class="font-mono text-[10px] {Math.abs(row.drift_ms ?? 0) > 1000 ? 'text-warning' : 'text-text-muted'}">{row.drift_ms ?? '—'}</span>
            {:else if col.key === 'last_sync'}<span class="font-mono text-[10px] text-text-muted">{(row.last_sync ?? '').slice(0, 19)}</span>
            {:else}<span class="font-mono text-[10px]">{row[col.key] ?? '—'}</span>{/if}
          {/snippet}
        </DataTable>
      </div>
      <div class="bg-surface-1 border border-border-primary rounded-md overflow-hidden flex flex-col">
        <div class="flex items-center gap-2 p-3 border-b border-border-primary">
          <span class="text-[10px] uppercase tracking-widest font-bold">Sequence Violations</span>
        </div>
        <DataTable data={violations} columns={[
          { key: 'timestamp', label: 'When', width: '160px' },
          { key: 'reason',    label: 'Reason' },
          { key: 'severity',  label: 'Sev',  width: '70px' },
        ]} compact>
          {#snippet render({ col, row })}
            {#if col.key === 'severity'}<Badge variant={row.severity === 'high' ? 'critical' : 'warning'} size="xs">{row.severity ?? '—'}</Badge>
            {:else if col.key === 'timestamp'}<span class="font-mono text-[10px] text-text-muted">{(row.timestamp ?? '').slice(0, 19)}</span>
            {:else}<span class="text-[11px]">{row[col.key] ?? '—'}</span>{/if}
          {/snippet}
        </DataTable>
        {#if violations.length === 0}<div class="p-8 text-center text-sm text-text-muted">No violations detected.</div>{/if}
      </div>
    </div>
  </div>
</PageLayout>
