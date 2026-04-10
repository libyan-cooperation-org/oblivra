<!--
  OBLIVRA — TitleBar (Svelte 5)
  Top bar with window controls, brand, SSH quick-connect, and user avatar.
-->
<script lang="ts">
  import { onMount } from 'svelte';
  import { IS_BROWSER } from '@lib/context';
  import { appStore } from '@lib/stores/app.svelte';

  let quickVal = $state('');
  let isMaximized = $state(false);

  function handleQuickConnect() {
    const val = quickVal.trim();
    if (!val) return;
    quickVal = '';
    appStore.connectToHost(val);
  }

  onMount(async () => {
    if (IS_BROWSER) return;
    try {
      const { WindowIsMaximised } = await import('@wailsjs/runtime/runtime.js');
      const checkMax = async () => {
        const max = await WindowIsMaximised();
        isMaximized = max;
      };
      checkMax();
      const interval = setInterval(checkMax, 1000);
      return () => clearInterval(interval);
    } catch { /* runtime not available */ }
  });

  async function windowClose() {
    const r = await import('@wailsjs/runtime/runtime.js');
    r.Quit();
  }
  async function windowMinimize() {
    const r = await import('@wailsjs/runtime/runtime.js');
    r.WindowMinimise();
  }
  async function windowToggleMax() {
    const r = await import('@wailsjs/runtime/runtime.js');
    r.WindowToggleMaximise();
  }
</script>

<header
  class="flex items-center h-[var(--header-height)] bg-[#0d0e10] border-b border-black select-none z-100 gap-0"
  style="-webkit-app-region: drag;"
>
  <!-- macOS traffic lights (desktop only) -->
  {#if !IS_BROWSER}
    <div class="flex items-center gap-1.5 px-3.5 shrink-0" style="-webkit-app-region: no-drag;">
      <button
        class="w-3 h-3 rounded-full bg-[#ff5f57] border-none cursor-pointer hover:brightness-120 hover:scale-110 transition-all duration-fast"
        onclick={windowClose}
        title="Close"
        aria-label="Close OBLIVRA"
      ></button>
      <button
        class="w-3 h-3 rounded-full bg-[#ffbd2e] border-none cursor-pointer hover:brightness-120 hover:scale-110 transition-all duration-fast"
        onclick={windowMinimize}
        title="Minimize"
        aria-label="Minimize Window"
      ></button>
      <button
        class="w-3 h-3 rounded-full bg-[#28c840] border-none cursor-pointer hover:brightness-120 hover:scale-110 transition-all duration-fast"
        onclick={windowToggleMax}
        title={isMaximized ? 'Restore' : 'Maximize'}
        aria-label={isMaximized ? 'Restore Window' : 'Maximize Window'}
      ></button>
    </div>
  {/if}

  <!-- Brand -->
  <div class="flex items-center gap-2 px-4 bg-accent-cta h-full shrink-0">
    <svg width="18" height="18" viewBox="0 0 24 24" fill="none">
      <path d="M12 2L4 6.5v11L12 22l8-4.5v-11L12 2z" stroke="var(--accent-primary)" stroke-width="1.5" fill="none"/>
      <circle cx="12" cy="12" r="2.5" fill="var(--accent-primary)"/>
    </svg>
    <span class="font-[var(--font-ui)] text-[13px] font-extrabold text-white tracking-wide uppercase">
      Sovereign
    </span>
  </div>

  <!-- Quick Connect -->
  <div class="flex-1 flex justify-center items-center px-3" style="-webkit-app-region: no-drag;">
    <div class="flex items-center bg-surface-2 border border-border-primary rounded-sm px-2 h-[26px] gap-2 w-[300px] transition-all duration-fast focus-within:border-accent focus-within:shadow-glow">
      <span class="text-accent text-[9px] font-bold font-mono tracking-widest whitespace-nowrap opacity-80 shrink-0">SSH</span>
      <div class="w-px h-3.5 bg-border-primary shrink-0"></div>
      <input
        class="flex-1 bg-transparent border-none outline-none text-text-primary font-mono text-[11px] placeholder:text-text-muted"
        type="text"
        placeholder="user@host or ip:port"
        bind:value={quickVal}
        onkeydown={(e) => { if (e.key === 'Enter') handleQuickConnect(); }}
      />
      <span class="text-text-muted text-[10px] font-mono opacity-50 shrink-0">↵</span>
    </div>
  </div>

  <!-- Right controls -->
  <div class="flex items-center gap-2.5 shrink-0 pr-3" style="-webkit-app-region: no-drag;">
    <span class="font-mono text-[9px] text-text-muted tracking-wider opacity-50">v0.1</span>
    <div
      class="w-[26px] h-[26px] bg-accent rounded-sm flex items-center justify-center text-[11px] font-bold font-[var(--font-ui)] text-white cursor-pointer hover:brightness-110 hover:scale-105 transition-all duration-fast shrink-0"
      title="Profile"
    >K</div>
  </div>
</header>
