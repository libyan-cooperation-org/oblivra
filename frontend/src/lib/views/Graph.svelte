<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { graphStats, graphSubgraph, type GraphStats, type GraphEdge } from '../bridge';
  import Tile from '../components/Tile.svelte';

  let stats = $state<GraphStats | null>(null);
  let edges = $state<GraphEdge[]>([]);
  let kind = $state('case');
  let id = $state('');
  let depth = $state(2);
  let error = $state<string | null>(null);
  let timer: ReturnType<typeof setInterval> | null = null;

  async function refreshStats() {
    try { stats = await graphStats(); } catch (e) { error = (e as Error).message; }
  }
  async function refreshSub() {
    if (!id) return;
    try { edges = await graphSubgraph(kind, id, depth); error = null; }
    catch (e) { error = (e as Error).message; }
  }

  onMount(() => { void refreshStats(); timer = setInterval(refreshStats, 6000); });
  onDestroy(() => { if (timer) clearInterval(timer); });

  // Build a tiny SVG graph: arrange unique nodes around a circle, draw edges.
  const layout = $derived.by(() => {
    if (edges.length === 0) return { nodes: [], lines: [] };
    const nodes = new Map<string, { x: number; y: number; label: string; kind: string }>();
    const cx = 320, cy = 220, r = 170;
    const ids: string[] = [];
    for (const e of edges) {
      for (const n of [e.from, e.to]) {
        const k = n.kind + ':' + n.id;
        if (!nodes.has(k)) {
          ids.push(k);
          nodes.set(k, { x: 0, y: 0, label: n.id, kind: n.kind });
        }
      }
    }
    const N = ids.length;
    ids.forEach((k, i) => {
      const a = (i / N) * 2 * Math.PI;
      const node = nodes.get(k)!;
      node.x = cx + r * Math.cos(a);
      node.y = cy + r * Math.sin(a);
    });
    const lines = edges.map((e) => {
      const f = nodes.get(e.from.kind + ':' + e.from.id)!;
      const t = nodes.get(e.to.kind + ':' + e.to.id)!;
      return { x1: f.x, y1: f.y, x2: t.x, y2: t.y, kind: e.kind };
    });
    return { nodes: Array.from(nodes.values()), lines };
  });

  const colorFor = (k: string) =>
    k === 'event' ? '#7c3aed'
    : k === 'alert' ? '#ef4444'
    : k === 'case' ? '#22c55e'
    : k === 'evidence' ? '#f59e0b'
    : k === 'session' ? '#38bdf8'
    : '#94a3b8';
</script>

<div class="mx-auto max-w-7xl space-y-6 p-8">
  <header>
    <p class="text-xs uppercase tracking-widest text-night-300">Evidence Graph</p>
    <h2 class="text-2xl font-semibold tracking-tight">Cross-references between events, alerts, cases, sessions, evidence</h2>
  </header>

  <section class="grid grid-cols-2 gap-4 lg:grid-cols-4">
    <Tile label="Edges" value={stats?.edges ?? 0} />
    <Tile label="Nodes" value={stats?.nodes ?? 0} />
    <Tile label="Edge kinds" value={stats?.byKind ? Object.keys(stats.byKind).length : 0} />
    <Tile label="Subgraph rendered" value={edges.length} hint={id ? `${kind}:${id}` : '—'} />
  </section>

  <section class="rounded-xl border border-night-700 bg-night-900/70 p-4 space-y-3">
    <div class="flex flex-wrap items-center gap-2">
      <select bind:value={kind} class="rounded-md border border-night-600 bg-night-800/70 px-2 py-1.5 text-xs text-slate-100">
        <option>event</option>
        <option>alert</option>
        <option>case</option>
        <option>evidence</option>
        <option>session</option>
        <option>indicator</option>
      </select>
      <input bind:value={id} placeholder="node id"
        class="flex-1 min-w-[20ch] rounded-md border border-night-600 bg-night-800/70 px-3 py-1.5 text-xs text-slate-100 font-mono" />
      <input type="number" bind:value={depth} min="1" max="5"
        class="w-16 rounded-md border border-night-600 bg-night-800/70 px-2 py-1.5 text-xs text-slate-100" />
      <button class="rounded-md bg-accent-500 px-3 py-1.5 text-xs font-medium text-white hover:bg-accent-600"
        onclick={refreshSub}>Render</button>
    </div>
    {#if error}<p class="text-xs text-signal-error">{error}</p>{/if}
  </section>

  <section class="rounded-xl border border-night-700 bg-night-900/70 p-4">
    {#if edges.length === 0}
      <div class="py-12 text-center text-sm text-night-300">
        Pick a node above to render its subgraph.
      </div>
    {:else}
      <svg viewBox="0 0 640 440" class="w-full max-h-[440px]">
        {#each layout.lines as l}
          <line x1={l.x1} y1={l.y1} x2={l.x2} y2={l.y2} stroke="#475569" stroke-width="1" />
        {/each}
        {#each layout.nodes as n}
          <g>
            <circle cx={n.x} cy={n.y} r="8" fill={colorFor(n.kind)} />
            <text x={n.x} y={n.y - 12} text-anchor="middle" fill="#cbd5e1" font-size="10" font-family="ui-monospace, monospace">
              {n.kind}:{n.label}
            </text>
          </g>
        {/each}
      </svg>
    {/if}
  </section>

  {#if Object.keys(stats?.byKind ?? {}).length > 0}
    <section class="rounded-xl border border-night-700 bg-night-900/70">
      <div class="border-b border-night-700 px-4 py-3 text-sm font-semibold tracking-wide">Edge kinds</div>
      <ul class="divide-y divide-night-700/70 font-mono text-xs">
        {#each Object.entries(stats?.byKind ?? {}) as [kindLabel, count]}
          <li class="flex items-center gap-3 px-4 py-2">
            <span class="text-slate-100">{kindLabel}</span>
            <span class="ml-auto text-night-300">{count}</span>
          </li>
        {/each}
      </ul>
    </section>
  {/if}
</div>
