<!--
  ShellPage — full-bleed mount for the new Shell workspace.

  Pure shell. No SOC chrome inside the page itself; the standard AppLayout
  already wraps it with sidebar + status bar. The Workspace component
  fills the available height, owns its tabs strip + pane tree + bottom
  shell dock.
-->
<script lang="ts">
  import Workspace from '@components/shell/Workspace.svelte';
</script>

<div class="h-full w-full">
  <Workspace />
</div>
