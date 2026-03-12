import { Component, For, Show, createSignal, createResource, onMount } from 'solid-js';
import { SearchHostEvents } from '../../../../wailsjs/go/app/SIEMService';

export const LogSearchPanel: Component = () => {
    const [query, setQuery] = createSignal('*');
    const [logs, { refetch }] = createResource(query, async (q) => {
        try {
            const data = await SearchHostEvents(q || '*', 100);
            return (data || []).map((log: any) => ({
                time: log.Timestamp ? new Date(log.Timestamp).toLocaleTimeString() : 'N/A',
                src: log.Source || log.RemoteAddr || 'Local',
                dst: log.Destination || 'Internal',
                proto: log.Protocol || 'TCP',
                msg: log.Message || log.Content || '—',
            }));
        } catch (e) {
            console.error('SIEM Search Failed:', e);
            return [];
        }
    });

    onMount(() => refetch());

    const cell: (extra?: Record<string, string>) => Record<string, string> = (extra = {}) => ({
        padding: '5px 8px',
        'border-right': '1px solid var(--border-primary)',
        'white-space': 'nowrap',
        ...extra,
    });

    return (
        <div style={{ display: 'flex', 'flex-direction': 'column', height: '100%', background: 'var(--surface-0)', 'font-family': 'var(--font-mono)', 'font-size': '11px' }}>
            {/* Query bar */}
            <div style={{ padding: '6px 8px', background: 'var(--surface-1)', 'border-bottom': '1px solid var(--border-primary)', display: 'flex', gap: '6px', 'flex-shrink': 0 }}>
                <div style={{ position: 'relative', flex: 1 }}>
                    <span style={{ position: 'absolute', left: '8px', top: '50%', transform: 'translateY(-50%)', color: 'var(--status-online)', 'font-size': '9px', 'font-weight': 800 }}>SQL›</span>
                    <input
                        value={query()}
                        onInput={(e) => setQuery(e.currentTarget.value)}
                        onKeyDown={(e) => e.key === 'Enter' && refetch()}
                        placeholder="SELECT * FROM siem.events WHERE severity > 50…"
                        style={{
                            width: '100%', background: 'var(--surface-0)', border: '1px solid var(--border-primary)',
                            'border-radius': '3px', padding: '5px 8px 5px 40px', color: 'var(--status-online)',
                            'font-family': 'var(--font-mono)', 'font-size': '11px', outline: 'none',
                            'box-sizing': 'border-box',
                        }}
                    />
                </div>
                <button
                    onClick={() => refetch()}
                    disabled={logs.loading}
                    style={{
                        background: logs.loading ? 'var(--surface-2)' : 'var(--accent-primary)',
                        border: 'none', 'border-radius': '3px', padding: '5px 14px', cursor: 'pointer',
                        color: '#000', 'font-family': 'var(--font-mono)', 'font-size': '10px', 'font-weight': 800,
                        'text-transform': 'uppercase', 'flex-shrink': 0,
                    }}
                >
                    {logs.loading ? '…' : 'EXEC'}
                </button>
            </div>

            {/* Table */}
            <div style={{ flex: 1, overflow: 'auto', position: 'relative' }}>
                <Show when={logs.loading}>
                    <div style={{ position: 'absolute', inset: 0, background: 'rgba(0,0,0,0.5)', display: 'flex', 'align-items': 'center', 'justify-content': 'center', 'z-index': 10, 'backdrop-filter': 'blur(2px)' }}>
                        <span style={{ color: 'var(--accent-primary)', 'font-weight': 800, 'letter-spacing': '3px', 'font-size': '10px' }}>INDEX SCAN…</span>
                    </div>
                </Show>

                <table style={{ width: '100%', 'border-collapse': 'collapse', 'font-size': '10px' }}>
                    <thead>
                        <tr style={{ background: 'var(--surface-1)', position: 'sticky', top: 0, 'z-index': 5, 'border-bottom': '1px solid var(--border-primary)' }}>
                            <th style={{ ...cell(), color: 'var(--text-muted)', 'font-weight': 800, 'text-transform': 'uppercase', 'font-size': '9px', 'letter-spacing': '0.5px', width: '72px' }}>TIME</th>
                            <th style={{ ...cell(), color: 'var(--text-muted)', 'font-weight': 800, 'text-transform': 'uppercase', 'font-size': '9px', width: '110px' }}>SOURCE</th>
                            <th style={{ ...cell(), color: 'var(--text-muted)', 'font-weight': 800, 'text-transform': 'uppercase', 'font-size': '9px', width: '110px' }}>DEST</th>
                            <th style={{ ...cell(), color: 'var(--text-muted)', 'font-weight': 800, 'text-transform': 'uppercase', 'font-size': '9px', width: '50px' }}>PROTO</th>
                            <th style={{ ...cell({ 'border-right': 'none' }), color: 'var(--text-muted)', 'font-weight': 800, 'text-transform': 'uppercase', 'font-size': '9px' }}>MESSAGE</th>
                        </tr>
                    </thead>
                    <tbody>
                        <For each={logs() || []}>
                            {(log) => (
                                <tr style={{ 'border-bottom': '1px solid var(--border-primary)' }}>
                                    <td style={{ ...cell(), color: 'var(--text-muted)' }}>{log.time}</td>
                                    <td style={{ ...cell(), color: 'var(--accent-primary)', 'font-weight': 700 }}>{log.src}</td>
                                    <td style={{ ...cell(), color: 'var(--status-online)', 'font-weight': 700 }}>{log.dst}</td>
                                    <td style={{ ...cell(), color: 'var(--alert-medium)', opacity: '0.8' }}>{log.proto}</td>
                                    <td style={{ ...cell({ 'border-right': 'none', overflow: 'hidden', 'text-overflow': 'ellipsis', 'max-width': '0', color: 'var(--text-secondary)' }) }} title={log.msg}>{log.msg}</td>
                                </tr>
                            )}
                        </For>
                    </tbody>
                </table>

                <Show when={!logs.loading && (!logs() || logs()!.length === 0)}>
                    <div style={{ padding: '40px', 'text-align': 'center', opacity: '0.25', 'font-style': 'italic', 'font-size': '10px', 'text-transform': 'uppercase' }}>
                        No events matching criteria
                    </div>
                </Show>
            </div>

            {/* Footer */}
            <div style={{ padding: '4px 12px', background: 'var(--surface-1)', 'border-top': '1px solid var(--border-primary)', display: 'flex', 'justify-content': 'space-between', 'align-items': 'center', 'flex-shrink': 0 }}>
                <div style={{ display: 'flex', gap: '16px', 'font-size': '9px', color: 'var(--text-muted)', 'text-transform': 'uppercase' }}>
                    <span>HITS: <span style={{ color: 'var(--text-secondary)' }}>{logs()?.length ?? 0}</span></span>
                    <span>SHARDS: <span style={{ color: 'var(--text-secondary)' }}>01/01</span></span>
                </div>
                <div style={{ display: 'flex', gap: '12px', 'font-size': '9px' }}>
                    <button onClick={() => refetch()} style={{ color: 'var(--accent-primary)', background: 'none', border: 'none', cursor: 'pointer', 'font-family': 'var(--font-mono)', 'font-weight': 800, 'text-transform': 'uppercase' }}>↻ REFRESH</button>
                </div>
            </div>
        </div>
    );
};
