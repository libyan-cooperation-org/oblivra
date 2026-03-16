import { Component, createSignal, onMount, For, Show } from 'solid-js';

interface TrustStatus {
    component: string;
    status: string;
    detail: string;
    last_check: string;
}

export const RuntimeTrust: Component = () => {
    const [score, setScore] = createSignal<number>(0);
    const [components, setComponents] = createSignal<TrustStatus[]>([]);
    const [loading, setLoading] = createSignal(true);

    const loadTrustState = async () => {
        setLoading(true);
        try {
            const svc = (window as any).go?.services?.RuntimeTrustService;
            if (!svc) return;

            // Force a refresh
            await svc.VerifyIntegrity();

            const [newScore, newComponents] = await Promise.all([
                svc.CalculateTrustIndex(),
                svc.GetAggregatedStatus()
            ]);

            setScore(newScore);
            setComponents(newComponents || []);
        } catch (err) {
            console.error("Failed to load trust index:", err);
        } finally {
            setLoading(false);
        }
    };

    onMount(() => {
        loadTrustState();
    });

    const getScoreColor = (sc: number) => {
        if (sc >= 90) return 'text-emerald-400 drop-shadow-[0_0_8px_rgba(52,211,153,0.8)]';
        if (sc >= 70) return 'text-amber-400 drop-shadow-[0_0_8px_rgba(251,191,36,0.8)]';
        return 'text-red-500 drop-shadow-[0_0_8px_rgba(239,68,68,0.8)]';
    };

    const getStatusColor = (status: string) => {
        switch (status) {
            case 'TRUSTED': return 'text-emerald-400 bg-emerald-950/30 border-emerald-500/50';
            case 'WARNING': return 'text-amber-400 bg-amber-950/30 border-amber-500/50';
            case 'UNTRUSTED': return 'text-red-400 bg-red-950/30 border-red-500/50';
            default: return 'text-gray-400 bg-gray-900 border-gray-700';
        }
    };

    return (
        <div class="p-8 h-full flex flex-col bg-gray-950 text-gray-100 font-mono">
            <div class="flex justify-between items-center mb-10 border-b border-gray-800 pb-6">
                <div>
                    <h1 class="text-3xl font-black tracking-widest text-slate-100">ADAPTIVE TRUST WEIGHTING</h1>
                    <p class="text-slate-500 mt-2 font-bold tracking-widest text-sm">GLOBAL MACHINE INTEGRITY CONSENSUS</p>
                </div>
                <button
                    onClick={loadTrustState}
                    disabled={loading()}
                    class="px-6 py-3 bg-slate-900 border border-slate-700 hover:border-emerald-500 transition-colors uppercase tracking-widest text-sm font-bold disabled:opacity-50"
                >
                    {loading() ? 'CALCULATING...' : 'RE-VERIFY INVARIANTS'}
                </button>
            </div>

            <div class="grid grid-cols-1 lg:grid-cols-3 gap-8 h-full">
                {/* Score Widget */}
                <div class="lg:col-span-1 flex flex-col items-center justify-center p-10 border border-slate-800 bg-slate-900/50 relative overflow-hidden">
                    <div class="absolute top-0 left-0 w-full h-1 bg-gradient-to-r from-transparent via-emerald-500 to-transparent opacity-50"></div>
                    <div class="text-slate-500 font-bold tracking-[0.2em] mb-4">GLOBAL TRUST SCORE</div>

                    <div class="relative flex items-center justify-center">
                        <svg class="w-64 h-64 transform -rotate-90">
                            <circle cx="128" cy="128" r="110" stroke="currentColor" stroke-width="8" fill="transparent" class="text-slate-800" />
                            <circle
                                cx="128" cy="128" r="110"
                                stroke="currentColor"
                                stroke-width="8"
                                fill="transparent"
                                stroke-dasharray={String(110 * 2 * Math.PI)}
                                stroke-dashoffset={String(110 * 2 * Math.PI * (1 - (score() / 100)))}
                                class={`transition-all duration-1000 ease-out ${getScoreColor(score()).split(' ')[0]}`}
                            />
                        </svg>
                        <div class="absolute inset-0 flex flex-col items-center justify-center">
                            <span class={`text-7xl font-black ${getScoreColor(score())}`}>
                                {score().toFixed(1)}
                            </span>
                            <span class="text-slate-500 text-sm mt-1">/ 100.0</span>
                        </div>
                    </div>

                    <div class="mt-8 text-center text-sm text-slate-400 px-4">
                        Machine runtime integrity calculated using a deterministic 5-factor weighting model.
                    </div>
                </div>

                {/* Pillars Grid */}
                <div class="lg:col-span-2 grid grid-cols-1 md:grid-cols-2 gap-4">
                    <For each={components().sort((a, b) => a.component.localeCompare(b.component))}>
                        {(comp) => (
                            <div class={`p-6 border border-l-4 ${getStatusColor(comp.status)} flex flex-col justify-between`}>
                                <div>
                                    <div class="flex justify-between items-start mb-4">
                                        <h3 class="font-bold text-lg">{comp.component}</h3>
                                        <span class="text-xs px-2 py-1 font-bold tracking-widest border border-current">
                                            {comp.status}
                                        </span>
                                    </div>
                                    <p class="text-sm opacity-80 h-10 overflow-hidden text-ellipsis line-clamp-2">{comp.detail}</p>
                                </div>
                                <div class="mt-4 text-xs opacity-50 flex items-center gap-2">
                                    <svg viewBox="0 0 24 24" width="12" height="12" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="10" /><polyline points="12 6 12 12 16 14" /></svg>
                                    Last verified: {new Date(comp.last_check).toLocaleTimeString()}
                                </div>
                            </div>
                        )}
                    </For>
                    <Show when={components().length === 0 && !loading()}>
                        <div class="md:col-span-2 flex items-center justify-center p-12 border border-slate-800 text-slate-500 border-dashed">
                            No trust pillars reported yet. Awaiting first invariant pass.
                        </div>
                    </Show>
                </div>
            </div>
        </div>
    );
};
