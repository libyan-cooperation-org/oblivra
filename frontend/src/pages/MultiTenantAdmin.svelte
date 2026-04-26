<!--
  OBLIVRA — Multi-Tenant Admin (Svelte 5)
  Super-admin control plane for sovereign cluster orchestration.
-->
<script lang="ts">
  import { PageLayout, Badge, Button, KPI, DataTable } from '@components/ui';
  import { Shield, Activity, Database, Users, Server, HardDrive, Key, History, Zap, ShieldAlert } from 'lucide-svelte';
  import { appStore } from '@lib/stores/app.svelte';
  import { tenantStore } from '@lib/stores/tenant.svelte.ts';

  const stats = $derived(tenantStore.platformMetrics);
  const tenants = $derived(tenantStore.tenants);

  let selectedTenant = $state(tenants[0]);

  const navItems = [
    { label: 'TENANTS', type: 'header' },
    { label: 'Tenant Overview', icon: Activity, active: true },
    { label: 'New Tenant', icon: Zap },
    { label: 'Isolation Audit', icon: Shield },
    { label: 'PLATFORM', type: 'header' },
    { label: 'Cluster Health', icon: Server },
    { label: 'License Usage', icon: Database },
    { label: 'Ingestion Pipeline', icon: Activity },
    { label: 'Storage Quotas', icon: HardDrive },
    { label: 'SECURITY', type: 'header' },
    { label: 'Global RBAC', icon: Users },
    { label: 'Admin Audit Trail', icon: History },
    { label: 'HSM / Key Mgmt', icon: Key },
  ];
</script>

<PageLayout title="Multi-Tenant Control Plane" subtitle="Sovereign cluster orchestration and resource management">
  {#snippet toolbar()}
    <div class="flex items-center gap-2">
      <Badge variant="accent" class="font-mono text-[9px] uppercase tracking-tighter">Super Admin Mode</Badge>
      <div class="h-4 w-px bg-border-primary mx-2"></div>
      <div class="flex gap-1">
        <Badge variant="muted" class="text-[9px]">⌘T</Badge>
        <Badge variant="muted" class="text-[9px]">⌘N</Badge>
      </div>
    </div>
  {/snippet}

  <div class="flex h-full gap-0 -m-6">
    <!-- LEFT NAV -->
    <div class="w-52 border-r border-border-primary bg-surface-1 flex flex-col shrink-0">
        <div class="px-3 py-2 bg-surface-2 border-b border-border-primary text-[8px] font-mono text-text-muted uppercase tracking-widest">
            Admin Navigation
        </div>
        <div class="flex-1 overflow-auto py-2">
            {#each navItems as item}
                {#if item.type === 'header'}
                    <div class="px-4 py-2 mt-2 text-[8px] font-mono font-bold text-text-muted uppercase tracking-widest">{item.label}</div>
                {:else}
                    <button 
                        class="w-full px-4 py-1.5 flex items-center gap-2.5 text-[10px] transition-colors border-l-2 {item.active ? 'bg-surface-3 text-text-heading border-accent' : 'text-text-muted hover:text-text-secondary border-transparent'}"
                    >
                        {#if item.icon}
                            <item.icon size={12} class={item.active ? 'text-accent' : ''} />
                        {/if}
                        {item.label}
                    </button>
                {/if}
            {/each}
        </div>
    </div>

    <!-- CENTER CONTENT -->
    <div class="flex-1 flex flex-col min-w-0 border-r border-border-primary">
        <!-- METRICS -->
        <div class="grid grid-cols-4 gap-px bg-border-primary border-b border-border-primary shrink-0">
            <div class="bg-surface-2 p-3">
                <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">Active Tenants</div>
                <div class="text-xl font-mono font-bold text-text-heading">{stats.activeTenants}</div>
                <div class="text-[9px] text-text-muted mt-1">2 sovereign · 2 cloud</div>
            </div>
            <div class="bg-surface-2 p-3">
                <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">Total Agents</div>
                <div class="text-xl font-mono font-bold text-text-heading">{stats.totalAgents.toLocaleString()}</div>
                <div class="text-[9px] text-success mt-1">98.1% online</div>
            </div>
            <div class="bg-surface-2 p-3">
                <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">Platform EPS</div>
                <div class="text-xl font-mono font-bold text-success">{stats.platformEps}</div>
                <div class="text-[9px] text-text-muted mt-1">Combined ingestion</div>
            </div>
            <div class="bg-surface-2 p-3">
                <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-1">Active Incidents</div>
                <div class="text-xl font-mono font-bold text-error">{stats.totalIncidents}</div>
                <div class="text-[9px] text-error/80 mt-1 uppercase animate-pulse">Critical action</div>
            </div>
        </div>

        <!-- ROSTER HEADER -->
        <div class="px-4 py-2 bg-surface-1 border-b border-border-primary flex items-center justify-between shrink-0">
            <span class="text-[9px] font-mono font-bold text-text-muted uppercase tracking-widest">Tenant Roster</span>
            <div class="flex gap-2">
                <Button variant="secondary" size="xs" class="text-[9px]">FILTER</Button>
                <Button variant="primary" size="xs" class="text-[9px]">+ PROVISION TENANT</Button>
            </div>
        </div>

        <!-- ROSTER GRID -->
        <div class="flex-1 overflow-auto p-4 grid grid-cols-1 md:grid-cols-2 gap-3 content-start bg-surface-0">
            {#each tenants as tenant}
                <button 
                    class="bg-surface-2 border border-border-primary rounded-sm overflow-hidden text-left hover:bg-surface-3 transition-all {selectedTenant.id === tenant.id ? 'border-accent/40 ring-1 ring-accent/20' : ''}"
                    onclick={() => selectedTenant = tenant}
                >
                    <div class="p-3 border-b border-border-primary flex items-center gap-3">
                        <div class="w-8 h-8 rounded-sm flex items-center justify-center font-mono font-bold text-[11px]" style="background: {tenant.color}15; color: {tenant.color}">
                            {tenant.abbr}
                        </div>
                        <div class="flex-1 min-w-0">
                            <div class="text-[11px] font-bold text-text-heading truncate">{tenant.name}</div>
                            <div class="text-[9px] font-mono text-text-muted">{tenant.id}</div>
                        </div>
                        <div class="flex gap-1.5">
                            <Badge variant={tenant.mode === 'SOVEREIGN' ? 'accent' : 'muted'} size="xs" class="text-[8px]">{tenant.mode}</Badge>
                            {#if tenant.incidents > 0}
                                <Badge variant="critical" size="xs" class="text-[8px]">{tenant.incidents} INC</Badge>
                            {/if}
                        </div>
                    </div>
                    <div class="p-3 grid grid-cols-4 gap-2">
                        <div class="text-center">
                            <div class="text-[10px] font-mono font-bold text-text-heading">{tenant.agents}</div>
                            <div class="text-[7px] font-mono text-text-muted uppercase">Agents</div>
                        </div>
                        <div class="text-center">
                            <div class="text-[10px] font-mono font-bold text-text-heading">{tenant.eps}</div>
                            <div class="text-[7px] font-mono text-text-muted uppercase">EPS</div>
                        </div>
                        <div class="text-center">
                            <div class="text-[10px] font-mono font-bold text-text-heading">{tenant.storage}</div>
                            <div class="text-[7px] font-mono text-text-muted uppercase">Storage</div>
                        </div>
                        <div class="text-center">
                            <div class="text-[10px] font-mono font-bold text-success">98.4%</div>
                            <div class="text-[7px] font-mono text-text-muted uppercase">Health</div>
                        </div>
                    </div>
                    <div class="px-3 pb-3">
                        <div class="h-1 w-full bg-border-primary rounded-full overflow-hidden">
                            <div class="h-full bg-accent" style="width: 65%"></div>
                        </div>
                    </div>
                </button>
            {/each}
        </div>
    </div>

    <!-- RIGHT PANEL -->
    <div class="w-64 bg-surface-1 flex flex-col shrink-0">
        <div class="px-3 py-2 bg-surface-2 border-b border-border-primary text-[8px] font-mono text-text-muted uppercase tracking-widest">
            Selected Tenant
        </div>
        <div class="flex-1 overflow-auto">
            <div class="p-4 border-b border-border-primary space-y-4">
                <div>
                    <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-2">{selectedTenant.id} · DETAIL</div>
                    <div class="space-y-2">
                        <div class="flex justify-between text-[10px]">
                            <span class="text-text-muted">Mode</span>
                            <span class="font-mono text-accent font-bold">{selectedTenant.mode}</span>
                        </div>
                        <div class="flex justify-between text-[10px]">
                            <span class="text-text-muted">Classification</span>
                            <span class="font-mono text-text-heading">SECRET</span>
                        </div>
                        <div class="flex justify-between text-[10px]">
                            <span class="text-text-muted">Region</span>
                            <span class="font-mono text-text-heading">EU-GOV-DC-01</span>
                        </div>
                        <div class="flex justify-between text-[10px]">
                            <span class="text-text-muted">Active Incidents</span>
                            <span class="font-mono text-error font-bold">{selectedTenant.incidents} CRITICAL</span>
                        </div>
                    </div>
                </div>
            </div>

            <div class="p-4 border-b border-border-primary">
                <div class="text-[8px] font-mono text-text-muted uppercase tracking-widest mb-3">RESOURCE QUOTAS</div>
                <div class="space-y-4">
                    <div class="space-y-1.5">
                        <div class="flex justify-between text-[9px] font-mono">
                            <span class="text-text-muted">Agents</span>
                            <span class="text-text-heading">4,900 / 8,000</span>
                        </div>
                        <div class="h-1 w-full bg-border-primary rounded-full overflow-hidden">
                            <div class="h-full bg-accent" style="width: 61%"></div>
                        </div>
                    </div>
                    <div class="space-y-1.5">
                        <div class="flex justify-between text-[9px] font-mono">
                            <span class="text-text-muted">Storage</span>
                            <span class="text-text-heading">18.4 TB / 50 TB</span>
                        </div>
                        <div class="h-1 w-full bg-border-primary rounded-full overflow-hidden">
                            <div class="h-full bg-success" style="width: 37%"></div>
                        </div>
                    </div>
                    <div class="space-y-1.5">
                        <div class="flex justify-between text-[9px] font-mono">
                            <span class="text-text-muted">EPS Ingestion</span>
                            <span class="text-text-heading">148K / 250K</span>
                        </div>
                        <div class="h-1 w-full bg-border-primary rounded-full overflow-hidden">
                            <div class="h-full bg-success" style="width: 59%"></div>
                        </div>
                    </div>
                </div>
            </div>

            <div class="p-4 bg-surface-2 m-4 border border-border-primary rounded-sm space-y-3">
                <div class="flex items-center gap-2 text-success">
                    <Shield size={12} />
                    <span class="text-[8px] font-mono font-bold tracking-widest uppercase">Isolation Seals</span>
                </div>
                <div class="space-y-2">
                    <div class="flex items-center justify-between text-[9px]">
                        <span class="text-text-muted">Data Namespace</span>
                        <span class="text-success font-bold font-mono">VERIFIED</span>
                    </div>
                    <div class="flex items-center justify-between text-[9px]">
                        <span class="text-text-muted">Network Segment</span>
                        <span class="text-success font-bold font-mono">VERIFIED</span>
                    </div>
                    <div class="flex items-center justify-between text-[9px]">
                        <span class="text-text-muted">Encryption Keys</span>
                        <span class="text-success font-bold font-mono">VERIFIED</span>
                    </div>
                </div>
            </div>
        </div>
    </div>
  </div>

  <div class="bg-surface-2 border-t border-border-primary px-3 py-1 flex items-center gap-4 text-[8px] font-mono text-text-muted shrink-0">
    <span>CLUSTER: <span class="text-success font-bold">HEALTHY</span></span>
    <span class="text-border-primary">|</span>
    <span>NODES: <span class="text-success">12/12 ONLINE</span></span>
    <span class="text-border-primary">|</span>
    <span>TOTAL EPS: <span class="text-success">441K</span></span>
    <div class="ml-auto text-accent font-bold">SOVEREIGN CLUSTER · v2.14.1</div>
  </div>
</PageLayout>

<style>
  ::-webkit-scrollbar { width: 3px; }
  ::-webkit-scrollbar-thumb { background: var(--border-primary); border-radius: 1px; }
</style>
