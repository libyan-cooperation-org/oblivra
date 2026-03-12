import { Component, Show, For, createSignal } from 'solid-js';
import { useApp, AppState } from '@core/store';
import { useNavigate } from '@solidjs/router';
import { usePanelManager, cascadePos } from './PanelManager';

// ── Lazy panel loader (proper component so signals have a reactive owner) ──────
const LazyPanel: Component<{ importFn: () => Promise<any> }> = (props) => {
    const [Comp, setComp] = createSignal<any>(null);
    props.importFn()
        .then(m => setComp(() => m.default ?? Object.values(m)[0]))
        .catch(err => console.error('[CommandRail] lazy import failed:', err));
    return <Show when={Comp()}>{(C) => <C />}</Show>;
};

type NavTab = AppState['activeNavTab'] | 'temporal' | 'lineage' | 'decisions' | 'ledger' | 'replay' | 'soc'
    | 'agents' | 'ueba' | 'threat-hunter' | 'ndr' | 'purple-team' | 'graph'
    | 'war-mode' | 'forensics' | 'identity' | 'response' | 'ransomware' | 'credentials'
    | 'executive' | 'simulation' | 'data-destruction' | 'risk' | 'governance' | 'features'
    | 'recordings' | 'snippets' | 'notes' | 'sync' | 'tunnels' | 'ai-assistant' | 'mitre-heatmap';

// ── Icons ────────────────────────────────────────────────────────────────────
const Icons = {
    SIEM:       () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6"><path d="M2 3h6a4 4 0 014 4v14a3 3 0 00-3-3H2z"/><path d="M22 3h-6a4 4 0 00-4 4v14a3 3 0 013-3h7z"/></svg>,
    Ops:        () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6"><line x1="18" y1="20" x2="18" y2="10"/><line x1="12" y1="20" x2="12" y2="4"/><line x1="6" y1="20" x2="6" y2="14"/></svg>,
    Topology:   () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6"><circle cx="12" cy="12" r="3"/><circle cx="5" cy="18" r="2"/><circle cx="19" cy="18" r="2"/><circle cx="5" cy="6" r="2"/><circle cx="19" cy="6" r="2"/><path d="M7 16l4-2M17 16l-4-2M7 8l4 2M17 8l-4 2"/></svg>,
    Terminal:   () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6"><rect x="2" y="4" width="20" height="16" rx="3"/><polyline points="8 10 11 12 8 14"/><line x1="13" y1="15" x2="16" y2="15"/></svg>,
    Hosts:      () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6"><rect x="4" y="4" width="16" height="6" rx="2"/><rect x="4" y="14" width="16" height="6" rx="2"/><circle cx="7" cy="7" r="1"/><circle cx="7" cy="17" r="1"/></svg>,
    Compliance: () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6"><path d="M9 11l3 3L22 4"/><path d="M21 12v7a2 2 0 01-2 2H5a2 2 0 01-2-2V5a2 2 0 012-2h11"/></svg>,
    Vault:      () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6"><rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0110 0v4"/><circle cx="12" cy="16" r="1"/></svg>,
    Settings:   () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 00.33 1.82l.06.06a2 2 0 01-2.83 2.83l-.06-.06a1.65 1.65 0 00-1.82-.33 1.65 1.65 0 00-1 1.51V21a2 2 0 01-4 0v-.09A1.65 1.65 0 009 19.4a1.65 1.65 0 00-1.82.33l-.06.06a2 2 0 01-2.83-2.83l.06-.06A1.65 1.65 0 004.68 15a1.65 1.65 0 00-1.51-1H3a2 2 0 010-4h.09A1.65 1.65 0 004.6 9a1.65 1.65 0 00-.33-1.82l-.06-.06a2 2 0 012.83-2.83l.06.06A1.65 1.65 0 009 4.68a1.65 1.65 0 001-1.51V3a2 2 0 014 0v.09a1.65 1.65 0 001 1.51 1.65 1.65 0 001.82-.33l.06-.06a2 2 0 012.83 2.83l-.06.06A1.65 1.65 0 0019.4 9a1.65 1.65 0 001.51 1H21a2 2 0 010 4h-.09a1.65 1.65 0 00-1.51 1z"/></svg>,
    Dashboard:  () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6"><rect x="3" y="3" width="7" height="9" rx="1.5"/><rect x="14" y="3" width="7" height="5" rx="1.5"/><rect x="14" y="12" width="7" height="9" rx="1.5"/><rect x="3" y="16" width="7" height="5" rx="1.5"/></svg>,
    Tunnels:    () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6"><circle cx="12" cy="12" r="10"/><line x1="2" y1="12" x2="22" y2="12"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/></svg>,
    Snippets:   () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6"><polyline points="16 18 22 12 16 6"/><polyline points="8 6 2 12 8 18"/></svg>,
    Recordings: () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6"><circle cx="12" cy="12" r="10"/><circle cx="12" cy="12" r="3"/></svg>,
    Notes:      () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="16" y1="13" x2="8" y2="13"/><line x1="16" y1="17" x2="8" y2="17"/></svg>,
    Team:       () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>,
    Sync:       () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6"><polyline points="23 4 23 10 17 10"/><polyline points="1 20 1 14 7 14"/><path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"/></svg>,
    Plugins:    () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6"><path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/><polyline points="3.27 6.96 12 12.01 20.73 6.96"/><line x1="12" y1="22.08" x2="12" y2="12"/></svg>,
    Health:     () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6"><path d="M22 12h-4l-3 9L9 3l-3 9H2"/></svg>,
    Metrics:    () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6"><path d="M21.21 15.89A10 10 0 1 1 8 2.83"/><path d="M22 12A10 10 0 0 0 12 2v10z"/></svg>,
    Alerts:     () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6"><path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9"/><path d="M13.73 21a2 2 0 0 1-3.46 0"/></svg>,
    Security:   () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/></svg>,
    SOC:        () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6"><rect x="3" y="3" width="18" height="18" rx="3"/><path d="M3 9h18"/><path d="M9 21V9"/></svg>,
    Audit:      () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/><polyline points="9 12 11 14 15 10"/></svg>,
    ChevronRight: () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5"><polyline points="9 18 15 12 9 6"/></svg>,
    Agents:     () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6"><rect x="4" y="4" width="16" height="16" rx="3"/><line x1="9" y1="9" x2="9.01" y2="9"/><line x1="15" y1="9" x2="15.01" y2="9"/><path d="M9 15h6"/></svg>,
    UEBA:       () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6"><path d="M16 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="8.5" cy="7" r="4"/><path d="M20 8v6M23 11h-6"/></svg>,
    Hunter:     () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/><line x1="8" y1="11" x2="14" y2="11"/><line x1="11" y1="8" x2="11" y2="14"/></svg>,
    NDR:        () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6"><circle cx="6" cy="6" r="3"/><circle cx="18" cy="6" r="3"/><circle cx="6" cy="18" r="3"/><circle cx="18" cy="18" r="3"/><line x1="6" y1="9" x2="6" y2="15"/><line x1="18" y1="9" x2="18" y2="15"/><line x1="9" y1="6" x2="15" y2="6"/><line x1="9" y1="18" x2="15" y2="18"/></svg>,
    Purple:     () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6"><polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"/></svg>,
    Graph:      () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6"><circle cx="12" cy="5" r="3"/><circle cx="5" cy="19" r="3"/><circle cx="19" cy="19" r="3"/><line x1="12" y1="8" x2="5" y2="16"/><line x1="12" y1="8" x2="19" y2="16"/><line x1="8" y1="19" x2="16" y2="19"/></svg>,
    War:        () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>,
    Forensics:  () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>,
    Identity:   () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6"><rect x="3" y="4" width="18" height="16" rx="2"/><circle cx="9" cy="10" r="2"/><path d="M15 8h2M15 12h2"/><path d="M7 16h10"/></svg>,
    Response:   () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6"><path d="M4 15s1-1 4-1 5 2 8 2 4-1 4-1V3s-1 1-4 1-5-2-8-2-4 1-4 1z"/><line x1="4" y1="22" x2="4" y2="15"/></svg>,
    Executive:  () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6"><path d="M12 20V10"/><path d="M18 20V4"/><path d="M6 20v-4"/></svg>,
    AI:         () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6"><path d="M12 2a10 10 0 1 0 10 10H12V2z"/><path d="M12 12L2.69 7.11"/><path d="M12 12l4.89 8.69"/></svg>,
    Mitre:      () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.6"><rect x="3" y="3" width="18" height="18" rx="2"/><path d="M3 9h18"/><path d="M3 15h18"/><path d="M9 3v18"/><path d="M15 3v18"/></svg>,
};

// ── Route map ─────────────────────────────────────────────────────────────────
const routeMap: Record<string, string> = {
    dashboard:  '/dashboard',
    siem:       '/siem',
    alerts:     '/alerts',
    topology:   '/topology',
    health:     '/monitoring',
    terminal:   '/terminal',
    hosts:      '/hosts',
    agents:     '/agents',
    ops:        '/ops',
    soc:        '/soc',
    response:   '/response',
    ueba:           '/ueba',
    'threat-hunter':'/threat-hunter',
    ndr:            '/ndr',
    'purple-team':  '/purple-team',
    graph:          '/graph',
    ransomware:     '/ransomware',
    simulation:     '/simulation',
    compliance: '/compliance',
    vault:      '/vault',
    identity:   '/identity',
    'war-mode': '/war-mode',
    forensics:  '/forensics',
    security:   '/trust',
    team:       '/team',
    temporal:   '/temporal-integrity',
    lineage:    '/lineage',
    decisions:  '/decisions',
    ledger:     '/ledger',
    replay:     '/response-replay',
    plugins:    '/plugins',
    executive:  '/executive',
    settings:   '/workspace',
    recordings: '/recordings',
    snippets:   '/snippets',
    notes:      '/notes',
    sync:       '/sync',
    tunnels:    '/tunnels',
    'ai-assistant': '/ai-assistant',
    'mitre-heatmap': '/mitre-heatmap',
};

type PrimaryItem = { id: NavTab; icon: () => any; label: string; badge?: () => number; urgent?: boolean };

const auditItems: { id: NavTab; label: string }[] = [
    { id: 'temporal',  label: 'Temporal Integrity' },
    { id: 'lineage',   label: 'Data Lineage' },
    { id: 'decisions', label: 'Decision Log' },
    { id: 'ledger',    label: 'Evidence Ledger' },
    { id: 'replay',    label: 'Response Replay' },
];

const NavButton: Component<{
    item: PrimaryItem;
    active: boolean;
    onClick: () => void;
    onCtrlClick?: () => void;
}> = (props) => {
    const badge = props.item.badge?.() ?? 0;
    return (
        <button
            title={`${props.item.label}  (Ctrl+Click → open as panel)`}
            onClick={(e) => { if (e.ctrlKey || e.metaKey) { e.preventDefault(); props.onCtrlClick?.(); } else { props.onClick(); } }}
            class={`cr-nav-btn${props.active ? ' cr-nav-btn--active' : ''}`}
        >
            <div class="cr-nav-icon">
                <props.item.icon />
            </div>
            <span class="cr-nav-label">{props.item.label}</span>
            <Show when={badge > 0}>
                <div class={`cr-nav-badge${props.item.urgent ? ' cr-nav-badge--urgent' : ''}`}>
                    {badge > 99 ? '99+' : badge}
                </div>
            </Show>
        </button>
    );
};

const AuditFlyout: Component<{
    active: boolean;
    currentTab: string;
    onSelect: (id: NavTab) => void;
    onClose: () => void;
}> = (props) => (
    <div class="cr-flyout" onMouseLeave={props.onClose}>
        <div class="cr-flyout-header">AUDIT TRAIL</div>
        <For each={auditItems}>
            {(item) => (
                <button
                    onClick={() => { props.onSelect(item.id); props.onClose(); }}
                    class={`cr-flyout-item${props.currentTab === item.id ? ' cr-flyout-item--active' : ''}`}
                >
                    {item.label}
                </button>
            )}
        </For>
    </div>
);

const SectionDivider: Component<{ label: string }> = (props) => (
    <div class="cr-divider">
        <span class="cr-divider-label">{props.label}</span>
    </div>
);

export const CommandRail: Component = () => {
    const [state, actions] = useApp();
    const navigate = useNavigate();
    const [auditOpen, setAuditOpen] = createSignal(false);
    const [flyoutTop, setFlyoutTop] = createSignal(0);

    const { openPanel } = usePanelManager();

    const go = (id: NavTab) => {
        actions.setActiveNavTab(id as any);
        const route = routeMap[id as string];
        if (route) navigate(route);
    };

    // Ctrl+Click: open the nav view as a floating panel
    // Each entry maps a NavTab id to a lazy dynamic import of its component.
    // We use a function-per-id pattern so the import only fires on demand.
    // Maps NavTab id -> dynamic import fn using ACTUAL file paths verified against the tree.
    // Falls back to a placeholder div for unmapped ids.
    const panelImports: Record<string, () => Promise<any>> = {
        // OBSERVE
        dashboard:       () => import('../dashboard/Dashboard'),
        siem:            () => import('../siem/SIEMPanel'),
        alerts:          () => import('../siem/AlertDashboard'),
        topology:        () => import('../intelligence/NetworkMap'),
        health:          () => import('../monitoring/HealthPanel'),
        // OPERATE
        terminal:        () => import('../terminal/TerminalLayout'),
        hosts:           () => import('../sidebar/HostTree'),
        agents:          () => import('../fleet/AgentConsole'),
        ops:             () => import('../ops/LiveTailPanel'),
        soc:             () => import('../soc/SOCWorkspace'),
        response:        () => import('../incident/CommandCenter'),
        // INTEL
        ueba:            () => import('../intelligence/UEBAPanel'),
        'threat-hunter': () => import('../security/ThreatHunter'),
        ndr:             () => import('../intelligence/NetworkMap'),
        'purple-team':   () => import('../../pages/PurpleTeam'),
        graph:           () => import('../intelligence/ThreatGraph'),
        // GOVERN
        compliance:      () => import('../compliance/CompliancePanel'),
        vault:           () => import('../vault/VaultManager'),
        identity:        () => import('../settings/UsersPanel'),
        security:        () => import('../security/SecurityPanel'),
        forensics:       () => import('../security/ForensicView'),
        'war-mode':      () => import('../../pages/WarMode'),
        // SYSTEM
        executive:       () => import('../../pages/ExecutiveDashboard'),
        plugins:         () => import('../plugins/PluginPanel'),
        settings:        () => import('../settings/SettingsManager'),
        workspaces:      () => import('../workspace/WorkspacePanel'),
        // AUDIT
        temporal:        () => import('../../pages/TemporalIntegrity'),
        lineage:         () => import('../../pages/LineageExplorer'),
        decisions:       () => import('../../pages/DecisionInspector'),
        ledger:          () => import('../../pages/EvidenceLedger'),
        replay:          () => import('../../pages/ResponseReplay'),
        // EXTRAS
        fleet:           () => import('../fleet/FleetDashboard'),
        incidents:       () => import('../incident/CommandCenter'),
    };

    const openAsPanel = (id: NavTab, label: string) => {
        const importFn = panelImports[String(id)];
        openPanel({
            id: String(id),
            title: label,
            defaultPos: cascadePos({ x: 100, y: 80 }),
            defaultSize: { w: 1100, h: 700 },
            // Wrap in a proper component so createSignal has a reactive owner
            content: importFn
                ? () => <LazyPanel importFn={importFn} />
                : () => (
                    <div style="padding:20px;color:var(--text-muted);font-family:var(--font-mono);font-size:11px;text-transform:uppercase;letter-spacing:1px">
                        {label}
                    </div>
                ),
        });
    };

    const auditActive = () => auditItems.some(i => i.id === state.activeNavTab);

    const observe: PrimaryItem[] = [
        { id: 'dashboard', icon: Icons.Dashboard, label: 'Dash' },
        { id: 'siem',      icon: Icons.SIEM,      label: 'SIEM' },
        { id: 'alerts',    icon: Icons.Alerts,    label: 'Alerts',
          badge: () => state.notifications.filter((n: any) => n.type === 'error').length,
          urgent: true },
        { id: 'recordings',icon: Icons.Recordings,label: 'Recs' },
        { id: 'topology',  icon: Icons.Topology,  label: 'Net' },
        { id: 'mitre-heatmap', icon: Icons.Mitre, label: 'Mitre' },
        { id: 'health',    icon: Icons.Health,    label: 'Health' },
    ];

    const operate: PrimaryItem[] = [
        { id: 'terminal',  icon: Icons.Terminal,  label: 'Shell' },
        { id: 'tunnels',   icon: Icons.Tunnels,   label: 'Tunnels' },
        { id: 'hosts',     icon: Icons.Hosts,     label: 'Hosts' },
        { id: 'agents',    icon: Icons.Agents,    label: 'Agents' },
        { id: 'ops',       icon: Icons.Ops,       label: 'Ops' },
        { id: 'soc',       icon: Icons.SOC,       label: 'SOC' },
        { id: 'snippets',  icon: Icons.Snippets,  label: 'Snips' },
        { id: 'notes',     icon: Icons.Notes,     label: 'Notes' },
        { id: 'ai-assistant', icon: Icons.AI,     label: 'AI Shell' },
        { id: 'response',  icon: Icons.Response,  label: 'SOAR' },
    ];

    const intel: PrimaryItem[] = [
        { id: 'ueba',           icon: Icons.UEBA,   label: 'UEBA' },
        { id: 'threat-hunter',  icon: Icons.Hunter, label: 'Hunt' },
        { id: 'ndr',            icon: Icons.NDR,    label: 'NDR' },
        { id: 'purple-team',    icon: Icons.Purple, label: 'Purple' },
        { id: 'graph',          icon: Icons.Graph,  label: 'Graph' },
    ];

    const govern: PrimaryItem[] = [
        { id: 'compliance', icon: Icons.Compliance, label: 'Comply' },
        { id: 'vault',      icon: Icons.Vault,      label: 'Vault' },
        { id: 'identity',   icon: Icons.Identity,   label: 'Users' },
        { id: 'security',   icon: Icons.Security,   label: 'Trust' },
        { id: 'forensics',  icon: Icons.Forensics,  label: 'Forensics' },
        { id: 'war-mode',   icon: Icons.War,        label: 'WarMode' },
    ];

    const system: PrimaryItem[] = [
        { id: 'executive', icon: Icons.Executive, label: 'Exec' },
        { id: 'plugins',   icon: Icons.Plugins,   label: 'Plugins' },
        { id: 'sync',      icon: Icons.Sync,      label: 'Sync' },
        { id: 'settings',  icon: Icons.Settings,  label: 'Config' },
    ];

    return (
        <>
            <style>{`
                .cr-rail {
                    width: 64px;
                    min-width: 64px;
                    height: 100%;
                    background: var(--surface-1);
                    border-right: 1px solid var(--border-primary);
                    display: flex;
                    flex-direction: column;
                    z-index: 1000;
                    overflow-y: auto;
                    overflow-x: hidden;
                }
                .cr-rail::-webkit-scrollbar { width: 0; }

                .cr-logo {
                    height: 48px;
                    display: flex;
                    align-items: center;
                    justify-content: center;
                    border-bottom: 1px solid var(--border-primary);
                    flex-shrink: 0;
                    background: var(--surface-1);
                }

                .cr-nav-btn {
                    position: relative;
                    background: transparent;
                    border: none;
                    color: var(--text-muted);
                    width: 100%;
                    padding: 8px 0 6px 0;
                    cursor: pointer;
                    display: flex;
                    flex-direction: column;
                    align-items: center;
                    gap: 3px;
                    transition: color var(--transition-fast), background var(--transition-fast);
                    border-radius: 0;
                }
                .cr-nav-btn:hover {
                    color: var(--text-secondary);
                    background: rgba(87, 139, 255, 0.06);
                }
                .cr-nav-btn--active {
                    color: var(--accent-primary) !important;
                    background: rgba(87, 139, 255, 0.12) !important;
                }
                .cr-nav-btn--active::before {
                    content: '';
                    position: absolute;
                    left: 0;
                    top: 20%;
                    bottom: 20%;
                    width: 3px;
                    background: var(--accent-primary);
                    border-radius: 0 3px 3px 0;
                }

                .cr-nav-icon {
                    width: 18px;
                    height: 18px;
                    transition: opacity var(--transition-fast);
                }
                .cr-nav-btn:not(.cr-nav-btn--active) .cr-nav-icon {
                    opacity: 0.5;
                }
                .cr-nav-btn--active .cr-nav-icon {
                    opacity: 1;
                    filter: drop-shadow(0 0 6px rgba(87,139,255,0.4));
                }

                .cr-nav-label {
                    font-family: var(--font-ui);
                    font-size: 9px;
                    font-weight: 500;
                    text-transform: uppercase;
                    letter-spacing: 0.3px;
                    line-height: 1;
                }
                .cr-nav-btn--active .cr-nav-label {
                    font-weight: 700;
                }

                .cr-nav-badge {
                    position: absolute;
                    top: 4px;
                    right: 6px;
                    background: var(--accent-primary);
                    color: var(--surface-0);
                    font-family: var(--font-mono);
                    font-size: 7px;
                    font-weight: 800;
                    padding: 1px 4px;
                    line-height: 1.2;
                    min-width: 14px;
                    text-align: center;
                    border-radius: 8px;
                }
                .cr-nav-badge--urgent {
                    background: var(--alert-critical);
                }

                .cr-divider {
                    display: flex;
                    align-items: center;
                    justify-content: center;
                    padding: 8px 4px 3px 4px;
                    margin-top: 2px;
                }
                .cr-divider-label {
                    font-family: var(--font-mono);
                    font-size: 6.5px;
                    font-weight: 700;
                    color: var(--text-muted);
                    opacity: 0.35;
                    letter-spacing: 1px;
                    text-transform: uppercase;
                }

                .cr-audit-btn {
                    position: relative;
                    background: transparent;
                    border: none;
                    color: var(--text-muted);
                    width: 100%;
                    padding: 8px 0 6px 0;
                    cursor: pointer;
                    display: flex;
                    flex-direction: column;
                    align-items: center;
                    gap: 3px;
                    transition: all var(--transition-fast);
                }
                .cr-audit-btn:hover { 
                    color: var(--text-secondary);
                    background: rgba(87, 139, 255, 0.06);
                }
                .cr-audit-btn--active {
                    color: var(--accent-primary) !important;
                    background: rgba(87, 139, 255, 0.12) !important;
                }
                .cr-audit-btn--active::before {
                    content: '';
                    position: absolute;
                    left: 0;
                    top: 20%;
                    bottom: 20%;
                    width: 3px;
                    background: var(--accent-primary);
                    border-radius: 0 3px 3px 0;
                }
                .cr-audit-icon {
                    width: 18px;
                    height: 18px;
                    opacity: 0.5;
                }
                .cr-audit-btn--active .cr-audit-icon {
                    opacity: 1;
                }
                .cr-audit-label {
                    font-family: var(--font-ui);
                    font-size: 9px;
                    font-weight: 500;
                    text-transform: uppercase;
                    letter-spacing: 0.3px;
                    line-height: 1;
                }

                .cr-flyout {
                    position: fixed;
                    left: 65px;
                    background: var(--surface-2);
                    border: 1px solid var(--border-secondary);
                    border-radius: var(--radius-md);
                    z-index: 2000;
                    min-width: 192px;
                    padding: 4px;
                    box-shadow: var(--shadow-lg);
                }
                .cr-flyout-header {
                    font-family: var(--font-mono);
                    font-size: 9px;
                    font-weight: 800;
                    color: var(--text-muted);
                    padding: 8px 10px 6px 10px;
                    letter-spacing: 1px;
                    text-transform: uppercase;
                    border-bottom: 1px solid var(--border-primary);
                    margin-bottom: 4px;
                }
                .cr-flyout-item {
                    display: block;
                    width: 100%;
                    background: transparent;
                    border: none;
                    border-radius: var(--radius-sm);
                    color: var(--text-secondary);
                    padding: 7px 10px;
                    text-align: left;
                    font-family: var(--font-ui);
                    font-size: 12px;
                    cursor: pointer;
                    transition: all var(--transition-fast);
                }
                .cr-flyout-item:hover {
                    background: var(--surface-3);
                    color: var(--text-primary);
                }
                .cr-flyout-item--active {
                    background: rgba(87, 139, 255, 0.12);
                    color: var(--accent-primary);
                    font-weight: 600;
                }
            `}</style>

            <nav class="cr-rail">
                {/* Logo */}
                <div class="cr-logo">
                    <svg width="24" height="24" viewBox="0 0 24 24" fill="none">
                        <path d="M12 2L4 6.5v11L12 22l8-4.5v-11L12 2z" stroke="var(--accent-primary)" stroke-width="1.5" fill="none"/>
                        <path d="M12 6.5L7 9.25v5.5L12 17.5l5-2.75v-5.5L12 6.5z" fill="var(--accent-primary)" opacity="0.2"/>
                        <circle cx="12" cy="12" r="2" fill="var(--accent-primary)"/>
                    </svg>
                </div>

                {/* OBSERVE */}
                <SectionDivider label="OBS" />
                <For each={observe}>
                    {(item) => (
                        <NavButton item={item} active={state.activeNavTab === item.id} onClick={() => go(item.id)} onCtrlClick={() => openAsPanel(item.id, item.label)} />
                    )}
                </For>

                {/* OPERATE */}
                <SectionDivider label="OPS" />
                <For each={operate}>
                    {(item) => (
                        <NavButton item={item} active={state.activeNavTab === item.id} onClick={() => go(item.id)} onCtrlClick={() => openAsPanel(item.id, item.label)} />
                    )}
                </For>

                {/* INTEL */}
                <SectionDivider label="INTEL" />
                <For each={intel}>
                    {(item) => (
                        <NavButton item={item} active={state.activeNavTab === item.id} onClick={() => go(item.id)} onCtrlClick={() => openAsPanel(item.id, item.label)} />
                    )}
                </For>

                {/* GOVERN */}
                <SectionDivider label="GOV" />
                <For each={govern}>
                    {(item) => (
                        <NavButton item={item} active={state.activeNavTab === item.id} onClick={() => go(item.id)} onCtrlClick={() => openAsPanel(item.id, item.label)} />
                    )}
                </For>

                {/* AUDIT flyout */}
                <button
                    title="Audit Trail"
                    class={`cr-audit-btn${auditActive() ? ' cr-audit-btn--active' : ''}`}
                    onMouseEnter={(e) => {
                        setFlyoutTop(e.currentTarget.getBoundingClientRect().top);
                        setAuditOpen(true);
                    }}
                >
                    <div class="cr-audit-icon"><Icons.Audit /></div>
                    <span class="cr-audit-label">Audit</span>
                </button>

                <Show when={auditOpen()}>
                    <div style={{ position: 'fixed', left: '65px', top: `${flyoutTop()}px`, 'z-index': 2000 }}>
                        <AuditFlyout
                            active={auditActive()}
                            currentTab={state.activeNavTab as string}
                            onSelect={go}
                            onClose={() => setAuditOpen(false)}
                        />
                    </div>
                </Show>

                {/* Spacer */}
                <div style={{ flex: 1 }} />

                {/* SYSTEM */}
                <SectionDivider label="SYS" />
                <For each={system}>
                    {(item) => (
                        <NavButton item={item} active={state.activeNavTab === item.id} onClick={() => go(item.id)} />
                    )}
                </For>
                <div style={{ height: '8px' }} />
            </nav>
        </>
    );
};
