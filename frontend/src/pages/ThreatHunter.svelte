<!--
  Threat Hunter — query the SIEM with OQL/keyword and pivot from results.
  Replaces the hardcoded "hypotheses" array with the savedQueries store
  (real persisted hunts) and wires the EXECUTE button to siemStore.search.
-->
<script lang="ts">
  import { onMount } from 'svelte';
  import { KPI, PageLayout, Button, Badge, PopOutButton } from '@components/ui';
  import { Search, Crosshair, Save } from 'lucide-svelte';
  import { siemStore } from '@lib/stores/siem.svelte';
  import { savedQueriesStore } from '@lib/stores/savedQueries.svelte';
  import { appStore } from '@lib/stores/app.svelte';
  import { push } from '@lib/router.svelte';

  let query = $state('process.name == "lsass.exe" AND source.type == "external"');
  let running = $state(false);
  let results = $state<any[]>([]);

  async function execute() {
    if (!query.trim()) return;
    running = true;
    try {
      // siemStore.search may return events directly or push them into siemStore.events
      const search = (siemStore as any).search ?? (siemStore as any).runQuery;
      if (typeof search === 'function') {
        const out = await search.call(siemStore, query);
        if (Array.isArray(out)) results = out;
        else results = (siemStore as any).events ?? [];
      } else {
        // Fallback: pivot to the dedicated SIEM search page.
        push(`/siem-search?q=${encodeURIComponent(query)}`);
        return;
      }
      appStore.notify(`Hunt complete — ${results.length} matches`, results.length > 0 ? 'success' : 'info');
    } catch (e: any) {
      appStore.notify(`Hunt failed: ${e?.message ?? e}`, 'error');
    } finally { running = false; }
  }

  function saveAsHunt() {
    if (!query.trim()) return;
    const title = prompt('Save this hunt as:', query.slice(0, 40));
    if (!title) return;
    if (typeof (savedQueriesStore as any).save === 'function') {
      (savedQueriesStore as any).save({ title, query });
      appStore.notify('Hunt saved', 'success');
    } else {
      appStore.notify('Saved-queries store not available', 'warning');
    }
  }

  onMount(() => {
    if (typeof (savedQueriesStore as any).init === 'function') (savedQueriesStore as any).init();
    if (typeof (siemStore as any).init === 'function') (siemStore as any).init();
  });

  let activeHunts = $derived((savedQueriesStore as any).queries ?? (savedQueriesStore as any).items ?? []);
  let totalDetections = $derived(results.length);
</script>

<PageLayout title="Tactical Threat Hunter" subtitle="Hypothesis orchestration with live SIEM pivots">
  {#snippet toolbar()}
    <div class="flex items-center gap-2">
      <Button variant="secondary" size="sm" onclick={() => push('/siem-search')}>Hunt Library</Button>
      <Button variant="primary" size="sm" onclick={saveAsHunt}><Save size={11} class="mr-1" />Save Hunt</Button>
    </div>
    <PopOutButton route="/threat-hunter" title="Threat Hunter" />
  {/snippet}

  <div class="flex flex-col h-full gap-6">
    <div class="grid grid-cols-1 md:grid-cols-4 gap-4 shrink-0">
      <KPI label="Saved Hunts"   value={String(activeHunts.length)} trend="stable" trendValue="From store" />
      <KPI label="Last Run Hits" value={totalDetections.toString()} trend={totalDetections > 0 ? 'up' : 'stable'} variant={totalDetections > 0 ? 'critical' : 'muted'} />
      <KPI label="Engine"        value={running ? 'Running' : 'Idle'} variant={running ? 'accent' : 'muted'} />
      <KPI label="Mode"          value="OQL" variant="success" />
    </div>

    <!-- Saved Hunts -->
    <div class="grid grid-cols-1 md:grid-cols-3 gap-4 shrink-0">
      {#each activeHunts.slice(0, 6) as h, i (h.id ?? h.title ?? i)}
        <button
          class="bg-surface-1 border border-border-primary p-4 rounded-md text-left hover:border-accent transition-colors"
          onclick={() => { query = h.query ?? h.body ?? ''; void execute(); }}
        >
          <Badge variant={h.severity === 'critical' ? 'critical' : h.severity === 'high' ? 'warning' : 'info'} size="xs">
            {(h.severity ?? 'info').toUpperCase()}
          </Badge>
          <div class="text-[11px] font-bold mt-2 truncate">{h.title ?? 'untitled'}</div>
          <div class="text-[10px] text-text-muted font-mono mt-1 truncate">{h.query ?? h.body ?? ''}</div>
        </button>
      {:else}
        <div class="md:col-span-3 text-center text-sm text-text-muted py-4">
          No saved hunts yet. Run a query and save it to build your library.
        </div>
      {/each}
    </div>

    <!-- Hunting Box -->
    <div class="flex-1 flex flex-col bg-surface-1 border border-border-primary rounded-md overflow-hidden">
      <div class="p-4 bg-surface-2 border-b border-border-primary">
        <div class="flex items-center gap-3">
          <div class="flex-1 relative">
            <div class="absolute left-3 top-1/2 -translate-y-1/2 text-accent opacity-60"><Search size={14} /></div>
            <input
              type="text"
              bind:value={query}
              class="w-full bg-surface-0 border border-border-secondary rounded-sm pl-10 pr-4 py-2.5 text-[11px] font-mono text-accent focus:border-accent outline-none"
              placeholder="OQL or keyword query…"
              onkeydown={(e) => e.key === 'Enter' && execute()}
            />
          </div>
          <Button variant="cta" size="sm" onclick={execute} disabled={running}>
            {running ? 'EXECUTING…' : 'EXECUTE'}
          </Button>
        </div>
      </div>

      <div class="flex-1 overflow-auto p-4">
        {#if results.length === 0}
          <div class="flex flex-col items-center justify-center h-full text-center text-text-muted">
            <Crosshair size={32} class="text-accent opacity-20 mb-3" />
            <div class="text-xs uppercase tracking-widest">Ready for Forensics</div>
            <div class="text-[10px] mt-2 max-w-xs">Type an OQL probe above and hit EXECUTE to stream results from the SIEM.</div>
          </div>
        {:else}
          <div class="space-y-1 font-mono text-[11px]">
            {#each results.slice(0, 200) as r, i (r.id ?? i)}
              <div class="bg-surface-2 px-2 py-1 rounded border border-border-primary">
                <span class="text-text-muted text-[10px] mr-2">{r.timestamp ?? r.ts ?? ''}</span>
                <span class="text-accent">{r.host ?? r.host_id ?? ''}</span>
                <span class="ml-2">{r.title ?? r.message ?? r.raw_log ?? JSON.stringify(r).slice(0, 200)}</span>
              </div>
            {/each}
            {#if results.length > 200}
              <div class="text-[10px] text-text-muted text-center pt-2">+ {results.length - 200} more — refine query to narrow.</div>
            {/if}
          </div>
        {/if}
      </div>
    </div>
  </div>
</PageLayout>
