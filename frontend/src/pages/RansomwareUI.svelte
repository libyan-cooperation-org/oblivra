<!--
  Ransomware Response — emergency view: list isolated hosts, mass-isolate
  by tag/host group, trigger playbook actions. Bound to agentStore +
  alertStore (for ransomware-tagged alerts) + PlaybookService.
-->
<script lang="ts">
  import { onMount } from 'svelte';
  import { PageLayout, KPI, Badge, Button, DataTable, PopOutButton } from '@components/ui';
  import { ShieldAlert, ShieldOff, Lock, Zap } from 'lucide-svelte';
  import { agentStore } from '@lib/stores/agent.svelte';
  import { alertStore } from '@lib/stores/alerts.svelte';
  import { appStore } from '@lib/stores/app.svelte';
  import { IS_BROWSER } from '@lib/context';

  const ransomKeywords = ['ransom', 'crypto', 'lockbit', 'conti', 'wannacry', 'encrypt'];

  const ransomAlerts = $derived(
    alertStore.alerts.filter((a) => {
      const t = (a.title ?? '').toLowerCase();
      return ransomKeywords.some((k) => t.includes(k));
    }),
  );

  const isolatedHosts = $derived(
    agentStore.agents.filter((a: any) => a.quarantined === true || a.status === 'quarantined'),
  );

  const fleetCount = $derived(agentStore.agents.length);
  const isolationPct = $derived(fleetCount === 0 ? 0 : Math.round((isolatedHosts.length / fleetCount) * 100));

  async function isolateAll() {
    if (!confirm(`Quarantine ALL ${fleetCount} agents? This blocks every host's outbound traffic.`)) return;
    let ok = 0; let fail = 0;
    for (const a of agentStore.agents) {
      try { await agentStore.toggleQuarantine(a.id, true); ok++; } catch { fail++; }
    }
    appStore.notify(`Mass-isolation: ${ok} ok · ${fail} failed`, fail === 0 ? 'success' : 'warning');
  }

  async function releaseAll() {
    if (!confirm('Release ALL quarantined hosts?')) return;
    let ok = 0;
    for (const a of isolatedHosts) {
      try { await agentStore.toggleQuarantine(a.id, false); ok++; } catch {}
    }
    appStore.notify(`Released ${ok} hosts`, 'success');
  }

  async function runRansomwarePlaybook() {
    try {
      if (IS_BROWSER) { appStore.notify('Available in desktop only', 'warning'); return; }
      const { ListAvailableActions, ExecuteAction } = await import(
        '@wailsjs/github.com/kingknull/oblivrashell/internal/services/playbookservice'
      );
      const actions = ((await ListAvailableActions()) ?? []) as any[];
      const isoAction = actions.find((a) => /isolate|quarantine|ransom/i.test(a.name ?? a.id ?? ''));
      if (!isoAction) {
        appStore.notify('No ransomware playbook found', 'warning');
        return;
      }
      await ExecuteAction(isoAction.id ?? isoAction.name, { reason: 'ransomware-shield' });
      appStore.notify('Ransomware playbook dispatched', 'success');
    } catch (e: any) {
      appStore.notify(`Playbook failed: ${e?.message ?? e}`, 'error');
    }
  }

  onMount(() => {
    if (typeof agentStore.init === 'function') agentStore.init();
    if (typeof alertStore.init === 'function') alertStore.init();
  });
</script>

<PageLayout title="Ransomware Response" subtitle="Active encryption-event response and mass-isolation control">
  {#snippet toolbar()}
    <Button variant="warning" size="sm" onclick={isolateAll}>ISOLATE ALL</Button>
    <Button variant="secondary" size="sm" onclick={releaseAll} disabled={isolatedHosts.length === 0}>RELEASE ALL</Button>
    <Button variant="cta" size="sm" onclick={runRansomwarePlaybook}>RUN PLAYBOOK</Button>
    <PopOutButton route="/ransomware" title="Ransomware Response" />
  {/snippet}

  <div class="flex flex-col h-full gap-4">
    <div class="grid grid-cols-1 md:grid-cols-4 gap-3 shrink-0">
      <KPI label="Ransom-tagged Alerts" value={ransomAlerts.length.toString()} variant={ransomAlerts.length > 0 ? 'critical' : 'muted'} />
      <KPI label="Quarantined Hosts" value={isolatedHosts.length.toString()} variant={isolatedHosts.length > 0 ? 'warning' : 'muted'} />
      <KPI label="Fleet Coverage" value={`${isolationPct}%`} variant={isolationPct > 0 ? 'warning' : 'muted'} sublabel={`${fleetCount} total`} />
      <KPI label="Status" value={ransomAlerts.length > 0 ? 'ACTIVE' : 'CLEAR'} variant={ransomAlerts.length > 0 ? 'critical' : 'success'} />
    </div>

    <div class="grid grid-cols-1 lg:grid-cols-2 gap-4 flex-1 min-h-0">
      <section class="flex flex-col bg-surface-1 border border-border-primary rounded-md min-h-0">
        <div class="flex items-center gap-2 p-3 border-b border-border-primary">
          <ShieldAlert size={14} class="text-error" />
          <span class="text-[10px] uppercase tracking-widest font-bold">Recent Ransom-tagged Events</span>
        </div>
        <div class="flex-1 overflow-auto">
          {#if ransomAlerts.length === 0}
            <div class="p-8 text-center text-sm text-text-muted">No ransom-tagged alerts in window — fleet quiet.</div>
          {:else}
            <DataTable data={ransomAlerts} columns={[
              { key: 'severity', label: 'Sev', width: '70px' },
              { key: 'host',     label: 'Host', width: '140px' },
              { key: 'title',    label: 'Detection' },
              { key: 'timestamp', label: 'Time', width: '120px' },
            ]} compact>
              {#snippet render({ col, row })}
                {#if col.key === 'severity'}
                  <Badge variant={row.severity === 'critical' ? 'critical' : 'warning'} size="xs">{row.severity}</Badge>
                {:else if col.key === 'timestamp'}
                  <span class="font-mono text-[10px] text-text-muted">{row.timestamp?.slice(11, 19) ?? ''}</span>
                {:else}
                  <span class="text-[11px]">{row[col.key] ?? '—'}</span>
                {/if}
              {/snippet}
            </DataTable>
          {/if}
        </div>
      </section>

      <section class="flex flex-col bg-surface-1 border border-border-primary rounded-md min-h-0">
        <div class="flex items-center gap-2 p-3 border-b border-border-primary">
          <Lock size={14} class="text-warning" />
          <span class="text-[10px] uppercase tracking-widest font-bold">Quarantined Hosts</span>
        </div>
        <div class="flex-1 overflow-auto">
          {#if isolatedHosts.length === 0}
            <div class="p-8 text-center text-sm text-text-muted">No hosts in quarantine.</div>
          {:else}
            {#each isolatedHosts as h (h.id)}
              <div class="flex items-center gap-3 px-3 py-2 border-b border-border-primary">
                <ShieldOff size={12} class="text-warning" />
                <div class="flex-1">
                  <div class="text-[11px] font-bold">{h.hostname || h.id}</div>
                  <div class="text-[10px] font-mono text-text-muted">{h.remote_address ?? '—'}</div>
                </div>
                <Button variant="ghost" size="xs" onclick={() => agentStore.toggleQuarantine(h.id, false)}>Release</Button>
              </div>
            {/each}
          {/if}
        </div>
      </section>
    </div>
  </div>
</PageLayout>
