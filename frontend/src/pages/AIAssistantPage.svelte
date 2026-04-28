<!--
  OBLIVRA — Cognitive Cortex (Svelte 5)
  Autonomous mission orchestration and cognitive threat analysis.
-->
<script lang="ts">
  import { onMount } from 'svelte';
  import { PageLayout, Badge, Button, Modal } from '@components/ui';
  import { Bot, Terminal, Send, Sparkles, Activity, ShieldCheck, Database, Zap, Search, AlertTriangle, ShieldAlert } from 'lucide-svelte';
  import { appStore } from '@lib/stores/app.svelte';
  import { agentStore } from '@lib/stores/agent.svelte';
  import { alertStore } from '@lib/stores/alerts.svelte';
  import { IS_BROWSER } from '@lib/context';

  let query = $state('');
  let loading = $state(false);
  let chatHistory = $state<any[]>([]);
  let activeMode = $state('Analyst');
  let showIsolateModal = $state(false);
  let isIsolating = $state(false);

  // Real-time context derived from the agent + alert stores. Replaces
  // the hardcoded `DC-01-SECURE` placeholder so the Cortex panel
  // actually reflects what's happening in the fleet (carry-over fix).
  // Strategy:
  //   • Active host = the most recently-active agent (by last_seen).
  //   • Risk score  = open critical+high count, normalised against a
  //     50-alert ceiling so a clean fleet shows ~0 and a saturated one
  //     shows 100. Quick & defensible — full risk maths lives in the
  //     scoring service.
  //   • Last event  = the most recent alert title.
  const activeHostInfo = $derived.by(() => {
    const agents = agentStore.agents ?? [];
    if (agents.length === 0) return { hostname: 'NO-AGENTS', status: 'offline', lastSeen: '' };
    const sorted = [...agents].sort((a, b) =>
      String(b.last_seen ?? '').localeCompare(String(a.last_seen ?? '')),
    );
    const top = sorted[0];
    return {
      hostname: (top.hostname || top.id || 'UNKNOWN').toUpperCase(),
      status: top.status ?? 'unknown',
      lastSeen: top.last_seen ?? '',
    };
  });

  const riskScore = $derived.by(() => {
    const open = (alertStore.alerts ?? []).filter(
      (a) => a.status !== 'closed' && a.status !== 'resolved',
    );
    const weighted = open.reduce((sum, a) => {
      const sev = String(a.severity || '').toLowerCase();
      if (sev === 'critical') return sum + 4;
      if (sev === 'high') return sum + 2;
      if (sev === 'medium') return sum + 1;
      return sum + 0;
    }, 0);
    // Normalise: 50 weighted points = 100% risk.
    return Math.min(100, Math.round((weighted / 50) * 100));
  });

  const latestAlert = $derived.by(() => {
    const sorted = [...(alertStore.alerts ?? [])].sort((a, b) =>
      String(b.timestamp ?? '').localeCompare(String(a.timestamp ?? '')),
    );
    if (sorted.length === 0) return null;
    return sorted[0];
  });

  // Re-named context object — used everywhere downstream.
  const contextData = $derived({
    riskScore,
    activeHost: activeHostInfo.hostname,
    activeHostStatus: activeHostInfo.status,
    lastEvent: latestAlert?.title ?? 'NO_RECENT_EVENTS',
    lastEventDetectedAt: latestAlert?.timestamp ?? '',
    sovereigntyLevel: appStore.currentTenantId ? 'Scoped' : 'High',
  });

  async function loadHistory() {
    if (IS_BROWSER) {
        // Audit fix #8 — be honest. The browser-mode build has no
        // wired AIService and the previous "Cognitive Core online"
        // greeting + canned 1.2s `setTimeout` reply lied to the user
        // about a working AI loop. Show the offline state explicitly
        // so an operator sees they have to use the desktop shell to
        // reach the model.
        chatHistory = [
            { id: '1', Role: 'assistant', Content: 'AI Cortex is desktop-only. The browser/web build does not ship the inference bridge — open the OBLIVRA desktop app to run analyst, hunter, and response queries against the model.' }
        ];
        return;
    }
    try {
        const { GetChatHistory } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/aiservice');
        const history = await GetChatHistory();
        chatHistory = history || [];
    } catch (err) {
        console.error('Failed to load chat history', err);
    }
  }

  async function submitQuery() {
    if (!query.trim() || loading) return;
    
    const userMsg = query;
    chatHistory = [...chatHistory, { Role: 'user', Content: userMsg }];
    query = '';
    loading = true;

    if (IS_BROWSER) {
        // Audit fix #8 — no fake "Analysis phase initiated"
        // setTimeout. The web build has no AI backend; tell the user
        // honestly so they don't think the response was real (the
        // previous canned reply happily quoted the operator's
        // question back at them with fabricated correlation results).
        chatHistory = [
            ...chatHistory,
            {
                Role: 'assistant',
                Content:
                    'AI Cortex is unavailable in browser mode. Run this query from the OBLIVRA desktop app — the inference bridge ships with the desktop binary only.',
            },
        ];
        loading = false;
        return;
    }

    try {
        const { SendMessage } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/aiservice');
        const response = await SendMessage(userMsg);
        chatHistory = [...chatHistory, { Role: 'assistant', Content: response }];
    } catch (err) {
        appStore.notify('AI Core connection failure', 'error', (err as Error).message);
    } finally {
        loading = false;
    }
  }

  function handleFormSubmit(e: Event) {
    e.preventDefault();
    submitQuery();
  }

  function handleIsolateRequest() {
    showIsolateModal = true;
  }

  async function executeIsolation() {
    isIsolating = true;
    try {
        if (!IS_BROWSER) {
            const { IsolateHost } = await import('@wailsjs/github.com/kingknull/oblivrashell/internal/services/networkisolatorservice');
            await IsolateHost(contextData.activeHost, "Analyst requested immediate isolation via Cortex UI.");
        }
        appStore.notify('Isolation Successful', 'success', `Host ${contextData.activeHost} has been isolated.`);
    } catch (err) {
        appStore.notify('Isolation Failed', 'error', (err as Error).message);
    } finally {
        isIsolating = false;
        showIsolateModal = false;
    }
  }

  function setMode(mode: string) {
    activeMode = mode;
  }

  onMount(() => {
    loadHistory();
  });
</script>

<PageLayout title="OBLIVRA / Cortex AI" subtitle="Autonomous Agentic Intelligence & Tactical Orchestration">
  {#snippet toolbar()}
    <div class="flex items-center gap-3">
      <Badge variant="success" dot>SOVEREIGNTY: OPTIMAL</Badge>
      <div class="h-4 w-px bg-border-primary mx-1"></div>
      <div class="flex items-center gap-1">
        {#each ['Analyst', 'Hunter', 'Response'] as mode}
          <button 
            class="px-3 py-1 text-[9px] font-bold uppercase tracking-wider rounded transition-all {activeMode === mode ? 'bg-accent text-inverted shadow-glow' : 'text-text-muted hover:text-text-primary'}"
            onclick={() => setMode(mode)}
          >
            {mode}
          </button>
        {/each}
      </div>
    </div>
  {/snippet}
  <div class="flex h-full gap-4 overflow-hidden">
    <!-- LEFT PANEL: Agent Identity & Intelligence -->
    <div class="w-80 flex flex-col gap-4 overflow-auto">
      <!-- Agent Profile Card -->
      <div class="p-5 bg-surface-1 border border-border-primary rounded-md shadow-premium relative group">
        <div class="absolute top-3 right-3">
           <Activity size={14} class="text-accent animate-pulse" />
        </div>
        <div class="flex items-center gap-4 mb-5">
           <div class="w-14 h-14 rounded-md bg-accent/10 border border-accent/30 flex items-center justify-center shadow-glow">
              <Bot size={28} class="text-accent" />
           </div>
           <div>
              <div class="text-[12px] font-bold text-text-heading">CORTEX-VX</div>
              <div class="text-[9px] font-mono text-accent uppercase tracking-tighter">Autonomous SIEM Core</div>
           </div>
        </div>
        
        <div class="space-y-2">
           <div class="flex justify-between items-center text-[10px]">
              <span class="text-text-muted">Status</span>
              <span class="text-success font-bold">READY</span>
           </div>
           <div class="flex justify-between items-center text-[10px]">
              <span class="text-text-muted">Active Model</span>
              <span class="text-text-secondary">GPT-4-OBLIVRA</span>
           </div>
           <div class="flex justify-between items-center text-[10px]">
              <span class="text-text-muted">Inference Latency</span>
              <span class="text-accent">142ms</span>
           </div>
        </div>
      </div>

      <!-- Context Awareness Panel -->
      <div class="flex-1 min-h-0 bg-surface-1 border border-border-primary rounded-md overflow-hidden flex flex-col">
         <div class="p-3 bg-surface-2 border-b border-border-primary text-[9px] font-bold uppercase tracking-widest text-text-muted flex items-center gap-2">
            <Zap size={10} class="text-accent" /> Intelligence Context
         </div>
         <div class="p-4 space-y-6 overflow-auto">
            <div class="space-y-2">
               <div class="flex justify-between items-end mb-1">
                  <span class="text-[10px] text-text-muted uppercase font-bold">Global Risk Score</span>
                  <span class="text-[14px] font-mono {contextData.riskScore >= 60 ? 'text-error' : contextData.riskScore >= 30 ? 'text-warning' : 'text-accent'}">{contextData.riskScore}%</span>
               </div>
               <div class="h-1 bg-surface-3 rounded-full overflow-hidden">
                  <div class="h-full shadow-glow transition-all duration-300 {contextData.riskScore >= 60 ? 'bg-error' : contextData.riskScore >= 30 ? 'bg-warning' : 'bg-accent'}" style="width: {contextData.riskScore}%"></div>
               </div>
            </div>

            <div class="grid grid-cols-1 gap-3">
               <div class="p-3 bg-surface-2/50 border border-border-subtle rounded text-[11px]">
                  <div class="text-text-muted text-[9px] uppercase font-bold mb-1 flex items-center gap-1.5">
                     <ShieldCheck size={10} class={contextData.activeHostStatus === 'online' ? 'text-success' : 'text-warning'} /> Active Asset
                  </div>
                  <div class="text-text-primary font-mono">{contextData.activeHost}</div>
                  <div class="text-[9px] text-text-muted mt-1">Status: {String(contextData.activeHostStatus).toUpperCase()}</div>
               </div>
               <div class="p-3 bg-surface-2/50 border border-border-subtle rounded text-[11px]">
                  <div class="text-text-muted text-[9px] uppercase font-bold mb-1 flex items-center gap-1.5">
                     <AlertTriangle size={10} class="text-warning" /> Latest Alert
                  </div>
                  <div class="text-text-primary font-mono truncate">{contextData.lastEvent}</div>
                  <div class="text-[9px] text-text-muted mt-1">{contextData.lastEventDetectedAt ? `Detected: ${contextData.lastEventDetectedAt.slice(0, 19)}` : 'No alerts'}</div>
               </div>
            </div>

            <div class="pt-2">
               <div class="text-text-muted text-[9px] uppercase font-bold mb-3">Capabilities</div>
               <div class="grid grid-cols-2 gap-2">
                  <Button variant="ghost" size="sm" class="bg-surface-2 hover:bg-accent/10 border-border-primary text-[9px]">
                     <Search size={10} class="mr-1.5" /> INVESTIGATE
                  </Button>
                  <Button variant="ghost" size="sm" class="bg-surface-2 hover:bg-accent/10 border-border-primary text-[9px]">
                     <Zap size={10} class="mr-1.5 text-accent" /> CORRELATE
                  </Button>
                  <Button 
                     variant="ghost" 
                     size="sm" 
                     class="bg-surface-2 hover:bg-critical/10 border-border-primary text-critical/80 text-[9px]"
                     onclick={handleIsolateRequest}
                  >
                     <ShieldCheck size={10} class="mr-1.5" /> ISOLATE
                  </Button>
                  <Button variant="ghost" size="sm" class="bg-surface-2 hover:bg-accent/10 border-border-primary text-[9px]">
                     <Database size={10} class="mr-1.5" /> DUMP
                  </Button>
               </div>
            </div>
         </div>
      </div>
    </div>

    <!-- MAIN CANVAS: Agent Brain / Chat -->
    <div class="flex-1 flex flex-col bg-surface-1 border border-border-primary rounded-md overflow-hidden shadow-premium">
       <div class="p-3 bg-surface-2 border-b border-border-primary flex justify-between items-center text-[9px] font-bold uppercase tracking-widest text-text-muted font-mono">
          <span>Mission Runbook — {activeMode} MODE</span>
          <span class="flex items-center gap-1.5 text-accent"><Activity size={10}/> STREAMING_TELEMETRY</span>
       </div>

       <div class="flex-1 overflow-auto p-6 space-y-6 scrollbar-thin">
          {#each chatHistory as msg}
             <div class="flex {msg.Role === 'user' ? 'justify-end' : 'justify-start'} items-start gap-4 animate-fade-in-up">
                {#if msg.Role === 'assistant'}
                   <div class="w-8 h-8 rounded bg-accent/20 flex items-center justify-center border border-accent/40 shrink-0 shadow-glow">
                      <Sparkles size={14} class="text-accent" />
                   </div>
                {/if}
                <div class="max-w-[80%] rounded-lg overflow-hidden {msg.Role === 'user' ? 'bg-surface-3 border border-border-secondary' : 'glass-panel border border-border-primary shadow-xl'}">
                   {#if msg.Role === 'assistant'}
                      <div class="px-4 py-2 bg-accent/5 border-b border-accent/10 flex items-center justify-between">
                         <div class="text-[8px] font-black text-accent uppercase tracking-[0.2em]">Cortex Strategy Synthesis</div>
                         <Badge variant="success" size="xs" dot>SECURE</Badge>
                      </div>
                   {/if}
                   <div class="p-4">
                      <p class="text-[11px] leading-relaxed text-text-secondary {msg.Role === 'assistant' ? 'font-sans' : 'font-mono'} whitespace-pre-wrap">
                        {msg.Content}
                      </p>
                   </div>
                </div>
                {#if msg.Role === 'user'}
                   <div class="w-8 h-8 rounded bg-surface-2 flex items-center justify-center border border-border-primary shrink-0 font-bold text-[10px] text-accent">
                      OP
                   </div>
                {/if}
             </div>
          {/each}
          {#if loading}
             <div class="flex justify-start items-start gap-4">
                <div class="w-8 h-8 rounded bg-accent/10 flex items-center justify-center border border-accent/20 shrink-0 animate-pulse">
                   <div class="w-1.5 h-1.5 bg-accent rounded-full animate-ping"></div>
                </div>
                <div class="py-3 px-4 rounded-lg bg-surface-2/40 border border-border-dashed italic text-text-muted text-[10px] animate-pulse">
                   Synthesizing mission logic...
                </div>
             </div>
          {/if}
       </div>

       <!-- COMMAND INPUT (The OBLIVRA Terminal Input) -->
       <div class="p-6 bg-surface-2/50 backdrop-blur-xl border-t border-border-primary">
          <div class="max-w-3xl mx-auto">
             <form class="flex items-center gap-4 bg-surface-0 border border-border-primary rounded-md pl-4 pr-2 py-2 shadow-inner focus-within:border-accent group transition-all" onsubmit={handleFormSubmit}>
                <Terminal size={14} class="text-text-muted group-focus-within:text-accent" />
                <input 
                  type="text" 
                  class="flex-1 bg-transparent border-none outline-none text-[12px] text-text-primary py-1 font-mono placeholder:text-text-muted/40" 
                  placeholder="Type command or query for Cortex..." 
                  bind:value={query}
                  disabled={loading}
                />
                <Button variant="primary" size="sm" type="submit" disabled={loading} class="h-8 w-8 !p-0">
                   <Send size={14} class={loading ? 'animate-pulse' : ''} />
                </Button>
             </form>
             <div class="mt-3 flex justify-center gap-6 opacity-40">
                <div class="flex items-center gap-1.5">
                   <kbd class="px-1.5 py-0.5 rounded bg-surface-3 border border-border-primary text-[8px] font-bold text-text-muted uppercase">ALT+I</kbd>
                   <span class="text-[8px] text-text-muted uppercase font-bold tracking-widest">Auto-Investigate</span>
                </div>
                <div class="flex items-center gap-1.5">
                   <kbd class="px-1.5 py-0.5 rounded bg-surface-3 border border-border-primary text-[8px] font-bold text-text-muted uppercase">CMD+K</kbd>
                   <span class="text-[8px] text-text-muted uppercase font-bold tracking-widest">Tactical Clear</span>
                </div>
             </div>
          </div>
       </div>
    </div>
  </div>


<Modal 
  open={showIsolateModal} 
  title="CRITICAL ACTION PREVIEW" 
  onClose={() => showIsolateModal = false}
  size="sm"
>
  <div class="space-y-4">
    <div class="flex items-center gap-3 p-3 bg-critical/10 border border-critical/20 rounded">
      <ShieldAlert class="text-critical" size={24} />
      <div>
        <div class="text-[11px] font-bold text-critical uppercase leading-tight">Network Isolation</div>
        <div class="text-[10px] text-text-muted">Target Code: {contextData.activeHost}</div>
      </div>
    </div>
    
    <div class="text-[11px] text-text-primary leading-relaxed bg-surface-1 p-3 rounded font-mono border border-border-primary">
      <span class="text-text-muted">> systemctl isolate-host --node {contextData.activeHost}</span><br/>
      <span class="text-text-muted">> status:</span> Awaiting Operator Confirmation...
    </div>

    <div class="text-[10px] text-text-muted italic border-l-2 border-accent pl-3">
      CAUTION: This will terminate all active PTY sessions and block all inbound/outbound L3 traffic for the target node.
    </div>
  </div>

  {#snippet footer()}
    <Button variant="ghost" onclick={() => showIsolateModal = false} disabled={isIsolating}>Cancel</Button>
    <Button variant="primary" class="bg-critical hover:bg-critical/90" onclick={executeIsolation} loading={isIsolating}>
      Confirm Isolation
    </Button>
  {/snippet}
</Modal>

</PageLayout>


<style>
  .glass-panel {
    background: linear-gradient(135deg, rgba(26, 35, 48, 0.7) 0%, rgba(18, 24, 33, 0.9) 100%);
    backdrop-filter: blur(12px);
  }
</style>

