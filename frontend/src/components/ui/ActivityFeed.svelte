<!--
  ActivityFeed — global "what's happening RIGHT NOW" widget.

  Phase 30.4b: closes the "real-time activity feed" gap from the audit.
  Drops onto Dashboard, OpsCenter, or any page that wants a glanceable
  pulse of the platform.

  What it shows:
    - New alerts (severity-coloured)
    - Agent status changes (online ↔ offline)
    - Critical / suspicious events

  Sources:
    - alertStore (already live-streams via subscribe('security.alert')
      and subscribe('detection.match'))
    - agentStore (refreshes every 10s; we diff against the previous
      snapshot to detect transitions)

  Render budget: 30 most-recent entries, sorted DESC. Scrollable.
  Click on an entry pivots to the relevant page (host or alert).
-->
<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { AlertTriangle, ServerCog, Zap, Activity, Circle } from 'lucide-svelte';
  import { alertStore } from '@lib/stores/alerts.svelte';
  import { agentStore } from '@lib/stores/agent.svelte';
  import { push } from '@lib/router.svelte';

  interface Props {
    /** Optional cap on rendered entries. Default 30. */
    limit?: number;
    /** Hide the header strip if embedded inside a card that has its own. */
    bare?: boolean;
  }
  let { limit = 30, bare = false }: Props = $props();

  type Entry = {
    id: string;
    ts: string;
    kind: 'alert' | 'agent-online' | 'agent-offline' | 'event';
    severity?: 'debug' | 'info' | 'warn' | 'error' | 'critical';
    title: string;
    detail?: string;
    target?: { route: string };
  };

  let entries = $state<Entry[]>([]);

  // ── Snapshot the agent statuses so we can diff on each refresh ───
  let lastAgentStatus = new Map<string, string>();
  let pollHandle: ReturnType<typeof setInterval> | null = null;

  function pushEntry(e: Entry) {
    entries = [e, ...entries].slice(0, limit);
  }

  function severityForAlert(s: string): Entry['severity'] {
    const v = (s ?? '').toLowerCase();
    if (v === 'critical') return 'critical';
    if (v === 'high' || v === 'error') return 'error';
    if (v === 'medium' || v === 'warning' || v === 'warn') return 'warn';
    if (v === 'low' || v === 'info') return 'info';
    if (v === 'debug') return 'debug';
    return 'info';
  }

  // Watch alertStore for new alerts. We use $effect to react to changes
  // in `alertStore.alerts.length` — every time the array grows we add
  // the new alert (front of the array per `init()`'s `[newAlert, ...]`
  // pattern) to the feed.
  let lastAlertCount = 0;
  $effect(() => {
    const cur = alertStore.alerts.length;
    if (cur > lastAlertCount) {
      const fresh = alertStore.alerts.slice(0, cur - lastAlertCount);
      for (const a of fresh) {
        pushEntry({
          id: `alert-${a.id}-${a.timestamp}`,
          ts: a.timestamp,
          kind: 'alert',
          severity: severityForAlert(a.severity),
          title: a.title,
          detail: a.host !== 'unknown' ? `host: ${a.host}` : a.description,
          target: a.host !== 'unknown' && a.host !== 'remote'
            ? { route: `/host/${encodeURIComponent(a.host)}?alert=${encodeURIComponent(a.id)}` }
            : undefined,
        });
      }
    }
    lastAlertCount = cur;
  });

  // Agent online/offline transitions — poll every 10s (matches the
  // store's own refresh cadence) and diff statuses.
  function syncAgentTransitions() {
    for (const a of agentStore.agents) {
      const prev = lastAgentStatus.get(a.id);
      const cur = a.status ?? 'unknown';
      if (prev !== undefined && prev !== cur) {
        // transition
        const goingUp = cur === 'online' && prev !== 'online';
        const goingDown = cur !== 'online' && prev === 'online';
        if (goingUp || goingDown) {
          pushEntry({
            id: `agent-${a.id}-${a.last_seen}`,
            ts: a.last_seen ?? new Date().toISOString(),
            kind: goingUp ? 'agent-online' : 'agent-offline',
            severity: goingUp ? 'info' : 'warn',
            title: goingUp
              ? `Agent connected: ${a.hostname}`
              : `Agent went offline: ${a.hostname}`,
            detail: `version ${a.version}`,
            target: { route: `/host/${encodeURIComponent(a.id)}` },
          });
        }
      }
      lastAgentStatus.set(a.id, cur);
    }
  }

  onMount(() => {
    // Initial seed: don't fire transitions for the first observation.
    for (const a of agentStore.agents) {
      lastAgentStatus.set(a.id, a.status ?? 'unknown');
    }
    pollHandle = setInterval(syncAgentTransitions, 10_000);
  });

  onDestroy(() => {
    if (pollHandle) clearInterval(pollHandle);
  });

  function relativeTime(iso: string): string {
    const t = new Date(iso).getTime();
    if (!isFinite(t)) return iso;
    const diff = (Date.now() - t) / 1000;
    if (diff < 60) return `${Math.floor(diff)}s ago`;
    if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
    if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
    return new Date(iso).toLocaleDateString();
  }

  function activate(entry: Entry) {
    if (entry.target?.route) push(entry.target.route);
  }

  // Static icon map per the Phase 29 lesson — never reach into the
  // lucide-svelte namespace by string at runtime.
  function iconFor(kind: Entry['kind']) {
    switch (kind) {
      case 'alert':         return AlertTriangle;
      case 'agent-online':  return Circle;
      case 'agent-offline': return ServerCog;
      case 'event':         return Zap;
      default:              return Activity;
    }
  }
</script>

<div class="feed" class:bare>
  {#if !bare}
    <header class="feed-header">
      <Activity class="feed-header-icon" size={12} strokeWidth={1.6} />
      <span class="feed-header-title">ACTIVITY</span>
      <span class="feed-header-count">{entries.length}</span>
    </header>
  {/if}

  <div class="feed-body">
    {#if entries.length === 0}
      <div class="feed-empty">
        <Activity size={20} strokeWidth={1.4} />
        <p>Waiting for live activity…</p>
        <span class="feed-empty-sub">New alerts and agent transitions appear here.</span>
      </div>
    {:else}
      <ol class="feed-list">
        {#each entries as entry (entry.id)}
          {@const Icon = iconFor(entry.kind)}
          <li>
            <button
              type="button"
              class="feed-entry sev-{entry.severity ?? 'info'} kind-{entry.kind}"
              onclick={() => activate(entry)}
              disabled={!entry.target}
            >
              <span class="feed-entry-icon" aria-hidden="true">
                <Icon size={11} strokeWidth={1.8} />
              </span>
              <span class="feed-entry-body">
                <span class="feed-entry-title">{entry.title}</span>
                {#if entry.detail}
                  <span class="feed-entry-detail">{entry.detail}</span>
                {/if}
              </span>
              <span class="feed-entry-time">{relativeTime(entry.ts)}</span>
            </button>
          </li>
        {/each}
      </ol>
    {/if}
  </div>
</div>

<style>
  .feed {
    background: var(--color-surface-1);
    border: 1px solid var(--color-border-primary);
    border-radius: 6px;
    display: flex;
    flex-direction: column;
    overflow: hidden;
    min-width: 0;
  }
  .feed.bare {
    background: transparent;
    border: none;
    border-radius: 0;
  }

  .feed-header {
    display: flex;
    align-items: center;
    gap: 6px;
    padding: 8px 10px;
    background: var(--color-surface-2);
    border-bottom: 1px solid var(--color-border-primary);
  }
  :global(.feed-header-icon) { color: var(--color-accent); }

  .feed-header-title {
    font-family: var(--font-mono);
    font-size: 9px;
    font-weight: 800;
    color: var(--color-text-heading);
    letter-spacing: 0.15em;
  }
  .feed-header-count {
    margin-left: auto;
    font-family: var(--font-mono);
    font-size: 9px;
    color: var(--color-text-muted);
  }

  .feed-body { flex: 1; overflow-y: auto; min-height: 0; }
  .feed-list {
    list-style: none;
    margin: 0;
    padding: 4px;
  }
  .feed-list li + li { margin-top: 2px; }

  .feed-entry {
    display: flex;
    align-items: center;
    gap: 8px;
    width: 100%;
    padding: 6px 8px;
    background: transparent;
    border: 1px solid transparent;
    border-radius: 4px;
    text-align: left;
    cursor: pointer;
    transition: background 100ms, border-color 100ms;
  }
  .feed-entry:not(:disabled):hover {
    background: var(--color-surface-2);
    border-color: var(--color-border-primary);
  }
  .feed-entry:disabled { cursor: default; }

  .feed-entry-icon {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 18px;
    height: 18px;
    border-radius: 50%;
    flex-shrink: 0;
  }

  .sev-debug    .feed-entry-icon { color: var(--color-sev-debug);    background: var(--color-sev-debug-bg); }
  .sev-info     .feed-entry-icon { color: var(--color-sev-info);     background: var(--color-sev-info-bg); }
  .sev-warn     .feed-entry-icon { color: var(--color-sev-warn);     background: var(--color-sev-warn-bg); }
  .sev-error    .feed-entry-icon { color: var(--color-sev-error);    background: var(--color-sev-error-bg); }
  .sev-critical .feed-entry-icon { color: var(--color-sev-critical); background: var(--color-sev-critical-bg); }

  .feed-entry-body {
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: 1px;
    min-width: 0;
  }
  .feed-entry-title {
    font-family: var(--font-ui);
    font-size: 11px;
    font-weight: 600;
    color: var(--color-text-heading);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .feed-entry-detail {
    font-family: var(--font-mono);
    font-size: 9px;
    color: var(--color-text-muted);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .feed-entry-time {
    font-family: var(--font-mono);
    font-size: 9px;
    color: var(--color-text-muted);
    flex-shrink: 0;
    text-align: right;
  }

  .feed-empty {
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    padding: 24px 16px;
    gap: 6px;
    color: var(--color-text-muted);
    text-align: center;
  }
  .feed-empty p {
    margin: 0;
    font-family: var(--font-mono);
    font-size: 10px;
    text-transform: uppercase;
    letter-spacing: 0.1em;
  }
  .feed-empty-sub {
    font-family: var(--font-ui);
    font-size: 10px;
    opacity: 0.6;
  }
</style>
