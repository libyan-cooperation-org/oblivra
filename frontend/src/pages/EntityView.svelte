<!-- Entity View — pivot inspector for users / hosts / identities. -->
<script lang="ts">
  import { PageLayout, Button, KPI, PopOutButton } from '@components/ui';
  import { Search, Users, Server, KeyRound } from 'lucide-svelte';
  import { agentStore } from '@lib/stores/agent.svelte';
  import { push } from '@lib/router.svelte';

  let entityID = $state('');

  function inspectHost() { if (entityID) push(`/host/${encodeURIComponent(entityID)}`); }
  function inspectUser() { if (entityID) push(`/ueba-overview?user=${encodeURIComponent(entityID)}`); }
  function inspectIdentity() { if (entityID) push(`/identity-admin?user=${encodeURIComponent(entityID)}`); }
</script>

<PageLayout title="Entity View" subtitle="Pivot any user / host / identity into its dedicated detail view">
  {#snippet toolbar()}
    <PopOutButton route="/entity" title="Entity View" />
  {/snippet}
  <div class="flex flex-col h-full gap-4">
    <div class="grid grid-cols-1 md:grid-cols-3 gap-3">
      <KPI label="Indexed Hosts" value={agentStore.agents.length.toString()} variant="accent" />
      <KPI label="Type" value="Host / User / Identity" variant="muted" />
      <KPI label="Mode" value="Router" variant="muted" />
    </div>
    <div class="bg-surface-1 border border-border-primary rounded-md p-4 flex items-center gap-2">
      <Search size={14} class="text-text-muted" />
      <input class="flex-1 bg-surface-2 border border-border-primary rounded px-2 py-1.5 text-xs outline-none focus:border-accent font-mono" placeholder="Entity id (host id / username / ip)…" bind:value={entityID} />
    </div>
    <div class="grid grid-cols-1 md:grid-cols-3 gap-3">
      <button class="bg-surface-1 border border-border-primary rounded-md p-4 hover:border-accent text-left" onclick={inspectHost}>
        <Server size={18} class="text-accent mb-2" />
        <div class="font-bold text-sm">As Host</div>
        <div class="text-[11px] text-text-muted">Open the host detail page (timeline, alerts, network).</div>
      </button>
      <button class="bg-surface-1 border border-border-primary rounded-md p-4 hover:border-accent text-left" onclick={inspectUser}>
        <Users size={18} class="text-accent mb-2" />
        <div class="font-bold text-sm">As User</div>
        <div class="text-[11px] text-text-muted">Open the UEBA profile (anomaly score, baseline).</div>
      </button>
      <button class="bg-surface-1 border border-border-primary rounded-md p-4 hover:border-accent text-left" onclick={inspectIdentity}>
        <KeyRound size={18} class="text-accent mb-2" />
        <div class="font-bold text-sm">As Identity</div>
        <div class="text-[11px] text-text-muted">Open the IAM admin view (role, MFA, connectors).</div>
      </button>
    </div>
  </div>
</PageLayout>
