<script lang="ts">
  import { NAV, GROUP_LABEL, type NavId, type NavItem } from '../nav';

  let { open = $bindable(true), active = $bindable<NavId>('overview') } = $props();

  const grouped = $derived(
    (['siem', 'respond', 'manage'] as const).map((g) => ({
      group: g,
      items: NAV.filter((n) => n.group === g),
    })),
  );

  function select(item: NavItem) {
    active = item.id;
  }
</script>

<aside
  class="flex flex-col border-r border-night-700 bg-night-900/80 backdrop-blur transition-all"
  class:w-64={open}
  class:w-16={!open}
>
  <div class="flex h-14 items-center gap-3 border-b border-night-700 px-4">
    <div
      class="grid h-8 w-8 place-items-center rounded-md bg-gradient-to-br from-accent-500 to-accent-600 font-bold text-white shadow-lg shadow-accent-500/30"
    >
      Ø
    </div>
    {#if open}
      <div class="flex flex-col leading-tight">
        <span class="text-sm font-semibold tracking-wide text-slate-100">OBLIVRA</span>
        <span class="text-[10px] uppercase tracking-widest text-night-300">Sovereign SIEM</span>
      </div>
    {/if}
  </div>

  <nav class="flex-1 overflow-y-auto scrollbar-thin px-2 py-3">
    {#each grouped as section (section.group)}
      {#if open}
        <div class="px-3 pb-1 pt-3 text-[10px] font-semibold uppercase tracking-widest text-night-300">
          {GROUP_LABEL[section.group]}
        </div>
      {/if}
      <ul class="flex flex-col gap-0.5">
        {#each section.items as item (item.id)}
          <li>
            <button
              type="button"
              onclick={() => select(item)}
              class="group flex w-full items-center gap-3 rounded-md px-3 py-2 text-left text-sm text-night-200 transition hover:bg-night-700/50 hover:text-white"
              class:bg-accent-500={active === item.id}
              class:text-white={active === item.id}
              class:shadow-md={active === item.id}
              class:shadow-accent-500={active === item.id}
              title={item.hint ?? item.label}
            >
              <span class="text-base leading-none">{item.icon}</span>
              {#if open}
                <span class="flex-1 truncate">{item.label}</span>
              {/if}
            </button>
          </li>
        {/each}
      </ul>
    {/each}
  </nav>

  <button
    type="button"
    onclick={() => (open = !open)}
    class="border-t border-night-700 px-4 py-2 text-left text-xs text-night-300 hover:bg-night-700/40 hover:text-white"
    title="Toggle sidebar (Ctrl/Cmd+B)"
  >
    {open ? '◀ Collapse' : '▶'}
  </button>
</aside>
