<script lang="ts">
  let { title, hint }: { title: string; hint?: string } = $props();

  // Clicking the search box dispatches the same Ctrl+K as the global
  // shortcut. We want the keystroke and the click to feel identical so
  // analysts who picked up either habit get the palette.
  function openPalette(e: Event) {
    e.preventDefault();
    const ev = new KeyboardEvent('keydown', {
      key: 'k', ctrlKey: true, metaKey: true, bubbles: true,
    });
    window.dispatchEvent(ev);
  }
</script>

<header
  class="flex h-14 items-center justify-between border-b border-night-700 bg-night-900/60 px-6 backdrop-blur"
>
  <div class="flex items-baseline gap-3">
    <h1 class="text-base font-semibold tracking-tight text-slate-100">{title}</h1>
    {#if hint}
      <span class="text-xs text-night-300">{hint}</span>
    {/if}
  </div>

  <div class="flex items-center gap-2">
    <button
      type="button"
      onclick={openPalette}
      class="flex w-72 items-center gap-2 rounded-md border border-night-600 bg-night-800/70 px-3 py-1.5 text-left text-xs text-night-300 transition hover:border-accent-500 hover:text-slate-100 focus:border-accent-500 focus:outline-none focus:ring-1 focus:ring-accent-500"
      aria-label="Open command palette"
    >
      <span class="text-base leading-none">⌕</span>
      <span class="flex-1">Search events, hosts, rules…</span>
      <kbd class="rounded border border-night-600 bg-night-900 px-1.5 py-0.5 text-[10px] text-night-300">
        Ctrl K
      </kbd>
    </button>
  </div>
</header>
