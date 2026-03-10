import { Component, createSignal, Show } from 'solid-js';

export const SyncPanel: Component = () => {
    const [syncing, setSyncing] = createSignal(false);
    const [result, setResult] = createSignal<string | null>(null);

    const handleSync = async () => {
        setSyncing(true);
        setResult(null);
        try {
            const { Sync } = await import('../../../wailsjs/go/app/SyncService');
            await Sync();
            setResult('Sync completed successfully ✓');
        } catch (e: unknown) { setResult(`Sync failed: ${(e as Error)?.message || e}`); }
        setSyncing(false);
    };

    return (
        <div style="padding: 12px; display: flex; flex-direction: column; gap: 12px;">
            <div class="section-label">Sync Settings</div>
            <div class="placeholder" style="padding: 16px;">
                <div style="font-size: 32px; margin-bottom: 8px;">☁️</div>
                <div style="font-size: 13px; color: var(--text-primary); margin-bottom: 4px;">Encrypted Sync</div>
                <div style="font-size: 11px; color: var(--text-muted);">Sync your hosts, credentials, and settings across devices with end-to-end encryption.</div>
            </div>
            <button class="action-btn primary" style="align-self: center; padding: 8px 24px;" onClick={handleSync} disabled={syncing()}>
                {syncing() ? '⟳ Syncing...' : '☁️ Sync Now'}
            </button>
            <Show when={result()}>
                <div style={`font-size: 11px; padding: 6px 8px; border-radius: var(--radius-xs); text-align: center; ${result()!.includes('✓') ? 'color: var(--success); background: var(--success-bg);' : 'color: var(--error); background: var(--error-bg);'}`}>{result()}</div>
            </Show>
        </div>
    );
};
