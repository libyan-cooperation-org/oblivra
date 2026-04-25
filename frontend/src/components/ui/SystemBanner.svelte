<!-- OBLIVRA — SystemBanner v2 — design tokens, no raw Tailwind color names -->
<script lang="ts">
  import { appStore } from '@lib/stores/app.svelte';
  import { slide } from 'svelte/transition';

  const isDegraded = $derived(appStore.systemHealth?.status === 'degraded');
  const isCritical = $derived(appStore.systemHealth?.status === 'critical');
  const isVisible  = $derived(isDegraded || isCritical);
  const message    = $derived(appStore.systemHealth?.message || 'Backpressure detected in core ingestion pipeline.');
</script>

{#if isVisible}
  <div
    transition:slide={{ duration: 200 }}
    class="relative z-50 flex items-center justify-between px-4 py-1.5 border-b shrink-0"
    style="{isCritical
      ? 'background:rgba(192,40,40,0.09); border-color:rgba(192,40,40,0.28); color:var(--cr2);'
      : 'background:rgba(184,96,0,0.08); border-color:rgba(184,96,0,0.22); color:var(--hi2);'}"
    role="alert"
    aria-live="assertive"
  >
    <div class="flex items-center gap-3">
      <!-- Pulsing indicator -->
      <span class="w-2 h-2 rounded-full shrink-0"
        style="background:currentColor; animation:var(--animate-banner-pulse);"></span>

      <!-- Label + message -->
      <div class="flex flex-col">
        <span class="font-mono text-[9px] font-bold uppercase tracking-widest">
          SYSTEM {isCritical ? 'CRITICAL' : 'DEGRADED'}
        </span>
        <span class="font-mono text-[9px] opacity-70 mt-px">{message}</span>
      </div>
    </div>

    <button
      class="font-mono text-[9px] font-bold uppercase tracking-wider px-2 py-0.5 rounded-sm border border-current/20 hover:bg-current/10 transition-colors cursor-pointer"
      onclick={() => appStore.setActiveNavTab('health')}
    >View Diagnostics</button>
  </div>
{/if}
