import { Component, createSignal, onMount, For, Show } from 'solid-js';
import { RecordingPlayer } from './RecordingPlayer';
import '../../styles/recordingplayer.css';

export const RecordingPanel: Component = () => {
    const [recordings, setRecordings] = createSignal<any[]>([]);
    const [searchResults, setSearchResults] = createSignal<any[] | null>(null);
    const [searchQuery, setSearchQuery] = createSignal('');
    const [loading, setLoading] = createSignal(true);
    const [searching, setSearching] = createSignal(false);
    const [activePlaybackId, setActivePlaybackId] = createSignal<string | null>(null);

    const reload = async () => {
        try {
            const { ListRecordings } = await import('../../../wailsjs/go/app/RecordingService');
            setRecordings(await ListRecordings() || []);
        } catch (e) { console.error('Recording load:', e); }
        setLoading(false);
    };

    onMount(reload);

    const handleSearch = async (e: Event) => {
        e.preventDefault();
        const query = searchQuery().trim();
        if (!query) {
            setSearchResults(null);
            return;
        }

        setSearching(true);
        try {
            const { SearchRecordings } = await import('../../../wailsjs/go/app/RecordingService');
            const results = await SearchRecordings(query);
            setSearchResults(results || []);
        } catch (e) {
            console.error('Search failed:', e);
        } finally {
            setSearching(false);
        }
    };

    const handleDelete = async (id: string) => {
        try {
            const { DeleteRecording } = await import('../../../wailsjs/go/app/RecordingService');
            await DeleteRecording(id);
            await reload();
        } catch (e) { console.error('Delete recording:', e); }
    };

    const handleExport = async (id: string, hostLabel: string) => {
        try {
            // @ts-ignore - Wails runtime
            const filename = await window.runtime.SaveFileDialog({
                Title: 'Export Audit Recording',
                DefaultFilename: `${hostLabel || 'session'}_${id.substring(0, 8)}.cast`,
                Filters: [{ Name: 'Asciinema Recording', Pattern: '*.cast' }]
            });

            if (filename) {
                const { ExportRecording } = await import('../../../wailsjs/go/app/RecordingService');
                await ExportRecording(id, filename);
                console.log('Exported to:', filename);
            }
        } catch (e) {
            console.error('Export failed:', e);
        }
    };

    const formatDate = (dateStr: string) => {
        try {
            return new Date(dateStr).toLocaleString();
        } catch { return dateStr; }
    };

    return (
        <div style="display: flex; flex-direction: column; height: 100%; position: relative;">
            <div class="drawer-header"><span class="drawer-title">Terminal Audit & Forensic Search</span></div>

            <div style="padding: 12px; border-bottom: 1px solid var(--border-primary);">
                <form onSubmit={handleSearch} style="display: flex; gap: 8px;">
                    <input
                        type="text"
                        placeholder="Search for commands or output (e.g. sudo, apt)..."
                        value={searchQuery()}
                        onInput={(e) => setSearchQuery(e.currentTarget.value)}
                        style="flex: 1; background: var(--bg-secondary); border: 1px solid var(--border-primary); color: var(--text-primary); border-radius: 4px; padding: 6px 10px; font-size: 12px;"
                    />
                    <button
                        type="submit"
                        disabled={searching()}
                        style="background: var(--accent-primary); border: none; color: white; border-radius: 4px; padding: 6px 12px; font-size: 12px; font-weight: 600; cursor: pointer;"
                    >
                        {searching() ? '...' : 'Search'}
                    </button>
                </form>
            </div>

            <div style="flex: 1; overflow-y: auto; padding: 8px;">
                <Show when={loading() && !searchResults()}><div class="placeholder">Loading inventory...</div></Show>

                <Show when={searchResults() !== null}>
                    <div style="margin-bottom: 16px;">
                        <div style="display: flex; justify-content: space-between; align-items: center; margin-bottom: 8px;">
                            <span style="font-size: 11px; font-weight: 700; color: var(--accent-primary); text-transform: uppercase;">Search Results</span>
                            <button onClick={() => { setSearchResults(null); setSearchQuery(''); }} style="background: none; border: none; color: var(--text-muted); cursor: pointer; font-size: 11px;">Clear Results</button>
                        </div>
                        <For each={searchResults()!}>
                            {(res) => (
                                <div style="background: rgba(var(--accent-primary-rgb), 0.05); border: 1px solid var(--accent-primary); border-radius: var(--radius-sm); padding: 10px; margin-bottom: 8px; cursor: pointer;" onClick={() => setActivePlaybackId(res.id)}>
                                    <div style="font-size: 12px; font-weight: 600; color: var(--text-primary);">🎬 {res.host_label || 'Local Session'}</div>
                                    <div
                                        style="font-size: 11px; color: var(--text-muted); margin-top: 6px; font-family: var(--font-mono); background: var(--bg-secondary); padding: 4px; border-radius: 3px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap;"
                                        textContent={res.highlight}
                                    />
                                    <div style="font-size: 9px; color: var(--text-muted); margin-top: 4px;">{formatDate(res.started_at)}</div>
                                </div>
                            )}
                        </For>
                        <Show when={searchResults()?.length === 0}>
                            <div class="placeholder">No matches found in audit history.</div>
                        </Show>
                        <hr style="border: none; border-top: 1px solid var(--border-primary); margin: 16px 0;" />
                    </div>
                </Show>

                <Show when={!loading() && (searchResults() === null)}>
                    <div style="font-size: 11px; font-weight: 700; color: var(--text-muted); text-transform: uppercase; margin-bottom: 12px; padding-left: 4px;">Session Inventory</div>
                    <For each={recordings()} fallback={<div class="placeholder">No recordings yet. Connect to a host to start auditing.</div>}>
                        {(rec) => (
                            <div style="background: var(--bg-tertiary); border: 1px solid var(--border-primary); border-radius: var(--radius-sm); padding: 10px; margin-bottom: 8px; transition: border-color 0.2s;" class="recording-item">
                                <div style="display: flex; justify-content: space-between; align-items: flex-start;">
                                    <div style="flex: 1; cursor: pointer;" onClick={() => setActivePlaybackId(rec.id)}>
                                        <div style="font-size: 13px; font-weight: 600; color: var(--text-primary);">🎬 {rec.host_label || 'Local Session'}</div>
                                        <div style="font-size: 11px; color: var(--text-muted); margin-top: 4px; display: flex; gap: 8px; align-items: center;">
                                            <span>⏱ {rec.duration?.toFixed(1)}s</span>
                                            <span>📊 {rec.event_count} events</span>
                                        </div>
                                        <div style="font-size: 10px; color: var(--text-muted); margin-top: 4px;">
                                            {formatDate(rec.started_at)}
                                        </div>
                                    </div>
                                    <div style="display: flex; gap: 8px;">
                                        <button
                                            onClick={() => setActivePlaybackId(rec.id)}
                                            style="background: var(--bg-secondary); border: 1px solid var(--border-primary); color: var(--accent-primary); border-radius: 4px; padding: 4px 8px; font-size: 11px; cursor: pointer;"
                                            title="Play Session"
                                        >
                                            Play
                                        </button>
                                        <button
                                            onClick={() => handleExport(rec.id, rec.host_label)}
                                            style="background: var(--bg-secondary); border: 1px solid var(--border-primary); color: var(--text-primary); border-radius: 4px; padding: 4px 8px; font-size: 11px; cursor: pointer;"
                                            title="Download Signed Audit Log"
                                        >
                                            Export
                                        </button>
                                        <button onClick={() => handleDelete(rec.id)} style="background: none; border: none; color: var(--error); cursor: pointer; font-size: 12px; padding: 4px;" title="Delete">🗑</button>
                                    </div>
                                </div>
                            </div>
                        )}
                    </For>
                </Show>
            </div>

            <Show when={activePlaybackId()}>
                <RecordingPlayer
                    recordingId={activePlaybackId()!}
                    onClose={() => setActivePlaybackId(null)}
                />
            </Show>
        </div>
    );
};
