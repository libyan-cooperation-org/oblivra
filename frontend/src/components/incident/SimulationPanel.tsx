import { Component, createSignal, For, Show } from 'solid-js';
import { ListScenarios, RunScenario } from '../../../wailsjs/go/simulation/SimulationService';
import { ListHosts } from '../../../wailsjs/go/app/HostService';

export const SimulationPanel: Component = () => {
    const [scenarios, setScenarios] = createSignal<any[]>([]);
    const [hosts, setHosts] = createSignal<any[]>([]);
    const [selectedScenario, setSelectedScenario] = createSignal("");
    const [selectedHost, setSelectedHost] = createSignal("");
    const [status, setStatus] = createSignal<{ type: 'info' | 'success' | 'error', msg: string } | null>(null);

    const loadData = async () => {
        try {
            const [s, h] = await Promise.all([ListScenarios(), ListHosts()]);
            setScenarios(s || []);
            setHosts(h || []);
        } catch (err) {
            console.error("Failed to load simulation data:", err);
        }
    };

    loadData();

    const handleRun = async () => {
        if (!selectedScenario() || !selectedHost()) {
            setStatus({ type: 'error', msg: "Select both scenario and target host" });
            return;
        }

        try {
            setStatus({ type: 'info', msg: `Triggering ${selectedScenario()}...` });
            await RunScenario(selectedScenario(), selectedHost());
            setStatus({ type: 'success', msg: "Simulation event published to bus" });
        } catch (err: any) {
            setStatus({ type: 'error', msg: err.toString() });
        }
    };

    return (
        <div class="ob-card page-enter" style="padding: 32px; background: rgba(0,0,0,0.4);">
            <h2 style="font-size: 20px; font-weight: 800; color: var(--status-offline); margin: 0 0 32px 0; display: flex; align-items: center; gap: 12px; letter-spacing: -0.5px; text-transform: uppercase;">
                <svg style="width: 24px; height: 24px;" fill="none" viewBox="0 0 24 24" stroke="currentColor">
                    <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z" />
                </svg>
                Threat Simulation Center
            </h2>

            <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 48px;">
                <div>
                    <label style="display: block; color: var(--text-muted); font-size: 10px; font-weight: 800; text-transform: uppercase; margin-bottom: 12px; letter-spacing: 1px;">Select Scenario</label>
                    <div style="display: flex; flex-direction: column; gap: 12px;">
                        <For each={scenarios()}>
                            {(s) => (
                                <button
                                    onClick={() => setSelectedScenario(s.id)}
                                    style={`width: 100%; text-align: left; padding: 20px; border-radius: 8px; border: 1px solid ${selectedScenario() === s.id ? 'var(--status-offline)' : 'var(--border-primary)'}; background: ${selectedScenario() === s.id ? 'rgba(239, 68, 68, 0.1)' : 'rgba(0,0,0,0.2)'}; color: ${selectedScenario() === s.id ? 'var(--text-primary)' : 'var(--text-muted)'}; transition: all 120ms ease; cursor: pointer;`}
                                    onMouseEnter={(e) => { if (selectedScenario() !== s.id) e.currentTarget.style.borderColor = 'rgba(255,255,255,0.2)'; }}
                                    onMouseLeave={(e) => { if (selectedScenario() !== s.id) e.currentTarget.style.borderColor = 'var(--border-primary)'; }}
                                >
                                    <div style="font-weight: 800; font-size: 14px; margin-bottom: 4px;">{s.name}</div>
                                    <div style="font-size: 11px; opacity: 0.7; line-height: 1.5;">{s.description}</div>
                                </button>
                            )}
                        </For>
                    </div>
                </div>

                <div style="display: flex; flex-direction: column; justify-content: space-between;">
                    <div>
                        <label style="display: block; color: var(--text-muted); font-size: 10px; font-weight: 800; text-transform: uppercase; margin-bottom: 12px; letter-spacing: 1px;">Target Entity</label>
                        <select
                            value={selectedHost()}
                            onInput={(e) => setSelectedHost(e.currentTarget.value)}
                            class="ob-input"
                            style="width: 100%; padding: 16px; font-family: var(--font-mono); font-size: 12px; background: rgba(0,0,0,0.5);"
                        >
                            <option value="">-- SELECT TARGET HOST --</option>
                            <For each={hosts()}>
                                {(h) => <option value={h.id}>{h.hostname} ({h.ip_address})</option>}
                            </For>
                        </select>
                        <div style="margin-top: 24px; padding: 16px; background: rgba(59, 130, 246, 0.05); border: 1px solid rgba(59, 130, 246, 0.2); border-radius: 8px; color: rgba(59, 130, 246, 0.8); font-size: 11px; font-style: italic; line-height: 1.6;">
                            Simulation mode injects events directly into the internal bus. No actual systems are harmed, but detection engines and playbooks will react as if the attack is live.
                        </div>
                    </div>

                    <div style="margin-top: 48px;">
                        <Show when={status()}>
                            <div style={`padding: 16px; border-radius: 8px; margin-bottom: 24px; font-size: 11px; font-family: var(--font-mono); ${status()?.type === 'success' ? 'background: rgba(34, 197, 94, 0.1); color: var(--status-online); border: 1px solid rgba(34, 197, 94, 0.2);' :
                                    status()?.type === 'error' ? 'background: rgba(239, 68, 68, 0.1); color: var(--status-offline); border: 1px solid rgba(239, 68, 68, 0.2);' :
                                        'background: rgba(59, 130, 246, 0.1); color: var(--accent-primary); border: 1px solid rgba(59, 130, 246, 0.2);'
                                }`}>
                                [{status()?.type.toUpperCase()}] {status()?.msg}
                            </div>
                        </Show>
                        <button
                            onClick={handleRun}
                            disabled={!selectedScenario() || !selectedHost()}
                            class="ob-btn ob-btn-primary"
                            style={`width: 100%; padding: 20px; font-size: 12px; letter-spacing: 2px; ${selectedScenario() && selectedHost() ? 'background: var(--status-offline); border-color: var(--status-offline); color: white;' : 'opacity: 0.5; cursor: not-allowed;'
                                }`}
                        >
                            EXECUTE SIMULATION
                        </button>
                    </div>
                </div>
            </div>
        </div>
    );
};
