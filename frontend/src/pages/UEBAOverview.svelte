<!--
  UEBA Overview — anomaly + profile feed from UEBAService.
-->
<script lang="ts">
  import { onMount } from 'svelte';
  import { PageLayout, Badge, Button, KPI, DataTable, PopOutButton } from '@components/ui';
  import { Users, AlertTriangle, RefreshCw, TrendingUp } from 'lucide-svelte';
  import { IS_BROWSER } from '@lib/context';
  import { push } from '@lib/router.svelte';
  import { appStore } from '@lib/stores/app.svelte';

  type Anomaly = {
    id?: string; user_id?: string; entity?: string; score?: number;
    description?: string; severity?: string; detected_at?: string;
  };
  type Profile = {
    user_id?: string; baseline?: any; risk_score?: number; last_seen?: string;
  };

  let anomalies = $state<Anomaly[]>([]);
  let profiles = $state<Profile[]>([]);
  let loading = $state(false);

  const stats = $derived.by(() => ({
    anomalies: anomalies.length,
    high: anomalies.filter((a) => (a.severity ?? '').toLowerCase() === 'high' || (a.score ?? 0) >= 0.8).length,
    profiles: profiles.length,
    avgRisk: profiles.length === 0 ? 0
      : Math.round(profiles.reduce((s, p) => s + (p.risk_score ?? 0), 0) / profiles.length * 100),
  }));

  async function refresh() {
    loading = true;
    try {
      if (IS_BROWSER) { anomalies = []; profiles = []; return; }
      const ueba = await import(
        '@wailsjs/github.com/kingknull/oblivrashell/internal/services/uebaservice'
      );
      const [an, pr] = await Promise.all([ueba.GetAnomalies(), ueba.GetProfiles()]);
      anomalies = (an ?? []) as Anomaly[];
      profiles = (pr ?? []) as Profile[];
    } catch (e: any) {
      appStore.notify(`UEBA load failed: ${e?.message ?? e}`, 'error');
    } finally { loading = false; }
  }

  onMount(refresh);
</script>

<PageLayout title="User & Entity Behaviour Analytics" subtitle="Detected anomalies and risky-user profiles">
  {#snippet toolbar()}
    <Button variant="secondary" size="sm" icon={RefreshCw} onclick={refresh}>{loading ? 'Loading…' : 'Refresh'}</Button>
    <PopOutButton route="/ueba" title="UEBA Overview" />
  {/snippet}

  <div class="flex flex-col h-full gap-4">
    <div class="grid grid-cols-1 md:grid-cols-4 gap-3 shrink-0">
      <KPI label="Live Anomalies"   value={stats.anomalies.toString()} variant={stats.anomalies > 0 ? 'critical' : 'muted'} />
      <KPI label="High Severity"    value={stats.high.toString()}      variant={stats.high > 0 ? 'critical' : 'muted'} />
      <KPI label="Profiled Users"   value={stats.profiles.toString()}  variant="accent" />
      <KPI label="Avg Risk Score"   value={`${stats.avgRisk}%`}        variant={stats.avgRisk >= 60 ? 'warning' : 'muted'} />
    </div>

    <div class="grid grid-cols-1 lg:grid-cols-3 gap-4 flex-1 min-h-0">
      <section class="lg:col-span-2 flex flex-col bg-surface-1 border border-border-primary rounded-md min-h-0">
        <div class="flex items-center gap-2 p-3 border-b border-border-primary">
          <AlertTriangle size={14} class="text-warning" />
          <span class="text-[10px] uppercase tracking-widest font-bold">Recent Anomalies</span>
          <span class="ml-auto text-[10px] text-text-muted">{anomalies.length}</span>
        </div>
        <div class="flex-1 overflow-auto">
          {#if anomalies.length === 0}
            <div class="p-6 text-sm text-text-muted text-center">{loading ? 'Loading…' : 'No anomalies detected.'}</div>
          {:else}
            <DataTable
              data={anomalies}
              columns={[
                { key: 'severity',    label: 'Sev',    width: '70px' },
                { key: 'entity',      label: 'Entity', width: '140px' },
                { key: 'description', label: 'Anomaly' },
                { key: 'score',       label: 'Score',  width: '70px' },
                { key: 'detected_at', label: 'Time',   width: '140px' },
              ]}
              compact
            >
              {#snippet render({ col, row })}
                {#if col.key === 'severity'}
                  <Badge variant={(row.severity ?? '').toLowerCase() === 'high' ? 'critical' : (row.severity ?? '').toLowerCase() === 'medium' ? 'warning' : 'info'} size="xs">
                    {row.severity ?? '—'}
                  </Badge>
                {:else if col.key === 'entity'}
                  <span class="font-mono text-[10px] text-accent">{row.entity ?? row.user_id ?? '—'}</span>
                {:else if col.key === 'score'}
                  <span class="font-mono text-[10px] {(row.score ?? 0) >= 0.8 ? 'text-error' : (row.score ?? 0) >= 0.5 ? 'text-warning' : 'text-text-muted'}">{row.score?.toFixed(2) ?? '—'}</span>
                {:else if col.key === 'detected_at'}
                  <span class="font-mono text-[10px] text-text-muted">{row.detected_at?.slice(0, 19) ?? '—'}</span>
                {:else}
                  <span class="text-[11px]">{row[col.key] ?? '—'}</span>
                {/if}
              {/snippet}
            </DataTable>
          {/if}
        </div>
      </section>

      <section class="flex flex-col bg-surface-1 border border-border-primary rounded-md min-h-0">
        <div class="flex items-center gap-2 p-3 border-b border-border-primary">
          <TrendingUp size={14} class="text-accent" />
          <span class="text-[10px] uppercase tracking-widest font-bold">Top Risk Users</span>
        </div>
        <div class="flex-1 overflow-auto p-2">
          {#if profiles.length === 0}
            <div class="p-4 text-sm text-text-muted text-center">{loading ? 'Loading…' : 'No profiles yet.'}</div>
          {:else}
            {#each profiles.slice().sort((a, b) => (b.risk_score ?? 0) - (a.risk_score ?? 0)).slice(0, 20) as p (p.user_id)}
              <button
                class="w-full text-left px-2 py-1.5 rounded-md hover:bg-surface-2 flex items-center gap-2"
                onclick={() => push(`/ueba-overview?user=${encodeURIComponent(p.user_id ?? '')}`)}
              >
                <Users size={11} class="text-text-muted" />
                <span class="font-mono text-[11px] flex-1 truncate">{p.user_id ?? '—'}</span>
                <span class="font-mono text-[10px] {(p.risk_score ?? 0) >= 0.8 ? 'text-error' : (p.risk_score ?? 0) >= 0.5 ? 'text-warning' : 'text-text-muted'}">
                  {Math.round((p.risk_score ?? 0) * 100)}
                </span>
              </button>
            {/each}
          {/if}
        </div>
      </section>
    </div>
  </div>
</PageLayout>
