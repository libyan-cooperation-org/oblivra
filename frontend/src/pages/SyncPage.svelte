<!--
  OBLIVRA — Sync & Cloud Edge (Svelte 5)
  Orchestration of encrypted cross-device synchronization and peer-to-peer relay.
-->
<script lang="ts">
  import { KPI, Badge, PageLayout, Button, Spinner } from '@components/ui';

  let syncing = $state(false);
  let lastSync = $state('2m ago');

  function triggerSync() {
    syncing = true;
    setTimeout(() => {
      syncing = false;
      lastSync = 'Just now';
    }, 2000);
  }
</script>

<PageLayout title="Workspace Sync" subtitle="End-to-end encrypted synchronization across your fleet and devices">
  <div class="flex flex-col h-full gap-8 max-w-4xl">
    
    <!-- Hero Status -->
    <div class="bg-surface-1 border border-border-primary rounded-lg p-8 flex flex-col items-center text-center gap-6 relative overflow-hidden">
      <!-- Background pulse -->
      <div class="absolute inset-0 bg-accent/5 animate-pulse"></div>
      
      <div class="relative w-20 h-20 rounded-full border-4 border-accent/20 flex items-center justify-center bg-surface-2 shadow-glow-accent/20">
        {#if syncing}
          <div class="absolute inset-0 animate-spin border-4 border-accent border-t-transparent rounded-full"></div>
          <span class="text-2xl">🔄</span>
        {:else}
          <span class="text-3xl text-accent">☁️</span>
        {/if}
      </div>

      <div class="relative">
        <h2 class="text-xl font-bold text-text-heading">Synchronized</h2>
        <p class="text-xs text-text-muted mt-1">Your configuration, vault, and snippets are up to date across 3 devices.</p>
      </div>

      <div class="relative flex gap-4">
        <Button variant="cta" size="sm" onclick={triggerSync} disabled={syncing}>
          {#if syncing} <Spinner size="sm" /> {:else} Force Sync Now {/if}
        </Button>
        <Button variant="secondary" size="sm">Manage Devices</Button>
      </div>
    </div>

    <!-- Sync Metadata -->
    <div class="grid grid-cols-1 md:grid-cols-2 gap-4">
      <KPI title="Encrypted Payload" value="1.4 MB" trend="Optimized" />
      <KPI title="Last Handshake" value={lastSync} trend="Relay: AMS" variant="success" />
    </div>

    <!-- Device List -->
    <div class="flex flex-col gap-4">
      <h3 class="text-xs font-bold uppercase tracking-widest text-text-muted">Linked Terminal Instances</h3>
      <div class="flex flex-col gap-2">
        {#each ['Macbook Pro (M3)', 'Ubuntu Workstation', 'iPhone 15 Pro'] as device}
          <div class="bg-surface-1 border border-border-primary rounded px-4 py-3 flex justify-between items-center group hover:border-accent/40 transition-colors">
            <div class="flex items-center gap-3">
              <span class="text-lg opacity-40">💻</span>
              <div class="flex flex-col">
                <span class="text-xs font-bold text-text-heading">{device}</span>
                <span class="text-[9px] text-text-muted">Last seen: {Math.floor(Math.random() * 10)}m ago</span>
              </div>
            </div>
            <Badge variant="success">Online</Badge>
          </div>
        {/each}
      </div>
    </div>
  </div>
</PageLayout>
