import { Component, createSignal, onMount, onCleanup, For, Show } from 'solid-js';
import { subscribe } from '@core/bridge';
import { EmptyState } from '../ui/EmptyState';
import '@styles/tunnels.css';

interface TunnelConfig {
    type: 'local' | 'remote' | 'dynamic';
    local_host: string;
    local_port: number;
    remote_host: string;
    remote_port: number;
}

interface TunnelInfo {
    id: string;
    config: TunnelConfig;
    state: string;
    started_at: string;
    conns: number;
}

interface TunnelManagerProps {
    sessionId: string;
}

const tunnelSvc = (window as any).go?.app?.TunnelService;

export const TunnelManager: Component<TunnelManagerProps> = (props) => {
    const [tunnels, setTunnels] = createSignal<TunnelInfo[]>([]);

    // Form state
    const [isFormOpen, setIsFormOpen] = createSignal(false);
    const [type, setType] = createSignal<'local' | 'remote' | 'dynamic'>('local');
    const [localHost, setLocalHost] = createSignal('127.0.0.1');
    const [localPort, setLocalPort] = createSignal(8080);
    const [remoteHost, setRemoteHost] = createSignal('127.0.0.1');
    const [remotePort, setRemotePort] = createSignal(80);

    let unsubAdd: (() => void) | undefined;
    let unsubRem: (() => void) | undefined;

    const loadTunnels = async () => {
        if (!tunnelSvc) return;
        try {
            const list = await tunnelSvc.GetBySession(props.sessionId);
            setTunnels((list as any[]) || []);
        } catch (e) {
            console.error("Failed to load tunnels:", e);
        }
    };

    onMount(() => {
        loadTunnels();

        unsubAdd = subscribe('tunnel.added', () => loadTunnels());
        unsubRem = subscribe('tunnel.removed', () => loadTunnels());
    });

    onCleanup(() => {
        if (unsubAdd) unsubAdd();
        if (unsubRem) unsubRem();
    });

    const handleAdd = async (e: Event) => {
        e.preventDefault();
        if (!tunnelSvc) return;

        const cfg: TunnelConfig = {
            type: type(),
            local_host: localHost(),
            local_port: localPort(),
            remote_host: type() === 'dynamic' ? '' : remoteHost(),
            remote_port: type() === 'dynamic' ? 0 : remotePort(),
        };

        try {
            await (tunnelSvc as any).CreateTunnel(props.sessionId, cfg.type, cfg.local_host, cfg.local_port, cfg.remote_host, cfg.remote_port, 0);
            setIsFormOpen(false);
            // Reset common defaults
            setLocalPort(localPort() + 1);
        } catch (err) {
            alert(`Failed to start tunnel: ${err}`);
        }
    };

    const handleStop = async (tunnelId: string) => {
        if (!tunnelSvc) return;
        try {
            await tunnelSvc.StopTunnel(tunnelId);
        } catch (err) {
            alert(`Failed to stop tunnel: ${err}`);
        }
    };

    const formatLabel = (cfg: TunnelConfig) => {
        if (cfg.type === 'dynamic') {
            return `SOCKS5 Proxy on ${cfg.local_host}:${cfg.local_port}`;
        }
        if (cfg.type === 'local') {
            return `Local ${cfg.local_host}:${cfg.local_port} -> ${cfg.remote_host}:${cfg.remote_port}`;
        }
        return `Remote ${cfg.remote_host}:${cfg.remote_port} -> ${cfg.local_host}:${cfg.local_port}`;
    };

    return (
        <div class="tunnel-container">
            <div class="tunnel-header">
                <h2>Port Fowarding</h2>
                <button class="btn-primary" onClick={() => setIsFormOpen(!isFormOpen())}>
                    {isFormOpen() ? 'Cancel' : 'New Tunnel'}
                </button>
            </div>

            <Show when={isFormOpen()}>
                <form class="tunnel-form" onSubmit={handleAdd}>
                    <div class="form-row">
                        <label>Type</label>
                        <select value={type()} onChange={e => setType(e.currentTarget.value as any)}>
                            <option value="local">Local Port Forwarding (-L)</option>
                            <option value="remote">Remote Port Forwarding (-R)</option>
                            <option value="dynamic">Dynamic SOCKS5 Proxy (-D)</option>
                        </select>
                    </div>

                    <div class="form-row split">
                        <div class="form-group">
                            <label>Local Bind Address</label>
                            <input type="text" value={localHost()} onInput={e => setLocalHost(e.currentTarget.value)} required />
                        </div>
                        <div class="form-group">
                            <label>Local Port</label>
                            <input type="number" value={localPort()} onInput={e => setLocalPort(parseInt(e.currentTarget.value) || 0)} required />
                        </div>
                    </div>

                    <Show when={type() !== 'dynamic'}>
                        <div class="form-row split">
                            <div class="form-group">
                                <label>Remote Target Address</label>
                                <input type="text" value={remoteHost()} onInput={e => setRemoteHost(e.currentTarget.value)} required />
                            </div>
                            <div class="form-group">
                                <label>Remote Port</label>
                                <input type="number" value={remotePort()} onInput={e => setRemotePort(parseInt(e.currentTarget.value) || 0)} required />
                            </div>
                        </div>
                    </Show>

                    <div class="form-actions">
                        <button type="submit" class="btn-success">Start Tunnel</button>
                    </div>
                </form>
            </Show>

            <div class="tunnel-list">
                <Show when={tunnels().length === 0}>
                    <EmptyState
                        icon="🔗"
                        title="No Active Tunnels"
                        description="Create a local, remote, or SOCKS5 tunnel to forward ports through your SSH connection."
                        action="New Tunnel"
                        onAction={() => setIsFormOpen(true)}
                        compact
                    />
                </Show>

                <For each={tunnels()}>
                    {t => (
                        <div class="tunnel-item">
                            <div class="tunnel-info">
                                <strong>{t.config.type.toUpperCase()}</strong>
                                <span class="tunnel-path">{formatLabel(t.config)}</span>
                                <div class="tunnel-meta">
                                    <span class={`status-dot ${t.state}`}></span> {t.state}
                                    <span class="meta-divider">|</span>
                                    {t.conns} connections
                                </div>
                            </div>
                            <div class="tunnel-actions">
                                <button class="btn-danger btn-small" onClick={() => handleStop(t.id)}>Stop</button>
                            </div>
                        </div>
                    )}
                </For>
            </div>
        </div>
    );
};
