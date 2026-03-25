import { Component, createResource, For, Show } from 'solid-js';
import { GetRiskHistory } from '../../wailsjs/go/services/RiskService';
import { 
    PageLayout, 
    KPIGrid, 
    KPI, 
    Panel, 
    Badge, 
    Button, 
    SectionHeader, 
    LoadingState, 
    normalizeSeverity,
    formatTimestamp 
} from '@components/ui';

export const ConfigRisk: Component = () => {
    const [history, { refetch }] = createResource(async () => {
        try {
            return await GetRiskHistory();
        } catch {
            return [];
        }
    });

    const getRiskColor = (score: number) => {
        if (score >= 75) return 'var(--alert-critical)';
        if (score >= 50) return 'var(--alert-high)';
        if (score >= 25) return 'var(--accent-primary)';
        return 'var(--status-online)';
    };

    return (
        <PageLayout
            title="Configuration Risk Audit"
            subtitle="BLAST_RADIUS_EVALUATION_&_POSTURE_IMPACT"
            actions={
                <Button variant="default" onClick={refetch}>
                    RE-SYNC_MODELS
                </Button>
            }
        >
            <KPIGrid cols={3}>
                <KPI 
                    label="TOTAL_AUDITS" 
                    value={history()?.length || 0} 
                    color="var(--text-primary)" 
                />
                <KPI 
                    label="CRITICAL_RISKS" 
                    value={history()?.filter(r => r.score >= 75).length || 0} 
                    color="var(--alert-critical)" 
                />
                <KPI 
                    label="AVG_EXPOSURE" 
                    value={history()?.length ? Math.round(history()!.reduce((acc, curr) => acc + curr.score, 0) / history()!.length) : 0} 
                    deltaLabel="/ 100"
                    color="var(--alert-high)" 
                />
            </KPIGrid>

            <div style="display: flex; flex-direction: column; gap: var(--gap-lg); margin-top: var(--gap-lg); max-width: 1000px; margin-left: auto; margin-right: auto;">
                <Show when={history.loading}>
                    <LoadingState message="RECONSTRUCTING_RISK_TIMELINE..." />
                </Show>

                <Show when={!history.loading && history()?.length === 0}>
                    <div style="display: flex; flex-direction: column; align-items: center; justify-content: center; padding: 64px; border: 1px dashed var(--border-primary); color: var(--text-muted); border-radius: var(--radius-md);">
                        <div style="font-size: 24px; margin-bottom: 12px;">✓</div>
                        <div style="font-family: var(--font-mono); font-size: 13px; font-weight: 800;">ZERO_EXPOSURE_DETECTED</div>
                        <div style="font-size: 12px;">No configuration changes logged in current epoch.</div>
                    </div>
                </Show>

                <For each={history()}>
                    {(risk) => (
                        <Panel 
                            title={risk.reason} 
                            actions={
                                <div style="display: flex; align-items: center; gap: 8px;">
                                    <Badge severity={normalizeSeverity(risk.level)}>
                                        {risk.level}
                                    </Badge>
                                    <span style="font-family: var(--font-mono); font-size: 10px; color: var(--text-muted);">ID:{risk.id.substring(0,8)}</span>
                                </div>
                            }
                        >
                            <div style="display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: var(--gap-md);">
                                <div style="display: flex; align-items: center; gap: 12px;">
                                    <div style={{
                                        width: '4px',
                                        height: '24px',
                                        background: getRiskColor(risk.score)
                                    }} />
                                    <div>
                                        <div style="font-size: 11px; color: var(--text-muted); text-transform: uppercase; letter-spacing: 0.5px;">Exposure Score</div>
                                        <div style="font-family: var(--font-mono); font-size: 16px; font-weight: 800;">{risk.score}<span style="font-size: 10px; color: var(--text-muted);">/100</span></div>
                                    </div>
                                </div>
                                <div style="text-align: right;">
                                    <div style="font-size: 10px; color: var(--text-muted); text-transform: uppercase;">Observed At</div>
                                    <div style="font-family: var(--font-mono); font-size: 11px;">{formatTimestamp(risk.timestamp)}</div>
                                </div>
                            </div>
                            
                            <SectionHeader>IMPACT_ANALYSIS</SectionHeader>
                            <div style={{
                                background: 'var(--surface-0)',
                                padding: '12px 16px',
                                'border-radius': 'var(--radius-sm)',
                                border: '1px solid var(--border-subtle)',
                                'font-family': 'var(--font-mono)',
                                'font-size': '12px',
                                color: 'var(--text-secondary)',
                                'line-height': '1.6'
                            }}>
                                {risk.impact}
                            </div>
                        </Panel>
                    )}
                </For>
            </div>
        </PageLayout>
    );
};
