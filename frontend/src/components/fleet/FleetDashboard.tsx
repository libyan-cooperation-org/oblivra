import { Component, createSignal, onMount, onCleanup, For, Show } from 'solid-js';
import { useApp } from '@core/store';
import { IS_BROWSER } from '@core/context';
import { DashboardCard } from './DashboardCard';
import FleetHeatmap from '../intelligence/FleetHeatmap';
import { EmptyState } from '../ui/EmptyState';
import '../../styles/fleet.css';

export const FleetDashboard: Component = () => {
    const [state] = useApp();
    type HostTelemetry = { host_id: string, cpu_usage: number, mem_used_mb: number, mem_total_mb: number, disk_used_gb: number, disk_total_gb: number, load_avg: number };
    const [telemetryMap, setTelemetryMap] = createSignal<Record<string, HostTelemetry>>({});
    const [loading, setLoading] = createSignal(true);

    const updateTelemetry = async () => {
        if (IS_BROWSER) { setLoading(false); return; }
        try {
            const { GetFleetTelemetry } = await import('../../../wailsjs/go/services/TelemetryService');
            const data = await GetFleetTelemetry();
            if (data && data.length > 0) {
                const newMap: Record<string, HostTelemetry> = {};
                data.forEach((t: any) => { newMap[t.host_id] = t as HostTelemetry; });
                setTelemetryMap(newMap);
            }
        } catch (err) {
            console.error('Failed to fetch fleet telemetry:', err);
        } finally {
            setLoading(false);
        }
    };

    let intervalId: ReturnType<typeof setInterval> | undefined;

    onMount(() => {
        updateTelemetry();
        intervalId = setInterval(updateTelemetry, 5000); // 5s refresh for UI
    });

    onCleanup(() => {
        if (intervalId) clearInterval(intervalId);
    });

    const monitoredHosts = () => {
        return state.hosts.filter(h => telemetryMap()[h.id]);
    };

    return (
        <div class="ob-page page-enter">
            <FleetHeatmap />

            <div class="fleet-grid" style="display: grid; grid-template-columns: repeat(auto-fill, minmax(350px, 1fr)); gap: 24px; margin-top: 24px;">
                <Show when={loading() && Object.keys(telemetryMap()).length === 0}>
                    <For each={[1, 2, 3]}>
                        {() => (
                            <div class="ob-card" style="padding: 20px; display: flex; flex-direction: column; gap: 16px;">
                                <div style="display: flex; justify-content: space-between; align-items: flex-start;">
                                    <div style="width: 60%;">
                                        <div class="ob-skeleton" style="width: 100%; height: 16px; margin-bottom: 4px; border-radius: 2px;" />
                                        <div class="ob-skeleton" style="width: 70%; height: 10px; border-radius: 2px;" />
                                    </div>
                                    <div class="ob-skeleton" style="width: 8px; height: 8px; border-radius: 50%;" />
                                </div>
                                <div class="ob-skeleton" style="width: 100%; height: 40px; margin: 8px 0; border-radius: 2px;" />
                                <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 16px;">
                                    <For each={[1, 2, 3, 4]}>
                                        {() => (
                                            <div style="display: flex; flex-direction: column; gap: 4px;">
                                                <div class="ob-skeleton" style="width: 80%; height: 9px; border-radius: 2px;" />
                                                <div class="ob-skeleton" style="width: 50%; height: 14px; border-radius: 2px;" />
                                                <div class="ob-skeleton" style="width: 100%; height: 2px; margin-top: 4px; border-radius: 1px;" />
                                            </div>
                                        )}
                                    </For>
                                </div>
                            </div>
                        )}
                    </For>
                </Show>

                <For each={monitoredHosts()}>
                    {(host) => (
                        <DashboardCard
                            hostId={host.id}
                            hostLabel={host.label || host.hostname}
                            telemetry={telemetryMap()[host.id]}
                        />
                    )}
                </For>
            </div>

            <Show when={monitoredHosts().length === 0 && !loading() && Object.keys(telemetryMap()).length === 0}>
                <div style="margin-top: 24px;">
                    <EmptyState
                        icon="VACUUM"
                        title="NO ACTIVE TELEMETRY"
                        description="Telemetry is aggregated automatically upon SSH tunnel establishment. Establish host connection to proceed."
                    />
                </div>
            </Show>
        </div>
    );
};
