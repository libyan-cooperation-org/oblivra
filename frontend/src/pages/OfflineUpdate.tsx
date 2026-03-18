import { Component, createSignal, Show, onCleanup } from 'solid-js';

export const OfflineUpdate: Component = () => {
    const [path, setPath] = createSignal('');
    const [running, setRunning] = createSignal(false);
    const [phase, setPhase] = createSignal<string>('');
    const [progress, setProgress] = createSignal(0);
    const [success, setSuccess] = createSignal(false);
    const [errorMsg, setErrorMsg] = createSignal('');

    let intervalId: any;

    onCleanup(() => {
        clearInterval(intervalId);
    });

    const triggerUpload = async () => {
        if (!path().trim()) {
            setErrorMsg("Please specify a valid bundle path.");
            return;
        }
        
        setRunning(true);
        setSuccess(false);
        setErrorMsg('');
        setProgress(0);
        
        // Mocking the air-gapped update sequence
        const phases = [
            { name: "Verifying SHA256 sidecar signatures...", duration: 1500, target: 20 },
            { name: "Decrypting payload manifest...", duration: 1200, target: 35 },
            { name: "Checking downgrading prevention...", duration: 800, target: 45 },
            { name: "Staging new binaries...", duration: 2500, target: 80 },
            { name: "Committing A/B partition swap...", duration: 1500, target: 100 }
        ];

        let currentPhase = 0;

        const runPhase = () => {
            if (currentPhase >= phases.length) {
                setPhase("Offline Update applied successfully. Reboot required to switch active partition.");
                setRunning(false);
                setSuccess(true);
                return;
            }

            const p = phases[currentPhase];
            setPhase(p.name);
            
            const startProgress = progress();
            const distance = p.target - startProgress;
            const steps = 10;
            const stepTime = p.duration / steps;
            let currentStep = 0;

            intervalId = setInterval(() => {
                currentStep++;
                setProgress(startProgress + (distance * (currentStep / steps)));
                
                if (currentStep >= steps) {
                    clearInterval(intervalId);
                    currentPhase++;
                    runPhase();
                }
            }, stepTime);
        };

        runPhase();
    };

    const handleBrowseClick = async () => {
        // If Wails runtime is available, use it. Otherwise, mock it.
        try {
            // @ts-ignore
            if (window.runtime && window.runtime.OpenFileDialog) {
                // @ts-ignore
                const selected = await window.runtime.OpenFileDialog({
                    Title: 'Select Offline Update Bundle',
                    Filters: [{ Name: 'Oblivra Bundles', Pattern: '*.ovb;*.zip;*.tar.gz' }]
                });
                if (selected) setPath(selected);
            } else {
                // Fallback for browser dev mode
                setErrorMsg("Native file dialog requires desktop runtime.");
            }
        } catch (e) {
            console.error("File dialog error:", e);
        }
    };

    return (
        <div class="flex flex-col h-full w-full bg-[#0B0D14] text-gray-200 overflow-hidden relative">
            <div class="p-6 border-b border-gray-800/60 flex justify-between items-center bg-[#11131A]/80 backdrop-blur-md z-10">
                <div>
                    <h1 class="text-2xl font-bold tracking-tight text-white font-mono flex items-center gap-3">
                        <svg class="w-6 h-6 text-cyan-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 11H5m14 0a2 2 0 012 2v6a2 2 0 01-2 2H5a2 2 0 01-2-2v-6a2 2 0 012-2m14 0V9a2 2 0 00-2-2M5 11V9a2 2 0 012-2m0 0V5a2 2 0 012-2h6a2 2 0 012 2v2M7 7h10"></path></svg>
                        Air-Gapped Update
                    </h1>
                    <p class="text-sm text-gray-500 mt-1">Deploy cryptographic bundles to isolated sovereign nodes.</p>
                </div>
                
                <div class="flex items-center gap-2 px-3 py-1.5 bg-red-500/10 border border-red-500/20 rounded-md text-red-500 font-mono text-xs font-semibold tracking-widest uppercase">
                    <span class="w-2 h-2 rounded-full bg-red-500 animate-pulse"></span>
                    Isolated Environment
                </div>
            </div>

            <div class="flex-1 overflow-y-auto p-8 custom-scrollbar flex flex-col items-center">
                
                <div class="w-full max-w-3xl mt-6">
                    
                    {/* Main Interaction Area */}
                    <div class="bg-[#161923]/80 border border-gray-700/80 rounded-2xl overflow-hidden shadow-2xl relative backdrop-blur-sm">
                        
                        {/* Status Header */}
                        <div class="p-5 border-b border-gray-800/80 bg-black/20 flex justify-between items-center group">
                            <div class="flex items-center gap-3">
                                <div class="w-10 h-10 rounded-full bg-cyan-500/10 border border-cyan-500/30 flex items-center justify-center text-cyan-400">
                                    <svg class="w-5 h-5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-8l-4-4m0 0L8 8m4-4v12"></path></svg>
                                </div>
                                <div>
                                    <h2 class="text-lg font-semibold text-gray-100">Bundle Importer</h2>
                                    <p class="text-xs text-gray-400">Secure payload verification & staging</p>
                                </div>
                            </div>
                        </div>

                        <div class="p-8 space-y-8">
                            
                            <Show when={errorMsg()}>
                                <div class="p-4 bg-red-500/10 border border-red-500/20 rounded-lg text-red-400 text-sm flex items-start gap-3 animate-fade-in">
                                    <svg class="w-5 h-5 shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 9v2m0 4h.01m-6.938 4h13.856c1.54 0 2.502-1.667 1.732-3L13.732 4c-.77-1.333-2.694-1.333-3.464 0L3.34 16c-.77 1.333.192 3 1.732 3z"></path></svg>
                                    <span>{errorMsg()}</span>
                                </div>
                            </Show>

                            {/* Input form */}
                            <div class={`transition-all duration-500 ${running() || success() ? 'opacity-50 pointer-events-none' : 'opacity-100'}`}>
                                <label class="block text-xs font-mono font-medium text-gray-400 mb-2 uppercase tracking-wider">Signed Update Package (.ovb, .tar.gz)</label>
                                <div class="flex gap-3">
                                    <div class="relative flex-1">
                                        <div class="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                                            <svg class="w-5 h-5 text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z"></path></svg>
                                        </div>
                                        <input
                                            type="text"
                                            value={path()}
                                            onInput={(e) => setPath(e.currentTarget.value)}
                                            placeholder="/mnt/media/updates/node-v2.1.0-signed.ovb"
                                            class="w-full bg-[#090A0F] border border-gray-700/80 rounded-lg pl-10 pr-4 py-2.5 text-sm font-mono text-gray-200 focus:outline-none focus:border-cyan-500/50 focus:ring-1 focus:ring-cyan-500/50 transition-all shadow-inner"
                                            disabled={running() || success()}
                                        />
                                    </div>
                                    <button 
                                        onClick={handleBrowseClick}
                                        class="px-4 py-2.5 bg-[#1C202B] hover:bg-[#252A38] border border-gray-700 text-gray-300 rounded-lg text-sm font-medium transition-colors"
                                        disabled={running() || success()}
                                    >
                                        Browse
                                    </button>
                                </div>
                            </div>
                            
                            <Show when={running() || success() || phase()}>
                                {/* Interactive Terminal/Status Display */}
                                <div class="bg-[#090A0F] border border-gray-800 rounded-lg p-5 font-mono text-sm shadow-inner relative overflow-hidden animate-fade-in">
                                    <div class="flex justify-between items-end mb-4 relative z-10">
                                        <span class={`font-semibold ${success() ? 'text-emerald-400' : 'text-cyan-400'}`}>
                                            {success() ? '>_ UPGRADE_COMPLETE' : '>_ SYSTEM_UPGRADE_INITIALIZED'}
                                        </span>
                                        <span class="text-gray-500 text-xs">{Math.round(progress())}%</span>
                                    </div>
                                    
                                    {/* Progress Bar Container */}
                                    <div class="h-1.5 w-full bg-gray-900 rounded-full overflow-hidden mb-4 relative z-10">
                                        <div 
                                            class={`h-full transition-all duration-300 ${success() ? 'bg-emerald-500' : 'bg-cyan-500'}`}
                                            style={{ width: `${progress()}%` }}
                                        >
                                            <Show when={running()}>
                                                <div class="absolute top-0 right-0 bottom-0 left-0 bg-[linear-gradient(45deg,transparent_25%,rgba(0,0,0,0.2)_50%,transparent_75%,transparent_100%)] bg-[length:20px_20px] animate-stripe"></div>
                                            </Show>
                                        </div>
                                    </div>
                                    
                                    <div class="text-gray-400 relative z-10 min-h-[1.5rem] flex items-center">
                                        <Show when={running()}>
                                            <svg class="w-4 h-4 mr-2 animate-spin text-cyan-500" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"></path></svg>
                                        </Show>
                                        <Show when={success()}>
                                            <svg class="w-4 h-4 mr-2 text-emerald-500" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7"></path></svg>
                                        </Show>
                                        <span class={success() ? 'text-emerald-400/80 font-semibold' : 'opacity-80'}>{phase()}</span>
                                    </div>

                                    {/* Decorative background grid */}
                                    <div class="absolute inset-0 z-0 opacity-[0.03] pointer-events-none" style="background-image: linear-gradient(rgba(255, 255, 255, 0.5) 1px, transparent 1px), linear-gradient(90deg, rgba(255, 255, 255, 0.5) 1px, transparent 1px); background-size: 20px 20px;"></div>
                                </div>
                            </Show>

                            <div class="pt-4 flex justify-end">
                                <Show when={success()} fallback={
                                    <button
                                        class="px-6 py-2.5 bg-cyan-600 hover:bg-cyan-500 text-white rounded-lg border border-cyan-500 font-medium transition-all shadow-[0_0_15px_rgba(6,182,212,0.2)] hover:shadow-[0_0_20px_rgba(6,182,212,0.4)] disabled:opacity-50 disabled:cursor-not-allowed disabled:hover:shadow-[0_0_15px_rgba(6,182,212,0.2)] disabled:hover:bg-cyan-600 flex items-center gap-2"
                                        disabled={running() || !path().trim()}
                                        onClick={triggerUpload}
                                    >
                                        <Show when={running()} fallback={
                                            <><svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z"></path></svg> Verify & Apply</>
                                        }>
                                            <div class="w-4 h-4 border-2 border-white/30 border-t-white rounded-full animate-spin"></div> Deploying...
                                        </Show>
                                    </button>
                                }>
                                    <button
                                        class="px-6 py-2.5 bg-emerald-600 hover:bg-emerald-500 text-white rounded-lg border border-emerald-500 font-medium transition-all shadow-[0_0_15px_rgba(16,185,129,0.2)] hover:shadow-[0_0_20px_rgba(16,185,129,0.4)] flex items-center gap-2"
                                        onClick={() => {
                                            setPath('');
                                            setPhase('');
                                            setSuccess(false);
                                            setProgress(0);
                                        }}
                                    >
                                        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 4v5h.582m15.356 2A8.001 8.001 0 004.582 9m0 0H9m11 11v-5h-.581m0 0a8.003 8.003 0 01-15.357-2m15.357 2H15"></path></svg> Reset Deployer
                                    </button>
                                </Show>
                            </div>

                        </div>
                    </div>
                    
                    {/* Security Info Cards */}
                    <div class="grid grid-cols-1 md:grid-cols-2 gap-4 mt-6">
                        <div class="bg-black/20 border border-gray-800/80 rounded-xl p-4 flex gap-4">
                            <div class="w-10 h-10 rounded-full bg-indigo-500/10 border border-indigo-500/20 flex items-center justify-center shrink-0">
                                <svg class="w-5 h-5 text-indigo-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M9 12l2 2 4-4m5.618-4.016A11.955 11.955 0 0112 2.944a11.955 11.955 0 01-8.618 3.04A12.02 12.02 0 003 9c0 5.591 3.824 10.29 9 11.622 5.176-1.332 9-6.03 9-11.622 0-1.042-.133-2.052-.382-3.016z"></path></svg>
                            </div>
                            <div>
                                <h3 class="text-sm font-semibold text-gray-200 mb-1">Cryptographically Verified</h3>
                                <p class="text-xs text-gray-500 leading-relaxed">All payloads must match SHA-256 signatures derived from Sovereign quorum verification paths.</p>
                            </div>
                        </div>
                        <div class="bg-black/20 border border-gray-800/80 rounded-xl p-4 flex gap-4">
                            <div class="w-10 h-10 rounded-full bg-amber-500/10 border border-amber-500/20 flex items-center justify-center shrink-0">
                                <svg class="w-5 h-5 text-amber-500" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V7M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4M4 7c0-2.21 3.582-4 8-4s8 1.79 8 4m0 5c0 2.21-3.582 4-8 4s-8-1.79-8-4"></path></svg>
                            </div>
                            <div>
                                <h3 class="text-sm font-semibold text-gray-200 mb-1">A/B Partitioning</h3>
                                <p class="text-xs text-gray-500 leading-relaxed">Updates stage to a standby partition. Failed boots cleanly revert to the last working state.</p>
                            </div>
                        </div>
                    </div>

                </div>
            </div>

            <style>{`
                .custom-scrollbar::-webkit-scrollbar { width: 6px; }
                .custom-scrollbar::-webkit-scrollbar-track { background: transparent; }
                .custom-scrollbar::-webkit-scrollbar-thumb { background-color: rgba(75, 85, 99, 0.4); border-radius: 20px; }
                .custom-scrollbar::-webkit-scrollbar-thumb:hover { background-color: rgba(107, 114, 128, 0.6); }
                
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
