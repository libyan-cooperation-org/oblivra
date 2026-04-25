<!--
  SessionRestoreBanner — Phase 23.2.

  Backend (internal/services/session_persistence.go) saves the active SSH
  session host IDs and tab order on graceful shutdown. On the next startup
  this banner offers the operator one click to reconnect everything they
  had open at quit time, instead of forcing them to manually re-pick each
  bookmark from the sidebar.

  The banner is shown exactly once per app launch — once the operator has
  decided (Restore / Dismiss), we don't bring it back until next launch.

  Backend integration is opportunistic: if the SessionPersistence binding
  isn't available (browser mode, or the session file doesn't exist yet),
  the banner silently never renders.
-->
<script lang="ts">
  import { onMount } from 'svelte';
  import { History, X, Check } from 'lucide-svelte';
  import { IS_BROWSER } from '@lib/context';
  import { appStore } from '@lib/stores/app.svelte';

  interface PersistedSession {
    host_id: string;
    label?: string;
  }

  let candidates = $state<PersistedSession[]>([]);
  let restoring = $state(false);
  let dismissed = $state(false);

  onMount(async () => {
    if (IS_BROWSER) return;
    try {
      const mod = await import(
        '../../../bindings/github.com/kingknull/oblivrashell/internal/services/sessionpersistence.js'
      );
      // The persistence service exposes the saved-session list under
      // various names depending on Wails binding generation. Try the
      // common ones; bail out silently if none match.
      const fn =
        (mod as any).LoadState ??
        (mod as any).GetSavedSessions ??
        (mod as any).List;
      if (typeof fn !== 'function') return;
      const result = await fn();
      const sessions = Array.isArray(result)
        ? result
        : Array.isArray(result?.sessions)
          ? result.sessions
          : [];
      candidates = sessions
        .filter((s: any) => s && (s.host_id || s.hostId))
        .map((s: any) => ({
          host_id: s.host_id ?? s.hostId,
          label: s.label ?? s.host_id ?? s.hostId,
        }));
    } catch {
      // Binding unavailable in this build — silently no-op.
    }
  });

  async function restore() {
    if (restoring) return;
    restoring = true;
    try {
      for (const s of candidates) {
        try {
          await appStore.connectToHost?.(s.host_id);
        } catch (err) {
          appStore.notify?.(`Failed to reconnect ${s.label}: ${err}`, 'error');
        }
      }
      appStore.notify?.(`Restored ${candidates.length} previous session${candidates.length === 1 ? '' : 's'}`, 'info');
      dismissed = true;
    } finally {
      restoring = false;
    }
  }

  const visible = $derived(!dismissed && candidates.length > 0);
</script>

{#if visible}
  <div
    class="flex items-center gap-3 px-3 h-7 bg-accent/8 border border-accent/30 rounded-sm text-[10px] font-mono"
    role="status"
    aria-live="polite"
  >
    <History class="w-3.5 h-3.5 shrink-0 text-accent" />
    <span class="text-text-heading">
      Restore {candidates.length} previous session{candidates.length === 1 ? '' : 's'}?
    </span>
    <span class="text-text-muted truncate max-w-md">
      {candidates.map((s) => s.label).join(', ')}
    </span>

    <div class="flex-1"></div>

    <button
      class="flex items-center gap-1 text-accent hover:text-text-heading hover:bg-accent transition-colors bg-transparent border border-accent/40 rounded-sm px-2 cursor-pointer text-[9px] uppercase tracking-wider font-bold disabled:opacity-50"
      onclick={restore}
      disabled={restoring}
      title="Reconnect all"
    >
      <Check class="w-3 h-3" />
      <span>{restoring ? 'Restoring…' : 'Restore'}</span>
    </button>

    <button
      class="text-text-muted hover:text-text-heading transition-colors bg-transparent border-none cursor-pointer p-0.5"
      onclick={() => (dismissed = true)}
      aria-label="Dismiss"
      title="Dismiss"
    >
      <X class="w-3 h-3" />
    </button>
  </div>
{/if}
