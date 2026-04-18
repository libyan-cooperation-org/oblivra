import { For, createSignal, onMount } from 'solid-js';
import SeverityIcon, { type Severity } from './SeverityIcon';

interface Incident {
  id: string;
  timestamp: string;
  source: string;
  event: string;
  severity: Severity;
  tenant: string;
}

export default function WarRoomGrid() {
  const [incidents, setIncidents] = createSignal<Incident[]>([]);

  onMount(() => {
    // Mock incident telemetry
    const mockIncidents: Incident[] = [
      { id: '1', timestamp: '21:44:02', source: 'LNUX-PROD-01', event: 'SSH Brute Force Detected', severity: 'high', tenant: 'GLOBAL_CORP' },
      { id: '2', timestamp: '21:43:55', source: 'WIN-DC-04', event: 'Kerberoasting Attempt', severity: 'critical', tenant: 'GLOBAL_CORP' },
      { id: '3', timestamp: '21:43:12', source: 'AP-WEST-2', event: 'S3 Bucket Public Access', severity: 'medium', tenant: 'CLOUD_OPS' },
      { id: '4', timestamp: '21:42:50', source: 'HQ-WORKSTATION-12', event: 'Mimikatz Pattern Found', severity: 'critical', tenant: 'GLOBAL_CORP' },
      { id: '5', timestamp: '21:41:05', source: 'WEB-FE-01', event: 'XSS Payload in GET', severity: 'low', tenant: 'ECOMM_PLATFORM' },
    ];
    setIncidents(mockIncidents);
  });

  return (
    <section class="space-y-4 font-mono">
      <div class="flex justify-between items-end border-b border-[var(--border-bold)] pb-2">
        <h3 class="text-xs font-black uppercase tracking-[0.2em] text-[var(--accent-primary)]">
          Live Incident Stream // War Room Mode
        </h3>
        <span class="text-[9px] text-[var(--text-muted)] uppercase tracking-widest">
          Polling state: STABLE // Latency: 42ms
        </span>
      </div>

      <div class="border border-[var(--border-subtle)] bg-black/20 overflow-x-auto">
        <table class="w-full text-left border-collapse">
          <thead>
            <tr class="bg-[var(--bg-muted)] text-[10px] uppercase tracking-widest text-[var(--text-muted)] border-b border-[var(--border-bold)]">
              <th class="p-3 font-black">Status</th>
              <th class="p-3 font-black">Timestamp</th>
              <th class="p-3 font-black">Tenant</th>
              <th class="p-3 font-black">Source</th>
              <th class="p-3 font-black">Event / Indicator</th>
            </tr>
          </thead>
          <tbody class="text-[11px]">
            <For each={incidents()}>
              {(incident) => (
                <tr class="border-b border-[var(--border-subtle)] hover:bg-white/[0.02] transition-colors group">
                  <td class="p-3">
                    <div class="flex items-center gap-2">
                      <SeverityIcon severity={incident.severity} size={12} />
                      <span class={`font-bold uppercase tracking-tighter ${
                        incident.severity === 'critical' ? 'text-[var(--alert-critical)]' : 
                        incident.severity === 'high' ? 'text-[var(--alert-high)]' : 'text-zinc-400'
                      }`}>
                        {incident.severity}
                      </span>
                    </div>
                  </td>
                  <td class="p-3 text-[var(--text-muted)]">{incident.timestamp}</td>
                  <td class="p-3">
                    <span class="px-1.5 py-0.5 bg-zinc-800 text-zinc-400 rounded text-[9px] font-bold">
                      {incident.tenant}
                    </span>
                  </td>
                  <td class="p-3 font-bold text-zinc-300">{incident.source}</td>
                  <td class="p-3">
                    <div class="flex items-center justify-between">
                      <span class="text-zinc-400 group-hover:text-white transition-colors">{incident.event}</span>
                      <button class="opacity-0 group-hover:opacity-100 px-2 py-0.5 border border-[var(--accent-primary)] text-[var(--accent-primary)] text-[9px] uppercase font-bold hover:bg-[var(--accent-primary)] hover:text-[var(--bg-deep)] transition-all">
                        Triage
                      </button>
                    </div>
                  </td>
                </tr>
              )}
            </For>
          </tbody>
        </table>
      </div>
    </section>
  );
}
