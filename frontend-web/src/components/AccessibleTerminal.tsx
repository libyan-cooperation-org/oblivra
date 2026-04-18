import { createSignal, onMount, For, Show } from 'solid-js';

interface TerminalLine {
  id: string;
  content: string;
  type: 'command' | 'output' | 'error';
}

export default function AccessibleTerminal() {
  const [lines, setLines] = createSignal<TerminalLine[]>([]);
  const [lastAnnouncement, setLastAnnouncement] = createSignal('');

  onMount(() => {
    // Initial banner
    const initialLines: TerminalLine[] = [
      { id: '1', content: 'OBLIVRA-OS v0.4.1 // SECURE_SHELL_INITIALIZED', type: 'output' },
      { id: '2', content: 'substrate login --tenant GLOBAL_CORP', type: 'command' },
      { id: '3', content: 'Authentication successful. Access granted to 42 nodes.', type: 'output' },
    ];
    setLines(initialLines);
  });

  const addLine = (content: string, type: 'command' | 'output' | 'error') => {
    const newLine: TerminalLine = { id: Math.random().toString(36).substr(2, 9), content, type };
    setLines([...lines(), newLine]);
    setLastAnnouncement(`${type.toUpperCase()}: ${content}`);
  };

  return (
    <section class="space-y-4 font-mono">
      <div class="flex justify-between items-center border-b border-[var(--border-bold)] pb-2">
        <h3 class="text-xs font-black uppercase tracking-[0.2em] text-[var(--accent-primary)]">
          Sovereign Control Substrate // ARIA_AUDIT_MODE
        </h3>
        <span class="text-[9px] text-[var(--text-muted)] uppercase tracking-widest">
          TTY: /dev/pts/0 // SHELL: oblivra-sh
        </span>
      </div>

      <div 
        class="bg-black/80 border border-zinc-800 p-6 h-64 overflow-y-auto space-y-2 flex flex-col justify-end"
        role="log"
        aria-label="Accessible Terminal Output"
        aria-live="polite"
      >
        <For each={lines()}>
          {(line) => (
            <div class={`text-xs leading-relaxed ${
              line.type === 'command' ? 'text-[var(--accent-primary)]' : 
              line.type === 'error' ? 'text-red-500 font-bold' : 'text-zinc-300'
            }`}>
              <Show when={line.type === 'command'}>
                <span class="text-zinc-600 mr-2">❯</span>
              </Show>
              {line.content}
            </div>
          )}
        </For>
        
        {/* Cursor emulator */}
        <div class="flex items-center gap-2 text-xs">
          <span class="text-zinc-600">❯</span>
          <div class="w-2 h-4 bg-[var(--accent-primary)] animate-pulse"></div>
        </div>
      </div>

      {/* Screen Reader Only Announcement for new output */}
      <div class="sr-only" role="status" aria-live="polite">
        {lastAnnouncement()}
      </div>

      <div class="flex gap-4">
        <button 
          onClick={() => addLine('oblivra-scan --target intranet', 'command')}
          class="px-3 py-1 bg-zinc-800 border border-zinc-700 text-[10px] font-bold uppercase text-zinc-400 hover:text-[var(--accent-primary)] transition-all"
        >
          Run Scan
        </button>
        <button 
          onClick={() => addLine('Scan completed. No threats found in /intranet/', 'output')}
          class="px-3 py-1 bg-zinc-800 border border-zinc-700 text-[10px] font-bold uppercase text-zinc-400 hover:text-green-500 transition-all"
        >
          Mock Output
        </button>
      </div>
    </section>
  );
}
