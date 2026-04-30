<script lang="ts">
  let { title, hint }: { title: string; hint?: string } = $props();
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
    <input
      type="search"
      placeholder="Search events, hosts, rules…"
      class="w-72 rounded-md border border-night-600 bg-night-800/70 px-3 py-1.5 text-xs text-slate-100 placeholder:text-night-300 focus:border-accent-500 focus:outline-none focus:ring-1 focus:ring-accent-500"
    />
    <kbd class="hidden rounded border border-night-600 bg-night-800 px-1.5 py-0.5 text-[10px] text-night-300 sm:inline">
      Ctrl+K
    </kbd>
  </div>
</header>
