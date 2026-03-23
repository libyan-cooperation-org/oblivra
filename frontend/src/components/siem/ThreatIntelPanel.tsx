import { Component, createSignal, createEffect, For, Show } from 'solid-js';
import { useApp } from '@core/store';
import { IS_BROWSER } from '@core/context';
import { EmptyState } from '../ui/EmptyState';
import { showModal } from '../ui/ModalSystem';

export const ThreatIntelPanel: Component = () => {
    const [state] = useApp();
    const [stats, setStats] = createSignal<Record<string, number>>({});
    const [isLoading, setIsLoading] = createSignal(true);

    const fetchStats = async () => {
        if (IS_BROWSER) { setIsLoading(false); return; }
        try {
            const { GetThreatIntelStats } = await import('../../../wailsjs/go/services/SIEMService');
            const res = await GetThreatIntelStats();
            setStats(res || {});
        } catch (err) {
            console.error('Failed to load TI stats:', err);
        } finally {
            setIsLoading(false);
        }
    };

    createEffect(() => {
        if (state.activeNavTab === 'siem') {
            fetchStats();
            const interval = setInterval(fetchStats, 10000);
            return () => clearInterval(interval);
        }
    });

    const handleUploadClick = () => {
        showModal({
            title: 'Import Offline IOCs',
            message: 'In air-gapped deployments, you can manually upload STIX 2.1 JSON or CSV indicator files to enrich raw logs.',
            confirmText: 'Import .stix / .csv',
            onConfirm: async () => {
                try {
                    const sampleIOCs = [
                        { type: 'ipv4-addr', value: '185.191.171.13', source: 'Manual Import', severity: 'critical', description: 'Known C2 Infrastructure' },
                        { type: 'domain-name', value: 'evil-updates.com', source: 'Manual Import', severity: 'high', description: 'Malware dropper domain' }
                    ];
                    const { LoadOfflineIOCs } = await import('../../../wailsjs/go/services/SIEMService');
                    const loaded = await LoadOfflineIOCs(sampleIOCs as any);
                    showModal({ title: 'Success', message: `Loaded ${loaded} indicators into memory.`, onConfirm: async () => fetchStats(), onCancel: () => { }, cancelText: '' });
                } catch (e) {
                    console.error("Import failed:", e);
                }
            },
            onCancel: () => { },
        });
    };

    return (
        <div class="ob-card" style="padding: 24px; background: transparent; border: none; box-shadow: none;">
            <div style="display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 24px; border-bottom: 1px solid var(--border-primary); padding-bottom: 16px;">
                <div>
                    <h2 style="font-size: 18px; font-weight: 800; color: var(--text-primary); margin: 0; font-family: var(--font-ui); text-transform: uppercase; letter-spacing: 1px;">Threat Intelligence</h2>
                    <p style="font-size: 11px; color: var(--text-muted); margin: 4px 0 0 0; text-transform: uppercase; letter-spacing: 0.5px;">
                        TAXII FEED MANAGEMENT AND OFFLINE INDICATOR ENRICHMENT
                    </p>
                </div>
                <div style="display: flex; gap: 8px;">
                    <button class="ob-btn ob-btn-secondary ob-btn-sm" onClick={handleUploadClick}>
                        ↓ IMPORT_IOC
                    </button>
                    <button class="ob-btn ob-btn-primary ob-btn-sm">
                        + ADD_TAXII_SERVER
                    </button>
                </div>
            </div>

            <Show when={!isLoading()} fallback={
                <div class="dash-kpi-grid" style="border: 1px solid var(--border-primary);">
                    <For each={[1, 2, 3]}>{() =>
                        <div class="dash-kpi" style="padding: 16px;">
                            <div class="ob-skeleton" style="width: 80px; height: 10px; margin-bottom: 8px;" />
                            <div class="ob-skeleton" style="width: 60px; height: 24px;" />
                        </div>
                    }</For>
                </div>
            }>
                <div class="dash-kpi-grid" style="border: 1px solid var(--border-primary);">
                    <div class="dash-kpi" style="padding: 16px;">
                        <div class="dash-kpi-label">Total Active Indicators</div>
                        <div class="dash-kpi-value" style="color: var(--accent-primary);">
                            {Object.values(stats()).reduce((a, b) => a + b, 0).toLocaleString()}
                        </div>
                    </div>

                    <For each={Object.entries(stats())}>
                        {([type, count]) => (
                            <div class="dash-kpi" style="padding: 16px;">
                                <div class="dash-kpi-label">{type.toUpperCase()}</div>
                                <div class="dash-kpi-value">
                                    {count.toLocaleString()}
                                </div>
                            </div>
                        )}
                    </For>
                </div>

                <Show when={Object.keys(stats()).length === 0}>
                    <div style="margin-top: 24px; padding: 40px; border: 1px dashed var(--border-secondary); border-radius: 4px; background: rgba(0,0,0,0.1);">
                        <EmptyState
                            icon="INTEL_SILENCE"
                            title="NO INTELLIGENCE FEEDS DETECTED"
                            description="STIX 2.1 or TAXII 2.1 integration required for automated forensic enrichment. Import offline bundle to proceed."
                            action="INITIALIZE_IMPORT"
                            onAction={handleUploadClick}
                        />
                    </div>
                </Show>
            </Show>
        </div>
    );
};
