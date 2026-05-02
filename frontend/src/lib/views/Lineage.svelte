<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { lineageHosts, lineageTree, type LineageNode, type LineageTree } from '../bridge';
  import Tile from '../components/Tile.svelte';

  let hosts = $state<string[]>([]);
  let host = $state<string>('');
  let tree = $state<LineageTree | null>(null);
  let loadError = $state<string | null>(null);
  let timer: ReturnType<typeof setInterval> | null = null;
  let inFlight = 0;

  // Layout state — tidy-tree positions computed from the parent/child edges.
  type Laid = LineageNode & { x: number; y: number; depth: number };
  let nodes = $state<Laid[]>([]);
  let edges = $state<{ from: Laid; to: Laid }[]>([]);

  let hovered = $state<Laid | null>(null);

  async function refreshHosts() {
    try {
      hosts = await lineageHosts();
      if (!host && hosts.length > 0) host = hosts[0];
    } catch (err) {
      loadError = (err as Error).message;
    }
  }

  async function refreshTree() {
    if (!host) {
      tree = null;
      nodes = [];
      edges = [];
      return;
    }
    const seq = ++inFlight;
    try {
      const t = await lineageTree(host);
      if (seq !== inFlight) return;
      tree = t;
      [nodes, edges] = layoutTidyTree(t.nodes ?? []);
      loadError = null;
    } catch (err) {
      if (seq !== inFlight) return;
      loadError = (err as Error).message;
    }
  }

  onMount(async () => {
    await refreshHosts();
    await refreshTree();
    timer = setInterval(() => void refreshTree(), 5000);
  });
  onDestroy(() => { if (timer) clearInterval(timer); });

  $effect(() => { void host; void refreshTree(); });

  // ---- Tidy-tree layout ----
  // Pure-functional placement: roots are nodes whose parent isn't in the
  // set. Each subtree gets a width equal to its leaf count; siblings
  // pack horizontally from the leftmost subtree forward. Depth × 80px
  // gives the y. Output is positions in a virtual coordinate space; the
  // viewport SVG scales to fit.
  function layoutTidyTree(input: LineageNode[]): [Laid[], { from: Laid; to: Laid }[]] {
    if (input.length === 0) return [[], []];

    // Index by pid; parents who aren't in the set become roots.
    const byPid = new Map<number, LineageNode>();
    for (const n of input) byPid.set(n.pid, n);

    const childrenOf = new Map<number, LineageNode[]>();
    for (const n of input) {
      const parentPid = byPid.has(n.ppid) ? n.ppid : -1;
      const list = childrenOf.get(parentPid) ?? [];
      list.push(n);
      childrenOf.set(parentPid, list);
    }
    for (const [, list] of childrenOf) {
      list.sort((a, b) => (a.firstSeen < b.firstSeen ? -1 : 1));
    }

    const NODE_W = 200;
    const NODE_H = 64;
    const X_GAP = 12;
    const Y_GAP = 24;

    // Walk subtree to compute leaf count.
    const leafCount = new Map<number, number>();
    const computeLeaves = (pid: number): number => {
      const kids = childrenOf.get(pid) ?? [];
      if (kids.length === 0) {
        leafCount.set(pid, 1);
        return 1;
      }
      let total = 0;
      for (const k of kids) total += computeLeaves(k.pid);
      leafCount.set(pid, total);
      return total;
    };

    // Place each node by recursing over the children of -1 (the synthetic
    // root). Each subtree gets its own horizontal slot.
    const out: Laid[] = [];
    const roots = childrenOf.get(-1) ?? [];
    let cursor = 0;
    const place = (n: LineageNode, depth: number, leftEdge: number): number => {
      const span = (leafCount.get(n.pid) ?? 1) * (NODE_W + X_GAP);
      const x = leftEdge + span / 2 - NODE_W / 2;
      const y = depth * (NODE_H + Y_GAP);
      out.push({ ...n, x, y, depth });
      const kids = childrenOf.get(n.pid) ?? [];
      let cur = leftEdge;
      for (const k of kids) {
        const childSpan = (leafCount.get(k.pid) ?? 1) * (NODE_W + X_GAP);
        place(k, depth + 1, cur);
        cur += childSpan;
      }
      return span;
    };
    for (const r of roots) {
      computeLeaves(r.pid);
      const span = (leafCount.get(r.pid) ?? 1) * (NODE_W + X_GAP);
      place(r, 0, cursor);
      cursor += span;
    }

    // Build edges from each child to its parent (only when both placed).
    const positions = new Map<number, Laid>();
    for (const p of out) positions.set(p.pid, p);
    const edges: { from: Laid; to: Laid }[] = [];
    for (const n of out) {
      const parent = positions.get(n.ppid);
      if (parent) edges.push({ from: parent, to: n });
    }
    return [out, edges];
  }

  // Visual width/height for the SVG viewBox. Pad a bit so cards don't clip.
  const viewBox = $derived.by(() => {
    if (nodes.length === 0) return '0 0 800 200';
    let maxX = 0, maxY = 0;
    for (const n of nodes) {
      if (n.x + 200 > maxX) maxX = n.x + 200;
      if (n.y + 64 > maxY) maxY = n.y + 64;
    }
    return `0 0 ${maxX + 24} ${maxY + 24}`;
  });

  function relTime(ts: string): string {
    const sec = Math.floor((Date.now() - new Date(ts).getTime()) / 1000);
    if (sec < 60) return `${sec}s ago`;
    if (sec < 3600) return `${Math.floor(sec / 60)}m ago`;
    if (sec < 86400) return `${Math.floor(sec / 3600)}h ago`;
    return `${Math.floor(sec / 86400)}d ago`;
  }

  // Heuristic suspiciousness — flagged nodes get a red border so an
  // analyst's eye lands on them first. Not a formal detection; matches
  // the cmdline-suspicious rule pack's substring list.
  const SUSPICIOUS_RE = /\b(powershell .* -enc|comsvcs.dll.*MiniDump|mshta http|rundll32 .*\\Temp\\|certutil .* -urlcache|bitsadmin.*\/transfer|vssadmin delete shadows|wbadmin delete catalog|whoami \/all|net user .* \/add|reg add .*\\Run\b)/i;
  function suspicious(n: LineageNode): boolean {
    if (!n.command) return false;
    return SUSPICIOUS_RE.test(n.command);
  }
</script>

<div class="mx-auto max-w-7xl space-y-6 p-8">
  <header class="flex items-baseline justify-between">
    <div>
      <p class="text-xs uppercase tracking-widest text-night-300">Process lineage</p>
      <h2 class="text-2xl font-semibold tracking-tight">Parent / child execution graph</h2>
    </div>
    <select bind:value={host} class="rounded-md border border-night-600 bg-night-800/70 px-3 py-1.5 text-xs text-slate-100">
      {#each hosts as h}
        <option value={h}>{h}</option>
      {/each}
    </select>
  </header>

  <section class="grid grid-cols-2 gap-4 lg:grid-cols-4">
    <Tile label="Hosts with lineage" value={hosts.length} />
    <Tile label="Processes" value={tree?.nodes.length ?? '—'} hint={host || 'no host'} />
    <Tile label="Suspicious" value={(tree?.nodes ?? []).filter(suspicious).length} />
    <Tile label="Roots" value={(tree?.nodes ?? []).filter((n) => !(tree?.nodes ?? []).some((p) => p.pid === n.ppid)).length} />
  </section>

  {#if loadError}<p class="text-xs text-signal-error">Failed to load: {loadError}</p>{/if}

  {#if !host}
    <div class="rounded-xl border border-night-700 bg-night-900/70 p-12 text-center text-sm text-night-300">
      No hosts have process-lineage data yet.<br />
      <span class="text-night-400">Lineage builds from process_creation events flowing through the agent or syslog.</span>
    </div>
  {:else if nodes.length === 0}
    <div class="rounded-xl border border-night-700 bg-night-900/70 p-12 text-center text-sm text-night-300">
      No processes recorded for {host}.
    </div>
  {:else}
    <section class="overflow-x-auto rounded-xl border border-night-700 bg-night-900/70 p-2">
      <svg viewBox={viewBox} class="min-w-full" style:height="calc(100vh - 22rem)" preserveAspectRatio="xMinYMin meet">
        <!-- Edges first so nodes draw on top. -->
        {#each edges as e}
          <path
            d={`M ${e.from.x + 100} ${e.from.y + 64} C ${e.from.x + 100} ${e.from.y + 64 + 32}, ${e.to.x + 100} ${e.to.y - 32}, ${e.to.x + 100} ${e.to.y}`}
            stroke="rgb(120 130 145 / 0.65)"
            stroke-width="1.5"
            fill="none"
          />
        {/each}
        <!-- Nodes. -->
        {#each nodes as n (n.pid)}
          {@const sus = suspicious(n)}
          <g
            class="cursor-pointer"
            role="button"
            tabindex="0"
            aria-label="Process {n.name ?? n.pid}, pid {n.pid}"
            onmouseenter={() => (hovered = n)}
            onmouseleave={() => (hovered = null)}
            onfocus={() => (hovered = n)}
            onblur={() => (hovered = null)}
          >
            <rect
              x={n.x}
              y={n.y}
              width="200"
              height="64"
              rx="6"
              fill={sus ? 'rgb(220 68 68 / 0.15)' : 'rgb(20 24 33 / 0.95)'}
              stroke={sus ? 'rgb(220 68 68 / 0.7)' : 'rgb(80 95 120 / 0.7)'}
              stroke-width="1.2"
            />
            <text x={n.x + 10} y={n.y + 18} fill="rgb(220 224 232)" font-size="11" font-family="monospace">
              {n.name || `pid ${n.pid}`}
            </text>
            <text x={n.x + 10} y={n.y + 34} fill="rgb(150 160 175)" font-size="10" font-family="monospace">
              pid {n.pid} · {n.events} ev
            </text>
            <text x={n.x + 10} y={n.y + 52} fill="rgb(150 160 175)" font-size="10" font-family="monospace">
              {(n.command ?? '').slice(0, 30)}{(n.command ?? '').length > 30 ? '…' : ''}
            </text>
          </g>
        {/each}
      </svg>
    </section>

    {#if hovered}
      <section class="rounded-xl border border-accent-500/40 bg-night-900/70 p-4 text-xs">
        <div class="grid gap-3 sm:grid-cols-3">
          <div><div class="text-night-300">PID / PPID</div><div class="font-mono text-slate-100">{hovered.pid} / {hovered.ppid}</div></div>
          <div><div class="text-night-300">Name</div><div class="font-mono text-slate-100">{hovered.name ?? '—'}</div></div>
          <div><div class="text-night-300">Events</div><div class="font-mono text-slate-100">{hovered.events}</div></div>
          <div class="sm:col-span-3"><div class="text-night-300">Command</div><div class="font-mono text-slate-100 break-words">{hovered.command ?? '—'}</div></div>
          <div><div class="text-night-300">First seen</div><div class="font-mono text-slate-100">{relTime(hovered.firstSeen)}</div></div>
          <div><div class="text-night-300">Last seen</div><div class="font-mono text-slate-100">{relTime(hovered.lastSeen)}</div></div>
          <div><div class="text-night-300">Suspicious</div><div class="font-mono {suspicious(hovered) ? 'text-signal-error' : 'text-slate-100'}">{suspicious(hovered) ? 'yes (cmdline pattern)' : 'no'}</div></div>
        </div>
      </section>
    {/if}
  {/if}
</div>
