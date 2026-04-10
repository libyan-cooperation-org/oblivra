<!--
  OBLIVRA — Network Map (Svelte 5)
  Geospatial network visualization: Mapping platform presence across global zones.
-->
<script lang="ts">
  import { KPI, PageLayout, Button, Chart } from '@components/ui';
  import type { EChartsOption } from 'echarts';

  const chartOption: EChartsOption = {
    backgroundColor: 'transparent',
    tooltip: { trigger: 'item', formatter: '{b}' },
    geo: {
      map: 'world',
      roam: true,
      label: { emphasis: { show: false } },
      itemStyle: { areaColor: '#1a1b26', borderColor: '#33467c' },
      emphasis: { itemStyle: { areaColor: '#24283b' } },
    },
    series: [
      {
        type: 'effectScatter',
        coordinateSystem: 'geo',
        data: [
          { name: 'US-East-1', value: [-74.006, 40.7128, 100] },
          { name: 'EU-West-1', value: [-0.1278, 51.5074, 100] },
          { name: 'AP-South-1', value: [103.8198, 1.3521, 100] },
        ],
        symbolSize: 8,
        showEffectOn: 'render',
        rippleEffect: { brushType: 'stroke' },
        label: { formatter: '{b}', position: 'right', show: false },
        itemStyle: { color: '#7aa2f7', shadowBlur: 10, shadowColor: '#7aa2f7' },
        zlevel: 1,
      },
    ],
  };
</script>

<PageLayout title="Global Geospatial Map" subtitle="Mapping real-time platform presence and cross-border data flows">
  {#snippet toolbar()}
     <Button variant="secondary" size="sm">Animate Flows</Button>
     <Button variant="primary" size="sm">Zone Re-allocate</Button>
  {/snippet}

  <div class="flex flex-col h-full gap-5">
    <div class="grid grid-cols-1 md:grid-cols-4 gap-4 shrink-0">
      <KPI label="Active Zones" value="14" trend="stable" trendValue="Global" />
      <KPI label="Ingress Points" value="412" trend="stable" trendValue="Nominal" variant="success" />
      <KPI label="Mean Latency" value="112ms" trend="stable" trendValue="Optimal" variant="success" />
      <KPI label="Encryption Level" value="v3.1" trend="stable" trendValue="Hardened" variant="accent" />
    </div>

    <div class="flex-1 bg-surface-1 border border-border-primary rounded-md p-4 relative overflow-hidden shadow-card">
       <div class="absolute inset-0 opacity-[0.02] pointer-events-none"
         style="background-image: linear-gradient(#7aa2f7 1px, transparent 1px), linear-gradient(90deg, #7aa2f7 1px, transparent 1px); background-size: 60px 60px;">
       </div>
       
       <div class="h-full w-full relative z-10">
          <Chart option={chartOption} />
       </div>

       <div class="absolute bottom-6 right-6 flex flex-col gap-2 z-20">
          <div class="bg-surface-2/80 border border-border-secondary p-4 rounded-md shadow-lg flex flex-col gap-2 min-w-[200px]">
             <div class="text-[10px] font-bold text-text-muted uppercase tracking-widest border-b border-border-primary pb-2">Mesh Density</div>
             <div class="flex justify-between items-center text-[11px] text-text-secondary">
                <span>North America</span>
                <span class="font-bold text-accent">High</span>
             </div>
             <div class="flex justify-between items-center text-[11px] text-text-secondary">
                <span>European Core</span>
                <span class="font-bold text-accent">Critical</span>
             </div>
             <div class="flex justify-between items-center text-[11px] text-text-secondary">
                <span>Asia Pacific</span>
                <span class="font-bold text-text-primary">Moderate</span>
             </div>
          </div>
       </div>
    </div>
  </div>
</PageLayout>
