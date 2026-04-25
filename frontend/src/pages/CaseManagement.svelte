<!--
  OBLIVRA — Case Management (Svelte 5)
  Orchestration hub for incident investigations and legal readiness.
-->
<script lang="ts">
  import { KPI, Badge, DataTable, PageLayout, Button, Input, PopOutButton} from '@components/ui';
  import { User, ExternalLink } from 'lucide-svelte';

  const cases: Record<string, any>[] = [
    { id: 'CASE-7721', title: 'Data Exfiltration - Node Alpha', severity: 'critical', owner: 'maverick', status: 'active', drift: '2.4h' },
    { id: 'CASE-7722', title: 'Persistent SSH Tunneling', severity: 'high', owner: 'iceman', status: 'investigating', drift: '12m' },
    { id: 'CASE-7723', title: 'Suspicious Cloud API calls', severity: 'medium', owner: 'system', status: 'closed', drift: '0s' },
  ];

  let searchQuery = $state('');
  const filteredCases = $derived(cases.filter(c => c.title.toLowerCase().includes(searchQuery.toLowerCase())));
</script>

<PageLayout title="Investigation Center" subtitle="Comprehensive management of security incidents and legal cases">
  {#snippet toolbar()}
    <div class="flex items-center gap-2">
      <Input variant="search" placeholder="Filter cases..." bind:value={searchQuery} class="w-64" />
      <Button variant="primary" size="sm">New Investigation</Button>
    </div>
      <PopOutButton route="/cases" title="Case Management" />
    {/snippet}

  <div class="flex flex-col h-full gap-6">
    <div class="grid grid-cols-1 md:grid-cols-4 gap-4">
      <KPI label="Active Investigations" value={cases.filter(c => c.status !== 'closed').length} trend="stable" trendValue="Priority" />
      <KPI label="Mean Time to Resolution" value="1.5 days" trend="down" trendValue="12%" variant="success" />
      <KPI label="Backlog Volume" value="12" trend="up" trendValue="+2" variant="warning" />
      <KPI label="Legal Readiness" value="READY" trend="stable" trendValue="Locked" variant="success" />
    </div>

    <div class="flex-1 bg-surface-1 border border-border-primary rounded-md overflow-hidden flex flex-col shadow-sm">
      <div class="p-3 bg-surface-2 border-b border-border-primary flex justify-between items-center">
        <div class="text-[10px] font-bold uppercase tracking-widest text-text-muted">Master Case Index</div>
        <div class="flex gap-2">
           <Badge variant="info" size="xs">EXTERNAL SYNC: ON</Badge>
        </div>
      </div>
      
      <div class="flex-1 overflow-auto">
        <DataTable data={filteredCases} columns={[
          { key: 'id', label: 'Case ID', width: '100px' },
          { key: 'title', label: 'Incident Description' },
          { key: 'severity', label: 'Tier', width: '100px' },
          { key: 'owner', label: 'Lead Investigator', width: '150px' },
          { key: 'status', label: 'State', width: '120px' },
          { key: 'action', label: '', width: '60px' }
        ]} compact>
          {#snippet render({ col: column, row })}
            {#if column.key === 'severity'}
               <Badge variant={row.severity === 'critical' ? 'critical' : row.severity === 'high' ? 'warning' : 'info'}>{row.severity}</Badge>
            {:else if column.key === 'status'}
               <div class="flex items-center gap-2">
                 <div class="w-1.5 h-1.5 rounded-full {row.status === 'active' ? 'bg-accent animate-pulse' : row.status === 'investigating' ? 'bg-warning' : 'bg-text-muted'}"></div>
                 <span class="text-[10px] uppercase font-bold text-text-secondary">{row.status}</span>
               </div>
            {:else if column.key === 'id'}
               <span class="text-[10px] font-mono font-bold text-accent">{row.id}</span>
            {:else if column.key === 'title'}
               <span class="text-[11px] font-bold text-text-heading">{row.title}</span>
            {:else if column.key === 'owner'}
               <div class="flex items-center gap-2 text-[10px] text-text-secondary font-bold">
                  <User size={12} class="opacity-40" />
                  {row.owner}
               </div>
            {:else if column.key === 'action'}
               <button class="text-text-muted hover:text-accent transition-colors" title="View Case Details">
                  <ExternalLink size={14} />
               </button>
            {:else}
              <span class="text-[11px] text-text-secondary">{row[column.key]}</span>
            {/if}
          {/snippet}
        </DataTable>
      </div>
    </div>
  </div>
</PageLayout>
