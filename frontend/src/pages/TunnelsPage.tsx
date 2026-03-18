import { Component, createSignal, onMount, onCleanup, For, Show } from 'solid-js';
import { useApp } from '@core/store';
import { subscribe } from '@core/bridge';

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

const tunnelSvc = (window as any).go?.backend?.services?.TunnelService || (window as any).go?.services?.TunnelService;

export const TunnelsPage: Component = () => {
    const [state] = useApp();
    const [tunnels, setTunnels] = createSignal<TunnelInfo[]>([]);
    
    // Modal Form State
    const [isFormOpen, setIsFormOpen] = createSignal(false);
    const [type, setType] = createSignal<'local' | 'remote' | 'dynamic'>('local');
    const [localHost, setLocalHost] = createSignal('127.0.0.1');
    const [localPort, setLocalPort] = createSignal(8080);
    const [remoteHost, setRemoteHost] = createSignal('127.0.0.1');
    const [remotePort, setRemotePort] = createSignal(80);

    let unsubAdd: (() => void) | undefined;
    let unsubRem: (() => void) | undefined;

    const loadTunnels = async () => {
        if (!tunnelSvc || !state.activeSessionId) return;
        try {
            const list = await tunnelSvc.GetBySession(state.activeSessionId);
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
        if (!tunnelSvc || !state.activeSessionId) {
            alert("Please connect to a terminal session first.");
            return;
        }

        const cfg: TunnelConfig = {
            type: type(),
            local_host: localHost(),
            local_port: localPort(),
            remote_host: type() === 'dynamic' ? '' : remoteHost(),
            remote_port: type() === 'dynamic' ? 0 : remotePort(),
        };

        try {
            await tunnelSvc.CreateTunnel(state.activeSessionId, cfg.type, cfg.local_host, cfg.local_port, cfg.remote_host, cfg.remote_port, 0);
            setIsFormOpen(false);
            setLocalPort(localPort() + 1); // increment for convenience
            loadTunnels();
        } catch (err) {
            alert(`Failed to start tunnel: ${err}`);
        }
    };

    const handleStop = async (tunnelId: string) => {
        if (!tunnelSvc) return;
        try {
            await tunnelSvc.StopTunnel(tunnelId);
            loadTunnels();
        } catch (err) {
            alert(`Failed to stop tunnel: ${err}`);
        }
    };

    const getTypeColor = (t: string) => {
        switch(t) {
            case 'local': return 'text-sky-400 bg-sky-400/10 border-sky-400/20';
            case 'remote': return 'text-purple-400 bg-purple-400/10 border-purple-400/20';
            case 'dynamic': return 'text-emerald-400 bg-emerald-400/10 border-emerald-400/20';
            default: return 'text-gray-400 bg-gray-400/10 border-gray-400/20';
        }
    };

    const formatTimeActive = (startStr: string) => {
        try {
            const start = new Date(startStr).getTime();
            const diff = Date.now() - start;
            const mins = Math.floor(diff / 60000);
            if (mins < 60) return `${mins}m`;
            return `${Math.floor(mins/60)}h ${mins%60}m`;
        } catch { return '-'; }
    };

    return (
        <div class="flex flex-col h-full w-full bg-[#0B0D14] text-gray-200 overflow-hidden relative">
            <div class="p-6 border-b border-gray-800/60 flex justify-between items-center bg-[#11131A]/80 backdrop-blur-md z-10">
                <div>
                    <h1 class="text-2xl font-bold tracking-tight text-white font-mono flex items-center gap-3">
                        <svg class="w-6 h-6 text-emerald-500" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 10V3L4 14h7v7l9-11h-7z"></path></svg>
                        Secure Tunnels
                    </h1>
                    <p class="text-sm text-gray-500 mt-1">Manage active port forwarding and SOCKS proxies.</p>
                </div>
                
                <button 
                    onClick={() => setIsFormOpen(true)}
                    class="px-4 py-2 bg-emerald-600 hover:bg-emerald-500 text-white rounded-lg border border-emerald-500 text-sm font-medium transition-all shadow-[0_0_15px_rgba(16,185,129,0.2)] hover:shadow-[0_0_20px_rgba(16,185,129,0.4)] flex items-center gap-2"
                >
                    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"></path></svg>
                    New Tunnel
                </button>
            </div>

            <div class="flex-1 overflow-y-auto p-6 custom-scrollbar">
                
                <Show when={!state.activeSessionId}>
                    <div class="w-full max-w-2xl mx-auto mt-12 mb-8 p-6 bg-amber-900/10 border border-amber-500/20 rounded-xl flex items-start gap-4">
                        <svg class="w-6 h-6 text-amber-500 shrink-0 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"></path></svg>
                        <div>
                            <h3 class="text-amber-500 font-semibold mb-1">No Active Session</h3>
                            <p class="text-sm text-gray-400">Open a terminal session to a host first before binding SSH tunnels to it.</p>
                        </div>
                    </div>
                </Show>

                <div class="w-full max-w-6xl mx-auto mt-4 relative">
                    {/* Tunnel Data Table */}
                    <div class="bg-[#161923]/60 border border-gray-800 rounded-xl overflow-hidden backdrop-blur-sm shadow-xl">
                        <div class="overflow-x-auto">
                            <table class="w-full text-left border-collapse">
                                <thead>
                                    <tr class="bg-gray-900/40 border-b border-gray-800 text-xs uppercase tracking-wider text-gray-400 font-semibold">
                                        <th class="px-6 py-4">Status</th>
                                        <th class="px-6 py-4">Type</th>
                                        <th class="px-6 py-4">Local Bind</th>
                                        <th class="px-6 py-4">Remote Target</th>
                                        <th class="px-6 py-4">Stats</th>
                                        <th class="px-6 py-4 text-right">Actions</th>
                                    </tr>
                                </thead>
                                <tbody class="divide-y divide-gray-800/60">
                                    <Show when={tunnels().length === 0}>
                                        <tr>
                                            <td colspan="6" class="px-6 py-16 text-center text-gray-500">
                                                <div class="flex flex-col items-center">
                                                    <svg class="w-12 h-12 text-gray-700 mb-3" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M8 7h12m0 0l-4-4m4 4l-4 4m0 6H4m0 0l4 4m-4-4l4-4"></path></svg>
                                                    <p>No active tunnels for this session.</p>
                                                </div>
                                            </td>
                                        </tr>
                                    </Show>
                                    <For each={tunnels()}>
                                        {(t) => (
                                            <tr class="hover:bg-white/[0.02] transition-colors group">
                                                <td class="px-6 py-4 whitespace-nowrap">
                                                    <div class="flex items-center gap-2">
                                                        <span class={`w-2 h-2 rounded-full ${t.state === 'running' || t.state === 'active' ? 'bg-emerald-500 animate-pulse' : 'bg-red-500'}`}></span>
                                                        <span class="text-sm text-gray-300 capitalize">{t.state}</span>
                                                    </div>
                                                </td>
                                                <td class="px-6 py-4 whitespace-nowrap">
                                                    <span class={`px-2.5 py-1 rounded text-xs font-mono font-medium border uppercase tracking-wider ${getTypeColor(t.config.type)}`}>
                                                        {t.config.type}
                                                    </span>
                                                </td>
                                                <td class="px-6 py-4 whitespace-nowrap font-mono text-sm">
                                                    <span class="text-gray-400">{t.config.local_host}:</span><span class="text-white font-semibold">{t.config.local_port}</span>
                                                </td>
                                                <td class="px-6 py-4 whitespace-nowrap font-mono text-sm">
                                                    <Show when={t.config.type !== 'dynamic'} fallback={<span class="text-gray-600 italic">SOCKS Proxy</span>}>
                                                        <span class="text-gray-400">{t.config.remote_host}:</span><span class="text-white font-semibold">{t.config.remote_port}</span>
                                                    </Show>
                                                </td>
                                                <td class="px-6 py-4 whitespace-nowrap">
                                                    <div class="flex flex-col text-xs">
                                                        <span class="text-gray-300"><span class="text-gray-500">Conns:</span> {t.conns}</span>
                                                        <span class="text-gray-300"><span class="text-gray-500">Active:</span> {formatTimeActive(t.started_at)}</span>
                                                    </div>
                                                </td>
                                                <td class="px-6 py-4 whitespace-nowrap text-right">
                                                    <button 
                                                        onClick={() => handleStop(t.id)}
                                                        class="px-3 py-1 bg-red-500/10 hover:bg-red-500/20 text-red-400 border border-red-500/20 hover:border-red-500/30 rounded transition-colors text-xs font-medium opacity-0 group-hover:opacity-100"
                                                    >
                                                        Terminate
                                                    </button>
                                                </td>
                                            </tr>
                                        )}
                                    </For>
                                </tbody>
                            </table>
                        </div>
                    </div>
                </div>
            </div>

            {/* Creation Modal Overlay */}
            <Show when={isFormOpen()}>
                <div class="absolute inset-0 z-50 bg-black/60 backdrop-blur-sm flex items-center justify-center p-4 animate-fade-in">
                    <div class="w-full max-w-lg bg-[#11131A] border border-gray-700 rounded-xl shadow-[0_0_40px_rgba(0,0,0,0.5)] overflow-hidden flex flex-col scale-in">
                        <div class="p-5 border-b border-gray-800 flex justify-between items-center text-white">
                            <h2 class="text-lg font-semibold tracking-tight">Establish New Tunnel</h2>
                            <button onClick={() => setIsFormOpen(false)} class="text-gray-500 hover:text-white transition-colors">
                                <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path></svg>
                            </button>
                        </div>
                        
                        <form onSubmit={handleAdd} class="p-6 space-y-6">
                            <div>
                                <label class="block text-sm font-medium text-gray-400 mb-2">Forwarding Protocol</label>
                                <div class="grid grid-cols-3 gap-2">
                                    <button type="button" onClick={() => setType('local')} class={`px-3 py-2 text-xs font-medium rounded-lg border transition-all ${type() === 'local' ? 'bg-sky-500/20 border-sky-500/50 text-sky-400' : 'bg-[#090A0F] border-gray-800 text-gray-400 hover:border-gray-600'}`}>Local (-L)</button>
                                    <button type="button" onClick={() => setType('remote')} class={`px-3 py-2 text-xs font-medium rounded-lg border transition-all ${type() === 'remote' ? 'bg-purple-500/20 border-purple-500/50 text-purple-400' : 'bg-[#090A0F] border-gray-800 text-gray-400 hover:border-gray-600'}`}>Remote (-R)</button>
                                    <button type="button" onClick={() => setType('dynamic')} class={`px-3 py-2 text-xs font-medium rounded-lg border transition-all ${type() === 'dynamic' ? 'bg-emerald-500/20 border-emerald-500/50 text-emerald-400' : 'bg-[#090A0F] border-gray-800 text-gray-400 hover:border-gray-600'}`}>SOCKS5 (-D)</button>
                                </div>
                            </div>
                            
                            <div class="bg-black/30 p-4 rounded-lg border border-gray-800/50 space-y-4">
                                <div class="flex items-center gap-4">
                                    <div class="flex-1">
                                        <label class="block text-xs font-medium text-gray-500 mb-1">Local Bind Address</label>
                                        <input type="text" value={localHost()} onInput={e => setLocalHost(e.currentTarget.value)} required class="w-full bg-[#090A0F] border border-gray-700 rounded-md px-3 py-2 text-sm text-gray-200 focus:outline-none focus:border-emerald-500/50" />
                                    </div>
                                    <div class="w-24">
                                        <label class="block text-xs font-medium text-gray-500 mb-1">Local Port</label>
                                        <input type="number" value={localPort()} onInput={e => setLocalPort(parseInt(e.currentTarget.value) || 0)} required class="w-full bg-[#090A0F] border border-gray-700 rounded-md px-3 py-2 text-sm text-gray-200 focus:outline-none focus:border-emerald-500/50 font-mono" />
                                    </div>
                                </div>
                                
                                <Show when={type() !== 'dynamic'}>
                                    <div class="flex items-center justify-center h-4">
                                        <svg class="w-5 h-5 text-gray-600" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 14l-7 7m0 0l-7-7m7 7V3"></path></svg>
                                    </div>
                                    <div class="flex items-center gap-4">
                                        <div class="flex-1">
                                            <label class="block text-xs font-medium text-gray-500 mb-1">Remote Target Host</label>
                                            <input type="text" value={remoteHost()} onInput={e => setRemoteHost(e.currentTarget.value)} required class="w-full bg-[#090A0F] border border-gray-700 rounded-md px-3 py-2 text-sm text-gray-200 focus:outline-none focus:border-emerald-500/50" />
                                        </div>
                                        <div class="w-24">
                                            <label class="block text-xs font-medium text-gray-500 mb-1">Remote Port</label>
                                            <input type="number" value={remotePort()} onInput={e => setRemotePort(parseInt(e.currentTarget.value) || 0)} required class="w-full bg-[#090A0F] border border-gray-700 rounded-md px-3 py-2 text-sm text-gray-200 focus:outline-none focus:border-emerald-500/50 font-mono" />
                                        </div>
                                    </div>
                                </Show>
                            </div>
                            
                            <div class="flex justify-end gap-3 pt-4">
                                <button type="button" onClick={() => setIsFormOpen(false)} class="px-4 py-2 bg-[#1C202B] hover:bg-[#252A38] border border-gray-700 rounded-lg text-sm text-gray-300 font-medium transition-colors">Cancel</button>
                                <button type="submit" class="px-5 py-2 bg-emerald-600 hover:bg-emerald-500 text-white rounded-lg border border-emerald-500 text-sm font-medium transition-colors shadow-[0_0_10px_rgba(16,185,129,0.2)]">Establish Connection</button>
                            </div>
                        </form>
                    </div>
                </div>
            </Show>

            <style>{`
                .custom-scrollbar::-webkit-scrollbar { width: 6px; }
                .custom-scrollbar::-webkit-scrollbar-track { background: transparent; }
                .custom-scrollbar::-webkit-scrollbar-thumb { background-color: rgba(75, 85, 99, 0.4); border-radius: 20px; }
                .custom-scrollbar::-webkit-scrollbar-thumb:hover { background-color: rgba(107, 114, 128, 0.6); }
                
                @keyframes fadeIn {
                    from { opacity: 0; }
                    to { opacity: 1; }
                }
                .animate-fade-in { animation: fadeIn 0.2s ease-out forwards; }
                
                @keyframes scaleIn {
                    from { opacity: 0; transform: scale(0.95); }
                    to { opacity: 1; transform: scale(1); }
                }
                .scale-in { animation: scaleIn 0.2s cubic-bezier(0.16, 1, 0.3, 1) forwards; }
            `}</style>
        </div>
    );
};
