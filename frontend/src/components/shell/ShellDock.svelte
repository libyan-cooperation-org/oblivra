<!--
  ShellDock — bottom strip listing every open shell session across every
  tab in the workspace. Operator-requested companion to the Shell sidebar
  icon: "add separate sidebar icon for it call shell with dock for shell
  on the app".

  Design:
    - One chip per leaf in shellStore.tabs[*].root tree
    - Chip shows: kind icon (local / remote), short title, broadcast badge
    - Click a chip → switch to its parent tab and mark its leaf active
    - X on hover → close that leaf (bubbles into Workspace's onclose path
      via shellStore.updateTabRoot, which Pane.svelte already handles)
    - "+" button at end → spawn a new tab with a fresh local leaf

  This is intentionally NOT the same as the route-switching BottomDock —
  that one navigates between sidebar groups. THIS dock manages live shell
  sessions and only lives inside /shell.
-->
<script lang="ts">
  import { Plus, Terminal as TerminalIcon, Plug, X, Radio } from 'lucide-svelte';
  import { shellStore } from '@lib/stores/shell.svelte';
  import { leaves, closeLeaf, type LeafNode } from './panes';

  type DockEntry = {
    tabID: string;
    tabLabel: string;
    leaf: LeafNode;
    meta: { kind: 'local' | 'remote'; title: string };
    inBroadcast: boolean;
  };

  // Flatten every tab's pane tree into a list of dock entries. Leaves that
  // haven't reported metadata yet (fresh PTYs that just spawned) get a
  // generic "local" placeholder so they still render.
  let entries = $derived.by<DockEntry[]>(() =>
    shellStore.tabs.flatMap((tab) =>
      leaves(tab.root).map<DockEntry>((leaf) => ({
        tabID: tab.id,
        tabLabel: tab.label,
        leaf,
        meta: tab.leafMeta[leaf.id] ?? { kind: 'local', title: 'shell' },
        inBroadcast: shellStore.broadcastSet.has(leaf.sessionID),
      })),
    ),
  );

  function focusLeaf(entry: DockEntry) {
    shellStore.setActiveTab(entry.tabID);
    const tab = shellStore.tabs.find((t) => t.id === entry.tabID);
    if (tab) shellStore.updateTabRoot(entry.tabID, tab.root, entry.leaf.id);
  }

  function closeLeafEntry(e: MouseEvent, entry: DockEntry) {
    e.stopPropagation();
    const tab = shellStore.tabs.find((t) => t.id === entry.tabID);
    if (!tab) return;
    const next = closeLeaf(tab.root, entry.leaf.id);
    if (next === null) {
      shellStore.closeTab(entry.tabID);
      return;
    }
    const nextActive = next.kind === 'leaf' ? next.id : tab.activeLeafID;
    shellStore.updateTabRoot(entry.tabID, next, nextActive);
    shellStore.forgetLeafMeta(entry.tabID, entry.leaf.id);
  }

  function newShell() {
    shellStore.spawnTab(`Local ${shellStore.tabs.length + 1}`);
  }
</script>

<div class="flex h-9 shrink-0 items-center gap-1 overflow-x-auto border-t border-[var(--b1)] bg-[var(--s1)] px-2 py-1">
  <span class="mr-1 shrink-0 px-1 font-mono text-[9px] uppercase tracking-[0.16em] text-[var(--tx3)]">
    Shells
  </span>

  {#each entries as entry (entry.tabID + ':' + entry.leaf.id)}
    {@const isActive =
      entry.tabID === shellStore.activeTabID &&
      entry.leaf.id ===
        (shellStore.tabs.find((t) => t.id === entry.tabID)?.activeLeafID ?? null)}
    <!-- Two real buttons in a flex row — see Workspace.svelte tabs for
         why we don't nest <button> inside <button>. -->
    <div
      class="group flex shrink-0 items-center gap-1.5 rounded-md border pl-2 pr-1 py-1 text-[11px] transition-colors {isActive
        ? 'border-cyan-400/50 bg-cyan-400/10 text-[var(--tx)]'
        : 'border-[var(--b1)] bg-[var(--s2)] text-[var(--tx3)] hover:border-[var(--b2)] hover:text-[var(--tx2)]'}"
      title={`${entry.tabLabel} · ${entry.meta.title}`}
    >
      <button
        class="flex flex-1 items-center gap-1.5"
        onclick={() => focusLeaf(entry)}
      >
        {#if entry.meta.kind === 'remote'}
          <Plug size={11} class="text-cyan-400" />
        {:else}
          <TerminalIcon size={11} class={isActive ? 'text-cyan-400' : 'text-[var(--tx3)]'} />
        {/if}
        <span class="max-w-[180px] truncate">{entry.meta.title}</span>
        {#if entry.inBroadcast && shellStore.broadcastEnabled}
          <Radio size={9} class="text-amber-400" />
        {/if}
      </button>
      <button
        class="rounded p-0.5 opacity-0 transition-opacity group-hover:opacity-60 hover:!opacity-100 hover:bg-rose-400/20 hover:text-rose-400"
        aria-label={`Close ${entry.meta.title}`}
        onclick={(e) => closeLeafEntry(e, entry)}
      >
        <X size={9} />
      </button>
    </div>
  {/each}

  <button
    class="ml-1 shrink-0 rounded-md border border-dashed border-[var(--b1)] p-1 text-[var(--tx3)] hover:border-cyan-400/40 hover:text-[var(--tx)]"
    title="Spawn a new shell tab"
    onclick={newShell}
  >
    <Plus size={12} />
  </button>
</div>
