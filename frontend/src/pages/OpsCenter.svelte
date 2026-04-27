<!-- Ops Center — operator command-and-control hub. Mixes real stores. -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { PageLayout, KPI, Button, PopOutButton } from '@components/ui';
  import { Monitor, RefreshCw, Activity } from 'lucide-svelte';
  import { agentStore } from '@lib/stores/agent.svelte';
  import { alertStore } from '@lib/stores/alerts.svelte';
  import { diagnosticsStore } from '@lib/stores/diagnostics.svelte';
  import { push } from '@lib/router.svelte';

  onMount(() => {
    if (typeof agentStore.init === 'function') agentStore.init();
    if (typeof alertStore.init === 'function') alertStore.init();
    diagnosticsStore.init();
  });
</script>

<PageLayout title="Ops Center" subtitle="Command-and-control hub">
  {#snippet toolbar()}
    <Button variant="secondary" size="sm" icon={RefreshCw} onclick={() => { agentStore.init?.(); alertStore.init?.(); }}>Refresh</Button>
    <PopOutButton route="/ops" title="Ops Center" />
  {/snippet}
  <div class="flex flex-col h-full gap-4">
    <div class="grid grid-cols-1 md:grid-cols-4 gap-3">
      <KPI label="Fleet" value={agentStore.agents.length.toString()} variant="accent" />
      <KPI label="Active Alerts" value={alertStore.alerts.length.toString()} variant={alertStore.alerts.length > 0 ? 'warning' : 'muted'} />
      <KPI label="EPS" value={diagnosticsStore.eps.toString()} variant="muted" />
      <KPI label="Health" value={diagnosticsStore.healthGrade} variant={diagnosticsStore.healthGrade.startsWith('A') ? 'success' : 'warning'} />
    </div>
    <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-3">
      <Button variant="secondary" onclick={() => push('/dashboard')}><Monitor class="mr-1" size={12} />Dashboard</Button>
      <Button variant="secondary" onclick={() => push('/cases')}>Incidents</Button>
      <Button variant="secondary" onclick={() => push('/operator')}>Operator Mode</Button>
      <Button variant="secondary" onclick={() => push('/war-mode')}>War Mode</Button>
      <Button variant="secondary" onclick={() => push('/agents')}>Agent Console</Button>
      <Button variant="secondary" onclick={() => push('/fleet')}>Fleet</Button>
      <Button variant="secondary" onclick={() => push('/response')}>SOAR</Button>
      <Button variant="secondary" onclick={() => push('/ransomware')}>Ransomware Shield</Button>
    </div>
    <div class="flex-1 bg-surface-1 border border-border-primary rounded-md p-4">
      <div class="flex items-center gap-2 mb-3"><Activity size={14} class="text-accent" /><span class="text-[10px] uppercase tracking-widest font-bold">Live System Pulse</span></div>
      <div class="grid grid-cols-1 md:grid-cols-3 gap-3">
        <div class="bg-surface-2 rounded p-3"><div class="text-[10px] text-text-muted uppercase">Goroutines</div><div class="font-mono text-lg">{diagnosticsStore.snapshot?.runtime.goroutines ?? '—'}</div></div>
        <div class="bg-surface-2 rounded p-3"><div class="text-[10px] text-text-muted uppercase">Heap (MB)</div><div class="font-mono text-lg">{diagnosticsStore.snapshot?.runtime.heap_alloc_mb ?? '—'}</div></div>
        <div class="bg-surface-2 rounded p-3"><div class="text-[10px] text-text-muted uppercase">Query P99 (ms)</div><div class="font-mono text-lg">{diagnosticsStore.snapshot?.query.p99_query_ms ?? '—'}</div></div>
      </div>
    </div>
  </div>
</PageLayout>
