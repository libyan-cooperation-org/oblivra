<!--
  OBLIVRA — Button (Svelte 5)
  Primary UI button with variants and sizes.
-->
<script lang="ts">
  import type { HTMLButtonAttributes } from 'svelte/elements';

  interface Props extends HTMLButtonAttributes {
    variant?: 'primary' | 'secondary' | 'ghost' | 'danger' | 'cta';
    size?: 'sm' | 'md' | 'lg';
    loading?: boolean;
    icon?: string;
  }

  let {
    variant = 'secondary',
    size = 'md',
    loading = false,
    icon,
    disabled,
    children,
    class: className = '',
    ...restProps
  }: Props = $props();

  const variantClasses: Record<string, string> = {
    primary: 'bg-accent text-white hover:bg-accent-hover border-accent',
    secondary: 'bg-surface-2 text-text-secondary hover:bg-surface-3 hover:text-text-primary border-border-primary',
    ghost: 'bg-transparent text-text-secondary hover:bg-surface-3 hover:text-text-primary border-transparent',
    danger: 'bg-error/10 text-error hover:bg-error/20 border-error/30',
    cta: 'bg-accent-cta text-white hover:bg-accent-cta-hover border-accent-cta',
  };

  const sizeClasses: Record<string, string> = {
    sm: 'h-6 px-2 text-[10px] gap-1 rounded-xs',
    md: 'h-7 px-3 text-[11px] gap-1.5 rounded-sm',
    lg: 'h-8 px-4 text-xs gap-2 rounded-sm',
  };
</script>

<button
  class="inline-flex items-center justify-center font-[var(--font-ui)] font-semibold border cursor-pointer transition-all duration-fast whitespace-nowrap select-none
    disabled:opacity-40 disabled:cursor-not-allowed disabled:pointer-events-none
    {variantClasses[variant]} {sizeClasses[size]} {className}"
  disabled={disabled || loading}
  {...restProps}
>
  {#if loading}
    <span class="w-3 h-3 border border-current border-t-transparent rounded-full animate-spin shrink-0"></span>
  {:else if icon}
    <span class="text-[1.1em] shrink-0">{icon}</span>
  {/if}
  {@render children?.()}
</button>
