<!--
  OBLIVRA — Fleet Dashboard (Svelte 5)
  Real-time visibility into the sovereign agent fleet.
-->
<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { PageLayout, Badge, Button, DataTable, Input, Tabs, PopOutButton } from '@components/ui';
  import { Activity, Terminal, ShieldAlert, MoreHorizontal, Monitor, Clock, ShieldCheck } from 'lucide-svelte';
  import { agentStore } from '@lib/stores/agent.svelte';
  import { appStore } from '@lib/stores/app.svelte';
  import { push } from '@lib/router.svelte';

  let searchQuery = $state('');
  let activeTab = $state('ALL HOSTS');
  let now = $state(Date.now());
  let lastSync = $state<Date | null>(null);
  let tickTimer: ReturnType<typeof setInterval> | null = null;

  // Health = online / total. Honest 0% when fleet empty rather than fake "98.2%".
  const stats = $derived.by(() => {
    const all = agentStore.agents ?? [];
    const total = all.length;
    const online = all.filter((a) => a.status === 'online' || a.status === 'active').length;
    const critical = all.filter((a) => (a as any).severity === 'critical' || (a as any).quarantined).length;
    const healthPct = total === 0 ? 0 : Math.round((online / total) * 100);
    return { total, online, critical, health: total === 0 ? '—' : `${healthPct}%` };
  });

  const filteredAgents = $derived(
    (agentStore.agents ?? []).filter((a) => {
      const q = searchQuery.toLowerCase();
      const matchesSearch =
        (a.hostname ?? '').toLowerCase().includes(q) ||
        (a.remote_address ?? '').toLowerCase().includes(q);
      const matchesTab = activeTab === 'ALL HOSTS' || (a.status ?? '').toUpperCase() === activeTab;
      return matchesSearch && matchesTab;
    }),
  );

  const tabItems = [
    { id: 'ALL HOSTS', label: 'ALL HOSTS' },
    { id: 'ONLINE',    label: 'ONLINE' },
    { id: 'OFFLINE',   label: 'OFFLINE' },
  ];

  function fmtAgo(d: Date | null): string {
    if (!d) return '—';
    const sec = Math.floor((now - d.getTime()) / 1000);
    if (sec < 60) return `${sec}s ago`;
    if (sec < 3600) return `${Math.floor(sec / 60)}m ago`;
    return `${Math.floor(sec / 3600)}h ago`;
  }

  function exportList() {
    const csv = [
      'id,hostname,address,status,os,arch,version',
      ...filteredAgents.map((a) =>
        [a.id, a.hostname, a.remote_address, a.status, a.os, a.arch, a.version]
          .map((x) => `"${(x ?? '').toString().replace(/"/g, '""')}"`)
          .join(','),
      ),
    ].join('\n');
    const blob = new Blob([csv], { type: 'text/csv' });
    const url = URL.createObjectURL(blob);
    const a = document.createElement('a');
    a.href = url;
    a.download = `oblivra-fleet-${new Date().toISOString().slice(0, 10)}.csv`;
    a.click();
    URL.revokeObjectURL(url);
    appStore.notify(`Exported ${filteredAgents.length} agents`, 'success');
  }

  function deployAgent() {
    push('/agents');
  }

  function openTerminalForAgent(agentId: string, hostname: string) {
    // Pivot to /shell so the operator can spawn a session for this host.
    appStore.notify(`Opening shell for ${hostname || agentId}`, 'info');
    push('/shell');
  }

  async function isolate(agentId: string) {
    if (!confirm(`Quarantine agent ${agentId}? This blocks its outbound traffic.`)) return;
    try {
      await agentStore.toggleQuarantine(agentId, true);
      appStore.notify(`Agent ${agentId} isolated`, 'warning');
    } catch (e: any) {
      appStore.notify(`Isolation failed: ${e?.message ?? e}`, 'error');
    }
  }

  onMount(() => {
    if (typeof agentStore.init === 'function') {
      const r = agentStore.init();
      if (r && typeof (r as any).then === 'function') void (r as Promise<unknown>).then(() => (lastSync = new Date()));
      else lastSync = new Date();
    } else {
      lastSync = new Date();
    }
    tickTimer = setInterval(() => (now = Date.now()), 1000);
  });
  onDestroy(() => {
    if (tickTimer) clearInterval(tickTimer);
  });
</script>

<PageLayout title="Fleet Management" subtitle="Real-time orchestration of the sovereign agent mesh">
  {#snippet toolbar()}
    <div class="flex items-center gap-2">
      <Input variant="search" placeholder="Filter agents..." bind:value={searchQuery} class="w-64" />
      <Button variant="secondary" size="sm" onclick={exportList}>EXPORT LIST</Button>
      <Button variant="primary"   size="sm" onclick={deployAgent}>DEPLOY AGENT</Button>
      <PopOutButton route="/fleet" title="Fleet Management" />
    </div>
  {/snippet}

  <div class="flex flex-col h-full gap-0 -m-6">
    <!-- METRIC STRIP -->
    <div class="grid grid-cols-4 gap-px bg-border-primary border-b border-border-primary shrink-0">
        <div class="bg-surface-2 p-3 group hover:bg-surface-3 transition-colors cursor-pointer">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">Total Fleet</div>
            <div class="text-xl font-mono font-bold text-text-heading">{stats.total}</div>
            <div class="text-[9px] text-text-muted mt-1">Managed endpoints</div>
        </div>
        <div class="bg-surface-2 p-3 group hover:bg-surface-3 transition-colors cursor-pointer">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">Live Nodes</div>
            <div class="text-xl font-mono font-bold text-success">{stats.online}</div>
            <div class="text-[9px] text-success mt-1">Active heartbeats</div>
        </div>
        <div class="bg-surface-2 p-3 group hover:bg-surface-3 transition-colors cursor-pointer">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">Health Score</div>
            <div class="text-xl font-mono font-bold text-accent">{stats.health}</div>
            <div class="text-[9px] text-accent mt-1">Fleet stability nominal</div>
        </div>
        <div class="bg-surface-2 p-3 group hover:bg-surface-3 transition-colors cursor-pointer border-r-0">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">Critical Issues</div>
            <div class="text-xl font-mono font-bold text-error">{stats.critical}</div>
            <div class="text-[9px] text-error mt-1 uppercase animate-pulse">Action required</div>
        </div>
    </div>

    <!-- MAIN TABLE AREA -->
    <div class="flex-1 flex flex-col min-h-0 bg-surface-1">
        <div class="px-4 py-2 border-b border-border-primary flex items-center justify-between shrink-0">
            <Tabs tabs={tabItems} bind:active={activeTab} />
            <div class="flex items-center gap-4 text-[8px] font-mono text-text-muted">
                <span>Last Sync: {fmtAgo(lastSync)}</span>
                <span class="w-2 h-2 rounded-full {lastSync ? 'bg-success animate-pulse' : 'bg-text-muted'}"></span>
            </div>
        </div>

        <div class="flex-1 overflow-auto mask-fade-bottom">
            <DataTable 
                data={filteredAgents} 
                columns={[
                    { key: 'status', label: 'STATUS', width: '80px' },
                    { key: 'hostname', label: 'ENDPOINT_NAME' },
                    { key: 'remote_address', label: 'IP_ADDRESS', width: '120px' },
                    { key: 'os', label: 'PLATFORM', width: '100px' },
                    { key: 'arch', label: 'ARCH', width: '80px' },
                    { key: 'version', label: 'VERSION', width: '80px' },
                    { key: 'id', label: 'ACTIONS', width: '120px' }
                ]} 
                compact
            >
                {#snippet render({ col, row })}
                    {#if col.key === 'status'}
                        <div class="flex items-center gap-2">
                            <div class="w-1.5 h-1.5 rounded-full {row.status === 'online' ? 'bg-success shadow-[0_0_4px_rgba(34,197,94,0.6)]' : 'bg-text-muted'}"></div>
                            <Badge variant={row.status === 'online' ? 'success' : 'muted'} size="xs">{row.status}</Badge>
                        </div>
                    {:else if col.key === 'hostname'}
                        <!-- Drill-down: clicking the hostname pivots to the
                             single-pane-of-glass HostDetail page (Phase 30.1).
                             The keyboard variant fires on Enter/Space so this
                             row is fully accessible. -->
                        <button
                            type="button"
                            class="flex items-center gap-2 py-1 bg-transparent border-none cursor-pointer text-left hover:text-accent transition-colors w-full"
                            onclick={() => push(`/host/${encodeURIComponent(row.id)}`)}
                            title="Open host detail page"
                        >
                            <Monitor size={12} class="text-text-muted" />
                            <div class="flex flex-col">
                                <span class="text-[10px] font-bold text-text-heading uppercase hover:text-accent">{row.hostname}</span>
                                <span class="text-[8px] font-mono text-text-muted opacity-60 tabular-nums">{row.id}</span>
                            </div>
                        </button>
                    {:else if col.key === 'remote_address'}
                        <span class="text-[9px] font-mono text-accent tabular-nums">{row.remote_address}</span>
                    {:else if col.key === 'os'}
                        <div class="flex items-center gap-1.5">
                           <span class="text-[9px] font-mono text-text-muted uppercase">{row.os || 'Linux'}</span>
                        </div>
                    {:else if col.key === 'arch'}
                        <span class="text-[8px] font-mono text-text-muted uppercase opacity-60">{row.arch || 'x64'}</span>
                    {:else if col.key === 'version'}
                        <span class="text-[8px] font-mono text-text-muted">v{row.version || '1.0'}</span>
                    {:else if col.key === 'id'}
                        <div class="flex items-center gap-1.5 opacity-0 group-hover:opacity-100 transition-opacity">
                            <button
                                class="p-1 hover:bg-surface-3 rounded-sm text-accent border border-accent/20 transition-colors"
                                title="Open terminal for this host"
                                onclick={() => openTerminalForAgent(row.id, row.hostname)}
                            >
                                <Terminal size={12} />
                            </button>
                            <button
                                class="p-1 hover:bg-surface-3 rounded-sm text-error border border-error/20 transition-colors"
                                title="Isolate (quarantine)"
                                onclick={() => isolate(row.id)}
                            >
                                <ShieldAlert size={12} />
                            </button>
                            <button
                                class="p-1 hover:bg-surface-3 rounded-sm text-text-muted border border-border-primary transition-colors"
                                title="Open host detail"
                                onclick={() => push(`/host/${encodeURIComponent(row.id)}`)}
                            >
                                <MoreHorizontal size={12} />
                            </button>
                        </div>
                    {/if}
                {/snippet}
            </DataTable>
        </div>
    </div>

    <!-- STATUS BAR -->
    <div class="bg-surface-2 border-t border-border-primary px-3 py-1.5 flex items-center justify-between shrink-0">
        <div class="flex items-center gap-6">
            <div class="flex items-center gap-2 text-[8px] font-mono text-text-muted uppercase">
                <Activity size={10} class={stats.online > 0 ? 'text-success' : 'text-text-muted'} />
                <span>Sync Pipeline: {stats.online > 0 ? 'Nominal' : 'Idle'}</span>
            </div>
            <div class="flex items-center gap-2 text-[8px] font-mono text-text-muted uppercase">
                <Clock size={10} class="text-accent" />
                <span>Last Sync: {fmtAgo(lastSync)}</span>
            </div>
            <div class="flex items-center gap-2 text-[8px] font-mono text-text-muted uppercase">
                <ShieldCheck size={10} class={stats.critical === 0 ? 'text-success' : 'text-warning'} />
                <span>Integrity: {stats.health}</span>
            </div>
        </div>
        <div class="text-[8px] font-mono text-text-muted uppercase tracking-[0.2em] opacity-40">
            Fleet_Core v1.4.2 — Sovereign Mesh
        </div>
    </div>
  </div>
</PageLayout>
