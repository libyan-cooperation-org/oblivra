<!--
  OBLIVRA — Crisis Decision Support panel (Phase 32, UIUX P1).

  Slides in when crisisStore.active becomes true. Shows three things,
  no more:

    1. The top 3 affected hosts (most recent critical alerts).
    2. The single recommended containment action with one-click execute.
    3. Three buttons: Seal evidence now · Open war-room view · Stand down.

  Three. Not thirty. The whole point of this panel is to reduce
  cognitive load to the operator's hands during an incident. Anything
  beyond three options goes to a sub-page.

  The panel is dismissible (operator hit ✕ but crisis remains armed)
  but reappears on the next route change so it can't be permanently
  hidden while the platform is on fire.
-->
<script lang="ts">
  import { crisisStore } from '@lib/stores/crisis.svelte';
  import { alertStore } from '@lib/stores/alerts.svelte';
  import { agentStore } from '@lib/stores/agent.svelte';
  import { appStore } from '@lib/stores/app.svelte';
  import { Badge, Button } from '@components/ui';
  import { Siren, ShieldAlert, Lock, Eye, X } from 'lucide-svelte';
  import { push } from '@lib/router.svelte';

  let dismissed = $state(false);

  // Reset dismissal when crisis re-arms.
  $effect(() => {
    if (crisisStore.active) dismissed = false;
  });

  // Top 3 hosts by recent-critical-alert count. We compute this on
  // the fly so the panel reflects the latest fan-out as alerts stream.
  let topHosts = $derived.by(() => {
    const hostCounts: Record<string, number> = {};
    for (const a of alertStore.alerts) {
      if (a.severity?.toLowerCase() !== 'critical') continue;
      if (a.status === 'closed' || a.status === 'suppressed') continue;
      const t = Date.parse(a.timestamp ?? '');
      if (!Number.isFinite(t) || Date.now() - t > 60 * 60 * 1000) continue;
      hostCounts[a.host || 'unknown'] = (hostCounts[a.host || 'unknown'] || 0) + 1;
    }
    return Object.entries(hostCounts)
      .sort((a, b) => b[1] - a[1])
      .slice(0, 3)
      .map(([host, count]) => ({ host, count }));
  });

  async function isolateHost(host: string) {
    const agent = agentStore.agents.find((a) => a.id === host || a.hostname === host);
    if (!agent) {
      appStore.notify(`No agent for ${host}`, 'warning');
      return;
    }
    try {
      const { apiPostJSON } = await import('@lib/apiClient');
      // Existing endpoint — see rest_phase8_12.go handleRansomwareIsolate.
      const res = await apiPostJSON('/api/v1/ransomware/isolate', {
        host_id: agent.id,
      });
      if (!res.ok) throw new Error(`HTTP ${res.status}`);
      appStore.notify(`Quarantined ${host}`, 'success');
    } catch (e: any) {
      appStore.notify(`Isolate ${host} failed`, 'error', e?.message ?? String(e));
    }
  }

  async function sealEvidence() {
    try {
      // No "seal current" endpoint exists yet; instead, jump to the
      // evidence ledger so the operator can review what's seal-able
      // and choose. The page itself uses /api/v1/forensics/evidence/{id}/seal
      // for the per-item action.
      push('/evidence?crisis=1&reason=' + encodeURIComponent(crisisStore.reason ?? 'crisis'));
      appStore.notify('Open evidence ledger to seal current package', 'info');
    } catch (e: any) {
      appStore.notify('Open evidence failed', 'error', e?.message ?? String(e));
    }
  }
</script>

{#if crisisStore.active && !dismissed && appStore.profileRules.crisisAffordance !== 'fullscreen-takeover'}
  <aside
    class="fixed top-16 right-4 z-[9500] w-[360px] bg-surface-1 border border-error/40 rounded-md shadow-2xl overflow-hidden"
    role="region"
    aria-label="Crisis decision support"
    style="box-shadow: 0 8px 32px rgba(192,40,40,0.30);"
  >
    <header class="flex items-center justify-between px-4 py-2 bg-error/10 border-b border-error/30">
      <div class="flex items-center gap-2">
        <Siren size={14} class="text-error animate-pulse" />
        <span class="text-[var(--fs-label)] font-bold uppercase tracking-widest text-error">Decision Support</span>
      </div>
      <button
        class="text-text-muted hover:text-text-primary p-1"
        onclick={() => (dismissed = true)}
        aria-label="Dismiss panel"
        title="Dismiss (re-appears on next route change)"
      ><X size={14} /></button>
    </header>

    <div class="p-4 flex flex-col gap-3">
      <!-- Reason -->
      <div class="text-[var(--fs-label)] text-text-secondary leading-relaxed">
        <span class="text-[var(--fs-micro)] uppercase tracking-widest text-text-muted block mb-1">Trigger</span>
        {crisisStore.reason ?? 'Crisis Mode active'}
      </div>

      <!-- Top affected hosts -->
      <section class="bg-surface-2 border border-border-primary rounded p-3 flex flex-col gap-2">
        <span class="text-[var(--fs-micro)] font-bold uppercase tracking-widest text-text-muted">Top affected hosts (1h)</span>
        {#if topHosts.length === 0}
          <span class="text-[var(--fs-label)] text-text-muted italic">No critical alerts in the last hour.</span>
        {:else}
          <ul class="flex flex-col gap-1.5">
            {#each topHosts as h (h.host)}
              <li class="flex items-center gap-2">
                <Badge variant="critical" size="xs">×{h.count}</Badge>
                <span class="font-mono text-[var(--fs-label)] text-text-primary truncate">{h.host}</span>
                <button
                  class="ml-auto px-1.5 py-0.5 rounded text-[var(--fs-micro)] font-mono text-error border border-error/40 hover:bg-error/10"
                  onclick={() => isolateHost(h.host)}
                  title="Quarantine host"
                >Isolate</button>
              </li>
            {/each}
          </ul>
        {/if}
      </section>

      <!-- Three buttons. Three. No more. -->
      <div class="grid grid-cols-1 gap-2">
        <Button variant="primary" size="sm" onclick={sealEvidence}>
          <Lock size={11} class="mr-1.5" /> Seal evidence now
        </Button>
        <Button variant="secondary" size="sm" onclick={() => push('/war-mode')}>
          <Eye size={11} class="mr-1.5" /> Open war-room view
        </Button>
        <Button variant="critical" size="sm" onclick={() => crisisStore.standDown()}>
          <ShieldAlert size={11} class="mr-1.5" /> Stand down
        </Button>
      </div>

      <p class="text-[var(--fs-micro)] text-text-muted leading-relaxed">
        Noise floor lifted to <span class="font-mono text-text-secondary">critical-only</span> for the duration of the incident. Restored on stand-down.
      </p>
    </div>
  </aside>
{/if}
