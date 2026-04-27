<!-- Evidence Ledger — bound to LedgerService. -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { PageLayout, KPI, Button, Badge, DataTable, PopOutButton } from '@components/ui';
  import { BookLock, RefreshCw, ShieldCheck, Download } from 'lucide-svelte';
  import { IS_BROWSER } from '@lib/context';
  import { appStore } from '@lib/stores/app.svelte';

  let chain = $state<any[]>([]);
  let verified = $state<{ valid: boolean; head?: string } | null>(null);
  let loading = $state(false);

  async function refresh() {
    loading = true;
    try {
      if (IS_BROWSER) return;
      const svc = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/ledgerservice');
      chain = ((await svc.GetChain()) ?? []) as any[];
      const v = await svc.VerifyChain();
      verified = v as any;
    } finally { loading = false; }
  }

  async function exportChain() {
    try {
      const { ExportChain } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/ledgerservice');
      const data = await ExportChain();
      const blob = new Blob([typeof data === 'string' ? data : JSON.stringify(data)], { type: 'application/json' });
      const url = URL.createObjectURL(blob);
      const a = document.createElement('a'); a.href = url; a.download = `evidence-ledger-${Date.now()}.json`; a.click();
      URL.revokeObjectURL(url);
      appStore.notify('Chain exported', 'success');
    } catch (e: any) { appStore.notify(`Export failed: ${e?.message ?? e}`, 'error'); }
  }

  onMount(refresh);
</script>

<PageLayout title="Evidence Ledger" subtitle="Hash-linked, append-only audit trail">
  {#snippet toolbar()}
    <Button variant="secondary" size="sm" icon={RefreshCw} onclick={refresh}>{loading ? 'Loading…' : 'Refresh'}</Button>
    <Button variant="primary" size="sm" icon={Download} onclick={exportChain}>Export Chain</Button>
    <PopOutButton route="/ledger" title="Ledger" />
  {/snippet}

  <div class="flex flex-col h-full gap-4">
    <div class="grid grid-cols-1 md:grid-cols-3 gap-3">
      <KPI label="Records" value={chain.length.toString()} variant="accent" />
      <KPI label="Chain Valid" value={verified?.valid === true ? 'Yes' : verified === null ? '—' : 'BROKEN'} variant={verified?.valid ? 'success' : 'critical'} />
      <KPI label="Head Hash" value={(verified?.head ?? chain.at(-1)?.hash ?? '—').toString().slice(0, 16)} variant="muted" />
    </div>
    <div class="flex-1 bg-surface-1 border border-border-primary rounded-md overflow-hidden">
      <div class="flex items-center gap-2 p-3 border-b border-border-primary">
        <BookLock size={14} class="text-accent" />
        <span class="text-[10px] uppercase tracking-widest font-bold">Chain Entries</span>
        {#if verified?.valid}<ShieldCheck size={12} class="text-success ml-auto" />{/if}
      </div>
      <DataTable data={chain} columns={[
        { key: 'index',     label: '#',     width: '70px' },
        { key: 'timestamp', label: 'When',  width: '180px' },
        { key: 'data_type', label: 'Type',  width: '120px' },
        { key: 'hash',      label: 'Hash' },
      ]} compact>
        {#snippet render({ col, row })}
          {#if col.key === 'data_type'}<Badge variant="info" size="xs">{row.data_type ?? '—'}</Badge>
          {:else if col.key === 'hash'}<span class="font-mono text-[10px] text-text-muted truncate">{(row.hash ?? '').slice(0, 24)}…</span>
          {:else if col.key === 'timestamp'}<span class="font-mono text-[10px] text-text-muted">{(row.timestamp ?? '').slice(0, 19)}</span>
          {:else}<span class="font-mono text-[10px]">{row[col.key] ?? '—'}</span>{/if}
        {/snippet}
      </DataTable>
      {#if chain.length === 0}<div class="p-8 text-center text-sm text-text-muted">{loading ? 'Loading…' : 'Chain is empty.'}</div>{/if}
    </div>
  </div>
</PageLayout>
