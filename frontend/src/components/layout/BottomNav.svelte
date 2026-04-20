<!--
  OBLIVRA — BottomNav (Svelte 5)
  Mobile-specific navigation bar for field ops.
-->
<script lang="ts">
  import { appStore } from '@lib/stores/app.svelte';
  import { push } from '@lib/router.svelte';
  import { LayoutDashboard, ShieldAlert, Search, Monitor, MoreHorizontal } from 'lucide-svelte';

  const navItems = [
    { id: 'dashboard', icon: LayoutDashboard, label: 'DASH', path: '/dashboard' },
    { id: 'alerts', icon: ShieldAlert, label: 'ALERTS', path: '/alerts' },
    { id: 'siem', icon: Search, label: 'SIEM', path: '/siem' },
    { id: 'fleet', icon: Monitor, label: 'FLEET', path: '/fleet' },
    { id: 'settings', icon: MoreHorizontal, label: 'MORE', path: '/settings' },
  ];

  function go(item: typeof navItems[0]) {
    appStore.setActiveNavTab(item.id as any);
    push(item.path);
  }
</script>

<nav class="h-[52px] bg-surface-1 border-t border-border-primary flex items-center justify-around px-2 shrink-0 md:hidden pb-safe">
  {#each navItems as item}
    <button 
      class="flex flex-col items-center gap-1 p-2 rounded-sm transition-colors {appStore.activeNavTab === item.id ? 'text-accent' : 'text-text-muted hover:text-text-secondary'}"
      onclick={() => go(item)}
    >
      <item.icon size={18} class={appStore.activeNavTab === item.id ? 'text-accent' : ''} />
      <span class="text-[7px] font-mono font-bold tracking-tighter uppercase">{item.label}</span>
    </button>
  {/each}
</nav>

<style>
  .pb-safe {
    padding-bottom: env(safe-area-inset-bottom, 0px);
  }
</style>
