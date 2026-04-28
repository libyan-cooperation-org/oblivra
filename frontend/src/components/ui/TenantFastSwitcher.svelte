<!--
  OBLIVRA — Tenant Fast Switcher (Phase 32, MSP profile UX).

  Cmd+T opens a focused, type-to-filter palette for the operator's
  tenants. Designed for MSP / platform admins who context-switch across
  many customer tenants per shift.

  Shortcut behaviour:
    • Cmd+T (or Ctrl+T) opens the picker — only when the active
      profile sets tenantChrome to 'switcher-bar'. Other profiles get
      the regular dropdown via the chrome's TenantSwitcher.
    • Esc closes.
    • Up/Down navigate.
    • Enter picks.
    • "All Tenants" entry at the top selects null scope.

  Why a separate component vs. extending TenantSwitcher: the dropdown
  inside the chrome is mouse-friendly; the fast switcher is keyboard-
  first. They share the same data source (tenantStore) but have
  different interaction targets, so keeping them separate avoids
  bloating either with the other's concerns.
-->
<script lang="ts">
  import { onMount, onDestroy, tick } from 'svelte';
  import { Building2, Check } from 'lucide-svelte';
  import { appStore } from '@lib/stores/app.svelte';
  import { tenantStore, type Tenant } from '@lib/stores/tenant.svelte';

  let open = $state(false);
  let query = $state('');
  let cursor = $state(0);
  let inputEl: HTMLInputElement | null = $state(null);

  let tenants = $derived(tenantStore.tenants ?? []);

  type Item = { id: string | null; label: string; sub: string; tenant: Tenant | null };
  let items = $derived.by<Item[]>(() => {
    const all: Item[] = [
      { id: null, label: 'All Tenants', sub: 'Platform-admin view', tenant: null },
      ...tenants.map((t) => ({
        id: t.id,
        label: t.name ?? t.id,
        sub: t.id,
        tenant: t,
      })),
    ];
    const q = query.trim().toLowerCase();
    if (!q) return all;
    return all.filter((i) =>
      i.label.toLowerCase().includes(q) || i.sub.toLowerCase().includes(q),
    );
  });

  function close() {
    open = false;
    query = '';
    cursor = 0;
  }

  async function show() {
    if (tenants.length === 0) {
      try { await tenantStore.refresh?.(); } catch { /* ignore */ }
    }
    open = true;
    cursor = 0;
    query = '';
    await tick();
    inputEl?.focus();
  }

  function pick(idx: number) {
    const item = items[idx];
    if (!item) return;
    appStore.setCurrentTenant(item.id);
    close();
  }

  function onKeyDown(e: KeyboardEvent) {
    // ⌘T / Ctrl+T toggles the switcher when the profile asks for it.
    // Browsers steal Ctrl+T to open a tab — only Cmd works on Mac and
    // we accept Ctrl+Alt+T as the fallback on Windows/Linux so we
    // don't fight the browser.
    const isPlatformOpen =
      (e.metaKey && !e.ctrlKey && e.key.toLowerCase() === 't') ||
      (e.ctrlKey && e.altKey && e.key.toLowerCase() === 't');
    if (isPlatformOpen && appStore.profileRules.tenantChrome === 'switcher-bar') {
      e.preventDefault();
      open ? close() : void show();
      return;
    }

    if (!open) return;

    if (e.key === 'Escape') { e.preventDefault(); close(); return; }
    if (e.key === 'ArrowDown') {
      e.preventDefault();
      cursor = Math.min(cursor + 1, items.length - 1);
      return;
    }
    if (e.key === 'ArrowUp') {
      e.preventDefault();
      cursor = Math.max(0, cursor - 1);
      return;
    }
    if (e.key === 'Enter') {
      e.preventDefault();
      pick(cursor);
      return;
    }
  }

  onMount(() => window.addEventListener('keydown', onKeyDown));
  onDestroy(() => window.removeEventListener('keydown', onKeyDown));
</script>

{#if open}
  <div
    class="fixed inset-0 z-[10000] bg-black/50 flex items-start justify-center pt-[15vh]"
    role="presentation"
    onclick={close}
  >
    <div
      class="w-full max-w-md bg-surface-1 border border-border-secondary rounded-md shadow-2xl overflow-hidden"
      role="dialog"
      aria-label="Tenant fast switcher"
      onclick={(e) => e.stopPropagation()}
    >
      <div class="flex items-center gap-2 px-3 py-2 border-b border-border-primary">
        <Building2 size={12} class="text-accent" />
        <input
          bind:this={inputEl}
          type="text"
          class="flex-1 bg-transparent border-none outline-none text-text-primary text-[var(--fs-body)] font-mono placeholder:text-text-muted"
          placeholder="Switch tenant…"
          bind:value={query}
        />
        <kbd class="font-mono text-[var(--fs-micro)] text-text-muted px-1.5 py-0.5 bg-surface-3 rounded border border-border-primary">⌘T</kbd>
      </div>

      <ul class="flex flex-col max-h-[60vh] overflow-auto py-1">
        {#each items as item, i}
          {@const isActive = i === cursor}
          {@const isCurrent = appStore.currentTenantId === item.id}
          <li>
            <button
              class="w-full flex items-center gap-2 px-3 py-2 text-start transition-colors duration-fast {isActive ? 'bg-accent/10 border-l-2 border-accent' : 'border-l-2 border-transparent hover:bg-surface-2'}"
              onclick={() => pick(i)}
            >
              <Building2 size={11} class="text-text-muted shrink-0" />
              <div class="flex-1 min-w-0">
                <div class="text-[var(--fs-label)] font-bold text-text-secondary truncate">{item.label}</div>
                <div class="font-mono text-[var(--fs-micro)] text-text-muted truncate">{item.sub}</div>
              </div>
              {#if isCurrent}
                <Check size={11} class="text-success shrink-0" />
              {/if}
            </button>
          </li>
        {/each}
        {#if items.length === 0}
          <li class="px-3 py-3 text-[var(--fs-label)] text-text-muted italic">No tenants match.</li>
        {/if}
      </ul>

      <footer class="px-3 py-1.5 border-t border-border-primary bg-surface-2/40 flex items-center justify-between text-[var(--fs-micro)] font-mono text-text-muted">
        <span>↑↓ navigate · ⏎ select · esc close</span>
        <span>{items.length} tenant{items.length === 1 ? '' : 's'}</span>
      </footer>
    </div>
  </div>
{/if}
