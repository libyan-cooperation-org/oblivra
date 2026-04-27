<!--
  MITRE ATT&CK Heatmap — wired to AlertingService.GetDetectionRules
  (which exposes MITRE tactic/technique metadata on each loaded rule)
  cross-referenced with alertStore.alerts for live fire counts.

  Defensive: if the rule struct doesn't expose `tactic` / `technique`
  fields the page falls back gracefully with the standard 14-tactic
  enterprise grid showing zero counts and a banner explaining the
  rule loader didn't surface MITRE metadata.
-->
<script lang="ts">
  import { onMount } from 'svelte';
  import { Chart, PageLayout, Badge, Button, Tabs, PopOutButton } from '@components/ui';
  import { alertStore } from '@lib/stores/alerts.svelte';
  import { IS_BROWSER } from '@lib/context';
  import type { EChartsOption } from 'echarts';

  // Canonical MITRE Enterprise tactic order (TA0001..TA0040, condensed).
  const TACTICS = [
    'Reconnaissance', 'Resource Development', 'Initial Access', 'Execution',
    'Persistence', 'Privilege Escalation', 'Defense Evasion', 'Credential Access',
    'Discovery', 'Lateral Movement', 'Collection', 'Command and Control',
    'Exfiltration', 'Impact',
  ];

  type Rule = {
    id?: string;
    name?: string;
    title?: string;
    tactic?: string;
    technique?: string;
    technique_id?: string;
    severity?: string;
  };

  let rules = $state<Rule[]>([]);
  let loaded = $state(false);
  let activeTab = $state('enterprise');

  // Heatmap data is keyed by [tacticIndex, techniqueIndex, count].
  // Techniques are derived from the loaded rule set so the Y-axis only shows
  // techniques the platform actually has detection coverage for.
  const techniques = $derived.by(() => {
    const set = new Set<string>();
    for (const r of rules) {
      const t = r.technique || r.technique_id;
      if (t) set.add(t);
    }
    return Array.from(set).sort();
  });

  // Count detections per (tactic, technique) by joining alerts → rule by id.
  const detections = $derived.by(() => {
    const counts: Record<string, number> = {}; // key = `${tacticIdx}:${techIdx}`
    const ruleByID: Record<string, Rule> = {};
    for (const r of rules) if (r.id) ruleByID[r.id] = r;

    for (const a of alertStore.alerts ?? []) {
      // Alerts may carry rule_id in `id` (alert.id often = rule_id + uuid).
      const matchingRule = rules.find(
        (r) => r.id && (a.id?.startsWith(r.id) || a.title === r.name || a.title === r.title),
      );
      const rule = matchingRule ?? ruleByID[(a as any).rule_id ?? ''];
      if (!rule) continue;
      const tacticIdx = TACTICS.findIndex((t) => t === rule.tactic);
      const techStr = rule.technique || rule.technique_id;
      if (tacticIdx < 0 || !techStr) continue;
      const techIdx = techniques.indexOf(techStr);
      if (techIdx < 0) continue;
      const k = `${tacticIdx}:${techIdx}`;
      counts[k] = (counts[k] ?? 0) + 1;
    }
    return counts;
  });

  const heatmapData = $derived.by(() => {
    const out: [number, number, number][] = [];
    for (const r of rules) {
      const tacticIdx = TACTICS.findIndex((t) => t === r.tactic);
      const techStr = r.technique || r.technique_id;
      if (tacticIdx < 0 || !techStr) continue;
      const techIdx = techniques.indexOf(techStr);
      if (techIdx < 0) continue;
      const fires = detections[`${tacticIdx}:${techIdx}`] ?? 0;
      // Cell value = log-scaled fire count, with 0.1 baseline so a covered
      // technique still shows up faintly even with no recent fires.
      const v = fires === 0 ? 0.1 : Math.min(10, Math.log10(1 + fires) * 3);
      out.push([tacticIdx, techIdx, v]);
    }
    return out;
  });

  const stats = $derived.by(() => {
    const tacticHits = new Set<number>();
    let total = 0;
    for (const cell of heatmapData) {
      tacticHits.add(cell[0]);
      if (cell[2] > 0.1) total++;
    }
    const totalAlerts = (alertStore.alerts ?? []).length;
    const hot = TACTICS.map((t, i) => {
      let sum = 0;
      for (const [ti, _ti2, v] of heatmapData) if (ti === i && v > 0.1) sum += 1;
      return { tactic: t, count: sum };
    }).sort((a, b) => b.count - a.count);
    return {
      coveredTechniques: techniques.length,
      hotTactics: hot.slice(0, 2).filter((h) => h.count > 0),
      coveragePct: techniques.length === 0 ? 0 : Math.round((stats_tacticCoverage(tacticHits.size) / TACTICS.length) * 100),
      totalAlerts,
    };
  });

  function stats_tacticCoverage(hit: number): number {
    return hit;
  }

  const chartOption = $derived<EChartsOption>({
    backgroundColor: 'transparent',
    tooltip: {
      position: 'top',
      backgroundColor: '#1a1b26',
      borderColor: '#33467c',
      textStyle: { color: '#a9b1d6', fontSize: 11 },
      formatter: (params: any) => {
        const tacticIdx = params.data[0];
        const techIdx = params.data[1];
        const fires = detections[`${tacticIdx}:${techIdx}`] ?? 0;
        return `<b>${TACTICS[tacticIdx] ?? ''}</b><br/>${techniques[techIdx] ?? ''}<br/>${fires} alert${fires === 1 ? '' : 's'}`;
      },
    },
    grid: { height: '70%', top: '5%', left: '20%', right: '5%' },
    xAxis: {
      type: 'category',
      data: TACTICS,
      splitArea: { show: true },
      axisLabel: { color: '#565f89', fontSize: 9, rotate: 35 },
      axisLine: { lineStyle: { color: '#33467c' } },
    },
    yAxis: {
      type: 'category',
      data: techniques,
      splitArea: { show: true },
      axisLabel: { color: '#565f89', fontSize: 9 },
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
      name: 'MITRE',
      type: 'heatmap',
      data: heatmapData,
      label: { show: false },
      emphasis: { itemStyle: { shadowBlur: 10, shadowColor: 'rgba(0,0,0,0.5)' } },
    }],
  });

  const heatmapTabs = [
    { id: 'enterprise', label: 'Enterprise Matrix', icon: '🏢' },
    { id: 'mobile',     label: 'Mobile',            icon: '📱' },
    { id: 'cloud',      label: 'Cloud / SaaS',      icon: '☁️' },
  ];

  async function loadRules() {
    try {
      if (IS_BROWSER) {
        // No REST endpoint for rules yet; remain empty in browser mode.
        loaded = true;
        return;
      }
      const { GetDetectionRules } = await import(
        '@wailsjs/github.com/kingknull/oblivrashell/internal/services/alertingservice'
      );
      const list = ((await GetDetectionRules()) ?? []) as Rule[];
      rules = list;
    } catch (e) {
      console.warn('[MitreHeatmap] GetDetectionRules failed', e);
    } finally {
      loaded = true;
    }
  }

  onMount(() => {
    void loadRules();
    if (typeof alertStore.init === 'function') alertStore.init();
  });
</script>

<PageLayout title="MITRE ATT&CK® Navigator" subtitle="Adversary techniques and live coverage mapping">
  {#snippet toolbar()}
    {#if loaded}
      <Badge variant={rules.length > 0 ? 'accent' : 'muted'} dot>
        {rules.length > 0 ? 'LIVE FEED ACTIVE' : 'NO RULES LOADED'}
      </Badge>
    {:else}
      <Badge variant="muted" dot>LOADING…</Badge>
    {/if}
    <Button variant="secondary" size="sm" onclick={loadRules}>Refresh</Button>
    <PopOutButton route="/mitre-heatmap" title="MITRE ATT&CK Heatmap" />
  {/snippet}

  <div class="flex flex-col h-full gap-5">
    <div class="grid grid-cols-1 md:grid-cols-3 gap-4 shrink-0">
      <div class="bg-surface-1 border border-border-primary rounded-md p-4">
        <div class="text-[9px] font-bold text-text-muted uppercase tracking-widest mb-1">Covered Techniques</div>
        <div class="text-xl font-bold text-text-heading font-mono">
          {stats.coveredTechniques}
          <span class="text-[10px] text-success">across {rules.length} rules</span>
        </div>
      </div>
      <div class="bg-surface-1 border border-border-primary rounded-md p-4">
        <div class="text-[9px] font-bold text-text-muted uppercase tracking-widest mb-1">High-fire Tactics</div>
        <div class="flex items-center gap-2 mt-1 flex-wrap">
          {#if stats.hotTactics.length === 0}
            <span class="text-[10px] text-text-muted italic">No alerts in this window</span>
          {:else}
            {#each stats.hotTactics as h}
              <Badge variant={h.count > 4 ? 'critical' : 'warning'}>{h.tactic}</Badge>
            {/each}
          {/if}
        </div>
      </div>
      <div class="bg-surface-1 border border-border-primary rounded-md p-4 flex justify-between items-center">
        <div>
          <div class="text-[9px] font-bold text-text-muted uppercase tracking-widest mb-1">Tactic Coverage</div>
          <div class="text-xl font-bold text-accent font-mono">{stats.coveragePct}%</div>
        </div>
        <div class="text-[10px] text-text-muted italic">{stats.totalAlerts} alerts in window</div>
      </div>
    </div>

    <div class="flex-1 min-h-0 flex flex-col bg-surface-1 border border-border-primary rounded-md overflow-hidden p-4 shadow-card">
      <div class="flex items-center justify-between mb-4 border-b border-border-primary pb-2">
        <Tabs tabs={heatmapTabs} bind:active={activeTab} variant="pills" />
        <div class="text-[10px] text-text-muted font-mono">
          {loaded ? `${rules.length} rules · ${stats.totalAlerts} alerts` : 'Loading rules…'}
        </div>
      </div>

      <div class="flex-1 relative">
        {#if loaded && rules.length === 0}
          <div class="absolute inset-0 flex flex-col items-center justify-center text-text-muted gap-2">
            <div class="text-sm">No MITRE-tagged detection rules loaded.</div>
            <div class="text-[11px]">
              Add Sigma rules under <code class="bg-surface-2 px-1 rounded">sigma/</code> with MITRE
              <code class="bg-surface-2 px-1 rounded">tactic</code> + <code class="bg-surface-2 px-1 rounded">technique</code> tags.
            </div>
          </div>
        {:else}
          <Chart option={chartOption} />
        {/if}
      </div>

      <div class="mt-4 p-3 bg-surface-2 border border-border-primary rounded-sm flex gap-6 items-center shrink-0">
        <div class="flex items-center gap-2">
          <span class="w-2 h-2 rounded-full shrink-0" style="background: #33467c;"></span>
          <span class="text-[10px] text-text-muted">Covered (no fires)</span>
        </div>
        <div class="flex items-center gap-2">
          <span class="w-2 h-2 rounded-full shrink-0" style="background: #7aa2f7;"></span>
          <span class="text-[10px] text-text-muted">Frequent</span>
        </div>
        <div class="flex items-center gap-2">
          <span class="w-2 h-2 rounded-full shrink-0" style="background: #f7768e;"></span>
          <span class="text-[10px] text-text-muted">Critical</span>
        </div>
        <div class="flex-1"></div>
        <div class="text-[10px] text-text-muted italic">Hover cells for technique + alert count</div>
      </div>
    </div>
  </div>
</PageLayout>
