import { Component, createMemo, createSignal, onMount, onCleanup, Show, For } from 'solid-js';
import { useApp } from '@core/store';
import { GetAllHealth } from '../../../wailsjs/go/services/HealthService';
import { EventsOn, EventsOff } from '../../../wailsjs/runtime/runtime';
import { usePanelManager } from './PanelManager';

export const StatusBar: Component<{ onToggleTransfers?: () => void }> = (props) => {
    const [state] = useApp();
    const activeCount = () => state.sessions.filter((s) => s.status === 'active').length;
    const transferCount = () => state.transfers.filter(t => t.status === 'active' || t.status === 'pending').length;
    const [time, setTime] = createSignal('');
    const [healthMap, setHealthMap] = createSignal<Record<string, unknown>>({});
    const [diagGrade, setDiagGrade] = createSignal<string | null>(null);
    const { openPanelCount } = usePanelManager();

    // Subscribe to diagnostics broadcast for live health grade in status bar
    onMount(() => {
        EventsOn('diagnostics:snapshot', (data: any) => {
            if (data?.health_grade) setDiagGrade(data.health_grade);
        });
    });
    onCleanup(() => EventsOff('diagnostics:snapshot'));

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

    const sep = () => <span style="color: var(--border-secondary); font-family: var(--font-mono);">|</span>;

    return (
        <footer style="
            display: flex;
            align-items: center;
            justify-content: space-between;
            height: 24px;
            background: var(--surface-1);
            border-top: 1px solid var(--border-primary);
            padding: 0 12px;
            font-size: 10px;
            font-family: var(--font-mono);
            color: var(--text-muted);
            font-weight: 500;
            letter-spacing: 0.3px;
            flex-shrink: 0;
            z-index: 50;
        ">
            {/* Left cluster */}
            <div style="display: flex; align-items: center; gap: 10px;">
                <span style={`
                    display: inline-block;
                    width: 5px;
                    height: 5px;
                    border-radius: 50%;
                    background: ${activeHost() ? 'var(--status-online)' : 'var(--text-muted)'};
                    box-shadow: ${activeHost() ? '0 0 6px var(--status-online)' : 'none'};
                    flex-shrink: 0;
                `} />
                <span style={`color: ${activeHost() ? 'var(--text-secondary)' : 'var(--text-muted)'}`}>
                    {activeHost() ? activeHost()?.label : 'no active session'}
                </span>
                <Show when={activeLatency() !== null}>
                    {sep()}
                    <span style="color: var(--text-muted);">{activeLatency()}ms</span>
                </Show>
            </div>

            {/* Right cluster */}
            <div style="display: flex; align-items: center; gap: 10px;">
                <For each={state.pluginStatusIcons}>
                    {(icon: any) => (
                        <span title={`${icon.plugin_id}: ${icon.tooltip}`} style="cursor: help; opacity: 0.7;">{icon.icon}</span>
                    )}
                </For>

                <Show when={transferCount() > 0}>
                    <span
                        style="color: var(--accent-primary); cursor: pointer;"
                        onClick={props.onToggleTransfers}
                    >⇅ {transferCount()} xfer</span>
                    {sep()}
                </Show>

                <span>
                    <span style="color: var(--text-muted);">{activeCount()}</span>
                    <span style="opacity: 0.35;"> sess · </span>
                    <span style="color: var(--text-muted);">{state.hosts.length}</span>
                    <span style="opacity: 0.35;"> hosts</span>
                </span>

                <Show when={openPanelCount() > 0}>
                    {sep()}
                    <span style="color: var(--accent-primary); font-weight: 700;">{openPanelCount()} panel{openPanelCount() === 1 ? '' : 's'}</span>
                </Show>

                {sep()}

                <Show when={diagGrade()}>
                    <span
                        title="Platform health grade — click SelfMonitor for details"
                        style={{
                            'font-family': 'var(--font-mono)',
                            'font-size': '9px',
                            'font-weight': '800',
                            'letter-spacing': '0.5px',
                            color: diagGrade() === 'A' ? 'var(--status-online)'
                                 : diagGrade() === 'B' ? '#d29922'
                                 : diagGrade() === 'C' ? '#f0883e'
                                 : '#f85149',
                        }}
                    >
                        ● {diagGrade()}
                    </span>
                    {sep()}
                </Show>

                <Show
                    when={state.vaultUnlocked}
                    fallback={<span style="color: var(--text-muted); opacity: 0.4;">locked</span>}
                >
                    <span style="color: var(--accent-primary);">&#9679; secure</span>
                </Show>

                {sep()}

                <span style="color: var(--text-secondary); font-weight: 600; letter-spacing: 0.5px;">{time()}</span>
            </div>
        </footer>
    );
};
