import { Component, createSignal, onMount, For, Show } from 'solid-js';
import { GetActiveTier, GetCapabilitiesMatrix, SetTier } from '../../../wailsjs/go/app/PolicyService';
import '../../styles/siem.css';

export const FeatureGovernance: Component = () => {
    const [tier, setTier] = createSignal<string>('free');
    const [features, setFeatures] = createSignal<Record<string, boolean>>({});
    const [, setLoading] = createSignal(true);
    const [error, setError] = createSignal<string | null>(null);

    const loadGovernanceData = async () => {
        setLoading(true);
        setError(null);
        try {
            const currentTier = await GetActiveTier();
            setTier(currentTier);

            const matrix = await GetCapabilitiesMatrix();
            setFeatures(matrix);
        } catch (err) {
            console.error("Failed to load governance data:", err);
            setError(String(err));
        } finally {
            setLoading(false);
        }
    };

    onMount(() => {
        loadGovernanceData();
    });

    const handleTierChange = async (newTier: string) => {
        try {
            await SetTier(newTier);
            await loadGovernanceData();
        } catch (err) {
            setError(String(err));
        }
    };

    return (
        <div class="alert-dashboard">
            <header class="alert-header">
                <div>
                    <h2 class="host-name" style="margin:0;">Feature Governance Engine</h2>
                    <p class="host-id" style="margin:4px 0 0 0;">PLATFORM_TIER_CONTROL_MATRIX</p>
                </div>
                <button class="tactical-btn secondary" onClick={loadGovernanceData}>
                    <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="margin-right: 4px"><path d="M21.5 2v6h-6M21.34 15.57a10 10 0 1 1-.59-9.21l5.67-5.67" /></svg>
                    SYNC_STATE
                </button>
            </header>

            <div class="incident-filters">
                <button
                    class={`filter-btn ${tier() === 'free' ? 'active' : ''}`}
                    onClick={() => handleTierChange('free')}
                >
                    TIER_FREE
                </button>
                <button
                    class={`filter-btn ${tier() === 'pro' ? 'active' : ''}`}
                    onClick={() => handleTierChange('pro')}
                >
                    TIER_PRO
                </button>
                <button
                    class={`filter-btn ${tier() === 'enterprise' ? 'active' : ''}`}
                    onClick={() => handleTierChange('enterprise')}
                >
                    TIER_ENTERPRISE
                </button>
                <button
                    class={`filter-btn ${tier() === 'sovereign' ? 'active' : ''}`}
                    onClick={() => handleTierChange('sovereign')}
                >
                    TIER_SOVEREIGN
                </button>
            </div>

            <Show when={error()}>
                <div style="padding: 12px; border-bottom: 1px solid var(--status-offline); color: var(--status-offline); font-family: var(--font-mono); font-size: 11px; background: rgba(239,68,68,0.05);">
                    GOVERNANCE_INTERFACE_ERROR: {error()}
                </div>
            </Show>

            <div style="padding: 24px; flex: 1; overflow-y: auto;">
                <div style="display: flex; justify-content: space-between; align-items: flex-end; margin-bottom: 16px;">
                    <h3 style="margin: 0; color: var(--text-primary); font-family: var(--font-mono); font-size: 14px; text-transform: uppercase;">
                        Capability Access Matrix
                    </h3>
                    <div style="font-family: var(--font-mono); font-size: 10px; color: var(--text-muted);">
                        ACTIVE_MODE: <span style="color: var(--accent-primary); font-weight: bold;">{tier().toUpperCase()}</span>
                    </div>
                </div>

                <div style="display: grid; grid-template-columns: repeat(auto-fill, minmax(300px, 1fr)); gap: 16px;">
                    <For each={Object.entries(features())}>
                        {([featureId, isEnabled]) => (
                            <div class="incident-card" style={`border-left: 4px solid ${isEnabled ? 'var(--status-online)' : 'var(--text-muted)'};`}>
                                <div class="incident-card-header" style="align-items: center;">
                                    <div class="incident-title" style="margin: 0;">{featureId.replace('feature.', '').toUpperCase().replace('_', ' ')}</div>
                                    <div class="severity-badge" style={`color: ${isEnabled ? 'var(--status-online)' : 'var(--text-muted)'}; border-color: ${isEnabled ? 'var(--status-online)' : 'var(--text-muted)'};`}>
                                        {isEnabled ? 'GRANTED' : 'RESTRICTED'}
                                    </div>
                                </div>
                                <div class="incident-body">
                                    <div class="incident-description" style="color: var(--text-muted); font-family: var(--font-mono); font-size: 10px;">
                                        ID: {featureId}
                                    </div>
                                    <Show when={!isEnabled}>
                                        <div style="margin-top: 12px; font-size: 10px; color: var(--accent-secondary); font-weight: bold; font-family: var(--font-mono);">
                                            REQUIRES LICENSE UPGRADE
                                        </div>
                                    </Show>
                                </div>
                            </div>
                        )}
                    </For>
                </div>
            </div>
        </div>
    );
};
