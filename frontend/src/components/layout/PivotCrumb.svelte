<!--
  OBLIVRA — Pivot Crumb chrome (Phase 32, UIUX cognitive-load fix).

  Renders the operator's recent pivot trail as a clickable strip.
  Mounted in App.svelte's chrome between the page header and content
  area; auto-hides when the trail is empty so it doesn't take chrome
  space on first launch.

  Each chip shows an icon + a short label. Clicking jumps back; right-
  click clears from that point forward (planned for future PR).
-->
<script lang="ts">
  import { sessionContext, type PivotKind } from '@lib/stores/sessionContext.svelte';
  import { push } from '@lib/router.svelte';
  import {
    Bell,
    Cpu,
    User,
    Box,
    Globe,
    Terminal,
    Folder,
    Filter,
    Building2,
    FileText,
    X,
    ChevronRight,
  } from 'lucide-svelte';

  const ICONS: Record<PivotKind, any> = {
    alert:   Bell,
    host:    Cpu,
    user:    User,
    process: Box,
    ip:      Globe,
    session: Terminal,
    case:    Folder,
    rule:    Filter,
    tenant:  Building2,
    page:    FileText,
  };

  function jumpBack(id: string) {
    const c = sessionContext.trail.find((p) => p.id === id);
    if (!c) return;
    sessionContext.jumpTo(id);
    // Encode params back into the URL so destination pages can hydrate
    // their filters from query string.
    const qs = c.params ? '?' + new URLSearchParams(c.params).toString() : '';
    push(c.route + qs);
  }

  function clearAll() {
    sessionContext.clear();
  }
</script>

{#if sessionContext.trail.length > 0}
  <nav
    class="flex items-center gap-1 px-3 py-1 border-b border-border-primary bg-surface-1/60 overflow-x-auto whitespace-nowrap"
    style="min-height: 26px;"
    aria-label="Pivot history"
  >
    <span class="text-[var(--fs-micro)] font-mono uppercase tracking-widest text-text-muted shrink-0">trail:</span>
    {#each sessionContext.trail as crumb, i (crumb.id)}
      {@const Icon = ICONS[crumb.kind]}
      <button
        class="flex items-center gap-1 px-1.5 py-0.5 rounded-sm bg-surface-2 border border-border-primary hover:border-accent hover:bg-accent/5 transition-colors duration-fast text-[var(--fs-micro)]"
        onclick={() => jumpBack(crumb.id)}
        title="Jump back to {crumb.kind}: {crumb.label}"
      >
        <Icon size={10} class="text-text-muted" />
        <span class="font-mono text-text-secondary">{crumb.label}</span>
      </button>
      {#if i < sessionContext.trail.length - 1}
        <ChevronRight size={10} class="text-text-muted shrink-0" />
      {/if}
    {/each}
    <button
      class="ml-auto p-0.5 rounded-sm text-text-muted hover:text-text-primary hover:bg-surface-2"
      onclick={clearAll}
      aria-label="Clear pivot trail"
      title="Clear pivot trail"
    ><X size={11} /></button>
  </nav>
{/if}
