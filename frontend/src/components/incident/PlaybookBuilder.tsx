import { Component, createSignal, onMount, For, Show } from 'solid-js';
import { RunPlaybook } from '../../../wailsjs/go/app/PlaybookService';
import { GetTimeline } from '../../../wailsjs/go/app/IncidentService';
import { database } from '../../../wailsjs/go/models';

export const PlaybookBuilder: Component<{ incident: database.Incident }> = (props) => {
    const [running, setRunning] = createSignal(false);
    const [steps, setSteps] = createSignal<{ name: string, status: 'pending' | 'running' | 'completed' | 'failed' }[]>([]);
    const [timeline, setTimeline] = createSignal<database.AuditLog[]>([]);

    onMount(async () => {
        loadTimeline();
    });

    const loadTimeline = async () => {
        try {
            const logs = await GetTimeline(null as any, props.incident.id);
            setTimeline(logs || []);
        } catch (err) {
            console.error("Failed to load timeline:", err);
        }
    };

    const runPlaybook = async (id: string) => {
        setRunning(true);
        setSteps([
            { name: "Neutralizing active sessions", status: 'running' },
            { name: "Applying host-level firewall block", status: 'pending' },
            { name: "Collecting forensic memory dump", status: 'pending' }
        ]);

        try {
            await RunPlaybook(null as any, id, props.incident.id);
            // Simulate progression for UI feedback
            setSteps(prev => prev.map((s, i) => i === 0 ? { ...s, status: 'completed' } : i === 1 ? { ...s, status: 'running' } : s));
            await new Promise(r => setTimeout(r, 1000));
            setSteps(prev => prev.map((s, i) => i === 1 ? { ...s, status: 'completed' } : i === 2 ? { ...s, status: 'running' } : s));
            await new Promise(r => setTimeout(r, 1000));
            setSteps(prev => prev.map(s => ({ ...s, status: 'completed' })));

            loadTimeline();
        } catch (err) {
            console.error("Playbook failed:", err);
            setSteps(prev => prev.map(s => s.status === 'running' ? { ...s, status: 'failed' } : s));
        } finally {
            setRunning(false);
        }
    };

    return (
        <div class="playbook-builder">
            <div class="playbook-header">
                <select disabled={running()}>
                    <option value="contain_brute_force">Brute Force Containment</option>
                    <option value="isolate_host">Full Host Isolation</option>
                </select>
                <button
                    class="action-button"
                    onClick={() => runPlaybook("contain_brute_force")}
                    disabled={running()}
                >
                    {running() ? 'NEUTRALIZING...' : 'RUN PLAYBOOK'}
                </button>
            </div>

            <Show when={steps().length > 0}>
                <div class="playbook-steps animate-fade-in">
                    <For each={steps()}>
                        {(step) => (
                            <div class={`playbook-step ${step.status}`}>
                                <Show when={step.status === 'running'}>
                                    <div class="spinner-small" />
                                </Show>
                                <Show when={step.status === 'completed'}>
                                    <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" width="16"><polyline points="20 6 9 17 4 12" /></svg>
                                </Show>
                                <span>{step.name}</span>
                            </div>
                        )}
                    </For>
                </div>
            </Show>

            <div class="forensic-timeline">
                <h3>Forensic Timeline</h3>
                <div class="timeline-view">
                    <For each={timeline()}>
                        {(log) => (
                            <div class="timeline-item">
                                <span class="timeline-time">{new Date(log.timestamp.toString()).toLocaleTimeString()}</span>
                                <span class="timeline-type">{log.event_type}</span>
                                <span class="timeline-details">{JSON.stringify(log.details)}</span>
                            </div>
                        )}
                    </For>
                </div>
            </div>
        </div>
    );
};
