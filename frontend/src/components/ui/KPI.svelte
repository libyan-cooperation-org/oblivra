<!--
  OBLIVRA — KPI Card (Svelte 5)
  Key Performance Indicator card used in dashboards.
-->
<script lang="ts">
  interface Props {
    label: string;
    value: string | number;
    sublabel?: string;
    trend?: 'up' | 'down' | 'stable';
    trendValue?: string;
    variant?: 'default' | 'success' | 'warning' | 'critical' | 'accent';
  }

  let { label, value, sublabel, trend, trendValue, variant = 'default' }: Props = $props();

  const trendIcon: Record<string, string> = {
    up: '↑', down: '↓', stable: '→',
  };
  const trendColor: Record<string, string> = {
    up: 'text-success', down: 'text-error', stable: 'text-text-muted',
  };
  const valueColor: Record<string, string> = {
    default: 'text-text-heading',
    success: 'text-success',
    warning: 'text-warning',
    critical: 'text-error',
    accent: 'text-accent',
  };
</script>

<div role="status" class="bg-surface-1 border border-border-primary rounded-md p-4 flex flex-col gap-1 transition-all duration-fast hover:border-border-hover group">
  <div class="text-[10px] font-bold uppercase tracking-wider text-text-muted font-[var(--font-mono)]">
    {label}
  </div>
  <div class="flex items-end gap-2">
    <span class="text-2xl font-bold {valueColor[variant]} font-[var(--font-ui)] leading-none group-hover:scale-[1.02] transition-transform">
      {value}
    </span>
    {#if trend && trendValue}
      <span class="text-[10px] font-semibold {trendColor[trend]} font-mono pb-0.5">
        {trendIcon[trend]} {trendValue}
      </span>
    {/if}
  </div>
  {#if sublabel}
    <div class="text-[10px] text-text-muted font-[var(--font-ui)] mt-0.5">{sublabel}</div>
  {/if}
</div>
