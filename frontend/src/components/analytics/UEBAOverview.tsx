// UEBAOverview.tsx — Phase 10 Web: entity analytics and risk visualization
import { Component, createSignal, onMount, For, Show, createMemo } from 'solid-js';
import { IS_BROWSER } from '@core/context';
import { 
    PageLayout, 
    Panel, 
    Badge, 
    KPI, 
    KPIGrid,
    Input,
    Button,
    EmptyState,
    LoadingScreen
} from '@components/ui';
import '../../styles/ueba-overview.css';

export const UEBAOverview: Component = () => {
    const [profiles, setProfiles] = createSignal<any[]>([]);
    const [loading, setLoading] = createSignal(true);
    const [selected, setSelected] = createSignal<any>(null);
    const [sortBy, setSortBy] = createSignal<'risk' | 'entity'>('risk');
    const [filter, setFilter] = createSignal('');

    const fetchData = async () => {
        setLoading(true);
        if (IS_BROWSER) { setLoading(false); return; }
        try {
            const { GetProfiles } = await import('../../../wailsjs/go/services/UEBAService');
            const data = await (GetProfiles as any)() ?? [];
            setProfiles(data);
        } catch (err) {
            console.error('Failed to fetch UEBA profiles:', err);
            setProfiles([]);
        } finally {
            setLoading(false);
        }
    };

    onMount(() => fetchData());

    const sorted = createMemo(() => {
        let list = profiles().filter(p =>
            !filter() || p.entity_id?.toLowerCase().includes(filter().toLowerCase())
        );
        if (sortBy() === 'risk') {
            list = [...list].sort((a, b) => (b.risk_score ?? 0) - (a.risk_score ?? 0));
        } else {
            list = [...list].sort((a, b) => (a.entity_id ?? '').localeCompare(b.entity_id ?? ''));
        }
        return list;
    });

    const getRiskSeverity = (score: number) => {
        if (score > 75) return 'critical';
        if (score > 45) return 'high';
        if (score > 20) return 'medium';
        return 'low';
    };

    const getRiskColor = (score: number) => {
        if (score > 75) return 'var(--alert-critical)';
        if (score > 45) return 'var(--alert-high)';
        if (score > 20) return 'var(--alert-medium)';
        return 'var(--alert-low)';
    };

    const avgRisk = createMemo(() => {
        const p = profiles();
        if (!p.length) return 0;
        return p.reduce((acc, x) => acc + (x.risk_score ?? 0), 0) / p.length;
    });

    const criticalCount = createMemo(() => profiles().filter(p => (p.risk_score ?? 0) > 75).length);

    return (
        <PageLayout 
            title="User & Entity Behavior Analytics" 
            subtitle="BEHAVIORAL_BASELINE_DETECTION"
            actions={
                <div style="display: flex; gap: 8px;">
                    <Button variant="ghost" size="sm" onClick={fetchData}>REFRESH_PROFILES</Button>
                </div>
            }
        >
            <div class="ueba-container">
                <KPIGrid cols={3}>
                    <KPI 
                        label="TOTAL_ENTITIES" 
                        value={profiles().length} 
                    />
                    <KPI 
                        label="CRITICAL_RISK" 
                        value={criticalCount()} 
                        color="var(--alert-critical)" 
                    />
                    <KPI 
                        label="AVERAGE_RISK_SCORE" 
                        value={avgRisk().toFixed(1)} 
                        color={getRiskColor(avgRisk())} 
                    />
                </KPIGrid>

                <div class="ueba-controls">
                    <Input 
                        placeholder="Filter by entity ID..." 
                        value={filter()} 
                        onInput={setFilter}
                        style="width: 260px;"
                    />
                    <Button 
                        variant={sortBy() === 'risk' ? 'blue' : 'ghost'} 
                        size="sm" 
                        onClick={() => setSortBy('risk')}
                    >
                        SORT_BY_RISK
                    </Button>
                    <Button 
                        variant={sortBy() === 'entity' ? 'blue' : 'ghost'} 
                        size="sm" 
                        onClick={() => setSortBy('entity')}
                    >
                        SORT_BY_ENTITY
                    </Button>
                </div>

                <Show when={loading()}>
                    <div style="flex: 1; min-height: 200px; display: flex; align-items: center; justify-content: center;">
                        <LoadingScreen message="ANALYZING_BEHAVIORAL_PROFILES..." />
                    </div>
                </Show>

                <Show when={!loading() && sorted().length === 0}>
                    <EmptyState
                        icon="🧠"
                        title="NO_BEHAVIORAL_PROFILES"
                        description="Profiles are built automatically as agents report activity. Baselines typically require 24-48 hours of telemetry."
                    />
                </Show>

                <Show when={!loading() && sorted().length > 0}>
                    <div class="ueba-grid">
                        <For each={sorted()}>
                            {(profile) => {
                                const score = profile.risk_score ?? 0;
                                const severity = getRiskSeverity(score);
                                const color = getRiskColor(score);
                                const isSelected = () => selected()?.entity_id === profile.entity_id;

                                return (
                                    <div
                                        class="ueba-card"
                                        classList={{ selected: isSelected() }}
                                        style={{ "border-left-color": color }}
                                        onClick={() => setSelected(isSelected() ? null : profile)}
                                    >
                                        <div class="ueba-card-header">
                                            <div>
                                                <div class="ueba-entity-id">{profile.entity_id ?? '—'}</div>
                                                <div class="ueba-entity-type">{profile.entity_type ?? 'user'}</div>
                                            </div>
                                            <div class="ueba-risk-score">
                                                <div class="ueba-score-value" style={{ color: color }}>{score.toFixed(0)}</div>
                                                <div class="ueba-score-label" style={{ color: color }}>{severity.toUpperCase()}</div>
                                            </div>
                                        </div>
                                        <div class="ueba-risk-bar-bg">
                                            <div class="ueba-risk-bar-fill" style={{ width: `${Math.min(100, score)}%`, background: color }} />
                                        </div>
                                        <Show when={profile.anomaly_count}>
                                            <div class="ueba-meta">
                                                {profile.anomaly_count} ANOMALIES · {profile.last_activity?.slice(0, 10) ?? 'UNKNOWN'}
                                            </div>
                                        </Show>
                                    </div>
                                );
                            }}
                        </For>
                    </div>
                </Show>

                <Show when={selected()}>
                    <div class="ueba-detail-panel">
                        <div class="ob-section-header">ENTITY_DETAIL: {selected()?.entity_id}</div>
                        <div class="ueba-detail-grid">
                            <For each={Object.entries(selected() ?? {}).filter(([k]) => !['entity_id', 'risk_score'].includes(k))}>
                                {([k, v]) => (
                                    <div class="ueba-detail-item">
                                        <div class="ueba-detail-label">{k.replace(/_/g, ' ')}</div>
                                        <div class="ueba-detail-val">{String(v) || '—'}</div>
                                    </div>
                                )}
                            </For>
                        </div>
                    </div>
                </Show>
            </div>
        </PageLayout>
);
};

export default UEBAOverview;
