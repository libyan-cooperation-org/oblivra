import { Component, createSignal, onMount, For, Show } from 'solid-js';
import { IS_BROWSER } from '@core/context';
import { database } from '../../../wailsjs/go/models';
import { EmptyState } from '../ui/EmptyState';
import '../../styles/siem.css';

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

    return (
        <div class="alert-dashboard">
            <header class="alert-header">
                <div>
                    <h2 class="host-name" style="margin:0;">Mission-Critical Incidents</h2>
                    <p class="host-id" style="margin:4px 0 0 0;">AGGREGATED_THREAT_DETECTION</p>
                </div>
                <button class="tactical-btn secondary" onClick={fetchIncidents}>
                    <svg width="12" height="12" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" style="margin-right: 4px"><path d="M21.5 2v6h-6M21.34 15.57a10 10 0 1 1-.59-9.21l5.67-5.67" /></svg>
                    SYNC_STATE
                </button>
            </header>

            <div class="incident-filters">
                <button
                    class={`filter-btn ${statusFilter() === 'New' ? 'active' : ''}`}
                    onClick={() => { setStatusFilter('New'); fetchIncidents(); }}
                >
                    NEW_THREATS
                </button>
                <button
                    class={`filter-btn ${statusFilter() === 'Active' ? 'active' : ''}`}
                    onClick={() => { setStatusFilter('Active'); fetchIncidents(); }}
                >
                    ACTIVE
                </button>
                <button
                    class={`filter-btn ${statusFilter() === 'Investigating' ? 'active' : ''}`}
                    onClick={() => { setStatusFilter('Investigating'); fetchIncidents(); }}
                >
                    INVESTIGATING
                </button>
                <button
                    class={`filter-btn ${statusFilter() === 'Closed' ? 'active' : ''}`}
                    onClick={() => { setStatusFilter('Closed'); fetchIncidents(); }}
                >
                    RESOLVED
                </button>
            </div>

            <Show when={error()}>
                <div style="padding: 12px; border-bottom: 1px solid var(--status-offline); color: var(--status-offline); font-family: var(--font-mono); font-size: 11px; background: rgba(239,68,68,0.05);">
                    CRITICAL_INTERFACE_ERROR: {error()}
                </div>
            </Show>

            <Show when={loading() && incidents().length === 0}>
                <div class="host-id" style="padding: 20px;">POLLING_THREAT_DATABASE...</div>
            </Show>

            <Show when={!loading() && incidents().length === 0 && !error()}>
                <div style="padding: 40px;">
                    <EmptyState
                        icon="SHIELD_OK"
                        title={`NO ${statusFilter().toUpperCase()} INCIDENTS`}
                        description="Heuristic engine reports zero critical threshold exceedances for the selected filter."
                    />
                </div>
            </Show>

            <Show when={incidents().length > 0}>
                <div class="incident-grid">
                    <For each={incidents()}>
                        {(incident) => (
                            <div class={`incident-card ${(incident.severity || 'low').toLowerCase()}`}>
                                <div class="incident-card-header">
                                    <div>
                                        <div class="incident-title">{incident.title}</div>
                                        <div class="incident-meta">
                                            <span class={`severity-badge severity-${(incident.severity || 'low').toLowerCase()}`}>
                                                {(incident.severity || 'LOW').toUpperCase()}
                                            </span>
                                            <span>{new Date(incident.first_seen_at as unknown as string).toISOString().replace('T', ' ').slice(0, 19)}</span>

                                        </div>
                                    </div>
                                    <div style="font-size: 10px; color: var(--text-muted); font-family: var(--font-mono);">
                                        {incident.id.split('-').pop()}
                                    </div>
                                </div>

                                <div class="incident-body">
                                    <div class="incident-description">
                                        {incident.description}
                                    </div>

                                    <div class="incident-stats">
                                        <div class="stat-item">
                                            <span class="stat-item-label">Entity</span>
                                            <span class="stat-item-value">{incident.group_key}</span>
                                        </div>
                                        <div class="stat-item">
                                            <span class="stat-item-label">Events</span>
                                            <span class="stat-item-value">{incident.event_count}</span>
                                        </div>
                                        <div class="stat-item">
                                            <span class="stat-item-label">Rule</span>
                                            <span class="stat-item-value">{incident.rule_id}</span>
                                        </div>
                                    </div>
                                </div>

                                <div class="incident-footer">
                                    <Show when={incident.status === 'New' || incident.status === 'Active'}>
                                        <button class="tactical-btn" onClick={() => handleStatusChange(incident.id, 'Investigating')}>
                                            INVESTIGATE
                                        </button>
                                        <button class="tactical-btn primary" onClick={() => handleStatusChange(incident.id, 'Closed')}>
                                            RESOLVE
                                        </button>
                                    </Show>
                                    <Show when={incident.status === 'Investigating'}>
                                        <button class="tactical-btn primary" onClick={() => handleStatusChange(incident.id, 'Closed')}>
                                            RESOLVE
                                        </button>
                                    </Show>
                                    <Show when={incident.status === 'Closed'}>
                                        <button class="tactical-btn" onClick={() => handleStatusChange(incident.id, 'New')}>
                                            REOPEN
                                        </button>
                                    </Show>
                                </div>
                            </div>
                        )}
                    </For>
                </div>
            </Show>
        </div>
    );
};

