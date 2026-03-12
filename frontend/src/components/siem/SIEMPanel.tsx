import { Component, createSignal, Show, For } from 'solid-js';
import { useApp } from '@core/store';
import { ThreatMap } from './ThreatMap';
import { useNavigate, useLocation } from '@solidjs/router';
import { EmptyState } from '../ui/EmptyState';
import { ThreatIntelPanel } from './ThreatIntelPanel';
import { AlertDashboard } from './AlertDashboard';
import { CompliancePanel } from '../compliance/CompliancePanel';
import { MitreHeatmap } from './MitreHeatmap';
import { LiveTailPanel } from '../ops/LiveTailPanel';
import '../../styles/siem.css';

export const SIEMPanel: Component = () => {
    const [state] = useApp();
    const [selectedHost, setSelectedHost] = createSignal<string>('global');
    const navigate = useNavigate();
    const location = useLocation();

    // Map pathname to active tab visually
    const activeTab = () => {
        const path = location.pathname;
        if (path.includes('intel')) return 'intel';
        if (path.includes('compliance')) return 'compliance';
        if (path.includes('alerts')) return 'alerts';
        if (path.includes('mitre')) return 'mitre';
        return 'logs';
    };

    return (
        <div class="ob-page page-enter">
            <div class="ob-tabs" style="margin-bottom: 24px;">
                <button
                    class={`ob-tab ${activeTab() === 'logs' ? 'active' : ''}`}
                    onClick={() => navigate('/siem')}
                >
                    Terminal Forensics
                </button>
                <button
                    class={`ob-tab ${activeTab() === 'intel' ? 'active' : ''}`}
                    onClick={() => navigate('/siem/intel')}
                >
                    Threat Intelligence
                </button>
                <button
                    class={`ob-tab ${activeTab() === 'compliance' ? 'active' : ''}`}
                    onClick={() => navigate('/siem/compliance')}
                >
                    Compliance Workflow
                </button>
                <button
                    class={`ob-tab ${activeTab() === 'alerts' ? 'active' : ''}`}
                    onClick={() => navigate('/siem/alerts')}
                >
                    Active Alerts
                </button>
                <button
                    class={`ob-tab ${activeTab() === 'mitre' ? 'active' : ''}`}
                    onClick={() => navigate('/siem/mitre')}
                >
                    MITRE Matrix
                </button>
            </div>

            <Show when={activeTab() === 'logs'}>
                <div style="flex: 1; min-height: 0; display: flex; flex-direction: column;">
                    <LiveTailPanel />
                </div>
            </Show>

            <Show when={activeTab() === 'intel'}>
                <div style="display: flex; flex-direction: column; gap: 24px; flex: 1; min-height: 0;">
                    <ThreatIntelPanel />

                    <div style="display: flex; align-items: center; gap: 12px;">
                        <span style="font-size: 11px; font-weight: 700; color: var(--text-muted); text-transform: uppercase;">Host Selection:</span>
                        <select
                            class="ob-select ob-select-sm"
                            style="width: 250px;"
                            value={selectedHost() === 'global' ? (state.hosts[0]?.id || '') : selectedHost()}
                            onChange={(e) => setSelectedHost(e.currentTarget.value)}
                        >
                            <For each={state.hosts}>
                                {(host) => (
                                    <option value={host.id}>{host.label || host.hostname}</option>
                                )}
                            </For>
                        </select>
                    </div>

                    <div style="flex: 1; min-height: 0;">
                        <Show when={selectedHost() && selectedHost() !== 'global'}>
                            <ThreatMap hostId={selectedHost()} />
                        </Show>
                        <Show when={selectedHost() === 'global' && state.hosts.length > 0}>
                            <ThreatMap hostId={state.hosts[0].id} />
                        </Show>
                        <Show when={state.hosts.length === 0}>
                            <EmptyState
                                icon="SECURITY_SHIELD"
                                title="NO HOSTS CONFIGURED"
                                description="Host telemetry required for forensic initialization. Configure edge nodes to proceed."
                            />
                        </Show>
                    </div>
                </div>
            </Show>

            <Show when={activeTab() === 'compliance'}>
                <div style="flex: 1; min-height: 0; display: flex; flex-direction: column;">
                    <CompliancePanel />
                </div>
            </Show>

            <Show when={activeTab() === 'alerts'}>
                <div style="flex: 1; min-height: 0; display: flex; flex-direction: column;">
                    <AlertDashboard />
                </div>
            </Show>

            <Show when={activeTab() === 'mitre'}>
                <div style="flex: 1; min-height: 0; display: flex; flex-direction: column;">
                    <MitreHeatmap />
                </div>
            </Show>
        </div>
    );
};
