import { Component, createSignal, For, Show, onMount, onCleanup } from 'solid-js';
import { Violation } from '../wails';

export const TemporalIntegrity: Component = () => {
    const [violations, setViolations] = createSignal<Violation[]>([]);
    const [agentDrift, setAgentDrift] = createSignal<Record<string, number>>({});

    const loadData = async () => {
        try {
            if (window.go?.app?.TemporalService) {
                const [vios, drift] = await Promise.all([
                    window.go.app.TemporalService.GetViolations(),
                    window.go.app.TemporalService.GetAgentDrift()
                ]);
                setViolations(vios || []);
                setAgentDrift(drift || {});
            }
        } catch (err) {
            console.error("Failed to load temporal data:", err);
        }
    };

    onMount(() => {
        loadData();
        const timer = setInterval(loadData, 3000);
        onCleanup(() => clearInterval(timer));
    });

    const getTypeColor = (t: string) => {
        switch (t) {
            case 'clock_drift': return '#f59e0b';
            case 'future_event': return '#ef4444';
            case 'stale_event': return '#8b5cf6';
            default: return '#6b7280';
        }
    };

    const getDriftColor = (ms: number) => {
        const abs = Math.abs(ms);
        if (abs > 5000) return '#ef4444';
        if (abs > 2000) return '#f59e0b';
        return '#10b981';
    };

    return (
        <div class="temporal-page">
            <header class="temporal-header">
                <div>
                    <h1>TEMPORAL INTEGRITY</h1>
                    <p>Event timestamp validation and fleet clock drift monitoring</p>
                </div>
                <div class="policy-badge">
                    <span>Future Skew: 1h</span>
                    <span>Max Age: 30d</span>
                    <span>Drift Alert: 5000ms</span>
                </div>
            </header>

            <div class="temporal-grid">
                {/* Clock Drift Monitor */}
                <section class="drift-panel">
                    <h3>FLEET CLOCK DRIFT</h3>
                    <div class="drift-list">
                        <For each={Object.entries(agentDrift())}>
                            {([host, drift]) => (
                                <div class="drift-entry">
                                    <span class="drift-host">{host}</span>
                                    <div class="drift-bar-container">
                                        <div class="drift-bar" style={{
                                            width: `${Math.min(Math.abs(drift) / 100, 100)}%`,
                                            background: getDriftColor(drift),
                                            'margin-left': drift < 0 ? 'auto' : '0',
                                        }}></div>
                                    </div>
                                    <span class="drift-value" style={{ color: getDriftColor(drift) }}>
                                        {drift > 0 ? '+' : ''}{drift}ms
                                    </span>
                                </div>
                            )}
                        </For>
                    </div>
                </section>

                {/* Violation Feed */}
                <section class="violation-panel">
                    <h3>TEMPORAL VIOLATIONS</h3>
                    <div class="violation-list">
                        <For each={violations()}>
                            {(v) => (
                                <div class="violation-entry" style={{ 'border-left-color': getTypeColor(v.type) }}>
                                    <div class="violation-top">
                                        <span class="violation-type" style={{ color: getTypeColor(v.type) }}>{v.type.toUpperCase().replace('_', ' ')}</span>
                                        <span class="violation-time">{new Date(v.timestamp).toLocaleTimeString()}</span>
                                    </div>
                                    <div class="violation-host">Host: {v.host_id}</div>
                                    <div class="violation-detail">{v.detail}</div>
                                </div>
                            )}
                        </For>
                        <Show when={violations().length === 0}>
                            <div class="empty-state">No temporal violations detected.</div>
                        </Show>
                    </div>
                </section>
            </div>

            <style>{`
                .temporal-page { padding: 1.5rem; height: calc(100vh - 60px); display: flex; flex-direction: column; gap: 1.5rem; overflow: hidden; }
                .temporal-header { display: flex; justify-content: space-between; align-items: center; }
                .temporal-header h1 { font-size: 1.4rem; letter-spacing: 2px; margin: 0; }
                .temporal-header p { color: var(--tactical-gray); font-size: 0.8rem; margin: 0.25rem 0 0; }
                .policy-badge { display: flex; gap: 1rem; font-size: 0.65rem; color: var(--tactical-gray); }
                .policy-badge span { background: var(--tactical-surface); border: 1px solid var(--tactical-border); padding: 4px 8px; border-radius: 3px; }

                .temporal-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 1.5rem; flex: 1; overflow: hidden; }

                .drift-panel, .violation-panel { background: var(--tactical-surface); border: 1px solid var(--tactical-border); border-radius: 8px; padding: 1.25rem; display: flex; flex-direction: column; overflow: hidden; }
                .drift-panel h3, .violation-panel h3 { font-size: 0.7rem; letter-spacing: 2px; color: var(--tactical-gray); margin: 0 0 1rem; }

                .drift-list { display: flex; flex-direction: column; gap: 0.75rem; overflow: auto; }
                .drift-entry { display: grid; grid-template-columns: 120px 1fr 80px; align-items: center; gap: 0.75rem; }
                .drift-host { font-family: monospace; font-size: 0.75rem; color: #d1d5db; }
                .drift-bar-container { height: 6px; background: rgba(255, 255, 255, 0.05); border-radius: 3px; overflow: hidden; }
                .drift-bar { height: 100%; border-radius: 3px; transition: width 0.5s; min-width: 2px; }
                .drift-value { font-family: monospace; font-size: 0.7rem; text-align: right; font-weight: 700; }

                .violation-list { display: flex; flex-direction: column; gap: 0.5rem; overflow: auto; flex: 1; }
                .violation-entry { border-left: 3px solid #6b7280; padding: 0.75rem; background: rgba(0, 0, 0, 0.2); border-radius: 0 4px 4px 0; }
                .violation-top { display: flex; justify-content: space-between; margin-bottom: 0.25rem; }
                .violation-type { font-size: 0.65rem; font-weight: 800; letter-spacing: 1px; }
                .violation-time { font-size: 0.6rem; color: var(--tactical-gray); font-family: monospace; }
                .violation-host { font-size: 0.7rem; color: #9ca3af; margin-bottom: 0.25rem; }
                .violation-detail { font-size: 0.7rem; color: #d1d5db; }
                .empty-state { text-align: center; padding: 2rem; color: var(--tactical-gray); font-style: italic; }
            `}</style>
        </div>
    );
};
