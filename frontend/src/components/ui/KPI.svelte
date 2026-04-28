<!-- OBLIVRA — KPI Card v2.1 — token-aligned colours + explicit trend polarity.

  UIUX_IMPROVEMENTS.md P1 #9: the previous default treated up = bad (red),
  down = good (green) which is correct for *risk* but wrong for EPS,
  uptime, throughput, and any other "more is better" metric. Callers
  must now pass `trendPolarity` explicitly so the colour reflects intent.

  Labels also bumped from text-[8px] (sub-pixel rendered, fails WCAG floor
  for sustained use) to var(--fs-micro) which is 10 px in the new
  comfortable density and 9 px in compact. See P0 #2.
-->
<script lang="ts">
  interface Props {
    label: string;
    value: string | number;
    sublabel?: string;
    trend?: 'up' | 'down' | 'stable';
    trendValue?: string;
    /**
     * Polarity of the trend arrow.
     *  - 'up-bad'  : up arrow renders red, down arrow renders green
     *                (default — risk score, threat count, error rate)
     *  - 'up-good' : up arrow renders green, down arrow renders red
     *                (EPS, uptime, throughput, healthy hosts)
     *  - 'neutral' : muted regardless of direction (count of items)
     */
    trendPolarity?: 'up-bad' | 'up-good' | 'neutral';
    /**
     * default   → neutral heading colour
     * critical  → error red
     * high      → warning orange
     * med       → amber
     * success   → green
     * accent    → blue
     * muted     → de-emphasised (use for "count" tiles that are not signals)
     */
    variant?: 'default' | 'success' | 'warning' | 'critical' | 'accent' | 'high' | 'med' | 'muted' | 'info';
    /** 0–100: drives the 2px bottom bar fill */
    progress?: number;
  }

  let {
    label,
    value,
    sublabel,
    trend,
    trendValue,
    trendPolarity = 'up-bad',
    variant = 'default',
    progress,
  }: Props = $props();

  const trendIcon: Record<string, string> = { up: '▲', down: '▼', stable: '●' };

  // Resolve the trend colour by combining direction with the caller-
  // declared polarity. `neutral` always returns a muted colour.
  function resolveTrendStyle(t: 'up' | 'down' | 'stable', polarity: Props['trendPolarity']): string {
    if (polarity === 'neutral' || t === 'stable') return 'color:var(--tx3)';
    if (polarity === 'up-good') {
      return t === 'up' ? 'color:var(--ok2)' : 'color:var(--cr2)';
    }
    // 'up-bad' (default) — security/risk semantics
    return t === 'up' ? 'color:var(--cr2)' : 'color:var(--ok2)';
  }

  const barStyle: Record<string, string> = {
    default:  'background:var(--b2)',
    success:  'background:var(--ok2)',
    warning:  'background:var(--md2)',
    critical: 'background:var(--cr2)',
    high:     'background:var(--hi2)',
    med:      'background:var(--md2)',
    accent:   'background:var(--ac)',
    muted:    'background:var(--b2)',
    info:     'background:var(--ac2)',
  };

  const valueStyle: Record<string, string> = {
    default:  'color:var(--tx)',
    success:  'color:var(--ok2)',
    warning:  'color:var(--md2)',
    critical: 'color:var(--cr2)',
    high:     'color:var(--hi2)',
    med:      'color:var(--md2)',
    accent:   'color:var(--ac2)',
    muted:    'color:var(--tx2)',
    info:     'color:var(--ac2)',
  };
</script>

<div
  role="status"
  class="bg-surface-2 border border-border-primary rounded-sm p-2.5 flex flex-col gap-1
         transition-colors hover:border-border-hover relative overflow-hidden"
>
  <!-- Label — uses the typography ramp so density toggle scales it. -->
  <div
    class="font-mono font-bold uppercase tracking-[0.12em] text-text-muted"
    style="font-size: var(--fs-micro, 10px);"
  >
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
      <span class="font-mono pb-px" style="font-size: var(--fs-micro, 10px); {resolveTrendStyle(trend, trendPolarity)}">
        {trendIcon[trend]} {trendValue}
      </span>
    {/if}
  </div>

  <!-- Sublabel -->
  {#if sublabel}
    <div class="font-sans text-text-muted opacity-65 leading-tight" style="font-size: var(--fs-micro, 10px);">{sublabel}</div>
  {/if}

  <!-- Progress bar — 2px, bottom-anchored -->
  {#if progress !== undefined}
    <div
      class="absolute bottom-0 left-0 h-[2px] rounded-none transition-all"
      style="width:{progress}%; {barStyle[variant]}"
    ></div>
  {/if}
</div>
