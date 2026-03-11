import { Component, Show, For, createSignal } from 'solid-js';
import { useApp, AppState } from '@core/store';
import { useNavigate } from '@solidjs/router';

type NavTab = AppState['activeNavTab'] | 'temporal' | 'lineage' | 'decisions' | 'ledger' | 'replay' | 'soc'
    | 'agents' | 'ueba' | 'threat-hunter' | 'ndr' | 'purple-team' | 'graph'
    | 'war-mode' | 'forensics' | 'identity' | 'response' | 'ransomware' | 'credentials'
    | 'executive' | 'simulation' | 'data-destruction' | 'risk' | 'governance' | 'features';

// ── Icons ────────────────────────────────────────────────────────────────────
const Icons = {
    SIEM:       () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M2 3h6a4 4 0 014 4v14a3 3 0 00-3-3H2z"/><path d="M22 3h-6a4 4 0 00-4 4v14a3 3 0 013-3h7z"/></svg>,
    Ops:        () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><line x1="18" y1="20" x2="18" y2="10"/><line x1="12" y1="20" x2="12" y2="4"/><line x1="6" y1="20" x2="6" y2="14"/></svg>,
    Topology:   () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="3"/><circle cx="5" cy="18" r="2"/><circle cx="19" cy="18" r="2"/><circle cx="5" cy="6" r="2"/><circle cx="19" cy="6" r="2"/><path d="M7 16l4-2M17 16l-4-2M7 8l4 2M17 8l-4 2"/></svg>,
    Terminal:   () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="2" y="4" width="20" height="16" rx="2"/><polyline points="8 10 11 12 8 14"/><line x1="13" y1="15" x2="16" y2="15"/></svg>,
    Hosts:      () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="4" y="4" width="16" height="6" rx="1.5"/><rect x="4" y="14" width="16" height="6" rx="1.5"/><circle cx="7" cy="7" r="1"/><circle cx="7" cy="17" r="1"/></svg>,
    Compliance: () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M9 11l3 3L22 4"/><path d="M21 12v7a2 2 0 01-2 2H5a2 2 0 01-2-2V5a2 2 0 012-2h11"/></svg>,
    Vault:      () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="3" y="11" width="18" height="11" rx="2"/><path d="M7 11V7a5 5 0 0110 0v4"/><circle cx="12" cy="16" r="1"/></svg>,
    Settings:   () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="3"/><path d="M19.4 15a1.65 1.65 0 00.33 1.82l.06.06a2 2 0 01-2.83 2.83l-.06-.06a1.65 1.65 0 00-1.82-.33 1.65 1.65 0 00-1 1.51V21a2 2 0 01-4 0v-.09A1.65 1.65 0 009 19.4a1.65 1.65 0 00-1.82.33l-.06.06a2 2 0 01-2.83-2.83l.06-.06A1.65 1.65 0 004.68 15a1.65 1.65 0 00-1.51-1H3a2 2 0 010-4h.09A1.65 1.65 0 004.6 9a1.65 1.65 0 00-.33-1.82l-.06-.06a2 2 0 012.83-2.83l.06.06A1.65 1.65 0 009 4.68a1.65 1.65 0 001-1.51V3a2 2 0 014 0v.09a1.65 1.65 0 001 1.51 1.65 1.65 0 001.82-.33l.06-.06a2 2 0 012.83 2.83l-.06.06A1.65 1.65 0 0019.4 9a1.65 1.65 0 001.51 1H21a2 2 0 010 4h-.09a1.65 1.65 0 00-1.51 1z"/></svg>,
    Dashboard:  () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="3" y="3" width="7" height="9"/><rect x="14" y="3" width="7" height="5"/><rect x="14" y="12" width="7" height="9"/><rect x="3" y="16" width="7" height="5"/></svg>,
    Tunnels:    () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="10"/><line x1="2" y1="12" x2="22" y2="12"/><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z"/></svg>,
    Snippets:   () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><polyline points="16 18 22 12 16 6"/><polyline points="8 6 2 12 8 18"/></svg>,
    Recordings: () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="10"/><circle cx="12" cy="12" r="3"/></svg>,
    Notes:      () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z"/><polyline points="14 2 14 8 20 8"/><line x1="16" y1="13" x2="8" y2="13"/><line x1="16" y1="17" x2="8" y2="17"/></svg>,
    Team:       () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M23 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/></svg>,
    Sync:       () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><polyline points="23 4 23 10 17 10"/><polyline points="1 20 1 14 7 14"/><path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15"/></svg>,
    Plugins:    () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z"/><polyline points="3.27 6.96 12 12.01 20.73 6.96"/><line x1="12" y1="22.08" x2="12" y2="12"/></svg>,
    Health:     () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M22 12h-4l-3 9L9 3l-3 9H2"/></svg>,
    Metrics:    () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M21.21 15.89A10 10 0 1 1 8 2.83"/><path d="M22 12A10 10 0 0 0 12 2v10z"/></svg>,
    Updater:    () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4"/><polyline points="7 10 12 15 17 10"/><line x1="12" y1="15" x2="12" y2="3"/></svg>,
    Alerts:     () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9"/><path d="M13.73 21a2 2 0 0 1-3.46 0"/></svg>,
    Security:   () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/></svg>,
    SOC:        () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="3" y="3" width="18" height="18" rx="2"/><path d="M3 9h18"/><path d="M9 21V9"/></svg>,
    Audit:      () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z"/><polyline points="9 12 11 14 15 10"/></svg>,
    ChevronRight: () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><polyline points="9 18 15 12 9 6"/></svg>,
    // New icons for missing pages
    Agents:     () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="4" y="4" width="16" height="16" rx="2"/><line x1="9" y1="9" x2="9.01" y2="9"/><line x1="15" y1="9" x2="15.01" y2="9"/><path d="M9 15h6"/></svg>,
    UEBA:       () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M16 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2"/><circle cx="8.5" cy="7" r="4"/><path d="M20 8v6M23 11h-6"/></svg>,
    Hunter:     () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/><line x1="8" y1="11" x2="14" y2="11"/><line x1="11" y1="8" x2="11" y2="14"/></svg>,
    NDR:        () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="6" cy="6" r="3"/><circle cx="18" cy="6" r="3"/><circle cx="6" cy="18" r="3"/><circle cx="18" cy="18" r="3"/><line x1="6" y1="9" x2="6" y2="15"/><line x1="18" y1="9" x2="18" y2="15"/><line x1="9" y1="6" x2="15" y2="6"/><line x1="9" y1="18" x2="15" y2="18"/></svg>,
    Purple:     () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><polygon points="13 2 3 14 12 14 11 22 21 10 12 10 13 2"/></svg>,
    Graph:      () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="5" r="3"/><circle cx="5" cy="19" r="3"/><circle cx="19" cy="19" r="3"/><line x1="12" y1="8" x2="5" y2="16"/><line x1="12" y1="8" x2="19" y2="16"/><line x1="8" y1="19" x2="16" y2="19"/></svg>,
    War:        () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M10.29 3.86L1.82 18a2 2 0 0 0 1.71 3h16.94a2 2 0 0 0 1.71-3L13.71 3.86a2 2 0 0 0-3.42 0z"/><line x1="12" y1="9" x2="12" y2="13"/><line x1="12" y1="17" x2="12.01" y2="17"/></svg>,
    Forensics:  () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>,
    Identity:   () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="3" y="4" width="18" height="16" rx="2"/><circle cx="9" cy="10" r="2"/><path d="M15 8h2M15 12h2"/><path d="M7 16h10"/></svg>,
    Response:   () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M4 15s1-1 4-1 5 2 8 2 4-1 4-1V3s-1 1-4 1-5-2-8-2-4 1-4 1z"/><line x1="4" y1="22" x2="4" y2="15"/></svg>,
    Executive:  () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M12 20V10"/><path d="M18 20V4"/><path d="M6 20v-4"/></svg>,
};

// ── Route map ─────────────────────────────────────────────────────────────────
const routeMap: Record<string, string> = {
    // OBSERVE
    dashboard:  '/dashboard',
    siem:       '/siem',
    alerts:     '/siem',  // Alerts are within SIEM
    topology:   '/topology',
    health:     '/monitoring',
    // OPERATE
    terminal:   '/terminal',
    hosts:      '/hosts',
    agents:     '/agents',
    ops:        '/ops',
    soc:        '/soc',
    tunnels:    '/terminal',
    response:   '/response',
    // INTEL
    ueba:           '/ueba',
    'threat-hunter':'/threat-hunter',
    ndr:            '/ndr',
    'purple-team':  '/purple-team',
    graph:          '/graph',
    ransomware:     '/ransomware',
    simulation:     '/simulation',
    // GOVERN
    compliance: '/compliance',
    vault:      '/vault',
    identity:   '/identity',
    'war-mode': '/war-mode',
    forensics:  '/forensics',
    security:   '/trust',
    team:       '/team',
    // AUDIT flyout
    temporal:   '/temporal-integrity',
    lineage:    '/lineage',
    decisions:  '/decisions',
    ledger:     '/ledger',
    replay:     '/response-replay',
    // SYSTEM
    plugins:    '/plugins',
    executive:  '/executive',
    settings:   '/workspace',
};

// ── Primary nav items (always visible) ───────────────────────────────────────
type PrimaryItem = { id: NavTab; icon: () => any; label: string; badge?: () => number; urgent?: boolean };

// ── Audit submenu items ───────────────────────────────────────────────────────
const auditItems: { id: NavTab; label: string }[] = [
    { id: 'temporal',  label: 'Temporal Integrity' },
    { id: 'lineage',   label: 'Data Lineage' },
    { id: 'decisions', label: 'Decision Log' },
    { id: 'ledger',    label: 'Evidence Ledger' },
    { id: 'replay',    label: 'Response Replay' },
];

// ── NavButton ─────────────────────────────────────────────────────────────────
const NavButton: Component<{
    item: PrimaryItem;
    active: boolean;
    onClick: () => void;
}> = (props) => {
    const badge = props.item.badge?.() ?? 0;
    return (
        <button
            title={props.item.label}
            onClick={props.onClick}
            style={{
                background: props.active ? 'rgba(0,212,170,0.07)' : 'transparent',
                border: 'none',
                'border-left': props.active ? '2px solid var(--accent-primary)' : '2px solid transparent',
                color: props.active ? 'var(--accent-primary)' : 'var(--text-muted)',
                width: '100%',
                padding: '7px 0 5px 0',
                cursor: 'pointer',
                display: 'flex',
                'flex-direction': 'column',
                'align-items': 'center',
                gap: '3px',
                position: 'relative',
                transition: 'background var(--transition-fast), color var(--transition-fast), border-color var(--transition-fast)',
            }}
        >
            <div style={{
                width: '15px',
                height: '15px',
                opacity: props.active ? '1' : '0.6',
                transition: 'opacity var(--transition-fast)',
            }}>
                <props.item.icon />
            </div>
            <span style={{
                'font-family': 'var(--font-ui)',
                'font-size': '7.5px',
                'font-weight': props.active ? 700 : 500,
                'text-transform': 'uppercase',
                'letter-spacing': '0.4px',
                'line-height': 1,
                color: props.active ? 'var(--accent-primary)' : 'var(--text-muted)',
            }}>
                {props.item.label}
            </span>
            <Show when={badge > 0}>
                <div style={{
                    position: 'absolute',
                    top: '3px',
                    right: '5px',
                    background: props.item.urgent ? 'var(--alert-critical)' : 'var(--accent-primary)',
                    color: '#000',
                    'font-family': 'var(--font-mono)',
                    'font-size': '7px',
                    'font-weight': 800,
                    padding: '1px 3px',
                    'line-height': 1,
                    'min-width': '13px',
                    'text-align': 'center',
                    'border-radius': '2px',
                }}>
                    {badge > 99 ? '99+' : badge}
                </div>
            </Show>
        </button>
    );
};

// ── AuditFlyout ───────────────────────────────────────────────────────────────
const AuditFlyout: Component<{
    active: boolean;
    currentTab: string;
    onSelect: (id: NavTab) => void;
    onClose: () => void;
}> = (props) => (
    <div
        style={{
            position: 'fixed',
            left: '64px',
            top: 'auto',
            background: 'var(--surface-1)',
            border: '1px solid var(--border-primary)',
            'border-left': '2px solid var(--accent-primary)',
            'z-index': 2000,
            'min-width': '180px',
            padding: '4px 0',
        }}
        onMouseLeave={props.onClose}
    >
        <div style={{
            'font-family': 'var(--font-mono)',
            'font-size': '9px',
            'font-weight': 800,
            color: 'var(--text-muted)',
            padding: '6px 12px 4px 12px',
            'letter-spacing': '1px',
            'text-transform': 'uppercase',
            'border-bottom': '1px solid var(--border-primary)',
            'margin-bottom': '2px',
        }}>
            AUDIT TRAIL
        </div>
        <For each={auditItems}>
            {(item) => (
                <button
                    onClick={() => { props.onSelect(item.id); props.onClose(); }}
                    style={{
                        display: 'block',
                        width: '100%',
                        background: props.currentTab === item.id ? 'var(--surface-2)' : 'transparent',
                        border: 'none',
                        'border-left': props.currentTab === item.id ? '2px solid var(--accent-primary)' : '2px solid transparent',
                        color: props.currentTab === item.id ? 'var(--text-primary)' : 'var(--text-secondary)',
                        padding: '8px 12px',
                        'text-align': 'left',
                        'font-family': 'var(--font-ui)',
                        'font-size': '11px',
                        cursor: 'pointer',
                    }}
                >
                    {item.label}
                </button>
            )}
        </For>
    </div>
);

// ── Divider ───────────────────────────────────────────────────────────────────
const Divider: Component<{ label: string }> = (props) => (
    <div style={{
        display: 'flex',
        'align-items': 'center',
        padding: '8px 8px 3px 8px',
        'margin-top': '4px',
        gap: '4px',
    }}>
        <div style={{ flex: 1, height: '1px', background: 'var(--border-subtle)' }} />
        <span style={{
            'font-family': 'var(--font-mono)',
            'font-size': '6.5px',
            'font-weight': 700,
            color: 'var(--text-muted)',
            opacity: 0.45,
            'letter-spacing': '0.8px',
            'white-space': 'nowrap',
        }}>
            {props.label}
        </span>
        <div style={{ flex: 1, height: '1px', background: 'var(--border-subtle)' }} />
    </div>
);

// ── CommandRail ───────────────────────────────────────────────────────────────
export const CommandRail: Component = () => {
    const [state, actions] = useApp();
    const navigate = useNavigate();
    const [auditOpen, setAuditOpen] = createSignal(false);
    const [flyoutTop, setFlyoutTop] = createSignal(0);

    const go = (id: NavTab) => {
        actions.setActiveNavTab(id as any);
        const route = routeMap[id as string];
        if (route) navigate(route);
    };

    const auditActive = () => auditItems.some(i => i.id === state.activeNavTab);

    // ── OBSERVE — Monitoring & Detection ──────────────────────────────────
    const observe: PrimaryItem[] = [
        { id: 'dashboard', icon: Icons.Dashboard, label: 'Dash' },
        { id: 'siem',      icon: Icons.SIEM,      label: 'SIEM' },
        { id: 'alerts',    icon: Icons.Alerts,     label: 'Alerts',
          badge: () => state.notifications.filter((n: any) => n.type === 'error').length,
          urgent: true },
        { id: 'topology',  icon: Icons.Topology,   label: 'Net' },
        { id: 'health',    icon: Icons.Health,      label: 'Health' },
    ];

    // ── OPERATE — Hands-on Infrastructure ─────────────────────────────────
    const operate: PrimaryItem[] = [
        { id: 'terminal',  icon: Icons.Terminal,  label: 'Shell' },
        { id: 'hosts',     icon: Icons.Hosts,     label: 'Hosts' },
        { id: 'agents',    icon: Icons.Agents,    label: 'Agents' },
        { id: 'ops',       icon: Icons.Ops,       label: 'Ops' },
        { id: 'soc',       icon: Icons.SOC,       label: 'SOC' },
        { id: 'response',  icon: Icons.Response,  label: 'SOAR' },
    ];

    // ── INTEL — Threat Intelligence & Hunting ─────────────────────────────
    const intel: PrimaryItem[] = [
        { id: 'ueba',           icon: Icons.UEBA,     label: 'UEBA' },
        { id: 'threat-hunter',  icon: Icons.Hunter,   label: 'Hunt' },
        { id: 'ndr',            icon: Icons.NDR,      label: 'NDR' },
        { id: 'purple-team',    icon: Icons.Purple,   label: 'Purple' },
        { id: 'graph',          icon: Icons.Graph,     label: 'Graph' },
    ];

    // ── GOVERN — Compliance, Identity, Security ───────────────────────────
    const govern: PrimaryItem[] = [
        { id: 'compliance',  icon: Icons.Compliance, label: 'Comply' },
        { id: 'vault',       icon: Icons.Vault,      label: 'Vault' },
        { id: 'identity',    icon: Icons.Identity,   label: 'Users' },
        { id: 'security',    icon: Icons.Security,   label: 'Trust' },
        { id: 'forensics',   icon: Icons.Forensics,  label: 'Forensic' },
        { id: 'war-mode',    icon: Icons.War,        label: 'WarMode' },
    ];

    // ── SYSTEM — Configuration & Admin ────────────────────────────────────
    const system: PrimaryItem[] = [
        { id: 'executive', icon: Icons.Executive, label: 'Exec' },
        { id: 'plugins',   icon: Icons.Plugins,   label: 'Plugin' },
        { id: 'settings',  icon: Icons.Settings,  label: 'Config' },
    ];

    return (
        <nav style={{
            width: '56px',
            'min-width': '56px',
            height: '100%',
            background: 'var(--surface-0)',
            'border-right': '1px solid var(--border-primary)',
            display: 'flex',
            'flex-direction': 'column',
            'padding-top': '0',
            'z-index': 1000,
            'overflow-y': 'auto',
            'overflow-x': 'hidden',
        }}>
            {/* Logo Mark */}
            <div style={{
                height: '40px',
                display: 'flex',
                'align-items': 'center',
                'justify-content': 'center',
                'border-bottom': '1px solid var(--border-primary)',
                'flex-shrink': 0,
            }}>
                <svg width="20" height="20" viewBox="0 0 20 20" fill="none">
                    <path d="M10 2L3 6v8l7 4 7-4V6L10 2z" stroke="var(--accent-primary)" stroke-width="1.5" fill="none"/>
                    <path d="M10 6l-4 2.3v4.6L10 15l4-2.1V8.3L10 6z" fill="var(--accent-primary)" opacity="0.25"/>
                    <circle cx="10" cy="10" r="1.5" fill="var(--accent-primary)"/>
                </svg>
            </div>

            {/* OBSERVE */}
            <Divider label="OBS" />
            <For each={observe}>
                {(item) => (
                    <NavButton
                        item={item}
                        active={state.activeNavTab === item.id}
                        onClick={() => go(item.id)}
                    />
                )}
            </For>

            {/* OPERATE */}
            <Divider label="OPS" />
            <For each={operate}>
                {(item) => (
                    <NavButton
                        item={item}
                        active={state.activeNavTab === item.id}
                        onClick={() => go(item.id)}
                    />
                )}
            </For>

            {/* INTEL */}
            <Divider label="INTEL" />
            <For each={intel}>
                {(item) => (
                    <NavButton
                        item={item}
                        active={state.activeNavTab === item.id}
                        onClick={() => go(item.id)}
                    />
                )}
            </For>

            {/* GOVERN */}
            <Divider label="GOV" />
            <For each={govern}>
                {(item) => (
                    <NavButton
                        item={item}
                        active={state.activeNavTab === item.id}
                        onClick={() => go(item.id)}
                    />
                )}
            </For>

            {/* AUDIT — single entry that opens flyout */}
            <button
                title="Audit Trail"
                onMouseEnter={(e) => {
                    setFlyoutTop(e.currentTarget.getBoundingClientRect().top);
                    setAuditOpen(true);
                }}
                style={{
                    background: auditActive() ? 'var(--surface-2)' : 'transparent',
                    border: 'none',
                    'border-left': auditActive() ? '2px solid var(--accent-primary)' : '2px solid transparent',
                    color: auditActive() ? 'var(--text-primary)' : 'var(--text-secondary)',
                    width: '100%',
                    padding: '6px 0',
                    cursor: 'pointer',
                    display: 'flex',
                    'flex-direction': 'column',
                    'align-items': 'center',
                    gap: '2px',
                    position: 'relative',
                }}
            >
                <div style={{ width: '16px', height: '16px' }}>
                    <Icons.Audit />
                </div>
                <span style={{
                    'font-family': 'var(--font-ui)',
                    'font-size': '8px',
                    'font-weight': auditActive() ? 700 : 400,
                    'text-transform': 'uppercase',
                    'letter-spacing': '0.3px',
                    'line-height': 1,
                }}>
                    Audit
                </span>
                <div style={{ position: 'absolute', right: '2px', top: '50%', transform: 'translateY(-50%)', width: '8px', height: '8px', opacity: 0.4 }}>
                    <Icons.ChevronRight />
                </div>
            </button>

            <Show when={auditOpen()}>
                <div style={{ position: 'fixed', left: '64px', top: `${flyoutTop()}px`, 'z-index': 2000 }}>
                    <AuditFlyout
                        active={auditActive()}
                        currentTab={state.activeNavTab as string}
                        onSelect={go}
                        onClose={() => setAuditOpen(false)}
                    />
                </div>
            </Show>

            {/* Push system to bottom */}
            <div style={{ flex: 1 }} />

            {/* SYSTEM */}
            <Divider label="SYS" />
            <For each={system}>
                {(item) => (
                    <NavButton
                        item={item}
                        active={state.activeNavTab === item.id}
                        onClick={() => go(item.id)}
                    />
                )}
            </For>
            <div style={{ height: '4px' }} />
        </nav>
    );
};
