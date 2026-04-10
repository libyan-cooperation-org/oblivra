<!--
  OBLIVRA — Network Topology (Svelte 5)
  Interactive relationship graph of hosts, containers, and network flows.
-->
<script lang="ts">
  import { Chart, PageLayout, Button, Badge } from '@components/ui';
  import type { EChartsOption } from 'echarts';

  // Mock graph data
  const nodes = [
    { name: 'Edge-GW', category: 0, value: 30, symbolSize: 30 },
    { name: 'Web-Cluster', category: 1, value: 20, symbolSize: 20 },
    { name: 'DB-Master', category: 1, value: 25, symbolSize: 25 },
    { name: 'Staging-App', category: 2, value: 15, symbolSize: 15 },
    { name: 'VPN-Asian', category: 0, value: 10, symbolSize: 10 },
  ];

  const links = [
    { source: 'Edge-GW', target: 'Web-Cluster' },
    { source: 'Edge-GW', target: 'DB-Master' },
    { source: 'Web-Cluster', target: 'DB-Master' },
    { source: 'VPN-Asian', target: 'Edge-GW' },
    { source: 'Staging-App', target: 'DB-Master' },
  ];

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
      <KPI title="Total Nodes" value="48" trend="Mesh Active" />
      <KPI title="Active Edges" value="122" trend="Synced" />
      <KPI title="Cross-Zone Flow" value="3.4 GB/s" variant="accent" trend="+12%" />
      <KPI title="Network Health" value="Optimal" variant="success" trend="Stable" />
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
             <div class="w-2 h-1 bg-accent rounded-full"></div>
             <div class="w-1 h-1 bg-text-muted/20 rounded-full"></div>
             <div class="w-1 h-1 bg-text-muted/20 rounded-full"></div>
          </div>
        </div>
      </div>
    </div>
  </div>
</PageLayout>
