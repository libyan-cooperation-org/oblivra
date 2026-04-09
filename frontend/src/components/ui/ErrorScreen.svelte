<!--
  OBLIVRA — Error Screen (Svelte 5)
  Displayed when the Wails bridge fails to initialize.
-->
<script lang="ts">
  interface Props {
    message: string;
  }

  let { message }: Props = $props();
</script>

<div class="flex flex-col items-center justify-center h-screen gap-3 text-error">
  <div class="text-2xl">⚠</div>
  <div class="text-sm font-semibold">Initialization Failed</div>
  <div class="text-xs text-text-muted max-w-md text-center font-mono leading-relaxed">
    {message}
  </div>
</div>
