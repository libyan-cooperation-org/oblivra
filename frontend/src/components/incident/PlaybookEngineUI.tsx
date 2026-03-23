// PlaybookEngineUI.tsx — Phase 8 Web: visual automation builder
import { Component, createSignal, onMount, For, Show } from 'solid-js';
import { IS_BROWSER } from '@core/context';

interface Step {
    id: string;
    action: string;
    params: Record<string, string>;
    enabled: boolean;
}

const STEP_ICONS: Record<string, string> = {
    isolate_host: '🔒', lock_account: '👤', kill_process: '⚡',
    collect_logs: '📋', snapshot_memory: '💾', notify_team: '📢',
    run_scan: '🔍', block_ip: '🚫', default: '⚙️',
};

export const PlaybookEngineUI: Component = () => {
    const [availableActions, setAvailableActions] = createSignal<string[]>([]);
    const [steps, setSteps] = createSignal<Step[]>([]);
    const [name, setName] = createSignal('');
    const [targetIncident, setTargetIncident] = createSignal('');
    const [running, setRunning] = createSignal(false);
    const [result, setResult] = createSignal('');
    const [dragging, setDragging] = createSignal<number | null>(null);

    onMount(async () => {
        if (IS_BROWSER) return;
        try {
            const { ListAvailableActions } = await import('../../../wailsjs/go/services/PlaybookService') as any;
            setAvailableActions(await ListAvailableActions() ?? []);
        } catch { }
    });

    const addStep = (action: string) => {
        setSteps(s => [...s, {
            id: `step-${Date.now()}`,
            action,
            params: {},
            enabled: true,
        }]);
    };

    const removeStep = (id: string) => setSteps(s => s.filter(x => x.id !== id));

    const toggleStep = (id: string) =>
        setSteps(s => s.map(x => x.id === id ? { ...x, enabled: !x.enabled } : x));

    const moveUp = (idx: number) => {
        if (idx === 0) return;
        setSteps(s => { const a = [...s]; [a[idx - 1], a[idx]] = [a[idx], a[idx - 1]]; return a; });
    };
    const moveDown = (idx: number) => {
        const s = steps();
        if (idx >= s.length - 1) return;
        setSteps(a => { const b = [...a]; [b[idx], b[idx + 1]] = [b[idx + 1], b[idx]]; return b; });
    };

    const executePlaybook = async () => {
        if (!targetIncident().trim()) { setResult('✗ Enter a target incident ID first.'); return; }
        if (steps().length === 0) { setResult('✗ Add at least one step.'); return; }
        setRunning(true);
        setResult('');
        try {
            const playbookID = name() || 'custom-playbook';
            const { RunPlaybook } = await import('../../../wailsjs/go/services/PlaybookService') as any;
            await RunPlaybook('', playbookID, targetIncident());
            setResult(`✓ Playbook "${playbookID}" executed against incident ${targetIncident()}`);
        } catch (e: any) {
            setResult('✗ ' + (e?.message ?? e));
        }
        setRunning(false);
    };

    return (
        <div style="padding: 0; height: 100%; background: var(--bg-primary); color: var(--text-primary); font-family: var(--font-ui); display: flex; flex-direction: column; overflow: hidden;">
            <div style="height: var(--header-height); border-bottom: 1px solid var(--glass-border); display: flex; align-items: center; gap: 0.75rem; padding: 0 1.5rem; background: var(--bg-secondary); flex-shrink: 0;">
                <span style="font-size: 16px;">🔧</span>
                <h2 style="font-size: 13px; letter-spacing: 2px; font-weight: 700; margin: 0; text-transform: uppercase;">Playbook Engine</h2>
            </div>

            <div style="flex: 1; display: grid; grid-template-columns: 240px 1fr 280px; gap: 0; overflow: hidden;">

                {/* Action palette */}
                <div style="border-right: 1px solid var(--glass-border); overflow-y: auto; padding: 1rem;">
                    <div style="font-size: 9px; text-transform: uppercase; letter-spacing: 1.5px; color: var(--text-muted); font-family: var(--font-mono); margin-bottom: 0.75rem;">Available Actions</div>
                    <Show when={availableActions().length === 0}>
                        <div style="color: var(--text-muted); font-size: 10px; font-family: var(--font-mono); padding: 0.5rem 0;">No actions available</div>
                    </Show>
                    <For each={availableActions()}>
                        {(action) => (
                            <div
                                onClick={() => addStep(action)}
                                style="padding: 8px 10px; border: 1px solid var(--glass-border); border-radius: 4px; margin-bottom: 6px; cursor: pointer; font-family: var(--font-mono); font-size: 10px; display: flex; align-items: center; gap: 8px; background: var(--bg-secondary);"
                                onMouseEnter={e => (e.currentTarget as HTMLElement).style.borderColor = 'rgba(87,139,255,0.5)'}
                                onMouseLeave={e => (e.currentTarget as HTMLElement).style.borderColor = 'var(--glass-border)'}
                            >
                                <span>{STEP_ICONS[action] ?? STEP_ICONS.default}</span>
                                <span style="color: var(--text-primary);">{action.replace(/_/g, ' ')}</span>
                                <span style="margin-left: auto; color: var(--accent-primary); font-size: 12px; font-weight: 700;">+</span>
                            </div>
                        )}
                    </For>
                </div>

                {/* Canvas */}
                <div style="overflow-y: auto; padding: 1.5rem; display: flex; flex-direction: column; gap: 0.75rem;">
                    <div style="display: flex; gap: 0.75rem; align-items: center; margin-bottom: 0.5rem;">
                        <input placeholder="Playbook name..." value={name()} onInput={e => setName((e.target as HTMLInputElement).value)}
                            style="background: var(--bg-secondary); border: 1px solid var(--glass-border); color: var(--text-primary); padding: 7px 10px; border-radius: 4px; font-family: var(--font-mono); font-size: 11px; flex: 1;" />
                        <span style="color: var(--text-muted); font-size: 10px; font-family: var(--font-mono);">{steps().length} STEPS</span>
                    </div>

                    <Show when={steps().length === 0}>
                        <div style="border: 2px dashed rgba(255,255,255,0.1); border-radius: 8px; padding: 3rem; text-align: center; color: var(--text-muted); font-family: var(--font-mono); font-size: 11px; letter-spacing: 1px;">
                            DRAG ACTIONS FROM THE LEFT PANEL<br/>OR CLICK TO ADD STEPS
                        </div>
                    </Show>

                    <For each={steps()}>
                        {(step, idx) => (
                            <div style={`background: var(--bg-secondary); border: 1px solid ${step.enabled ? 'var(--glass-border)' : 'rgba(255,255,255,0.05)'}; border-radius: 6px; padding: 12px; display: flex; align-items: center; gap: 10px; opacity: ${step.enabled ? 1 : 0.45};`}>
                                <div style="display: flex; flex-direction: column; gap: 3px;">
                                    <button onClick={() => moveUp(idx())} style="background: transparent; border: none; color: var(--text-muted); cursor: pointer; font-size: 12px; padding: 0; line-height: 1;">▲</button>
                                    <button onClick={() => moveDown(idx())} style="background: transparent; border: none; color: var(--text-muted); cursor: pointer; font-size: 12px; padding: 0; line-height: 1;">▼</button>
                                </div>
                                <div style="width: 24px; height: 24px; border-radius: 50%; background: rgba(87,139,255,0.15); display: flex; align-items: center; justify-content: center; font-size: 10px; color: var(--text-muted); font-family: var(--font-mono); font-weight: 700; flex-shrink: 0;">{idx() + 1}</div>
                                <span style="font-size: 16px;">{STEP_ICONS[step.action] ?? STEP_ICONS.default}</span>
                                <div style="flex: 1;">
                                    <div style="font-size: 11px; font-weight: 700; font-family: var(--font-mono); color: var(--text-primary);">{step.action.replace(/_/g, ' ').toUpperCase()}</div>
                                </div>
                                <button onClick={() => toggleStep(step.id)} style={`padding: 3px 8px; font-size: 9px; font-family: var(--font-mono); font-weight: 700; text-transform: uppercase; letter-spacing: 0.5px; border-radius: 3px; cursor: pointer; border: 1px solid ${step.enabled ? 'rgba(63,185,80,0.4)' : 'var(--glass-border)'}; background: ${step.enabled ? 'rgba(63,185,80,0.1)' : 'transparent'}; color: ${step.enabled ? '#3fb950' : 'var(--text-muted)'};`}>
                                    {step.enabled ? 'ON' : 'OFF'}
                                </button>
                                <button onClick={() => removeStep(step.id)} style="background: transparent; border: none; color: var(--text-muted); cursor: pointer; font-size: 16px; padding: 0; line-height: 1;">✕</button>
                            </div>
                        )}
                    </For>
                </div>

                {/* Execute panel */}
                <div style="border-left: 1px solid var(--glass-border); padding: 1.5rem; display: flex; flex-direction: column; gap: 1rem; overflow-y: auto;">
                    <div style="font-size: 9px; text-transform: uppercase; letter-spacing: 1.5px; color: var(--text-muted); font-family: var(--font-mono);">Execute Against</div>
                    <input placeholder="Incident ID..." value={targetIncident()} onInput={e => setTargetIncident((e.target as HTMLInputElement).value)}
                        style="background: var(--bg-secondary); border: 1px solid var(--glass-border); color: var(--text-primary); padding: 7px 10px; border-radius: 4px; font-family: var(--font-mono); font-size: 11px; width: 100%; box-sizing: border-box;" />
                    <button onClick={executePlaybook} disabled={running()}
                        style={`padding: 10px; background: ${running() ? 'rgba(87,139,255,0.1)' : 'rgba(87,139,255,0.2)'}; border: 1px solid rgba(87,139,255,0.4); color: var(--accent-primary); border-radius: 4px; cursor: pointer; font-family: var(--font-mono); font-size: 11px; font-weight: 700; text-transform: uppercase; letter-spacing: 1px; width: 100%;`}>
                        {running() ? '⏳ EXECUTING...' : '▶ RUN PLAYBOOK'}
                    </button>
                    <Show when={result()}>
                        <div style={`padding: 10px; border-radius: 4px; font-family: var(--font-mono); font-size: 11px; background: ${result().startsWith('✓') ? 'rgba(63,185,80,0.1)' : 'rgba(248,81,73,0.1)'}; border: 1px solid ${result().startsWith('✓') ? 'rgba(63,185,80,0.3)' : 'rgba(248,81,73,0.3)'}; color: ${result().startsWith('✓') ? '#3fb950' : '#f85149'}; line-height: 1.5;`}>
                            {result()}
                        </div>
                    </Show>
                    <div style="margin-top: auto; padding: 1rem; background: rgba(255,255,255,0.03); border-radius: 4px; border: 1px solid var(--glass-border);">
                        <div style="font-size: 9px; text-transform: uppercase; letter-spacing: 1px; color: var(--text-muted); font-family: var(--font-mono); margin-bottom: 0.5rem;">Summary</div>
                        <div style="font-size: 10px; font-family: var(--font-mono); color: var(--text-secondary); display: flex; flex-direction: column; gap: 3px;">
                            <span>Steps: {steps().length}</span>
                            <span>Active: {steps().filter(s => s.enabled).length}</span>
                            <span>Disabled: {steps().filter(s => !s.enabled).length}</span>
                        </div>
                    </div>
                </div>
            </div>
        </div>
    );
};
