import { Component, createSignal, onMount, For, Show } from 'solid-js';
import { IS_BROWSER } from '@core/context';
import { database } from '../../../wailsjs/go/models';
import { 
    EmptyState, 
    Panel, 
    Badge, 
    Button, 
    Toolbar, 
    ToolbarSpacer, 
    TabBar,
    KPIGrid,
    KPI,
    normalizeSeverity,
    formatTimestamp 
} from '@components/ui';
import '../../styles/alert-dashboard.css';

export const AlertDashboard: Component = () => {
    const [incidents, setIncidents] = createSignal<database.Incident[]>([]);
    const [loading, setLoading] = createSignal(true);
    const [error, setError] = createSignal<string | null>(null);
    const [statusFilter, setStatusFilter] = createSignal<string>('New');

    const fetchIncidents = async () => {
        setLoading(true);
        setError(null);
        if (IS_BROWSER) { setLoading(false); return; }
        try {
            const { ListIncidents } = await import('../../../wailsjs/go/services/AlertingService');
            const res = await ListIncidents(statusFilter(), 100);
            setIncidents(res || []);
        } catch (err) {
            setError(String(err));
        } finally {
            setLoading(false);
        }
    };

    onMount(() => {
        fetchIncidents();
    });

    const handleStatusChange = async (id: string, newStatus: string) => {
        if (IS_BROWSER) return;
        try {
            const { UpdateIncidentStatus } = await import('../../../wailsjs/go/services/AlertingService');
            await UpdateIncidentStatus(id, newStatus, 'Status updated via Tactical Dashboard');
            fetchIncidents();
        } catch (err) {
            console.error('Failed to update incident:', err);
            setError(String(err));
        }
    };

    const tabs = [
        { id: 'New', label: 'NEW_THREATS' },
        { id: 'Active', label: 'ACTIVE' },
        { id: 'Investigating', label: 'INVESTIGATING' },
        { id: 'Closed', label: 'RESOLVED' }
    ];

    return (
        <div class="alerts-container">
            <Toolbar>
                <TabBar 
                    tabs={tabs} 
                    active={statusFilter()} 
                    onSelect={(id) => { setStatusFilter(id); fetchIncidents(); }} 
                    class="compact"
                />
                <ToolbarSpacer />
                <Button 
                    variant="ghost" 
                    size="sm" 
                    onClick={fetchIncidents}
                >
                    REFRESH_INCIDENTS
                </Button>
            </Toolbar>

            <div class="alerts-scroll-area">
                <Show when={error()}>
                    <div class="alerts-error-notice">
                        <strong>CRITICAL_INTERFACE_ERROR:</strong> {error()}
                    </div>
                </Show>

                <Show when={incidents().length > 0}>
                    <KPIGrid cols={4} class="mb-6">
                        <KPI 
                            label="TOTAL_INCIDENTS" 
                            value={incidents().length} 
                        />
                        <KPI 
                            label="CRITICAL_ALERTS" 
                            value={incidents().filter(i => i.severity === 'Critical').length} 
                            color="var(--alert-critical)" 
                        />
                        <KPI 
                            label="INVESTIGATIONS" 
                            value={incidents().filter(i => i.status === 'Investigating').length} 
                            color="var(--alert-medium)" 
                        />
                        <KPI 
                            label="RESOLVED_EPOCH" 
                            value={incidents().filter(i => i.status === 'Closed').length} 
                            color="var(--status-online)" 
                        />
                    </KPIGrid>

                    <div class="alerts-grid">
                        <For each={incidents()}>
                            {(incident) => (
                                <Panel 
                                    title={incident.title}
                                    subtitle={`ID:${incident.id.split('-').pop()?.toUpperCase()}`}
                                    actions={
                                        <Badge severity={normalizeSeverity(incident.severity || 'low')}>
                                            {(incident.severity || 'LOW').toUpperCase()}
                                        </Badge>
                                    }
                                >
                                    <div class="alert-panel-content">
                                        <div class="alert-description">
                                            {incident.description}
                                        </div>
                                        <div class="alert-timestamp">
                                            DETECTED_{formatTimestamp(incident.first_seen_at as any)}
                                        </div>
                                    </div>

                                    <div class="alert-stats-strip">
                                        <div class="alert-stat-item">
                                            <span class="alert-stat-label">Entity</span>
                                            <span class="alert-stat-value entity-id">{incident.group_key}</span>
                                        </div>
                                        <div class="alert-stat-item">
                                            <span class="alert-stat-label">Events</span>
                                            <span class="alert-stat-value">{incident.event_count}</span>
                                        </div>
                                        <div class="alert-stat-item">
                                            <span class="alert-stat-label">Rule_ID</span>
                                            <span class="alert-stat-value">{incident.rule_id}</span>
                                        </div>
                                    </div>

                                    <div class="alert-actions">
                                        <Show when={incident.status === 'New' || incident.status === 'Active'}>
                                            <Button variant="ghost" size="sm" onClick={() => handleStatusChange(incident.id, 'Investigating')}>
                                                INVESTIGATE
                                            </Button>
                                            <Button variant="primary" size="sm" onClick={() => handleStatusChange(incident.id, 'Closed')}>
                                                RESOLVE
                                            </Button>
                                        </Show>
                                        <Show when={incident.status === 'Investigating'}>
                                            <Button variant="primary" size="sm" onClick={() => handleStatusChange(incident.id, 'Closed')}>
                                                RESOLVE
                                            </Button>
                                        </Show>
                                        <Show when={incident.status === 'Closed'}>
                                            <Button variant="ghost" size="sm" onClick={() => handleStatusChange(incident.id, 'New')}>
                                                REOPEN
                                            </Button>
                                        </Show>
                                    </div>
                                </Panel>
                            )}
                        </For>
                    </div>
                </Show>

                <Show when={loading() && incidents().length === 0}>
                    <div style="padding: 100px 0;">
                        <EmptyState 
                            icon="📡" 
                            title="POLLING_THREAT_DATABASE" 
                            description="Fetching latest incident records from the alerting heuristic engine..." 
                        />
                    </div>
                </Show>

                <Show when={!loading() && incidents().length === 0 && !error()}>
                    <div style="padding: 100px 0;">
                        <EmptyState
                            icon="🛡️"
                            title={`NO_${statusFilter().toUpperCase()}_INCIDENTS`}
                            description="Heuristic engine reports zero threshold exceedances for the current filter."
                        />
                    </div>
                </Show>
            </div>
        </div>
    );
};
