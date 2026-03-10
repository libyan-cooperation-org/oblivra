import { Component, createMemo, createSignal, onMount, onCleanup, Show, For } from 'solid-js';
import { useApp } from '@core/store';
import { GetAllHealth } from '../../../wailsjs/go/app/HealthService';

export const StatusBar: Component<{ onToggleTransfers?: () => void }> = (props) => {
    const [state] = useApp();
    const activeCount = () => state.sessions.filter((s) => s.status === 'active').length;
    const transferCount = () => state.transfers.filter(t => t.status === 'active' || t.status === 'pending').length;
    const [time, setTime] = createSignal('');
    const [healthMap, setHealthMap] = createSignal<Record<string, unknown>>({});

    const activeHost = createMemo(() => {
        if (!state.activeSessionId) return null;
        const session = state.sessions.find(s => s.id === state.activeSessionId);
        if (!session) return null;
        return {
            id: session.hostId,
            label: session.hostLabel || session.hostId
        };
    });

    const activeLatency = createMemo(() => {
        const host = activeHost();
        if (!host || !(healthMap()[host.id] as any)?.latency_ms) return null;
        return (healthMap()[host.id] as any).latency_ms;
    });

    let timer: ReturnType<typeof setInterval>;

    onMount(() => {
        const updateTimeAndHealth = async () => {
            setTime(new Date().toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' }));
            try {
                const h = await GetAllHealth();
                setHealthMap(h || {});
            } catch { /* ignore in statusbar */ }
        };
        updateTimeAndHealth();
        timer = setInterval(updateTimeAndHealth, 30_000);
    });

    onCleanup(() => clearInterval(timer));

    return (
        <footer class="status-bar">
            <div class="status-left">
                <div class="status-indicator">
                    <span class={`dot ${activeHost() ? 'online' : 'disconnected'}`} />
                    <span style="letter-spacing: 0.02em;">{activeHost() ? activeHost()?.label : 'Standby'}</span>
                    <Show when={activeLatency() !== null}>
                        <span style="opacity: 0.4; margin-left: 8px; font-family: var(--font-mono); font-size: 9px;">{activeLatency()}ms</span>
                    </Show>
                </div>
            </div>

            <div class="status-right">
                <For each={state.pluginStatusIcons}>
                    {(icon: any) => (
                        <span
                            class="plugin-status-icon"
                            title={`${icon.plugin_id}: ${icon.tooltip}`}
                            style="cursor: help;"
                        >
                            {icon.icon}
                        </span>
                    )}
                </For>

                <div class="status-pill">
                    <span style="opacity: 0.5; margin-right: 4px;">WORKSPACE</span>
                    <span>{state.workspace}</span>
                </div>

                <div class="status-group">
                    <span>{activeCount()} SESSIONS</span>
                    <span style="opacity: 0.3;">/</span>
                    <span>{state.hosts.length} HOSTS</span>
                </div>

                <Show when={transferCount() > 0 || state.transfers.length > 0}>
                    <div
                        class={`status-pill clickable ${transferCount() > 0 ? 'active' : ''}`}
                        onClick={props.onToggleTransfers}
                    >
                        <span class="pulse-icon">{transferCount() > 0 ? '⟳' : '⚐'}</span>
                        <span>{transferCount()} TRANSFERS</span>
                    </div>
                </Show>

                <Show when={state.vaultUnlocked}>
                    <span style="color: var(--success); font-size: 10px;">SECURE</span>
                </Show>

                <span style="font-weight: 600; min-width: 50px; text-align: right;">{time()}</span>
            </div>
        </footer>
    );
};
