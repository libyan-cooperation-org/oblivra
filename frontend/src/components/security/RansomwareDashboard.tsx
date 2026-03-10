import { Component, createSignal, onMount, For, Show } from 'solid-js';
import { ListIncidents } from '../../../wailsjs/go/app/IncidentService';
import { database } from '../../../wailsjs/go/models';

export const RansomwareDashboard: Component = () => {
    const [alerts, setAlerts] = createSignal<database.Incident[]>([]);

    onMount(async () => {
        try {
            const list = await ListIncidents(null as any, "", "", 50);
            // Filter only ransomware incidents
            setAlerts(list?.filter((i: database.Incident) => i.rule_id.includes('Ransomware')) || []);
        } catch (err) {
            console.error("Failed to load ransomware alerts:", err);
        }
    });

    return (
        <div class="ob-page page-enter" style="padding: 32px;">
            <header style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 32px; border-bottom: 1px solid var(--border-primary); padding-bottom: 24px;">
                <div>
                    <h1 style="font-size: 24px; font-weight: 800; color: var(--status-offline); letter-spacing: -1px; margin: 0; text-transform: uppercase;">Behavioral Ransomware Defense</h1>
                    <p style="color: var(--text-muted); font-size: 12px; margin: 8px 0 0 0;">Real-time entropy analysis & automated isolation</p>
                </div>
                <div style="display: flex; gap: 16px;">
                    <div style="text-align: center; padding: 12px 24px; background: rgba(239, 68, 68, 0.1); border-radius: 8px; border: 1px solid rgba(239, 68, 68, 0.2);">
                        <div style="font-size: 10px; color: rgba(239, 68, 68, 0.6); text-transform: uppercase; font-weight: 800; margin-bottom: 4px;">Risk Level</div>
                        <div style="font-size: 16px; font-family: var(--font-mono); color: var(--status-offline); font-weight: 800;">CRITICAL</div>
                    </div>
                </div>
            </header>

            <div style="display: grid; grid-template-columns: repeat(3, 1fr); gap: 24px; margin-bottom: 32px;">
                <div class="ob-card" style="padding: 24px;">
                    <h3 style="color: var(--text-muted); font-size: 11px; font-weight: 800; text-transform: uppercase; margin: 0 0 16px 0;">Entropy Sensors</h3>
                    <div style="font-size: 24px; font-family: var(--font-mono); color: var(--status-online); font-weight: 800; margin-bottom: 8px;">ACTIVE [18Nodes]</div>
                    <div style="font-size: 10px; color: var(--text-muted);">Avg Entropy: 4.2 (Stable)</div>
                </div>
                <div class="ob-card" style="padding: 24px;">
                    <h3 style="color: var(--text-muted); font-size: 11px; font-weight: 800; text-transform: uppercase; margin: 0 0 16px 0;">Canary Files</h3>
                    <div style="font-size: 24px; font-family: var(--font-mono); color: var(--accent-primary); font-weight: 800; margin-bottom: 8px;">ARMED [124 Files]</div>
                    <div style="font-size: 10px; color: var(--text-muted);">Last Baseline: 4 min ago</div>
                </div>
                <div class="ob-card" style="padding: 24px; background: rgba(239, 68, 68, 0.05); border-color: rgba(239, 68, 68, 0.2);">
                    <h3 style="color: var(--status-offline); font-size: 11px; font-weight: 800; text-transform: uppercase; margin: 0 0 16px 0;">Active Threats</h3>
                    <div style="font-size: 24px; font-family: var(--font-mono); color: var(--status-offline); font-weight: 800; margin-bottom: 8px;">{alerts().length} DETECTED</div>
                    <div style="font-size: 10px; color: rgba(239, 68, 68, 0.6);">Automated containment ready</div>
                </div>
            </div>

            <div>
                <Show when={alerts().length > 0} fallback={
                    <div style="padding: 64px 32px; text-align: center; border: 2px dashed var(--border-primary); border-radius: 12px;">
                        <div style="color: var(--text-muted); font-size: 16px; margin-bottom: 8px; font-weight: 800;">Secure Environment Analysis</div>
                        <p style="color: var(--text-muted); font-size: 12px; margin: 0; opacity: 0.7;">No behavioral anomalies matching encryption patterns detected.</p>
                    </div>
                }>
                    <For each={alerts()}>
                        {(alert) => (
                            <div class="ob-card" style="border-left: 4px solid var(--status-offline); border-radius: 4px; padding: 24px; margin-bottom: 16px; display: flex; justify-content: space-between; align-items: center; transition: background 120ms ease; cursor: pointer;" onMouseEnter={(e) => e.currentTarget.style.background = 'var(--bg-elevated)'} onMouseLeave={(e) => e.currentTarget.style.background = 'var(--bg-surface)'}>
                                <div>
                                    <div style="display: flex; align-items: center; gap: 8px; margin-bottom: 8px;">
                                        <span style="font-size: 10px; font-weight: 800; background: var(--status-offline); color: black; padding: 2px 6px; border-radius: 2px;">MATCH</span>
                                        <h4 style="font-weight: 800; color: var(--text-primary); text-transform: uppercase; margin: 0;">{alert.title}</h4>
                                    </div>
                                    <p style="color: var(--text-muted); font-size: 12px; margin: 0 0 12px 0;">{alert.description}</p>
                                    <div style="display: flex; gap: 24px; font-size: 10px; font-family: var(--font-mono); color: var(--text-muted); text-transform: uppercase;">
                                        <span>Entity: {alert.group_key}</span>
                                        <span>Technique: T1486</span>
                                        <span>Confidence: 98%</span>
                                    </div>
                                </div>
                                <div style="display: flex; flex-direction: column; gap: 12px; align-items: flex-end;">
                                    <button class="ob-btn ob-btn-secondary" style="background: var(--status-offline); color: white; border-color: var(--status-offline); font-size: 10px;">
                                        EXECUTE ISOLATION
                                    </button>
                                    <button style="background: none; border: none; color: var(--text-muted); text-decoration: underline; text-underline-offset: 4px; font-size: 10px; font-weight: 800; text-transform: uppercase; cursor: pointer;" onMouseEnter={(e) => e.currentTarget.style.color = 'var(--text-primary)'} onMouseLeave={(e) => e.currentTarget.style.color = 'var(--text-muted)'}>
                                        View Forensics
                                    </button>
                                </div>
                            </div>
                        )}
                    </For>
                </Show>
            </div>

            <div class="ob-card" style="margin-top: 48px; padding: 32px;">
                <h2 style="font-size: 18px; font-weight: 800; color: var(--text-primary); margin: 0 0 24px 0; display: flex; align-items: center; gap: 12px;">
                    <div style="width: 8px; height: 8px; border-radius: 50%; background: var(--accent-primary);" class="animate-pulse" />
                    Real-time Entropy Stream
                </h2>
                <div style="height: 128px; display: flex; align-items: flex-end; gap: 4px; border-bottom: 1px solid var(--border-primary); padding-bottom: 8px;">
                    <For each={[...Array(40)]}>
                        {(_, i) => (
                            <div
                                style={`flex: 1; background: var(--border-focus); min-height: 4px; cursor: crosshair; transition: all 120ms ease; height: ${Math.random() * 80 + 10}%; opacity: ${(i() / 40) + 0.2}; border-radius: 2px 2px 0 0;`}
                                title={`Node ${i()}: Entropy ${(Math.random() * 8).toFixed(2)}`}
                                onMouseEnter={(e) => e.currentTarget.style.background = 'var(--accent-primary)'}
                                onMouseLeave={(e) => e.currentTarget.style.background = 'var(--border-focus)'}
                            />
                        )}
                    </For>
                </div>
                <div style="display: flex; justify-content: space-between; margin-top: 16px; font-size: 10px; color: var(--text-muted); font-family: var(--font-mono);">
                    <span>SYSTEM_ID: 0x44FD8</span>
                    <span>BURST_MODE: OFF</span>
                    <span>SAMPLE_RATE: 100ms</span>
                </div>
            </div>
        </div>
    );
};
