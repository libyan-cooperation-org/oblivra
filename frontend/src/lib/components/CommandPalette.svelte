<script lang="ts">
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
      label: n.label,
      hint: n.hint,
      icon: n.icon,
      nav: n,
    }));
    const cmdActions: Action[] = [
      { kind: 'cmd', id: 'cmd:reload',    label: 'Reload window',         icon: '↻', hint: 'Hard refresh the dashboard', run: () => location.reload() },
      { kind: 'cmd', id: 'cmd:copy-url',  label: 'Copy current URL',      icon: '⎘', hint: 'Includes view + filters',    run: () => copy(location.href, 'URL copied') },
      { kind: 'cmd', id: 'cmd:print',     label: 'Print this view',       icon: '⎙', hint: 'For evidence handoff',       run: () => window.print() },
      { kind: 'cmd', id: 'cmd:shortcuts', label: 'Show keyboard shortcuts',icon: '?', hint: 'Press ?',                   run: () => window.dispatchEvent(new CustomEvent('oblivra:shortcuts')) },
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
      return q.split(/\s+/).every((t) => hay.includes(t));
    });
  });

  $effect(() => { if (cursor >= filtered.length) cursor = 0; });

  let inputEl: HTMLInputElement | null = $state(null);

  $effect(() => {
    if (open) {
      query = '';
      cursor = 0;
      queueMicrotask(() => inputEl?.focus());
    }
  });

  function run(a: Action) {
    if (a.kind === 'nav') active = a.nav.id;
    else void a.run();
    open = false;
  }

  function onKey(e: KeyboardEvent) {
    if (!open) return;
    if (e.key === 'Escape')   { e.preventDefault(); open = false; return; }
    if (e.key === 'ArrowDown'){ e.preventDefault(); cursor = (cursor + 1) % Math.max(1, filtered.length); return; }
    if (e.key === 'ArrowUp')  { e.preventDefault(); cursor = (cursor - 1 + filtered.length) % Math.max(1, filtered.length); return; }
    if (e.key === 'Enter')    { e.preventDefault(); const a = filtered[cursor]; if (a) run(a); return; }
  }
</script>

<svelte:window on:keydown={onKey} />

{#if open}
  <div
    class="fixed inset-0 z-50 grid place-items-start justify-center"
    style="background:rgba(6,10,15,0.85); backdrop-filter:blur(8px); padding-top:12vh;"
    role="dialog"
    aria-modal="true"
    aria-label="Command palette"
    tabindex="-1"
    onclick={(e) => { if (e.target === e.currentTarget) open = false; }}
    onkeydown={(e) => { if (e.target === e.currentTarget && e.key === 'Escape') open = false; }}
  >
    <div style="width:100%; max-width:600px; border:1px solid var(--color-cyan-600); background:var(--color-base-900); box-shadow:0 0 60px rgba(0,188,216,0.15), 0 30px 80px rgba(0,0,0,0.8); overflow:hidden;">
      <!-- Search input -->
      <div class="flex items-center gap-3 border-b border-base-700" style="padding:10px 16px;">
        <span style="font-family:'Share Tech Mono',monospace; font-size:14px; color:var(--color-cyan-500);">⌕</span>
        <input
          bind:this={inputEl}
          bind:value={query}
          type="text"
          placeholder="Jump to view, run a command…"
          style="
            flex:1;
            background:transparent;
            border:none;
            outline:none;
            font-family:'Share Tech Mono',monospace;
            font-size:13px;
            letter-spacing:0.5px;
            color:#e8f4f8;
          "
          autocomplete="off"
          spellcheck="false"
        />
        <kbd style="padding:2px 6px; border:1px solid var(--color-base-600); background:var(--color-base-800); font-family:'Share Tech Mono',monospace; font-size:9px; letter-spacing:1px; color:var(--color-base-300);">ESC</kbd>
      </div>

      <!-- Results -->
      <ul class="scrollbar-thin" style="max-height:56vh; overflow-y:auto; padding:4px 0;">
        {#each filtered as a, i (a.id)}
          <li>
            <button
              type="button"
              onmouseenter={() => (cursor = i)}
              onclick={() => run(a)}
              class="flex w-full items-center gap-3 text-left transition-all duration-75"
              style="
                padding: 8px 16px;
                background: {cursor === i ? 'rgba(0,188,216,0.12)' : 'transparent'};
                border-left: 2px solid {cursor === i ? 'var(--color-cyan-500)' : 'transparent'};
              "
            >
              <!-- Icon badge -->
              <span style="
                display:grid; place-items:center;
                width:28px; height:28px;
                background: {cursor === i ? 'rgba(0,188,216,0.15)' : 'var(--color-base-800)'};
                font-size:13px;
                color: {cursor === i ? 'var(--color-cyan-400)' : 'var(--color-base-200)'};
                flex-shrink:0;
              ">{a.icon}</span>

              <!-- Label + hint -->
              <span class="flex flex-col" style="line-height:1.2; min-width:0;">
                <span style="
                  font-family:'Rajdhani',sans-serif;
                  font-weight:600;
                  font-size:13px;
                  letter-spacing:1px;
                  text-transform:uppercase;
                  color: {cursor === i ? 'var(--color-cyan-400)' : '#e8f4f8'};
                ">{a.label}</span>
                {#if a.hint}
                  <span style="font-family:'Share Tech Mono',monospace; font-size:10px; color:var(--color-base-300); letter-spacing:0.3px;">{a.hint}</span>
                {/if}
              </span>

              <!-- Type badge -->
              <span style="
                margin-left:auto;
                flex-shrink:0;
                font-family:'Share Tech Mono',monospace;
                font-size:9px;
                letter-spacing:2px;
                color: {a.kind === 'nav' ? 'var(--color-cyan-600)' : 'var(--color-sig-warn)'};
                text-transform:uppercase;
              ">{a.kind === 'nav' ? 'NAV' : 'RUN'}</span>
            </button>
          </li>
        {:else}
          <li style="padding:24px 16px; text-align:center; font-family:'Share Tech Mono',monospace; font-size:11px; letter-spacing:1px; color:var(--color-base-300);">NO MATCHES</li>
        {/each}
      </ul>

      <!-- Footer -->
      <div class="flex items-center justify-between border-t border-base-700" style="padding:6px 16px; background:var(--color-base-850);">
        <div class="flex items-center gap-4" style="font-family:'Share Tech Mono',monospace; font-size:9px; letter-spacing:1px; color:var(--color-base-300);">
          <span>↑↓ NAVIGATE</span>
          <span>↵ SELECT</span>
          <span>ESC CLOSE</span>
        </div>
        <span style="font-family:'Share Tech Mono',monospace; font-size:9px; letter-spacing:1px; color:var(--color-base-400);">{filtered.length} RESULT{filtered.length === 1 ? '' : 'S'}</span>
      </div>
    </div>
  </div>
{/if}
