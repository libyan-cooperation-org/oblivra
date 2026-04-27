// shell.svelte.ts — state for the new Blacknode-style Shell workspace.
//
// What this store does (and what it doesn't):
//   - Tracks the open shell tabs (each tab = one Workspace pane tree)
//   - Tracks the active leaf within the active tab so the dock can highlight
//     it and broadcast knows where to fan out
//   - Bridges OBLIVRA's HostService into a flat list the host-picker can
//     consume without knowing OBLIVRA's full Host shape
//   - Owns the multi-cursor broadcast set + sinks (so any pane can register
//     a write callback and any other pane can fan out keystrokes into it)
//
// What this store doesn't do:
//   - It does NOT spawn local PTYs or open SSH sessions; that's the
//     individual `Terminal.svelte` instance's job (it owns its xterm
//     and its sessionID lifecycle so split-and-close is local to the pane).
//   - It does NOT persist; tabs are deliberately ephemeral. A future phase
//     can add a tabs-survive-restart pass once we know what operators want.
//
// Naming note: this is the SHELL store, not the legacy AppStore.sessions.
// As of Stage 2, the old /terminal subsystem is deleted; SnippetsPage and
// StatusBar now read from this store.

import type { PaneNode, LeafNode } from '@components/shell/panes';
import { newLeaf } from '@components/shell/panes';

export interface ShellHostSummary {
  id: string;
  name: string;
  host: string;
  port: number;
  username: string;
  authMethod: string;       // 'password' | 'key' | 'agent' | ''
  environment?: string;     // bridged from OBLIVRA Host.Tags / Category if available
}

export interface ShellTab {
  id: string;
  label: string;
  root: PaneNode;
  activeLeafID: string | null;
  /** Per-leaf metadata so the ShellDock can show "local" vs "user@host". */
  leafMeta: Record<string, { kind: 'local' | 'remote'; title: string }>;
}

const uuid = () =>
  typeof crypto !== 'undefined' && crypto.randomUUID
    ? crypto.randomUUID()
    : Math.random().toString(36).slice(2);

class ShellStore {
  // ── Tabs ─────────────────────────────────────────────────────────────
  tabs = $state<ShellTab[]>([]);
  activeTabID = $state<string | null>(null);

  /**
   * leafID → real backend session id (assigned by `LocalService.StartLocalSession`
   * or `SSHService.Connect`). Terminal.svelte stamps this when its session
   * opens; SnippetsPage / palette / AI insert use it to inject text into
   * the right PTY without needing to walk the pane tree.
   */
  leafSessions = $state<Record<string, string>>({});

  /** Active backend session id (real PTY/SSH id), or null if no leaf is focused. */
  get activeSessionID(): string | null {
    const tab = this.tabs.find((t) => t.id === this.activeTabID);
    if (!tab || !tab.activeLeafID) return null;
    return this.leafSessions[tab.activeLeafID] ?? null;
  }

  /** Total number of live shell sessions across every tab. Used by StatusBar. */
  get sessionCount(): number {
    return Object.keys(this.leafSessions).length;
  }

  registerLeafSession(leafID: string, backendSessionID: string) {
    this.leafSessions[leafID] = backendSessionID;
  }
  forgetLeafSession(leafID: string) {
    delete this.leafSessions[leafID];
  }

  // ── Hosts (bridged from OBLIVRA HostService) ─────────────────────────
  hosts = $state<ShellHostSummary[]>([]);
  /** Runtime password cache, never persisted. Keyed by hostID. */
  hostPasswords = $state<Record<string, string>>({});
  /** Currently selected host in the picker — surfaces its name in the toolbar. */
  selectedHostID = $state<string | null>(null);

  // ── Settings (mirrored from OBLIVRA SettingsService where available) ──
  recordingsEnabled = $state(false);
  theme = $state<'dark' | 'light'>('dark');

  // ── Multi-cursor broadcast ───────────────────────────────────────────
  broadcastEnabled = $state(false);
  broadcastSet = $state<Set<string>>(new Set());
  // svelte-ignore state_referenced_locally
  private broadcastSinks = $state<Record<string, (data: string) => void>>({});

  registerBroadcastSink(sessionID: string, write: (data: string) => void) {
    this.broadcastSinks[sessionID] = write;
  }
  unregisterBroadcastSink(sessionID: string) {
    delete this.broadcastSinks[sessionID];
    if (this.broadcastSet.has(sessionID)) {
      const next = new Set(this.broadcastSet);
      next.delete(sessionID);
      this.broadcastSet = next;
    }
  }
  toggleBroadcastMember(sessionID: string) {
    const next = new Set(this.broadcastSet);
    if (next.has(sessionID)) next.delete(sessionID);
    else next.add(sessionID);
    this.broadcastSet = next;
  }
  fanOutBroadcast(sourceSessionID: string, data: string) {
    if (!this.broadcastEnabled) return;
    if (!this.broadcastSet.has(sourceSessionID)) return;
    for (const sid of this.broadcastSet) {
      if (sid === sourceSessionID) continue;
      const sink = this.broadcastSinks[sid];
      if (sink) sink(data);
    }
  }

  // ── Pending host-connect channel ─────────────────────────────────────
  /**
   * One-shot signal: the next Terminal that mounts should skip the idle
   * empty-state and immediately call SSH.Connect for this host. Used by
   * HostList → spawnTabForHost so clicking a saved host opens a new tab
   * already on its way to "connected".
   */
  pendingHostConnect = $state<string | null>(null);

  /** Spawn a fresh tab whose first leaf auto-connects to the given host. */
  spawnTabForHost(hostID: string) {
    const host = this.hosts.find((h) => h.id === hostID);
    const label = host ? host.name : 'Remote';
    this.pendingHostConnect = hostID;
    this.spawnTab(label);
  }

  /** Consume + clear the pending-host signal. Returns the hostID once. */
  consumePendingHost(): string | null {
    const v = this.pendingHostConnect;
    this.pendingHostConnect = null;
    return v;
  }

  // ── AI/palette → terminal write channel ──────────────────────────────
  pendingTerminalInsert = $state<
    { id: string; sessionID: string; text: string } | null
  >(null);
  insertIntoTerminal(sessionID: string, text: string) {
    this.pendingTerminalInsert = {
      id: uuid(),
      sessionID,
      text,
    };
  }

  // ── Tab lifecycle ────────────────────────────────────────────────────
  spawnTab(label = 'Local'): ShellTab {
    const leaf = newLeaf();
    const tab: ShellTab = {
      id: uuid(),
      label,
      root: leaf,
      activeLeafID: leaf.id,
      leafMeta: { [leaf.id]: { kind: 'local', title: 'local shell' } },
    };
    this.tabs = [...this.tabs, tab];
    this.activeTabID = tab.id;
    return tab;
  }

  closeTab(id: string) {
    const idx = this.tabs.findIndex((t) => t.id === id);
    if (idx < 0) return;
    const next = this.tabs.filter((t) => t.id !== id);
    this.tabs = next;
    if (this.activeTabID === id) {
      // Land on the neighbour, or null if none left.
      this.activeTabID = next[idx] ? next[idx].id : next[idx - 1] ? next[idx - 1].id : null;
    }
  }

  setActiveTab(id: string) {
    if (this.tabs.some((t) => t.id === id)) this.activeTabID = id;
  }

  /** Replace the pane tree of a tab (Pane.svelte calls this on split/close). */
  updateTabRoot(id: string, root: PaneNode | null, activeLeafID: string | null) {
    if (root === null) {
      this.closeTab(id);
      return;
    }
    this.tabs = this.tabs.map((t) =>
      t.id === id ? { ...t, root, activeLeafID } : t,
    );
  }

  /** Stamp metadata onto a leaf — Terminal.svelte calls this when its mode flips. */
  setLeafMeta(tabID: string, leafID: string, meta: { kind: 'local' | 'remote'; title: string }) {
    this.tabs = this.tabs.map((t) =>
      t.id === tabID
        ? { ...t, leafMeta: { ...t.leafMeta, [leafID]: meta } }
        : t,
    );
  }

  /** Forget a leaf when it's pruned (split close / tab close). */
  forgetLeafMeta(tabID: string, leafID: string) {
    this.tabs = this.tabs.map((t) => {
      if (t.id !== tabID) return t;
      const { [leafID]: _, ...rest } = t.leafMeta;
      return { ...t, leafMeta: rest };
    });
  }

  setPassword(hostID: string, password: string) {
    this.hostPasswords[hostID] = password;
  }

  // ── Host bridge ──────────────────────────────────────────────────────
  /**
   * Refresh hosts from OBLIVRA's HostService. Defensive: this store
   * mounts in any browser-side import (including hot-reload), but the
   * Wails bindings only resolve in the desktop runtime. We dynamic-import
   * + swallow on failure so the shell page still renders in dev mode.
   */
  async refreshHosts() {
    try {
      const { ListHosts } = await import(
        '@wailsjs/github.com/kingknull/oblivrashell/internal/services/hostservice'
      );
      const list = ((await ListHosts()) ?? []) as Array<{
        id: string;
        label?: string;
        hostname?: string;
        port?: number;
        username?: string;
        auth_method?: string;
        category?: string;
      }>;
      this.hosts = list.map((h) => ({
        id: h.id,
        name: h.label || h.hostname || h.id,
        host: h.hostname || '',
        port: h.port || 22,
        username: h.username || '',
        authMethod: h.auth_method || '',
        environment: h.category,
      }));
    } catch {
      // Wails not ready (e.g. SSR / vite preview / test) — keep last-known list.
    }
  }
}

export const shellStore = new ShellStore();
