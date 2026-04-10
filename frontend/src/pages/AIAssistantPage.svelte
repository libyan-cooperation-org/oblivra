<!--
  OBLIVRA — Cognitive Cortex (Svelte 5)
  Autonomous mission orchestration and cognitive threat analysis.
-->
<script lang="ts">
  import { onMount } from 'svelte';
  import { KPI, PageLayout, Badge, Button } from '@components/ui';
  import { Bot, Terminal, Send, Sparkles } from 'lucide-svelte';
  import { appStore } from '@lib/stores/app.svelte';
  import { IS_BROWSER } from '@lib/context';

  let query = $state('');
  let loading = $state(false);
  let chatHistory = $state<any[]>([]);

  async function loadHistory() {
    if (IS_BROWSER) {
        chatHistory = [
            { id: '1', Role: 'assistant', Content: 'Operational environment loaded. I am ready to assist with mission orchestration or forensic analysis.' }
        ];
        return;
    }
    try {
        const { GetChatHistory } = await import('@wailsjs/go/services/AIService.js');
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
            chatHistory = [...chatHistory, { Role: 'assistant', Content: `Analyzing signal context for "${userMsg}"... Applying behavioral baseline recalibration. [BROWSER_MODE]` }];
            loading = false;
        }, 1000);
        return;
    }

    try {
        const { SendMessage } = await import('@wailsjs/go/services/AIService.js');
        const response = await SendMessage(userMsg);
        chatHistory = [...chatHistory, { Role: 'assistant', Content: response }];
    } catch (err) {
        appStore.notify('AI Core connection failure', 'error', (err as Error).message);
    } finally {
        loading = false;
    }
  }

  onMount(() => {
    loadHistory();
  });
</script>

<PageLayout title="Cognitive Cortex" subtitle="Autonomous mission orchestration and cognitive threat analysis: AI-driven platform sovereignty">
  {#snippet toolbar()}
     <Badge variant="accent" dot>COGNITIVE CORE: {loading ? 'PROCESSING' : 'READY'}</Badge>
  {/snippet}

  <div class="flex flex-col h-full gap-6">
    <!-- Stats simulation remains for aesthetic until AnalyticsService integrated -->
    <div class="grid grid-cols-1 md:grid-cols-4 gap-4 shrink-0">
      <KPI label="AI Confidence" value="98.4%" trend="stable" trendValue="Optimal" variant="success" />
      <KPI label="Active Tasks" value="3" trend="stable" trendValue="Autonomous" variant="accent" />
      <KPI label="Context Window" value="128k" trend="stable" trendValue="High Density" />
      <KPI label="Logic Drift" value="Zero" trend="stable" trendValue="Calibrated" variant="success" />
    </div>

    <div class="flex-1 min-h-0 bg-surface-1 border border-border-primary rounded-md flex flex-col shadow-premium relative overflow-hidden group">
       <!-- AI Sparkle Background -->
       <div class="absolute inset-0 pointer-events-none opacity-[0.02] flex items-center justify-center grayscale group-hover:scale-105 transition-transform duration-1000">
          <Bot size={500} />
       </div>

       <div class="p-3 bg-surface-2 border-b border-border-primary flex justify-between items-center text-[10px] font-bold uppercase tracking-widest text-text-muted font-mono">
          Autonomous Mission Log (Shell Mode)
       </div>
       
       <div class="flex-1 overflow-auto p-8 space-y-8 relative z-10">
          {#each chatHistory as msg}
             <div class="flex {(msg.Role || msg.role) === 'user' ? 'justify-end' : 'justify-start'} items-start gap-5">
                {#if (msg.Role || msg.role) === 'assistant'}
                   <div class="w-10 h-10 rounded-full bg-accent/20 flex items-center justify-center border border-accent/40 shrink-0 shadow-glow-accent/10">
                      <Sparkles size={16} class="text-accent" />
                   </div>
                {/if}
                <div class="max-w-[75%] p-5 rounded-md {(msg.Role || msg.role) === 'user' ? 'bg-surface-3 border border-border-secondary' : 'bg-background/80 backdrop-blur-md border border-border-primary shadow-lg'}">
                   {#if (msg.Role || msg.role) === 'assistant'}
                      <div class="flex items-center gap-2 mb-3">
                         <div class="text-[9px] font-bold text-accent uppercase tracking-widest">Cortex Logic Synthesis</div>
                         <Badge variant="success" size="xs">SECURE</Badge>
                      </div>
                   {/if}
                   <p class="text-[11px] leading-relaxed text-text-secondary {(msg.Role || msg.role) === 'assistant' ? 'font-medium' : 'font-mono'} whitespace-pre-wrap">
                    {msg.Content || msg.text || msg.content}
                   </p>
                </div>
                {#if (msg.Role || msg.role) === 'user'}
                   <div class="w-10 h-10 rounded-full bg-surface-3 flex items-center justify-center border border-border-primary shrink-0 font-bold text-[11px] shadow-sm">
                      M
                   </div>
                {/if}
             </div>
          {/each}
          {#if loading}
             <div class="flex justify-start items-start gap-5 animate-pulse">
                <div class="w-10 h-10 rounded-full bg-accent/10 flex items-center justify-center border border-accent/20 shrink-0">
                   <span class="w-2 h-2 bg-accent rounded-full animate-ping"></span>
                </div>
                <div class="max-w-[75%] p-5 rounded-md bg-background/40 border border-border-dashed italic text-text-muted text-[10px]">
                   AI is synthesizing response...
                </div>
             </div>
          {/if}
       </div>

       <!-- Chat Input Area -->
       <div class="p-6 bg-surface-2/80 backdrop-blur-xl border-t border-border-primary relative z-20">
          <div class="max-w-4xl mx-auto">
             <form class="flex items-center gap-4 bg-background/50 border border-border-primary rounded-md px-5 py-3 shadow-premium focus-within:border-accent transition-all" onsubmit={(e) => { e.preventDefault(); submitQuery(); }}>
                <Terminal size={18} class="text-text-muted" />
                <input 
                  type="text" 
                  class="flex-1 bg-transparent border-none outline-none text-[12px] text-text-primary py-1 placeholder:text-text-muted/30" 
                  placeholder="Issue tactical command to OBLIVRA Cortex..." 
                  bind:value={query}
                  disabled={loading}
                />
                <Button variant="ghost" size="sm" type="submit" disabled={loading} class="hover:bg-accent/10">
                   <Send size={16} class="text-accent {loading ? 'animate-pulse' : ''}" />
                </Button>
             </form>
             <div class="mt-3 flex justify-center gap-6">
                <div class="flex items-center gap-1.5 opacity-40">
                   <kbd class="px-1.5 py-0.5 rounded bg-surface-3 border border-border-primary text-[8px] font-bold text-text-muted uppercase">ESC</kbd>
                   <span class="text-[8px] text-text-muted uppercase font-bold tracking-widest">Clear context</span>
                </div>
                <div class="flex items-center gap-1.5 opacity-40">
                   <kbd class="px-1.5 py-0.5 rounded bg-surface-3 border border-border-primary text-[8px] font-bold text-text-muted uppercase">Enter</kbd>
                   <span class="text-[8px] text-text-muted uppercase font-bold tracking-widest">Execute logic</span>
                </div>
             </div>
          </div>
       </div>
    </div>
  </div>
</PageLayout>
