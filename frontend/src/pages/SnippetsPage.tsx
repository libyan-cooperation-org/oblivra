import { Component, createSignal, createEffect, For, Show } from 'solid-js';
import { Create, List, Delete, ExtractVariables } from '../../wailsjs/go/services/SnippetService';

export const SnippetsPage: Component = () => {
    const [snippets, setSnippets] = createSignal<any[]>([]);
    const [searchQuery, setSearchQuery] = createSignal('');
    const [isCreating, setIsCreating] = createSignal(false);
    
    // Form fields
    const [title, setTitle] = createSignal('');
    const [command, setCommand] = createSignal('');
    const [description, setDescription] = createSignal('');

    const loadSnippets = async () => {
        try {
            const result = await List();
            setSnippets(result || []);
        } catch (err) {
            console.error("Failed to load snippets", err);
        }
    };

    createEffect(() => loadSnippets());

    const handleSave = async () => {
        if (!title() || !command()) return;
        try {
            const extractedVars = await ExtractVariables(command());
            await Create(title(), command(), description(), [], extractedVars || []);
            setIsCreating(false);
            setTitle('');
            setCommand('');
            setDescription('');
            loadSnippets();
        } catch (err) {
            console.error("Snippet save failed", err);
        }
    };

    const handleDelete = async (id: string, e: Event) => {
        e.stopPropagation();
        if (!confirm("Delete this snippet?")) return;
        try {
            await Delete(id);
            loadSnippets();
        } catch (err) { }
    };

    const copyToClipboard = (text: string, e: Event) => {
        e.stopPropagation();
        navigator.clipboard.writeText(text);
        // Simple visual feedback could go here
    };

    const filteredSnippets = () => snippets().filter(s => 
        (s.title?.toLowerCase().includes(searchQuery().toLowerCase())) || 
        (s.description?.toLowerCase().includes(searchQuery().toLowerCase())) ||
        (s.command?.toLowerCase().includes(searchQuery().toLowerCase()))
    );

    return (
        <div class="flex flex-col h-full w-full bg-[#0B0D14] text-gray-200 overflow-hidden">
            <div class="p-6 border-b border-gray-800/60 flex justify-between items-center bg-[#11131A]/80 backdrop-blur-md z-10">
                <div>
                    <h1 class="text-2xl font-bold tracking-tight text-white font-mono flex items-center gap-3">
                        <svg class="w-6 h-6 text-blue-500" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M10 20l4-16m4 4l4 4-4 4M6 16l-4-4 4-4"></path></svg>
                        Command Snippets
                    </h1>
                    <p class="text-sm text-gray-500 mt-1">Reusable playbooks, commands, and scripts.</p>
                </div>
                <div class="flex items-center gap-4">
                    <div class="relative group">
                        <div class="absolute inset-y-0 left-0 pl-3 flex items-center pointer-events-none">
                            <svg class="w-4 h-4 text-gray-500" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M21 21l-6-6m2-5a7 7 0 11-14 0 7 7 0 0114 0z"></path></svg>
                        </div>
                        <input 
                            type="text" 
                            placeholder="Find snippet..." 
                            value={searchQuery()}
                            onInput={(e) => setSearchQuery(e.currentTarget.value)}
                            class="w-64 bg-black/40 border border-gray-700/50 rounded-lg pl-10 pr-4 py-2 text-sm text-gray-200 focus:outline-none focus:border-blue-500/50 focus:ring-1 focus:ring-blue-500/50 transition-all shadow-inner"
                        />
                    </div>
                    <button 
                        onClick={() => setIsCreating(!isCreating())}
                        class="px-4 py-2 bg-blue-600 hover:bg-blue-500 text-white rounded-lg border border-blue-500 text-sm font-medium transition-colors shadow-[0_0_10px_rgba(37,99,235,0.2)] flex items-center gap-2"
                    >
                        {isCreating() ? 'Cancel' : 
                            <><svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 4v16m8-8H4"></path></svg> New Snippet</>
                        }
                    </button>
                </div>
            </div>

            <div class="flex-1 overflow-y-auto p-6 custom-scrollbar">
                <Show when={isCreating()}>
                    <div class="mb-8 p-6 bg-[#161923]/80 border border-blue-500/30 rounded-xl shadow-[0_0_20px_rgba(37,99,235,0.05)] backdrop-blur-sm animate-fade-in-down">
                        <h2 class="text-lg font-semibold text-white mb-4">Create New Snippet</h2>
                        <div class="grid grid-cols-2 gap-6 mb-4">
                            <div>
                                <label class="block text-xs font-medium text-gray-400 mb-1">Title</label>
                                <input type="text" value={title()} onInput={(e) => setTitle(e.currentTarget.value)} class="w-full bg-black/40 border border-gray-700 rounded-lg px-3 py-2 text-sm text-gray-200 focus:outline-none focus:border-blue-500/50" placeholder="e.g. Flush DNS Cache" />
                            </div>
                            <div>
                                <label class="block text-xs font-medium text-gray-400 mb-1">Description</label>
                                <input type="text" value={description()} onInput={(e) => setDescription(e.currentTarget.value)} class="w-full bg-black/40 border border-gray-700 rounded-lg px-3 py-2 text-sm text-gray-200 focus:outline-none focus:border-blue-500/50" placeholder="Brief explanation of what this does" />
                            </div>
                        </div>
                        <div class="mb-4">
                            <div class="flex justify-between items-end mb-1">
                                <label class="block text-xs font-medium text-gray-400">Command / Playbook</label>
                                <span class="text-[10px] text-blue-400">Supports {'{{variables}}'}</span>
                            </div>
                            <textarea value={command()} onInput={(e) => setCommand(e.currentTarget.value)} class="w-full h-32 bg-[#090A0F] border border-gray-700 rounded-lg p-3 text-sm text-emerald-400 font-mono focus:outline-none focus:border-blue-500/50 resize-y" placeholder="sudo systemd-resolve --flush-caches"></textarea>
                        </div>
                        <div class="flex justify-end">
                            <button class="px-6 py-2 bg-blue-600 hover:bg-blue-500 text-white rounded-lg text-sm font-medium transition-colors" onClick={handleSave}>Save Snippet</button>
                        </div>
                    </div>
                </Show>

                <div class="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-6">
                    <Show when={snippets().length === 0 && !isCreating()}>
                        <div class="col-span-full h-64 flex flex-col items-center justify-center text-gray-600">
                            <svg class="w-12 h-12 mb-4 text-gray-700" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1" d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z"></path></svg>
                            <p>No snippets found.</p>
                        </div>
                    </Show>

                    <For each={filteredSnippets()}>
                        {(snippet) => (
                            <div class="group bg-[#161923]/60 border border-gray-800/80 hover:border-gray-600 rounded-xl overflow-hidden flex flex-col transition-all duration-300 hover:shadow-lg hover:shadow-black/20 hover:-translate-y-1">
                                <div class="p-4 border-b border-gray-800/50 flex justify-between items-start bg-gradient-to-b from-white/[0.02] to-transparent">
                                    <div>
                                        <h3 class="font-semibold text-gray-200 group-hover:text-blue-100 transition-colors">{snippet.title}</h3>
                                        <p class="text-xs text-gray-500 mt-1 line-clamp-1">{snippet.description || 'No description'}</p>
                                    </div>
                                    <div class="flex space-x-1 opacity-0 group-hover:opacity-100 transition-opacity">
                                        <button class="p-1.5 text-gray-500 hover:text-blue-400 bg-black/20 hover:bg-blue-500/10 rounded" title="Copy to clipboard" onClick={(e) => copyToClipboard(snippet.command, e)}>
                                            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z"></path></svg>
                                        </button>
                                        <button class="p-1.5 text-gray-500 hover:text-red-400 bg-black/20 hover:bg-red-500/10 rounded" title="Delete snippet" onClick={(e) => handleDelete(snippet.id, e)}>
                                            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16"></path></svg>
                                        </button>
                                    </div>
                                </div>
                                <div class="p-4 flex-1 bg-[#0d1117] overflow-x-auto relative">
                                    <pre class="text-xs font-mono leading-relaxed text-emerald-400/90 whitespace-pre-wrap word-break"><code>{snippet.command}</code></pre>
                                </div>
                                <div class="px-4 py-3 bg-[#11131A] border-t border-gray-800/80 flex justify-between items-center text-xs">
                                    <span class="text-gray-500 flex items-center gap-1">
                                        <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 7h.01M7 3h5c.512 0 1.024.195 1.414.586l7 7a2 2 0 010 2.828l-7 7a2 2 0 01-2.828 0l-7-7A1.994 1.994 0 013 12V7a4 4 0 014-4z"></path></svg>
                                        {(snippet.variables?.length || 0)} variables
                                    </span>
                                    <span class="text-xs font-medium px-2 py-0.5 rounded bg-gray-800 text-gray-400">Playbook</span>
                                </div>
                            </div>
                        )}
                    </For>
                </div>
            </div>

            <style>{`
                .custom-scrollbar::-webkit-scrollbar { width: 6px; height: 6px; }
                .custom-scrollbar::-webkit-scrollbar-track { background: transparent; }
                .custom-scrollbar::-webkit-scrollbar-thumb { background-color: rgba(75, 85, 99, 0.4); border-radius: 20px; }
                .custom-scrollbar::-webkit-scrollbar-thumb:hover { background-color: rgba(107, 114, 128, 0.6); }
                @keyframes fadeInDown {
                    from { opacity: 0; transform: translateY(-10px); }
                    to { opacity: 1; transform: translateY(0); }
                }
                .animate-fade-in-down { animation: fadeInDown 0.3s ease-out forwards; }
            `}</style>
        </div>
    );
};
