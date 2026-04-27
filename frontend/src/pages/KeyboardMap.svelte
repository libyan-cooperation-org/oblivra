<!-- Keyboard Map — static reference for app shortcuts. -->
<script lang="ts">
  import { PageLayout, PopOutButton } from '@components/ui';
  import { Keyboard } from 'lucide-svelte';

  type Group = { title: string; items: { keys: string; desc: string }[] };

  // Pulled directly from App.svelte's keybind handlers + the per-page menu wiring.
  // Kept here as the canonical reference instead of drift-prone docs.
  const SHORTCUTS: Group[] = [
    { title: 'Global', items: [
      { keys: 'Ctrl/⌘ K',     desc: 'Open command palette' },
      { keys: 'Ctrl/⌘ /',     desc: 'Focus search' },
      { keys: 'Ctrl/⌘ ,',     desc: 'Open settings' },
      { keys: 'Ctrl/⌘ Shift T', desc: 'New shell tab' },
    ] },
    { title: 'Investigation', items: [
      { keys: 'Ctrl/⌘ Shift I', desc: 'Isolate active host (operator mode)' },
      { keys: 'Ctrl/⌘ Shift E', desc: 'Capture forensic evidence on active host' },
      { keys: 'Ctrl/⌘ Shift F', desc: 'Pivot to SIEM scoped to host' },
    ] },
    { title: 'Shell Workspace', items: [
      { keys: 'Right-click pane', desc: 'Paste from OS clipboard' },
      { keys: 'Select text',    desc: 'Auto-copy to clipboard' },
      { keys: 'Click host',     desc: 'New tab pre-connected via SSH' },
      { keys: 'cast button',    desc: 'Add pane to broadcast group' },
    ] },
    { title: 'Navigation', items: [
      { keys: 'Click sidebar group', desc: 'Switch context (Overview / Security / …)' },
      { keys: 'Click dock card',     desc: 'Navigate to feature page' },
    ] },
  ];
</script>

<PageLayout title="Keyboard Shortcuts" subtitle="Reference for every chord recognised by the app">
  {#snippet toolbar()}
    <PopOutButton route="/shortcuts" title="Shortcuts" />
  {/snippet}
  <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
    {#each SHORTCUTS as g}
      <div class="bg-surface-1 border border-border-primary rounded-md p-4">
        <div class="flex items-center gap-2 mb-3">
          <Keyboard size={14} class="text-accent" />
          <span class="text-[10px] uppercase tracking-widest font-bold">{g.title}</span>
        </div>
        <table class="w-full text-[11px]">
          <tbody>
            {#each g.items as it}
              <tr class="border-b border-border-primary">
                <td class="py-1.5 pr-3"><kbd class="bg-surface-2 border border-border-primary rounded px-1.5 py-0.5 font-mono text-[10px]">{it.keys}</kbd></td>
                <td class="py-1.5 text-text-muted">{it.desc}</td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/each}
  </div>
</PageLayout>
