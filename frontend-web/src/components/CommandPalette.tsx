import { createSignal, createMemo, For, Show, createEffect } from 'solid-js';

export interface PaletteAction {
  id: string;
  label: string;
  description?: string;
  icon: string;
  shortcut?: string;
  action: () => void;
}

interface CommandPaletteProps {
  open: boolean;
  onClose: () => void;
  actions: PaletteAction[];
}

export default function CommandPalette(props: CommandPaletteProps) {
  const [query, setQuery] = createSignal('');
  const [selectedIndex, setSelectedIndex] = createSignal(0);
  const [announcement, setAnnouncement] = createSignal('');
  let inputRef: HTMLInputElement | undefined;

  const filtered = createMemo(() => {
    const q = query().toLowerCase().trim();
    if (!q) return props.actions.slice(0, 10);
    return props.actions
      .filter(a => a.label.toLowerCase().includes(q) || a.description?.toLowerCase().includes(q))
      .slice(0, 10);
  });

  createEffect(() => {
    // Reset selection when results change
    filtered();
    setSelectedIndex(0);
    
    // Accessibility: Announce result count to screen readers
    const visibleCount = filtered().length;
    if (props.open) {
      setAnnouncement(`${visibleCount} result${visibleCount === 1 ? '' : 's'} available.`);
    }
  });

  createEffect(() => {
    if (props.open) {
      setTimeout(() => inputRef?.focus(), 50);
    }
  });

  const handleKeyDown = (e: KeyboardEvent) => {
    switch (e.key) {
      case 'ArrowDown':
        e.preventDefault();
        setSelectedIndex((i) => Math.min(i + 1, filtered().length - 1));
        const nextItem = filtered()[selectedIndex()];
        if (nextItem) setAnnouncement(`Selected ${nextItem.label}`);
        break;
      case 'ArrowUp':
        e.preventDefault();
        setSelectedIndex((i) => Math.max(i - 1, 0));
        const prevItem = filtered()[selectedIndex()];
        if (prevItem) setAnnouncement(`Selected ${prevItem.label}`);
        break;
      case 'Enter':
        e.preventDefault();
        const selected = filtered()[selectedIndex()];
        if (selected) {
          selected.action();
          props.onClose();
        }
        break;
      case 'Escape':
        e.preventDefault();
        props.onClose();
        break;
    }
  };

  return (
    <Show when={props.open}>
      <div 
        class="fixed inset-0 bg-black/80 z-[100] flex items-start justify-center pt-[15vh] p-4 animate-in fade-in duration-200"
        onClick={props.onClose}
      >
        <div 
          class="w-full max-w-2xl bg-zinc-900 border border-zinc-700 shadow-2xl font-mono"
          onClick={(e) => e.stopPropagation()}
          role="dialog"
          aria-modal="true"
          aria-label="Command Palette"
        >
          {/* ARIA Live Region for dynamic announcements */}
          <div class="sr-only" role="status" aria-live="polite">
            {announcement()}
          </div>

          <div class="p-4 border-b border-zinc-800 flex items-center gap-4">
            <span class="text-zinc-500 text-lg">🔍</span>
            <input
              ref={inputRef}
              type="text"
              class="w-full bg-transparent border-none focus:outline-none text-white text-lg"
              placeholder="Search commands or sessions..."
              value={query()}
              onInput={(e) => setQuery(e.currentTarget.value)}
              onKeyDown={handleKeyDown}
              role="combobox"
              aria-autocomplete="list"
              aria-expanded="true"
              aria-haspopup="listbox"
              aria-controls="palette-results"
              aria-activedescendant={`palette-item-${selectedIndex()}`}
            />
            <kbd class="text-[10px] bg-zinc-800 px-1.5 py-0.5 border border-zinc-700 text-zinc-500 rounded">ESC</kbd>
          </div>

          <div 
            id="palette-results"
            class="max-h-[60vh] overflow-y-auto p-2"
            role="listbox"
          >
            <For each={filtered()}>
              {(item, index) => (
                <div
                  id={`palette-item-${index()}`}
                  role="option"
                  aria-selected={selectedIndex() === index()}
                  class={`p-3 flex items-center justify-between cursor-pointer transition-colors ${
                    selectedIndex() === index() ? 'bg-red-600 text-white' : 'hover:bg-zinc-800 text-zinc-400'
                  }`}
                  onClick={() => {
                    item.action();
                    props.onClose();
                  }}
                >
                  <div class="flex items-center gap-3">
                    <span class="text-lg opacity-80">{item.icon}</span>
                    <div class="flex flex-col">
                      <span class={`font-bold uppercase tracking-tight ${selectedIndex() === index() ? 'text-white' : 'text-zinc-200'}`}>
                        {item.label}
                      </span>
                      {item.description && (
                        <span class="text-[10px] uppercase opacity-60 tracking-widest">{item.description}</span>
                      )}
                    </div>
                  </div>
                  {item.shortcut && (
                    <kbd class={`text-[10px] px-1 py-0.5 rounded border ${
                      selectedIndex() === index() ? 'border-red-400 bg-red-700' : 'border-zinc-700 bg-zinc-800'
                    }`}>
                      {item.shortcut}
                    </kbd>
                  )}
                </div>
              )}
            </For>
            <Show when={filtered().length === 0}>
              <div class="p-8 text-center text-zinc-600 uppercase text-xs tracking-widest">
                No telemetry matches found.
              </div>
            </Show>
          </div>

          <div class="p-3 border-t border-zinc-800 bg-black/20 flex justify-between items-center text-[9px] uppercase tracking-widest text-zinc-600">
             <div class="flex gap-4">
               <span><kbd class="bg-zinc-800 p-0.5 border border-zinc-700 text-zinc-400">↑↓</kbd> Navigate</span>
               <span><kbd class="bg-zinc-800 p-0.5 border border-zinc-700 text-zinc-400">↵</kbd> Execute</span>
             </div>
             <div>OBLIVRA-OS // COMMANDS_READY</div>
          </div>
        </div>
      </div>
    </Show>
  );
}
