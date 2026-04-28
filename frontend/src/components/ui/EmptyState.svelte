<!--
  OBLIVRA — EmptyState (Svelte 5)

  Shown when a list or panel has no data.

  UIUX_IMPROVEMENTS.md P1 #8 — moved away from the U+2205 default
  glyph (renders as a broken rectangle on stripped-down Windows
  installs). Prefer a `type` prop that picks a sensible lucide icon
  for the common cases; fall back to a string `icon` when callers
  want a custom emoji/symbol (e.g. DSRConsole's "📜").
-->
<script lang="ts">
  import type { Snippet } from 'svelte';
  import { Inbox, Search, AlertCircle, type Icon } from 'lucide-svelte';

  type EmptyType = 'list' | 'search' | 'error';

  interface Props {
    /** When provided, renders this string/emoji instead of the typed icon. */
    icon?: string;
    /** Picks a default lucide icon when `icon` is not set. */
    type?: EmptyType;
    title: string;
    description?: string;
    action?: Snippet;
  }

  let { icon, type = 'list', title, description, action }: Props = $props();

  const ICONS: Record<EmptyType, typeof Icon> = {
    list:   Inbox,
    search: Search,
    error:  AlertCircle,
  };
  const TypeIcon = $derived(ICONS[type] ?? Inbox);
</script>

<div class="flex flex-col items-center justify-center py-12 px-6 text-center gap-3">
  <div class="w-12 h-12 rounded-lg bg-surface-2 border border-border-primary flex items-center justify-center text-text-muted">
    {#if icon}
      <span class="text-xl">{icon}</span>
    {:else}
      <TypeIcon size={20} aria-hidden="true" />
    {/if}
  </div>
  <div class="text-sm font-semibold text-text-heading font-sans">{title}</div>
  {#if description}
    <div class="text-xs text-text-muted max-w-sm leading-relaxed font-sans">{description}</div>
  {/if}
  {#if action}
    <div class="mt-2">
      {@render action()}
    </div>
  {/if}
</div>
