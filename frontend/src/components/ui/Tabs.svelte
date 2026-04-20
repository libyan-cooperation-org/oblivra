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

<div class="tab-group {variant === 'pills' ? 'gap-1' : ''}" class:border-b={variant === 'default'}>
  {#each tabs as tab}
    <button
      class="tab {active === tab.id ? 'active' : ''}"
      onclick={() => select(tab.id)}
    >
      {#if tab.icon}
        <span class="text-[1.1em]">{tab.icon}</span>
      {/if}
      {tab.label}
      {#if tab.badge !== undefined}
        <span class="badge">
          {tab.badge > 99 ? '99+' : tab.badge}
        </span>
      {/if}
    </button>
  {/each}
</div>
