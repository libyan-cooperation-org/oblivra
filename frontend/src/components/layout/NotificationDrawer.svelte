<!--
  NotificationDrawer — slide-in panel listing the operator's recent
  notifications. Triggered by the bell icon in TitleBar; complements the
  ephemeral toast notifications by giving them a persistent home.

  Action Center-style behaviour:
    - Click the entry → navigate to the route / dismiss
    - Per-entry trash → remove from history
    - Footer: "Mark all read" / "Clear all"
-->
<script lang="ts">
  import { Bell, X, Trash2, Check, ChevronRight } from 'lucide-svelte';
  import { tick } from 'svelte';
  import { notificationStore, type NotificationEntry } from '@lib/stores/notifications.svelte';
  import { appStore } from '@lib/stores/app.svelte';

  const open = $derived(notificationStore.drawerOpen);

  let asideEl = $state<HTMLElement | null>(null);
  let lastFocused = $state<HTMLElement | null>(null);

  // Focus management — when the drawer opens, remember what had focus,
  // shift focus into the drawer, and restore on close. Escape closes.
  $effect(() => {
    if (open) {
      lastFocused = (document.activeElement as HTMLElement) ?? null;
      tick().then(() => {
        // First focusable element inside the drawer
        const first = asideEl?.querySelector<HTMLElement>(
          'button, [href], input, [tabindex]:not([tabindex="-1"])',
        );
        first?.focus();
      });
    } else {
      // Restore focus to the bell button (or whatever opened us).
      lastFocused?.focus?.();
    }
  });

  function onWindowKey(e: KeyboardEvent) {
    if (!open) return;
    if (e.key === 'Escape') {
      e.preventDefault();
      notificationStore.drawerOpen = false;
      return;
    }
    // Tab trap: keep focus inside the drawer while open. Without this,
    // tabbing escapes back into the page underneath.
    if (e.key === 'Tab' && asideEl) {
      const focusables = asideEl.querySelectorAll<HTMLElement>(
        'button:not([disabled]), [href], input:not([disabled]), [tabindex]:not([tabindex="-1"])',
      );
      if (focusables.length === 0) return;
      const first = focusables[0];
      const last = focusables[focusables.length - 1];
      if (e.shiftKey && document.activeElement === first) {
        e.preventDefault();
        last.focus();
      } else if (!e.shiftKey && document.activeElement === last) {
        e.preventDefault();
        first.focus();
      }
    }
  }

  function levelClass(level: NotificationEntry['level']) {
    switch (level) {
      case 'critical':
      case 'error':   return 'border-error/40 text-error';
      case 'warning': return 'border-warning/40 text-warning';
      case 'success': return 'border-success/40 text-success';
      default:        return 'border-accent/30 text-accent';
    }
  }

  function relativeTime(iso: string): string {
    const then = new Date(iso).getTime();
    if (!isFinite(then)) return iso;
    const diff = (Date.now() - then) / 1000;
    if (diff < 60) return 'just now';
    if (diff < 3600) return `${Math.floor(diff / 60)}m ago`;
    if (diff < 86400) return `${Math.floor(diff / 3600)}h ago`;
    return new Date(iso).toLocaleString();
  }

  function activate(entry: NotificationEntry) {
    notificationStore.markRead(entry.id);
    if (entry.action?.route) {
      appStore.navigate(entry.action.route);
      notificationStore.drawerOpen = false;
    } else if (entry.action?.href) {
      window.open(entry.action.href, '_blank', 'noopener,noreferrer');
    }
  }
</script>

<svelte:window onkeydown={onWindowKey} />

{#if open}
  <!-- Backdrop — pointer cursor signals it's clickable; full-area button
       gives mouse + keyboard parity (operators using only the keyboard
       can Tab to it and Enter). -->
  <button
    class="fixed inset-0 bg-black/30 z-40 border-none cursor-pointer"
    onclick={() => (notificationStore.drawerOpen = false)}
    aria-label="Close notifications panel"
    tabindex="-1"
  ></button>

  <aside
    bind:this={asideEl}
    class="fixed top-8 right-0 bottom-0 w-96 bg-surface-1 border-l border-border-primary z-50 flex flex-col shadow-2xl"
    role="dialog"
    aria-modal="true"
    aria-labelledby="notification-drawer-title"
  >
    <!-- Header -->
    <header class="flex items-center justify-between px-4 h-10 border-b border-border-primary shrink-0">
      <div class="flex items-center gap-2">
        <Bell class="w-3.5 h-3.5 text-text-muted" />
        <span id="notification-drawer-title" class="text-[11px] font-mono font-bold uppercase tracking-widest text-text-heading">
          Notifications
        </span>
        {#if notificationStore.unreadCount > 0}
          <span class="px-1.5 py-px text-[8px] font-mono font-bold rounded-sm bg-accent/20 text-accent border border-accent/40">
            {notificationStore.unreadCount} NEW
          </span>
        {/if}
      </div>
      <button
        class="p-1 text-text-muted hover:text-text-heading transition-colors bg-transparent border-none cursor-pointer"
        onclick={() => (notificationStore.drawerOpen = false)}
        aria-label="Close"
      >
        <X class="w-4 h-4" />
      </button>
    </header>

    <!-- List -->
    <div class="flex-1 overflow-y-auto">
      {#if notificationStore.entries.length === 0}
        <div class="flex flex-col items-center justify-center py-20 px-6 gap-3 text-center">
          <Bell class="w-8 h-8 text-text-muted opacity-30" />
          <p class="text-[11px] font-mono text-text-muted uppercase tracking-wider">No notifications yet</p>
          <p class="text-[10px] text-text-muted opacity-60">Alerts, system events, and warnings appear here.</p>
        </div>
      {:else}
        {#each notificationStore.entries as entry (entry.id)}
          <div
            class="px-4 py-3 border-b border-border-primary/50 hover:bg-surface-2 transition-colors group"
            class:unread={!entry.read}
          >
            <div class="flex items-start gap-2">
              <div class="w-1 self-stretch rounded-full {levelClass(entry.level)} bg-current/40 shrink-0" aria-hidden="true"></div>

              <button
                class="flex-1 min-w-0 text-left bg-transparent border-none cursor-pointer p-0"
                onclick={() => activate(entry)}
              >
                <div class="flex items-center gap-2">
                  <span class="text-[10px] font-mono font-bold uppercase tracking-wider {levelClass(entry.level).split(' ')[1]}">
                    {entry.level}
                  </span>
                  {#if !entry.read}
                    <span class="w-1.5 h-1.5 rounded-full bg-accent shrink-0"></span>
                  {/if}
                  <span class="text-[9px] font-mono text-text-muted ml-auto shrink-0">{relativeTime(entry.ts)}</span>
                </div>
                <div class="text-[11px] font-mono font-semibold text-text-heading mt-1 truncate">
                  {entry.title}
                </div>
                {#if entry.message}
                  <div class="text-[10px] text-text-muted mt-0.5 line-clamp-2">{entry.message}</div>
                {/if}
                {#if entry.action?.label}
                  <div class="flex items-center gap-1 text-[9px] font-mono uppercase tracking-wider text-accent mt-1.5 group-hover:text-accent-hover">
                    <span>{entry.action.label}</span>
                    <ChevronRight class="w-2.5 h-2.5" />
                  </div>
                {/if}
              </button>

              <button
                class="opacity-0 group-hover:opacity-60 hover:opacity-100 transition-opacity bg-transparent border-none cursor-pointer p-0.5"
                onclick={(e) => { e.stopPropagation(); notificationStore.remove(entry.id); }}
                aria-label="Remove notification"
                title="Remove"
              >
                <Trash2 class="w-3 h-3 text-text-muted" />
              </button>
            </div>
          </div>
        {/each}
      {/if}
    </div>

    <!-- Footer -->
    {#if notificationStore.entries.length > 0}
      <footer class="flex items-center gap-2 px-3 h-9 border-t border-border-primary shrink-0">
        <button
          class="flex items-center gap-1 text-[9px] font-mono uppercase tracking-wider text-text-muted hover:text-text-heading bg-transparent border-none cursor-pointer"
          onclick={() => notificationStore.markAllRead()}
        >
          <Check class="w-3 h-3" />
          <span>Mark all read</span>
        </button>
        <div class="flex-1"></div>
        <button
          class="flex items-center gap-1 text-[9px] font-mono uppercase tracking-wider text-text-muted hover:text-error bg-transparent border-none cursor-pointer"
          onclick={() => notificationStore.clearAll()}
        >
          <Trash2 class="w-3 h-3" />
          <span>Clear all</span>
        </button>
      </footer>
    {/if}
  </aside>
{/if}

<style>
  .unread {
    background: rgba(0, 153, 224, 0.04);
  }

  .line-clamp-2 {
    display: -webkit-box;
    -webkit-line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }
</style>
