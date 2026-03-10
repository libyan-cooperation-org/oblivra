import { Component, For, createSignal, createResource, onMount } from 'solid-js';
import { SearchHostEvents } from '../../../../wailsjs/go/app/SIEMService';

export const LogSearchPanel: Component = () => {
    const [query, setQuery] = createSignal("*");
    const [logs, { refetch }] = createResource(query, async (q) => {
        try {
            // Wails binding for SearchHostEvents(query, limit)
            const data = await SearchHostEvents(q || "*", 100);
            return data.map((log: any) => ({
                time: log.Timestamp ? new Date(log.Timestamp).toLocaleTimeString() : 'N/A',
                src: log.Source || log.RemoteAddr || 'Local',
                dst: log.Destination || 'Internal',
                proto: log.Protocol || 'TCP',
                msg: log.Message || log.Content || 'No message content'
            }));
        } catch (e) {
            console.error("SIEM Search Failed:", e);
            return [];
        }
    });

    onMount(() => {
        refetch();
    });

    return (
        <div class="h-full flex flex-col bg-black text-gray-300 font-mono text-[11px]">
            {/* Query Bar */}
            <div class="p-2 bg-gray-900 border-b border-gray-800 flex gap-2">
                <div class="relative flex-1">
                    <span class="absolute left-3 top-1/2 -translate-y-1/2 text-green-800 text-[10px] font-bold">SQL{'>'}</span>
                    <input
                        value={query()}
                        onInput={(e) => setQuery(e.currentTarget.value)}
                        onKeyPress={(e) => e.key === 'Enter' && refetch()}
                        class="w-full bg-black border border-gray-700 pl-10 pr-3 py-1 outline-none text-green-500 focus:border-green-800 transition-colors"
                        placeholder="SELECT * FROM siem.events..."
                    />
                </div>
                <button
                    onClick={() => refetch()}
                    class="px-4 py-1 bg-blue-600 hover:bg-blue-700 text-white font-bold transition-all active:scale-95 disabled:opacity-50"
                    disabled={logs.loading}
                >
                    {logs.loading ? '...' : 'EXECUTE'}
                </button>
            </div>

            <div class="flex-1 overflow-auto bg-[#050505] relative">
                {logs.loading && (
                    <div class="absolute inset-0 bg-black/50 backdrop-blur-sm z-10 flex items-center justify-center">
                        <span class="text-blue-500 animate-pulse font-bold tracking-[0.2em] text-[10px]">INDEX_SCAN_IN_PROGRESS</span>
                    </div>
                )}

                <table class="w-full text-left border-collapse">
                    <thead class="sticky top-0 bg-gray-900 text-gray-500 text-[10px] uppercase border-b border-gray-800 z-20">
                        <tr>
                            <th class="p-2 font-bold whitespace-nowrap border-r border-gray-800 w-24">TIMESTAMP</th>
                            <th class="p-2 font-bold whitespace-nowrap border-r border-gray-800 w-32">SOURCE</th>
                            <th class="p-2 font-bold whitespace-nowrap border-r border-gray-800 w-32">DESTINATION</th>
                            <th class="p-2 font-bold whitespace-nowrap border-r border-gray-800 w-16">PROTO</th>
                            <th class="p-2 font-bold">MESSAGE</th>
                        </tr>
                    </thead>
                    <tbody>
                        <For each={logs() || []}>
                            {(log) => (
                                <tr class="border-b border-gray-900 hover:bg-blue-900/5 transition-colors group">
                                    <td class="p-2 text-gray-600 whitespace-nowrap border-r border-gray-900">{log.time}</td>
                                    <td class="p-2 text-blue-400 font-bold whitespace-nowrap border-r border-gray-900">{log.src}</td>
                                    <td class="p-2 text-green-400 font-bold whitespace-nowrap border-r border-gray-900">{log.dst}</td>
                                    <td class="p-2 text-orange-400/70 whitespace-nowrap border-r border-gray-900 text-[10px]">{log.proto}</td>
                                    <td class="p-2 group-hover:text-white transition-colors truncate max-w-0" title={log.msg}>{log.msg}</td>
                                </tr>
                            )}
                        </For>
                    </tbody>
                </table>

                {(!logs() || logs()?.length === 0) && !logs.loading && (
                    <div class="h-32 flex items-center justify-center opacity-20 italic text-[10px]">
                        NO EVENTS MATCHING CRITERIA
                    </div>
                )}
            </div>

            <div class="p-1 px-3 bg-gray-900 border-t border-gray-800 flex items-center justify-between text-[9px] text-gray-500 uppercase">
                <div class="flex gap-4">
                    <span>HITS: <span class="text-gray-300">{logs()?.length || 0}</span></span>
                    <span>LATENCY: <span class="text-gray-300">42ms</span></span>
                    <span>SHARDS: <span class="text-gray-300">01/01</span></span>
                </div>
                <div class="flex gap-3">
                    <span class="text-blue-500 hover:text-white cursor-pointer transition-colors" onClick={() => refetch()}>REFRESH</span>
                    <span class="text-blue-500 hover:text-white cursor-pointer transition-colors">EXPORT</span>
                </div>
            </div>
        </div>
    );
};
