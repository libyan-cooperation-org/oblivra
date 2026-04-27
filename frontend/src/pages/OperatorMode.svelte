<!--
  OBLIVRA — Operator Mode (Svelte 5)
  Tactical forensic interface for host response.
-->
<script lang="ts">
  import { onMount } from 'svelte';
  import { PageLayout, Badge, Button, KPI } from '@components/ui';
  import { ShieldAlert, Zap, MoreHorizontal, Terminal as TerminalIcon, Search, ShieldCheck, Activity, Lock, Database } from 'lucide-svelte';
  import Terminal from '@components/shell/Terminal.svelte';
  import { appStore } from '@lib/stores/app.svelte';
  import { agentStore } from '@lib/stores/agent.svelte';
  import { alertStore } from '@lib/stores/alerts.svelte.ts';

  // Read the agent ID injected by appStore.navigate('operator', { id: '...' })
  function readNavParam(key: string): string {
    try {
      const raw = sessionStorage.getItem('oblivra:nav_params');
      if (!raw) return '';
      return (JSON.parse(raw)[key] as string) ?? '';
    } catch { return ''; }
  }

  let agentID = $state(readNavParam('id'));
  const agent = $derived(agentStore.agents.find(a => a.id === agentID || a.hostname === agentID));
  
  // Filter events specifically for this host from the global alertStore
  const hostEvents = $derived(
    alertStore.alerts.filter(a => a.host === agentID || a.host === agent?.hostname)
  );

  onMount(() => {
    const id = readNavParam('id');
    if (id) {
        agentID = id;
    }

    // Listen for global keybinds dispatched by App.svelte's Ctrl+Shift+I/E
    // handlers. The keybind triggers the same flow as clicking the toolbar
    // buttons, but no host context is required at the global level — we
    // resolve the active host here.
    const onIsolate = () => {
      if (!agentID) {
        appStore.notify('No host selected — pick a host to isolate', 'warning');
        return;
      }
      void isolateHost();
    };
    const onCaptureEvent = () => {
      if (!agentID) {
        appStore.notify('No host selected — pick a host to capture evidence', 'warning');
        return;
      }
      void captureEvidence();
    };
    window.addEventListener('oblivra:isolate-host', onIsolate);
    window.addEventListener('oblivra:capture-evidence', onCaptureEvent);
    return () => {
      window.removeEventListener('oblivra:isolate-host', onIsolate);
      window.removeEventListener('oblivra:capture-evidence', onCaptureEvent);
    };
  });

  async function isolateHost() {
    if (!agentID) return;
    try {
        await agentStore.toggleQuarantine(agentID, true);
        appStore.notify(`Host ${agentID} isolated from network`, 'warning');
    } catch (err: any) {
        appStore.notify(`Isolation failed: ${err}`, 'error');
    }
  }

  async function captureEvidence() {
    if (!agentID) return;
    appStore.notify(`Forensic acquisition starting for ${agentID}…`, 'info');
    try {
      const { AcquireMemoryDump } = await import(
        '@wailsjs/github.com/kingknull/oblivrashell/internal/services/forensicsservice'
      );
      // We don't have an incident ID yet — pass agentID as the incident
      // tag so the captured evidence is at least attributable.
      const item = await AcquireMemoryDump(`host:${agentID}`);
      appStore.notify(`Memory dump sealed (${(item as any)?.id ?? 'pending'})`, 'success');
      void loadEvidence();
    } catch (e: any) {
      appStore.notify(`Forensic capture failed: ${e?.message ?? e}`, 'error');
    }
  }

  function pivotToSIEM() {
    appStore.navigate('/siem-search', { query: `host.id == "${agentID}"` });
  }

  // Real evidence list (was hardcoded). Pulled from ForensicsService.
  type Evidence = { id: string; name: string; size: string; ts: string; sealed: boolean };
  let evidence = $state<Evidence[]>([]);
  async function loadEvidence() {
    if (!agentID) return;
    try {
      const { ListEvidence } = await import(
        '@wailsjs/github.com/kingknull/oblivrashell/internal/services/forensicsservice'
      );
      const list = ((await ListEvidence(`host:${agentID}`)) ?? []) as any[];
      evidence = list.map((e) => ({
        id: e.id ?? '',
        name: e.name ?? e.evidence_type ?? 'evidence',
        size: typeof e.size_bytes === 'number' ? fmtSize(e.size_bytes) : '—',
        ts: (e.created_at ?? '').slice(11, 16) + 'Z',
        sealed: !!e.sealed_at,
      }));
    } catch {
      // If ForensicsService isn't reachable yet, leave the list empty
      // rather than show fake "EVD-044" rows.
      evidence = [];
    }
  }
  function fmtSize(b: number): string {
    if (b < 1024) return `${b}B`;
    if (b < 1024 * 1024) return `${(b / 1024).toFixed(1)}KB`;
    if (b < 1024 * 1024 * 1024) return `${(b / 1024 / 1024).toFixed(1)}MB`;
    return `${(b / 1024 / 1024 / 1024).toFixed(1)}GB`;
  }
  $effect(() => {
    if (agentID) void loadEvidence();
  });
</script>

<PageLayout title="Operator Mode" subtitle="Tactical response orchestration and forensic control">
  {#snippet toolbar()}
    <div class="flex items-center gap-3">
        <div class="flex items-center gap-2 px-3 py-1 bg-error/10 border border-error/30 rounded-full animate-pulse">
            <ShieldAlert size={12} class="text-error" />
            <span class="text-[9px] font-bold text-error uppercase tracking-widest">Compromised State</span>
        </div>
        <div class="flex gap-1">
          <Badge variant="muted" class="font-mono text-[9px]">⌃⇧I</Badge>
          <Badge variant="muted" class="font-mono text-[9px]">⌃⇧E</Badge>
          <Badge variant="muted" class="font-mono text-[9px]">⌃⇧F</Badge>
        </div>
    </div>
  {/snippet}

  <div class="flex flex-col h-full gap-0 -m-6">
    <!-- ISOLATED BANNER -->
    <div class="bg-error/10 border-b border-error/40 px-4 py-2 flex items-center justify-between shrink-0">
        <div class="flex items-center gap-3">
            <span class="text-error text-sm font-bold">▲</span>
            <div>
                <div class="text-[10px] font-mono font-bold text-error tracking-tight">ANOMALY DETECTED — INC-2026-0419-007 ACTIVE</div>
                <div class="text-[9px] font-mono text-error/70">Credential dump + lateral movement · Risk 97 · SLA breach in <span class="font-bold underline">08:12</span></div>
            </div>
        </div>
        <div class="flex gap-2">
            <Button variant="danger" size="xs" class="font-mono text-[9px]" onclick={pivotToSIEM}>SIEM FILTER ⌃⇧F</Button>
            <Button variant="danger" size="xs" class="font-mono text-[9px]" onclick={isolateHost}>ISOLATE NOW ⌃⇧I</Button>
        </div>
    </div>

    <!-- MAIN BODY -->
    <div class="flex-1 grid grid-cols-1 lg:grid-cols-3 min-h-0">
        <!-- TERMINAL AREA -->
        <div class="lg:col-span-2 flex flex-col border-r border-border-primary bg-background">
            <div class="flex items-center gap-2 px-3 py-1.5 bg-surface-2 border-b border-border-primary shrink-0">
                <div class="w-2 h-2 rounded-full bg-success"></div>
                <span class="text-[9px] font-mono font-bold uppercase tracking-widest text-text-muted">Terminal — oblivra@{agent?.hostname || 'HOST-FIN-044'}</span>
                <div class="ml-auto flex gap-2">
                    <Badge variant="muted" class="text-[8px]">⌃C</Badge>
                    <Badge variant="muted" class="text-[8px]">⌃D</Badge>
                </div>
            </div>
            <div class="flex-1 min-h-0 bg-[#030608]">
                <Terminal />
            </div>
        </div>

        <!-- CONTEXT SIDEBAR -->
        <div class="flex flex-col bg-surface-1 overflow-auto">
            <!-- HOST CONTEXT -->
            <div class="border-b border-border-primary">
                <div class="flex items-center justify-between px-3 py-2 bg-surface-2 border-b border-border-primary">
                    <span class="text-[9px] font-mono font-bold text-text-muted tracking-widest uppercase">Host Context</span>
                    <span class="text-[10px] font-mono font-bold text-error">RISK 97</span>
                </div>
                <div class="p-3 grid grid-cols-2 gap-4">
                    <div class="space-y-0.5">
                        <div class="text-[8px] font-mono text-text-muted uppercase">Hostname</div>
                        <div class="text-[10px] font-mono text-text-heading font-bold">{agent?.hostname || 'HOST-FIN-044'}</div>
                    </div>
                    <div class="space-y-0.5">
                        <div class="text-[8px] font-mono text-text-muted uppercase">IP Address</div>
                        <div class="text-[10px] font-mono text-accent font-bold">{agent?.remote_address || '10.18.44.44'}</div>
                    </div>
                    <div class="space-y-0.5">
                        <div class="text-[8px] font-mono text-text-muted uppercase">OS</div>
                        <div class="text-[10px] font-mono text-text-heading">{agent?.version || 'WIN-SERVER-2022'}</div>
                    </div>
                    <div class="space-y-0.5">
                        <div class="text-[8px] font-mono text-text-muted uppercase">Status</div>
                        <div class="text-[10px] font-mono text-error font-bold uppercase">Compromised</div>
                    </div>
                </div>
            </div>

            <!-- OPERATOR ACTIONS -->
            <div class="border-b border-border-primary">
                <div class="px-3 py-2 bg-surface-2 border-b border-border-primary">
                    <span class="text-[9px] font-mono font-bold text-text-muted tracking-widest uppercase">Operator Actions</span>
                </div>
                <div class="p-3 grid grid-cols-2 gap-2">
                    <button 
                        class="flex flex-col gap-1 p-2 bg-error/10 border border-error/30 rounded-sm text-left hover:bg-error/20 transition-colors"
                        onclick={isolateHost}
                    >
                        <span class="text-[10px] font-bold text-error tracking-tight uppercase">Isolate Host</span>
                        <span class="text-[8px] font-mono text-error/70">⌃⇧I · Cut Network</span>
                    </button>
                    <button 
                        class="flex flex-col gap-1 p-2 bg-warning/10 border border-warning/30 rounded-sm text-left hover:bg-warning/20 transition-colors"
                        onclick={captureEvidence}
                    >
                        <span class="text-[10px] font-bold text-warning tracking-tight uppercase">Evidence</span>
                        <span class="text-[8px] font-mono text-warning/70">⌃⇧E · Mem + Disk</span>
                    </button>
                    <button class="flex flex-col gap-1 p-2 bg-accent/10 border border-accent/30 rounded-sm text-left hover:bg-accent/20 transition-colors">
                        <span class="text-[10px] font-bold text-accent tracking-tight uppercase">Kill Process</span>
                        <span class="text-[8px] font-mono text-accent/70">PID 4412 · C2</span>
                    </button>
                    <button 
                        class="flex flex-col gap-1 p-2 bg-accent/10 border border-accent/30 rounded-sm text-left hover:bg-accent/20 transition-colors"
                        onclick={pivotToSIEM}
                    >
                        <span class="text-[10px] font-bold text-accent tracking-tight uppercase">SIEM Filter</span>
                        <span class="text-[8px] font-mono text-accent/70">⌃⇧F · Pivot Host</span>
                    </button>
                </div>
            </div>

            <!-- RECENT EVENTS -->
            <div class="border-b border-border-primary">
                <div class="px-3 py-2 bg-surface-2 border-b border-border-primary">
                    <span class="text-[9px] font-mono font-bold text-text-muted tracking-widest uppercase">Recent Events</span>
                </div>
                <div class="p-2 space-y-1">
                    {#each hostEvents as event}
                        <div class="flex items-start gap-2 p-1.5 hover:bg-surface-2 rounded-sm cursor-pointer transition-colors group">
                            <span class="text-[8px] font-mono text-text-muted w-8 shrink-0">{new Date(event.timestamp).toLocaleTimeString([], {hour: '2-digit', minute:'2-digit'})}</span>
                            <div class="w-1.5 h-1.5 rounded-full mt-1 shrink-0 {event.severity === 'critical' ? 'bg-error' : event.severity === 'high' ? 'bg-warning' : 'bg-info'}"></div>
                            <span class="text-[9px] {event.severity === 'critical' ? 'text-error font-bold' : 'text-text-secondary'}">{event.title}</span>
                        </div>
                    {:else}
                        <div class="p-4 text-center text-[9px] text-text-muted font-mono uppercase tracking-widest opacity-40">
                            No active telemetry for host
                        </div>
                    {/each}
                </div>
            </div>

            <!-- EVIDENCE VAULT -->
            <div class="flex-1">
                <div class="flex items-center justify-between px-3 py-2 bg-surface-2 border-b border-border-primary">
                    <span class="text-[9px] font-mono font-bold text-text-muted tracking-widest uppercase">Evidence Vault</span>
                    <span class="text-[8px] font-mono text-success">3 SEALED</span>
                </div>
                <div class="p-2 space-y-2">
                    {#each evidence as item}
                        <div class="flex items-center gap-3 p-2 bg-surface-2 border border-border-primary rounded-sm hover:border-border-hover cursor-pointer transition-colors">
                            <div class="p-1.5 bg-accent/10 rounded-sm text-accent">
                                {#if item.name.includes('mem')}
                                    <Database size={14} />
                                {:else}
                                    <Activity size={14} />
                                {/if}
                            </div>
                            <div class="flex-1 min-w-0">
                                <div class="text-[10px] font-mono text-text-heading font-bold truncate">{item.name}</div>
                                <div class="text-[8px] font-mono text-text-muted">{item.size} · {item.ts}</div>
                            </div>
                            <div class="text-[7px] font-mono font-bold px-1.5 py-0.5 bg-success/10 text-success border border-success/20 rounded-xs">SEALED</div>
                        </div>
                    {/each}
                </div>
            </div>
        </div>
    </div>

    <!-- STATUS BAR -->
    <div class="bg-surface-2 border-t border-border-primary px-3 py-1 flex items-center gap-4 text-[8px] font-mono text-text-muted shrink-0">
        <div class="flex items-center gap-1.5">
            <span>SSH:</span>
            <span class="text-success font-bold">CONNECTED</span>
        </div>
        <span class="text-border-primary">|</span>
        <div class="flex items-center gap-1.5">
            <span>VAULT:</span>
            <span class="text-success">3 ITEMS SEALED</span>
        </div>
        <div class="ml-auto font-bold text-accent tracking-tighter">AIR-GAPPED MODE · SOVEREIGN</div>
    </div>
  </div>
</PageLayout>

<style>
  /* Custom scrollbar for tactical feel */
  ::-webkit-scrollbar {
    width: 3px;
    height: 3px;
  }
  ::-webkit-scrollbar-thumb {
    background: var(--border-primary);
    border-radius: 1px;
  }
</style>
