import { Component, createSignal, onMount, onCleanup, For, Show } from 'solid-js';
import { GetActiveCampaigns } from '../../wailsjs/go/services/FusionService';
import { Card, Badge } from '../components/ui/TacticalComponents';

// MITRE Tactic Order for Kill Chain Visualization
const TACTIC_ORDER = [
    { id: "TA0001", name: "Initial Access" },
    { id: "TA0002", name: "Execution" },
    { id: "TA0003", name: "Persistence" },
    { id: "TA0004", name: "Privilege Escalation" },
    { id: "TA0005", name: "Defense Evasion" },
    { id: "TA0006", name: "Credential Access" },
    { id: "TA0007", name: "Discovery" },
    { id: "TA0008", name: "Lateral Movement" },
    { id: "TA0009", name: "Collection" },
    { id: "TA0011", name: "Command and Control" },
    { id: "TA0010", name: "Exfiltration" },
    { id: "TA0040", name: "Impact" }
];

export const FusionDashboard: Component = () => {
    const [campaigns, setCampaigns] = createSignal<any[]>([]);
    const [loading, setLoading] = createSignal(true);

    const refresh = async () => {
        try {
            const res = await GetActiveCampaigns();
            setCampaigns(res || []);
        } catch (err) {
            console.error("Failed to fetch fusion campaigns:", err);
        } finally {
            setLoading(false);
        }
    };

    onMount(() => {
        refresh();
        const interval = setInterval(refresh, 5000);
        onCleanup(() => clearInterval(interval));
    });

    const getProbabilityColor = (prob: number) => {
        if (prob > 0.8) return "text-red-500";
        if (prob > 0.5) return "text-orange-500";
        if (prob > 0.2) return "text-yellow-500";
        return "text-blue-400";
    };

    return (
        <div class="p-6 h-full overflow-auto bg-[#0d1117] text-gray-200">
            <header class="mb-8 flex justify-between items-end border-b border-gray-800 pb-4">
                <div>
                    <h1 class="text-3xl font-bold tracking-tighter text-white">ATTACK FUSION EXPLORER</h1>
                    <p class="text-gray-400 mt-1">Probabilistic Kill Chain Correlation (Phase 10.6)</p>
                </div>
                <div class="text-right">
                    <Badge severity="info" class="border-blue-500/50 text-blue-400">BAYESIAN SCORING ACTIVE</Badge>
                </div>
            </header>

            <div class="grid grid-cols-1 gap-6">
                <Show when={loading() && campaigns().length === 0}>
                    <div class="flex items-center justify-center h-64 italic text-gray-500 border border-dashed border-gray-800 rounded-lg">
                        Initalizing Fusion Engine...
                    </div>
                </Show>

                <Show when={!loading() && campaigns().length === 0}>
                    <div class="flex items-center justify-center h-64 italic text-gray-500 border border-dashed border-gray-800 rounded-lg">
                        No active multi-stage campaigns detected. All systems nominal.
                    </div>
                </Show>

                <For each={campaigns().sort((a, b) => b.probability - a.probability)}>
                    {(camp) => (
                        <Card class="bg-gray-900/40 border-gray-800 overflow-hidden relative group">
                            {/* Probabilistic Glow */}
                            <div 
                                class="absolute top-0 right-0 w-32 h-32 blur-3xl opacity-10 pointer-events-none transition-all duration-1000"
                                style={{ background: camp.probability > 0.7 ? 'red' : 'blue' }}
                            />

                            <div class="p-5 flex flex-col gap-6">
                                <div class="flex justify-between items-start">
                                    <div>
                                        <div class="flex items-center gap-3">
                                            <h3 class="text-xl font-mono font-bold text-white">{camp.entity_id}</h3>
                                            <Show when={camp.is_triggered}>
                                                <Badge severity="error" class="bg-red-900/50 text-red-100 border-red-500 animate-pulse">
                                                    CRITICAL FUSION
                                                </Badge>
                                            </Show>
                                        </div>
                                        <div class="text-sm text-gray-400 mt-1 font-mono uppercase tracking-widest">
                                            Campaign Start: {new Date(camp.first_seen).toLocaleString()}
                                        </div>
                                    </div>

                                    <div class="text-right">
                                        <p class="text-xs text-gray-500 uppercase font-bold tracking-tighter">Certainty Score</p>
                                        <p class={`text-4xl font-mono font-black ${getProbabilityColor(camp.probability)}`}>
                                            {(camp.probability * 100).toFixed(1)}%
                                        </p>
                                    </div>
                                </div>

                                {/* Kill Chain Visualization */}
                                <div class="relative py-8">
                                    <div class="absolute top-1/2 left-0 w-full h-0.5 bg-gray-800 -translate-y-1/2" />
                                    <div class="flex justify-between relative z-10 w-full">
                                        <For each={TACTIC_ORDER}>
                                            {(tactic) => {
                                                const active = camp.tactics && camp.tactics[tactic.id];
                                                return (
                                                    <div class="flex flex-col items-center group/tactic">
                                                        <div 
                                                            class={`w-4 h-4 rounded-full border-2 transition-all duration-500 ${
                                                                active 
                                                                ? 'bg-blue-500 border-white shadow-[0_0_15px_rgba(59,130,246,0.8)]' 
                                                                : 'bg-gray-900 border-gray-700'
                                                            }`}
                                                        />
                                                        <div class={`mt-3 text-[10px] font-mono tracking-tighter uppercase whitespace-pre transition-colors duration-300 ${
                                                                active ? 'text-blue-300 font-bold' : 'text-gray-600'
                                                            }`}>
                                                            {tactic.name.replace(' ', '\n')}
                                                        </div>
                                                    </div>
                                                );
                                            }}
                                        </For>
                                    </div>
                                </div>

                                {/* Alert Chain */}
                                <div>
                                    <h4 class="text-xs font-bold text-gray-500 uppercase tracking-widest mb-3 flex items-center gap-2">
                                        Contributing Evidence
                                        <div class="h-px flex-1 bg-gray-800" />
                                    </h4>
                                    <div class="space-y-2 max-h-48 overflow-y-auto pr-2 custom-scrollbar">
                                        <For each={camp.alerts}>
                                            {(alert) => (
                                                <div class="flex items-center justify-between text-xs font-mono bg-black/40 p-2 rounded border border-gray-800/50 hover:border-gray-700 transition-colors">
                                                    <div class="flex items-center gap-4">
                                                        <span class="text-gray-600">[{new Date(alert.timestamp).toLocaleTimeString()}]</span>
                                                        <span class="text-blue-400 font-bold">{alert.tactic}</span>
                                                        <span class="text-gray-300 italic">{alert.name}</span>
                                                    </div>
                                                    <span class="text-gray-600 text-[10px]">{alert.rule_id}</span>
                                                </div>
                                            )}
                                        </For>
                                    </div>
                                </div>
                            </div>
                        </Card>
                    )}
                </For>
            </div>
        </div>
    );
};
