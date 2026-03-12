import { Component, createSignal, onMount, For, Show } from 'solid-js';
import { GetChain, VerifyChain, ExportChain } from '../../wailsjs/go/app/LedgerService';

interface LedgerBlock {
    index: number;
    timestamp: string;
    data: string;
    data_type: string;
    previous_hash: string;
    hash: string;
}

type VerifyStatus = 'UNKNOWN' | 'VERIFYING' | 'VALID' | 'INVALID';

const statusColor = (s: VerifyStatus) => {
    switch (s) {
        case 'VALID':     return 'var(--alert-low)';
        case 'INVALID':   return 'var(--alert-critical)';
        case 'VERIFYING': return 'var(--alert-medium)';
        default:          return 'var(--text-muted)';
    }
};

const Label: Component<{ children: any }> = (p) => (
    <div style={{
        'font-family': 'var(--font-mono)',
        'font-size': '9px',
        'font-weight': 800,
        color: 'var(--text-muted)',
        'text-transform': 'uppercase',
        'letter-spacing': '1px',
        'margin-bottom': '4px',
    }}>{p.children}</div>
);

const Value: Component<{ children: any; accent?: string }> = (p) => (
    <div style={{
        'font-family': 'var(--font-mono)',
        'font-size': '11px',
        color: p.accent ?? 'var(--text-secondary)',
        'word-break': 'break-all',
        'line-height': 1.5,
    }}>{p.children}</div>
);

export const EvidenceLedger: Component = () => {
    const [blocks, setBlocks]             = createSignal<LedgerBlock[]>([]);
    const [status, setStatus]             = createSignal<VerifyStatus>('UNKNOWN');
    const [errorMsg, setErrorMsg]         = createSignal('');
    const [selectedBlock, setSelectedBlock] = createSignal<LedgerBlock | null>(null);
    const [loading, setLoading]           = createSignal(true);
    const [loadErr, setLoadErr]           = createSignal('');

    const loadChain = async () => {
        setLoading(true);
        setLoadErr('');
        try {
            const data = await GetChain();
            setBlocks(data || []);
        } catch (err: any) {
            setLoadErr(err?.message ?? String(err));
        } finally {
            setLoading(false);
        }
    };

    onMount(loadChain);

    const verifyLedger = async () => {
        setStatus('VERIFYING');
        setErrorMsg('');
        try {
            const res = await VerifyChain();
            setStatus(res === 'VALID' ? 'VALID' : 'INVALID');
            if (res !== 'VALID') setErrorMsg(res);
        } catch (err: any) {
            setStatus('INVALID');
            setErrorMsg(err?.message ?? String(err));
        }
    };

    const exportLedger = async () => {
        try {
            const jsonStr = await ExportChain();
            const blob = new Blob([jsonStr], { type: 'application/json' });
            const url = URL.createObjectURL(blob);
            const a = Object.assign(document.createElement('a'), {
                href: url,
                download: `sovereign_evidence_ledger_${Date.now()}.json`,
            });
            a.click();
            URL.revokeObjectURL(url);
        } catch (err: any) {
            setLoadErr(err?.message ?? String(err));
        }
    };

    const decodeData = (b64: string) => {
        try { return atob(b64); } catch { return b64; }
    };

    return (
        <div style={{
            display: 'flex',
            'flex-direction': 'column',
            height: '100%',
            overflow: 'hidden',
            background: 'var(--surface-0)',
            padding: '28px 32px 0 32px',
        }}>
            {/* ── Header ── */}
            <div style={{
                display: 'flex',
                'justify-content': 'space-between',
                'align-items': 'flex-end',
                'margin-bottom': '20px',
                'padding-bottom': '16px',
                'border-bottom': '1px solid var(--border-primary)',
                'flex-shrink': 0,
            }}>
                <div>
                    <div style={{ 'font-family': 'var(--font-mono)', 'font-size': '10px', 'font-weight': 700, color: 'var(--text-muted)', 'text-transform': 'uppercase', 'letter-spacing': '2px', 'margin-bottom': '4px' }}>
                        CRYPTOGRAPHIC AUDIT
                    </div>
                    <h1 style={{ 'font-family': 'var(--font-mono)', 'font-size': '20px', 'font-weight': 800, color: 'var(--text-primary)', margin: 0, 'letter-spacing': '-0.5px' }}>
                        Evidence Ledger
                    </h1>
                </div>
                <div style={{ display: 'flex', gap: '8px' }}>
                    <button
                        onClick={verifyLedger}
                        disabled={status() === 'VERIFYING'}
                        style={{
                            background: 'transparent',
                            border: `1px solid ${statusColor(status())}`,
                            color: statusColor(status()),
                            'font-family': 'var(--font-mono)',
                            'font-size': '10px',
                            'font-weight': 800,
                            'text-transform': 'uppercase',
                            'letter-spacing': '0.5px',
                            padding: '6px 16px',
                            cursor: status() === 'VERIFYING' ? 'wait' : 'pointer',
                        }}
                    >
                        {status() === 'VERIFYING' ? 'VERIFYING...' : 'VERIFY INTEGRITY'}
                    </button>
                    <button
                        onClick={exportLedger}
                        style={{
                            background: 'transparent',
                            border: '1px solid var(--border-primary)',
                            color: 'var(--text-secondary)',
                            'font-family': 'var(--font-mono)',
                            'font-size': '10px',
                            'font-weight': 700,
                            'text-transform': 'uppercase',
                            padding: '6px 16px',
                            cursor: 'pointer',
                        }}
                    >
                        EXPORT JSON
                    </button>
                </div>
            </div>

            {/* ── Verify result banner ── */}
            <Show when={status() !== 'UNKNOWN'}>
                <div style={{
                    padding: '10px 16px',
                    'margin-bottom': '16px',
                    border: `1px solid ${statusColor(status())}`,
                    background: status() === 'VALID' ? 'rgba(132,204,22,0.05)' : 'rgba(220,38,38,0.05)',
                    display: 'flex',
                    'align-items': 'center',
                    'justify-content': 'space-between',
                    'flex-shrink': 0,
                }}>
                    <span style={{ 'font-family': 'var(--font-mono)', 'font-size': '11px', 'font-weight': 800, color: statusColor(status()), 'text-transform': 'uppercase', 'letter-spacing': '0.5px' }}>
                        {status() === 'VALID' ? '✓ CHAIN CRYPTOGRAPHICALLY VALID' : `⚠ INTEGRITY VIOLATION — ${errorMsg()}`}
                    </span>
                </div>
            </Show>

            {/* ── Load error ── */}
            <Show when={loadErr()}>
                <div style={{ padding: '10px 16px', 'margin-bottom': '16px', border: '1px solid var(--alert-critical)', 'font-family': 'var(--font-mono)', 'font-size': '11px', color: 'var(--alert-critical)', 'flex-shrink': 0 }}>
                    {loadErr()}
                </div>
            </Show>

            {/* ── Main split ── */}
            <div style={{ display: 'flex', gap: '16px', flex: 1, 'min-height': 0, 'padding-bottom': '28px' }}>

                {/* Block list */}
                <div style={{
                    width: '280px',
                    'min-width': '280px',
                    background: 'var(--surface-1)',
                    border: '1px solid var(--border-primary)',
                    display: 'flex',
                    'flex-direction': 'column',
                    overflow: 'hidden',
                }}>
                    <div style={{
                        padding: '10px 14px',
                        'border-bottom': '1px solid var(--border-primary)',
                        'font-family': 'var(--font-mono)',
                        'font-size': '9px',
                        'font-weight': 800,
                        color: 'var(--text-muted)',
                        'text-transform': 'uppercase',
                        'letter-spacing': '1px',
                        'flex-shrink': 0,
                    }}>
                        BLOCKS ({blocks().length})
                    </div>
                    <div style={{ flex: 1, 'overflow-y': 'auto', padding: '8px' }}>
                        <Show when={loading()}>
                            <div style={{ padding: '24px', 'text-align': 'center', 'font-family': 'var(--font-mono)', 'font-size': '11px', color: 'var(--text-muted)' }}>LOADING...</div>
                        </Show>
                        <For each={blocks()}>
                            {(block) => {
                                const isSelected = () => selectedBlock()?.index === block.index;
                                const isGenesis = block.index === 0;
                                return (
                                    <div
                                        onClick={() => setSelectedBlock(block)}
                                        style={{
                                            padding: '10px 12px',
                                            'margin-bottom': '4px',
                                            background: isSelected() ? 'var(--surface-2)' : 'transparent',
                                            border: `1px solid ${isSelected() ? 'var(--accent-primary)' : 'var(--border-primary)'}`,
                                            'border-left': `3px solid ${isGenesis ? 'var(--alert-medium)' : isSelected() ? 'var(--accent-primary)' : 'var(--border-primary)'}`,
                                            cursor: 'pointer',
                                        }}
                                    >
                                        <div style={{ display: 'flex', 'justify-content': 'space-between', 'margin-bottom': '4px' }}>
                                            <span style={{ 'font-family': 'var(--font-mono)', 'font-size': '11px', 'font-weight': 800, color: isGenesis ? 'var(--alert-medium)' : 'var(--text-primary)' }}>
                                                #{block.index} {isGenesis ? '(GENESIS)' : ''}
                                            </span>
                                            <span style={{ 'font-family': 'var(--font-mono)', 'font-size': '9px', color: 'var(--text-muted)', 'text-transform': 'uppercase' }}>
                                                {block.data_type}
                                            </span>
                                        </div>
                                        <div style={{ 'font-family': 'var(--font-mono)', 'font-size': '10px', color: 'var(--accent-primary)', 'margin-bottom': '4px' }}>
                                            {block.hash.substring(0, 20)}…
                                        </div>
                                        <div style={{ 'font-family': 'var(--font-mono)', 'font-size': '9px', color: 'var(--text-muted)' }}>
                                            {new Date(block.timestamp).toISOString().replace('T', ' ').substring(0, 19)}
                                        </div>
                                    </div>
                                );
                            }}
                        </For>
                        <Show when={!loading() && blocks().length === 0 && !loadErr()}>
                            <div style={{ padding: '24px', 'text-align': 'center', 'font-family': 'var(--font-mono)', 'font-size': '11px', color: 'var(--text-muted)', 'text-transform': 'uppercase' }}>
                                CHAIN EMPTY
                            </div>
                        </Show>
                    </div>
                </div>

                {/* Block detail */}
                <div style={{
                    flex: 1,
                    background: 'var(--surface-1)',
                    border: '1px solid var(--border-primary)',
                    display: 'flex',
                    'flex-direction': 'column',
                    overflow: 'hidden',
                }}>
                    <Show when={!selectedBlock()}>
                        <div style={{ display: 'flex', 'align-items': 'center', 'justify-content': 'center', height: '100%', 'font-family': 'var(--font-mono)', 'font-size': '11px', color: 'var(--text-muted)', 'text-transform': 'uppercase', 'letter-spacing': '1px' }}>
                            SELECT A BLOCK TO INSPECT
                        </div>
                    </Show>
                    <Show when={selectedBlock()}>
                        {(block) => (
                            <div style={{ padding: '20px 24px', 'overflow-y': 'auto', flex: 1 }}>
                                <div style={{ 'font-family': 'var(--font-mono)', 'font-size': '10px', 'font-weight': 800, color: 'var(--text-muted)', 'text-transform': 'uppercase', 'letter-spacing': '1.5px', 'margin-bottom': '20px', 'padding-bottom': '12px', 'border-bottom': '1px solid var(--border-primary)' }}>
                                    BLOCK #{block().index} DETAILS
                                </div>
                                <div style={{ display: 'grid', 'grid-template-columns': '120px 1fr', gap: '12px 16px', 'margin-bottom': '24px' }}>
                                    <Label>Index</Label>       <Value>{block().index}</Value>
                                    <Label>Timestamp</Label>   <Value>{new Date(block().timestamp).toLocaleString()}</Value>
                                    <Label>Data Type</Label>   <Value accent="var(--accent-primary)">{block().data_type}</Value>
                                    <Label>Hash</Label>        <Value accent="var(--accent-primary)">{block().hash}</Value>
                                    <Label>Prev Hash</Label>   <Value>{block().previous_hash || '0'.repeat(64) + ' (genesis)'}</Value>
                                </div>
                                <Label>Evidence Payload (Decoded)</Label>
                                <pre style={{
                                    background: 'var(--surface-0)',
                                    border: '1px solid var(--border-primary)',
                                    'border-left': '3px solid var(--accent-primary)',
                                    padding: '16px',
                                    'font-family': 'var(--font-mono)',
                                    'font-size': '11px',
                                    color: 'var(--text-secondary)',
                                    'overflow-x': 'auto',
                                    'white-space': 'pre-wrap',
                                    'word-break': 'break-all',
                                    'line-height': 1.6,
                                    'margin-top': '8px',
                                }}>
                                    {decodeData(block().data)}
                                </pre>
                            </div>
                        )}
                    </Show>
                </div>
            </div>
        </div>
    );
};
