<!--
  InvestigationPanel — global slide-out drawer for entity context.

  Phase 31 SOC redesign. Mounted once at the App root; opens
  whenever investigationStore.open is true. Renders the active
  entity's context: header (entity label + type chip + back button),
  related-alerts strip, related-logs strip, an activity timeline,
  and a footer with quick-jump actions.

  Why this is the right UX abstraction (per the audit spec):
    - One single panel for every entity type. Visual consistency.
    - Stays mounted across navigation — operator can drill from any
      page without losing context.
    - Slide-from-right with smooth 220ms transition; backdrop
      dismiss + Escape close + back button to walk the history stack.
-->
<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import {
    X, ChevronLeft, ExternalLink,
    Server, User as UserIcon, Globe, Cpu, Hash as HashIcon, Network as NetIcon,
    AlertTriangle, Clock, Search,
    type Icon as IconType,
  } from 'lucide-svelte';
  import { investigationStore, type EntityType } from '@lib/stores/investigation.svelte';
  import { alertStore } from '@lib/stores/alerts.svelte';
  import { agentStore } from '@lib/stores/agent.svelte';
  import { siemStore } from '@lib/stores/siem.svelte';
  import { push } from '@lib/router.svelte';

  // ── Reactive view of the store ───────────────────────────────────
  const entity = $derived(investigationStore.active);
  const open   = $derived(investigationStore.open);
  const canGoBack = $derived(investigationStore.history.length > 0);

  // Static entity-type → icon map. Phase 29 lesson: explicit, not
  // reflective lookup — Vite would tree-shake otherwise.
  const ENTITY_ICONS: Record<EntityType, typeof IconType> = {
    host:    Server,
    user:    UserIcon,
    ip:      Globe,
    process: Cpu,
    hash:    HashIcon,
    domain:  NetIcon,
    alert:   AlertTriangle,
  };

  // ── Related data, filtered to the active entity ──────────────────
  // Each entity type has its own matcher: a host matches alerts where
  // alert.host == id; an IP matches alerts whose description / metadata
  // mentions the IP, etc.
  const relatedAlerts = $derived.by(() => {
    if (!entity) return [];
    return alertStore.alerts.filter((a) => {
      switch (entity.type) {
        case 'host':
          return a.host === entity.id || a.host === entity.label;
        case 'user':
          // hosts often carry the user in description/raw — best-effort
          return (a.description ?? '').toLowerCase().includes(entity.id.toLowerCase()) ||
                 (a.host ?? '').toLowerCase().includes(entity.id.toLowerCase());
        case 'ip':
        case 'domain':
        case 'hash':
        case 'process':
          return (a.description ?? '').includes(entity.id) ||
                 (a.title ?? '').includes(entity.id);
        case 'alert':
          return a.id === entity.id;
        default:
          return false;
      }
    }).slice(0, 50);
  });

  const relatedEvents = $derived.by(() => {
    if (!entity) return [];
    return ((siemStore as any).results ?? []).filter((e: any) => {
      const haystack = JSON.stringify(e ?? {}).toLowerCase();
      return haystack.includes(entity.id.toLowerCase());
    }).slice(0, 50);
  });

  const matchingAgent = $derived.by(() => {
    if (!entity || entity.type !== 'host') return undefined;
    return agentStore.agents.find((a) => a.id === entity.id || a.hostname === entity.id);
  });

  // ── Quick-jump actions vary by entity type ───────────────────────
  function pivotToFullPage() {
    if (!entity) return;
    switch (entity.type) {
      case 'host':
        push(`/host/${encodeURIComponent(entity.id)}`);
        break;
      case 'alert':
        push(`/alert-management?alert=${encodeURIComponent(entity.id)}`);
        break;
      default:
        push(`/siem-search?q=${encodeURIComponent(entity.id)}`);
    }
    investigationStore.close();
  }

  function pivotToSIEM() {
    if (!entity) return;
    const param = entity.type === 'host' ? 'host' :
                  entity.type === 'user' ? 'user' :
                  entity.type === 'ip'   ? 'src_ip' : 'q';
    push(`/siem-search?${param}=${encodeURIComponent(entity.id)}`);
    investigationStore.close();
  }

  function severityClass(sev?: string): string {
    switch ((sev ?? '').toLowerCase()) {
      case 'critical': return 'sev-critical';
      case 'high':
      case 'error':    return 'sev-error';
      case 'warning':
      case 'warn':
      case 'medium':   return 'sev-warn';
      case 'info':     return 'sev-info';
      case 'debug':    return 'sev-debug';
      default:         return 'sev-info';
    }
  }

  function relativeTime(iso: string): string {
    const t = new Date(iso).getTime();
    if (!isFinite(t)) return iso;
    const diff = (Date.now() - t) / 1000;
    if (diff < 60) return `${Math.floor(diff)}s ago`;
    if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
    if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
    return new Date(iso).toLocaleString();
  }

  function onWindowKey(e: KeyboardEvent) {
    if (!open) return;
    if (e.key === 'Escape') {
      e.preventDefault();
      investigationStore.close();
    } else if (e.key === 'Backspace' && (e.metaKey || e.altKey)) {
      e.preventDefault();
      investigationStore.back();
    }
  }

  onMount(() => window.addEventListener('keydown', onWindowKey));
  onDestroy(() => window.removeEventListener('keydown', onWindowKey));
</script>

{#if open && entity}
  {@const Icon = ENTITY_ICONS[entity.type] ?? Search}
  <!-- Backdrop — clickable to dismiss. Pointer-events only when open
       so it doesn't interfere with normal page interaction. -->
  <button
    class="ip-backdrop"
    aria-label="Close investigation panel"
    onclick={() => investigationStore.close()}
  ></button>

  <div
    class="ip-panel"
    role="dialog"
    aria-modal="true"
    aria-labelledby="ip-title"
  >
    <!-- Header -->
    <header class="ip-header">
      <div class="ip-header-row">
        {#if canGoBack}
          <button
            class="ip-icon-btn"
            title="Back to previous entity (⌘⌫)"
            aria-label="Back"
            onclick={() => investigationStore.back()}
          >
            <ChevronLeft size={14} strokeWidth={1.8} />
          </button>
        {/if}

        <span class="ip-entity-icon" aria-hidden="true">
          <Icon size={16} strokeWidth={1.6} />
        </span>

        <div class="ip-title-block">
          <span id="ip-title" class="ip-title">{entity.label ?? entity.id}</span>
          <span class="ip-subtitle">
            <span class="ip-type-chip">{entity.type}</span>
            <span class="ip-id">{entity.id}</span>
          </span>
        </div>

        <button
          class="ip-icon-btn"
          title="Close (Esc)"
          aria-label="Close"
          onclick={() => investigationStore.close()}
        >
          <X size={14} strokeWidth={1.8} />
        </button>
      </div>

      <!-- Quick actions -->
      <div class="ip-actions">
        <button class="ip-action" onclick={pivotToFullPage}>
          <ExternalLink size={11} strokeWidth={1.7} />
          <span>Full page</span>
        </button>
        <button class="ip-action" onclick={pivotToSIEM}>
          <Search size={11} strokeWidth={1.7} />
          <span>Pivot to SIEM</span>
        </button>
      </div>
    </header>

    <!-- Body — tab-less single scroll, sections stacked. -->
    <div class="ip-body">

      <!-- Host metadata, when applicable -->
      {#if matchingAgent}
        <section class="ip-section">
          <h3 class="ip-section-title">Host Status</h3>
          <dl class="ip-kv">
            <dt>Status</dt><dd>{matchingAgent.status ?? 'unknown'}</dd>
            <dt>OS</dt><dd>{matchingAgent.os ?? '—'}</dd>
            <dt>Version</dt><dd>{matchingAgent.version}</dd>
            <dt>Trust</dt><dd>{matchingAgent.trust_level ?? 'unverified'}</dd>
            <dt>Last seen</dt><dd>{relativeTime(matchingAgent.last_seen)}</dd>
          </dl>
        </section>
      {/if}

      <!-- Related Alerts -->
      <section class="ip-section">
        <h3 class="ip-section-title">
          <AlertTriangle size={11} strokeWidth={1.7} class="ip-section-icon" />
          Related Alerts <span class="ip-count">{relatedAlerts.length}</span>
        </h3>
        {#if relatedAlerts.length === 0}
          <p class="ip-empty">No related alerts.</p>
        {:else}
          <ul class="ip-list">
            {#each relatedAlerts.slice(0, 10) as a (a.id)}
              <li class="ip-row {severityClass(a.severity)}">
                <span class="ip-row-dot" aria-hidden="true"></span>
                <span class="ip-row-title">{a.title}</span>
                <span class="ip-row-meta">{relativeTime(a.timestamp)}</span>
              </li>
            {/each}
          </ul>
        {/if}
      </section>

      <!-- Activity timeline (interleaved alerts + events, DESC). -->
      <section class="ip-section">
        <h3 class="ip-section-title">
          <Clock size={11} strokeWidth={1.7} class="ip-section-icon" />
          Activity Timeline <span class="ip-count">{relatedAlerts.length + relatedEvents.length}</span>
        </h3>
        {#if relatedAlerts.length === 0 && relatedEvents.length === 0}
          <p class="ip-empty">No activity yet — events and alerts populate this timeline as they arrive.</p>
        {:else}
          <ol class="ip-timeline">
            {#each [...relatedAlerts.map((a) => ({ ts: a.timestamp, kind: 'alert', title: a.title, sev: a.severity })), ...relatedEvents.map((e: any) => ({ ts: e.timestamp ?? new Date().toISOString(), kind: 'event', title: e.event_type ?? e.EventType ?? 'event', sev: 'info' }))]
              .sort((x, y) => new Date(y.ts).getTime() - new Date(x.ts).getTime())
              .slice(0, 25) as item, i (item.ts + i)}
              <li class="ip-tl-entry {severityClass(item.sev)}">
                <span class="ip-tl-marker" aria-hidden="true"></span>
                <span class="ip-tl-kind">{item.kind}</span>
                <span class="ip-tl-title">{item.title}</span>
                <span class="ip-tl-time">{relativeTime(item.ts)}</span>
              </li>
            {/each}
          </ol>
        {/if}
      </section>
    </div>
  </div>
{/if}

<style>
  /* Pinned to the right edge, below TitleBar (top: 32px to clear it). */
  .ip-backdrop {
    position: fixed;
    inset: 32px 0 0 0;
    background: rgba(0, 0, 0, 0.32);
    border: none;
    cursor: pointer;
    z-index: 80;
    animation: ip-fade-in 220ms ease-in-out;
  }
  .ip-panel {
    position: fixed;
    top: 32px;
    right: 0;
    bottom: 0;
    width: min(440px, 95vw);
    background: var(--color-surface-1);
    border-left: 1px solid var(--color-border-primary);
    box-shadow: -8px 0 32px rgba(0, 0, 0, 0.45);
    z-index: 90;
    display: flex;
    flex-direction: column;
    animation: ip-slide-in 240ms cubic-bezier(0.2, 0.7, 0.2, 1);
  }

  @keyframes ip-slide-in {
    from { transform: translateX(8%); opacity: 0; }
    to   { transform: translateX(0);  opacity: 1; }
  }
  @keyframes ip-fade-in {
    from { opacity: 0; }
    to   { opacity: 1; }
  }
  @media (prefers-reduced-motion: reduce) {
    .ip-panel { animation: ip-fade-in 100ms ease-in-out; }
    .ip-backdrop { animation: none; }
  }

  /* ── Header ────────────────────────────────────────────────── */
  .ip-header {
    flex-shrink: 0;
    padding: 12px 14px;
    border-bottom: 1px solid var(--color-border-primary);
    background: var(--color-surface-2);
  }
  .ip-header-row {
    display: flex;
    align-items: center;
    gap: 10px;
  }
  .ip-icon-btn {
    background: transparent;
    border: none;
    padding: 4px;
    border-radius: 4px;
    color: var(--color-text-muted);
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: color 100ms, background 100ms;
  }
  .ip-icon-btn:hover {
    color: var(--color-text-heading);
    background: var(--color-surface-3);
  }

  .ip-entity-icon {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 28px;
    height: 28px;
    border-radius: 6px;
    background: var(--color-sev-info-bg);
    color: var(--color-accent);
    flex-shrink: 0;
  }

  .ip-title-block {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }
  .ip-title {
    font-family: var(--font-ui);
    font-size: 13px;
    font-weight: 700;
    color: var(--color-text-heading);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .ip-subtitle {
    display: flex;
    align-items: center;
    gap: 6px;
    font-family: var(--font-mono);
    font-size: 9px;
  }
  .ip-type-chip {
    background: var(--color-surface-3);
    color: var(--color-text-secondary);
    text-transform: uppercase;
    letter-spacing: 0.08em;
    padding: 1px 5px;
    border-radius: 3px;
    font-weight: 700;
  }
  .ip-id {
    color: var(--color-text-muted);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .ip-actions {
    display: flex;
    gap: 6px;
    margin-top: 10px;
  }
  .ip-action {
    display: inline-flex;
    align-items: center;
    gap: 4px;
    padding: 4px 8px;
    background: transparent;
    border: 1px solid var(--color-border-primary);
    border-radius: 3px;
    color: var(--color-text-muted);
    font-family: var(--font-mono);
    font-size: 9px;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.08em;
    cursor: pointer;
    transition: color 100ms, border-color 100ms, background 100ms;
  }
  .ip-action:hover {
    color: var(--color-text-heading);
    border-color: var(--color-accent);
    background: var(--color-sev-info-bg);
  }

  /* ── Body ───────────────────────────────────────────────────── */
  .ip-body {
    flex: 1;
    overflow-y: auto;
    padding: 12px 14px;
    display: flex;
    flex-direction: column;
    gap: 16px;
  }
  .ip-section { display: flex; flex-direction: column; gap: 6px; }
  .ip-section-title {
    margin: 0;
    display: flex;
    align-items: center;
    gap: 6px;
    font-family: var(--font-mono);
    font-size: 9px;
    font-weight: 800;
    color: var(--color-text-muted);
    text-transform: uppercase;
    letter-spacing: 0.12em;
  }
  :global(.ip-section-icon) { color: var(--color-accent); }
  .ip-count {
    margin-left: auto;
    background: var(--color-surface-3);
    color: var(--color-text-secondary);
    padding: 1px 6px;
    border-radius: 8px;
    font-size: 8px;
  }
  .ip-empty {
    margin: 0;
    color: var(--color-text-muted);
    font-family: var(--font-mono);
    font-size: 10px;
    opacity: 0.6;
    padding: 6px 0;
  }

  .ip-kv {
    margin: 0;
    display: grid;
    grid-template-columns: max-content 1fr;
    gap: 4px 12px;
    font-family: var(--font-mono);
    font-size: 10px;
  }
  .ip-kv dt { color: var(--color-text-muted); text-transform: uppercase; letter-spacing: 0.08em; }
  .ip-kv dd { margin: 0; color: var(--color-text-heading); overflow-wrap: anywhere; }

  /* ── Lists ──────────────────────────────────────────────────── */
  .ip-list { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 3px; }
  .ip-row {
    display: grid;
    grid-template-columns: 6px 1fr auto;
    align-items: center;
    gap: 8px;
    padding: 6px 8px;
    background: var(--color-surface-2);
    border-left: 2px solid transparent;
    border-radius: 4px;
  }
  .ip-row-dot {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: currentColor;
    opacity: 0.7;
  }
  .ip-row-title {
    font-family: var(--font-ui);
    font-size: 11px;
    font-weight: 600;
    color: var(--color-text-heading);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .ip-row-meta {
    font-family: var(--font-mono);
    font-size: 9px;
    color: var(--color-text-muted);
  }

  .ip-row.sev-critical { border-left-color: var(--color-sev-critical); color: var(--color-sev-critical); }
  .ip-row.sev-error    { border-left-color: var(--color-sev-error);    color: var(--color-sev-error); }
  .ip-row.sev-warn     { border-left-color: var(--color-sev-warn);     color: var(--color-sev-warn); }
  .ip-row.sev-info     { border-left-color: var(--color-sev-info);     color: var(--color-sev-info); }
  .ip-row.sev-debug    { border-left-color: var(--color-sev-debug);    color: var(--color-sev-debug); }

  /* ── Timeline ───────────────────────────────────────────────── */
  .ip-timeline { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 2px; }
  .ip-tl-entry {
    display: grid;
    grid-template-columns: 8px max-content 1fr auto;
    align-items: center;
    gap: 8px;
    padding: 4px 6px;
    border-radius: 3px;
    font-family: var(--font-mono);
    font-size: 10px;
  }
  .ip-tl-entry:hover { background: var(--color-surface-2); }
  .ip-tl-marker {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: currentColor;
  }
  .ip-tl-kind {
    color: var(--color-text-muted);
    text-transform: uppercase;
    letter-spacing: 0.08em;
    font-size: 8px;
    font-weight: 800;
  }
  .ip-tl-title {
    color: var(--color-text-heading);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .ip-tl-time {
    color: var(--color-text-muted);
    font-size: 9px;
  }

  .ip-tl-entry.sev-critical { color: var(--color-sev-critical); }
  .ip-tl-entry.sev-error    { color: var(--color-sev-error); }
  .ip-tl-entry.sev-warn     { color: var(--color-sev-warn); }
  .ip-tl-entry.sev-info     { color: var(--color-sev-info); }
  .ip-tl-entry.sev-debug    { color: var(--color-sev-debug); }
</style>
