<!--
  OBLIVRA — First-run Operator Profile wizard.

  One question. The whole UI re-aligns based on the answer. Power users
  visit Settings → Operator Profile later to fork into 'custom' and
  override individual rules.

  Mounted unconditionally by App.svelte and self-gates on
  `appStore.profileChosen === false`. Idempotent — closing it sets
  `profileChosen = true` even if the user picked "Skip".
-->
<script lang="ts">
  import { appStore, PROFILE_LABELS, type OperatorProfileId } from '@lib/stores/app.svelte';
  import { Shield, Search, Siren, Network, X } from 'lucide-svelte';

  /**
   * Lucide-icon mapping per profile so the cards have a glyph that
   * communicates the role at a glance. Custom is excluded — it's not
   * an option in the wizard, only in Settings.
   */
  const ICONS: Record<Exclude<OperatorProfileId, 'custom'>, typeof Shield> = {
    'soc-analyst':        Shield,
    'threat-hunter':      Search,
    'incident-commander': Siren,
    'msp-admin':          Network,
  };

  const PROFILES: Array<Exclude<OperatorProfileId, 'custom'>> = [
    'soc-analyst',
    'threat-hunter',
    'incident-commander',
    'msp-admin',
  ];

  function pick(id: Exclude<OperatorProfileId, 'custom'>) {
    appStore.setProfile(id);
  }

  function skip() {
    appStore.dismissProfileWizard();
  }
</script>

{#if !appStore.profileChosen}
  <div
    class="fixed inset-0 z-[10001] bg-black/70 backdrop-blur-sm flex items-center justify-center p-6"
    role="dialog"
    aria-modal="true"
    aria-labelledby="profile-wizard-title"
  >
    <div class="w-full max-w-3xl bg-surface-1 border border-border-secondary rounded-md shadow-premium overflow-hidden">
      <div class="px-6 py-4 border-b border-border-primary flex items-center justify-between">
        <div>
          <h2 id="profile-wizard-title" class="text-[var(--fs-heading)] font-bold text-text-heading">Welcome to OBLIVRA</h2>
          <p class="text-[var(--fs-label)] text-text-muted mt-0.5">What's your job today? Pick a profile — we'll re-align the chrome to match. Change it any time in <span class="font-mono text-accent">Settings → Operator Profile</span>.</p>
        </div>
        <button
          class="text-text-muted hover:text-text-primary p-1 rounded-sm hover:bg-surface-2"
          onclick={skip}
          aria-label="Skip"
          title="Skip — keep defaults"
        >
          <X size={16} />
        </button>
      </div>

      <div class="grid grid-cols-1 md:grid-cols-2 gap-3 p-5">
        {#each PROFILES as id}
          {@const Icon = ICONS[id]}
          {@const meta = PROFILE_LABELS[id]}
          <button
            class="text-start flex flex-col gap-2 p-4 bg-surface-2 border border-border-primary rounded-md hover:border-accent hover:bg-accent/5 transition-colors duration-fast group"
            onclick={() => pick(id)}
          >
            <div class="flex items-center gap-3">
              <div class="w-10 h-10 rounded bg-accent/10 border border-accent/20 flex items-center justify-center group-hover:bg-accent/20 transition-colors">
                <Icon size={18} class="text-accent" />
              </div>
              <div>
                <div class="text-[var(--fs-body)] font-bold text-text-heading">{meta.name}</div>
                <div class="text-[var(--fs-micro)] font-mono text-text-muted uppercase tracking-widest">{id}</div>
              </div>
            </div>
            <p class="text-[var(--fs-label)] text-text-secondary leading-relaxed">{meta.subtitle}</p>
          </button>
        {/each}
      </div>

      <div class="px-6 py-3 border-t border-border-primary flex items-center justify-between bg-surface-2/50">
        <span class="text-[var(--fs-micro)] text-text-muted">Picking a profile flips ~9 settings (home route, density, palette, vim leader, tenant chrome, crisis behaviour, noise floor, layout, primary metric).</span>
        <button
          class="text-[var(--fs-label)] text-text-muted hover:text-text-secondary px-2 py-1"
          onclick={skip}
        >Skip</button>
      </div>
    </div>
  </div>
{/if}
