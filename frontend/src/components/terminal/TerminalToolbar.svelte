<!--
  OBLIVRA — Terminal Toolbar (Svelte 5)
  Action rail for active PTY sessions.
-->
<script lang="ts">
  import { appStore } from '@lib/stores/app.svelte';
  import { Button } from '@components/ui';
  import { Settings } from 'lucide-svelte';

  function toggleWarMode() {
    appStore.notify('War Mode Engaged', 'error', 'Applying high-performance compute limits and hardened shell isolation.');
  }

  function connectToLocal() {
    appStore.connectToLocal();
  }
</script>

<div class="flex items-center gap-2">
  <div class="flex items-center gap-1 bg-surface-2 rounded-sm border border-border-primary p-0.5">
    <Button variant="ghost" size="xs" onclick={connectToLocal} title="Start new local shell">
      <span class="text-[10px]">+ Local</span>
    </Button>
    <Button variant="ghost" size="xs" title="SSH Bookmark Manager">
      <span class="text-[10px]">Bookmarks</span>
    </Button>
  </div>

  <div class="h-4 w-px bg-border-primary mx-1"></div>

  <div class="flex items-center gap-1">
    <Button 
      variant="ghost" 
      size="xs" 
      class="text-error hover:bg-error/10" 
      onclick={toggleWarMode}
      title="Engage maximum security isolation"
    >
      <span class="text-[10px] font-bold">⚔ WAR MODE</span>
    </Button>
    
    <Button variant="ghost" size="xs" title="Broadcast command to all panes">
      <span class="text-[10px]">λ Multi</span>
    </Button>
  </div>

  <div class="flex-1"></div>

  <div class="flex items-center gap-2">
    <div class="flex items-center gap-1 text-[9px] font-mono text-text-muted bg-black/20 px-2 py-0.5 rounded-full">
      <span class="w-1.5 h-1.5 rounded-full bg-status-online animate-pulse"></span>
      <span>SECURE ENGINE ACTIVE</span>
    </div>
    
    <Button variant="ghost" size="xs" title="Session Settings">
      <Settings size={12} class="opacity-60" />
    </Button>
  </div>
</div>
