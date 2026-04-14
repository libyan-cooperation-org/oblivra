<!--
  OBLIVRA — XTerm Wrapper (Svelte 5)
  Low-level xterm.js instance with Wails PTY bridging.
-->
<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { Terminal } from '@xterm/xterm';
  import { FitAddon } from '@xterm/addon-fit';
  import { WebLinksAddon } from '@xterm/addon-web-links';
  import { WebglAddon } from '@xterm/addon-webgl';
  import { subscribe } from '@lib/bridge';
  import '@xterm/xterm/css/xterm.css';

  interface Props {
    sessionId: string;
    isActive?: boolean;
  }

  let { sessionId, isActive = true }: Props = $props();

  let terminalContainer = $state<HTMLDivElement>();
  let term: Terminal;
  let fitAddon: FitAddon;
  let unsubscribes: (() => void)[] = [];

  // Theme constants to match OBLIVRA surface-0 / surface-1
  const termTheme = {
    background: '#0a0b10',
    foreground: '#a9b1d6',
    cursor: '#c0caf5',
    selectionBackground: '#33467c',
    black: '#15161e',
    red: '#f7768e',
    green: '#9ece6a',
    yellow: '#e0af68',
    blue: '#7aa2f7',
    magenta: '#bb9af7',
    cyan: '#7dcfff',
    white: '#a9b1d6',
    brightBlack: '#414868',
    brightRed: '#f7768e',
    brightGreen: '#9ece6a',
    brightYellow: '#e0af68',
    brightBlue: '#7aa2f7',
    brightMagenta: '#bb9af7',
    brightCyan: '#7dcfff',
    brightWhite: '#c0caf5',
  };

  onMount(async () => {
    if (!terminalContainer) return;

    term = new Terminal({
      cursorBlink: true,
      fontSize: 13,
      fontFamily: 'JetBrains Mono, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace',
      theme: termTheme,
      allowProposedApi: true,
      scrollback: 5000,
    });

    fitAddon = new FitAddon();
    term.loadAddon(fitAddon);
    term.loadAddon(new WebLinksAddon());

    try {
      const webgl = new WebglAddon();
      term.loadAddon(webgl);
    } catch (e) {
      console.warn('WebGL addon failed to load', e);
    }

    term.open(terminalContainer);
    fitAddon.fit();

    // ── Incoming Data (Backend -> Frontend)
    const unsubOut = subscribe(`terminal:out:${sessionId}`, (data: string) => {
      term.write(data);
    });
    unsubscribes.push(unsubOut);

    // ── Outgoing Data (Frontend -> Backend)
    // Detect session type by prefix: 'local-' sessions use LocalService, rest use SSHService
    const isLocal = sessionId.startsWith('local-');

    term.onData(async (data) => {
      try {
        if (isLocal) {
          const { SendInput } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/localservice');
          await SendInput(sessionId, data);
        } else {
          const { SendInput } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/sshservice');
          await SendInput(sessionId, data);
        }
      } catch (e) {
        console.warn('[xterm] SendInput failed:', e);
      }
    });

    // ── Resize handling
    term.onResize(async ({ cols, rows }) => {
      try {
        if (isLocal) {
          const { Resize } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/localservice');
          await Resize(sessionId, cols, rows);
        } else {
          const { Resize } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/sshservice');
          await Resize(sessionId, cols, rows);
        }
      } catch (e) {
        console.warn('[xterm] Resize failed:', e);
      }
    });

    window.addEventListener('resize', handleResize);

    // Initial resize sync
    setTimeout(() => handleResize(), 100);
  });

  function handleResize() {
    if (fitAddon) {
      fitAddon.fit();
    }
  }

  $effect(() => {
    if (isActive) {
      setTimeout(() => handleResize(), 50);
      term?.focus();
    }
  });

  onDestroy(() => {
    unsubscribes.forEach(u => u());
    window.removeEventListener('resize', handleResize);
    term?.dispose();
  });
</script>

<div 
  bind:this={terminalContainer} 
  class="w-full h-full bg-[#0a0b10] p-2 overflow-hidden"
  class:hidden={!isActive}
></div>

<style>
  :global(.xterm-viewport) {
    background-color: transparent !important;
  }
  :global(.xterm-screen) {
    padding-left: 2px;
  }
</style>
