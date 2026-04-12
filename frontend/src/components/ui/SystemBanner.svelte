<!--
  OBLIVRA — System Health Banner (Svelte 5)
  
  Displayed at the top of the App Layout when the backend reports
  Degraded or Critical health states (e.g. Ingestion Overload).
-->
<script lang="ts">
  import { appStore } from '@lib/stores/app.svelte';
  import { ShieldAlert, AlertTriangle, X } from 'lucide-svelte';
  import { slide } from 'svelte/transition';
  
  interface Props {
    onClose?: () => void;
  }

  let { onClose }: Props = $props();

  const isDegraded = $derived(appStore.systemHealth.status === 'degraded');
  const isCritical = $derived(appStore.systemHealth.status === 'critical');
  const isVisible = $derived(isDegraded || isCritical);

  function dismiss() {
    // We don't actually dismiss a system-wide health state from the UI,
    // but we can provide a "Hide" for the current session if desired.
    // For now, it stays visible until the backend clears it.
  }
</script>

{#if isVisible}
  <div 
    transition:slide={{ duration: 300 }}
    class="relative z-50 flex items-center justify-between px-4 py-2 border-b transition-colors duration-500"
    class:bg-amber-500/10={isDegraded}
    class:border-amber-500/30={isDegraded}
    class:text-amber-400={isDegraded}
    class:bg-red-500/10={isCritical}
    class:border-red-500/30={isCritical}
    class:text-red-400={isCritical}
  >
    <div class="flex items-center gap-3">
      <div class="flex items-center justify-center w-8 h-8 rounded-full bg-current/10 animate-pulse">
        {#if isCritical}
          <ShieldAlert size={18} />
        {:else}
          <AlertTriangle size={18} />
        {/if}
      </div>
      
      <div class="flex flex-col">
        <span class="text-sm font-semibold tracking-wide uppercase">
          System {isCritical ? 'Critical' : 'Degraded'}
        </span>
        <span class="text-xs opacity-80 font-mono">
          {appStore.systemHealth.message || 'Backpressure detected in core ingestion pipeline.'}
        </span>
      </div>
    </div>

    <div class="flex items-center gap-4">
      <button 
        class="text-[10px] font-bold tracking-tighter uppercase px-2 py-1 rounded border border-current/20 hover:bg-current/10 transition-colors"
        onclick={() => appStore.setActiveNavTab('health')}
      >
        View Diagnostics
      </button>
    </div>
  </div>
{/if}

<style>
  /* Subtle glassmorphism */
  div {
    backdrop-filter: blur(8px);
  }
</style>
