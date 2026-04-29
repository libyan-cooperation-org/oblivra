<!--
  OBLIVRA — SearchBar (Svelte 5)
  Search input with icon, keyboard shortcut hint, and clear button.
-->
<script lang="ts">
  interface Props {
    value?: string;
    placeholder?: string;
    shortcut?: string;
    compact?: boolean;
    oninput?: (value: string) => void;
  }

  let {
    value = $bindable(''),
    placeholder = 'Search…',
    shortcut,
    compact = false,
    oninput,
  }: Props = $props();

  function handleInput(e: Event) {
    value = (e.target as HTMLInputElement).value;
    oninput?.(value);
  }

  function clear() {
    value = '';
    oninput?.('');
  }
</script>

<div role="search" aria-label="Search form" class="relative flex items-center">
  <!-- Search icon -->
  <span class="absolute left-2.5 text-text-muted text-[11px] pointer-events-none">⌕</span>

  <input
    type="text"
    {value}
    {placeholder}
    oninput={handleInput}
    aria-label="Search"
    class="w-full bg-surface-0 border border-border-primary rounded-sm text-text-primary font-sans outline-none transition-all duration-fast
      placeholder:text-text-muted
      focus:border-accent focus:shadow-glow
      {compact ? 'text-[11px] h-6 pl-7 pr-12' : 'text-xs h-7 pl-7 pr-14'}"
  />

  <!-- Right side: clear + keyboard shortcut -->
  <div class="absolute right-2 flex items-center gap-1.5">
    {#if value}
      <button
        class="w-4 h-4 flex items-center justify-center text-text-muted hover:text-text-secondary text-[10px] cursor-pointer bg-transparent border-none transition-colors duration-fast"
        onclick={clear}
        aria-label="Clear"
      >✕</button>
    {/if}
    {#if shortcut && !value}
      <span class="text-[9px] font-mono text-text-muted bg-surface-2 px-1 py-px rounded-xs border border-border-primary">
        {shortcut}
      </span>
    {/if}
  </div>
</div>
