<!--
  OBLIVRA — Threat Intel Panel (Svelte 5)
  The Intelligence Orbit: Strategic threat landscape and actor profiles.
-->
<script lang="ts">
  import { PageLayout, Badge, Button, DataTable, Input, PopOutButton} from '@components/ui';
  import { Shield, Target, Activity, Zap, ExternalLink, Filter, RefreshCw, AlertTriangle, ChevronRight, Globe, Fingerprint } from 'lucide-svelte';

  let intelSearch = $state('');

  const actors = [
    { name: 'APT-28 (Fancy Bear)', origin: 'Russia', motive: 'Espionage', risk: 'Critical', status: 'Active' },
    { name: 'Lazarus Group', origin: 'North Korea', motive: 'Financial', risk: 'High', status: 'Active' },
    { name: 'Wizard Spider', origin: 'EEU', motive: 'Ransomware', risk: 'Critical', status: 'Dormant' },
    { name: 'Volt Typhoon', origin: 'China', motive: 'Sabotage', risk: 'High', status: 'Active' }
  ];

  const iocFeed = [
    { type: 'IP', value: '185.156.74.55', threat: 'C2 Beacon', time: '12m ago' },
    { type: 'Hash', value: 'a4c5..89e1', threat: 'Trojan.Dropper', time: '45m ago' },
    { type: 'Domain', value: 'secure-update.com', threat: 'Phishing', time: '2h ago' }
  ];
</script>

<PageLayout title="Threat Intelligence" subtitle="Strategic intelligence and real-time IOC correlation mesh">
  {#snippet toolbar()}
    <div class="flex items-center gap-2">
      <Input variant="search" placeholder="Search Intel Mesh..." bind:value={intelSearch} class="w-64" />
      <Button variant="secondary" size="sm" icon={RefreshCw}>SYNC FEEDS</Button>
      <Button variant="primary" size="sm">NEW IOC</Button>
    </div>
      <PopOutButton route="/threat-intel" title="Threat Intelligence" />
    {/snippet}

  <div class="flex flex-col h-full gap-4">
    <!-- INTEL STRIP -->
    <div class="grid grid-cols-4 gap-4 shrink-0">
        <div class="bg-surface-2 border border-border-primary p-4 rounded-sm flex flex-col gap-2 group hover:border-accent transition-colors">
            <div class="flex items-center justify-between text-[9px] font-mono text-text-muted uppercase tracking-widest">
                <span>Active Campaigns</span>
                <Target size={14} class="text-error" />
            </div>
            <div class="text-2xl font-mono font-bold text-text-heading">14</div>
            <div class="h-1 bg-surface-1 rounded-full overflow-hidden">
                <div class="h-full bg-error" style="width: 62%"></div>
            </div>
        </div>
        <div class="bg-surface-2 border border-border-primary p-4 rounded-sm flex flex-col gap-2 group hover:border-accent transition-colors">
            <div class="flex items-center justify-between text-[9px] font-mono text-text-muted uppercase tracking-widest">
                <span>Global Risk Score</span>
                <Shield size={14} class="text-warning" />
            </div>
            <div class="text-2xl font-mono font-bold text-warning">8.4</div>
            <div class="text-[9px] text-text-muted uppercase tracking-tighter">▲ 1.2 from last 24h</div>
        </div>
        <div class="bg-surface-2 border border-border-primary p-4 rounded-sm flex flex-col gap-2 group hover:border-accent transition-colors">
            <div class="flex items-center justify-between text-[9px] font-mono text-text-muted uppercase tracking-widest">
                <span>IOC Hit Rate</span>
                <Activity size={14} class="text-success" />
            </div>
            <div class="text-2xl font-mono font-bold text-success">92.4%</div>
            <div class="text-[9px] text-text-muted uppercase tracking-tighter">Verified matches</div>
        </div>
        <div class="bg-surface-2 border border-border-primary p-4 rounded-sm flex flex-col gap-2 group hover:border-accent transition-colors">
            <div class="flex items-center justify-between text-[9px] font-mono text-text-muted uppercase tracking-widest">
                <span>TTP Matches</span>
                <Zap size={14} class="text-accent" />
            </div>
            <div class="text-2xl font-mono font-bold text-text-heading">1,402</div>
            <div class="text-[9px] text-text-muted uppercase tracking-tighter">MITRE mapped events</div>
        </div>
    </div>

    <!-- MAIN GRID -->
    <div class="flex-1 grid grid-cols-12 gap-4 min-h-0">
        <!-- ACTOR LEDGER -->
        <div class="col-span-8 bg-surface-1 border border-border-primary rounded-sm flex flex-col min-h-0">
            <div class="flex items-center justify-between p-3 border-b border-border-primary bg-surface-2 shrink-0">
                <div class="flex items-center gap-2">
                    <Globe size={14} class="text-accent" />
                    <span class="text-[10px] font-bold text-text-heading uppercase tracking-widest">Global Actor Ledger</span>
                </div>
                <Button variant="ghost" size="xs" icon={Filter}>FILTERS</Button>
            </div>
            <div class="flex-1 overflow-auto mask-fade-bottom">
                <DataTable 
                    data={actors} 
                    columns={[
                        { key: 'name', label: 'ACTOR / CAMPAIGN' },
                        { key: 'origin', label: 'ORIGIN', width: '100px' },
                        { key: 'motive', label: 'MOTIVE', width: '120px' },
                        { key: 'risk', label: 'RISK', width: '80px' },
                        { key: 'status', label: 'STATUS', width: '100px' }
                    ]} 
                    compact
                >
                    {#snippet render({ col, row })}
                        {#if col.key === 'name'}
                            <div class="flex items-center gap-2 py-0.5">
                                <Fingerprint size={12} class="text-accent opacity-60" />
                                <span class="text-[10px] font-bold text-text-heading uppercase">{row.name}</span>
                            </div>
                        {:else if col.key === 'origin'}
                            <span class="text-[9px] font-mono text-text-muted uppercase">{row.origin}</span>
                        {:else if col.key === 'motive'}
                            <span class="text-[9px] font-mono text-text-muted uppercase">{row.motive}</span>
                        {:else if col.key === 'risk'}
                            <Badge variant={row.risk === 'Critical' ? 'critical' : 'warning'} size="xs">{row.risk}</Badge>
                        {:else if col.key === 'status'}
                            <Badge variant={row.status === 'Active' ? 'accent' : 'muted'} size="xs" dot>{row.status}</Badge>
                        {/if}
                    {/snippet}
                </DataTable>
            </div>
        </div>

        <!-- IOC REAL-TIME FEED -->
        <div class="col-span-4 flex flex-col gap-4 min-h-0">
            <div class="bg-surface-2 border border-border-primary rounded-sm flex-1 flex flex-col min-h-0 shadow-premium">
                <div class="flex items-center justify-between p-3 border-b border-border-primary shrink-0">
                    <div class="flex items-center gap-2">
                        <Activity size={14} class="text-error animate-pulse" />
                        <span class="text-[9px] font-mono font-bold text-text-heading uppercase tracking-widest">Real-time IOC Hits</span>
                    </div>
                </div>
                <div class="flex-1 overflow-auto p-3 space-y-3 mask-fade-bottom">
                    {#each iocFeed as ioc}
                        <div class="p-2.5 bg-surface-1 border border-border-primary rounded-sm space-y-2 group hover:border-error transition-colors">
                            <div class="flex justify-between items-start">
                                <Badge variant="info" size="xs" class="text-[8px]">{ioc.type}</Badge>
                                <span class="text-[8px] font-mono text-text-muted opacity-40">{ioc.time}</span>
                            </div>
                            <code class="block text-[10px] font-mono text-accent truncate">{ioc.value}</code>
                            <div class="flex items-center justify-between">
                                <span class="text-[9px] font-mono text-error font-bold uppercase tracking-tighter">{ioc.threat}</span>
                                <ExternalLink size={10} class="text-text-muted cursor-pointer hover:text-text-secondary" />
                            </div>
                        </div>
                    {/each}
                </div>
            </div>

            <div class="bg-surface-3 border border-border-primary p-4 rounded-sm space-y-3 shadow-premium">
                 <div class="flex items-center gap-2 mb-1">
                    <AlertTriangle size={14} class="text-warning" />
                    <span class="text-[9px] font-mono font-bold text-text-muted uppercase tracking-widest">Global Advisories</span>
                 </div>
                 <div class="space-y-2">
                    <div class="flex items-start gap-2 group cursor-pointer">
                        <ChevronRight size={12} class="text-text-muted mt-0.5 group-hover:text-accent transition-transform" />
                        <div class="flex flex-col">
                            <span class="text-[9px] font-bold text-text-secondary uppercase">Zero-Day in Core Mesh Sharding</span>
                            <span class="text-[8px] font-mono text-text-muted">CVE-2026-1042 — CVSS 9.8</span>
                        </div>
                    </div>
                    <div class="flex items-start gap-2 group cursor-pointer">
                        <ChevronRight size={12} class="text-text-muted mt-0.5 group-hover:text-accent transition-transform" />
                        <div class="flex flex-col">
                            <span class="text-[9px] font-bold text-text-secondary uppercase">New APT Campaign: "Iron Veil"</span>
                            <span class="text-[8px] font-mono text-text-muted">Targeting Sovereign Shards</span>
                        </div>
                    </div>
                 </div>
            </div>
        </div>
    </div>
  </div>
</PageLayout>
