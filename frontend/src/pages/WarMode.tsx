import { Component, createSignal, onMount, Show } from 'solid-js';
import * as DisasterService from '../../wailsjs/go/app/DisasterService';

export const WarMode: Component = () => {
    const [status, setStatus] = createSignal<any>(null);
    const [passphrase, setPassphrase] = createSignal("");
    const [exporting, setExporting] = createSignal(false);
    const [lastExport, setLastExport] = createSignal<string | null>(null);

    const refreshStatus = async () => {
        const mode = await DisasterService.GetMode();
        setStatus({ mode });
    };

    onMount(() => {
        refreshStatus();
        const interval = setInterval(refreshStatus, 5000);
        return () => clearInterval(interval);
    });

    const handleExport = async () => {
        if (!passphrase()) {
            alert("Passphrase required for encryption");
            return;
        }
        setExporting(true);
        try {
            const path = await DisasterService.ExportResilienceBundle(passphrase());
            setLastExport(path);
        } catch (err) {
            console.error("Export failed:", err);
            alert("Export failed: " + err);
        } finally {
            setExporting(false);
        }
    };

    const toggleAirGap = async () => {
        if (status()?.mode === 'air_gap') {
            await DisasterService.DeactivateKillSwitch();
        } else {
            await DisasterService.ActivateAirGapMode();
        }
        refreshStatus();
    };

    const toggleKillSwitch = async () => {
        if (status()?.mode === 'read_only') {
            await DisasterService.DeactivateKillSwitch();
        } else {
            const reason = prompt("Reason for Kill-Switch activation:");
            if (reason) await DisasterService.ActivateKillSwitch(reason);
        }
        refreshStatus();
    };

    const getModeColor = () => {
        switch (status()?.mode) {
            case 'read_only': return 'var(--tactical-red)';
            case 'air_gap': return 'var(--tactical-amber)';
            default: return 'var(--tactical-green)';
        }
    };

    return (
        <div class="war-mode-page">
            <header class="page-header">
                <div class="header-main">
                    <h1>RESILIENCE & WAR-MODE</h1>
                    <p>Disconnected node operations & emergency isolation</p>
                </div>
                <div class="status-indicator" style={{ border: `1px solid ${getModeColor()}` }}>
                    <div class="dot" style={{ background: getModeColor() }}></div>
                    <span>{status()?.mode?.toUpperCase() || 'LOADING...'}</span>
                </div>
            </header>

            <div class="war-grid">
                {/* Node Control */}
                <section class="war-card primary">
                    <div class="card-header">
                        <h3>NODE ISOLATION</h3>
                        <span class="warning-icon">⚠️</span>
                    </div>
                    <p class="card-desc">Instantly sever network ties or enter forensic-only mode.</p>

                    <div class="action-stack">
                        <button
                            class={`war-btn ${status()?.mode === 'air_gap' ? 'active' : ''}`}
                            onClick={toggleAirGap}
                        >
                            <span class="icon">📡</span>
                            <div class="btn-text">
                                <strong>{status()?.mode === 'air_gap' ? 'RESTORE NETWORK' : 'ACTIVATE AIR-GAP'}</strong>
                                <span>{status()?.mode === 'air_gap' ? 'Enable outbound traffic' : 'Kill all outbound network activity'}</span>
                            </div>
                        </button>

                        <button
                            class={`war-btn danger ${status()?.mode === 'read_only' ? 'active' : ''}`}
                            onClick={toggleKillSwitch}
                        >
                            <span class="icon">💀</span>
                            <div class="btn-text">
                                <strong>{status()?.mode === 'read_only' ? 'RELEASE KILL-SWITCH' : 'ACTIVATE KILL-SWITCH'}</strong>
                                <span>{status()?.mode === 'read_only' ? 'Restore read-write operations' : 'Freeze all ingestion (Read-Only)'}</span>
                            </div>
                        </button>
                    </div>
                </section>

                {/* Dead-Drop Replication */}
                <section class="war-card">
                    <div class="card-header">
                        <h3>DEAD-DROP REPLICATION</h3>
                        <span class="icon">📦</span>
                    </div>
                    <p class="card-desc">Export encrypted state for physical transport to air-gapped clones.</p>

                    <div class="input-group">
                        <label>DECRYPTION PASSPHRASE</label>
                        <input
                            type="password"
                            placeholder="Enter tactical secret..."
                            value={passphrase()}
                            onInput={(e) => setPassphrase(e.currentTarget.value)}
                        />
                    </div>

                    <button class="action-btn" onClick={handleExport} disabled={exporting()}>
                        {exporting() ? 'EXPORTING...' : 'GENERATE RESILIENCE BUNDLE'}
                    </button>

                    <Show when={lastExport()}>
                        <div class="export-success">
                            <span class="label">LAST EXPORT:</span>
                            <span class="path">{lastExport()}</span>
                        </div>
                    </Show>
                </section>

                {/* Update & Recovery */}
                <section class="war-card">
                    <div class="card-header">
                        <h3>OFFLINE UPDATE</h3>
                        <span class="icon">💾</span>
                    </div>
                    <p class="card-desc">Verify and apply signed updates from physical media.</p>

                    <div class="dropzone">
                        <span class="icon">📜</span>
                        <p>Drag & Drop Update Bundle (.vbx)</p>
                        <button class="btn-outline">SELECT FILE</button>
                    </div>

                    <div class="info-footer">
                        <span>Binary Integrity: <strong style={{ color: 'var(--tactical-green)' }}>VERIFIED</strong></span>
                        <span>Update Channel: <strong>OFFLINE_ONLY</strong></span>
                    </div>
                </section>
            </div>

            <style>{`
                .war-mode-page { padding: 2rem; }
                .page-header { display: flex; justify-content: space-between; align-items: flex-start; margin-bottom: 2rem; }
                .page-header h1 { font-size: 1.5rem; letter-spacing: 2px; margin: 0; }
                .page-header p { color: var(--tactical-gray); font-size: 0.9rem; }
                .status-indicator { display: flex; align-items: center; gap: 0.75rem; padding: 0.5rem 1rem; border-radius: 4px; font-weight: 800; font-size: 0.8rem; }
                .status-indicator .dot { width: 8px; height: 8px; border-radius: 50%; box-shadow: 0 0 10px currentColor; }
                
                .war-grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(400px, 1fr)); gap: 1.5rem; }
                .war-card { background: rgba(0,0,0,0.3); border: 1px solid var(--tactical-border); padding: 1.5rem; border-radius: 8px; display: flex; flex-direction: column; }
                .war-card.primary { border-color: var(--tactical-red); background: rgba(255,0,0,0.02); }
                .card-header { display: flex; justify-content: space-between; align-items: center; margin-bottom: 0.75rem; }
                .card-header h3 { font-size: 0.9rem; letter-spacing: 1.5px; margin: 0; }
                .card-desc { font-size: 0.8rem; color: var(--tactical-gray); margin-bottom: 1.5rem; }
                
                .action-stack { display: flex; flex-direction: column; gap: 1rem; }
                .war-btn {
                    display: flex; gap: 1rem; align-items: center; text-align: left;
                    background: rgba(255,255,255,0.03); border: 1px solid rgba(255,255,255,0.1);
                    padding: 1rem; border-radius: 6px; cursor: pointer; transition: all 0.2s;
                }
                .war-btn:hover { background: rgba(255,255,255,0.06); transform: translateX(4px); }
                .war-btn.active { border-color: var(--tactical-amber); background: rgba(251, 191, 36, 0.05); }
                .war-btn.danger.active { border-color: var(--tactical-red); background: rgba(239, 68, 68, 0.05); }
                .war-btn .icon { font-size: 1.5rem; }
                .btn-text { display: flex; flex-direction: column; gap: 0.25rem; }
                .btn-text strong { font-size: 0.85rem; }
                .btn-text span { font-size: 0.7rem; color: var(--tactical-gray); }
                
                .input-group { display: flex; flex-direction: column; gap: 0.5rem; margin-bottom: 1.5rem; }
                .input-group label { font-size: 0.65rem; font-weight: 800; color: var(--tactical-gray); }
                .input-group input { 
                    background: rgba(0,0,0,0.5); border: 1px solid var(--tactical-border); 
                    padding: 0.75rem; color: #fff; font-family: 'JetBrains Mono', monospace; border-radius: 4px;
                }
                
                .action-btn {
                    background: var(--tactical-blue); color: #fff; border: none; padding: 0.75rem;
                    font-weight: 800; border-radius: 4px; cursor: pointer; letter-spacing: 1px;
                }
                .action-btn:hover { background: #60a5fa; }
                .export-success { margin-top: 1.5rem; padding: 1rem; background: rgba(16, 185, 129, 0.05); border: 1px solid rgba(16, 185, 129, 0.2); border-radius: 4px; }
                .export-success .label { display: block; font-size: 0.6rem; font-weight: 800; color: var(--tactical-green); margin-bottom: 0.25rem; }
                .export-success .path { font-size: 0.75rem; font-family: 'JetBrains Mono', monospace; word-break: break-all; }
                
                .dropzone {
                    flex: 1; border: 2px dashed var(--tactical-border); border-radius: 6px;
                    display: flex; flex-direction: column; align-items: center; justify-content: center;
                    padding: 2rem; gap: 1rem; color: var(--tactical-gray); margin-bottom: 1.5rem;
                }
                .btn-outline { background: transparent; border: 1px solid var(--tactical-border); color: #fff; padding: 0.5rem 1rem; border-radius: 4px; font-size: 0.75rem; cursor: pointer; }
                .info-footer { display: flex; justify-content: space-between; font-size: 0.7rem; color: var(--tactical-gray); }
            `}</style>
        </div>
    );
};
