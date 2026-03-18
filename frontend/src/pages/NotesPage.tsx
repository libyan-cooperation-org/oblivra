import { Component, createSignal, onMount, For, Show } from 'solid-js';
import { useApp } from '@core/store';

// Use any for rapid prototyping until bindings align perfectly
type Note = any;

export const NotesPage: Component = () => {
    const [state] = useApp();
    const [notes, setNotes] = createSignal<Note[]>([]);
    const [activeNote, setActiveNote] = createSignal<Note | null>(null);
    const [searchQuery, setSearchQuery] = createSignal('');
    const [loading, setLoading] = createSignal(true);

    const reload = async () => {
        try {
            const { GetNotes } = await import('../../wailsjs/go/services/NotesService');
            const n = await GetNotes(''); // Get all notes
            setNotes(n || []);
        } catch (e) { console.error('Notes load:', e); }
        setLoading(false);
    };

    onMount(reload);

    const filteredNotes = () => notes().filter(n => 
        (n.title?.toLowerCase().includes(searchQuery().toLowerCase())) || 
        (n.content?.toLowerCase().includes(searchQuery().toLowerCase()))
    );

    const handleSave = async () => {
        const note = activeNote();
        if (!note) return;
        try {
            const { SaveNote } = await import('../../wailsjs/go/services/NotesService');
            await SaveNote(note);
            await reload();
            // keep it active
        } catch (e) { console.error('Save note:', e); }
    };

    const handleDelete = async (id: string, e: Event) => {
        e.stopPropagation();
        try {
            const { DeleteNote } = await import('../../wailsjs/go/services/NotesService');
            await DeleteNote(id);
            if (activeNote()?.id === id) setActiveNote(null);
            await reload();
        } catch (err) { console.error('Delete note:', err); }
    };

    const createNewNote = () => {
        // We cast to any here to satisfy the Wails generated struct signature
        const newNote: any = {
            id: '',
            title: 'New Note',
            content: '',
            session_id: state.activeSessionId || '',
            tags: [],
            category: '',
            created_at: '',
            updated_at: '',
            is_pinned: false
        };
        setActiveNote(newNote);
    };

    return (
        <div class="flex h-full w-full bg-[#0B0D14] text-gray-200 overflow-hidden">
            {/* Sidebar */}
            <div class="w-1/3 max-w-sm flex flex-col border-r border-gray-800/60 bg-[#11131A]/80 backdrop-blur-md">
                <div class="p-5 border-b border-gray-800/60">
                    <div class="flex justify-between items-center mb-4">
                        <h1 class="text-xl font-bold tracking-tight text-white font-mono">Operations Log</h1>
                        <button 
                            onClick={createNewNote}
                            class="bg-blue-600/20 hover:bg-blue-600/40 text-blue-400 p-1.5 rounded-lg border border-blue-500/30 transition-all duration-200"
                        >
                            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"></path></svg>
                        </button>
                    </div>
                    <div class="relative group">
                        <div class="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                            <svg class="w-4 h-4 text-gray-500 group-focus-within:text-blue-400 transition-colors" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"></path></svg>
                        </div>
                        <input 
                            type="text" 
                            placeholder="Search notes..." 
                            value={searchQuery()}
                            onInput={(e) => setSearchQuery(e.currentTarget.value)}
                            class="w-full bg-black/40 border border-gray-700/50 rounded-lg pl-10 pr-4 py-2 text-sm text-gray-200 placeholder-gray-600 focus:outline-none focus:border-blue-500/50 focus:ring-1 focus:ring-blue-500/50 transition-all shadow-inner"
                        />
                    </div>
                </div>

                <div class="flex-1 overflow-y-auto p-3 space-y-2 custom-scrollbar">
                    <Show when={loading()}>
                        <div class="flex flex-col animate-pulse space-y-2">
                            <div class="h-20 bg-gray-800/40 rounded-lg"></div>
                            <div class="h-20 bg-gray-800/40 rounded-lg"></div>
                            <div class="h-20 bg-gray-800/40 rounded-lg"></div>
                        </div>
                    </Show>
                    <Show when={!loading() && filteredNotes().length === 0}>
                        <div class="text-center text-gray-500 text-sm mt-10">No notes found.</div>
                    </Show>
                    <For each={filteredNotes()}>
                        {(n) => (
                            <div 
                                onClick={() => setActiveNote({...n})}
                                class={`p-3 rounded-lg cursor-pointer transition-all duration-200 border group
                                ${activeNote()?.id === n.id 
                                    ? 'bg-blue-900/10 border-blue-500/40 shadow-[0_0_15px_rgba(59,130,246,0.1)]' 
                                    : 'bg-[#161923]/60 border-gray-800 hover:border-gray-600 hover:bg-[#1C202B]'}`}
                            >
                                <div class="flex justify-between items-start mb-1">
                                    <h3 class={`text-sm font-semibold truncate pr-2 ${activeNote()?.id === n.id ? 'text-blue-100' : 'text-gray-200'}`}>
                                        {n.title || 'Untitled'}
                                    </h3>
                                    <button 
                                        onClick={(e) => handleDelete(n.id, e)}
                                        class="text-gray-600 rounded hover:text-red-400 opacity-0 group-hover:opacity-100 transition-opacity"
                                    >
                                        <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12"></path></svg>
                                    </button>
                                </div>
                                <p class="text-xs text-gray-500 line-clamp-2 leading-relaxed">
                                    {n.content || 'Empty note...'}
                                </p>
                            </div>
                        )}
                    </For>
                </div>
            </div>

            {/* Main Editor */}
            <div class="flex-1 flex flex-col bg-[#0B0D14] relative">
                <Show when={activeNote()} fallback={
                    <div class="h-full flex flex-col items-center justify-center text-gray-600 space-y-4">
                        <div class="p-4 rounded-full bg-gray-900/50 border border-gray-800">
                            <svg class="w-8 h-8 text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z"></path></svg>
                        </div>
                        <p class="text-sm font-medium">Select a note or create a new one</p>
                    </div>
                }>
                    <div class="p-6 border-b border-gray-800/60 flex justify-between items-center bg-[#11131A]/40 backdrop-blur-sm z-10 sticky top-0">
                        <input 
                            type="text"
                            value={activeNote()?.title || ''}
                            onInput={(e) => setActiveNote(prev => prev ? {...prev, title: e.currentTarget.value} : null)}
                            placeholder="Note Title"
                            class="bg-transparent text-2xl font-bold tracking-tight text-white border-none outline-none focus:ring-0 w-3/4 placeholder-gray-700 font-mono"
                        />
                        <div class="flex space-x-3">
                            <button class="px-4 py-1.5 bg-[#1C202B] hover:bg-[#252A38] text-gray-300 rounded border border-gray-700/80 text-sm font-medium transition-colors shadow-sm" onClick={() => reload()}>Revert</button>
                            <button class="px-4 py-1.5 bg-blue-600 hover:bg-blue-500 text-white rounded border border-blue-500 text-sm font-medium transition-colors shadow-[0_0_10px_rgba(37,99,235,0.2)] hover:shadow-[0_0_15px_rgba(37,99,235,0.4)]" onClick={handleSave}>Save</button>
                        </div>
                    </div>
                    
                    <div class="flex-1 p-6 overflow-y-auto">
                        <textarea 
                            value={activeNote()?.content || ''}
                            onInput={(e) => setActiveNote(prev => prev ? {...prev, content: e.currentTarget.value} : null)}
                            placeholder="Start typing your operational logs, evidence, or thoughts here... Supports markdown format."
                            class="w-full h-full min-h-[500px] bg-transparent text-gray-300 resize-none border-none outline-none focus:ring-0 leading-relaxed font-mono text-sm placeholder-gray-700"
                        ></textarea>
                    </div>
                </Show>
            </div>
            
            <style>{`
                .custom-scrollbar::-webkit-scrollbar {
                    width: 6px;
                }
                .custom-scrollbar::-webkit-scrollbar-track {
                    background: transparent;
                }
                .custom-scrollbar::-webkit-scrollbar-thumb {
                    background-color: rgba(75, 85, 99, 0.4);
                    border-radius: 20px;
                }
                .custom-scrollbar::-webkit-scrollbar-thumb:hover {
                    background-color: rgba(107, 114, 128, 0.6);
                }
            `}</style>
        </div>
    );
};

