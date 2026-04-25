<!--
  OBLIVRA — Network Topology (Svelte 5)
  Interactive relationship graph of hosts, containers, and network flows.
-->
<script lang="ts">
  import { Chart, PageLayout, Button, Badge, KPI } from '@components/ui';
  import type { EChartsOption } from 'echarts';

  // Network topology state - will be populated by live telemetry
  let nodes = $state<any[]>([]);
  let links = $state<any[]>([]);
  const categories = [{ name: 'Gateway' }, { name: 'Production' }, { name: 'Staging' }];

  const chartOption: EChartsOption = {
    backgroundColor: 'transparent',
    tooltip: {},
    legend: {
      data: categories.map(c => c.name),
      textStyle: { color: '#a9b1d6', fontSize: 10 },
      bottom: '2%'
    },
    series: [
      {
        name: 'Network Topology',
        type: 'graph',
        layout: 'force',
        data: nodes,
        links: links,
        categories: categories,
        roam: true,
        label: {
          show: true,
          position: 'right',
          color: '#a9b1d6',
          fontSize: 10
        },
        force: {
          repulsion: 1000,
          edgeLength: 150
        },
        lineStyle: {
          color: '#33467c',
          curveness: 0.1
        },
        itemStyle: {
          borderColor: '#1a1b26',
          borderWidth: 2
        }
      }
    ]
  };
</script>

<PageLayout title="Network Topology" subtitle="Visualizing real-time relational telemetry and data flow">
  {#snippet toolbar()}
    <Button variant="secondary" size="sm">Freeze Layout</Button>
    <Button variant="ghost" size="sm" icon="📡">Sniff Flows</Button>
  {/snippet}

  <div class="flex flex-col h-full gap-5">
    <div class="grid grid-cols-1 md:grid-cols-4 gap-4 shrink-0">
      <KPI label="Total Nodes" value="0" trend="stable" trendValue="0 Active" />
      <KPI label="Active Edges" value="0" trend="stable" trendValue="Synced" />
      <KPI label="Cross-Zone Flow" value="0 B/s" variant="accent" trend="stable" trendValue="Nominal" />
      <KPI label="Network Health" value="Optimal" variant="success" trend="stable" trendValue="Stable" />
    </div>

    <div class="flex-1 min-h-[500px] bg-surface-1 border border-border-primary rounded-md p-4 relative overflow-hidden shadow-premium">
      <!-- Grid Background -->
      <div class="absolute inset-0 opacity-[0.03] pointer-events-none grayscale" style="background-image: linear-gradient(#7aa2f7 1px, transparent 1px), linear-gradient(90deg, #7aa2f7 1px, transparent 1px); background-size: 40px 40px;"></div>
      
      <div class="h-full w-full relative z-10">
        <Chart option={chartOption} />
      </div>

      <!-- Overlay Map controls -->
      <div class="absolute top-4 left-4 flex flex-col gap-2 z-20">
        <div class="bg-surface-2 border border-border-secondary p-2 rounded shadow-xl flex flex-col gap-1">
          <Badge variant="success" dot>LIVE MAPPING</Badge>
          <div class="text-[9px] text-text-muted uppercase font-bold tracking-widest mt-1">Force Simulation</div>
          <div class="flex gap-1 mt-1">
             <div class="w-1 h-1 bg-text-muted/20 rounded-full"></div>
             <div class="w-1 h-1 bg-text-muted/20 rounded-full"></div>
             <div class="w-1 h-1 bg-text-muted/20 rounded-full"></div>
          </div>
        </div>
      </div>
    </div>
  </div>
</PageLayout>
