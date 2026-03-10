import { Component, Show, For } from 'solid-js';
import { useApp, AppState } from '@core/store';
import { useNavigate } from '@solidjs/router';

type NavTab = AppState['activeNavTab'] | 'temporal' | 'lineage' | 'decisions' | 'ledger' | 'replay' | 'soc';

type NavGroup = {
    label: string;
    items: {
        id: NavTab;
        icon: () => any;
        label: string;
        badge?: number;
        urgent?: boolean;
    }[];
};

// Tactical SVGs
const Icons = {
    SIEM: () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M2 3h6a4 4 0 014 4v14a3 3 0 00-3-3H2z" /><path d="M22 3h-6a4 4 0 00-4 4v14a3 3 0 013-3h7z" /></svg>,
    Ops: () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><line x1="18" y1="20" x2="18" y2="10" /><line x1="12" y1="20" x2="12" y2="4" /><line x1="6" y1="20" x2="6" y2="14" /></svg>,
    Topology: () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="3" /><circle cx="5" cy="18" r="2" /><circle cx="19" cy="18" r="2" /><circle cx="5" cy="6" r="2" /><circle cx="19" cy="6" r="2" /><path d="M7 16l4-2M17 16l-4-2M7 8l4 2M17 8l-4 2" /></svg>,
    Terminal: () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="2" y="4" width="20" height="16" rx="2" /><polyline points="8 10 11 12 8 14" /><line x1="13" y1="15" x2="16" y2="15" /></svg>,
    Fleet: () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="4" y="4" width="16" height="6" rx="1.5" /><rect x="4" y="14" width="16" height="6" rx="1.5" /><circle cx="7" cy="7" r="1" /><circle cx="7" cy="17" r="1" /></svg>,
    Hosts: () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="4" y="4" width="16" height="6" rx="1.5" /><rect x="4" y="14" width="16" height="6" rx="1.5" /><circle cx="7" cy="7" r="1" /><circle cx="7" cy="17" r="1" /></svg>,
    Compliance: () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M9 11l3 3L22 4" /><path d="M21 12v7a2 2 0 01-2 2H5a2 2 0 01-2-2V5a2 2 0 012-2h11" /></svg>,
    Vault: () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="3" y="11" width="18" height="11" rx="2" /><path d="M7 11V7a5 5 0 0110 0v4" /><circle cx="12" cy="16" r="1" /></svg>,
    Settings: () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="3" /><path d="M19.4 15a1.65 1.65 0 00.33 1.82l.06.06a2 2 0 01-2.83 2.83l-.06-.06a1.65 1.65 0 00-1.82-.33 1.65 1.65 0 00-1 1.51V21a2 2 0 01-4 0v-.09A1.65 1.65 0 009 19.4a1.65 1.65 0 00-1.82.33l-.06.06a2 2 0 01-2.83-2.83l.06-.06A1.65 1.65 0 004.68 15a1.65 1.65 0 00-1.51-1H3a2 2 0 010-4h.09A1.65 1.65 0 004.6 9a1.65 1.65 0 00-.33-1.82l-.06-.06a2 2 0 012.83-2.83l.06.06A1.65 1.65 0 009 4.68a1.65 1.65 0 001-1.51V3a2 2 0 014 0v.09a1.65 1.65 0 001 1.51 1.65 1.65 0 001.82-.33l.06-.06a2 2 0 012.83 2.83l-.06.06A1.65 1.65 0 0019.4 9a1.65 1.65 0 001.51 1H21a2 2 0 010 4h-.09a1.65 1.65 0 00-1.51 1z" /></svg>,
    Dashboard: () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="3" y="3" width="7" height="9" /><rect x="14" y="3" width="7" height="5" /><rect x="14" y="12" width="7" height="9" /><rect x="3" y="16" width="7" height="5" /></svg>,
    Tunnels: () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="10" /><line x1="2" y1="12" x2="22" y2="12" /><path d="M12 2a15.3 15.3 0 0 1 4 10 15.3 15.3 0 0 1-4 10 15.3 15.3 0 0 1-4-10 15.3 15.3 0 0 1 4-10z" /></svg>,
    Snippets: () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><polyline points="16 18 22 12 16 6" /><polyline points="8 6 2 12 8 18" /></svg>,
    Recordings: () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><circle cx="12" cy="12" r="10" /><circle cx="12" cy="12" r="3" /></svg>,
    Notes: () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M14 2H6a2 2 0 0 0-2 2v16a2 2 0 0 0 2 2h12a2 2 0 0 0 2-2V8z" /><polyline points="14 2 14 8 20 8" /><line x1="16" y1="13" x2="8" y2="13" /><line x1="16" y1="17" x2="8" y2="17" /><polyline points="10 9 9 9 8 9" /></svg>,
    Team: () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M17 21v-2a4 4 0 0 0-4-4H5a4 4 0 0 0-4 4v2" /><circle cx="9" cy="7" r="4" /><path d="M23 21v-2a4 4 0 0 0-3-3.87" /><path d="M16 3.13a4 4 0 0 1 0 7.75" /></svg>,
    Sync: () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><polyline points="23 4 23 10 17 10" /><polyline points="1 20 1 14 7 14" /><path d="M3.51 9a9 9 0 0 1 14.85-3.36L23 10M1 14l4.64 4.36A9 9 0 0 0 20.49 15" /></svg>,
    Plugins: () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M21 16V8a2 2 0 0 0-1-1.73l-7-4a2 2 0 0 0-2 0l-7 4A2 2 0 0 0 3 8v8a2 2 0 0 0 1 1.73l7 4a2 2 0 0 0 2 0l7-4A2 2 0 0 0 21 16z" /><polyline points="3.27 6.96 12 12.01 20.73 6.96" /><line x1="12" y1="22.08" x2="12" y2="12" /></svg>,
    Health: () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M22 12h-4l-3 9L9 3l-3 9H2" /></svg>,
    Metrics: () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M21.21 15.89A10 10 0 1 1 8 2.83" /><path d="M22 12A10 10 0 0 0 12 2v10z" /></svg>,
    Updater: () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M21 15v4a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-4" /><polyline points="7 10 12 15 17 10" /><line x1="12" y1="15" x2="12" y2="3" /></svg>,
    Workspace: () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="2" y="3" width="20" height="14" rx="2" ry="2" /><line x1="8" y1="21" x2="16" y2="21" /><line x1="12" y1="17" x2="12" y2="21" /></svg>,
    Alerts: () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M18 8A6 6 0 0 0 6 8c0 7-3 9-3 9h18s-3-2-3-9" /><path d="M13.73 21a2 2 0 0 1-3.46 0" /></svg>,
    Security: () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10z" /></svg>,
    SOC: () => <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.5"><rect x="3" y="3" width="18" height="18" rx="2" /><path d="M3 9h18" /><path d="M9 21V9" /></svg>,
};

export const CommandRail: Component = () => {
    const [state, actions] = useApp();
    const navigate = useNavigate();

    // Maps nav IDs to their actual router paths (from index.tsx).
    // Items NOT in this map are "drawer-only" — they open a sidebar panel
    // without changing the main content area.
    const routeMap: Record<string, string> = {
        dashboard: '/dashboard',
        siem: '/siem',
        topology: '/topology',
        terminal: '/terminal',
        hosts: '/hosts',
        ops: '/ops',
        compliance: '/compliance',
        vault: '/vault',
        team: '/team',
        plugins: '/plugins',
        workspace: '/workspace',
        security: '/trust',
        health: '/monitoring',
        metrics: '/executive',
        settings: '/workspace',
        temporal: '/temporal-integrity',
        lineage: '/lineage',
        decisions: '/decisions',
        ledger: '/ledger',
        replay: '/response-replay',
        soc: '/soc',
    };

    const groups: NavGroup[] = [
        {
            label: "OBSERVE",
            items: [
                { id: 'dashboard', icon: Icons.Dashboard, label: "Dash" },
                { id: 'siem', icon: Icons.SIEM, label: "SIEM" },
                { id: 'alerts', icon: Icons.Alerts, label: "Alerts", badge: state.notifications.filter(n => n.type === 'error').length, urgent: true },
                { id: 'topology', icon: Icons.Topology, label: "Topology" },
                { id: 'health', icon: Icons.Health, label: "Health" },
                { id: 'metrics', icon: Icons.Metrics, label: "Metrics" }
            ]
        },
        {
            label: "OPERATE",
            items: [
                { id: 'terminal', icon: Icons.Terminal, label: "Terminal" },
                { id: 'hosts', icon: Icons.Hosts, label: "Hosts" },
                { id: 'tunnels', icon: Icons.Tunnels, label: "Tunnels" },
                { id: 'ops', icon: Icons.Ops, label: "Ops" },
                { id: 'soc', icon: Icons.SOC, label: "Elite SOC" },
                { id: 'snippets', icon: Icons.Snippets, label: "Snippets" }
            ]
        },
        {
            label: "RECORD",
            items: [
                { id: 'recordings', icon: Icons.Recordings, label: "Replay" },
                { id: 'notes', icon: Icons.Notes, label: "Notes" },
                { id: 'workspace', icon: Icons.Workspace, label: "Work" }
            ]
        },
        {
            label: "GOVERN",
            items: [
                { id: 'security', icon: Icons.Security, label: "Secure" },
                { id: 'compliance', icon: Icons.Compliance, label: "Comply" },
                { id: 'vault', icon: Icons.Vault, label: "Vault" },
                { id: 'team', icon: Icons.Team, label: "Team" },
                { id: 'temporal', icon: Icons.Metrics, label: "Time" },
                { id: 'lineage', icon: Icons.Sync, label: "Lineage" },
                { id: 'decisions', icon: Icons.Topology, label: "Decisions" },
                { id: 'ledger', icon: Icons.Recordings, label: "Ledger" },
                { id: 'replay', icon: Icons.Snippets, label: "Replay" }
            ]
        },
        {
            label: "SYSTEM",
            items: [
                { id: 'plugins', icon: Icons.Plugins, label: "Plugins" },
                { id: 'sync', icon: Icons.Sync, label: "Sync" },
                { id: 'updater', icon: Icons.Updater, label: "Update" },
                { id: 'settings', icon: Icons.Settings, label: "Settings" }
            ]
        }
    ];

    return (
        <nav class="command-rail" style={{
            width: '80px', /* slightly wider for brutalism */
            'background-color': 'var(--surface-1)',
            'border-right': '2px solid var(--border-primary)',
            display: 'flex',
            'flex-direction': 'column',
            'padding-top': '16px',
            'z-index': 1000,
            overflow: 'auto',
        }}>
            <div style={{
                'font-family': 'var(--font-mono)',
                'font-weight': 800,
                color: 'var(--text-primary)',
                'font-size': '16px',
                'text-align': 'center',
                'margin-bottom': '24px',
                'letter-spacing': '2px'
            }}>
                OBLIVRA
            </div>

            <For each={groups}>
                {(group) => (
                    <div style={{ 'margin-bottom': '24px' }}>
                        <div style={{
                            'font-family': 'var(--font-mono)',
                            'font-size': '10px',
                            'font-weight': 700,
                            color: 'var(--text-muted)',
                            'padding-left': '8px',
                            'margin-bottom': '8px',
                            'letter-spacing': '1px'
                        }}>
                            {group.label}
                        </div>
                        <div style={{ display: 'flex', 'flex-direction': 'column', gap: '4px' }}>
                            <For each={group.items}>
                                {(item) => (
                                    <button
                                        onClick={() => {
                                            actions.setActiveNavTab(item.id as any);
                                            const route = routeMap[item.id as string];
                                            if (route) navigate(route);
                                        }}
                                        style={{
                                            background: state.activeNavTab === item.id ? 'var(--surface-2)' : 'transparent',
                                            border: 'none',
                                            'border-left': state.activeNavTab === item.id ? '3px solid var(--accent-primary)' : '3px solid transparent',
                                            color: state.activeNavTab === item.id ? 'var(--text-primary)' : 'var(--text-secondary)',
                                            padding: '8px',
                                            cursor: 'pointer',
                                            display: 'flex',
                                            'flex-direction': 'column',
                                            'align-items': 'center',
                                            gap: '4px',
                                            transition: 'none',
                                            'border-radius': '0',
                                            position: 'relative'
                                        }}
                                        title={item.label}
                                    >
                                        <div style={{ height: '20px', width: '20px', stroke: "currentColor", fill: "none", "stroke-width": state.activeNavTab === item.id ? "2.5" : "1.5" }}>
                                            <item.icon />
                                        </div>
                                        <span style={{
                                            'font-family': 'var(--font-ui)',
                                            'font-size': '10px',
                                            'font-weight': state.activeNavTab === item.id ? 600 : 400
                                        }}>
                                            {item.label}
                                        </span>

                                        <Show when={item.badge !== undefined && item.badge > 0}>
                                            <div style={{
                                                position: 'absolute',
                                                top: '4px',
                                                right: '8px',
                                                background: item.urgent ? 'var(--alert-critical)' : 'var(--accent-primary)',
                                                color: item.urgent ? '#fff' : 'var(--surface-0)',
                                                'font-family': 'var(--font-mono)',
                                                'font-size': '9px',
                                                'font-weight': 800,
                                                padding: '2px 4px',
                                                'line-height': 1
                                            }}>
                                                {item.badge}
                                            </div>
                                        </Show>
                                    </button>
                                )}
                            </For>
                        </div>
                    </div>
                )}
            </For>
        </nav>
    );
};
