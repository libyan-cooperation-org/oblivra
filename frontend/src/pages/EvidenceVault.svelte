<!-- Evidence Vault — bound to ForensicsService.ListEvidence + LedgerService.GetChain. -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { PageLayout, KPI, Badge, Button, DataTable, PopOutButton } from '@components/ui';
  import { Lock, RefreshCw, ShieldCheck } from 'lucide-svelte';
  import { IS_BROWSER } from '@lib/context';
  import { appStore } from '@lib/stores/app.svelte';

  let evidence = $state<any[]>([]);
  let chain = $state<any[]>([]);
  let loading = $state(false);

  async function refresh() {
    loading = true;
    try {
      if (IS_BROWSER) return;
      const fs = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/forensicsservice');
      const ld = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/ledgerservice');
      const [e, c] = await Promise.all([fs.ListEvidence(''), ld.GetChain()]);
      evidence = (e ?? []) as any[];
      chain = (c ?? []) as any[];
    } catch (e: any) {
      appStore.notify(`Vault load failed: ${e?.message ?? e}`, 'error');
    } finally { loading = false; }
  }
  onMount(refresh);

  let sealed = $derived(evidence.filter((e) => e.sealed_at).length);
</script>

<PageLayout title="Evidence Vault" subtitle="Sealed forensic artefacts, hash-chain backed">
  {#snippet toolbar()}
    <Button variant="secondary" size="sm" icon={RefreshCw} onclick={refresh}>{loading ? 'Loading…' : 'Refresh'}</Button>
    <PopOutButton route="/evidence-vault" title="Evidence Vault" />
  {/snippet}
  <div class="flex flex-col h-full gap-4">
    <div class="grid grid-cols-1 md:grid-cols-3 gap-3">
      <KPI label="Items" value={evidence.length.toString()} variant="accent" />
      <KPI label="Sealed" value={`${sealed}/${evidence.length || 1}`} variant={sealed === evidence.length ? 'success' : 'warning'} />
      <KPI label="Chain Records" value={chain.length.toString()} variant="muted" />
    </div>
    <div class="flex-1 bg-surface-1 border border-border-primary rounded-md overflow-hidden">
      <div class="flex items-center gap-2 p-3 border-b border-border-primary">
        <Lock size={14} class="text-accent" />
        <span class="text-[10px] uppercase tracking-widest font-bold">Vault Contents</span>
      </div>
      <DataTable data={evidence} columns={[
        { key: 'name',         label: 'Item' },
        { key: 'evidence_type', label: 'Type', width: '120px' },
        { key: 'created_at',   label: 'Captured', width: '160px' },
        { key: 'sealed',       label: '', width: '80px' },
      ]} compact>
        {#snippet render({ col, row })}
          {#if col.key === 'sealed'}{#if row.sealed_at}<Badge variant="success" size="xs"><ShieldCheck size={9} class="mr-0.5 inline" />sealed</Badge>{:else}<Badge variant="muted" size="xs">unsealed</Badge>{/if}
          {:else if col.key === 'created_at'}<span class="font-mono text-[10px] text-text-muted">{(row.created_at ?? '').slice(0, 19)}</span>
          {:else}<span class="text-[11px]">{row[col.key] ?? '—'}</span>{/if}
        {/snippet}
      </DataTable>
      {#if evidence.length === 0}<div class="p-8 text-center text-sm text-text-muted">{loading ? 'Loading…' : 'Vault is empty.'}</div>{/if}
    </div>
  </div>
</PageLayout>
