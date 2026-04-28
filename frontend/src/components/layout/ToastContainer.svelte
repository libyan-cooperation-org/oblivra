<!--
  OBLIVRA — Toast Container (Svelte 5).

  A11y notes (UIUX_IMPROVEMENTS.md P2 #13):
   • The container splits into TWO regions — `aria-live="polite"` for
     info/success and `aria-live="assertive"` for warning/error. Mixing
     both into one region is the most common a11y bug in toast UIs;
     screen-readers can drown important errors under chatty success
     toasts.
   • Cap visible toasts at 4. Older toasts are dropped from view (the
     toastStore can hold more if it wants — the cap is presentational).
   • Auto-dismiss can pause on hover (the toastStore handles the timer
     side; we just stop reporting hover changes back to it for now —
     hover-pause is a Phase 33 follow-up that needs a per-toast timer
     pause API in the store).
-->
<script lang="ts">
  import { toastStore } from '@lib/stores/toast.svelte';
  import { slide } from 'svelte/transition';

  const typeStyles: Record<string, string> = {
    info:    'border-accent bg-accent/5 text-accent',
    success: 'border-success bg-success/5 text-success',
    warning: 'border-warning bg-warning/5 text-warning',
    error:   'border-error bg-error/5 text-error',
  };

  const typeIcons: Record<string, string> = {
    info:    'ℹ',
    success: '✓',
    warning: '⚠',
    error:   '✕',
  };

  const MAX_VISIBLE = 4;

  // Split into polite (info/success) and assertive (warning/error)
  // queues. The whole list is sorted oldest-first; we cap to 4 visible
  // PER queue so a flood of one type doesn't drown the other.
  const politeToasts = $derived(
    toastStore.items.filter((t) => t.type === 'info' || t.type === 'success').slice(-MAX_VISIBLE),
  );
  const assertiveToasts = $derived(
    toastStore.items.filter((t) => t.type === 'warning' || t.type === 'error').slice(-MAX_VISIBLE),
  );
</script>

<!-- Two stacked live regions — assertive (errors) on top so they catch
     the eye first; polite below. Both are pointer-events-none on the
     container so background clicks pass through; toasts opt back in. -->
<div class="fixed top-3 right-3 z-[10000] flex flex-col gap-2 pointer-events-none">
  <div role="status" aria-live="assertive" aria-atomic="false" class="flex flex-col gap-2">
    {#each assertiveToasts as toast (toast.id)}
      <div
        class="pointer-events-auto flex items-start gap-3 px-4 py-3 rounded-md border shadow-lg backdrop-blur-sm min-w-[300px] max-w-[420px] font-sans text-xs {typeStyles[toast.type]}"
        transition:slide={{ duration: 200 }}
      >
        <span class="text-sm shrink-0 mt-px" aria-hidden="true">{typeIcons[toast.type]}</span>
        <div class="flex-1 min-w-0">
          <div class="font-semibold text-text-heading text-[13px]">{toast.title}</div>
          {#if toast.message}
            <div class="text-text-secondary mt-0.5 leading-relaxed">{toast.message}</div>
          {/if}
        </div>
        <button
          class="text-text-muted hover:text-text-primary text-sm shrink-0 cursor-pointer bg-transparent border-none p-0"
          onclick={() => toastStore.remove(toast.id)}
          aria-label="Dismiss"
        >✕</button>
      </div>
    {/each}
  </div>

  <div role="status" aria-live="polite" aria-atomic="false" class="flex flex-col gap-2">
    {#each politeToasts as toast (toast.id)}
      <div
        class="pointer-events-auto flex items-start gap-3 px-4 py-3 rounded-md border shadow-lg backdrop-blur-sm min-w-[300px] max-w-[420px] font-sans text-xs {typeStyles[toast.type]}"
        transition:slide={{ duration: 200 }}
      >
        <span class="text-sm shrink-0 mt-px" aria-hidden="true">{typeIcons[toast.type]}</span>
        <div class="flex-1 min-w-0">
          <div class="font-semibold text-text-heading text-[13px]">{toast.title}</div>
          {#if toast.message}
            <div class="text-text-secondary mt-0.5 leading-relaxed">{toast.message}</div>
          {/if}
        </div>
        <button
          class="text-text-muted hover:text-text-primary text-sm shrink-0 cursor-pointer bg-transparent border-none p-0"
          onclick={() => toastStore.remove(toast.id)}
          aria-label="Dismiss"
        >✕</button>
      </div>
    {/each}
  </div>
</div>
