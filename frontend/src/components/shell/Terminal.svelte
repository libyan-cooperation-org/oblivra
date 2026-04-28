<!--
  OBLIVRA — Shell Terminal (Blacknode-style xterm.js wrapper)

  Ported and adapted from Blacknode (MIT). Differences vs upstream:

   1. Bindings target OBLIVRA's existing `LocalService` and `SSHService`
      Wails services, not Blacknode's `LocalShellService` /
      Blacknode-`SSHService`. Method names mapped:
        Blacknode               →  OBLIVRA
        LocalShellService.Open  →  LocalService.StartLocalSession (no args; returns sessionID)
        LocalShellService.Write →  LocalService.SendInput (base64 input)
        LocalShellService.Close →  LocalService.CloseSession
        SSHService.ConnectByHost(sid, hostID, password, cols, rows)
                                →  SSHService.Connect(hostID) returns sessionID
        SSHService.Write        →  SSHService.SendInput (base64 input)
        SSHService.Disconnect   →  (no direct equivalent; SSHService.CloseAll
                                   reaps all; per-session close is implicit
                                   when the underlying ssh.Session ends)

   2. Event delivery: OBLIVRA emits `terminal:out:{sessionID}` with a
      base64-encoded payload, not Blacknode's `terminal:data` with
      {sessionID, data}. We subscribe per-session and decode here.

   3. Latency: OBLIVRA's SSHService doesn't expose a Latency() RPC, so
      the latency badge is hidden until a future phase wires one up.

   4. Multi-cursor broadcast + AI-insert hooks come from `shellStore`
      (the new Shell-workspace store), not Blacknode's `app` store.

  The xterm.js options, theme palette, OSC 52 clipboard, auto-copy on
  selection, and right-click paste all match the Blacknode original
  (with OBLIVRA's surface-0 palette).
-->
<script lang="ts">
  import { onDestroy, onMount } from 'svelte';
  import { Terminal } from '@xterm/xterm';
  import { FitAddon } from '@xterm/addon-fit';
  import { WebLinksAddon } from '@xterm/addon-web-links';
  import { WebglAddon } from '@xterm/addon-webgl';
  import { subscribe } from '@lib/bridge';
  import { shellStore } from '@lib/stores/shell.svelte';
  import { toastStore } from '@lib/stores/toast.svelte';
  import { scanForIOCs, hasIOCs, type IOCMatch } from '@lib/iocMatcher';
  import {
    Terminal as TerminalIcon,
    Server,
    Plug,
    Unplug,
    Loader2,
    Lock,
    AlertTriangle,
    Circle,
    Radio,
  } from 'lucide-svelte';
  import '@xterm/xterm/css/xterm.css';

  type Props = {
    /** Optional pre-allocated session id; usually omitted — the local
     *  PTY backend assigns one and we adopt it. Set when a tab tear-out
     *  re-attaches to an existing PTY. */
    sessionID?: string;
    /** Leaf id from panes.ts. Tab id is read from shellStore.activeTabID
     *  at mount time — Pane.svelte doesn't know its own tab. */
    leafID?: string;
  };
  let { sessionID: incomingSessionID, leafID }: Props = $props();
  // Capture the tab id at mount; if the operator switches tabs after the
  // pane is mounted, leafMeta still belongs with this leaf's original tab.
  let tabID = $state<string | null>(null);

  type Mode = 'local' | 'remote';
  type Status = 'starting' | 'running' | 'connecting' | 'connected' | 'idle' | 'error';

  let mode: Mode = $state('local');
  // Start idle — operator decides whether to open a local shell or connect
  // to a saved host. Auto-spawning a PowerShell/bash made the empty state
  // feel like "the OS terminal" rather than a deliberate workspace.
  let status: Status = $state('idle');
  let errorMsg = $state('');
  let connectedHostID = $state<string | null>(null);
  let promptingPassword = $state(false);
  let runtimePassword = $state('');
  let showHostPicker = $state(false);

  // Real session id used at runtime — empty until either the prop seeds
  // it (rare: tab tear-out re-attach) or LocalService.StartLocalSession()
  // resolves and we adopt the backend-assigned id. Initialised in onMount
  // to keep Svelte 5's "no prop reads in $state init" rule happy.
  let sessionID = $state('');

  let containerEl: HTMLDivElement | undefined = $state();
  let term: Terminal | undefined;
  let fit: FitAddon | undefined;
  let dataOff: (() => void) | undefined;
  let resizeObs: ResizeObserver | undefined;

  // Clipboard permission warnings — toasted once per mount (matches OBLIVRA's
  // existing XTerm behaviour from audit fix M-12).
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

  // AI/palette → terminal write channel. Fires when a sibling component
  // wants to insert text into this specific session.
  $effect(() => {
    const p = shellStore.pendingTerminalInsert;
    if (!p || p.sessionID !== sessionID) return;
    if (mode === 'local' && status === 'running') void writeBackend(p.text);
    else if (mode === 'remote' && status === 'connected') void writeBackend(p.text);
    shellStore.pendingTerminalInsert = null;
  });

  function termTheme() {
    if (shellStore.theme === 'light') {
      return {
        background: '#ffffff',
        foreground: '#0a0e18',
        cursor: '#0891b2',
        cursorAccent: '#ffffff',
        selectionBackground: 'rgba(8, 145, 178, 0.20)',
        black: '#1f2533',
        brightBlack: '#525866',
        red: '#c53030',
        brightRed: '#9b1c1c',
        green: '#16a34a',
        brightGreen: '#15803d',
        yellow: '#b25800',
        brightYellow: '#92400e',
        blue: '#1d4ed8',
        brightBlue: '#1e3a8a',
        magenta: '#7e22ce',
        brightMagenta: '#581c87',
        cyan: '#0891b2',
        brightCyan: '#0e7490',
        white: '#7a8092',
        brightWhite: '#0a0e18',
      };
    }
    // OBLIVRA surface-0 palette to match the rest of the app chrome.
    return {
      background: '#0a0b10',
      foreground: '#a9b1d6',
      cursor: '#22d3ee',
      cursorAccent: '#0a0b10',
      selectionBackground: 'rgba(34, 211, 238, 0.25)',
      black: '#15161e',
      brightBlack: '#414868',
      red: '#f7768e',
      brightRed: '#f7768e',
      green: '#9ece6a',
      brightGreen: '#9ece6a',
      yellow: '#e0af68',
      brightYellow: '#e0af68',
      blue: '#7aa2f7',
      brightBlue: '#7aa2f7',
      magenta: '#bb9af7',
      brightMagenta: '#bb9af7',
      cyan: '#7dcfff',
      brightCyan: '#7dcfff',
      white: '#a9b1d6',
      brightWhite: '#c0caf5',
    };
  }

  // Wire keystrokes → backend. Both Local and SSH OBLIVRA services accept
  // base64 input (matches the legacy XTerm.svelte path), so we encode here.
  async function writeBackend(data: string) {
    if (!sessionID) return;
    const encoded = btoa(unescape(encodeURIComponent(data)));
    try {
      if (mode === 'local') {
        const { SendInput } = await import(
          '@wailsjs/github.com/kingknull/oblivrashell/internal/services/localservice'
        );
        await SendInput(sessionID, encoded);
      } else {
        const { SendInput } = await import(
          '@wailsjs/github.com/kingknull/oblivrashell/internal/services/sshservice'
        );
        await SendInput(sessionID, encoded);
      }
    } catch (e) {
      console.warn('[shell] SendInput failed:', e);
    }
  }

  function writeLocal(d: string) {
    void writeBackend(d);
  }

  function toggleBroadcastMember() {
    if (sessionID) shellStore.toggleBroadcastMember(sessionID);
  }

  let inBroadcast = $derived(sessionID ? shellStore.broadcastSet.has(sessionID) : false);
  let broadcastActive = $derived(shellStore.broadcastEnabled && inBroadcast);

  onMount(() => {
    if (!containerEl) return;
    tabID = shellStore.activeTabID;
    if (incomingSessionID) sessionID = incomingSessionID;
    term = new Terminal({
      fontFamily: 'JetBrains Mono, Menlo, Monaco, Consolas, "Liberation Mono", "Courier New", monospace',
      fontSize: 13,
      lineHeight: 1.25,
      letterSpacing: 0,
      cursorBlink: true,
      cursorStyle: 'bar',
      allowProposedApi: true,
      scrollback: 5000,
      theme: termTheme(),
    });
    fit = new FitAddon();
    term.loadAddon(fit);
    term.loadAddon(new WebLinksAddon());

    try {
      term.loadAddon(new WebglAddon());
    } catch (e) {
      console.warn('[shell] WebGL addon failed; falling back to canvas', e);
    }

    term.open(containerEl);
    fit.fit();

    // OSC 52 — let remote programs (vim/tmux/…) push to the OS clipboard.
    term.parser.registerOscHandler(52, (data) => {
      const semi = data.indexOf(';');
      if (semi < 0) return false;
      const b64 = data.slice(semi + 1);
      try {
        const text = atob(b64);
        if (navigator.clipboard?.writeText) {
          navigator.clipboard.writeText(text).catch((err) =>
            warnClipboardOnce('OSC 52 copy denied', err),
          );
        }
      } catch {
        /* malformed base64 — ignore */
      }
      return true;
    });

    // Auto-copy on selection.
    term.onSelectionChange(() => {
      const sel = term?.getSelection();
      if (sel && navigator.clipboard?.writeText) {
        navigator.clipboard.writeText(sel).catch((err) =>
          warnClipboardOnce('selection-copy denied', err),
        );
      }
    });

    // Right-click paste.
    containerEl.addEventListener('contextmenu', async (ev) => {
      ev.preventDefault();
      try {
        const text = await navigator.clipboard?.readText?.();
        if (text) await writeBackend(text);
      } catch (e) {
        console.warn('[shell] paste failed:', e);
      }
    });

    term.onData((d) => {
      writeLocal(d);
      // Broadcast fan-out only fires if this leaf is in the broadcast set
      // AND broadcasting is enabled — otherwise it's a no-op.
      if (sessionID) shellStore.fanOutBroadcast(sessionID, d);
    });

    term.onResize(({ cols, rows }) => {
      if (!sessionID) return;
      void resizeBackend(cols, rows);
    });

    resizeObs = new ResizeObserver(() => fit?.fit());
    resizeObs.observe(containerEl);
    // Intentionally NOT calling openLocal() here. The xterm renders empty
    // (just a blinking cursor) and the empty-state overlay invites the
    // operator to either spawn a local PTY or pick a saved host.
    //
    // Exception: if HostList just spawned this tab via spawnTabForHost(),
    // it left a one-shot hostID for us to pick up. Honour it now.
    const pendingHost = shellStore.consumePendingHost();
    if (pendingHost) {
      void switchToRemote(pendingHost);
    }
  });

  onDestroy(() => {
    dataOff?.();
    resizeObs?.disconnect();
    if (sessionID) shellStore.unregisterBroadcastSink(sessionID);
    term?.dispose();
    void closeBackendSession();
    if (leafID) shellStore.forgetLeafSession(leafID);
    if (tabID && leafID) shellStore.forgetLeafMeta(tabID, leafID);
  });

  async function resizeBackend(cols: number, rows: number) {
    try {
      if (mode === 'local') {
        const { Resize } = await import(
          '@wailsjs/github.com/kingknull/oblivrashell/internal/services/localservice'
        );
        await Resize(sessionID, cols, rows);
      } else {
        const { Resize } = await import(
          '@wailsjs/github.com/kingknull/oblivrashell/internal/services/sshservice'
        );
        await Resize(sessionID, cols, rows);
      }
    } catch (e) {
      console.warn('[shell] Resize failed:', e);
    }
  }

  async function closeBackendSession() {
    if (!sessionID) return;
    try {
      if (mode === 'local') {
        const { CloseSession } = await import(
          '@wailsjs/github.com/kingknull/oblivrashell/internal/services/localservice'
        );
        await CloseSession(sessionID);
      }
      // SSH: OBLIVRA's SSHService doesn't expose a per-session Disconnect
      // — sessions clean up when their underlying ssh.Session ends.
      // CloseAll() is the only batch reaper but we don't want to nuke
      // siblings here.
    } catch {
      /* ignore — service may already be down */
    }
  }

  // Tracks IOC matches discovered in stdout so we can re-render hover
  // popovers and the alert pane's "IOCs in this session" badge.
  // Keyed by `${start}:${value}` to coalesce repeats on terminal repaint.
  let recentIOCMatches = $state<IOCMatch[]>([]);
  function recordIOCMatches(matches: IOCMatch[]) {
    if (matches.length === 0) return;
    // Keep last 50 — enough for the right-rail popover, bounded so the
    // array doesn't grow unbounded over a long session.
    const merged = [...recentIOCMatches, ...matches].slice(-50);
    recentIOCMatches = merged;
  }

  /**
   * Decorate any IOC matches in `chunk` with a 1-px red underline
   * inline in the terminal. Implementation note: we cannot mutate
   * stdout text (that breaks ANSI cursor moves), so we use xterm's
   * registerMarker + registerDecoration — they overlay graphics on
   * top of cells without modifying the terminal's character buffer.
   *
   * The decoration is line-level; it underlines the entire line that
   * contains a match. Per-cell-range decoration would require pixel
   * math against xterm's internal renderer and isn't available in
   * the public API as of @xterm/xterm 5.x. Line-level is loud enough
   * to catch the eye and conservative enough to avoid mis-positioning.
   */
  function decorateIOCMatches(chunk: string) {
    if (!term || !hasIOCs() || !chunk) return;
    const matches = scanForIOCs(chunk);
    if (matches.length === 0) return;
    recordIOCMatches(matches);
    // Defer to next tick so xterm has rendered the new content before
    // we ask it for a line marker.
    queueMicrotask(() => {
      if (!term) return;
      try {
        // Anchor the decoration to the line currently at the cursor's
        // top-of-screen. After write() returns, the cursor is at the
        // end of the chunk, so its current line is the last line that
        // contains stdout from this chunk.
        const marker = term.registerMarker(0);
        if (!marker) return;
        const dec = term.registerDecoration({
          marker,
          backgroundColor: '#7f1d1d',  // dark red translucent
          width: term.cols,
        });
        // Auto-dispose after 6 seconds — long enough for the eye to
        // catch it, short enough that an old line of stdout isn't
        // forever painted red.
        setTimeout(() => {
          try { dec?.dispose(); marker.dispose(); } catch { /* already gone */ }
        }, 6000);
      } catch (e) {
        // Decoration API can throw if the renderer is mid-tear-down;
        // we never want a UX nicety to break the shell.
        console.warn('[shell] IOC decoration failed:', e);
      }
    });
  }

  function attachOutputListener() {
    dataOff?.();
    if (!sessionID) return;
    dataOff = subscribe(`terminal:out:${sessionID}`, (data: string) => {
      let decoded: string;
      try {
        decoded = atob(data);
      } catch {
        // Some legacy paths emit raw text — accept it gracefully.
        decoded = data;
      }
      term?.write(decoded, () => {
        // The IOC scan runs AFTER the bytes are committed to xterm's
        // buffer so the marker registration aligns with what's on
        // screen. write() takes an optional callback for exactly this.
        decorateIOCMatches(decoded);
      });
    });
  }

  async function openLocal() {
    status = 'starting';
    errorMsg = '';
    try {
      const { StartLocalSession } = await import(
        '@wailsjs/github.com/kingknull/oblivrashell/internal/services/localservice'
      );
      const id = await StartLocalSession();
      if (typeof id !== 'string' || !id) throw new Error('no session id returned');
      sessionID = id;
      mode = 'local';
      status = 'running';
      attachOutputListener();
      if (sessionID) shellStore.registerBroadcastSink(sessionID, writeLocal);
      if (leafID) shellStore.registerLeafSession(leafID, sessionID);
      if (tabID && leafID) shellStore.setLeafMeta(tabID, leafID, { kind: 'local', title: 'local shell' });
      // Re-emit a resize so the backend matches xterm dimensions.
      if (term) await resizeBackend(term.cols, term.rows);
      term?.focus();
    } catch (e: any) {
      status = 'error';
      errorMsg = String(e?.message ?? e);
    }
  }

  async function switchToRemote(hostID: string) {
    showHostPicker = false;
    const host = shellStore.hosts.find((h) => h.id === hostID);
    if (!host) return;
    if ((host.environment ?? '').toLowerCase() === 'production') {
      const ok = confirm(
        `⚠️ ${host.name} is tagged PRODUCTION.\n\nConnect anyway?`,
      );
      if (!ok) return;
    }
    // Tear down the local PTY before launching SSH on the same leaf.
    if (mode === 'local' && status === 'running') {
      await closeBackendSession();
      if (sessionID) shellStore.unregisterBroadcastSink(sessionID);
    }
    shellStore.selectedHostID = hostID;
    mode = 'remote';

    if (host.authMethod === 'password') {
      const cached = shellStore.hostPasswords[host.id];
      if (!cached) {
        promptingPassword = true;
        return;
      }
      runtimePassword = cached;
    } else {
      runtimePassword = '';
    }
    await actuallyConnect(host.id);
  }

  async function submitPassword() {
    if (!runtimePassword || !shellStore.selectedHostID) return;
    shellStore.setPassword(shellStore.selectedHostID, runtimePassword);
    promptingPassword = false;
    await actuallyConnect(shellStore.selectedHostID);
  }

  async function actuallyConnect(hostID: string) {
    status = 'connecting';
    errorMsg = '';
    try {
      // OBLIVRA's SSHService.Connect uses the credential vault keyed by
      // host.credential_id; runtime passwords are pushed via PushCredential
      // separately (out of scope for the first pass). For now we just call
      // Connect and surface any auth failure verbatim.
      const { Connect } = await import(
        '@wailsjs/github.com/kingknull/oblivrashell/internal/services/sshservice'
      );
      const id = (await Connect(hostID)) as string;
      if (!id) throw new Error('SSH connect returned empty session id');
      sessionID = id;
      status = 'connected';
      connectedHostID = hostID;
      attachOutputListener();
      shellStore.registerBroadcastSink(sessionID, writeLocal);
      if (leafID) shellStore.registerLeafSession(leafID, sessionID);
      const host = shellStore.hosts.find((h) => h.id === hostID);
      if (tabID && leafID && host) {
        shellStore.setLeafMeta(tabID, leafID, {
          kind: 'remote',
          title: `${host.username}@${host.host}`,
        });
      }
      if (term) await resizeBackend(term.cols, term.rows);
      term?.focus();
    } catch (e: any) {
      status = 'error';
      errorMsg = String(e?.message ?? e);
    }
  }

  async function disconnectRemote() {
    // OBLIVRA SSHService doesn't offer per-session disconnect; the safe
    // path is to drop our subscription, mark idle, and reopen a local PTY
    // so the leaf isn't dead. The remote process keeps its own server
    // session until the next CloseAll / agent shutdown.
    if (sessionID) shellStore.unregisterBroadcastSink(sessionID);
    dataOff?.();
    connectedHostID = null;
    status = 'idle';
    await openLocal();
  }

  let connectedHost = $derived(
    connectedHostID ? shellStore.hosts.find((h) => h.id === connectedHostID) ?? null : null,
  );

  let connectedEnv = $derived.by(() => {
    const env = (connectedHost?.environment ?? '').toLowerCase();
    if (env === 'production' || env === 'prod') {
      return { isProd: true, bg: 'rgba(239, 68, 68, 0.10)', color: '#fca5a5', border: 'rgba(239, 68, 68, 0.40)' };
    }
    return { isProd: false, bg: '', color: '', border: '' };
  });
</script>

<div class="relative flex h-full w-full flex-col bg-[#0a0b10]">
  {#if connectedEnv.isProd && status === 'connected'}
    <div
      class="flex items-center justify-center gap-1.5 border-b py-0.5 text-[10px] font-semibold uppercase tracking-[0.2em]"
      style:background={connectedEnv.bg}
      style:color={connectedEnv.color}
      style:border-color={connectedEnv.border}
    >
      <AlertTriangle size={10} />
      production session
      <AlertTriangle size={10} />
    </div>
  {/if}

  <div class="flex items-center gap-2 border-b border-[var(--b1)] bg-[var(--s1)] px-3 py-1.5 text-xs">
    {#if mode === 'local'}
      <TerminalIcon size={14} class="text-[var(--tx3)]" />
      <span class="text-[var(--tx2)]">local</span>
      <span class="font-mono text-[10px] text-[var(--tx3)]">· {sessionID.slice(0, 6) || '…'}</span>
      {#if status === 'starting'}
        <Loader2 size={12} class="animate-spin text-[var(--tx3)]" />
        <span class="text-[var(--tx3)]">starting…</span>
      {:else if status === 'running'}
        <span class="ml-1 h-1.5 w-1.5 rounded-full bg-emerald-400"></span>
      {/if}

      <div class="relative ml-auto">
        <button
          class="flex items-center gap-1.5 rounded px-2 py-1 text-[var(--tx2)] hover:bg-[var(--s2)] hover:text-[var(--tx)]"
          onclick={() => (showHostPicker = !showHostPicker)}
        >
          <Server size={12} />
          <span>connect to host</span>
        </button>
        {#if showHostPicker}
          <div class="absolute right-0 top-full z-30 mt-1 w-64 overflow-hidden rounded-md border border-[var(--b1)] bg-[var(--s2)] shadow-2xl shadow-black/40">
            <div class="px-3 py-2 text-[10px] uppercase tracking-wider text-[var(--tx3)]">Saved hosts</div>
            {#each shellStore.hosts as h (h.id)}
              <button
                class="flex w-full items-center gap-2 px-3 py-1.5 text-left text-xs hover:bg-[var(--s3)]"
                onclick={() => switchToRemote(h.id)}
              >
                <Server size={12} class="text-[var(--tx3)]" />
                <div class="min-w-0 flex-1">
                  <div class="truncate text-[var(--tx)]">{h.name}</div>
                  <div class="truncate text-[10px] text-[var(--tx3)]">{h.username}@{h.host}</div>
                </div>
              </button>
            {/each}
            {#if shellStore.hosts.length === 0}
              <div class="px-3 py-3 text-center text-[11px] text-[var(--tx3)]">
                No saved hosts yet.
              </div>
            {/if}
          </div>
        {/if}
      </div>
    {:else}
      <Plug size={14} class="text-cyan-400" />
      {#if connectedHost}
        <span class="font-mono text-[var(--tx)]">{connectedHost.username}@{connectedHost.host}</span>
        <span class="font-mono text-[10px] text-[var(--tx3)]">:{connectedHost.port}</span>
        <span class="ml-1 h-1.5 w-1.5 rounded-full bg-emerald-400"></span>
      {:else if status === 'connecting'}
        <Loader2 size={12} class="animate-spin text-[var(--tx3)]" />
        <span class="text-[var(--tx3)]">connecting…</span>
      {/if}
      <button
        class="ml-auto flex items-center gap-1.5 rounded px-2 py-1 text-[var(--tx2)] hover:bg-[var(--s2)] hover:text-rose-400"
        onclick={disconnectRemote}
      >
        <Unplug size={12} />
        <span>disconnect</span>
      </button>
    {/if}

    {#if errorMsg}
      <span class="ml-2 truncate font-mono text-[10px] text-rose-400" title={errorMsg}>{errorMsg}</span>
    {/if}

    {#if shellStore.recordingsEnabled && (status === 'running' || status === 'connected')}
      <span
        class="ml-1 flex items-center gap-1 rounded-md border border-rose-400/30 bg-rose-400/10 px-1.5 py-0.5 text-[9px] font-medium uppercase tracking-wider text-rose-400"
        title="This session is being recorded"
      >
        <Circle size={6} class="fill-rose-400 text-rose-400" />
        REC
      </span>
    {/if}

    <button
      class="flex items-center gap-1 rounded-md px-1.5 py-0.5 text-[10px] {inBroadcast
        ? broadcastActive
          ? 'bg-amber-400/15 text-amber-400 border border-amber-400/40'
          : 'bg-[var(--s3)] text-[var(--tx)] border border-[var(--b1)]'
        : 'text-[var(--tx3)] hover:bg-[var(--s2)] hover:text-[var(--tx)] border border-transparent'}"
      onclick={toggleBroadcastMember}
      title={inBroadcast ? 'Remove this pane from the broadcast group' : 'Add this pane to the broadcast group'}
    >
      <Radio size={10} />
      <span>cast</span>
    </button>
  </div>

  <div class="relative flex-1 overflow-hidden">
    <div bind:this={containerEl} class="absolute inset-0 p-2"></div>
    <!-- Empty-state overlay: shown when no PTY/SSH session has been opened
         in this leaf. The xterm itself is still rendered underneath (so the
         cursor blinks the moment the operator picks an option) but a soft
         glass card sits above with the two CTAs. -->
    {#if status === 'idle' && !sessionID}
      <div class="pointer-events-none absolute inset-0 flex items-center justify-center">
        <div class="pointer-events-auto flex max-w-md flex-col items-center gap-4 rounded-xl border border-[var(--b1)] bg-[var(--s1)]/85 px-6 py-5 text-center backdrop-blur-sm shadow-2xl shadow-black/40">
          <div class="text-[10px] font-mono uppercase tracking-[0.18em] text-[var(--tx3)]">
            Empty shell
          </div>
          <div class="text-sm text-[var(--tx2)]">
            Pick what to open in this pane.
          </div>
          <div class="flex flex-wrap items-center justify-center gap-2">
            <button
              class="flex items-center gap-1.5 rounded-md border border-cyan-400/40 bg-cyan-400/10 px-3 py-1.5 text-xs text-cyan-200 hover:bg-cyan-400/20"
              onclick={() => void openLocal()}
            >
              <TerminalIcon size={12} />
              <span>Open local shell</span>
            </button>
            <button
              class="flex items-center gap-1.5 rounded-md border border-[var(--b1)] bg-[var(--s2)] px-3 py-1.5 text-xs text-[var(--tx2)] hover:bg-[var(--s3)] hover:text-[var(--tx)]"
              onclick={() => (showHostPicker = !showHostPicker)}
            >
              <Server size={12} />
              <span>Connect to host</span>
            </button>
          </div>
          <div class="text-[10px] text-[var(--tx3)]">
            Tip: use the host list on the left to launch a tab pre-connected to a saved host.
          </div>
        </div>
      </div>
    {/if}
  </div>

  {#if promptingPassword && shellStore.selectedHostID}
    {@const host = shellStore.hosts.find((h) => h.id === shellStore.selectedHostID)}
    {#if host}
      <div class="border-t border-[var(--b1)] bg-[var(--s1)] px-3 py-2">
        <div class="flex items-center gap-2 text-xs">
          <Lock size={12} class="text-[var(--tx3)]" />
          <span class="text-[var(--tx3)]">Password for {host.username}@{host.host}</span>
          <input
            type="password"
            class="flex-1 rounded bg-[var(--s3)] px-2 py-1 outline-none text-[var(--tx)]"
            bind:value={runtimePassword}
            onkeydown={(e) => e.key === 'Enter' && submitPassword()}
          />
          <button
            class="rounded bg-cyan-500 px-2 py-1 text-[#0a0b10] font-medium"
            onclick={submitPassword}
          >OK</button>
          <button
            class="rounded px-2 py-1 text-[var(--tx3)] hover:bg-[var(--s2)]"
            onclick={() => {
              promptingPassword = false;
              void openLocal();
            }}
          >cancel</button>
        </div>
      </div>
    {/if}
  {/if}
</div>

<style>
  :global(.xterm-viewport) { background-color: transparent !important; }
  :global(.xterm-screen) { padding-left: 2px; }
</style>
