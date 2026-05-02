<script lang="ts">
  // Keyboard shortcuts overlay. Bound to `?` (Shift+/) globally.
  // Lives next to the command palette so an analyst pressing `?`
  // mid-investigation gets reminded what's available without leaving
  // the keyboard.

  let { open = $bindable(false) }: { open: boolean } = $props();

  type Group = {
    title: string;
    items: { keys: string[]; label: string }[];
  };

  const GROUPS: Group[] = [
    {
      title: 'Navigation',
      items: [
        { keys: ['Ctrl/Cmd', 'K'], label: 'Open command palette' },
        { keys: ['Ctrl/Cmd', 'B'], label: 'Toggle sidebar' },
        { keys: ['?'],            label: 'Show this help' },
        { keys: ['Esc'],          label: 'Close any open dialog' },
      ],
    },
    {
      title: 'Search & list',
      items: [
        { keys: ['/'],            label: 'Focus the search box (where present)' },
        { keys: ['↑', '↓'],       label: 'Move selection' },
        { keys: ['↵'],            label: 'Open / activate' },
      ],
    },
    {
      title: 'Evidence handling',
      items: [
        { keys: ['c'],            label: 'Copy event ID / hash under cursor' },
        { keys: ['e'],            label: 'Export view (where supported)' },
      ],
    },
  ];
</script>

<svelte:window
  on:keydown={(e) => {
    if (open && e.key === 'Escape') {
      e.preventDefault();
      open = false;
    }
  }}
/>

{#if open}
  <div
    class="fixed inset-0 z-50 grid place-items-center bg-night-950/70 backdrop-blur"
    role="dialog"
    aria-modal="true"
    aria-label="Keyboard shortcuts"
    tabindex="-1"
    onclick={(e) => {
      if (e.target === e.currentTarget) open = false;
    }}
    onkeydown={(e) => {
      if (e.target === e.currentTarget && e.key === 'Escape') open = false;
    }}
  >
    <div class="w-full max-w-2xl overflow-hidden rounded-xl border border-night-600 bg-night-900 shadow-2xl shadow-black/60">
      <header class="flex items-center justify-between border-b border-night-700 px-5 py-3">
        <h2 class="text-sm font-semibold tracking-tight text-slate-100">Keyboard shortcuts</h2>
        <button
          type="button"
          onclick={() => (open = false)}
          class="text-xs text-night-300 hover:text-white"
        >
          esc to close
        </button>
      </header>
      <div class="grid grid-cols-1 gap-6 p-5 sm:grid-cols-2">
        {#each GROUPS as g}
          <div>
            <h3 class="mb-2 text-[11px] font-semibold uppercase tracking-widest text-night-300">
              {g.title}
            </h3>
            <dl class="flex flex-col gap-1.5">
              {#each g.items as item}
                <div class="flex items-center justify-between gap-3 text-sm">
                  <dt class="text-night-100">{item.label}</dt>
                  <dd class="flex items-center gap-1">
                    {#each item.keys as k, i}
                      <kbd class="rounded border border-night-600 bg-night-800 px-1.5 py-0.5 text-[11px] text-slate-200">
                        {k}
                      </kbd>
                      {#if i < item.keys.length - 1}
                        <span class="text-night-300">+</span>
                      {/if}
                    {/each}
                  </dd>
                </div>
              {/each}
            </dl>
          </div>
        {/each}
      </div>
    </div>
  </div>
{/if}
