<!-- Chain of Custody — bound to ForensicsService.GetChainOfCustody / ListEvidence. -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { PageLayout, KPI, Badge, Button, DataTable, PopOutButton } from '@components/ui';
  import { Link, RefreshCw } from 'lucide-svelte';
  import { IS_BROWSER } from '@lib/context';
  import { appStore } from '@lib/stores/app.svelte';

  let evidence = $state<any[]>([]);
  let custodyByItem = $state<Record<string, any[]>>({});
  let selected = $state<string | null>(null);
  let loading = $state(false);

  async function refresh() {
    loading = true;
    try {
      if (IS_BROWSER) return;
      const svc = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/forensicsservice');
      evidence = ((await svc.ListEvidence('')) ?? []) as any[];
    } finally { loading = false; }
  }

  async function loadCustody(itemID: string) {
    selected = itemID;
    try {
      const { GetChainOfCustody } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/forensicsservice');
      const c = ((await GetChainOfCustody(itemID)) ?? []) as any[];
      custodyByItem = { ...custodyByItem, [itemID]: c };
    } catch (e: any) {
      appStore.notify(`Custody load failed: ${e?.message ?? e}`, 'error');
    }
  }

  onMount(refresh);
</script>

<PageLayout title="Chain of Custody" subtitle="Forensic evidence handling trail">
  {#snippet toolbar()}
    <Button variant="secondary" size="sm" icon={RefreshCw} onclick={refresh}>{loading ? 'Loading…' : 'Refresh'}</Button>
    <PopOutButton route="/chain-of-custody" title="Custody" />
  {/snippet}
  <div class="flex flex-col h-full gap-4">
    <div class="grid grid-cols-1 md:grid-cols-3 gap-3">
      <KPI label="Evidence Items" value={evidence.length.toString()} variant="accent" />
      <KPI label="Sealed" value={evidence.filter((e) => e.sealed_at).length.toString()} variant="muted" />
      <KPI label="Mode" value={IS_BROWSER ? 'Browser (read-only)' : 'Desktop'} variant="muted" />
    </div>
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-3 flex-1 min-h-0">
      <div class="bg-surface-1 border border-border-primary rounded-md flex flex-col min-h-0">
        <div class="flex items-center gap-2 p-3 border-b border-border-primary">
          <Link size={14} class="text-accent" />
          <span class="text-[10px] uppercase tracking-widest font-bold">Evidence Items</span>
        </div>
        <div class="flex-1 overflow-auto">
          {#each evidence as e (e.id)}
            <button class="w-full text-left px-3 py-2 border-b border-border-primary hover:bg-surface-2 {selected === e.id ? 'bg-surface-2' : ''}" onclick={() => loadCustody(e.id)}>
              <div class="flex items-center gap-2">
                <span class="text-[11px] font-bold flex-1 truncate">{e.name ?? e.evidence_type ?? e.id}</span>
                {#if e.sealed_at}<Badge variant="success" size="xs">sealed</Badge>{/if}
              </div>
              <div class="text-[10px] font-mono text-text-muted truncate">{e.id}</div>
            </button>
          {:else}
            <div class="p-8 text-center text-sm text-text-muted">{loading ? 'Loading…' : 'No evidence items.'}</div>
          {/each}
        </div>
      </div>
      <div class="bg-surface-1 border border-border-primary rounded-md flex flex-col min-h-0">
        <div class="flex items-center gap-2 p-3 border-b border-border-primary">
          <span class="text-[10px] uppercase tracking-widest font-bold">Custody Log</span>
        </div>
        <div class="flex-1 overflow-auto p-3">
          {#if !selected}
            <div class="text-center text-sm text-text-muted py-8">Pick an item to view its custody chain.</div>
          {:else}
            {#each custodyByItem[selected] ?? [] as ev, i (ev.id ?? i)}
              <div class="border-b border-border-primary py-2 text-[11px]">
                <span class="font-mono text-[10px] text-text-muted mr-2">{(ev.timestamp ?? '').slice(0, 19)}</span>
                <span class="font-bold">{ev.actor ?? '—'}</span>
                <span class="text-text-muted">{ev.action ?? ev.event ?? '—'}</span>
                {#if ev.notes}<div class="text-[10px] text-text-muted ml-4">{ev.notes}</div>{/if}
              </div>
            {/each}
          {/if}
        </div>
      </div>
    </div>
  </div>
</PageLayout>
