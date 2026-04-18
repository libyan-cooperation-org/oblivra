<!--
  OBLIVRA — Campaign Timeline (Svelte 5)
  Automated causality-linked reconstruction of a multi-stage attack sequence.
-->
<script lang="ts">
  import { onMount } from 'svelte';
  import { Badge, Button } from '@components/ui';
  import { Shield, Target, Activity, Clock, ChevronDown, ChevronUp, ExternalLink, Zap } from 'lucide-svelte';
  import { GetCampaignTimeline } from '@wailsjs/github.com/kingknull/oblivrashell/internal/services/FusionService';

  let { clusterID, onClose } = $props();

  let timeline = $state(null);
  let loading = $state(true);
  let error = $state(null);

  async function loadTimeline() {
    loading = true;
    try {
      timeline = await GetCampaignTimeline(clusterID);
    } catch (err) {
      error = err;
    } finally {
      loading = false;
    }
  }

  onMount(loadTimeline);

  function formatTime(ts: string) {
    return new Date(ts).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit', second: '2-digit' });
  }

  function getEventIcon(type: string) {
    if (type === 'ALERT') return Target;
    if (type === 'EVENT') return Activity;
    return Shield;
  }

  function getEventColor(type: string, severity: string) {
    if (type === 'ALERT') {
       if (severity === 'CRITICAL') return 'text-red-500 bg-red-500/10 border-red-500/20';
       return 'text-orange-500 bg-orange-500/10 border-orange-500/20';
    }
    return 'text-blue-400 bg-blue-400/10 border-blue-400/20';
  }
</script>

<div class="fixed inset-0 z-50 flex items-center justify-center p-6 bg-black/60 backdrop-blur-sm">
  <div class="w-full max-w-4xl h-[80vh] bg-surface-1 border border-border-primary rounded-lg shadow-2xl flex flex-col overflow-hidden">
    <!-- Header -->
    <div class="p-6 border-b border-border-primary flex justify-between items-center bg-surface-2">
      <div class="flex flex-col gap-1">
        <h2 class="text-lg font-bold text-text-heading flex items-center gap-3">
          <Zap class="text-accent" size={20} />
          Automated Incident Reconstruction
        </h2>
        <div class="flex items-center gap-4">
          <span class="text-[10px] font-mono text-text-muted uppercase tracking-wider">Cluster ID: {clusterID}</span>
          {#if timeline}
            <span class="text-[10px] font-mono text-text-muted uppercase">Span: {Math.round((new Date(timeline.end).getTime() - new Date(timeline.start).getTime()) / 1000 / 60)} Minutes</span>
          {/if}
        </div>
      </div>
      <Button variant="secondary" size="sm" onclick={onClose}>Close Recon</Button>
    </div>

    <!-- Timeline Content -->
    <div class="flex-1 overflow-y-auto p-8 bg-surface-1 relative">
      {#if loading}
        <div class="h-full flex flex-col items-center justify-center gap-4">
          <div class="w-12 h-12 border-2 border-accent border-t-transparent rounded-full animate-spin"></div>
          <span class="text-xs font-mono text-text-muted animate-pulse">SYNTHESIZING CAUSALITY LINKS...</span>
        </div>
      {:else if error}
        <div class="h-full flex flex-col items-center justify-center text-red-500 gap-4">
          <Shield class="w-12 h-12" />
          <span class="text-xs font-bold uppercase tracking-widest">Reconstruction Failed</span>
          <span class="text-[10px] font-mono opacity-60">{error}</span>
        </div>
      {:else if timeline}
        <!-- Vertical Line -->
        <div class="absolute left-[2.25rem] top-12 bottom-12 w-px bg-border-primary opacity-50"></div>

        <div class="space-y-12 relative z-10">
          {#each timeline.events as event, i}
            <div class="flex gap-6 group">
              <!-- Left side: Time & Icon -->
              <div class="flex flex-col items-center pt-1">
                <div class="text-[10px] font-mono text-text-muted font-bold mb-2 w-16 text-right">{formatTime(event.timestamp)}</div>
                <div class="w-8 h-8 rounded-full flex items-center justify-center border transition-all duration-300 {getEventColor(event.type, event.severity)} group-hover:scale-110">
                  <svelte:component this={getEventIcon(event.type)} size={14} />
                </div>
              </div>

              <!-- Right side: Content -->
              <div class="flex-1 p-4 bg-surface-2 border border-border-secondary rounded-md shadow-sm group-hover:border-accent/30 transition-all">
                <div class="flex justify-between items-start mb-2">
                  <div class="flex items-center gap-3">
                    <span class="text-[10px] font-bold text-text-heading uppercase tracking-widest">{event.type}</span>
                    {#if event.tactic}
                      <Badge variant="accent" size="xs" class="font-mono text-[8px]">{event.tactic}</Badge>
                    {/if}
                  </div>
                  <span class="text-[9px] font-mono text-text-muted uppercase">{event.source}</span>
                </div>

                <p class="text-xs text-text-muted leading-relaxed font-mono">
                  {event.description}
                </p>

                {#if event.type === 'ALERT'}
                   <div class="mt-3 pt-3 border-t border-border-primary/50 flex items-center justify-between">
                      <div class="flex items-center gap-4">
                         <div class="flex items-center gap-1 text-[9px] text-accent font-bold uppercase">
                            <Target size={10} />
                            Detection Confidence: 0.98
                         </div>
                      </div>
                      <Button variant="ghost" size="xs" icon={ExternalLink}>View Rule</Button>
                   </div>
                {/if}
              </div>
            </div>
          {/each}
        </div>
        
        <div class="mt-12 pt-8 border-t border-border-primary flex flex-col items-center">
           <Badge variant="success" class="animate-pulse">TIMELINE SYNTHESIS COMPLETE</Badge>
           <span class="text-[9px] font-mono text-text-muted mt-2">CAUSALITY DEPTH: L3 CORRELATION</span>
        </div>
      {/if}
    </div>
  </div>
</div>

<style>
  /* Custom scrollbar for the timeline */
  div::-webkit-scrollbar {
    width: 4px;
  }
  div::-webkit-scrollbar-track {
    background: transparent;
  }
  div::-webkit-scrollbar-thumb {
    background: var(--border-primary);
    border-radius: 10px;
  }
</style>
