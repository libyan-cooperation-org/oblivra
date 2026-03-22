import { Component, createSignal, onMount, onCleanup, For, Show } from 'solid-js';
import * as AgentService from '../../../wailsjs/go/services/AgentService';

interface Agent {
    id: string;
    hostname: string;
    platform: string;
    version: string;
    status: string;
    last_seen: string;
    ip_address: string;
    cpu_usage?: number;
    mem_usage?: number;
    tags?: string[];
}

const statusColor = (s: string) => {
    switch (s) {
        case 'online': return '#3fb950';
        case 'degraded': return '#d29922';
        case 'offline': return '#f85149';
        default: return '#6b7280';
    }
};

export const FleetManagement: Component = () => {
    const [agents, setAgents] = createSignal<Agent[]>([]);
    const [loading, setLoading] = createSignal(true);
    const [search, setSearch] = createSignal('');
    const [selected, setSelected] = createSignal<Set<string>>(new Set());
    const [pushResult, setPushResult] = createSignal('');
    const [tab, setTab] = createSignal<'overview' | 'config' | 'deploy'>('overview');

    // Config push fields
    const [cfgLogLevel, setCfgLogLevel] = createSignal('info');
    const [cfgSyslog, setCfgSyslog] = createSignal('');
    const [cfgTags, setCfgTags] = createSignal('');
    const [cfgPushInterval, setCfgPushInterval] = createSignal(30);

    const refresh = async () => {
        try {
            const data = await (AgentService as any).ListAgents();
            setAgents(data || []);
        } catch { /* agent service may not have remote agents */ }
        setLoading(false);
    };

    let timer: any;
    onMount(() => { refresh(); timer = setInterval(refresh, 8000); });
    onCleanup(() => clearInterval(timer));

    const filtered = () => {
        const q = search().toLowerCase();
        return agents().filter(a =>
            !q || a.hostname?.toLowerCase().includes(q) ||
            a.ip_address?.toLowerCase().includes(q) ||
            a.platform?.toLowerCase().includes(q)
        );
    };

    const toggleSelect = (id: string) => {
        const s = new Set(selected());
        s.has(id) ? s.delete(id) : s.add(id);
        setSelected(s);
    };

    const selectAll = () => setSelected(new Set(filtered().map(a => a.id)));
    const clearSel = () => setSelected(new Set());

    const pushConfig = async () => {
        const targets = selected().size > 0
            ? filtered().filter(a => selected().has(a.id)).map(a => a.id)
            : filtered().map(a => a.id);
        if (targets.length === 0) { setPushResult('No agents targeted.'); return; }
        try {
            for (const id of targets) {
                await (AgentService as any).PushFleetConfig(
                    id, cfgLogLevel(), cfgSyslog(), cfgTags(), cfgPushInterval()
                );
            }
            setPushResult(`✓ Config pushed to ${targets.length} agent(s)`);
        } catch (e: any) {
            setPushResult('✗ Push failed: ' + (e?.message ?? e));
        }
        setTimeout(() => setPushResult(''), 4000);
    };

    const onlineCount = () => agents().filter(a => a.status === 'online').length;
    const offlineCount = () => agents().filter(a => a.status === 'offline').length;

    return (
        <div style="padding: 0; height: 100%; overflow-y: auto; background: var(--bg-primary); color: var(--text-primary); font-family: var(--font-ui);">
            {/* Header */}
            <div style="height: var(--header-height); border-bottom: 1px solid var(--glass-border); display: flex; justify-content: space-between; align-items: center; padding: 0 1.5rem; background: var(--bg-secondary);">
                <div style="display: flex; align-items: center; gap: 0.75rem;">
                    <span style="font-size: 16px;">🛰️</span>
                    <h2 style="font-size: 13px; letter-spacing: 2px; font-weight: 700; margin: 0; text-transform: uppercase;">Fleet Management</h2>
                </div>
                <div style="display: flex; gap: 1rem; font-family: var(--font-mono); font-size: 11px;">
                    <span style="color: #3fb950;">● {onlineCount()} ONLINE</span>
                    <span style="color: #f85149;">● {offlineCount()} OFFLINE</span>
                    <span style="color: var(--text-muted);">{agents().length} TOTAL</span>
                </div>
            </div>

            {/* Tab bar */}
            <div style="display: flex; gap: 0; border-bottom: 1px solid var(--glass-border); background: var(--bg-secondary);">
                {(['overview', 'config', 'deploy'] as const).map(t => (
                    <button
                        onClick={() => setTab(t)}
                        style={`padding: 10px 20px; font-size: 11px; font-weight: 700; letter-spacing: 1px; text-transform: uppercase; font-family: var(--font-mono); border: none; cursor: pointer; background: transparent; border-bottom: 2px solid ${tab() === t ? 'var(--accent-primary)' : 'transparent'}; color: ${tab() === t ? 'var(--accent-primary)' : 'var(--text-muted)'};`}
                    >{t}</button>
                ))}
            </div>

            <div style="padding: 1.5rem;">

                {/* Overview Tab */}
                <Show when={tab() === 'overview'}>
                    {/* Search + bulk actions */}
                    <div style="display: flex; gap: 0.75rem; margin-bottom: 1rem; align-items: center;">
                        <input
                            placeholder="Search hostname, IP, platform..."
                            value={search()}
                            onInput={e => setSearch((e.target as HTMLInputElement).value)}
                            style="flex: 1; background: var(--bg-secondary); border: 1px solid var(--glass-border); color: var(--text-primary); padding: 7px 12px; border-radius: 4px; font-family: var(--font-mono); font-size: 11px;"
                        />
                        <button onClick={selectAll} style="padding: 7px 12px; font-size: 10px; font-family: var(--font-mono); background: rgba(87,139,255,0.1); border: 1px solid rgba(87,139,255,0.3); color: var(--accent-primary); border-radius: 4px; cursor: pointer; text-transform: uppercase; letter-spacing: 1px;">Select All</button>
                        <button onClick={clearSel} style="padding: 7px 12px; font-size: 10px; font-family: var(--font-mono); background: transparent; border: 1px solid var(--glass-border); color: var(--text-muted); border-radius: 4px; cursor: pointer; text-transform: uppercase; letter-spacing: 1px;">Clear</button>
                        <button onClick={refresh} style="padding: 7px 12px; font-size: 10px; font-family: var(--font-mono); background: transparent; border: 1px solid var(--glass-border); color: var(--text-muted); border-radius: 4px; cursor: pointer; text-transform: uppercase; letter-spacing: 1px;">↻ Refresh</button>
                    </div>

                    <Show when={loading()}>
                        <div style="color: var(--text-muted); font-family: var(--font-mono); font-size: 11px; padding: 2rem; text-align: center;">POLLING FLEET...</div>
                    </Show>

                    <Show when={!loading() && filtered().length === 0}>
                        <div style="text-align: center; padding: 3rem; color: var(--text-muted); font-family: var(--font-mono); font-size: 12px;">
                            <div style="font-size: 2rem; margin-bottom: 1rem; opacity: 0.3;">🛰️</div>
                            NO AGENTS REGISTERED<br/>
                            <span style="font-size: 10px; margin-top: 0.5rem; display: block;">Deploy agents to endpoints and register them with this server to populate fleet management.</span>
                        </div>
                    </Show>

                    {/* Agent table */}
                    <Show when={filtered().length > 0}>
                        <div style="border: 1px solid var(--glass-border); border-radius: 6px; overflow: hidden;">
                            <table style="width: 100%; border-collapse: collapse; font-size: 11px; font-family: var(--font-mono);">
                                <thead>
                                    <tr style="background: var(--bg-secondary); border-bottom: 1px solid var(--glass-border);">
                                        <th style="padding: 10px 12px; text-align: left; color: var(--text-muted); font-weight: 600; letter-spacing: 1px; width: 32px;">
                                            <input type="checkbox" onChange={e => e.currentTarget.checked ? selectAll() : clearSel()} />
                                        </th>
                                        {['STATUS', 'HOSTNAME', 'PLATFORM', 'IP ADDRESS', 'VERSION', 'LAST SEEN', 'CPU', 'MEM'].map(h => (
                                            <th style="padding: 10px 12px; text-align: left; color: var(--text-muted); font-weight: 600; letter-spacing: 1px;">{h}</th>
                                        ))}
                                    </tr>
                                </thead>
                                <tbody>
                                    <For each={filtered()}>
                                        {(agent) => (
                                            <tr
                                                style={`border-bottom: 1px solid rgba(255,255,255,0.04); background: ${selected().has(agent.id) ? 'rgba(87,139,255,0.06)' : 'transparent'}; cursor: pointer;`}
                                                onClick={() => toggleSelect(agent.id)}
                                            >
                                                <td style="padding: 10px 12px;">
                                                    <input type="checkbox" checked={selected().has(agent.id)} onChange={() => toggleSelect(agent.id)} onClick={e => e.stopPropagation()} />
                                                </td>
                                                <td style="padding: 10px 12px;">
                                                    <span style={`font-size: 8px; color: ${statusColor(agent.status)};`}>●</span>
                                                    <span style={`margin-left: 6px; color: ${statusColor(agent.status)}; text-transform: uppercase;`}>{agent.status}</span>
                                                </td>
                                                <td style="padding: 10px 12px; color: var(--text-primary); font-weight: 600;">{agent.hostname || '—'}</td>
                                                <td style="padding: 10px 12px; color: var(--text-secondary);">{agent.platform || '—'}</td>
                                                <td style="padding: 10px 12px; color: var(--text-secondary);">{agent.ip_address || '—'}</td>
                                                <td style="padding: 10px 12px; color: var(--text-muted);">{agent.version || '—'}</td>
                                                <td style="padding: 10px 12px; color: var(--text-muted); font-size: 10px;">{agent.last_seen ? new Date(agent.last_seen).toLocaleTimeString() : '—'}</td>
                                                <td style="padding: 10px 12px;">
                                                    <Show when={agent.cpu_usage !== undefined}>
                                                        <div style="display: flex; align-items: center; gap: 6px;">
                                                            <div style={`width: 40px; height: 3px; background: rgba(255,255,255,0.1); border-radius: 2px; overflow: hidden;`}>
                                                                <div style={`height: 100%; width: ${agent.cpu_usage}%; background: ${(agent.cpu_usage ?? 0) > 80 ? '#f85149' : '#3fb950'};`} />
                                                            </div>
                                                            <span style="color: var(--text-muted); font-size: 10px;">{agent.cpu_usage?.toFixed(0)}%</span>
                                                        </div>
                                                    </Show>
                                                </td>
                                                <td style="padding: 10px 12px;">
                                                    <Show when={agent.mem_usage !== undefined}>
                                                        <span style="color: var(--text-muted); font-size: 10px;">{agent.mem_usage?.toFixed(0)}%</span>
                                                    </Show>
                                                </td>
                                            </tr>
                                        )}
                                    </For>
                                </tbody>
                            </table>
                        </div>
                        <div style="margin-top: 0.75rem; font-size: 10px; color: var(--text-muted); font-family: var(--font-mono);">
                            {selected().size > 0 ? `${selected().size} selected` : `${filtered().length} agents`}
                        </div>
                    </Show>
                </Show>

                {/* Config Push Tab */}
                <Show when={tab() === 'config'}>
                    <div style="max-width: 600px; display: flex; flex-direction: column; gap: 1.25rem;">
                        <div style="font-size: 12px; color: var(--text-secondary); line-height: 1.6;">
                            Push configuration to selected agents (or all agents if none selected). Changes take effect on the next agent heartbeat cycle.
                        </div>

                        {[
                            { label: 'Log Level', note: 'Agent logging verbosity' },
                        ].map(() => null)}

                        <div style="display: flex; flex-direction: column; gap: 0.4rem;">
                            <label style="font-size: 10px; letter-spacing: 1px; text-transform: uppercase; color: var(--text-muted); font-family: var(--font-mono);">Log Level</label>
                            <select value={cfgLogLevel()} onChange={e => setCfgLogLevel(e.currentTarget.value)}
                                style="background: var(--bg-secondary); border: 1px solid var(--glass-border); color: var(--text-primary); padding: 8px 12px; border-radius: 4px; font-family: var(--font-mono); font-size: 11px;">
                                {['debug', 'info', 'warn', 'error'].map(l => <option value={l}>{l}</option>)}
                            </select>
                        </div>

                        <div style="display: flex; flex-direction: column; gap: 0.4rem;">
                            <label style="font-size: 10px; letter-spacing: 1px; text-transform: uppercase; color: var(--text-muted); font-family: var(--font-mono);">Syslog Endpoint (optional)</label>
                            <input placeholder="syslog://192.168.1.100:514"
                                value={cfgSyslog()} onInput={e => setCfgSyslog((e.target as HTMLInputElement).value)}
                                style="background: var(--bg-secondary); border: 1px solid var(--glass-border); color: var(--text-primary); padding: 8px 12px; border-radius: 4px; font-family: var(--font-mono); font-size: 11px;" />
                        </div>

                        <div style="display: flex; flex-direction: column; gap: 0.4rem;">
                            <label style="font-size: 10px; letter-spacing: 1px; text-transform: uppercase; color: var(--text-muted); font-family: var(--font-mono);">Tags (comma-separated)</label>
                            <input placeholder="prod,webserver,dmz"
                                value={cfgTags()} onInput={e => setCfgTags((e.target as HTMLInputElement).value)}
                                style="background: var(--bg-secondary); border: 1px solid var(--glass-border); color: var(--text-primary); padding: 8px 12px; border-radius: 4px; font-family: var(--font-mono); font-size: 11px;" />
                        </div>

                        <div style="display: flex; flex-direction: column; gap: 0.4rem;">
                            <label style="font-size: 10px; letter-spacing: 1px; text-transform: uppercase; color: var(--text-muted); font-family: var(--font-mono);">Push Interval (seconds)</label>
                            <input type="number" min="5" max="3600"
                                value={cfgPushInterval()} onInput={e => setCfgPushInterval(parseInt((e.target as HTMLInputElement).value))}
                                style="background: var(--bg-secondary); border: 1px solid var(--glass-border); color: var(--text-primary); padding: 8px 12px; border-radius: 4px; font-family: var(--font-mono); font-size: 11px; width: 120px;" />
                        </div>

                        <div style="display: flex; align-items: center; gap: 1rem;">
                            <button onClick={pushConfig}
                                style="background: rgba(87,139,255,0.15); border: 1px solid rgba(87,139,255,0.4); color: var(--accent-primary); padding: 9px 20px; border-radius: 4px; cursor: pointer; font-family: var(--font-mono); font-size: 11px; font-weight: 700; letter-spacing: 1px; text-transform: uppercase;">
                                ⬆ PUSH TO {selected().size > 0 ? `${selected().size} SELECTED` : 'ALL AGENTS'}
                            </button>
                            <Show when={pushResult()}>
                                <span style={`font-family: var(--font-mono); font-size: 11px; color: ${pushResult().startsWith('✓') ? '#3fb950' : '#f85149'};`}>{pushResult()}</span>
                            </Show>
                        </div>
                    </div>
                </Show>

                {/* Deploy Tab */}
                <Show when={tab() === 'deploy'}>
                    <div style="max-width: 700px; display: flex; flex-direction: column; gap: 1.25rem;">
                        <div style="font-size: 12px; color: var(--text-secondary); line-height: 1.6;">
                            Deploy agents to new endpoints using the installation commands below. Agents auto-register with this server using the embedded enrollment token.
                        </div>

                        {[
                            { label: 'Linux / macOS (curl)', cmd: 'curl -fsSL https://<server>:8443/install.sh | sudo bash -s -- --token <TOKEN>' },
                            { label: 'Windows (PowerShell)', cmd: 'iwr https://<server>:8443/install.ps1 | iex; Install-OBLIVRAAgent -Token <TOKEN>' },
                            { label: 'Docker', cmd: 'docker run -d --net=host -e OBLIVRA_TOKEN=<TOKEN> ghcr.io/kingknull/oblivra-agent:latest' },
                            { label: 'Kubernetes DaemonSet', cmd: 'kubectl apply -f https://<server>:8443/k8s/agent-daemonset.yaml' },
                        ].map(({ label, cmd }) => (
                            <div style="display: flex; flex-direction: column; gap: 0.4rem;">
                                <div style="font-size: 10px; letter-spacing: 1px; text-transform: uppercase; color: var(--text-muted); font-family: var(--font-mono);">{label}</div>
                                <div style="background: var(--bg-primary); border: 1px solid var(--glass-border); border-radius: 4px; padding: 12px 14px; font-family: var(--font-mono); font-size: 11px; color: #3fb950; overflow-x: auto; white-space: nowrap;">
                                    {cmd}
                                </div>
                            </div>
                        ))}
                    </div>
                </Show>

            </div>
        </div>
    );
};
