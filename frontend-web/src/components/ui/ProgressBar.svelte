<!--
  OBLIVRA — ProgressBar (Svelte 5)
  Tactical progress indicator.
-->
<script lang="ts">
  interface Props {
    value?: number;
    max?: number;
    variant?: 'primary' | 'success' | 'warning' | 'error' | 'info';
    size?: 'xs' | 'sm' | 'md';
    height?: string;
    color?: string;
    class?: string;
  }

  let { 
    value = 0, 
    max = 100, 
    variant = 'primary', 
    size = 'sm', 
    height,
    color,
    class: className = '' 
  }: Props = $props();

  const percentage = $derived(Math.min(Math.max((value / max) * 100, 0), 100));

  const variantClasses = {
    primary: 'bg-accent',
    success: 'bg-success',
    warning: 'bg-warning',
    error: 'bg-error',
    info: 'bg-info'
  };

  const sizeClasses = {
    xs: 'h-0.5',
    sm: 'h-1',
    md: 'h-2'
  };
</script>

<div class="w-full bg-surface-3 rounded-full overflow-hidden {sizeClasses[size]} {className}" style={height ? `height: ${height}` : ''}>
  <div 
    class="h-full transition-all duration-slow {variantClasses[variant]}" 
    style="width: {percentage}%; {color ? `background-color: ${color}` : ''}"
  ></div>
</div>
