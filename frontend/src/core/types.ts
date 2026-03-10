import { database } from '../../wailsjs/go/models';

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
    id: string; // jobID from backend
    name: string;
    type: 'upload' | 'download';
    status: 'pending' | 'active' | 'completed' | 'failed' | 'cancelled';
    progress: number; // 0-100
    size: number;
    speed?: string;
    error?: string;
}
