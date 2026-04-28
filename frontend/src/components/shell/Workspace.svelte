<!--
  Shell Workspace — Blacknode-style 3-column DevOps console.

  Layout (rendered INSIDE OBLIVRA's AppLayout, so the outer 64px
  AppSidebar + status bar are still around it):

  ┌─[44 view-nav]─┬─[280 HostList]─┬─[main: terminal panes / panel]─┐
  │ Terminals     │ Search…        │ Tabs strip + global toolbar     │
  │ Multi-host    │ ─ Production   │ ────────────────────────────── │
  │ Files         │   prod-web-1   │                                  │
  │ Metrics       │   prod-web-2   │       Terminal pane tree        │
  │ Logs          │ ─ Staging      │       (or panel for the         │
  │ Forwards      │   stg-api      │        currently-selected view) │
  │ Recordings    │ ─ Ungrouped    │                                  │
  │ Containers    │   …            │                                  │
  │ Network       │                │ ────────────────────────────── │
  │ Processes     │                │ ShellDock (open shells, all     │
  │ HTTP          │                │  tabs)                           │
  │ Database      │                │                                  │
  │ Snippets      │                │                                  │
  │ History       │                │                                  │
  │ Keys          │                │                                  │
  │ Settings      │                │                                  │
  └───────────────┴────────────────┴──────────────────────────────────┘

  Currently fully wired: Terminals view (Pane tree + ShellDock).

  Other views render a `ViewPlaceholder` honest about being a stub, with
  a link to OBLIVRA's existing equivalent page when one exists. Stage 3
  of the Blacknode migration ports each panel one at a time as the
  underlying Go services land in OBLIVRA.
-->
<script lang="ts">
  import { onMount } from 'svelte';
  import {
    Plus,
    X,
    Radio,
    MoonStar,
    Sun,
    TerminalSquare,
    Zap,
    Folder,
    Activity,
    KeyRound,
    Network,
    ScrollText,
    Film,
    Boxes,
    Radar,
    Cpu,
    Globe2,
    Database,
    Bookmark,
    History as HistoryIcon,
    Settings as SettingsIcon,
  } from 'lucide-svelte';
  import { shellStore } from '@lib/stores/shell.svelte';
  import { splitLeaf, closeLeaf, setRatio } from './panes';
  import Pane from './Pane.svelte';
  import HostList from './HostList.svelte';
  import ShellDock from './ShellDock.svelte';
  import ShellAlertRail from './ShellAlertRail.svelte';
  import ViewPlaceholder from './ViewPlaceholder.svelte';
  // Real Blacknode-style panels — all wired now.
  import ExecPanel from './ExecPanel.svelte';
  import SFTPPanel from './SFTPPanel.svelte';
  import MetricsPanel from './MetricsPanel.svelte';
  import LogsPanel from './LogsPanel.svelte';
  import ForwardsPanel from './ForwardsPanel.svelte';
  import RecordingsPanel from './RecordingsPanel.svelte';
  import ContainersPanel from './ContainersPanel.svelte';
  import NetworkPanel from './NetworkPanel.svelte';
  import ProcessesPanel from './ProcessesPanel.svelte';
  import HTTPPanel from './HTTPPanel.svelte';
  import SnippetsPanel from './SnippetsPanel.svelte';
  import HistoryPanel from './HistoryPanel.svelte';
  import KeysPanel from './KeysPanel.svelte';
  import SettingsPanel from './SettingsPanel.svelte';

  type View =
    | 'terminals'
    | 'exec'
    | 'files'
    | 'metrics'
    | 'logs'
    | 'forwards'
    | 'recordings'
    | 'containers'
    | 'network'
    | 'processes'
    | 'http'
    | 'database'
    | 'snippets'
    | 'history'
    | 'keys'
    | 'settings';

  const VIEWS: { id: View; label: string; Icon: any }[] = [
    { id: 'terminals',  label: 'Terminals',  Icon: TerminalSquare },
    { id: 'exec',       label: 'Multi-host', Icon: Zap },
    { id: 'files',      label: 'Files',      Icon: Folder },
    { id: 'metrics',    label: 'Metrics',    Icon: Activity },
    { id: 'logs',       label: 'Logs',       Icon: ScrollText },
    { id: 'forwards',   label: 'Forwards',   Icon: Network },
    { id: 'recordings', label: 'Recordings', Icon: Film },
    { id: 'containers', label: 'Containers', Icon: Boxes },
    { id: 'network',    label: 'Network',    Icon: Radar },
    { id: 'processes',  label: 'Processes',  Icon: Cpu },
    { id: 'http',       label: 'HTTP',       Icon: Globe2 },
    { id: 'database',   label: 'Database',   Icon: Database },
    { id: 'snippets',   label: 'Snippets',   Icon: Bookmark },
    { id: 'history',    label: 'History',    Icon: HistoryIcon },
    { id: 'keys',       label: 'Keys',       Icon: KeyRound },
    { id: 'settings',   label: 'Settings',   Icon: SettingsIcon },
  ];

  let view = $state<View>('terminals');

  onMount(() => {
    if (shellStore.tabs.length === 0) {
      shellStore.spawnTab('Local');
    }
    void shellStore.refreshHosts();
  });

  let activeTab = $derived(
    shellStore.tabs.find((t) => t.id === shellStore.activeTabID) ?? null,
  );

  function newTab() {
    shellStore.spawnTab(`Local ${shellStore.tabs.length + 1}`);
  }

  function closeTab(e: MouseEvent, id: string) {
    e.stopPropagation();
    shellStore.closeTab(id);
  }

  function activateTab(id: string) {
    shellStore.setActiveTab(id);
  }

  function onActivateLeaf(tabID: string, leafID: string) {
    const tab = shellStore.tabs.find((t) => t.id === tabID);
    if (!tab) return;
    shellStore.updateTabRoot(tabID, tab.root, leafID);
  }
  function onSplit(tabID: string, leafID: string, dir: 'horizontal' | 'vertical') {
    const tab = shellStore.tabs.find((t) => t.id === tabID);
    if (!tab) return;
    const next = splitLeaf(tab.root, leafID, dir);
    shellStore.updateTabRoot(tabID, next, tab.activeLeafID);
  }
  function onClose(tabID: string, leafID: string) {
    const tab = shellStore.tabs.find((t) => t.id === tabID);
    if (!tab) return;
    const next = closeLeaf(tab.root, leafID);
    if (next === null) {
      shellStore.closeTab(tabID);
      return;
    }
    const nextActive = next.kind === 'leaf' ? next.id : tab.activeLeafID;
    shellStore.updateTabRoot(tabID, next, nextActive);
    shellStore.forgetLeafMeta(tabID, leafID);
  }
  function onResize(tabID: string, splitID: string, ratio: number) {
    const tab = shellStore.tabs.find((t) => t.id === tabID);
    if (!tab) return;
    const next = setRatio(tab.root, splitID, ratio);
    shellStore.updateTabRoot(tabID, next, tab.activeLeafID);
  }

  function toggleBroadcast() {
    shellStore.broadcastEnabled = !shellStore.broadcastEnabled;
  }
  function toggleTheme() {
    shellStore.theme = shellStore.theme === 'dark' ? 'light' : 'dark';
  }

  // Per-view placeholder content. Honest about being a stub and
  // (where applicable) points to OBLIVRA's existing equivalent page.
  const PLACEHOLDERS: Record<Exclude<View, 'terminals'>, { title: string; description: string; existingPath?: string; existingLabel?: string }> = {
    exec: {
      title: 'Multi-host execution',
      description: 'Run a command across N selected hosts in parallel and stream the per-host output side by side. Great for "patch level on all prod boxes" or "is service X up everywhere?". Backend will reuse OBLIVRA\'s SSH session pool.',
    },
    files: {
      title: 'Remote file browser',
      description: 'Two-pane SFTP browser with drag-drop transfer, inline rename / chmod, and a grep-into-remote-paths action. Backend hooks already exist on SSHService (ListDirectory, ReadFile, WriteFile, SftpDownloadAsync) — just needs the panel UI.',
    },
    metrics: {
      title: 'Live host metrics',
      description: 'Top-style CPU/memory/disk/network sparklines per saved host. Refresh every few seconds. Will rely on a small metrics-collector service that doesn\'t exist yet — Stage 4 work.',
    },
    logs: {
      title: 'System log tail',
      description: 'tail -F on /var/log/syslog (or journalctl) inside an SSH host. Distinct from OBLIVRA\'s SIEM stream — this is per-host, ad-hoc, no ingestion.',
      existingPath: '/siem',
      existingLabel: 'Open SIEM stream (different scope)',
    },
    forwards: {
      title: 'SSH port forwards',
      description: 'Create / list / kill SSH port forwards (local + remote). Lives on the existing OBLIVRA Tunnels page.',
      existingPath: '/tunnels',
      existingLabel: 'Open Tunnels',
    },
    recordings: {
      title: 'Session recordings',
      description: 'Replay past terminal sessions with scrub, speed control, and search. Lives on OBLIVRA\'s existing Recordings page.',
      existingPath: '/recordings',
      existingLabel: 'Open Recordings',
    },
    containers: {
      title: 'Docker / containerd',
      description: 'List running containers on a host, attach to logs, exec a shell into a container, restart / kill. Needs a containers backend service (Stage 4).',
    },
    network: {
      title: 'Network inspector',
      description: 'Per-host listening sockets, established connections, ARP / route tables. Useful during incident triage. Stage 4.',
    },
    processes: {
      title: 'Process explorer',
      description: 'top / ps with sort + filter, kill / renice from the UI, diff snapshots. Stage 4.',
    },
    http: {
      title: 'HTTP client',
      description: 'Postman-style request runner pinned to a host (curl-via-SSH so requests originate from the target). Stage 4.',
    },
    database: {
      title: 'Database client',
      description: 'mysql / psql shells over SSH tunnels with saved credentials. Stage 4.',
    },
    snippets: {
      title: 'Command snippets',
      description: 'Saved one-liners. Lives on OBLIVRA\'s existing Snippets page.',
      existingPath: '/snippets',
      existingLabel: 'Open Snippets',
    },
    history: {
      title: 'Command history',
      description: 'Searchable history across every shell session. OBLIVRA\'s CommandHistoryService already records this — needs the browse UI.',
    },
    keys: {
      title: 'SSH keys',
      description: 'Generate / import / list SSH public keys, deploy to a host with one click. OBLIVRA\'s SSHService.DeployKey already exists; needs a key-management UI.',
      existingPath: '/vault',
      existingLabel: 'Open Vault',
    },
    settings: {
      title: 'Shell settings',
      description: 'Theme, default shell, recording opt-in, broadcast confirmation. Most of this lives in OBLIVRA\'s general settings.',
      existingPath: '/workspace',
      existingLabel: 'Open Settings',
    },
  };
</script>

<div class="grid h-full w-full grid-cols-[44px_280px_1fr] overflow-hidden bg-[var(--s0)] text-[var(--tx)]">
  <!-- ── 44px view nav ──────────────────────────────────────────── -->
  <nav class="flex flex-col items-center gap-1 border-r border-[var(--b1)] bg-[var(--s1)] py-2">
    {#each VIEWS as v (v.id)}
      <button
        title={v.label}
        class="group relative flex h-9 w-9 items-center justify-center rounded-md transition-colors {view === v.id
          ? 'text-cyan-400'
          : 'text-[var(--tx3)] hover:bg-[var(--s3)] hover:text-[var(--tx)]'}"
        onclick={() => (view = v.id)}
      >
        {#if view === v.id}
          <span class="absolute left-0 top-1.5 bottom-1.5 w-0.5 rounded-r bg-cyan-400"></span>
        {/if}
        <v.Icon size={16} />
      </button>
    {/each}
  </nav>

  <!-- ── 280px HostList ─────────────────────────────────────────── -->
  <aside class="overflow-hidden border-r border-[var(--b1)]">
    <HostList />
  </aside>

  <!-- ── Main ───────────────────────────────────────────────────── -->
  <main class="flex min-w-0 flex-col overflow-hidden">
    {#if view === 'terminals'}
      <!-- Tabs strip + global toolbar -->
      <div class="flex items-center gap-1 border-b border-[var(--b1)] bg-[var(--s1)] px-2 py-1">
        <div class="flex items-center gap-1 overflow-x-auto">
          {#each shellStore.tabs as tab (tab.id)}
            {@const isActive = tab.id === shellStore.activeTabID}
            <div
              class="group flex items-center gap-1 rounded-t-md border-b-2 transition-colors {isActive
                ? 'border-cyan-400 bg-[var(--s2)] text-[var(--tx)]'
                : 'border-transparent text-[var(--tx3)] hover:bg-[var(--s2)] hover:text-[var(--tx2)]'}"
            >
              <button class="px-3 py-1 text-xs" onclick={() => activateTab(tab.id)}>{tab.label}</button>
              <button
                class="mr-1 rounded p-0.5 opacity-0 transition-opacity group-hover:opacity-60 hover:!opacity-100 hover:bg-rose-400/20 hover:text-rose-400"
                aria-label={`Close ${tab.label}`}
                onclick={(e) => closeTab(e, tab.id)}
              >
                <X size={10} />
              </button>
            </div>
          {/each}
          <button
            class="ml-1 rounded p-1 text-[var(--tx3)] hover:bg-[var(--s2)] hover:text-[var(--tx)]"
            title="New tab"
            onclick={newTab}
          >
            <Plus size={14} />
          </button>
        </div>

        <div class="ml-auto flex items-center gap-1">
          <button
            class="flex items-center gap-1 rounded-md px-2 py-1 text-[10px] {shellStore.broadcastEnabled
              ? 'bg-amber-400/15 text-amber-400 border border-amber-400/40'
              : 'text-[var(--tx3)] hover:bg-[var(--s2)] hover:text-[var(--tx)] border border-transparent'}"
            onclick={toggleBroadcast}
            title="Broadcast keystrokes to all panes flagged 'cast'"
          >
            <Radio size={11} />
            <span class="uppercase tracking-wider">broadcast {shellStore.broadcastEnabled ? 'on' : 'off'}</span>
          </button>
          <button
            class="rounded-md p-1 text-[var(--tx3)] hover:bg-[var(--s2)] hover:text-[var(--tx)]"
            onclick={toggleTheme}
            title="Toggle theme"
          >
            {#if shellStore.theme === 'dark'}
              <MoonStar size={13} />
            {:else}
              <Sun size={13} />
            {/if}
          </button>
        </div>
      </div>

      <!-- Active pane tree + alert rail.
           Alert rail (Phase 32) auto-scopes to the host the active
           terminal session is connected to, and lets the operator
           Ctrl+Click an alert to inject a comment-block + suggested
           command into the active xterm. The bidirectional fusion is
           the moat — see ShellAlertRail.svelte. -->
      <div class="flex min-h-0 flex-1 overflow-hidden">
        <div class="flex-1 min-w-0">
          {#if activeTab}
            {#key activeTab.id}
              <Pane
                node={activeTab.root}
                activeLeafID={activeTab.activeLeafID}
                onactivate={(leafID) => onActivateLeaf(activeTab!.id, leafID)}
                onsplit={(leafID, dir) => onSplit(activeTab!.id, leafID, dir)}
                onclose={(leafID) => onClose(activeTab!.id, leafID)}
                onresize={(splitID, ratio) => onResize(activeTab!.id, splitID, ratio)}
              />
            {/key}
          {:else}
            <div class="flex h-full items-center justify-center text-sm text-[var(--tx3)]">
              No tabs open. Click + to spawn a shell.
            </div>
          {/if}
        </div>
        <ShellAlertRail />
      </div>

      <ShellDock />
    {:else if view === 'exec'}
      <ExecPanel />
    {:else if view === 'files'}
      <SFTPPanel />
    {:else if view === 'metrics'}
      <MetricsPanel />
    {:else if view === 'logs'}
      <LogsPanel />
    {:else if view === 'forwards'}
      <ForwardsPanel />
    {:else if view === 'recordings'}
      <RecordingsPanel />
    {:else if view === 'containers'}
      <ContainersPanel />
    {:else if view === 'network'}
      <NetworkPanel />
    {:else if view === 'processes'}
      <ProcessesPanel />
    {:else if view === 'http'}
      <HTTPPanel />
    {:else if view === 'database'}
      {@const cfg = PLACEHOLDERS.database}
      {@const Icon = VIEWS.find((v) => v.id === 'database')!.Icon}
      <ViewPlaceholder
        title={cfg.title}
        description={cfg.description}
        icon={Icon}
        existingPath={cfg.existingPath}
        existingLabel={cfg.existingLabel}
      />
    {:else if view === 'snippets'}
      <SnippetsPanel />
    {:else if view === 'history'}
      <HistoryPanel />
    {:else if view === 'keys'}
      <KeysPanel />
    {:else if view === 'settings'}
      <SettingsPanel />
    {/if}
  </main>
</div>
