<!-- OBLIVRA Web — EnrichmentViewer (Svelte 5) -->
<script lang="ts">
  import { onMount } from 'svelte';
  import { KPI, Badge, PageLayout, Button, DataTable, SearchBar, Spinner, ProgressBar } from '@components/ui';
  import { Globe, Shield, Database, Zap, Activity, Target, History, Info } from 'lucide-svelte';
  import { request } from '../services/api';

  // -- Types --
  interface GeoResult {
    ip: string;
    country_code: string;
    country_name: string;
    city: string;
    latitude: number;
    longitude: number;
    asn: string;
    org: string;
    isp: string;
  }
  interface DNSResult {
    hostname: string;
    ptr: string;
    asn: string;
    abuse_contact?: string;
    is_tor?: boolean;
    is_vpn?: boolean;
  }
  interface AssetResult {
    ip: string;
    host_id?: string;
    hostname?: string;
    os?: string;
    tags?: string[];
    last_seen?: string;
    risk_score?: number;
  }
  interface EnrichmentResult {
    query: string;
    geo?: GeoResult;
    dns?: DNSResult;
    asset?: AssetResult;
    ioc_match?: { matched: boolean; severity?: string; source?: string; description?: string };
  }

  // -- State --
  let query     = $state('');
  let submitted = $state('');
  let loading   = $state(false);
  let result    = $state<EnrichmentResult | null>(null);
  let recent    = $state<EnrichmentResult[]>([]);
  let recentLoading = $state(false);

  // -- Actions --
  async function fetchRecent() {
    recentLoading = true;
    try {
      recent = await request<EnrichmentResult[]>('/enrich/recent') ?? [];
    } catch {
      recent = [];
    } finally {
      recentLoading = false;
    }
  }

  async function runEnrichment() {
    const q = query.trim();
    if (!q) return;
    
    submitted = q;
    loading = true;
    result = null;

    try {
      result = await request<EnrichmentResult>(`/enrich?q=${encodeURIComponent(q)}`);
    } catch (e) {
      console.error('Enrichment failed', e);
    } finally {
      loading = false;
      fetchRecent();
    }
  }

  function riskColor(score?: number): string {
    if (score === undefined) return 'var(--text-muted)';
    if (score >= 80) return 'var(--alert-critical)';
    if (score >= 60) return 'var(--alert-high)';
    if (score >= 40) return 'var(--alert-medium)';
    return 'var(--status-online)';
  }

  onMount(() => {
    fetchRecent();
  });
</script>

<PageLayout 
  title="Enrichment Core" 
  subtitle="Unified context explorer: Deep-level telemetry combined with federated intelligence and asset mapping"
>
  {#snippet toolbar()}
    <div class="flex items-center gap-2">
      <Button variant="secondary" size="sm" onclick={fetchRecent} disabled={recentLoading}>
        {#if recentLoading}<Spinner size="xs" class="mr-2" />{/if}
        History Refresh
      </Button>
    </div>
  {/snippet}

  <div class="flex flex-col h-full gap-6">
    <!-- Pulse Stats -->
    <div class="grid grid-cols-1 md:grid-cols-4 gap-4 shrink-0">
      <KPI title="Active Resolvers" value="14" trend="Synced" variant="accent" />
      <KPI title="Intelligence Hits" value={recent.filter(r => r.ioc_match?.matched).length.toString()} trend="Last 24h" variant="warning" />
      <KPI title="Asset Mapping" value="842" trend="+12 today" variant="success" />
      <KPI title="Pipeline Status" value="Online" trend="0ms latency" variant="success" />
    </div>

    <!-- Search Section -->
    <div class="bg-surface-1 border border-border-primary p-4 rounded-md shadow-premium">
      <div class="flex gap-2">
        <div class="flex-1 relative">
          <input
            type="text"
            bind:value={query}
            placeholder="Search IP, Hostname, or User (e.g., 10.0.0.1, suspicious.com)"
            class="w-full bg-surface-0 border border-border-primary text-text-primary px-4 py-2.5 rounded-sm font-mono text-sm focus:outline-hidden focus:border-accent-primary transition-all"
            onkeydown={(e) => e.key === 'Enter' && runEnrichment()}
          />
          <div class="absolute right-3 top-2.5 opacity-20 pointer-events-none">
            <Search size={18} />
          </div>
        </div>
        <Button variant="primary" onclick={runEnrichment} disabled={loading}>
          {#if loading}
            <Spinner size="xs" class="mr-2" />
          {:else}
            <Zap size={16} class="mr-2" />
          {/if}
          Run Enrichment
        </Button>
      </div>
    </div>

    <!-- Results Section -->
    {#if loading}
      <div class="flex-1 flex flex-col items-center justify-center gap-4 text-text-muted">
        <Spinner size="lg" />
        <p class="font-mono text-xs uppercase tracking-widest animate-pulse">Running Federated Resolution for {submitted}...</p>
      </div>
    {:else if result}
      <div class="grid grid-cols-1 lg:grid-cols-2 gap-4">
        <!-- GeoIP Card -->
        {#if result.geo}
          <div class="bg-surface-1 border border-border-primary border-t-2 border-t-accent-primary p-5 rounded-md relative overflow-hidden group">
            <div class="absolute -right-4 -bottom-4 opacity-[0.05] grayscale group-hover:scale-110 transition-transform duration-700">
              <Globe size={120} />
            </div>
            <div class="flex justify-between items-start mb-4">
              <h3 class="text-[10px] font-bold text-text-muted uppercase tracking-widest">GeoIP Attribution</h3>
              <Badge variant="accent" size="xs">{result.geo.country_code}</Badge>
            </div>
            <div class="space-y-3 relative z-10">
              {#each [
                ['Country', `${result.geo.country_name} (${result.geo.country_code})`],
                ['City', result.geo.city || '—'],
                ['ASN', result.geo.asn || '—'],
                ['ISP / Organization', result.geo.org || result.geo.isp || '—'],
                ['Coordinates', result.geo.latitude ? `${result.geo.latitude.toFixed(4)}, ${result.geo.longitude.toFixed(4)}` : '—']
              ] as [label, value]}
                <div class="flex justify-between items-center border-b border-border-subtle pb-1.5">
                  <span class="text-[11px] text-text-muted font-mono uppercase tracking-tighter">{label}</span>
                  <span class="text-xs text-text-primary font-bold">{value}</span>
                </div>
              {/each}
            </div>
          </div>
        {/if}

        <!-- DNS Card -->
        {#if result.dns}
          <div class="bg-surface-1 border border-border-primary border-t-2 border-t-status-online p-5 rounded-md relative overflow-hidden group">
            <div class="absolute -right-4 -bottom-4 opacity-[0.05] grayscale group-hover:scale-110 transition-transform duration-700">
              <Activity size={120} />
            </div>
            <div class="flex justify-between items-start mb-4">
              <h3 class="text-[10px] font-bold text-text-muted uppercase tracking-widest">DNS / ASN Resolution</h3>
              <Badge variant="success" size="xs">RESOLVED</Badge>
            </div>
            <div class="space-y-3 relative z-10">
              {#each [
                ['Hostname', result.dns.hostname || '—'],
                ['PTR Record', result.dns.ptr || '—'],
                ['ASN', result.dns.asn || '—'],
                ['Tor Exit Node', result.dns.is_tor ? '⚠️ YES' : 'CLEAN'],
                ['VPN / Proxy', result.dns.is_vpn ? '⚠️ DETECTED' : 'CLEAN']
              ] as [label, value]}
                <div class="flex justify-between items-center border-b border-border-subtle pb-1.5">
                  <span class="text-[11px] text-text-muted font-mono uppercase tracking-tighter">{label}</span>
                  <span class="text-xs font-bold {value.includes('⚠️') ? 'text-alert-high' : 'text-text-primary'}">{value}</span>
                </div>
              {/each}
            </div>
          </div>
        {/if}

        <!-- Asset Card -->
        {#if result.asset}
          <div class="bg-surface-1 border border-border-primary border-t-2 border-t-alert-medium p-5 rounded-md relative overflow-hidden group">
             <div class="absolute -right-4 -bottom-4 opacity-[0.05] grayscale group-hover:scale-110 transition-transform duration-700">
              <Database size={120} />
            </div>
            <div class="flex justify-between items-start mb-4">
              <h3 class="text-[10px] font-bold text-text-muted uppercase tracking-widest">Internal Asset Record</h3>
              <Badge variant="warning" size="xs">MAPPED</Badge>
            </div>
            <div class="space-y-3 relative z-10">
              {#each [
                ['Host ID', result.asset.host_id || '—'],
                ['System Name', result.asset.hostname || '—'],
                ['Operating System', result.asset.os || '—'],
                ['Last Observed', result.asset.last_seen ? new Date(result.asset.last_seen).toLocaleString() : '—']
              ] as [label, value]}
                <div class="flex justify-between items-center border-b border-border-subtle pb-1.5">
                  <span class="text-[11px] text-text-muted font-mono uppercase tracking-tighter">{label}</span>
                  <span class="text-xs text-text-primary font-bold">{value}</span>
                </div>
              {/each}
              
              {#if result.asset.risk_score !== undefined}
                <div class="pt-2">
                  <div class="flex justify-between items-center mb-1.5">
                    <span class="text-[10px] font-bold text-text-muted uppercase tracking-widest">Behavioral Risk Score</span>
                    <span class="text-xs font-bold" style="color: {riskColor(result.asset.risk_score)}">{result.asset.risk_score}/100</span>
                  </div>
                  <ProgressBar value={result.asset.risk_score} color={riskColor(result.asset.risk_score)} height="4px" />
                </div>
              {/if}
            </div>
          </div>
        {/if}

        <!-- IOC Match Card -->
        {#if result.ioc_match}
          <div class="bg-surface-1 border border-border-primary border-t-2 p-5 rounded-md relative overflow-hidden group" style="border-top-color: {result.ioc_match.matched ? 'var(--alert-critical)' : 'var(--border-primary)'}">
            <div class="absolute -right-4 -bottom-4 opacity-[0.05] grayscale group-hover:scale-110 transition-transform duration-700">
              <Shield size={120} />
            </div>
            <div class="flex justify-between items-start mb-4">
              <h3 class="text-[10px] font-bold text-text-muted uppercase tracking-widest">Intelligence Correlation</h3>
              <Badge variant={result.ioc_match.matched ? 'danger' : 'secondary'} size="xs">{result.ioc_match.matched ? 'MATCH' : 'CLEAN'}</Badge>
            </div>
            <div class="relative z-10">
              <div class="text-lg font-black italic uppercase tracking-tighter mb-2" style="color: {result.ioc_match.matched ? 'var(--alert-critical)' : 'var(--status-online)'}">
                {result.ioc_match.matched ? '⚠️ MALICIOUS MATCH FOUND' : '✓ NO KNOWN THREATS'}
              </div>
              
              {#if result.ioc_match.matched}
                <div class="space-y-3 mt-4">
                  {#each [
                    ['Severity', result.ioc_match.severity ?? '—'],
                    ['Source Feed', result.ioc_match.source ?? '—'],
                    ['Context', result.ioc_match.description ?? '—']
                  ] as [label, value]}
                    <div class="border-b border-border-subtle pb-1.5">
                      <div class="text-[10px] text-text-muted font-mono uppercase tracking-tighter mb-0.5">{label}</div>
                      <div class="text-xs text-text-primary font-bold">{value}</div>
                    </div>
                  {/each}
                </div>
              {/if}
            </div>
          </div>
        {/if}
      </div>
    {:else}
      <!-- Empty State / Recent History -->
      <div class="flex-1 flex flex-col min-h-0 gap-4">
        <div class="bg-surface-1 border border-border-primary rounded-md overflow-hidden flex flex-col flex-1 shadow-premium">
          <div class="p-3 bg-surface-2 border-b border-border-primary flex justify-between items-center text-[10px] font-bold uppercase tracking-widest text-text-muted font-mono">
            Recent Intelligence Resolution History
            <History size={14} />
          </div>
          <div class="flex-1 overflow-auto">
            {#if recentLoading}
               <div class="h-full flex items-center justify-center">
                  <Spinner />
               </div>
            {:else if recent.length === 0}
               <div class="h-full flex flex-col items-center justify-center text-text-muted gap-2 opacity-50">
                  <Database size={48} />
                  <p class="font-mono text-xs uppercase tracking-widest">No recent resolution data found</p>
               </div>
            {:else}
               <DataTable data={recent} columns={[
                 { key: 'query', label: 'Resolution Target', width: '250px' },
                 { key: 'context', label: 'Federated Context' },
                 { key: 'status', label: 'Intelligence State', width: '150px' }
               ]} rowKey="query" onRowClick={(row) => { query = row.query; runEnrichment(); }}>
                 {#snippet cell({ column, row })}
                   {#if column.key === 'query'}
                      <div class="flex items-center gap-2">
                        <Zap size={10} class="text-accent-primary opacity-50" />
                        <span class="font-bold text-text-heading">{row.query}</span>
                      </div>
                   {:else if column.key === 'context'}
                      <div class="text-[10px] text-text-muted font-mono uppercase truncate max-w-[400px]">
                        {#if row.geo}{row.geo.country_name} ({row.geo.org}){:else}NO_GEO_DATA{/if}
                        // 
                        {#if row.dns}{row.dns.hostname || 'NO_PTR'}{:else}NO_DNS_DATA{/if}
                      </div>
                   {:else if column.key === 'status'}
                      {#if row.ioc_match?.matched}
                         <Badge variant="danger" size="xs" class="font-bold animate-pulse">MALICIOUS</Badge>
                      {:else}
                         <Badge variant="secondary" size="xs">CLEAN</Badge>
                      {/if}
                   {/if}
                 {/snippet}
               </DataTable>
            {/if}
          </div>
        </div>
      </div>
    {/if}
  </div>
</PageLayout>

<style>
  /* Custom scrollbar for tactical feel */
  :global(.flex-1::-webkit-scrollbar) {
    width: 6px;
    height: 6px;
  }
  :global(.flex-1::-webkit-scrollbar-track) {
    background: var(--surface-0);
  }
  :global(.flex-1::-webkit-scrollbar-thumb) {
    background: var(--border-primary);
    border-radius: 3px;
  }
  :global(.flex-1::-webkit-scrollbar-thumb:hover) {
    background: var(--text-muted);
  }
</style>
