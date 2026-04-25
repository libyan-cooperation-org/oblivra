<!-- OBLIVRA — Badge v2 — token-aligned, adds high/med severity variants -->
<script lang="ts">
  import type { Snippet } from 'svelte';

  interface Props {
    /**
     * critical  → error red
     * high      → orange
     * med       → amber
     * info      → accent blue
     * success   → green
     * warning   → orange (alias for high)
     * muted     → surface-3, dim text
     * accent    → blue highlight
     * purple    → for desktop/vault contexts
     */
    variant?: 'info' | 'success' | 'warning' | 'critical' | 'high' | 'med' | 'muted' | 'accent' | 'purple';
    size?: 'xs' | 'sm';
    dot?: boolean;
    pulse?: boolean;
    class?: string;
    children: Snippet;
  }

  let { variant = 'muted', size = 'xs', dot = false, pulse = false, class: className = '', children }: Props = $props();

  // All token-based — no raw Tailwind color names
  const variantStyle: Record<string, string> = {
    info:     'background:rgba(24,120,200,0.1);   color:var(--ac2);  border-color:rgba(24,120,200,0.22);',
    success:  'background:rgba(20,120,72,0.1);    color:var(--ok2);  border-color:rgba(20,120,72,0.22);',
    warning:  'background:rgba(184,96,0,0.12);    color:var(--hi2);  border-color:rgba(184,96,0,0.25);',
    high:     'background:rgba(184,96,0,0.12);    color:var(--hi2);  border-color:rgba(184,96,0,0.25);',
    med:      'background:rgba(154,128,0,0.1);    color:var(--md2);  border-color:rgba(154,128,0,0.2);',
    critical: 'background:rgba(192,40,40,0.12);   color:var(--cr2);  border-color:rgba(192,40,40,0.28);',
    muted:    'background:var(--s3);               color:var(--tx2);  border-color:var(--b1);',
    accent:   'background:rgba(24,120,200,0.15);  color:var(--ac2);  border-color:rgba(24,120,200,0.3);',
    purple:   'background:rgba(112,80,192,0.1);   color:var(--pu2);  border-color:rgba(112,80,192,0.22);',
  };

  const dotColor: Record<string, string> = {
    info: 'var(--ac2)', success: 'var(--ok2)', warning: 'var(--hi2)',
    high: 'var(--hi2)', med: 'var(--md2)', critical: 'var(--cr2)',
    muted: 'var(--tx3)', accent: 'var(--ac2)', purple: 'var(--pu2)',
  };

  const sizeClasses: Record<string, string> = {
    xs: 'text-[8px] px-1.5 py-px',
    sm: 'text-[10px] px-2 py-0.5',
  };
</script>

<span
  role="status"
  class="inline-flex items-center gap-1 font-mono font-bold uppercase tracking-wider border rounded-sm whitespace-nowrap {sizeClasses[size]} {className}"
  style="{variantStyle[variant]}"
>
  {#if dot}
    <span
      class="w-1.5 h-1.5 rounded-full shrink-0"
      style="background:{dotColor[variant]}; {pulse ? 'animation:var(--animate-dot-pulse);' : ''}"
    ></span>
  {/if}
  {@render children()}
</span>
