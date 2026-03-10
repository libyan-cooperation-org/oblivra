import { Component, createSignal, createResource, For, Show } from 'solid-js';
import * as App from '../../../wailsjs/go/app/App';
import { ForensicView } from './ForensicView';
import * as UEBAService from '../../../wailsjs/go/app/UEBAService';
import * as IncidentService from '../../../wailsjs/go/app/IncidentService';
import { ueba } from '../../../wailsjs/go/models';

interface EntityProfile extends ueba.EntityProfile { }

export const ThreatHunter: Component = () => {
    const [searchQuery, setSearchQuery] = createSignal('');
    const [searchResults, setSearchResults] = createSignal<any[]>([]);
    const [selectedEntity, setSelectedEntity] = createSignal<EntityProfile | null>(null);
    const [isHunting, setIsHunting] = createSignal(false);
    const [showForensics, setShowForensics] = createSignal<string | null>(null);

    // Fetch profiles from UEBA
    const [profiles, { refetch }] = createResource<EntityProfile[]>(async () => {
        if (!UEBAService) return [];
        return await UEBAService.GetProfiles();
    });

    const handleSearch = async (e: Event) => {
        e.preventDefault();
        setIsHunting(true);
        try {
            const results = await App.SearchLogs(searchQuery(), 'lucene', 50, 0);
            setSearchResults(results);
        } catch (err) {
            console.error('Hunting search failed:', err);
        } finally {
            setIsHunting(false);
        }
    };

    const createCase = async (entity: EntityProfile) => {
        if (!IncidentService) return;
        const confirmed = confirm(`Create formal incident case for ${entity.id}?`);
        if (!confirmed) return;

        try {
            await IncidentService.Upsert({} as any, {
                title: `Behavorial Anomaly: ${entity.id}`,
                description: `High risk score (${(entity.risk_score * 100).toFixed(0)}%) detected via ${entity.type} profiling. Peer Group: ${entity.peer_group_id}`,
                severity: entity.risk_score > 0.8 ? 'Critical' : 'High',
                status: 'New',
                group_key: entity.id,
                first_seen_at: new Date().toISOString() as any,
                last_seen_at: new Date().toISOString() as any,
                mitre_tactics: ['Initial Access', 'Persistence'],
            } as any);
            alert('Incident case created successfully.');
        } catch (err) {
            console.error('Failed to create case:', err);
        }
    };


    return (
        <div class="ob-page page-enter" style="display: flex; height: 100%; overflow: hidden; padding: 0;">
            {/* Sidebar: Entity Watchlist */}
            <aside style="width: 320px; border-right: 1px solid var(--border-primary); background: var(--bg-surface); display: flex; flex-direction: column;">
                <header style="padding: 24px; border-bottom: 1px solid var(--border-primary); background: rgba(0,0,0,0.2);">
                    <h2 style="font-size: 11px; font-weight: 800; text-transform: uppercase; letter-spacing: 1px; color: var(--text-muted); margin: 0 0 4px 0;">Entity Watchlist</h2>
                    <p style="font-size: 10px; color: var(--text-muted); font-style: italic; margin: 0;">Sorted by Risk Criticality</p>
                </header>
                <div style="flex: 1; overflow-y: auto;">
                    <Show when={!profiles.loading} fallback={
                        <div style="padding: 32px; text-align: center; color: var(--text-muted); font-size: 11px; text-transform: uppercase; font-family: var(--font-mono); opacity: 0.5;">Scanning Profiles...</div>
                    }>
                        <For each={profiles()?.sort((a, b) => b.risk_score - a.risk_score)}>
                            {(p) => (
                                <button
                                    onClick={() => setSelectedEntity(p)}
                                    style={`width: 100%; padding: 20px; display: flex; flex-direction: column; align-items: flex-start; border-bottom: 1px solid var(--border-primary); transition: all 120ms ease; text-align: left; position: relative; background: ${selectedEntity()?.id === p.id ? 'var(--bg-elevated)' : 'transparent'}; cursor: pointer; border-left: 3px solid ${selectedEntity()?.id === p.id ? 'var(--accent-primary)' : 'transparent'};`}
                                    onMouseEnter={(e) => { if (selectedEntity()?.id !== p.id) e.currentTarget.style.background = 'rgba(255,255,255,0.02)'; }}
                                    onMouseLeave={(e) => { if (selectedEntity()?.id !== p.id) e.currentTarget.style.background = 'transparent'; }}
                                >
                                    <div style="display: flex; justify-content: space-between; width: 100%; margin-bottom: 8px;">
                                        <span style={`font-size: 10px; font-weight: 800; text-transform: uppercase; letter-spacing: 1px; color: ${p.type === 'user' ? 'var(--accent-primary)' : 'var(--status-online)'};`}>{p.type}</span>
                                        <span style={`font-size: 12px; font-weight: 800; color: ${p.risk_score > 0.8 ? 'var(--status-offline)' : (p.risk_score > 0.5 ? 'var(--status-degraded)' : 'var(--status-online)')};`}>{(p.risk_score * 100).toFixed(0)}%</span>
                                    </div>
                                    <div style="font-size: 14px; font-weight: 700; color: var(--text-primary); margin-bottom: 4px; width: 100%; white-space: nowrap; overflow: hidden; text-overflow: ellipsis;">{p.id}</div>
                                    <div style="font-size: 10px; color: var(--text-muted); font-family: var(--font-mono); opacity: 0.6;">PEER_GROUP: {p.peer_group_id || 'DEFAULT'}</div>
                                </button>
                            )}
                        </For>
                    </Show>
                </div>
                <footer style="padding: 16px; border-top: 1px solid var(--border-primary);">
                    <button onClick={refetch} class="ob-btn ob-btn-secondary" style="width: 100%;">Refresh Profiles</button>
                </footer>
            </aside>

            {/* Main Area: Investigation Workspace */}
            <main style="flex: 1; display: flex; flex-direction: column; background: var(--bg-base); position: relative; overflow: hidden;">
                {/* Unified Hunting Bar */}
                <div style="padding: 24px; background: rgba(0,0,0,0.5); border-bottom: 1px solid var(--border-primary); position: sticky; top: 0; z-index: 10; backdrop-filter: blur(8px);">
                    <form onSubmit={handleSearch} style="display: flex; gap: 16px;">
                        <input
                            type="text"
                            placeholder="HUNTING_QUERY: query (e.g. user:admin OR host:prod*)"
                            value={searchQuery()}
                            onInput={(e) => setSearchQuery(e.currentTarget.value)}
                            class="ob-input"
                            style="flex: 1; font-family: var(--font-mono);"
                        />
                        <button
                            type="submit"
                            disabled={isHunting()}
                            class="ob-btn ob-btn-primary"
                            style="padding: 0 32px;"
                        >
                            {isHunting() ? 'HUNTING...' : 'INITIATE HUNT'}
                        </button>
                    </form>
                </div>

                <div style="flex: 1; overflow-y: auto; overflow-x: hidden; padding: 32px;">
                    <Show when={selectedEntity()} fallback={
                        <div style="display: flex; flex-direction: column; items-center: center; justify-content: center; height: 100%; text-align: center;">
                            <p style="font-size: 11px; font-family: var(--font-mono); text-transform: uppercase; letter-spacing: 4px; color: var(--text-muted); margin-bottom: 8px;">Awaiting_Entity_Selection</p>
                            <p style="font-size: 10px; color: var(--text-muted); font-style: italic;">Select a profile from the watchlist to initiate deep analysis</p>
                        </div>
                    }>
                        <div style="display: flex; flex-direction: column; gap: 32px;" class="page-enter">
                            {/* Entity Summary Card */}
                            <div style="display: grid; grid-template-columns: 1fr 3fr; gap: 24px;">
                                <div class="ob-card" style="padding: 24px;">
                                    <div style="font-size: 10px; font-weight: 800; color: var(--text-muted); text-transform: uppercase; letter-spacing: 1px; margin-bottom: 16px;">Aggregate Risk</div>
                                    <div style={`font-size: 48px; font-weight: 800; margin-bottom: 8px; color: ${selectedEntity()!.risk_score > 0.8 ? 'var(--status-offline)' : (selectedEntity()!.risk_score > 0.5 ? 'var(--status-degraded)' : 'var(--status-online)')};`}>
                                        {(selectedEntity()!.risk_score * 100).toFixed(0)}<span style="font-size: 14px; opacity: 0.5;">%</span>
                                    </div>
                                    <div style="font-size: 10px; color: var(--text-muted); font-family: var(--font-mono); text-transform: uppercase;">Last Seen: {new Date(selectedEntity()!.last_seen as any).toLocaleString()}</div>
                                </div>

                                <div class="ob-card" style="padding: 24px; position: relative;">
                                    <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 24px;">
                                        <div style="font-size: 10px; font-weight: 800; color: var(--text-muted); text-transform: uppercase; letter-spacing: 1px;">Risk Trajectory</div>
                                        <div style="display: flex; gap: 8px;">
                                            <button onClick={() => createCase(selectedEntity()!)} class="ob-btn ob-btn-secondary ob-btn-sm" style="border-color: var(--status-offline); color: var(--status-offline);">CREATE CASE</button>
                                            <button onClick={() => setShowForensics(selectedEntity()!.id)} class="ob-btn ob-btn-primary ob-btn-sm">DEEP DIVE</button>
                                        </div>
                                    </div>
                                    <div style="height: 112px; display: flex; align-items: flex-end; gap: 6px; padding: 8px 16px; background: rgba(0,0,0,0.3); border: 1px solid var(--border-primary); border-radius: 4px; position: relative;">
                                        <div style="position: absolute; top: 8px; right: 12px; font-size: 8px; font-family: var(--font-mono); color: var(--text-muted); text-transform: uppercase; letter-spacing: -0.5px;">Live_Telemetry_Feed</div>
                                        <For each={selectedEntity()!.risk_history}>
                                            {(p) => {
                                                const pointColor = p.score > 0.8 ? 'var(--status-offline)' : (p.score > 0.5 ? 'var(--status-degraded)' : 'var(--status-online)');
                                                return (
                                                    <div
                                                        style={`flex: 1; min-width: 3px; border-radius: 2px 2px 0 0; transition: all 120ms ease; opacity: 0.7; background: ${pointColor}; height: ${Math.max(p.score * 100, 4)}%;`}
                                                        title={`${new Date(p.timestamp as any).toLocaleTimeString()}: ${(p.score * 100).toFixed(1)}%`}
                                                        onMouseEnter={(e) => e.currentTarget.style.opacity = '1'}
                                                        onMouseLeave={(e) => e.currentTarget.style.opacity = '0.7'}
                                                    />
                                                );
                                            }}
                                        </For>
                                    </div>
                                </div>
                            </div>

                            {/* Behavioral Features vs Peers */}
                            <section style="display: flex; flex-direction: column; gap: 16px;">
                                <h3 style="font-size: 12px; font-weight: 800; text-transform: uppercase; color: var(--text-muted); letter-spacing: 1px; margin: 0;">Feature Vector Analysis (Peer Group: {selectedEntity()!.peer_group_id || 'N/A'})</h3>
                                <div style="display: grid; grid-template-columns: repeat(auto-fill, minmax(300px, 1fr)); gap: 16px;">
                                    <For each={Object.entries(selectedEntity()!.features)}>
                                        {([name, val]) => (
                                            <div class="ob-card" style="padding: 16px;">
                                                <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px;">
                                                    <span style="font-size: 10px; font-family: var(--font-mono); color: var(--text-muted); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; margin-right: 8px;">{name.replace(/_/g, ' ')}</span>
                                                    <span style="font-size: 12px; font-weight: 800; color: var(--text-primary);">{(val as number).toFixed(2)}</span>
                                                </div>
                                                <div style="height: 4px; background: rgba(0,0,0,0.5); border-radius: 2px; overflow: hidden;">
                                                    <div style={`height: 100%; background: var(--accent-primary); width: ${Math.min(val as number, 100)}%;`} />
                                                </div>
                                            </div>
                                        )}
                                    </For>
                                </div>
                            </section>

                            {/* Hunting Results for this Entity */}
                            <section style="display: flex; flex-direction: column; gap: 16px;">
                                <h3 style="font-size: 12px; font-weight: 800; text-transform: uppercase; color: var(--text-muted); letter-spacing: 1px; margin: 0;">Recent Activity Streams</h3>
                                <div class="ob-card" style="padding: 0; overflow: hidden;">
                                    <Show when={searchResults().length > 0} fallback={<div style="padding: 48px; text-align: center; color: var(--text-muted); font-style: italic; font-size: 12px;">Execute a hunt to populate activity streams</div>}>
                                        <div style="display: flex; flex-direction: column; divide-y: 1px solid var(--border-primary);">
                                            <For each={searchResults()}>
                                                {(res) => (
                                                    <div style="padding: 16px; display: flex; align-items: flex-start; gap: 16px; font-family: var(--font-mono); transition: all 120ms ease; cursor: pointer;" onMouseEnter={(e) => e.currentTarget.style.background = 'rgba(255,255,255,0.02)'} onMouseLeave={(e) => e.currentTarget.style.background = 'transparent'}>
                                                        <div style="color: var(--text-muted); font-size: 9px; width: 96px; flex-shrink: 0;">{new Date(res.timestamp as any).toLocaleTimeString()}</div>
                                                        <div style="flex: 1;">
                                                            <div style="font-size: 12px; color: var(--text-muted); margin-bottom: 4px;">{res.event_type} <span style="color: var(--text-muted); margin-left: 8px;">@{res.host_id}</span></div>
                                                            <div style="font-size: 10px; color: var(--text-muted); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; max-width: 600px;">{res.raw_log}</div>
                                                        </div>
                                                        <div style={`font-size: 9px; font-weight: 800; text-transform: uppercase; color: ${res.severity === 'high' ? 'var(--status-offline)' : 'var(--text-muted)'};`}>
                                                            {res.severity || 'low'}
                                                        </div>
                                                    </div>
                                                )}
                                            </For>
                                        </div>
                                    </Show>
                                </div>
                            </section>
                        </div>
                    </Show>
                </div>
            </main>

            {/* Overlays */}
            <Show when={showForensics()}>
                <ForensicView evidenceId={showForensics()!} onClose={() => setShowForensics(null)} />
            </Show>
        </div>
    );
};
