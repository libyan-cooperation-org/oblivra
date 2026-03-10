import { Component, For, createResource, onMount } from 'solid-js';
import { GetAggregatedStatus } from '../../../../wailsjs/go/app/RuntimeTrustService';
import { GetTrustDriftMetrics } from '../../../../wailsjs/go/app/App';

export const HardwareTrustPanel: Component = () => {
    const [trustData, { refetch }] = createResource(async () => {
        try {
            const status = await GetAggregatedStatus();
            const metrics = await GetTrustDriftMetrics();
            return { status, index: metrics.current_score, metrics };
        } catch (e) {
            console.error("Failed to fetch trust status:", e);
            return { status: [], index: 0, metrics: null };
        }
    });

    onMount(() => {
        const interval = setInterval(refetch, 30000);
        return () => clearInterval(interval);
    });

    const getStatusColor = (status: string) => {
        switch (status) {
            case 'TRUSTED': return 'text-green-500';
            case 'WARNING': return 'text-yellow-500';
            case 'UNTRUSTED': return 'text-red-500';
            default: return 'text-gray-500';
        }
    };

    const getIndexColor = (index: number) => {
        if (index > 90) return 'text-green-500';
        if (index > 70) return 'text-yellow-500';
        return 'text-red-500';
    };

    return (
        <div class="h-full flex flex-col bg-surface-0 font-mono text-[11px] overflow-hidden">
            <div class="p-3 border-b border-border-primary flex justify-between items-center bg-surface-1 px-4">
                <span class="text-text-muted font-bold tracking-widest uppercase">Platform Integrity</span>
                <div class="flex items-center gap-4">
                    {trustData()?.metrics && (
                        <div class={`flex flex-col items-end ${trustData()!.metrics.is_bleeding ? 'animate-pulse text-red-500' : 'text-text-muted opacity-80'}`}>
                            <span class="text-[8px] tracking-widest uppercase font-bold">Trend: {trustData()!.metrics.velocity_per_hour > 0 ? '+' : ''}{trustData()!.metrics.velocity_per_hour.toFixed(2)}/hr</span>
                            {trustData()!.metrics.is_bleeding && (
                                <span class="text-[9px] font-black uppercase tracking-tighter text-red-400">ETTF: {trustData()!.metrics.estimated_failure_time}</span>
                            )}
                        </div>
                    )}
                    <div class="flex items-center gap-2">
                        <span class="text-[9px] text-text-muted opacity-60 uppercase">Trust Index:</span>
                        <span class={`font-black text-sm ${getIndexColor(trustData()?.index || 0)}`}>
                            {trustData()?.index ? trustData()!.index.toFixed(1) : '0.0'}%
                        </span>
                    </div>
                </div>
            </div>

            <div class="flex-1 overflow-auto p-3 space-y-3">
                {/* Visual Index Bar */}
                <div class="bg-surface-1 h-2 w-full overflow-hidden border border-border-primary">
                    <div
                        class={`h-full transition-all duration-1000 ${trustData()?.index && trustData()!.index > 80 ? 'bg-green-600' : 'bg-yellow-600'}`}
                        style={{ width: `${trustData()?.index || 0}%` }}
                    ></div>
                </div>

                <div class="grid gap-2">
                    <For each={trustData()?.status || []}>
                        {(item) => (
                            <div class="p-2 bg-surface-1 border border-border-primary flex flex-col gap-1 group hover:border-accent-primary transition-colors">
                                <div class="flex justify-between items-center">
                                    <span class="text-gray-300 font-bold uppercase tracking-tighter">{item.component}</span>
                                    <span class={`text-[9px] font-black ${getStatusColor(item.status)}`}>{item.status}</span>
                                </div>
                                <div class="text-[9px] text-gray-500 italic leading-tight uppercase opacity-70 group-hover:opacity-100 transition-opacity">
                                    {item.detail}
                                </div>
                                <div class="text-[8px] text-gray-700 font-mono mt-1">
                                    Last Checked: {new Date(item.last_check).toLocaleTimeString()}
                                </div>
                            </div>
                        )}
                    </For>
                </div>
            </div>

            <div class="p-2 border-t border-border-primary bg-surface-1 flex justify-between items-center px-4">
                <span class="text-[9px] text-text-muted">ATTESTATION: <span class="text-accent-primary">HARDWARE-BOUND</span></span>
                <button
                    onClick={refetch}
                    class="text-[9px] text-accent-primary font-bold hover:text-white transition-colors uppercase"
                >
                    Re-Verify
                </button>
            </div>
        </div>
    );
};
