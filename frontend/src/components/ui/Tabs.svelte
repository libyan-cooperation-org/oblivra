<!--
  OBLIVRA — Tabs (Svelte 5)
  Horizontal tab bar with content switching.
-->
<script lang="ts">
  import type { Snippet } from 'svelte';

  export interface TabItem {
    id: string;
    label: string;
    icon?: string;
    badge?: number;
  }

  interface Props {
    tabs: TabItem[];
    active?: string;
    onChange?: (id: string) => void;
    variant?: 'default' | 'pills';
  }

  let {
    tabs,
    active = $bindable(tabs[0]?.id ?? ''),
    onChange,
    variant = 'default',
  }: Props = $props();

  function select(id: string) {
    active = id;
    onChange?.(id);
  }
</script>

<div class="flex items-center gap-0 border-b border-border-primary shrink-0"
  class:gap-1={variant === 'pills'}
  class:border-b-0={variant === 'pills'}
  class:p-1={variant === 'pills'}
>
  {#each tabs as tab}
    <button
      class="relative flex items-center gap-1.5 bg-transparent border-none cursor-pointer font-[var(--font-ui)] font-medium whitespace-nowrap transition-all duration-fast
        {variant === 'default'
          ? `px-3.5 h-8 text-[11px] border-b-2 ${active === tab.id ? 'text-accent border-accent-cta' : 'text-text-muted border-transparent hover:text-text-secondary hover:bg-surface-3/30'}`
          : `px-2.5 py-1 text-[10px] rounded-sm ${active === tab.id ? 'bg-accent/15 text-accent font-semibold' : 'text-text-muted hover:text-text-secondary hover:bg-surface-3'}`
        }"
      onclick={() => select(tab.id)}
    >
      {#if tab.icon}
        <span class="text-[1.1em]">{tab.icon}</span>
      {/if}
      {tab.label}
      {#if tab.badge && tab.badge > 0}
        <span class="text-[7px] font-extrabold bg-accent text-surface-0 rounded-full px-1 py-px font-mono min-w-[14px] text-center">
          {tab.badge > 99 ? '99+' : tab.badge}
        </span>
      {/if}
    </button>
  {/each}
</div>
