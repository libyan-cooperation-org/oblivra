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

  type Laid = LineageNode & { x: number; y: number; depth: number };
  let nodes = $state<Laid[]>([]);
  let edges = $state<{ from: Laid; to: Laid }[]>([]);
  let hovered = $state<Laid | null>(null);

  async function refreshHosts() {
    try {
      hosts = await lineageHosts();
      if (!host && hosts.length > 0) host = hosts[0];
    } catch (err) { loadError = (err as Error).message; }
  }

  async function refreshTree() {
    if (!host) { tree = null; nodes = []; edges = []; return; }
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

  function layoutTidyTree(input: LineageNode[]): [Laid[], { from: Laid; to: Laid }[]] {
    if (input.length === 0) return [[], []];
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
    const NODE_W = 200, NODE_H = 64, X_GAP = 12, Y_GAP = 24;
    const leafCount = new Map<number, number>();
    const computeLeaves = (pid: number): number => {
      const kids = childrenOf.get(pid) ?? [];
      if (kids.length === 0) { leafCount.set(pid, 1); return 1; }
      let total = 0;
      for (const k of kids) total += computeLeaves(k.pid);
      leafCount.set(pid, total);
      return total;
    };
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
    const positions = new Map<number, Laid>();
    for (const p of out) positions.set(p.pid, p);
    const edgesOut: { from: Laid; to: Laid }[] = [];
    for (const n of out) {
      const parent = positions.get(n.ppid);
      if (parent) edgesOut.push({ from: parent, to: n });
    }
    return [out, edgesOut];
  }

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

  const SUSPICIOUS_RE = /\b(powershell .* -enc|comsvcs\.dll.*MiniDump|mshta http|rundll32 .*\\Temp\\|certutil .* -urlcache|bitsadmin.*\/transfer|vssadmin delete shadows|wbadmin delete catalog|whoami \/all|net user .* \/add|reg add .*\\Run\b)/i;
  function suspicious(n: LineageNode): boolean {
    return n.command ? SUSPICIOUS_RE.test(n.command) : false;
  }
</script>

<div class="mx-auto max-w-7xl space-y-6 p-8">
  <header class="flex items-baseline justify-between">
    <div>
      <p class="text-xs uppercase tracking-widest text-night-300">Process Lineage</p>
      <h2 class="text-2xl font-semibold tracking-tight">Parent / child execution graph</h2>
    </div>
    <select bind:value={host} style="padding:5px 10px; border:1px solid var(--color-base-600); background:var(--color-base-800); color:#e8f4f8; font-family:'Share Tech Mono',monospace; font-size:10px; letter-spacing:1px; outline:none;">
      {#each hosts as h}<option value={h}>{h}</option>{/each}
    </select>
  </header>

  <section class="grid grid-cols-2 gap-4 lg:grid-cols-4">
    <Tile label="Hosts with lineage" value={hosts.length} />
    <Tile label="Processes"  value={tree?.nodes.length ?? '—'} hint={host || 'no host'} />
    <Tile label="Suspicious" value={(tree?.nodes ?? []).filter(suspicious).length} warn />
    <Tile label="Roots"      value={(tree?.nodes ?? []).filter((n) => !(tree?.nodes ?? []).some((p) => p.pid === n.ppid)).length} />
  </section>

  {#if loadError}<p class="text-xs text-signal-error">Failed to load: {loadError}</p>{/if}

  {#if !host}
    <div style="border:1px solid var(--color-base-700); background:var(--color-base-900); padding:48px 24px; text-align:center; font-family:'Share Tech Mono',monospace; font-size:10px; letter-spacing:2px; color:var(--color-base-300);">
      NO HOSTS HAVE PROCESS-LINEAGE DATA YET<br />
      <span style="color:var(--color-base-400); letter-spacing:1px;">Lineage builds from process_creation events via the agent or syslog.</span>
    </div>
  {:else if nodes.length === 0}
    <div style="border:1px solid var(--color-base-700); background:var(--color-base-900); padding:48px 24px; text-align:center; font-family:'Share Tech Mono',monospace; font-size:10px; letter-spacing:2px; color:var(--color-base-300);">
      NO PROCESSES RECORDED FOR {host.toUpperCase()}
    </div>
  {:else}
    <section style="overflow-x:auto; border:1px solid var(--color-base-700); background:var(--color-base-900); padding:8px;">
      <svg viewBox={viewBox} style="min-width:100%; height:calc(100vh - 22rem);" preserveAspectRatio="xMinYMin meet">
        <!-- Bezier edges -->
        {#each edges as e}
          <path
            d={`M ${e.from.x + 100} ${e.from.y + 64} C ${e.from.x + 100} ${e.from.y + 96}, ${e.to.x + 100} ${e.to.y - 32}, ${e.to.x + 100} ${e.to.y}`}
            stroke="rgba(78,105,135,0.55)"
            stroke-width="1.5"
            fill="none"
            stroke-dasharray="5 3"
          />
        {/each}

        <!-- Nodes -->
        {#each nodes as n (n.pid)}
          {@const sus = suspicious(n)}
          <g
            role="button"
            tabindex="0"
            aria-label="Process {n.name ?? n.pid}"
            style="cursor:pointer;"
            onmouseenter={() => (hovered = n)}
            onmouseleave={() => (hovered = null)}
            onfocus={() => (hovered = n)}
            onblur={() => (hovered = null)}
          >
            <!-- Card background -->
            <rect x={n.x} y={n.y} width="200" height="64" rx="0"
              fill={sus ? 'rgba(255,61,87,0.10)' : 'rgba(11,16,23,0.97)'}
              stroke={sus ? 'rgba(255,61,87,0.65)' : 'rgba(30,42,61,0.9)'}
              stroke-width="1.2"
            />
            <!-- Top accent bar -->
            <line x1={n.x} y1={n.y} x2={n.x+200} y2={n.y}
              stroke={sus ? 'rgba(255,61,87,0.7)' : 'rgba(0,188,216,0.3)'}
              stroke-width="2"
            />
            <!-- Corner brackets (top-left) -->
            <polyline points="{n.x+2},{n.y+10} {n.x+2},{n.y+2} {n.x+10},{n.y+2}"
              fill="none" stroke={sus ? 'rgba(255,61,87,0.6)' : 'rgba(0,188,216,0.5)'} stroke-width="1"/>
            <!-- Corner brackets (bottom-right) -->
            <polyline points="{n.x+190},{n.y+62} {n.x+198},{n.y+62} {n.x+198},{n.y+54}"
              fill="none" stroke={sus ? 'rgba(255,61,87,0.4)' : 'rgba(0,188,216,0.3)'} stroke-width="1"/>

            <!-- Process name -->
            <text x={n.x+10} y={n.y+18}
              fill={sus ? '#ff3d57' : '#e8f4f8'}
              font-size="11" font-family="'Rajdhani',sans-serif" font-weight="600" letter-spacing="0.5">
              {n.name || `pid ${n.pid}`}
            </text>
            <!-- PID line -->
            <text x={n.x+10} y={n.y+33}
              fill="#4d6680" font-size="9" font-family="'JetBrains Mono',monospace" letter-spacing="0.3">
              pid {n.pid} · ppid {n.ppid} · {n.events} ev
            </text>
            <!-- Command snippet -->
            <text x={n.x+10} y={n.y+50}
              fill={sus ? 'rgba(255,61,87,0.8)' : '#7a95b0'}
              font-size="9" font-family="'JetBrains Mono',monospace">
              {(n.command ?? '').slice(0, 28)}{(n.command ?? '').length > 28 ? '…' : ''}
            </text>
          </g>
        {/each}
      </svg>
    </section>

    <!-- Hover detail -->
    {#if hovered}
      <section style="border:1px solid rgba(0,188,216,0.4); background:var(--color-base-900); padding:16px;">
        <div style="display:grid; grid-template-columns:repeat(3,1fr); gap:12px; font-size:11px;">
          {#each [
            {k:'PID / PPID', v:`${hovered.pid} / ${hovered.ppid}`},
            {k:'NAME',       v:hovered.name ?? '—'},
            {k:'EVENTS',     v:String(hovered.events)},
          ] as row}
            <div>
              <div style="font-family:'Share Tech Mono',monospace; font-size:8px; letter-spacing:2px; color:var(--color-base-300); margin-bottom:3px;">{row.k}</div>
              <div style="font-family:'JetBrains Mono',monospace; font-size:11px; color:#e8f4f8;">{row.v}</div>
            </div>
          {/each}
          <div style="grid-column:1/-1;">
            <div style="font-family:'Share Tech Mono',monospace; font-size:8px; letter-spacing:2px; color:var(--color-base-300); margin-bottom:3px;">COMMAND</div>
            <div style="font-family:'JetBrains Mono',monospace; font-size:10px; color:{suspicious(hovered) ? 'var(--color-sig-error)' : '#e8f4f8'}; word-break:break-all; padding:6px 8px; background:var(--color-base-800);">{hovered.command ?? '—'}</div>
          </div>
          {#each [
            {k:'FIRST SEEN', v:relTime(hovered.firstSeen)},
            {k:'LAST SEEN',  v:relTime(hovered.lastSeen)},
            {k:'SUSPICIOUS', v:suspicious(hovered) ? 'YES — CMDLINE PATTERN' : 'CLEAN'},
          ] as row}
            <div>
              <div style="font-family:'Share Tech Mono',monospace; font-size:8px; letter-spacing:2px; color:var(--color-base-300); margin-bottom:3px;">{row.k}</div>
              <div style="font-family:'JetBrains Mono',monospace; font-size:11px; color:{row.k === 'SUSPICIOUS' && suspicious(hovered) ? 'var(--color-sig-error)' : '#e8f4f8'};">{row.v}</div>
            </div>
          {/each}
        </div>
      </section>
    {/if}
  {/if}
</div>
