<!--
  OBLIVRA — Escalation Center (Svelte 5)
  Mission-critical escalation management and emergency protocol orchestration.
-->
<script lang="ts">
  import { KPI, Badge, PageLayout, Button } from '@components/ui';
  import { Zap, Phone, Activity } from 'lucide-svelte';
  import { appStore } from '@lib/stores/app.svelte';

  const escalationPaths = [
    { tier: 1, team: 'SOC Analysts', method: 'Slack/PagerDuty', status: 'engaged' },
    { tier: 2, team: 'SRE / DevSecOps', method: 'Emergency Bridge', status: 'standby' },
    { tier: 3, team: 'CISO / Legal', method: 'Encrypted Voice', status: 'dormant' },
  ];
</script>

<PageLayout title="Escalation Command" subtitle="Mission-critical communication tiers and containment escalation protocols">
  {#snippet toolbar()}
    <div class="flex items-center gap-2">
      <Button variant="danger" size="sm" icon="🚨">ENGAGE WAR MODE</Button>
      <Button variant="primary" size="sm">Modify Call Tree</Button>
    </div>
  {/snippet}

  <div class="flex flex-col h-full gap-6">
    <div class="grid grid-cols-1 md:grid-cols-4 gap-4 shrink-0">
      <KPI label="Active Bridges" value="1" trend="stable" trendValue="Tier 1" variant="accent" />
      <KPI label="Response SLA" value="12m" trend="down" trendValue="-4m" variant="success" />
      <KPI label="Triage Load" value="Normal" trend="stable" trendValue="Nominal" />
      <KPI label="System Trust" value="100%" trend="stable" trendValue="Verified" variant="success" />
    </div>

    <div class="flex-1 min-h-0 grid grid-cols-1 lg:grid-cols-2 gap-6">
      <!-- Escalation Tiers -->
      <div class="bg-surface-1 border border-border-primary rounded-md overflow-hidden flex flex-col shadow-card">
         <div class="p-3 bg-surface-2 border-b border-border-primary text-[10px] font-bold uppercase tracking-widest text-text-muted font-mono">
            Operational Escalation Tiers
         </div>
         <div class="flex-1 p-6 space-y-5 overflow-auto">
            {#each escalationPaths as path}
               <div class="relative flex items-center justify-between p-5 bg-surface-2 border rounded-md transition-colors
                 {path.status === 'engaged' ? 'border-accent' : 'border-border-secondary hover:border-border-primary'} group">
                  {#if path.status === 'engaged'}
                    <div class="absolute -top-2.5 left-4 px-2 bg-accent text-[8px] font-bold uppercase rounded-sm text-white tracking-widest font-mono">active engagement</div>
                  {/if}
                  <div class="flex items-center gap-5">
                     <div class="w-12 h-12 rounded-full bg-surface-3 flex items-center justify-center border border-border-primary">
                        <span class="text-[12px] font-bold font-mono text-text-heading">T{path.tier}</span>
                     </div>
                     <div class="flex flex-col gap-1">
                        <span class="text-[13px] font-bold text-text-heading">{path.team}</span>
                        <div class="flex items-center gap-2 text-[9px] text-text-muted uppercase font-bold tracking-tighter opacity-70">
                           <Phone size={10} class="text-accent" />
                           <span>{path.method}</span>
                        </div>
                     </div>
                  </div>
                  <Badge
                    variant={path.status === 'engaged' ? 'accent' : path.status === 'standby' ? 'info' : 'muted'}
                    size="sm"
                    dot={path.status === 'engaged'}
                  >
                     {path.status.toUpperCase()}
                  </Badge>
               </div>
            {/each}
         </div>
      </div>

      <!-- Emergency Actions -->
      <div class="flex flex-col gap-6">
         <div class="bg-surface-1 border border-border-primary rounded-md p-8 flex flex-col gap-5 text-center items-center justify-center relative overflow-hidden group shadow-card hover:border-error/40 transition-all">
            <div class="absolute inset-0 bg-error/5 group-hover:bg-error/10 transition-colors pointer-events-none"></div>
            <div class="relative z-10 w-20 h-20 rounded-full bg-error/20 flex items-center justify-center border-2 border-error/40 animate-pulse">
               <Zap size={40} class="text-error" />
            </div>
            <div class="relative z-10 space-y-2">
               <h4 class="text-lg font-bold text-text-heading tracking-tight uppercase">Emergency Containment</h4>
               <p class="text-[10px] text-text-muted max-w-sm leading-relaxed px-4">
                  Engaging protocols will trigger global network isolation and force administrative re-authentication with hardware keys.
               </p>
            </div>
            <div class="grid grid-cols-2 gap-4 w-full relative z-10 px-4">
               <Button variant="secondary" class="border-error/50 hover:bg-error/10 text-error">KILL ALL TUNNELS</Button>
               <Button variant="secondary">LOCK WORKSPACE</Button>
            </div>
         </div>

         <div class="flex-1 bg-surface-1 border border-border-primary rounded-md p-6 flex flex-col gap-4 shadow-card">
            <div class="text-[10px] font-bold text-text-muted uppercase tracking-widest border-b border-border-primary pb-3 flex items-center gap-2">
               <Activity size={12} />
               Last Incident Timeline
            </div>
            <div class="flex-1 flex flex-col gap-5 overflow-y-auto">
               {#each Array(3) as _, i}
                  <div class="flex gap-5 items-start group">
                     <div class="w-1.5 h-1.5 rounded-full bg-border-secondary mt-1.5 group-hover:bg-accent transition-colors shrink-0"></div>
                     <div class="flex flex-col gap-1">
                        <span class="text-[11px] font-bold text-text-heading group-hover:text-accent transition-colors">Escalation to T{3 - i} initiated — Logic confirmed</span>
                        <span class="text-[9px] text-text-muted font-mono uppercase tracking-widest font-bold opacity-60">2026-04-10 12:{42 - i * 5}:00</span>
                     </div>
                  </div>
               {/each}
            </div>
         </div>
      </div>
    </div>
  </div>
</PageLayout>
