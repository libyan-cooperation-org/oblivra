// RemoteForensics.tsx — Phase 6 Web: evidence access and visualization
import { Component, createSignal, onMount, For, Show } from 'solid-js';
import * as ForensicsService from '../../../wailsjs/go/services/ForensicsService';

export const RemoteForensics: Component = () => {
    const [items, setItems] = createSignal<any[]>([]);
    const [selected, setSelected] = createSignal<any>(null);
    const [chain, setChain] = createSignal<any[]>([]);
    const [loading, setLoading] = createSignal(true);
    const [filter, setFilter] = createSignal('');

    onMount(async () => {
        try {
            // ListEvidence requires an incident_id; pass empty string for all
            const ev = await (ForensicsService as any).ListEvidence('');
            setItems(ev ?? []);
        } catch { setItems([]); }
        setLoading(false);
    });

    const selectItem = async (item: any) => {
        setSelected(item);
        try {
            const c = await (ForensicsService as any).GetChainOfCustody?.(item.id) ?? [];
            setChain(c);
        } catch { setChain([]); }
    };

    const TYPE_ICONS: Record<string, string> = {
        pcap: '📡', memory: '💾', log: '📋', screenshot: '🖼️', disk: '💿', default: '🔍'
    };

    const filtered = () => {
        const q = filter().toLowerCase();
        return items().filter(i => !q || i.name?.toLowerCase().includes(q) || i.incident_id?.toLowerCase().includes(q));
    };

    return (
        <div style="padding: 0; height: 100%; background: var(--bg-primary); color: var(--text-primary); font-family: var(--font-ui); display: flex; flex-direction: column; overflow: hidden;">
            <div style="height: var(--header-height); border-bottom: 1px solid var(--glass-border); display: flex; justify-content: space-between; align-items: center; padding: 0 1.5rem; background: var(--bg-secondary); flex-shrink: 0;">
                <div style="display: flex; align-items: center; gap: 0.75rem;">
                    <span style="font-size: 16px;">🔬</span>
                    <h2 style="font-size: 13px; letter-spacing: 2px; font-weight: 700; margin: 0; text-transform: uppercase;">Remote Forensics</h2>
                </div>
                <span style="font-size: 10px; color: var(--text-muted); font-family: var(--font-mono);">{items().length} EVIDENCE ITEMS</span>
            </div>

            <div style="flex: 1; display: grid; grid-template-columns: 320px 1fr; overflow: hidden;">
                {/* Evidence list */}
                <div style="border-right: 1px solid var(--glass-border); display: flex; flex-direction: column; overflow: hidden;">
                    <div style="padding: 0.75rem; border-bottom: 1px solid var(--glass-border);">
                        <input placeholder="Filter evidence..." value={filter()} onInput={e => setFilter((e.target as HTMLInputElement).value)}
                            style="width: 100%; background: var(--bg-primary); border: 1px solid var(--glass-border); color: var(--text-primary); padding: 6px 10px; border-radius: 4px; font-family: var(--font-mono); font-size: 11px; box-sizing: border-box;" />
                    </div>
                    <div style="flex: 1; overflow-y: auto;">
                        <Show when={loading()}>
                            <div style="padding: 2rem; text-align: center; color: var(--text-muted); font-family: var(--font-mono); font-size: 11px;">LOADING...</div>
                        </Show>
                        <Show when={!loading() && filtered().length === 0}>
                            <div style="padding: 3rem; text-align: center; color: var(--text-muted); font-family: var(--font-mono); font-size: 11px; letter-spacing: 1px;">
                                <div style="font-size: 2rem; opacity: 0.2; margin-bottom: 0.75rem;">🔬</div>
                                NO EVIDENCE ITEMS
                            </div>
                        </Show>
                        <For each={filtered()}>
                            {(item) => {
                                const isSelected = selected()?.id === item.id;
                                const icon = TYPE_ICONS[item.type ?? 'default'] ?? TYPE_ICONS.default;
                                return (
                                    <div onClick={() => selectItem(item)}
                                        style={`padding: 10px 1rem; border-bottom: 1px solid var(--glass-border); cursor: pointer; background: ${isSelected ? 'rgba(87,139,255,0.08)' : 'transparent'}; border-left: 3px solid ${isSelected ? 'var(--accent-primary)' : 'transparent'};`}>
                                        <div style="display: flex; align-items: center; gap: 8px; margin-bottom: 3px;">
                                            <span>{icon}</span>
                                            <span style="font-size: 11px; font-weight: 600; color: var(--text-primary);">{item.name ?? item.id}</span>
                                        </div>
                                        <div style="font-size: 10px; font-family: var(--font-mono); color: var(--text-muted);">
                                            {item.type?.toUpperCase() ?? 'UNKNOWN'} · INC: {item.incident_id ?? 'none'}
                                        </div>
                                        <div style="font-size: 9px; color: var(--text-muted); margin-top: 2px;">{item.collected_at?.slice(0, 16)?.replace('T', ' ')}</div>
                                    </div>
                                );
                            }}
                        </For>
                    </div>
                </div>

                {/* Detail */}
                <div style="overflow-y: auto; padding: 1.5rem; display: flex; flex-direction: column; gap: 1.25rem;">
                    <Show when={!selected()}>
                        <div style="flex: 1; display: flex; align-items: center; justify-content: center; color: var(--text-muted); font-family: var(--font-mono); font-size: 11px; letter-spacing: 1px; flex-direction: column; gap: 1rem; opacity: 0.4;">
                            <span style="font-size: 3rem;">🔬</span>SELECT AN EVIDENCE ITEM
                        </div>
                    </Show>

                    <Show when={selected()}>
                        {/* Meta */}
                        <div style="background: var(--bg-secondary); border: 1px solid var(--glass-border); border-radius: 6px; padding: 1.25rem;">
                            <div style="font-size: 10px; text-transform: uppercase; letter-spacing: 1px; color: var(--text-muted); font-family: var(--font-mono); margin-bottom: 0.75rem;">Evidence Item</div>
                            <div style="display: grid; grid-template-columns: 1fr 1fr; gap: 0.75rem;">
                                {[['ID', selected()?.id], ['Type', selected()?.type], ['Name', selected()?.name], ['Incident', selected()?.incident_id], ['Size', selected()?.size_bytes ? `${(selected().size_bytes / 1024).toFixed(1)} KB` : '—'], ['Hash (SHA-256)', selected()?.sha256_hash], ['Collected', selected()?.collected_at?.slice(0, 16)?.replace('T', ' ')], ['Collector', selected()?.collected_by]].map(([label, value]) => (
                                    <div>
                                        <div style="font-size: 9px; color: var(--text-muted); font-family: var(--font-mono); text-transform: uppercase; letter-spacing: 0.5px; margin-bottom: 2px;">{label}</div>
                                        <div style="font-size: 11px; color: var(--text-primary); font-family: var(--font-mono); word-break: break-all;">{value ?? '—'}</div>
                                    </div>
                                ))}
                            </div>
                        </div>

                        {/* Chain of custody */}
                        <div style="background: var(--bg-secondary); border: 1px solid var(--glass-border); border-radius: 6px; padding: 1.25rem;">
                            <div style="font-size: 10px; text-transform: uppercase; letter-spacing: 1px; color: var(--text-muted); font-family: var(--font-mono); margin-bottom: 0.75rem;">Chain of Custody ({chain().length} entries)</div>
                            <Show when={chain().length === 0}>
                                <div style="color: var(--text-muted); font-family: var(--font-mono); font-size: 11px;">No chain entries recorded</div>
                            </Show>
                            <div style="display: flex; flex-direction: column; gap: 0.5rem; max-height: 220px; overflow-y: auto;">
                                <For each={chain()}>
                                    {(entry: any, idx) => (
                                        <div style={`padding: 8px 10px; border-radius: 4px; background: rgba(255,255,255,0.03); border-left: 2px solid ${idx() === 0 ? '#3fb950' : 'var(--glass-border)'};`}>
                                            <div style="display: flex; justify-content: space-between; font-size: 10px; font-family: var(--font-mono); margin-bottom: 2px;">
                                                <span style="color: var(--text-primary); font-weight: 600;">{entry.action?.toUpperCase() ?? 'ACTION'}</span>
                                                <span style="color: var(--text-muted);">{entry.timestamp?.slice(0, 16)?.replace('T', ' ')}</span>
                                            </div>
                                            <div style="font-size: 10px; color: var(--text-muted);">by {entry.actor ?? 'system'}</div>
                                            <Show when={entry.notes}><div style="font-size: 10px; color: var(--text-secondary); margin-top: 3px; font-style: italic;">{entry.notes}</div></Show>
                                        </div>
                                    )}
                                </For>
                            </div>
                        </div>
                    </Show>
                </div>
            </div>
        </div>
    );
};
