<!--
  OBLIVRA TitleBar — platform-aware window chrome.

  Frameless Wails windows (main_gui.go sets Frameless: true) leave the OS
  with no native min/max/close, so we render our own. Mac users expect
  traffic-light dots on the LEFT; Windows / Linux users expect explicit
  Minimize / Maximize / Close icons on the RIGHT. Showing the wrong
  pattern is the difference between "obvious, native, professional" and
  the user reporting "there is no maximize close."

  Drag region: header has -webkit-app-region: drag so the operator can
  drag the window between monitors. Every interactive element overrides
  with -webkit-app-region: no-drag so clicks work.
-->
<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { IS_BROWSER } from '@lib/context';
  import { appStore } from '@lib/stores/app.svelte';
  import { alertStore } from '@lib/stores/alerts.svelte';
  import { Minus, Square, X, Copy as Restore, Monitor, ExternalLink, Layout } from 'lucide-svelte';

  // Platform detection — userAgent is reliable enough for picking chrome style.
  // We default to "win" if uncertain because that's the dominant SOC operator
  // platform and shows the more discoverable explicit icon controls.
  type Platform = 'mac' | 'win' | 'linux';
  let platform: Platform = $state('win');
  let isMaximised = $state(false);
  let popoutCount = $state(0);

  const critCount = $derived(
    alertStore?.alerts?.filter((a: any) => a.severity === 'critical').length ?? 0
  );
  const highCount = $derived(
    alertStore?.alerts?.filter((a: any) => a.severity === 'high').length ?? 0
  );

  let pollInterval: ReturnType<typeof setInterval> | undefined;

  onMount(() => {
    // Platform detection
    const ua = (typeof navigator !== 'undefined' ? navigator.userAgent : '').toLowerCase();
    if (ua.includes('mac os') || ua.includes('macintosh')) platform = 'mac';
    else if (ua.includes('linux') && !ua.includes('android')) platform = 'linux';
    else platform = 'win';

    if (IS_BROWSER) return;

    const pollState = async () => {
      try {
        const { Window } = await import('@wailsio/runtime');
        // Window.IsMaximised returns a Promise<boolean>; await it so the
        // state stays accurate as the operator drags the window between
        // monitors and toggles the chrome.
        isMaximised = await (Window.IsMaximised as any)();
      } catch {
        /* dev mode — runtime may not be wired yet */
      }

      try {
        const mod = await import('../../../bindings/github.com/kingknull/oblivrashell/internal/services/windowservice.js');
        if (mod && typeof mod.ListPopouts === 'function') {
          const ids = await mod.ListPopouts();
          popoutCount = Array.isArray(ids) ? ids.length : 0;
        }
      } catch {
        // Bindings only exist in desktop builds; ignore in web.
      }
    };

    pollState();
    pollInterval = setInterval(pollState, 1500);
  });

  onDestroy(() => {
    if (pollInterval) clearInterval(pollInterval);
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
    // Optimistic update so the icon flips before the next poll tick.
    isMaximised = !isMaximised;
  }

  async function closeAllPopouts() {
    try {
      const mod = await import('../../../bindings/github.com/kingknull/oblivrashell/internal/services/windowservice.js');
      if (mod && typeof mod.CloseAllPopouts === 'function') {
        await mod.CloseAllPopouts();
        popoutCount = 0;
      }
    } catch { /* dev */ }
  }
</script>

<header
  class="flex items-center h-8 bg-surface-1 border-b border-border-primary select-none z-50 px-2 gap-3 shrink-0"
  style="-webkit-app-region: drag;"
>
  <!-- macOS traffic lights (left side, Mac-only) -->
  {#if !IS_BROWSER && platform === 'mac'}
    <div class="flex items-center gap-1.5 shrink-0 pl-1 pr-1" style="-webkit-app-region: no-drag;">
      <button
        class="w-3 h-3 rounded-full bg-[#ff5f57] hover:opacity-80 transition-opacity border-none cursor-pointer flex items-center justify-center group"
        onclick={windowClose}
        aria-label="Close window"
      >
        <X class="w-2 h-2 text-black/60 opacity-0 group-hover:opacity-100" />
      </button>
      <button
        class="w-3 h-3 rounded-full bg-[#ffbd2e] hover:opacity-80 transition-opacity border-none cursor-pointer flex items-center justify-center group"
        onclick={windowMinimize}
        aria-label="Minimize window"
      >
        <Minus class="w-2 h-2 text-black/60 opacity-0 group-hover:opacity-100" />
      </button>
      <button
        class="w-3 h-3 rounded-full bg-[#28c840] hover:opacity-80 transition-opacity border-none cursor-pointer flex items-center justify-center group"
        onclick={windowToggleMax}
        aria-label="Toggle maximize window"
      >
        <Square class="w-2 h-2 text-black/60 opacity-0 group-hover:opacity-100" />
      </button>
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

  <div class="w-px h-3.5 bg-border-primary shrink-0"></div>

  <!-- Sovereign status -->
  <div class="flex items-center gap-1.5 shrink-0" style="-webkit-app-region: no-drag;">
    <div class="w-1.5 h-1.5 rounded-full bg-success shrink-0"></div>
    <span class="text-[9px] font-mono text-text-muted uppercase tracking-wider">Sovereign Cloud</span>
  </div>

  <!-- Pop-out indicator — visible only when one or more pop-outs are open -->
  {#if popoutCount > 0}
    <div class="flex items-center gap-1.5 shrink-0" style="-webkit-app-region: no-drag;">
      <div class="w-px h-3.5 bg-border-primary"></div>
      <button
        class="flex items-center gap-1 text-[8px] font-mono text-text-muted hover:text-text-heading uppercase tracking-wider bg-transparent border-none cursor-pointer"
        onclick={closeAllPopouts}
        title="Close all pop-out windows"
      >
        <Monitor class="w-3 h-3" />
        <span>{popoutCount} POP-OUT{popoutCount === 1 ? '' : 'S'}</span>
      </button>
    </div>
  {/if}

  <!-- Severity chips -->
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

  <!-- Centered command search — fills the drag area but doesn't drag itself -->
  <div class="flex-1 flex justify-center" style="-webkit-app-region: no-drag;">
    <button
      type="button"
      class="flex items-center bg-surface-3 border border-border-primary rounded-sm px-2.5 h-[20px] gap-2 w-[240px]
             hover:border-border-hover transition-colors cursor-pointer"
      onclick={() => appStore.toggleCommandPalette()}
    >
      <span class="text-text-muted text-[8px] font-mono tracking-wide opacity-60">Search commands…</span>
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

  <!-- Global Desktop Actions (Pop-out, etc) -->
  {#if !IS_BROWSER}
    <div class="flex items-center shrink-0 ml-auto" style="-webkit-app-region: no-drag;">
      <button
        class="h-8 px-2 flex items-center justify-center gap-1.5 text-text-muted hover:text-accent hover:bg-surface-2 transition-colors border-none bg-transparent cursor-pointer group"
        onclick={() => appStore.launchSOCExperience()}
        title="Launch SOC Multi-Monitor Experience (3+ Windows)"
      >
        <Layout class="w-3.5 h-3.5" />
        <span class="text-[9px] font-mono font-bold tracking-widest hidden lg:block opacity-60 group-hover:opacity-100">SOC MODE</span>
      </button>

      <button
        class="h-8 w-10 flex items-center justify-center text-text-muted hover:text-accent hover:bg-surface-2 transition-colors border-none bg-transparent cursor-pointer"
        onclick={() => appStore.popOut()}
        aria-label="Pop out into new window"
        title="Pop out into new window"
      >
        <ExternalLink class="w-3.5 h-3.5" />
      </button>

      {#if platform !== 'mac'}
        <button
          class="h-8 w-10 flex items-center justify-center text-text-muted hover:text-text-heading hover:bg-surface-2 transition-colors border-none bg-transparent cursor-pointer"
          onclick={windowMinimize}
          aria-label="Minimize window"
          title="Minimize"
        >
          <Minus class="w-3.5 h-3.5" />
        </button>
        <button
          class="h-8 w-10 flex items-center justify-center text-text-muted hover:text-text-heading hover:bg-surface-2 transition-colors border-none bg-transparent cursor-pointer"
          onclick={windowToggleMax}
          aria-label={isMaximised ? 'Restore window' : 'Maximize window'}
          title={isMaximised ? 'Restore' : 'Maximize'}
        >
          {#if isMaximised}
            <Restore class="w-3.5 h-3.5" />
          {:else}
            <Square class="w-3.5 h-3.5" />
          {/if}
        </button>
        <button
          class="h-8 w-10 flex items-center justify-center text-text-muted hover:text-white hover:bg-error transition-colors border-none bg-transparent cursor-pointer"
          onclick={windowClose}
          aria-label="Close window"
          title="Close"
        >
          <X class="w-3.5 h-3.5" />
        </button>
      {/if}
    </div>
  {/if}
</header>
