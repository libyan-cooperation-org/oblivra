<!-- OBLIVRA Web — SeverityIcon (Svelte 5) -->
<script lang="ts">
  export type Severity = 'info' | 'low' | 'medium' | 'high' | 'critical';
  interface Props { severity: Severity; size?: number; class?: string; }
  let { severity, size = 16, class: cls = '' }: Props = $props();

  const colorMap: Record<Severity, string> = {
    info:     'var(--alert-info)',
    low:      'var(--alert-low)',
    medium:   'var(--alert-medium)',
    high:     'var(--alert-high)',
    critical: 'var(--alert-critical)',
  };
  const color = $derived(colorMap[severity] ?? '#607070');
</script>

<svg viewBox="0 0 24 24" width={size} height={size} style="color:{color}" class={cls} aria-hidden="true">
  {#if severity === 'info'}
    <circle cx="12" cy="12" r="10" fill="none" stroke="currentColor" stroke-width="2"/>
    <rect x="11" y="10" width="2" height="7" fill="currentColor"/>
    <circle cx="12" cy="7" r="1.5" fill="currentColor"/>
  {:else if severity === 'low'}
    <path d="M12 21L2 5H22L12 21Z" fill="currentColor"/>
  {:else if severity === 'medium'}
    <rect x="4" y="4" width="16" height="16" fill="currentColor"/>
  {:else if severity === 'high'}
    <path d="M12 3L2 21H22L12 3Z" fill="currentColor"/>
  {:else}
    <path d="M12 2L22 12L12 22L2 12L12 2Z" fill="currentColor"/>
  {/if}
</svg>
