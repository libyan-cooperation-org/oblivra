<!--
  OBLIVRA — Playbook Builder (Svelte 5)
  Designing tactical response playbooks: Automating containment and investigation logic.
-->
<script lang="ts">
  import { PageLayout, Badge, Button, PopOutButton} from '@components/ui';
  import { Zap, Plus, History, GitBranch, Save, Play } from 'lucide-svelte';
  import { playbookStore } from '@lib/stores/playbook.svelte';
  import { onMount } from 'svelte';

  const actions = $derived(playbookStore.actions.map(a => ({
      type: 'ACTION',
      name: a.replace(/_/g, ' '),
      desc: `Automated response: ${a}`
  })));

  const metrics = $derived(playbookStore.metrics);
  const recentExecutions = $derived(playbookStore.metrics.recent_executions);

  onMount(() => {
    playbookStore.refresh();
  });
</script>

<PageLayout title="Playbook Orchestration" subtitle="Mission-critical response logic: Designing and validating automated tactical playbooks">
  {#snippet toolbar()}
    <div class="flex items-center gap-2">
      <Button variant="secondary" size="sm" icon={Save}>SAVE DESIGN</Button>
      <Button variant="primary" size="sm" icon={Play}>TEST PLAYBOOK</Button>
    </div>
      <PopOutButton route="/playbook-builder" title="Playbook Builder" />
    {/snippet}

  <div class="flex flex-col h-full gap-0 -m-6">
    <!-- METRIC STRIP -->
    <div class="grid grid-cols-4 gap-px bg-border-primary border-b border-border-primary shrink-0">
        <div class="bg-surface-2 p-3">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">Executions</div>
            <div class="text-xl font-mono font-bold text-success">{metrics.total_executions}</div>
            <div class="text-[9px] text-success mt-1">▲ Total automated runs</div>
        </div>
        <div class="bg-surface-2 p-3">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">Success Rate</div>
            <div class="text-xl font-mono font-bold text-accent">{metrics.total_executions > 0 ? Math.round((metrics.success_count / metrics.total_executions) * 100) : 0}%</div>
            <div class="text-[9px] text-accent mt-1">Tactical events handled</div>
        </div>
        <div class="bg-surface-2 p-3">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">Avg Execution Time</div>
            <div class="text-xl font-mono font-bold text-text-heading">{metrics.avg_duration_ms}ms</div>
            <div class="text-[9px] text-success mt-1">Zero-lag response</div>
        </div>
        <div class="bg-surface-2 p-3">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">Failures</div>
            <div class="text-xl font-mono font-bold text-error">{metrics.failure_count}</div>
            <div class="text-[9px] text-error mt-1">Requires manual review</div>
        </div>
    </div>

    <!-- MAIN BODY -->
    <div class="flex-1 flex min-h-0">
        <!-- LEFT: ACTION LIBRARY -->
        <div class="w-72 bg-surface-2 border-r border-border-primary flex flex-col shrink-0">
            <div class="px-3 py-2 bg-surface-3 border-b border-border-primary flex items-center justify-between">
                <div class="flex items-center gap-2">
                    <Zap size={14} class="text-accent" />
                    <span class="text-[9px] font-mono font-bold uppercase tracking-widest text-text-heading">Action Library</span>
                </div>
                <button class="p-1 hover:bg-surface-4 rounded-sm transition-colors text-text-muted"><Plus size={12} /></button>
            </div>
            
            <div class="flex-1 overflow-auto p-3 space-y-2">
                {#each actions as action}
                    <div class="bg-surface-1 border border-border-primary p-3 rounded-sm space-y-1 hover:border-accent transition-colors cursor-grab active:cursor-grabbing group">
                        <div class="flex justify-between items-start">
                            <span class="text-[8px] font-mono font-bold {action.type === 'TRIGGER' ? 'text-warning' : action.type === 'LOGIC' ? 'text-info' : 'text-accent'} uppercase">{action.type}</span>
                        </div>
                        <div class="text-[10px] font-bold text-text-heading uppercase tracking-tighter">{action.name}</div>
                        <p class="text-[8px] text-text-muted font-mono leading-tight opacity-60 group-hover:opacity-100 transition-opacity">{action.desc}</p>
                    </div>
                {/each}
            </div>

            <div class="h-48 border-t border-border-primary p-4 bg-surface-3/30">
                 <div class="flex items-center gap-2 mb-2">
                    <History size={14} class="text-text-muted" />
                    <span class="text-[9px] font-mono font-bold text-text-muted uppercase tracking-widest">Recent Executions</span>
                 </div>
                 <div class="space-y-2 overflow-auto h-32 mask-fade-bottom">
                    {#each recentExecutions as exec}
                        <div class="flex justify-between items-center text-[9px] font-mono">
                            <div class="flex flex-col">
                                <span class="text-text-secondary truncate w-32">{exec.playbook}</span>
                                <span class="text-[7px] text-text-muted">{exec.time}</span>
                            </div>
                            <div class="flex items-center gap-2">
                                <span class="text-[7px] text-text-muted uppercase">{exec.duration}</span>
                                <div class="w-1.5 h-1.5 rounded-full {exec.status === 'success' ? 'bg-success' : 'bg-error'}"></div>
                            </div>
                        </div>
                    {/each}
                 </div>
            </div>
        </div>

        <!-- RIGHT: DESIGN CANVAS -->
        <div class="flex-1 flex flex-col min-w-0 bg-surface-1">
            <div class="bg-surface-2 border-b border-border-primary p-3 flex items-center justify-between shrink-0">
                <div class="flex items-center gap-2">
                    <GitBranch size={14} class="text-accent" />
                    <span class="text-[10px] font-mono font-bold uppercase tracking-widest text-text-heading">Playbook: Ransomware Containment v4.2</span>
                </div>
                <div class="flex gap-2">
                    <Badge variant="success" size="xs">STABLE</Badge>
                    <Badge variant="info" size="xs">VERSION 4.2.1</Badge>
                </div>
            </div>

            <div class="flex-1 relative overflow-hidden bg-[radial-gradient(var(--b1)_1px,transparent_1px)] bg-[size:20px_20px]">
                <!-- SIMULATED CANVAS CONTENT -->
                <div class="absolute inset-0 flex flex-col items-center justify-center p-8 gap-8">
                    <!-- NODE 1 -->
                    <div class="w-48 bg-surface-2 border-2 border-warning p-3 rounded-sm relative shadow-premium">
                        <div class="text-[8px] font-mono font-bold text-warning uppercase mb-1">Trigger</div>
                        <div class="text-[10px] font-bold text-text-heading uppercase">High Entropy Detected</div>
                        <div class="absolute -bottom-4 left-1/2 -translate-x-1/2 w-px h-4 bg-border-primary"></div>
                    </div>

                    <!-- NODE 2 -->
                    <div class="w-48 bg-surface-2 border-2 border-info p-3 rounded-sm relative shadow-premium mt-4">
                        <div class="text-[8px] font-mono font-bold text-info uppercase mb-1">Logic</div>
                        <div class="text-[10px] font-bold text-text-heading uppercase">Is Critical Asset?</div>
                        <div class="absolute -bottom-4 left-1/2 -translate-x-1/2 w-px h-4 bg-border-primary"></div>
                        <div class="absolute -right-4 top-1/2 -translate-y-1/2 w-4 h-px bg-border-primary"></div>
                        <span class="absolute -right-8 top-1/2 -translate-y-1/2 text-[7px] font-mono text-text-muted">NO</span>
                        <span class="absolute -bottom-8 left-1/2 -translate-x-1/2 text-[7px] font-mono text-text-muted">YES</span>
                    </div>

                    <!-- NODE 3 -->
                    <div class="w-48 bg-surface-2 border-2 border-accent p-3 rounded-sm relative shadow-premium mt-4">
                        <div class="text-[8px] font-mono font-bold text-accent uppercase mb-1">Action</div>
                        <div class="text-[10px] font-bold text-text-heading uppercase">Platform Lockdown</div>
                    </div>

                    <div class="absolute bottom-8 right-8 flex flex-col gap-2">
                        <button class="w-8 h-8 bg-surface-2 border border-border-primary flex items-center justify-center rounded-sm hover:bg-surface-3 text-text-muted">+</button>
                        <button class="w-8 h-8 bg-surface-2 border border-border-primary flex items-center justify-center rounded-sm hover:bg-surface-3 text-text-muted">-</button>
                        <button class="w-8 h-8 bg-surface-2 border border-border-primary flex items-center justify-center rounded-sm hover:bg-surface-3 text-text-muted">⌖</button>
                    </div>
                </div>
            </div>
        </div>
    </div>

    <!-- STATUS BAR -->
    <div class="bg-surface-2 border-t border-border-primary px-3 py-1 flex items-center gap-4 text-[8px] font-mono text-text-muted shrink-0">
        <div class="flex items-center gap-1.5">
            <span>SYNTHESIS:</span>
            <span class="text-success font-bold">READY</span>
        </div>
        <span class="text-border-primary">|</span>
        <div class="flex items-center gap-1.5">
            <span>VALIDATION:</span>
            <span class="text-success font-bold">PASSED</span>
        </div>
        <span class="text-border-primary">|</span>
        <div class="flex items-center gap-1.5">
            <span>INTEGRITY:</span>
            <span class="text-accent font-bold">SIGNED</span>
        </div>
        <div class="ml-auto uppercase tracking-widest opacity-60">ORCHESTRATOR v1.4.1</div>
    </div>
  </div>
</PageLayout>

<style>
  .overflow-auto {
    mask-image: linear-gradient(to bottom, transparent 0px, black 12px, black calc(100% - 16px), transparent 100%);
  }
</style>
