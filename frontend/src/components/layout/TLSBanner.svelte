<!--
  OBLIVRA — TLS plaintext banner (Slice 3, guardrail #2).

  Mounted in App.svelte alongside the crisis banner. Polls
  /api/v1/tls/state every 60 s; when the server reports
  `off: true`, renders a thin red strip above the page content
  reading "⚠ PLAINTEXT TRAFFIC". Click reveals the reason.

  This is an OPERATOR-FACING SAFETY BANNER, not a UX flourish — the
  whole point is that running with TLS off should look obviously
  wrong every time the operator opens the app. Don't suppress it.
-->
<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { ShieldOff, X } from 'lucide-svelte';
  import { apiFetch } from '@lib/apiClient';

  let off = $state(false);
  let reason = $state('');
  let dismissed = $state(false);
  let timer: ReturnType<typeof setInterval> | null = null;

  async function refresh() {
    try {
      const res = await apiFetch('/api/v1/tls/state');
      if (!res.ok) return;
      const body = await res.json();
      const wasOff = off;
      off = !!body.off;
      reason = String(body.reason ?? '');
      // Re-show the banner if TLS was just turned off via hot-reload —
      // an operator can't dismiss into thinking the platform's
      // protected when it isn't.
      if (off && !wasOff) dismissed = false;
    } catch { /* network down — leave previous state */ }
  }

  onMount(() => {
    void refresh();
    // Poll every 60 s. Faster than that wastes RPCs; slower means the
    // operator goes a minute without warning when TLS hot-reloads off.
    timer = setInterval(refresh, 60_000);
  });

  onDestroy(() => {
    if (timer) clearInterval(timer);
  });
</script>

{#if off && !dismissed}
  <div
    class="flex items-center gap-2 px-3 bg-error/15 border-b border-error/40 shrink-0"
    style="height: var(--banner-h, 28px); font-size: var(--fs-micro, 10px);"
    role="alert"
    aria-live="polite"
    title={reason}
  >
    <ShieldOff size={11} class="text-error animate-pulse shrink-0" />
    <span class="font-mono font-bold uppercase tracking-widest text-error">PLAINTEXT TRAFFIC</span>
    <span class="font-mono text-error/80 truncate">— TLS disabled. Compliance frameworks require encryption-in-transit.</span>
    <button
      class="ml-auto text-error/60 hover:text-error p-1 rounded shrink-0"
      onclick={() => (dismissed = true)}
      aria-label="Dismiss"
      title="Dismiss for this session — banner reappears on next reload"
    >
      <X size={11} />
    </button>
  </div>
{/if}
