import { Component, createSignal, Show } from 'solid-js';
import { useApp } from '@core/store';
import { useNavigate, useLocation } from '@solidjs/router';
import { 
    PageLayout, 
    TabBar, 
    Select, 
    EmptyState 
} from '@components/ui';

import { ThreatMap } from './ThreatMap';
import { ThreatIntelPanel } from './ThreatIntelPanel';
import { AlertDashboard } from './AlertDashboard';
import { CompliancePanel } from '../compliance/CompliancePanel';
import { MitreHeatmap } from './MitreHeatmap';
import { TerminalForensics } from './TerminalForensics';
import '../../styles/siem-panel.css';

export const SIEMPanel: Component = () => {
    const [state] = useApp();
    const [selectedHost, setSelectedHost] = createSignal<string>('global');
    const navigate = useNavigate();
    const location = useLocation();

    // Mapping routes to tab IDs
    const tabs = [
        { id: 'logs', label: 'TERMINAL_FORENSICS' },
        { id: 'intel', label: 'THREAT_INTELLIGENCE' },
        { id: 'compliance', label: 'COMPLIANCE_WORKFLOW' },
        { id: 'alerts', label: 'ACTIVE_ALERTS' },
        { id: 'mitre', label: 'MITRE_MATRIX' }
    ];

    const activeTab = () => {
        const path = location.pathname;
        if (path.includes('intel')) return 'intel';
        if (path.includes('compliance')) return 'compliance';
        if (path.includes('alerts')) return 'alerts';
        if (path.includes('mitre')) return 'mitre';
        return 'logs';
    };

    const handleTabSelect = (id: string) => {
        const route = id === 'logs' ? '/siem' : `/siem/${id}`;
        navigate(route);
    };

    return (
        <PageLayout 
            title="Cyber-Intelligence & SIEM"
            subtitle="ADVANCED_THREAT_DETECTION_ENGINE"
        >
            <TabBar 
                tabs={tabs} 
                active={activeTab()} 
                onSelect={handleTabSelect} 
                class="mb-6"
            />

            <div class="siem-content-wrap">
                <Show when={activeTab() === 'logs'}>
                    <div class="siem-tab-content">
                        <TerminalForensics />
                    </div>
                </Show>

                <Show when={activeTab() === 'intel'}>
                    <div class="siem-scroll-container">
                        <ThreatIntelPanel />

                        <div class="host-selection-strip">
                            <span class="host-selection-label">Host Selection:</span>
                            <Select
                                style="width: 250px;"
                                value={selectedHost() === 'global' ? (state.hosts[0]?.id || '') : selectedHost()}
                                onChange={setSelectedHost}
                                options={state.hosts.map(h => ({ label: (h.label || h.hostname).toUpperCase(), value: h.id }))}
                            />
                        </div>

                        <div class="threat-map-container">
                            <Show when={selectedHost() && selectedHost() !== 'global'}>
                                <ThreatMap hostId={selectedHost()} />
                            </Show>
                            <Show when={selectedHost() === 'global' && state.hosts.length > 0}>
                                <ThreatMap hostId={state.hosts[0].id} />
                            </Show>
                            <Show when={state.hosts.length === 0}>
                                <EmptyState
                                    icon="🛡️"
                                    title="NO_HOSTS_CONFIGURED"
                                    description="Host telemetry required for forensic initialization. Configure edge nodes to proceed."
                                />
                            </Show>
                        </div>
                    </div>
                </Show>

                <Show when={activeTab() === 'compliance'}>
                    <div class="siem-full-height">
                        <CompliancePanel />
                    </div>
                </Show>

                <Show when={activeTab() === 'alerts'}>
                    <div class="siem-full-height">
                        <AlertDashboard />
                    </div>
                </Show>

                <Show when={activeTab() === 'mitre'}>
                    <div class="siem-full-height">
                        <MitreHeatmap />
                    </div>
                </Show>
            </div>
        </PageLayout>
    );
};
