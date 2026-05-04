<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import {
    siemIngest, siemSearch, siemStats, liveTail, eventGet,
    siemSearchExportUrl,
    type OblivraEvent, type EventDetail, type IngestStats,
    type Severity, type SearchResponse, type LiveTailHandle,
  } from '../bridge';
  import Tile from '../components/Tile.svelte';

  // ── Mock data (shown when backend is unreachable) ───────────────────────
  function makeMockEvent(i: number, offsetSec: number): OblivraEvent {
    const sevs: Severity[] = ['debug','info','info','info','notice','warning','error','critical'];
    const msgs = [
      'sshd: Accepted publickey for admin from 192.168.1.10 port 49221 ssh2',
      'kernel: iptables DROP IN=eth0 SRC=10.0.0.5 DST=10.0.0.1 PROTO=TCP DPT=22',
      'systemd: Started Daily apt upgrade and clean activities.',
      'sudo: admin : TTY=pts/0 ; PWD=/home/admin ; USER=root ; COMMAND=/bin/bash',
      'sshd: Failed password for root from 203.0.113.42 port 51234 ssh2',
      'kernel: Out of memory: Kill process 4821 (chrome) score 900 or sacrifice child',
      'nginx: 192.168.1.55 - - [GET /admin/config.php HTTP/1.1] 404',
      'cron: (root) CMD (/usr/lib/apt/apt.systemd.daily)',
      'auditd: type=SYSCALL msg=audit: arch=c000003e syscall=59 success=yes',
      'firewalld: ACCEPT IN=lo OUT= SRC=127.0.0.1 DST=127.0.0.1',
    ];
    const hosts = ['web-01','db-primary','edge-router','win-workstation','backup-srv'];
    const sources = ['sshd','kernel','systemd','sudo','nginx','auditd','firewalld','cron'];
    const ts = new Date(Date.now() - offsetSec * 1000).toISOString();
    const sev = sevs[i % sevs.length];
    return {
      id: `mock-${i}`,
      tenantId: 'default',
      timestamp: ts,
      receivedAt: ts,
      source: sources[i % sources.length],
      hostId: hosts[i % hosts.length],
      eventType: sources[i % sources.length],
      severity: sev,
      message: msgs[i % msgs.length],
      raw: msgs[i % msgs.length],
      fields: {},
    };
  }

  function mockResult(): SearchResponse {
    const events: OblivraEvent[] = [];
    for (let i = 0; i < 40; i++) events.push(makeMockEvent(i, i * 47 + Math.floor(Math.random()*30)));
    return { events, total: 40, took: '0ms', mode: 'chrono' };
  }

  const mockStats: IngestStats = {
    total: 1247,
    hotCount: 312,
    wal: { path: '', bytes: 8192, count: 14 },
    eps: 3,
    generatedAt: new Date().toISOString(),
  };

  let isMock = $state(false);

  // ── Stats ──────────────────────────────────────────────────────────────
  let stats     = $state<IngestStats | null>(null);
  let result    = $state<SearchResponse | null>(null);
  let busy      = $state(false);
  let error     = $state<string | null>(null);
  let selected  = $state<EventDetail | null>(null);
  let detailErr = $state<string | null>(null);

  // ── Filters ────────────────────────────────────────────────────────────
  let query         = $state('');
  let live          = $state(true);
  let limit         = $state(100);
  let severityFilter= $state<Severity | ''>('');
  let hostFilter    = $state('');
  let sourceFilter  = $state('');

  // Date range — default: last 1 hour
  let fromDT = $state('');
  let toDT   = $state('');

  function setQuickRange(minutes: number) {
    const now = new Date();
    const from = new Date(now.getTime() - minutes * 60_000);
    toDT   = toLocal(now);
    fromDT = toLocal(from);
    void refresh();
  }

  function clearRange() { fromDT = ''; toDT = ''; void refresh(); }

  // "Since OS install" — uses Windows install date via WMI approximation.
  // We fetch /api/v1/system/osinstall if available; otherwise fall back to
  // a conservative 1-year-ago anchor so the query isn't unbounded.
  async function setOsInstallRange() {
    let installDate: Date;
    try {
      const res = await fetch('/api/v1/system/osinstall', { credentials: 'same-origin' });
      if (res.ok) {
        const j = await res.json() as { installedAt?: string };
        installDate = j.installedAt ? new Date(j.installedAt) : new Date(Date.now() - 365 * 24 * 3600_000);
      } else {
        installDate = new Date(Date.now() - 365 * 24 * 3600_000);
      }
    } catch {
      installDate = new Date(Date.now() - 365 * 24 * 3600_000);
    }
    const now = new Date();
    toDT   = toLocal(now);
    fromDT = toLocal(installDate);
    void refresh();
  }

  function toLocal(d: Date): string {
    // datetime-local value format: YYYY-MM-DDTHH:mm
    const pad = (n: number) => String(n).padStart(2, '0');
    return `${d.getFullYear()}-${pad(d.getMonth()+1)}-${pad(d.getDate())}T${pad(d.getHours())}:${pad(d.getMinutes())}`;
  }

  function parseRange(): { fromUnix?: number; toUnix?: number } {
    let fromUnix: number | undefined;
    let toUnix:   number | undefined;
    if (fromDT) { const d = new Date(fromDT); if (!isNaN(d.getTime())) fromUnix = Math.floor(d.getTime()/1000); }
    if (toDT)   { const d = new Date(toDT);   if (!isNaN(d.getTime())) toUnix   = Math.floor(d.getTime()/1000); }
    return { fromUnix, toUnix };
  }

  // Build query string that includes severity/host/source chips
  function buildQuery(): string {
    const parts: string[] = [];
    if (query.trim())         parts.push(query.trim());
    if (severityFilter)       parts.push(`severity:${severityFilter}`);
    if (hostFilter.trim())    parts.push(`hostId:${hostFilter.trim()}`);
    if (sourceFilter.trim())  parts.push(`source:${sourceFilter.trim()}`);
    return parts.join(' ');
  }

  // ── Live tail ──────────────────────────────────────────────────────────
  let tail:         LiveTailHandle | null = null;
  let tailEvents    = $state<OblivraEvent[]>([]);
  let tailDropped   = $state(0);
  let timer:        ReturnType<typeof setInterval> | null = null;
  let tailPaused    = $state(false);

  // ── Fetch ──────────────────────────────────────────────────────────────
  async function refresh() {
    try {
      const [s, q] = await Promise.all([
        siemStats(),
        siemSearch({ query: buildQuery(), limit, newestFirst: true, ...parseRange() }),
      ]);
      stats  = s;
      result = q;
      error  = null;
      isMock = false;
    } catch (e) {
      // Show mock data so the panel isn't blank on first launch
      if (!result) {
        stats  = mockStats;
        result = mockResult();
        isMock = true;
        error  = null;
      } else {
        error = (e as Error).message;
      }
    }
  }

  function startTimer() {
    if (timer) clearInterval(timer);
    if (live) timer = setInterval(() => void refresh(), 5000);
  }
  function startTail() {
    if (tail) { tail.close(); tail = null; }
    if (!live) return;
    tail = liveTail(
      (ev) => {
        if (tailPaused) return;
        tailEvents = [ev, ...tailEvents].slice(0, 300);
      },
      () => {},
    );
  }

  $effect(() => { startTimer(); startTail(); });

  onMount(()    => { void refresh(); });
  onDestroy(()  => {
    if (timer) clearInterval(timer);
    if (tail)  tail.close();
  });

  async function openEvent(id: string) {
    detailErr = null;
    selected  = null;
    try   { selected = await eventGet(id); }
    catch (e) { detailErr = (e as Error).message; }
  }

  function handleKey(e: KeyboardEvent) {
    if (e.key === 'Enter') { e.preventDefault(); void refresh(); }
  }

  // ── Severity colour system ─────────────────────────────────────────────
  type SevKey = Severity | 'default';

  // Row background (very subtle tint on the full row)
  const rowBg: Record<SevKey, string> = {
    debug:    'transparent',
    info:     'transparent',
    notice:   'rgba(34,212,240,0.03)',
    warning:  'rgba(255,171,0,0.07)',
    error:    'rgba(255,61,87,0.09)',
    critical: 'rgba(255,0,51,0.14)',
    alert:    'rgba(255,0,51,0.18)',
    default:  'transparent',
  };

  // Row left border colour
  const rowBorder: Record<SevKey, string> = {
    debug:    'transparent',
    info:     'transparent',
    notice:   'var(--color-sig-info)',
    warning:  'var(--color-sig-warn)',
    error:    'var(--color-sig-error)',
    critical: '#ff0033',
    alert:    '#ff0033',
    default:  'transparent',
  };

  // Severity badge
  const sevBadge: Record<SevKey, { bg: string; fg: string; label: string }> = {
    debug:    { bg:'rgba(74,96,112,0.3)',  fg:'var(--color-base-300)', label:'DBG' },
    info:     { bg:'rgba(64,196,255,0.12)',fg:'var(--color-sig-info)', label:'INF' },
    notice:   { bg:'rgba(64,196,255,0.18)',fg:'var(--color-sig-info)', label:'NTC' },
    warning:  { bg:'rgba(255,171,0,0.18)', fg:'var(--color-sig-warn)', label:'WRN' },
    error:    { bg:'rgba(255,61,87,0.20)', fg:'var(--color-sig-error)',label:'ERR' },
    critical: { bg:'rgba(255,0,51,0.28)',  fg:'#ff0033',              label:'CRT' },
    alert:    { bg:'rgba(255,0,51,0.28)',  fg:'#ff0033',              label:'ALT' },
    default:  { bg:'rgba(74,96,112,0.3)',  fg:'var(--color-base-300)', label:'???'},
  };

  function sev(s?: string): SevKey {
    const known: SevKey[] = ['debug','info','notice','warning','error','critical','alert'];
    return (known.includes(s as SevKey) ? s : 'default') as SevKey;
  }

  // Quick filter chips
  const chips = [
    { label: 'CRITICAL',  q: '', sev: 'critical' as Severity },
    { label: 'ERROR',     q: '', sev: 'error'    as Severity },
    { label: 'WARNING',   q: '', sev: 'warning'  as Severity },
    { label: 'INFO',      q: '', sev: 'info'     as Severity },
    { label: 'DEBUG',     q: '', sev: 'debug'    as Severity },
  ];

  // Unique hosts derived from current result (for quick host buttons)
  const knownHosts = $derived.by(() => {
    const hosts = new Set<string>();
    for (const ev of result?.events ?? []) if (ev.hostId) hosts.add(ev.hostId);
    return [...hosts].slice(0, 8);
  });

  // Export URL using current filter state
  const exportUrl = (fmt: 'csv' | 'ndjson') =>
    siemSearchExportUrl({ query: buildQuery(), limit, newestFirst: true, ...parseRange() }, fmt);
</script>

<!-- ── ROOT ─────────────────────────────────────────────────────────────── -->
<div style="display:flex; flex-direction:column; height:100%; overflow:hidden;">

  <!-- ── TOP STATS ───────────────────────────────────────────────────────── -->
  <div style="padding:12px 20px 0; flex-shrink:0;">
    <div style="display:grid; grid-template-columns:repeat(4,1fr); gap:10px;">
      <Tile label="Events ingested" value={stats?.total     ?? '—'} hint="LIFETIME" />
      <Tile label="Hot store"       value={stats?.hotCount  ?? '—'} hint="BADGERDB ROWS" />
      <Tile label="EPS"             value={stats?.eps ?? 0}         hint="ROLLING 1s" />
      <Tile label="WAL"             value={stats ? `${(stats.wal.bytes/1024).toFixed(1)} KiB` : '—'}
                                    hint={`${stats?.wal.count ?? 0} ENTRIES`} />
    </div>
  </div>

  <!-- ── BODY: filter rail + log stream ──────────────────────────────────── -->
  <div style="display:flex; flex:1; overflow:hidden; gap:0; margin-top:12px;">

    <!-- FILTER RAIL ──────────────────────────────────────────────────────── -->
    <aside style="
      width:220px; flex-shrink:0;
      border-right:1px solid var(--color-base-700);
      background:var(--color-base-900);
      display:flex; flex-direction:column;
      overflow-y:auto;
      padding:12px 0;
    " class="scrollbar-thin">

      <!-- Section: time range -->
      <div style="padding:0 12px 8px;">
        <div style="font-family:'Share Tech Mono',monospace; font-size:8px; letter-spacing:3px; color:var(--color-base-300); text-transform:uppercase; margin-bottom:8px;">Time Range</div>

        <!-- Quick presets -->
        <div style="display:flex; flex-direction:column; gap:3px; margin-bottom:8px;">
          {#each [
            { label:'LAST 15m',  min:15 },
            { label:'LAST 1h',   min:60 },
            { label:'LAST 6h',   min:360 },
            { label:'LAST 24h',  min:1440 },
            { label:'LAST 7d',   min:10080 },
            { label:'LAST 30d',  min:43200 },
            { label:'LAST 60d',  min:86400 },
            { label:'LAST 90d',  min:129600 },
          ] as r}
            <button
              onclick={() => setQuickRange(r.min)}
              style="
                padding:4px 8px; border:1px solid var(--color-base-600);
                background:transparent; text-align:left;
                font-family:'Share Tech Mono',monospace; font-size:9px;
                letter-spacing:1px; color:var(--color-base-200);
                cursor:pointer; transition:all .1s;
              "
              onmouseenter={(e) => { (e.currentTarget as HTMLElement).style.borderColor='var(--color-cyan-500)'; (e.currentTarget as HTMLElement).style.color='var(--color-cyan-400)'; }}
              onmouseleave={(e) => { (e.currentTarget as HTMLElement).style.borderColor='var(--color-base-600)'; (e.currentTarget as HTMLElement).style.color='var(--color-base-200)'; }}
            >{r.label}</button>
          {/each}
          <button
            onclick={setOsInstallRange}
            style="
              padding:4px 8px; border:1px solid var(--color-base-500);
              background:rgba(0,188,216,0.04); text-align:left;
              font-family:'Share Tech Mono',monospace; font-size:9px;
              letter-spacing:1px; color:var(--color-cyan-400);
              cursor:pointer; transition:all .1s;
            "
            onmouseenter={(e) => { (e.currentTarget as HTMLElement).style.borderColor='var(--color-cyan-500)'; }}
            onmouseleave={(e) => { (e.currentTarget as HTMLElement).style.borderColor='var(--color-base-500)'; }}
          >SINCE OS INSTALL</button>
          {#if fromDT || toDT}
            <button
              onclick={clearRange}
              style="padding:4px 8px; border:1px solid var(--color-sig-warn); background:transparent; text-align:left; font-family:'Share Tech Mono',monospace; font-size:9px; letter-spacing:1px; color:var(--color-sig-warn); cursor:pointer;"
            >CLEAR RANGE</button>
          {/if}
        </div>

        <!-- FROM -->
        <div style="margin-bottom:4px;">
          <div style="font-family:'Share Tech Mono',monospace; font-size:8px; letter-spacing:1px; color:var(--color-base-300); margin-bottom:2px;">FROM</div>
          <input
            type="datetime-local"
            bind:value={fromDT}
            onchange={() => void refresh()}
            style="
              width:100%; padding:4px 6px; border:1px solid var(--color-base-600);
              background:var(--color-base-800); color:#e8f4f8;
              font-family:'JetBrains Mono',monospace; font-size:10px;
              outline:none; box-sizing:border-box;
            "
          />
        </div>

        <!-- TO -->
        <div>
          <div style="font-family:'Share Tech Mono',monospace; font-size:8px; letter-spacing:1px; color:var(--color-base-300); margin-bottom:2px;">TO</div>
          <input
            type="datetime-local"
            bind:value={toDT}
            onchange={() => void refresh()}
            style="
              width:100%; padding:4px 6px; border:1px solid var(--color-base-600);
              background:var(--color-base-800); color:#e8f4f8;
              font-family:'JetBrains Mono',monospace; font-size:10px;
              outline:none; box-sizing:border-box;
            "
          />
        </div>
      </div>

      <div style="height:1px; background:var(--color-base-700); margin:8px 0;"></div>

      <!-- Section: severity -->
      <div style="padding:0 12px 8px;">
        <div style="font-family:'Share Tech Mono',monospace; font-size:8px; letter-spacing:3px; color:var(--color-base-300); text-transform:uppercase; margin-bottom:8px;">Severity</div>

        <button
          onclick={() => { severityFilter = ''; void refresh(); }}
          style="
            display:block; width:100%; padding:4px 8px; margin-bottom:3px;
            border:1px solid {severityFilter === '' ? 'var(--color-cyan-500)' : 'var(--color-base-600)'};
            background:{severityFilter === '' ? 'rgba(0,188,216,0.10)' : 'transparent'};
            text-align:left; font-family:'Share Tech Mono',monospace; font-size:9px;
            letter-spacing:1px; color:{severityFilter === '' ? 'var(--color-cyan-400)' : 'var(--color-base-200)'};
            cursor:pointer;
          "
        >ALL LEVELS</button>

        {#each ['debug','info','notice','warning','error','critical'] as s}
          {@const badge = sevBadge[s as SevKey]}
          <button
            onclick={() => { severityFilter = s as Severity; void refresh(); }}
            style="
              display:flex; align-items:center; gap:6px;
              width:100%; padding:4px 8px; margin-bottom:3px;
              border:1px solid {severityFilter === s ? badge.fg : 'var(--color-base-600)'};
              background:{severityFilter === s ? badge.bg : 'transparent'};
              text-align:left; cursor:pointer;
            "
          >
            <span style="
              padding:1px 5px; background:{badge.bg};
              font-family:'Share Tech Mono',monospace; font-size:8px;
              letter-spacing:1px; color:{badge.fg};
            ">{badge.label}</span>
            <span style="font-family:'Share Tech Mono',monospace; font-size:9px; letter-spacing:1px; color:{severityFilter === s ? badge.fg : 'var(--color-base-200)'}; text-transform:uppercase;">{s}</span>
          </button>
        {/each}
      </div>

      <div style="height:1px; background:var(--color-base-700); margin:8px 0;"></div>

      <!-- Section: host filter -->
      <div style="padding:0 12px 8px;">
        <div style="font-family:'Share Tech Mono',monospace; font-size:8px; letter-spacing:3px; color:var(--color-base-300); text-transform:uppercase; margin-bottom:8px;">Host / Agent</div>
        <input
          bind:value={hostFilter}
          onkeydown={(e) => e.key === 'Enter' && void refresh()}
          placeholder="e.g. web-01"
          style="
            width:100%; padding:4px 6px; border:1px solid var(--color-base-600);
            background:var(--color-base-800); color:#e8f4f8;
            font-family:'JetBrains Mono',monospace; font-size:10px;
            outline:none; box-sizing:border-box; margin-bottom:6px;
          "
        />
        <!-- Known hosts from result -->
        {#if knownHosts.length > 0}
          <div style="display:flex; flex-direction:column; gap:3px;">
            {#each knownHosts as h}
              <button
                onclick={() => { hostFilter = h; void refresh(); }}
                style="
                  padding:3px 8px; border:1px solid {hostFilter===h ? 'var(--color-cyan-500)' : 'var(--color-base-600)'};
                  background:{hostFilter===h ? 'rgba(0,188,216,0.10)' : 'transparent'};
                  text-align:left; font-family:'JetBrains Mono',monospace; font-size:9px;
                  color:{hostFilter===h ? 'var(--color-cyan-400)' : 'var(--color-base-200)'};
                  cursor:pointer; overflow:hidden; text-overflow:ellipsis; white-space:nowrap;
                "
              >{h}</button>
            {/each}
          </div>
        {/if}
      </div>

      <div style="height:1px; background:var(--color-base-700); margin:8px 0;"></div>

      <!-- Section: limit -->
      <div style="padding:0 12px 8px;">
        <div style="font-family:'Share Tech Mono',monospace; font-size:8px; letter-spacing:3px; color:var(--color-base-300); text-transform:uppercase; margin-bottom:8px;">Row Limit</div>
        <select
          bind:value={limit}
          onchange={() => void refresh()}
          style="
            width:100%; padding:4px 6px; border:1px solid var(--color-base-600);
            background:var(--color-base-800); color:#e8f4f8;
            font-family:'Share Tech Mono',monospace; font-size:10px; outline:none;
          "
        >
          <option value={25}>25 rows</option>
          <option value={50}>50 rows</option>
          <option value={100}>100 rows</option>
          <option value={250}>250 rows</option>
          <option value={500}>500 rows</option>
        </select>
      </div>

      <!-- Live toggle -->
      <div style="padding:0 12px; margin-top:auto;">
        <label style="display:flex; align-items:center; gap:8px; cursor:pointer;">
          <input type="checkbox" bind:checked={live} style="accent-color:var(--color-cyan-500);" />
          <span style="font-family:'Share Tech Mono',monospace; font-size:9px; letter-spacing:1px; color:{live ? 'var(--color-sig-ok)' : 'var(--color-base-300)'}; text-transform:uppercase;">
            {#if live}
              <span class="animate-glow" style="display:inline-block;width:5px;height:5px;border-radius:50%;background:var(--color-sig-ok);margin-right:4px;"></span>
              LIVE
            {:else}
              ◉ PAUSED
            {/if}
          </span>
        </label>
      </div>
    </aside>

    <!-- LOG STREAM PANEL ──────────────────────────────────────────────────── -->
    <div style="flex:1; display:flex; flex-direction:column; overflow:hidden; min-width:0;">

      <!-- Search bar -->
      <div style="
        display:flex; align-items:center; gap:8px;
        padding:10px 16px; border-bottom:1px solid var(--color-base-700);
        background:var(--color-base-900); flex-shrink:0;
      ">
        <span style="font-family:'Share Tech Mono',monospace; font-size:11px; color:var(--color-cyan-500); letter-spacing:1px;">›_</span>
        <input
          bind:value={query}
          onkeydown={handleKey}
          placeholder='search — severity:warning, message:sshd, hostId:web-01, "Failed password"'
          style="
            flex:1; padding:5px 10px;
            border:1px solid var(--color-base-600);
            background:var(--color-base-800); color:#e8f4f8;
            font-family:'JetBrains Mono',monospace; font-size:11px;
            outline:none;
          "
        />
        <button
          onclick={() => void refresh()}
          style="
            padding:5px 14px; border:none;
            background:var(--color-cyan-500); color:#000;
            font-family:'Rajdhani',sans-serif; font-weight:700;
            font-size:12px; letter-spacing:2px; text-transform:uppercase;
            cursor:pointer;
          "
        >SEARCH</button>
        <button
          onclick={() => { query=''; severityFilter=''; hostFilter=''; sourceFilter=''; fromDT=''; toDT=''; void refresh(); }}
          style="
            padding:5px 10px; border:1px solid var(--color-base-600);
            background:transparent; color:var(--color-base-200);
            font-family:'Share Tech Mono',monospace; font-size:10px;
            letter-spacing:1px; cursor:pointer;
          "
        >CLEAR</button>

        <!-- Meta info -->
        {#if result}
          <span style="font-family:'Share Tech Mono',monospace; font-size:9px; letter-spacing:1px; color:var(--color-base-300); flex-shrink:0;">
            {result.total} HITS · {result.took} · {result.mode.toUpperCase()}
          </span>
        {/if}

        <!-- Export -->
        {#if (result?.events.length ?? 0) > 0}
          <a href={exportUrl('csv')}    download style="font-family:'Share Tech Mono',monospace; font-size:9px; letter-spacing:1px; color:var(--color-cyan-500); text-decoration:none; flex-shrink:0;" onmouseenter={(e)=>{(e.currentTarget as HTMLElement).style.color='var(--color-cyan-400)'}} onmouseleave={(e)=>{(e.currentTarget as HTMLElement).style.color='var(--color-cyan-500)'}}>CSV↓</a>
          <a href={exportUrl('ndjson')} download style="font-family:'Share Tech Mono',monospace; font-size:9px; letter-spacing:1px; color:var(--color-cyan-500); text-decoration:none; flex-shrink:0;">NDJSON↓</a>
        {/if}
      </div>

      <!-- Mock data banner -->
      {#if isMock}
        <div style="padding:5px 16px; background:rgba(255,171,0,0.10); border-bottom:1px solid var(--color-sig-warn); font-family:'Share Tech Mono',monospace; font-size:10px; letter-spacing:0.5px; color:var(--color-sig-warn); flex-shrink:0; display:flex; align-items:center; gap:8px;">
          ⚠ DEMO MODE — backend not reachable — showing sample data
        </div>
      {/if}

      <!-- Error banner -->
      {#if error}
        <div style="padding:6px 16px; background:rgba(255,61,87,0.12); border-bottom:1px solid var(--color-sig-error); font-family:'Share Tech Mono',monospace; font-size:10px; letter-spacing:0.5px; color:var(--color-sig-error); flex-shrink:0;">
          ✗ {error}
        </div>
      {/if}

      <!-- LIVE TAIL section -->
      {#if live && tailEvents.length > 0}
        <div style="flex-shrink:0; border-bottom:1px solid var(--color-base-700);">
          <div style="
            display:flex; align-items:center; justify-content:space-between;
            padding:5px 16px; background:rgba(0,188,216,0.05);
            border-bottom:1px solid rgba(0,188,216,0.15);
          ">
            <span style="display:flex; align-items:center; gap:6px; font-family:'Share Tech Mono',monospace; font-size:9px; letter-spacing:1.5px; color:var(--color-cyan-400);">
              <span class="animate-glow-cyan" style="display:inline-block;width:5px;height:5px;border-radius:50%;background:var(--color-cyan-500);"></span>
              LIVE TAIL · WebSocket · {tailEvents.length} buffered{tailDropped ? ` · ${tailDropped} DROPPED` : ''}
            </span>
            <div style="display:flex; gap:8px;">
              <button
                onclick={() => (tailPaused = !tailPaused)}
                style="font-family:'Share Tech Mono',monospace; font-size:9px; letter-spacing:1px; color:var(--color-base-200); background:transparent; border:1px solid var(--color-base-600); padding:2px 8px; cursor:pointer;"
              >{tailPaused ? '▶ RESUME' : '⏸ PAUSE'}</button>
              <button
                onclick={() => { tailEvents = []; tailDropped = 0; }}
                style="font-family:'Share Tech Mono',monospace; font-size:9px; letter-spacing:1px; color:var(--color-base-300); background:transparent; border:none; cursor:pointer;"
              >CLEAR</button>
            </div>
          </div>

          <!-- Live tail rows — max 15 shown, scrollable -->
          <div style="max-height:160px; overflow-y:auto; background:rgba(6,10,15,0.6);" class="scrollbar-thin">
            {#each tailEvents.slice(0, 30) as ev (ev.id)}
              {@const s = sev(ev.severity)}
              <div style="
                display:flex; align-items:flex-start; gap:0;
                padding:3px 0; border-left:2px solid {rowBorder[s]};
                background:{rowBg[s]};
                border-bottom:1px solid rgba(24,32,48,0.5);
              ">
                <!-- Timestamp -->
                <span style="font-family:'JetBrains Mono',monospace; font-size:10px; color:var(--color-base-300); padding:0 10px; flex-shrink:0; white-space:nowrap;">
                  {new Date(ev.timestamp).toLocaleTimeString()}
                </span>

                <!-- Severity badge -->
                <span style="
                  padding:1px 5px; margin-right:8px; flex-shrink:0;
                  background:{sevBadge[s].bg}; font-family:'Share Tech Mono',monospace;
                  font-size:8px; letter-spacing:1px; color:{sevBadge[s].fg};
                ">{sevBadge[s].label}</span>

                <!-- Host -->
                {#if ev.hostId}
                  <span style="font-family:'JetBrains Mono',monospace; font-size:10px; color:var(--color-base-200); margin-right:8px; flex-shrink:0; max-width:100px; overflow:hidden; text-overflow:ellipsis; white-space:nowrap;">{ev.hostId}</span>
                {/if}

                <!-- Message -->
                <span style="font-family:'JetBrains Mono',monospace; font-size:10px; color:#e8f4f8; flex:1; white-space:nowrap; overflow:hidden; text-overflow:ellipsis; padding-right:10px;">{ev.message}</span>
              </div>
            {/each}
          </div>
        </div>
      {/if}

      <!-- HISTORICAL LOG TABLE ───────────────────────────────────────────── -->
      <div style="flex:1; overflow:hidden; display:flex; flex-direction:column;">

        <!-- Table header -->
        <div style="
          display:grid; align-items:center;
          grid-template-columns: 85px 46px 100px 90px 1fr;
          padding:4px 10px; border-bottom:1px solid var(--color-base-600);
          background:var(--color-base-850); flex-shrink:0;
          font-family:'Share Tech Mono',monospace; font-size:8px;
          letter-spacing:2px; color:var(--color-base-300); text-transform:uppercase;
          gap:8px;
        ">
          <span>TIME</span>
          <span>SEV</span>
          <span>HOST</span>
          <span>TYPE</span>
          <span>MESSAGE</span>
        </div>

        <!-- Rows -->
        <div style="flex:1; overflow-y:auto;" class="scrollbar-thin">
          {#if !result || result.events.length === 0}
            <div style="
              display:flex; flex-direction:column; align-items:center; justify-content:center;
              height:200px; gap:12px;
            ">
              <span style="font-family:'Share Tech Mono',monospace; font-size:9px; letter-spacing:2px; color:var(--color-base-300);">
                {query || severityFilter || hostFilter || fromDT ? '— NO EVENTS MATCH CURRENT FILTERS —' : isMock ? '— MOCK DATA — BACKEND UNREACHABLE — START THE SERVER TO SEE REAL EVENTS —' : '— NO EVENTS YET — SEND SOME LOGS OR USE THE PROBE BELOW —'}
              </span>
            </div>
          {:else}
            {#each result.events as ev (ev.id)}
              {@const s = sev(ev.severity)}
              {@const isSelected = selected?.event.id === ev.id}
              <button
                type="button"
                onclick={() => openEvent(ev.id)}
                style="
                  display:grid; width:100%; text-align:left;
                  grid-template-columns: 85px 46px 100px 90px 1fr;
                  align-items:center; gap:8px;
                  padding:5px 10px;
                  border-left:2px solid {isSelected ? 'var(--color-cyan-500)' : rowBorder[s]};
                  background:{isSelected ? 'rgba(0,188,216,0.08)' : rowBg[s]};
                  border-bottom:1px solid rgba(24,32,48,0.6);
                  cursor:pointer; transition:background .08s;
                "
                onmouseenter={(e) => { if (!isSelected) (e.currentTarget as HTMLElement).style.background = s === 'default' || s === 'info' || s === 'debug' ? 'rgba(24,32,48,0.8)' : rowBg[s]; }}
                onmouseleave={(e) => { if (!isSelected) (e.currentTarget as HTMLElement).style.background = rowBg[s]; }}
              >
                <!-- Timestamp -->
                <span style="font-family:'JetBrains Mono',monospace; font-size:10px; color:var(--color-base-300); white-space:nowrap;">
                  {new Date(ev.timestamp).toLocaleTimeString()}
                </span>

                <!-- Severity badge -->
                <span style="
                  padding:1px 5px; display:inline-block;
                  background:{sevBadge[s].bg}; font-family:'Share Tech Mono',monospace;
                  font-size:8px; letter-spacing:1px; color:{sevBadge[s].fg};
                  white-space:nowrap;
                ">{sevBadge[s].label}</span>

                <!-- Host -->
                <span style="font-family:'JetBrains Mono',monospace; font-size:10px; color:var(--color-base-200); white-space:nowrap; overflow:hidden; text-overflow:ellipsis;">{ev.hostId ?? '—'}</span>

                <!-- Event type -->
                <span style="font-family:'JetBrains Mono',monospace; font-size:10px; color:var(--color-base-300); white-space:nowrap; overflow:hidden; text-overflow:ellipsis;">{ev.eventType ?? ev.source ?? '—'}</span>

                <!-- Message -->
                <span style="font-family:'JetBrains Mono',monospace; font-size:10px; color:#e8f4f8; white-space:nowrap; overflow:hidden; text-overflow:ellipsis;">{ev.message}</span>
              </button>
            {/each}
          {/if}
        </div>
      </div>
    </div>

    <!-- EVENT DETAIL PANEL ───────────────────────────────────────────────── -->
    {#if selected || detailErr}
      <div style="
        width:360px; flex-shrink:0;
        border-left:1px solid var(--color-cyan-500);
        background:var(--color-base-900);
        display:flex; flex-direction:column;
        overflow:hidden;
      ">
        <!-- Panel header -->
        <div style="
          display:flex; align-items:center; justify-content:space-between;
          padding:8px 14px; border-bottom:1px solid var(--color-base-700);
          flex-shrink:0;
        ">
          <span style="font-family:'Rajdhani',sans-serif; font-weight:700; font-size:13px; letter-spacing:2px; text-transform:uppercase; color:var(--color-cyan-400);">Event Detail</span>
          <button
            onclick={() => { selected = null; detailErr = null; }}
            style="background:none; border:none; color:var(--color-base-300); font-size:16px; cursor:pointer; padding:0 4px; line-height:1;"
          >✕</button>
        </div>

        <div style="flex:1; overflow-y:auto; padding:14px;" class="scrollbar-thin">
          {#if detailErr}
            <p style="font-family:'Share Tech Mono',monospace; font-size:10px; letter-spacing:0.5px; color:var(--color-sig-error);">✗ {detailErr}</p>
          {:else if selected}
            {@const ev = selected.event}
            {@const s = sev(ev.severity)}

            <!-- Severity highlight strip -->
            <div style="
              padding:6px 10px; margin-bottom:12px;
              border-left:3px solid {rowBorder[s]};
              background:{rowBg[s]};
              font-family:'Share Tech Mono',monospace; font-size:9px;
              letter-spacing:1px; color:{sevBadge[s].fg};
            ">{ev.severity?.toUpperCase() ?? 'UNKNOWN'}</div>

            <!-- KV grid -->
            {#each [
              { k:'ID',         v: ev.id },
              { k:'TIMESTAMP',  v: new Date(ev.timestamp).toISOString() },
              { k:'RECEIVED',   v: new Date(ev.receivedAt).toISOString() },
              { k:'HOST',       v: ev.hostId ?? '—' },
              { k:'SOURCE',     v: ev.source },
              { k:'EVENT TYPE', v: ev.eventType ?? '—' },
              { k:'TENANT',     v: ev.tenantId },
            ] as row}
              <div style="margin-bottom:8px;">
                <div style="font-family:'Share Tech Mono',monospace; font-size:8px; letter-spacing:2px; color:var(--color-base-300); margin-bottom:2px;">{row.k}</div>
                <div style="font-family:'JetBrains Mono',monospace; font-size:10px; color:#e8f4f8; word-break:break-all;">{row.v}</div>
              </div>
            {/each}

            <div style="height:1px; background:var(--color-base-700); margin:10px 0;"></div>

            <!-- Message -->
            <div style="margin-bottom:10px;">
              <div style="font-family:'Share Tech Mono',monospace; font-size:8px; letter-spacing:2px; color:var(--color-base-300); margin-bottom:4px;">MESSAGE</div>
              <div style="font-family:'JetBrains Mono',monospace; font-size:10px; color:#e8f4f8; white-space:pre-wrap; word-break:break-words; line-height:1.5; padding:8px; background:var(--color-base-800);">{ev.message}</div>
            </div>

            {#if ev.raw && ev.raw !== ev.message}
              <div style="margin-bottom:10px;">
                <div style="font-family:'Share Tech Mono',monospace; font-size:8px; letter-spacing:2px; color:var(--color-base-300); margin-bottom:4px;">RAW</div>
                <div style="font-family:'JetBrains Mono',monospace; font-size:9px; color:var(--color-base-200); white-space:pre-wrap; word-break:break-words; line-height:1.5; padding:8px; background:var(--color-base-800);">{ev.raw}</div>
              </div>
            {/if}

            <!-- Fields -->
            {#if ev.fields && Object.keys(ev.fields).length > 0}
              <div style="height:1px; background:var(--color-base-700); margin:10px 0;"></div>
              <div style="font-family:'Share Tech Mono',monospace; font-size:8px; letter-spacing:2px; color:var(--color-base-300); margin-bottom:6px;">FIELDS ({Object.keys(ev.fields).length})</div>
              {#each Object.entries(ev.fields) as [k, v]}
                <div style="display:flex; gap:6px; margin-bottom:4px; font-family:'JetBrains Mono',monospace; font-size:9px;">
                  <span style="color:var(--color-base-300); flex-shrink:0;">{k}:</span>
                  <span style="color:#e8f4f8; word-break:break-all;">{v}</span>
                </div>
              {/each}
            {/if}

            <!-- Related events -->
            {#if selected.related.length > 0}
              <div style="height:1px; background:var(--color-base-700); margin:10px 0;"></div>
              <div style="font-family:'Share Tech Mono',monospace; font-size:8px; letter-spacing:2px; color:var(--color-base-300); margin-bottom:6px;">RELATED ON HOST ±60s ({selected.related.length})</div>
              {#each selected.related as r (r.id)}
                {@const rs = sev(r.severity)}
                <button
                  type="button"
                  onclick={() => openEvent(r.id)}
                  style="
                    display:flex; width:100%; gap:6px; align-items:flex-start; text-align:left;
                    padding:4px 6px; margin-bottom:2px;
                    border:1px solid var(--color-base-700); background:transparent;
                    cursor:pointer; border-left:2px solid {rowBorder[rs]};
                  "
                  onmouseenter={(e) => { (e.currentTarget as HTMLElement).style.background='var(--color-base-800)'; }}
                  onmouseleave={(e) => { (e.currentTarget as HTMLElement).style.background='transparent'; }}
                >
                  <span style="font-family:'JetBrains Mono',monospace; font-size:9px; color:var(--color-base-300); flex-shrink:0;">{new Date(r.timestamp).toLocaleTimeString()}</span>
                  <span style="font-family:'JetBrains Mono',monospace; font-size:9px; color:#e8f4f8; overflow:hidden; text-overflow:ellipsis; white-space:nowrap; flex:1;">{r.message}</span>
                </button>
              {/each}
            {/if}
          {/if}
        </div>
      </div>
    {/if}
  </div>
</div>
