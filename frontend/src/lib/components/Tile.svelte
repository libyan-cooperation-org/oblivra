<script lang="ts">
  let {
    label,
    value,
    hint,
    accent = false,
    warn = false,
    danger = false,
  }: {
    label: string;
    value: string | number;
    hint?: string;
    accent?: boolean;
    warn?: boolean;
    danger?: boolean;
  } = $props();

  const accentColor = $derived(
    danger ? 'var(--color-sig-error)' :
    warn   ? 'var(--color-sig-warn)' :
    accent ? 'var(--color-cyan-500)' :
             'var(--color-cyan-500)'
  );
</script>

<div
  class="bracket-card relative overflow-hidden"
  style="
    padding: 14px 16px 12px;
    border: 1px solid var(--color-base-700);
    background: linear-gradient(135deg, rgba(11,16,23,0.95) 0%, rgba(14,20,30,0.95) 100%);
  "
>
  <!-- Top accent bar -->
  <div style="
    position: absolute;
    top: 0; left: 0; right: 0;
    height: 1px;
    background: linear-gradient(90deg, transparent, {accentColor}, transparent);
    opacity: 0.6;
  "></div>

  <!-- Label -->
  <div style="
    font-family: 'Share Tech Mono', monospace;
    font-size: 9px;
    letter-spacing: 2px;
    text-transform: uppercase;
    color: var(--color-base-300);
    margin-bottom: 6px;
  ">{label}</div>

  <!-- Value -->
  <div style="
    font-family: 'JetBrains Mono', monospace;
    font-size: 24px;
    font-weight: 500;
    line-height: 1;
    color: {accentColor};
    text-shadow: 0 0 20px {accentColor}40;
    letter-spacing: -0.5px;
  ">{value}</div>

  {#if hint}
    <div style="
      margin-top: 6px;
      font-family: 'Share Tech Mono', monospace;
      font-size: 9px;
      letter-spacing: 1px;
      color: var(--color-base-300);
    ">{hint}</div>
  {/if}

  <!-- Background grid texture -->
  <div style="
    position: absolute;
    inset: 0;
    pointer-events: none;
    opacity: 0.03;
    background-image: linear-gradient(var(--color-base-200) 1px, transparent 1px),
                      linear-gradient(90deg, var(--color-base-200) 1px, transparent 1px);
    background-size: 16px 16px;
  "></div>
</div>
