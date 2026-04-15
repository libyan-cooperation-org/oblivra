<!--
  OBLIVRA — Tunnel Manager (Svelte 5)
  SSH Port Forwarding and Reverse Tunnels orchestration — live from TunnelService.
-->
<script lang="ts">
  import { onMount } from 'svelte';
  import { appStore } from '@lib/stores/app.svelte';
  import { KPI, Badge, DataTable, PageLayout, Button, EmptyState, SearchBar } from '@components/ui';
  import { IS_BROWSER } from '@lib/context';

  interface TunnelInfo {
    ID: string;
    SessionID: string;
    Type: string;
    LocalHost: string;
    LocalPort: number;
    RemoteHost: string;
    RemotePort: number;
    Status: string;
    BytesSent: number;
    BytesReceived: number;
    StartedAt: string;
  }

  let tunnels  = $state<TunnelInfo[]>([]);
  let loading  = $state(false);
  let searchQ  = $state('');

  const filtered = $derived(
    searchQ.trim()
      ? tunnels.filter(t =>
          `${t.LocalPort} ${t.RemoteHost} ${t.RemotePort} ${t.Type} ${t.SessionID}`
            .toLowerCase().includes(searchQ.toLowerCase())
        )
      : tunnels
  );

  const activeCnt = $derived(tunnels.filter(t => t.Status === 'active' || t.Status === 'running').length);

  async function load() {
    if (IS_BROWSER) return;
    loading = true;
    try {
      const { GetAll } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/tunnelservice');
      tunnels = ((await GetAll()) || []) as TunnelInfo[];
    } catch (e: any) {
      appStore.notify('Could not load tunnels', 'error', e?.message);
    } finally {
      loading = false;
    }
  }

  async function stopTunnel(id: string) {
    if (IS_BROWSER) return;
    try {
      const { StopTunnel } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/tunnelservice');
      await StopTunnel(id);
      await load();
      appStore.notify('Tunnel stopped', 'info');
    } catch (e: any) {
      appStore.notify('Stop failed', 'error', e?.message);
    }
  }

  const columns = [
    { key: 'type',        label: 'Type',         width: '90px'  },
    { key: 'local',       label: 'Local',         width: '110px' },
    { key: 'remote',      label: 'Remote',        width: '180px' },
    { key: 'session',     label: 'Session',       width: '130px' },
    { key: 'status',      label: 'Status',        width: '100px' },
    { key: 'actions',     label: '',              width: '60px'  },
  ];

  onMount(load);
</script>

<PageLayout title="Tunnel Manager" subtitle="Encrypted port forwarding and reverse proxy orchestration">
  {#snippet toolbar()}
    <SearchBar bind:value={searchQ} placeholder="Filter tunnels…" compact />
    <Button variant="secondary" size="sm" onclick={load}>
      {loading ? '⟳' : '↺'} Refresh
    </Button>
  {/snippet}

  <div class="flex flex-col h-full gap-5">
    <div class="grid grid-cols-1 md:grid-cols-3 gap-4 shrink-0">
      <KPI label="Active Tunnels"  value={activeCnt}       variant={activeCnt > 0 ? 'success' : 'muted'} />
      <KPI label="Total Tunnels"   value={tunnels.length}  trend="stable" />
      <KPI label="Mode"            value={IS_BROWSER ? 'Browser (read-only)' : 'Desktop'} variant="accent" />
    </div>

    {#if IS_BROWSER}
      <EmptyState title="Desktop feature" description="Tunnel management requires the OBLIVRA desktop binary." icon="🔒" />
    {:else if loading && tunnels.length === 0}
      <div class="text-[11px] text-text-muted font-mono p-4 animate-pulse">Loading tunnels…</div>
    {:else if filtered.length === 0}
      <EmptyState
        title={searchQ ? 'No matches' : 'No tunnels active'}
        description={searchQ ? 'Try a different filter.' : 'Open an SSH session and create a tunnel from the terminal toolbar.'}
        icon="🚇"
      />
    {:else}
      <div class="flex-1 min-h-0 overflow-auto bg-surface-1 border border-border-primary rounded-sm">
        <table class="w-full text-left min-w-[600px]">
          <thead class="sticky top-0">
            <tr class="bg-surface-2 border-b border-border-primary text-[9px] font-bold uppercase tracking-widest text-text-muted">
              <th class="px-3 py-2">Type</th>
              <th class="px-3 py-2">Local</th>
              <th class="px-3 py-2">Remote</th>
              <th class="px-3 py-2">Session</th>
              <th class="px-3 py-2">Status</th>
              <th class="px-3 py-2 text-right">Actions</th>
            </tr>
          </thead>
          <tbody>
            {#each filtered as t (t.ID)}
              <tr class="border-b border-border-primary hover:bg-surface-2/50 transition-colors">
                <td class="px-3 py-2">
                  <Badge variant="muted">{t.Type || 'local'}</Badge>
                </td>
                <td class="px-3 py-2 font-mono text-[11px] text-accent">:{t.LocalPort}</td>
                <td class="px-3 py-2 font-mono text-[11px] text-text-secondary">{t.RemoteHost}:{t.RemotePort}</td>
                <td class="px-3 py-2 text-[10px] text-text-muted font-mono">{t.SessionID?.slice(0, 10) ?? '—'}</td>
                <td class="px-3 py-2">
                  <Badge variant={t.Status === 'active' || t.Status === 'running' ? 'success' : t.Status === 'error' ? 'critical' : 'warning'} dot>
                    {t.Status}
                  </Badge>
                </td>
                <td class="px-3 py-2 text-right">
                  <Button variant="danger" size="xs" onclick={() => stopTunnel(t.ID)}>Stop</Button>
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/if}
  </div>
</PageLayout>
