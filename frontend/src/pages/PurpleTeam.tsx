import { Component, createSignal, onMount, For, Show } from 'solid-js';
import * as SimulationService from '../../wailsjs/go/simulation/SimulationService';
import { 
    PageLayout, 
    KPIGrid, 
    KPI, 
    Table, 
    Badge, 
    Panel, 
    Button, 
    Notice,
    TabBar,
    Progress,
    formatTimestamp,
    Column
} from '@components/ui';

export const PurpleTeam: Component = () => {
    const [report, setReport] = createSignal<any>(null);
    const [coverage, setCoverage] = createSignal<any>(null);
    const [history, setHistory] = createSignal<any[]>([]);
    const [activeTab, setActiveTab] = createSignal<'overview' | 'matrix' | 'history' | 'execute'>('overview');
    const [validating, setValidating] = createSignal(false);
    const [msg, setMsg] = createSignal('');
    const [expandedTactic, setExpandedTactic] = createSignal<string | null>(null);

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

    onMount(async () => await refreshData());

    const runValidation = async () => {
        setValidating(true);
        setMsg('⚡ CONTINUOUS_VALIDATION_IN_PROGRESS — executing all scenario vectors...');
        try {
            await SimulationService.RunContinuousValidation();
            setMsg('✅ Validation cycle complete. Refreshing metrics...');
            await refreshData();
            setTimeout(() => setMsg(''), 5000);
        } catch (e) {
            setMsg(`❌ Validation failed: ${e}`);
        } finally {
            setValidating(false);
        }
    };

    const getGradeColor = (grade: string) => {
        switch (grade) {
            case 'A': return 'var(--status-online)';
            case 'B': return 'var(--accent-primary)';
            case 'C': return 'var(--alert-medium)';
            case 'D': return 'var(--alert-high)';
            default: return 'var(--alert-critical)';
        }
    };

    const getProgressColor = (pct: number): 'green' | 'orange' | 'red' | 'blue' => {
        if (pct >= 66) return 'green';
        if (pct >= 33) return 'orange';
        return 'red';
    };

    const historyColumns: Column<any>[] = [
        { key: 'id', label: 'RUN_ID', width: '100px', mono: true, render: (r) => r.id || '—' },
        { 
            key: 'timestamp', 
            label: 'TIMESTAMP', 
            width: '180px', 
            mono: true, 
            render: (r) => formatTimestamp(r.timestamp) 
        },
        { key: 'detected', label: 'DETECTED', width: '90px' },
        { key: 'missed', label: 'MISSED', width: '90px' },
        { 
            key: 'pass_rate', 
            label: 'PASS_RATE',
            render: (r) => (
                <Badge severity={r.pass_rate >= 80 ? 'success' : r.pass_rate >= 50 ? 'warning' : 'danger'}>
                    {r.pass_rate.toFixed(0)}%
                </Badge>
            )
        },
        { 
            key: 'coverage_index', 
            label: 'COVERAGE', 
            mono: true, 
            render: (r) => `${r.coverage_index.toFixed(1)}%` 
        },
        { 
            key: 'duration_ms', 
            label: 'DURATION', 
            mono: true, 
            render: (r) => `${r.duration_ms}ms` 
        }
    ];

    const tabs = [
        { id: 'overview', label: 'PERFORMANCE_OVERVIEW' },
        { id: 'matrix', label: 'MITRE_MATRIX' },
        { id: 'history', label: 'VALIDATION_HISTORY' },
        { id: 'execute', label: 'ADVERSARY_SIMULATION' }
    ];

    return (
        <PageLayout
            title="Purple Team Engine"
            subtitle="CONTINUOUS_CONTROLS_VALIDATION_&_RESILIENCE"
            actions={
                <>
                    <Button variant="ghost" onClick={refreshData}>REFRESH</Button>
                    <Button
                        variant="primary"
                        onClick={runValidation}
                        loading={validating()}
                        style="background: var(--alert-critical); color: #fff; border-color: transparent;"
                    >
                        RUN_FULL_VALIDATION
                    </Button>
                </>
            }
        >
            <Show when={msg()}>
                <Notice level={msg().startsWith('❌') ? 'error' : 'info'}>
                    {msg()}
                </Notice>
            </Show>

            <KPIGrid cols={5} class="mb-6">
                <KPI 
                    label="RESILIENCE_GRADE" 
                    value={report()?.resilience_grade || '—'} 
                    color={getGradeColor(report()?.resilience_grade || 'F')}
                    subtitle={`SCORE: ${(report()?.resilience_score ?? 0).toFixed(1)}`}
                />
                <KPI 
                    label="DETECTION_RATE" 
                    value={`${(report()?.detection_rate ?? 0).toFixed(0)}%`} 
                    sparkData={[65, 72, 68, 80, 85]}
                    sparkColor="var(--status-online)"
                />
                <KPI 
                    label="COVERAGE_INDEX" 
                    value={`${(report()?.coverage_index ?? 0).toFixed(1)}%`} 
                    subtitle={`${coverage()?.covered_techniques ?? 0}/${coverage()?.total_techniques ?? 0} techniques`}
                />
                <KPI 
                    label="MEAN_RESPONSE" 
                    value={`${report()?.mean_response_ms ?? '—'}ms`} 
                    color="var(--accent-primary)"
                />
                <KPI 
                    label="VALIDATION_RUNS" 
                    value={history().length} 
                    subtitle="historical passes"
                />
            </KPIGrid>

            <TabBar 
                tabs={tabs} 
                active={activeTab()} 
                onSelect={(id) => setActiveTab(id as any)} 
                class="mb-6"
            />

            <div style="flex: 1; overflow-y: auto;">
                {/* ── OVERVIEW ── */}
                <Show when={activeTab() === 'overview'}>
                    <div style="display: grid; grid-template-columns: 1fr 1fr; gap: var(--gap-lg);">
                        <Panel title="TACTIC_COVERAGE_METRICS">
                            <div style="display: flex; flex-direction: column; gap: 16px;">
                                <For each={coverage()?.tactic_breakdown || []}>
                                    {(tc: any) => (
                                        <div style="display: flex; flex-direction: column; gap: 6px;">
                                            <div style="display: flex; justify-content: space-between; font-size: 11px; font-weight: 700; color: var(--text-primary);">
                                                <span style="text-transform: uppercase;">{tc.tactic}</span>
                                                <span style="font-family: var(--font-mono); color: var(--text-muted);">{tc.covered}/{tc.total}</span>
                                            </div>
                                            <Progress value={tc.percent} color={getProgressColor(tc.percent)} />
                                        </div>
                                    )}
                                </For>
                            </div>
                        </Panel>

                        <Panel title="DETECTION_GAP_ANALYSIS">
                            <div style="display: flex; flex-direction: column; gap: 8px;">
                                <For each={coverage()?.gap_ids || []}>
                                    {(id: string) => (
                                        <div style="display: flex; align-items: center; justify-content: space-between; padding: 8px 12px; background: rgba(0,0,0,0.1); border-radius: var(--radius-sm); border-left: 3px solid var(--alert-critical);">
                                            <div style="display: flex; align-items: center; gap: 8px;">
                                                <span style="color: var(--alert-critical); font-size: 14px;">○</span>
                                                <span style="font-family: var(--font-mono); font-size: 11px; font-weight: 700;">{id}</span>
                                            </div>
                                            <Badge severity="danger" size="sm">UNCOVERED_GAP</Badge>
                                        </div>
                                    )}
                                </For>
                                <Show when={(coverage()?.gap_ids || []).length === 0}>
                                    <div style="text-align: center; padding: 48px 0; color: var(--status-online); font-size: 12px; font-weight: 700;">
                                        ✅ FULL COVERAGE ACHIEVED ACROSS ALL MONITORED VECTORS
                                    </div>
                                </Show>
                            </div>
                        </Panel>
                    </div>
                </Show>

                {/* ── MITRE MATRIX ── */}
                <Show when={activeTab() === 'matrix'}>
                    <div style="display: flex; gap: 8px; overflow-x: auto; padding-bottom: 12px; min-height: 400px;">
                        <For each={coverage()?.tactic_breakdown || []}>
                            {(tc: any) => (
                                <div 
                                    onClick={() => setExpandedTactic(expandedTactic() === tc.tactic ? null : tc.tactic)}
                                    style={{
                                        'width': expandedTactic() === tc.tactic ? '300px' : '120px',
                                        'min-width': expandedTactic() === tc.tactic ? '300px' : '120px',
                                        'background': 'var(--surface-1)',
                                        'border': '1px solid var(--border-primary)',
                                        'transition': 'all 0.3s cubic-bezier(0.4, 0, 0.2, 1)',
                                        'border-radius': 'var(--radius-sm)',
                                        'display': 'flex',
                                        'flex-direction': 'column',
                                        'cursor': 'pointer'
                                    }}
                                >
                                    <div style="padding: 12px 8px; border-bottom: 2px solid var(--border-secondary); background: var(--surface-2); text-align: center;">
                                        <div style="font-size: 9px; font-weight: 800; color: var(--text-primary); text-transform: uppercase; white-space: nowrap; overflow: hidden; text-overflow: ellipsis;">{tc.tactic}</div>
                                        <div style={{ 'font-family': 'var(--font-mono)', 'font-size': '12px', 'font-weight': '900', 'color': `var(--status-${getProgressColor(tc.percent)})`, 'margin-top': '4px' }}>
                                            {tc.percent.toFixed(0)}%
                                        </div>
                                    </div>
                                    <div style="flex: 1; position: relative;">
                                        <div style={{
                                            'position': 'absolute', 'bottom': '0', 'left': '0', 'right': '0',
                                            'height': `${tc.percent}%`, 'background': `var(--status-${getProgressColor(tc.percent)})`,
                                            'opacity': '0.15', 'transition': 'height 0.5s ease-out'
                                        }} />
                                        <Show when={expandedTactic() === tc.tactic}>
                                            <div style="position: relative; padding: 12px; display: flex; flex-direction: column; gap: 6px; overflow-y: auto;">
                                                <For each={tc.techniques || []}>
                                                    {(tech: any) => (
                                                        <div style={{
                                                            'padding': '6px 8px', 'background': 'rgba(0,0,0,0.2)', 'border-radius': 'var(--radius-xs)',
                                                            'border-left': `2px solid ${tech.covered ? 'var(--status-online)' : 'var(--alert-critical)'}`,
                                                            'display': 'flex', 'flex-direction': 'column', 'gap': '2px'
                                                        }}>
                                                            <div style="display: flex; justify-content: space-between; align-items: center;">
                                                                <span style="font-family: var(--font-mono); font-size: 10px; font-weight: 800; color: var(--accent-primary);">{tech.id}</span>
                                                                <span style={{'font-size': '9px', 'color': tech.covered ? 'var(--status-online)' : 'var(--alert-critical)'}}>{tech.covered ? '●' : '○'}</span>
                                                            </div>
                                                            <div style="font-size: 10px; color: var(--text-secondary); white-space: nowrap; overflow: hidden; text-overflow: ellipsis;">{tech.name}</div>
                                                        </div>
                                                    )}
                                                </For>
                                            </div>
                                        </Show>
                                    </div>
                                </div>
                            )}
                        </For>
                    </div>
                </Show>

                {/* ── VALIDATION HISTORY ── */}
                <Show when={activeTab() === 'history'}>
                    <Panel noPadding>
                        <Table 
                            columns={historyColumns} 
                            data={[...history()].reverse()} 
                            emptyText="NO VALIDATION RUNS DETECTED"
                            striped
                        />
                    </Panel>
                </Show>

                {/* ── EXECUTE ── */}
                <Show when={activeTab() === 'execute'}>
                    <div style="display: grid; grid-template-columns: 1fr 1fr; gap: var(--gap-lg);">
                        <Panel title="FULL_SPECTRUM_VALIDATION">
                            <div style="margin-bottom: var(--gap-lg); font-size: 13px; color: var(--text-secondary); line-height: 1.6;">
                                Execute all {coverage()?.total_techniques ?? 0} mapped scenarios sequentially. 
                                Each scenario fires simulated events through the detection pipeline and waits for alert correlation.
                            </div>
                            <Button
                                variant="primary"
                                onClick={runValidation}
                                loading={validating()}
                                style="width: 100%; height: 48px; background: var(--alert-critical); color: #fff; font-weight: 800;"
                            >
                                {validating() ? 'VALIDATION_CYCLE_RUNNING...' : 'INITIATE_VALIDATION_PASS'}
                            </Button>
                        </Panel>

                        <Panel title="SCHEDULED_CONTROLS_SELF-TEST">
                            <div style="margin-bottom: var(--gap-lg); font-size: 13px; color: var(--text-secondary); line-height: 1.6;">
                                Configure a periodic validation interval. Detections are correlated automatically and coverage metrics updated via the OBLIVRA automation engine.
                            </div>
                            <div style="display: flex; justify-content: space-between; align-items: center; padding: 12px; background: var(--surface-2); border-radius: var(--radius-sm); border: 1px solid var(--border-primary);">
                                <span style="font-size: 11px; font-weight: 700; color: var(--text-muted); text-transform: uppercase;">Engine Status</span>
                                <Badge severity="neutral">MANUAL_TRIGGER_ONLY</Badge>
                            </div>
                        </Panel>
                    </div>
                </Show>
            </div>
        </PageLayout>
    );
};

export default PurpleTeam;
