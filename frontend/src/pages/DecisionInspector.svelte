<!-- Decision Inspector — bound to DecisionService. -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { PageLayout, KPI, Button, DataTable, PopOutButton } from '@components/ui';
  import { Brain, RefreshCw, Search } from 'lucide-svelte';
  import { IS_BROWSER } from '@lib/context';
  import { appStore } from '@lib/stores/app.svelte';

  let decisions = $state<any[]>([]);
  let stats = $state<any>(null);
  let selected = $state<any>(null);
  let trace = $state<any>(null);
  let loading = $state(false);

  async function refresh() {
    loading = true;
    try {
      if (IS_BROWSER) return;
      const svc = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/decisionservice');
      decisions = ((await svc.ListRecentDecisions(200)) ?? []) as any[];
      stats = await svc.GetStats();
    } finally { loading = false; }
  }

  async function inspect(id: string) {
    try {
      const svc = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/decisionservice');
      [selected, trace] = await Promise.all([svc.GetExplanation(id), svc.GetDecisionTrace(id)]);
    } catch (e: any) {
      appStore.notify(`Trace load failed: ${e?.message ?? e}`, 'error');
    }
  }

  onMount(refresh);
</script>

<PageLayout title="Decision Inspector" subtitle="Audit autonomous platform decisions">
  {#snippet toolbar()}
    <Button variant="secondary" size="sm" icon={RefreshCw} onclick={refresh}>{loading ? 'Loading…' : 'Refresh'}</Button>
    <PopOutButton route="/decisions" title="Decisions" />
  {/snippet}

  <div class="flex flex-col h-full gap-4">
    <div class="grid grid-cols-1 md:grid-cols-3 gap-3">
      <KPI label="Total Decisions" value={(stats?.total ?? decisions.length).toString()} variant="accent" />
      <KPI label="Last 24h"        value={(stats?.recent_24h ?? '—').toString()} variant="muted" />
      <KPI label="Auto-actions"    value={(stats?.automated ?? '—').toString()} variant="muted" />
    </div>

    <div class="grid grid-cols-1 lg:grid-cols-2 gap-3 flex-1 min-h-0">
      <div class="bg-surface-1 border border-border-primary rounded-md overflow-hidden flex flex-col min-h-0">
        <div class="flex items-center gap-2 p-3 border-b border-border-primary">
          <Brain size={14} class="text-accent" />
          <span class="text-[10px] uppercase tracking-widest font-bold">Recent Decisions</span>
        </div>
        <div class="flex-1 overflow-auto">
          <DataTable data={decisions} columns={[
            { key: 'timestamp', label: 'When', width: '160px' },
            { key: 'action',    label: 'Action' },
            { key: 'inspect',   label: '', width: '60px' },
          ]} compact>
            {#snippet render({ col, row })}
              {#if col.key === 'timestamp'}<span class="font-mono text-[10px] text-text-muted">{(row.timestamp ?? '').slice(0, 19)}</span>
              {:else if col.key === 'inspect'}<Button variant="ghost" size="xs" onclick={() => inspect(row.id)}><Search size={11} /></Button>
              {:else}<span class="text-[11px]">{row[col.key] ?? '—'}</span>{/if}
            {/snippet}
          </DataTable>
        </div>
      </div>

      <div class="bg-surface-1 border border-border-primary rounded-md flex flex-col min-h-0">
        <div class="flex items-center gap-2 p-3 border-b border-border-primary">
          <span class="text-[10px] uppercase tracking-widest font-bold">Trace + Explanation</span>
        </div>
        <div class="flex-1 overflow-auto p-3">
          {#if !selected}
            <div class="text-center text-sm text-text-muted py-8">Pick a decision to inspect.</div>
          {:else}
            <pre class="font-mono text-[10px] whitespace-pre-wrap">{JSON.stringify({ explanation: selected, trace }, null, 2)}</pre>
          {/if}
        </div>
      </div>
    </div>
  </div>
</PageLayout>
