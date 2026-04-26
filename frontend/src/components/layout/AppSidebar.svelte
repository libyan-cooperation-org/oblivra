<!--
  AppSidebar — vertical group selector for the new sidebar+dock UX.

  Renders 6 group buttons (Operate / Detect / Investigate / Defend /
  Govern / System), each showing an icon + label. Clicking a group
  sets it active in `navigationStore`; the BottomDock listens and
  renders the corresponding items.

  Visual language (Linear.app-inspired, SOC dark mode):
    - 64px wide, slightly wider than CommandRail to fit the label
    - Active group: accent-coloured left rail + tinted background
    - Hover: subtle surface-2 fill, no animation cost
    - Pinned items: small bottom strip below the group list, always-on
      shortcuts that survive group switches

  Coexistence: this component does NOT replace CommandRail directly —
  AppLayout opts into one or the other based on `appStore.useGroupedNav`.
-->
<script lang="ts">
  import {
    Activity,
    Radar,
    Search,
    Shield,
    Scale,
    Settings,
    PinOff,
    type Icon as IconType,
  } from 'lucide-svelte';
  import { navigationStore, type NavGroupId } from '@lib/stores/navigation.svelte';
  import { NAV_GROUPS, allItems } from '@lib/nav-config';
  import { appStore } from '@lib/stores/app.svelte';
  import { push } from '@lib/router.svelte';
  import { IS_BROWSER, IS_DESKTOP, IS_HYBRID } from '@lib/context';

  // Map icon names → component refs. Keeping this local (rather than
  // computing from lucide-svelte by string) preserves Vite tree-shaking.
  const GROUP_ICONS: Record<NavGroupId, typeof IconType> = {
    operate: Activity,
    detect: Radar,
    investigate: Search,
    defend: Shield,
    govern: Scale,
    system: Settings,
  };

  // Filter pinned items by current platform context, since a desktop-only
  // route shouldn't render in the web build even if it was pinned earlier.
  const visiblePinned = $derived.by(() => {
    const all = allItems();
    return navigationStore.pinned
      .map((id) => all.find((it) => it.id === id))
      .filter((it): it is NonNullable<typeof it> => {
        if (!it) return false;
        const ctx = it.context ?? 'both';
        if (ctx === 'both') return true;
        if (ctx === 'desktop') return IS_DESKTOP || IS_HYBRID;
        if (ctx === 'browser') return IS_BROWSER || IS_HYBRID;
        return true;
      });
  });

  function selectGroup(g: NavGroupId) {
    navigationStore.setActiveGroup(g);
    // If the group has a remembered route, navigate there for parity with
    // the operator's last view.
    const last = navigationStore.lastRouteByGroup[g];
    if (last) {
      const item = allItems().find((it) => it.id === last);
      if (item) {
        appStore.setActiveNavTab(item.id as any);
        push(item.route);
      }
    }
  }

  function activatePinned(routeId: string, route: string) {
    appStore.setActiveNavTab(routeId as any);
    push(route);
  }

  function unpin(e: MouseEvent, routeId: string) {
    e.stopPropagation();
    navigationStore.togglePin(routeId);
  }
</script>

<aside class="sidebar" aria-label="Primary navigation">
  <!-- Brand mark -->
  <div class="brand">
    <span class="brand-text">O<em class="brand-em">I</em></span>
  </div>

  <!-- Groups -->
  <nav class="groups" aria-label="Navigation groups">
    {#each NAV_GROUPS as group (group.id)}
      {@const Icon = GROUP_ICONS[group.id]}
      {@const isActive = navigationStore.activeGroup === group.id}
      <button
        type="button"
        class="group-btn"
        class:active={isActive}
        title={group.subtitle}
        aria-pressed={isActive}
        onclick={() => selectGroup(group.id)}
      >
        <Icon class="group-icon" size={18} strokeWidth={1.6} />
        <span class="group-label">{group.label}</span>
      </button>
    {/each}
  </nav>

  <!-- Spacer pushes pinned strip to the bottom -->
  <div class="spacer"></div>

  <!-- Pinned items -->
  {#if visiblePinned.length > 0}
    <div class="pinned" aria-label="Pinned items">
      <div class="pinned-label">PINNED</div>
      {#each visiblePinned as item (item.id)}
        <!-- Two sibling buttons inside a row container — must NOT nest
             <button> inside <button> (invalid HTML; browsers will lift
             the inner button out and break Svelte's hydration). -->
        <div
          class="pin-row"
          class:active={appStore.activeNavTab === item.id}
        >
          <button
            type="button"
            class="pin-btn"
            title={item.label}
            onclick={() => activatePinned(item.id, item.route)}
          >
            <span class="pin-dot" aria-hidden="true"></span>
            <span class="pin-label">{item.label}</span>
          </button>
          <button
            type="button"
            class="pin-remove"
            aria-label="Unpin {item.label}"
            title="Unpin"
            onclick={(e) => unpin(e, item.id)}
          >
            <PinOff size={11} strokeWidth={1.8} />
          </button>
        </div>
      {/each}
    </div>
  {/if}
</aside>

<style>
  .sidebar {
    width: 64px;
    min-width: 64px;
    height: 100%;
    background: var(--s1);
    border-right: 1px solid var(--b1);
    display: flex;
    flex-direction: column;
    z-index: 1000;
    overflow: hidden;
    flex-shrink: 0;
  }

  .brand {
    height: 44px;
    min-height: 44px;
    display: flex;
    align-items: center;
    justify-content: center;
    border-bottom: 1px solid var(--b1);
    flex-shrink: 0;
  }
  .brand-text {
    font-family: var(--mn);
    font-size: 9px;
    font-weight: 700;
    color: #d0e8f8;
    letter-spacing: 0.12em;
  }
  .brand-em { color: var(--cr2); font-style: normal; }

  .groups {
    display: flex;
    flex-direction: column;
    padding: 6px 0;
    gap: 2px;
  }

  .group-btn {
    position: relative;
    width: 100%;
    height: 56px;
    background: transparent;
    border: none;
    color: var(--tx3);
    cursor: pointer;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    gap: 4px;
    padding: 6px 4px;
    transition: color 150ms ease-in-out, background 150ms ease-in-out;
    flex-shrink: 0;
    border-radius: 8px;
    margin: 0 6px;
  }
  .group-btn:hover {
    color: var(--tx);
    background: var(--s2);
  }
  .group-btn.active {
    color: var(--ac2);
    background: rgba(24, 120, 200, 0.12);
  }
  .group-btn.active::before {
    content: '';
    position: absolute;
    left: -6px;
    top: 25%;
    bottom: 25%;
    width: 2px;
    background: var(--ac);
    border-radius: 0 2px 2px 0;
    box-shadow: 0 0 6px var(--ac);
  }
  :global([dir='rtl']) .group-btn.active::before {
    left: auto;
    right: -6px;
    border-radius: 2px 0 0 2px;
  }

  :global(.group-icon) {
    transition: opacity 150ms ease-in-out;
  }
  .group-btn:not(.active) :global(.group-icon) { opacity: 0.55; }
  .group-btn.active :global(.group-icon) { opacity: 1; }

  .group-label {
    font-family: var(--mn);
    font-size: 9px;
    font-weight: 700;
    letter-spacing: 0.06em;
    text-transform: uppercase;
    line-height: 1;
  }

  .spacer { flex: 1; min-height: 8px; }

  /* ── Pinned strip ──────────────────────────────────────────── */
  .pinned {
    border-top: 1px solid var(--b1);
    padding: 6px 4px 8px;
    display: flex;
    flex-direction: column;
    gap: 2px;
    max-height: 30%;
    overflow-y: auto;
  }
  .pinned-label {
    font-family: var(--mn);
    font-size: 8px;
    font-weight: 800;
    color: var(--tx3);
    letter-spacing: 0.15em;
    padding: 2px 4px 4px;
  }
  .pin-row {
    position: relative;
    display: flex;
    align-items: center;
    width: 100%;
    border-radius: 4px;
    transition: background 100ms;
    color: var(--tx3);
  }
  .pin-row:hover { background: var(--s2); color: var(--tx); }
  .pin-row.active { color: var(--ac2); background: rgba(24, 120, 200, 0.08); }

  .pin-btn {
    position: relative;
    display: flex;
    align-items: center;
    gap: 4px;
    flex: 1;
    min-width: 0;
    background: transparent;
    border: none;
    color: inherit;
    cursor: pointer;
    padding: 4px 4px;
    border-radius: 4px;
    text-align: left;
  }

  .pin-dot {
    width: 4px;
    height: 4px;
    border-radius: 50%;
    background: currentColor;
    flex-shrink: 0;
    opacity: 0.6;
  }
  .pin-label {
    font-family: var(--sn);
    font-size: 9px;
    font-weight: 600;
    line-height: 1;
    flex: 1;
    text-align: left;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .pin-remove {
    background: transparent;
    border: none;
    color: var(--tx3);
    cursor: pointer;
    padding: 2px;
    opacity: 0;
    transition: opacity 150ms;
    border-radius: 2px;
    flex-shrink: 0;
  }
  .pin-row:hover .pin-remove { opacity: 0.7; }
  .pin-remove:hover { opacity: 1; color: var(--tx); }
</style>
