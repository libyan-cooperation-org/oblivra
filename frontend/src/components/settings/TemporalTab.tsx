import { Component, createSignal } from 'solid-js';

export const TemporalTab: Component = () => {
    const [drift, setDrift] = createSignal(12); // ms

    return (
        <div class="brutalist-panel">
            <h2 class="brutalist-header">Temporal Integrity</h2>

            <div class="status-block" style="flex-direction: column; align-items: flex-start; gap: 16px;">
                <div style="width: 100%; display: flex; justify-content: space-between; align-items: center;">
                    <div>
                        <div style="font-weight: 800; font-size: 16px; margin-bottom: 4px;">Secure NTP Synchronization</div>
                        <div style="font-size: 12px; color: var(--text-muted); font-weight: 600;">Upstream: time.google.com (Authenticated)</div>
                    </div>
                    <div class="status-badge trusted">Bound</div>
                </div>

                <div style="width: 100%; background: var(--bg-surface); padding: 12px; border: 2px dashed var(--text-muted);">
                    <div style="display: flex; justify-content: space-between; margin-bottom: 8px;">
                        <span style="font-size: 12px; font-weight: 800; text-transform: uppercase;">Estimated Clock Drift</span>
                        <span style="font-size: 12px; font-weight: 800; color: var(--warning);">{drift()}ms</span>
                    </div>
                    <div style="width: 100%; height: 8px; background: var(--bg-primary); border: 1px solid var(--text-primary);">
                        <div style={`width: ${Math.min(100, (drift() / 50) * 100)}%; height: 100%; background: var(--warning);`} />
                    </div>
                </div>
            </div>

            <div style="margin-top: 32px;">
                <button class="brutalist-btn" onClick={() => { setDrift(Math.floor(Math.random() * 5)); alert('NTP Resync Triggered'); }}>
                    Force NTP Resync
                </button>
            </div>
        </div>
    );
};
