<!--
  ViewPlaceholder — interim card shown for Blacknode workspace views that
  haven't had their backend wired into OBLIVRA yet (Containers, DBPanel,
  HTTPPanel, NetworkPanel, ProcessesPanel, MetricsPanel, LogsPanel,
  ExecPanel, SFTPPanel).

  Honest about being a stub: explains what the view will do once wired,
  links to the equivalent OBLIVRA page if one exists, and gives the
  operator a "+ open shell" escape hatch so they don't get stuck.
-->
<script lang="ts">
  import { ArrowRight, TerminalSquare } from 'lucide-svelte';
  import { push } from '@lib/router.svelte';
  import { shellStore } from '@lib/stores/shell.svelte';

  type Props = {
    title: string;
    description: string;
    icon: any;
    /** Optional path to OBLIVRA's existing equivalent page. */
    existingPath?: string;
    /** Label for the existing-page link. */
    existingLabel?: string;
  };

  let { title, description, icon: Icon, existingPath, existingLabel }: Props = $props();

  function backToShell() {
    if (shellStore.tabs.length === 0) shellStore.spawnTab('Local');
  }
</script>

<div class="flex h-full w-full items-center justify-center p-8">
  <div class="flex max-w-lg flex-col items-center gap-4 rounded-xl border border-[var(--b1)] bg-[var(--s1)] px-8 py-7 text-center">
    <div class="flex h-12 w-12 items-center justify-center rounded-lg bg-[var(--s2)] text-cyan-400">
      <Icon size={22} />
    </div>
    <div class="text-base font-semibold text-[var(--tx)]">{title}</div>
    <div class="text-[10px] font-mono uppercase tracking-[0.18em] text-amber-300">
      Coming next · backend not yet wired
    </div>
    <p class="text-sm leading-relaxed text-[var(--tx2)]">
      {description}
    </p>

    <div class="flex flex-wrap items-center justify-center gap-2">
      {#if existingPath}
        <button
          class="flex items-center gap-1.5 rounded-md border border-cyan-400/40 bg-cyan-400/10 px-3 py-1.5 text-xs text-cyan-200 hover:bg-cyan-400/20"
          onclick={() => push(existingPath!)}
        >
          <span>{existingLabel ?? 'Open existing page'}</span>
          <ArrowRight size={11} />
        </button>
      {/if}
      <button
        class="flex items-center gap-1.5 rounded-md border border-[var(--b1)] bg-[var(--s2)] px-3 py-1.5 text-xs text-[var(--tx2)] hover:bg-[var(--s3)] hover:text-[var(--tx)]"
        onclick={backToShell}
      >
        <TerminalSquare size={11} />
        <span>Back to shell</span>
      </button>
    </div>
  </div>
</div>
