<!--
  OBLIVRA — UEBA Overview (Svelte 5)
  User and Entity Behavior Analytics with anomaly detection and scoring.
-->
<script lang="ts">
  import { Chart, KPI, Badge, DataTable, PageLayout, Button, Tabs } from '@components/ui';
  import type { EChartsOption } from 'echarts';

  const uebaStats = [
    { label: 'Risk Score (Avg)', value: '14.2', variant: 'info' as const },
    { label: 'Critical Anomalies', value: '3', variant: 'critical' as const },
    { label: 'Monitored Entities', value: '1.2k', variant: 'accent' as const },
  ];

  // Anomaly timeline chart
  const timelineOption: EChartsOption = {
    backgroundColor: 'transparent',
    tooltip: { trigger: 'axis' },
    grid: { left: '3%', right: '4%', bottom: '3%', top: '10%', containLabel: true },
    xAxis: {
      type: 'category',
      boundaryGap: false,
      data: ['00:00', '04:00', '08:00', '12:00', '16:00', '20:00', '23:59'],
      axisLabel: { color: '#565f89' },
      axisLine: { lineStyle: { color: '#33467c' } }
    },
    yAxis: {
      type: 'value',
      axisLabel: { color: '#565f89' },
      splitLine: { lineStyle: { color: '#1a1b26' } }
    },
    series: [
      {
        name: 'Risk Level',
        type: 'line',
        smooth: true,
        data: [12, 10, 45, 30, 85, 40, 35],
        areaStyle: {
          color: {
            type: 'linear',
            x: 0, y: 0, x2: 0, y2: 1,
            colorStops: [{ offset: 0, color: '#f7768e' }, { offset: 1, color: 'transparent' }]
          },
          opacity: 0.2
        },
        itemStyle: { color: '#f7768e' }
      }
    ]
  };

  const riskyEntities = [
    { name: 'adm_service_01', type: 'Service Account', score: 92, status: 'critical' },
    { name: 'maverick@oblivra.sh', type: 'Admin User', score: 78, status: 'warning' },
    { name: 'vpn_node_asia_04', type: 'Infrastructure', score: 45, status: 'muted' },
  ];

  const columns = [
    { key: 'name', label: 'Entity', sortable: true },
    { key: 'type', label: 'Entity Type', width: '150px' },
    { key: 'score', label: 'Behavior Score', width: '120px' },
    { key: 'status', label: 'Level', width: '100px' },
  ];
</script>

<PageLayout title="Behavioral Intelligence" subtitle="Detecting insider threats and compromised accounts via anomaly scoring">
  {#snippet toolbar()}
    <Button variant="secondary" size="sm">Download Dataset</Button>
    <Button variant="primary" size="sm">Recalibrate ML Model</Button>
  {/snippet}

  <div class="flex flex-col h-full gap-5">
    <div class="grid grid-cols-1 md:grid-cols-3 gap-4 shrink-0">
      {#each uebaStats as stat}
        <KPI label={stat.label} value={stat.value} variant={stat.variant} />
      {/each}
    </div>

    <div class="grid grid-cols-1 lg:grid-cols-3 gap-5 flex-1 min-h-0">
      <!-- Risk Timeline -->
      <div class="lg:col-span-2 bg-surface-1 border border-border-primary rounded-md p-4 flex flex-col">
        <h3 class="text-xs font-bold uppercase tracking-widest text-text-muted mb-4">Risk Aggregation (24h)</h3>
        <div class="flex-1">
          <Chart option={timelineOption} />
        </div>
      </div>

      <!-- Risky Entities -->
      <div class="bg-surface-1 border border-border-primary rounded-md flex flex-col h-full overflow-hidden">
        <div class="p-4 border-b border-border-primary">
          <h3 class="text-xs font-bold uppercase tracking-widest text-text-muted">High Risk Entities</h3>
        </div>
        <div class="flex-1 overflow-hidden">
          <DataTable data={riskyEntities} {columns} compact>
            {#snippet render({ value, col, row })}
              {#if col.key === 'score'}
                <div class="flex items-center gap-2">
                  <div class="flex-1 h-1.5 bg-surface-2 rounded-full overflow-hidden">
                    <div 
                      class="h-full {value > 80 ? 'bg-error' : value > 60 ? 'bg-warning' : 'bg-accent'}" 
                      style="width: {value}%"
                    ></div>
                  </div>
                  <span class="font-mono text-[10px] w-6">{value}</span>
                </div>
              {:else if col.key === 'status'}
                <Badge variant={value === 'critical' ? 'critical' : value === 'warning' ? 'warning' : 'muted'}>
                  {value.toUpperCase()}
                </Badge>
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
