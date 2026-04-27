<!--
  NDR Overview — live network traffic from NDRService.GetLiveTraffic.
-->
<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { PageLayout, Badge, Button, KPI, DataTable, PopOutButton } from '@components/ui';
  import { Wifi, AlertTriangle, RefreshCw } from 'lucide-svelte';
  import { IS_BROWSER } from '@lib/context';
  import { appStore } from '@lib/stores/app.svelte';

  type Flow = {
    id?: string;
    src_ip?: string; dst_ip?: string; src_port?: number; dst_port?: number;
    protocol?: string; bytes?: number; packets?: number;
    classification?: string; suspicious?: boolean; first_seen?: string;
  };

  let flows = $state<Flow[]>([]);
  let loading = $state(false);
  let timer: ReturnType<typeof setInterval> | null = null;

  const stats = $derived.by(() => {
    const total = flows.length;
    const suspicious = flows.filter((f) => f.suspicious).length;
    const totalBytes = flows.reduce((s, f) => s + (f.bytes ?? 0), 0);
    const protocols = new Set(flows.map((f) => f.protocol).filter(Boolean)).size;
    return { total, suspicious, totalBytes, protocols };
  });

  function fmtBytes(b: number): string {
    if (b < 1024) return `${b}B`;
    if (b < 1024 * 1024) return `${(b / 1024).toFixed(1)}KB`;
    if (b < 1024 * 1024 * 1024) return `${(b / 1024 / 1024).toFixed(1)}MB`;
    return `${(b / 1024 / 1024 / 1024).toFixed(2)}GB`;
  }

  async function refresh() {
    loading = true;
    try {
      if (IS_BROWSER) { flows = []; return; }
      const { GetLiveTraffic } = await import(
        '@wailsjs/github.com/kingknull/oblivrashell/internal/services/ndrservice'
      );
      flows = ((await GetLiveTraffic()) ?? []) as Flow[];
    } catch (e: any) {
      appStore.notify(`NDR fetch failed: ${e?.message ?? e}`, 'error');
    } finally { loading = false; }
  }

  onMount(() => {
    void refresh();
    timer = setInterval(refresh, 10_000);
  });
  onDestroy(() => { if (timer) clearInterval(timer); });
</script>

<PageLayout title="Network Detection & Response" subtitle="Live flow analytics and traffic intel">
  {#snippet toolbar()}
    <Button variant="secondary" size="sm" icon={RefreshCw} onclick={refresh}>{loading ? 'Loading…' : 'Refresh'}</Button>
    <PopOutButton route="/ndr" title="NDR Overview" />
  {/snippet}

  <div class="flex flex-col h-full gap-4">
    <div class="grid grid-cols-1 md:grid-cols-4 gap-3 shrink-0">
      <KPI label="Active Flows"   value={stats.total.toString()}                variant="accent" />
      <KPI label="Suspicious"     value={stats.suspicious.toString()}           variant={stats.suspicious > 0 ? 'critical' : 'muted'} />
      <KPI label="Total Volume"   value={fmtBytes(stats.totalBytes)}            variant="muted" />
      <KPI label="Protocols Seen" value={stats.protocols.toString()}            variant="muted" />
    </div>

    <div class="flex-1 min-h-0 bg-surface-1 border border-border-primary rounded-md overflow-hidden">
      <div class="flex items-center gap-2 p-3 border-b border-border-primary">
        <Wifi size={14} class="text-accent" />
        <span class="text-[10px] uppercase tracking-widest font-bold">Live Flow Table</span>
        <span class="ml-auto text-[10px] text-text-muted">refreshes every 10s</span>
      </div>
      {#if flows.length === 0}
        <div class="p-12 text-center text-sm text-text-muted">{loading ? 'Loading…' : 'No active flows.'}</div>
      {:else}
        <DataTable
          data={flows}
          columns={[
            { key: 'flag',           label: '',         width: '24px' },
            { key: 'protocol',       label: 'Proto',    width: '70px' },
            { key: 'src_ip',         label: 'Source',   width: '160px' },
            { key: 'dst_ip',         label: 'Dest',     width: '160px' },
            { key: 'bytes',          label: 'Bytes',    width: '90px' },
            { key: 'packets',        label: 'Packets',  width: '80px' },
            { key: 'classification', label: 'Class' },
            { key: 'first_seen',     label: 'Seen',     width: '140px' },
          ]}
          compact
        >
          {#snippet render({ col, row })}
            {#if col.key === 'flag'}
              {#if row.suspicious}<AlertTriangle size={11} class="text-error" />{/if}
            {:else if col.key === 'protocol'}
              <span class="font-mono text-[10px] uppercase">{row.protocol ?? '?'}</span>
            {:else if col.key === 'src_ip'}
              <span class="font-mono text-[10px] text-accent">{row.src_ip ?? '?'}{row.src_port ? `:${row.src_port}` : ''}</span>
            {:else if col.key === 'dst_ip'}
              <span class="font-mono text-[10px] text-accent">{row.dst_ip ?? '?'}{row.dst_port ? `:${row.dst_port}` : ''}</span>
            {:else if col.key === 'bytes'}
              <span class="font-mono text-[10px] text-text-muted">{fmtBytes(row.bytes ?? 0)}</span>
            {:else if col.key === 'classification'}
              {#if row.classification}
                <Badge variant={row.suspicious ? 'critical' : 'info'} size="xs">{row.classification}</Badge>
              {/if}
            {:else if col.key === 'first_seen'}
              <span class="font-mono text-[10px] text-text-muted">{(row.first_seen ?? '').slice(11, 19) || '—'}</span>
            {:else}
              <span class="text-[11px]">{row[col.key] ?? '—'}</span>
            {/if}
          {/snippet}
        </DataTable>
      {/if}
    </div>
  </div>
</PageLayout>
