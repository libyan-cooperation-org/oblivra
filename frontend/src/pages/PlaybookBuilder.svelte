<!--
  OBLIVRA — Playbook Builder (Svelte 5)
  Designing tactical response playbooks: Automating containment and investigation logic.
-->
<script lang="ts">
  import { PageLayout, Badge, Button } from '@components/ui';
  import { Zap, Plus, History, GitBranch, Save, Play } from 'lucide-svelte';

  const actions = [
    { type: 'TRIGGER', name: 'Alert Severity > 80', desc: 'Starts playbook when risk exceeds threshold' },
    { type: 'LOGIC', name: 'If Asset == Critical', desc: 'Conditional routing based on asset gravity' },
    { type: 'ACTION', name: 'Isolate Asset', desc: 'Terminate all egress on target node' },
    { type: 'ACTION', name: 'Purge RAM', desc: 'Wipe volatile memory for forensics' },
    { type: 'ACTION', name: 'Snapshot Disk', desc: 'Create immutable forensic copy' }
  ];

  const recentExecutions = [
    { time: '10:42:15', playbook: 'Ransomware Containment', status: 'success', duration: '1.2s' },
    { time: '10:30:12', playbook: 'Exfil Block', status: 'success', duration: '0.8s' },
    { time: '09:12:44', playbook: 'Lateral Detection', status: 'failed', duration: '4.2s' }
  ];
</script>

<PageLayout title="Playbook Orchestration" subtitle="Mission-critical response logic: Designing and validating automated tactical playbooks">
  {#snippet toolbar()}
    <div class="flex items-center gap-2">
      <Button variant="secondary" size="sm" icon={Save}>SAVE DESIGN</Button>
      <Button variant="primary" size="sm" icon={Play}>TEST PLAYBOOK</Button>
    </div>
  {/snippet}

  <div class="flex flex-col h-full gap-0 -m-6">
    <!-- METRIC STRIP -->
    <div class="grid grid-cols-4 gap-px bg-border-primary border-b border-border-primary shrink-0">
        <div class="bg-surface-2 p-3">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">Response Logic</div>
            <div class="text-xl font-mono font-bold text-success">42</div>
            <div class="text-[9px] text-success mt-1">▲ Verified playbooks</div>
        </div>
        <div class="bg-surface-2 p-3">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">Automation Rate</div>
            <div class="text-xl font-mono font-bold text-accent">88%</div>
            <div class="text-[9px] text-accent mt-1">Tactical events handled</div>
        </div>
        <div class="bg-surface-2 p-3">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">Avg Execution Time</div>
            <div class="text-xl font-mono font-bold text-text-heading">1.4s</div>
            <div class="text-[9px] text-success mt-1">Zero-lag response</div>
        </div>
        <div class="bg-surface-2 p-3">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">Logic Integrity</div>
            <div class="text-xl font-mono font-bold text-success">VAL-OK</div>
            <div class="text-[9px] text-success mt-1">All logic signed</div>
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
