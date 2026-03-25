import { Component, For, Show, createSignal, createResource, onMount } from 'solid-js';
import { IS_BROWSER } from '@core/context';

interface Bookmark {
    id: string;
    label: string;
    hostname: string;
    port: number;
    username: string;
    auth_method: string;
    tags: string[];
    category: string;
    color: string;
    notes: string;
    is_favorite: boolean;
    has_password: boolean;
    last_connected_at: string | null;
    connection_count: number;
}

interface SSHBookmarksProps {
    onConnect: (hostId: string) => void;
    onClose: () => void;
}

const COLORS = ['#e04040', '#f58b00', '#f5c518', '#5cc05c', '#0099e0', '#b87fff', '#00c8d4', '#ff69b4'];

export const SSHBookmarks: Component<SSHBookmarksProps> = (props) => {
    const [searchQuery, setSearchQuery] = createSignal('');
    const [activeTag, setActiveTag] = createSignal<string | null>(null);
    const [showFavoritesOnly, setShowFavoritesOnly] = createSignal(false);
    const [editingId, setEditingId] = createSignal<string | null>(null);
    const [showAddForm, setShowAddForm] = createSignal(false);

    // Form state
    const [formLabel, setFormLabel] = createSignal('');
    const [formHostname, setFormHostname] = createSignal('');
    const [formPort, setFormPort] = createSignal(22);
    const [formUsername, setFormUsername] = createSignal('root');
    const [formPassword, setFormPassword] = createSignal('');
    const [formTags, setFormTags] = createSignal('');
    const [formColor, setFormColor] = createSignal('#0099e0');

    // Data fetching
    const fetchBookmarks = async () => {
        if (IS_BROWSER) return [];
        try {
            const { ListAll } = await import('../../../wailsjs/go/services/BookmarkService');
            return await ListAll() as Bookmark[];
        } catch { return []; }
    };

    const fetchTags = async () => {
        if (IS_BROWSER) return [];
        try {
            const { GetAllTags } = await import('../../../wailsjs/go/services/BookmarkService');
            return await GetAllTags() as string[];
        } catch { return []; }
    };

    const [bookmarks, { refetch }] = createResource(fetchBookmarks);
    const [tags] = createResource(fetchTags);

    // Filtered bookmarks
    const filtered = () => {
        let items = bookmarks() || [];
        if (showFavoritesOnly()) items = items.filter(b => b.is_favorite);
        if (activeTag()) items = items.filter(b => b.tags?.includes(activeTag()!));
        const q = searchQuery().toLowerCase();
        if (q) items = items.filter(b =>
            b.label.toLowerCase().includes(q) ||
            b.hostname.toLowerCase().includes(q) ||
            b.username.toLowerCase().includes(q) ||
            b.tags?.some(t => t.toLowerCase().includes(q))
        );
        return items;
    };

    const handleConnect = async (hostId: string) => {
        props.onConnect(hostId);
    };

    const handleAdd = async () => {
        if (IS_BROWSER || !formHostname()) return;
        try {
            const { Create } = await import('../../../wailsjs/go/services/BookmarkService');
            await Create({
                label: formLabel() || formHostname(),
                hostname: formHostname(),
                port: formPort(),
                username: formUsername(),
                password: formPassword(),
                auth_method: formPassword() ? 'password' : 'key',
                tags: formTags().split(',').map(t => t.trim()).filter(Boolean),
                category: '',
                color: formColor(),
                notes: '',
                jump_host_id: '',
            });
            resetForm();
            setShowAddForm(false);
            refetch();
        } catch (e) { console.error('Add bookmark failed:', e); }
    };

    const handleDelete = async (id: string) => {
        if (IS_BROWSER) return;
        try {
            const { Delete } = await import('../../../wailsjs/go/services/BookmarkService');
            await Delete(id);
            refetch();
        } catch (e) { console.error('Delete failed:', e); }
    };

    const handleToggleFavorite = async (id: string) => {
        if (IS_BROWSER) return;
        try {
            const { ToggleFavorite } = await import('../../../wailsjs/go/services/BookmarkService');
            await ToggleFavorite(id);
            refetch();
        } catch {}
    };

    const resetForm = () => {
        setFormLabel(''); setFormHostname(''); setFormPort(22);
        setFormUsername('root'); setFormPassword(''); setFormTags(''); setFormColor('#0099e0');
    };

    const panelStyle = {
        display: 'flex',
        'flex-direction': 'column' as const,
        width: '320px',
        height: '100%',
        background: 'var(--surface-1)',
        'border-right': '1px solid var(--border-primary)',
        'font-family': 'var(--font-ui)',
        overflow: 'hidden',
    };

    const inputStyle = {
        background: 'var(--surface-3)',
        border: '1px solid var(--border-primary)',
        color: 'var(--text-primary)',
        padding: '6px 10px',
        'border-radius': '3px',
        'font-size': '12px',
        'font-family': 'var(--font-mono)',
        width: '100%',
        'box-sizing': 'border-box' as const,
    };

    return (
        <div style={panelStyle}>
            {/* Header */}
            <div style={{
                display: 'flex', 'align-items': 'center', 'justify-content': 'space-between',
                padding: '10px 14px', 'border-bottom': '1px solid var(--border-primary)',
                'flex-shrink': '0',
            }}>
                <span style={{ 'font-size': '12px', 'font-weight': '700', 'text-transform': 'uppercase',
                    'letter-spacing': '1.5px', color: '#0099e0' }}>
                    SSH Bookmarks
                </span>
                <div style={{ display: 'flex', gap: '6px' }}>
                    <button onClick={() => setShowAddForm(!showAddForm())}
                        style={{ background: 'none', border: 'none', color: '#5cc05c', cursor: 'pointer',
                            'font-size': '18px', padding: '0 4px' }}
                        title="Add bookmark">+</button>
                    <button onClick={props.onClose}
                        style={{ background: 'none', border: 'none', color: 'var(--text-muted)',
                            cursor: 'pointer', 'font-size': '14px', padding: '0 4px' }}
                        title="Close panel">×</button>
                </div>
            </div>

            {/* Search */}
            <div style={{ padding: '8px 14px', 'flex-shrink': '0' }}>
                <input type="text" placeholder="🔍  Search hosts..." value={searchQuery()}
                    onInput={(e) => setSearchQuery(e.currentTarget.value)}
                    style={inputStyle} />
            </div>

            {/* Filters */}
            <div style={{
                display: 'flex', gap: '6px', padding: '0 14px 8px',
                'flex-wrap': 'wrap', 'flex-shrink': '0',
            }}>
                <button onClick={() => setShowFavoritesOnly(!showFavoritesOnly())}
                    style={{
                        background: showFavoritesOnly() ? 'rgba(245,197,24,0.15)' : 'var(--surface-3)',
                        border: `1px solid ${showFavoritesOnly() ? 'rgba(245,197,24,0.4)' : 'var(--border-primary)'}`,
                        color: showFavoritesOnly() ? '#f5c518' : 'var(--text-muted)',
                        padding: '2px 8px', 'border-radius': '10px', 'font-size': '10px',
                        cursor: 'pointer',
                    }}>★ Favorites</button>
                <For each={tags() || []}>
                    {(tag) => (
                        <button onClick={() => setActiveTag(activeTag() === tag ? null : tag)}
                            style={{
                                background: activeTag() === tag ? 'rgba(0,153,224,0.15)' : 'var(--surface-3)',
                                border: `1px solid ${activeTag() === tag ? 'rgba(0,153,224,0.4)' : 'var(--border-primary)'}`,
                                color: activeTag() === tag ? '#0099e0' : 'var(--text-muted)',
                                padding: '2px 8px', 'border-radius': '10px', 'font-size': '10px',
                                cursor: 'pointer',
                            }}>{tag}</button>
                    )}
                </For>
            </div>

            {/* Add Form */}
            <Show when={showAddForm()}>
                <div style={{
                    padding: '10px 14px', 'border-bottom': '1px solid var(--border-primary)',
                    display: 'flex', 'flex-direction': 'column', gap: '6px', 'flex-shrink': '0',
                    background: 'var(--surface-2)',
                }}>
                    <input placeholder="Label (optional)" value={formLabel()} onInput={e => setFormLabel(e.currentTarget.value)} style={inputStyle} />
                    <input placeholder="hostname or IP *" value={formHostname()} onInput={e => setFormHostname(e.currentTarget.value)} style={inputStyle} />
                    <div style={{ display: 'flex', gap: '6px' }}>
                        <input placeholder="user" value={formUsername()} onInput={e => setFormUsername(e.currentTarget.value)}
                            style={{ ...inputStyle, flex: '1' }} />
                        <input placeholder="port" type="number" value={formPort()} onInput={e => setFormPort(+e.currentTarget.value)}
                            style={{ ...inputStyle, width: '60px' }} />
                    </div>
                    <input placeholder="password (encrypted in Vault)" type="password" value={formPassword()}
                        onInput={e => setFormPassword(e.currentTarget.value)} style={inputStyle} />
                    <input placeholder="tags (comma separated)" value={formTags()}
                        onInput={e => setFormTags(e.currentTarget.value)} style={inputStyle} />
                    <div style={{ display: 'flex', gap: '4px', 'align-items': 'center' }}>
                        <span style={{ 'font-size': '10px', color: 'var(--text-muted)' }}>Color:</span>
                        <For each={COLORS}>
                            {(c) => (
                                <div onClick={() => setFormColor(c)} style={{
                                    width: '16px', height: '16px', 'border-radius': '50%', background: c,
                                    cursor: 'pointer', border: formColor() === c ? '2px solid white' : '2px solid transparent',
                                }} />
                            )}
                        </For>
                    </div>
                    <button onClick={handleAdd} style={{
                        background: '#5cc05c', border: 'none', color: '#0d0e10', padding: '7px',
                        'border-radius': '3px', 'font-weight': '700', 'font-size': '11px',
                        cursor: 'pointer', 'text-transform': 'uppercase', 'letter-spacing': '0.5px',
                    }}>Save Bookmark</button>
                </div>
            </Show>

            {/* Bookmark List */}
            <div style={{ flex: '1', 'overflow-y': 'auto', padding: '4px 0' }}>
                <For each={filtered()} fallback={
                    <div style={{
                        padding: '30px 14px', 'text-align': 'center',
                        color: 'var(--text-muted)', 'font-size': '11px',
                    }}>
                        {bookmarks()?.length === 0
                            ? 'No bookmarks yet. Click + to add one.'
                            : 'No matches found.'}
                    </div>
                }>
                    {(bm) => (
                        <div style={{
                            display: 'flex', 'align-items': 'center', gap: '8px',
                            padding: '8px 14px', cursor: 'pointer',
                            'border-bottom': '1px solid rgba(255,255,255,0.04)',
                            transition: 'background 0.1s',
                        }}
                            onMouseEnter={e => e.currentTarget.style.background = 'var(--surface-3)'}
                            onMouseLeave={e => e.currentTarget.style.background = 'transparent'}
                            onDblClick={() => handleConnect(bm.id)}
                        >
                            {/* Color dot */}
                            <div style={{
                                width: '8px', height: '8px', 'border-radius': '50%',
                                background: bm.color || '#0099e0', 'flex-shrink': '0',
                            }} />

                            {/* Info */}
                            <div style={{ flex: '1', 'min-width': '0' }}>
                                <div style={{
                                    'font-size': '12px', 'font-weight': '600',
                                    color: 'var(--text-primary)', overflow: 'hidden',
                                    'text-overflow': 'ellipsis', 'white-space': 'nowrap',
                                }}>{bm.label}</div>
                                <div style={{
                                    'font-size': '10px', color: 'var(--text-muted)',
                                    'font-family': 'var(--font-mono)',
                                    overflow: 'hidden', 'text-overflow': 'ellipsis', 'white-space': 'nowrap',
                                }}>
                                    {bm.username}@{bm.hostname}:{bm.port}
                                </div>
                                <Show when={bm.tags?.length > 0}>
                                    <div style={{ display: 'flex', gap: '3px', 'margin-top': '2px', 'flex-wrap': 'wrap' }}>
                                        <For each={bm.tags}>
                                            {(tag) => (
                                                <span style={{
                                                    'font-size': '8px', padding: '1px 4px',
                                                    'border-radius': '2px', background: 'rgba(0,153,224,0.1)',
                                                    color: '#0099e0', border: '1px solid rgba(0,153,224,0.2)',
                                                }}>{tag}</span>
                                            )}
                                        </For>
                                    </div>
                                </Show>
                            </div>

                            {/* Actions */}
                            <div style={{ display: 'flex', gap: '4px', 'flex-shrink': '0' }}>
                                <button onClick={(e) => { e.stopPropagation(); handleToggleFavorite(bm.id); }}
                                    style={{
                                        background: 'none', border: 'none', cursor: 'pointer',
                                        color: bm.is_favorite ? '#f5c518' : 'var(--text-muted)',
                                        'font-size': '14px', padding: '0',
                                    }} title={bm.is_favorite ? 'Unfavorite' : 'Favorite'}>
                                    {bm.is_favorite ? '★' : '☆'}
                                </button>
                                <button onClick={(e) => { e.stopPropagation(); handleConnect(bm.id); }}
                                    style={{
                                        background: 'rgba(92,192,92,0.1)', border: '1px solid rgba(92,192,92,0.3)',
                                        color: '#5cc05c', cursor: 'pointer', 'font-size': '9px',
                                        padding: '2px 8px', 'border-radius': '2px', 'font-weight': '700',
                                    }} title="Connect">
                                    SSH
                                </button>
                                <button onClick={(e) => { e.stopPropagation(); handleDelete(bm.id); }}
                                    style={{
                                        background: 'none', border: 'none', cursor: 'pointer',
                                        color: 'var(--text-muted)', 'font-size': '12px', padding: '0 2px',
                                    }} title="Delete">🗑</button>
                            </div>
                        </div>
                    )}
                </For>
            </div>

            {/* Footer */}
            <div style={{
                padding: '6px 14px', 'border-top': '1px solid var(--border-primary)',
                display: 'flex', 'justify-content': 'space-between',
                'font-size': '10px', color: 'var(--text-muted)', 'flex-shrink': '0',
            }}>
                <span>{filtered().length} hosts</span>
                <span style={{ 'font-family': 'var(--font-mono)' }}>
                    {bookmarks()?.filter(b => b.is_favorite).length || 0} ★
                </span>
            </div>
        </div>
    );
};
