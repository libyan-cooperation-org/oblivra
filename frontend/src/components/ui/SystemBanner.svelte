<script lang="ts">
  /**
   * OBLIVRA — System Health Banner (Svelte 5)
   *
   * Displayed at the top of the App Layout when the backend reports
   * Degraded or Critical health states (e.g. Ingestion Overload).
   */
  import { appStore } from '@lib/stores/app.svelte';
  import { ShieldAlert, AlertTriangle } from 'lucide-svelte';
  import { slide } from 'svelte/transition';

  const isDegraded = $derived(appStore.systemHealth.status === 'degraded');
  const isCritical = $derived(appStore.systemHealth.status === 'critical');
  const isVisible = $derived(isDegraded || isCritical);

  const bannerClasses = $derived(
    isCritical
      ? 'bg-red-500/10 border-red-500/30 text-red-400'
      : 'bg-amber-500/10 border-amber-500/30 text-amber-400'
  );
</script>

{#if isVisible}
  <div
    transition:slide={{ duration: 300 }}
    class="relative z-50 flex items-center justify-between px-4 py-2 border-b transition-colors duration-500 {bannerClasses}"
  >
    <div class="flex items-center gap-3">
      <div class="flex items-center justify-center w-8 h-8 rounded-full animate-pulse" style="background: color-mix(in srgb, currentColor 10%, transparent)">
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
