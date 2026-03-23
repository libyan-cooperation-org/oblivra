import { Component, createSignal, onMount, For, Show } from 'solid-js';
import { subscribe } from '@core/bridge';
import { IS_BROWSER } from '@core/context';
import { ueba } from '../../../wailsjs/go/models';

export const UEBAPanel: Component = () => {
    const [profiles, setProfiles] = createSignal<ueba.EntityProfile[]>([]);
    const [selectedProfile, setSelectedProfile] = createSignal<ueba.EntityProfile | null>(null);
    const [anomalies, setAnomalies] = createSignal<any[]>([]);

    onMount(async () => {
        if (IS_BROWSER) return;
        const { GetProfiles } = await import('../../../wailsjs/go/services/UEBAService');
        setProfiles(await GetProfiles() || []);

        subscribe('siem:anomaly', (data: any) => {
            setAnomalies(prev => [data, ...prev].slice(0, 50));
            import('../../../wailsjs/go/services/UEBAService')
                .then(m => m.GetProfiles()).then(p => setProfiles(p || []));
        });

        const interval = setInterval(async () => {
            const { GetProfiles: gp } = await import('../../../wailsjs/go/services/UEBAService');
            setProfiles(await gp() || []);
        }, 10000);

        return () => clearInterval(interval);
    });

    const getRiskColor = (score: number) => {
        if (score > 0.8) return 'var(--tactical-red)';
        if (score > 0.5) return 'var(--tactical-amber)';
        return 'var(--tactical-green)';
    };

    return (
        <div class="siem-panel">
            <header class="siem-header">
                <div class="siem-title">
                    <span class="siem-icon">🧠</span>
                    <h2>UEBA & BEHAVIORAL ANALYTICS</h2>
                </div>
            </header>

            <div class="siem-grid" style={{ "grid-template-columns": "minmax(300px, 350px) 1fr", "gap": "0" }}>
                {/* Entity List */}
                <div class="siem-sidebar">
                    <div class="siem-sidebar-header">
                        <h3>ENTITIES</h3>
                        <span class="count-badge">{profiles().length}</span>
                    </div>
                    <div class="entity-list custom-scrollbar">
                        <For each={profiles().sort((a, b) => b.risk_score - a.risk_score)}>
                            {(profile) => (
                                <div
                                    class="entity-item group"
                                    classList={{ active: selectedProfile()?.id === profile.id }}
                                    onClick={() => setSelectedProfile(profile)}
                                >
                                    <div class="entity-info">
                                        <div class={`status-dot ${profile.risk_score > 0.5 ? 'pulse' : ''}`} style={{ background: getRiskColor(profile.risk_score) }}></div>
                                        <div class="entity-meta">
                                            <span class="entity-id">{profile.id}</span>
                                            <span class="entity-type">{profile.type.toUpperCase()}</span>
                                        </div>
                                    </div>
                                    <div class="risk-indicator" style={{
                                        color: getRiskColor(profile.risk_score),
                                        border: `1px solid ${getRiskColor(profile.risk_score)}22`,
                                        background: `${getRiskColor(profile.risk_score)}11`
                                    }}>
                                        {(profile.risk_score * 100).toFixed(0)}%
                                    </div>
                                </div>
                            )}
                        </For>
                    </div>
                </div>

                {/* Detail View */}
                <div class="siem-content">
                    <Show when={selectedProfile()} fallback={
                        <div class="empty-state">
                            <span class="empty-icon">📂</span>
                            <p>Select an entity to view behavioral details</p>
                        </div>
                    }>
                        <div class="profile-detail">
                            <div class="detail-header">
                                <div class="detail-title">
                                    <h3>{selectedProfile()?.id}</h3>
                                    <span class="entity-subtitle">{selectedProfile()?.type?.toUpperCase()}</span>
                                </div>
                                <div class="risk-badge" style={{ "border-color": getRiskColor(selectedProfile()!.risk_score) }}>
                                    <span class="label">RISK SCORE:</span>
                                    <span class="value">{(selectedProfile()!.risk_score * 100).toFixed(1)}%</span>
                                </div>
                            </div>

                            <div class="detail-section">
                                <header class="section-header">
                                    <h4>BEHAVIORAL FEATURES</h4>
                                    <div class="header-line"></div>
                                </header>
                                <div class="feature-grid">
                                    <For each={Object.entries(selectedProfile()!.features || {})}>
                                        {([name, fallback]) => (
                                            <div class="feature-card">
                                                <div class="card-bg"></div>
                                                <span class="feature-name">{name.replace(/_/g, ' ')}</span>
                                                <span class="feature-value">{fallback.toFixed(3)}</span>
                                                <div class="feature-bar">
                                                    <div class="fill" style={{ width: `${Math.min(fallback * 10, 100)}%` }}></div>
                                                </div>
                                            </div>
                                        )}
                                    </For>
                                </div>
                            </div>

                            <div class="detail-section">
                                <header class="section-header">
                                    <h4>ANOMALY HISTORY</h4>
                                    <div class="header-line"></div>
                                </header>
                                <div class="anomaly-history">
                                    <For each={anomalies().filter(a => a.entity_id === selectedProfile()?.id)}>
                                        {(anomaly) => (
                                            <div class="anomaly-event-card">
                                                <div class="anomaly-meta">
                                                    <span class="anomaly-type">BEHAVIOR_ANOMALY</span>
                                                    <span class="anomaly-time">{new Date().toLocaleTimeString()}</span>
                                                </div>
                                                <div class="evidence-list">
                                                    <For each={anomaly.evidence || []}>
                                                        {(ev: any) => (
                                                            <div class="evidence-item">
                                                                <span class="ev-key">{ev.key}:</span>
                                                                <span class="ev-val">{typeof ev.value === 'number' ? ev.value.toFixed(3) : ev.value}</span>
                                                                <Show when={ev.threshold}>
                                                                    <span class="ev-threshold">(limit: {ev.threshold})</span>
                                                                </Show>
                                                                <div class="ev-desc">{ev.description}</div>
                                                            </div>
                                                        )}
                                                    </For>
                                                </div>
                                                <button class="fp-button" onClick={() => { if (!IS_BROWSER) import('../../../wailsjs/go/services/GovernanceService').then(m => m.MarkFalsePositive(anomaly.entity_id, 'User observation', anomaly.evidence)); }}>
                                                    MARK FALSE POSITIVE
                                                </button>
                                            </div>
                                        )}
                                    </For>
                                </div>
                            </div>
                        </div>
                    </Show>
                </div>
            </div>

            <style>{`
                .siem-grid { height: calc(100vh - 64px); }
                .siem-sidebar { border-right: 1px solid var(--tactical-border); background: rgba(0,0,0,0.2); }
                .entity-list { height: 100%; overflow-y: auto; }
                .entity-item {
                    padding: 1.25rem 1.5rem;
                    border-bottom: 1px solid rgba(255,255,255,0.03);
                    display: flex;
                    justify-content: space-between;
                    align-items: center;
                    cursor: pointer;
                    transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
                    position: relative;
                }
                .entity-item:hover { background: rgba(255,255,255,0.02); padding-left: 1.75rem; }
                .entity-item.active { background: rgba(59, 130, 246, 0.05); }
                .entity-item.active::after {
                    content: '';
                    position: absolute;
                    left: 0; top: 0; bottom: 0;
                    width: 3px;
                    background: var(--tactical-blue);
                    box-shadow: 0 0 15px var(--tactical-blue);
                }
                .entity-info { display: flex; align-items: center; gap: 1rem; }
                .status-dot { width: 6px; height: 6px; border-radius: 50%; }
                .status-dot.pulse { animation: status-pulse 2s infinite; }
                @keyframes status-pulse {
                    0% { transform: scale(1); opacity: 1; box-shadow: 0 0 0 0 currentColor; }
                    70% { transform: scale(1.5); opacity: 0; box-shadow: 0 0 0 6px currentColor; }
                    100% { transform: scale(1); opacity: 0; }
                }
                .entity-meta { display: flex; flex-direction: column; }
                .entity-id { font-family: 'JetBrains Mono', monospace; font-size: 0.85rem; color: #fff; font-weight: 600; }
                .entity-type { font-size: 0.65rem; color: var(--tactical-gray); letter-spacing: 1px; }
                .risk-indicator { 
                    font-size: 0.7rem; 
                    font-weight: 800; 
                    padding: 3px 8px; 
                    border-radius: 6px;
                    font-family: 'JetBrains Mono', monospace;
                }
                .profile-detail { padding: 2rem; max-width: 1200px; margin: 0 auto; }
                .section-header { display: flex; align-items: center; gap: 1.5rem; margin-bottom: 1.5rem; }
                .section-header h4 { font-size: 0.75rem; font-weight: 900; color: var(--tactical-gray); letter-spacing: 2px; flex-shrink: 0; }
                .header-line { height: 1px; flex-grow: 1; background: linear-gradient(to right, var(--tactical-border), transparent); }
                .feature-grid { 
                    display: grid; 
                    grid-template-columns: repeat(auto-fill, minmax(220px, 1fr)); 
                    gap: 1.25rem; 
                }
                .feature-card {
                    background: rgba(255,255,255,0.02);
                    border: 1px solid rgba(255,255,255,0.05);
                    padding: 1.25rem;
                    display: flex;
                    flex-direction: column;
                    position: relative;
                    overflow: hidden;
                    border-radius: 12px;
                }
                .feature-name { font-size: 0.65rem; color: var(--tactical-gray); text-transform: uppercase; margin-bottom: 0.5rem; z-index: 1; }
                .feature-value { font-family: 'JetBrains Mono', monospace; font-size: 1.4rem; color: var(--tactical-blue); font-weight: 700; z-index: 1; }
                .feature-bar { height: 2px; width: 100%; background: rgba(0,0,0,0.3); margin-top: 1rem; border-radius: 1px; }
                .feature-bar .fill { height: 100%; background: var(--tactical-blue); box-shadow: 0 0 10px var(--tactical-blue); }
                
                .anomaly-history { display: flex; flex-direction: column; gap: 1rem; }
                .anomaly-event-card {
                    background: rgba(255,50,50,0.05);
                    border: 1px solid rgba(255,50,50,0.2);
                    padding: 1rem;
                    border-radius: 8px;
                }
                .anomaly-meta { display: flex; justify-content: space-between; margin-bottom: 1rem; }
                .anomaly-type { font-weight: 800; color: var(--tactical-red); font-size: 0.65rem; letter-spacing: 1px; }
                .anomaly-time { font-size: 0.65rem; color: var(--tactical-gray); }
                
                .evidence-list { display: flex; flex-direction: column; gap: 0.5rem; margin-bottom: 1rem; }
                .evidence-item { font-size: 0.75rem; display: flex; flex-wrap: wrap; align-items: center; gap: 0.5rem; }
                .ev-key { color: var(--tactical-gray); font-weight: 700; }
                .ev-val { font-family: 'JetBrains Mono', monospace; font-weight: 700; color: #fff; }
                .ev-threshold { color: var(--tactical-red); opacity: 0.7; }
                .ev-desc { width: 100%; font-size: 0.6rem; color: var(--tactical-gray); font-style: italic; }
                
                .fp-button {
                    background: transparent;
                    border: 1px solid var(--tactical-gray);
                    color: var(--tactical-gray);
                    font-size: 0.6rem;
                    font-weight: 800;
                    padding: 4px 12px;
                    border-radius: 4px;
                    cursor: pointer;
                    transition: all 0.2s;
                }
                .fp-button:hover {
                    background: var(--tactical-red);
                    border-color: var(--tactical-red);
                    color: white;
                }

                .stats-row { display: flex; gap: 4rem; margin-top: 1rem; }
                .stat-item { display: flex; flex-direction: column; gap: 0.25rem; }
                .stat-item .label { font-size: 0.65rem; font-weight: 800; color: var(--tactical-gray); letter-spacing: 1px; }
                .stat-item .value { font-family: 'JetBrains Mono', monospace; font-size: 1.1rem; color: #fff; }
                .custom-scrollbar::-webkit-scrollbar { width: 4px; }
                .custom-scrollbar::-webkit-scrollbar-track { background: transparent; }
                .custom-scrollbar::-webkit-scrollbar-thumb { background: rgba(255,255,255,0.1); border-radius: 10px; }
            `}</style>
        </div>
    );
};
