<!--
  OBLIVRA — Global Fleet Map (Svelte 5)
  Geographic distribution of managed nodes and real-time threat telemetry.
-->
<script lang="ts">
  import { Chart, PageLayout, KPI, DataTable, Badge, Button } from '@components/ui';
  import type { EChartsOption } from 'echarts';

  // Mock fleet locations [lon, lat, count, risk]
  const locations = [
    { name: 'London Data Center', value: [-0.1278, 51.5074, 12, 5], status: 'active' },
    { name: 'New York Edge', value: [-74.0060, 40.7128, 8, 82], status: 'alert' },
    { name: 'Tokyo Node', value: [139.6917, 35.6895, 14, 10], status: 'active' },
    { name: 'Singapore Hub', value: [103.8198, 1.3521, 6, 45], status: 'active' },
    { name: 'Berlin Stack', value: [13.4050, 52.5200, 9, 2], status: 'active' },
    { name: 'Sydney Relay', value: [151.2093, -33.8688, 4, 15], status: 'active' },
  ];

  const chartOption: EChartsOption = {
    backgroundColor: 'transparent',
    tooltip: {
      trigger: 'item',
      backgroundColor: '#1a1b26',
      borderColor: '#33467c',
      textStyle: { color: '#a9b1d6', fontSize: 11 },
      formatter: (params: any) => {
        return `<b>${params.name}</b><br/>Nodes: ${params.value[2]}<br/>Risk Score: ${params.value[3]}`;
      }
    },
    geo: {
      map: 'world', // We'll hope it degrades gracefully or use scatter
      roam: true,
      silent: true,
      itemStyle: {
        areaColor: '#1a1b26',
        borderColor: '#33467c'
      }
    },
    series: [
      {
        name: 'Fleet Distribution',
        type: 'scatter',
        coordinateSystem: 'geo',
        data: locations,
        symbolSize: (val: any) => val[2] * 2,
        itemStyle: {
          color: (params: any) => params.value[3] > 80 ? '#f7768e' : params.value[3] > 40 ? '#ff9e64' : '#7aa2f7',
          shadowBlur: 10,
          shadowColor: 'rgba(0, 0, 0, 0.5)'
        }
      },
      {
        name: 'Top Risks',
        type: 'effectScatter',
        coordinateSystem: 'geo',
        data: locations.filter(l => l.value[3] > 80),
        symbolSize: (val: any) => val[2] * 3,
        showEffectOn: 'render',
        rippleEffect: { brushType: 'stroke' },
        itemStyle: { color: '#f7768e', shadowBlur: 10, shadowColor: '#f7768e' },
        zlevel: 1
      }
    ]
  };

  const columns = [
    { key: 'name', label: 'Region / DC' },
    { key: 'count', label: 'Nodes', width: '80px' },
    { key: 'risk', label: 'Risk', width: '100px' },
    { key: 'status', label: 'Status', width: '100px' },
  ];

  const tableData = locations.map(l => ({ name: l.name, count: l.value[2], risk: l.value[3], status: l.status }));
</script>

<PageLayout title="Global Fleet Intelligence" subtitle="Geographical orchestration and regional threat mapping">
  {#snippet toolbar()}
    <Button variant="secondary" size="sm">Configure GEO-Blocking</Button>
    <Button variant="cta" size="sm">Zoom to Incidents</Button>
  {/snippet}

  <div class="flex flex-col h-full gap-5">
    <div class="grid grid-cols-1 md:grid-cols-4 gap-4 shrink-0">
      <KPI label="Global Regions" value="6" variant="default" />
      <KPI label="Total Node Count" value="53" variant="accent" />
      <KPI label="Inter-DC Latency" value="48ms" variant="info" />
      <KPI label="Active Conflicts" value="1" variant="critical" />
    </div>

    <div class="flex-1 min-h-[500px] grid grid-cols-1 lg:grid-cols-3 gap-5">
      <!-- Map Area -->
      <div class="lg:col-span-2 bg-surface-1 border border-border-primary rounded-md p-4 relative overflow-hidden">
        <div class="absolute inset-0 opacity-[0.05] pointer-events-none" style="background-image: radial-gradient(#7aa2f7 1px, transparent 1px); background-size: 20px 20px;"></div>
        <div class="h-full w-full relative z-10">
          <Chart option={chartOption} />
        </div>
      </div>

      <!-- Region List -->
      <div class="bg-surface-1 border border-border-primary rounded-md flex flex-col h-full overflow-hidden">
        <div class="p-4 border-b border-border-primary">
          <h3 class="text-xs font-bold uppercase tracking-widest text-text-muted">Regional Performance</h3>
        </div>
        <div class="flex-1 overflow-hidden">
          <DataTable data={tableData} {columns} compact>
            {#snippet render({ value, col, row })}
              {#if col.key === 'risk'}
                <div class="flex items-center gap-2">
                   <div class="flex-1 h-1 bg-surface-2 rounded-full overflow-hidden">
                     <div class="h-full {value > 80 ? 'bg-error' : 'bg-accent'}" style="width: {value}%"></div>
                   </div>
                   <span class="font-mono text-[9px] w-4">{value}</span>
                </div>
              {:else if col.key === 'status'}
                <Badge variant={value === 'alert' ? 'critical' : 'success'} dot>{value}</Badge>
              {:else}
                {value}
              {/if}
            {/snippet}
          </DataTable>
        </div>
      </div>
    </div>
  </div>
</PageLayout>
