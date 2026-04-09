<!--
  OBLIVRA — Dashboard (Svelte 5)
  Placeholder/skeleton — will be fully migrated in Phase 6.
  For now provides a working landing page to validate the build.
-->
<script lang="ts">
  import { appStore } from '@lib/stores/app.svelte';
  import { APP_CONTEXT, IS_DESKTOP, IS_HYBRID } from '@lib/context';

  const contextLabel: Record<string, string> = {
    desktop: 'Desktop',
    browser: 'Browser',
    hybrid:  'Hybrid',
  };

  const contextColor: Record<string, string> = {
    desktop: 'bg-status-online',
    browser: 'bg-accent',
    hybrid:  'bg-warning',
  };
</script>

<div class="flex flex-col h-full overflow-auto bg-surface-0 p-6">
  <!-- Header -->
  <div class="flex items-center justify-between mb-6 pb-4 border-b border-border-primary">
    <div>
      <h1 class="text-xl font-bold text-text-heading font-[var(--font-ui)]">
        OBLIVRA Dashboard
      </h1>
      <p class="text-xs text-text-muted mt-1 font-mono">
        Sovereign Security Operations Platform
      </p>
    </div>
    <div class="flex items-center gap-2">
      <span class="w-2 h-2 rounded-full {contextColor[APP_CONTEXT]}"></span>
      <span class="text-[9px] font-bold uppercase tracking-widest text-text-muted font-mono">
        {contextLabel[APP_CONTEXT]} Mode
      </span>
    </div>
  </div>

  <!-- KPI Cards -->
  <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-6">
    <div class="bg-surface-1 border border-border-primary rounded-md p-4">
      <div class="text-[10px] font-bold uppercase tracking-wider text-text-muted mb-2">Hosts</div>
      <div class="text-2xl font-bold text-text-heading">{appStore.hosts.length}</div>
    </div>
    <div class="bg-surface-1 border border-border-primary rounded-md p-4">
      <div class="text-[10px] font-bold uppercase tracking-wider text-text-muted mb-2">Active Sessions</div>
      <div class="text-2xl font-bold text-text-heading">{appStore.sessions.length}</div>
    </div>
    <div class="bg-surface-1 border border-border-primary rounded-md p-4">
      <div class="text-[10px] font-bold uppercase tracking-wider text-text-muted mb-2">Vault Status</div>
      <div class="text-2xl font-bold {appStore.vaultUnlocked ? 'text-success' : 'text-error'}">
        {appStore.vaultUnlocked ? 'Unlocked' : 'Locked'}
      </div>
    </div>
    <div class="bg-surface-1 border border-border-primary rounded-md p-4">
      <div class="text-[10px] font-bold uppercase tracking-wider text-text-muted mb-2">Notifications</div>
      <div class="text-2xl font-bold text-text-heading">{appStore.notifications.length}</div>
    </div>
  </div>

  <!-- Migration Notice -->
  <div class="bg-surface-1 border border-accent/30 rounded-md p-6 text-center">
    <div class="text-accent text-lg mb-2">🔄</div>
    <div class="text-sm font-semibold text-text-heading mb-1">Svelte 5 Migration In Progress</div>
    <div class="text-xs text-text-muted max-w-lg mx-auto leading-relaxed">
      This dashboard is running on the new Svelte 5 + Tailwind CSS stack.
      Components are being migrated incrementally from SolidJS.
      The full navigation and all pages will be wired in as migration progresses.
    </div>
  </div>
</div>
