import { Component, createSignal, onMount, onCleanup, For, Show } from 'solid-js';
import { GetFleetTelemetry } from '../../../wailsjs/go/app/TelemetryService';
import { monitoring } from '../../../wailsjs/go/models';
import '../../styles/heatmap.css';

const FleetHeatmap: Component = () => {
    const [fleetData, setFleetData] = createSignal<monitoring.HostTelemetry[]>([]);
    const [lastUpdate, setLastUpdate] = createSignal<Date>(new Date());
    const [loading, setLoading] = createSignal(true);

    const fetchTelemetry = async () => {
        try {
            const data = await GetFleetTelemetry();
            if (data) {
                setFleetData(data);
                setLastUpdate(new Date());
            }
        } catch (err) {
            console.error('Failed to fetch fleet telemetry:', err);
        } finally {
            setLoading(false);
        }
    };

    onMount(() => {
        fetchTelemetry();
        const interval = setInterval(fetchTelemetry, 10000);
        onCleanup(() => clearInterval(interval));
    });

    const getHeatColor = (usage: number) => {
        if (usage > 85) return 'var(--status-offline)';
        if (usage > 60) return 'var(--status-degraded)';
        return 'var(--status-online)';
    };

    const calculateOverallHealthColor = (t: monitoring.HostTelemetry) => {
        const avg = (t.cpu_usage + (t.mem_used_mb / t.mem_total_mb * 100)) / 2;
        return getHeatColor(avg);
    };

    return (
        <div class="ob-card" style="padding: 24px;">
            <div style="display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 24px; border-bottom: 1px solid var(--border-primary); padding-bottom: 16px;">
                <div style="display: flex; flex-direction: column; gap: 8px;">
                    <h2 style="font-size: 16px; font-weight: 800; color: var(--text-primary); margin: 0; font-family: var(--font-ui); text-transform: uppercase;">
                        Fleet Health Intelligence
                    </h2>
                    <div style="display: flex; gap: 16px; font-family: var(--font-mono); font-size: 10px; color: var(--text-muted); text-transform: uppercase;">
                        <div style="display: flex; align-items: center; gap: 6px;">
                            <div style="width: 8px; height: 8px; border-radius: 50%; background: var(--status-online);" />
                            OPTIMAL
                        </div>
                        <div style="display: flex; align-items: center; gap: 6px;">
                            <div style="width: 8px; height: 8px; border-radius: 50%; background: var(--status-degraded);" />
                            STRESSED
                        </div>
                        <div style="display: flex; align-items: center; gap: 6px;">
                            <div style="width: 8px; height: 8px; border-radius: 50%; background: var(--status-offline);" />
                            CRITICAL
                        </div>
                    </div>
                </div>
                <span style="font-size: 10px; color: var(--text-muted); font-family: var(--font-mono); text-transform: uppercase;">
                    LAST_POLL: {lastUpdate().toLocaleTimeString()}
                </span>
            </div>

            <div style="display: grid; grid-template-columns: repeat(auto-fill, minmax(280px, 1fr)); gap: 16px;">
                <Show when={loading() && fleetData().length === 0}>
                    <For each={[1, 2, 3, 4]}>{() =>
                        <div style="border: 1px solid var(--border-primary); padding: 16px; background: rgba(0,0,0,0.1); border-radius: 4px;">
                            <div class="ob-skeleton" style="width: 40%; height: 16px; margin-bottom: 16px;" />
                            <div class="ob-skeleton" style="width: 100%; height: 8px; margin-bottom: 12px;" />
                            <div class="ob-skeleton" style="width: 100%; height: 8px; margin-bottom: 12px;" />
                            <div class="ob-skeleton" style="width: 100%; height: 8px;" />
                        </div>
                    }</For>
                </Show>

                <For each={fleetData()}>
                    {(host) => (
                        <div style="border: 1px solid var(--border-primary); padding: 16px; background: rgba(0,0,0,0.2); border-left: 3px solid transparent; border-left-color: clamp(var(--status-online), 100%, 100%); transition: all 120ms ease; position: relative;">
                            <div style={`position: absolute; left: -3px; top: 0; bottom: 0; width: 3px; background: ${calculateOverallHealthColor(host)};`} />

                            <div style="font-size: 12px; font-weight: 700; color: var(--text-primary); font-family: var(--font-mono); margin-bottom: 16px;">
                                {host.host_id}
                            </div>

                            <div style="display: flex; flex-direction: column; gap: 12px;">
                                <div style="display: grid; grid-template-columns: 40px 1fr; align-items: center; gap: 8px;">
                                    <span style="font-size: 10px; color: var(--text-muted); font-family: var(--font-mono); text-transform: uppercase;">CPU</span>
                                    <div style="width: 100%; height: 4px; background: rgba(255,255,255,0.1);">
                                        <div style={`height: 100%; width: ${host.cpu_usage}%; background: ${getHeatColor(host.cpu_usage)};`} />
                                    </div>
                                </div>

                                <div style="display: grid; grid-template-columns: 40px 1fr; align-items: center; gap: 8px;">
                                    <span style="font-size: 10px; color: var(--text-muted); font-family: var(--font-mono); text-transform: uppercase;">MEM</span>
                                    <div style="width: 100%; height: 4px; background: rgba(255,255,255,0.1);">
                                        <div style={`height: 100%; width: ${(host.mem_used_mb / host.mem_total_mb * 100)}%; background: ${getHeatColor((host.mem_used_mb / host.mem_total_mb * 100))};`} />
                                    </div>
                                </div>

                                <div style="display: grid; grid-template-columns: 40px 1fr; align-items: center; gap: 8px;">
                                    <span style="font-size: 10px; color: var(--text-muted); font-family: var(--font-mono); text-transform: uppercase;">DISK</span>
                                    <div style="width: 100%; height: 4px; background: rgba(255,255,255,0.1);">
                                        <div style={`height: 100%; width: ${(host.disk_used_gb / host.disk_total_gb * 100)}%; background: ${getHeatColor((host.disk_used_gb / host.disk_total_gb * 100))};`} />
                                    </div>
                                </div>
                            </div>
                        </div>
                    )}
                </For>

                <Show when={fleetData().length === 0 && !loading()}>
                    <div style="grid-column: 1 / -1; text-align: center; padding: 24px; color: var(--text-muted); font-size: 11px; font-family: var(--font-mono); text-transform: uppercase;">
                        WAITING_FOR_TELEMETRY_DATA
                    </div>
                </Show>
            </div>
        </div>
    );
};

export default FleetHeatmap;
