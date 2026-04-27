<!-- Global Topology — bound to agentStore for fleet count + GraphService for relations. -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { KPI, PageLayout, Button, PopOutButton } from '@components/ui';
  import { Globe, RefreshCw, ExternalLink } from 'lucide-svelte';
  import { agentStore } from '@lib/stores/agent.svelte';
  import { IS_BROWSER } from '@lib/context';
  import { push } from '@lib/router.svelte';

  let edges = $state<any[]>([]);

  async function refresh() {
    if (IS_BROWSER) return;
    try {
      const svc = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/graphservice');
      const fn = (svc as any).GetTopology ?? (svc as any).GetEdges ?? (svc as any).Snapshot;
      if (typeof fn === 'function') edges = ((await fn()) ?? []) as any[];
    } catch {}
  }
  onMount(() => { if (typeof agentStore.init === 'function') agentStore.init(); void refresh(); });
</script>

<PageLayout title="Global Topology" subtitle="Cross-fleet relationship graph">
  {#snippet toolbar()}
    <Button variant="secondary" size="sm" icon={RefreshCw} onclick={refresh}>Refresh</Button>
    <PopOutButton route="/global-topology" title="Topology" />
  {/snippet}
  <div class="flex flex-col h-full gap-4">
    <div class="grid grid-cols-1 md:grid-cols-3 gap-3">
      <KPI label="Nodes (agents)" value={agentStore.agents.length.toString()} variant="accent" />
      <KPI label="Edges" value={edges.length.toString()} variant="muted" />
      <KPI label="View" value="Graph" variant="muted" />
    </div>
    <div class="bg-surface-1 border border-border-primary rounded-md p-6 flex-1 flex flex-col items-center justify-center gap-3">
      <Globe size={36} class="text-accent opacity-30" />
      <div class="text-sm text-text-muted">
        Interactive graph rendering lives on the Threat Graph page.
      </div>
      <Button variant="cta" size="sm" onclick={() => push('/graph')}>Open Threat Graph<ExternalLink size={11} class="ml-1" /></Button>
    </div>
  </div>
</PageLayout>
