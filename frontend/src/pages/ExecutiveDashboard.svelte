<!--
  OBLIVRA — Executive Dashboard (Svelte 5)
  High-level security posture and strategic mission metrics.
-->
<script lang="ts">
  import { KPI, PageLayout, Badge, Button } from '@components/ui';
  import { Shield, Target, Activity, Zap, TrendingUp, Globe, Database } from 'lucide-svelte';
  import { appStore } from '@lib/stores/app.svelte';

  const riskMetrics = [
    { label: 'Platform Resilience', value: '94%', goal: '98%', status: 'nominal' },
    { label: 'Credential Hygiene', value: '82%', goal: '95%', status: 'warning' },
    { label: 'Endpoint Coverage', value: '100%', goal: '100%', status: 'success' },
    { label: 'MTTD (Mean Time to Detect)', value: '4.2m', goal: '< 5m', status: 'success' },
    { label: 'MTTR (Mean Time to Respond)', value: '1.4h', goal: '< 1h', status: 'warning' },
  ];
</script>

<PageLayout title="Strategic Posture" subtitle="Executive oversight: Global platform resilience and mission-set efficiency">
  {#snippet toolbar()}
     <div class="flex items-center gap-2">
       <Button variant="secondary" size="sm" icon="📊">Export Report (PDF)</Button>
       <Button variant="primary" size="sm">Audit Historical State</Button>
     </div>
  {/snippet}

  <div class="flex flex-col h-full gap-6">
    <div class="grid grid-cols-1 md:grid-cols-4 gap-4">
      <KPI label="Global Risk Score" value="0.12" trend="down" trendValue="-0.04" variant="success" />
      <KPI label="Active War-Sets" value="4" trend="stable" trendValue="Prioritized" variant="accent" />
      <KPI label="Detection Efficiency" value="88%" trend="up" trendValue="+4%" />
      <KPI label="Compliance Level" value="92%" trend="stable" trendValue="Synced" variant="success" />
    </div>

    <div class="flex-1 min-h-0 grid grid-cols-1 lg:grid-cols-3 gap-6">
      <!-- Strategy Board -->
      <div class="lg:col-span-2 bg-surface-1 border border-border-primary rounded-md p-6 flex flex-col shadow-premium gap-6">
         <div class="text-[10px] font-bold text-text-muted uppercase tracking-widest border-b border-border-primary pb-2 flex justify-between items-center">
            Strategic Resilience Metrics
            <Badge variant="info" size="xs">Updated real-time</Badge>
         </div>
         
         <div class="flex-1 grid grid-cols-1 md:grid-cols-2 gap-8 content-start">
            {#each riskMetrics as metric}
               <div class="flex flex-col gap-2 p-4 bg-surface-2 border border-border-secondary rounded-md">
                  <div class="flex justify-between items-center">
                     <span class="text-xs font-bold text-text-heading">{metric.label}</span>
                     <Badge variant={metric.status === 'success' ? 'success' : metric.status === 'warning' ? 'warning' : 'info'} size="xs">
                        {metric.status.toUpperCase()}
                     </Badge>
                  </div>
                  <div class="flex items-end justify-between">
                     <div class="flex flex-col">
                        <span class="text-2xl font-bold font-mono tracking-tight">{metric.value}</span>
                        <span class="text-[9px] text-text-muted uppercase tracking-widest">Goal: {metric.goal}</span>
                     </div>
                     <div class="w-24 h-6 opacity-30">
                        <!-- Tiny trend sparkline -->
                        <svg viewBox="0 0 100 20" class="w-full h-full text-accent stroke-current fill-none stroke-2">
                           <path d="M0 15 L 20 8 L 40 12 L 60 4 L 80 10 L 100 2" stroke-linecap="round" />
                        </svg>
                     </div>
                  </div>
                  <div class="w-full bg-surface-3 h-1 rounded-full overflow-hidden mt-1">
                     <div class="h-full {metric.status === 'success' ? 'bg-success' : metric.status === 'warning' ? 'bg-warning' : 'bg-accent'}" style="width: {metric.value.includes('%') ? metric.value : '100%'}"></div>
                  </div>
               </div>
            {/each}
         </div>
      </div>

      <!-- High-Level Visuals -->
      <div class="flex flex-col gap-6">
         <div class="bg-surface-1 border border-border-primary rounded-md p-6 flex flex-col items-center justify-center text-center gap-4 relative overflow-hidden group">
            <div class="absolute inset-0 bg-accent/5 opacity-50"></div>
            <Globe class="text-accent opacity-20" size={120} />
            <div class="absolute inset-0 flex flex-col items-center justify-center">
               <span class="text-3xl font-bold text-text-heading font-mono tracking-tight">1.2<span class="text-sm">PB</span></span>
               <span class="text-[10px] text-text-muted uppercase tracking-widest font-bold">Encrypted Data Mass</span>
            </div>
         </div>

         <div class="flex-1 bg-surface-1 border border-border-primary rounded-md p-4 space-y-4">
            <div class="text-[10px] font-bold text-text-muted uppercase tracking-widest border-b border-border-primary pb-2 flex items-center gap-2">
               <TrendingUp size={12} />
               Platform Growth (30D)
            </div>
            <div class="flex-1 h-32 flex items-end justify-between px-2 gap-2">
               {#each Array(10) as _, i}
                  <div class="flex-1 bg-accent/20 rounded-t-sm hover:bg-accent/40 transition-colors border-x border-t border-accent/30" style="height: {20 + Math.random() * 80}%"></div>
               {/each}
            </div>
            <div class="flex justify-between text-[9px] text-text-muted font-mono uppercase px-1">
               <span>March</span>
               <span>April</span>
            </div>
         </div>
      </div>
    </div>
  </div>
</PageLayout>
