<!--
  OBLIVRA — Cognitive Cortex (Svelte 5)
  Autonomous mission orchestration and cognitive threat analysis.
-->
<script lang="ts">
  import { onMount } from 'svelte';
  import { KPI, PageLayout, Badge, Button, DataTable, Modal } from '@components/ui';
  import { Bot, Terminal, Send, Sparkles, Activity, ShieldCheck, Database, Zap, Cpu, Search, AlertTriangle, ShieldAlert } from 'lucide-svelte';
  import { appStore } from '@lib/stores/app.svelte';
  import { IS_BROWSER } from '@lib/context';

  let query = $state('');
  let loading = $state(false);
  let chatHistory = $state<any[]>([]);
  let activeMode = $state('Analyst');
  let showIsolateModal = $state(false);
  let isIsolating = $state(false);

  // Simulated context data for agent intelligence
  const contextData = {
    riskScore: 32,
    activeHost: 'DC-01-SECURE',
    lastEvent: 'SSH_SUCCESS_ADMIN',
    sovereigntyLevel: 'High'
  };

  async function loadHistory() {
    if (IS_BROWSER) {
        chatHistory = [
            { id: '1', Role: 'assistant', Content: 'Cognitive Core online. Tactical context baseline established for tenant [GLOBAL]. Analyzing last 24h signal drift...' }
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
        setTimeout(() => {
            chatHistory = [...chatHistory, { Role: 'assistant', Content: `Analysis phase initiated for: "${userMsg}". 
Targeting: ${contextData.activeHost}.
Correlation Result: No immediate IOC chains detected. Applying secondary heuristic pass...` }];
            loading = false;
        }, 1200);
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
                  <span class="text-[14px] font-mono text-accent">32%</span>
               </div>
               <div class="h-1 bg-surface-3 rounded-full overflow-hidden">
                  <div class="h-full bg-accent w-[32%] shadow-glow"></div>
               </div>
            </div>

            <div class="grid grid-cols-1 gap-3">
               <div class="p-3 bg-surface-2/50 border border-border-subtle rounded text-[11px]">
                  <div class="text-text-muted text-[9px] uppercase font-bold mb-1 flex items-center gap-1.5">
                     <ShieldCheck size={10} class="text-success" /> Active Asset
                  </div>
                  <div class="text-text-primary font-mono">{contextData.activeHost}</div>
                  <div class="text-[9px] text-text-muted mt-1">Status: CALIBRATED (Phase 4)</div>
               </div>
               <div class="p-3 bg-surface-2/50 border border-border-subtle rounded text-[11px]">
                  <div class="text-text-muted text-[9px] uppercase font-bold mb-1 flex items-center gap-1.5">
                     <AlertTriangle size={10} class="text-warning" /> Critical Alert
                  </div>
                  <div class="text-text-primary font-mono truncate">UNUSUAL_ADMIN_LOGIN</div>
                  <div class="text-[9px] text-text-muted mt-1">Detected: 14m ago</div>
               </div>
            </div>

            <div class="pt-2">
               <div class="text-text-muted text-[9px] uppercase font-bold mb-3">Capabilities</div>
               <div class="grid grid-cols-2 gap-2">
                  <Button variant="ghost" size="xs" class="bg-surface-2 hover:bg-accent/10 border-border-primary text-[9px]">
                     <Search size={10} class="mr-1.5" /> INVESTIGATE
                  </Button>
                  <Button variant="ghost" size="xs" class="bg-surface-2 hover:bg-accent/10 border-border-primary text-[9px]">
                     <Zap size={10} class="mr-1.5 text-accent" /> CORRELATE
                  </Button>
                  <Button 
                     variant="ghost" 
                     size="xs" 
                     class="bg-surface-2 hover:bg-critical/10 border-border-primary text-critical/80 text-[9px]"
                     onclick={handleIsolateRequest}
                  >
                     <ShieldCheck size={10} class="mr-1.5" /> ISOLATE
                  </Button>
                  <Button variant="ghost" size="xs" class="bg-surface-2 hover:bg-accent/10 border-border-primary text-[9px]">
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
                         <Badge variant="success" size="xs" class="px-1 py-0 h-3 text-[7px]" dot>SECURE</Badge>
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
                <Button variant="accent" size="sm" type="submit" disabled={loading} class="h-8 w-8 !p-0">
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
</PageLayout>

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


<style>
  .glass-panel {
    background: linear-gradient(135deg, rgba(26, 35, 48, 0.7) 0%, rgba(18, 24, 33, 0.9) 100%);
    backdrop-filter: blur(12px);
  }
</style>

