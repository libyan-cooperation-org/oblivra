<!--
  OBLIVRA — Mitre Attack Heatmap (Svelte 5)
  Interactive visualization of adversary techniques detected in the environment.
-->
<script lang="ts">
  import { Chart, PageLayout, Badge, Button, Tabs } from '@components/ui';
  import type { EChartsOption } from 'echarts';

  const tactics = [
    'Initial Access', 'Execution', 'Persistence', 'Privilege Escalation',
    'Defense Evasion', 'Credential Access', 'Discovery', 'Lateral Movement',
  ];

  const techniques = [
    'Valid Accounts', 'Spearphishing', 'Command & Script Interpreter',
    'Scheduled Task', 'Process Injection', 'OS Credential Dumping',
    'Network Service Scanning', 'Remote Services',
  ];

  const data = [
    [0, 0, 5], [0, 1, 1], [1, 2, 8], [1, 3, 2],
    [2, 3, 4], [3, 4, 7], [4, 4, 3], [5, 5, 9],
    [6, 6, 2], [7, 7, 6], [0, 2, 1], [1, 4, 3],
  ];

  const chartOption = $derived<EChartsOption>({
    backgroundColor: 'transparent',
    tooltip: {
      position: 'top',
      backgroundColor: '#1a1b26',
      borderColor: '#33467c',
      textStyle: { color: '#a9b1d6', fontSize: 11 },
      formatter: (params: any) =>
        `<b>${tactics[params.data[0]]}</b><br/>${techniques[params.data[1]]}: ${params.data[2]} detections`,
    },
    grid: { height: '75%', top: '10%', left: '10%', right: '5%' },
    xAxis: {
      type: 'category',
      data: tactics,
      splitArea: { show: true },
      axisLabel: { color: '#565f89', fontSize: 10, rotate: 30 },
      axisLine: { lineStyle: { color: '#33467c' } },
    },
    yAxis: {
      type: 'category',
      data: techniques,
      splitArea: { show: true },
      axisLabel: { color: '#565f89', fontSize: 10 },
      axisLine: { lineStyle: { color: '#33467c' } },
    },
    visualMap: {
      min: 0,
      max: 10,
      calculable: true,
      orient: 'horizontal',
      left: 'center',
      bottom: '2%',
      inRange: { color: ['#1a1b26', '#33467c', '#7aa2f7', '#f7768e'] },
      textStyle: { color: '#565f89', fontSize: 9 },
    },
    series: [{
      name: 'MITRE Tech',
      type: 'heatmap',
      data: data,
      label: { show: false },
      emphasis: { itemStyle: { shadowBlur: 10, shadowColor: 'rgba(0,0,0,0.5)' } },
    }],
  });

  const heatmapTabs = [
    { id: 'enterprise', label: 'Enterprise Matrix', icon: '🏢' },
    { id: 'mobile', label: 'Mobile', icon: '📱' },
    { id: 'cloud', label: 'Cloud / SaaS', icon: '☁️' },
  ];
  let activeTab = $state('enterprise');
</script>

<PageLayout title="MITRE ATT&CK® Navigator" subtitle="Adversary techniques and coverage mapping">
  {#snippet toolbar()}
    <Badge variant="accent" dot>LIVE FEED ACTIVE</Badge>
    <Button variant="secondary" size="sm">Export Report</Button>
  {/snippet}

  <div class="flex flex-col h-full gap-5">
    <div class="grid grid-cols-1 md:grid-cols-3 gap-4 shrink-0">
      <div class="bg-surface-1 border border-border-primary rounded-md p-4">
        <div class="text-[9px] font-bold text-text-muted uppercase tracking-widest mb-1">Total Techniques</div>
        <div class="text-xl font-bold text-text-heading font-mono">42 <span class="text-[10px] text-success">/ 191</span></div>
      </div>
      <div class="bg-surface-1 border border-border-primary rounded-md p-4">
        <div class="text-[9px] font-bold text-text-muted uppercase tracking-widest mb-1">High Intensity Tactics</div>
        <div class="flex items-center gap-2 mt-1">
          <Badge variant="critical">Credential Access</Badge>
          <Badge variant="warning">Execution</Badge>
        </div>
      </div>
      <div class="bg-surface-1 border border-border-primary rounded-md p-4 flex justify-between items-center">
        <div>
          <div class="text-[9px] font-bold text-text-muted uppercase tracking-widest mb-1">Defense Coverage</div>
          <div class="text-xl font-bold text-accent font-mono">68%</div>
        </div>
        <div class="text-[10px] text-text-muted italic">Above global avg</div>
      </div>
    </div>

    <div class="flex-1 min-h-0 flex flex-col bg-surface-1 border border-border-primary rounded-md overflow-hidden p-4 shadow-card">
      <div class="flex items-center justify-between mb-4 border-b border-border-primary pb-2">
        <Tabs tabs={heatmapTabs} bind:active={activeTab} variant="pills" />
        <div class="text-[10px] text-text-muted font-mono">Last updated: 2m ago</div>
      </div>
      
      <div class="flex-1 relative">
        <Chart option={chartOption} />
      </div>

      <div class="mt-4 p-3 bg-surface-2 border border-border-primary rounded-sm flex gap-6 items-center shrink-0">
        <div class="flex items-center gap-2">
          <span class="w-2 h-2 rounded-full shrink-0" style="background: #33467c;"></span>
          <span class="text-[10px] text-text-muted">Low Activity</span>
        </div>
        <div class="flex items-center gap-2">
          <span class="w-2 h-2 rounded-full shrink-0" style="background: #7aa2f7;"></span>
          <span class="text-[10px] text-text-muted">Frequent Activity</span>
        </div>
        <div class="flex items-center gap-2">
          <span class="w-2 h-2 rounded-full shrink-0" style="background: #f7768e;"></span>
          <span class="text-[10px] text-text-muted">Critical Threat</span>
        </div>
        <div class="flex-1"></div>
        <div class="text-[10px] text-text-muted italic">Click cells to view specific event lineage</div>
      </div>
    </div>
  </div>
</PageLayout>
