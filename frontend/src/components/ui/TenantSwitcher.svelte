<!--
  TenantSwitcher — top-bar dropdown that selects the active tenant
  context. Phase 30.4d: closes the "no tenant switcher in UI" gap from
  the operator UX audit.

  Reads from `multiTenantStore.tenants` (server-fetched list).
  Writes to `appStore.currentTenantId` (UI scope).

  When `currentTenantId === null` the operator is viewing platform-wide
  state (admin perspective). When a specific tenant is selected the
  switcher renders the tenant's color + abbreviation as a chip.

  Note on backend wiring: this is the UI-side primitive. Plumbing the
  selected tenant into outbound API requests is a separate task —
  tracked in task.md as a follow-up to Phase 30.4d.
-->
<script lang="ts">
  import { onMount } from 'svelte';
  import { Building2, Check, ChevronDown, X } from 'lucide-svelte';
  import { appStore } from '@lib/stores/app.svelte';

  // Lazy-import the multi-tenant store so this widget can render even
  // when the platform is in single-tenant deployment mode (the store
  // returns an empty list and the dropdown collapses to a static label).
  // We use top-level import here because `MultiTenantStore` is a
  // regular Svelte 5 store; if the backend doesn't populate tenants
  // the list is just empty.
  import { tenantStore, type Tenant } from '@lib/stores/tenant.svelte';

  let open = $state(false);
  let containerEl = $state<HTMLDivElement | null>(null);

  const tenants = $derived<Tenant[]>(tenantStore.tenants ?? []);
  const current = $derived<Tenant | null>(
    appStore.currentTenantId
      ? tenants.find((t) => t.id === appStore.currentTenantId) ?? null
      : null,
  );

  function toggle() { open = !open; }

  function pick(t: Tenant | null) {
    appStore.setCurrentTenant(t?.id ?? null);
    open = false;
  }

  function onDocClick(e: MouseEvent) {
    if (!open) return;
    if (containerEl && !containerEl.contains(e.target as Node)) {
      open = false;
    }
  }

  onMount(() => {
    // Lazy-fetch tenant list if empty so the switcher populates on
    // first interaction.
    if (tenants.length === 0) {
      tenantStore.refresh?.();
    }
    document.addEventListener('mousedown', onDocClick);
    return () => document.removeEventListener('mousedown', onDocClick);
  });
</script>

<div class="ts-root" bind:this={containerEl}>
  <button
    type="button"
    class="ts-trigger"
    class:active={open}
    onclick={toggle}
    title={current ? `Tenant: ${current.name}` : 'All tenants (platform admin view)'}
    aria-haspopup="listbox"
    aria-expanded={open}
  >
    {#if current}
      <span
        class="ts-chip"
        style="background-color: {current.color ?? 'var(--color-accent)'}"
      >{current.abbr}</span>
      <span class="ts-name">{current.name}</span>
    {:else}
      <Building2 size={11} strokeWidth={1.6} class="ts-icon" />
      <span class="ts-name">All Tenants</span>
    {/if}
    <ChevronDown size={11} strokeWidth={1.8} class="ts-caret" />
  </button>

  {#if open}
    <div class="ts-popover" role="listbox" aria-label="Switch tenant">
      <button
        type="button"
        class="ts-item"
        class:selected={!current}
        onclick={() => pick(null)}
      >
        <Building2 size={11} strokeWidth={1.6} />
        <span class="ts-item-name">All Tenants</span>
        <span class="ts-item-sub">platform admin</span>
        {#if !current}
          <Check size={10} strokeWidth={2} class="ts-check" />
        {/if}
      </button>

      {#if tenants.length > 0}
        <div class="ts-divider"></div>
        {#each tenants as t (t.id)}
          <button
            type="button"
            class="ts-item"
            class:selected={current?.id === t.id}
            onclick={() => pick(t)}
          >
            <span
              class="ts-chip ts-chip-sm"
              style="background-color: {t.color ?? 'var(--color-accent)'}"
            >{t.abbr}</span>
            <span class="ts-item-name">{t.name}</span>
            <span class="ts-item-sub">{t.tier ?? ''} · {t.agents ?? 0} agents</span>
            {#if current?.id === t.id}
              <Check size={10} strokeWidth={2} class="ts-check" />
            {/if}
          </button>
        {/each}
      {:else}
        <div class="ts-empty">
          No tenants registered. Single-tenant deployment.
        </div>
      {/if}

      {#if current}
        <div class="ts-divider"></div>
        <button type="button" class="ts-item ts-item-clear" onclick={() => pick(null)}>
          <X size={10} strokeWidth={1.8} />
          <span class="ts-item-name">Clear scope</span>
        </button>
      {/if}
    </div>
  {/if}
</div>

<style>
  .ts-root { position: relative; display: inline-block; }

  .ts-trigger {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    padding: 4px 8px;
    background: transparent;
    border: 1px solid var(--color-border-primary);
    border-radius: 4px;
    color: var(--color-text-secondary);
    cursor: pointer;
    font-family: var(--font-mono);
    font-size: 10px;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    transition: color 100ms, border-color 100ms, background 100ms;
  }
  .ts-trigger:hover,
  .ts-trigger.active {
    color: var(--color-text-heading);
    border-color: var(--color-border-hover);
    background: var(--color-surface-2);
  }
  :global(.ts-icon) { color: var(--color-text-muted); }
  :global(.ts-caret) { color: var(--color-text-muted); margin-left: 2px; }

  .ts-chip {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    width: 18px;
    height: 18px;
    border-radius: 4px;
    font-family: var(--font-mono);
    font-size: 9px;
    font-weight: 800;
    color: white;
    letter-spacing: 0;
    flex-shrink: 0;
  }
  .ts-chip-sm {
    width: 16px;
    height: 16px;
    font-size: 8px;
  }
  .ts-name { font-weight: 700; }

  /* ── Popover ─────────────────────────────────────────── */
  .ts-popover {
    position: absolute;
    top: calc(100% + 4px);
    right: 0;
    min-width: 240px;
    background: var(--color-surface-2);
    border: 1px solid var(--color-border-primary);
    border-radius: 6px;
    padding: 4px;
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.4);
    z-index: 100;
    display: flex;
    flex-direction: column;
    gap: 1px;
    max-height: 320px;
    overflow-y: auto;
  }

  .ts-divider {
    height: 1px;
    background: var(--color-border-primary);
    margin: 2px 0;
  }

  .ts-item {
    display: grid;
    grid-template-columns: 18px 1fr auto auto;
    align-items: center;
    gap: 8px;
    padding: 6px 8px;
    background: transparent;
    border: none;
    border-radius: 3px;
    cursor: pointer;
    text-align: left;
    transition: background 100ms;
  }
  .ts-item:hover { background: var(--color-surface-3); }
  .ts-item.selected {
    background: var(--color-sev-info-bg);
  }

  .ts-item-name {
    font-family: var(--font-ui);
    font-size: 11px;
    font-weight: 600;
    color: var(--color-text-heading);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .ts-item-sub {
    font-family: var(--font-mono);
    font-size: 9px;
    color: var(--color-text-muted);
    text-transform: uppercase;
    letter-spacing: 0.05em;
    white-space: nowrap;
  }
  :global(.ts-check) { color: var(--color-accent); }

  .ts-item-clear {
    color: var(--color-text-muted);
  }
  .ts-item-clear:hover {
    color: var(--color-text-heading);
  }

  .ts-empty {
    padding: 12px 8px;
    font-family: var(--font-mono);
    font-size: 10px;
    color: var(--color-text-muted);
    text-align: center;
    opacity: 0.7;
  }
</style>
