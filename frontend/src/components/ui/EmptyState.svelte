<!--
  OBLIVRA — EmptyState (Svelte 5)
  Shown when a list or panel has no data.
-->
<script lang="ts">
  import type { Snippet } from 'svelte';

  interface Props {
    icon?: string;
    title: string;
    description?: string;
    action?: Snippet;
  }

  let { icon = '∅', title, description, action }: Props = $props();
</script>

<div class="flex flex-col items-center justify-center py-12 px-6 text-center gap-3">
  <div class="w-12 h-12 rounded-lg bg-surface-2 border border-border-primary flex items-center justify-center text-xl text-text-muted">
    {icon}
  </div>
  <div class="text-sm font-semibold text-text-heading font-[var(--font-ui)]">{title}</div>
  {#if description}
    <div class="text-xs text-text-muted max-w-sm leading-relaxed font-[var(--font-ui)]">{description}</div>
  {/if}
  {#if action}
    <div class="mt-2">
      {@render action()}
    </div>
  {/if}
</div>
