<!--
  OBLIVRA — Terminal Tabs (Svelte 5)
  Tab orchestration for PTY sessions.

  Phase 23.x — drag-to-pop-out: an operator can grab a tab and drag it
  out of the tab strip. On drop OUTSIDE the strip's bounds we spawn a
  new Wails window pinned to that session via WindowService.PopOut.
  This matches the Termius / VSCode pattern of "tear off into its own
  window for a second monitor."
-->
<script lang="ts">
  import { appStore } from '@lib/stores/app.svelte';
  import { IS_BROWSER } from '@lib/context';
  import { toastStore } from '@lib/stores/toast.svelte';
  import { t } from '@lib/i18n';
  import { Terminal as TerminalIcon, X } from 'lucide-svelte';

  let stripEl = $state<HTMLDivElement | null>(null);
  let dragSessionId = $state<string | null>(null);

  function closeSession(id: string) {
    appStore.removeSession(id);
  }

  /**
   * Drag start — record the session being dragged. We use the dataTransfer
   * for the standard "draggable" payload but also keep a Svelte-side ref
   * since the dataTransfer isn't readable until drop in some browsers.
   */
  function onDragStart(e: DragEvent, sessionId: string) {
    if (IS_BROWSER) return; // pop-out only meaningful in Wails desktop
    dragSessionId = sessionId;
    if (e.dataTransfer) {
      e.dataTransfer.effectAllowed = 'move';
      e.dataTransfer.setData('application/x-oblivra-session', sessionId);
    }
  }

  function onDragEnd(e: DragEvent) {
    // If the pointer left the tab strip's bounding rect, treat that as
    // "tear off into a new window." Browsers fire dragend on the source
    // regardless of where the drop happened, so we check coordinates.
    const id = dragSessionId;
    dragSessionId = null;
    if (!id || IS_BROWSER) return;

    if (!stripEl) return;
    const rect = stripEl.getBoundingClientRect();
    const insideStrip =
      e.clientX >= rect.left &&
      e.clientX <= rect.right &&
      e.clientY >= rect.top &&
      e.clientY <= rect.bottom;
    if (insideStrip) return; // dropped back into the strip — no-op

    // Tear off — popout the terminal route with the session id.
    void popOutSession(id);
  }

  async function popOutSession(sessionId: string) {
    try {
      const mod = await import(
        '../../../bindings/github.com/kingknull/oblivrashell/internal/services/windowservice.js'
      );
      if (typeof mod.PopOut !== 'function') {
        toastStore.add({
          type: 'warning',
          title: t('popout.unavailable.title'),
          message: 'WindowService binding missing PopOut method.',
        });
        return;
      }
      // Encode the session id in the route so the new window can resume
      // it (TerminalPage reads ?session=<id> on mount). Falls back to a
      // bare /terminal if the page ignores the query.
      const route = `/terminal?session=${encodeURIComponent(sessionId)}`;
      const session = appStore.sessions.find((s) => s.id === sessionId);
      const title = session?.hostLabel ?? `Terminal — ${sessionId.slice(0, 8)}`;
      await mod.PopOut(route, title);
    } catch (err) {
      console.error('[TerminalTabs] tear-off pop-out failed:', err);
      toastStore.add({
        type: 'error',
        title: t('popout.failed.title'),
        message: err instanceof Error ? err.message : String(err),
      });
    }
  }
</script>

<div
  bind:this={stripEl}
  class="flex items-center gap-1 border-b border-border-primary shrink-0 overflow-x-auto bg-surface-1"
>
  {#each appStore.sessions as session (session.id)}
    <div
      role="tab"
      tabindex="0"
      draggable="true"
      class="group flex items-center gap-2 px-3 py-1.5 min-w-[120px] max-w-[200px] border-b-2 transition-all duration-fast cursor-pointer
        {appStore.activeSessionId === session.id
          ? 'border-accent bg-accent/5 text-text-primary'
          : 'border-transparent text-text-muted hover:bg-surface-2 hover:text-text-secondary'}"
      class:dragging={dragSessionId === session.id}
      onclick={() => appStore.setActiveSession(session.id)}
      onkeydown={(e) => e.key === 'Enter' && appStore.setActiveSession(session.id)}
      ondragstart={(e) => onDragStart(e, session.id)}
      ondragend={onDragEnd}
      title="Drag tab outside the bar to pop it into a new window"
    >
      <TerminalIcon class="w-3 h-3 shrink-0" />
      <span class="text-[11px] font-bold truncate flex-1 text-left">{session.hostLabel || 'Local Shell'}</span>
      <button
        class="opacity-0 group-hover:opacity-100 px-1 text-text-muted hover:text-error transition-opacity cursor-pointer border-none bg-transparent"
        onclick={(e) => { e.stopPropagation(); closeSession(session.id); }}
        aria-label={t('common.close')}
        title={t('common.close')}
      >
        <X class="w-3 h-3" />
      </button>
    </div>
  {/each}
</div>

<style>
  .dragging {
    opacity: 0.5;
    cursor: grabbing;
  }
</style>
