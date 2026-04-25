<!-- OBLIVRA — KPI Card v2 — correct metric font size, token-aligned colors -->
<script lang="ts">
  interface Props {
    label: string;
    value: string | number;
    sublabel?: string;
    trend?: 'up' | 'down' | 'stable';
    trendValue?: string;
    /**
     * default   → neutral heading color
     * critical  → error red
     * high      → warning orange
     * med       → amber
     * success   → green
     * accent    → blue
     */
    variant?: 'default' | 'success' | 'warning' | 'critical' | 'accent' | 'high' | 'med';
    /** 0–100: drives the 2px bottom bar fill */
    progress?: number;
  }

  let { label, value, sublabel, trend, trendValue, variant = 'default', progress }: Props = $props();

  const trendIcon: Record<string, string>  = { up: '▲', down: '▼', stable: '●' };

  // trend color: up = bad (red), down = good (green) for security metrics
  const trendStyle: Record<string, string> = {
    up:     'color:var(--cr2)',
    down:   'color:var(--ok2)',
    stable: 'color:var(--tx3)',
  };

  const barStyle: Record<string, string> = {
    default:  'background:var(--b2)',
    success:  'background:var(--ok2)',
    warning:  'background:var(--md2)',
    critical: 'background:var(--cr2)',
    high:     'background:var(--hi2)',
    med:      'background:var(--md2)',
    accent:   'background:var(--ac)',
  };

  const valueStyle: Record<string, string> = {
    default:  'color:var(--tx)',
    success:  'color:var(--ok2)',
    warning:  'color:var(--md2)',
    critical: 'color:var(--cr2)',
    high:     'color:var(--hi2)',
    med:      'color:var(--md2)',
    accent:   'color:var(--ac2)',
  };
</script>

<div
  role="status"
  class="bg-surface-2 border border-border-primary rounded-sm p-2.5 flex flex-col gap-1
         transition-colors hover:border-border-hover relative overflow-hidden"
>
  <!-- Label -->
  <div class="font-mono text-[8px] font-bold uppercase tracking-[0.12em] text-text-muted">
    {label}
  </div>

  <!-- Value row -->
  <div class="flex items-end gap-1.5">
    <!-- Value: 20px mono — not text-2xl (which is 24px and too large at 11px base) -->
    <span
      class="font-mono font-bold leading-none tracking-tight tabular"
      style="font-size:20px; {valueStyle[variant]}"
    >{value}</span>

    {#if trend && trendValue}
      <span class="font-mono text-[9px] pb-px" style="{trendStyle[trend]}">
        {trendIcon[trend]} {trendValue}
      </span>
    {/if}
  </div>

  <!-- Sublabel -->
  {#if sublabel}
    <div class="font-sans text-[9px] text-text-muted opacity-65 leading-tight">{sublabel}</div>
  {/if}

  <!-- Progress bar — 2px, bottom-anchored -->
  {#if progress !== undefined}
    <div
      class="absolute bottom-0 left-0 h-[2px] rounded-none transition-all"
      style="width:{progress}%; {barStyle[variant]}"
    ></div>
  {/if}
</div>
