import { Component, createSignal, onMount, For, Show } from 'solid-js';

interface LedgerBlock {
    index: number;
    timestamp: string;
    data: string;          // base64 encoded by wails originally, but actually the fields say []byte which goes over JSON as base64 string
    data_type: string;
    previous_hash: string;
    hash: string;
}

export const EvidenceLedger: Component = () => {
    const [blocks, setBlocks] = createSignal<LedgerBlock[]>([]);
    const [status, setStatus] = createSignal<'UNKNOWN' | 'VERIFYING' | 'VALID' | 'INVALID'>('UNKNOWN');
    const [errorMsg, setErrorMsg] = createSignal<string>('');
    const [selectedBlock, setSelectedBlock] = createSignal<LedgerBlock | null>(null);

    const loadChain = async () => {
        try {
            const svc = (window as any).go?.app?.LedgerService;
            if (!svc) return;
            const chainData = await svc.GetChain();
            setBlocks(chainData || []);
        } catch (err) {
            console.error("Failed to load ledger:", err);
            setErrorMsg(String(err));
        }
    };

    onMount(() => {
        loadChain();
    });

    const verifyLedger = async () => {
        setStatus('VERIFYING');
        try {
            const svc = (window as any).go?.app?.LedgerService;
            if (!svc) return;
            const res = await svc.VerifyChain();
            if (res === 'VALID') {
                setStatus('VALID');
                setErrorMsg('');
            } else {
                setStatus('INVALID');
                setErrorMsg(res);
            }
        } catch (err) {
            setStatus('INVALID');
            setErrorMsg(String(err));
        }
    };

    const exportLedger = async () => {
        try {
            const svc = (window as any).go?.app?.LedgerService;
            if (!svc) return;
            const jsonStr = await svc.ExportChain();
            const blob = new Blob([jsonStr], { type: 'application/json' });
            const url = window.URL.createObjectURL(blob);
            const a = document.createElement('a');
            a.href = url;
            a.download = `sovereign_evidence_ledger_${new Date().getTime()}.json`;
            a.click();
            window.URL.revokeObjectURL(url);
        } catch (err) {
            alert("Failed to export ledger: " + err);
        }
    };

    const decodeData = (b64: string) => {
        try {
            return atob(b64);
        } catch (e) {
            return b64;
        }
    };

    return (
        <div class="p-6 h-full flex flex-col bg-gray-900 text-gray-100 font-mono">
            <div class="flex justify-between items-center mb-6 border-b border-gray-700 pb-4">
                <div>
                    <h1 class="text-2xl font-bold text-emerald-400">Cryptographic Evidence Ledger</h1>
                    <p class="text-sm text-gray-400">Immutable, verifiable blockchain of platform decisions and events</p>
                </div>
                <div class="flex gap-4">
                    <button
                        onClick={verifyLedger}
                        class="px-4 py-2 bg-slate-800 border border-emerald-500 text-emerald-400 hover:bg-emerald-900 transition-colors"
                        disabled={status() === 'VERIFYING'}
                    >
                        {status() === 'VERIFYING' ? 'Verifying hashes...' : 'Verify Cryptographic Integrity'}
                    </button>
                    <button
                        onClick={exportLedger}
                        class="px-4 py-2 bg-slate-800 border border-slate-600 hover:bg-slate-700 transition-colors"
                    >
                        Export JSON
                    </button>
                </div>
            </div>

            <Show when={status() !== 'UNKNOWN'}>
                <div class={`p-4 mb-6 border ${status() === 'VALID' ? 'border-emerald-500 bg-emerald-900/30' : 'border-red-500 bg-red-900/30'}`}>
                    <h3 class={`text-lg font-bold ${status() === 'VALID' ? 'text-emerald-400' : 'text-red-400'}`}>
                        {status() === 'VALID' ? '✓ Ledger Cryptographically Verified' : '⚠ Integrity Violation Detected'}
                    </h3>
                    <Show when={errorMsg()}>
                        <p class="text-red-300 mt-2">{errorMsg()}</p>
                    </Show>
                </div>
            </Show>

            <div class="flex gap-6 h-full overflow-hidden">
                <div class="w-1/3 border border-gray-700 overflow-y-auto bg-gray-950 p-4">
                    <h2 class="text-sm text-gray-500 mb-4 sticky top-0 bg-gray-950">LEDGER BLOCKS ({blocks().length})</h2>
                    <div class="flex flex-col gap-4 relative">
                        {/* Connecting line */}
                        <div class="absolute left-4 top-0 bottom-0 w-px bg-emerald-500/30"></div>

                        <For each={blocks()}>
                            {(block) => (
                                <div
                                    class={`relative ml-10 p-3 border cursor-pointer transition-colors ${selectedBlock()?.index === block.index ? 'border-emerald-500 bg-slate-800' : 'border-gray-700 hover:border-gray-500 bg-gray-900'}`}
                                    onClick={() => setSelectedBlock(block)}
                                >
                                    {/* Link node */}
                                    <div class={`absolute -left-[30px] top-4 w-3 h-3 rounded-full border border-gray-900 ${block.index === 0 ? 'bg-amber-500' : 'bg-emerald-400'}`}></div>

                                    <div class="flex justify-between items-center mb-1">
                                        <span class="font-bold text-xs"># {block.index}</span>
                                        <span class="text-[10px] text-gray-500 uppercase">{block.data_type}</span>
                                    </div>
                                    <div class="text-[11px] text-emerald-400 mb-2 truncate" title={block.hash}>
                                        {block.hash.substring(0, 16)}...
                                    </div>
                                    <div class="text-[10px] text-gray-400">
                                        {new Date(block.timestamp).toISOString()}
                                    </div>
                                </div>
                            )}
                        </For>
                    </div>
                </div>

                <div class="w-2/3 border border-gray-700 bg-gray-950 p-6 overflow-y-auto">
                    <Show
                        when={selectedBlock()}
                        fallback={<div class="flex h-full items-center justify-center text-gray-600">Select a block to inspect its payload</div>}
                    >
                        {(block) => (
                            <div>
                                <h2 class="text-xl font-bold text-gray-200 mb-6 border-b border-gray-700 pb-2">Block Details</h2>

                                <div class="grid grid-cols-[120px_1fr] gap-4 mb-8">
                                    <div class="text-gray-500 text-sm">Index</div>
                                    <div class="text-gray-200">{block().index}</div>

                                    <div class="text-gray-500 text-sm">Timestamp</div>
                                    <div class="text-gray-200">{new Date(block().timestamp).toLocaleString()}</div>

                                    <div class="text-gray-500 text-sm">Data Type</div>
                                    <div class="text-emerald-400">{block().data_type}</div>

                                    <div class="text-gray-500 text-sm">Hash</div>
                                    <div class="text-emerald-400 text-xs break-all">{block().hash}</div>

                                    <div class="text-gray-500 text-sm">Previous Hash</div>
                                    <div class="text-gray-400 text-xs break-all">{block().previous_hash || '0000000000000000000000000000000000000000000000000000000000000000 (Genesis)'}</div>
                                </div>

                                <div class="mb-2 text-gray-500 text-sm">Evidence Payload (Decoded)</div>
                                <pre class="bg-gray-900 border border-gray-700 p-4 text-sm text-gray-300 overflow-x-auto whitespace-pre-wrap">
                                    {decodeData(block().data)}
                                </pre>
                            </div>
                        )}
                    </Show>
                </div>
            </div>
        </div>
    );
};
