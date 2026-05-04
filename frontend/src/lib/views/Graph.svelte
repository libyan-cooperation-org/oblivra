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

  // Layout: arrange unique nodes around a circle, draw bezier edges.
  const layout = $derived.by(() => {
    if (edges.length === 0) return { nodes: [], lines: [] };
    const nodes = new Map<string, { x: number; y: number; label: string; kind: string }>();
    const cx = 320, cy = 220, r = 165;
    const ids: string[] = [];
    for (const e of edges) {
      for (const n of [e.from, e.to]) {
        const k = n.kind + ':' + n.id;
        if (!nodes.has(k)) { ids.push(k); nodes.set(k, { x: 0, y: 0, label: n.id, kind: n.kind }); }
      }
    }
    const N = ids.length;
    ids.forEach((k, i) => {
      const a = (i / N) * 2 * Math.PI - Math.PI / 2;
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

  // Node colours using forensic signal palette
  const colorFor = (k: string) =>
    k === 'event'     ? '#00bcd8'   // cyan
    : k === 'alert'   ? '#ff3d57'   // error red
    : k === 'case'    ? '#00e676'   // ok green
    : k === 'evidence'? '#ffab00'   // warn amber
    : k === 'session' ? '#40c4ff'   // info blue
    : k === 'indicator'? '#ff3d57'  // ioc = danger
    : '#4d6680';                    // base-300 muted

  const glowFor = (k: string) =>
    k === 'alert' || k === 'indicator' ? 'drop-shadow(0 0 4px #ff3d57)'
    : k === 'event'   ? 'drop-shadow(0 0 4px #00bcd8)'
    : k === 'case'    ? 'drop-shadow(0 0 4px #00e676)'
    : 'none';
</script>

<div class="mx-auto max-w-7xl space-y-6 p-8">
  <header>
    <p class="text-xs uppercase tracking-widest text-night-300">Evidence Graph</p>
    <h2 class="text-2xl font-semibold tracking-tight">Cross-references — events · alerts · cases · sessions · evidence</h2>
  </header>

  <section class="grid grid-cols-2 gap-4 lg:grid-cols-4">
    <Tile label="Edges"            value={stats?.edges ?? 0} />
    <Tile label="Nodes"            value={stats?.nodes ?? 0} />
    <Tile label="Edge kinds"       value={stats?.byKind ? Object.keys(stats.byKind).length : 0} />
    <Tile label="Subgraph rendered" value={edges.length} hint={id ? `${kind}:${id}` : '—'} />
  </section>

  <!-- Controls -->
  <section style="border:1px solid var(--color-base-700); background:var(--color-base-900); padding:12px 16px; display:flex; flex-wrap:wrap; align-items:center; gap:8px;">
    <select bind:value={kind} style="padding:5px 8px; border:1px solid var(--color-base-600); background:var(--color-base-800); color:#e8f4f8; font-family:'Share Tech Mono',monospace; font-size:10px; letter-spacing:1px; outline:none;">
      <option>event</option><option>alert</option><option>case</option>
      <option>evidence</option><option>session</option><option>indicator</option>
    </select>

    <input
      bind:value={id}
      placeholder="node id…"
      style="flex:1; min-width:180px; padding:5px 10px; border:1px solid var(--color-base-600); background:var(--color-base-800); color:#e8f4f8; font-family:'JetBrains Mono',monospace; font-size:11px; outline:none;"
    />

    <label style="display:flex; align-items:center; gap:6px; font-family:'Share Tech Mono',monospace; font-size:10px; letter-spacing:1px; color:var(--color-base-200);">
      DEPTH
      <input type="number" bind:value={depth} min="1" max="5"
        style="width:48px; padding:5px 6px; border:1px solid var(--color-base-600); background:var(--color-base-800); color:#e8f4f8; font-family:'Share Tech Mono',monospace; font-size:10px; outline:none; text-align:center;"
      />
    </label>

    <button
      onclick={refreshSub}
      style="padding:5px 16px; background:var(--color-cyan-500); color:#000; font-family:'Rajdhani',sans-serif; font-weight:700; font-size:12px; letter-spacing:2px; text-transform:uppercase; border:none; cursor:pointer;"
    >RENDER</button>

    {#if error}
      <span style="font-family:'Share Tech Mono',monospace; font-size:10px; color:var(--color-sig-error);">✗ {error}</span>
    {/if}
  </section>

  <!-- Graph canvas -->
  <section style="border:1px solid var(--color-base-700); background:var(--color-base-900); padding:12px;">
    {#if edges.length === 0}
      <div style="padding:48px 0; text-align:center; font-family:'Share Tech Mono',monospace; font-size:10px; letter-spacing:2px; color:var(--color-base-300);">
        PICK A NODE ABOVE TO RENDER ITS SUBGRAPH
      </div>
    {:else}
      <!-- Legend -->
      <div style="display:flex; flex-wrap:wrap; gap:12px; margin-bottom:12px; padding:0 4px; font-family:'Share Tech Mono',monospace; font-size:9px; letter-spacing:1px; color:var(--color-base-300);">
        {#each [['event','#00bcd8'],['alert','#ff3d57'],['case','#00e676'],['evidence','#ffab00'],['session','#40c4ff'],['indicator','#ff3d57']] as [kl, col]}
          <span style="display:flex; align-items:center; gap:5px;">
            <span style="width:8px;height:8px;border-radius:50%;background:{col};display:inline-block;"></span>
            {kl.toUpperCase()}
          </span>
        {/each}
      </div>

      <svg viewBox="0 0 640 440" style="width:100%; max-height:440px; display:block;">
        <!-- Grid lines -->
        {#each [1,2,3,4,5,6,7,8] as i}
          <line x1={i*80} y1="0" x2={i*80} y2="440" stroke="rgba(24,32,48,0.6)" stroke-width="1"/>
          <line x1="0" y1={i*55} x2="640" y2={i*55} stroke="rgba(24,32,48,0.6)" stroke-width="1"/>
        {/each}

        <!-- Edges -->
        {#each layout.lines as l}
          <line
            x1={l.x1} y1={l.y1} x2={l.x2} y2={l.y2}
            stroke="rgba(74,96,112,0.5)" stroke-width="1"
            stroke-dasharray="4 3"
          />
        {/each}

        <!-- Nodes -->
        {#each layout.nodes as n}
          {@const col = colorFor(n.kind)}
          <g style="filter:{glowFor(n.kind)};">
            <!-- Outer ring -->
            <circle cx={n.x} cy={n.y} r="12" fill="none" stroke={col} stroke-width="1" opacity="0.4"/>
            <!-- Inner fill -->
            <circle cx={n.x} cy={n.y} r="8" fill={col} opacity="0.9"/>
            <!-- Label -->
            <text
              x={n.x} y={n.y - 16}
              text-anchor="middle"
              fill="#e8f4f8"
              font-size="9"
              font-family="'JetBrains Mono', monospace"
              letter-spacing="0.5"
            >{n.kind}:{n.label.slice(0,10)}</text>
          </g>
        {/each}
      </svg>
    {/if}
  </section>

  <!-- Edge kind breakdown -->
  {#if Object.keys(stats?.byKind ?? {}).length > 0}
    <section style="border:1px solid var(--color-base-700); background:var(--color-base-900);">
      <div style="padding:8px 16px; border-bottom:1px solid var(--color-base-700); font-family:'Rajdhani',sans-serif; font-weight:600; font-size:11px; letter-spacing:2px; text-transform:uppercase; color:#e8f4f8;">Edge Kinds</div>
      {#each Object.entries(stats?.byKind ?? {}) as [kindLabel, count]}
        <div style="display:flex; align-items:center; gap:12px; padding:7px 16px; border-bottom:1px solid rgba(24,32,48,0.8);">
          <span style="font-family:'Share Tech Mono',monospace; font-size:10px; color:var(--color-base-200); flex:1;">{kindLabel}</span>
          <span style="font-family:'JetBrains Mono',monospace; font-size:11px; color:var(--color-cyan-400);">{count}</span>
        </div>
      {/each}
    </section>
  {/if}
</div>
