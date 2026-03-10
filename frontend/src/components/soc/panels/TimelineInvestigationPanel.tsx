import { Component, For, createResource, onMount } from 'solid-js';
import { ListIncidents } from '../../../../wailsjs/go/app/IncidentService';

export const TimelineInvestigationPanel: Component = () => {
    const [incidents, { refetch }] = createResource(async () => {
        try {
            // @ts-ignore - handling potential wails context arg mismatch
            const data = await ListIncidents("", "", 50);
            return data.map((inc: any) => ({
                id: inc.ID || Math.random(),
                time: inc.CreatedAt ? new Date(inc.CreatedAt).toLocaleTimeString() : 'N/A',
                type: inc.Type || 'DETECTION',
                title: inc.Title || 'Security Event',
                state: inc.Status === 'OPEN' ? 'critical' : 'warning',
                description: inc.Description || 'System initiated investigation'
            }));
        } catch (e) {
            console.error("Failed to fetch incidents:", e);
            return [];
        }
    });

    onMount(() => {
        const interval = setInterval(refetch, 15000);
        return () => clearInterval(interval);
    });

    return (
        <div class="h-full flex flex-col bg-gray-950 font-mono text-[11px] overflow-hidden">
            <div class="p-3 border-b border-gray-800 flex justify-between items-center bg-gray-900/30">
                <span class="text-gray-400 font-bold tracking-widest uppercase">Incident Chronology</span>
                {incidents.loading && <span class="text-blue-500 animate-pulse text-[9px]">SYNCING...</span>}
            </div>

            <div class="flex-1 overflow-auto p-4 relative">
                <div class="absolute left-[23px] top-0 bottom-0 w-px bg-gray-800"></div>

                <div class="space-y-6">
                    <For each={incidents() || []}>
                        {(event) => (
                            <div class="relative pl-10 group">
                                <div class={`absolute left-0 top-1 w-4 h-4 rounded-full border-2 border-gray-950 z-10 
                                    ${event.state === 'critical' ? 'bg-red-600 shadow-[0_0_8px_rgba(220,38,38,0.5)]' : 'bg-orange-500'}
                                `}></div>

                                <div class="flex flex-col gap-1">
                                    <div class="flex items-center gap-3">
                                        <span class="text-gray-500 text-[10px]">{event.time}</span>
                                        <span class="px-1.5 py-0.5 bg-gray-900 border border-gray-800 text-[9px] font-bold uppercase tracking-tight text-gray-400">
                                            {event.type}
                                        </span>
                                    </div>
                                    <div class="text-white font-bold text-[12px] group-hover:text-blue-400 transition-colors cursor-pointer capitalize">
                                        {event.title}
                                    </div>
                                    <div class="text-gray-500 text-[10px] leading-relaxed italic max-w-md">
                                        {event.description}
                                    </div>
                                </div>
                            </div>
                        )}
                    </For>

                    {(!incidents() || incidents()?.length === 0) && !incidents.loading && (
                        <div class="py-10 text-center opacity-30 italic">
                            NO SIGNIFICANT INCIDENTS RECORDED
                        </div>
                    )}
                </div>
            </div>

            <div class="p-2 border-t border-gray-800 bg-gray-900/50 flex justify-end">
                <button class="text-[9px] text-blue-500 font-bold uppercase tracking-widest hover:text-white transition-colors">Incident Manager</button>
            </div>
        </div>
    );
};
