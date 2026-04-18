<!--
  OBLIVRA — PageLayout (Svelte 5)
  Standard page wrapper with header, toolbar, and scrollable content area.
  Used by every page for consistent layout.
-->
<script lang="ts">
  import type { Snippet } from 'svelte';

  interface Props {
    title: string;
    subtitle?: string;
    toolbar?: Snippet;
    children: Snippet;
  }

  let { title, subtitle, toolbar, children }: Props = $props();
</script>

<div class="flex flex-col h-full overflow-hidden bg-surface-0">
  <!-- Page header -->
  <div class="flex items-center justify-between px-6 py-4 border-b border-border-primary shrink-0">
    <div>
      <h1 class="text-lg font-bold text-text-heading font-[var(--font-ui)]">{title}</h1>
      {#if subtitle}
        <p class="text-[11px] text-text-muted font-mono mt-0.5">{subtitle}</p>
      {/if}
    </div>
    {#if toolbar}
      <div class="flex items-center gap-2">
        {@render toolbar()}
      </div>
    {/if}
  </div>

  <!-- Scrollable content -->
  <div class="flex-1 overflow-auto p-6">
    {@render children()}
  </div>
</div>
