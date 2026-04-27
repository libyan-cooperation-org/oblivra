<!-- Fleet Map — fleet geographic distribution. Uses agentStore. -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { PageLayout, KPI, Button, PopOutButton, Badge } from '@components/ui';
  import { Map as MapIcon, RefreshCw, ExternalLink, Server } from 'lucide-svelte';
  import { agentStore } from '@lib/stores/agent.svelte';
  import { push } from '@lib/router.svelte';

  onMount(() => { if (typeof agentStore.init === 'function') agentStore.init(); });

  let agents = $derived(agentStore.agents ?? []);
  let regions = $derived.by(() => {
    const groups: Record<string, any[]> = {};
    for (const a of agents) {
      const k = (a as any).country ?? (a as any).region ?? (a as any).datacenter ?? 'Unknown';
      (groups[k] ??= []).push(a);
    }
    return Object.entries(groups).sort((a, b) => b[1].length - a[1].length);
  });
</script>

<PageLayout title="Fleet Map" subtitle="Geographic distribution of managed agents">
  {#snippet toolbar()}
    <Button variant="secondary" size="sm" icon={RefreshCw} onclick={() => agentStore.init?.()}>Refresh</Button>
    <PopOutButton route="/fleet-map" title="Fleet Map" />
  {/snippet}
  <div class="flex flex-col h-full gap-4">
    <div class="grid grid-cols-1 md:grid-cols-3 gap-3">
      <KPI label="Total Agents" value={agents.length.toString()} variant="accent" />
      <KPI label="Distinct Regions" value={regions.length.toString()} variant="muted" />
      <KPI label="Online" value={agents.filter((a) => a.status === 'online' || a.status === 'active').length.toString()} variant="success" />
    </div>
    <div class="grid grid-cols-1 md:grid-cols-2 gap-3 flex-1 min-h-0">
      <div class="bg-surface-1 border border-border-primary rounded-md flex flex-col">
        <div class="flex items-center gap-2 p-3 border-b border-border-primary">
          <MapIcon size={14} class="text-accent" />
          <span class="text-[10px] uppercase tracking-widest font-bold">Regional Distribution</span>
        </div>
        <div class="flex-1 overflow-auto">
          {#each regions as [name, list]}
            <div class="flex items-center gap-3 px-3 py-2 border-b border-border-primary">
              <Server size={11} class="text-text-muted" />
              <span class="text-[11px] flex-1 truncate">{name}</span>
              <Badge variant="info" size="xs">{list.length}</Badge>
            </div>
          {:else}
            <div class="p-8 text-center text-sm text-text-muted">No regions yet.</div>
          {/each}
        </div>
      </div>
      <div class="bg-surface-1 border border-border-primary rounded-md p-6 flex flex-col items-center justify-center gap-3">
        <MapIcon size={32} class="text-accent opacity-30" />
        <div class="text-sm text-text-muted">For an interactive 3D globe, use the dedicated Threat Map.</div>
        <Button variant="cta" size="sm" onclick={() => push('/threat-map')}>Open Threat Map<ExternalLink size={11} class="ml-1" /></Button>
      </div>
    </div>
  </div>
</PageLayout>
