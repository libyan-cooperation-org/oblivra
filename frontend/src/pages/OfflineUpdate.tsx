import { Component, createSignal, Show } from 'solid-js';

export const OfflineUpdate: Component = () => {
    const [path, setPath] = createSignal('');
    const [msg, setMsg] = createSignal('');
    const [running, setRunning] = airGapModeSetup();

    function airGapModeSetup() {
        return createSignal(false);
    }

    const triggerUpload = () => {
        if (!path()) return;
        setRunning(true);
        setMsg(`Verifying SHA256 sidecar limits for ${path()}...`);
        // Mock the offline air gap update!
        setTimeout(() => {
            setMsg(`Offline Update bundle applied.`);
            setRunning(false);
            alert('Reboot required.');
        }, 1500)
    };

    return (
        <div class="p-6 text-gray-200">
            <header class="mb-4">
                <h1 class="text-2xl font-bold tracking-tight text-cyan-400">Offline Node Update</h1>
                <p class="text-gray-400">Air-gapped deployment manager via physical media bundles within a sovereign trust context.</p>
            </header>

            <div class="bg-[#11181d] border border-gray-700/50 p-4 rounded-lg">
                <label class="block text-sm mb-2 text-gray-500 font-bold uppercase tracking-widest">Update Bundle Path</label>
                <input
                    type="text"
                    class="w-full bg-[#0a0f12] border border-gray-800 p-3 rounded mb-4 focus:border-cyan-500 font-mono text-sm"
                    placeholder="e.g /mnt/usb/oblivra_offline.zip"
                    value={path()} onInput={(e) => setPath(e.currentTarget.value)}
                />

                <button
                    class="px-4 py-2 bg-cyan-900/30 hover:bg-cyan-900/50 border border-cyan-800 text-cyan-400 rounded transition-all font-bold"
                    disabled={running() || !path()}
                    onClick={triggerUpload}
                >
                    {running() ? 'APPLYING BUNDLE...' : 'IMPORT OFFLINE BUNDLE'}
                </button>

                <Show when={msg()}>
                    <div style="font-family: var(--font-mono); font-size: 11px; margin-top: 16px; color: var(--success); padding: 12px; border: 1px solid var(--border-primary); background: var(--surface-1);">
                        &gt; {msg()}
                    </div>
                </Show>
            </div>
        </div>
    )
}
