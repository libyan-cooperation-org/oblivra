<!--
  OBLIVRA — Executive Dashboard (Svelte 5)
  Global platform health, risk scoring, and security posture summary.
-->
<script lang="ts">
  import { Chart, PageLayout, KPI, DataTable, Badge } from '@components/ui';
  import type { EChartsOption } from 'echarts';

  const riskOption: EChartsOption = {
    backgroundColor: 'transparent',
    tooltip: { trigger: 'item' },
    series: [
      {
        name: 'Risk Distribution',
        type: 'pie',
        radius: ['40%', '70%'],
        avoidLabelOverlap: false,
        itemStyle: {
          borderRadius: 4,
          borderColor: '#1a1b26',
          borderWidth: 2
        },
        label: {
          show: false,
          position: 'center'
        },
        emphasis: {
          label: {
            show: true,
            fontSize: 14,
            fontWeight: 'bold',
            color: '#a9b1d6'
          }
        },
        labelLine: { show: false },
        data: [
          { value: 1048, name: 'Critical', itemStyle: { color: '#f7768e' } },
          { value: 735, name: 'High', itemStyle: { color: '#ff9e64' } },
          { value: 580, name: 'Medium', itemStyle: { color: '#e0af68' } },
          { value: 484, name: 'Low', itemStyle: { color: '#9ece6a' } },
          { value: 300, name: 'Info', itemStyle: { color: '#7aa2f7' } }
        ]
      }
    ]
  };

  const activityOption: EChartsOption = {
     backgroundColor: 'transparent',
     xAxis: { type: 'category', data: ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun'], axisLabel: { color: '#565f89' } },
     yAxis: { type: 'value', splitLine: { lineStyle: { color: '#1a1b26' } }, axisLabel: { color: '#565f89' } },
     series: [{ data: [120, 200, 150, 80, 70, 110, 130], type: 'bar', itemStyle: { color: '#7aa2f7' } }]
  }
</script>

<PageLayout title="Executive Overview" subtitle="High-level security posture and fleet health metrics">
  <div class="flex flex-col h-full gap-5">
    <!-- Top KPIs -->
    <div class="grid grid-cols-1 md:grid-cols-4 gap-4 shrink-0">
      <KPI label="Platform Integrity" value="99.9%" variant="success" />
      <KPI label="Fleet Risk Score" value="12.4" variant="info" />
      <KPI label="Active Threats" value="2" variant="critical" />
      <KPI label="Audit Compliance" value="Passed" variant="accent" />
    </div>

    <!-- Charts Row -->
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-5 flex-1 min-h-0">
      <div class="bg-surface-1 border border-border-primary rounded-md p-6 flex flex-col">
          <h3 class="text-xs font-bold uppercase tracking-widest text-text-muted mb-6">Threat Severity breakdown</h3>
          <div class="flex-1">
            <Chart option={riskOption} />
          </div>
      </div>
      <div class="bg-surface-1 border border-border-primary rounded-md p-6 flex flex-col">
          <h3 class="text-xs font-bold uppercase tracking-widest text-text-muted mb-6">Analyst Activity (Weekly)</h3>
          <div class="flex-1">
            <Chart option={activityOption} />
          </div>
      </div>
    </div>

    <!-- Recent Milestones / Posture -->
    <div class="p-4 bg-accent/5 border border-accent/20 rounded-md">
       <div class="flex justify-between items-center">
          <div class="flex items-center gap-3">
             <div class="w-2 h-2 rounded-full bg-accent animate-ping"></div>
             <span class="text-xs font-bold text-text-heading">AI Shield is actively mitigating 12 credential stuffing attempts.</span>
          </div>
          <Badge variant="accent">AUTOMATED RESPONSE ACTIVE</Badge>
       </div>
    </div>
  </div>
</PageLayout>
