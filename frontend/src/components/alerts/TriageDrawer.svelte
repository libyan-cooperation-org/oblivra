<!--
  OBLIVRA — Triage Drawer (Phase 32, UIUX P1 #8).

  Adjacent panel (NOT modal) that appears next to the alert queue when
  an alert is selected. Surfaces:

    1. Alert metadata (severity, host, time, MITRE)
    2. Next-Best-Action recommendation with the recommended button
       pre-highlighted in --color-accent. Alternatives below.
    3. One-key pivots (t timeline, s shell, g graph, e evidence)
    4. Close + advance to next alert (⌘+Enter)

  This is the highest-leverage UX surface in the product. Every design
  decision below optimises for "operator can decide and act in <3s
  without leaving the keyboard".
-->
<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { Badge, Button } from '@components/ui';
  import { ChevronRight, Clock, Cpu, Network, ShieldAlert, ShieldCheck, Eye, FileText, Zap, X } from 'lucide-svelte';
  import { recommend, fallbackRecommendation, ACTION_LABEL, ACTION_VARIANT, type RecommendedAction, type NBAAction } from '@lib/nba';
  import { appStore } from '@lib/stores/app.svelte';
  import { agentStore } from '@lib/stores/agent.svelte';
  import { alertStore } from '@lib/stores/alerts.svelte';
  import { sessionContext } from '@lib/stores/sessionContext.svelte';
  import { push } from '@lib/router.svelte';

  interface Alert {
    id: string;
    timestamp: string;
    title: string;
    severity: string;
    host: string;
    status: string;
    description: string;
    category?: string;
    raw?: any;
  }

  interface Props {
    /** Currently-selected alert, or null when nothing is selected. */
    alert: Alert | null;
    /** Called when the drawer wants to close (Esc, click ✕, or after action). */
    onClose: () => void;
    /** Called after the operator picks an action — parent decides what to do
     *  next (close + advance to next row is the typical pattern). */
    onAction?: (action: NBAAction, alert: Alert) => void;
  }

  let { alert, onClose, onAction }: Props = $props();

  let recommendation = $state<RecommendedAction | null>(null);
  let loadingRec = $state(false);

  /**
   * Build an NBA fact-set from the alert + surrounding store state.
   * The richer the facts, the better the recommendation — agentStore
   * tells us if the host is even known, the category comes from the
   * detection engine, etc.
   */
  function buildFacts(a: Alert): import('@lib/nba').NBAFacts {
    const hostKnown = agentStore.agents.some(
      (ag) => ag.id === a.host || ag.hostname === a.host,
    );
    return {
      alert_id: a.id,
      severity: a.severity,
      category: a.category ?? '',
      host_known: hostKnown,
      // The richer signals (IOC match, beaconing, first-time-binary)
      // need to come from the alert payload once the detection engine
      // surfaces them. For now we let the server fall back to the
      // severity baseline.
      has_ioc_match: Boolean((a.raw ?? {}).ioc_match),
      has_outbound_c2_beacon: Boolean((a.raw ?? {}).c2_beacon),
      is_first_time_binary: Boolean((a.raw ?? {}).first_time_binary),
      is_repeat_offender: Boolean((a.raw ?? {}).repeat_offender),
      host_is_critical: Boolean((a.raw ?? {}).host_critical),
      user_is_service: Boolean((a.raw ?? {}).user_is_service),
      is_from_crown_jewel: Boolean((a.raw ?? {}).crown_jewel),
    };
  }

  // Recompute the recommendation whenever the selected alert changes.
  $effect(() => {
    if (!alert) {
      recommendation = null;
      return;
    }
    loadingRec = true;
    const facts = buildFacts(alert);
    recommend(facts)
      .then((r) => { recommendation = r; })
      .catch((e) => {
        console.warn('[TriageDrawer] recommend failed, using fallback:', e);
        recommendation = fallbackRecommendation(alert.severity);
      })
      .finally(() => { loadingRec = false; });
  });

  function execute(action: NBAAction) {
    if (!alert) return;
    onAction?.(action, alert);
  }

  /**
   * One-key pivots (t/s/g/e). The drawer captures keys only while
   * mounted with an alert selected; the global App.svelte handlers
   * still fire when nothing is selected.
   */
  function onKey(e: KeyboardEvent) {
    if (!alert) return;
    // Don't hijack typing into inputs.
    const t = e.target as HTMLElement | null;
    if (t?.tagName === 'INPUT' || t?.tagName === 'TEXTAREA' || t?.isContentEditable) return;

    if (e.key === 'Escape') { onClose(); return; }

    // Each pivot drops a crumb so the operator's path is visible in
    // the chrome strip and they can jump back without re-running the
    // alert query. See sessionContext.svelte.ts.
    const pushCrumb = (kind: 'alert' | 'host' | 'session', route: string, params: Record<string, string>) => {
      sessionContext.push({
        id: `${kind}:${alert.id}`,
        kind,
        label: (kind === 'alert' ? alert.title : alert.host).slice(0, 24),
        route, params,
      });
    };
    if (e.key === 't') { e.preventDefault(); pushCrumb('alert', '/alert-management', { alert: alert.id }); push(`/timeline/${encodeURIComponent(alert.host)}/host/${alert.timestamp}`); return; }
    if (e.key === 's') { e.preventDefault(); pushCrumb('alert', '/alert-management', { alert: alert.id }); push(`/shell?host=${encodeURIComponent(alert.host)}`); return; }
    if (e.key === 'g') { e.preventDefault(); pushCrumb('alert', '/alert-management', { alert: alert.id }); push(`/threat-graph?seed=${encodeURIComponent(alert.host)}`); return; }
    if (e.key === 'e') { e.preventDefault(); pushCrumb('alert', '/alert-management', { alert: alert.id }); push(`/evidence?alert=${encodeURIComponent(alert.id)}`); return; }
    if (e.key === 'x') { e.preventDefault(); execute('suppress_as_fp'); return; }

    // ⌘/Ctrl + Enter — execute the recommended action and close.
    if ((e.metaKey || e.ctrlKey) && e.key === 'Enter' && recommendation) {
      e.preventDefault();
      execute(recommendation.action);
    }
  }

  onMount(() => window.addEventListener('keydown', onKey));
  onDestroy(() => window.removeEventListener('keydown', onKey));
</script>

{#if alert}
  <aside
    class="flex flex-col h-full bg-surface-1 border-l border-border-primary"
    aria-label="Alert triage"
  >
    <!-- Header -->
    <header class="flex items-center justify-between px-4 py-3 border-b border-border-primary shrink-0">
      <div class="flex items-center gap-2">
        <Badge
          variant={alert.severity === 'critical' ? 'critical' : alert.severity === 'high' ? 'warning' : 'info'}
          size="xs"
        >{alert.severity.toUpperCase()}</Badge>
        <span class="font-mono text-[var(--fs-micro)] text-text-muted">{alert.id.slice(0, 12)}</span>
      </div>
      <button
        class="text-text-muted hover:text-text-primary p-1 rounded-sm hover:bg-surface-2"
        onclick={onClose}
        aria-label="Close triage drawer"
      ><X size={14} /></button>
    </header>

    <!-- Body -->
    <div class="flex-1 overflow-auto p-4 flex flex-col gap-4">
      <!-- Title + meta -->
      <div class="flex flex-col gap-1.5">
        <h3 class="text-[var(--fs-body)] font-bold text-text-heading leading-snug">{alert.title}</h3>
        <div class="flex flex-wrap items-center gap-x-3 gap-y-1 text-[var(--fs-micro)] text-text-muted">
          <span class="flex items-center gap-1"><Clock size={10} />{(alert.timestamp ?? '').slice(0, 19)}</span>
          <span class="flex items-center gap-1"><Cpu size={10} />{alert.host || '—'}</span>
          {#if alert.category}
            <span class="flex items-center gap-1"><Network size={10} />{alert.category}</span>
          {/if}
        </div>
      </div>

      <!-- Recommendation -->
      <section
        class="bg-surface-2 border border-border-primary rounded-md p-3 flex flex-col gap-2"
        aria-label="Next-Best-Action recommendation"
      >
        <div class="flex items-center justify-between">
          <div class="flex items-center gap-1.5">
            <Zap size={11} class="text-accent" />
            <span class="text-[var(--fs-micro)] font-bold uppercase tracking-widest text-text-muted">
              Recommendation
            </span>
          </div>
          {#if recommendation}
            <span
              class="font-mono text-[var(--fs-micro)] {recommendation.confidence >= 0.6 ? 'text-success' : recommendation.confidence > 0 ? 'text-warning' : 'text-text-muted'}"
              title="Confidence"
            >conf {Math.round(recommendation.confidence * 100)}%</span>
          {/if}
        </div>

        {#if loadingRec}
          <div class="text-[var(--fs-label)] text-text-muted italic">Computing recommendation…</div>
        {:else if recommendation}
          <p class="text-[var(--fs-label)] text-text-secondary leading-relaxed">{recommendation.reason}</p>

          <!-- Pre-highlighted recommended action -->
          <Button
            variant={ACTION_VARIANT[recommendation.action]}
            size="sm"
            onclick={() => execute(recommendation!.action)}
            class="justify-start"
          >
            <span class="mr-2">▸</span>{ACTION_LABEL[recommendation.action]}
            <span class="ml-auto font-mono text-[var(--fs-micro)] opacity-70">⌘⏎</span>
          </Button>

          <!-- Alternatives -->
          {#if recommendation.alternatives.length > 0}
            <div class="flex flex-col gap-1.5 pt-1">
              <span class="text-[var(--fs-micro)] uppercase tracking-widest text-text-muted">Alternatives</span>
              {#each recommendation.alternatives as alt}
                <Button
                  variant="secondary"
                  size="sm"
                  onclick={() => execute(alt)}
                  class="justify-start"
                >{ACTION_LABEL[alt]}</Button>
              {/each}
            </div>
          {/if}
        {/if}
      </section>

      <!-- One-key pivots -->
      <section
        class="bg-surface-2 border border-border-primary rounded-md p-3 flex flex-col gap-2"
        aria-label="Pivots"
      >
        <span class="text-[var(--fs-micro)] font-bold uppercase tracking-widest text-text-muted">Pivots</span>
        <div class="grid grid-cols-2 gap-2">
          <button class="flex items-center gap-2 px-2 py-1.5 bg-surface-3 border border-border-primary hover:border-accent rounded text-[var(--fs-label)] text-text-secondary"
                  onclick={() => push(`/timeline/${encodeURIComponent(alert!.host)}/host/${alert!.timestamp}`)}>
            <Clock size={12} /> Timeline <kbd class="ml-auto font-mono text-[var(--fs-micro)] opacity-60">t</kbd>
          </button>
          <button class="flex items-center gap-2 px-2 py-1.5 bg-surface-3 border border-border-primary hover:border-accent rounded text-[var(--fs-label)] text-text-secondary"
                  onclick={() => push(`/shell?host=${encodeURIComponent(alert!.host)}`)}>
            <Cpu size={12} /> Shell <kbd class="ml-auto font-mono text-[var(--fs-micro)] opacity-60">s</kbd>
          </button>
          <button class="flex items-center gap-2 px-2 py-1.5 bg-surface-3 border border-border-primary hover:border-accent rounded text-[var(--fs-label)] text-text-secondary"
                  onclick={() => push(`/threat-graph?seed=${encodeURIComponent(alert!.host)}`)}>
            <Network size={12} /> Graph <kbd class="ml-auto font-mono text-[var(--fs-micro)] opacity-60">g</kbd>
          </button>
          <button class="flex items-center gap-2 px-2 py-1.5 bg-surface-3 border border-border-primary hover:border-accent rounded text-[var(--fs-label)] text-text-secondary"
                  onclick={() => push(`/evidence?alert=${encodeURIComponent(alert!.id)}`)}>
            <FileText size={12} /> Evidence <kbd class="ml-auto font-mono text-[var(--fs-micro)] opacity-60">e</kbd>
          </button>
        </div>
      </section>

      <!-- Description / raw -->
      {#if alert.description}
        <section class="bg-surface-2 border border-border-primary rounded-md p-3 flex flex-col gap-2">
          <span class="text-[var(--fs-micro)] font-bold uppercase tracking-widest text-text-muted">Detail</span>
          <p class="text-[var(--fs-label)] text-text-secondary font-mono whitespace-pre-wrap break-words leading-relaxed">{alert.description}</p>
        </section>
      {/if}
    </div>

    <!-- Footer keymap legend — always visible so new operators learn -->
    <footer class="px-4 py-2 border-t border-border-primary bg-surface-2/40 shrink-0">
      <div class="text-[var(--fs-micro)] font-mono text-text-muted leading-relaxed">
        <kbd>⌘⏎</kbd> recommended · <kbd>t</kbd> timeline · <kbd>s</kbd> shell · <kbd>g</kbd> graph · <kbd>e</kbd> evidence · <kbd>x</kbd> suppress · <kbd>esc</kbd> close
      </div>
    </footer>
  </aside>
{/if}

<style>
  kbd {
    background: var(--color-surface-3);
    border: 1px solid var(--color-border-primary);
    border-radius: 2px;
    padding: 0 4px;
    font-size: 9px;
  }
</style>
