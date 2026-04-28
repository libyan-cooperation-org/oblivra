<!--
  OBLIVRA — Loading Screen (Svelte 5)
  Displayed while the Wails bridge initializes.
-->
<script lang="ts">
  // No props needed — pure presentational component
</script>

<div class="flex flex-col items-center justify-center h-screen gap-4 bg-surface-0">
  <div class="w-7 h-7 border-2 border-border-secondary border-t-accent rounded-full animate-spin"></div>
  <p class="text-text-muted text-xs font-sans">Initializing OblivraShell…</p>
</div>
