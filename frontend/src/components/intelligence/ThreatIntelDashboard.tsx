import { Component, createSignal, onMount, For, Show } from 'solid-js';
import * as SIEMService from '../../../wailsjs/go/services/SIEMService';

interface ThreatStat { label: string; value: string | number; accent?: boolean; }

export const ThreatIntelDashboard: Component = () => {
    const [stats, setStats] = createSignal<any>(null);
    const [trend, setTrend] = createSignal<any[]>([]);
    const [loading, setLoading] = createSignal(true);
    const [query, setQuery] = createSignal('');
    const [iocResults, setIocResults] = createSignal<any[]>([]);
    const [searching, setSearching] = createSignal(false);

    onMount(async () => {
        try {
            const [s, t] = await Promise.all([
                (SIEMService as any).GetGlobalThreatStats(),
                (SIEMService as any).GetEventTrend(7),
            ]);
            setStats(s);
            setTrend(t || []);
        } catch { /* no-op */ }
        setLoading(false);
    });

    const searchIOC = async () => {
        if (!query().trim()) return;
        setSearching(true);
        try {
            const results = await (SIEMService as any).SearchHostEvents(query().trim(), 50);
            setIocResults(results || []);
        } catch { setIocResults([]); }
        setSearching(false);
    };

    const topStats = (): ThreatStat[] => {
        if (!stats()) return [];
        return [
            { label: 'Total Events (24h)', value: stats().total_events?.toLocaleString?.() ?? '—' },
            { label: 'Critical Alerts', value: stats().critical_alerts ?? 0, accent: true },
            { label: 'Unique Source IPs', value: stats().unique_sources ?? '—' },
            { label: 'Avg Risk Score', value: stats().avg_risk_score?.toFixed?.(1) ?? '—' },
            { label: 'IOC Matches', value: stats().ioc_matches ?? 0, accent: true },
            { label: 'Active Incidents', value: stats().active_incidents ?? 0 },
        ];
    };

    // Compute mini sparkline path from trend data
    const sparkPath = () => {
        const t = trend();
        if (t.length < 2) return '';
        const vals = t.map((d: any) => d.count ?? 0);
        const max = Math.max(...vals, 1);
        const w = 300, h = 50;
        const pts = vals.map((v, i) => `${(i / (vals.length - 1)) * w},${h - (v / max) * h}`);
        return `M ${pts.join(' L ')}`;
    };

    return (
        <div style="padding: 0; height: 100%; overflow-y: auto; background: var(--bg-primary); color: var(--text-primary); font-family: var(--font-ui);">
            <div style="height: var(--header-height); border-bottom: 1px solid var(--glass-border); display: flex; justify-content: space-between; align-items: center; padding: 0 1.5rem; background: var(--bg-secondary);">
                <div style="display: flex; align-items: center; gap: 0.75rem;">
                    <span style="font-size: 16px;">🔴</span>
                    <h2 style="font-size: 13px; letter-spacing: 2px; font-weight: 700; margin: 0; text-transform: uppercase;">Threat Intelligence</h2>
                </div>
                <span style="font-size: 10px; color: var(--text-muted); font-family: var(--font-mono);">LIVE · {new Date().toLocaleTimeString()}</span>
            </div>

            <div style="padding: 1.5rem; display: flex; flex-direction: column; gap: 1.5rem;">

                {/* KPI Row */}
                <div style="display: grid; grid-template-columns: repeat(auto-fill, minmax(160px, 1fr)); gap: 1px; background: var(--glass-border); border: 1px solid var(--glass-border); border-radius: 6px; overflow: hidden;">
                    <Show when={loading()}>
                        <div style="grid-column: 1 / -1; padding: 2rem; text-align: center; color: var(--text-muted); font-family: var(--font-mono); font-size: 11px;">LOADING THREAT DATA...</div>
                    </Show>
                    <For each={topStats()}>
                        {(s) => (
                            <div style="padding: 1.25rem; background: var(--bg-secondary); display: flex; flex-direction: column; gap: 0.4rem;">
                                <div style="font-size: 9px; text-transform: uppercase; letter-spacing: 1px; color: var(--text-muted); font-family: var(--font-mono);">{s.label}</div>
                                <div style={`font-size: 1.5rem; font-weight: 900; font-family: var(--font-mono); color: ${s.accent ? '#f85149' : 'var(--text-primary)'};`}>{s.value}</div>
                            </div>
                        )}
                    </For>
                </div>

                {/* Trend sparkline */}
                <Show when={trend().length > 0}>
                    <div style="background: var(--bg-secondary); border: 1px solid var(--glass-border); border-radius: 6px; padding: 1.25rem;">
                        <div style="font-size: 10px; text-transform: uppercase; letter-spacing: 1px; color: var(--text-muted); font-family: var(--font-mono); margin-bottom: 0.75rem;">7-Day Event Trend</div>
                        <svg viewBox="0 0 300 50" style="width: 100%; height: 60px;" preserveAspectRatio="none">
                            <path d={sparkPath()} fill="none" stroke="#f85149" stroke-width="1.5" />
                            <path d={sparkPath() + ` L 300,50 L 0,50 Z`} fill="rgba(248,81,73,0.08)" />
                        </svg>
                        <div style="display: flex; justify-content: space-between; font-size: 9px; color: var(--text-muted); font-family: var(--font-mono); margin-top: 4px;">
                            <For each={trend().slice(0, 7)}>
                                {(d: any) => <span>{d.date?.slice(-5) ?? ''}</span>}
                            </For>
                        </div>
                    </div>
                </Show>

                {/* IOC Lookup */}
                <div style="background: var(--bg-secondary); border: 1px solid var(--glass-border); border-radius: 6px; padding: 1.25rem; display: flex; flex-direction: column; gap: 1rem;">
                    <div style="font-size: 10px; text-transform: uppercase; letter-spacing: 1px; color: var(--text-muted); font-family: var(--font-mono);">IOC / Event Search</div>
                    <div style="display: flex; gap: 0.75rem;">
                        <input
                            placeholder="Search IP, hash, hostname, signature..."
                            value={query()}
                            onInput={e => setQuery((e.target as HTMLInputElement).value)}
                            onKeyDown={e => e.key === 'Enter' && searchIOC()}
                            style="flex: 1; background: var(--bg-primary); border: 1px solid var(--glass-border); color: var(--text-primary); padding: 8px 12px; border-radius: 4px; font-family: var(--font-mono); font-size: 11px;"
                        />
                        <button onClick={searchIOC} disabled={searching()}
                            style="padding: 8px 16px; background: rgba(248,81,73,0.15); border: 1px solid rgba(248,81,73,0.4); color: #f85149; border-radius: 4px; cursor: pointer; font-family: var(--font-mono); font-size: 11px; font-weight: 700; text-transform: uppercase; letter-spacing: 1px; white-space: nowrap;">
                            {searching() ? '⏳ SCANNING...' : '🔍 SEARCH'}
                        </button>
                    </div>

                    <Show when={iocResults().length > 0}>
                        <div style="border: 1px solid var(--glass-border); border-radius: 4px; overflow: hidden; max-height: 300px; overflow-y: auto;">
                            <table style="width: 100%; border-collapse: collapse; font-size: 10px; font-family: var(--font-mono);">
                                <thead>
                                    <tr style="background: var(--bg-primary); border-bottom: 1px solid var(--glass-border);">
                                        {['Timestamp', 'Host', 'Event Type', 'Severity', 'Source IP'].map(h => (
                                            <th style="padding: 8px 12px; text-align: left; color: var(--text-muted); font-weight: 600; letter-spacing: 0.5px;">{h}</th>
                                        ))}
                                    </tr>
                                </thead>
                                <tbody>
                                    <For each={iocResults()}>
                                        {(evt: any) => (
                                            <tr style="border-bottom: 1px solid rgba(255,255,255,0.04);">
                                                <td style="padding: 7px 12px; color: var(--text-muted);">{evt.timestamp?.slice(0, 19)?.replace('T', ' ') ?? '—'}</td>
                                                <td style="padding: 7px 12px; color: var(--text-primary); font-weight: 600;">{evt.host_id ?? '—'}</td>
                                                <td style="padding: 7px 12px; color: var(--text-secondary);">{evt.event_type ?? '—'}</td>
                                                <td style="padding: 7px 12px;">
                                                    <span style={`padding: 2px 6px; border-radius: 3px; font-size: 9px; font-weight: 700; background: rgba(${evt.severity === 'critical' ? '248,81,73' : evt.severity === 'high' ? '240,136,62' : '210,153,34'},0.15); color: ${evt.severity === 'critical' ? '#f85149' : evt.severity === 'high' ? '#f0883e' : '#d29922'};`}>
                                                        {(evt.severity ?? 'INFO').toUpperCase()}
                                                    </span>
                                                </td>
                                                <td style="padding: 7px 12px; color: var(--text-muted);">{evt.source_ip ?? '—'}</td>
                                            </tr>
                                        )}
                                    </For>
                                </tbody>
                            </table>
                        </div>
                        <div style="font-size: 10px; color: var(--text-muted); font-family: var(--font-mono);">{iocResults().length} results</div>
                    </Show>
                </div>
            </div>
        </div>
    );
};
