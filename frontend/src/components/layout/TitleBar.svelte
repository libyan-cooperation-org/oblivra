<!--
  OBLIVRA — TitleBar (Svelte 5)
  Top bar with window controls, brand, SSH quick-connect, and user avatar.
-->
<script lang="ts">
  import { onMount } from 'svelte';
  import { IS_BROWSER } from '@lib/context';
  import { appStore } from '@lib/stores/app.svelte';

  onMount(() => {
    if (IS_BROWSER) return;
    
    let interval: any;
    
    // Wrapper for async initialization
    const init = async () => {
      try {
        const { Window } = await import('@wailsio/runtime');
        const checkMax = async () => {
          await Window.IsMaximised();
        };
        checkMax();
        interval = setInterval(checkMax, 1000);
      } catch { /* runtime not available */ }
    };

    init();

    return () => {
      if (interval) clearInterval(interval);
    };
  });

  async function windowClose() {
    const { Application } = await import('@wailsio/runtime');
    Application.Quit();
  }
  async function windowMinimize() {
    const { Window } = await import('@wailsio/runtime');
    Window.Minimise();
  }
  async function windowToggleMax() {
    const { Window } = await import('@wailsio/runtime');
    Window.ToggleMaximise();
  }
</script>

<header
  class="flex items-center h-7 bg-[var(--s1)] border-b border-[var(--b1)] select-none z-100 px-3 gap-4"
  style="-webkit-app-region: drag;"
>
  <!-- macOS traffic lights (desktop only) -->
  {#if !IS_BROWSER}
    <div class="flex items-center gap-1.5 shrink-0 pr-2" style="-webkit-app-region: no-drag;">
      <button class="w-3 h-3 rounded-full bg-[#ff5f57] border-none cursor-pointer" onclick={windowClose} title="Close"></button>
      <button class="w-3 h-3 rounded-full bg-[#ffbd2e] border-none cursor-pointer" onclick={windowMinimize} title="Minimize"></button>
      <button class="w-3 h-3 rounded-full bg-[#28c840] border-none cursor-pointer" onclick={windowToggleMax} title="Maximize"></button>
    </div>
  {/if}

  <!-- Brand -->
  <div class="flex items-center gap-2 h-full shrink-0">
    <div style="font-family:var(--mn); font-size:10px; font-weight:600; color:#d0e8f8; letter-spacing:0.1em;">
      OBL<em style="color:#e05050; font-style:normal;">IV</em>RA
    </div>
    <div class="px-1.5 py-0.5 bg-accent/10 border border-accent/20 rounded-sm text-[8px] font-mono text-accent font-bold">
      {IS_BROWSER ? 'WEB' : 'DESKTOP'}
    </div>
  </div>

  <!-- Tenant / Status -->
  <div class="flex items-center gap-3 ml-4 h-full">
     <div class="flex items-center gap-1.5 px-2 py-0.5 bg-surface-2 border border-border-primary rounded-sm h-[18px]">
        <div class="w-1.5 h-1.5 rounded-full bg-success"></div>
        <span class="text-[9px] font-mono text-text-muted uppercase">Sovereign Cloud</span>
     </div>
  </div>

  <!-- Quick Search / Command -->
  <div class="flex-1 flex justify-center items-center" style="-webkit-app-region: no-drag;">
    <button 
      type="button"
      class="flex items-center bg-surface-2 border border-border-primary rounded-sm px-2 h-[18px] gap-2 w-[240px] hover:border-border-hover cursor-pointer" 
      onclick={() => appStore.toggleCommandPalette()}
      style="-webkit-app-region: no-drag;"
    >
      <span class="text-text-muted text-[8px] font-mono uppercase tracking-widest opacity-60">Search commands...</span>
      <span class="ml-auto text-text-muted text-[8px] font-mono opacity-40">⌃K</span>
    </button>
  </div>

  <!-- Right controls -->
  <div class="flex items-center gap-3 shrink-0" style="-webkit-app-region: no-drag;">
    <div class="flex items-center gap-2">
        <span class="text-[9px] font-mono text-text-muted">OPERATOR:</span>
        <span class="text-[9px] font-mono text-text-heading font-bold uppercase tracking-tight">K. MAVERICK</span>
    </div>
    <div class="w-5 h-5 bg-accent/20 border border-accent/40 rounded-sm flex items-center justify-center text-[10px] font-bold font-mono text-accent">
        KM
    </div>
  </div>
</header>
