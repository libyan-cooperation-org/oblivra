<!--
  OBLIVRA — Sovereignty badge (Phase 32).

  Compact pill in the top chrome that shows the deployment posture at a
  glance. Hover or click to expand a popover with the four sub-scores.
  CISOs reviewing the platform see this immediately and ask the right
  follow-up question ("on-prem? TPM? air-gap?") instead of opening a
  ticket.
-->
<script lang="ts">
  import { onMount } from 'svelte';
  import { sovereigntyStore } from '@lib/stores/sovereignty.svelte';
  import { Lock, ShieldCheck, ShieldAlert } from 'lucide-svelte';

  let open = $state(false);

  onMount(() => {
    sovereigntyStore.start();
  });

  function close() { open = false; }

  function tierColour(tier: string | undefined): string {
    switch (tier) {
      case 'gold':   return 'border-success/40 bg-success/10 text-success';
      case 'silver': return 'border-accent/40 bg-accent/10 text-accent';
      case 'bronze': return 'border-warning/40 bg-warning/10 text-warning';
      default:       return 'border-border-primary bg-surface-2 text-text-muted';
    }
  }
</script>

<div class="relative">
  <button
    class="flex items-center gap-1 px-1.5 py-0.5 rounded-sm border font-mono text-[var(--fs-micro)] uppercase tracking-widest hover:bg-accent/5 transition-colors duration-fast {tierColour(sovereigntyStore.score?.tier)}"
    onclick={() => (open = !open)}
    aria-expanded={open}
    title="Deployment sovereignty score — click for breakdown"
  >
    <Lock size={9} />
    <span>{sovereigntyStore.score ? sovereigntyStore.score.score : '—'}</span>
    {#if sovereigntyStore.score?.tier && sovereigntyStore.score.tier !== 'unverified'}
      <span class="opacity-70">·{sovereigntyStore.score.tier}</span>
    {/if}
  </button>

  {#if open && sovereigntyStore.score}
    <!-- Click-outside trap: cover the rest of the page so any click closes -->
    <div
      class="fixed inset-0 z-[8000]"
      role="presentation"
      onclick={close}
    ></div>

    <div
      class="absolute right-0 top-full z-[8001] mt-1 w-80 bg-surface-1 border border-border-secondary rounded-md shadow-2xl overflow-hidden"
      role="dialog"
      aria-label="Sovereignty score detail"
    >
      <header class="px-4 py-3 border-b border-border-primary flex items-center justify-between">
        <div class="flex items-center gap-2">
          <Lock size={12} class="text-accent" />
          <span class="text-[var(--fs-label)] font-bold uppercase tracking-widest text-text-heading">Sovereignty</span>
        </div>
        <span class="font-mono text-[var(--fs-heading)] {sovereigntyStore.score.score >= 70 ? 'text-success' : sovereigntyStore.score.score >= 40 ? 'text-warning' : 'text-error'}">
          {sovereigntyStore.score.score}
          <span class="text-[var(--fs-label)] text-text-muted">/100</span>
        </span>
      </header>

      <ul class="flex flex-col">
        {#each sovereigntyStore.score.components as c}
          <li class="px-4 py-2 border-b border-border-primary/50 last:border-b-0 flex items-start gap-2">
            {#if c.ok}
              <ShieldCheck size={12} class="text-success mt-0.5 shrink-0" />
            {:else}
              <ShieldAlert size={12} class="text-warning mt-0.5 shrink-0" />
            {/if}
            <div class="flex-1 min-w-0">
              <div class="flex items-center justify-between gap-2">
                <span class="text-[var(--fs-label)] font-bold text-text-secondary">{c.name}</span>
                <span class="font-mono text-[var(--fs-micro)] text-text-muted shrink-0">{c.earned}/{c.weight}</span>
              </div>
              <p class="text-[var(--fs-micro)] text-text-muted leading-relaxed mt-0.5">{c.reason}</p>
            </div>
          </li>
        {/each}
      </ul>

      <footer class="px-4 py-2 bg-surface-2/50 border-t border-border-primary">
        <p class="text-[var(--fs-micro)] text-text-muted leading-relaxed">
          Tier: <span class="font-mono uppercase tracking-widest text-text-secondary">{sovereigntyStore.score.tier}</span>. Set <span class="font-mono">OBLIVRA_AIRGAP=1</span> / disable remote KMS to maximise.
        </p>
      </footer>
    </div>
  {/if}
</div>
