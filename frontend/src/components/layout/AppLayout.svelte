<!--
  OBLIVRA — App Layout Shell (Svelte 5)

  Top bar + Command Rail + Main Content + Status Bar.
  Wraps the router content in the standard OBLIVRA shell.
-->
<script lang="ts">
  import { onMount, onDestroy, type Snippet } from 'svelte';
  import { appStore } from '@lib/stores/app.svelte';
  import { IS_BROWSER } from '@lib/context';
  import TitleBar from './TitleBar.svelte';
  import StatusBar from './StatusBar.svelte';
  import CommandRail from './CommandRail.svelte';
  import CommandPalette from '@components/ui/CommandPalette.svelte';
  import AddHostModal from '@components/sidebar/AddHostModal.svelte';
  import TransferDrawer from '@components/terminal/TransferDrawer.svelte';

  interface Props {
    children: Snippet;
  }

  let { children }: Props = $props();

  let showTransferDrawer = $state(false);
  let showCommandPalette = $state(false);
  let showAddHostModal = $state(false);

  // Global keyboard shortcuts
  function handleKeyDown(e: KeyboardEvent) {
    const mod = e.metaKey || e.ctrlKey;
    
    // Command Palette (Cmd+K or Cmd+P)
    if (mod && (e.key.toLowerCase() === 'k' || e.key.toLowerCase() === 'p')) {
      e.preventDefault();
      showCommandPalette = !showCommandPalette;
      return;
    }

    // Add Host (Cmd+N)
    if (mod && e.key.toLowerCase() === 'n') {
      e.preventDefault();
      showAddHostModal = true;
      return;
    }

    if (!mod) return;

    if (e.shiftKey && e.key.toLowerCase() === 'f') {
      e.preventDefault();
      appStore.toggleFocusMode();
    }
    if (e.key === 'b') {
      e.preventDefault();
      appStore.toggleSidebar();
    }
    if (e.key === ',') {
      e.preventDefault();
      appStore.setActiveNavTab('settings');
    }
  }

  onMount(async () => {
    window.addEventListener('keydown', handleKeyDown);

    // Restore workspace state (desktop only)
    if (!IS_BROWSER) {
      try {
        const { GetActive } = await import('../../../wailsjs/go/services/WorkspaceService');
        const ws = await GetActive();
        if (ws?.active_tab) {
          appStore.setActiveNavTab(ws.active_tab as any);
        }
      } catch { /* WorkspaceService may not exist */ }
    }
  });

  onDestroy(() => {
    window.removeEventListener('keydown', handleKeyDown);
  });
</script>

<div class="flex flex-col h-screen w-screen bg-surface-0 overflow-hidden" class:war-mode-active={false}>
  <!-- Title Bar -->
  {#if !appStore.focusMode}
    <TitleBar />
  {/if}

  <!-- Body: Rail + Content -->
  <div class="flex flex-1 overflow-hidden bg-surface-0">
    <!-- Command Rail -->
    {#if !appStore.focusMode}
      <CommandRail />
    {/if}

    <!-- Main Content Area -->
    <main class="flex-1 bg-surface-0 overflow-auto flex flex-col min-w-0">
      {@render children()}
    </main>
  </div>

  <!-- Status Bar -->
    {#if !appStore.focusMode}
      <StatusBar onToggleTransfers={() => showTransferDrawer = !showTransferDrawer} />
    {/if}

  <!-- Global UI Overlays -->
  <CommandPalette
    open={showCommandPalette}
    onClose={() => showCommandPalette = false}
  />

  <AddHostModal
    open={showAddHostModal}
    onClose={() => showAddHostModal = false}
  />

  <TransferDrawer
    open={showTransferDrawer}
    onClose={() => showTransferDrawer = false}
  />
</div>
