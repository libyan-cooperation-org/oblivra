import { Component, createSignal, onMount, For, Show } from 'solid-js';
import { RecordingPlayer } from '../components/recordings/RecordingPlayer';

export const RecordingsPage: Component = () => {
    const [recordings, setRecordings] = createSignal<any[]>([]);
    const [searchQuery, setSearchQuery] = createSignal('');
    const [loading, setLoading] = createSignal(true);
    const [searching, setSearching] = createSignal(false);
    const [activePlaybackId, setActivePlaybackId] = createSignal<string | null>(null);

    const reload = async () => {
        try {
            const { ListRecordings } = await import('../../wailsjs/go/services/RecordingService');
            setRecordings(await ListRecordings() || []);
        } catch (e) { console.error('Recording load:', e); }
        setLoading(false);
    };

    onMount(reload);

    const performSearch = async (query: string) => {
        if (!query.trim()) {
            await reload();
            return;
        }
        setSearching(true);
        try {
            const { SearchRecordings } = await import('../../wailsjs/go/services/RecordingService');
            const results = await SearchRecordings(query.trim());
            setRecordings(results || []);
        } catch (e) { console.error('Search failed:', e); }
        setSearching(false);
    };

    let searchTimeout: any;
    const handleSearchInput = (val: string) => {
        setSearchQuery(val);
        clearTimeout(searchTimeout);
        searchTimeout = setTimeout(() => performSearch(val), 300);
    };

    const handleDelete = async (id: string, e: Event) => {
        e.stopPropagation();
        if (!confirm("Delete this immutable audit recording?")) return;
        try {
            const { DeleteRecording } = await import('../../wailsjs/go/services/RecordingService');
            await DeleteRecording(id);
            await reload();
        } catch (e) { console.error('Delete recording:', e); }
    };

    const handleExport = async (id: string, hostLabel: string, e: Event) => {
        e.stopPropagation();
        try {
            // @ts-ignore
            const filename = await window.runtime.SaveFileDialog({
                Title: 'Export Audit Recording',
                DefaultFilename: `${hostLabel || 'session'}_${id.substring(0, 8)}.cast`,
                Filters: [{ Name: 'Asciinema Recording', Pattern: '*.cast' }]
            });

            if (filename) {
                const { ExportRecording } = await import('../../wailsjs/go/services/RecordingService');
                await ExportRecording(id, filename);
            }
        } catch (err) { console.error('Export failed:', err); }
    };

    const formatDate = (dateStr: string) => {
        try {
            const d = new Date(dateStr);
            return d.toLocaleDateString() + ' ' + d.toLocaleTimeString([], {hour: '2-digit', minute:'2-digit'});
        } catch { return dateStr; }
    };
    
    // Formatting duration like 01:23
    const formatDuration = (seconds?: number) => {
        if (!seconds) return '00:00';
        const m = Math.floor(seconds / 60);
        const s = Math.floor(seconds % 60);
        return `${m.toString().padStart(2,'0')}:${s.toString().padStart(2,'0')}`;
    };

    return (
        <div class="flex flex-col h-full w-full bg-[#0B0D14] text-gray-200 overflow-hidden relative">
            <div class="p-6 border-b border-gray-800/60 flex justify-between items-center bg-[#11131A]/80 backdrop-blur-md z-10">
                <div>
                    <h1 class="text-2xl font-bold tracking-tight text-white font-mono flex items-center gap-3">
                        <svg class="w-6 h-6 text-purple-500" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 10l4.553-2.276A1 1 0 0121 8.618v6.764a1 1 0 01-1.447.894L15 14M5 18h8a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v8a2 2 0 002 2z"></path></svg>
                        Session Recordings
                    </h1>
                    <p class="text-sm text-gray-500 mt-1">Immutable audit logs of terminal sessions.</p>
                </div>
                <div class="relative group">
                    <div class="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                        <Show when={searching()} fallback={
                            <svg class="w-4 h-4 text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"></path></svg>
                        }>
                            <div class="w-4 h-4 border-2 border-purple-500 border-t-transparent rounded-full animate-spin"></div>
                        </Show>
                    </div>
                    <input 
                        type="text" 
                        placeholder="Search transcripts (e.g. sudo, root)..." 
                        value={searchQuery()}
                        onInput={(e) => handleSearchInput(e.currentTarget.value)}
                        class="w-72 bg-black/40 border border-gray-700/50 rounded-lg pl-10 pr-4 py-2 text-sm text-gray-200 focus:outline-none focus:border-purple-500/50 focus:ring-1 focus:ring-purple-500/50 transition-all shadow-inner"
                    />
                </div>
            </div>

            <div class="flex-1 overflow-y-auto p-6 custom-scrollbar">
                <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4 gap-6">
                    <Show when={!loading() && recordings().length === 0}>
                        <div class="col-span-full h-64 flex flex-col items-center justify-center text-gray-600">
                            <svg class="w-16 h-16 mb-4 text-gray-800" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M15 10l4.553-2.276A1 1 0 0121 8.618v6.764a1 1 0 01-1.447.894L15 14M5 18h8a2 2 0 002-2V8a2 2 0 00-2-2H5a2 2 0 00-2 2v8a2 2 0 002 2z"></path></svg>
                            <p class="text-lg font-medium text-gray-500">No recordings found</p>
                            <p class="text-sm mt-1">Connect to a host to start auditing.</p>
                        </div>
                    </Show>

                    <For each={recordings()}>
                        {(rec) => (
                            <div 
                                class="group relative bg-[#161923]/60 border border-gray-800/80 rounded-xl overflow-hidden flex flex-col transition-all duration-300 hover:border-purple-500/40 hover:shadow-lg hover:shadow-purple-900/10 hover:-translate-y-1 cursor-pointer"
                                onClick={() => setActivePlaybackId(rec.id)}
                            >
                                {/* Thumbnail Area */}
                                <div class="h-32 bg-[#090A0F] border-b border-gray-800/50 relative flex items-center justify-center overflow-hidden">
                                    {/* Abstract code lines representation for thumbnail */}
                                    <div class="absolute inset-0 opacity-20 p-4 font-mono text-[8px] leading-tight text-emerald-500 select-none overflow-hidden">
                                        <p>root@box:~# ./deploy.sh</p><p>Starting deployment...</p><p>checking dependencies: [OK]</p>
                                        <p>mounting volumes: [OK]</p><p>starting service... done.</p><p>root@box:~# _</p>
                                    </div>
                                    
                                    <div class="w-12 h-12 rounded-full bg-purple-600/20 flex items-center justify-center border border-purple-500/30 transform scale-90 group-hover:scale-110 group-hover:bg-purple-600 transition-all duration-300 group-hover:shadow-[0_0_20px_rgba(147,51,234,0.4)] z-10">
                                        <svg class="w-5 h-5 text-purple-100 ml-1" fill="currentColor" viewBox="0 0 20 20"><path d="M4 4l12 6-12 6z"></path></svg>
                                    </div>
                                    
                                    <div class="absolute bottom-2 right-2 bg-black/80 backdrop-blur text-xs font-mono px-2 py-0.5 rounded text-gray-300 border border-gray-800/80 shadow-sm z-10">
                                        {formatDuration(rec.duration)}
                                    </div>
                                </div>
                                
                                {/* Info Area */}
                                <div class="p-4 flex-1 flex flex-col bg-gradient-to-b from-white/[0.02] to-transparent">
                                    <h3 class="font-semibold text-gray-200 truncate mb-1" title={rec.host_label || 'Local Session'}>
                                        {rec.host_label || 'Local Session'}
                                    </h3>
                                    
                                    <div class="flex items-center text-xs text-gray-500 mb-3 gap-3">
                                        <span class="flex items-center gap-1">
                                            <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 7V3m8 4V3m-9 8h10M5 21h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"></path></svg>
                                            {formatDate(rec.started_at)}
                                        </span>
                                    </div>

                                    {/* Action Buttons */}
                                    <div class="flex justify-between items-center mt-auto pt-3 border-t border-gray-800/60 opacity-0 group-hover:opacity-100 transition-opacity">
                                        <span class="text-xs font-mono text-emerald-500/80 bg-emerald-500/10 px-2 py-0.5 rounded border border-emerald-500/20">
                                            {rec.event_count || 0} events
                                        </span>
                                        <div class="flex gap-1">
                                            <button 
                                                class="p-1.5 text-gray-400 hover:text-white bg-black/20 hover:bg-gray-700 rounded transition-colors" 
                                                title="Export verified audit log"
                                                onClick={(e) => handleExport(rec.id, rec.host_label, e)}
                                            >
                                                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M4 16v1a3 3 0 003 3h10a3 3 0 003-3v-1m-4-4l-4 4m0 0l-4-4m4 4V4"></path></svg>
                                            </button>
                                            <button 
                                                class="p-1.5 text-gray-400 hover:text-red-400 bg-black/20 hover:bg-red-500/10 rounded transition-colors" 
                                                title="Delete recording"
                                                onClick={(e) => handleDelete(rec.id, e)}
                                            >
                                                <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"></path></svg>
                                            </button>
                                        </div>
                                    </div>
                                    
                                    <Show when={rec.highlight}>
                                        <div class="mt-3 text-xs p-2 bg-purple-900/10 border border-purple-500/20 rounded-md font-mono text-purple-300 truncate" innerHTML={rec.highlight}></div>
                                    </Show>
                                </div>
                            </div>
                        )}
                    </For>
                </div>
            </div>

            {/* Video Player Modal */}
            <Show when={activePlaybackId()}>
                <div class="absolute inset-0 z-50 bg-black/80 backdrop-blur-sm flex items-center justify-center p-8 animate-fade-in">
                    <div class="w-full max-w-6xl h-[80vh] bg-[#0B0D14] border border-gray-700/80 rounded-xl shadow-[0_0_50px_rgba(0,0,0,0.5)] overflow-hidden flex flex-col relative">
                        <RecordingPlayer
                            recordingId={activePlaybackId()!}
                            onClose={() => setActivePlaybackId(null)}
                        />
                    </div>
                </div>
            </Show>

            <style>{`
                .custom-scrollbar::-webkit-scrollbar { width: 6px; height: 6px; }
                .custom-scrollbar::-webkit-scrollbar-track { background: transparent; }
                .custom-scrollbar::-webkit-scrollbar-thumb { background-color: rgba(75, 85, 99, 0.4); border-radius: 20px; }
                .custom-scrollbar::-webkit-scrollbar-thumb:hover { background-color: rgba(107, 114, 128, 0.6); }
                @keyframes fadeIn {
                    from { opacity: 0; backdrop-filter: blur(0px); }
                    to { opacity: 1; backdrop-filter: blur(4px); }
                }
                .animate-fade-in { animation: fadeIn 0.2s ease-out forwards; }
            `}</style>
        </div>
    );
};
