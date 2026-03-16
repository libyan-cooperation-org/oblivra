import { Component, createSignal, onMount, For, Show } from 'solid-js';

export const CompliancePanel: Component = () => {
    const [reports, setReports] = createSignal<any[]>([]);
    const [loading, setLoading] = createSignal(true);

    const reload = async () => {
        try {
            const { ListReports } = await import('../../../wailsjs/go/services/ComplianceService');
            setReports(await ListReports() || []);
        } catch (e) { console.error('Compliance load:', e); }
        setLoading(false);
    };

    onMount(reload);

    const generate = async () => {
        setLoading(true);
        try {
            const { GenerateReport } = await import('../../../wailsjs/go/services/ComplianceService');
            const now = Date.now();
            await GenerateReport('SOC2', now - 30 * 24 * 3600 * 1000, now);
            await reload();
        } catch (e) { console.error('Generate report:', e); }
        setLoading(false);
    };

    return (
        <div style="display: flex; flex-direction: column; height: 100%;">
            <div class="drawer-header">
                <span class="drawer-title">Compliance Reports</span>
                <button class="action-btn primary" style="padding: 2px 8px; font-size: 10px;" onClick={generate}>+ Generate</button>
            </div>
            <div style="flex: 1; overflow-y: auto; padding: 8px;">
                <Show when={loading()}><div class="placeholder">Loading...</div></Show>
                <Show when={!loading()}>
                    <For each={reports()} fallback={<div class="placeholder">No compliance reports. Generate one to audit session activity.</div>}>
                        {(r) => (
                            <div style="background: var(--bg-tertiary); border: 1px solid var(--border-primary); border-radius: var(--radius-sm); padding: 10px; margin-bottom: 6px;">
                                <div style="display: flex; justify-content: space-between; align-items: center;">
                                    <div style="font-size: 12px; font-weight: 500; color: var(--text-primary);">📋 {r.framework || 'Report'}</div>
                                    <span class="tag-pill" style={`background: ${r.compliant ? 'var(--success-bg)' : 'var(--error-bg)'}; color: ${r.compliant ? 'var(--success)' : 'var(--error)'}; border: none;`}>
                                        {r.compliant ? '✓ Compliant' : '✗ Issues'}
                                    </span>
                                </div>
                                <div style="font-size: 10px; color: var(--text-muted); margin-top: 4px;">
                                    {r.total_sessions || 0} sessions • {r.findings?.length || 0} findings • {r.generated_at ? new Date(r.generated_at).toLocaleDateString() : ''}
                                </div>
                            </div>
                        )}
                    </For>
                </Show>
            </div>
        </div>
    );
};
