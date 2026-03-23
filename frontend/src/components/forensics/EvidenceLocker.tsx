import { Component, createSignal, For, Show, createResource, onCleanup, createEffect } from 'solid-js';
import { subscribe } from '@core/bridge';
import { IS_BROWSER } from '@core/context';
import { ForensicView } from '../security/ForensicView';

// Evidence Locker — Chain-of-Custody Forensics UI
export const EvidenceLocker: Component = () => {
    const [activeTab, setActiveTab] = createSignal<'evidence' | 'chain' | 'verify'>('evidence');
    const [selectedItem, setSelectedItem] = createSignal<string | null>(null);
    const [showForensics, setShowForensics] = createSignal(false);

    // Backend resources
    const [evidence, { refetch: refetchEvidence }] = createResource(async () => {
        if (IS_BROWSER) return [];
        const { ListEvidence } = await import('../../../wailsjs/go/services/ForensicsService');
        return ListEvidence('');
    });
    const [chainEntries, { refetch: refetchChain }] = createResource(selectedItem, async (id) => {
        if (IS_BROWSER || !id) return [];
        const { GetChainOfCustody } = await import('../../../wailsjs/go/services/ForensicsService');
        return GetChainOfCustody(id);
    });

    // Listen for new evidence events to auto-refresh
    createEffect(() => {
        const off1 = subscribe('forensics:collected', () => refetchEvidence());
        const off2 = subscribe('forensics:sealed', () => {
            refetchEvidence();
            if (selectedItem()) refetchChain();
        });
        onCleanup(() => { off1(); off2(); });
    });

    const typeIcon = (type: string) => {
        const icons: Record<string, string> = { log: '📋', pcap: '🔌', file: '📁', screenshot: '📸', memory_dump: '💾', artifact: '🔬' };
        return icons[type] || '📦';
    };

    const actionColor = (action: string) => {
        const colors: Record<string, string> = {
            collected: '#22c55e', transferred: '#3b82f6', analyzed: '#8b5cf6', sealed: '#f59e0b', verified: '#06b6d4', exported: '#ec4899'
        };
        return colors[action] || '#6b7280';
    };

    const handleVerify = async (id: string) => {
        if (IS_BROWSER) return;
        try {
            const { VerifyEvidence } = await import('../../../wailsjs/go/services/ForensicsService');
            const result = await VerifyEvidence(id);
            if (result.valid) alert(`✅ Integrity Verified: Chain for ${id} is cryptographically sound.`);
            else alert(`❌ INTEGRITY BREACH: Chain for ${id} has been tampered with!`);
        } catch (e) { console.error(e); }
    };

    return (
        <div class="evidence-locker app-entry-animation" style={{
            padding: '1.5rem', height: '100%', overflow: 'auto',
            background: 'var(--bg-primary, #0f1117)', color: 'var(--text-primary, #e4e6f0)'
        }}>
            <div style={{ display: 'flex', 'align-items': 'center', 'justify-content': 'space-between', 'margin-bottom': '1.5rem' }}>
                <div>
                    <h1 style={{ 'font-size': '1.5rem', 'font-weight': '700', margin: 0 }}>🔒 Evidence Locker</h1>
                    <p style={{ color: '#8b8fa3', 'font-size': '0.85rem', margin: '0.25rem 0 0' }}>Chain-of-custody forensic evidence management</p>
                </div>
                <div style={{ display: 'flex', gap: '0.5rem' }}>
                    <button
                        onClick={() => refetchEvidence()}
                        style={{
                            background: '#1a1d27', color: '#8b8fa3', border: '1px solid #2a2d3a',
                            padding: '0.5rem 1rem', 'border-radius': '6px', cursor: 'pointer'
                        }}
                    >Refresh</button>
                    <button style={{
                        background: 'linear-gradient(135deg, #6366f1, #8b5cf6)',
                        color: 'white', border: 'none', padding: '0.5rem 1.25rem', 'border-radius': '8px', cursor: 'pointer', 'font-weight': '600'
                    }}>+ Collect Evidence</button>
                </div>
            </div>

            {/* Tab Bar */}
            <div style={{ display: 'flex', gap: '0.5rem', 'margin-bottom': '1.5rem', 'border-bottom': '1px solid #2a2d3a', 'padding-bottom': '0.5rem' }}>
                {(['evidence', 'chain', 'verify'] as const).map(tab => (
                    <button
                        onClick={() => setActiveTab(tab)}
                        style={{
                            background: activeTab() === tab ? 'rgba(99,102,241,0.15)' : 'transparent',
                            color: activeTab() === tab ? '#6366f1' : '#8b8fa3',
                            border: 'none', padding: '0.5rem 1rem', 'border-radius': '6px', cursor: 'pointer', 'font-size': '0.85rem', 'font-weight': '600',
                            'text-transform': 'capitalize'
                        }}
                    >{tab === 'chain' ? 'Chain of Custody' : tab === 'verify' ? 'Verify Integrity' : 'Evidence Items'}</button>
                ))}
            </div>

            {/* Evidence Items Tab */}
            <Show when={activeTab() === 'evidence'}>
                <div style={{ display: 'grid', gap: '0.75rem' }}>
                    <For each={evidence()}>
                        {(item: any) => (
                            <div
                                onClick={() => setSelectedItem(item.id)}
                                style={{
                                    background: selectedItem() === item.id ? 'rgba(99,102,241,0.08)' : '#1a1d27',
                                    border: `1px solid ${selectedItem() === item.id ? '#6366f1' : '#2a2d3a'}`,
                                    'border-radius': '10px', padding: '1rem', cursor: 'pointer', transition: 'all 0.2s'
                                }}
                            >
                                <div style={{ display: 'flex', 'align-items': 'center', 'justify-content': 'space-between' }}>
                                    <div style={{ display: 'flex', 'align-items': 'center', gap: '0.75rem' }}>
                                        <span style={{ 'font-size': '1.5rem' }}>{typeIcon(item.type)}</span>
                                        <div>
                                            <div style={{ 'font-weight': '600' }}>{item.name}</div>
                                            <div style={{ 'font-size': '0.75rem', color: '#8b8fa3' }}>
                                                {item.id} · {item.incident_id} · SHA256: {item.sha256.substring(0, 16)}...
                                            </div>
                                        </div>
                                    </div>
                                    <div style={{ display: 'flex', gap: '0.5rem', 'align-items': 'center' }}>
                                        <span style={{
                                            background: item.sealed ? 'rgba(245,158,11,0.15)' : 'rgba(34,197,94,0.15)',
                                            color: item.sealed ? '#f59e0b' : '#22c55e',
                                            padding: '2px 8px', 'border-radius': '4px', 'font-size': '0.7rem', 'font-weight': '700'
                                        }}>{item.sealed ? '🔒 SEALED' : '🟢 ACTIVE'}</span>
                                        <span style={{ color: '#8b8fa3', 'font-size': '0.75rem' }}>
                                            {item.chain_of_custody?.length || 0} entries
                                        </span>
                                        <button
                                            onClick={(e) => { e.stopPropagation(); setSelectedItem(item.id); setShowForensics(true); }}
                                            class="analyze-btn"
                                            style={{
                                                background: 'rgba(99,102,241,0.2)', border: '1px solid #6366f1', color: '#818cf8',
                                                padding: '4px 10px', 'border-radius': '6px', 'font-size': '0.7rem', 'font-weight': '700',
                                                cursor: 'pointer'
                                            }}
                                        >ANALYZE</button>
                                    </div>
                                </div>
                                <div style={{ 'font-size': '0.75rem', color: '#6b7280', 'margin-top': '0.5rem' }}>
                                    Collected by {item.collector} at {new Date(item.collected_at).toLocaleString()}
                                </div>
                            </div>
                        )}
                    </For>
                    <Show when={evidence()?.length === 0}>
                        <div style={{ padding: '2rem', 'text-align': 'center', opacity: 0.5 }}>No evidence items collected.</div>
                    </Show>
                </div>
            </Show>

            {/* Chain of Custody Tab */}
            <Show when={activeTab() === 'chain'}>
                <Show when={selectedItem()} fallback={<div style={{ padding: '2rem', 'text-align': 'center', opacity: 0.5 }}>Select an item to view its chain of custody.</div>}>
                    <div style={{ position: 'relative', 'padding-left': '2rem' }}>
                        <div style={{ position: 'absolute', left: '0.75rem', top: '0', bottom: '0', width: '2px', background: 'linear-gradient(180deg, #6366f1, #22c55e)' }} />
                        <For each={chainEntries()}>
                            {(entry: any) => (
                                <div style={{ position: 'relative', 'margin-bottom': '1.5rem' }}>
                                    <div style={{
                                        position: 'absolute', left: '-1.65rem', top: '0.25rem', width: '12px', height: '12px',
                                        'border-radius': '50%', background: actionColor(entry.action), border: '2px solid #0f1117'
                                    }} />
                                    <div style={{ background: '#1a1d27', border: '1px solid #2a2d3a', 'border-radius': '8px', padding: '1rem' }}>
                                        <div style={{ display: 'flex', 'justify-content': 'space-between', 'align-items': 'center' }}>
                                            <span style={{
                                                color: actionColor(entry.action), 'font-weight': '700', 'font-size': '0.85rem', 'text-transform': 'uppercase'
                                            }}>{entry.action}</span>
                                            <span style={{ color: '#6b7280', 'font-size': '0.75rem' }}>{new Date(entry.timestamp).toLocaleString()}</span>
                                        </div>
                                        <div style={{ 'margin-top': '0.5rem' }}>
                                            <span style={{ color: '#c4c6d0' }}>by </span>
                                            <span style={{ color: '#818cf8', 'font-weight': '600' }}>{entry.actor}</span>
                                        </div>
                                        <div style={{ color: '#8b8fa3', 'font-size': '0.85rem', 'margin-top': '0.25rem' }}>{entry.notes}</div>
                                        <div style={{ color: '#4b5563', 'font-size': '0.7rem', 'margin-top': '0.5rem', 'font-family': 'monospace' }}>
                                            HMAC: {entry.entry_hash.substring(0, 32)}...
                                        </div>
                                    </div>
                                </div>
                            )}
                        </For>
                    </div>
                </Show>
            </Show>

            {/* Verify Integrity Tab */}
            <Show when={activeTab() === 'verify'}>
                <div style={{ display: 'grid', gap: '1rem', 'max-width': '600px' }}>
                    <div style={{ background: '#1a1d27', border: '1px solid #2a2d3a', 'border-radius': '10px', padding: '1.5rem', 'text-align': 'center' }}>
                        <div style={{ 'font-size': '3rem', 'margin-bottom': '0.5rem' }}>🛡️</div>
                        <div style={{ 'font-size': '1.25rem', 'font-weight': '700', color: 'var(--accent-primary, #6366f1)' }}>System Integrity Scan</div>
                        <p style={{ color: '#8b8fa3', 'font-size': '0.85rem', 'margin-top': '0.5rem' }}>
                            Verify HMAC chain links and Merkle root hashes for all collected evidence.
                        </p>
                        <button
                            onClick={() => selectedItem() && handleVerify(selectedItem()!)}
                            disabled={!selectedItem()}
                            style={{
                                'margin-top': '1rem', background: '#6366f1', color: 'white', border: 'none',
                                padding: '0.6rem 1.5rem', 'border-radius': '6px', cursor: selectedItem() ? 'pointer' : 'not-allowed',
                                opacity: selectedItem() ? 1 : 0.5
                            }}
                        >Verify Selected Item</button>
                    </div>

                    <div style={{ display: 'grid', 'grid-template-columns': '1fr 1fr 1fr', gap: '0.75rem' }}>
                        <div style={{ background: '#1a1d27', border: '1px solid #2a2d3a', 'border-radius': '8px', padding: '1rem', 'text-align': 'center' }}>
                            <div style={{ 'font-size': '1.5rem', 'font-weight': '900', color: '#6366f1' }}>{evidence()?.length || 0}</div>
                            <div style={{ 'font-size': '0.75rem', color: '#8b8fa3' }}>Total Items</div>
                        </div>
                        <div style={{ background: '#1a1d27', border: '1px solid #2a2d3a', 'border-radius': '8px', padding: '1rem', 'text-align': 'center' }}>
                            <div style={{ 'font-size': '1.5rem', 'font-weight': '900', color: '#f59e0b' }}>{evidence()?.filter((e: any) => e.sealed).length || 0}</div>
                            <div style={{ 'font-size': '0.75rem', color: '#8b8fa3' }}>Sealed</div>
                        </div>
                        <div style={{ background: '#1a1d27', border: '1px solid #2a2d3a', 'border-radius': '8px', padding: '1rem', 'text-align': 'center' }}>
                            <div style={{ 'font-size': '1.5rem', 'font-weight': '900', color: '#22c55e' }}>{evidence()?.reduce((acc: number, curr: any) => acc + (curr.chain_of_custody?.length || 0), 0)}</div>
                            <div style={{ 'font-size': '0.75rem', color: '#8b8fa3' }}>Chain Entries</div>
                        </div>
                    </div>
                </div>
            </Show>

            <Show when={showForensics()}>
                <ForensicView evidenceId={selectedItem()!} onClose={() => setShowForensics(false)} />
            </Show>
        </div>
    );
};
