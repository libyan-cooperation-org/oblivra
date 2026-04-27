<!-- Enrichment Viewer — bound to SIEMService for raw event lookup. -->
<script lang="ts">
  import { PageLayout, Button, KPI, PopOutButton } from '@components/ui';
  import { Sparkles, Search } from 'lucide-svelte';
  import { siemStore } from '@lib/stores/siem.svelte';
  import { appStore } from '@lib/stores/app.svelte';

  let q = $state('');
  let event = $state<any>(null);
  let loading = $state(false);

  async function fetchEvent() {
    if (!q.trim()) return;
    loading = true;
    try {
      const fn = (siemStore as any).getEvent ?? (siemStore as any).enrich;
      if (typeof fn === 'function') event = await fn.call(siemStore, q);
      else appStore.notify('Enrichment RPC not available', 'warning');
    } catch (e: any) {
      appStore.notify(`Enrich failed: ${e?.message ?? e}`, 'error');
    } finally { loading = false; }
  }
</script>

<PageLayout title="Enrichment Viewer" subtitle="Inspect a raw event with TI, geo, and identity overlays">
  {#snippet toolbar()}
    <PopOutButton route="/enrichment" title="Enrichment" />
  {/snippet}
  <div class="flex flex-col h-full gap-4">
    <div class="grid grid-cols-1 md:grid-cols-3 gap-3">
      <KPI label="Mode" value="By event id" variant="muted" />
      <KPI label="Status" value={event ? 'Enriched' : '—'} variant={event ? 'success' : 'muted'} />
      <KPI label="TI Sources" value={(event?.ti_sources?.length ?? 0).toString()} variant="muted" />
    </div>
    <div class="bg-surface-1 border border-border-primary rounded-md p-3 flex items-center gap-2">
      <Search size={14} class="text-text-muted" />
      <input class="flex-1 bg-surface-2 border border-border-primary rounded px-2 py-1.5 text-xs outline-none focus:border-accent font-mono" placeholder="Event id (uuid)…" bind:value={q} onkeydown={(e) => e.key === 'Enter' && fetchEvent()} />
      <Button variant="cta" size="sm" onclick={fetchEvent} disabled={loading}>{loading ? 'Loading…' : 'Enrich'}</Button>
    </div>
    <div class="flex-1 bg-surface-1 border border-border-primary rounded-md overflow-hidden">
      <div class="flex items-center gap-2 p-3 border-b border-border-primary">
        <Sparkles size={14} class="text-accent" />
        <span class="text-[10px] uppercase tracking-widest font-bold">Enriched View</span>
      </div>
      <div class="p-3 overflow-auto h-full">
        {#if event}
          <pre class="font-mono text-[11px] whitespace-pre-wrap">{JSON.stringify(event, null, 2)}</pre>
        {:else}
          <div class="text-center text-sm text-text-muted py-12">{loading ? 'Loading…' : 'Paste an event id above and hit Enrich.'}</div>
        {/if}
      </div>
    </div>
  </div>
</PageLayout>
