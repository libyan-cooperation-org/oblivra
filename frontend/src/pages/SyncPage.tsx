import { Component, createSignal, Show, onMount } from 'solid-js';

export const SyncPage: Component = () => {
    const [syncing, setSyncing] = createSignal(false);
    const [result, setResult] = createSignal<string | null>(null);
    const [lastSync, setLastSync] = createSignal<Date | null>(null);
    const [progress, setProgress] = createSignal(0);

    // Mock initial state for polish
    onMount(() => {
        setLastSync(new Date(Date.now() - 1000 * 60 * 45)); // 45 mins ago
    });

    const handleSync = async () => {
        if (syncing()) return;
        setSyncing(true);
        setResult(null);
        setProgress(0);

        // Fake progress animation for UI polish while backend syncs
        const interval = setInterval(() => {
            setProgress(p => (p < 90 ? p + Math.random() * 10 : p));
        }, 200);

        try {
            const { Sync } = await import('../../wailsjs/go/services/SyncService');
            await Sync();
            clearInterval(interval);
            setProgress(100);
            setResult('Synchronization complete');
            setLastSync(new Date());
            
            setTimeout(() => setProgress(0), 2000);
        } catch (e: unknown) {
            clearInterval(interval);
            setProgress(0);
            setResult(`Sync failed: ${(e as Error)?.message || e}`);
        } finally {
            setTimeout(() => setSyncing(false), 500);
        }
    };

    const formatTimeAgo = (date: Date) => {
        const mins = Math.floor((Date.now() - date.getTime()) / 60000);
        if (mins < 1) return 'Just now';
        if (mins < 60) return `${mins}m ago`;
        const hours = Math.floor(mins / 60);
        if (hours < 24) return `${hours}h ago`;
        return date.toLocaleDateString();
    };

    return (
        <div class="flex flex-col h-full w-full bg-[#0B0D14] text-gray-200 overflow-hidden relative">
            <div class="p-6 border-b border-gray-800/60 flex justify-between items-center bg-[#11131A]/80 backdrop-blur-md z-10">
                <div>
                    <h1 class="text-2xl font-bold tracking-tight text-white font-mono flex items-center gap-3">
                        <div class="relative">
                            <svg class={`w-6 h-6 text-sky-400 ${syncing() ? 'animate-spin' : ''}`} fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"></path></svg>
                        </div>
                        Cloud Synchronization
                    </h1>
                    <p class="text-sm text-gray-500 mt-1">End-to-end encrypted cluster replication.</p>
                </div>
                
                <Show when={lastSync()}>
                    <div class="text-sm flex flex-col items-end">
                        <span class="text-gray-500">Last successful sync</span>
                        <span class="text-sky-400 font-mono font-medium">{formatTimeAgo(lastSync()!)}</span>
                    </div>
                </Show>
            </div>

            <div class="flex-1 overflow-y-auto p-8 custom-scrollbar flex flex-col items-center">
                
                {/* Visual Topology Map */}
                <div class="w-full max-w-4xl mt-6 relative h-64 flex items-center justify-center">
                    {/* SVG Connector Lines */}
                    <svg class="absolute inset-0 w-full h-full" style="z-index: 0" preserveAspectRatio="none">
                        <path 
                            d="M 25% 50% L 50% 50% L 75% 50%" 
                            fill="none" 
                            stroke="rgba(56, 189, 248, 0.2)" 
                            stroke-width="2" 
                            stroke-dasharray="6,6"
                            class={syncing() ? 'animate-dash' : ''}
                        />
                    </svg>

                    <div class="w-full flex justify-between items-center px-16 z-10 relative">
                        {/* Local Node */}
                        <div class="flex flex-col items-center gap-3">
                            <div class="w-20 h-20 rounded-2xl bg-gradient-to-br from-gray-800 to-gray-900 border border-gray-700 shadow-xl flex items-center justify-center relative group view-box">
                                <svg class="w-8 h-8 text-gray-300" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9.75 17L9 20l-1 1h8l-1-1-.75-3M3 13h18M5 17h14a2 2 0 002-2V5a2 2 0 00-2-2H5a2 2 0 00-2 2v10a2 2 0 002 2z"></path></svg>
                                <div class="absolute -bottom-1 -right-1 w-4 h-4 rounded-full bg-emerald-500 border-2 border-[#0B0D14]"></div>
                            </div>
                            <span class="text-sm font-semibold text-gray-300 font-mono">Local Node</span>
                            <span class="text-xs text-emerald-500 bg-emerald-500/10 px-2 py-0.5 rounded">Online</span>
                        </div>

                        {/* Sync Hub */}
                        <div class="flex flex-col items-center gap-4 group">
                            <div class={`relative flex items-center justify-center w-32 h-32 rounded-full border-2 
                                ${syncing() ? 'border-sky-500/50 bg-sky-900/20' : 'border-gray-700/50 bg-[#161923]'} 
                                transition-all duration-500`}>
                                
                                <Show when={syncing()}>
                                    <div class="absolute inset-0 rounded-full bg-sky-500/10 animate-ping" style="animation-duration: 2s;"></div>
                                    <div class="absolute inset-0 rounded-full border border-sky-400/30 animate-pulse"></div>
                                </Show>

                                <button 
                                    onClick={handleSync}
                                    disabled={syncing()}
                                    class="w-24 h-24 rounded-full bg-gradient-to-br from-[#1C202B] to-[#11131A] shadow-[0_0_30px_rgba(0,0,0,0.5)] flex items-center justify-center border border-gray-700 hover:border-sky-500/50 transition-all z-10 group-hover:scale-105 active:scale-95 disabled:scale-100 cursor-pointer disabled:cursor-wait"
                                >
                                    <svg class={`w-10 h-10 ${syncing() ? 'text-sky-400 animate-spin' : 'text-gray-400 group-hover:text-sky-300'}`} fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"></path></svg>
                                </button>
                            </div>
                            <span class="text-sm font-semibold text-sky-400 font-mono tracking-wider">
                                {syncing() ? 'SYNCING...' : 'SYNC NOW'}
                            </span>
                        </div>

                        {/* Cloud Target */}
                        <div class="flex flex-col items-center gap-3">
                            <div class="w-20 h-20 rounded-2xl bg-gradient-to-br from-gray-800 to-gray-900 border border-gray-700 shadow-xl flex items-center justify-center relative view-box border-dashed border-sky-900">
                                <svg class="w-8 h-8 text-sky-500" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"></path></svg>
                                <Show when={syncing()}>
                                    <div class="absolute -top-1 -right-1 w-3 h-3 rounded-full bg-sky-500 animate-pulse"></div>
                                </Show>
                            </div>
                            <span class="text-sm font-semibold text-gray-300 font-mono">Sovereign Vault</span>
                            <span class="text-xs text-sky-500 bg-sky-500/10 px-2 py-0.5 rounded">Remote</span>
                        </div>
                    </div>
                </div>

                {/* Progress / Status Bar */}
                <div class="w-full max-w-2xl mt-8">
                    <div class="h-1 w-full bg-gray-800/80 rounded-full overflow-hidden mb-4 relative drop-shadow-md">
                        <div 
                            class="h-full bg-gradient-to-r from-sky-600 to-emerald-400 transition-all duration-300 relative" 
                            style={{ width: `${progress()}%` }}
                        >
                            <Show when={syncing()}>
                                <div class="absolute top-0 right-0 bottom-0 left-0 bg-[linear-gradient(45deg,transparent_25%,rgba(255,255,255,0.2)_50%,transparent_75%,transparent_100%)] bg-[length:20px_20px] animate-stripe"></div>
                            </Show>
                        </div>
                    </div>
                    
                    <div class="flex justify-center min-h-[2rem]">
                        <Show when={result()}>
                            <div class={`px-4 py-1.5 rounded-full text-xs font-medium backdrop-blur-sm border shadow-sm animate-fade-in
                                ${result()!.includes('complete') || result()!.includes('✓') 
                                    ? 'bg-emerald-500/10 text-emerald-400 border-emerald-500/20' 
                                    : 'bg-red-500/10 text-red-400 border-red-500/20'}`}
                            >
                                {result()}
                            </div>
                        </Show>
                    </div>
                </div>

                {/* Data Types Summary Cards */}
                <div class="w-full max-w-4xl mt-16 grid grid-cols-1 md:grid-cols-3 gap-6">
                    <div class="bg-[#161923]/60 border border-gray-800 rounded-xl p-5 hover:border-gray-600 transition-colors">
                        <div class="flex items-center gap-3 mb-4 text-emerald-400">
                            <div class="p-2 bg-emerald-500/10 rounded-lg"><svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 7a2 2 0 012 2m4 0a6 6 0 01-7.743 5.743L11 17H9v2H7v2H4a1 1 0 01-1-1v-2.586a1 1 0 01.293-.707l5.964-5.964A6 6 0 1121 9z"></path></svg></div>
                            <h3 class="font-semibold text-gray-200">Credentials</h3>
                        </div>
                        <p class="text-xs text-gray-500">Vault items, SSH keys, API tokens. Fully encrypted before sync.</p>
                    </div>

                    <div class="bg-[#161923]/60 border border-gray-800 rounded-xl p-5 hover:border-gray-600 transition-colors">
                        <div class="flex items-center gap-3 mb-4 text-purple-400">
                            <div class="p-2 bg-purple-500/10 rounded-lg"><svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"></path></svg></div>
                            <h3 class="font-semibold text-gray-200">Playbooks</h3>
                        </div>
                        <p class="text-xs text-gray-500">Snippets, notes, runbooks, and automations.</p>
                    </div>

                    <div class="bg-[#161923]/60 border border-gray-800 rounded-xl p-5 hover:border-gray-600 transition-colors">
                        <div class="flex items-center gap-3 mb-4 text-sky-400">
                            <div class="p-2 bg-sky-500/10 rounded-lg"><svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 12h14M5 12a2 2 0 01-2-2V6a2 2 0 012-2h14a2 2 0 012 2v4a2 2 0 01-2 2M5 12a2 2 0 00-2 2v4a2 2 0 002 2h14a2 2 0 002-2v-4a2 2 0 00-2-2m-2-4h.01M17 16h.01"></path></svg></div>
                            <h3 class="font-semibold text-gray-200">Configuration</h3>
                        </div>
                        <p class="text-xs text-gray-500">Terminal preferences, layout profiles, connected endpoints.</p>
                    </div>
                </div>

            </div>

            <style>{`
                .custom-scrollbar::-webkit-scrollbar { width: 6px; }
                .custom-scrollbar::-webkit-scrollbar-track { background: transparent; }
                .custom-scrollbar::-webkit-scrollbar-thumb { background-color: rgba(75, 85, 99, 0.4); border-radius: 20px; }
                .custom-scrollbar::-webkit-scrollbar-thumb:hover { background-color: rgba(107, 114, 128, 0.6); }
                
                @keyframes dash {
                    to { stroke-dashoffset: -12; }
                }
                .animate-dash { animation: dash 0.5s linear infinite; }
                
                @keyframes stripe {
                    100% { background-position: 20px 0; }
                }
                .animate-stripe { animation: stripe 1s linear infinite; }
                
                @keyframes fadeIn {
                    from { opacity: 0; transform: translateY(5px); }
                    to { opacity: 1; transform: translateY(0); }
                }
                .animate-fade-in { animation: fadeIn 0.3s ease-out forwards; }
            `}</style>
        </div>
    );
};
