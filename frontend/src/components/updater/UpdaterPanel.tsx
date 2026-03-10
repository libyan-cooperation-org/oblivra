import { Component, createSignal, Show } from 'solid-js';

export const UpdaterPanel: Component = () => {
    const [checking, setChecking] = createSignal(false);
    const [updateInfo, setUpdateInfo] = createSignal<any>(null);
    const [applying, setApplying] = createSignal(false);
    const [result, setResult] = createSignal<string | null>(null);

    const checkUpdate = async () => {
        setChecking(true);
        setResult(null);
        try {
            const { CheckForUpdate } = await import('../../../wailsjs/go/app/UpdaterService');
            const info = await CheckForUpdate();
            setUpdateInfo(info);
            if (!info || !info.available) setResult('You are on the latest version ✓');
        } catch (e) { setResult(`Check failed: ${(e as Error)?.message || e}`); }
        setChecking(false);
    };

    const applyUpdate = async () => {
        setApplying(true);
        try {
            const { ApplyUpdate } = await import('../../../wailsjs/go/app/UpdaterService');
            await ApplyUpdate();
            setResult('Update applied! Restart to complete.');
        } catch (e) { setResult(`Update failed: ${(e as Error)?.message || e}`); }
        setApplying(false);
    };

    return (
        <div style="padding: 12px; display: flex; flex-direction: column; gap: 12px; align-items: center; justify-content: center; height: 100%;">
            <div style="font-size: 36px;">🔄</div>
            <div class="section-label" style="text-align: center;">App Updates</div>
            <div style="font-size: 12px; color: var(--text-muted); text-align: center;">OblivraShell v0.1.0</div>
            <button class="action-btn" onClick={checkUpdate} disabled={checking()} style="padding: 8px 20px;">
                {checking() ? '⟳ Checking...' : '🔍 Check for Updates'}
            </button>
            <Show when={updateInfo()?.available}>
                <div style="background: var(--info-bg); border: 1px solid rgba(77,166,255,0.3); border-radius: var(--radius-sm); padding: 10px; text-align: center; width: 100%; max-width: 300px;">
                    <div style="font-size: 12px; color: var(--info); font-weight: 500;">Update Available: v{updateInfo()?.version || 'new'}</div>
                    <div style="font-size: 10px; color: var(--text-muted); margin: 4px 0;">{updateInfo()?.notes || ''}</div>
                    <button class="action-btn primary" style="padding: 5px 16px; font-size: 11px; margin-top: 6px;" onClick={applyUpdate} disabled={applying()}>
                        {applying() ? 'Applying...' : '⬇ Install Update'}
                    </button>
                </div>
            </Show>
            <Show when={result()}>
                <div style={`font-size: 11px; ${result()!.includes('✓') || result()!.includes('Restart') ? 'color: var(--success);' : 'color: var(--error);'}`}>{result()}</div>
            </Show>
        </div>
    );
};
