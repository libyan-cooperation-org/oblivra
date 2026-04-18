<!--
  OBLIVRA — SIEM Search (Svelte 5)
  Deep investigation via Oblivra Query Language (OQL).
-->
<script lang="ts">
  import { onMount } from 'svelte';
  import { PageLayout, Button, DataTable, Badge } from '@components/ui';
  import { IS_BROWSER } from '@lib/context';

  let query = $state('*');
  let results = $state<any[]>([]);
  let searching = $state(false);
  let federationStatus = $state<any>({ active: false, peer_count: 0 });

  onMount(async () => {
    if (!IS_BROWSER) {
      try {
        const { GetFederationStatus } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/siemservice.js');
        federationStatus = await GetFederationStatus();
      } catch (err) {
        console.warn("Failed to fetch federation status:", err);
      }
    }
  });

  function formatEvent(ev: any) {
    return {
      id: ev.id || ev.ID || crypto.randomUUID(),
      timestamp: new Date(ev.timestamp || ev.Timestamp || Date.now()).toLocaleString(),
      host: ev.host_id || ev.HostID || 'unknown',
      message: ev.raw_log || ev.RawLog || 'No log data',
      risk: ev.risk_score || 0,
      severity: ev.event_type || 'info'
    };
  }

  async function executeSearch() {
    searching = true;
    try {
      let events = [];
      if (IS_BROWSER) {
        const res = await fetch('/api/v1/siem/search?q=' + encodeURIComponent(query), {
          headers: { 'Authorization': 'Bearer oblivra-dev-key' }
        });
        if (!res.ok) throw new Error('API error: ' + res.status);
        const data = await res.json();
        events = data.events || [];
      } else {
        const { SearchHostEvents } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/siemservice');
        events = await SearchHostEvents(query, 100);
      }
      results = (events || []).map(formatEvent);
    } catch (err) {
      console.error('[SIEMSearch] Search failed:', err);
    } finally {
      searching = false;
    }
  }

  const columns = [
    { key: 'timestamp', label: 'Time', width: '180px' },
    { key: 'host', label: 'Node', width: '120px' },
    { key: 'message', label: 'Message Details' },
    { key: 'risk', label: 'Risk', width: '100px' },
  ];
</script>

<PageLayout title="Threat Hunt & Search" subtitle="Query the OBLIVRA hyper-graph using OQL syntax">
  {#snippet toolbar()}
    <Button variant="secondary" size="sm">History</Button>
    <Button variant="ghost" size="sm" icon="📖">Syntax Help</Button>
  {/snippet}

  <div class="flex flex-col h-full gap-5">
    <!-- Query Editor Area -->
    <div class="bg-surface-1 border border-border-primary rounded-md p-4 flex flex-col gap-3">
      <div class="flex items-center gap-2 mb-1">
        <span class="text-[9px] font-bold text-accent uppercase tracking-widest">OQL EDITOR</span>
        <div class="h-px flex-1 bg-border-primary opacity-30"></div>
      </div>
      <div class="relative">
        <textarea
          bind:value={query}
          class="w-full h-24 bg-surface-2 border border-border-secondary rounded px-3 py-2 font-mono text-sm text-text-primary focus:outline-none focus:border-accent transition-colors resize-none"
          placeholder="e.g. host='*' | top 10 message"
        ></textarea>
        <div class="absolute bottom-2 right-2 flex gap-2">
           <Button variant="cta" size="sm" onclick={executeSearch} loading={searching}>Execute Search</Button>
        </div>
      </div>
    </div>

    <!-- Results Area -->
    <div class="flex-1 min-h-0 flex flex-col bg-surface-1 border border-border-primary rounded-md overflow-hidden">
      {#if results.length > 0}
        <div class="flex items-center justify-between px-4 py-2 border-b border-border-primary bg-surface-2/30">
          <div class="flex items-center gap-3">
             <span class="text-[10px] font-bold text-text-muted">{results.length} events found in 0.4s</span>
             {#if federationStatus.active}
               <div class="h-3 w-px bg-border-primary opacity-30"></div>
               <Badge variant="success" size="xs" dot>Distributed: {federationStatus.peer_count + 1} shards active</Badge>
             {/if}
          </div>
          <div class="flex gap-2">
            <Button variant="ghost" size="sm">Export JSON</Button>
            <Button variant="ghost" size="sm">Save as Alert</Button>
          </div>
        </div>
        <div class="flex-1 overflow-hidden">
          <DataTable data={results} {columns} compact striped>
            {#snippet render({ value, col })}
              {#if col.key === 'risk'}
                <div class="flex items-center gap-2">
                   <div class="w-2 h-2 rounded-full {value > 80 ? 'bg-error' : value > 50 ? 'bg-warning' : 'bg-success'}"></div>
                   <span class="font-mono text-[10px] tabular-nums">{value}</span>
                </div>
              {:else if col.key === 'timestamp'}
                <span class="text-[10px] font-mono text-text-muted opacity-60 tabular-nums">{value}</span>
              {:else if col.key === 'message'}
                <span class="font-mono text-[11px] {value.toLowerCase().includes('failed') ? 'text-error' : ''}">
                  {value}
                </span>
              {:else}
                {value}
              {/if}
            {/snippet}
          </DataTable>
        </div>
      {:else if searching}
        <div class="flex-1 flex items-center justify-center italic text-text-muted text-sm gap-2">
           <span class="animate-spin text-accent">⏳</span> Scanning indices...
        </div>
      {:else}
        <div class="flex-1 flex flex-col items-center justify-center opacity-40 text-center p-12">
          <span class="text-4xl mb-4">🔎</span>
          <h3 class="text-sm font-bold text-text-heading">No active investigation</h3>
          <p class="text-xs text-text-muted mt-1 max-w-xs">Use the OQL editor above to query logs, events, and telemetry from your fleet.</p>
        </div>
      {/if}
    </div>
  </div>
</PageLayout>
