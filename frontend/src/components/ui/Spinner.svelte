<!--
  OBLIVRA — Spinner (Svelte 5)
  Loading spinner with optional text.
-->
<script lang="ts">
  interface Props {
    size?: 'sm' | 'md' | 'lg';
    label?: string;
  }

  let { size = 'md', label }: Props = $props();

  const sizeClasses: Record<string, string> = {
    sm: 'w-4 h-4 border',
    md: 'w-7 h-7 border-2',
    lg: 'w-10 h-10 border-2',
  };
</script>

<div class="flex flex-col items-center justify-center gap-3">
  <div class="{sizeClasses[size]} border-border-secondary border-t-accent rounded-full animate-spin"></div>
  {#if label}
    <span class="text-text-muted text-[11px] font-mono uppercase tracking-wider">{label}</span>
  {/if}
</div>
