import { Component, createSignal, onMount, For, Show } from 'solid-js';
import { LineageRecord } from '../wails';

export const LineageExplorer: Component = () => {
    const [records, setRecords] = createSignal<LineageRecord[]>([]);
    const [stats, setStats] = createSignal<any>(null);
    const [loading, setLoading] = createSignal(true);

    const loadData = async () => {
        try {
            if (window.go?.app?.LineageService) {
                const [recs, st] = await Promise.all([
                    window.go.app.LineageService.GetRecentLineage(100),
                    window.go.app.LineageService.GetStats()
                ]);
                setRecords(recs || []);
                setStats(st);
            }
        } catch (err) {
            console.error("Failed to load lineage data:", err);
        } finally {
            setLoading(false);
        }
    };

    onMount(() => {
        loadData();
        const timer = setInterval(loadData, 2000);
        return () => clearInterval(timer);
    });

    const getStageColor = (stage: string) => {
        switch (stage) {
            case 'ingested': return '#3b82f6'; // blue
            case 'parsed': return '#8b5cf6'; // purple
            case 'enriched': return '#10b981'; // green
            case 'detected': return '#f59e0b'; // yellow
            case 'alerted': return '#ef4444'; // red
            case 'responded': return '#ec4899'; // pink
            default: return '#6b7280'; // gray
        }
    };

    return (
        <div class="lineage-page">
            <header class="page-header">
                <div>
                    <h1>DATA LINEAGE</h1>
                    <p>Cryptographic provenance tracking across pipeline stages</p>
                </div>
                <div class="stats-badge">
                    <Show when={stats()}>
                        <span>{stats()?.total_records || 0} Records Mapped</span>
                    </Show>
                </div>
            </header>

            <div class="lineage-panel">
                <div class="table-container">
                    <table class="lineage-table">
                        <thead>
                            <tr>
                                <th>TIMESTAMP</th>
                                <th>ENTITY ID</th>
                                <th>STAGE</th>
                                <th>PROVENANCE HASH</th>
                            </tr>
                        </thead>
                        <tbody>
                            <Show when={loading() && records().length === 0}>
                                <tr><td colspan="4" class="empty-state">Loading provenance graph...</td></tr>
                            </Show>
                            <Show when={!loading() && records().length === 0}>
                                <tr><td colspan="4" class="empty-state">No pipeline data recorded yet.</td></tr>
                            </Show>
                            <For each={records()}>
                                {(r) => (
                                    <tr>
                                        <td class="mono">{new Date(r.timestamp).toISOString()}</td>
                                        <td class="mono entity">{r.entity_id}</td>
                                        <td>
                                            <span class="stage-tag" style={{
                                                'background-color': `${getStageColor(r.stage)}20`,
                                                'color': getStageColor(r.stage),
                                                'border-color': `${getStageColor(r.stage)}50`
                                            }}>
                                                {r.stage.toUpperCase()}
                                            </span>
                                        </td>
                                        <td class="mono hash">{r.proof_hash.substring(0, 24)}...</td>
                                    </tr>
                                )}
                            </For>
                        </tbody>
                    </table>
                </div>
            </div>

            <style>{`
                .lineage-page { padding: 1.5rem; height: calc(100vh - 60px); display: flex; flex-direction: column; gap: 1.5rem; background: var(--tactical-bg); }
                .page-header { display: flex; justify-content: space-between; align-items: center; }
                .page-header h1 { font-size: 1.4rem; letter-spacing: 2px; margin: 0; color: #f3f4f6; }
                .page-header p { color: var(--tactical-gray); font-size: 0.8rem; margin: 0.25rem 0 0; }
                
                .stats-badge span { background: var(--tactical-surface); border: 1px solid var(--tactical-border); padding: 6px 12px; border-radius: 4px; font-size: 0.75rem; color: #10b981; font-weight: 600; font-family: monospace; }

                .lineage-panel { background: var(--tactical-surface); border: 1px solid var(--tactical-border); border-radius: 8px; flex: 1; min-height: 0; display: flex; flex-direction: column; overflow: hidden; }
                
                .table-container { flex: 1; overflow: auto; }
                .lineage-table { width: 100%; border-collapse: collapse; text-align: left; }
                .lineage-table th { position: sticky; top: 0; background: #111827; padding: 1rem; font-size: 0.65rem; letter-spacing: 2px; color: var(--tactical-gray); border-bottom: 2px solid var(--tactical-border); }
                .lineage-table td { padding: 0.85rem 1rem; border-bottom: 1px solid var(--tactical-border); font-size: 0.8rem; color: #d1d5db; }
                .lineage-table tr:hover td { background: rgba(255,255,255,0.02); }
                
                .mono { font-family: monospace; font-size: 0.75rem; }
                .entity { color: #60a5fa; }
                .hash { color: var(--tactical-gray); }
                
                .stage-tag { padding: 4px 8px; border-radius: 4px; font-size: 0.65rem; font-weight: 700; letter-spacing: 1px; border: 1px solid transparent; }
                
                .empty-state { text-align: center; padding: 3rem !important; color: var(--tactical-gray) !important; font-style: italic; }
            `}</style>
        </div>
    );
};
