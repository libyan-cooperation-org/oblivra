<!--
  OBLIVRA — SIEM Search (Svelte 5)
  Deep investigation via Oblivra Query Language (OQL).
-->
<script lang="ts">
  import { evaluateOQL } from '@lib/oql';
  import { PageLayout, Button, DataTable } from '@components/ui';

  const mockData = [
    { id: 'ev_1', timestamp: '2026-04-10 01:40:01', host: 'prod-web-01', message: 'Failed SSH login attempt from 14.2.1.4', severity: 'high', risk: 82 },
    { id: 'ev_2', timestamp: '2026-04-10 01:38:22', host: 'prod-web-02', message: 'Unauthorized /etc/passwd read', severity: 'critical', risk: 95 },
    { id: 'ev_3', timestamp: '2026-04-10 01:35:10', host: 'prod-web-03', message: 'Outbound connection to TOR entry node', severity: 'medium', risk: 74 },
    { id: 'ev_4', timestamp: '2026-04-10 01:30:00', host: 'prod-web-01', message: 'Nginx process restarted by root', severity: 'low', risk: 40 },
    { id: 'ev_5', timestamp: '2026-04-09 23:55:00', host: 'staging-api-01', message: 'Postgres leak detected', severity: 'high', risk: 88 },
  ];

  let query = $state('host="prod-web-*" AND severity="high"');
  let results = $state<any[]>([]);
  let searching = $state(false);

  function executeSearch() {
    searching = true;
    // Mock latency
    setTimeout(() => {
      // Normalize query to something the simple parser understands for the demo
      const simplifiedQuery = query.replace('host="prod-web-*"', 'host~"prod-web"');
      results = evaluateOQL(simplifiedQuery, mockData);
      searching = false;
    }, 400);
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
          <span class="text-[10px] font-bold text-text-muted">{results.length} events found in 0.4s</span>
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
