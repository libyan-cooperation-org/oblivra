import { Component, createSignal, onMount, For, Show } from 'solid-js';

// DeterministicResponseService interface via Wails
interface ExecutionSignature {
    id: string;
    timestamp: string;
    event_batch_id: string;
    policy_hash: string;
    action_taken: string;
    input_hash: string;
    final_hash: string;
}

export const ResponseReplay: Component = () => {
    const [signatures, setSignatures] = createSignal<ExecutionSignature[]>([]);

    // Testing fields
    const [inputAction, setInputAction] = createSignal('IsolateHost');
    const [inputEvent, setInputEvent] = createSignal('{"event_id": "EVT-123", "type": "LateralMovement"}');
    const [inputPolicy, setInputPolicy] = createSignal('8e3a2b4cd9');

    const [replayData, setReplayData] = createSignal<any>(null);

    const loadSigs = async () => {
        try {
            const svc = (window as any).go?.app?.DeterministicResponseService;
            if (svc) {
                const sigs = await svc.GetSignatures();
                setSignatures(sigs || []);
            }
        } catch (err) {
            console.error(err);
        }
    };

    onMount(() => {
        loadSigs();
    });

    const runSimulation = async () => {
        setReplayData(null);
        try {
            const svc = (window as any).go?.app?.DeterministicResponseService;
            if (svc) {
                // To simulate hashing a payload from the UI without doing sha256 in JS,
                // we'll fetch the replay via Wails. But wait, Wails `Replay` takes inputHash, not raw input.
                // We'll just call `MapResponse` to create a new one to demonstrate determinism.
                const newSig = await svc.MapResponse(inputAction(), inputEvent(), inputPolicy());

                // Now verify it via Replay
                const verify = await svc.Replay(newSig.input_hash, inputPolicy(), inputAction());

                setReplayData({
                    signature: newSig,
                    verified: verify
                });

                loadSigs();
            }
        } catch (err) {
            alert("Error running simulation: " + err);
        }
    };

    return (
        <div class="p-8 h-full flex flex-col bg-gray-950 text-gray-100 font-mono">
            <h1 class="text-3xl font-black tracking-widest text-slate-100 mb-2">DETERMINISTIC RESPONSE ENGINE</h1>
            <p class="text-slate-500 mb-8 font-bold tracking-widest text-sm border-b border-gray-800 pb-6">
                PROVE RESPONSE EXECUTIONS USING MATHEMATICAL CERTAINTY
            </p>

            <div class="grid grid-cols-1 lg:grid-cols-2 gap-8 h-full">

                {/* Simulation Panel */}
                <div class="border border-slate-800 bg-slate-900/50 p-6 flex flex-col gap-4 overflow-y-auto">
                    <h2 class="text-sm tracking-widest font-bold text-slate-400 mb-2">EXECUTION SIMULATOR</h2>

                    <div>
                        <label class="block text-xs uppercase text-slate-500 mb-1">Response Action</label>
                        <select
                            value={inputAction()}
                            onChange={(e) => setInputAction(e.currentTarget.value)}
                            class="w-full bg-slate-950 border border-slate-700 text-sm p-2 text-emerald-400 focus:border-emerald-500 outline-none"
                        >
                            <option value="IsolateHost">IsolateHost</option>
                            <option value="BlockIP">BlockIP</option>
                            <option value="KillProcess">KillProcess</option>
                            <option value="DisableUser">DisableUser</option>
                        </select>
                    </div>

                    <div>
                        <label class="block text-xs uppercase text-slate-500 mb-1">State Policy Hash (Hex)</label>
                        <input
                            type="text"
                            value={inputPolicy()}
                            onInput={(e) => setInputPolicy(e.currentTarget.value)}
                            class="w-full bg-slate-950 border border-slate-700 text-sm p-2 text-slate-300 focus:border-emerald-500 outline-none"
                            placeholder="e.g. 8e3a2b4cd9"
                        />
                    </div>

                    <div>
                        <label class="block text-xs uppercase text-slate-500 mb-1">Input Trigger Event (JSON)</label>
                        <textarea
                            value={inputEvent()}
                            onInput={(e) => setInputEvent(e.currentTarget.value)}
                            rows={4}
                            class="w-full bg-slate-950 border border-slate-700 text-sm p-3 text-slate-300 font-mono focus:border-emerald-500 outline-none"
                        ></textarea>
                    </div>

                    <button
                        onClick={runSimulation}
                        class="mt-4 w-full py-3 bg-slate-800 border border-emerald-500 text-emerald-400 hover:bg-emerald-900 transition-colors uppercase tracking-widest text-sm font-bold"
                    >
                        Execute & Generate Signature
                    </button>

                    <Show when={replayData()}>
                        <div class="mt-6 border-t border-slate-800 pt-6">
                            <h3 class="text-xs tracking-widest text-slate-500 mb-4">COMPUTED EXECUTION SIGNATURE</h3>
                            <div class="bg-gray-950 border border-slate-700 p-4 relative">
                                <div class="absolute top-2 right-2 flex items-center gap-1">
                                    <Show when={replayData().verified.matched_past}>
                                        <span class="text-[10px] bg-emerald-900/50 text-emerald-400 border border-emerald-500/50 px-2 py-0.5 uppercase tracking-wider">Historical Match</span>
                                    </Show>
                                </div>
                                <div class="grid grid-cols-[100px_1fr] gap-y-2 text-xs">
                                    <span class="text-slate-500">ID:</span>
                                    <span class="text-emerald-400">{replayData().signature.id}</span>

                                    <span class="text-slate-500">Input Hash:</span>
                                    <span class="text-slate-400 break-all">{replayData().signature.input_hash}</span>

                                    <span class="text-slate-500">Policy Hash:</span>
                                    <span class="text-slate-400">{replayData().signature.policy_hash}</span>

                                    <span class="text-slate-500">Final Proof:</span>
                                    <span class="text-amber-400 break-all">{replayData().signature.final_hash}</span>
                                </div>
                            </div>
                        </div>
                    </Show>
                </div>

                {/* History Panel */}
                <div class="border border-slate-800 bg-slate-900/30 p-6 flex flex-col overflow-y-auto">
                    <h2 class="text-sm tracking-widest text-slate-500 mb-4 flex justify-between">
                        <span>IMMUTABLE RESPONSE HISTORY</span>
                        <span>{signatures().length} EXECUTIONS</span>
                    </h2>

                    <div class="flex flex-col gap-3">
                        <For each={signatures().sort((a, b) => new Date(b.timestamp).getTime() - new Date(a.timestamp).getTime())}>
                            {(sig) => (
                                <div class="p-3 border border-slate-800 bg-slate-950 hover:border-slate-600 transition-colors text-xs flex flex-col gap-2">
                                    <div class="flex justify-between border-b border-slate-800 pb-2">
                                        <span class="text-emerald-400 font-bold">{sig.action_taken}</span>
                                        <span class="text-slate-500">{new Date(sig.timestamp).toLocaleTimeString()}</span>
                                    </div>
                                    <div class="grid grid-cols-[80px_1fr] gap-1">
                                        <span class="text-slate-600">INPUT:</span>
                                        <span class="text-slate-400 font-mono truncate">{sig.input_hash.substring(0, 24)}...</span>
                                        <span class="text-slate-600">PROOF:</span>
                                        <span class="text-amber-500 font-mono truncate">{sig.final_hash.substring(0, 24)}...</span>
                                    </div>
                                </div>
                            )}
                        </For>

                        <Show when={signatures().length === 0}>
                            <div class="text-center p-8 text-slate-600 text-sm italic">
                                No deterministic responses recorded yet. Run the simulator to generate a proof.
                            </div>
                        </Show>
                    </div>
                </div>

            </div>
        </div>
    );
};
