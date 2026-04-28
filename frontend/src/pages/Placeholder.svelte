<!--
  OBLIVRA — Placeholder Page (Svelte 5)
  Rendered for routes whose SolidJS components haven't been migrated yet.
  Shows the page name, route path, and migration context.
-->
<script lang="ts">
  import { APP_CONTEXT, IS_DESKTOP, IS_HYBRID, isRouteAvailable, routeUnavailableReason } from '@lib/context';

  interface Props {
    params?: Record<string, string>;
  }

  let { params = {} }: Props = $props();

  // Extract route info from the hash
  const path = $derived(window.location.hash.replace('#', '') || '/');
  const pageName = $derived(
    path.split('/').filter(Boolean)[0]?.replace(/-/g, ' ').replace(/\b\w/g, c => c.toUpperCase()) || 'Unknown'
  );
  const available = $derived(isRouteAvailable(path));
  const unavailableReason = $derived(routeUnavailableReason(path));

  const contextLabel: Record<string, string> = {
    desktop: 'Desktop',
    browser: 'Browser',
    hybrid:  'Hybrid',
  };
</script>

{#if !available && unavailableReason}
  <!-- Unavailable screen -->
  <div class="flex flex-col items-center justify-center h-full gap-5 p-12 bg-surface-0 text-center">
    <div class="w-12 h-12 rounded-lg bg-warning/10 border border-warning/30 flex items-center justify-center text-[22px]">
      ⊘
    </div>
    <div class="font-sans text-[15px] font-bold text-text-heading">
      Not available in {contextLabel[APP_CONTEXT]} mode
    </div>
    <div class="font-sans text-[13px] text-text-muted max-w-[480px] leading-relaxed">
      {unavailableReason}
    </div>
    {#if IS_DESKTOP}
      <div class="font-mono text-[11px] text-text-muted bg-surface-2 border border-border-primary px-4 py-2.5 rounded-sm max-w-[520px]">
        To access server-only features, configure a remote OBLIVRA server in
        <span class="text-accent">Settings → Server Connection</span>
        to enable Hybrid mode.
      </div>
    {:else if !IS_HYBRID}
      <div class="font-mono text-[11px] text-text-muted bg-surface-2 border border-border-primary px-4 py-2.5 rounded-sm max-w-[520px]">
        Download and run the
        <span class="text-accent">Oblivra Desktop binary</span>
        for local PTY terminal, OS keychain, and direct SFTP access.
      </div>
    {/if}
    <div class="flex items-center gap-1.5 font-mono text-[9px] font-bold uppercase tracking-widest text-text-muted opacity-50">
      <span class="w-1.5 h-1.5 rounded-full {IS_DESKTOP ? 'bg-status-online' : IS_HYBRID ? 'bg-warning' : 'bg-accent'} inline-block"></span>
      Running in {contextLabel[APP_CONTEXT]} mode · Route: {path}
    </div>
  </div>
{:else}
  <!-- Migration placeholder -->
  <div class="flex flex-col items-center justify-center h-full gap-4 p-12 bg-surface-0 text-center">
    <div class="w-12 h-12 rounded-lg bg-accent/10 border border-accent/30 flex items-center justify-center text-[22px]">
      🔄
    </div>
    <div class="font-sans text-[15px] font-bold text-text-heading">
      {pageName}
    </div>
    <div class="font-sans text-[13px] text-text-muted max-w-[400px] leading-relaxed">
      This page is being migrated to Svelte 5. The SolidJS version
      is still available on the <code class="text-accent text-[11px] font-mono">main</code> branch.
    </div>
    <div class="font-mono text-[9px] text-text-muted opacity-40 uppercase tracking-widest">
      Route: {path}
    </div>
  </div>
{/if}
