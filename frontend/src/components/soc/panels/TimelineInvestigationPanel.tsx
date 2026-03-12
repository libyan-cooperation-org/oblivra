import { Component, For, Show, createResource, onMount, onCleanup } from 'solid-js';
import { ListIncidents } from '../../../../wailsjs/go/app/IncidentService';

export const TimelineInvestigationPanel: Component = () => {
    const [incidents, { refetch }] = createResource(async () => {
        try {
            // @ts-ignore - handling potential wails context arg mismatch
            const data = await ListIncidents('', '', 50);
            return (data || []).map((inc: any) => ({
                id: inc.ID || Math.random(),
                time: inc.CreatedAt ? new Date(inc.CreatedAt).toLocaleTimeString() : 'N/A',
                type: inc.Type || 'DETECTION',
                title: inc.Title || 'Security Event',
                critical: inc.Status === 'OPEN',
                description: inc.Description || 'System initiated investigation',
            }));
        } catch (e) {
            console.error('Failed to fetch incidents:', e);
            return [];
        }
    });

    onMount(() => {
        const interval = setInterval(refetch, 15000);
        onCleanup(() => clearInterval(interval));
    });

    return (
        <div style={{ display: 'flex', 'flex-direction': 'column', height: '100%', background: 'var(--surface-0)', 'font-family': 'var(--font-mono)', 'font-size': '11px' }}>
            {/* Header */}
            <div style={{ padding: '8px 12px', 'border-bottom': '1px solid var(--border-primary)', display: 'flex', 'justify-content': 'space-between', 'align-items': 'center', background: 'var(--surface-1)', 'flex-shrink': 0 }}>
                <span style={{ color: 'var(--text-muted)', 'font-weight': 800, 'letter-spacing': '2px', 'text-transform': 'uppercase', 'font-size': '10px' }}>Incident Chronology</span>
                <Show when={incidents.loading}>
                    <span style={{ 'font-size': '9px', color: 'var(--accent-primary)' }}>SYNCING…</span>
                </Show>
            </div>

            {/* Timeline */}
            <div style={{ flex: 1, 'overflow-y': 'auto', padding: '16px 16px 16px 12px', position: 'relative' }}>
                {/* Vertical spine */}
                <div style={{ position: 'absolute', left: '22px', top: 0, bottom: 0, width: '1px', background: 'var(--border-primary)' }} />

                <div style={{ display: 'flex', 'flex-direction': 'column', gap: '20px' }}>
                    <For each={incidents() || []}>
                        {(event) => (
                            <div style={{ position: 'relative', 'padding-left': '28px' }}>
                                {/* Timeline dot */}
                                <div style={{
                                    position: 'absolute', left: '14px', top: '3px',
                                    width: '14px', height: '14px', 'border-radius': '50%',
                                    background: event.critical ? 'var(--alert-critical)' : 'var(--alert-medium)',
                                    border: '2px solid var(--surface-0)',
                                    'box-shadow': event.critical ? '0 0 8px rgba(240,64,64,0.5)' : 'none',
                                    'z-index': 1,
                                }} />

                                <div style={{ display: 'flex', 'align-items': 'center', gap: '8px', 'margin-bottom': '4px' }}>
                                    <span style={{ color: 'var(--text-muted)', 'font-size': '9px' }}>{event.time}</span>
                                    <span style={{ padding: '1px 5px', background: 'var(--surface-2)', border: '1px solid var(--border-primary)', 'font-size': '8px', 'font-weight': 800, 'text-transform': 'uppercase', 'letter-spacing': '0.5px', color: 'var(--text-muted)' }}>{event.type}</span>
                                </div>
                                <div style={{ color: event.critical ? 'var(--alert-critical)' : 'var(--text-primary)', 'font-weight': 700, 'font-size': '12px', 'margin-bottom': '3px', cursor: 'pointer' }}>{event.title}</div>
                                <div style={{ color: 'var(--text-muted)', 'font-size': '10px', 'line-height': '1.5', 'font-style': 'italic', opacity: '0.7' }}>{event.description}</div>
                            </div>
                        )}
                    </For>

                    <Show when={!incidents.loading && (!incidents() || incidents()!.length === 0)}>
                        <div style={{ 'padding-top': '40px', 'text-align': 'center', opacity: '0.25', 'font-style': 'italic', 'font-size': '10px', 'text-transform': 'uppercase' }}>
                            No significant incidents recorded
                        </div>
                    </Show>
                </div>
            </div>

            {/* Footer */}
            <div style={{ padding: '6px 12px', 'border-top': '1px solid var(--border-primary)', background: 'var(--surface-1)', display: 'flex', 'justify-content': 'flex-end', 'flex-shrink': 0 }}>
                <button
                    style={{ 'font-size': '9px', color: 'var(--accent-primary)', background: 'none', border: 'none', cursor: 'pointer', 'font-weight': 800, 'font-family': 'var(--font-mono)', 'text-transform': 'uppercase' }}
                >
                    Incident Manager →
                </button>
            </div>
        </div>
    );
};
