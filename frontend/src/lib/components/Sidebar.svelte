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
  class="relative flex flex-col border-r border-base-700 bg-base-900 transition-all duration-200 select-none"
  style="width: {open ? '220px' : '52px'}; flex-shrink: 0;"
>
  <!-- Subtle vertical cyan glow line on right edge -->
  <div class="pointer-events-none absolute right-0 top-0 bottom-0 w-px"
       style="background: linear-gradient(to bottom, transparent, rgba(0,188,216,0.15) 30%, rgba(0,188,216,0.15) 70%, transparent);"></div>

  <!-- Logo / brand -->
  <div class="flex h-12 items-center gap-3 border-b border-base-700 px-3" style="flex-shrink:0;">
    <div class="relative grid h-8 w-8 place-items-center"
         style="flex-shrink:0; background: linear-gradient(135deg, #00bcd8 0%, #006b80 100%); clip-path: polygon(15% 0%, 85% 0%, 100% 15%, 100% 85%, 85% 100%, 15% 100%, 0% 85%, 0% 15%);">
      <span style="font-family: 'Rajdhani', sans-serif; font-weight:700; font-size:13px; color:#000; letter-spacing:-1px;">Ø</span>
    </div>
    {#if open}
      <div class="flex flex-col leading-tight overflow-hidden">
        <span style="font-family:'Rajdhani',sans-serif; font-weight:700; font-size:15px; letter-spacing:3px; color:#e8f4f8;">OBLIVRA</span>
        <span style="font-family:'Share Tech Mono',monospace; font-size:9px; letter-spacing:2px; color: var(--color-cyan-500); opacity:0.8;">SOVEREIGN · SIEM</span>
      </div>
    {/if}
  </div>

  <!-- Navigation -->
  <nav class="flex-1 overflow-y-auto scrollbar-thin py-2">
    {#each grouped as section (section.group)}
      {#if open}
        <div class="flex items-center gap-2 px-3 pb-1 pt-3">
          <span style="font-family:'Share Tech Mono',monospace; font-size:9px; letter-spacing:3px; color:var(--color-base-300); text-transform:uppercase;">{GROUP_LABEL[section.group]}</span>
          <div class="flex-1 h-px" style="background: var(--color-base-600);"></div>
        </div>
      {:else}
        <div class="py-2 px-1.5">
          <div class="h-px" style="background: var(--color-base-700);"></div>
        </div>
      {/if}

      <ul class="flex flex-col gap-0.5 px-1.5">
        {#each section.items as item (item.id)}
          {@const isActive = active === item.id}
          <li>
            <button
              type="button"
              onclick={() => select(item)}
              class="group relative flex w-full items-center gap-2.5 text-left transition-all duration-100"
              style="
                padding: {open ? '6px 10px 6px 12px' : '7px 0'};
                justify-content: {open ? 'flex-start' : 'center'};
                border-radius: 2px;
                background: {isActive ? 'rgba(0,188,216,0.10)' : 'transparent'};
                border: 1px solid {isActive ? 'rgba(0,188,216,0.25)' : 'transparent'};
              "
              title={item.hint ?? item.label}
            >
              {#if isActive}
                <span class="nav-active-bar"></span>
              {/if}

              <!-- Icon -->
              <span style="
                font-size: 13px;
                line-height: 1;
                color: {isActive ? 'var(--color-cyan-400)' : 'var(--color-base-300)'};
                filter: {isActive ? 'drop-shadow(0 0 4px var(--color-cyan-500))' : 'none'};
                transition: color 0.15s, filter 0.15s;
                flex-shrink: 0;
                {!open ? 'text-align: center; width: 100%;' : ''}
              ">{item.icon}</span>

              {#if open}
                <span style="
                  font-family: 'Rajdhani', sans-serif;
                  font-weight: {isActive ? '600' : '500'};
                  font-size: 13px;
                  letter-spacing: 0.5px;
                  color: {isActive ? 'var(--color-cyan-400)' : 'var(--color-base-200)'};
                  flex: 1;
                  white-space: nowrap;
                  overflow: hidden;
                  text-overflow: ellipsis;
                  text-transform: uppercase;
                ">{item.label}</span>
              {/if}
            </button>
          </li>
        {/each}
      </ul>
    {/each}
  </nav>

  <!-- Collapse toggle -->
  <button
    type="button"
    onclick={() => (open = !open)}
    class="border-t border-base-700 transition-colors"
    style="
      padding: 8px {open ? '12px' : '0'};
      display: flex;
      align-items: center;
      justify-content: {open ? 'space-between' : 'center'};
      gap: 8px;
      background: transparent;
      color: var(--color-base-300);
      font-family: 'Share Tech Mono', monospace;
      font-size: 10px;
      letter-spacing: 1px;
    "
    title="Toggle sidebar (Ctrl/Cmd+B)"
    onmouseenter={(e) => { (e.currentTarget as HTMLElement).style.color = 'var(--color-cyan-400)'; }}
    onmouseleave={(e) => { (e.currentTarget as HTMLElement).style.color = 'var(--color-base-300)'; }}
  >
    {#if open}
      <span>◀ COLLAPSE</span>
    {:else}
      <span>▶</span>
    {/if}
  </button>
</aside>
