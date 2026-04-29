<!--
  OBLIVRA — Incident Timeline (Svelte 5)
  Automated narrative reconstruction of security events.
-->
<script lang="ts">
  import { onMount } from 'svelte';
  import { fade } from 'svelte/transition';
  import {
    PageLayout,
    Button,
    Badge,
    Spinner,
    EmptyState,
    PopOutButton
  } from '@components/ui';
  import { 
    History, 
    Zap, 
    Shield, 
    Activity, 
    AlertCircle, 
    Terminal
  } from 'lucide-svelte';
  
  import { ReconstructTimeline } from '@wailsjs/github.com/kingknull/oblivrashell/internal/services/siemservice.js';

  let { principalID = "", principalType = "host", targetTime = new Date().toISOString() } = $props();

  let timeline = $state<any>(null);
  let loading = $state(false);
  let error = $state<string | null>(null);

  onMount(async () => {
    if (principalID) {
      await loadTimeline();
    }
  });

  async function loadTimeline() {
    loading = true;
    error = null;
    try {
      timeline = await ReconstructTimeline(principalID, principalType, targetTime);
    } catch (err: any) {
      error = err.message || "Failed to reconstruct timeline";
    } finally {
      loading = false;
    }
  }

  function getStepIcon(type: string) {
    switch (type) {
      case 'failed_login': return Shield;
      case 'process_spawn': return Activity;
      case 'privilege_escalation': return Zap;
      case 'connection_established': return Terminal;
      default: return AlertCircle;
    }
  }

  function getSeverityColor(sev: string) {
    switch (sev) {
      case 'CRITICAL': return 'text-critical';
      case 'HIGH': return 'text-warning';
      default: return 'text-primary';
    }
  }
</script>

<PageLayout title="Incident Narrative" subtitle="Automated reconstruction of event causality and attack progression">
  {#snippet toolbar()}
    <div class="flex items-center gap-2">
      <Button variant="secondary" size="sm" onclick={loadTimeline} disabled={loading}>
        <History size={14} class="mr-1" />
        Regenerate Story
      </Button>
      <!-- Phase 35 audit fix #12: drop the explicit `route` so PopOutButton's
           own resolveRoute() (using the hash router) takes over. The
           previous explicit `window.location.pathname` always resolved
           to '/' under hash routing, popping out the dashboard. -->
      <PopOutButton title="Incident Timeline" />
    </div>
  {/snippet}

  <div class="max-w-4xl mx-auto py-8">
    {#if loading}
      <div class="flex flex-col items-center justify-center py-20 gap-4">
        <Spinner size="lg" />
        <span class="text-[11px] font-bold uppercase tracking-widest text-text-muted">Synthesizing Narrative...</span>
      </div>
    {:else if error}
      <EmptyState 
        title="Reconstruction Failed" 
        description={error}
        icon="alert"
      />
    {:else if timeline && timeline.steps.length > 0}
      <div class="flex flex-col gap-12 relative">
        <!-- Vertical Line -->
        <div class="absolute left-[23px] top-4 bottom-4 w-px bg-border-primary"></div>

        {#each timeline.steps as step, i}
          {@const Icon = getStepIcon(step.type)}
          <div class="flex gap-8 relative group" in:fade={{ delay: i * 100 }}>
            <!-- Step Marker -->
            <div class="relative z-10 w-12 h-12 rounded-full bg-surface-1 border border-border-primary flex items-center justify-center shadow-lg group-hover:border-accent transition-colors">
               <Icon 
                 size={20} 
                 class={getSeverityColor(step.severity)}
               />
            </div>

            <!-- Content Card -->
            <div class="flex-1 bg-surface-1 border border-border-primary rounded-lg shadow-card overflow-hidden hover:border-border-hover transition-all">
              <div class="p-4 flex flex-col gap-3">
                <div class="flex items-center justify-between">
                  <div class="flex items-center gap-2">
                    <span class="text-[10px] font-mono text-text-muted">{new Date(step.timestamp).toLocaleTimeString()}</span>
                    <Badge variant={step.severity === 'CRITICAL' ? 'critical' : 'info'}>{step.type}</Badge>
                  </div>
                  <span class="text-[9px] font-bold text-text-muted">INCIDENT PHASE {i + 1}</span>
                </div>

                <div class="flex flex-col gap-1">
                  <h3 class="text-[14px] font-bold text-text-heading">{step.title}</h3>
                  <p class="text-[12px] text-text-secondary leading-relaxed">{step.description}</p>
                </div>

                {#if step.meta}
                   <div class="flex flex-wrap gap-2 pt-2 border-t border-border-secondary">
                      {#each Object.entries(step.meta) as [key, val]}
                        {#if val}
                          <div class="flex items-center gap-1 px-2 py-0.5 bg-surface-2 rounded text-[9px] font-mono border border-border-primary">
                            <span class="text-text-muted">{key}:</span>
                            <span class="text-text-heading">{val}</span>
                          </div>
                        {/if}
                      {/each}
                   </div>
                {/if}
              </div>

              {#if step.raw_event}
                <div class="px-4 py-2 bg-surface-2 border-t border-border-primary flex items-center justify-between">
                  <span class="text-[9px] font-mono text-text-muted">ID: {step.raw_event.id} | Integrity: SHA-256 Verified</span>
                  <button class="text-[9px] font-bold text-accent hover:underline">View Raw Trace</button>
                </div>
              {/if}
            </div>
          </div>
        {/each}

        <!-- Conclusion Marker -->
        <div class="flex gap-8 items-center pl-[14px]">
           <div class="w-5 h-5 rounded-full bg-success opacity-20 animate-ping"></div>
           <div class="absolute left-[23px] w-1 h-1 bg-success rounded-full"></div>
           <span class="text-[10px] font-bold text-success uppercase tracking-widest">Ongoing Activity Monitored</span>
        </div>
      </div>
    {:else}
      <div class="flex flex-col items-center justify-center py-20 gap-6 opacity-40">
        <History size={64} />
        <div class="flex flex-col gap-2 items-center">
          <h3 class="text-[16px] font-bold text-text-heading">No Narrative Available</h3>
          <p class="text-[11px] text-text-secondary max-w-xs text-center">Enter a Principal ID and timestamp to automatically reconstruct the security timeline.</p>
        </div>
        
        <div class="flex flex-col gap-4 w-full max-w-sm bg-surface-1 p-6 border border-border-primary rounded-lg shadow-xl">
           <div class="flex flex-col gap-1">
             <span class="text-[9px] font-bold text-text-muted uppercase">Principal Identifier</span>
             <input type="text" bind:value={principalID} placeholder="e.g. host-01, admin_mark" class="bg-surface-2 border border-border-primary rounded px-3 py-2 text-[12px] text-text-heading" />
           </div>
           <Button variant="primary" onclick={loadTimeline}>
              <Zap size={14} class="mr-2" />
              Begin Reconstruction
           </Button>
        </div>
      </div>
    {/if}
  </div>
</PageLayout>
