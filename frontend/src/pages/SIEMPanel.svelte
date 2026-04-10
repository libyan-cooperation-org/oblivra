<!--
  OBLIVRA — SIEM Panel (Svelte 5)
  Central hub for security event monitoring and log aggregation.
-->
<script lang="ts">
  import { onMount } from 'svelte';
  import { appStore } from '@lib/stores/app.svelte';
  import { IS_BROWSER } from '@lib/context';
  import { KPI, Badge, DataTable, PageLayout, Button, SearchBar, Tabs } from '@components/ui';



  const siemTabs = [
    { id: 'feed', label: 'Live Feed', icon: '📡' },
    { id: 'analytics', label: 'Analytics', icon: '📊' },
    { id: 'threat-map', label: 'Threat Map', icon: '🗺️' },
    { id: 'detections', label: 'Rules Control', icon: '🛡️' },
  ];

  const logColumns = [
    { key: 'time', label: 'TIMESTAMP', width: '140px' },
    { key: 'type', label: 'TYPE', width: '80px' },
    { key: 'host', label: 'HOST', width: '120px' },
    { key: 'message', label: 'EVENT MESSAGE' },
  ];

  const mockLogs = [
    { id: 1, time: '2026-04-10 02:40:01', type: 'critical', host: 'oblivra-core-01', message: 'Unauthorized egress attempt to 45.33.2.1 blocked by NDR policy #442' },
    { id: 2, time: '2026-04-10 02:41:12', type: 'info', host: 'jump-proxy-de', message: 'SSH session established for user: maverick' },
    { id: 3, time: '2026-04-10 02:42:05', type: 'warning', host: 'prod-api-05', message: 'Unexpected process "nc" executed in /tmp' },
    { id: 4, time: '2026-04-10 02:43:33', type: 'error', host: 'vault-auth-01', message: 'MFA failure: Invalid TOTP for account: root' },
    { id: 5, time: '2026-04-10 02:44:10', type: 'info', host: 'oblivra-core-01', message: 'Kernel integrity check passed' },
  ];

  let activeTab = $state('feed');
  let searchQuery = $state('');
  let logs = $state<any[]>([]);
  let loading = $state(false);

  async function refreshLogs() {
    if (IS_BROWSER) {
      logs = mockLogs;
      return;
    }
    
    loading = true;
    try {
      const { ExecuteOQL } = await import('@wailsjs/go/services/SIEMService.js');
      const query = searchQuery || 'severity:>=info';
      const result = await ExecuteOQL(query);
      
      // Map OQL results to UI table format
      if (result && result.Events) {
        logs = result.Events.map((ev: any) => ({
          id: ev.ID,
          time: new Date(ev.Timestamp).toLocaleString(),
          type: ev.Severity || 'info',
          host: ev.HostName || 'unknown',
          message: ev.Message || ev.RawData || 'No message'
        }));
      }
    } catch (err) {
      appStore.notify('Failed to execute OQL query', 'error', (err as Error).message);
    } finally {
      loading = false;
    }
  }

  onMount(() => {
    refreshLogs();
  });

  // ── Stats simulation ──────────────────────────────────────────────────
  let eps = $state(1420);
  let epsTrend = $state('+12%');
  
  $effect(() => {
    const interval = setInterval(() => {
      eps = 1400 + Math.floor(Math.random() * 50);
    }, 2000);
    return () => clearInterval(interval);
  });
</script>

<PageLayout title="SIEM Console" subtitle="Unified event orchestration and threat detection">
  {#snippet toolbar()}
    <div class="flex items-center gap-3">
      <Input variant="search" placeholder="Search events..." bind:value={searchQuery} onkeydown={(e: KeyboardEvent) => e.key === 'Enter' && refreshLogs()} />
      <Button variant="secondary" size="sm" onclick={refreshLogs}>Refresh</Button>
      <Button variant="cta" size="sm" onclick={() => appStore.notify('Feed Paused', 'warning')}>Pause Feed</Button>
    </div>
  {/snippet}

  <div class="flex flex-col h-full gap-5">
    <!-- Top KPI Grid -->
    <div class="grid grid-cols-1 md:grid-cols-4 gap-4 shrink-0 px-1 pt-1">
      <KPI label="Event Rate" value="{eps} EPS" trend="stable" trendValue={epsTrend} />
      <KPI label="Storage Index" value="98.2%" trend="stable" trendValue="optimal" variant="success" />
      <KPI label="Active Agents" value="41" trend="up" trendValue="+3" variant="critical" />
      <KPI label="Data Hygiene" value="99.9%" trend="stable" trendValue="verified" variant="success" />
    </div>

    <!-- Main Content Area -->
    <div class="flex-1 min-h-0 flex flex-col bg-surface-1 border border-border-primary rounded-md overflow-hidden shadow-sm">
      <Tabs tabs={siemTabs} bind:active={activeTab} />

      <div class="flex-1 overflow-hidden">
        {#if activeTab === 'feed'}
          <div class="h-full flex flex-col relative">
            {#if loading}
              <div class="absolute inset-0 bg-surface-1/50 backdrop-blur-xs z-10 flex items-center justify-center">
                <div class="flex items-center gap-3 px-4 py-2 bg-surface-2 border border-border-primary rounded-md shadow-lg animate-in fade-in zoom-in duration-300">
                  <span class="w-4 h-4 border-2 border-accent border-t-transparent rounded-full animate-spin"></span>
                  <span class="text-xs font-bold text-text-muted uppercase tracking-widest">Executing OQL Query...</span>
                </div>
              </div>
            {/if}
            <DataTable data={logs} columns={logColumns} striped compact>
              {#snippet render({ col, row, value })}
                {#if col.key === 'type'}
                  <Badge variant={value === 'error' ? 'critical' : value === 'warning' ? 'warning' : 'info'}>
                    {value}
                  </Badge>
                {:else if col.key === 'time'}
                  <span class="font-mono text-text-muted opacity-60 tabular-nums">{value}</span>
                {:else if col.key === 'host'}
                  <span class="font-bold text-accent cursor-pointer hover:underline">{value}</span>
                {:else if col.key === 'message'}
                  <span class="whitespace-pre-wrap font-mono text-[11px] {value.toLowerCase().includes('exploit') ? 'text-error font-bold' : 'text-text-secondary'}">
                    {value}
                  </span>
                {:else}
                  <span class="text-text-muted">{value}</span>
                {/if}
              {/snippet}
            </DataTable>
          </div>
        {:else if activeTab === 'analytics'}
          <div class="p-6 h-full space-y-6 overflow-y-auto bg-surface-0">
            <div class="grid grid-cols-2 gap-6 h-80">
              <div class="bg-surface-1 border border-border-secondary p-4 rounded-sm">
                <div class="text-[10px] font-bold text-text-muted uppercase mb-4">Traffic Volume (24h)</div>
                <div class="flex items-center justify-center h-full opacity-30 italic text-xs">Chart Engine Initializing...</div>
              </div>
              <div class="bg-surface-1 border border-border-secondary p-4 rounded-sm">
                <div class="text-[10px] font-bold text-text-muted uppercase mb-4">Top Offensive Endpoints</div>
                <div class="space-y-3">
                  {#each Array(5) as _, i}
                    <div class="flex items-center justify-between">
                      <span class="text-[11px] font-mono text-text-secondary">192.168.1.{100 + i}</span>
                      <div class="flex-1 mx-4 bg-surface-3 h-1.5 rounded-full overflow-hidden">
                        <div class="bg-error h-full" style="width: {90 - i * 15}%"></div>
                      </div>
                      <span class="text-[10px] font-bold text-text-muted">{500 - i * 80} hits</span>
                    </div>
                  {/each}
                </div>
              </div>
            </div>
          </div>
        {:else}
          <div class="flex flex-col items-center justify-center h-full p-12 text-center opacity-40 bg-surface-2/30">
            <span class="text-4xl mb-4">🚧</span>
            <div class="text-sm font-bold text-text-heading">Advanced Visualization Module</div>
            <div class="text-xs text-text-muted mt-1 max-w-sm">We are currently integrating the sovereign-grade mapping engine for real-time attribution.</div>
          </div>
        {/if}
      </div>

      <!-- Action Footer -->
      <div class="px-4 py-1.5 border-t border-border-primary bg-surface-2 flex items-center justify-between">
        <div class="flex gap-4">
          <div class="text-[9px] font-bold text-text-muted uppercase flex items-center gap-2">
            <span class="w-1.5 h-1.5 rounded-full bg-status-online"></span>
            Correlation: 🟢 Optimal
          </div>
          <div class="text-[9px] font-bold text-text-muted uppercase flex items-center gap-2">
            <span class="w-1.5 h-1.5 rounded-full bg-accent animate-pulse"></span>
            Queue: 128 MB
          </div>
        </div>
        <div class="text-[9px] font-mono text-text-muted">ID: node-alpha-01</div>
      </div>
    </div>
  </div>
</PageLayout>
