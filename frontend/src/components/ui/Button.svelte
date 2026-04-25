<!-- OBLIVRA — Button v2 — fixed cta variant, consistent token usage -->
<script lang="ts">
  import type { HTMLButtonAttributes } from 'svelte/elements';

  interface Props extends HTMLButtonAttributes {
    /**
     * primary   → solid accent blue
     * secondary → surface-2, muted text (default)
     * ghost     → transparent, no border
     * danger    → error red
     * warning   → amber/orange
     * cta       → solid accent (was broken — now same as primary with stronger saturation)
     */
    variant?: 'primary' | 'secondary' | 'ghost' | 'danger' | 'warning' | 'cta';
    size?: 'xs' | 'sm' | 'md' | 'lg';
    loading?: boolean;
    icon?: any;
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

  // All variants use design tokens — no Tailwind color names
  const variantStyle: Record<string, string> = {
    primary:   'background:rgba(24,120,200,0.15); color:var(--ac2); border-color:rgba(24,120,200,0.35);',
    secondary: 'background:var(--s2); color:var(--tx2); border-color:var(--b1);',
    ghost:     'background:transparent; color:var(--tx2); border-color:transparent;',
    danger:    'background:rgba(192,40,40,0.12); color:var(--cr2); border-color:rgba(192,40,40,0.3);',
    warning:   'background:rgba(184,96,0,0.12); color:var(--hi2); border-color:rgba(184,96,0,0.25);',
    cta:       'background:rgba(24,120,200,0.22); color:var(--ac2); border-color:rgba(24,120,200,0.5); font-weight:600;',
  };

  const variantHover: Record<string, string> = {
    primary:   'hover:bg-accent/25 hover:border-accent',
    secondary: 'hover:bg-surface-3 hover:text-text-primary hover:border-border-secondary',
    ghost:     'hover:bg-surface-3 hover:text-text-primary',
    danger:    'hover:bg-error/20',
    warning:   'hover:bg-warning/20',
    cta:       'hover:bg-accent/30',
  };

  const sizeClasses: Record<string, string> = {
    xs: 'h-5 px-1.5 text-[9px]  gap-1   rounded-sm',
    sm: 'h-6 px-2   text-[10px] gap-1   rounded-sm',
    md: 'h-7 px-3   text-[11px] gap-1.5 rounded-sm',
    lg: 'h-8 px-4   text-xs     gap-2   rounded-sm',
  };
</script>

<button
  class="inline-flex items-center justify-center font-[var(--font-mono)] border cursor-pointer
         transition-all duration-[100ms] whitespace-nowrap select-none
         disabled:opacity-35 disabled:cursor-not-allowed disabled:pointer-events-none
         focus-visible:ring-0
         {variantHover[variant]}
         {sizeClasses[size]}
         {className}"
  style="letter-spacing:0.05em; {variantStyle[variant]}"
  disabled={disabled || loading}
  {...restProps}
>
  {#if loading}
    <span class="w-3 h-3 border border-current border-t-transparent rounded-full animate-spin shrink-0"></span>
  {:else if icon}
    <span class="shrink-0 flex items-center justify-center" style="font-size:1.1em;">
      {#if typeof icon === 'string'}
        {icon}
      {:else}
        {@const Icon = icon}
        <Icon size={size === 'xs' ? 10 : 12} />
      {/if}
    </span>
  {/if}
  {@render children?.()}
</button>
