<!--
  OBLIVRA — Terminal Page (Svelte 5)
  Main interface for active shell sessions.
-->
<script lang="ts">
  import { appStore } from '@lib/stores/app.svelte';
  import { KPI, Badge, PageLayout, Button, EmptyState } from '@components/ui';
  import XTerm from '@components/terminal/XTerm.svelte';
  import TerminalTabs from '@components/terminal/TerminalTabs.svelte';
  import TerminalToolbar from '@components/terminal/TerminalToolbar.svelte';
  import OperatorBanner from '@components/terminal/OperatorBanner.svelte';
  import SessionRestoreBanner from '@components/terminal/SessionRestoreBanner.svelte';

  const activeSession = $derived(
    appStore.sessions.find(s => s.id === appStore.activeSessionId) || appStore.sessions[0]
  );

  // Best-effort host derivation from the active session — used by
  // OperatorBanner to filter SIEM alerts to the host the operator is
  // currently looking at. Schemas vary across stores; prefer hostname
  // then any host-like field.
  const activeHost = $derived(
    (activeSession as any)?.hostname ??
    (activeSession as any)?.host ??
    (activeSession as any)?.host_id ??
    ''
  );
</script>

<PageLayout title="Terminal" subtitle="Secure PTY orchestration">
  {#snippet toolbar()}
    <TerminalToolbar />
  {/snippet}

  {#if appStore.sessions.length > 0}
    <div class="flex flex-col h-full gap-2">
      <!-- Restore-from-last-session prompt (visible only on cold start) -->
      <SessionRestoreBanner />

      <!-- SIEM alerts for the currently-active SSH host -->
      {#if activeHost}
        <OperatorBanner host={activeHost} />
      {/if}

      <!-- Session Tabs -->
      <TerminalTabs />

      <!-- Terminal Area -->
      <div class="flex-1 min-h-0 bg-[#0a0b10] rounded-sm border border-border-primary overflow-hidden">
        {#each appStore.sessions as session (session.id)}
          <XTerm 
            sessionId={session.id} 
            isActive={appStore.activeSessionId === session.id} 
          />
        {/each}
      </div>

      <!-- Terminal Footer / Info -->
      <div class="flex items-center justify-between px-2 py-1 bg-surface-1 border border-border-primary rounded-sm text-[9px] font-mono text-text-muted">
        <div class="flex gap-4">
          <span>ID: {activeSession?.id.slice(0, 8)}</span>
          <span>PTY: /dev/pts/1</span>
          <span>BUFFER: 5000 lines</span>
        </div>
        <div class="flex gap-4">
          <span class="text-success">● ENCRYPTED</span>
          <span>LATENCY: 12ms</span>
        </div>
      </div>
    </div>
  {:else}
    <EmptyState 
      title="No active sessions" 
      description="Connect to a remote host via the sidebar or start a local shell to begin."
      icon="⌨"
    >
      {#snippet action()}
        <Button variant="primary" onclick={() => appStore.connectToLocal()}>Start Local Shell</Button>
      {/snippet}
    </EmptyState>
  {/if}
</PageLayout>
