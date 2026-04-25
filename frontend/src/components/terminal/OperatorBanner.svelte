<!--
  OperatorBanner — surfaces SIEM alerts for the currently-active SSH host
  inside the terminal view. Phase 23.4 / 22.4 Operator Mode.

  Signal source: alertStore (live WebSocket feed) filtered by host. We show
  a dismissible banner only when there's something the operator should
  actually know about — this is a notification surface, not a permanent
  status bar item.

  Two click-throughs:
    1. "View events" — drills into SIEMSearch filtered to this host
    2. "Isolate"     — fires the same Ctrl+Shift+I action used elsewhere
-->
<script lang="ts">
  import { ChevronRight, ShieldAlert, X, Eye, ShieldOff } from 'lucide-svelte';
  import { alertStore } from '@lib/stores/alerts.svelte.ts';
  import { appStore } from '@lib/stores/app.svelte';

  interface Props {
    /** Hostname of the active SSH session — used to filter alerts. */
    host: string;
  }

  let { host }: Props = $props();
  let dismissed = $state(false);

  const hostAlerts = $derived(
    (alertStore?.alerts ?? []).filter(
      (a: any) => a.host === host || a.hostname === host || a.host_id === host
    )
  );

  const critCount = $derived(hostAlerts.filter((a: any) => a.severity === 'critical').length);
  const highCount = $derived(hostAlerts.filter((a: any) => a.severity === 'high').length);
  const totalRecent = $derived(hostAlerts.length);

  // Re-show the banner when severity escalates even if previously dismissed.
  let lastTotal = 0;
  $effect(() => {
    if (totalRecent > lastTotal && totalRecent > 0) dismissed = false;
    lastTotal = totalRecent;
  });

  const visible = $derived(!dismissed && totalRecent > 0 && host);

  function viewEvents() {
    appStore.navigate('/siem-search', { query: `host == "${host}"` });
  }

  function isolate() {
    // Reuse the same global event the Ctrl+Shift+I keybind dispatches —
    // OperatorMode.svelte handles it (toggleQuarantine + audit toast).
    window.dispatchEvent(new CustomEvent('oblivra:isolate-host'));
  }
</script>

{#if visible}
  <div
    class="flex items-center gap-3 px-3 h-7 bg-warning/10 border border-warning/30 rounded-sm text-[10px] font-mono"
    class:critical={critCount > 0}
    role="alert"
    aria-live="polite"
  >
    <ShieldAlert class="w-3.5 h-3.5 shrink-0 {critCount > 0 ? 'text-error' : 'text-warning'}" />
    <span class="font-bold uppercase tracking-wider {critCount > 0 ? 'text-error' : 'text-warning'}">
      {totalRecent} alert{totalRecent === 1 ? '' : 's'} on {host}
    </span>
    {#if critCount > 0}
      <span class="px-1.5 py-px text-[8px] font-bold rounded-sm bg-error/20 text-error border border-error/40">
        CRIT {critCount}
      </span>
    {/if}
    {#if highCount > 0}
      <span class="px-1.5 py-px text-[8px] font-bold rounded-sm bg-warning/20 text-warning border border-warning/40">
        HIGH {highCount}
      </span>
    {/if}

    <div class="flex-1"></div>

    <button
      class="flex items-center gap-1 text-text-muted hover:text-text-heading transition-colors bg-transparent border-none cursor-pointer text-[9px] uppercase tracking-wider"
      onclick={viewEvents}
      title="View matching events in SIEM Search"
    >
      <Eye class="w-3 h-3" />
      <span>View events</span>
      <ChevronRight class="w-3 h-3" />
    </button>

    <button
      class="flex items-center gap-1 text-error hover:text-white hover:bg-error transition-colors bg-transparent border border-error/40 rounded-sm px-1.5 cursor-pointer text-[9px] uppercase tracking-wider font-bold"
      onclick={isolate}
      title="Isolate host (Ctrl+Shift+I)"
    >
      <ShieldOff class="w-3 h-3" />
      <span>Isolate</span>
    </button>

    <button
      class="text-text-muted hover:text-text-heading transition-colors bg-transparent border-none cursor-pointer p-0.5"
      onclick={() => (dismissed = true)}
      aria-label="Dismiss"
      title="Dismiss"
    >
      <X class="w-3 h-3" />
    </button>
  </div>
{/if}
