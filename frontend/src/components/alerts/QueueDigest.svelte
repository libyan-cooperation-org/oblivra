<!--
  OBLIVRA — Queue digest (Phase 32).

  At the top of the operator's first AlertManagement visit each day,
  surface a small card listing critical/high alerts from yesterday that
  are still OPEN. The "shift handover" problem in incumbent tools is
  that the queue is a flat list — whatever was unresolved yesterday
  drops below the top-20 freshness window.

  Mechanism:
   • On mount, compare today's date to localStorage `oblivra:queueDigestSeen`.
   • If different day, scan alertStore for severity ≥ high, status open,
     timestamp < startOfToday().
   • If non-empty, render the card with up to 5 entries + "see all" link.
   • Operator dismisses → set seen-key to today's date so it doesn't
     reappear later in the day. Re-fires next morning.

  Dismissal is operator-scoped (per-browser localStorage) — perfect for
  v1. Cross-device sync goes via the user-prefs table later.
-->
<script lang="ts">
  import { onMount } from 'svelte';
  import { alertStore, type Alert } from '@lib/stores/alerts.svelte';
  import { Badge } from '@components/ui';
  import { Sunrise, X, ChevronRight } from 'lucide-svelte';
  import { push } from '@lib/router.svelte';

  const SEEN_KEY = 'oblivra:queueDigestSeen';

  let dismissed = $state(true);

  function todayKey(): string {
    const d = new Date();
    return `${d.getFullYear()}-${d.getMonth() + 1}-${d.getDate()}`;
  }

  function startOfToday(): number {
    const d = new Date();
    d.setHours(0, 0, 0, 0);
    return d.getTime();
  }

  onMount(() => {
    let seen: string | null = null;
    try { seen = localStorage.getItem(SEEN_KEY); } catch { /* private mode */ }
    if (seen !== todayKey()) {
      dismissed = false;
    }
  });

  // Yesterday's high+critical alerts that are still open.
  let leftovers = $derived.by(() => {
    const cutoff = startOfToday();
    return alertStore.alerts
      .filter((a) => {
        if (a.status === 'closed' || a.status === 'suppressed') return false;
        const sev = (a.severity ?? '').toLowerCase();
        if (sev !== 'critical' && sev !== 'high') return false;
        const t = Date.parse(a.timestamp ?? '');
        return Number.isFinite(t) && t < cutoff;
      })
      .sort((a, b) => String(b.timestamp).localeCompare(String(a.timestamp)))
      .slice(0, 5);
  });

  function dismiss() {
    dismissed = true;
    try { localStorage.setItem(SEEN_KEY, todayKey()); } catch { /* private mode */ }
  }

  function openAlert(a: Alert) {
    push(`/alert-management?alert=${encodeURIComponent(a.id)}`);
  }
</script>

{#if !dismissed && leftovers.length > 0}
  <div
    class="bg-warning/5 border border-warning/30 rounded-md p-3 flex flex-col gap-2"
    role="region"
    aria-labelledby="queue-digest-title"
  >
    <header class="flex items-center justify-between">
      <div class="flex items-center gap-2">
        <Sunrise size={12} class="text-warning" />
        <span id="queue-digest-title" class="text-[var(--fs-label)] font-bold uppercase tracking-widest text-text-heading">
          Carry-over from yesterday
        </span>
        <Badge variant="warning" size="xs">{leftovers.length}</Badge>
      </div>
      <button
        class="text-text-muted hover:text-text-primary p-1 rounded-sm hover:bg-surface-2"
        onclick={dismiss}
        aria-label="Dismiss"
        title="Dismiss until tomorrow"
      ><X size={12} /></button>
    </header>

    <p class="text-[var(--fs-label)] text-text-secondary leading-relaxed">
      {leftovers.length} high-or-critical alert{leftovers.length === 1 ? '' : 's'} from before today are still open. Triage these first.
    </p>

    <ul class="flex flex-col gap-1">
      {#each leftovers as a (a.id)}
        <li>
          <button
            class="w-full flex items-center gap-2 px-2 py-1.5 bg-surface-2 hover:bg-surface-3 border border-border-primary rounded text-start transition-colors duration-fast"
            onclick={() => openAlert(a)}
          >
            <Badge
              variant={a.severity === 'critical' ? 'critical' : 'warning'}
              size="xs"
            >{a.severity.toUpperCase()}</Badge>
            <span class="font-mono text-[var(--fs-micro)] text-text-muted">{(a.timestamp ?? '').slice(0, 10)}</span>
            <span class="text-[var(--fs-label)] text-text-secondary truncate flex-1">{a.title}</span>
            <ChevronRight size={11} class="text-text-muted shrink-0" />
          </button>
        </li>
      {/each}
    </ul>
  </div>
{/if}
