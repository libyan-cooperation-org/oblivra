<!--
  OBLIVRA — UEBA Overview (Svelte 5)
  Real-time behavioral intelligence and anomaly orchestration.
-->
<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { Chart, KPI, Badge, DataTable, PageLayout, Button } from '@components/ui';
  import { Shield, Activity, Fingerprint, Network, AlertTriangle } from 'lucide-svelte';
  import type { EChartsOption } from 'echarts';
  import { GetProfiles, GetAnomalies } from '@wailsjs/github.com/kingknull/oblivrashell/internal/services/uebaservice';
  import { subscribe } from '@lib/bridge';

  let profiles = $state<any[]>([]);
  let anomalies = $state<any[]>([]);
  let loading = $state(true);

  // Computed Stats
  let avgRisk = $derived(profiles.length ? (profiles.reduce((acc, p) => acc + (p.RiskScore || 0), 0) / profiles.length) * 100 : 0);
  let criticalCount = $derived(anomalies.filter(a => a.risk_score > 0.8).length);
  let entityCount = $derived(profiles.length);

  let unsubAnomaly: () => void;

  async function loadData() {
    loading = true;
    try {
      const [p, a] = await Promise.all([GetProfiles(), GetAnomalies()]);
      profiles = p || [];
      anomalies = (a || []).reverse(); // Newest first
    } catch (err) {
      console.error('[ueba] failed to load behavioral data:', err);
    } finally {
      loading = false;
    }
  }

  onMount(() => {
    loadData();
    unsubAnomaly = subscribe('siem:anomaly', (anomaly) => {
      anomalies = [anomaly, ...anomalies].slice(0, 100);
      // Refresh profiles as risk scores changed
      GetProfiles().then(p => profiles = p || []);
    });
  });

  onDestroy(() => {
    unsubAnomaly?.();
  });

  const columns = [
    { key: 'ID', label: 'Entity Identifier', sortable: true },
    { key: 'Type', label: 'Type', width: '100px' },
    { key: 'PeerGroupID', label: 'Peer Group', width: '120px' },
    { key: 'RiskScore', label: 'Behavior Score', width: '150px' },
  ];

  // Distribution chart option
  let distributionOption = $derived<EChartsOption>({
    backgroundColor: 'transparent',
    tooltip: { trigger: 'item' },
    series: [
      {
        name: 'Risk Distribution',
        type: 'pie',
        radius: ['40%', '70%'],
        avoidLabelOverlap: false,
        itemStyle: { borderRadius: 10, borderColor: '#1a1b26', borderWidth: 2 },
        label: { show: false },
        emphasis: { label: { show: true, fontSize: 12, fontWeight: 'bold', color: '#fff' } },
        data: [
          { value: profiles.filter(p => p.RiskScore > 0.8).length, name: 'Critical', itemStyle: { color: '#f7768e' } },
          { value: profiles.filter(p => p.RiskScore > 0.5 && p.RiskScore <= 0.8).length, name: 'Warning', itemStyle: { color: '#ff9e64' } },
          { value: profiles.filter(p => p.RiskScore <= 0.5).length, name: 'Normal', itemStyle: { color: '#73daca' } },
        ]
      }
    ]
  });
</script>

<PageLayout title="Behavioral Intelligence" subtitle="Autonomous anomaly scoring and peer group analysis engine">
  {#snippet toolbar()}
    <Button variant="secondary" size="sm" onclick={loadData}>Refresh Baselines</Button>
    <Button variant="primary" size="sm">Download Audit Report</Button>
  {/snippet}

  <div class="flex flex-col h-full gap-5">
    <!-- KPI Row -->
    <div class="grid grid-cols-1 md:grid-cols-3 gap-4 shrink-0">
      <KPI label="Average Risk Score" value={avgRisk.toFixed(1)} variant="info" icon={Activity} />
      <KPI label="Critical Deviations" value={criticalCount.toString()} variant="critical" icon={AlertTriangle} />
      <KPI label="Observed Entities" value={entityCount.toString()} variant="accent" icon={Fingerprint} />
    </div>

    <div class="grid grid-cols-1 lg:grid-cols-3 gap-5 flex-1 min-h-0">
      <!-- Live Anomaly Feed -->
      <div class="lg:col-span-2 bg-slate-900/40 border border-white/5 rounded-lg flex flex-col overflow-hidden backdrop-blur-md">
        <div class="p-4 border-b border-white/5 flex items-center justify-between">
            <div class="flex items-center gap-2">
                <Network class="w-4 h-4 text-pink-400" />
                <h3 class="text-xs font-bold uppercase tracking-widest text-slate-300">Active Deviations</h3>
            </div>
            <span class="text-[10px] font-mono text-green-400 bg-green-400/10 px-2 py-0.5 rounded-full animate-pulse">Live</span>
        </div>
        <div class="flex-1 overflow-y-auto p-4 space-y-3">
          {#if anomalies.length === 0}
            <div class="h-full flex flex-col items-center justify-center text-slate-500 gap-2 opacity-40">
                <Activity class="w-10 h-10" />
                <p class="text-[10px] font-mono uppercase">Waiting for behavioral events...</p>
            </div>
          {:else}
            {#each anomalies as anomaly}
              <div class="p-3 bg-white/5 border border-white/5 rounded-lg flex items-start gap-4 hover:border-pink-500/20 transition-all cursor-pointer group">
                <div class="p-2 rounded bg-pink-500/10 border border-pink-500/20 group-hover:scale-110 transition-transform">
                    <Shield class="w-4 h-4 text-pink-400" />
                </div>
                <div class="flex-1 min-w-0">
                  <div class="flex items-center justify-between mb-1">
                    <span class="text-xs font-bold text-white truncate">{anomaly.entity_id}</span>
                    <Badge variant="critical">RISK: {(anomaly.risk_score * 100).toFixed(0)}</Badge>
                  </div>
                  <div class="text-[10px] font-mono text-slate-500 mb-2 truncate">PEER GROUP: {anomaly.peer_group_id} | {anomaly.timestamp}</div>
                  <div class="grid grid-cols-3 gap-2">
                    {#each anomaly.evidence as ev}
                      <div class="px-2 py-1 bg-black/20 rounded border border-white/5">
                        <div class="text-[8px] text-slate-500 uppercase truncate">{ev.key}</div>
                        <div class="text-[10px] font-bold text-white">{(ev.value * 100).toFixed(0)}%</div>
                      </div>
                    {/each}
                  </div>
                </div>
              </div>
            {/each}
          {/if}
        </div>
      </div>

      <!-- Entity Risk Table -->
      <div class="bg-slate-900/40 border border-white/5 rounded-lg flex flex-col h-full overflow-hidden backdrop-blur-md">
        <div class="p-4 border-b border-white/5 flex flex-col gap-4">
          <h3 class="text-xs font-bold uppercase tracking-widest text-slate-300">Entity Risk Distribution</h3>
          <div class="h-48">
            <Chart option={distributionOption} />
          </div>
        </div>
        <div class="flex-1 overflow-hidden flex flex-col">
            <div class="px-4 py-2 border-b border-white/5">
                <h3 class="text-[10px] font-bold uppercase tracking-widest text-slate-500">Top Risky Profiles</h3>
            </div>
            <div class="flex-1 overflow-hidden">
                <DataTable data={profiles.sort((a,b) => b.RiskScore - a.RiskScore).slice(0, 50)} {columns} compact>
                    {#snippet render({ value, col, row })}
                        {#if col.key === 'RiskScore'}
                            <div class="flex items-center gap-2">
                                <div class="flex-1 h-1 bg-white/5 rounded-full overflow-hidden">
                                    <div 
                                        class="h-full {value > 0.8 ? 'bg-pink-500' : value > 0.5 ? 'bg-orange-500' : 'bg-green-500'}" 
                                        style="width: {value * 100}%"
                                    ></div>
                                </div>
                                <span class="font-mono text-[10px] w-8 text-right">{(value * 100).toFixed(0)}</span>
                            </div>
                        {:else if col.key === 'Type'}
                            <Badge variant="muted">{value.toUpperCase()}</Badge>
                        {:else}
                            <span class="text-[10px] font-mono">{value}</span>
                        {/if}
                    {/snippet}
                </DataTable>
            </div>
        </div>
      </div>
    </div>
  </div>
</PageLayout>
