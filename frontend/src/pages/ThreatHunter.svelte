<script lang="ts">
  import { KPI, PageLayout, Button, Badge, PopOutButton} from '@components/ui';
  import { Search, Crosshair } from 'lucide-svelte';

  const hypotheses = [
    { title: 'Lateral Movement via WMI', confidence: 'high', detections: 2, status: 'pivoted' },
    { title: 'DNS Tunneling patterns', confidence: 'low', detections: 0, status: 'hunting' },
    { title: 'Process injection in LSASS', confidence: 'critical', detections: 1, status: 'escalated' },
  ];

  let query = $state('process.name == "lsass.exe" AND source.type == "external"');
</script>

<PageLayout title="Tactical Threat Hunter" subtitle="Advanced hypothesis orchestration and forensic digging: Deconstructing adversary TTPs across the sovereign fleet">
  {#snippet toolbar()}
    <div class="flex items-center gap-2">
      <Button variant="secondary" size="sm">Hunt Library</Button>
      <Button variant="primary" size="sm" icon="🔍">Begin New Mission</Button>
    </div>
      <PopOutButton route="/threat-hunter" title="Threat Hunter" />
    {/snippet}

  <div class="flex flex-col h-full gap-6">
    <!-- Stats Row -->
    <div class="grid grid-cols-1 md:grid-cols-4 gap-4 shrink-0">
      <KPI label="Active Hunts" value={hypotheses.length} trend="stable" trendValue="Nominal" />
      <KPI label="Detections" value="3" trend="up" trendValue="Critical" variant="critical" />
      <KPI label="Data Throttling" value="None" trend="stable" trendValue="Optimal" variant="success" />
      <KPI label="Logic Accuracy" value="99.2%" trend="stable" trendValue="Stable" variant="success" />
    </div>

    <!-- Active Hypotheses -->
    <div class="grid grid-cols-1 md:grid-cols-3 gap-4 shrink-0">
      {#each hypotheses as hyp}
        <div class="bg-surface-1 border border-border-primary p-4 rounded-md shadow-premium relative group overflow-hidden cursor-pointer hover:border-accent transition-all">
          <div class="absolute inset-0 bg-accent/5 opacity-0 group-hover:opacity-100 transition-opacity"></div>
          <div class="flex flex-col gap-3 relative z-10">
            <div class="flex justify-between items-center">
              <Badge variant={hyp.confidence === 'critical' ? 'critical' : hyp.confidence === 'high' ? 'warning' : 'info'} size="xs">
                {hyp.confidence.toUpperCase()} CONFIDENCE
              </Badge>
              <span class="text-[9px] font-bold text-accent uppercase tracking-widest">{hyp.status}</span>
            </div>
            <div class="text-[11px] font-bold text-text-heading group-hover:text-accent transition-colors">{hyp.title}</div>
            <div class="flex items-center justify-between text-[10px] text-text-muted mt-1">
              <span class="text-accent font-bold font-mono">{hyp.detections} Matches</span>
              <span class="font-mono">Updated 4m ago</span>
            </div>
          </div>
        </div>
      {/each}
    </div>

    <!-- Hunting Box -->
    <div class="flex-1 flex flex-col bg-surface-1 border border-border-primary rounded-md overflow-hidden shadow-premium">
      <div class="p-4 bg-surface-2 border-b border-border-primary">
        <div class="flex items-center gap-3">
          <div class="flex-1 relative">
            <div class="absolute left-3 top-1/2 -translate-y-1/2 text-accent opacity-60">
              <Search size={14} />
            </div>
            <input 
              type="text" 
              bind:value={query}
              class="w-full bg-surface-0 border border-border-secondary rounded-sm pl-10 pr-4 py-2.5 text-[11px] font-mono text-accent focus:border-accent outline-none ring-1 ring-transparent focus:ring-accent/10 transition-all placeholder:text-text-muted/40"
              placeholder="Enter OQL (Oblivra Query Language) logic..."
            />
          </div>
          <Button variant="cta" size="sm" icon="⚡">EXECUTE</Button>
        </div>
      </div>

      <div class="flex-1 flex flex-col items-center justify-center p-12 text-center relative overflow-hidden">
        <!-- Background Grid -->
        <div class="absolute inset-0 opacity-[0.02] pointer-events-none grayscale" style="background-image: radial-gradient(var(--color-accent) 1px, transparent 1px); background-size: 30px 30px;"></div>
        
        <div class="relative z-10 flex flex-col items-center">
           <div class="w-20 h-20 rounded-full border-2 border-dashed border-border-secondary flex items-center justify-center mb-6 group-hover:rotate-12 transition-transform">
              <Crosshair size={32} class="text-accent opacity-20" />
           </div>
           <h4 class="text-xs font-bold uppercase tracking-widest text-text-heading mb-2">Ready for Forensics</h4>
           <p class="text-[10px] text-text-muted max-w-xs leading-relaxed">
              Execute an OQL probe above to pivot through multi-dimensional telemetry. Results are streamed in real-time from mesh nodes.
           </p>
        </div>
      </div>
    </div>
  </div>
</PageLayout>
