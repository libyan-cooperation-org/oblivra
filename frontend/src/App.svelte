<script lang="ts">
  import Sidebar from './lib/components/Sidebar.svelte';
  import TopBar from './lib/components/TopBar.svelte';
  import StatusBar from './lib/components/StatusBar.svelte';
  import CommandPalette from './lib/components/CommandPalette.svelte';
  import ShortcutsOverlay from './lib/components/ShortcutsOverlay.svelte';
  import ToastStack from './lib/components/ToastStack.svelte';
  import Overview from './lib/views/Overview.svelte';
  import Siem from './lib/views/Siem.svelte';
  import Categories from './lib/views/Categories.svelte';
  import Services from './lib/views/Services.svelte';
  import SavedSearches from './lib/views/SavedSearches.svelte';
  import Detection from './lib/views/Detection.svelte';
  import Mitre from './lib/views/Mitre.svelte';
  import Lineage from './lib/views/Lineage.svelte';
  import Investigations from './lib/views/Investigations.svelte';
  import Cases from './lib/views/Cases.svelte';
  import Reconstruction from './lib/views/Reconstruction.svelte';
  import Trust from './lib/views/Trust.svelte';
  import Evidence from './lib/views/Evidence.svelte';
  import Graph from './lib/views/Graph.svelte';
  import Vault from './lib/views/Vault.svelte';
  import Webhooks from './lib/views/Webhooks.svelte';
  import Notifications from './lib/views/Notifications.svelte';
  import Fleet from './lib/views/Fleet.svelte';
  import Admin from './lib/views/Admin.svelte';
  import Placeholder from './lib/views/Placeholder.svelte';
  import { NAV, type NavId } from './lib/nav';

  let active = $state<NavId>('overview');
  let sidebarOpen = $state(true);
  let paletteOpen = $state(false);
  let shortcutsOpen = $state(false);

  const current = $derived(NAV.find((n) => n.id === active)!);

  function handleKey(e: KeyboardEvent) {
    // Ignore shortcuts while user is typing in inputs/textareas/contenteditable —
    // except Ctrl/Cmd+K which we want to override globally.
    const target = e.target as HTMLElement | null;
    const isTyping = !!target && (
      target.tagName === 'INPUT' ||
      target.tagName === 'TEXTAREA' ||
      target.isContentEditable
    );

    // Ctrl/Cmd+K — command palette
    if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'k') {
      e.preventDefault();
      paletteOpen = !paletteOpen;
      return;
    }
    // Ctrl/Cmd+B — toggle sidebar
    if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'b') {
      e.preventDefault();
      sidebarOpen = !sidebarOpen;
      return;
    }
    // ? — keyboard shortcuts
    if (!isTyping && e.key === '?' && !e.ctrlKey && !e.metaKey && !e.altKey) {
      e.preventDefault();
      shortcutsOpen = !shortcutsOpen;
      return;
    }
    // Esc — close any modal
    if (e.key === 'Escape') {
      if (paletteOpen) { paletteOpen = false; e.preventDefault(); return; }
      if (shortcutsOpen) { shortcutsOpen = false; e.preventDefault(); return; }
    }
    // / — focus the global search (currently in TopBar)
    if (!isTyping && e.key === '/' && !e.ctrlKey && !e.metaKey && !e.altKey) {
      const search = document.querySelector<HTMLInputElement>('header input[type="search"]');
      if (search) {
        e.preventDefault();
        search.focus();
        search.select();
      }
    }
  }

  // Listen for the dispatched custom event from the command palette so
  // "Show keyboard shortcuts" works even though the palette doesn't
  // hold a reference to App's state directly. Registered manually
  // because Svelte's <svelte:window> binding doesn't carry a type for
  // arbitrary CustomEvent names.
  $effect(() => {
    const handler = () => { shortcutsOpen = true; };
    window.addEventListener('oblivra:shortcuts', handler);
    return () => window.removeEventListener('oblivra:shortcuts', handler);
  });
</script>

<svelte:window on:keydown={handleKey} />

<div class="flex h-screen w-screen overflow-hidden bg-night-950 text-slate-100">
  <Sidebar bind:open={sidebarOpen} bind:active />

  <div class="flex flex-1 flex-col overflow-hidden">
    <TopBar title={current.label} hint={current.hint} />

    <main class="flex-1 overflow-auto scrollbar-thin">
      {#if active === 'overview'}
        <Overview />
      {:else if active === 'siem'}
        <Siem />
      {:else if active === 'services'}
        <Services />
      {:else if active === 'categories'}
        <Categories />
      {:else if active === 'saved-searches'}
        <SavedSearches />
      {:else if active === 'detection'}
        <Detection />
      {:else if active === 'mitre'}
        <Mitre />
      {:else if active === 'investigations'}
        <Investigations />
      {:else if active === 'cases'}
        <Cases />
      {:else if active === 'reconstruction'}
        <Reconstruction />
      {:else if active === 'lineage'}
        <Lineage />
      {:else if active === 'trust'}
        <Trust />
      {:else if active === 'evidence'}
        <Evidence />
      {:else if active === 'graph'}
        <Graph />
      {:else if active === 'vault'}
        <Vault />
      {:else if active === 'webhooks'}
        <Webhooks />
      {:else if active === 'notifications'}
        <Notifications />
      {:else if active === 'fleet'}
        <Fleet />
      {:else if active === 'admin'}
        <Admin />
      {:else}
        <Placeholder name={current.label} />
      {/if}
    </main>

    <StatusBar />
  </div>

  <CommandPalette bind:open={paletteOpen} bind:active />
  <ShortcutsOverlay bind:open={shortcutsOpen} />
  <ToastStack />
</div>
