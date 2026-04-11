<!-- OBLIVRA Web — WarRoomGrid (Svelte 5) -->
<script lang="ts">
  import { onMount } from 'svelte';
  import SeverityIcon, { type Severity } from './SeverityIcon.svelte';

  interface Incident { id: string; timestamp: string; source: string; event: string; severity: Severity; tenant: string; }

  let incidents = $state<Incident[]>([]);

  onMount(() => {
    incidents = [
      { id:'1', timestamp:'21:44:02', source:'LNUX-PROD-01',      event:'SSH Brute Force Detected',  severity:'high',     tenant:'GLOBAL_CORP'    },
      { id:'2', timestamp:'21:43:55', source:'WIN-DC-04',          event:'Kerberoasting Attempt',     severity:'critical', tenant:'GLOBAL_CORP'    },
      { id:'3', timestamp:'21:43:12', source:'AP-WEST-2',          event:'S3 Bucket Public Access',   severity:'medium',   tenant:'CLOUD_OPS'      },
      { id:'4', timestamp:'21:42:50', source:'HQ-WORKSTATION-12',  event:'Mimikatz Pattern Found',    severity:'critical', tenant:'GLOBAL_CORP'    },
      { id:'5', timestamp:'21:41:05', source:'WEB-FE-01',          event:'XSS Payload in GET',        severity:'low',      tenant:'ECOMM_PLATFORM' },
    ];
  });

  const sevColor: Record<Severity, string> = {
    critical: 'var(--alert-critical)', high: 'var(--alert-high)',
    medium: 'var(--alert-medium)', low: 'var(--alert-low)', info: 'var(--alert-info)',
  };
</script>

<section class="wrg-wrap">
  <div class="wrg-header">
    <span class="wrg-title">Live Incident Stream // War Room Mode</span>
    <span class="wrg-sub">Polling state: STABLE // Latency: 42ms</span>
  </div>

  <div class="wrg-table-wrap">
    <table class="wrg-table">
      <thead>
        <tr>
          {#each ['Status','Timestamp','Tenant','Source','Event / Indicator'] as h}
            <th>{h}</th>
          {/each}
        </tr>
      </thead>
      <tbody>
        {#each incidents as inc (inc.id)}
          <tr class="wrg-row">
            <td>
              <div class="wrg-sev">
                <SeverityIcon severity={inc.severity} size={12} />
                <span style="color:{sevColor[inc.severity]}">{inc.severity}</span>
              </div>
            </td>
            <td class="wrg-muted">{inc.timestamp}</td>
            <td><span class="wrg-badge">{inc.tenant}</span></td>
            <td class="wrg-src">{inc.source}</td>
            <td>
              <div class="wrg-event-cell">
                <span class="wrg-event">{inc.event}</span>
                <button class="wrg-triage">Triage</button>
              </div>
            </td>
          </tr>
        {/each}
      </tbody>
    </table>
  </div>
</section>

<style>
  .wrg-wrap { font-family: var(--font-mono); display: flex; flex-direction: column; gap: 12px; }
  .wrg-header { display: flex; justify-content: space-between; align-items: flex-end; border-bottom: 1px solid var(--border-bold,#1e3040); padding-bottom: 8px; }
  .wrg-title  { font-size: 10px; font-weight: 800; text-transform: uppercase; letter-spacing: .2em; color: var(--accent-primary); }
  .wrg-sub    { font-size: 9px; text-transform: uppercase; letter-spacing: .1em; color: var(--text-muted); }
  .wrg-table-wrap { border: 1px solid rgba(255,255,255,0.04); background: rgba(0,0,0,0.2); overflow-x: auto; }
  .wrg-table { width: 100%; border-collapse: collapse; font-size: 11px; text-align: left; }
  .wrg-table thead tr { background: rgba(0,0,0,0.3); border-bottom: 1px solid #1e3040; }
  .wrg-table th { padding: 10px 12px; font-size: 10px; text-transform: uppercase; letter-spacing: .12em; color: var(--text-muted); font-weight: 400; }
  .wrg-row { border-bottom: 1px solid rgba(255,255,255,0.04); transition: background 100ms ease; }
  .wrg-row:hover { background: rgba(255,255,255,0.02); }
  .wrg-row:hover .wrg-triage { opacity: 1; }
  .wrg-row td { padding: 10px 12px; }
  .wrg-sev  { display: flex; align-items: center; gap: 6px; text-transform: uppercase; font-weight: 700; letter-spacing: .06em; font-size: 10px; }
  .wrg-muted { color: var(--text-muted); }
  .wrg-src   { font-weight: 700; color: #c8d8d8; }
  .wrg-badge { background: #1e3040; color: #9b9ea4; padding: 2px 8px; border-radius: 3px; font-size: 9px; font-weight: 700; }
  .wrg-event-cell { display: flex; align-items: center; justify-content: space-between; gap: 8px; }
  .wrg-event { color: #9b9ea4; }
  .wrg-triage {
    opacity: 0; padding: 2px 8px; border: 1px solid var(--accent-primary);
    color: var(--accent-primary); font-size: 9px; font-weight: 700; text-transform: uppercase;
    background: transparent; cursor: pointer; transition: all 100ms ease; font-family: var(--font-mono);
    white-space: nowrap;
  }
  .wrg-triage:hover { background: var(--accent-primary); color: #080f12; }
</style>
