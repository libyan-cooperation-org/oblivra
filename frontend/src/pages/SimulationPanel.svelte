<!--
  Simulation Panel — adversary simulation playbook runner.
  Bound to PlaybookService.RunPlaybook (atomic-red-team-style scenarios
  are stored as playbooks server-side; the panel just lets the operator
  pick + run + watch).
-->
<script lang="ts">
  import { onMount } from 'svelte';
  import { PageLayout, KPI, Button, Badge, PopOutButton } from '@components/ui';
  import { Swords, Play, Skull } from 'lucide-svelte';
  import { IS_BROWSER } from '@lib/context';
  import { appStore } from '@lib/stores/app.svelte';
  import { push } from '@lib/router.svelte';

  type Action = { id?: string; name?: string; description?: string; kind?: string };
  let scenarios = $state<Action[]>([]);
  let running = $state<string | null>(null);
  let loading = $state(false);

  async function refresh() {
    loading = true;
    try {
      if (IS_BROWSER) { scenarios = []; return; }
      const { ListAvailableActions } = await import(
        '@wailsjs/github.com/kingknull/oblivrashell/internal/services/playbookservice'
      );
      const all = ((await ListAvailableActions()) ?? []) as Action[];
      // Heuristic: simulation-flavoured actions tend to be tagged "sim" /
      // "redteam" / "purple". Fall back to all if no taxonomy.
      const sim = all.filter((a) => /sim|red.?team|purple|atomic|attack/i.test(`${a.kind ?? ''} ${a.name ?? ''}`));
      scenarios = sim.length > 0 ? sim : all;
    } finally { loading = false; }
  }

  async function run(id: string, name: string) {
    if (!confirm(`Run simulation "${name}"? This may trigger detections.`)) return;
    running = id;
    try {
      const { ExecuteAction } = await import(
        '@wailsjs/github.com/kingknull/oblivrashell/internal/services/playbookservice'
      );
      await ExecuteAction(id, { mode: 'simulation' });
      appStore.notify(`Simulation "${name}" dispatched`, 'success');
    } catch (e: any) {
      appStore.notify(`Run failed: ${e?.message ?? e}`, 'error');
    } finally { running = null; }
  }

  onMount(refresh);
</script>

<PageLayout title="Adversary Simulation" subtitle="Resilience drills and detection-validation playbooks">
  {#snippet toolbar()}
    <Button variant="secondary" size="sm" onclick={refresh}>{loading ? 'Loading…' : 'Refresh'}</Button>
    <Button variant="primary" size="sm" onclick={() => push('/purple-team')}>Purple Team</Button>
    <PopOutButton route="/simulation" title="Simulation" />
  {/snippet}

  <div class="flex flex-col h-full gap-4">
    <div class="grid grid-cols-1 md:grid-cols-3 gap-3 shrink-0">
      <KPI label="Available Scenarios" value={scenarios.length.toString()} variant="accent" />
      <KPI label="Active Runs" value={running ? '1' : '0'} variant={running ? 'warning' : 'muted'} />
      <KPI label="Engine" value={IS_BROWSER ? 'Browser (no exec)' : 'Desktop'} variant="muted" />
    </div>

    <div class="flex-1 grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3 overflow-auto">
      {#each scenarios as s (s.id ?? s.name)}
        <div class="bg-surface-1 border border-border-primary rounded-md p-4 flex flex-col gap-2">
          <div class="flex items-start gap-2">
            <Swords size={14} class="text-accent shrink-0" />
            <div class="font-bold text-[12px] flex-1 truncate">{s.name ?? s.id ?? '—'}</div>
            <Badge variant="info" size="xs">{s.kind ?? 'sim'}</Badge>
          </div>
          {#if s.description}
            <div class="text-[10px] text-text-muted line-clamp-2">{s.description}</div>
          {/if}
          <div class="mt-auto pt-2">
            <Button variant="cta" size="sm" onclick={() => run(s.id ?? s.name ?? '', s.name ?? s.id ?? '')} disabled={running === (s.id ?? s.name)}>
              {running === (s.id ?? s.name) ? 'Running…' : 'Run scenario'}
              <Play size={10} class="ml-1" />
            </Button>
          </div>
        </div>
      {:else}
        <div class="md:col-span-3 text-center text-sm text-text-muted py-12">
          {loading ? 'Loading…' : 'No simulation scenarios registered. Drop YAML playbooks under sigma/playbooks/.'}
        </div>
      {/each}
    </div>
  </div>
</PageLayout>
