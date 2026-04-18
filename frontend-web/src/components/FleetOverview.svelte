<!-- OBLIVRA Web — FleetOverview (Svelte 5) -->
<script lang="ts">
  import { onMount } from 'svelte';

  interface FleetNode { id: string; name: string; tenant: string; platform: 'linux'|'windows'|'darwin'; status: 'online'|'degraded'|'offline'; ip: string; }

  let nodes = $state<FleetNode[]>([]);

  onMount(() => {
    nodes = [
      { id:'1', name:'PRD-LNX-01', tenant:'GLOBAL_CORP',  platform:'linux',   status:'online',   ip:'192.168.1.10' },
      { id:'2', name:'PRD-LNX-02', tenant:'GLOBAL_CORP',  platform:'linux',   status:'online',   ip:'192.168.1.11' },
      { id:'3', name:'PRD-WIN-01', tenant:'GLOBAL_CORP',  platform:'windows', status:'degraded', ip:'192.168.1.12' },
      { id:'4', name:'DEV-DAR-01', tenant:'INNOVATION',   platform:'darwin',  status:'offline',  ip:'10.0.4.45'    },
      { id:'5', name:'STG-LNX-01', tenant:'GLOBAL_CORP',  platform:'linux',   status:'online',   ip:'192.168.5.20' },
    ];
  });

  const platformIcon: Record<string,string> = { windows:'🪟', linux:'🐧', darwin:'🍎' };
  const statusColorMap: Record<string,string> = { online:'var(--status-online)', degraded:'var(--status-degraded)', offline:'var(--status-offline)' };
</script>

<section class="fo-wrap">
  <div class="fo-header">
    <span class="fo-title">Fleet Infrastructure Overview</span>
    <div class="fo-legend">
      <span><i style="background:var(--status-online)"></i> {nodes.filter(n=>n.status==='online').length} Online</span>
      <span><i style="background:var(--status-degraded)"></i> {nodes.filter(n=>n.status==='degraded').length} Warning</span>
    </div>
  </div>

  <div class="fo-grid">
    {#each nodes as node (node.id)}
      <div class="fo-card" role="listitem">
        <div class="fo-bar" style="background:{statusColorMap[node.status] ?? '#607070'}"></div>
        <div class="fo-info">
          <div class="fo-name">
            <span>{platformIcon[node.platform] ?? '🖥️'}</span>
            <strong>{node.name}</strong>
          </div>
          <span class="fo-meta">{node.tenant} // {node.ip}</span>
        </div>
        <span class="fo-cta">DRILL_DOWN →</span>
      </div>
    {/each}
  </div>
</section>

<style>
  .fo-wrap { font-family: var(--font-mono); display: flex; flex-direction: column; gap: 12px; }
  .fo-header { display: flex; justify-content: space-between; align-items: flex-end; border-bottom: 1px solid var(--border-bold,#1e3040); padding-bottom: 8px; }
  .fo-title  { font-size: 10px; font-weight: 800; text-transform: uppercase; letter-spacing: .2em; color: var(--accent-primary); }
  .fo-legend { display: flex; gap: 14px; font-size: 9px; text-transform: uppercase; letter-spacing: .1em; color: var(--text-muted); }
  .fo-legend i { display: inline-block; width: 6px; height: 6px; border-radius: 50%; margin-right: 4px; }
  .fo-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(200px, 1fr)); gap: 10px; }
  .fo-card {
    display: flex; align-items: center; gap: 10px;
    border: 1px solid var(--border-subtle,rgba(255,255,255,0.04));
    background: rgba(0,0,0,0.1); padding: 10px;
    cursor: pointer; transition: border-color 100ms ease;
  }
  .fo-card:hover { border-color: var(--accent-primary); }
  .fo-card:hover .fo-cta { opacity: 1; }
  .fo-bar { width: 3px; height: 28px; border-radius: 1px; flex-shrink: 0; }
  .fo-info { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 2px; }
  .fo-name { display: flex; align-items: center; gap: 6px; font-size: 11px; font-weight: 800; text-transform: uppercase; color: #e2e8f0; }
  .fo-meta  { font-size: 9px; text-transform: uppercase; color: var(--text-muted); letter-spacing: .04em; }
  .fo-cta   { font-size: 10px; color: var(--accent-primary); opacity: 0; transition: opacity 100ms ease; white-space: nowrap; }
</style>
