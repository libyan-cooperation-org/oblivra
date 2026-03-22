import { Component, createSignal, For, Show } from 'solid-js';
import * as SIEMService from '../../../wailsjs/go/services/SIEMService';

export const EnrichmentViewer: Component = () => {
    const [ip, setIp] = createSignal('');
    const [result, setResult] = createSignal<any>(null);
    const [loading, setLoading] = createSignal(false);
    const [error, setError] = createSignal('');
    const [history, setHistory] = createSignal<string[]>([]);

    const lookup = async (target?: string) => {
        const q = target ?? ip().trim();
        if (!q) return;
        setLoading(true);
        setError('');
        setResult(null);
        try {
            // GetFailedLoginsByHost serves as a proxy for host-level enrichment view
            const r = await (SIEMService as any).GetFailedLoginsByHost(q);
            setResult({ host_id: q, failed_logins: r ?? [], enriched: true });
            setHistory(prev => [q, ...prev.filter(h => h !== q)].slice(0, 10));
        } catch (e: any) {
            setError(e?.message ?? String(e));
        }
        setLoading(false);
    };

    const riskScore = (data: any): number => {
        const logins = data?.failed_logins?.length ?? 0;
        return Math.min(100, logins * 12);
    };

    const riskColor = (score: number) => score > 75 ? '#f85149' : score > 40 ? '#d29922' : '#3fb950';

    return (
        <div style="padding: 0; height: 100%; overflow-y: auto; background: var(--bg-primary); color: var(--text-primary); font-family: var(--font-ui);">
            <div style="height: var(--header-height); border-bottom: 1px solid var(--glass-border); display: flex; align-items: center; gap: 0.75rem; padding: 0 1.5rem; background: var(--bg-secondary);">
                <span style="font-size: 16px;">🔬</span>
                <h2 style="font-size: 13px; letter-spacing: 2px; font-weight: 700; margin: 0; text-transform: uppercase;">Enrichment Viewer</h2>
            </div>

            <div style="padding: 1.5rem; display: grid; grid-template-columns: 280px 1fr; gap: 1.5rem; align-items: start;">

                {/* Left: search + history */}
                <div style="display: flex; flex-direction: column; gap: 1rem;">
                    <div style="background: var(--bg-secondary); border: 1px solid var(--glass-border); border-radius: 6px; padding: 1.25rem; display: flex; flex-direction: column; gap: 0.75rem;">
                        <div style="font-size: 10px; text-transform: uppercase; letter-spacing: 1px; color: var(--text-muted); font-family: var(--font-mono);">Entity Lookup</div>
                        <input
                            placeholder="Host ID, IP, hostname..."
                            value={ip()}
                            onInput={e => setIp((e.target as HTMLInputElement).value)}
                            onKeyDown={e => e.key === 'Enter' && lookup()}
                            style="background: var(--bg-primary); border: 1px solid var(--glass-border); color: var(--text-primary); padding: 8px 10px; border-radius: 4px; font-family: var(--font-mono); font-size: 11px; width: 100%; box-sizing: border-box;"
                        />
                        <button onClick={() => lookup()}
                            style="padding: 8px; background: rgba(87,139,255,0.15); border: 1px solid rgba(87,139,255,0.4); color: var(--accent-primary); border-radius: 4px; cursor: pointer; font-family: var(--font-mono); font-size: 11px; font-weight: 700; text-transform: uppercase; letter-spacing: 1px;">
                            ENRICH
                        </button>
                    </div>

                    <Show when={history().length > 0}>
                        <div style="background: var(--bg-secondary); border: 1px solid var(--glass-border); border-radius: 6px; padding: 1rem;">
                            <div style="font-size: 10px; text-transform: uppercase; letter-spacing: 1px; color: var(--text-muted); font-family: var(--font-mono); margin-bottom: 0.5rem;">Recent Lookups</div>
                            <For each={history()}>
                                {(h) => (
                                    <div
                                        onClick={() => { setIp(h); lookup(h); }}
                                        style="padding: 6px 8px; font-family: var(--font-mono); font-size: 11px; color: var(--text-secondary); cursor: pointer; border-radius: 3px; margin-bottom: 2px;"
                                        onMouseEnter={e => (e.currentTarget as HTMLElement).style.background = 'rgba(255,255,255,0.05)'}
                                        onMouseLeave={e => (e.currentTarget as HTMLElement).style.background = 'transparent'}
                                    >
                                        {h}
                                    </div>
                                )}
                            </For>
                        </div>
                    </Show>
                </div>

                {/* Right: enrichment result */}
                <div>
                    <Show when={loading()}>
                        <div style="color: var(--text-muted); font-family: var(--font-mono); font-size: 11px; padding: 2rem;">ENRICHING ENTITY...</div>
                    </Show>

                    <Show when={error()}>
                        <div style="background: rgba(248,81,73,0.1); border: 1px solid rgba(248,81,73,0.3); border-radius: 6px; padding: 1rem; color: #f85149; font-family: var(--font-mono); font-size: 11px;">{error()}</div>
                    </Show>

                    <Show when={!loading() && !error() && result()}>
                        <div style="display: flex; flex-direction: column; gap: 1rem;">
                            {/* Risk score banner */}
                            <div style={`background: rgba(${riskScore(result()) > 75 ? '248,81,73' : riskScore(result()) > 40 ? '210,153,34' : '63,185,80'},0.1); border: 1px solid rgba(${riskScore(result()) > 75 ? '248,81,73' : riskScore(result()) > 40 ? '210,153,34' : '63,185,80'},0.3); border-radius: 6px; padding: 1.25rem; display: flex; justify-content: space-between; align-items: center;`}>
                                <div>
                                    <div style="font-size: 10px; text-transform: uppercase; letter-spacing: 1px; color: var(--text-muted); font-family: var(--font-mono); margin-bottom: 4px;">Risk Score</div>
                                    <div style={`font-size: 2.5rem; font-weight: 900; font-family: var(--font-mono); color: ${riskColor(riskScore(result()))};`}>{riskScore(result())}</div>
                                </div>
                                <div style="text-align: right;">
                                    <div style="font-size: 10px; text-transform: uppercase; letter-spacing: 1px; color: var(--text-muted); font-family: var(--font-mono);">Entity</div>
                                    <div style="font-size: 14px; font-weight: 700; color: var(--text-primary); font-family: var(--font-mono); margin-top: 4px;">{result()?.host_id}</div>
                                </div>
                            </div>

                            {/* Failed logins */}
                            <div style="background: var(--bg-secondary); border: 1px solid var(--glass-border); border-radius: 6px; padding: 1.25rem;">
                                <div style="font-size: 10px; text-transform: uppercase; letter-spacing: 1px; color: var(--text-muted); font-family: var(--font-mono); margin-bottom: 0.75rem;">
                                    Failed Logins ({result()?.failed_logins?.length ?? 0})
                                </div>
                                <Show when={result()?.failed_logins?.length > 0} fallback={
                                    <div style="color: #3fb950; font-family: var(--font-mono); font-size: 11px;">✓ No failed logins detected</div>
                                }>
                                    <table style="width: 100%; border-collapse: collapse; font-size: 10px; font-family: var(--font-mono);">
                                        <thead>
                                            <tr style="border-bottom: 1px solid var(--glass-border);">
                                                {['Username', 'Source IP', 'Count', 'Last Attempt'].map(h => (
                                                    <th style="padding: 6px 10px; text-align: left; color: var(--text-muted); font-weight: 600;">{h}</th>
                                                ))}
                                            </tr>
                                        </thead>
                                        <tbody>
                                            <For each={result()?.failed_logins ?? []}>
                                                {(login: any) => (
                                                    <tr style="border-bottom: 1px solid rgba(255,255,255,0.04);">
                                                        <td style="padding: 6px 10px; color: var(--text-primary);">{login.username ?? '—'}</td>
                                                        <td style="padding: 6px 10px; color: var(--text-secondary);">{login.source_ip ?? '—'}</td>
                                                        <td style="padding: 6px 10px; color: #f0883e; font-weight: 700;">{login.count ?? 1}</td>
                                                        <td style="padding: 6px 10px; color: var(--text-muted);">{login.last_seen?.slice(0, 19)?.replace('T', ' ') ?? '—'}</td>
                                                    </tr>
                                                )}
                                            </For>
                                        </tbody>
                                    </table>
                                </Show>
                            </div>
                        </div>
                    </Show>

                    <Show when={!loading() && !error() && !result()}>
                        <div style="text-align: center; padding: 4rem; color: var(--text-muted); font-family: var(--font-mono); font-size: 11px; letter-spacing: 1px;">
                            <div style="font-size: 3rem; margin-bottom: 1rem; opacity: 0.2;">🔬</div>
                            ENTER AN ENTITY ID TO BEGIN ENRICHMENT
                        </div>
                    </Show>
                </div>
            </div>
        </div>
    );
};
