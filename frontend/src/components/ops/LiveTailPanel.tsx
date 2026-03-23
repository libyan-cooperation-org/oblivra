import { Component, createSignal, For, onCleanup, onMount, Show } from 'solid-js';
import { IS_BROWSER } from '@core/context';
import { subscribe } from '@core/bridge';


interface TailLog {
    sessionID: string;
    hostLabel: string;
    data: string;
    timestamp: string;
}

export const LiveTailPanel: Component = () => {
    const [logs, setLogs] = createSignal<TailLog[]>([]);
    const [sessions, setSessions] = createSignal<any[]>([]);
    const [activeTails, setActiveTails] = createSignal<Set<string>>(new Set());

    let unsubTail: (() => void) | undefined;

    onMount(async () => {
        if (IS_BROWSER) return;
        const { GetActiveSessions } = await import('../../../wailsjs/go/services/SSHService');
        const sess = await GetActiveSessions();
        setSessions(sess || []);

        unsubTail = subscribe('tail:update', (data: any) => {
            const decoded = atob(data.data);
            setLogs(prev => [{
                sessionID: data.session_id,
                hostLabel: data.host_label,
                data: decoded,
                timestamp: data.timestamp
            }, ...prev.slice(0, 499)]);
        });
    });

    onCleanup(async () => {
        unsubTail?.();
        if (!IS_BROWSER) {
            const { StopTailing } = await import('../../../wailsjs/go/services/TailingService');
            activeTails().forEach(id => StopTailing(id));
        }
    });

    const toggleTail = async (sessionID: string) => {
        if (IS_BROWSER) return;
        const { StartTailing, StopTailing } = await import('../../../wailsjs/go/services/TailingService');
        const next = new Set(activeTails());
        if (next.has(sessionID)) {
            await StopTailing(sessionID);
            next.delete(sessionID);
        } else {
            await StartTailing(sessionID);
            next.add(sessionID);
        }
        setActiveTails(next);
    };

    return (
        <div class="live-tail-panel">
            <div class="tail-sidebar">
                <h4 style="font-family: var(--font-ui); font-size: 10px; opacity: 0.6; margin-bottom: 8px;">ACTIVE SESSIONS</h4>
                <For each={sessions()}>
                    {(s) => (
                        <div class={`tail-session-item ${activeTails().has(s.id) ? 'active' : ''}`} onClick={() => toggleTail(s.id)}>
                            <div class="session-dot" style={`background: ${activeTails().has(s.id) ? 'var(--accent-primary)' : 'transparent'}; border: 1px solid var(--border-primary);`}></div>
                            <span class="session-label">{s.hostLabel}</span>
                        </div>
                    )}
                </For>
                <Show when={sessions().length === 0}>
                    <p style="font-size: 10px; color: var(--text-muted);">No active SSH sessions found.</p>
                </Show>
            </div>

            <div class="tail-stream">
                <For each={logs()}>
                    {(log) => (
                        <div class="tail-line">
                            <span class="tail-time">[{new Date(log.timestamp).toLocaleTimeString()}]</span>
                            <span class="tail-host">{log.hostLabel}</span>
                            <span class="tail-data">{log.data}</span>
                        </div>
                    )}
                </For>
                <Show when={logs().length === 0}>
                    <div class="tail-empty">
                        <p>Aggregated log stream is empty.</p>
                        <p style="font-size: 11px; opacity: 0.5;">Select active sessions on the left to start tailing.</p>
                    </div>
                </Show>
            </div>
        </div>
    );
};
