<!--
  OBLIVRA — Modal (Svelte 5)

  Overlay modal with title, body, and footer slots.

  A11y notes (UIUX_IMPROVEMENTS.md P1 #5):
    - Backdrop is `role="presentation"` so screen readers don't announce
      "Close, button" before the modal title. The visible ✕ button is
      the canonical close affordance.
    - Focus is moved to the dialog container on open, and restored to
      the previously-focused element on close.
    - Tab cycles inside the dialog (focus trap) so keyboard operators
      can't escape to the page underneath while the modal is open.
    - `aria-labelledby` points at the title heading when one is given.
-->
<script lang="ts">
  import { onMount, onDestroy, type Snippet } from 'svelte';
  import { fade, scale } from 'svelte/transition';

  interface Props {
    open: boolean;
    title?: string;
    size?: 'sm' | 'md' | 'lg' | 'xl';
    onClose: () => void;
    children: Snippet;
    footer?: Snippet;
  }

  let { open, title, size = 'md', onClose, children, footer }: Props = $props();

  const sizeClasses: Record<string, string> = {
    sm: 'max-w-sm',
    md: 'max-w-lg',
    lg: 'max-w-2xl',
    xl: 'max-w-4xl',
  };

  // Stable id for aria-labelledby. A bare counter is fine — modals are
  // singleton-ish in practice and the id collision risk is moot since
  // it lives only inside the open dialog.
  const titleId = `modal-title-${Math.random().toString(36).slice(2, 9)}`;

  let dialogEl = $state<HTMLDivElement | null>(null);
  let prevFocus: HTMLElement | null = null;

  function handleKeyDown(e: KeyboardEvent) {
    if (!open) return;
    if (e.key === 'Escape') {
      e.preventDefault();
      onClose();
      return;
    }
    if (e.key === 'Tab' && dialogEl) {
      // Focus trap — cycle within the dialog's focusables.
      const focusables = dialogEl.querySelectorAll<HTMLElement>(
        'button:not([disabled]), [href], input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])'
      );
      if (focusables.length === 0) {
        e.preventDefault();
        dialogEl.focus();
        return;
      }
      const first = focusables[0];
      const last = focusables[focusables.length - 1];
      const active = document.activeElement as HTMLElement | null;
      if (e.shiftKey) {
        if (active === first || !dialogEl.contains(active)) {
          e.preventDefault();
          last.focus();
        }
      } else {
        if (active === last) {
          e.preventDefault();
          first.focus();
        }
      }
    }
  }

  // Cache previous focus on open, restore on close, and move focus into
  // the dialog. Using $effect so reactive changes to `open` (the parent
  // toggling visibility) drive the focus dance.
  $effect(() => {
    if (open) {
      prevFocus = (document.activeElement as HTMLElement) ?? null;
      // Defer one frame so the dialog has mounted before .focus() runs.
      queueMicrotask(() => dialogEl?.focus());
    } else if (prevFocus) {
      try { prevFocus.focus(); } catch { /* node may be gone */ }
      prevFocus = null;
    }
  });

  onMount(() => window.addEventListener('keydown', handleKeyDown));
  onDestroy(() => window.removeEventListener('keydown', handleKeyDown));
</script>

{#if open}
  <!-- Backdrop — purely decorative for ATs. The close affordance is the
       visible ✕ in the header; we still react to clicks here so the
       interaction pattern users expect (click-outside-to-dismiss) keeps
       working. -->
  <div
    class="fixed inset-0 bg-black/55 z-[9000] flex items-start justify-center pt-[10vh]"
    transition:fade={{ duration: 150 }}
    role="presentation"
    onclick={onClose}
  >
    <!-- Modal container -->
    <div
      bind:this={dialogEl}
      class="w-full {sizeClasses[size]} bg-surface-2 border border-border-secondary rounded-lg shadow-xl overflow-hidden focus:outline-none"
      transition:scale={{ duration: 200, start: 0.95 }}
      role="dialog"
      aria-modal="true"
      aria-labelledby={title ? titleId : undefined}
      tabindex="-1"
      onclick={(e) => e.stopPropagation()}
      onkeydown={(e) => e.stopPropagation()}
    >
      <!-- Header -->
      {#if title}
        <div class="flex items-center justify-between px-5 py-3.5 border-b border-border-primary">
          <h2 id={titleId} class="text-sm font-bold text-text-heading font-sans">{title}</h2>
          <button
            class="w-6 h-6 flex items-center justify-center bg-transparent border-none text-text-muted hover:text-text-primary hover:bg-surface-3 rounded-sm cursor-pointer transition-all duration-fast text-sm"
            onclick={onClose}
            aria-label="Close"
          >✕</button>
        </div>
      {/if}

      <!-- Body -->
      <div class="p-5">
        {@render children()}
      </div>

      <!-- Footer -->
      {#if footer}
        <div class="flex items-center justify-end gap-2 px-5 py-3 border-t border-border-primary bg-surface-1/50">
          {@render footer()}
        </div>
      {/if}
    </div>
  </div>
{/if}
