<!--
  OBLIVRA — Badge (Svelte 5)
  Inline status badge / pill with severity variants.
-->
<script lang="ts">
  import type { Snippet } from 'svelte';

  interface Props {
    variant?: 'info' | 'success' | 'warning' | 'critical' | 'muted' | 'accent' | 'danger' | 'secondary' | 'primary';
    size?: 'xs' | 'sm';
    dot?: boolean;
    class?: string;
    children: Snippet;
  }

  let { variant = 'muted', size = 'xs', dot = false, class: className = '', children }: Props = $props();

  const variantClasses: Record<string, string> = {
    info:     'bg-accent/10 text-accent border-accent/20',
    success:  'bg-success/10 text-success border-success/20',
    warning:  'bg-warning/10 text-warning border-warning/20',
    critical: 'bg-error/10 text-error border-error/20',
    muted:    'bg-surface-3 text-text-muted border-border-primary',
    accent:   'bg-accent/15 text-accent border-accent/30',
    danger:   'bg-error/10 text-error border-error/20',
    secondary: 'bg-surface-3 text-text-muted border-border-primary',
  };

  const dotColors: Record<string, string> = {
    info: 'bg-accent', success: 'bg-success', warning: 'bg-warning',
    critical: 'bg-error', muted: 'bg-text-muted', accent: 'bg-accent',
    danger: 'bg-error', secondary: 'bg-text-muted',
  };

  const sizeClasses: Record<string, string> = {
    xs: 'text-[9px] px-1.5 py-px',
    sm: 'text-[10px] px-2 py-0.5',
  };
</script>

<span role="status" class="inline-flex items-center gap-1 font-[var(--font-mono)] font-bold uppercase tracking-wider border rounded-xs whitespace-nowrap {variantClasses[variant]} {sizeClasses[size]} {className}">
  {#if dot}
    <span class="w-1.5 h-1.5 rounded-full shrink-0 {dotColors[variant]}"></span>
  {/if}
  {@render children()}
</span>
