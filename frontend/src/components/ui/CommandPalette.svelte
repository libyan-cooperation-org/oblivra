<!--
  OBLIVRA — CommandPalette (Svelte 5)
  Global command palette (Cmd+K / Cmd+P) for navigation and host discovery.
-->
<script lang="ts">
  import { fade, fly } from 'svelte/transition';
  import { appStore } from '@lib/stores/app.svelte';

  interface Props {
    open: boolean;
    onClose: () => void;
  }

  let { open, onClose }: Props = $props();

  let query = $state('');
  let selectedIndex = $state(0);
  let inputRef = $state<HTMLInputElement>();

  const results = $derived.by(() => {
    const q = query.toLowerCase().trim();
    if (!q) return [];

    const matches: { id: string; label: string; sublabel: string; type: 'route' | 'host' | 'action' }[] = [];

    // Search Routes (simplified for migration)
    const routes = [
      { id: 'dashboard', label: 'Dashboard', sublabel: 'Main Overview' },
      { id: 'siem', label: 'SIEM', sublabel: 'Security Information & Events' },
      { id: 'terminal', label: 'Terminal', sublabel: 'Active Shell Sessions' },
      { id: 'hosts', label: 'Hosts', sublabel: 'Infrastructure Management' },
      { id: 'vault', label: 'Vault', sublabel: 'Credential Manager' }
    ];

    for (const r of routes) {
      if (r.label.toLowerCase().includes(q)) {
        matches.push({ ...r, type: 'route' });
      }
    }

    // Search Hosts
    for (const h of appStore.hosts) {
      if (h.label.toLowerCase().includes(q) || h.hostname.toLowerCase().includes(q)) {
        matches.push({ id: h.id, label: h.label, sublabel: h.hostname, type: 'host' });
      }
    }

    return matches.slice(0, 10);
  });

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      onClose();
    } else if (e.key === 'ArrowDown') {
      e.preventDefault();
      selectedIndex = (selectedIndex + 1) % results.length;
    } else if (e.key === 'ArrowUp') {
      e.preventDefault();
      selectedIndex = (selectedIndex - 1 + results.length) % results.length;
    } else if (e.key === 'Enter') {
      e.preventDefault();
      if (results[selectedIndex]) {
        execute(results[selectedIndex]);
      }
    }
  }

  function execute(item: any) {
    if (item.type === 'route') {
      appStore.setActiveNavTab(item.id);
      window.location.hash = `#/${item.id}`;
    } else if (item.type === 'host') {
      appStore.connectToHost(item.id);
    }
    onClose();
    query = '';
  }

  $effect(() => {
    if (open) {
      query = '';
      selectedIndex = 0;
      setTimeout(() => inputRef?.focus(), 10);
    }
  });
</script>

{#if open}
  <div
    class="fixed inset-0 z-[10000] flex items-start justify-center pt-[15vh] px-4"
    transition:fade={{ duration: 150 }}
    role="button"
    tabindex="-1"
    onclick={onClose}
    onkeydown={(e) => e.key === 'Escape' && onClose()}
  >
    <div class="fixed inset-0 bg-black/60 backdrop-blur-sm"></div>

    <!-- Palette -->
    <div
      class="relative w-full max-w-2xl bg-surface-2 border border-border-secondary rounded-xl shadow-2xl overflow-hidden"
      transition:fly={{ y: -20, duration: 200 }}
      role="dialog"
      aria-modal="true"
      tabindex="-1"
      onclick={(e) => e.stopPropagation()}
      onkeydown={(e) => e.stopPropagation()}
    >
      <!-- Input Area -->
      <div class="flex items-center gap-3 px-4 py-3 border-b border-border-primary bg-surface-1" role="combobox" aria-controls="palette-results" aria-haspopup="listbox" aria-expanded={results.length > 0}>
        <span class="text-accent text-lg font-bold" aria-hidden="true">⌕</span>
        <input
          bind:this={inputRef}
          type="text"
          bind:value={query}
          onkeydown={handleKeydown}
          placeholder="Search commands, hosts, or navigate..."
          class="flex-1 bg-transparent border-none outline-none text-text-primary text-sm font-[var(--font-ui)] placeholder:text-text-muted"
          role="searchbox"
          aria-autocomplete="list"
          aria-controls="palette-results"
          aria-activedescendant={results.length > 0 ? `palette-item-${selectedIndex}` : undefined}
        />
        <div class="flex items-center gap-1" aria-hidden="true">
          <span class="text-[9px] font-mono text-text-muted bg-surface-3 px-1.5 py-0.5 rounded border border-border-primary">ESC</span>
        </div>
      </div>

      <!-- Results Area -->
      <div class="max-h-[400px] overflow-auto py-2" id="palette-results" role="listbox">
        {#if results.length > 0}
          {#each results as item, i}
            <button
              id={`palette-item-${i}`}
              role="option"
              aria-selected={i === selectedIndex}
              class="w-full flex items-center justify-between px-4 py-2.5 text-left transition-colors duration-fast outline-hidden
                {i === selectedIndex ? 'bg-accent/10 border-l-2 border-accent' : 'hover:bg-surface-3 border-l-2 border-transparent'}"
              onclick={() => execute(item)}
              onmouseenter={() => selectedIndex = i}
            >
              <div class="flex flex-col">
                <span class="text-xs font-bold {i === selectedIndex ? 'text-accent' : 'text-text-heading'}">
                  {item.label}
                </span>
                <span class="text-[10px] text-text-muted font-mono">{item.sublabel}</span>
              </div>
              <div class="flex items-center gap-2">
                <span class="text-[9px] font-mono font-bold uppercase tracking-wider text-text-muted opacity-50 px-1.5 py-0.5 border border-border-primary rounded bg-surface-2" aria-label="Type: {item.type}">
                  {item.type}
                </span>
              </div>
            </button>
          {/each}
        {:else if query}
          <div class="px-6 py-8 text-center">
            <div class="text-text-muted text-xs">No matches found for "{query}"</div>
          </div>
        {:else}
          <div class="px-6 py-6 text-center">
            <div class="text-[10px] font-bold uppercase tracking-widest text-text-muted opacity-40 mb-2">QUICK NAVIGATION</div>
            <div class="grid grid-cols-2 gap-2 text-left">
              <div class="text-[10px] text-text-muted flex items-center gap-2"><span class="text-accent">→</span> Dashboard</div>
              <div class="text-[10px] text-text-muted flex items-center gap-2"><span class="text-accent">→</span> Active Shells</div>
              <div class="text-[10px] text-text-muted flex items-center gap-2"><span class="text-accent">→</span> SIEM Feed</div>
              <div class="text-[10px] text-text-muted flex items-center gap-2"><span class="text-accent">→</span> Global Search</div>
            </div>
          </div>
        {/if}
      </div>

      <!-- Footer -->
      <div class="px-4 py-2 border-t border-border-primary bg-surface-1 flex items-center justify-between text-[9px] font-mono text-text-muted opacity-50">
        <div class="flex items-center gap-3">
          <span><span class="text-text-secondary">↑↓</span> to navigate</span>
          <span><span class="text-text-secondary">↵</span> to select</span>
        </div>
        <div>v0.1</div>
      </div>
    </div>
  </div>
{/if}
