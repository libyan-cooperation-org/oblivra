<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { ping, type Health } from '../bridge';

  let health = $state<Health | null>(null);
  let surface = $state<'desktop' | 'web'>(typeof window !== 'undefined' && window.wails ? 'desktop' : 'web');
  let timer: ReturnType<typeof setInterval> | null = null;

  async function tick() {
    try {
      health = await ping();
    } catch (err) {
      health = { status: 'error', timestamp: new Date().toISOString() };
    }
  }

  onMount(() => {
    void tick();
    timer = setInterval(() => void tick(), 5000);
  });

  onDestroy(() => {
    if (timer) clearInterval(timer);
  });
</script>

<footer class="flex h-7 items-center justify-between border-t border-night-700 bg-night-900/80 px-4 text-[11px] text-night-300">
  <div class="flex items-center gap-3">
    <span class="flex items-center gap-1.5">
      <span
        class="inline-block h-2 w-2 rounded-full"
        class:bg-signal-success={health?.status === 'ok'}
        class:bg-signal-warn={!health}
        class:bg-signal-error={health?.status === 'error'}
      ></span>
      {health?.status ?? 'connecting'}
    </span>
    <span class="text-night-400">·</span>
    <span>Surface: <span class="text-night-200">{surface}</span></span>
  </div>

  <div class="flex items-center gap-3 font-mono">
    <span>Phase 0</span>
    <span class="text-night-400">·</span>
    <span>v0.1.0</span>
  </div>
</footer>
