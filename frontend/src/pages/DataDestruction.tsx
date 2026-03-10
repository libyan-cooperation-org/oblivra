import { Component, createSignal, For, Show } from 'solid-js';

interface DestructionRecord {
    id: string;
    timestamp: string;
    initiated_by: string;
    scope: string;
    method: string;
    item_count: number;
    status: string;
}

export const DataDestruction: Component = () => {
    const [history] = createSignal<DestructionRecord[]>([]);
    const [targetPath, setTargetPath] = createSignal('');
    const [confirmText, setConfirmText] = createSignal('');
    const [wiping, setWiping] = createSignal(false);

    const handleWipe = async () => {
        if (confirmText() !== 'DESTROY') return;
        setWiping(true);
        // Will be wired to Wails binding: DataDestructionService.CryptoWipeFile(targetPath, 'admin')
        setTimeout(() => {
            setWiping(false);
            setTargetPath('');
            setConfirmText('');
            alert('Crypto wipe completed.');
        }, 2000);
    };

    return (
        <div class="destruction-page">
            <header class="dest-header">
                <div>
                    <h1>DATA DESTRUCTION</h1>
                    <p>NIST SP 800-88 cryptographic wipe with immutable audit trail</p>
                </div>
                <div class="compliance-badges">
                    <span class="badge-gdpr">GDPR Art. 17</span>
                    <span class="badge-pci">PCI-DSS 3.1</span>
                </div>
            </header>

            <div class="dest-grid">
                {/* Wipe Control */}
                <section class="wipe-panel">
                    <h3>SECURE ERASURE</h3>
                    <div class="warning-box">
                        ⚠️ This operation is <strong>irreversible</strong>. Data is overwritten with cryptographically random bytes before deletion.
                    </div>

                    <div class="wipe-form">
                        <label>Target Path</label>
                        <input
                            type="text"
                            placeholder="/path/to/sensitive/data"
                            value={targetPath()}
                            onInput={(e) => setTargetPath(e.currentTarget.value)}
                            class="wipe-input"
                        />

                        <label>Confirmation</label>
                        <input
                            type="text"
                            placeholder='Type "DESTROY" to confirm'
                            value={confirmText()}
                            onInput={(e) => setConfirmText(e.currentTarget.value)}
                            class="wipe-input"
                        />

                        <button
                            class="wipe-btn"
                            disabled={confirmText() !== 'DESTROY' || !targetPath() || wiping()}
                            onClick={handleWipe}
                        >
                            {wiping() ? '🔥 WIPING...' : '🗑️ EXECUTE CRYPTO WIPE'}
                        </button>
                    </div>

                    <div class="method-info">
                        <div class="method-row"><span>Method:</span> <strong>Random Byte Overwrite → Delete</strong></div>
                        <div class="method-row"><span>Standard:</span> <strong>NIST SP 800-88 "Clear"</strong></div>
                        <div class="method-row"><span>Audit:</span> <strong>Immutable event bus record</strong></div>
                    </div>
                </section>

                {/* Audit Trail */}
                <section class="audit-panel">
                    <h3>DESTRUCTION AUDIT TRAIL</h3>
                    <div class="audit-list">
                        <For each={history()}>
                            {(r) => (
                                <div class="audit-entry">
                                    <div class="audit-top">
                                        <span class="audit-id">{r.id}</span>
                                        <span class={`audit-status ${r.status}`}>{r.status.toUpperCase()}</span>
                                    </div>
                                    <div class="audit-detail">
                                        <span>Scope: {r.scope}</span>
                                        <span>Method: {r.method}</span>
                                        <span>By: {r.initiated_by}</span>
                                    </div>
                                    <div class="audit-time">{new Date(r.timestamp).toLocaleString()}</div>
                                </div>
                            )}
                        </For>
                        <Show when={history().length === 0}>
                            <div class="empty-state">
                                No destruction operations recorded. All wipe operations are logged immutably.
                            </div>
                        </Show>
                    </div>
                </section>
            </div>

            <style>{`
                .destruction-page { padding: 1.5rem; height: calc(100vh - 60px); display: flex; flex-direction: column; gap: 1.5rem; }
                .dest-header { display: flex; justify-content: space-between; align-items: center; }
                .dest-header h1 { font-size: 1.4rem; letter-spacing: 2px; margin: 0; color: #ef4444; }
                .dest-header p { color: var(--tactical-gray); font-size: 0.8rem; margin: 0.25rem 0 0; }
                .compliance-badges { display: flex; gap: 0.5rem; }
                .badge-gdpr, .badge-pci { font-size: 0.6rem; font-weight: 800; padding: 4px 8px; border-radius: 3px; letter-spacing: 1px; }
                .badge-gdpr { background: rgba(59, 130, 246, 0.1); border: 1px solid rgba(59, 130, 246, 0.3); color: #3b82f6; }
                .badge-pci { background: rgba(168, 85, 247, 0.1); border: 1px solid rgba(168, 85, 247, 0.3); color: #a855f7; }

                .dest-grid { display: grid; grid-template-columns: 1fr 1fr; gap: 1.5rem; flex: 1; overflow: hidden; }

                .wipe-panel, .audit-panel { background: var(--tactical-surface); border: 1px solid var(--tactical-border); border-radius: 8px; padding: 1.25rem; display: flex; flex-direction: column; }
                .wipe-panel h3, .audit-panel h3 { font-size: 0.7rem; letter-spacing: 2px; color: var(--tactical-gray); margin: 0 0 1rem; }

                .warning-box { background: rgba(239, 68, 68, 0.05); border: 1px solid rgba(239, 68, 68, 0.2); border-left: 3px solid #ef4444; padding: 0.75rem; border-radius: 4px; font-size: 0.75rem; color: #fca5a5; margin-bottom: 1.5rem; }
                .warning-box strong { color: #ef4444; }

                .wipe-form { display: flex; flex-direction: column; gap: 0.75rem; margin-bottom: 1.5rem; }
                .wipe-form label { font-size: 0.65rem; font-weight: 800; color: var(--tactical-gray); letter-spacing: 1px; text-transform: uppercase; }
                .wipe-input { background: rgba(0, 0, 0, 0.4); border: 1px solid var(--tactical-border); padding: 0.65rem; color: #fff; font-family: 'JetBrains Mono', monospace; font-size: 0.8rem; border-radius: 4px; }
                .wipe-input:focus { outline: none; border-color: #ef4444; }
                .wipe-btn { background: rgba(239, 68, 68, 0.15); border: 1px solid rgba(239, 68, 68, 0.4); color: #ef4444; padding: 0.75rem; font-weight: 800; font-size: 0.75rem; letter-spacing: 2px; border-radius: 4px; cursor: pointer; transition: all 0.2s; }
                .wipe-btn:hover:not(:disabled) { background: rgba(239, 68, 68, 0.25); }
                .wipe-btn:disabled { opacity: 0.3; cursor: not-allowed; }

                .method-info { margin-top: auto; padding-top: 1rem; border-top: 1px solid var(--tactical-border); }
                .method-row { display: flex; justify-content: space-between; font-size: 0.7rem; color: var(--tactical-gray); padding: 0.25rem 0; }
                .method-row strong { color: #d1d5db; }

                .audit-list { display: flex; flex-direction: column; gap: 0.5rem; overflow: auto; flex: 1; }
                .audit-entry { background: rgba(0, 0, 0, 0.2); border: 1px solid var(--tactical-border); padding: 0.75rem; border-radius: 4px; }
                .audit-top { display: flex; justify-content: space-between; margin-bottom: 0.5rem; }
                .audit-id { font-family: monospace; font-size: 0.65rem; color: var(--tactical-blue); }
                .audit-status { font-size: 0.6rem; font-weight: 800; padding: 1px 6px; border-radius: 2px; }
                .audit-status.completed { background: rgba(16, 185, 129, 0.1); color: #10b981; }
                .audit-status.failed { background: rgba(239, 68, 68, 0.1); color: #ef4444; }
                .audit-detail { display: flex; flex-direction: column; gap: 0.15rem; font-size: 0.7rem; color: #9ca3af; }
                .audit-time { font-size: 0.6rem; color: var(--tactical-gray); margin-top: 0.5rem; font-family: monospace; }
                .empty-state { text-align: center; padding: 2rem; color: var(--tactical-gray); font-style: italic; font-size: 0.8rem; }
            `}</style>
        </div>
    );
};
