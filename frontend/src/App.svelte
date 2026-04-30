<script lang="ts">
  import Sidebar from './lib/components/Sidebar.svelte';
  import TopBar from './lib/components/TopBar.svelte';
  import StatusBar from './lib/components/StatusBar.svelte';
  import Overview from './lib/views/Overview.svelte';
  import Placeholder from './lib/views/Placeholder.svelte';
  import { NAV, type NavId } from './lib/nav';

  let active = $state<NavId>('overview');
  let sidebarOpen = $state(true);

  const current = $derived(NAV.find((n) => n.id === active)!);

  function handleKey(e: KeyboardEvent) {
    if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'b') {
      e.preventDefault();
      sidebarOpen = !sidebarOpen;
    }
  }
</script>

<svelte:window on:keydown={handleKey} />

<div class="flex h-screen w-screen overflow-hidden bg-night-950 text-slate-100">
  <Sidebar bind:open={sidebarOpen} bind:active />

  <div class="flex flex-1 flex-col overflow-hidden">
    <TopBar title={current.label} hint={current.hint} />

    <main class="flex-1 overflow-auto scrollbar-thin">
      {#if active === 'overview'}
        <Overview />
      {:else}
        <Placeholder name={current.label} />
      {/if}
    </main>

    <StatusBar />
  </div>
</div>
