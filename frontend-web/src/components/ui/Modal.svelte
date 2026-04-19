<!--
  OBLIVRA — Modal (Svelte 5)
  Overlay modal with title, body, and footer slots.
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

  function handleKeyDown(e: KeyboardEvent) {
    if (e.key === 'Escape' && open) onClose();
  }

  onMount(() => window.addEventListener('keydown', handleKeyDown));
  onDestroy(() => window.removeEventListener('keydown', handleKeyDown));
</script>

{#if open}
  <!-- Backdrop -->
  <div
    class="fixed inset-0 bg-black/55 z-[9000] flex items-start justify-center pt-[10vh]"
    transition:fade={{ duration: 150 }}
    role="button"
    tabindex="-1"
    onclick={onClose}
    onkeydown={(e) => e.key === 'Escape' && onClose()}
  >
    <!-- Modal container -->
    <div
      class="w-full {sizeClasses[size]} bg-surface-2 border border-border-secondary rounded-lg shadow-xl overflow-hidden"
      transition:scale={{ duration: 200, start: 0.95 }}
      role="dialog"
      aria-modal="true"
      tabindex="-1"
      onclick={(e) => e.stopPropagation()}
      onkeydown={(e) => e.stopPropagation()}
    >
      <!-- Header -->
      {#if title}
        <div class="flex items-center justify-between px-5 py-3.5 border-b border-border-primary">
          <h2 class="text-sm font-bold text-text-heading font-[var(--font-ui)]">{title}</h2>
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
