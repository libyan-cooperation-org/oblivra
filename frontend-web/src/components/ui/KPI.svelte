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
    variant?: 'default' | 'success' | 'warning' | 'critical' | 'accent' | 'high' | 'med';
    progress?: number; // 0-100 for bottom bar
  }

  let { label, value, sublabel, trend, trendValue, variant = 'default', progress }: Props = $props();

  const trendIcon: Record<string, string> = {
    up: '▲', down: '▼', stable: '●',
  };
  const trendColor: Record<string, string> = {
    up: 'text-error', down: 'text-success', stable: 'text-text-muted',
  };
  const barColor: Record<string, string> = {
    default: 'bg-border-primary',
    success: 'bg-success',
    warning: 'bg-warning',
    critical: 'bg-error',
    high: 'bg-warning', // mapped to warning for consistency
    med: 'bg-warning/70',
    accent: 'bg-accent',
  };
  const valueColor: Record<string, string> = {
    default: 'text-text-heading',
    success: 'text-success',
    warning: 'text-warning',
    critical: 'text-error',
    high: 'text-warning',
    med: 'text-warning/90',
    accent: 'text-accent',
  };
</script>

<div role="status" class="bg-surface-2 border border-border-primary rounded-sm p-3 flex flex-col gap-1 transition-all duration-fast hover:border-border-hover group relative overflow-hidden">
  <div class="text-[9px] font-mono font-bold uppercase tracking-widest text-text-muted">
    {label}
  </div>
  <div class="flex items-end gap-2">
    <span class="text-2xl font-bold {valueColor[variant]} font-mono leading-none tracking-tight">
      {value}
    </span>
    {#if trend && trendValue}
      <span class="text-[9px] font-mono {trendColor[trend]} pb-0.5">
        {trendIcon[trend]} {trendValue}
      </span>
    {/if}
  </div>
  {#if sublabel}
    <div class="text-[9px] text-text-muted font-sans mt-0.5 opacity-70">{sublabel}</div>
  {/if}
  
  {#if progress !== undefined}
    <div class="absolute bottom-0 left-0 h-0.5 {barColor[variant]}" style="width: {progress}%"></div>
  {/if}
</div>
