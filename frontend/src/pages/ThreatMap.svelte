<!--
  OBLIVRA — Threat Map (Svelte 5)
  Geospatial attribution and attack origin visualization.
-->
<script lang="ts">
  import { KPI, PageLayout, Button, Badge, Spinner } from '@components/ui';
  import { onMount } from 'svelte';
  import { IS_BROWSER } from '@lib/context';

  let loading = $state(false);
  let threatStats = $state<any>({});

  const activity = [
    { country: 'CN', origin: 'Shenzhen', targets: 41, status: 'critical' },
    { country: 'RU', origin: 'Moscow', targets: 28, status: 'warning' },
    { country: 'KP', origin: 'Pyongyang', targets: 12, status: 'critical' },
    { country: 'US', origin: 'Ashburn', targets: 15, status: 'info' },
  ];

  async function loadThreatIntel() {
    if (IS_BROWSER) return;
    loading = true;
    try {
        const { GetThreatIntelStats } = await import('@wailsjs/go/services/SIEMService.js');
        threatStats = await GetThreatIntelStats();
    } catch (err) {
        console.error('Threat intel load failed', err);
    } finally {
        loading = false;
    }
  }

  onMount(() => {
    loadThreatIntel();
  });
</script>

<PageLayout title="Boundaries & Origins" subtitle="Geospatial attribution engine">
  {#snippet toolbar()}
    <div class="flex items-center gap-2">
      <Button variant="secondary" size="sm">Focus: Edge Nodes</Button>
      <Badge variant="critical" class="animate-pulse">12 SIGHTINGS</Badge>
    </div>
  {/snippet}

  <div class="grid grid-cols-1 lg:grid-cols-4 gap-6 h-full">
    <!-- Map Canvas (Placeholder) -->
    <div class="lg:col-span-3 bg-surface-1 border border-border-primary rounded-md relative overflow-hidden group">
      <div class="absolute inset-0 flex items-center justify-center">
        <div class="text-center opacity-20 pointer-events-none">
          <div class="text-6xl mb-4">🗺️</div>
          <div class="text-sm font-bold uppercase tracking-widest">Sovereign Mapping Engine</div>
          <div class="text-[10px] mt-1 italic">Initializing Vector Tiles...</div>
        </div>
      </div>

      <!-- Mock Map Overlays -->
      <div class="absolute top-4 left-4 flex flex-col gap-2">
        <div class="bg-black/60 backdrop-blur-md border border-white/10 p-2 rounded-sm space-y-1">
          <div class="text-[8px] font-bold text-white/40 uppercase">Filter</div>
          <div class="flex gap-2">
            <Badge variant="critical" size="xs">Exploits</Badge>
            <Badge variant="warning" size="xs">Scans</Badge>
          </div>
        </div>
      </div>
      
      <div class="absolute bottom-4 right-4 flex flex-col gap-2">
         <div class="bg-black/60 backdrop-blur-md border border-white/10 p-3 rounded-sm">
            <div class="text-[10px] font-bold text-success mb-2 font-mono">● LIVE ATTACK STREAM</div>
            <div class="space-y-1">
               {#each Array(3) as _, i}
                 <div class="text-[9px] font-mono text-white/60">
                    <span class="text-error">[{new Date().toLocaleTimeString()}]</span>
                    {activity[i].origin} → PROD-CLUSTER-{i+1}
                 </div>
               {/each}
            </div>
         </div>
      </div>
    </div>

    <!-- Sidebar Info -->
    <div class="flex flex-col gap-6">
      <div class="bg-surface-1 border border-border-primary rounded-md p-4 space-y-4">
        <div class="text-xs font-bold text-text-heading border-b border-border-primary pb-2 uppercase tracking-tight">Origin Analysis</div>
        <div class="space-y-4">
          {#each activity as item}
            <div class="flex flex-col gap-1">
              <div class="flex justify-between items-center text-[11px]">
                <span class="font-bold text-text-secondary">{item.origin}, {item.country}</span>
                <Badge variant={item.status === 'critical' ? 'critical' : item.status === 'warning' ? 'warning' : 'info'} size="xs">{item.targets} TGT</Badge>
              </div>
              <div class="w-full bg-surface-3 h-1 rounded-full overflow-hidden">
                <div class="bg-accent h-full" style="width: {item.targets * 2}%"></div>
              </div>
            </div>
          {/each}
        </div>
      </div>

      <div class="flex-1 bg-surface-1 border border-border-primary rounded-md p-4 flex flex-col gap-3 relative">
        {#if loading}
          <div class="absolute inset-0 bg-surface-1/40 backdrop-blur-xs z-10 flex items-center justify-center rounded-md">
              <Spinner />
          </div>
        {/if}
        <div class="text-xs font-bold text-text-heading uppercase tracking-tight">Detection Summary</div>
        <div class="flex-1 flex flex-col justify-center gap-4">
          <KPI label="Unique Sources" value={threatStats?.TotalUniqueSources || 0} size="sm" />
          <KPI label="Peak Intensity" value="{threatStats?.PeakIntensity || 0} PPS" trend="up" size="sm" variant="critical" />
          <KPI label="Attribution Conf." value="{threatStats?.AttributionConfidence || 94}%" trend="stable" size="sm" variant="success" />
        </div>
      </div>
    </div>
  </div>
</PageLayout>
