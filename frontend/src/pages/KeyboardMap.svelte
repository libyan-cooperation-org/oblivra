<!--
  Keyboard Map — auto-generated from `lib/shortcuts.ts`.

  This file used to maintain its own copy of the shortcut list, which
  inevitably drifted from App.svelte's actual handlers. As of Phase 32
  it reads directly from the registry — adding a new shortcut means
  editing `lib/shortcuts.ts` and the help page updates automatically.
  See UIUX_IMPROVEMENTS.md P2 #14.
-->
<script lang="ts">
  import { PageLayout, PopOutButton, Badge } from '@components/ui';
  import { Keyboard, Lock } from 'lucide-svelte';
  import { groupedShortcuts, type ShortcutDef } from '@lib/shortcuts';
  import { appStore } from '@lib/stores/app.svelte';

  const groups = groupedShortcuts();

  /**
   * `requires` predicates against the active profile rules. We render
   * gated shortcuts but disable+badge them when their predicate is
   * false, so operators understand that the shortcut exists but is
   * not currently active under their profile.
   */
  function gateActive(s: ShortcutDef): boolean {
    if (!s.requires) return true;
    if (s.requires === 'vimLeader') return appStore.profileRules.vimLeader;
    if (s.requires === 'paletteFront') return appStore.profileRules.paletteFront;
    if (s.requires === 'tenantSwitcherBar') return appStore.profileRules.tenantChrome === 'switcher-bar';
    return true;
  }
</script>

<PageLayout
  title="Keyboard Shortcuts"
  subtitle={`Auto-generated from lib/shortcuts.ts · Active profile: ${appStore.profile}`}
>
  {#snippet toolbar()}
    <PopOutButton route="/shortcuts" title="Shortcuts" />
  {/snippet}

  <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
    {#each Object.entries(groups) as [title, items]}
      <div class="bg-surface-1 border border-border-primary rounded-md p-4">
        <div class="flex items-center gap-2 mb-3">
          <Keyboard size={14} class="text-accent" />
          <span class="text-[var(--fs-label)] uppercase tracking-widest font-bold">{title}</span>
          <span class="ml-auto text-[var(--fs-micro)] font-mono text-text-muted">{items.length}</span>
        </div>
        <table class="w-full text-[var(--fs-label)]">
          <tbody>
            {#each items as it}
              {@const active = gateActive(it)}
              <tr class="border-b border-border-primary last:border-b-0 {active ? '' : 'opacity-50'}">
                <td class="py-1.5 pr-3 align-top whitespace-nowrap">
                  <kbd class="bg-surface-2 border border-border-primary rounded px-1.5 py-0.5 font-mono text-[var(--fs-micro)]">{it.keys}</kbd>
                </td>
                <td class="py-1.5 text-text-secondary">
                  {it.label}
                </td>
                <td class="py-1.5 pl-2 align-top text-right">
                  {#if !active && it.requires}
                    <span class="inline-flex items-center gap-0.5 text-[var(--fs-micro)] font-mono text-text-muted" title="Disabled by Operator Profile — change in Settings">
                      <Lock size={9} />
                      profile
                    </span>
                  {:else if it.scope !== 'global'}
                    <Badge variant="muted" size="xs">{it.scope}</Badge>
                  {/if}
                </td>
              </tr>
            {/each}
          </tbody>
        </table>
      </div>
    {/each}
  </div>
</PageLayout>
