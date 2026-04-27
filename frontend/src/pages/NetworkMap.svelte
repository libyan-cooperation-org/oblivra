<!-- Network Map — agent geo + flows. Uses agentStore + NDRService.GetLiveTraffic. -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { PageLayout, KPI, Button, PopOutButton } from '@components/ui';
  import { Map as MapIcon, RefreshCw, ExternalLink } from 'lucide-svelte';
  import { agentStore } from '@lib/stores/agent.svelte';
  import { IS_BROWSER } from '@lib/context';
  import { push } from '@lib/router.svelte';

  let flows = $state<any[]>([]);

  async function refreshFlows() {
    if (IS_BROWSER) return;
    try {
      const { GetLiveTraffic } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/ndrservice');
      flows = ((await GetLiveTraffic()) ?? []) as any[];
    } catch {}
  }

  onMount(() => {
    if (typeof agentStore.init === 'function') agentStore.init();
    void refreshFlows();
  });

  let agents = $derived(agentStore.agents ?? []);
  let geoCount = $derived(new Set(agents.map((a: any) => a.country ?? a.region).filter(Boolean)).size);
</script>

<PageLayout title="Network Map" subtitle="Fleet geography and live flows">
  {#snippet toolbar()}
    <Button variant="secondary" size="sm" icon={RefreshCw} onclick={refreshFlows}>Refresh</Button>
    <PopOutButton route="/network-map" title="Network Map" />
  {/snippet}
  <div class="flex flex-col h-full gap-4">
    <div class="grid grid-cols-1 md:grid-cols-3 gap-3">
      <KPI label="Agents" value={agents.length.toString()} variant="accent" />
      <KPI label="Distinct Regions" value={geoCount.toString()} variant="muted" />
      <KPI label="Active Flows" value={flows.length.toString()} variant={flows.length > 0 ? 'warning' : 'muted'} />
    </div>
    <div class="bg-surface-1 border border-border-primary rounded-md p-6 flex-1 flex flex-col items-center justify-center gap-4">
      <MapIcon size={36} class="text-accent opacity-30" />
      <div class="text-sm text-text-muted text-center max-w-md">
        Geo-rendered topology lives on the dedicated map pages. Pick:
      </div>
      <div class="grid grid-cols-1 md:grid-cols-3 gap-2 w-full max-w-2xl">
        <Button variant="secondary" size="sm" onclick={() => push('/threat-map')}>Threat Map<ExternalLink size={9} class="ml-1" /></Button>
        <Button variant="secondary" size="sm" onclick={() => push('/fleet-map')}>Fleet Map<ExternalLink size={9} class="ml-1" /></Button>
        <Button variant="secondary" size="sm" onclick={() => push('/topology')}>Connection Graph<ExternalLink size={9} class="ml-1" /></Button>
      </div>
    </div>
  </div>
</PageLayout>
