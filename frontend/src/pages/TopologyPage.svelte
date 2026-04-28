<!--
  OBLIVRA — Network Topology (Svelte 5)

  Force-directed graph of hosts, services, and the relationships
  recorded by the security-graph engine (internal/graph). Pulls live
  data from `GraphService.GetFullGraph` via Wails (or `/api/v1/graph/full`
  in browser mode), so the panel reflects the actual fleet rather than
  hardcoded sample data.
-->
<script lang="ts">
  import { onMount } from 'svelte';
  import { Chart, PageLayout, Button, Badge, KPI } from '@components/ui';
  import { RefreshCw, Snowflake } from 'lucide-svelte';
  import { agentStore } from '@lib/stores/agent.svelte';
  import { appStore } from '@lib/stores/app.svelte';
  import { apiFetch } from '@lib/apiClient';
  import { IS_BROWSER } from '@lib/context';
  import type { EChartsOption } from 'echarts';

  type GraphNode = {
    id: string;
    name?: string;
    type?: string;
    meta?: Record<string, string>;
  };
  type GraphEdge = {
    from: string;
    to: string;
    type?: string;
  };

  let nodes = $state<any[]>([]);
  let links = $state<any[]>([]);
  let loading = $state(false);
  let frozen = $state(false);
  let lastRefresh = $state<string>('');

  // Map graph node types onto ECharts categories. We keep a small,
  // canonical set of buckets so the legend is readable; anything we
  // don't recognise lands in "Other".
  const TYPE_BUCKETS: Record<string, number> = {
    host: 0,
    agent: 0,
    process: 1,
    service: 1,
    user: 2,
    ip: 3,
    network: 3,
    file: 4,
  };
  const categories = [
    { name: 'Host / Agent' },
    { name: 'Process / Service' },
    { name: 'User' },
    { name: 'Network / IP' },
    { name: 'File / Other' },
  ];

  function bucketFor(t?: string): number {
    return TYPE_BUCKETS[String(t ?? '').toLowerCase()] ?? 4;
  }

  async function fetchGraph(): Promise<{ nodes: GraphNode[]; edges: GraphEdge[] }> {
    if (IS_BROWSER) {
      const res = await apiFetch('/api/v1/graph/full');
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      const body = await res.json();
      return { nodes: body.nodes ?? [], edges: body.edges ?? [] };
    }
    try {
      const { GetFullGraph } = await import(
        '@wailsjs/github.com/kingknull/oblivrashell/internal/services/graphservice'
      );
      const out = await GetFullGraph();
      return {
        nodes: (out?.nodes as GraphNode[]) ?? [],
        edges: (out?.edges as GraphEdge[]) ?? [],
      };
    } catch (e) {
      // Falls through to empty graph; user will see "Total Nodes: 0".
      console.warn('[Topology] GraphService.GetFullGraph unavailable:', e);
      return { nodes: [], edges: [] };
    }
  }

  async function refresh() {
    if (frozen) {
      appStore.notify('Layout frozen — un-freeze to refresh', 'info');
      return;
    }
    loading = true;
    try {
      const { nodes: ns, edges: es } = await fetchGraph();
      nodes = ns.map((n) => ({
        id: n.id,
        name: n.name ?? n.id,
        category: bucketFor(n.type),
        symbolSize: n.type === 'host' || n.type === 'agent' ? 28 : 18,
        meta: n.meta ?? {},
      }));
      links = es.map((e) => ({
        source: e.from,
        target: e.to,
        label: e.type ? { show: false, formatter: e.type } : undefined,
      }));
      lastRefresh = new Date().toISOString().slice(11, 19);
    } catch (e: any) {
      appStore.notify('Topology refresh failed', 'error', e?.message ?? String(e));
    } finally {
      loading = false;
    }
  }

  // Cross-zone heuristic: count edges whose endpoints disagree on
  // `meta.zone`. Cheap and good enough for the KPI tile; the real
  // zone-flow sensor lives elsewhere.
  let crossZoneCount = $derived.by(() => {
    if (nodes.length === 0 || links.length === 0) return 0;
    const zoneOf = new Map<string, string>();
    for (const n of nodes) zoneOf.set(n.id, (n as any).meta?.zone ?? '');
    let count = 0;
    for (const e of links) {
      const a = zoneOf.get(e.source as string) ?? '';
      const b = zoneOf.get(e.target as string) ?? '';
      if (a && b && a !== b) count++;
    }
    return count;
  });

  let healthLabel = $derived.by(() => {
    const offline = (agentStore.agents ?? []).filter(
      (a) => a.status !== 'online' && a.status !== 'healthy',
    ).length;
    if (offline === 0) return { text: 'Optimal', variant: 'success' as const };
    if (offline <= 2) return { text: 'Degraded', variant: 'warning' as const };
    return { text: 'Critical', variant: 'critical' as const };
  });

  const chartOption = $derived<EChartsOption>({
    backgroundColor: 'transparent',
    tooltip: {
      formatter: (p: any) =>
        p.dataType === 'node'
          ? `<b>${p.data.name}</b><br/><span style="opacity:0.6">${categories[p.data.category]?.name ?? 'Other'}</span>`
          : `${p.data.source} → ${p.data.target}`,
    },
    legend: {
      data: categories.map((c) => c.name),
      textStyle: { color: '#a9b1d6', fontSize: 10 },
      bottom: '2%',
    },
    series: [
      {
        name: 'Network Topology',
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
          fontSize: 10,
        },
        force: {
          repulsion: 1000,
          edgeLength: 150,
        },
        lineStyle: {
          color: '#33467c',
          curveness: 0.1,
        },
        itemStyle: {
          borderColor: '#1a1b26',
          borderWidth: 2,
        },
      },
    ],
  });

  onMount(() => {
    void refresh();
    // Soft poll — every 30 s. Force-graph is a relatively heavy mount,
    // so we keep the cadence conservative.
    const id = setInterval(() => {
      if (!frozen) void refresh();
    }, 30_000);
    return () => clearInterval(id);
  });
</script>

<PageLayout title="Network Topology" subtitle="Visualizing real-time relational telemetry and data flow">
  {#snippet toolbar()}
    <Button variant="secondary" size="sm" icon={Snowflake} onclick={() => (frozen = !frozen)}>
      {frozen ? 'Unfreeze' : 'Freeze'} Layout
    </Button>
    <Button variant="secondary" size="sm" icon={RefreshCw} onclick={refresh} disabled={loading}>
      {loading ? 'Loading…' : 'Refresh'}
    </Button>
  {/snippet}

  <div class="flex flex-col h-full gap-5">
    <div class="grid grid-cols-1 md:grid-cols-4 gap-4 shrink-0">
      <KPI
        label="Total Nodes"
        value={nodes.length.toString()}
        trendPolarity="neutral"
        sublabel={lastRefresh ? `Refreshed ${lastRefresh}` : undefined}
      />
      <KPI
        label="Active Edges"
        value={links.length.toString()}
        trendPolarity="neutral"
      />
      <KPI
        label="Cross-Zone Flow"
        value={crossZoneCount.toString()}
        variant={crossZoneCount > 0 ? 'warning' : 'accent'}
        trendPolarity="up-bad"
      />
      <KPI
        label="Network Health"
        value={healthLabel.text}
        variant={healthLabel.variant}
      />
    </div>

    <div class="flex-1 min-h-[500px] bg-surface-1 border border-border-primary rounded-md p-4 relative overflow-hidden shadow-premium">
      <!-- Grid Background -->
      <div class="absolute inset-0 opacity-[0.03] pointer-events-none grayscale" style="background-image: linear-gradient(#7aa2f7 1px, transparent 1px), linear-gradient(90deg, #7aa2f7 1px, transparent 1px); background-size: 40px 40px;"></div>

      <div class="h-full w-full relative z-10">
        {#if nodes.length === 0 && !loading}
          <div class="flex flex-col items-center justify-center h-full gap-2 text-text-muted text-xs">
            <span>No topology data yet.</span>
            <span class="opacity-60">The graph populates as agents check in and detection events fire.</span>
          </div>
        {:else}
          <Chart option={chartOption} />
        {/if}
      </div>

      <!-- Overlay Map controls -->
      <div class="absolute top-4 left-4 flex flex-col gap-2 z-20">
        <div class="bg-surface-2 border border-border-secondary p-2 rounded shadow-xl flex flex-col gap-1">
          <Badge variant={frozen ? 'warning' : 'success'} dot>
            {frozen ? 'LAYOUT FROZEN' : 'LIVE MAPPING'}
          </Badge>
          <div class="text-[9px] text-text-muted uppercase font-bold tracking-widest mt-1">Force Simulation</div>
        </div>
      </div>
    </div>
  </div>
</PageLayout>
