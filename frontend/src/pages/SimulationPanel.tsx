import { Component, createSignal, onMount, For, Show } from 'solid-js';
import * as SimulationService from '../../wailsjs/go/simulation/SimulationService';
import { simulation } from '../../wailsjs/go/models';

const MITRE_TACTICS = [
    'Initial Access', 'Execution', 'Persistence', 'Privilege Escalation',
    'Defense Evasion', 'Credential Access', 'Discovery', 'Lateral Movement',
    'Collection', 'Command and Control', 'Exfiltration', 'Impact'
];

const SimulationPanel: Component = () => {
    const [scenarios, setScenarios] = createSignal<simulation.Scenario[]>([]);
    const [results, setResults] = createSignal<simulation.SimulationResult[]>([]);
    const [campaigns, setCampaigns] = createSignal<any[]>([]);
    const [msg, setMsg] = createSignal('');
    const [activeTab, setActiveTab] = createSignal<'scenarios' | 'matrix' | 'campaigns'>('scenarios');
    const [running, setRunning] = createSignal(false);

    onMount(async () => await refreshData());

    const refreshData = async () => {
        try {
            const [s, r, c] = await Promise.all([
                SimulationService.ListScenarios(),
                SimulationService.GetResults(),
                SimulationService.GetCampaigns()
            ]);
            setScenarios(s);
            setResults(r);
            setCampaigns(c || []);
        } catch (e) {
            console.error(e);
        }
    };

    const runScenario = async (id: string, target: string) => {
        setRunning(true);
        setMsg(`⚡ Launching ${id}...`);
        try {
            await SimulationService.RunScenario(id, target);
            setMsg(`✅ Scenario ${id} executed on ${target}.`);
            setTimeout(() => refreshData(), 2000);
        } catch (e) {
            setMsg(`❌ Error: ${e}`);
        } finally {
            setRunning(false);
        }
    };

    const runAllScenarios = async () => {
        setRunning(true);
        setMsg('🔥 FULL SPECTRUM VALIDATION — Running all scenarios...');
        for (const s of scenarios()) {
            await SimulationService.RunScenario(s.id, 'local-node');
        }
        setMsg('✅ Full spectrum validation complete.');
        setRunning(false);
        setTimeout(() => refreshData(), 3000);
    };

    const startCampaign = async () => {
        const ids = scenarios().map(s => s.id);
        const name = `Campaign_${new Date().toISOString().slice(0, 10)}`;
        try {
            const id = await SimulationService.StartCampaign(name, ids);
            setMsg(`📋 Campaign started: ${id}`);
            await refreshData();
        } catch (e) {
            setMsg(`❌ Campaign error: ${e}`);
        }
    };

    const resilienceScore = () => {
        const r = results();
        if (r.length === 0) return 0;
        return Math.round((r.filter(x => x.detected).length / r.length) * 100);
    };

    const coveredTactics = () => {
        const covered = new Set<string>();
        for (const s of scenarios()) {
            if (s.tactics) s.tactics.forEach(t => covered.add(t));
        }
        return covered;
    };

    return (
        <div class="simulation-page">
            <header class="sim-header">
                <div>
                    <h1>PURPLE TEAM ENGINE</h1>
                    <p>Autonomous MITRE ATT&CK technique replay and detection verification</p>
                </div>
                <div class="header-actions">
                    <button class="btn-action" onClick={refreshData}>REFRESH</button>
                    <button class="btn-danger" onClick={runAllScenarios} disabled={running()}>
                        {running() ? 'RUNNING...' : 'FULL SPECTRUM'}
                    </button>
                </div>
            </header>

            <Show when={msg()}>
                <div class="status-banner">[SIM_ENGINE]: {msg()}</div>
            </Show>

            {/* Tab Bar */}
            <nav class="sim-tabs">
                <button class={activeTab() === 'scenarios' ? 'active' : ''} onClick={() => setActiveTab('scenarios')}>SCENARIOS</button>
                <button class={activeTab() === 'matrix' ? 'active' : ''} onClick={() => setActiveTab('matrix')}>MITRE MATRIX</button>
                <button class={activeTab() === 'campaigns' ? 'active' : ''} onClick={() => setActiveTab('campaigns')}>CAMPAIGNS</button>
            </nav>

            <div class="sim-content">
                {/* Scenarios Tab */}
                <Show when={activeTab() === 'scenarios'}>
                    <div class="scenario-grid">
                        <For each={scenarios()}>
                            {(s) => (
                                <div class="scenario-card">
                                    <div class="card-top">
                                        <span class="mitre-id">{s.mitre_id}</span>
                                        <span class="target-type">{s.target_type}</span>
                                    </div>
                                    <h3>{s.name}</h3>
                                    <p class="desc">{s.description}</p>
                                    <div class="tactics-row">
                                        <For each={s.tactics}>
                                            {(t) => <span class="tactic-tag">{t.toUpperCase()}</span>}
                                        </For>
                                    </div>
                                    <div class="card-actions">
                                        <button class="btn-execute" onClick={() => runScenario(s.id, 'local-node')} disabled={running()}>
                                            EXECUTE
                                        </button>
                                    </div>
                                </div>
                            )}
                        </For>
                    </div>
                </Show>

                {/* MITRE Matrix Tab */}
                <Show when={activeTab() === 'matrix'}>
                    <div class="mitre-matrix">
                        <For each={MITRE_TACTICS}>
                            {(tactic) => {
                                const covered = coveredTactics().has(tactic);
                                return (
                                    <div class={`matrix-cell ${covered ? 'covered' : 'gap'}`}>
                                        <div class="cell-indicator">{covered ? '●' : '○'}</div>
                                        <div class="cell-label">{tactic}</div>
                                        <div class="cell-count">
                                            {scenarios().filter(s => s.tactics?.includes(tactic)).length} techniques
                                        </div>
                                    </div>
                                );
                            }}
                        </For>
                    </div>
                    <div class="coverage-summary">
                        <span>Coverage: {coveredTactics().size}/{MITRE_TACTICS.length} tactics</span>
                        <span class="coverage-pct">{Math.round((coveredTactics().size / MITRE_TACTICS.length) * 100)}%</span>
                    </div>
                </Show>

                {/* Campaigns Tab */}
                <Show when={activeTab() === 'campaigns'}>
                    <div class="campaign-section">
                        <button class="btn-action" onClick={startCampaign}>+ NEW CAMPAIGN (ALL SCENARIOS)</button>
                        <div class="campaign-list">
                            <For each={campaigns()}>
                                {(c) => (
                                    <div class="campaign-card">
                                        <span class="campaign-id">{c.id}</span>
                                        <span class="campaign-name">{c.name}</span>
                                        <span class="campaign-status">{c.status}</span>
                                    </div>
                                )}
                            </For>
                            <Show when={campaigns().length === 0}>
                                <div class="empty-state">No campaigns created yet.</div>
                            </Show>
                        </div>
                    </div>
                </Show>
            </div>

            {/* Resilience Sidebar */}
            <aside class="resilience-panel">
                <h3>PLATFORM RESILIENCE</h3>
                <div class="score-ring">
                    <svg viewBox="0 0 120 120">
                        <circle cx="60" cy="60" r="50" fill="none" stroke="#1e2930" stroke-width="8" />
                        <circle cx="60" cy="60" r="50" fill="none"
                            stroke={resilienceScore() >= 80 ? '#10b981' : resilienceScore() >= 50 ? '#f59e0b' : '#ef4444'}
                            stroke-width="8"
                            stroke-dasharray={`${resilienceScore() * 3.14} 314`}
                            stroke-linecap="round"
                            transform="rotate(-90 60 60)"
                        />
                        <text x="60" y="55" text-anchor="middle" fill="#e5e7eb" font-size="28" font-weight="bold">{resilienceScore()}</text>
                        <text x="60" y="75" text-anchor="middle" fill="#6b7280" font-size="10">SCORE</text>
                    </svg>
                </div>

                <div class="result-feed">
                    <For each={results().slice(0, 10)}>
                        {(r) => (
                            <div class={`result-entry ${r.detected ? 'detected' : 'missed'}`}>
                                <span class="result-id">{r.scenario_id}</span>
                                <span class={`result-status ${r.detected ? 'status-ok' : 'status-fail'}`}>
                                    {r.detected ? 'DETECTED' : 'BYPASSED'}
                                </span>
                            </div>
                        )}
                    </For>
                    <Show when={results().length === 0}>
                        <div class="empty-state">Run scenarios to see detection results.</div>
                    </Show>
                </div>
            </aside>

            <style>{`
                .simulation-page { display: grid; grid-template-columns: 1fr 280px; grid-template-rows: auto auto auto 1fr; gap: 1.5rem; padding: 1.5rem; height: calc(100vh - 60px); overflow: hidden; }
                .sim-header { grid-column: 1 / -1; display: flex; justify-content: space-between; align-items: center; }
                .sim-header h1 { font-size: 1.4rem; letter-spacing: 2px; margin: 0; }
                .sim-header p { color: var(--tactical-gray); font-size: 0.8rem; margin: 0.25rem 0 0; }
                .header-actions { display: flex; gap: 0.75rem; }
                .btn-action { background: var(--tactical-surface); border: 1px solid var(--tactical-border); color: var(--tactical-blue); padding: 0.5rem 1rem; font-size: 0.7rem; font-weight: 800; letter-spacing: 1px; border-radius: 4px; cursor: pointer; }
                .btn-action:hover { background: rgba(59, 130, 246, 0.1); }
                .btn-danger { background: rgba(239, 68, 68, 0.1); border: 1px solid rgba(239, 68, 68, 0.3); color: #ef4444; padding: 0.5rem 1rem; font-size: 0.7rem; font-weight: 800; letter-spacing: 1px; border-radius: 4px; cursor: pointer; }
                .btn-danger:hover { background: rgba(239, 68, 68, 0.2); }
                .btn-danger:disabled { opacity: 0.5; cursor: not-allowed; }

                .status-banner { grid-column: 1 / -1; background: rgba(6, 182, 212, 0.05); border: 1px solid rgba(6, 182, 212, 0.2); padding: 0.75rem; border-radius: 4px; font-family: 'JetBrains Mono', monospace; font-size: 0.7rem; color: #22d3ee; }

                .sim-tabs { grid-column: 1 / 2; display: flex; gap: 0; border-bottom: 1px solid var(--tactical-border); }
                .sim-tabs button { background: none; border: none; border-bottom: 2px solid transparent; color: var(--tactical-gray); padding: 0.5rem 1.25rem; font-size: 0.7rem; font-weight: 800; letter-spacing: 1.5px; cursor: pointer; transition: all 0.2s; }
                .sim-tabs button.active { color: var(--tactical-blue); border-bottom-color: var(--tactical-blue); }
                .sim-tabs button:hover { color: #e5e7eb; }

                .sim-content { grid-column: 1 / 2; overflow: auto; }

                .scenario-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(300px, 1fr)); gap: 1rem; }
                .scenario-card { background: var(--tactical-surface); border: 1px solid var(--tactical-border); padding: 1.25rem; border-radius: 6px; display: flex; flex-direction: column; transition: border-color 0.2s; }
                .scenario-card:hover { border-color: var(--tactical-blue); }
                .card-top { display: flex; justify-content: space-between; margin-bottom: 0.75rem; }
                .mitre-id { font-family: monospace; font-size: 0.65rem; color: #22d3ee; background: rgba(6, 182, 212, 0.1); padding: 2px 6px; border-radius: 3px; }
                .target-type { font-size: 0.6rem; color: var(--tactical-gray); text-transform: uppercase; }
                .scenario-card h3 { font-size: 0.95rem; margin: 0 0 0.5rem; }
                .desc { font-size: 0.75rem; color: var(--tactical-gray); flex: 1; }
                .tactics-row { display: flex; gap: 0.25rem; flex-wrap: wrap; margin-top: 0.75rem; }
                .tactic-tag { font-size: 0.55rem; background: rgba(255,255,255,0.03); border: 1px solid var(--tactical-border); padding: 1px 6px; border-radius: 2px; color: var(--tactical-gray); }
                .card-actions { margin-top: 1rem; padding-top: 0.75rem; border-top: 1px solid var(--tactical-border); }
                .btn-execute { width: 100%; background: rgba(6, 182, 212, 0.08); border: 1px solid rgba(6, 182, 212, 0.25); color: #22d3ee; padding: 0.5rem; font-size: 0.65rem; font-weight: 800; letter-spacing: 2px; border-radius: 4px; cursor: pointer; }
                .btn-execute:hover { background: rgba(6, 182, 212, 0.15); }
                .btn-execute:disabled { opacity: 0.4; cursor: not-allowed; }

                .mitre-matrix { display: grid; grid-template-columns: repeat(4, 1fr); gap: 0.75rem; }
                .matrix-cell { background: var(--tactical-surface); border: 1px solid var(--tactical-border); padding: 1rem; border-radius: 6px; text-align: center; transition: all 0.2s; }
                .matrix-cell.covered { border-color: rgba(16, 185, 129, 0.4); }
                .matrix-cell.gap { border-color: rgba(239, 68, 68, 0.2); opacity: 0.6; }
                .cell-indicator { font-size: 1.5rem; margin-bottom: 0.5rem; }
                .matrix-cell.covered .cell-indicator { color: #10b981; }
                .matrix-cell.gap .cell-indicator { color: #ef4444; }
                .cell-label { font-size: 0.7rem; font-weight: 700; margin-bottom: 0.25rem; }
                .cell-count { font-size: 0.6rem; color: var(--tactical-gray); }
                .coverage-summary { display: flex; justify-content: space-between; margin-top: 1.5rem; padding: 1rem; background: var(--tactical-surface); border: 1px solid var(--tactical-border); border-radius: 6px; font-size: 0.8rem; }
                .coverage-pct { color: var(--tactical-blue); font-weight: 800; font-size: 1.1rem; }

                .campaign-section { display: flex; flex-direction: column; gap: 1rem; }
                .campaign-list { display: flex; flex-direction: column; gap: 0.5rem; }
                .campaign-card { display: flex; justify-content: space-between; align-items: center; background: var(--tactical-surface); border: 1px solid var(--tactical-border); padding: 0.75rem 1rem; border-radius: 4px; font-size: 0.8rem; }
                .campaign-id { font-family: monospace; color: var(--tactical-blue); font-size: 0.7rem; }
                .campaign-name { font-weight: 600; }
                .campaign-status { font-size: 0.65rem; color: var(--tactical-gray); text-transform: uppercase; }

                .resilience-panel { grid-row: 3 / 5; background: var(--tactical-surface); border: 1px solid var(--tactical-border); border-radius: 8px; padding: 1.25rem; display: flex; flex-direction: column; overflow: auto; }
                .resilience-panel h3 { font-size: 0.7rem; letter-spacing: 2px; color: var(--tactical-gray); margin: 0 0 1rem; }
                .score-ring { display: flex; justify-content: center; margin-bottom: 1.5rem; }
                .score-ring svg { width: 120px; height: 120px; }

                .result-feed { display: flex; flex-direction: column; gap: 0.35rem; flex: 1; overflow: auto; }
                .result-entry { display: flex; justify-content: space-between; align-items: center; padding: 0.4rem 0.6rem; border-radius: 3px; font-size: 0.65rem; font-family: monospace; }
                .result-entry.detected { background: rgba(16, 185, 129, 0.05); border-left: 2px solid #10b981; }
                .result-entry.missed { background: rgba(239, 68, 68, 0.05); border-left: 2px solid #ef4444; }
                .result-id { color: #d1d5db; font-weight: 600; }
                .status-ok { color: #10b981; font-weight: 800; }
                .status-fail { color: #ef4444; font-weight: 800; }

                .empty-state { text-align: center; padding: 2rem; color: var(--tactical-gray); font-size: 0.8rem; font-style: italic; }
            `}</style>
        </div>
    );
};

export default SimulationPanel;
