<!--
  OBLIVRA — Incident Response (Svelte 5)
  Real-time response orchestration and containment controls.
-->
<script lang="ts">
  import { KPI, Badge, PageLayout, Button, Toggle } from '@components/ui';
  import { Shield, Zap, Lock, Power, Activity, Crosshair } from 'lucide-svelte';
  import { appStore } from '@lib/stores/app.svelte';

  let isolationActive = $state(false);
  let firewallActive = $state(true);

  const activeResponse = [
    { target: 'prod-web-01', action: 'Traffic Throttling', status: 'executing' },
    { target: 'db-cluster-b', action: 'Snapshot Backup', status: 'completed' },
    { target: '10.0.8.2', action: 'EDR Quarantine', status: 'pending' },
  ];
</script>

<PageLayout title="Strategic Response Center" subtitle="Dynamic containment and threat mitigation controls: Orchestrating real-time adversary eviction">
  {#snippet toolbar()}
    <div class="flex items-center gap-3">
       <Button variant="danger" size="sm" icon="🚨">PANIC LOCK</Button>
       <Button variant="primary" size="sm">New Response Playbook</Button>
    </div>
  {/snippet}

  <div class="flex flex-col h-full gap-6">
    <div class="grid grid-cols-1 md:grid-cols-4 gap-4 shrink-0">
      <KPI label="Active Containments" value={activeResponse.filter(r => r.status === 'executing').length} trend="up" trendValue="High Priority" variant="critical" />
      <KPI label="Edge Isolation" value={isolationActive ? 'ENABLED' : 'OFF'} trend="stable" trendValue="Network Perimeter" variant={isolationActive ? 'critical' : 'success'} />
      <KPI label="MTT-Respond" value="4.2m" trend="down" trendValue="-30s" variant="success" />
      <KPI label="Automated Logic" value="12" trend="stable" trendValue="Verified" variant="success" />
    </div>

    <div class="flex-1 min-h-0 grid grid-cols-1 lg:grid-cols-3 gap-6">
      <!-- Containment Controls -->
      <div class="flex flex-col gap-6">
        <div class="bg-surface-1 border border-border-primary rounded-md p-6 flex flex-col gap-5 shadow-card">
          <div class="text-[10px] font-bold text-text-muted uppercase tracking-widest border-b border-border-primary pb-3 flex items-center gap-2">
             <Shield size={12} class="text-accent" />
             Master Containment Orchestrator
          </div>
          <div class="space-y-6">
             <div class="flex justify-between items-center group">
                <div class="flex flex-col gap-1">
                   <span class="text-[12px] font-bold text-text-heading group-hover:text-accent transition-colors">Global Network Isolation</span>
                   <span class="text-[9px] text-text-muted max-w-[140px] leading-tight">Immediately disconnect all unverified edge nodes from mesh</span>
                </div>
                <Toggle bind:checked={isolationActive} />
             </div>
             <div class="flex justify-between items-center group">
                <div class="flex flex-col gap-1">
                   <span class="text-[12px] font-bold text-text-heading group-hover:text-accent transition-colors">Adaptive Mesh WAF</span>
                   <span class="text-[9px] text-text-muted max-w-[140px] leading-tight">Real-time signature-based blocking at the P2P layer</span>
                </div>
                <Toggle bind:checked={firewallActive} />
             </div>
          </div>
        </div>

        <div class="bg-surface-1 border border-border-primary border-dashed rounded-md p-5 flex flex-col gap-3">
           <Button variant="secondary" class="w-full flex items-center justify-center gap-2 hover:bg-error/5 hover:border-error/40 transition-all">
              <Power size={14} class="text-error" />
              KILL NON-ESSENTIAL SUBSYSTEMS
           </Button>
           <Button variant="secondary" class="w-full flex items-center justify-center gap-2">
              <Lock size={14} class="text-accent" />
              FORCE UNIVERSAL MFA RE-AUTH
           </Button>
        </div>
      </div>

      <!-- Live Response Ledger -->
      <div class="lg:col-span-2 bg-surface-1 border border-border-primary rounded-md overflow-hidden flex flex-col shadow-card relative">
        <div class="p-3 bg-surface-2 border-b border-border-primary flex justify-between items-center text-[10px] font-bold uppercase tracking-widest text-text-muted">
           Tactical Response Orchestration Log
           <Badge variant="critical" size="xs">LIVE FEED</Badge>
        </div>
        <div class="flex-1 p-4 space-y-4 overflow-y-auto">
           {#each activeResponse as res}
              <div class="flex items-center justify-between p-4 bg-surface-2 rounded-md border border-border-secondary hover:border-accent/40 transition-all group">
                 <div class="flex items-center gap-5">
                    <div class="w-2.5 h-2.5 rounded-full shrink-0
                      {res.status === 'executing' ? 'bg-error animate-pulse' : res.status === 'completed' ? 'bg-success' : 'bg-text-muted opacity-40'}">
                    </div>
                    <div class="flex flex-col gap-1">
                       <span class="text-[11px] font-bold text-text-heading group-hover:text-accent transition-colors uppercase tracking-tight">{res.action}</span>
                       <div class="flex items-center gap-2 text-[9px] text-text-muted font-mono uppercase tracking-widest font-bold">
                          <span class="opacity-60">TARGET:</span>
                          <span class="text-text-heading">{res.target}</span>
                       </div>
                    </div>
                 </div>
                 <div class="flex items-center gap-3">
                    <Badge
                      variant={res.status === 'executing' ? 'critical' : res.status === 'completed' ? 'success' : 'muted'}
                      size="xs"
                      dot={res.status === 'executing'}
                    >
                       {res.status.toUpperCase()}
                    </Badge>
                    <Button variant="ghost" size="xs">Abort</Button>
                 </div>
              </div>
           {:else}
              <div class="flex-1 flex flex-col items-center justify-center h-full opacity-20 py-20">
                 <Crosshair size={48} class="mb-4" />
                 <span class="text-xs font-bold uppercase tracking-widest">No Active Missions</span>
              </div>
           {/each}
        </div>
      </div>
    </div>
  </div>
</PageLayout>
