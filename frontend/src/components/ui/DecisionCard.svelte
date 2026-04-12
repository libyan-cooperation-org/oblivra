<!--
  OBLIVRA — DecisionCard (Svelte 5)
  Core decision primitive. Every critical alert surfaces here with:
  severity, confidence score, recommended action, time-to-impact counter,
  and one-click resolve / escalate / dismiss controls.
-->
<script lang="ts">
  import { appStore } from '@lib/stores/app.svelte';
  import type { DecisionCardData } from './decision.types';

  interface Props {
    card: DecisionCardData;
    onResolve?: (id: string) => void;
    onEscalate?: (id: string) => void;
    onDismiss?: (id: string) => void;
    compact?: boolean;
  }

  let { card, onResolve, onEscalate, onDismiss, compact = false }: Props = $props();

  // ── Countdown ─────────────────────────────────────────────────────
  let remaining = $state(0);
  let expired = $state(false);
  let interval: ReturnType<typeof setInterval>;

  $effect(() => {
    // Reset/initialize state when card configuration changes
    remaining = card.impactSeconds;
    expired = card.impactSeconds <= 0;

    if (interval) clearInterval(interval);

    if (remaining > 0) {
      interval = setInterval(() => {
        remaining = Math.max(0, remaining - 1);
        if (remaining === 0) {
          expired = true;
          clearInterval(interval);
        }
      }, 1000);
    }

    return () => {
      if (interval) clearInterval(interval);
    };
  });

  function formatCountdown(secs: number): string {
    if (secs <= 0) return 'NOW';
    const h = Math.floor(secs / 3600);
    const m = Math.floor((secs % 3600) / 60);
    const s = secs % 60;
    if (h > 0) return `${h}h ${m}m`;
    if (m > 0) return `${m}m ${s.toString().padStart(2, '0')}s`;
    return `${s}s`;
  }

  // ── Severity config ────────────────────────────────────────────────
  const severityConfig = {
    P0: { label: 'P0 · CRITICAL', barColor: 'var(--accent-danger)', textColor: 'text-error', borderColor: 'border-error', bg: 'rgba(224,64,64,0.07)' },
    P1: { label: 'P1 · HIGH',     barColor: 'var(--accent-cta)',    textColor: 'text-warning', borderColor: 'border-warning', bg: 'rgba(245,139,0,0.07)' },
    P2: { label: 'P2 · MEDIUM',   barColor: 'var(--accent-warning)', textColor: 'text-[#f5c518]', borderColor: 'border-[#f5c518]', bg: 'rgba(245,197,24,0.06)' },
    P3: { label: 'P3 · LOW',      barColor: 'var(--status-online)', textColor: 'text-success', borderColor: 'border-success', bg: 'rgba(92,192,92,0.06)' },
  };
  const sev = $derived(severityConfig[card.severity]);
  const countdownUrgent = $derived(remaining < 60 && !expired);

  function handleResolve() {
    appStore.notify(`Decision ${card.id} resolved`, 'success');
    onResolve?.(card.id);
  }
  function handleEscalate() {
    appStore.notify(`Decision ${card.id} escalated`, 'warning');
    onEscalate?.(card.id);
  }
  function handleDismiss() {
    onDismiss?.(card.id);
  }
</script>

<div
  role="article"
  aria-label="Decision card: {card.title}"
  class="dc-card bg-surface-1 border rounded-md transition-all duration-fast hover:border-border-hover group relative overflow-hidden {sev.borderColor} {compact ? 'p-3' : 'p-4'}"
  style="background: {sev.bg}; border-left: 3px solid {sev.barColor};"
>
  <!-- Severity bar top -->
  <div class="dc-severity-strip absolute top-0 left-3 right-0 h-[2px] opacity-30" style="background: {sev.barColor};"></div>

  <!-- Header row -->
  <div class="flex items-start justify-between gap-3 mb-3">
    <div class="flex items-center gap-2 min-w-0">
      <span
        class="text-[9px] font-extrabold font-mono uppercase tracking-widest shrink-0 px-1.5 py-0.5 rounded-sm {sev.textColor}"
        style="background: {sev.barColor}22;"
      >{sev.label}</span>
      {#if card.mitre}
        <span class="text-[9px] font-mono text-text-muted opacity-60 shrink-0">{card.mitre}</span>
      {/if}
    </div>

    <!-- Time-to-impact -->
    <div
      class="flex flex-col items-end shrink-0"
      aria-label="Time to impact: {formatCountdown(remaining)}"
    >
      <span class="text-[8px] font-bold uppercase tracking-widest text-text-muted opacity-50">IMPACT IN</span>
      <span
        class="text-sm font-mono font-black tabular-nums transition-colors {expired ? 'text-error animate-pulse' : countdownUrgent ? 'text-warning' : 'text-text-secondary'}"
      >{formatCountdown(remaining)}</span>
    </div>
  </div>

  <!-- Title -->
  <div class="text-[13px] font-bold text-text-heading leading-tight mb-1 {compact ? '' : 'mb-2'}">{card.title}</div>

  {#if !compact}
    <!-- Meta row -->
    <div class="flex items-center gap-3 mb-3 flex-wrap">
      {#if card.host}
        <span class="text-[10px] font-mono text-text-muted">
          <span class="text-text-muted opacity-40">HOST</span>
          <span class="text-text-secondary ml-1">{card.host}</span>
        </span>
      {/if}
      {#if card.source}
        <span class="text-[10px] font-mono text-text-muted">
          <span class="text-text-muted opacity-40">SRC</span>
          <span class="text-text-secondary ml-1">{card.source}</span>
        </span>
      {/if}
    </div>
  {/if}

  <!-- Confidence bar -->
  <div class="mb-3">
    <div class="flex justify-between items-center mb-1">
      <span class="text-[9px] font-bold uppercase tracking-widest text-text-muted opacity-60">Confidence</span>
      <span class="text-[10px] font-mono font-bold text-text-secondary tabular-nums">{card.confidence}%</span>
    </div>
    <div class="h-[3px] bg-surface-3 rounded-full overflow-hidden" aria-hidden="true">
      <div
        class="h-full rounded-full transition-all duration-slow"
        style="width: {card.confidence}%; background: {card.confidence >= 80 ? sev.barColor : 'var(--text-muted)'};"
      ></div>
    </div>
  </div>

  <!-- Recommended action -->
  <div class="dc-action bg-surface-2 border border-border-primary rounded-sm px-3 py-2 mb-3">
    <div class="text-[9px] font-bold uppercase tracking-widest text-text-muted opacity-50 mb-0.5">Recommended Action</div>
    <div class="text-[11px] font-semibold text-text-heading leading-snug">{card.recommendedAction}</div>
  </div>

  <!-- Controls -->
  <div class="flex items-center gap-2">
    <button
      class="dc-btn-resolve flex-1 h-[28px] rounded-sm text-[10px] font-bold uppercase tracking-wider font-mono transition-all duration-fast border"
      style="background: {sev.barColor}22; border-color: {sev.barColor}55; color: {sev.barColor};"
      onclick={handleResolve}
      aria-label="Resolve decision {card.id}"
    >Resolve</button>

    <button
      class="dc-btn-escalate h-[28px] px-3 rounded-sm text-[10px] font-bold uppercase tracking-wider font-mono text-warning transition-all duration-fast border border-warning/30 bg-warning/10 hover:bg-warning/20"
      onclick={handleEscalate}
      aria-label="Escalate decision {card.id}"
    >Escalate</button>

    <button
      class="dc-btn-dismiss h-[28px] px-2 rounded-sm text-[10px] font-bold uppercase tracking-wider font-mono text-text-muted transition-all duration-fast border border-border-primary hover:border-border-hover hover:text-text-secondary"
      onclick={handleDismiss}
      aria-label="Dismiss decision {card.id}"
    >✕</button>
  </div>
</div>

<style>
  .dc-btn-resolve:hover { filter: brightness(1.15); }
</style>
