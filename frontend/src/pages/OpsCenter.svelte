<!--
  OBLIVRA — Ops Center (Svelte 5)
  The Command and Control Hub: Managing clusters and active operator mission-sets.
-->
<script lang="ts">
  import { PageLayout, Badge, Button } from '@components/ui';
  import { Terminal, Cpu, Zap } from 'lucide-svelte';

  const commandQueue = [
    { time: '10:42:15', target: 'SRV-FIN-01', cmd: 'ls -la /opt/secure', status: 'completed' },
    { time: '10:42:18', target: 'WRK-HR-04', cmd: 'netstat -antp', status: 'running' },
    { time: '10:42:20', target: 'SRV-SQL-09', cmd: 'ps aux | grep sql', status: 'queued' }
  ];

  const systemLogs = [
    { time: '10:42:21', level: 'info', msg: 'Operator K. MAVERICK authenticated via hardware token' },
    { time: '10:42:18', level: 'warn', msg: 'Inbound session from 104.1.2.4 throttled (Rate Limit)' },
    { time: '10:42:15', level: 'info', msg: 'Fleet synchronization complete. 14 shards synchronized.' },
    { time: '10:42:10', level: 'error', msg: 'Core: Failed to rotate ephemeral keys on SRV-FIN-01' }
  ];
</script>

<PageLayout title="Operations Center" subtitle="Master command & control of all managed endpoints">
  {#snippet toolbar()}
    <div class="flex items-center gap-2">
      <div class="flex items-center gap-2 px-3 py-1 bg-accent/10 border border-accent/40 rounded-sm">
        <span class="w-1.5 h-1.5 rounded-full bg-accent animate-pulse"></span>
        <span class="text-[9px] font-mono text-accent font-bold uppercase tracking-widest">Operator Online</span>
      </div>
      <Button variant="primary" size="sm">NEW MISSION</Button>
    </div>
  {/snippet}

  <div class="flex flex-col h-full gap-0 -m-6">
    <!-- METRIC STRIP -->
    <div class="grid grid-cols-4 gap-px bg-border-primary border-b border-border-primary shrink-0">
        <div class="bg-surface-2 p-3">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">Active Sessions</div>
            <div class="text-xl font-mono font-bold text-accent">12</div>
            <div class="text-[9px] text-success mt-1">▲ Encrypted Tunnel active</div>
        </div>
        <div class="bg-surface-2 p-3">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">CPU Load (Cluster)</div>
            <div class="text-xl font-mono font-bold text-text-heading">24.2%</div>
            <div class="text-[9px] text-text-muted mt-1">Mean across 14 nodes</div>
        </div>
        <div class="bg-surface-2 p-3">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">Vault Latency</div>
            <div class="text-xl font-mono font-bold text-success">42ms</div>
            <div class="text-[9px] text-success mt-1">Zero-lag sync</div>
        </div>
        <div class="bg-surface-2 p-3">
            <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">Operator Auth</div>
            <div class="text-xl font-mono font-bold text-text-heading">LEVEL 5</div>
            <div class="text-[9px] text-text-muted mt-1">Full access granted</div>
        </div>
    </div>

    <!-- MAIN BODY -->
    <div class="flex-1 flex min-h-0">
        <!-- LEFT: CONSOLE / LOGS -->
        <div class="flex-1 flex flex-col min-w-0 bg-black/20">
            <div class="bg-surface-1 border-b border-border-primary p-3 flex items-center justify-between shrink-0">
                <div class="flex items-center gap-2">
                    <Terminal size={14} class="text-accent" />
                    <span class="text-[10px] font-mono font-bold uppercase tracking-widest text-text-heading">Global Audit Trail & Debug Console</span>
                </div>
                <div class="flex gap-2">
                    <Button variant="secondary" size="xs">CLEAR CONSOLE</Button>
                    <Button variant="secondary" size="xs">EXPORT LOGS</Button>
                </div>
            </div>

            <div class="flex-1 overflow-auto p-4 font-mono text-[11px] leading-relaxed selection:bg-accent/30 mask-fade-bottom">
                {#each systemLogs as log}
                    <div class="flex gap-4 py-1 group hover:bg-white/5 transition-colors px-2 rounded-sm">
                        <span class="text-text-muted opacity-40 shrink-0">{log.time}</span>
                        <span class="uppercase font-bold shrink-0 w-12 {log.level === 'error' ? 'text-error' : log.level === 'warn' ? 'text-warning' : 'text-accent'}">[{log.level}]</span>
                        <span class="text-text-secondary">{log.msg}</span>
                    </div>
                {/each}
                <div class="mt-4 flex gap-2 items-center text-accent animate-pulse">
                    <span>❯</span>
                    <span class="w-2 h-4 bg-accent"></span>
                </div>
            </div>
        </div>

        <!-- RIGHT: COMMAND QUEUE & SYSTEM -->
        <div class="w-96 bg-surface-2 border-l border-border-primary flex flex-col shrink-0">
            <!-- CMD QUEUE -->
            <div class="flex-1 flex flex-col min-h-0">
                <div class="px-3 py-2 bg-surface-3 border-b border-border-primary flex items-center gap-2">
                    <Zap size={14} class="text-warning" />
                    <span class="text-[9px] font-mono font-bold uppercase tracking-widest text-text-heading">Agent Command Queue</span>
                </div>
                <div class="flex-1 overflow-auto p-3 space-y-2">
                    {#each commandQueue as cmd}
                        <div class="bg-surface-1 border border-border-primary p-3 rounded-sm space-y-2 group hover:border-accent transition-colors">
                            <div class="flex justify-between items-start">
                                <span class="text-[10px] font-bold text-text-heading uppercase tracking-tight">{cmd.target}</span>
                                <Badge variant={cmd.status === 'completed' ? 'success' : cmd.status === 'running' ? 'warning' : 'info'} size="xs" class="text-[7px]">
                                    {cmd.status.toUpperCase()}
                                </Badge>
                            </div>
                            <code class="block text-[10px] font-mono text-accent bg-black/20 p-1.5 rounded-sm overflow-hidden truncate">
                                {cmd.cmd}
                            </code>
                            <div class="text-[8px] font-mono text-text-muted uppercase">
                                Initialized: {cmd.time}
                            </div>
                        </div>
                    {/each}
                </div>
            </div>

            <!-- SYSTEM RESOURCES -->
            <div class="h-64 border-t border-border-primary bg-surface-3 p-4 flex flex-col gap-4">
                <div class="flex items-center justify-between">
                    <span class="text-[9px] font-mono font-bold text-text-muted uppercase tracking-widest">Cluster Vital Signs</span>
                    <Cpu size={14} class="text-text-muted" />
                </div>
                <div class="space-y-4">
                    <div class="space-y-1.5">
                        <div class="flex justify-between text-[8px] font-mono uppercase">
                            <span class="text-text-muted">CPU Cluster Mean</span>
                            <span class="text-text-heading">24%</span>
                        </div>
                        <div class="h-1 bg-surface-1 rounded-full overflow-hidden">
                            <div class="h-full bg-success" style="width: 24%"></div>
                        </div>
                    </div>
                    <div class="space-y-1.5">
                        <div class="flex justify-between text-[8px] font-mono uppercase">
                            <span class="text-text-muted">Memory Allocation</span>
                            <span class="text-text-heading">1.2 TB</span>
                        </div>
                        <div class="h-1 bg-surface-1 rounded-full overflow-hidden">
                            <div class="h-full bg-accent" style="width: 62%"></div>
                        </div>
                    </div>
                    <div class="space-y-1.5">
                        <div class="flex justify-between text-[8px] font-mono uppercase">
                            <span class="text-text-muted">I/O Wait (Avg)</span>
                            <span class="text-text-heading">0.4ms</span>
                        </div>
                        <div class="h-1 bg-surface-1 rounded-full overflow-hidden">
                            <div class="h-full bg-success" style="width: 5%"></div>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    </div>

    <!-- STATUS BAR -->
    <div class="bg-surface-2 border-t border-border-primary px-3 py-1 flex items-center gap-4 text-[8px] font-mono text-text-muted shrink-0">
        <div class="flex items-center gap-1.5">
            <span>OPERATOR:</span>
            <span class="text-accent font-black uppercase">K. MAVERICK</span>
        </div>
        <span class="text-border-primary">|</span>
        <div class="flex items-center gap-1.5">
            <span>TERMINAL:</span>
            <span class="text-success font-bold">PTY_CONNECTED</span>
        </div>
        <span class="text-border-primary">|</span>
        <div class="flex items-center gap-1.5">
            <span>CLUSTER:</span>
            <span class="text-success font-bold">SYNC_NOMINAL</span>
        </div>
        <div class="ml-auto uppercase tracking-widest opacity-60">OPS_CORE v1.4.1</div>
    </div>
  </div>
</PageLayout>

<style>
  .overflow-auto {
    mask-image: linear-gradient(to bottom, transparent 0px, black 12px, black calc(100% - 16px), transparent 100%);
  }
</style>
