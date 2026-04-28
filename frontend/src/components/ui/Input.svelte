<!--
  OBLIVRA — Input (Svelte 5)
  Text input with label, icon, and error state.
-->
<script lang="ts">
  import type { HTMLInputAttributes } from 'svelte/elements';

  interface Props extends Omit<HTMLInputAttributes, 'value'> {
    value: any;
    label?: string;
    error?: string;
    icon?: string;
    variant?: 'default' | 'search';
  }

  let {
    value = $bindable(),
    label,
    error,
    icon,
    variant = 'default',
    class: className = '',
    ...restProps
  }: Props = $props();

  const id = `input-${Math.random().toString(36).substring(2, 9)}`;
</script>

<div class="flex flex-col gap-1 {className}">
  {#if label}
    <label for={id} class="text-[10px] font-bold uppercase tracking-wider text-text-muted font-sans">
      {label}
    </label>
  {/if}

  <div class="relative">
    {#if icon}
      <span class="absolute left-2 top-1/2 -translate-y-1/2 text-text-muted text-[11px] pointer-events-none">
        {icon}
      </span>
    {/if}

    <input
      {id}
      bind:value
      class="w-full bg-surface-0 border rounded-sm text-text-primary font-sans text-xs outline-none transition-all duration-fast
        placeholder:text-text-muted
        focus:border-accent focus:shadow-glow
        disabled:opacity-40 disabled:cursor-not-allowed
        {icon ? 'pl-7' : 'px-2.5'}
        {error ? 'border-error shadow-glow-danger' : 'border-border-primary'}
        {variant === 'search' ? 'h-7 bg-surface-1' : 'h-7'}"
      {...restProps}
    />
  </div>

  {#if error}
    <span class="text-[10px] text-error font-sans">{error}</span>
  {/if}
</div>
