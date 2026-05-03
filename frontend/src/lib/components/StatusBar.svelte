<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { ping, type Health } from '../bridge';

  let health = $state<Health | null>(null);
  let uptime = $state(0);
  let timer: ReturnType<typeof setInterval> | null = null;

  async function tick() {
    try {
      health = await ping();
    } catch (err) {
      health = { status: 'error', timestamp: new Date().toISOString() };
    }
    uptime++;
  }

  onMount(() => {
    void tick();
    timer = setInterval(() => void tick(), 5000);
  });

  onDestroy(() => {
    if (timer) clearInterval(timer);
  });

  const statusColor = $derived(
    health?.status === 'ok'    ? 'var(--color-sig-ok)' :
    health?.status === 'error' ? 'var(--color-sig-error)' :
                                 'var(--color-sig-warn)'
  );
  const statusLabel = $derived(health?.status?.toUpperCase() ?? 'CONNECTING');
</script>

<footer
  class="flex h-7 items-center justify-between border-t border-base-700 px-4"
  style="background: var(--color-base-900); flex-shrink: 0;"
>
  <!-- Left cluster -->
  <div class="flex items-center gap-4">
    <!-- Health dot + label -->
    <span class="flex items-center gap-1.5">
      <span
        class={health?.status === 'ok' ? 'animate-glow' : health?.status === 'error' ? 'animate-glow-err' : 'animate-glow-warn'}
        style="display:inline-block; width:6px; height:6px; border-radius:50%; background:{statusColor};"
      ></span>
      <span style="font-family:'Share Tech Mono',monospace; font-size:10px; letter-spacing:1px; color:{statusColor};">{statusLabel}</span>
    </span>

    <span style="color:var(--color-base-600);">·</span>

    <!-- Ping timestamp -->
    {#if health?.timestamp}
      <span style="font-family:'Share Tech Mono',monospace; font-size:10px; letter-spacing:0.5px; color:var(--color-base-300);">
        LAST PING {new Date(health.timestamp).toLocaleTimeString()}
      </span>
    {/if}
  </div>

  <!-- Right cluster -->
  <div class="flex items-center gap-3" style="font-family:'Share Tech Mono',monospace; font-size:10px; letter-spacing:1px; color:var(--color-base-300);">
    <span style="color:var(--color-cyan-500); opacity:0.6;">OBLIVRA</span>
    <span style="color:var(--color-base-600);">·</span>
    <span>PHASE 0</span>
    <span style="color:var(--color-base-600);">·</span>
    <span style="color:var(--color-base-200);">v0.1.0</span>
    <span style="color:var(--color-base-600);">·</span>
    <span>SOVEREIGN LOG PLATFORM</span>
  </div>
</footer>
