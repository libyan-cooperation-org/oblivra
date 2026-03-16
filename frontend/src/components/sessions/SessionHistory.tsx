import { Component, createSignal, onMount, For, Show } from 'solid-js';

export const SessionHistory: Component = () => {
    const [sessions, setSessions] = createSignal<any[]>([]);
    const [auditLogs, setAuditLogs] = createSignal<any[]>([]);
    const [tab, setTab] = createSignal<'sessions' | 'audit'>('sessions');
    const [loading, setLoading] = createSignal(true);

    onMount(async () => {
        try {
            const { GetHistory, GetAuditLogs } = await import('../../../wailsjs/go/services/SessionService');
            const [s, a] = await Promise.all([GetHistory(50), GetAuditLogs(50)]);
            setSessions(s || []);
            setAuditLogs(a || []);
        } catch (e) { console.error('SessionHistory load error:', e); }
        setLoading(false);
    });

    return (
        <div style="display: flex; flex-direction: column; height: 100%;">
            <div style="display: flex; gap: 2px; padding: 8px 12px; border-bottom: 1px solid var(--border-primary);">
                <button class={`header-tab ${tab() === 'sessions' ? 'active' : ''}`} onClick={() => setTab('sessions')} style="font-size: 11px; padding: 4px 10px;">📋 Sessions</button>
                <button class={`header-tab ${tab() === 'audit' ? 'active' : ''}`} onClick={() => setTab('audit')} style="font-size: 11px; padding: 4px 10px;">🔍 Audit Log</button>
            </div>
            <div style="flex: 1; overflow-y: auto; padding: 8px;">
                <Show when={loading()}><div class="placeholder">Loading...</div></Show>
                <Show when={!loading() && tab() === 'sessions'}>
                    <For each={sessions()} fallback={<div class="placeholder">No session history yet</div>}>
                        {(s) => (
                            <div style="display: flex; align-items: center; gap: 8px; padding: 6px 8px; border-radius: var(--radius-xs); cursor: pointer; font-size: 12px;" class="host-item">
                                <span style={`width: 8px; height: 8px; border-radius: 50%; background: ${s.ended_at ? 'var(--text-muted)' : 'var(--success)'};`} />
                                <div style="flex: 1; min-width: 0;">
                                    <div style="color: var(--text-primary); white-space: nowrap; overflow: hidden; text-overflow: ellipsis;">{s.host_id || 'Unknown'}</div>
                                    <div style="font-size: 10px; color: var(--text-muted);">{new Date(s.started_at).toLocaleString()} • {s.duration_seconds || 0}s</div>
                                </div>
                                <span style="font-size: 10px; color: var(--text-muted);">{s.bytes_sent ? `${(s.bytes_sent / 1024).toFixed(1)}KB` : ''}</span>
                            </div>
                        )}
                    </For>
                </Show>
                <Show when={!loading() && tab() === 'audit'}>
                    <For each={auditLogs()} fallback={<div class="placeholder">No audit logs yet</div>}>
                        {(log) => (
                            <div style="padding: 6px 8px; border-bottom: 1px solid var(--border-subtle); font-size: 11px;">
                                <div style="color: var(--text-primary);">{log.action}</div>
                                <div style="color: var(--text-muted); font-size: 10px;">{log.details} • {new Date(log.timestamp).toLocaleString()}</div>
                            </div>
                        )}
                    </For>
                </Show>
            </div>
        </div>
    );
};
