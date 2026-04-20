<!--
  OBLIVRA — Keyboard Interaction Map (Svelte 5)
  Command reference and shortcut orchestration guide.
-->
<script lang="ts">
  import { PageLayout, Badge, Button } from '@components/ui';
  import { Keyboard, MousePointer2, Command, Globe, ShieldAlert, Zap, Search, LayoutDashboard } from 'lucide-svelte';

  const sections = [
    {
      title: '01 · GLOBAL SHORTCUTS',
      shortcuts: [
        { keys: ['⌘', 'K'], desc: 'Open command palette', context: 'GLOBAL' },
        { keys: ['⌘', 'P'], desc: 'Global entity search', context: 'GLOBAL' },
        { keys: ['⌘', 'T'], desc: 'Switch tenant', context: 'GLOBAL' },
        { keys: ['⌘', 'N'], desc: 'New incident / alert note', context: 'GLOBAL' },
        { keys: ['⌘', '/'], desc: 'Show keyboard shortcuts', context: 'GLOBAL' },
        { keys: ['ESC'], desc: 'Close panel / dismiss modal', context: 'GLOBAL' },
        { keys: ['?'], desc: 'Context-sensitive help', context: 'GLOBAL' },
        { keys: ['⌘', '⇧', 'W'], desc: 'Toggle War Room mode', context: 'CRITICAL', variant: 'error' },
      ]
    },
    {
      title: '02 · NAVIGATION',
      shortcuts: [
        { keys: ['G', 'D'], desc: 'Go to SOC Dashboard', context: 'NAV' },
        { keys: ['G', 'A'], desc: 'Go to Alert Management', context: 'NAV' },
        { keys: ['G', 'S'], desc: 'Go to SIEM Search', context: 'NAV' },
        { keys: ['G', 'F'], desc: 'Go to Fleet Management', context: 'NAV' },
        { keys: ['G', 'P'], desc: 'Go to SOAR Playbooks', context: 'NAV' },
        { keys: ['G', 'V'], desc: 'Go to Evidence Vault', context: 'NAV' },
        { keys: ['J', 'K'], desc: 'Move down / up in list', context: 'NAV' },
        { keys: ['Enter'], desc: 'Select focused item', context: 'NAV' },
      ]
    },
    {
      title: '03 · OPERATOR ACTIONS',
      shortcuts: [
        { keys: ['⌃', '⇧', 'I'], desc: 'Isolate current host', context: 'CRITICAL', variant: 'error' },
        { keys: ['⌃', '⇧', 'E'], desc: 'Capture evidence → Vault', context: 'OPERATOR', variant: 'accent' },
        { keys: ['⌃', '⇧', 'F'], desc: 'SIEM filtered to context', context: 'SIEM', variant: 'success' },
        { keys: ['⌃', '⇧', 'T'], desc: 'Auto-assemble timeline', context: 'OPERATOR', variant: 'accent' },
        { keys: ['⌃', '⇧', 'P'], desc: 'Trigger SOAR playbook', context: 'OPERATOR', variant: 'accent' },
        { keys: ['⌃', '⇧', 'S'], desc: 'SSH to selected host', context: 'OPERATOR', variant: 'accent' },
      ]
    },
    {
      title: '04 · ALERT MANAGEMENT',
      shortcuts: [
        { keys: ['A'], desc: 'Acknowledge alert', context: 'ALERT' },
        { keys: ['F'], desc: 'Mark false positive', context: 'ALERT' },
        { keys: ['S'], desc: 'Suppress alert', context: 'ALERT' },
        { keys: ['1', '-', '4'], desc: 'Filter by severity', context: 'ALERT' },
        { keys: ['⌃', 'Enter'], desc: 'Open detail drawer', context: 'ALERT' },
      ]
    }
  ];
</script>

<PageLayout title="Keyboard Interaction Map" subtitle="Tactical command reference and shortcut orchestration">
  {#snippet toolbar()}
    <div class="flex items-center gap-2">
      <Badge variant="accent" class="font-mono text-[9px] uppercase tracking-widest">Shortcut Mode Active</Badge>
    </div>
  {/snippet}

  <div class="flex flex-col h-full gap-8">
    <div class="grid grid-cols-1 lg:grid-cols-2 gap-8">
        {#each sections as section}
            <div class="space-y-3">
                <div class="text-[8px] font-mono text-text-muted uppercase tracking-[0.2em] border-b border-border-primary pb-1.5 mb-2">
                    {section.title}
                </div>
                <div class="grid grid-cols-1 gap-1.5">
                    {#each section.shortcuts as shortcut}
                        <div class="flex items-center gap-3 p-2 bg-surface-2 border border-border-primary rounded-sm hover:bg-surface-3 transition-colors cursor-pointer group">
                            <div class="flex gap-1 min-w-[100px]">
                                {#each shortcut.keys as key}
                                    {#if key === '-' || key === 'then'}
                                        <span class="text-[9px] font-mono text-text-muted self-center mx-0.5">{key}</span>
                                    {:else}
                                        <kbd class="px-1.5 py-0.5 bg-surface-3 border border-border-hover rounded-sm text-[10px] font-mono font-bold text-text-heading shadow-xs">
                                            {key}
                                        </kbd>
                                    {/if}
                                {/each}
                            </div>
                            <div class="flex-1 text-[10px] text-text-secondary group-hover:text-text-heading transition-colors">
                                {shortcut.desc}
                            </div>
                            <div class="px-1.5 py-0.5 rounded-xs text-[8px] font-mono font-bold uppercase tracking-tighter 
                                {shortcut.variant === 'error' ? 'bg-error/10 text-error' : 
                                 shortcut.variant === 'accent' ? 'bg-accent/10 text-accent' :
                                 shortcut.variant === 'success' ? 'bg-success/10 text-success' :
                                 'bg-surface-3 text-text-muted'}">
                                {shortcut.context}
                            </div>
                        </div>
                    {/each}
                </div>
            </div>
        {/each}
    </div>

    <!-- VIM MODE INFO -->
    <div class="bg-surface-2 border border-border-primary rounded-md p-4 space-y-3">
        <div class="flex items-center gap-2 text-accent">
            <Command size={14} />
            <span class="text-[10px] font-mono font-bold uppercase tracking-widest">Vim-Style Navigation (Table Mode)</span>
        </div>
        <div class="grid grid-cols-4 gap-4">
            <div class="bg-background/50 p-3 border border-border-primary rounded-sm text-center">
                <div class="text-xl font-mono font-bold text-accent mb-1">H/L</div>
                <div class="text-[9px] text-text-muted uppercase">Horizontal Scroll</div>
            </div>
            <div class="bg-background/50 p-3 border border-border-primary rounded-sm text-center">
                <div class="text-xl font-mono font-bold text-accent mb-1">J/K</div>
                <div class="text-[9px] text-text-muted uppercase">Vertical Move</div>
            </div>
            <div class="bg-background/50 p-3 border border-border-primary rounded-sm text-center">
                <div class="text-xl font-mono font-bold text-accent mb-1">V</div>
                <div class="text-[9px] text-text-muted uppercase">Visual Select</div>
            </div>
            <div class="bg-background/50 p-3 border border-border-primary rounded-sm text-center">
                <div class="text-xl font-mono font-bold text-accent mb-1">/</div>
                <div class="text-[9px] text-text-muted uppercase">In-Page Filter</div>
            </div>
        </div>
        <div class="text-[9px] font-mono text-text-muted text-center pt-2">
            Table mode activates automatically when focus enters any data table. Press <span class="text-text-secondary font-bold">ESC</span> to return to page navigation.
        </div>
    </div>
  </div>
</PageLayout>
