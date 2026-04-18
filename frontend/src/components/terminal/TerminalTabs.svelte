<!--
  OBLIVRA — Terminal Tabs (Svelte 5)
  Tab orchestration for PTY sessions.
-->
<script lang="ts">
  import { appStore } from '@lib/stores/app.svelte';

  function closeSession(id: string) {
    appStore.removeSession(id);
  }
</script>

<div class="flex items-center gap-1 border-b border-border-primary shrink-0 overflow-x-auto bg-surface-1">
  {#each appStore.sessions as session}
    <div
      role="tab"
      tabindex="0"
      class="group flex items-center gap-2 px-3 py-1.5 min-w-[120px] max-w-[200px] border-b-2 transition-all duration-fast cursor-pointer
        {appStore.activeSessionId === session.id 
          ? 'border-accent bg-accent/5 text-text-primary' 
          : 'border-transparent text-text-muted hover:bg-surface-2 hover:text-text-secondary'}"
      onclick={() => appStore.setActiveSession(session.id)}
      onkeydown={(e) => e.key === 'Enter' && appStore.setActiveSession(session.id)}
    >
      <span class="text-[10px] shrink-0">🖥</span>
      <span class="text-[11px] font-bold truncate flex-1 text-left">{session.hostLabel || 'Local Shell'}</span>
      <button 
        class="opacity-0 group-hover:opacity-100 px-1 text-text-muted hover:text-error transition-opacity cursor-pointer border-none bg-transparent"
        onclick={(e) => { e.stopPropagation(); closeSession(session.id); }}
        title="Close session"
      >
        ✕
      </button>
    </div>
  {/each}
</div>
