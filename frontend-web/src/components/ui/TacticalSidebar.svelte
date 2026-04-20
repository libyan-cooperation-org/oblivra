<!-- OBLIVRA — Tactical Sidebar (Svelte 5) -->
<script lang="ts">
  import { push, router } from '../../core/router.svelte';
  import { 
    LayoutDashboard, 
    Monitor, 
    Search, 
    ShieldAlert, 
    Users, 
    Shield, 
    Database, 
    Zap,
    Terminal,
    Settings,
    ChevronLeft,
    ChevronRight,
    Lock
  } from 'lucide-svelte';
  import { fade } from 'svelte/transition';

  let collapsed = $state(false);

  const NAV_ITEMS = [
    { label: 'WAR ROOM', icon: LayoutDashboard, path: '/' },
    { label: 'FLEET', icon: Monitor, path: '/fleet' },
    { label: 'SEARCH', icon: Search, path: '/siem/search' },
    { label: 'ALERTS', icon: ShieldAlert, path: '/alerts' },
    { label: 'FUSION', icon: Zap, path: '/fusion' },
    { label: 'IDENTITY', icon: Users, path: '/identity' },
    { label: 'INVESTIGATION', icon: Shield, path: '/investigation' },
    { label: 'VAULT', icon: Lock, path: '/evidence' },
  ];

  const SECONDARY_ITEMS = [
    { label: 'LOOKUPS', icon: Database, path: '/lookups' },
    { label: 'SYSTEM', icon: Settings, path: '/regulator' },
  ];

  function navigate(path: string) {
    push(path);
  }
</script>

<aside 
  class="h-screen bg-surface-1 border-r border-border-primary flex flex-col transition-all duration-300 ease-in-out shadow-2xl z-40 shrink-0"
  class:w-64={!collapsed}
  class:w-16={collapsed}
>
  <!-- Brand -->
  <div class="h-16 flex items-center px-4 border-b border-border-primary shrink-0 overflow-hidden bg-surface-2/50 backdrop-blur-sm">
    <div class="w-8 h-8 bg-accent-primary flex items-center justify-center font-black italic text-black shrink-0">O</div>
    {#if !collapsed}
      <div class="ml-3 flex flex-col" transition:fade={{ duration: 150 }}>
        <span class="text-sm font-black tracking-tighter text-text-heading italic">OBLIVRA</span>
        <span class="text-[8px] font-mono text-text-muted tracking-[0.2em]">TACTICAL_OS_v3</span>
      </div>
    {/if}
  </div>

  <!-- Primary Navigation -->
  <nav class="flex-1 overflow-y-auto overflow-x-hidden p-2 space-y-1 py-4">
    {#each NAV_ITEMS as item}
      {@const active = router.path === item.path}
      <button 
        onclick={() => navigate(item.path)}
        class="w-full flex items-center gap-3 px-3 py-2.5 rounded-sm transition-all group relative
          {active ? 'bg-accent-primary/10 text-accent-primary' : 'text-text-muted hover:bg-surface-2 hover:text-text-heading'}"
        title={collapsed ? item.label : ''}
      >
        <item.icon size={18} class="shrink-0 {active ? 'text-accent-primary' : 'text-text-muted group-hover:text-text-heading'} transition-colors" />
        {#if !collapsed}
          <span class="text-[11px] font-black uppercase tracking-widest text-left" transition:fade={{ duration: 100 }}>{item.label}</span>
          {#if active}
             <div class="absolute right-2 w-1.5 h-1.5 bg-accent-primary rounded-full shadow-[0_0_8px_var(--accent-primary)]"></div>
          {/if}
        {/if}
      </button>
    {/each}

    <div class="my-6 border-t border-border-primary/30 mx-2"></div>

    {#each SECONDARY_ITEMS as item}
      {@const active = router.path === item.path}
      <button 
        onclick={() => navigate(item.path)}
        class="w-full flex items-center gap-3 px-3 py-2 rounded-sm transition-all group
          {active ? 'bg-surface-3 text-text-heading' : 'text-text-muted hover:bg-surface-2 hover:text-text-heading'}"
        title={collapsed ? item.label : ''}
      >
        <item.icon size={16} class="shrink-0" />
        {#if !collapsed}
          <span class="text-[10px] font-bold uppercase tracking-wider text-left" transition:fade={{ duration: 100 }}>{item.label}</span>
        {/if}
      </button>
    {/each}
  </nav>

  <!-- Footer -->
  <div class="p-2 border-t border-border-primary bg-surface-2/30">
    <button 
      onclick={() => collapsed = !collapsed}
      class="w-full flex items-center justify-center p-2 text-text-muted hover:text-text-heading hover:bg-surface-2 transition-all rounded-sm"
    >
      {#if collapsed}
        <ChevronRight size={16} />
      {:else}
        <div class="flex items-center gap-3 w-full px-1">
          <ChevronLeft size={16} />
          <span class="text-[9px] font-mono uppercase tracking-widest opacity-50">Collapse Shell</span>
        </div>
      {/if}
    </button>
    
    {#if !collapsed}
      <div class="mt-4 p-3 bg-surface-1 border border-border-primary rounded-sm space-y-2" transition:fade>
         <div class="flex justify-between items-center">
            <span class="text-[8px] font-mono text-text-muted uppercase tracking-widest">Platform Sync</span>
            <div class="w-1.5 h-1.5 rounded-full bg-status-online shadow-[0_0_5px_var(--status-online)]"></div>
         </div>
         <div class="flex items-center gap-2">
            <Terminal size={12} class="text-text-muted" />
            <span class="text-[9px] font-mono text-text-muted">ID: SEC-2918-FF</span>
         </div>
      </div>
    {/if}
  </div>
</aside>

<style>
  /* Sidebar specific styles */
</style>
