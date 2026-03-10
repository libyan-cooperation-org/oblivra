import { Component, createSignal, onMount, For, Show } from 'solid-js';
import { ListIncidents, UpdateIncidentStatus } from '../../../wailsjs/go/app/IncidentService';
import { database } from '../../../wailsjs/go/models';
import { PlaybookBuilder } from './PlaybookBuilder';

export const CommandCenter: Component = () => {
    const [incidents, setIncidents] = createSignal<database.Incident[]>([]);
    const [selectedId, setSelectedId] = createSignal<string | null>(null);
    const [loading, setLoading] = createSignal(true);

    onMount(async () => {
        try {
            const list = await ListIncidents(null as any, "", "", 50);
            setIncidents(list || []);
        } catch (err) {
            console.error("Failed to load incidents:", err);
        } finally {
            setLoading(false);
        }
    });

    const selectedIncident = () => incidents().find(i => i.id === selectedId());

    const handleUpdateStatus = async (id: string, status: string) => {
        try {
            await UpdateIncidentStatus(null as any, id, status, "Manual update from Command Center");
            const updated = await ListIncidents(null as any, "", "", 50);
            setIncidents(updated || []);
        } catch (err) {
            console.error("Failed to update status:", err);
        }
    };

    return (
        <div class="incident-page">
            <header class="page-header">
                <div>
                    <h1 class="glow-text">COMMAND CENTER</h1>
                    <p class="text-muted">Autonomous Neutralization & Case Management</p>
                </div>
                <div class="header-actions">
                    <button class="action-button secondary">EXPORT FORENSICS</button>
                    <button class="action-button">NEW INVESTIGATION</button>
                </div>
            </header>

            <div class="incident-grid">
                <section class="case-list">
                    <Show when={!loading()} fallback={<div class="loading-shimmer">Scanning repositories...</div>}>
                        <For each={incidents()}>
                            {(inc) => (
                                <div
                                    class={`case-card severity-${inc.severity} ${selectedId() === inc.id ? 'active' : ''}`}
                                    onClick={() => setSelectedId(inc.id)}
                                >
                                    <div class="case-header">
                                        <h3>{inc.title}</h3>
                                        <span class="case-status">{inc.status}</span>
                                    </div>
                                    <p class="case-description">{inc.description}</p>
                                    <div class="case-meta">
                                        <span>ID: {inc.id}</span>
                                        <span>Entity: {inc.group_key}</span>
                                        <span>Last Seen: {new Date(inc.last_seen_at.toString()).toLocaleTimeString()}</span>
                                    </div>
                                </div>
                            )}
                        </For>
                    </Show>
                </section>

                <aside class="case-details">
                    <Show when={selectedIncident()} fallback={<div class="empty-state">Select a case to view investigation details</div>}>
                        <div class="detail-content animate-slide-in">
                            <h2>Case Details: {selectedIncident()?.id}</h2>

                            <div class="detail-section">
                                <h3>Automated Playbooks</h3>
                                <PlaybookBuilder incident={selectedIncident()!} />
                            </div>

                            <div class="detail-section">
                                <h3>Actions</h3>
                                <div class="action-grid">
                                    <button onClick={() => handleUpdateStatus(selectedIncident()!.id, "In Progress")}>Mark Active</button>
                                    <button onClick={() => handleUpdateStatus(selectedIncident()!.id, "Resolved")}>Resolve</button>
                                    <button class="danger" onClick={() => handleUpdateStatus(selectedIncident()!.id, "Closed")}>Close Case</button>
                                </div>
                            </div>
                        </div>
                    </Show>
                </aside>
            </div>
        </div>
    );
};
