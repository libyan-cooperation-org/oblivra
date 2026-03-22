/**
 * PlaybookBuilder.tsx — Visual Playbook Automation Builder (Web Only — Phase 8)
 *
 * Drag-and-reorder step builder for SOAR playbooks.
 * Connects to /api/v1/playbooks/*.
 */

import { createSignal, createResource, For, Show } from 'solid-js';
import { request } from '../services/api';

interface PlaybookStep {
    id: string;
    action: string;
    params: Record<string, string>;
    enabled: boolean;
}

interface SavedPlaybook {
    id: string;
    name: string;
    steps: PlaybookStep[];
    created_at: string;
    last_run?: string;
}

const STEP_ICONS: Record<string, string> = {
    isolate_host: '🔒', lock_account: '👤', kill_process: '⚡',
    collect_logs: '📋', snapshot_memory: '💾', notify_team: '📢',
    run_scan: '🔍', block_ip: '🚫', webhook: '🌐', default: '⚙️',
};

async function fetchAvailableActions(): Promise<string[]> {
    try {
        const r = await request<{ actions: string[] }>('/playbooks/actions');
        return r.actions ?? [];
    } catch { return ['isolate_host', 'lock_account', 'kill_process', 'collect_logs', 'notify_team', 'block_ip', 'webhook']; }
}

async function fetchPlaybooks(): Promise<SavedPlaybook[]> {
    try {
        const r = await request<{ playbooks: SavedPlaybook[] }>('/playbooks');
        return r.playbooks ?? [];
    } catch { return []; }
}

export default function PlaybookBuilder() {
    const [availableActions] = createResource(fetchAvailableActions);
    const [savedPlaybooks, { refetch }] = createResource(fetchPlaybooks);

    const [steps, setSteps] = createSignal<PlaybookStep[]>([]);
    const [name, setName] = createSignal('');
    const [targetIncident, setTargetIncident] = createSignal('');
    const [running, setRunning] = createSignal(false);
    const [saving, setSaving] = createSignal(false);
    const [result, setResult] = createSignal('');

    const addStep = (action: string) => setSteps(s => [...s, {
        id: `step-${Date.now()}`,
        action, params: {}, enabled: true,
    }]);

    const removeStep = (id: string) => setSteps(s => s.filter(x => x.id !== id));
    const toggleStep = (id: string) => setSteps(s => s.map(x => x.id === id ? { ...x, enabled: !x.enabled } : x));

    const moveUp = (idx: number) => { if (idx === 0) return; setSteps(s => { const a = [...s]; [a[idx-1], a[idx]] = [a[idx], a[idx-1]]; return a; }); };
    const moveDown = (idx: number) => { setSteps(s => { if (idx >= s.length - 1) return s; const a = [...s]; [a[idx], a[idx+1]] = [a[idx+1], a[idx]]; return a; }); };

    const savePlaybook = async () => {
        if (!name().trim() || steps().length === 0) { setResult('✗ Name and at least one step required.'); return; }
        setSaving(true);
        try {
            await request('/playbooks', { method: 'POST', body: JSON.stringify({ name: name(), steps: steps() }) });
            setResult('✓ Playbook saved.');
            refetch();
        } catch (e: any) { setResult('✗ ' + (e?.message ?? e)); }
        setSaving(false);
        setTimeout(() => setResult(''), 4000);
    };

    const executePlaybook = async () => {
        if (!targetIncident().trim()) { setResult('✗ Enter a target incident ID.'); return; }
        if (steps().length === 0) { setResult('✗ Add at least one step.'); return; }
        setRunning(true);
        try {
            await request('/playbooks/run', { method: 'POST', body: JSON.stringify({
                name: name() || 'adhoc-playbook',
                steps: steps().filter(s => s.enabled),
                incident_id: targetIncident(),
            })});
            setResult(`✓ Playbook executed against ${targetIncident()}`);
        } catch (e: any) { setResult('✗ ' + (e?.message ?? e)); }
        setRunning(false);
        setTimeout(() => setResult(''), 5000);
    };

    return (
        <div style="padding:2rem; color:#c8d8d8; font-family:'JetBrains Mono',monospace; min-height:100vh; background:#080f12;">
            <div style="margin-bottom:1.5rem;">
                <h1 style="font-size:1.4rem; letter-spacing:0.15em; margin:0; color:#ff6600;">⬡ PLAYBOOK BUILDER</h1>
                <p style="margin:0.25rem 0 0; font-size:0.75rem; color:#607070;">Visual SOAR automation · Assemble, save, and execute response playbooks</p>
            </div>

            <div style="display:grid; grid-template-columns:220px 1fr 260px; gap:1rem; height:calc(100vh - 120px);">
                {/* Action palette */}
                <div style="background:#0d1a1f; border:1px solid #1e3040; border-radius:6px; padding:1rem; overflow-y:auto;">
                    <div style="font-size:0.65rem; color:#607070; letter-spacing:0.12em; margin-bottom:0.75rem;">AVAILABLE ACTIONS</div>
                    <Show when={!availableActions.loading}>
                        <For each={availableActions()}>
                            {(action) => (
                                <div onClick={() => addStep(action)} style="padding:8px 10px; border:1px solid #1e3040; border-radius:3px; margin-bottom:6px; cursor:pointer; font-size:0.76rem; display:flex; align-items:center; gap:8px; background:#0a1318; transition:border-color 0.15s;"
                                    onMouseEnter={e => (e.currentTarget as HTMLElement).style.borderColor = '#ff6600'}
                                    onMouseLeave={e => (e.currentTarget as HTMLElement).style.borderColor = '#1e3040'}>
                                    <span>{STEP_ICONS[action] ?? STEP_ICONS.default}</span>
                                    <span>{action.replace(/_/g,' ')}</span>
                                    <span style="margin-left:auto; color:#ff6600; font-weight:700;">+</span>
                                </div>
                            )}
                        </For>
                    </Show>
                    <div style="margin-top:1.5rem; border-top:1px solid #1e3040; padding-top:1rem;">
                        <div style="font-size:0.65rem; color:#607070; letter-spacing:0.12em; margin-bottom:0.5rem;">SAVED PLAYBOOKS</div>
                        <Show when={!savedPlaybooks.loading}>
                            <For each={savedPlaybooks()} fallback={<div style="color:#607070; font-size:0.72rem;">None saved yet.</div>}>
                                {(pb) => (
                                    <div style="padding:6px 8px; background:#0a1318; border:1px solid #1e3040; border-radius:3px; margin-bottom:4px; cursor:pointer; font-size:0.72rem;"
                                        onClick={() => { setName(pb.name); setSteps(pb.steps); }}
                                        onMouseEnter={e => (e.currentTarget as HTMLElement).style.background = '#111f28'}
                                        onMouseLeave={e => (e.currentTarget as HTMLElement).style.background = '#0a1318'}>
                                        <div style="color:#c8d8d8;">{pb.name}</div>
                                        <div style="color:#607070; font-size:0.65rem;">{pb.steps.length} steps</div>
                                    </div>
                                )}
                            </For>
                        </Show>
                    </div>
                </div>

                {/* Canvas */}
                <div style="background:#0d1a1f; border:1px solid #1e3040; border-radius:6px; padding:1rem; overflow-y:auto; display:flex; flex-direction:column; gap:0.75rem;">
                    <input placeholder="Playbook name…" value={name()} onInput={e => setName(e.currentTarget.value)}
                        style="background:#0a1318; border:1px solid #1e3040; color:#c8d8d8; padding:8px 10px; border-radius:3px; font-size:0.82rem; font-family:inherit;" />
                    <Show when={steps().length === 0}>
                        <div style="border:2px dashed #1e3040; border-radius:6px; padding:3rem; text-align:center; color:#607070; font-size:0.76rem; letter-spacing:0.1em;">
                            CLICK ACTIONS IN THE LEFT PANEL TO BUILD YOUR PLAYBOOK
                        </div>
                    </Show>
                    <For each={steps()}>
                        {(step, idx) => (
                            <div style={`background:#0a1318; border:1px solid ${step.enabled ? '#1e3040' : '#0d1a1f'}; border-radius:4px; padding:10px; display:flex; align-items:center; gap:10px; opacity:${step.enabled ? 1 : 0.4};`}>
                                <div style="display:flex; flex-direction:column; gap:2px;">
                                    <button onClick={() => moveUp(idx())} style="background:none; border:none; color:#607070; cursor:pointer; font-size:11px; line-height:1; padding:0;">▲</button>
                                    <button onClick={() => moveDown(idx())} style="background:none; border:none; color:#607070; cursor:pointer; font-size:11px; line-height:1; padding:0;">▼</button>
                                </div>
                                <div style="width:22px; height:22px; border-radius:50%; background:#1e3040; display:flex; align-items:center; justify-content:center; font-size:9px; color:#607070; font-weight:700; flex-shrink:0;">{idx()+1}</div>
                                <span style="font-size:14px;">{STEP_ICONS[step.action] ?? STEP_ICONS.default}</span>
                                <span style="flex:1; font-size:0.78rem; color:#c8d8d8;">{step.action.replace(/_/g,' ').toUpperCase()}</span>
                                <button onClick={() => toggleStep(step.id)}
                                    style={`padding:2px 7px; font-size:0.68rem; letter-spacing:0.1em; border-radius:2px; cursor:pointer; border:1px solid ${step.enabled ? '#00ff88' : '#1e3040'}; background:none; color:${step.enabled ? '#00ff88' : '#607070'};`}>
                                    {step.enabled ? 'ON' : 'OFF'}
                                </button>
                                <button onClick={() => removeStep(step.id)} style="background:none; border:none; color:#607070; cursor:pointer; font-size:15px; padding:0; line-height:1;">✕</button>
                            </div>
                        )}
                    </For>
                </div>

                {/* Execute panel */}
                <div style="background:#0d1a1f; border:1px solid #1e3040; border-radius:6px; padding:1rem; display:flex; flex-direction:column; gap:0.75rem; overflow-y:auto;">
                    <div style="font-size:0.65rem; color:#607070; letter-spacing:0.12em;">EXECUTE AGAINST</div>
                    <input placeholder="Incident ID…" value={targetIncident()} onInput={e => setTargetIncident(e.currentTarget.value)}
                        style="background:#0a1318; border:1px solid #1e3040; color:#c8d8d8; padding:8px 10px; border-radius:3px; font-size:0.78rem; font-family:inherit; width:100%; box-sizing:border-box;" />
                    <button onClick={executePlaybook} disabled={running()}
                        style={`width:100%; background:${running() ? 'rgba(255,102,0,0.1)' : '#ff6600'}; border:1px solid #ff6600; color:${running() ? '#ff6600' : '#080f12'}; padding:9px; border-radius:3px; cursor:pointer; font-weight:700; font-size:0.78rem; letter-spacing:0.1em;`}>
                        {running() ? '⏳ EXECUTING…' : '▶ RUN PLAYBOOK'}
                    </button>
                    <button onClick={savePlaybook} disabled={saving()}
                        style="width:100%; background:none; border:1px solid #1e3040; color:#607070; padding:8px; border-radius:3px; cursor:pointer; font-size:0.78rem; letter-spacing:0.1em;">
                        {saving() ? '⏳ SAVING…' : '💾 SAVE PLAYBOOK'}
                    </button>
                    <Show when={result()}>
                        <div style={`padding:8px 10px; border-radius:3px; font-size:0.76rem; background:${result().startsWith('✓') ? '#002a1a' : '#2a0d15'}; border:1px solid ${result().startsWith('✓') ? '#00ff88' : '#ff3355'}; color:${result().startsWith('✓') ? '#00ff88' : '#ff3355'}; line-height:1.5;`}>{result()}</div>
                    </Show>
                    <div style="margin-top:auto; padding:0.75rem; background:#0a1318; border-radius:3px; border:1px solid #1e3040;">
                        <div style="font-size:0.65rem; color:#607070; letter-spacing:0.1em; margin-bottom:0.4rem;">SUMMARY</div>
                        <div style="font-size:0.72rem; color:#c8d8d8; display:flex; flex-direction:column; gap:3px;">
                            <span>Total steps: {steps().length}</span>
                            <span>Active: {steps().filter(s => s.enabled).length}</span>
                            <span>Disabled: {steps().filter(s => !s.enabled).length}</span>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    );
}
