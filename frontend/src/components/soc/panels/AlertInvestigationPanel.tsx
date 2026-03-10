import { Component, For, createResource, onMount } from 'solid-js';
import { GetAlertHistory } from '../../../../wailsjs/go/app/AlertingService';
import { ConnectToSession } from '../../../../wailsjs/go/app/SSHService';
import { useToast } from '../../../core/toast';

export const AlertInvestigationPanel: Component = () => {
    const { addToast } = useToast();
    const [alerts, { refetch }] = createResource(async () => {
        try {
            const data = await GetAlertHistory();
            return data.map((item: any) => ({
                id: item.ID || Math.random(),
                title: item.Title || item.Name || 'Unknown Alert',
                host: item.Hostname || item.Host || 'N/A',
                user: item.User || 'N/A',
                risk: item.RiskScore || item.Severity || 50,
                time: item.Timestamp ? new Date(item.Timestamp).toLocaleTimeString() : 'N/A'
            }));
        } catch (e) {
            console.error("Failed to fetch alerts:", e);
            return [];
        }
    });

    onMount(() => {
        const interval = setInterval(refetch, 10000); // Poll every 10s
        return () => clearInterval(interval);
    });

    return (
        <div class="h-full flex flex-col bg-gray-950 font-mono text-[11px]">
            <div class="p-3 border-b border-gray-800 flex justify-between items-center bg-gray-900/30">
                <div class="flex items-center gap-2">
                    <span class="text-gray-400 font-bold tracking-widest uppercase text-[10px]">Active Ingress Alerts</span>
                    <span class="text-[9px] px-1 bg-red-950 text-red-500 border border-red-900 font-bold">LIVE-STREAM</span>
                </div>
                {alerts.loading && <span class="text-[9px] text-blue-500 animate-pulse">REFRESHING...</span>}
            </div>

            <div class="flex-1 overflow-auto p-2 space-y-2">
                <For each={alerts() || []}>
                    {(item) => (
                        <div class="group p-3 border border-gray-800 hover:border-red-900 hover:bg-red-950/10 transition-all cursor-pointer relative overflow-hidden">
                            <div class="flex justify-between mb-1">
                                <span class="text-red-500 font-bold text-[10px] uppercase truncate max-w-[180px]">{item.title}</span>
                                <span class="text-gray-600 text-[9px]">{item.time}</span>
                            </div>
                            <div class="flex gap-4 text-gray-500 text-[9px]">
                                <span>HOST: <span class="text-gray-300">{item.host}</span></span>
                                <span>USER: <span class="text-gray-300">{item.user}</span></span>
                            </div>

                            <div class="mt-2 flex items-center gap-2">
                                <div class="flex-1 h-1 bg-gray-900 rounded-full overflow-hidden">
                                    <div
                                        class={`h-full ${item.risk > 80 ? 'bg-red-600' : item.risk > 50 ? 'bg-orange-500' : 'bg-yellow-500'}`}
                                        style={{ width: `${item.risk}%` }}
                                    ></div>
                                </div>
                                <span class={`font-bold text-[10px] ${item.risk > 80 ? 'text-red-400' : 'text-gray-400'}`}>{item.risk}</span>
                            </div>

                            <div class="mt-3 flex justify-end opacity-0 group-hover:opacity-100 transition-opacity">
                                <button
                                    class="text-[8px] bg-red-900/40 border border-red-900/50 px-2 py-0.5 text-red-400 font-bold hover:bg-red-600 hover:text-white transition-all uppercase tracking-tighter"
                                    onClick={async (e) => {
                                        e.stopPropagation();
                                        try {
                                            // Trigger tactical pivot in the backend
                                            await ConnectToSession(item.host, "soc-session");
                                            addToast({
                                                type: 'info',
                                                title: 'Tactical Pivot',
                                                message: `Establishing SSH connection to ${item.host}...`
                                            });
                                        } catch (err) {
                                            addToast({
                                                type: 'error',
                                                title: 'Pivot Failed',
                                                message: `Could not connect to ${item.host}: ${err}`
                                            });
                                        }
                                    }}
                                >
                                    PIVOT TO TERMINAL
                                </button>
                            </div>

                            <div class="absolute inset-0 bg-gradient-to-r from-red-600/0 via-red-600/0 to-red-600/5 opacity-0 group-hover:opacity-100 transition-opacity pointer-events-none"></div>
                        </div>
                    )}
                </For>

                {!alerts.loading && (!alerts() || alerts()?.length === 0) && (
                    <div class="h-full flex items-center justify-center flex-col opacity-30 gap-2">
                        <div class="w-8 h-8 border border-dashed border-gray-500 rounded-full"></div>
                        <span class="text-[10px]">NO ACTIVE ALERTS DETECTED</span>
                    </div>
                )}
            </div>

            <div class="p-2 border-t border-gray-800 bg-gray-900/50 flex justify-between text-[10px] text-gray-600">
                <span>TOTAL: {alerts()?.length || 0} ITEMS</span>
                <span class="text-blue-500 hover:text-white transition-colors cursor-pointer uppercase font-bold text-[9px]" onClick={refetch}>Manual Refresh</span>
            </div>
        </div>
    );
};
