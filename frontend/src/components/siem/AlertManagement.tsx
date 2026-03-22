// AlertManagement.tsx — Phase 5 Web: alert queue management
import { Component, createSignal, onMount, For, Show } from 'solid-js';
import * as AlertingService from '../../../wailsjs/go/services/AlertingService';
import { database } from '../../../wailsjs/go/models';

const SEV_COLOR: Record<string, string> = {
    critical: '#f85149', high: '#f0883e', medium: '#d29922', low: '#3fb950'
};

export const AlertManagement: Component = () => {
    const [incidents, setIncidents] = createSignal<database.Incident[]>([]);
    const [filter, setFilter] = createSignal('New');
    const [loading, setLoading] = createSignal(true);
    const [bulkSelected, setBulkSelected] = createSignal<Set<string>>(new Set());

    const load = async () => {
        setLoading(true);
        // ListIncidents(status, limit) — AlertingService signature
        try { setIncidents(await (AlertingService as any).ListIncidents(filter(), 200) ?? []); } catch { setIncidents([]); }
        setLoading(false);
    };

    onMount(load);

    const act = async (id: string, status: string) => {
        try { await (AlertingService as any).UpdateIncidentStatus(id, status, 'Updated via Alert Management'); load(); } catch {}
    };

    const bulkAck = async () => {
        for (const id of bulkSelected()) await act(id, 'Investigating');
        setBulkSelected(new Set());
    };

    const bulkClose = async () => {
        for (const id of bulkSelected()) await act(id, 'Closed');
        setBulkSelected(new Set());
    };

    const toggle = (id: string) => {
        const s = new Set(bulkSelected());
        s.has(id) ? s.delete(id) : s.add(id);
        setBulkSelected(s);
    };

    const countBySev = (sev: string) => incidents().filter(i => (i.severity ?? '').toLowerCase() === sev).length;

    return (
        <div style="padding: 0; height: 100%; overflow-y: auto; background: var(--bg-primary); color: var(--text-primary); font-family: var(--font-ui);">
            <div style="height: var(--header-height); border-bottom: 1px solid var(--glass-border); display: flex; justify-content: space-between; align-items: center; padding: 0 1.5rem; background: var(--bg-secondary);">
                <div style="display: flex; align-items: center; gap: 0.75rem;">
                    <span style="font-size: 16px;">🚨</span>
                    <h2 style="font-size: 13px; letter-spacing: 2px; font-weight: 700; margin: 0; text-transform: uppercase;">Alert Management</h2>
                </div>
                <div style="display: flex; gap: 1.5rem; font-family: var(--font-mono); font-size: 10px;">
                    {['critical', 'high', 'medium', 'low'].map(s => (
                        <span style={`color: ${SEV_COLOR[s]};`}>{countBySev(s)} {s.toUpperCase()}</span>
                    ))}
                </div>
            </div>

            <div style="padding: 1rem 1.5rem; border-bottom: 1px solid var(--glass-border); display: flex; gap: 0.5rem; align-items: center; background: var(--bg-secondary);">
                {['New', 'Active', 'Investigating', 'Closed'].map(f => (
                    <button onClick={() => { setFilter(f); load(); }}
                        style={`padding: 5px 12px; font-size: 10px; font-family: var(--font-mono); font-weight: 700; text-transform: uppercase; letter-spacing: 1px; border-radius: 3px; cursor: pointer; border: 1px solid ${filter() === f ? 'var(--accent-primary)' : 'var(--glass-border)'}; background: ${filter() === f ? 'rgba(87,139,255,0.15)' : 'transparent'}; color: ${filter() === f ? 'var(--accent-primary)' : 'var(--text-muted)'};`}>
                        {f}
                    </button>
                ))}
                <div style="flex: 1;" />
                <Show when={bulkSelected().size > 0}>
                    <button onClick={bulkAck} style="padding: 5px 12px; font-size: 10px; font-family: var(--font-mono); font-weight: 700; text-transform: uppercase; letter-spacing: 1px; border-radius: 3px; cursor: pointer; border: 1px solid rgba(210,153,34,0.4); background: rgba(210,153,34,0.1); color: #d29922;">ACK {bulkSelected().size}</button>
                    <button onClick={bulkClose} style="padding: 5px 12px; font-size: 10px; font-family: var(--font-mono); font-weight: 700; text-transform: uppercase; letter-spacing: 1px; border-radius: 3px; cursor: pointer; border: 1px solid rgba(63,185,80,0.4); background: rgba(63,185,80,0.1); color: #3fb950;">CLOSE {bulkSelected().size}</button>
                </Show>
            </div>

            <Show when={loading()}>
                <div style="padding: 2rem; text-align: center; color: var(--text-muted); font-family: var(--font-mono); font-size: 11px;">LOADING ALERTS...</div>
            </Show>

            <Show when={!loading() && incidents().length === 0}>
                <div style="padding: 4rem; text-align: center; color: var(--text-muted); font-family: var(--font-mono); font-size: 12px;">
                    <div style="font-size: 2.5rem; margin-bottom: 1rem; opacity: 0.3;">✓</div>
                    NO {filter().toUpperCase()} ALERTS
                </div>
            </Show>

            <div style="padding: 0;">
                <For each={incidents()}>
                    {(inc) => {
                        const sev = (inc.severity ?? 'low').toLowerCase();
                        const color = SEV_COLOR[sev] ?? '#6b7280';
                        return (
                            <div style={`border-bottom: 1px solid var(--glass-border); padding: 12px 1.5rem; display: grid; grid-template-columns: 32px 1fr auto; gap: 12px; align-items: start; background: ${bulkSelected().has(inc.id) ? 'rgba(87,139,255,0.05)' : 'transparent'};`}>
                                <input type="checkbox" checked={bulkSelected().has(inc.id)} onChange={() => toggle(inc.id)} style="margin-top: 2px;" />
                                <div>
                                    <div style="display: flex; align-items: center; gap: 8px; margin-bottom: 4px;">
                                        <span style={`padding: 2px 6px; border-radius: 3px; font-size: 9px; font-weight: 800; letter-spacing: 0.5px; background: rgba(${color.slice(1).match(/../g)?.map(h => parseInt(h, 16)).join(',')},0.15); color: ${color};`}>{sev.toUpperCase()}</span>
                                        <span style="font-size: 12px; font-weight: 600; color: var(--text-primary);">{inc.title}</span>
                                    </div>
                                    <div style="font-size: 11px; color: var(--text-secondary); margin-bottom: 4px;">{inc.description}</div>
                                    <div style="display: flex; gap: 1rem; font-size: 10px; color: var(--text-muted); font-family: var(--font-mono);">
                                        <span>RULE: {inc.rule_id}</span>
                                        <span>ENTITY: {inc.group_key}</span>
                                        <span>COUNT: {inc.event_count}</span>
                                    </div>
                                </div>
                                <div style="display: flex; gap: 6px; align-items: flex-start; white-space: nowrap;">
                                    <Show when={inc.status !== 'Investigating' && inc.status !== 'Closed'}>
                                        <button onClick={() => act(inc.id, 'Investigating')} style="padding: 4px 10px; font-size: 9px; font-family: var(--font-mono); font-weight: 700; text-transform: uppercase; letter-spacing: 0.5px; border-radius: 3px; cursor: pointer; border: 1px solid var(--glass-border); background: transparent; color: var(--text-muted);">INVESTIGATE</button>
                                    </Show>
                                    <Show when={inc.status !== 'Closed'}>
                                        <button onClick={() => act(inc.id, 'Closed')} style="padding: 4px 10px; font-size: 9px; font-family: var(--font-mono); font-weight: 700; text-transform: uppercase; letter-spacing: 0.5px; border-radius: 3px; cursor: pointer; border: 1px solid rgba(63,185,80,0.4); background: rgba(63,185,80,0.1); color: #3fb950;">CLOSE</button>
                                    </Show>
                                    <Show when={inc.status === 'Closed'}>
                                        <button onClick={() => act(inc.id, 'New')} style="padding: 4px 10px; font-size: 9px; font-family: var(--font-mono); font-weight: 700; text-transform: uppercase; letter-spacing: 0.5px; border-radius: 3px; cursor: pointer; border: 1px solid var(--glass-border); background: transparent; color: var(--text-muted);">REOPEN</button>
                                    </Show>
                                </div>
                            </div>
                        );
                    }}
                </For>
            </div>
        </div>
    );
};
