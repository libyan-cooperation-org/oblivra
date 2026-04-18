<!-- OBLIVRA Web — CommandPalette (Svelte 5) -->
<script lang="ts">
  import { onMount, onDestroy } from 'svelte';

  export interface PaletteAction {
    id: string; label: string; description?: string; icon: string;
    shortcut?: string; action: () => void;
  }
  interface Props { open: boolean; onClose: () => void; actions: PaletteAction[]; }
  let { open, onClose, actions }: Props = $props();

  let query = $state('');
  let selectedIndex = $state(0);
  let announcement = $state('');
  let inputEl = $state<HTMLInputElement>();

  const filtered = $derived.by(() => {
    const q = query.toLowerCase().trim();
    const list = q
      ? actions.filter(a => a.label.toLowerCase().includes(q) || a.description?.toLowerCase().includes(q))
      : actions.slice(0, 10);
    return list.slice(0, 10);
  });

  $effect(() => {
    // reset selection when filtered list changes
    selectedIndex = 0;
    if (open) announcement = `${filtered.length} result${filtered.length === 1 ? '' : 's'} available.`;
  });

  $effect(() => {
    if (open) {
      query = '';
      setTimeout(() => inputEl?.focus(), 50);
    }
  });

  function execute(item: PaletteAction) {
    item.action();
    onClose();
  }

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'ArrowDown') { e.preventDefault(); selectedIndex = Math.min(selectedIndex + 1, filtered.length - 1); }
    else if (e.key === 'ArrowUp') { e.preventDefault(); selectedIndex = Math.max(selectedIndex - 1, 0); }
    else if (e.key === 'Enter') { e.preventDefault(); if (filtered[selectedIndex]) execute(filtered[selectedIndex]); }
    else if (e.key === 'Escape') { e.preventDefault(); onClose(); }
  }
</script>

{#if open}
  <!-- svelte-ignore a11y_click_events_have_key_events -->
  <div class="cp-overlay" role="button" tabindex="-1" onclick={onClose}>
    <div
      class="cp-modal"
      role="dialog"
      aria-modal="true"
      aria-label="Command Palette"
      onclick={(e) => e.stopPropagation()}
      onkeydown={(e) => e.stopPropagation()}
    >
      <div class="sr-only" role="status" aria-live="polite">{announcement}</div>

      <div class="cp-search">
        <span class="cp-icon" aria-hidden="true">🔍</span>
        <input
          bind:this={inputEl}
          type="text"
          bind:value={query}
          onkeydown={handleKeydown}
          placeholder="Search commands or sessions..."
          class="cp-input"
          role="combobox"
          aria-autocomplete="list"
          aria-expanded="true"
          aria-haspopup="listbox"
          aria-controls="cp-results"
          aria-activedescendant={filtered.length > 0 ? `cp-item-${selectedIndex}` : undefined}
        />
        <kbd class="cp-kbd">ESC</kbd>
      </div>

      <div id="cp-results" class="cp-results" role="listbox">
        {#each filtered as item, i}
          <div
            id="cp-item-{i}"
            role="option"
            aria-selected={selectedIndex === i}
            class="cp-item {selectedIndex === i ? 'cp-item--active' : ''}"
            onclick={() => execute(item)}
            onmouseenter={() => selectedIndex = i}
          >
            <div class="cp-item-left">
              <span class="cp-item-icon">{item.icon}</span>
              <div class="cp-item-text">
                <span class="cp-item-label">{item.label}</span>
                {#if item.description}<span class="cp-item-desc">{item.description}</span>{/if}
              </div>
            </div>
            {#if item.shortcut}<kbd class="cp-shortcut {selectedIndex === i ? 'cp-shortcut--active' : ''}">{item.shortcut}</kbd>{/if}
          </div>
        {/each}
        {#if filtered.length === 0}
          <div class="cp-empty">No telemetry matches found.</div>
        {/if}
      </div>

      <div class="cp-footer">
        <div class="cp-footer-keys">
          <span><kbd>↑↓</kbd> Navigate</span>
          <span><kbd>↵</kbd> Execute</span>
        </div>
        <div>OBLIVRA-OS // COMMANDS_READY</div>
      </div>
    </div>
  </div>
{/if}

<style>
  .cp-overlay {
    position: fixed; inset: 0; background: rgba(0,0,0,0.8); z-index: 100;
    display: flex; align-items: flex-start; justify-content: center; padding-top: 15vh;
  }
  .cp-modal {
    width: 100%; max-width: 640px; background: #18202a; border: 1px solid #2a3a48;
    box-shadow: 0 24px 60px rgba(0,0,0,0.6); font-family: var(--font-mono);
  }
  .cp-search {
    display: flex; align-items: center; gap: 14px; padding: 14px 16px;
    border-bottom: 1px solid #1e3040;
  }
  .cp-icon { font-size: 16px; color: #607070; flex-shrink: 0; }
  .cp-input {
    flex: 1; background: transparent; border: none; outline: none;
    color: #fff; font-size: 15px; font-family: inherit;
  }
  .cp-kbd { font-size: 10px; background: #0d1a1f; padding: 2px 7px; border: 1px solid #1e3040; color: #607070; }
  .cp-results { max-height: 60vh; overflow-y: auto; padding: 6px; }
  .cp-item {
    display: flex; align-items: center; justify-content: space-between;
    padding: 10px 12px; cursor: pointer; transition: background 80ms ease;
    color: #9b9ea4;
  }
  .cp-item--active { background: #b91c1c; color: #fff; }
  .cp-item:not(.cp-item--active):hover { background: #1e3040; }
  .cp-item-left { display: flex; align-items: center; gap: 12px; }
  .cp-item-icon { font-size: 16px; opacity: 0.8; }
  .cp-item-text { display: flex; flex-direction: column; gap: 2px; }
  .cp-item-label { font-weight: 700; font-size: 12px; text-transform: uppercase; letter-spacing: .04em; }
  .cp-item--active .cp-item-label { color: #fff; }
  .cp-item-desc  { font-size: 10px; text-transform: uppercase; letter-spacing: .12em; opacity: 0.6; }
  .cp-shortcut { font-size: 10px; padding: 2px 6px; border: 1px solid #1e3040; background: #0d1a1f; color: #607070; }
  .cp-shortcut--active { border-color: #f87171; background: #7f1d1d; color: #fff; }
  .cp-empty { padding: 32px; text-align: center; color: #607070; font-size: 11px; text-transform: uppercase; letter-spacing: .2em; }
  .cp-footer {
    display: flex; justify-content: space-between; align-items: center;
    padding: 10px 14px; border-top: 1px solid #1e3040; background: rgba(0,0,0,0.2);
    font-size: 9px; text-transform: uppercase; letter-spacing: .12em; color: #607070;
  }
  .cp-footer-keys { display: flex; gap: 14px; }
  .cp-footer-keys kbd { background: #0d1a1f; padding: 1px 4px; border: 1px solid #1e3040; color: #9b9ea4; }
  .sr-only { position: absolute; width: 1px; height: 1px; overflow: hidden; clip: rect(0,0,0,0); }
</style>
