<!--
  OBLIVRA — Toast Container (Svelte 5)
  Renders notification toasts from the toast store.
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
</script>

<div class="fixed top-3 right-3 z-[10000] flex flex-col gap-2 pointer-events-none">
  {#each toastStore.items as toast (toast.id)}
    <div
      class="pointer-events-auto flex items-start gap-3 px-4 py-3 rounded-md border shadow-lg backdrop-blur-sm min-w-[300px] max-w-[420px] font-sans text-xs {typeStyles[toast.type]}"
      transition:slide={{ duration: 200 }}
    >
      <span class="text-sm shrink-0 mt-px">{typeIcons[toast.type]}</span>
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
      >
        ✕
      </button>
    </div>
  {/each}
</div>
