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
                .simulation-page { display: grid; grid-template-columns: 1fr 280px; grid-template-rows: auto auto auto 1fr; gap: 16px; padding: 20px; height: calc(100vh - 64px); overflow: hidden; background: var(--surface-0); font-family: var(--font-ui); }
                .sim-header { grid-column: 1 / -1; display: flex; justify-content: space-between; align-items: center; }
                .sim-header h1 { font-size: 16px; font-weight: 700; letter-spacing: 0.5px; margin: 0; color: var(--text-heading); }
                .sim-header p { color: var(--text-muted); font-size: 11px; margin: 3px 0 0; }
                .header-actions { display: flex; gap: 8px; }
                .btn-action { background: var(--surface-2); border: 1px solid var(--border-secondary); color: var(--accent-primary); padding: 7px 14px; font-size: 11px; font-weight: 700; letter-spacing: 0.5px; border-radius: var(--radius-sm); cursor: pointer; font-family: var(--font-ui); transition: all var(--transition-fast); }
                .btn-action:hover { background: var(--surface-3); border-color: var(--accent-primary); }
                .btn-danger { background: rgba(224,64,64,0.1); border: 1px solid rgba(224,64,64,0.3); color: var(--alert-critical); padding: 7px 14px; font-size: 11px; font-weight: 700; letter-spacing: 0.5px; border-radius: var(--radius-sm); cursor: pointer; font-family: var(--font-ui); transition: all var(--transition-fast); }
                .btn-danger:hover { background: rgba(224,64,64,0.18); }
                .btn-danger:disabled { opacity: 0.45; cursor: not-allowed; }

                .status-banner { grid-column: 1 / -1; background: rgba(0,153,224,0.06); border: 1px solid rgba(0,153,224,0.2); border-left: 3px solid var(--accent-primary); padding: 10px 14px; border-radius: var(--radius-sm); font-family: var(--font-mono); font-size: 11px; color: var(--accent-primary); }

                .sim-tabs { grid-column: 1 / 2; display: flex; gap: 0; border-bottom: 1px solid var(--border-primary); }
                .sim-tabs button { background: none; border: none; border-bottom: 2px solid transparent; color: var(--text-muted); padding: 8px 18px; font-size: 11px; font-weight: 700; letter-spacing: 1px; cursor: pointer; font-family: var(--font-ui); text-transform: uppercase; transition: color var(--transition-fast); }
                .sim-tabs button.active { color: var(--accent-primary); border-bottom-color: var(--accent-cta); }
                .sim-tabs button:hover { color: var(--text-primary); }

                .sim-content { grid-column: 1 / 2; overflow: auto; padding-top: 4px; }

                .scenario-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(280px, 1fr)); gap: 12px; }
                .scenario-card { background: var(--surface-1); border: 1px solid var(--border-primary); padding: 16px; border-radius: var(--radius-md); display: flex; flex-direction: column; transition: border-color var(--transition-fast); }
                .scenario-card:hover { border-color: var(--border-secondary); }
                .card-top { display: flex; justify-content: space-between; margin-bottom: 10px; }
                .mitre-id { font-family: var(--font-mono); font-size: 10px; color: var(--accent-primary); background: rgba(0,153,224,0.1); padding: 2px 7px; border-radius: var(--radius-xs); border: 1px solid rgba(0,153,224,0.2); }
                .target-type { font-size: 10px; color: var(--text-muted); text-transform: uppercase; }
                .scenario-card h3 { font-size: 13px; margin: 0 0 6px; color: var(--text-heading); font-weight: 600; }
                .desc { font-size: 11px; color: var(--text-muted); flex: 1; line-height: 1.5; }
                .tactics-row { display: flex; gap: 4px; flex-wrap: wrap; margin-top: 10px; }
                .tactic-tag { font-size: 9px; background: var(--surface-3); border: 1px solid var(--border-primary); padding: 1px 6px; border-radius: var(--radius-xs); color: var(--text-muted); font-family: var(--font-ui); }
                .card-actions { margin-top: 12px; padding-top: 10px; border-top: 1px solid var(--border-primary); }
                .btn-execute { width: 100%; background: rgba(245,139,0,0.1); border: 1px solid rgba(245,139,0,0.3); color: var(--accent-cta); padding: 7px; font-size: 10px; font-weight: 700; letter-spacing: 1.5px; border-radius: var(--radius-sm); cursor: pointer; font-family: var(--font-ui); text-transform: uppercase; transition: all var(--transition-fast); }
                .btn-execute:hover { background: rgba(245,139,0,0.18); }
                .btn-execute:disabled { opacity: 0.4; cursor: not-allowed; }

                .mitre-matrix { display: grid; grid-template-columns: repeat(4, 1fr); gap: 10px; }
                .matrix-cell { background: var(--surface-1); border: 1px solid var(--border-primary); padding: 14px; border-radius: var(--radius-md); text-align: center; transition: all var(--transition-fast); }
                .matrix-cell.covered { border-color: rgba(92,192,92,0.4); background: rgba(92,192,92,0.04); }
                .matrix-cell.gap { border-color: rgba(224,64,64,0.2); opacity: 0.6; }
                .cell-indicator { font-size: 20px; margin-bottom: 6px; }
                .matrix-cell.covered .cell-indicator { color: var(--status-online); }
                .matrix-cell.gap .cell-indicator { color: var(--alert-critical); }
                .cell-label { font-size: 11px; font-weight: 600; margin-bottom: 3px; color: var(--text-primary); }
                .cell-count { font-size: 10px; color: var(--text-muted); }
                .coverage-summary { display: flex; justify-content: space-between; align-items: center; margin-top: 16px; padding: 12px 16px; background: var(--surface-1); border: 1px solid var(--border-primary); border-radius: var(--radius-md); font-size: 12px; color: var(--text-secondary); }
                .coverage-pct { color: var(--accent-primary); font-weight: 700; font-size: 18px; font-family: var(--font-mono); }

                .campaign-section { display: flex; flex-direction: column; gap: 12px; }
                .campaign-list { display: flex; flex-direction: column; gap: 6px; }
                .campaign-card { display: flex; justify-content: space-between; align-items: center; background: var(--surface-1); border: 1px solid var(--border-primary); padding: 10px 14px; border-radius: var(--radius-sm); font-size: 12px; }
                .campaign-id { font-family: var(--font-mono); color: var(--accent-primary); font-size: 10px; }
                .campaign-name { font-weight: 600; color: var(--text-primary); }
                .campaign-status { font-size: 10px; color: var(--text-muted); text-transform: uppercase; }

                .resilience-panel { grid-row: 3 / 5; background: var(--surface-1); border: 1px solid var(--border-primary); border-radius: var(--radius-md); padding: 16px; display: flex; flex-direction: column; overflow: auto; }
                .resilience-panel h3 { font-size: 10px; letter-spacing: 1.5px; color: var(--text-muted); margin: 0 0 14px; text-transform: uppercase; font-weight: 700; }
                .score-ring { display: flex; justify-content: center; margin-bottom: 16px; }
                .score-ring svg { width: 120px; height: 120px; }

                .result-feed { display: flex; flex-direction: column; gap: 4px; flex: 1; overflow: auto; }
                .result-entry { display: flex; justify-content: space-between; align-items: center; padding: 5px 8px; border-radius: var(--radius-xs); font-size: 10px; font-family: var(--font-mono); }
                .result-entry.detected { background: rgba(92,192,92,0.06); border-left: 2px solid var(--status-online); }
                .result-entry.missed { background: rgba(224,64,64,0.06); border-left: 2px solid var(--alert-critical); }
                .result-id { color: var(--text-secondary); font-weight: 600; }
                .status-ok { color: var(--status-online); font-weight: 700; }
                .status-fail { color: var(--alert-critical); font-weight: 700; }

                .empty-state { text-align: center; padding: 28px; color: var(--text-muted); font-size: 12px; font-family: var(--font-ui); }
            `}</style>
        </div>
    );
};

export default SimulationPanel;
