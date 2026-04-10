<!--
  OBLIVRA — Entity View (Svelte 5)
  Deep-dive analytics for platform entities: Users, Hosts, and Identities.
-->
<script lang="ts">
  import { KPI, PageLayout, Badge, Button, DataTable } from '@components/ui';
  import { User, Server, Shield, Activity, Zap, Layers, Globe, Cpu, Clock, History } from 'lucide-svelte';
  import { appStore } from '@lib/stores/app.svelte';

  const entity = {
    id: 'E-401',
    name: 'svc-bridge-alpha',
    type: 'Service Account',
    trustScore: 0.94,
    lastActive: 'Just now',
    domain: 'Infrastructure',
  };

  const activity = [
    { id: 1, action: 'SSH Login', host: 'prod-web-01', result: 'Success', time: '2m ago' },
    { id: 2, action: 'Vault Read', target: 'db-creds', result: 'Success', time: '14m ago' },
    { id: 3, action: 'Config Change', target: 'bgp-logic', result: 'Authorized', time: '1h ago' },
  ];
</script>

<PageLayout title="Entity Intelligence" subtitle="Deep-dive analytics: Deconstructing behavioral patterns and trust scores for platform entities">
  {#snippet toolbar()}
     <Button variant="secondary" size="sm">Audit Timeline</Button>
     <Button variant="primary" size="sm" icon="👤">Identity Re-Verify</Button>
  {/snippet}

  <div class="flex flex-col h-full gap-6">
    <!-- Entity Hero -->
    <div class="bg-surface-1 border border-border-primary rounded-md p-6 flex flex-col md:flex-row justify-between items-start md:items-center gap-6 shadow-premium relative overflow-hidden group">
       <div class="absolute inset-0 opacity-[0.02] pointer-events-none grayscale flex items-center justify-center">
          <Layers size={400} />
       </div>

       <div class="flex gap-6 items-center relative z-10">
          <div class="w-16 h-16 rounded-full bg-accent/20 border-2 border-accent/40 flex items-center justify-center shadow-glow-accent/10">
             <User size={32} class="text-accent" />
          </div>
          <div class="flex flex-col">
             <div class="flex items-center gap-2">
                <h2 class="text-xl font-bold text-text-heading">{entity.name}</h2>
                <Badge variant="success" size="xs">ACTIVE</Badge>
             </div>
             <span class="text-[10px] text-text-muted uppercase tracking-widest font-bold font-mono">{entity.id} — {entity.type}</span>
          </div>
       </div>

       <div class="grid grid-cols-2 md:grid-cols-3 gap-6 relative z-10">
          <div class="flex flex-col">
             <span class="text-[9px] text-text-muted uppercase font-bold tracking-widest">Trust Score</span>
             <span class="text-xl font-bold font-mono text-success">{(entity.trustScore * 100).toFixed(0)}%</span>
          </div>
          <div class="flex flex-col">
             <span class="text-[9px] text-text-muted uppercase font-bold tracking-widest">Last Signal</span>
             <span class="text-xl font-bold font-mono text-text-heading">{entity.lastActive}</span>
          </div>
          <div class="flex flex-col hidden md:flex">
             <span class="text-[9px] text-text-muted uppercase font-bold tracking-widest">Region</span>
             <span class="text-xl font-bold font-mono text-text-heading">US-EAST</span>
          </div>
       </div>
    </div>

    <div class="flex-1 min-h-0 grid grid-cols-1 lg:grid-cols-3 gap-6">
       <!-- Activity Timeline -->
       <div class="lg:col-span-2 bg-surface-1 border border-border-primary rounded-md overflow-hidden flex flex-col shadow-premium">
          <div class="p-3 bg-surface-2 border-b border-border-primary flex justify-between items-center text-[10px] font-bold uppercase tracking-widest text-text-muted">
             Evidence Timeline: Behavioral Signals
          </div>
          <div class="flex-1 overflow-auto">
             {#each activity as item}
                <div class="p-4 border-b border-border-primary/50 flex justify-between items-center hover:bg-surface-2/50 transition-colors cursor-pointer group">
                   <div class="flex gap-4 items-center">
                      <div class="w-8 h-8 rounded-full bg-surface-3 flex items-center justify-center opacity-40 group-hover:opacity-100 transition-opacity">
                         <History size={14} />
                      </div>
                      <div class="flex flex-col">
                         <span class="text-[11px] font-bold text-text-heading">{item.action} on {item.host || item.target}</span>
                         <span class="text-[9px] text-text-muted uppercase font-mono">{item.time}</span>
                      </div>
                   </div>
                   <Badge variant={item.result === 'Success' || item.result === 'Authorized' ? 'success' : 'error'} size="xs">{item.result}</Badge>
                </div>
             {/each}
          </div>
       </div>

       <!-- Risk & Context -->
       <div class="flex flex-col gap-6">
          <div class="bg-surface-1 border border-border-primary rounded-md p-6 space-y-4 shadow-sm">
             <div class="text-[10px] font-bold text-text-muted uppercase tracking-widest border-b border-border-primary pb-2 flex items-center gap-2">
                <Activity size={12} />
                Communication Graph
             </div>
             <div class="aspect-square bg-surface-2 border border-border-secondary rounded-md relative flex items-center justify-center group overflow-hidden">
                <div class="absolute inset-0 opacity-10 flex items-center justify-center">
                   <Globe size={180} class="animate-slow-spin" />
                </div>
                <div class="relative z-10 flex flex-col items-center gap-2">
                   <Shield size={32} class="text-accent opacity-40" />
                   <span class="text-xs font-bold text-text-muted uppercase">Mesh Isolated</span>
                </div>
             </div>
          </div>

          <div class="flex-1 bg-surface-1 border border-border-primary rounded-md p-4 space-y-4">
             <div class="text-[10px] font-bold text-text-muted uppercase tracking-widest border-b border-border-primary pb-2">Assigned Capabilities</div>
             <div class="flex flex-wrap gap-2">
                {#each ['Vault.Read', 'BGP.Modify', 'Host.SSH', 'OQL.Deploy'] as cap}
                   <Badge variant="secondary" size="xs">{cap}</Badge>
                {/each}
             </div>
          </div>
       </div>
    </div>
  </div>
</PageLayout>
