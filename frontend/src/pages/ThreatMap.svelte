<!--
  OBLIVRA — Threat Map (Svelte 5)
  Geospatial attribution and attack origin visualization.

  Audit fix — the previous page hardcoded a four-row geopolitical
  attribution table (CN:41 / RU:28 / KP:12 / US:15) that was completely
  decoupled from reality. The bottom-right "LIVE ATTACK STREAM" panel
  even quoted those fake origins back as if they were live events
  ("Shenzhen → PROD-CLUSTER-1") rendered with `new Date()` as the
  timestamp. Operators reading this could be misled into prioritising
  fake geopolitical attribution.

  Reality: the backend's GetThreatIntelStats returns a counter map
  (indicator-type counts) — there's no per-country breakdown service
  in this build (GeoIP is offline; rest.go:2352 returns
  country_code "??"). We now:
    • show indicator-type counts pulled from /api/v1/threatintel/stats
      or the Wails RPC, with honest empty/loading states
    • derive an "Active Sources" list from alertStore — grouping alerts
      by host so operators see WHICH assets are signal-hot, not
      fabricated geo attribution
    • remove the fake live-attack-stream until a real geo feed exists
-->
<script lang="ts">
  import { KPI, PageLayout, Button, Badge, Spinner } from '@components/ui';
  import { onMount } from 'svelte';
  import { IS_BROWSER } from '@lib/context';
  import { apiFetch } from '@lib/apiClient';
  import { alertStore } from '@lib/stores/alerts.svelte';

  let loading = $state(false);
  let threatStats = $state<Record<string, number>>({});
  let lastError = $state<string | null>(null);

  async function loadThreatIntel() {
    loading = true;
    lastError = null;
    try {
      if (IS_BROWSER) {
        const res = await apiFetch('/api/v1/threatintel/stats');
        if (res.ok) {
          const body = await res.json();
          threatStats = (body.stats ?? {}) as Record<string, number>;
        } else {
          lastError = `HTTP ${res.status}`;
        }
        return;
      }
      const { GetThreatIntelStats } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/siemservice');
      threatStats = (await GetThreatIntelStats()) as Record<string, number>;
    } catch (err: any) {
      console.error('Threat intel load failed', err);
      lastError = String(err?.message ?? err);
    } finally {
      loading = false;
    }
  }

  // Derive "Active Sources" from real alert hosts. We group by host
  // and rank by alert count + severity weight. Severity weighting:
  // critical=4, high=2, medium=1, anything else=0.5.
  type SourceRow = { host: string; count: number; weight: number; topSeverity: string };
  const activeSources = $derived.by<SourceRow[]>(() => {
    const alerts = alertStore.alerts ?? [];
    const map = new Map<string, SourceRow>();
    for (const a of alerts) {
      const host = String(a.host ?? a.entity ?? '').trim();
      if (!host) continue;
      const sev = String(a.severity ?? '').toLowerCase();
      const w = sev === 'critical' ? 4 : sev === 'high' ? 2 : sev === 'medium' ? 1 : 0.5;
      const cur = map.get(host) ?? { host, count: 0, weight: 0, topSeverity: 'low' };
      cur.count += 1;
      cur.weight += w;
      const rank = (s: string) => (s === 'critical' ? 4 : s === 'high' ? 3 : s === 'medium' ? 2 : 1);
      if (rank(sev) > rank(cur.topSeverity)) cur.topSeverity = sev;
      map.set(host, cur);
    }
    const arr = [...map.values()].sort((a, b) => b.weight - a.weight).slice(0, 6);
    return arr;
  });

  // Indicator-type breakdown for the Detection Summary card. The backend
  // returns a count map keyed by indicator type (ip, domain, hash, …);
  // we surface the totals but never invent a value.
  const totalIndicators = $derived(
    Object.values(threatStats).reduce((acc, n) => acc + (Number(n) || 0), 0),
  );
  const indicatorTypes = $derived(
    Object.entries(threatStats)
      .filter(([_, n]) => Number(n) > 0)
      .sort((x, y) => Number(y[1]) - Number(x[1])),
  );

  onMount(() => {
    loadThreatIntel();
    if (typeof alertStore.init === 'function') alertStore.init();
  });
</script>

<PageLayout title="Boundaries & Origins" subtitle="Threat-intel matcher and active alert sources">
  {#snippet toolbar()}
    <div class="flex items-center gap-2">
      <Button variant="secondary" size="sm" onclick={loadThreatIntel}>{loading ? 'Refreshing…' : 'Refresh'}</Button>
      <Badge variant={activeSources.length > 0 ? 'critical' : 'muted'}>
        {activeSources.length === 0 ? 'NO LIVE SOURCES' : `${activeSources.length} ACTIVE SOURCE${activeSources.length === 1 ? '' : 'S'}`}
      </Badge>
    </div>
  {/snippet}

  <div class="grid grid-cols-1 lg:grid-cols-4 gap-6 h-full">
    <!-- Map Canvas (Placeholder — no GeoIP lookup is wired) -->
    <div class="lg:col-span-3 bg-surface-1 border border-border-primary rounded-md relative overflow-hidden group">
      <div class="absolute inset-0 flex items-center justify-center">
        <div class="text-center opacity-25 pointer-events-none max-w-md px-6">
          <div class="text-6xl mb-4">🗺️</div>
          <div class="text-sm font-bold uppercase tracking-widest">Sovereign Mapping Engine</div>
          <div class="text-[10px] mt-2 italic leading-relaxed">
            Geospatial attribution requires online GeoIP. In offline /
            air-gapped mode this canvas is intentionally inert — see the
            sidebar for live alert-source attribution.
          </div>
        </div>
      </div>
    </div>

    <!-- Sidebar Info -->
    <div class="flex flex-col gap-6">
      <div class="bg-surface-1 border border-border-primary rounded-md p-4 space-y-4">
        <div class="text-xs font-bold text-text-heading border-b border-border-primary pb-2 uppercase tracking-tight">Active Alert Sources</div>
        {#if activeSources.length === 0}
          <div class="text-[11px] text-text-muted">No alerting hosts in the live feed.</div>
        {:else}
          <div class="space-y-4">
            {#each activeSources as item (item.host)}
              <div class="flex flex-col gap-1">
                <div class="flex justify-between items-center text-[11px]">
                  <span class="font-bold text-text-secondary truncate">{item.host}</span>
                  <Badge variant={item.topSeverity === 'critical' ? 'critical' : item.topSeverity === 'high' ? 'warning' : 'info'} size="xs">
                    {item.count} ALRT
                  </Badge>
                </div>
                <div class="w-full bg-surface-3 h-1 rounded-full overflow-hidden">
                  <div class="bg-accent h-full" style="width: {Math.min(100, Math.round(item.weight * 5))}%"></div>
                </div>
              </div>
            {/each}
          </div>
        {/if}
      </div>

      <div class="flex-1 bg-surface-1 border border-border-primary rounded-md p-4 flex flex-col gap-3 relative">
        {#if loading}
          <div class="absolute inset-0 bg-surface-1/40 backdrop-blur-xs z-10 flex items-center justify-center rounded-md">
              <Spinner />
          </div>
        {/if}
        <div class="text-xs font-bold text-text-heading uppercase tracking-tight">Detection Summary</div>
        {#if lastError}
          <div class="text-[10px] text-error font-mono">load failed: {lastError}</div>
        {/if}
        <div class="flex-1 flex flex-col justify-center gap-4">
          <KPI label="Total Indicators" value={totalIndicators.toLocaleString()} size="sm" variant={totalIndicators > 0 ? 'accent' : 'muted'} />
          <KPI
            label="Top Type"
            value={indicatorTypes[0]?.[0] ?? '—'}
            sublabel={indicatorTypes[0] ? `${indicatorTypes[0][1].toLocaleString()} indicators` : 'No matchers loaded'}
            size="sm"
            variant={indicatorTypes[0] ? 'critical' : 'muted'}
          />
          <KPI
            label="Indicator Types"
            value={indicatorTypes.length.toString()}
            sublabel="Distinct categories"
            size="sm"
            variant="success"
          />
        </div>
      </div>
    </div>
  </div>
</PageLayout>
