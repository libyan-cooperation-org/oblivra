<!-- OBLIVRA — StatusBar v2 — null-safe EPS, UTC clock, unified tokens -->
<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { appStore } from '@lib/stores/app.svelte';
  // shellStore was removed Phase 33 alongside the shell subsystem.
  // Active-session + active-host indicators are stubbed until the new
  // shell ships and registers a replacement store.
  import { APP_CONTEXT } from '@lib/context';
  import { subscribe } from '@lib/bridge';

  interface Props { onToggleTransfers?: () => void; }
  let { onToggleTransfers }: Props = $props();

  let time      = $state('');
  let diagGrade = $state<string | null>(null);
  let ingestEPS = $state<number | null>(null); // null = not yet received
  let loaded    = $state(false);

  // Shell-derived indicators are zeroed pending the new shell subsystem.
  // The status bar still renders — it just shows 0 sessions / no active
  // host until the replacement registers a session-count store.
  const activeCount   = $derived(0);
  const transferCount = $derived(appStore.transfers.filter((t: any) => t.status === 'active' || t.status === 'pending').length);
  const activeHost    = $derived<{ label: string } | null>(null);

  // Display strings — dashes while loading, never raw zeros on first paint
  const epsDisplay    = $derived(!loaded ? '—' : (ingestEPS ?? 0).toLocaleString());
  const gradeDisplay  = $derived(diagGrade ?? '—');

  const ctxDotStyle: Record<string, string> = {
    desktop: 'background:var(--pu2)',
    browser: 'background:var(--ac2)',
    hybrid:  'background:var(--md2)',
  };
  const ctxLabel: Record<string, string> = {
    desktop: 'DESKTOP', browser: 'BROWSER', hybrid: 'HYBRID',
  };
  const gradeColor = $derived.by(() => {
    if (!diagGrade) return 'var(--tx3)';
    const g = diagGrade[0];
    if (g === 'A') return 'var(--ok2)';
    if (g === 'B') return 'var(--md2)';
    if (g === 'C') return 'var(--hi2)';
    return 'var(--cr2)';
  });

  let timer: ReturnType<typeof setInterval>;
  let unsubDiag: (() => void) | undefined;

  onMount(() => {
    const tick = () => {
      const d = new Date();
      const pad = (n: number) => String(n).padStart(2, '0');
      time = `${pad(d.getUTCHours())}:${pad(d.getUTCMinutes())}:${pad(d.getUTCSeconds())}Z`;
    };
    tick();
    timer = setInterval(tick, 1000);

    unsubDiag = subscribe('diagnostics:snapshot', (data: any) => {
      if (data?.health_grade) diagGrade = data.health_grade;
      if (data?.ingest?.current_eps !== undefined) ingestEPS = data.ingest.current_eps;
      loaded = true;
    });

    // Fallback: show dashes → dashes after 2 s if no data
    setTimeout(() => { loaded = true; }, 2000);
  });

  onDestroy(() => {
    clearInterval(timer);
    unsubDiag?.();
  });
</script>

<footer
  class="flex items-center justify-between h-6 bg-surface-1 border-t border-border-primary px-3 shrink-0 z-50"
  style="font-family:var(--mn); font-size:9px; color:var(--tx2);"
>
  <!-- Left -->
  <div class="flex items-center gap-2">
    <!-- Context -->
    <div class="flex items-center gap-1.5">
      <span class="w-1.5 h-1.5 rounded-full shrink-0" style="{ctxDotStyle[APP_CONTEXT]}"></span>
      <span style="font-size:8px; font-weight:700; letter-spacing:0.15em; color:var(--tx3); opacity:0.65;">
        {ctxLabel[APP_CONTEXT]}
      </span>
    </div>
    <span style="color:var(--b2);">|</span>
    <!-- Session -->
    <span class="w-[5px] h-[5px] rounded-full shrink-0"
      style="background:{activeHost ? 'var(--ok2)' : 'var(--tx3)'}; {activeHost ? 'box-shadow:0 0 4px var(--ok2)' : ''};">
    </span>
    <span style="color:{activeHost ? 'var(--tx2)' : 'var(--tx3)'};">
      {activeHost ? activeHost.label : 'no active session'}
    </span>
    <span style="color:var(--b2);">|</span>
    <!-- EPS -->
    <span style="color:var(--ok2); opacity:0.8;">⚡</span>
    <span style="color:{loaded && ingestEPS === 0 ? 'var(--tx3)' : 'var(--tx2)'};">
      {epsDisplay} EPS
    </span>
  </div>

  <!-- Right -->
  <div class="flex items-center gap-2">
    <!-- Plugin icons -->
    {#each appStore.pluginStatusIcons as icon}
      <span title="{icon.plugin_id}: {icon.tooltip}" class="cursor-help" style="opacity:0.65;">{icon.icon}</span>
    {/each}
    <!-- Transfers -->
    {#if transferCount > 0}
      <button
        class="bg-transparent border-none font-mono cursor-pointer p-0"
        style="font-size:9px; color:var(--ac2);"
        onclick={() => onToggleTransfers?.()}
      >⇅ {transferCount} xfer</button>
      <span style="color:var(--b2);">|</span>
    {/if}
    <!-- Counts -->
    <span>
      <span style="color:var(--tx2);">{activeCount}</span>
      <span style="opacity:0.35;"> sess · </span>
      <span style="color:var(--tx2);">{appStore.hosts.length}</span>
      <span style="opacity:0.35;"> hosts</span>
    </span>
    <span style="color:var(--b2);">|</span>
    <!-- Health grade -->
    {#if diagGrade}
      <span
        class="font-mono font-bold cursor-pointer hover:opacity-70"
        style="font-size:9px; color:{gradeColor}; letter-spacing:0.08em;"
        title="Platform health grade"
      >● {gradeDisplay}</span>
      <span style="color:var(--b2);">|</span>
    {/if}
    <!-- Vault -->
    {#if appStore.vaultUnlocked}
      <span style="color:var(--ok2);">⊠ SEALED</span>
    {:else}
      <span style="color:var(--tx3); opacity:0.45;">⊠ locked</span>
    {/if}
    <span style="color:var(--b2);">|</span>
    <!-- UTC Clock -->
    <span style="color:var(--tx2); font-weight:600; letter-spacing:0.06em;">{time || '—'}</span>
  </div>
</footer>
