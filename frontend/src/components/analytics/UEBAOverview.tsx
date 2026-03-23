// UEBAOverview.tsx — Phase 10 Web: entity analytics and risk visualization
import { Component, createSignal, onMount, For, Show } from 'solid-js';
import { IS_BROWSER } from '@core/context';

export const UEBAOverview: Component = () => {
    const [profiles, setProfiles] = createSignal<any[]>([]);
    const [loading, setLoading] = createSignal(true);
    const [selected, setSelected] = createSignal<any>(null);
    const [sortBy, setSortBy] = createSignal<'risk' | 'entity'>('risk');
    const [filter, setFilter] = createSignal('');

    onMount(async () => {
        if (IS_BROWSER) { setLoading(false); return; }
        try {
            const { GetProfiles } = await import('../../../wailsjs/go/services/UEBAService');
            setProfiles(await (GetProfiles as any)() ?? []);
        } catch { setProfiles([]); }
        setLoading(false);
    });

    const sorted = () => {
        let list = profiles().filter(p =>
            !filter() || p.entity_id?.toLowerCase().includes(filter().toLowerCase())
        );
        if (sortBy() === 'risk') list = [...list].sort((a, b) => (b.risk_score ?? 0) - (a.risk_score ?? 0));
        else list = [...list].sort((a, b) => (a.entity_id ?? '').localeCompare(b.entity_id ?? ''));
        return list;
    };

    const riskColor = (score: number) => score > 75 ? '#f85149' : score > 45 ? '#d29922' : score > 20 ? '#f0883e' : '#3fb950';
    const riskLabel = (score: number) => score > 75 ? 'CRITICAL' : score > 45 ? 'HIGH' : score > 20 ? 'MEDIUM' : 'LOW';

    const avgRisk = () => {
        const p = profiles();
        if (!p.length) return 0;
        return p.reduce((acc, x) => acc + (x.risk_score ?? 0), 0) / p.length;
    };

    const criticalCount = () => profiles().filter(p => (p.risk_score ?? 0) > 75).length;

    return (
        <div style="padding: 0; height: 100%; overflow-y: auto; background: var(--bg-primary); color: var(--text-primary); font-family: var(--font-ui);">
            <div style="height: var(--header-height); border-bottom: 1px solid var(--glass-border); display: flex; justify-content: space-between; align-items: center; padding: 0 1.5rem; background: var(--bg-secondary);">
                <div style="display: flex; align-items: center; gap: 0.75rem;">
                    <span style="font-size: 16px;">🧠</span>
                    <h2 style="font-size: 13px; letter-spacing: 2px; font-weight: 700; margin: 0; text-transform: uppercase;">UEBA Overview</h2>
                </div>
                <div style="display: flex; gap: 1.5rem; font-family: var(--font-mono); font-size: 10px;">
                    <span style="color: var(--text-muted);">{profiles().length} ENTITIES</span>
                    <span style="color: #f85149;">{criticalCount()} CRITICAL</span>
                    <span style="color: var(--text-muted);">AVG RISK: <span style={`color: ${riskColor(avgRisk())};`}>{avgRisk().toFixed(1)}</span></span>
                </div>
            </div>

            <div style="padding: 1.5rem; display: flex; flex-direction: column; gap: 1.25rem;">
                {/* Controls */}
                <div style="display: flex; gap: 0.75rem; align-items: center;">
                    <input placeholder="Filter by entity..." value={filter()} onInput={e => setFilter((e.target as HTMLInputElement).value)}
                        style="background: var(--bg-secondary); border: 1px solid var(--glass-border); color: var(--text-primary); padding: 7px 10px; border-radius: 4px; font-family: var(--font-mono); font-size: 11px; width: 220px;" />
                    <button onClick={() => setSortBy('risk')} style={`padding: 6px 12px; font-size: 10px; font-family: var(--font-mono); font-weight: 700; text-transform: uppercase; letter-spacing: 0.5px; border-radius: 3px; cursor: pointer; border: 1px solid ${sortBy() === 'risk' ? 'var(--accent-primary)' : 'var(--glass-border)'}; background: ${sortBy() === 'risk' ? 'rgba(87,139,255,0.15)' : 'transparent'}; color: ${sortBy() === 'risk' ? 'var(--accent-primary)' : 'var(--text-muted)'};`}>Sort by Risk</button>
                    <button onClick={() => setSortBy('entity')} style={`padding: 6px 12px; font-size: 10px; font-family: var(--font-mono); font-weight: 700; text-transform: uppercase; letter-spacing: 0.5px; border-radius: 3px; cursor: pointer; border: 1px solid ${sortBy() === 'entity' ? 'var(--accent-primary)' : 'var(--glass-border)'}; background: ${sortBy() === 'entity' ? 'rgba(87,139,255,0.15)' : 'transparent'}; color: ${sortBy() === 'entity' ? 'var(--accent-primary)' : 'var(--text-muted)'};`}>Sort by Entity</button>
                </div>

                <Show when={loading()}>
                    <div style="padding: 3rem; text-align: center; color: var(--text-muted); font-family: var(--font-mono); font-size: 11px;">LOADING BEHAVIORAL PROFILES...</div>
                </Show>

                <Show when={!loading() && sorted().length === 0}>
                    <div style="padding: 4rem; text-align: center; color: var(--text-muted); font-family: var(--font-mono); font-size: 11px;">
                        <div style="font-size: 3rem; margin-bottom: 1rem; opacity: 0.2;">🧠</div>
                        NO BEHAVIORAL PROFILES<br/>
                        <span style="font-size: 10px; margin-top: 0.5rem; display: block; opacity: 0.6;">Profiles are built automatically as agents report activity. Baselines typically require 24-48 hours.</span>
                    </div>
                </Show>

                {/* Risk heatmap grid */}
                <Show when={sorted().length > 0}>
                    <div style="display: grid; grid-template-columns: repeat(auto-fill, minmax(280px, 1fr)); gap: 1px; background: var(--glass-border); border: 1px solid var(--glass-border); border-radius: 6px; overflow: hidden;">
                        <For each={sorted()}>
                            {(profile) => {
                                const score = profile.risk_score ?? 0;
                                const color = riskColor(score);
                                const isSelected = selected()?.entity_id === profile.entity_id;
                                return (
                                    <div
                                        onClick={() => setSelected(isSelected ? null : profile)}
                                        style={`background: ${isSelected ? 'rgba(87,139,255,0.08)' : 'var(--bg-secondary)'}; padding: 1.25rem; cursor: pointer; border-left: 3px solid ${color};`}
                                    >
                                        <div style="display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 8px;">
                                            <div>
                                                <div style="font-size: 11px; font-weight: 700; font-family: var(--font-mono); color: var(--text-primary);">{profile.entity_id ?? '—'}</div>
                                                <div style="font-size: 9px; color: var(--text-muted); margin-top: 2px; font-family: var(--font-mono);">{profile.entity_type ?? 'user'}</div>
                                            </div>
                                            <div style="text-align: right;">
                                                <div style={`font-size: 1.5rem; font-weight: 900; font-family: var(--font-mono); color: ${color}; line-height: 1;`}>{score.toFixed(0)}</div>
                                                <div style={`font-size: 8px; font-weight: 700; letter-spacing: 1px; color: ${color};`}>{riskLabel(score)}</div>
                                            </div>
                                        </div>
                                        <div style="height: 3px; background: rgba(255,255,255,0.06); border-radius: 2px; overflow: hidden;">
                                            <div style={`height: 100%; width: ${Math.min(100, score)}%; background: ${color}; transition: width 0.4s;`} />
                                        </div>
                                        <Show when={profile.anomaly_count}>
                                            <div style="margin-top: 6px; font-size: 10px; font-family: var(--font-mono); color: var(--text-muted);">
                                                {profile.anomaly_count} anomalies · {profile.last_activity?.slice(0, 10) ?? 'unknown'}
                                            </div>
                                        </Show>
                                    </div>
                                );
                            }}
                        </For>
                    </div>
                </Show>

                {/* Detail panel */}
                <Show when={selected()}>
                    <div style="background: var(--bg-secondary); border: 1px solid var(--glass-border); border-radius: 6px; padding: 1.5rem;">
                        <div style="font-size: 10px; text-transform: uppercase; letter-spacing: 1px; color: var(--text-muted); font-family: var(--font-mono); margin-bottom: 1rem;">Entity Detail: {selected()?.entity_id}</div>
                        <div style="display: grid; grid-template-columns: repeat(auto-fill, minmax(180px, 1fr)); gap: 1rem;">
                            {Object.entries(selected() ?? {}).filter(([k]) => k !== 'entity_id').map(([k, v]) => (
                                <div>
                                    <div style="font-size: 9px; color: var(--text-muted); font-family: var(--font-mono); text-transform: uppercase; letter-spacing: 0.5px; margin-bottom: 2px;">{k.replace(/_/g, ' ')}</div>
                                    <div style="font-size: 11px; color: var(--text-primary); font-family: var(--font-mono);">{String(v) || '—'}</div>
                                </div>
                            ))}
                        </div>
                    </div>
                </Show>
            </div>
        </div>
    );
};
