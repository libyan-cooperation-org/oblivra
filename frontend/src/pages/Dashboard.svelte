<!--
  OBLIVRA — Dashboard (Svelte 5)
  The Command Hub: Real-time platform status and mission critical telemetry.
-->
<script lang="ts">
  import { Shield, Activity, RefreshCw } from 'lucide-svelte';
  import { appStore } from '@lib/stores/app.svelte';
  import { onMount } from 'svelte';
  import { KPI, Badge, Button, PageLayout } from '@components/ui';
  import { IS_BROWSER } from '@lib/context';

  const activeSessionCount = $derived(appStore.sessions.filter(s => s.status === 'active').length);

  // State
  let health = $state<any>({});
  let siemStats = $state<any>({});
  let loading = $state(false);

  async function refreshDashboard() {
    if (IS_BROWSER) return;
    loading = true;
    try {
      const { GetAllHealth } = await import('@wailsjs/go/services/HealthService.js');
      const { GetGlobalThreatStats } = await import('@wailsjs/go/services/SIEMService.js');
      
      const [h, s] = await Promise.all([
        GetAllHealth(),
        GetGlobalThreatStats()
      ]);
      health = h || {};
      siemStats = s || {};
    } catch (err) {
      console.error('Dashboard refresh failed', err);
    } finally {
      loading = false;
    }
  }

  onMount(() => {
    refreshDashboard();
    const interval = setInterval(refreshDashboard, 30000); // 30s refresh
    return () => clearInterval(interval);
  });
</script>

<PageLayout title="OBLIVRA Command" subtitle="Sovereign Security Operations Platform">
  {#snippet toolbar()}
     <div class="flex items-center gap-3">
        <Badge variant="accent" dot>CLUSTER SYNC: OK</Badge>
        <Button variant="secondary" size="sm">Download Audit</Button>
     </div>
  {/snippet}

  <div class="flex flex-col h-full gap-6">
    <!-- Primary KPI Strip -->
    <div class="grid grid-cols-1 md:grid-cols-4 gap-4 shrink-0">
      <KPI label="Active Alerts" value={health.AlertCount || 0} trend="up" trendValue="+2 new" variant="critical" />
      <KPI label="Engine Status" value={health.Status || 'Operational'} trend="stable" trendValue="Nominal" variant="success" />
      <KPI label="Storage Capacity" value="{siemStats.StorageUsage || '42'}GB" trend="stable" />
      <KPI label="Events / Second" value={siemStats.EPS || '1.2k'} trend="stable" variant="accent" />
    </div>

    <div class="flex-1 min-h-0 grid grid-cols-1 lg:grid-cols-3 gap-6">
      <!-- Activity Feed -->
      <div class="bg-surface-1 border border-border-primary rounded-md flex flex-col shadow-premium">
        <div class="p-3 bg-surface-2 border-b border-border-primary flex justify-between items-center">
           <div class="text-[10px] font-bold uppercase tracking-widest text-text-muted flex items-center gap-2">
              <Activity size={12} />
              Real-time System Audit
           </div>
           <Button variant="secondary" size="sm" onclick={refreshDashboard}>
            <RefreshCw size={14} class="mr-1 {loading ? 'animate-spin' : ''}" />
            Refresh Stack
          </Button>
        </div>
        <div class="flex-1 overflow-auto p-2 space-y-1">
           {#each appStore.notifications.slice(0, 5) as item}
              <div class="flex items-start gap-3 p-2 hover:bg-surface-2 rounded-sm transition-all group cursor-pointer border border-transparent hover:border-border-secondary">
                 <div class="mt-0.5 p-1.5 bg-surface-3 rounded-full {item.type === 'error' ? 'text-error' : 'text-accent'}">
                    <Activity size={12} />
                 </div>
                 <div class="flex-1 flex flex-col">
                    <span class="text-[11px] font-bold text-text-heading group-hover:text-accent transition-colors">{item.message}</span>
                    <span class="text-[9px] text-text-muted font-mono">{item.details || 'System Broadcast'}</span>
                 </div>
              </div>
           {/each}
        </div>
      </div>

      <!-- Center Visualization -->
      <div class="lg:col-span-2 flex flex-col gap-6">
         <!-- Status Board -->
         <div class="bg-surface-1 border border-border-primary rounded-md p-6 h-full relative overflow-hidden group">
            <div class="absolute inset-0 bg-gradient-to-tr from-accent/5 to-transparent opacity-50"></div>
            
            <div class="relative z-10 flex flex-col h-full">
               <div class="flex justify-between items-start mb-8">
                  <div class="flex flex-col">
                     <h3 class="text-xl font-bold text-text-heading tracking-tight">Platform Integrity</h3>
                     <p class="text-[10px] text-text-muted uppercase tracking-widest">Global operational state verified</p>
                  </div>
                  <div class="flex items-center gap-4">
                     <div class="flex flex-col items-end">
                        <span class="text-[9px] text-text-muted">UPTIME</span>
                        <span class="text-xs font-mono font-bold text-accent">99.999%</span>
                     </div>
                     <div class="w-12 h-12 rounded-full border-2 border-accent/20 flex items-center justify-center">
                        <Shield class="text-accent" size={24} />
                     </div>
                  </div>
               </div>

                <div class="grid grid-cols-3 gap-8 mt-auto">
                   <div class="flex flex-col gap-1">
                      <span class="text-[9px] text-text-muted font-bold uppercase tracking-widest">Compute Load</span>
                      <div class="flex items-end gap-2">
                         <span class="text-2xl font-bold font-mono">{healthScores.compute?.usage || '14'}%</span>
                         <Badge variant={healthScores.compute?.status || 'success'} size="xs">NOMINAL</Badge>
                      </div>
                   </div>
                   <div class="flex flex-col gap-1">
                      <span class="text-[9px] text-text-muted font-bold uppercase tracking-widest">Net Latency</span>
                      <div class="flex items-end gap-2">
                         <span class="text-2xl font-bold font-mono">{healthScores.network?.latency || '2'}ms</span>
                         <Badge variant={healthScores.network?.status || 'success'} size="xs">IDEAL</Badge>
                      </div>
                   </div>
                   <div class="flex flex-col gap-1">
                      <span class="text-[9px] text-text-muted font-bold uppercase tracking-widest">Active I/O</span>
                      <div class="flex items-end gap-2">
                         <span class="text-2xl font-bold font-mono">{healthScores.storage?.throughput || '4.2'}<span class="text-xs">MB/s</span></span>
                         <Badge variant={healthScores.storage?.status || 'info'} size="xs">BURSTING</Badge>
                      </div>
                   </div>
                </div>
            </div>

            <!-- Fake Graph Wireframe -->
            <div class="absolute bottom-0 right-0 w-64 h-32 opacity-10 blur-sm pointer-events-none">
               <svg viewBox="0 0 100 100" class="w-full h-full text-accent fill-current">
                  <path d="M0 80 Q 20 20, 40 50 T 80 10 T 100 80 V 100 H 0 Z" />
               </svg>
            </div>
         </div>
      </div>
    </div>
  </div>
</PageLayout>
