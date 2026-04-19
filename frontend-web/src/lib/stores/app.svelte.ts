/**
 * OBLIVRA — App Store (Svelte 5 runes)
 * Minimal implementation for frontend-web stabilization.
 */
import { push } from '../router.svelte';

export interface PaletteAction {
    id: string;
    label: string;
    description?: string;
    icon?: string;
    shortcut?: string;
    action: () => void;
}

export type NavTab = 'dashboard' | 'siem' | 'alerts' | 'terminal' | 'hosts' | 'intel' | 'health' | 'settings' | 'vault';

export interface Notification {
    id: string;
    message: string;
    type: 'info' | 'success' | 'warning' | 'error';
    details?: string;
    duration: number;
}

export interface SystemHealth {
    status: 'healthy' | 'degraded' | 'critical';
    message?: string;
}

class AppStore {
    // -- State --
    hosts = $state<any[]>([]);
    notifications = $state<Notification[]>([]);
    systemHealth = $state<SystemHealth>({ status: 'healthy' });
    activeNavTab = $state<NavTab>('dashboard');
    showCommandPalette = $state(false);

    // -- Actions --
    notify(message: string, type: Notification['type'] = 'info', details?: string, duration = 5000) {
        const id = Math.random().toString(36).substring(2, 9);
        const n: Notification = { id, message, type, details, duration };
        this.notifications = [...this.notifications, n];
        if (duration > 0) {
            setTimeout(() => this.dismissNotification(id), duration);
        }
    }

    dismissNotification(id: string) {
        this.notifications = this.notifications.filter(n => n.id !== id);
    }

    setActiveNavTab(tab: NavTab) {
        this.activeNavTab = tab;
    }

    toggleCommandPalette() {
        this.showCommandPalette = !this.showCommandPalette;
    }

    setHealth(status: SystemHealth['status'], message?: string) {
        this.systemHealth = { status, message };
    }

    connectToHost(hostId: string) {
        // Mock for web version
        this.notify(`Connecting to ${hostId} (Terminal simulation)`, 'info');
        this.setActiveNavTab('terminal');
        push('/terminal');
    }
}

export const appStore = new AppStore();
