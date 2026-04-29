<!--
  OBLIVRA TitleBar — platform-aware window chrome.

  Frameless Wails windows (main_gui.go sets Frameless: true) leave the OS
  with no native min/max/close, so we render our own. Mac users expect
  traffic-light dots on the LEFT; Windows / Linux users expect explicit
  Minimize / Maximize / Close icons on the RIGHT. Showing the wrong
  pattern is the difference between "obvious, native, professional" and
  the user reporting "there is no maximize close."

  Drag region: header sets `--wails-draggable: drag` so the operator can
  drag the window between monitors. Every interactive element overrides
  with `--wails-draggable: no-drag` so clicks pass through to the button.
  (The older `-webkit-app-region: drag` is Electron's API; Wails v3
  silently ignores it — that's why the operator reported the app could
  not be dragged at all on the previous build.)
-->
<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { IS_BROWSER } from '@lib/context';
  import { appStore } from '@lib/stores/app.svelte';
  import { alertStore } from '@lib/stores/alerts.svelte';
  import { Minus, Square, X, Copy as Restore, Monitor, ExternalLink, Layout, Bell } from 'lucide-svelte';
  import { notificationStore } from '@lib/stores/notifications.svelte';
  import TenantSwitcher from '@components/ui/TenantSwitcher.svelte';
  import TimeRangePicker from '@components/ui/TimeRangePicker.svelte';
  import SovereigntyBadge from '@components/layout/SovereigntyBadge.svelte';
  import { timeRangeStore } from '@lib/stores/timeRange.svelte';

  // Platform detection — userAgent is constant per process, so we resolve it
  // once at module load instead of on every $derived re-evaluation.
  type Platform = 'mac' | 'win' | 'linux';
  const platform: Platform = (() => {
    if (typeof window === 'undefined') return 'win';
    const ua = window.navigator.userAgent.toLowerCase();
    if (ua.includes('mac')) return 'mac';
    if (ua.includes('linux')) return 'linux';
    return 'win';
  })();

  // Audit fix — window-control visibility used to gate solely on
  // `IS_BROWSER` from context.ts, computed at module load. The Wails
  // v3 runtime injects its globals (window._wails) AFTER
  // WindowLoadFinished, which fires after the JS bundle parses; on
  // Windows WebView2 the `window.chrome.webview` channel is normally
  // synchronous but can race the bundle on cold start. When detection
  // misfires the operator sees no min / max / close buttons at all
  // (the OS chrome is suppressed by Frameless: true in main_gui.go).
  //
  // We now treat "should we draw window controls" as a separate
  // reactive signal: we render them whenever the page lives inside a
  // Wails / WebView host, re-evaluated on every render so a late
  // runtime injection makes them appear within one tick.
  let inWailsHost = $state(false);
  function detectWailsHost(): boolean {
    if (typeof window === 'undefined') return false;
    const w = window as any;
    return !!(
      w.chrome?.webview                       // Windows WebView2
      || w.webkit?.messageHandlers?.external  // macOS / Linux WebKit2GTK
      || w._wails                             // Wails v3 namespace
      || w.__WAILS__                          // Wails v2 namespace
      || w.runtime                            // Wails v2 legacy
      || w.wails                              // Wails v2 legacy
    );
  }
  // showWindowControls = "render our own chrome buttons". Falls back
  // to the original IS_BROWSER signal when Wails detection is uncertain
  // so we never DOUBLE-render OS chrome on a real browser.
  const showWindowControls = $derived(inWailsHost || !IS_BROWSER);

  let isMaximised = $state(false);
  let popoutCount = $state(0);

  const critCount = $derived(
    alertStore?.alerts?.filter((a: any) => a.severity === 'critical').length ?? 0
  );
  const highCount = $derived(
    alertStore?.alerts?.filter((a: any) => a.severity === 'high').length ?? 0
  );

  let pollInterval: ReturnType<typeof setInterval> | undefined;

  onMount(() => {
    // Re-evaluate the Wails-host signal once on mount AND once after a
    // tick (covers a late `_wails` injection by Wails v3's
    // WindowLoadFinished hook). After this initial probe, the
    // `showWindowControls` derived expression is stable.
    inWailsHost = detectWailsHost();
    if (!inWailsHost) {
      // Try again after the next animation frame — Wails v3 injects
      // _wails during DOM ready, which can fire AFTER our first paint.
      requestAnimationFrame(() => {
        inWailsHost = detectWailsHost();
      });
      // And once more after 500ms as a belt-and-braces guard against
      // very slow WebView2 cold starts on Windows.
      setTimeout(() => {
        if (!inWailsHost) inWailsHost = detectWailsHost();
      }, 500);
    }

    if (IS_BROWSER && !inWailsHost) return;

    const pollState = async () => {
      try {
        const { Window } = await import('@wailsio/runtime');
        // Window.IsMaximised returns a Promise<boolean>; await it so the
        // state stays accurate as the operator drags the window between
        // monitors and toggles the chrome.
        isMaximised = await (Window.IsMaximised as any)();
      } catch {
        /* dev mode — runtime may not be wired yet */
      }

      try {
        const mod = await import('../../../bindings/github.com/kingknull/oblivrashell/internal/services/windowservice.js');
        if (mod && typeof mod.ListPopouts === 'function') {
          const ids = await mod.ListPopouts();
          const next = Array.isArray(ids) ? ids.length : 0;
          // Adaptive cadence — 1.5s when actively tracking pop-outs, 8s
          // when idle so we're not burning RPCs to confirm "still 0".
          if (next !== popoutCount) {
            popoutCount = next;
            schedule(next > 0 ? 1500 : 8000);
          }
        }
      } catch {
        // Bindings only exist in desktop builds; ignore in web.
      }
    };

    function schedule(delayMs: number) {
      if (pollInterval) clearInterval(pollInterval);
      pollInterval = setInterval(pollState, delayMs);
    }

    // Pause polling entirely while the window is hidden — saves RPCs
    // when the operator is in another app on a different monitor.
    function onVisibilityChange() {
      if (document.hidden) {
        if (pollInterval) {
          clearInterval(pollInterval);
          pollInterval = undefined;
        }
      } else {
        pollState();
        schedule(popoutCount > 0 ? 1500 : 8000);
      }
    }
    document.addEventListener('visibilitychange', onVisibilityChange);

    pollState();
    schedule(8000); // start in idle cadence; pollState bumps to 1.5s if needed

    return () => {
      document.removeEventListener('visibilitychange', onVisibilityChange);
    };
  });

  onDestroy(() => {
    if (pollInterval) {
      clearInterval(pollInterval);
      pollInterval = undefined;
    }
  });

  async function windowClose() {
    const { Application } = await import('@wailsio/runtime');
    Application.Quit();
  }
  async function windowMinimize() {
    const { Window } = await import('@wailsio/runtime');
    Window.Minimise();
  }
  async function windowToggleMax() {
    const { Window } = await import('@wailsio/runtime');
    Window.ToggleMaximise();
    // Optimistic update so the icon flips before the next poll tick.
    isMaximised = !isMaximised;
  }

  async function closeAllPopouts() {
    try {
      const mod = await import('../../../bindings/github.com/kingknull/oblivrashell/internal/services/windowservice.js');
      if (mod && typeof mod.CloseAllPopouts === 'function') {
        await mod.CloseAllPopouts();
        popoutCount = 0;
      }
    } catch { /* dev */ }
  }

  // Drag region is handled entirely by Wails v3's runtime (drag.js):
  // any element with the CSS custom property `--wails-draggable: drag`
  // becomes a drag handle, and children with `--wails-draggable: no-drag`
  // pass clicks through. The previous `-webkit-app-region: drag` was an
  // Electron-era property that Wails v3 silently ignores — that's why
  // the operator reported the app couldn't be dragged at all.
  //
  // The `Window.Drag()` JS fallback we used to call doesn't exist in the
  // Wails v3 runtime either; v3 sends the drag start as a `wails:drag`
  // IPC message internally. We don't need to dispatch it manually.
</script>

<header
  class="flex items-center h-8 bg-surface-1 border-b border-border-primary select-none z-50 px-2 gap-3 shrink-0 cursor-grab active:cursor-grabbing hover:bg-surface-2 transition-colors duration-200"
  style="--wails-draggable: drag;"
>
  <!-- macOS traffic lights (left side, Mac-only) -->
  {#if showWindowControls && platform === 'mac'}
    <div class="flex items-center gap-1.5 shrink-0 pl-1 pr-1" style="--wails-draggable: no-drag;">
      <button
        class="w-3 h-3 rounded-full bg-[#ff5f57] hover:opacity-80 transition-opacity border-none cursor-pointer flex items-center justify-center group"
        onclick={windowClose}
        aria-label="Close window"
      >
        <X class="w-2 h-2 text-black/60 opacity-0 group-hover:opacity-100" />
      </button>
      <button
        class="w-3 h-3 rounded-full bg-[#ffbd2e] hover:opacity-80 transition-opacity border-none cursor-pointer flex items-center justify-center group"
        onclick={windowMinimize}
        aria-label="Minimize window"
      >
        <Minus class="w-2 h-2 text-black/60 opacity-0 group-hover:opacity-100" />
      </button>
      <button
        class="w-3 h-3 rounded-full bg-[#28c840] hover:opacity-80 transition-opacity border-none cursor-pointer flex items-center justify-center group"
        onclick={windowToggleMax}
        aria-label="Toggle maximize window"
      >
        <Square class="w-2 h-2 text-black/60 opacity-0 group-hover:opacity-100" />
      </button>
    </div>
    <div class="w-px h-3.5 bg-border-primary shrink-0"></div>
  {/if}

  <!-- Brand + context badge -->
  <div class="flex items-center gap-2 shrink-0">
    <span class="text-text-heading font-mono text-[11px] font-semibold tracking-[0.1em]">
      OBL<em class="text-error not-italic">IV</em>RA
    </span>
    <span class="px-1.5 py-px text-[8px] font-mono font-bold tracking-widest rounded-sm border
      {IS_BROWSER
        ? 'text-accent-hover border-accent/30 bg-accent/8'
        : 'text-[#9878e0] border-[#9878e0]/30 bg-[#9878e0]/8'}">
      {IS_BROWSER ? 'WEB' : 'DESKTOP'}
    </span>
  </div>

  <div class="w-px h-3.5 bg-border-primary shrink-0"></div>

  <!-- Tenant switcher (Phase 30.4d) — operator-visible scope selector.
       In single-tenant deployments the dropdown collapses to a static
       "All Tenants" pill. Wrapped in no-drag so clicks don't initiate
       a window drag through the title bar. -->
  <div class="shrink-0" style="--wails-draggable: no-drag;">
    <TenantSwitcher />
  </div>

  <!-- Sovereignty badge (Phase 32) — single-glance posture indicator.
       Click to expand the four-pillar breakdown (on-prem · TPM · air-gap · KMS). -->
  <div class="shrink-0" style="--wails-draggable: no-drag;">
    <SovereigntyBadge />
  </div>

  <!-- Global time-range scope (Phase 31). Lives in the title bar so
       every page can read it from `timeRangeStore`. Pages that don't
       care (Settings, etc.) simply ignore it. -->
  <div class="shrink-0" style="--wails-draggable: no-drag;">
    <TimeRangePicker
      value={timeRangeStore.range}
      onChange={(r) => timeRangeStore.set(r)}
    />
  </div>

  <div class="w-px h-3.5 bg-border-primary shrink-0"></div>

  <!-- Sovereign status — hidden below xl so the chrome doesn't run
       the window-controls off the right edge on narrower viewports. -->
  <div class="hidden xl:flex items-center gap-1.5 shrink-0">
    <div class="w-1.5 h-1.5 rounded-full bg-success shrink-0"></div>
    <span class="text-[9px] font-mono text-text-muted uppercase tracking-wider">Sovereign Cloud</span>
  </div>

  <!-- Pop-out indicator — visible only when one or more pop-outs are open -->
  {#if popoutCount > 0}
    <div class="flex items-center gap-1.5 shrink-0">
      <div class="w-px h-3.5 bg-border-primary"></div>
      <button
        class="flex items-center gap-1 text-[8px] font-mono text-text-muted hover:text-text-heading uppercase tracking-wider bg-transparent border-none cursor-pointer"
        onclick={closeAllPopouts}
        title="Close all pop-out windows"
        style="--wails-draggable: no-drag;"
      >
        <Monitor class="w-3 h-3" />
        <span>{popoutCount} POP-OUT{popoutCount === 1 ? '' : 'S'}</span>
      </button>
    </div>
  {/if}

  <!-- Severity chips -->
  {#if critCount > 0 || highCount > 0}
    <div class="flex items-center gap-1.5 shrink-0">
      <div class="w-px h-3.5 bg-border-primary"></div>
      {#if critCount > 0}
        <span class="px-1.5 py-px text-[8px] font-mono font-bold rounded-sm
          bg-error/12 text-error border border-error/28">CRIT {critCount}</span>
      {/if}
      {#if highCount > 0}
        <span class="px-1.5 py-px text-[8px] font-mono font-bold rounded-sm
          bg-warning/12 text-warning border border-warning/25" style="--wails-draggable: no-drag;">HIGH {highCount}</span>
      {/if}
    </div>
  {/if}

  <!-- Centered command search — fills the drag area but is allowed to
       shrink (min-w-0) so the right-edge window controls always fit.
       Without min-w-0, flex-1 children won't shrink below their content
       width, which on a busy chrome pushes the close button off-screen. -->
  <div class="flex-1 min-w-0 flex justify-center">
    <button
      type="button"
      class="flex items-center bg-surface-3 border border-border-primary rounded-sm px-2.5 h-[20px] gap-2 max-w-[240px] w-full min-w-[100px]
             hover:border-border-hover transition-colors cursor-pointer"
      onclick={() => appStore.toggleCommandPalette()}
      style="--wails-draggable: no-drag;"
    >
      <span class="text-text-muted text-[8px] font-mono tracking-wide opacity-60 truncate">Search commands…</span>
      <span class="ml-auto text-text-muted text-[8px] font-mono opacity-40 shrink-0">⌃K</span>
    </button>
  </div>

  <!-- Notification bell -->
  <button
    class="relative h-8 w-8 flex items-center justify-center text-text-muted hover:text-text-heading hover:bg-surface-2 transition-colors border-none bg-transparent cursor-pointer shrink-0"
    onclick={() => notificationStore.toggleDrawer()}
    aria-label="Notifications {notificationStore.unreadCount > 0 ? `(${notificationStore.unreadCount} unread)` : ''}"
    title={notificationStore.unreadCount > 0 ? `${notificationStore.unreadCount} unread notifications` : 'Notifications'}
    style="--wails-draggable: no-drag;"
  >
    <Bell class="w-3.5 h-3.5" />
    {#if notificationStore.unreadCount > 0}
      <span
        class="absolute top-1 right-1 min-w-[14px] h-[14px] px-1 rounded-full text-[8px] font-mono font-bold flex items-center justify-center {notificationStore.criticalUnread > 0 ? 'bg-error text-white' : 'bg-accent text-black'}"
        aria-hidden="true"
      >
        {notificationStore.unreadCount > 99 ? '99+' : notificationStore.unreadCount}
      </span>
    {/if}
  </button>

  <!-- Operator — name hidden below lg so the chrome stays compact;
       initials chip stays visible at every breakpoint. -->
  <div class="flex items-center gap-2 shrink-0">
    <span class="hidden lg:inline text-[9px] font-mono text-text-muted">OPERATOR ·</span>
    <span class="hidden lg:inline text-[9px] font-mono text-text-heading font-semibold uppercase tracking-tight">K. MAVERICK</span>
    <div class="w-5 h-5 rounded-sm flex items-center justify-center text-[9px] font-bold font-mono
                bg-accent/15 border border-accent/30 text-accent-hover">KM</div>
  </div>

  <!-- Global Desktop Actions + window controls.
       Anchored to the right edge with `shrink-0` so a narrow viewport
       can't push them off — earlier symptom: the close button was
       invisible on a 1280-wide window because the search bar refused
       to shrink. The vertical separator before the controls helps the
       eye find them as a logical group.
       Audit fix — gating switched from IS_BROWSER (which can misfire
       on cold-start WebView2 timing) to showWindowControls, which
       reactively responds to a late Wails runtime injection. -->
  {#if showWindowControls}
    <div class="flex items-center shrink-0 ml-auto">
      <!-- SOC mode (multi-monitor pop-out shortcut) — label only on
           lg+ to keep the chrome compact on smaller windows. -->
      <button
        class="h-8 px-2 flex items-center justify-center gap-1.5 text-text-muted hover:text-accent hover:bg-surface-2 transition-colors border-none bg-transparent cursor-pointer group"
        onclick={() => appStore.launchSOCExperience()}
        title="Launch SOC Multi-Monitor Experience (3+ Windows)"
        style="--wails-draggable: no-drag;"
      >
        <Layout class="w-3.5 h-3.5" />
        <span class="text-[9px] font-mono font-bold tracking-widest hidden xl:block opacity-60 group-hover:opacity-100">SOC MODE</span>
      </button>

      <button
        class="h-8 w-10 flex items-center justify-center text-text-muted hover:text-accent hover:bg-surface-2 transition-colors border-none bg-transparent cursor-pointer"
        onclick={() => appStore.popOut()}
        aria-label="Pop out into new window"
        title="Pop out into new window"
        style="--wails-draggable: no-drag;"
      >
        <ExternalLink class="w-3.5 h-3.5" />
      </button>

      {#if platform !== 'mac'}
        <!-- Visible separator + window controls. Bumped icon size
             from 14 px → 16 px so they're unmistakable as window
             controls (the previous version was easy to miss against
             the dark header). The close button keeps its red-on-hover
             behaviour to match Windows native conventions. -->
        <span class="w-px h-4 bg-border-primary mx-1 shrink-0" aria-hidden="true"></span>
        <button
          class="h-8 w-11 flex items-center justify-center text-text-secondary hover:text-text-heading hover:bg-surface-2 transition-colors border-none bg-transparent cursor-pointer"
          onclick={windowMinimize}
          aria-label="Minimize window"
          title="Minimize"
          style="--wails-draggable: no-drag;"
        >
          <Minus class="w-4 h-4" />
        </button>
        <button
          class="h-8 w-11 flex items-center justify-center text-text-secondary hover:text-text-heading hover:bg-surface-2 transition-colors border-none bg-transparent cursor-pointer"
          onclick={windowToggleMax}
          aria-label={isMaximised ? 'Restore window' : 'Maximize window'}
          title={isMaximised ? 'Restore' : 'Maximize'}
          style="--wails-draggable: no-drag;"
        >
          {#if isMaximised}
            <Restore class="w-4 h-4" />
          {:else}
            <Square class="w-4 h-4" />
          {/if}
        </button>
        <button
          class="h-8 w-11 flex items-center justify-center text-text-secondary hover:text-white hover:bg-error transition-colors border-none bg-transparent cursor-pointer"
          onclick={windowClose}
          aria-label="Close window"
          title="Close"
          style="--wails-draggable: no-drag;"
        >
          <X class="w-4 h-4" />
        </button>
      {/if}
    </div>
  {/if}
</header>
