<script lang="ts">
  // Command palette — Ctrl/Cmd+K. The single most-used keystroke in
  // every elite SaaS console (Linear, Vercel, GitHub, Datadog, …).
  //
  // Two kinds of entries:
  //   1. Navigation — every NAV item registered in lib/nav.ts.
  //   2. Actions — global commands (refresh, copy fingerprint, theme
  //      toggle, open shortcuts overlay, etc.).
  //
  // Filtering is a simple lower-cased substring + word-prefix match —
  // good enough for ~50 items. We don't need fuzzy weighting for that
  // size; for huge command lists swap in a tokenizer later.

  import { NAV, type NavId, type NavItem } from '../nav';
  import { copy } from '../clipboard';

  let { open = $bindable(false), active = $bindable<NavId>('overview') }: {
    open: boolean;
    active: NavId;
  } = $props();

  type Action = {
    kind: 'nav';
    id: string;
    label: string;
    hint?: string;
    icon: string;
    nav: NavItem;
  } | {
    kind: 'cmd';
    id: string;
    label: string;
    hint?: string;
    icon: string;
    run: () => void | Promise<void>;
  };

  const actions: Action[] = $derived.by(() => {
    const navActions: Action[] = NAV.map((n) => ({
      kind: 'nav' as const,
      id: 'nav:' + n.id,
      label: 'Go to ' + n.label,
      hint: n.hint,
      icon: n.icon,
      nav: n,
    }));
    const cmdActions: Action[] = [
      {
        kind: 'cmd', id: 'cmd:reload', label: 'Reload window', icon: '↻',
        hint: 'Hard refresh the dashboard', run: () => location.reload(),
      },
      {
        kind: 'cmd', id: 'cmd:copy-url', label: 'Copy current URL', icon: '⎘',
        hint: 'Includes view + filters', run: () => copy(location.href, 'URL copied'),
      },
      {
        kind: 'cmd', id: 'cmd:print', label: 'Print this view', icon: '⎙',
        hint: 'For evidence handoff', run: () => window.print(),
      },
      {
        kind: 'cmd', id: 'cmd:shortcuts', label: 'Show keyboard shortcuts', icon: '?',
        hint: 'Press ?', run: () => {
          window.dispatchEvent(new CustomEvent('oblivra:shortcuts'));
        },
      },
    ];
    return [...navActions, ...cmdActions];
  });

  let query = $state('');
  let cursor = $state(0);

  const filtered = $derived.by(() => {
    const q = query.trim().toLowerCase();
    if (!q) return actions;
    return actions.filter((a) => {
      const hay = (a.label + ' ' + (a.hint ?? '')).toLowerCase();
      // every whitespace-separated token must appear somewhere
      return q.split(/\s+/).every((t) => hay.includes(t));
    });
  });

  $effect(() => {
    if (cursor >= filtered.length) cursor = 0;
  });

  let inputEl: HTMLInputElement | null = $state(null);

  $effect(() => {
    if (open) {
      query = '';
      cursor = 0;
      // focus on next paint — element is :focus-visible across browsers.
      queueMicrotask(() => inputEl?.focus());
    }
  });

  function run(a: Action) {
    if (a.kind === 'nav') {
      active = a.nav.id;
    } else {
      void a.run();
    }
    open = false;
  }

  function onKey(e: KeyboardEvent) {
    if (!open) return;
    if (e.key === 'Escape') {
      e.preventDefault();
      open = false;
      return;
    }
    if (e.key === 'ArrowDown') {
      e.preventDefault();
      cursor = (cursor + 1) % Math.max(1, filtered.length);
      return;
    }
    if (e.key === 'ArrowUp') {
      e.preventDefault();
      cursor = (cursor - 1 + filtered.length) % Math.max(1, filtered.length);
      return;
    }
    if (e.key === 'Enter') {
      e.preventDefault();
      const a = filtered[cursor];
      if (a) run(a);
      return;
    }
  }
</script>

<svelte:window on:keydown={onKey} />

{#if open}
  <!-- backdrop -->
  <div
    class="fixed inset-0 z-50 grid place-items-start justify-center bg-night-950/70 pt-[14vh] backdrop-blur"
    role="dialog"
    aria-modal="true"
    aria-label="Command palette"
    tabindex="-1"
    onclick={(e) => {
      if (e.target === e.currentTarget) open = false;
    }}
    onkeydown={(e) => {
      if (e.target === e.currentTarget && e.key === 'Escape') open = false;
    }}
  >
    <div class="w-full max-w-2xl overflow-hidden rounded-xl border border-night-600 bg-night-900 shadow-2xl shadow-black/60">
      <div class="flex items-center gap-3 border-b border-night-700 px-4 py-3">
        <span class="text-sm text-night-300">⌘</span>
        <input
          bind:this={inputEl}
          bind:value={query}
          type="text"
          placeholder="Jump to view, run a command…"
          class="flex-1 bg-transparent text-sm text-slate-100 placeholder:text-night-300 focus:outline-none"
          autocomplete="off"
          spellcheck="false"
        />
        <kbd class="rounded border border-night-600 bg-night-800 px-1.5 py-0.5 text-[10px] text-night-300">esc</kbd>
      </div>
      <ul class="max-h-[55vh] overflow-y-auto scrollbar-thin py-2">
        {#each filtered as a, i (a.id)}
          <li>
            <button
              type="button"
              onmouseenter={() => (cursor = i)}
              onclick={() => run(a)}
              class="flex w-full items-center gap-3 px-4 py-2 text-left text-sm transition"
              class:bg-accent-500={cursor === i}
              class:text-white={cursor === i}
              class:text-night-100={cursor !== i}
            >
              <span class="grid h-7 w-7 place-items-center rounded bg-night-800 text-base">
                {a.icon}
              </span>
              <span class="flex flex-col leading-tight">
                <span>{a.label}</span>
                {#if a.hint}
                  <span class="text-[11px] opacity-70">{a.hint}</span>
                {/if}
              </span>
              <span class="ml-auto text-[10px] uppercase tracking-widest opacity-60">
                {a.kind === 'nav' ? 'jump' : 'run'}
              </span>
            </button>
          </li>
        {:else}
          <li class="px-4 py-6 text-center text-sm text-night-300">No matches</li>
        {/each}
      </ul>
      <div class="flex items-center justify-between border-t border-night-700 bg-night-800/60 px-4 py-2 text-[11px] text-night-300">
        <div class="flex items-center gap-3">
          <span><kbd class="rounded border border-night-600 bg-night-800 px-1">↑</kbd> <kbd class="rounded border border-night-600 bg-night-800 px-1">↓</kbd> navigate</span>
          <span><kbd class="rounded border border-night-600 bg-night-800 px-1">↵</kbd> select</span>
          <span><kbd class="rounded border border-night-600 bg-night-800 px-1">esc</kbd> close</span>
        </div>
        <span>{filtered.length} item{filtered.length === 1 ? '' : 's'}</span>
      </div>
    </div>
  </div>
{/if}
