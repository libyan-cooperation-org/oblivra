// EscalationCenter.tsx — Phase 8 Web: incident triage and workflow
import { Component, createSignal, onMount, For, Show } from 'solid-js';
import * as IncidentService from '../../../wailsjs/go/services/IncidentService';
import * as PlaybookService from '../../../wailsjs/go/services/PlaybookService';

export const EscalationCenter: Component = () => {
    const [incidents, setIncidents] = createSignal<any[]>([]);
    const [playbooks, setPlaybooks] = createSignal<string[]>([]);
    const [selected, setSelected] = createSignal<any>(null);
    const [timeline, setTimeline] = createSignal<any[]>([]);
    const [evidence, setEvidence] = createSignal<any[]>([]);
    const [loading, setLoading] = createSignal(true);
    const [runningPlaybook, setRunningPlaybook] = createSignal('');
    const [playbookResult, setPlaybookResult] = createSignal('');

    onMount(async () => {
        try {
            const [inc, pb] = await Promise.all([
                // ListIncidents(ctx, status, owner, limit) — pass ctx as empty string
                (IncidentService as any).ListIncidents('', '', '', 100),
                (PlaybookService as any).ListAvailableActions(),
            ]);
            setIncidents(inc ?? []);
            setPlaybooks(pb ?? []);
        } catch { }
        setLoading(false);
    });

    const selectIncident = async (inc: any) => {
        setSelected(inc);
        try {
            // GetTimeline(ctx, id) and GetEvidence(ctx, id) — pass ctx as empty string
            const [tl, ev] = await Promise.all([
                (IncidentService as any).GetTimeline('', inc.id),
                (IncidentService as any).GetEvidence('', inc.id),
            ]);
            setTimeline(tl ?? []);
            setEvidence(ev ?? []);
        } catch {
            setTimeline([]);
            setEvidence([]);
        }
    };

    const runPlaybook = async (playbookID: string) => {
        if (!selected()) return;
        setRunningPlaybook(playbookID);
        setPlaybookResult('');
        try {
            await (PlaybookService as any).RunPlaybook('', playbookID, selected().id);
            setPlaybookResult('✓ Playbook executed');
        } catch (e: any) {
            setPlaybookResult('✗ ' + (e?.message ?? e));
        }
        setRunningPlaybook('');
        setTimeout(() => setPlaybookResult(''), 4000);
    };

    const SEV_COLORS: Record<string, string> = { critical: '#f85149', high: '#f0883e', medium: '#d29922', low: '#3fb950' };

    return (
        <div style="padding: 0; height: 100%; background: var(--bg-primary); color: var(--text-primary); font-family: var(--font-ui); display: flex; flex-direction: column; overflow: hidden;">
            <div style="height: var(--header-height); border-bottom: 1px solid var(--glass-border); display: flex; align-items: center; gap: 0.75rem; padding: 0 1.5rem; background: var(--bg-secondary); flex-shrink: 0;">
                <span style="font-size: 16px;">⚡</span>
                <h2 style="font-size: 13px; letter-spacing: 2px; font-weight: 700; margin: 0; text-transform: uppercase;">Escalation Center</h2>
                <span style="margin-left: auto; font-size: 10px; color: var(--text-muted); font-family: var(--font-mono);">{incidents().length} INCIDENTS</span>
            </div>

            <div style="flex: 1; display: grid; grid-template-columns: 360px 1fr; overflow: hidden;">
                {/* Incident list */}
                <div style="border-right: 1px solid var(--glass-border); overflow-y: auto;">
                    <Show when={loading()}>
                        <div style="padding: 2rem; text-align: center; color: var(--text-muted); font-family: var(--font-mono); font-size: 11px;">LOADING...</div>
                    </Show>
                    <For each={incidents()}>
                        {(inc) => {
                            const sev = (inc.severity ?? 'low').toLowerCase();
                            const color = SEV_COLORS[sev] ?? '#6b7280';
                            const isSelected = selected()?.id === inc.id;
                            return (
                                <div
                                    onClick={() => selectIncident(inc)}
                                    style={`padding: 12px 1rem; border-bottom: 1px solid var(--glass-border); cursor: pointer; background: ${isSelected ? 'rgba(87,139,255,0.08)' : 'transparent'}; border-left: 3px solid ${isSelected ? 'var(--accent-primary)' : color};`}
                                >
                                    <div style="display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 4px;">
                                        <span style="font-size: 11px; font-weight: 600; color: var(--text-primary);">{inc.title}</span>
                                        <span style={`font-size: 9px; font-weight: 700; padding: 1px 5px; border-radius: 2px; background: rgba(0,0,0,0.3); color: ${color};`}>{sev.toUpperCase()}</span>
                                    </div>
                                    <div style="font-size: 10px; color: var(--text-muted); font-family: var(--font-mono);">{inc.status} · {inc.group_key}</div>
                                </div>
                            );
                        }}
                    </For>
                </div>

                {/* Detail panel */}
                <div style="overflow-y: auto; padding: 1.5rem; display: flex; flex-direction: column; gap: 1.25rem;">
                    <Show when={!selected()}>
                        <div style="flex: 1; display: flex; align-items: center; justify-content: center; color: var(--text-muted); font-family: var(--font-mono); font-size: 11px; letter-spacing: 1px; flex-direction: column; gap: 1rem; opacity: 0.5;">
                            <span style="font-size: 3rem;">⚡</span>
                            SELECT AN INCIDENT TO TRIAGE
                        </div>
                    </Show>

                    <Show when={selected()}>
                        {/* Header */}
                        <div style="background: var(--bg-secondary); border: 1px solid var(--glass-border); border-radius: 6px; padding: 1.25rem;">
                            <div style="font-size: 16px; font-weight: 700; color: var(--text-primary); margin-bottom: 8px;">{selected()?.title}</div>
                            <div style="font-size: 12px; color: var(--text-secondary); margin-bottom: 10px;">{selected()?.description}</div>
                            <div style="display: flex; gap: 1.5rem; font-size: 10px; font-family: var(--font-mono); color: var(--text-muted);">
                                <span>STATUS: {selected()?.status}</span>
                                <span>EVENTS: {selected()?.event_count}</span>
                                <span>RULE: {selected()?.rule_id}</span>
                            </div>
                        </div>

                        {/* Playbooks */}
                        <Show when={playbooks().length > 0}>
                            <div style="background: var(--bg-secondary); border: 1px solid var(--glass-border); border-radius: 6px; padding: 1.25rem;">
                                <div style="font-size: 10px; text-transform: uppercase; letter-spacing: 1px; color: var(--text-muted); font-family: var(--font-mono); margin-bottom: 0.75rem;">Execute Playbook</div>
                                <div style="display: flex; flex-wrap: wrap; gap: 0.5rem;">
                                    <For each={playbooks()}>
                                        {(pb) => (
                                            <button onClick={() => runPlaybook(pb)} disabled={!!runningPlaybook()}
                                                style={`padding: 6px 12px; font-size: 10px; font-family: var(--font-mono); font-weight: 700; text-transform: uppercase; letter-spacing: 0.5px; border-radius: 3px; cursor: pointer; border: 1px solid rgba(87,139,255,0.3); background: ${runningPlaybook() === pb ? 'rgba(87,139,255,0.2)' : 'rgba(87,139,255,0.08)'}; color: var(--accent-primary);`}>
                                                {runningPlaybook() === pb ? '⏳ ' : ''}{pb.replace(/_/g, ' ')}
                                            </button>
                                        )}
                                    </For>
                                </div>
                                <Show when={playbookResult()}>
                                    <div style={`margin-top: 0.75rem; font-family: var(--font-mono); font-size: 11px; color: ${playbookResult().startsWith('✓') ? '#3fb950' : '#f85149'};`}>{playbookResult()}</div>
                                </Show>
                            </div>
                        </Show>

                        {/* Timeline */}
                        <Show when={timeline().length > 0}>
                            <div style="background: var(--bg-secondary); border: 1px solid var(--glass-border); border-radius: 6px; padding: 1.25rem;">
                                <div style="font-size: 10px; text-transform: uppercase; letter-spacing: 1px; color: var(--text-muted); font-family: var(--font-mono); margin-bottom: 0.75rem;">Timeline ({timeline().length})</div>
                                <div style="display: flex; flex-direction: column; gap: 0.5rem; max-height: 200px; overflow-y: auto;">
                                    <For each={timeline()}>
                                        {(entry: any) => (
                                            <div style="display: flex; gap: 10px; font-size: 10px; font-family: var(--font-mono);">
                                                <span style="color: var(--text-muted); white-space: nowrap; flex-shrink: 0;">{entry.timestamp?.slice(0, 16)?.replace('T', ' ')}</span>
                                                <span style="color: var(--text-secondary);">{entry.event_type ?? entry.details}</span>
                                            </div>
                                        )}
                                    </For>
                                </div>
                            </div>
                        </Show>

                        {/* Evidence */}
                        <Show when={evidence().length > 0}>
                            <div style="background: var(--bg-secondary); border: 1px solid var(--glass-border); border-radius: 6px; padding: 1.25rem;">
                                <div style="font-size: 10px; text-transform: uppercase; letter-spacing: 1px; color: var(--text-muted); font-family: var(--font-mono); margin-bottom: 0.75rem;">Evidence ({evidence().length})</div>
                                <For each={evidence()}>
                                    {(ev: any) => (
                                        <div style="padding: 6px 0; border-bottom: 1px solid rgba(255,255,255,0.04); font-size: 10px; font-family: var(--font-mono);">
                                            <span style="color: var(--accent-primary);">{ev.type ?? 'FILE'}</span>
                                            <span style="color: var(--text-secondary); margin-left: 8px;">{ev.name ?? ev.description}</span>
                                        </div>
                                    )}
                                </For>
                            </div>
                        </Show>
                    </Show>
                </div>
            </div>
        </div>
    );
};
