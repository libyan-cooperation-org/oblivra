<!--
  OBLIVRA — Threat Graph (Svelte 5)
  Adversary relationship mapping: Visualizing lateral movement and kill-chain progression.
-->
<script lang="ts">
  import { KPI, PageLayout, Badge, Button, Chart } from '@components/ui';
  import type { EChartsOption } from 'echarts';
  import { Sword, Shield, Target, Activity, Zap } from 'lucide-svelte';

  const nodes = [
    { name: 'Initial Access', category: 0, value: 30, symbolSize: 30 },
    { name: 'Persistence', category: 1, value: 20, symbolSize: 20 },
    { name: 'Lateral Movement', category: 1, value: 25, symbolSize: 25 },
    { name: 'Exfiltration', category: 2, value: 15, symbolSize: 15 },
    { name: 'C2 Beacon', category: 0, value: 10, symbolSize: 10 },
  ];

  const links = [
    { source: 'Initial Access', target: 'Persistence' },
    { source: 'Persistence', target: 'Lateral Movement' },
    { source: 'Lateral Movement', target: 'Exfiltration' },
    { source: 'C2 Beacon', target: 'Initial Access' },
  ];

  const categories = [{ name: 'Infection' }, { name: 'Progression' }, { name: 'Objective' }];

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
        name: 'Threat Kill-Chain',
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
          repulsion: 1500,
          edgeLength: 120
        },
        lineStyle: {
          color: '#f7768e',
          curveness: 0.2,
          width: 2
        },
        itemStyle: {
          borderColor: '#1a1b26',
          borderWidth: 2
        }
      }
    ]
  };
</script>

<PageLayout title="Threat Relationship Graph" subtitle="Visualizing lateral movement patterns and kill-chain progression logic">
  {#snippet toolbar()}
     <Button variant="secondary" size="sm">Logic Re-trace</Button>
     <Button variant="error" size="sm" icon="⚔️">Simulate Counter-Action</Button>
  {/snippet}

  <div class="flex flex-col h-full gap-5">
    <div class="grid grid-cols-1 md:grid-cols-4 gap-4 shrink-0">
      <KPI title="Detected Chains" value="3" trend="Active" variant="error" />
      <KPI title="Mean Progression" value="48%" trend="Lateral" variant="warning" />
      <KPI title="Observed Tactics" value="12" trend="MITRE Sync" />
      <KPI title="Containment Readiness" value="High" trend="SOAR Ready" variant="success" />
    </div>

    <div class="flex-1 bg-surface-1 border border-border-primary rounded-md p-4 relative overflow-hidden shadow-premium">
       <!-- Grid Background -->
       <div class="absolute inset-0 opacity-[0.03] pointer-events-none grayscale" style="background-image: linear-gradient(#f7768e 1px, transparent 1px), linear-gradient(90deg, #f7768e 1px, transparent 1px); background-size: 40px 40px;"></div>
       
       <div class="h-full w-full relative z-10">
          <Chart option={chartOption} />
       </div>

       <!-- Threat Level Overlay -->
       <div class="absolute top-6 left-6 flex flex-col gap-2 z-20">
          <div class="bg-surface-2/90 border border-error/30 p-3 rounded-md shadow-2xl flex flex-col gap-1 min-w-[150px]">
             <Badge variant="error" dot>HIGH ADVERSARY GRAVITY</Badge>
             <div class="text-[9px] text-text-muted uppercase font-bold tracking-widest mt-1">Kill-Chain State</div>
             <div class="flex items-center gap-2 mt-1">
                <div class="flex-1 flex gap-0.5">
                   <div class="h-1.5 w-4 bg-error rounded-full"></div>
                   <div class="h-1.5 w-4 bg-error rounded-full"></div>
                   <div class="h-1.5 w-4 bg-surface-3 rounded-full"></div>
                </div>
                <span class="text-[10px] font-bold text-error">Stage 2</span>
             </div>
          </div>
       </div>
    </div>
  </div>
</PageLayout>
