<!--
  BottomDock — slide-up dock panel showing the items belonging to the
  active group selected in `AppSidebar`.

  Behaviour:
    - Slides up from the bottom of the viewport when a group is active +
      the dock is expanded (state lives in `navigationStore`).
    - Items render as horizontally-scrolling cards (label + icon +
      optional description). Clicking an item routes there and updates
      `appStore.activeNavTab`.
    - Right-click (or ⋯ button) on an item toggles its pin — pinned
      items are duplicated into the AppSidebar's "Pinned" strip below
      the group buttons.
    - Header has the group title + a "Collapse" chevron that minimizes
      the dock to a 24px peek strip (still clickable to re-expand).
    - Honours `prefers-reduced-motion`: the slide animation drops to
      an opacity fade for users who disabled motion.

  Visual language:
    - Fixed to bottom of the main content area (NOT the viewport — sits
      above StatusBar so the operator's status info stays visible).
    - Dark surface with border + subtle shadow above the line so it
      reads as a separate plane.
    - Item cards: 8px rounded corners, 1px border, accent-tinted on
      hover, accent left-rail on the active route.
-->
<script lang="ts">
  /*
   * IMPORTANT: every icon used by `nav-config.ts` MUST be statically
   * imported here. Vite tree-shakes lucide-svelte aggressively in
   * production builds — `import * as LucideIcons` and then string
   * lookups DOES NOT WORK in the compiled exe (icons are silently
   * removed and the lookup returns undefined → blank screen).
   *
   * Add new nav-config items? Add the matching icon import below AND
   * an entry in ICON_MAP. The Phase 29 postmortem (task.md) covers
   * the same failure mode for missing imports.
   */
  import {
    // chrome / dock controls
    Activity, Radar, Search, Scale, Settings, FileText,
    ChevronDown, ChevronUp, Pin, PinOff, Circle,
    // overview / dashboards
    LayoutDashboard, TrendingUp, Monitor, Eye,
    // hosts / devices
    Server, Terminal, Crosshair, FolderOpen, Cpu, HardDrive, Boxes,
    // network / geo
    Network, Cable, Wifi, Map,
    // identity / users
    Users, UserCog, UsersRound, Video,
    // security / detection
    Shield, Bell, BellRing, ShieldAlert, ShieldCheck, ShieldHalf,
    Telescope, Globe, Grid3x3, Zap, Skull, Swords, Play, Siren, Flame,
    GitBranch, AlertTriangle,
    // logs / data
    Database, Sparkles, History, FileSearch, Microscope, BookLock, BookOpen,
    Link, Workflow, Clock4, Rewind, GitCompare, Code, StickyNote, HeartPulse,
    // governance / system
    ClipboardCheck, KeyRound, Lock, EyeOff, Award, Puzzle, RefreshCw,
    Bot, Keyboard,
    type Icon as IconType,
  } from 'lucide-svelte';
  import { navigationStore, type NavGroupId } from '@lib/stores/navigation.svelte';
  import { getGroup, type NavItem } from '@lib/nav-config';
  import { appStore } from '@lib/stores/app.svelte';
  import { push } from '@lib/router.svelte';
  import { IS_BROWSER, IS_DESKTOP, IS_HYBRID } from '@lib/context';

  const GROUP_HEADER_ICONS: Record<NavGroupId, typeof IconType> = {
    overview: LayoutDashboard,
    security: Shield,
    network:  Network,
    identity: UserCog,
    hosts:    Server,
    logs:     FileText,
    system:   Settings,
  };

  // Static lookup. Strings here MUST match `icon` field in nav-config.ts.
  // If a string has no entry, lookupIcon() falls back to Circle.
  // (Phase 29 lesson: NEVER use `import * as` + string lookup against
  // tree-shakeable libs — Vite strips icons not referenced by name.)
  const ICON_MAP: Record<string, typeof IconType> = {
    LayoutDashboard, Terminal, Crosshair, Zap, FolderOpen, Monitor, Eye,
    Server, Cable, Code, StickyNote, Video,
    Database, Search, Bell, BellRing, Telescope, Globe, Sparkles, Grid3x3,
    Network, Map, HeartPulse, Users,
    GitBranch, History, FileSearch, Microscope, HardDrive, BookLock,
    Link, Workflow, Clock4, Rewind, GitCompare,
    BookOpen, ShieldAlert, Skull, Wifi, Swords, Play, Siren, Flame,
    ClipboardCheck, KeyRound, Lock, UserCog, UsersRound, ShieldCheck,
    EyeOff, ShieldHalf, Award,
    Cpu, Boxes, Puzzle, RefreshCw, TrendingUp, Bot, Settings, Keyboard,
    Activity, Radar, Shield, Scale, FileText, AlertTriangle,
  };

  // Reactive: which group is showing right now, and which items belong
  // to it (filtered by platform context).
  const activeGroup = $derived(getGroup(navigationStore.activeGroup));

  const visibleItems = $derived.by<NavItem[]>(() => {
    if (!activeGroup) return [];
    return activeGroup.items.filter((it) => {
      const ctx = it.context ?? 'both';
      if (ctx === 'both') return true;
      if (ctx === 'desktop') return IS_DESKTOP || IS_HYBRID;
      if (ctx === 'browser') return IS_BROWSER || IS_HYBRID;
      return true;
    });
  });

  function activate(item: NavItem) {
    appStore.setActiveNavTab(item.id as any);
    navigationStore.rememberRoute(navigationStore.activeGroup, item.id);
    push(item.route);
  }

  function togglePin(e: MouseEvent, item: NavItem) {
    e.stopPropagation();
    e.preventDefault();
    navigationStore.togglePin(item.id);
  }

  function lookupIcon(name: string): typeof IconType {
    // Static map lookup — see ICON_MAP definition above for why we
    // can't use `import * as` + string access in production.
    return ICON_MAP[name] ?? Circle;
  }

  // Pulled into a derived so we don't recompute on every keystroke.
  const HeaderIcon = $derived(
    GROUP_HEADER_ICONS[navigationStore.activeGroup] ?? LayoutDashboard,
  );

  /**
   * Wheel-to-horizontal scroll. Mouse wheels emit vertical deltas
   * (`deltaY`) by default; without this, scrolling the wheel over the
   * dock does nothing useful (the parent page swallows it). We convert
   * vertical wheel motion into horizontal scrollLeft changes and only
   * consume the event when there's actually horizontal overflow — so
   * touchpad users with native horizontal scroll still get their
   * native behaviour, and trackpad two-finger swipes work too.
   *
   * Holding Shift gives the OS-default horizontal-wheel behaviour
   * across platforms; we keep that working untouched.
   */
  function onWheel(e: WheelEvent) {
    const el = e.currentTarget as HTMLElement;
    if (!el) return;
    const overflow = el.scrollWidth - el.clientWidth;
    if (overflow <= 0) return;             // nothing to scroll
    if (e.deltaX !== 0) return;            // OS already did it
    if (e.deltaY === 0) return;            // pinch / no-op
    el.scrollLeft += e.deltaY;
    e.preventDefault();
  }

  /** Keyboard scroll: arrow keys nudge by one card width. */
  function onItemsKeydown(e: KeyboardEvent) {
    const el = e.currentTarget as HTMLElement;
    if (!el) return;
    const step = 200; // ≈ one card + gap
    if (e.key === 'ArrowRight') {
      el.scrollLeft += step;
      e.preventDefault();
    } else if (e.key === 'ArrowLeft') {
      el.scrollLeft -= step;
      e.preventDefault();
    } else if (e.key === 'Home') {
      el.scrollLeft = 0;
      e.preventDefault();
    } else if (e.key === 'End') {
      el.scrollLeft = el.scrollWidth;
      e.preventDefault();
    }
  }
</script>

{#if activeGroup}
  <section
    class="dock"
    class:collapsed={!navigationStore.dockExpanded}
    aria-label="{activeGroup.label} tools"
  >
    <!-- Header -->
    <header class="dock-header">
      <div class="dock-title">
        <HeaderIcon class="dock-title-icon" size={14} strokeWidth={1.6} />
        <span class="dock-title-text">{activeGroup.label}</span>
        <span class="dock-subtitle">{activeGroup.subtitle}</span>
      </div>
      <button
        type="button"
        class="dock-toggle"
        title={navigationStore.dockExpanded ? 'Collapse dock' : 'Expand dock'}
        aria-label={navigationStore.dockExpanded ? 'Collapse dock' : 'Expand dock'}
        onclick={() => navigationStore.toggleDock()}
      >
        {#if navigationStore.dockExpanded}
          <ChevronDown size={14} strokeWidth={1.8} />
        {:else}
          <ChevronUp size={14} strokeWidth={1.8} />
        {/if}
      </button>
    </header>

    <!-- Items strip -->
    {#if navigationStore.dockExpanded}
      <!-- tabindex on the scrollable container so keyboard users can
           focus it and Arrow/Home/End scroll horizontally. -->
      <div
        class="dock-items"
        role="list"
        tabindex="0"
        onwheel={onWheel}
        onkeydown={onItemsKeydown}
      >
        {#each visibleItems as item (item.id)}
          {@const Icon = lookupIcon(item.icon)}
          {@const isActive = appStore.activeNavTab === item.id}
          {@const pinned = navigationStore.isPinned(item.id)}
          <!-- Two sibling buttons inside a card container — must NOT
               nest <button> inside <button> (invalid HTML). -->
          <div
            role="listitem"
            class="dock-item"
            class:active={isActive}
            oncontextmenu={(e) => togglePin(e, item)}
            title="{item.label}{item.description ? ' — ' + item.description : ''}"
          >
            <button
              type="button"
              class="dock-item-activate"
              onclick={() => activate(item)}
            >
              <span class="dock-item-icon" aria-hidden="true">
                <Icon size={16} strokeWidth={1.6} />
              </span>
              <span class="dock-item-text">
                <span class="dock-item-label">{item.label}</span>
                {#if item.description}
                  <span class="dock-item-desc">{item.description}</span>
                {/if}
              </span>
            </button>
            <button
              type="button"
              class="dock-item-pin"
              class:pinned
              aria-label={pinned ? 'Unpin' : 'Pin'}
              title={pinned ? 'Unpin (or right-click)' : 'Pin (or right-click)'}
              onclick={(e) => togglePin(e, item)}
            >
              {#if pinned}
                <Pin size={11} strokeWidth={1.8} />
              {:else}
                <PinOff size={11} strokeWidth={1.8} />
              {/if}
            </button>
          </div>
        {/each}
      </div>
    {/if}
  </section>
{/if}

<style>
  .dock {
    flex-shrink: 0;
    background: var(--s1);
    border-top: 1px solid var(--b1);
    box-shadow: 0 -4px 16px rgba(0, 0, 0, 0.25);
    display: flex;
    flex-direction: column;
    /* Slide-up animation: translate when dock first appears, then settle. */
    animation: dock-slide-up 220ms ease-in-out;
    transform-origin: bottom;
    max-height: 200px;
    /* CRITICAL for horizontal scroll: as a flex column child of the
       app shell, the dock must not allow its own width to grow with
       its content. `min-width: 0` overrides the flexbox default of
       `min-width: auto` (which equals content width), letting the
       inner `.dock-items` actually clip and scroll. */
    min-width: 0;
    width: 100%;
    /* `position: relative` is REQUIRED for z-index to take effect.
       Without it, the InvestigationPanel's `position: fixed` backdrop
       (z-index 80) would capture clicks meant for dock buttons. */
    position: relative;
    z-index: 70;
  }
  .dock.collapsed {
    max-height: 28px;
  }

  @keyframes dock-slide-up {
    from {
      transform: translateY(8px);
      opacity: 0;
    }
    to {
      transform: translateY(0);
      opacity: 1;
    }
  }

  @media (prefers-reduced-motion: reduce) {
    .dock { animation: dock-fade-in 100ms ease-in-out; }
    @keyframes dock-fade-in {
      from { opacity: 0; }
      to { opacity: 1; }
    }
  }

  /* ── Header ─────────────────────────────────────────────── */
  .dock-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    height: 28px;
    padding: 0 12px;
    border-bottom: 1px solid var(--b1);
    flex-shrink: 0;
  }
  .dock.collapsed .dock-header { border-bottom: none; }

  .dock-title {
    display: flex;
    align-items: center;
    gap: 8px;
    color: var(--tx2);
    min-width: 0;
  }
  :global(.dock-title-icon) { color: var(--ac2); flex-shrink: 0; }

  .dock-title-text {
    font-family: var(--mn);
    font-size: 10px;
    font-weight: 700;
    color: var(--tx);
    text-transform: uppercase;
    letter-spacing: 0.1em;
    white-space: nowrap;
  }
  .dock-subtitle {
    font-family: var(--sn);
    font-size: 10px;
    color: var(--tx3);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .dock-toggle {
    background: transparent;
    border: none;
    color: var(--tx3);
    cursor: pointer;
    padding: 4px;
    border-radius: 4px;
    transition: color 100ms, background 100ms;
    display: flex;
    align-items: center;
    justify-content: center;
  }
  .dock-toggle:hover { color: var(--tx); background: var(--s2); }

  /* ── Item strip ─────────────────────────────────────────── */
  .dock-items {
    display: flex;
    gap: 6px;
    padding: 8px 12px;
    overflow-x: auto;
    overflow-y: hidden;
    flex: 1;
    /* Same min-width: 0 trick as on the dock — the items strip is a
       flex item itself (vertical flex), and again the default
       min-width: auto would let it grow with its content. */
    min-width: 0;
    width: 100%;
    /* Smooth scroll for the wheel-handler's setLeft assignments and
       for Home/End keyboard jumps. */
    scroll-behavior: smooth;
    scrollbar-width: thin;
    scrollbar-color: var(--b2) transparent;
    /* Strip default focus ring when the operator clicks; keep it for
       keyboard tab-into so accessibility users can see they're on the
       scrollable region. */
    outline: none;
  }
  .dock-items:focus-visible {
    box-shadow: inset 0 0 0 1px var(--ac);
  }
  .dock-items::-webkit-scrollbar { height: 8px; }
  .dock-items::-webkit-scrollbar-track { background: transparent; }
  .dock-items::-webkit-scrollbar-thumb {
    background: var(--b2);
    border-radius: 4px;
  }
  .dock-items::-webkit-scrollbar-thumb:hover {
    background: var(--tx3);
  }

  .dock-item {
    position: relative;
    display: flex;
    align-items: stretch;
    min-width: 180px;
    max-width: 240px;
    background: var(--s2);
    border: 1px solid var(--b1);
    border-radius: 8px;
    color: var(--tx2);
    transition:
      background 150ms ease-in-out,
      border-color 150ms ease-in-out,
      color 150ms ease-in-out,
      transform 150ms ease-in-out;
    flex-shrink: 0;
    overflow: hidden;
  }
  .dock-item:hover {
    background: var(--s3, rgba(255, 255, 255, 0.04));
    border-color: var(--b2);
    color: var(--tx);
    transform: translateY(-1px);
  }

  .dock-item-activate {
    flex: 1;
    display: flex;
    align-items: center;
    gap: 8px;
    padding: 8px 10px;
    background: transparent;
    border: none;
    color: inherit;
    cursor: pointer;
    text-align: left;
    min-width: 0;
  }
  .dock-item.active {
    border-color: var(--ac);
    background: rgba(24, 120, 200, 0.08);
    color: var(--ac2);
  }
  .dock-item.active::before {
    content: '';
    position: absolute;
    left: -1px;
    top: 8px;
    bottom: 8px;
    width: 2px;
    background: var(--ac);
    border-radius: 0 2px 2px 0;
    box-shadow: 0 0 6px var(--ac);
  }
  :global([dir='rtl']) .dock-item.active::before {
    left: auto;
    right: -1px;
    border-radius: 2px 0 0 2px;
  }

  .dock-item-icon {
    flex-shrink: 0;
    color: currentColor;
    opacity: 0.85;
    display: flex;
    align-items: center;
  }

  .dock-item-text {
    flex: 1;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 1px;
  }
  .dock-item-label {
    font-family: var(--sn);
    font-size: 11px;
    font-weight: 600;
    line-height: 1.1;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .dock-item-desc {
    font-family: var(--sn);
    font-size: 10px;
    color: var(--tx3);
    line-height: 1.1;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .dock-item.active .dock-item-desc { color: var(--tx2); }

  .dock-item-pin {
    background: transparent;
    border: none;
    color: var(--tx3);
    cursor: pointer;
    padding: 4px;
    border-radius: 4px;
    opacity: 0;
    transition: opacity 150ms, color 100ms, background 100ms;
    flex-shrink: 0;
  }
  .dock-item:hover .dock-item-pin,
  .dock-item-pin.pinned { opacity: 0.85; }
  .dock-item-pin.pinned { color: var(--ac2); opacity: 1; }
  .dock-item-pin:hover { color: var(--tx); background: var(--s2); }
</style>
