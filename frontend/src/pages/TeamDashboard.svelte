<!--
  OBLIVRA — Team Dashboard (Svelte 5)
  Collaboration and access control management for SOC teams.
-->
<script lang="ts">
  import { KPI, Badge, DataTable, PageLayout, Button } from '@components/ui';

  const mockTeam = [
    { id: 'u1', name: 'Maverick', role: 'Global Admin', last_active: 'Now', status: 'online' },
    { id: 'u2', name: 'Iceman', role: 'Security Analyst', last_active: '2m ago', status: 'online' },
    { id: 'u3', name: 'Goose', role: 'Auditor', last_active: '4h ago', status: 'offline' },
    { id: 'u4', name: 'Viper', role: 'SOC Lead', last_active: '1d ago', status: 'offline' },
  ];

  const columns = [
    { key: 'name', label: 'Team Member' },
    { key: 'role', label: 'Role', width: '150px' },
    { key: 'last_active', label: 'Last Active', width: '120px' },
    { key: 'status', label: 'Status', width: '100px' },
  ];
</script>

<PageLayout title="Team Collaboration" subtitle="Manage permissions, roles, and real-time analyst presence">
  {#snippet toolbar()}
    <Button variant="secondary" size="sm" icon="+">Invite Member</Button>
  {/snippet}

  <div class="flex flex-col h-full gap-5">
    <div class="grid grid-cols-1 md:grid-cols-4 gap-4 shrink-0">
      <KPI label="Team Size" value={mockTeam.length} variant="default" />
      <KPI label="Active Now" value={mockTeam.filter(t => t.status === 'online').length} variant="success" />
      <KPI label="Pending Invites" value="1" variant="warning" />
      <KPI label="Org Tier" value="Enterprise" variant="accent" />
    </div>

    <div class="bg-surface-1 border border-border-primary rounded-md overflow-hidden">
      <DataTable data={mockTeam} {columns} striped>
        {#snippet render({ value, col, row })}
          {#if col.key === 'status'}
            <Badge variant={value === 'online' ? 'success' : 'muted'} dot>
              {value}
            </Badge>
          {:else if col.key === 'name'}
            <div class="flex items-center gap-3">
              <div class="w-7 h-7 rounded-full bg-accent/20 border border-accent/30 flex items-center justify-center text-[10px] font-bold text-accent">
                {value.charAt(0)}
              </div>
              <span class="font-bold text-text-heading">{value}</span>
            </div>
          {:else}
            {value}
          {/if}
        {/snippet}
      </DataTable>
    </div>

    <!-- Security Policy Notice -->
    <div class="p-4 bg-accent/5 border border-accent/20 rounded-md">
      <div class="flex items-center gap-2 mb-1">
        <span class="text-accent text-xs">🛡️</span>
        <h4 class="text-[10px] font-bold uppercase tracking-widest text-accent">Identity Policy</h4>
      </div>
      <p class="text-[11px] text-text-muted leading-relaxed">
        This workspace requires <span class="text-text-primary font-bold">Hardware MFA (U2F)</span> for all members with Administrator roles. 
        Auto-revocation is active for sessions inactive for more than 4 hours.
      </p>
    </div>
  </div>
</PageLayout>
