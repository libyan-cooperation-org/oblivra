import { Component, createSignal, createEffect, onCleanup, For, Show } from 'solid-js';
import { ListAgents } from '../../../wailsjs/go/services/AgentService';
import { services } from '../../../wailsjs/go/models';

export const AgentConsole: Component = () => {
    const [agents, setAgents] = createSignal<services.AgentDTO[]>([]);
    const [error, setError] = createSignal<string | null>(null);

    const fetchAgents = async () => {
        setError(null);
        try {
            const list = await ListAgents();
            setAgents(list || []);
        } catch (e: unknown) {
            console.error("Failed to list agents:", e);
            setError(e instanceof Error ? (e as Error).message : String(e));
        }
    };

    createEffect(() => {
        fetchAgents();
        const interval = setInterval(fetchAgents, 5000);
        onCleanup(() => clearInterval(interval));
    });

    const statusColor = (status?: string) => {
       if (status === 'online') return '#238636';
       if (status === 'degraded') return '#d29922';
       if (status === 'isolated') return '#f85149';
       return '#da3633';
    };

    return (
        <div class="agent-console">
            <div class="section-header" style="display: flex; justify-content: space-between; align-items: center;">
                <h3 style="margin: 0;">🛡️ Deployed Agents</h3>
                <button class="ops-btn-sm" onClick={fetchAgents}>🔄 Refresh</button>
            </div>
            <p style="color: var(--text-secondary); margin-bottom: 24px; font-size: 14px;">Monitor endpoints tracking metrics, syslog, and file integrity across your fleet natively.</p>

            <Show when={error()}>
                <div class="ops-error">{error()}</div>
            </Show>

            <div class="agent-grid" style="display: grid; grid-template-columns: repeat(auto-fill, minmax(320px, 1fr)); gap: 16px;">
                <Show when={agents().length === 0 && !error()}>
                    <div style="grid-column: 1 / -1; text-align: center; padding: 48px 24px; background: var(--bg-surface); border: 1px dashed var(--border-primary); border-radius: 4px;">
                        <span style="font-size: 32px;">📡</span>
                        <h4 style="margin: 12px 0 8px 0;">No Active Agents Detected</h4>
                        <p style="color: var(--text-secondary); margin: 0; max-width: 400px; margin-left: auto; margin-right: auto;">
                            No agents have heartbeated recently. Start an agent locally or deploy it to remote endpoints via SSH using the Multi-Exec capability.
                        </p>
                    </div>
                </Show>

                <For each={agents()}>
                    {(agent: any) => (
                        <div class="agent-card" style="background: var(--bg-surface); border: 1px solid var(--border-primary); border-radius: 4px; padding: 16px; display: flex; flex-direction: column; gap: 8px;">
                            <div style="display: flex; justify-content: space-between; align-items: flex-start;">
                                <h4 style="margin: 0; display: flex; align-items: center; gap: 8px; color: var(--text-primary);">
                                    <span style={`width: 8px; height: 8px; border-radius: 50%; background: ${statusColor(agent.status)}`}></span>
                                    {agent.hostname}
                                </h4>
                                <span style="font-size: 11px; font-family: var(--font-mono); background: var(--bg-primary); padding: 4px 6px; border-radius: 4px; color: var(--text-secondary); border: 1px solid var(--border-secondary);">
                                    v{agent.version}
                                </span>
                            </div>

                            <div style="font-size: 12px; display: flex; flex-direction: column; gap: 6px; font-family: var(--font-mono); padding: 8px; background: var(--bg-primary); border-radius: 4px;">
                                <div style="display: flex; justify-content: space-between;">
                                    <span style="color: var(--text-secondary);">ID</span>
                                    <span style="color: var(--text-primary);">{agent.id}</span>
                                </div>
                                <div style="display: flex; justify-content: space-between;">
                                    <span style="color: var(--text-secondary);">Platform</span>
                                    <span style="color: var(--text-primary); text-transform: uppercase;">
                                       <span style="background:var(--bg-secondary); padding:2px 4px; border-radius:2px; margin-right:4px;">{agent.os}</span>
                                       <span style="background:var(--bg-secondary); padding:2px 4px; border-radius:2px;">{agent.arch}</span>
                                    </span>
                                </div>
                                <div style="display: flex; justify-content: space-between;">
                                    <span style="color: var(--text-secondary);">Last Seen</span>
                                    <span style="color: var(--text-primary);">{new Date(agent.last_seen).toLocaleTimeString()}</span>
                                </div>
                            </div>
                            
                            <div style="display:flex; flex-wrap:wrap; gap:4px; margin-top:4px;">
                               <For each={agent.collectors || []}>
                                  {c => <span style="background:rgba(87,139,255,0.1); border:1px solid rgba(87,139,255,0.3); padding:2px 6px; border-radius:12px; font-size:10px; font-family:var(--font-mono); color:var(--accent-primary);">{c}</span>}
                               </For>
                            </div>

                            <div style="margin-top: 8px; border-top: 1px solid var(--border-primary); padding-top: 12px; display: flex; justify-content: flex-end; gap: 8px;">
                                <button class="ops-btn-sm" style="background: var(--bg-primary); border: 1px solid var(--border-secondary); color: var(--text-secondary);">Push Config</button>
                                <button class="ops-btn-sm" style="background: var(--bg-primary); border: 1px solid var(--border-secondary); color: var(--alert-medium);">Drop Auth</button>
                            </div>
                        </div>
                    )}
                </For>
            </div>

            <div class="deployment-hint" style="margin-top: 32px; padding: 16px; background: var(--bg-surface); border-left: 4px solid var(--accent-primary);">
                <h4 style="margin-top: 0; margin-bottom: 8px; color: var(--text-primary);">Quick Agent Deployment</h4>
                <p style="font-size: 13px; color: var(--text-secondary); margin-bottom: 12px;">Ensure port 8443 is exposed over your local overlay network or zero-trust tunnel.</p>
                <code style="display: block; background: var(--bg-primary); padding: 12px; border-radius: 4px; font-family: var(--font-mono); font-size: 13px; color: var(--accent-secondary); user-select: all;">
                    ./oblivra-agent --server localhost:8443 --metrics=true --syslog=true --fim=true
                </code>
            </div>
        </div>
    );
};
