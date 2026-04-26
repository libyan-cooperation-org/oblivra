<!--
  OBLIVRA — SIEM Panel (Svelte 5)
  Central hub for security event monitoring and log aggregation.
-->
<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { appStore } from '@lib/stores/app.svelte';
  import { IS_BROWSER } from '@lib/context';
  import { subscribe, emitLocal } from '@lib/bridge';
  import { apiFetch } from '@lib/apiClient';
  import { KPI, Badge, DataTable, PageLayout, Button, Input, Tabs } from '@components/ui';

  const MAX_LIVE_EVENTS = 500;

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

  let activeTab = $state('feed');
  let searchQuery = $state('');
  let logs = $state<any[]>([]);
  let loading = $state(false);
  let paused = $state(false);
  let liveCount = $state(0);
  let unsubStream: (() => void) | null = null;

  function formatEvent(ev: any) {
    return {
      id: ev.ID || ev.id || crypto.randomUUID(),
      time: new Date(ev.Timestamp || ev.timestamp || Date.now()).toLocaleString(),
      type: ev.EventType || ev.event_type || ev.Severity || ev.severity || 'info',
      host: ev.HostID || ev.host_id || ev.HostName || 'unknown',
      message: ev.RawLog || ev.raw_log || ev.Message || 'No message',
    };
  }

  async function refreshLogs() {
    loading = true;
    try {
      const query = searchQuery || 'severity:>=info';
      let result;

      if (IS_BROWSER) {
        const res = await apiFetch('/api/v1/siem/search?q=' + encodeURIComponent(query));
        if (!res.ok) throw new Error('API error: ' + res.status);
        result = await res.json();
        if (result.events) {
          result.Events = result.events;
        }
      } else {
        const { ExecuteOQL } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/siemservice');
        result = await ExecuteOQL(query);
      }

      if (result && result.Events) {
        logs = result.Events.map(formatEvent);
      } else {
        logs = [];
      }
    } catch (err) {
      console.error('[SIEM] Query failed:', err);
      appStore.notify('Failed to execute OQL query', 'error', (err as Error).message);
    } finally {
      loading = false;
    }
  }

  onMount(() => {
    refreshLogs();

    // Subscribe to real-time SIEM event stream from the backend
    console.log('[SIEM] Subscribing to siem-stream...');
    unsubStream = subscribe('siem-stream', (evt: any) => {
      console.log('[SIEM] Component received event:', evt);
      if (paused) return;
      liveCount++;
      const entry = formatEvent(evt);
      logs = [entry, ...logs].slice(0, MAX_LIVE_EVENTS);
    });
  });

  function emitTestEvent() {
    console.log('[SIEM] Emitting test event locally');
    emitLocal('siem-stream', {
      timestamp: new Date().toISOString(),
      event_type: 'TEST_HEARTBEAT',
      host: 'localhost',
      source_ip: '127.0.0.1',
      user: 'test-user',
      metadata: { action: 'verification' }
    });
  }

  onDestroy(() => {
    unsubStream?.();
  });

  let eps = $state(0);
  $effect(() => {
    // Measure real EPS from the live stream
    const startCount = liveCount;
    const interval = setInterval(() => {
      const delta = liveCount - startCount;
      eps = Math.round(delta / 2);
    }, 2000);
    return () => clearInterval(interval);
  });
</script>

<PageLayout title="SIEM Console" subtitle="Unified event orchestration and threat detection">
  {#snippet toolbar()}
    <div class="flex items-center gap-3">
      <button 
        onclick={emitTestEvent}
        class="flex items-center gap-2 rounded border border-border-primary px-3 py-1.5 text-[10px] font-medium transition-all hover:bg-surface-3"
      >
        Emit Test Event
      </button>
      <Input variant="search" placeholder="Search events..." bind:value={searchQuery}
        onkeydown={(e: KeyboardEvent) => e.key === 'Enter' && refreshLogs()} />
      <Button variant="secondary" size="sm" onclick={refreshLogs}>Refresh</Button>
      <Button variant="cta" size="sm" onclick={() => { paused = !paused; appStore.notify(paused ? 'Feed Paused' : 'Feed Resumed', paused ? 'warning' : 'info'); }}>{paused ? 'Resume Feed' : 'Pause Feed'}</Button>
    </div>
  {/snippet}

  <div class="flex flex-col h-full gap-5">
    <div class="grid grid-cols-1 md:grid-cols-4 gap-4 shrink-0">
      <KPI label="Event Rate" value={eps + " EPS"} trend={eps > 0 ? 'up' : 'stable'} trendValue="live" />
      <KPI label="Storage Index" value="0%" trend="stable" trendValue="PENDING" variant="default" />
      <KPI label="Live Events" value={liveCount} trend={liveCount > 0 ? 'up' : 'stable'} trendValue="ingested" variant="critical" />
      <KPI label="Data Hygiene" value="0%" trend="stable" trendValue="PENDING" variant="default" />
    </div>

    <div class="flex-1 min-h-0 flex flex-col bg-surface-1 border border-border-primary rounded-md overflow-hidden shadow-card">
      <Tabs tabs={siemTabs} bind:active={activeTab} />

      <div class="flex-1 overflow-hidden">
        {#if activeTab === 'feed'}
          <div class="h-full flex flex-col relative">
            {#if loading}
              <div class="absolute inset-0 bg-surface-1/50 z-10 flex items-center justify-center">
                <div class="flex items-center gap-3 px-4 py-2 bg-surface-2 border border-border-primary rounded-md shadow-lg">
                  <span class="w-4 h-4 border-2 border-accent border-t-transparent rounded-full animate-spin"></span>
                  <span class="text-xs font-bold text-text-muted uppercase tracking-widest">Executing OQL Query...</span>
                </div>
              </div>
            {/if}
            <DataTable data={logs} columns={logColumns} striped compact>
              {#snippet render({ col, value })}
                {#if col.key === 'type'}
                  <Badge variant={value === 'error' || value === 'critical' ? 'critical' : value === 'warning' ? 'warning' : 'info'}>
                    {value}
                  </Badge>
                {:else if col.key === 'time'}
                  <span class="font-mono text-text-muted opacity-60 tabular-nums">{value}</span>
                {:else if col.key === 'host'}
                  <span class="font-bold text-accent cursor-pointer hover:underline">{value}</span>
                {:else if col.key === 'message'}
                  <span class="whitespace-pre-wrap font-mono text-[11px] {String(value).toLowerCase().includes('exploit') ? 'text-error font-bold' : 'text-text-secondary'}">
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
                <div class="space-y-3 opacity-20 italic text-[10px] text-center pt-8">
                  No endpoint data synchronized
                </div>
              </div>
            </div>
          </div>
        {:else}
          <div class="flex flex-col items-center justify-center h-full p-12 text-center opacity-40 bg-surface-2/30">
            <span class="text-4xl mb-4">🚧</span>
            <div class="text-sm font-bold text-text-heading">Advanced Visualization Module</div>
            <div class="text-xs text-text-muted mt-1 max-w-sm">Integrating the sovereign-grade mapping engine for real-time attribution.</div>
          </div>
        {/if}
      </div>

      <div class="px-4 py-1.5 border-t border-border-primary bg-surface-2 flex items-center justify-between">
        <div class="flex gap-4">
          <div class="text-[9px] font-bold text-text-muted uppercase flex items-center gap-2">
            <span class="w-1.5 h-1.5 rounded-full bg-status-online"></span>
            Correlation: OPTIMAL
          </div>
          <div class="text-[9px] font-bold text-text-muted uppercase flex items-center gap-2">
            <span class="w-1.5 h-1.5 rounded-full bg-accent animate-pulse"></span>
            Queue: 128 MB
          </div>
        </div>
        <div class="text-[9px] font-mono text-text-muted">node-alpha-01</div>
      </div>
    </div>
  </div>
</PageLayout>
