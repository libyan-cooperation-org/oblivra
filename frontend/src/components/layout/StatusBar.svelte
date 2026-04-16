<!--
  OBLIVRA — StatusBar (Svelte 5)
  Bottom status bar showing session info, vault status, health grade, and clock.
-->
<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { appStore } from '@lib/stores/app.svelte';
  import { APP_CONTEXT } from '@lib/context';
  import { subscribe } from '@lib/bridge';

  interface Props {
    onToggleTransfers?: () => void;
  }

  let { onToggleTransfers }: Props = $props();

  let time = $state('');
  let diagGrade = $state<string | null>(null);
  let ingestEPS = $state(0);

  const activeCount = $derived(appStore.sessions.filter(s => s.status === 'active').length);
  const transferCount = $derived(appStore.transfers.filter(t => t.status === 'active' || t.status === 'pending').length);

  const activeHost = $derived.by(() => {
    if (!appStore.activeSessionId) return null;
    const session = appStore.sessions.find(s => s.id === appStore.activeSessionId);
    if (!session) return null;
    return { id: session.hostId, label: session.hostLabel || session.hostId };
  });

  const contextLabel: Record<string, string> = {
    desktop: 'DESKTOP',
    browser: 'BROWSER',
    hybrid:  'HYBRID',
  };

  const contextColor: Record<string, string> = {
    desktop: 'bg-status-online',
    browser: 'bg-accent',
    hybrid:  'bg-warning',
  };

  let timer: ReturnType<typeof setInterval>;
  let unsubDiag: (() => void) | undefined;

  onMount(() => {
    // Clock
    const updateTime = () => {
      time = new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' });
    };
    updateTime();
    timer = setInterval(updateTime, 30_000);

    // Diagnostics health grade & ingest metrics
    unsubDiag = subscribe('diagnostics:snapshot', (data: any) => {
      if (data?.health_grade) diagGrade = data.health_grade;
      if (data?.ingest) ingestEPS = data.ingest.current_eps;
    });
  });

  onDestroy(() => {
    clearInterval(timer);
    unsubDiag?.();
  });
</script>

<footer class="flex items-center justify-between h-6 bg-surface-1 border-t border-border-primary px-3 text-[10px] font-mono text-text-muted font-medium tracking-wide shrink-0 z-50">
  <!-- Left cluster -->
  <div class="flex items-center gap-2.5">
    <!-- Context badge -->
    <div class="flex items-center gap-1.5">
      <span class="w-1.5 h-1.5 rounded-full {contextColor[APP_CONTEXT]}"></span>
      <span class="text-[8px] font-bold uppercase tracking-widest opacity-60">{contextLabel[APP_CONTEXT]}</span>
    </div>

    <span class="text-border-secondary font-mono">|</span>

    <!-- Active session -->
    <span
      class="inline-block w-[5px] h-[5px] rounded-full shrink-0"
      style="background: {activeHost ? 'var(--status-online)' : 'var(--text-muted)'}; box-shadow: {activeHost ? '0 0 6px var(--status-online)' : 'none'};"
    ></span>
    <span class="text-{activeHost ? 'text-secondary' : 'text-muted'}">
      {activeHost ? activeHost.label : 'no active session'}
    </span>
    
    <span class="text-border-secondary font-mono">|</span>

    <!-- Ingest EPS -->
    <div class="flex items-center gap-1.5" title="Ingestion events per second">
      <span class="text-status-online opacity-80 animate-pulse">⚡</span>
      <span class={ingestEPS > 0 ? 'text-text-secondary' : 'text-text-muted opacity-40'}>{ingestEPS} EPS</span>
    </div>
  </div>

  <!-- Right cluster -->
  <div class="flex items-center gap-2.5">
    <!-- Plugin status icons -->
    {#each appStore.pluginStatusIcons as icon}
      <span title="{icon.plugin_id}: {icon.tooltip}" class="cursor-help opacity-70">{icon.icon}</span>
    {/each}

    <!-- Transfer count -->
    {#if transferCount > 0}
      <button
        class="text-accent cursor-pointer bg-transparent border-none font-mono text-[10px] p-0"
        onclick={() => onToggleTransfers?.()}
      >⇅ {transferCount} xfer</button>
      <span class="text-border-secondary font-mono">|</span>
    {/if}

    <!-- Session / host counts -->
    <span>
      <span class="text-text-muted">{activeCount}</span>
      <span class="opacity-35"> sess · </span>
      <span class="text-text-muted">{appStore.hosts.length}</span>
      <span class="opacity-35"> hosts</span>
    </span>

    <span class="text-border-secondary font-mono">|</span>

    <!-- Health grade -->
    {#if diagGrade}
      <span
        class="font-mono text-[9px] font-extrabold tracking-wider cursor-pointer hover:opacity-70"
        style="color: {diagGrade === 'A' ? 'var(--status-online)' : diagGrade === 'B' ? '#d29922' : diagGrade === 'C' ? '#f0883e' : '#f85149'};"
        title="Platform health"
      >● {diagGrade}</span>
      <span class="text-border-secondary font-mono">|</span>
    {/if}

    <!-- Vault status -->
    {#if appStore.vaultUnlocked}
      <span class="text-accent">● secure</span>
    {:else}
      <span class="text-text-muted opacity-40">locked</span>
    {/if}

    <span class="text-border-secondary font-mono">|</span>

    <!-- Clock -->
    <span class="text-text-secondary font-semibold tracking-wider">{time}</span>
  </div>
</footer>
