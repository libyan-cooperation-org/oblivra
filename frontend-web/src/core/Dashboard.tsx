import { createSignal, onMount, For, onCleanup, Show } from 'solid-js';
import { APP_CONTEXT } from './context';
import { logout } from '../services/auth';
import SeverityIcon, { type Severity } from '../components/SeverityIcon';
import CommandPalette, { type PaletteAction } from '../components/CommandPalette';
import WarRoomGrid from '../components/WarRoomGrid';
import FleetOverview from '../components/FleetOverview';
import AccessibleTerminal from '../components/AccessibleTerminal';

export default function Dashboard() {
  const handleLogout = async () => {
    await logout();
    window.location.href = '/login';
  };

  const [status, setStatus] = createSignal('INITIALIZING');
  const [heatmapData, setHeatmapData] = createSignal<Severity[]>([]);
  const [isPaletteOpen, setIsPaletteOpen] = createSignal(false);
  const [isWarRoomMode, setIsWarRoomMode] = createSignal(false);

  const nav = (path: string) => () => { window.location.href = path; };

  const paletteActions: PaletteAction[] = [
    { id: 'war-room',    label: 'Toggle War Room Mode',      description: 'High-density tactical view',             icon: '⚔️',  shortcut: 'W',     action: () => setIsWarRoomMode(!isWarRoomMode()) },
    { id: 'onboard',    label: 'Fleet Onboarding',           description: 'Deploy new agents',                      icon: '🖥️',  shortcut: 'G O',   action: nav('/onboarding') },
    { id: 'fleet',      label: 'Fleet Management',           description: 'Agent status and bulk config',            icon: '🛰️',  action: nav('/fleet') },
    { id: 'siem',       label: 'SIEM Search',                description: 'Search the event substrate',             icon: '🛡️',  shortcut: 'G S',   action: nav('/siem/search') },
    { id: 'alerts',     label: 'Alert Management',           description: 'Triage and manage alerts',                icon: '🚨',  action: nav('/alerts') },
    { id: 'escalation', label: 'Escalation Center',          description: 'Escalation policies and on-call',        icon: '⚡',  action: nav('/escalation') },
    { id: 'playbooks',  label: 'Playbook Builder',           description: 'Build and execute SOAR playbooks',       icon: '🔧',  action: nav('/playbooks') },
    { id: 'identity',   label: 'Identity Administration',    description: 'Users, roles, and federated SSO',        icon: '👥',  action: nav('/identity') },
    { id: 'threatintel',label: 'Threat Intelligence',        description: 'IOC browser and campaign correlation',   icon: '🔴',  action: nav('/threatintel') },
    { id: 'enrich',     label: 'Enrichment Viewer',          description: 'GeoIP, DNS, asset mapping',               icon: '🔬',  action: nav('/enrich') },
    { id: 'ueba',       label: 'UEBA Dashboard',             description: 'Entity behavior and risk scoring',       icon: '🧠',  action: nav('/ueba') },
    { id: 'ndr',        label: 'NDR Dashboard',              description: 'Network flows and lateral movement',     icon: '🌐',  action: nav('/ndr') },
    { id: 'ransomware', label: 'Ransomware Defense',         description: 'Fleet-wide ransomware protection',       icon: '🦠',  action: nav('/ransomware') },
    { id: 'mitre',      label: 'MITRE ATT&CK Heatmap',       description: 'Technique coverage visualization',       icon: '📊',  action: nav('/mitre-heatmap') },
    { id: 'evidence',   label: 'Evidence Vault',             description: 'Legal-grade chain of custody',           icon: '🔍',  action: nav('/evidence') },
    { id: 'regulator',  label: 'Regulator Portal',           description: 'Audit export and compliance packages',   icon: '📄',  action: nav('/regulator') },
    { id: 'lookups',       label: 'Lookup Tables',               description: 'CIDR/Exact/Regex enrichment tables',     icon: '🗒️',  action: nav('/lookups') },
    { id: 'pb-metrics',  label: 'Playbook Metrics',            description: 'MTTR, success rates, bottlenecks',       icon: '📊',  action: nav('/playbook-metrics') },
    { id: 'peer',        label: 'Peer Analytics',              description: 'Behavioral peer group outliers',          icon: '🧠',  action: nav('/peer-analytics') },
    { id: 'fusion',      label: 'Fusion Engine',               description: 'Kill chain and campaign clustering',      icon: '🔗',  action: nav('/fusion') },
    { id: 'logout',     label: 'Terminate Session',          description: 'Securely exit OBLIVRA',                  icon: '🔌',  shortcut: 'ALT+X', action: handleLogout },
  ];

  const handleGlobalKey = (e: KeyboardEvent) => {
    if ((e.metaKey || e.ctrlKey) && (e.key === 'k' || e.key === 'p')) {
      e.preventDefault();
      setIsPaletteOpen(true);
    }
  };

  onMount(() => {
    window.addEventListener('keydown', handleGlobalKey);
    setTimeout(() => setStatus('READY'), 1000);
    const severities: Severity[] = ['info', 'low', 'medium', 'high', 'critical'];
    setHeatmapData(Array.from({ length: 48 }, () => severities[Math.floor(Math.random() * severities.length)]));
  });

  onCleanup(() => {
    window.removeEventListener('keydown', handleGlobalKey);
  });

  return (
    <main class="min-h-screen bg-[var(--bg-deep)] text-[var(--text-primary)] font-mono selection:bg-[var(--accent-primary)] selection:text-[var(--bg-deep)]">
      <a href="#main-content" class="sr-only focus:not-sr-only focus:fixed focus:top-0 focus:left-0 focus:z-[100] focus:bg-[var(--accent-primary)] focus:text-[var(--bg-deep)] focus:p-4 focus:font-bold">
        Skip to main content
      </a>

      <div class="sr-only" role="status" aria-live="polite">
        {status() === 'READY' ? 'Sovereign Substrate is READY.' : 'Initializing OBLIVRA Enterprise...'}
      </div>

      <header class={`border-b border-[var(--border-bold)] bg-[var(--bg-muted)] p-4 flex justify-between items-center sticky top-0 z-50 transition-colors duration-500 ${isWarRoomMode() ? 'border-red-900/50 bg-red-950/20' : ''}`}>
        <div class="flex items-center gap-4">
          <div class={`w-8 h-8 ${isWarRoomMode() ? 'bg-red-600 shadow-[0_0_15px_rgba(224,64,64,0.4)]' : 'bg-[var(--accent-primary)]'} grid place-items-center font-bold text-[var(--bg-deep)] transition-all`}>
            {isWarRoomMode() ? '!' : 'O'}
          </div>
          <div>
            <h1 class="text-lg font-black tracking-tighter uppercase leading-none text-[var(--text-heading)]">
              OBLIVRA <span class={isWarRoomMode() ? 'text-red-500' : 'text-[var(--accent-primary)]'}>{isWarRoomMode() ? 'WAR ROOM' : 'ENTERPRISE'}</span>
            </h1>
            <p class="text-[10px] text-[var(--text-muted)] tracking-widest uppercase opacity-70">
              {isWarRoomMode() ? 'HIGH_DENSITY_TELEMETRY' : 'Sovereign SOC Platform'} // Context: {APP_CONTEXT}
            </p>
          </div>
        </div>

        <div class="flex items-center gap-6">
          <button onClick={() => setIsWarRoomMode(!isWarRoomMode())} class={`text-[10px] font-bold uppercase tracking-widest px-3 py-1 border transition-all ${isWarRoomMode() ? 'border-red-600 text-red-500 bg-red-900/20' : 'border-zinc-700 text-zinc-500'}`}>
            {isWarRoomMode() ? 'Disable War Mode' : 'Enter War Room'}
          </button>
          <div class="flex flex-col items-end">
            <span class="text-[10px] uppercase tracking-widest text-[var(--text-muted)]">System Status</span>
            <div class="flex items-center gap-2">
              <span class={`w-2 h-2 rounded-full ${status() === 'READY' ? (isWarRoomMode() ? 'bg-red-500 shadow-[0_0_8px_#ff5252]' : 'bg-[var(--status-online)] shadow-[0_0_8px_var(--status-online)]') : 'bg-[var(--status-offline)] animate-pulse'}`}></span>
              <span class="text-xs font-bold tracking-tight">{status()}</span>
            </div>
          </div>
          <button onClick={handleLogout} class="h-10 px-6 border border-[var(--border-bold)] text-[var(--text-muted)] font-bold uppercase tracking-tighter hover:bg-red-600 hover:text-white hover:border-red-600 transition-all">
            Terminate Session
          </button>
        </div>
      </header>

      <div id="main-content" class="p-8 max-w-7xl mx-auto space-y-12" tabindex="-1">
        <Show when={!isWarRoomMode()}>
          <section class="space-y-6">
            <div class="inline-block px-3 py-1 bg-[var(--accent-secondary)] text-[10px] font-bold uppercase tracking-[0.2em]">
              Deployment Phase 0.4.1 // Enterprise Scaling
            </div>
            <h2 class="text-6xl font-black italic tracking-tighter leading-[0.9] uppercase text-[var(--text-heading)]">
              Multi-Tenant <br />
              <span class="text-[var(--accent-primary)]">Orchestration.</span>
            </h2>
            <p class="text-xl text-[var(--text-muted)] max-w-2xl leading-relaxed">
              Managing secure telemetry across distributed infrastructure. The OBLIVRA <strong>War Room</strong> mode provides zero-latency visibility into multi-tenant threat patterns and fleet health.
            </p>
          </section>

          <div class="grid grid-cols-1 md:grid-cols-4 gap-1 border border-[var(--border-subtle)] bg-[var(--border-subtle)]">
            <StatBox label="Active Nodes" value="001" sub="Primary Headless Server" />
            <StatBox label="Event Velocity" value="12.4" sub="EPS (Ingest Active)" />
            <StatBox label="Active Alerts" value="003" sub="Requires Intervention" severity="high" />
            <StatBox label="Threat Index" value="ELEVATED" sub="Pattern Delta Detected" color="var(--alert-high)" severity="medium" />
          </div>

          <FleetOverview />

          <section class="space-y-4">
            <div class="flex justify-between items-end">
              <h3 class="text-xs font-black uppercase tracking-[0.2em] text-[var(--text-muted)]">Inbound Event Density (Pattern-Fills)</h3>
              <div class="flex gap-4 text-[10px] uppercase font-bold text-[var(--text-muted)]">
                <div class="flex items-center gap-1"><div class="w-3 h-3 bg-[var(--alert-low)]" style={{ "background-image": "var(--pattern-low)" }}></div> Low</div>
                <div class="flex items-center gap-1"><div class="w-3 h-3 bg-[var(--alert-medium)]" style={{ "background-image": "var(--pattern-medium)" }}></div> Med</div>
                <div class="flex items-center gap-1"><div class="w-3 h-3 bg-[var(--alert-high)]" style={{ "background-image": "var(--pattern-high)" }}></div> High</div>
                <div class="flex items-center gap-1"><div class="w-3 h-3 bg-[var(--alert-critical)]" style={{ "background-image": "var(--pattern-critical)" }}></div> Crit</div>
              </div>
            </div>
            <div class="grid grid-cols-12 md:grid-cols-24 gap-1 bg-[var(--border-subtle)] border border-[var(--border-subtle)]">
              <For each={heatmapData()}>
                {(s) => (
                  <div class="aspect-square transition-all hover:scale-110 hover:z-10 cursor-help border border-black/20" style={{ "background-color": `var(--alert-${s})`, "background-image": `var(--pattern-${s})`, "background-size": "4px 4px" }} title={`Severity: ${s.toUpperCase()}`} role="img" aria-label={`${s} severity event block`}></div>
                )}
              </For>
            </div>
          </section>
        </Show>

        <Show when={isWarRoomMode()}>
          <div class="animate-in fade-in slide-in-from-bottom-4 duration-500 space-y-12">
            <div class="grid grid-cols-1 lg:grid-cols-12 gap-8">
              <div class="lg:col-span-8">
                <WarRoomGrid />
              </div>
              <div class="lg:col-span-4 space-y-8">
                <section class="space-y-4">
                  <h3 class="text-xs font-black uppercase tracking-[0.2em] text-red-500">Node Criticality</h3>
                  <div class="border border-red-900/30 bg-red-950/10 p-4 space-y-4">
                    <div class="flex justify-between items-center">
                      <span class="text-[10px] text-zinc-400">LNUX-PROD-01</span>
                      <span class="text-[10px] text-red-500 font-bold">CRITICAL</span>
                    </div>
                    <div class="w-full bg-zinc-800 h-1">
                      <div class="bg-red-600 h-full w-[85%] shadow-[0_0_10px_rgba(224,64,64,0.5)]"></div>
                    </div>
                  </div>
                </section>
                <section class="space-y-4">
                  <h3 class="text-xs font-black uppercase tracking-[0.2em] text-zinc-500">Global Ingest Load</h3>
                  <div class="h-32 border border-zinc-800 bg-black/40 relative overflow-hidden">
                    <div class="absolute inset-0 opacity-10" style={{ "background-image": "var(--pattern-medium)", "background-size": "8px 8px" }}></div>
                    <div class="absolute bottom-0 left-0 right-0 h-1/2 bg-gradient-to-t from-red-600/20 to-transparent"></div>
                  </div>
                </section>
                <AccessibleTerminal />
              </div>
            </div>
          </div>
        </Show>

        <div class="grid grid-cols-1 md:grid-cols-2 gap-8 pb-12">
          <ActionCard title="Fleet Onboarding" desc="Deploy sovereign agents to your infrastructure and establish real-time telemetry." shortcut="F1" onClick={() => window.location.href = '/onboarding'} />
          <ActionCard title="Vault Orchestration" desc="Access shared enterprise secrets and manage multi-tenant compliance policies." shortcut="F2" />
        </div>
      </div>

      <footer class="fixed bottom-0 w-full border-t border-[var(--border-bold)] bg-[var(--bg-deep)] p-2 pointer-events-none overflow-hidden">
        <div class="whitespace-nowrap animate-marquee inline-block text-[10px] tracking-[0.3em] uppercase text-[var(--text-muted)] opacity-30">SECURE_SUBSTRATE // RAFT_CONSENSUS_STABLE // SIEM_PIPELINE_ACTIVE // NO_THREATS_DETECTED // OBLIVRA_ENTERPRISE_READY</div>
      </footer>

      <CommandPalette open={isPaletteOpen()} onClose={() => setIsPaletteOpen(false)} actions={paletteActions} />
    </main>
  );
}

function StatBox(props: { label: string; value: string; sub: string; color?: string; severity?: Severity }) {
  return (
    <div class="bg-[var(--bg-deep)] p-6 space-y-1 relative overflow-hidden group">
      <div class="flex justify-between items-start">
        <span class="text-[10px] uppercase tracking-widest text-[var(--text-muted)]">{props.label}</span>
        {props.severity && <SeverityIcon severity={props.severity} size={14} class="opacity-80" />}
      </div>
      <div class="text-4xl font-black italic tracking-tighter" style={{ color: props.color || 'inherit' }}>{props.value}</div>
      <p class="text-xs text-[var(--text-muted)] opacity-60 tracking-tight">{props.sub}</p>
      {props.severity && <div class="absolute inset-0 opacity-[0.03] pointer-events-none" style={{ "background-image": `var(--pattern-${props.severity})`, "background-size": "4px 4px" }}></div>}
    </div>
  );
}

function ActionCard(props: { title: string; desc: string; shortcut: string; onClick?: () => void }) {
  return (
    <div onClick={props.onClick} class="border border-[var(--border-bold)] bg-[var(--bg-muted)] p-8 space-y-4 hover:border-[var(--accent-primary)] transition-colors group cursor-pointer relative overflow-hidden">
      <div class="absolute top-0 right-0 p-2 text-[8px] font-bold text-[var(--text-muted)] opacity-20 group-hover:opacity-100 transition-opacity">CMD + {props.shortcut}</div>
      <h3 class="text-3xl font-black uppercase tracking-tighter italic group-hover:text-[var(--accent-primary)] transition-colors">{props.title}</h3>
      <p class="text-[var(--text-muted)] leading-snug">{props.desc}</p>
      <div class="flex items-center gap-2 text-xs font-bold uppercase tracking-widest text-[var(--accent-primary)] opacity-0 group-hover:opacity-100 transition-all translate-x-[-10px] group-hover:translate-x-0">Launch Operation <span>→</span></div>
    </div>
  );
}
