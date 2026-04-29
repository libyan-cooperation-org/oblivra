<!--
  PopOutButton — small action button that pops the current route into its
  own native Wails window. Drop into the toolbar of any page that operators
  want to drag onto a second monitor.

  Hidden in browser mode (no Wails runtime) and gracefully no-ops if the
  WindowService binding isn't loaded yet.
-->
<script lang="ts">
  import { ExternalLink } from 'lucide-svelte';
  import { IS_BROWSER } from '@lib/context';
  import { toastStore } from '@lib/stores/toast.svelte';
  import { t } from '@lib/i18n';
  import { getCurrentPath } from '@lib/router.svelte';

  interface Props {
    /** Route to pop out, e.g. "/siem-search". Defaults to the current
     *  view (the route the operator is looking at when they click). */
    route?: string;
    /** Window title; defaults to "OBLIVRA <route>". */
    title?: string;
    /** Optional CSS class hook. */
    class?: string;
  }

  let { route, title, class: cls = '' }: Props = $props();

  // Resolve the route to pop out.
  //
  // Bug fix: previously fell back to `window.location.pathname`, but the
  // app uses HASH routing (`#/siem-search`) — pathname is always `/`, so
  // every pop-out spawned the dashboard regardless of which view the
  // operator was looking at. We now read the hash via `getCurrentPath()`
  // (the same helper the router uses) so the pop-out shows the CURRENT
  // view, which is what the toolbar button promises.
  function resolveRoute(): string {
    if (route) return route;
    const path = getCurrentPath(); // strips leading '#'
    return path && path !== '' ? path : '/';
  }

  async function popOut() {
    if (IS_BROWSER) {
      // Web mode: open a same-origin tab with ?popout=1 so the spawned tab
      // hides the sidebar. Browsers handle window placement via the OS.
      const target = `${window.location.origin}/?popout=1&route=${encodeURIComponent(resolveRoute())}`;
      window.open(target, '_blank', 'noopener,noreferrer');
      return;
    }

    let mod: any = null;
    try {
      mod = await import(
        '../../../bindings/github.com/kingknull/oblivrashell/internal/services/windowservice.js'
      );
    } catch (err) {
      // Distinguish missing-file (refactor moved binding path) from any
      // other import error — both go to the console with detail so the
      // dev tools have the real error, while the toast stays user-readable.
      console.error('[PopOutButton] WindowService binding import failed:', err);
      toastStore.add({
        type: 'error',
        title: t('popout.unavailable.title'),
        message: 'WindowService binding could not be loaded — see console for details.',
      });
      return;
    }

    if (!mod || typeof mod.PopOut !== 'function') {
      console.warn('[PopOutButton] WindowService imported but PopOut not exported:', mod);
      toastStore.add({
        type: 'warning',
        title: t('popout.unavailable.title'),
        message: 'WindowService binding loaded without PopOut method (rebuild needed?).',
      });
      return;
    }

    try {
      const r = resolveRoute();
      const winTitle = title ?? '';
      await mod.PopOut(r, winTitle);
    } catch (err) {
      console.error('[PopOutButton] PopOut RPC failed:', err);
      toastStore.add({
        type: 'error',
        title: t('popout.failed.title'),
        message: err instanceof Error ? err.message : String(err),
      });
    }
  }
</script>

<button
  type="button"
  class="inline-flex items-center gap-1.5 px-2 h-7 text-[10px] font-mono uppercase tracking-wider
         text-text-muted hover:text-text-heading hover:bg-surface-2 border border-border-primary
         hover:border-border-hover transition-colors rounded-sm cursor-pointer {cls}"
  onclick={popOut}
  title={t('popout.button.tooltip')}
  aria-label={t('popout.button.tooltip')}
>
  <ExternalLink class="w-3 h-3" />
  <span>{t('popout.button')}</span>
</button>
