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
  import { toastStore } from '@lib/stores/toast.svelte';
  import '@xterm/xterm/css/xterm.css';

  // Audit fix M-12: surface clipboard permission denials once per
  // mount so operators understand why select-to-copy isn't working
  // (typically: browser denied clipboard permission, or running over
  // non-localhost HTTP where the API is gated).
  let clipboardWarned = false;
  function warnClipboardOnce(label: string, err: unknown) {
    if (clipboardWarned) return;
    clipboardWarned = true;
    const msg = err instanceof Error ? err.message : String(err);
    toastStore.add({
      type: 'warning',
      title: 'Clipboard access denied',
      message: `${label}: ${msg}. Use Ctrl+C / Ctrl+Shift+V manually, or grant clipboard permission in browser settings.`,
    });
  }

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

    // ── Phase 23.5 — Clipboard (OSC 52 + native fallback) ───────────────
    // OSC 52 is a terminal escape sequence that lets remote programs (vim,
    // tmux, etc.) push text into the OS clipboard. xterm.js fires the
    // event but doesn't write to the clipboard itself — we wire that up
    // here. Pair with auto-copy-on-selection and right-click paste so the
    // operator gets the same affordances as Termius / Windows Terminal.
    term.parser.registerOscHandler(52, (data) => {
      // OSC 52 payload: "<targets>;<base64>". We accept any target (c=clipboard,
      // p=primary, etc.) and route everything to the OS clipboard since the
      // distinction doesn't translate cleanly to most desktop OSes.
      const semi = data.indexOf(';');
      if (semi < 0) return false;
      const b64 = data.slice(semi + 1);
      try {
        const text = atob(b64);
        if (navigator.clipboard?.writeText) {
          // Audit fix M-12: surface clipboard permission denials
          // instead of silently dropping them — operators trying to
          // copy from a remote shell deserve to know why nothing
          // happened. Toast once per session to avoid spam.
          navigator.clipboard.writeText(text).catch((err) => {
            warnClipboardOnce('OSC 52 copy denied', err);
          });
        }
      } catch {
        // Malformed base64; ignore.
      }
      return true;
    });

    // Auto-copy on selection — most SOC operators expect "select = copied"
    // rather than the xterm default which requires a separate ⌃C.
    term.onSelectionChange(() => {
      const sel = term.getSelection();
      if (sel && navigator.clipboard?.writeText) {
        navigator.clipboard.writeText(sel).catch((err) => {
          warnClipboardOnce('selection-copy denied', err);
        });
      }
    });

    // Right-click paste — reads from the OS clipboard and writes through the
    // same SendInput path as keystrokes, so the remote shell sees a normal
    // paste stream.
    terminalContainer.addEventListener('contextmenu', async (ev) => {
      ev.preventDefault();
      try {
        const text = await navigator.clipboard?.readText?.();
        if (!text) return;
        if (sessionId.startsWith('local-')) {
          const { SendInput } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/localservice');
          await SendInput(sessionId, text);
        } else {
          const { SendInput } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/sshservice');
          await SendInput(sessionId, text);
        }
      } catch (e) {
        console.warn('[xterm] paste failed:', e);
      }
    });

    // ── Incoming Data (Backend -> Frontend)
    const unsubOut = subscribe(`terminal:out:${sessionId}`, (data: string) => {
      try {
        // Backend emits base64 encoded chunks to safely transport control characters/binary
        const decoded = atob(data);
        term.write(decoded);
      } catch (e) {
        console.warn('[xterm] failed to decode terminal output:', e);
        // Fallback: if it's not base64, just write it raw (legacy/direct paths)
        term.write(data);
      }
    });
    unsubscribes.push(unsubOut);

    // ── Outgoing Data (Frontend -> Backend)
    // Detect session type by prefix: 'local-' sessions use LocalService, rest use SSHService
    const isLocal = sessionId.startsWith('local-');

    term.onData(async (data) => {
      try {
        // Backend expects base64 encoded input to safely handle control characters
        // We use the btoa(unescape(encodeURIComponent(s))) pattern to ensure 
        // safe binary-to-string conversion even for UTF-8/extended chars.
        const encoded = btoa(unescape(encodeURIComponent(data)));
        
        if (isLocal) {
          const { SendInput } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/localservice');
          await SendInput(sessionId, encoded);
        } else {
          const { SendInput } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/sshservice');
          await SendInput(sessionId, encoded);
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
    if (isActive && term) {
      // Ensure terminal is focused and resized when it becomes the active view
      setTimeout(() => {
        handleResize();
        term.focus();
      }, 50);
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
