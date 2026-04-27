<!--
  Team Dashboard — members + activity from TeamService.
-->
<script lang="ts">
  import { onMount } from 'svelte';
  import { PageLayout, KPI, Badge, Button, DataTable, PopOutButton } from '@components/ui';
  import { Users, RefreshCw, UserPlus } from 'lucide-svelte';
  import { IS_BROWSER } from '@lib/context';
  import { appStore } from '@lib/stores/app.svelte';

  let teamName = $state('');
  let members = $state<any[]>([]);
  let activity = $state<any[]>([]);
  let loading = $state(false);

  async function refresh() {
    loading = true;
    try {
      if (IS_BROWSER) return;
      const svc = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/teamservice');
      teamName = ((await svc.GetTeamName()) ?? '') as string;
      members  = ((await svc.ListMembers()) ?? []) as any[];
      activity = ((await svc.ListActivity()) ?? []) as any[];
    } catch (e: any) {
      appStore.notify(`Team load failed: ${e?.message ?? e}`, 'error');
    } finally { loading = false; }
  }

  async function addMember() {
    const email = prompt('Email:'); if (!email) return;
    const name  = prompt('Name:', email) ?? email;
    const role  = prompt('Role (analyst | admin | viewer):', 'analyst') ?? 'analyst';
    try {
      const { AddMember } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/teamservice');
      await AddMember(email, name, role);
      appStore.notify(`Invited ${email}`, 'success');
      void refresh();
    } catch (e: any) {
      appStore.notify(`Invite failed: ${e?.message ?? e}`, 'error');
    }
  }

  onMount(refresh);
</script>

<PageLayout title={teamName ? `Team · ${teamName}` : 'Team Dashboard'} subtitle="Operators and recent activity">
  {#snippet toolbar()}
    <Button variant="secondary" size="sm" icon={RefreshCw} onclick={refresh}>{loading ? 'Loading…' : 'Refresh'}</Button>
    <Button variant="primary" size="sm" icon={UserPlus} onclick={addMember}>Invite</Button>
    <PopOutButton route="/team" title="Team" />
  {/snippet}

  <div class="flex flex-col h-full gap-4">
    <div class="grid grid-cols-1 md:grid-cols-3 gap-3 shrink-0">
      <KPI label="Members" value={members.length.toString()} variant="accent" />
      <KPI label="Activity (24h)" value={activity.length.toString()} variant="muted" />
      <KPI label="Team" value={teamName || '—'} variant="muted" />
    </div>

    <div class="grid grid-cols-1 lg:grid-cols-2 gap-4 flex-1 min-h-0">
      <section class="flex flex-col bg-surface-1 border border-border-primary rounded-md min-h-0">
        <div class="flex items-center gap-2 p-3 border-b border-border-primary">
          <Users size={14} class="text-accent" />
          <span class="text-[10px] uppercase tracking-widest font-bold">Members</span>
        </div>
        <div class="flex-1 overflow-auto">
          {#each members as m (m.id ?? m.email)}
            <div class="px-3 py-2 border-b border-border-primary flex items-center gap-3">
              <div class="flex-1 min-w-0">
                <div class="text-[11px] font-bold truncate">{m.name ?? m.email}</div>
                <div class="text-[10px] text-text-muted truncate">{m.email}</div>
              </div>
              <Badge variant="info" size="xs">{m.role ?? '—'}</Badge>
            </div>
          {:else}
            <div class="p-8 text-center text-sm text-text-muted">{loading ? 'Loading…' : 'No team members.'}</div>
          {/each}
        </div>
      </section>

      <section class="flex flex-col bg-surface-1 border border-border-primary rounded-md min-h-0">
        <div class="flex items-center gap-2 p-3 border-b border-border-primary">
          <span class="text-[10px] uppercase tracking-widest font-bold">Recent Activity</span>
        </div>
        <div class="flex-1 overflow-auto">
          {#each activity.slice(0, 100) as e, i (e.id ?? i)}
            <div class="px-3 py-1.5 border-b border-border-primary text-[11px]">
              <span class="font-mono text-[10px] text-text-muted mr-2">{(e.timestamp ?? '').slice(11, 19)}</span>
              <span class="font-bold">{e.actor ?? '—'}</span>
              <span class="text-text-muted">{e.action ?? e.event ?? '—'}</span>
            </div>
          {:else}
            <div class="p-8 text-center text-sm text-text-muted">No recorded activity.</div>
          {/each}
        </div>
      </section>
    </div>
  </div>
</PageLayout>
