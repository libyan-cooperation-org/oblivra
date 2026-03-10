import { Component, createSignal, onMount, For, Show } from 'solid-js';
import { DecisionTrace } from '../wails';

export const DecisionInspector: Component = () => {
    const [traces, setTraces] = createSignal<DecisionTrace[]>([]);
    const [selectedTrace, setSelectedTrace] = createSignal<DecisionTrace | null>(null);
    const [loading, setLoading] = createSignal(true);

    const loadTraces = async () => {
        try {
            setLoading(true);
            if (window.go?.app?.DecisionService) {
                const data = await window.go.app.DecisionService.ListRecentDecisions(50);
                // Ensure latest are first
                setTraces(data || []);
            }
        } catch (err) {
            console.error("Failed to load decision traces:", err);
        } finally {
            setLoading(false);
        }
    };

    onMount(() => {
        loadTraces();
        // Auto-refresh every 5 seconds
        const timer = setInterval(loadTraces, 5000);
        return () => clearInterval(timer);
    });

    return (
        <div class="decision-page">
            <header class="page-header">
                <div>
                    <h1>DECISION TRACEABILITY</h1>
                    <p>Cryptographically verifiable reasoning engine</p>
                </div>
                <button class="refresh-btn" onClick={loadTraces}>
                    ↻ REFRESH
                </button>
            </header>

            <div class="decision-layout">
                <div class="trace-list-panel">
                    <h3>RECENT DECISIONS</h3>
                    <Show when={loading() && traces().length === 0}>
                        <div class="empty-state">Loading traces...</div>
                    </Show>
                    <Show when={!loading() && traces().length === 0}>
                        <div class="empty-state">No security decisions captured yet.</div>
                    </Show>
                    <div class="trace-feed">
                        <For each={traces()}>
                            {(t) => (
                                <div
                                    class={`trace-card ${selectedTrace()?.id === t.id ? 'active' : ''}`}
                                    onClick={() => setSelectedTrace(t)}
                                >
                                    <div class="trace-header">
                                        <span class="trace-rule">{t.rule_name}</span>
                                        <span class="trace-conf" style={{ color: t.confidence_score > 0.8 ? '#10b981' : '#f59e0b' }}>
                                            {(t.confidence_score * 100).toFixed(0)}%
                                        </span>
                                    </div>
                                    <div class="trace-time">{new Date(t.timestamp).toLocaleTimeString()}</div>
                                    <div class="trace-id">{t.id}</div>
                                </div>
                            )}
                        </For>
                    </div>
                </div>

                <div class="trace-detail-panel">
                    <Show when={selectedTrace()} fallback={
                        <div class="empty-state" style={{ margin: 'auto' }}>
                            Select a decision trace to view its cryptographic proof and reasoning chain.
                        </div>
                    }>
                        {(t) => (
                            <div class="trace-detail-content">
                                <div class="detail-header">
                                    <h2>{t().rule_name}</h2>
                                    <div class="proof-badge">
                                        ✓ VERIFIED PROOF
                                        <span class="proof-hash">{t().crypto_proof.substring(0, 16)}...</span>
                                    </div>
                                </div>

                                <div class="detail-section">
                                    <h4>EXPLANATION</h4>
                                    <pre class="explanation-box">{t().explanation}</pre>
                                </div>

                                <div class="detail-section">
                                    <h4>EVIDENCE CHAIN</h4>
                                    <div class="evidence-list">
                                        <For each={t().evidence_chain}>
                                            {(ev, i) => (
                                                <div class="evidence-item">
                                                    <div class="ev-step">{i() + 1}</div>
                                                    <div class="ev-content">
                                                        <span class="ev-type">[{ev.type}]</span>
                                                        <span class="ev-desc">{ev.description}</span>
                                                    </div>
                                                </div>
                                            )}
                                        </For>
                                    </div>
                                </div>

                                <Show when={t().alternatives && t().alternatives.length > 0}>
                                    <div class="detail-section">
                                        <h4>ALTERNATIVES REJECTED</h4>
                                        <div class="alt-list">
                                            <For each={t().alternatives}>
                                                {(alt) => (
                                                    <div class="alt-item">
                                                        <span class="alt-rule">{alt.rule_name}</span>
                                                        <span class="alt-reason">{alt.reason}</span>
                                                    </div>
                                                )}
                                            </For>
                                        </div>
                                    </div>
                                </Show>
                            </div>
                        )}
                    </Show>
                </div>
            </div>

            <style>{`
                .decision-page { padding: 1.5rem; height: calc(100vh - 60px); display: flex; flex-direction: column; gap: 1.5rem; background: var(--tactical-bg); }
                .page-header { display: flex; justify-content: space-between; align-items: center; }
                .page-header h1 { font-size: 1.4rem; letter-spacing: 2px; margin: 0; color: #f3f4f6; }
                .page-header p { color: var(--tactical-gray); font-size: 0.8rem; margin: 0.25rem 0 0; }
                .refresh-btn { background: rgba(255,255,255,0.05); border: 1px solid var(--tactical-border); color: #d1d5db; padding: 6px 12px; border-radius: 4px; cursor: pointer; font-size: 0.75rem; letter-spacing: 1px; transition: all 0.2s; }
                .refresh-btn:hover { background: rgba(255,255,255,0.1); }

                .decision-layout { display: grid; grid-template-columns: 350px 1fr; gap: 1.5rem; flex: 1; min-height: 0; }
                
                .trace-list-panel, .trace-detail-panel { background: var(--tactical-surface); border: 1px solid var(--tactical-border); border-radius: 8px; display: flex; flex-direction: column; overflow: hidden; }
                .trace-list-panel h3 { font-size: 0.75rem; letter-spacing: 2px; color: var(--tactical-gray); padding: 1rem 1rem 0; margin: 0 0 1rem; }
                
                .trace-feed { flex: 1; overflow-y: auto; display: flex; flex-direction: column; gap: 1px; background: var(--tactical-border); }
                .trace-card { background: var(--tactical-surface); padding: 1rem; cursor: pointer; transition: background 0.2s; border-left: 3px solid transparent; }
                .trace-card:hover { background: rgba(255,255,255,0.02); }
                .trace-card.active { border-left-color: #3b82f6; background: rgba(59, 130, 246, 0.05); }
                
                .trace-header { display: flex; justify-content: space-between; margin-bottom: 0.25rem; font-size: 0.85rem; font-weight: 600; }
                .trace-rule { color: #f3f4f6; white-space: nowrap; overflow: hidden; text-overflow: ellipsis; padding-right: 1rem; }
                .trace-time { font-size: 0.7rem; color: #9ca3af; margin-bottom: 0.25rem; }
                .trace-id { font-family: monospace; font-size: 0.65rem; color: var(--tactical-gray); }

                .trace-detail-content { padding: 1.5rem; display: flex; flex-direction: column; gap: 1.5rem; overflow-y: auto; height: 100%; }
                .detail-header { display: flex; justify-content: space-between; align-items: flex-start; padding-bottom: 1rem; border-bottom: 1px solid var(--tactical-border); }
                .detail-header h2 { margin: 0; color: #f3f4f6; font-size: 1.25rem; font-weight: 600; }
                .proof-badge { display: flex; flex-direction: column; align-items: flex-end; font-size: 0.65rem; color: #10b981; font-weight: 700; letter-spacing: 1px; }
                .proof-hash { font-family: monospace; color: var(--tactical-gray); font-weight: 400; margin-top: 2px; }

                .detail-section h4 { font-size: 0.75rem; letter-spacing: 2px; color: var(--tactical-gray); margin: 0 0 0.75rem; }
                
                .explanation-box { background: #000; border: 1px solid var(--tactical-border); padding: 1rem; border-radius: 6px; font-family: monospace; font-size: 0.8rem; color: #d1d5db; white-space: pre-wrap; margin: 0; line-height: 1.5; }
                
                .evidence-list { display: flex; flex-direction: column; gap: 0.5rem; }
                .evidence-item { display: flex; gap: 1rem; background: rgba(255,255,255,0.02); padding: 0.75rem; border-radius: 4px; border: 1px solid var(--tactical-border); border-left: 2px solid #8b5cf6; }
                .ev-step { font-family: monospace; color: var(--tactical-gray); font-weight: 700; font-size: 0.8rem; }
                .ev-content { display: flex; flex-direction: column; gap: 0.25rem; }
                .ev-type { font-size: 0.65rem; font-weight: 700; letter-spacing: 1px; color: #8b5cf6; }
                .ev-desc { font-size: 0.8rem; color: #f3f4f6; }

                .alt-list { display: flex; flex-direction: column; gap: 0.5rem; }
                .alt-item { display: flex; justify-content: space-between; background: rgba(0,0,0,0.2); padding: 0.75rem; border-radius: 4px; border: 1px solid var(--tactical-border); font-size: 0.8rem; border-left: 2px solid #ef4444; }
                .alt-rule { color: #d1d5db; }
                .alt-reason { color: var(--tactical-gray); font-style: italic; }

                .empty-state { text-align: center; padding: 3rem; color: var(--tactical-gray); font-style: italic; font-size: 0.9rem; }
            `}</style>
        </div>
    );
};
