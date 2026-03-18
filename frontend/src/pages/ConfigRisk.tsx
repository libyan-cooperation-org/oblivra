import { Component, createResource, For, Show } from 'solid-js';
import { GetRiskHistory } from '../../wailsjs/go/services/RiskService';

export const ConfigRisk: Component = () => {
    const [history, { refetch }] = createResource(async () => {
        try {
            return await GetRiskHistory();
        } catch {
            return [];
        }
    });

    const getRiskColor = (score: number) => {
        if (score >= 75) return 'text-red-500 border-red-500/50 bg-red-500/10 shadow-[0_0_15px_rgba(239,68,68,0.2)]';
        if (score >= 50) return 'text-amber-500 border-amber-500/50 bg-amber-500/10 shadow-[0_0_15px_rgba(245,158,11,0.1)]';
        if (score >= 25) return 'text-blue-400 border-blue-400/50 bg-blue-400/10 shadow-[0_0_15px_rgba(96,165,250,0.1)]';
        return 'text-emerald-400 border-emerald-400/50 bg-emerald-400/10 shadow-[0_0_15px_rgba(52,211,153,0.1)]';
    };

    const getBadgeClass = (level: string) => {
        const lvl = level.toLowerCase();
        if (lvl === 'critical') return 'bg-red-500/20 text-red-400 border-red-500/30';
        if (lvl === 'high') return 'bg-amber-500/20 text-amber-400 border-amber-500/30';
        if (lvl === 'medium') return 'bg-blue-500/20 text-blue-400 border-blue-500/30';
        return 'bg-emerald-500/20 text-emerald-400 border-emerald-500/30';
    };

    return (
        <div class="flex flex-col h-full w-full bg-[#0B0D14] text-gray-200 overflow-hidden relative">
            <div class="p-6 border-b border-gray-800/60 flex justify-between items-center bg-[#11131A]/80 backdrop-blur-md z-10">
                <div>
                    <h1 class="text-2xl font-bold tracking-tight text-white font-mono flex items-center gap-3">
                        <svg class="w-6 h-6 text-red-500" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"></path></svg>
                        Configuration Risk Audit
                    </h1>
                    <p class="text-sm text-gray-500 mt-1">Evaluation of blast radius and security posture impact.</p>
                </div>
                
                <button 
                    onClick={refetch}
                    class="px-4 py-2 bg-[#1C202B] hover:bg-[#252A38] text-gray-300 rounded-lg border border-gray-700 text-sm font-medium transition-colors flex items-center gap-2"
                >
                    <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"></path></svg>
                    Refresh Models
                </button>
            </div>

            <div class="flex-1 overflow-y-auto p-6 md:p-8 custom-scrollbar">
                <div class="max-w-5xl mx-auto">
                    
                    {/* Metrics Header */}
                    <div class="grid grid-cols-3 gap-6 mb-8">
                        <div class="bg-[#161923]/60 border border-gray-800 rounded-xl p-5 backdrop-blur-sm">
                            <h3 class="text-xs font-semibold text-gray-500 tracking-wider uppercase mb-1">Total Audits</h3>
                            <div class="text-3xl font-bold text-gray-200 font-mono">{history()?.length || 0}</div>
                        </div>
                        <div class="bg-[#161923]/60 border border-gray-800 rounded-xl p-5 backdrop-blur-sm">
                            <h3 class="text-xs font-semibold text-gray-500 tracking-wider uppercase mb-1">Critical Risks</h3>
                            <div class="text-3xl font-bold text-red-500 font-mono">
                                {history()?.filter(r => r.score >= 75).length || 0}
                            </div>
                        </div>
                        <div class="bg-[#161923]/60 border border-gray-800 rounded-xl p-5 backdrop-blur-sm">
                            <h3 class="text-xs font-semibold text-gray-500 tracking-wider uppercase mb-1">Avg Exposure</h3>
                            <div class="text-3xl font-bold text-amber-500 font-mono">
                                {history()?.length ? Math.round(history()!.reduce((acc, curr) => acc + curr.score, 0) / history()!.length) : 0}
                                <span class="text-sm text-gray-600 ml-1">/ 100</span>
                            </div>
                        </div>
                    </div>

                    <div class="space-y-4 relative before:absolute before:inset-y-0 before:left-[1.2rem] before:w-px before:bg-gray-800 ml-2">
                        
                        <Show when={!history() || history()?.length === 0}>
                            <div class="ml-10 p-8 bg-[#161923]/40 border border-gray-800/80 rounded-xl flex flex-col items-center justify-center text-gray-500 py-16">
                                <svg class="w-12 h-12 mb-4 text-gray-700" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z"></path></svg>
                                <p class="font-mono text-sm">No recent configuration changes logged.</p>
                            </div>
                        </Show>

                        <For each={history()}>
                            {(risk) => (
                                <div class="relative pl-10 pr-4 py-2 group">
                                    {/* Timeline Dot */}
                                    <div class={`absolute left-[1.2rem] top-1/2 -translate-y-1/2 w-3 h-3 rounded-full border-2 border-[#0B0D14] -ml-[0.4rem] z-10 transition-transform group-hover:scale-125 ${getRiskColor(risk.score).split(' ')[1].replace('border-', 'bg-')}`}></div>
                                    
                                    <div class={`bg-[#161923]/80 border-l-[3px] border-r border-t border-b border-gray-800 rounded-xl p-5 transition-all duration-300 hover:translate-x-1 ${getRiskColor(risk.score)}`}>
                                        
                                        <div class="flex justify-between items-start mb-3">
                                            <div class="flex items-center gap-3">
                                                <span class={`px-2.5 py-0.5 rounded text-xs font-mono font-bold uppercase border ${getBadgeClass(risk.level)}`}>
                                                    {risk.level}
                                                </span>
                                                <span class="text-xs text-gray-500 font-mono">ID: {risk.id.substring(0,8)}</span>
                                            </div>
                                            
                                            <div class="flex items-center gap-4 text-xs font-mono">
                                                <div class="flex items-center gap-1">
                                                    <span class="text-gray-500">Score:</span>
                                                    <span class="font-bold text-gray-300">{risk.score}/100</span>
                                                </div>
                                                <span class="text-gray-500 bg-gray-900 px-2 py-1 rounded">
                                                    {new Date(risk.timestamp.toString()).toLocaleString()}
                                                </span>
                                            </div>
                                        </div>
                                        
                                        <h3 class="text-lg font-semibold text-gray-200 mb-4">{risk.reason}</h3>
                                        
                                        <div class="bg-black/30 border border-gray-800/50 rounded-lg p-4">
                                            <h4 class="text-xs font-bold text-gray-500 uppercase tracking-wider mb-2 flex items-center gap-2">
                                                <svg class="w-4 h-4 text-blue-500" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 16h-1v-4h-1m1-4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z"></path></svg>
                                                Impact Analysis
                                            </h4>
                                            <p class="text-sm text-gray-400 leading-relaxed font-mono">
                                                {risk.impact}
                                            </p>
                                        </div>
                                    </div>
                                </div>
                            )}
                        </For>
                    </div>

                </div>
            </div>

            <style>{`
                .custom-scrollbar::-webkit-scrollbar { width: 6px; }
                .custom-scrollbar::-webkit-scrollbar-track { background: transparent; }
                .custom-scrollbar::-webkit-scrollbar-thumb { background-color: rgba(75, 85, 99, 0.4); border-radius: 20px; }
                .custom-scrollbar::-webkit-scrollbar-thumb:hover { background-color: rgba(107, 114, 128, 0.6); }
            `}</style>
        </div>
    );
};
