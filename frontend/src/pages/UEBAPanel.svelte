<!--
  OBLIVRA — UEBA Panel (Svelte 5)
  User and Entity Behavior Analytics: Deviation detection and identity risk scoring.

  Audit fix — every list and KPI on this page used to be hardcoded:
    • riskEntities had three fake users (maverick:88, svc_jenkins:42,
      operator_k:94)
    • Avg Risk Score, Observed Entities, AI Confidence were string
      literals (12.4, 1422, 94.2%)
    • Top Anomaly Sources used a hardcoded [['Unusual Hours', '42%'],
      ['Geo-Drift', '12%'], ['Process Lineage', '24%']] array
  Fake risk scores presented to operators are dangerous — phantom
  scores either trigger investigations of innocents or hide real
  threats. We now drive everything from the uebaStore which already
  fetches /api/v1/ueba/{profiles,anomalies,stats} (browser) — desktop
  mode uses the same endpoints since the UEBA service exposes them
  uniformly. Honest empty/loading states replace the fakes.
-->
<script lang="ts">
  import { onMount } from 'svelte';
  import { KPI, Badge, DataTable, PageLayout, Button } from '@components/ui';
  import { User, Activity } from 'lucide-svelte';
  import { uebaStore, type EntityProfile, type Anomaly } from '@lib/stores/ueba.svelte';

  type RiskRow = {
    id: string;
    name: string;
    type: string;
    score: number; // 0-100 scale for the bar; backend returns 0..1, we *100
    deviation: string;
    status: 'monitored' | 'isolated' | 'nominal';
  };

  // Map backend EntityProfile → row the table understands. Risk score
  // arrives as 0..1 from the backend; we present 0..100 for the bar.
  const riskEntities = $derived.by<RiskRow[]>(() => {
    const profiles = (uebaStore.profiles ?? []) as EntityProfile[];
    return profiles
      .map((p) => {
        const pct = Math.round(((p.risk_score ?? 0) * 100));
        let dev = 'Nominal';
        let status: RiskRow['status'] = 'nominal';
        if (pct >= 80) { dev = 'Extreme'; status = 'isolated'; }
        else if (pct >= 50) { dev = 'Critical'; status = 'monitored'; }
        else if (pct >= 25) { dev = 'Moderate'; status = 'monitored'; }
        return {
          id: String(p.id ?? ''),
          name: String(p.id ?? 'unknown'),
          type: String(p.type ?? 'entity'),
          score: pct,
          deviation: dev,
          status,
        };
      })
      .sort((a, b) => b.score - a.score)
      .slice(0, 50); // table virtualisation isn't enabled; cap to a sane window
  });

  // Average risk score is a real arithmetic mean over the full profile
  // set, not a fabricated number. If we have no profiles we show "—"
  // rather than 0.0 (which would imply "all green").
  const avgRiskScore = $derived.by(() => {
    const profiles = uebaStore.profiles ?? [];
    if (profiles.length === 0) return null;
    const sum = profiles.reduce((acc, p) => acc + (p.risk_score ?? 0), 0);
    return Math.round((sum / profiles.length) * 1000) / 10; // 0..100 with 1dp
  });

  const extremeCount = $derived(riskEntities.filter((e) => e.score > 80).length);

  // Top anomaly sources — derive from the `evidence` arrays attached to
  // each anomaly. Each evidence row carries a `key` (the dimension that
  // tripped) and a `value`; we count occurrences of each key and rank.
  const topAnomalySources = $derived.by(() => {
    const counts = new Map<string, number>();
    let total = 0;
    for (const a of (uebaStore.anomalies ?? []) as Anomaly[]) {
      for (const ev of a.evidence ?? []) {
        const key = String(ev.key ?? 'unknown').replace(/_/g, ' ');
        counts.set(key, (counts.get(key) ?? 0) + 1);
        total += 1;
      }
    }
    if (total === 0) return [];
    return [...counts.entries()]
      .sort((x, y) => y[1] - x[1])
      .slice(0, 5)
      .map(([label, n]) => {
        const pct = Math.round((n / total) * 100);
        return { label, pct, color: pct >= 30 ? 'error' : 'accent' };
      });
  });

  const columns: any[] = [
    { key: 'name', label: 'Entity Identity' },
    { key: 'score', label: 'Risk Score', width: '120px' },
    { key: 'deviation', label: 'Anomaly', width: '120px' },
    { key: 'status', label: 'State', width: '100px' },
    { key: 'action', label: '', width: '60px' },
  ];

  onMount(() => {
    uebaStore.refresh();
  });
</script>

<PageLayout title="Behavioral Analytics" subtitle="UEBA: Identity risk scoring and behavioral deviation detection">
  {#snippet toolbar()}
    <Button variant="secondary" size="sm" onclick={() => uebaStore.refresh()}>
      {uebaStore.loading ? 'Refreshing…' : 'Refresh Profiles'}
    </Button>
    <Button variant="primary" size="sm">Logic Re-calibrate</Button>
  {/snippet}

  <div class="flex flex-col h-full gap-6">
    <div class="grid grid-cols-1 md:grid-cols-4 gap-4">
      <KPI
        label="Avg Risk Score"
        value={avgRiskScore === null ? '—' : avgRiskScore.toFixed(1)}
        trend="stable"
        trendValue={avgRiskScore === null ? 'No data' : `over ${uebaStore.profiles.length} entities`}
        variant={avgRiskScore !== null && avgRiskScore > 50 ? 'critical' : avgRiskScore !== null && avgRiskScore > 25 ? 'warning' : 'success'}
      />
      <KPI label="Extreme Deviations" value={extremeCount.toString()} trend={extremeCount > 0 ? 'up' : 'stable'} trendValue={extremeCount > 0 ? 'Alerting' : 'Quiet'} variant={extremeCount > 0 ? 'critical' : 'muted'} />
      <KPI label="Observed Entities" value={(uebaStore.stats.total_entities ?? 0).toLocaleString()} trend="stable" trendValue="From fleet baseline" />
      <KPI label="Anomalies (24h)" value={(uebaStore.stats.anomalies_24h ?? 0).toString()} trend={uebaStore.stats.anomalies_24h > 0 ? 'up' : 'stable'} trendValue={uebaStore.stats.baselines_active > 0 ? `${uebaStore.stats.baselines_active} baselines active` : 'Baselines pending'} variant={uebaStore.stats.anomalies_24h > 0 ? 'warning' : 'success'} />
    </div>

    <div class="flex-1 min-h-0 grid grid-cols-1 lg:grid-cols-3 gap-6">
      <div class="lg:col-span-2 bg-surface-1 border border-border-primary rounded-md overflow-hidden flex flex-col shadow-card">
         <div class="p-3 bg-surface-2 border-b border-border-primary text-[10px] font-bold uppercase tracking-widest text-text-muted">
            Prioritized Risk Entities
         </div>
         <div class="flex-1 overflow-auto">
            {#if riskEntities.length === 0}
              <div class="p-12 text-center text-xs text-text-muted">
                {uebaStore.loading ? 'Loading entity profiles…' : 'No risk profiles ingested yet — the UEBA engine builds baselines as agents report telemetry.'}
              </div>
            {:else}
              <DataTable data={riskEntities} {columns} compact>
                {#snippet render({ col, row, value })}
                  {#if col.key === 'score'}
                     <div class="flex items-center gap-2">
                        <div class="flex-1 bg-surface-3 h-1 rounded-full overflow-hidden min-w-[40px]">
                           <div class="h-full {row.score > 80 ? 'bg-error' : 'bg-accent'}" style="width: {row.score}%"></div>
                        </div>
                        <span class="text-[11px] font-mono font-bold {row.score > 80 ? 'text-error' : 'text-text-primary'}">{value}</span>
                     </div>
                  {:else if col.key === 'status'}
                     <Badge variant={row.status === 'isolated' ? 'critical' : row.status === 'monitored' ? 'warning' : 'success'}>
                       {value}
                     </Badge>
                  {:else if col.key === 'name'}
                     <div class="flex items-center gap-2">
                        <User size={12} class="text-text-muted" />
                        <div class="flex flex-col">
                           <span class="text-[11px] font-bold text-text-heading">{value}</span>
                           <span class="text-[9px] text-text-muted uppercase tracking-tight">{row.type}</span>
                        </div>
                     </div>
                  {:else if col.key === 'action'}
                     <Button variant="ghost" size="sm">Profile</Button>
                  {:else}
                    <span class="text-[11px] text-text-secondary">{value}</span>
                  {/if}
                {/snippet}
              </DataTable>
            {/if}
         </div>
      </div>

      <div class="flex flex-col gap-6">
         <div class="bg-surface-1 border border-border-primary rounded-md p-4 flex flex-col gap-4">
            <div class="text-[10px] font-bold text-text-muted uppercase tracking-widest border-b border-border-primary pb-2">Top Anomaly Sources</div>
            <div class="space-y-4">
              {#if topAnomalySources.length === 0}
                <div class="text-[11px] text-text-muted">{uebaStore.loading ? 'Loading…' : 'No anomalies recorded yet.'}</div>
              {:else}
                {#each topAnomalySources as src}
                  <div>
                    <div class="flex justify-between text-[10px] mb-1">
                      <span class="text-text-secondary capitalize">{src.label}</span>
                      <span class="font-bold {src.color === 'error' ? 'text-error' : ''}">{src.pct}%</span>
                    </div>
                    <div class="w-full bg-surface-3 h-1 rounded-full overflow-hidden">
                      <div class="bg-{src.color} h-full" style="width: {src.pct}%"></div>
                    </div>
                  </div>
                {/each}
              {/if}
            </div>
         </div>

         <div class="flex-1 bg-surface-1 border border-border-primary rounded-md p-6 flex flex-col items-center justify-center text-center gap-2">
            <Activity size={32} class="text-accent opacity-40 {uebaStore.loading ? 'animate-pulse' : ''}" />
            <span class="text-xs font-bold text-text-heading mt-2">{uebaStore.stats.baselines_active > 0 ? 'AI ENGINE ENGAGED' : 'AI ENGINE IDLE'}</span>
            <p class="text-[9px] text-text-muted max-w-[180px]">
              {uebaStore.stats.baselines_active > 0
                ? `Behavioural model maintaining ${uebaStore.stats.baselines_active} baseline${uebaStore.stats.baselines_active === 1 ? '' : 's'}.`
                : 'Awaiting agent telemetry to build baselines.'}
            </p>
         </div>
      </div>
    </div>
  </div>
</PageLayout>
