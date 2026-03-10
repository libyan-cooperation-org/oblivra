import { Component, createSignal, onMount, For, Show } from 'solid-js';
import * as SimulationService from '../../wailsjs/go/simulation/SimulationService';
import '../styles/purple-team.css';

export const PurpleTeam: Component = () => {
    const [report, setReport] = createSignal<any>(null);
    const [coverage, setCoverage] = createSignal<any>(null);
    const [history, setHistory] = createSignal<any[]>([]);
    const [activeTab, setActiveTab] = createSignal<'overview' | 'matrix' | 'history' | 'execute'>('overview');
    const [validating, setValidating] = createSignal(false);
    const [msg, setMsg] = createSignal('');
    const [expandedTactic, setExpandedTactic] = createSignal<string | null>(null);

    onMount(async () => await refreshData());

    const refreshData = async () => {
        try {
            const [r, c, h] = await Promise.all([
                SimulationService.GetPurpleTeamReport(),
                SimulationService.GetCoverageReport(),
                SimulationService.GetValidationHistory()
            ]);
            setReport(r);
            setCoverage(c);
            setHistory(h || []);
        } catch (e) {
            console.error('Purple team data fetch failed:', e);
        }
    };

    const runValidation = async () => {
        setValidating(true);
        setMsg('⚡ CONTINUOUS VALIDATION — executing all scenario vectors...');
        try {
            await SimulationService.RunContinuousValidation();
            setMsg('✅ Validation cycle complete. Refreshing metrics...');
            await refreshData();
            setMsg('');
        } catch (e) {
            setMsg(`❌ Validation failed: ${e}`);
        } finally {
            setValidating(false);
        }
    };

    const gradeColor = (grade: string) => {
        switch (grade) {
            case 'A': return '#10b981';
            case 'B': return '#22d3ee';
            case 'C': return '#f59e0b';
            case 'D': return '#f97316';
            default: return '#ef4444';
        }
    };

    const coverageColor = (pct: number) => {
        if (pct >= 66) return '#10b981';
        if (pct >= 33) return '#f59e0b';
        return '#ef4444';
    };

    return (
        <div class="purple-page">
            <header class="purple-header">
                <div>
                    <h1>PURPLE TEAM ENGINE</h1>
                    <p>Detection coverage scoring · Continuous validation · Resilience grading</p>
                </div>
                <div class="header-actions">
                    <button class="ob-btn ob-btn-ghost" onClick={refreshData}>REFRESH</button>
                    <button
                        class="ob-btn ob-btn-danger"
                        onClick={runValidation}
                        disabled={validating()}
                    >
                        {validating() ? '◉ VALIDATING...' : '▶ RUN VALIDATION'}
                    </button>
                </div>
            </header>

            <Show when={msg()}>
                <div class="purple-status-banner">[PURPLE_ENGINE]: {msg()}</div>
            </Show>

            {/* KPI Strip */}
            <div class="purple-kpi-strip">
                <div class="kpi-card kpi-grade">
                    <span class="kpi-label">RESILIENCE</span>
                    <span
                        class="kpi-value grade"
                        style={{ color: gradeColor(report()?.resilience_grade || 'F') }}
                    >
                        {report()?.resilience_grade || '—'}
                    </span>
                    <span class="kpi-sub">{(report()?.resilience_score ?? 0).toFixed(1)} / 100</span>
                </div>
                <div class="kpi-card">
                    <span class="kpi-label">DETECTION RATE</span>
                    <span class="kpi-value">{(report()?.detection_rate ?? 0).toFixed(0)}%</span>
                    <span class="kpi-sub">of executed scenarios</span>
                </div>
                <div class="kpi-card">
                    <span class="kpi-label">COVERAGE INDEX</span>
                    <span class="kpi-value">{(report()?.coverage_index ?? 0).toFixed(1)}%</span>
                    <span class="kpi-sub">{coverage()?.covered_techniques ?? 0}/{coverage()?.total_techniques ?? 0} techniques</span>
                </div>
                <div class="kpi-card">
                    <span class="kpi-label">MEAN RESPONSE</span>
                    <span class="kpi-value">{report()?.mean_response_ms ?? '—'}<span class="kpi-unit">ms</span></span>
                    <span class="kpi-sub">avg per scenario</span>
                </div>
                <div class="kpi-card">
                    <span class="kpi-label">VALIDATION RUNS</span>
                    <span class="kpi-value">{history().length}</span>
                    <span class="kpi-sub">historical passes</span>
                </div>
            </div>

            {/* Tabs */}
            <nav class="purple-tabs">
                <button class={activeTab() === 'overview' ? 'active' : ''} onClick={() => setActiveTab('overview')}>OVERVIEW</button>
                <button class={activeTab() === 'matrix' ? 'active' : ''} onClick={() => setActiveTab('matrix')}>MITRE MATRIX</button>
                <button class={activeTab() === 'history' ? 'active' : ''} onClick={() => setActiveTab('history')}>HISTORY</button>
                <button class={activeTab() === 'execute' ? 'active' : ''} onClick={() => setActiveTab('execute')}>EXECUTE</button>
            </nav>

            <div class="purple-content">
                {/* ── OVERVIEW ── */}
                <Show when={activeTab() === 'overview'}>
                    <div class="overview-grid">
                        {/* Coverage bar per tactic */}
                        <div class="overview-section">
                            <h3>TACTIC COVERAGE</h3>
                            <div class="tactic-bars">
                                <For each={coverage()?.tactic_breakdown || []}>
                                    {(tc: any) => (
                                        <div class="tactic-bar-row">
                                            <span class="tactic-name">{tc.tactic}</span>
                                            <div class="tactic-bar-track">
                                                <div
                                                    class="tactic-bar-fill"
                                                    style={{
                                                        width: `${tc.percent}%`,
                                                        background: coverageColor(tc.percent)
                                                    }}
                                                />
                                            </div>
                                            <span class="tactic-ratio">{tc.covered}/{tc.total}</span>
                                        </div>
                                    )}
                                </For>
                            </div>
                        </div>

                        {/* Gap Report */}
                        <div class="overview-section">
                            <h3>DETECTION GAPS</h3>
                            <div class="gap-list">
                                <For each={coverage()?.gap_ids || []}>
                                    {(id: string) => (
                                        <div class="gap-item">
                                            <span class="gap-indicator">○</span>
                                            <span class="gap-id">{id}</span>
                                            <span class="gap-status">UNCOVERED</span>
                                        </div>
                                    )}
                                </For>
                                <Show when={(coverage()?.gap_ids || []).length === 0}>
                                    <div class="empty-state">Full coverage achieved. No technique gaps.</div>
                                </Show>
                            </div>
                        </div>
                    </div>
                </Show>

                {/* ── MITRE MATRIX ── */}
                <Show when={activeTab() === 'matrix'}>
                    <div class="matrix-grid">
                        <For each={coverage()?.tactic_breakdown || []}>
                            {(tc: any) => (
                                <div
                                    class={`matrix-column ${expandedTactic() === tc.tactic ? 'expanded' : ''}`}
                                    onClick={() => setExpandedTactic(expandedTactic() === tc.tactic ? null : tc.tactic)}
                                >
                                    <div class="matrix-col-header">
                                        <span class="matrix-tactic-name">{tc.tactic}</span>
                                        <span
                                            class="matrix-tactic-pct"
                                            style={{ color: coverageColor(tc.percent) }}
                                        >
                                            {tc.percent.toFixed(0)}%
                                        </span>
                                    </div>
                                    <div class="matrix-col-bar">
                                        <div
                                            class="matrix-bar-fill"
                                            style={{
                                                height: `${tc.percent}%`,
                                                background: coverageColor(tc.percent)
                                            }}
                                        />
                                    </div>
                                    <Show when={expandedTactic() === tc.tactic}>
                                        <div class="matrix-techniques">
                                            <For each={tc.techniques || []}>
                                                {(tech: any) => (
                                                    <div class={`matrix-tech ${tech.covered ? 'covered' : 'gap'}`}>
                                                        <span class="tech-dot">{tech.covered ? '●' : '○'}</span>
                                                        <span class="tech-id">{tech.id}</span>
                                                        <span class="tech-name">{tech.name}</span>
                                                    </div>
                                                )}
                                            </For>
                                        </div>
                                    </Show>
                                </div>
                            )}
                        </For>
                    </div>
                </Show>

                {/* ── VALIDATION HISTORY ── */}
                <Show when={activeTab() === 'history'}>
                    <div class="history-section">
                        <Show when={history().length > 0} fallback={
                            <div class="empty-state">No validation runs yet. Execute a continuous validation pass.</div>
                        }>
                            <table class="history-table">
                                <thead>
                                    <tr>
                                        <th>RUN ID</th>
                                        <th>TIMESTAMP</th>
                                        <th>DETECTED</th>
                                        <th>MISSED</th>
                                        <th>PASS RATE</th>
                                        <th>COVERAGE</th>
                                        <th>DURATION</th>
                                    </tr>
                                </thead>
                                <tbody>
                                    <For each={[...history()].reverse()}>
                                        {(run: any) => (
                                            <tr>
                                                <td class="mono">{run.id}</td>
                                                <td class="mono">{new Date(run.timestamp).toLocaleString()}</td>
                                                <td class="pass">{run.detected}</td>
                                                <td class="fail">{run.missed}</td>
                                                <td>
                                                    <span class={`rate-badge ${run.pass_rate >= 80 ? 'good' : run.pass_rate >= 50 ? 'warn' : 'bad'}`}>
                                                        {run.pass_rate.toFixed(0)}%
                                                    </span>
                                                </td>
                                                <td class="mono">{run.coverage_index.toFixed(1)}%</td>
                                                <td class="mono">{run.duration_ms}ms</td>
                                            </tr>
                                        )}
                                    </For>
                                </tbody>
                            </table>
                        </Show>
                    </div>
                </Show>

                {/* ── EXECUTE (Quick Actions) ── */}
                <Show when={activeTab() === 'execute'}>
                    <div class="execute-section">
                        <div class="execute-card">
                            <h3>▶ FULL SPECTRUM VALIDATION</h3>
                            <p>Execute all {coverage()?.total_techniques ?? 0} mapped scenarios sequentially. Each scenario fires simulated events through the detection pipeline and waits for alert correlation.</p>
                            <button
                                class="ob-btn ob-btn-danger execute-btn"
                                onClick={runValidation}
                                disabled={validating()}
                            >
                                {validating() ? '◉ EXECUTING...' : 'INITIATE VALIDATION RUN'}
                            </button>
                        </div>
                        <div class="execute-card">
                            <h3>⟲ SCHEDULED SELF-TEST</h3>
                            <p>Configure a periodic validation interval. Detections are correlated automatically and coverage metrics updated.</p>
                            <div class="schedule-row">
                                <span class="schedule-label">Status:</span>
                                <span class="schedule-value">Manual only (schedule via cron or playbook)</span>
                            </div>
                        </div>
                    </div>
                </Show>
            </div>
        </div>
    );
};

export default PurpleTeam;
