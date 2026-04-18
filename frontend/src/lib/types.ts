import * as database from '@wailsjs/github.com/kingknull/oblivrashell/internal/database';

export type Host = database.Host;

export interface Session {
    id: string;
    hostId: string;
    hostLabel: string;
    status: 'active' | 'closed' | 'error';
    isRecording?: boolean;
    startedAt: string;
    durationSecs?: number;
}

export interface PluginPanel {
    plugin_id: string;
    panel_id: string;
    label: string;
    icon: string;
}

export interface PluginStatusIcon {
    plugin_id: string;
    icon_id: string;
    icon: string;
    tooltip: string;
}

export interface Notification {
    id: string;
    type: 'info' | 'success' | 'warning' | 'error';
    message: string;
    details?: string;
    duration?: number; // ms, 0 for persistent
}

export interface Transfer {
    id: string;
    name: string;
    type: 'upload' | 'download';
    status: 'pending' | 'active' | 'completed' | 'failed' | 'cancelled';
    progress: number;
    size: number;
    speed?: string;
    error?: string;
}

// NavTab must include EVERY id used in CommandRail so setActiveNavTab highlights correctly.
export type NavTab =
    // General
    | 'dashboard' | 'analytics' | 'executive' | 'health' | 'monitoring'
    // SIEM & Alerts
    | 'siem' | 'siem-search' | 'alerts' | 'alert-management'
    // Terminal & SSH
    | 'terminal' | 'ssh' | 'tunnels' | 'recordings' | 'session-playback'
    // Operations
    | 'ops' | 'snippets' | 'notes' | 'tasks' | 'ai-assistant'
    // Fleet & Agents
    | 'hosts' | 'agents' | 'fleet-management' | 'soc' | 'agent-console'
    // Security
    | 'vault' | 'security' | 'trust' | 'runtime-trust'
    | 'forensics' | 'remote-forensics' | 'terminal-forensics'
    | 'ransomware' | 'ransomware-ui' | 'war-mode' | 'data-destruction'
    | 'response' | 'escalation' | 'playbook-builder' | 'purple-team'
    | 'simulation' | 'cases'
    // Intel & Detection
    | 'threat-intel' | 'threat-intel-dashboard' | 'threat-hunter'
    | 'threat-map' | 'threat-graph' | 'graph'
    | 'ueba' | 'ueba-overview' | 'ndr' | 'ndr-overview'
    | 'enrichment' | 'credentials'
    // Topology
    | 'topology' | 'network-map' | 'global-topology' | 'mitre-heatmap'
    // Fusion
    | 'fusion'
    // Governance & Compliance
    | 'compliance' | 'governance' | 'identity' | 'identity-admin'
    | 'lineage' | 'decisions' | 'ledger' | 'chain-of-custody'
    | 'temporal' | 'temporal-integrity' | 'replay' | 'response-replay'
    | 'evidence' | 'oql' | 'soar'
    // System
    | 'plugins' | 'settings' | 'team' | 'sync'
    | 'offline-update' | 'license' | 'features' | 'risk'
    | 'entity' | 'workspace';

export type Workspace = 'Personal' | 'Work' | 'Team';

export interface SystemHealth {
    status: 'healthy' | 'degraded' | 'critical' | 'unknown';
    message?: string;
    last_updated?: string;
}
