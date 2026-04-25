<!-- OBLIVRA — TitleBar v2 — unified tokens, no inline hex/vars -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { IS_BROWSER } from '@lib/context';
  import { appStore } from '@lib/stores/app.svelte';
  import { alertStore } from '@lib/stores/alerts.svelte';

  const critCount = $derived(
    alertStore?.alerts?.filter((a: any) => a.severity === 'critical').length ?? 0
  );
  const highCount = $derived(
    alertStore?.alerts?.filter((a: any) => a.severity === 'high').length ?? 0
  );

  onMount(() => {
    if (IS_BROWSER) return;
    let interval: ReturnType<typeof setInterval>;
    const init = async () => {
      try {
        const { Window } = await import('@wailsio/runtime');
        interval = setInterval(() => Window.IsMaximised(), 2000);
      } catch { /* dev */ }
    };
    init();
    return () => clearInterval(interval);
  });

  async function windowClose()     { const { Application } = await import('@wailsio/runtime'); Application.Quit(); }
  async function windowMinimize()  { const { Window }      = await import('@wailsio/runtime'); Window.Minimise(); }
  async function windowToggleMax() { const { Window }      = await import('@wailsio/runtime'); Window.ToggleMaximise(); }
</script>

<header
  class="flex items-center h-7 bg-surface-1 border-b border-border-primary select-none z-50 px-3 gap-3 shrink-0"
  style="-webkit-app-region: drag;"
>
  <!-- macOS traffic lights -->
  {#if !IS_BROWSER}
    <div class="flex items-center gap-1.5 shrink-0 pr-1" style="-webkit-app-region: no-drag;">
      <button class="w-3 h-3 rounded-full bg-[#ff5f57] hover:opacity-80 transition-opacity border-none cursor-pointer" onclick={windowClose}    title="Close"></button>
      <button class="w-3 h-3 rounded-full bg-[#ffbd2e] hover:opacity-80 transition-opacity border-none cursor-pointer" onclick={windowMinimize} title="Minimize"></button>
      <button class="w-3 h-3 rounded-full bg-[#28c840] hover:opacity-80 transition-opacity border-none cursor-pointer" onclick={windowToggleMax} title="Maximize"></button>
    </div>
    <div class="w-px h-3.5 bg-border-primary shrink-0"></div>
  {/if}

  <!-- Brand + context badge -->
  <div class="flex items-center gap-2 shrink-0" style="-webkit-app-region: no-drag;">
    <span class="text-text-heading font-mono text-[11px] font-semibold tracking-[0.1em]">
      OBL<em class="text-error not-italic">IV</em>RA
    </span>
    <span class="px-1.5 py-px text-[8px] font-mono font-bold tracking-widest rounded-sm border
      {IS_BROWSER
        ? 'text-accent-hover border-accent/30 bg-accent/8'
        : 'text-[#9878e0] border-[#9878e0]/30 bg-[#9878e0]/8'}">
      {IS_BROWSER ? 'WEB' : 'DESKTOP'}
    </span>
  </div>

  <!-- Divider -->
  <div class="w-px h-3.5 bg-border-primary shrink-0"></div>

  <!-- Sovereign status -->
  <div class="flex items-center gap-1.5 shrink-0" style="-webkit-app-region: no-drag;">
    <div class="w-1.5 h-1.5 rounded-full bg-success shrink-0"></div>
    <span class="text-[9px] font-mono text-text-muted uppercase tracking-wider">Sovereign Cloud</span>
  </div>

  <!-- Live severity chips — only shown when alerts exist -->
  {#if critCount > 0 || highCount > 0}
    <div class="flex items-center gap-1.5 shrink-0" style="-webkit-app-region: no-drag;">
      <div class="w-px h-3.5 bg-border-primary"></div>
      {#if critCount > 0}
        <span class="px-1.5 py-px text-[8px] font-mono font-bold rounded-sm
          bg-error/12 text-error border border-error/28">CRIT {critCount}</span>
      {/if}
      {#if highCount > 0}
        <span class="px-1.5 py-px text-[8px] font-mono font-bold rounded-sm
          bg-warning/12 text-warning border border-warning/25">HIGH {highCount}</span>
      {/if}
    </div>
  {/if}

  <!-- Centered command search -->
  <div class="flex-1 flex justify-center" style="-webkit-app-region: no-drag;">
    <button
      type="button"
      class="flex items-center bg-surface-3 border border-border-primary rounded-sm px-2.5 h-[18px] gap-2 w-[220px]
             hover:border-border-hover transition-colors cursor-pointer"
      onclick={() => appStore.toggleCommandPalette()}
    >
      <span class="text-text-muted text-[8px] font-mono tracking-wide opacity-60">Search commands...</span>
      <span class="ml-auto text-text-muted text-[8px] font-mono opacity-40">⌃K</span>
    </button>
  </div>

  <!-- Operator -->
  <div class="flex items-center gap-2 shrink-0" style="-webkit-app-region: no-drag;">
    <span class="text-[9px] font-mono text-text-muted">OPERATOR ·</span>
    <span class="text-[9px] font-mono text-text-heading font-semibold uppercase tracking-tight">K. MAVERICK</span>
    <div class="w-5 h-5 rounded-sm flex items-center justify-center text-[9px] font-bold font-mono
                bg-accent/15 border border-accent/30 text-accent-hover">KM</div>
  </div>
</header>
