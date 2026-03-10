import { createSignal, For, Show } from 'solid-js';

export interface AlertNotification {
    id: string;
    title: string;
    message: string;
    severity: 'critical' | 'warning' | 'info';
    timestamp: Date;
    dismissed: boolean;
    source: string;
}

const [alerts, setAlerts] = createSignal<AlertNotification[]>([]);
const [soundEnabled, setSoundEnabled] = createSignal(true);
const [desktopEnabled, setDesktopEnabled] = createSignal(true);

let alertCounter = 0;

// Audio context for notification sounds
let audioCtx: AudioContext | null = null;

const playAlertSound = (severity: string) => {
    if (!soundEnabled()) return;
    try {
        if (!audioCtx) audioCtx = new AudioContext();
        const osc = audioCtx.createOscillator();
        const gain = audioCtx.createGain();
        osc.connect(gain);
        gain.connect(audioCtx.destination);

        // Different tones for severity
        osc.frequency.value = severity === 'critical' ? 880 : severity === 'warning' ? 660 : 440;
        osc.type = severity === 'critical' ? 'square' : 'sine';
        gain.gain.value = 0.1;
        gain.gain.exponentialRampToValueAtTime(0.001, audioCtx.currentTime + 0.5);

        osc.start();
        osc.stop(audioCtx.currentTime + 0.3);
    } catch { /* Audio not available */ }
};

const sendDesktopNotification = (title: string, body: string) => {
    if (!desktopEnabled()) return;
    if ('Notification' in window && Notification.permission === 'granted') {
        new Notification(title, { body, icon: '🛡' });
    } else if ('Notification' in window && Notification.permission !== 'denied') {
        Notification.requestPermission();
    }
};

export const pushAlert = (title: string, message: string, severity: AlertNotification['severity'], source = 'system') => {
    const alert: AlertNotification = {
        id: `alert-${++alertCounter}`,
        title, message, severity, source,
        timestamp: new Date(),
        dismissed: false,
    };
    setAlerts(prev => [alert, ...prev].slice(0, 50)); // Keep last 50
    playAlertSound(severity);
    sendDesktopNotification(`[${severity.toUpperCase()}] ${title}`, message);
};

export const dismissAlert = (id: string) => {
    setAlerts(prev => prev.map(a => a.id === id ? { ...a, dismissed: true } : a));
};

export const clearAllAlerts = () => setAlerts([]);
export const getAlerts = alerts;
export const toggleSound = () => setSoundEnabled(prev => !prev);
export const toggleDesktop = () => setDesktopEnabled(prev => !prev);
export const isSoundEnabled = soundEnabled;
export const isDesktopEnabled = desktopEnabled;

// Alert Toast UI Component
export const AlertToastContainer = () => {
    const visible = () => alerts().filter(a => !a.dismissed).slice(0, 5);

    return (
        <div class="alert-toast-container">
            <For each={visible()}>
                {(alert) => (
                    <div class={`alert-toast alert-toast-${alert.severity}`} onClick={() => dismissAlert(alert.id)}>
                        <div class="alert-toast-icon">
                            {alert.severity === 'critical' ? '🔴' : alert.severity === 'warning' ? '🟡' : '🔵'}
                        </div>
                        <div class="alert-toast-content">
                            <div class="alert-toast-title">{alert.title}</div>
                            <div class="alert-toast-message">{alert.message}</div>
                            <div class="alert-toast-meta">{alert.source} · {alert.timestamp.toLocaleTimeString()}</div>
                        </div>
                        <button class="alert-toast-close" onClick={(e) => { e.stopPropagation(); dismissAlert(alert.id); }}>×</button>
                    </div>
                )}
            </For>
        </div>
    );
};

// Alert Bell Icon with badge (for nav bar)
export const AlertBell = (props: { onClick?: () => void }) => {
    const unread = () => alerts().filter(a => !a.dismissed).length;

    return (
        <button class="alert-bell" onClick={props.onClick} title={`${unread()} alerts`}>
            🔔
            <Show when={unread() > 0}>
                <span class="alert-bell-badge">{unread() > 9 ? '9+' : unread()}</span>
            </Show>
        </button>
    );
};
