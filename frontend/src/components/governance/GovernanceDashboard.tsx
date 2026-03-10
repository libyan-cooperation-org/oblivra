import { Component, createSignal, For, Show } from 'solid-js';
import '../../styles/governance.css';

// Observability & Governance — System Self-Monitoring + Data Governance UI
export const GovernanceDashboard: Component = () => {
    const [activeTab, setActiveTab] = createSignal<'observe' | 'retention' | 'legal' | 'disaster'>('observe');

    // Metrics remain the same
    const [obsMetrics] = createSignal({
        goroutines: 247,
        goroutine_peak: 512,
        goroutine_limit: 1000,
        heap_alloc_mb: 142.3,
        heap_sys_mb: 256.0,
        gc_count: 1847,
        go_version: 'go1.24.0',
        pprof_addr: '127.0.0.1:6060',
        mode: 'normal',
    });

    const [policies] = createSignal([
        { data_type: 'audit_logs', hot_days: 90, cold_days: 2190, auto_purge: false, legal_hold: false },
        { data_type: 'security_events', hot_days: 365, cold_days: 1825, auto_purge: true, legal_hold: false },
        { data_type: 'sessions', hot_days: 180, cold_days: 730, auto_purge: true, legal_hold: false },
        { data_type: 'evidence', hot_days: 3650, cold_days: 3650, auto_purge: false, legal_hold: true },
    ]);

    const [legalHolds] = createSignal([
        { id: 'hold-001', data_type: 'audit_logs', start_date: '2024-01-01', end_date: '2024-06-30', reason: 'Regulatory investigation', applied_by: 'legal@oblivra.io', active: true },
    ]);

    const [snapshots] = createSignal([
        { id: 'snapshot-20240301-020000', created_at: '2024-03-01 02:00', size_bytes: 52428800, sha256: 'a1b2c3d4...', encrypted: true },
        { id: 'snapshot-20240228-020000', created_at: '2024-02-28 02:00', size_bytes: 48576000, sha256: 'e5f6g7h8...', encrypted: true },
    ]);

    return (
        <div class="governance-dashboard">
            <header class="governance-header">
                <div>
                    <h2 class="host-name">GOVERNANCE & OBSERVABILITY</h2>
                    <p class="host-id">SYSTEM_SELF_MONITORING // DATA_LIFECYCLE_POLICY</p>
                </div>
                <div class={`status-pill ${obsMetrics().mode === 'normal' ? 'online' : (obsMetrics().mode === 'read_only' ? 'offline' : 'degraded')}`}>
                    SYSTEM_MODE: {obsMetrics().mode.toUpperCase()}
                </div>
            </header>

            <div class="governance-tabs">
                <button
                    class={`governance-tab-btn ${activeTab() === 'observe' ? 'active' : ''}`}
                    onClick={() => setActiveTab('observe')}
                >
                    SELF_MONITOR
                </button>
                <button
                    class={`governance-tab-btn ${activeTab() === 'retention' ? 'active' : ''}`}
                    onClick={() => setActiveTab('retention')}
                >
                    RETENTION_POLICIES
                </button>
                <button
                    class={`governance-tab-btn ${activeTab() === 'legal' ? 'active' : ''}`}
                    onClick={() => setActiveTab('legal')}
                >
                    LEGAL_HOLDS
                </button>
                <button
                    class={`governance-tab-btn ${activeTab() === 'disaster' ? 'active' : ''}`}
                    onClick={() => setActiveTab('disaster')}
                >
                    DISASTER_RECOVERY
                </button>
            </div>

            <main class="metrics-section">
                {/* Self-Monitor Tab */}
                <Show when={activeTab() === 'observe'}>
                    <div class="stat-grid">
                        <div class="stat-card">
                            <div class="stat-label">Goroutines</div>
                            <div class={`stat-value ${obsMetrics().goroutines > 500 ? 'accent' : ''}`}>
                                {obsMetrics().goroutines}
                            </div>
                            <div class="host-id">PEAK: {obsMetrics().goroutine_peak} / LIMIT: {obsMetrics().goroutine_limit}</div>
                        </div>
                        <div class="stat-card">
                            <div class="stat-label">Heap Memory</div>
                            <div class={`stat-value ${obsMetrics().heap_alloc_mb > 512 ? 'accent' : ''}`}>
                                {obsMetrics().heap_alloc_mb.toFixed(0)} MB
                            </div>
                            <div class="host-id">SYSTEM: {obsMetrics().heap_sys_mb.toFixed(0)} MB</div>
                        </div>
                        <div class="stat-card">
                            <div class="stat-label">GC Runs</div>
                            <div class="stat-value">{obsMetrics().gc_count}</div>
                            <div class="host-id">TOTAL_RUNTIME_COLLECTIONS</div>
                        </div>
                        <div class="stat-card">
                            <div class="stat-label">Runtime Engine</div>
                            <div class="stat-value" style="font-size: 18px;">{obsMetrics().go_version}</div>
                            <div class="host-id">PPROF_ADDR: {obsMetrics().pprof_addr}</div>
                        </div>
                    </div>

                    <div class="governance-table-container">
                        <header class="alert-header">
                            <div class="stat-label">PROFILING_ENDPOINTS</div>
                        </header>
                        <table class="governance-table">
                            <tbody>
                                {['CPU Profile', 'Memory Profile', 'Goroutine Dump', 'Block Profile', 'Mutex Profile', 'Trace'].map(ep => (
                                    <tr>
                                        <td>{ep.toUpperCase().replace(' ', '_')}</td>
                                        <td style="text-align: right;">
                                            <span class="status-pill online">AVAILABLE</span>
                                        </td>
                                    </tr>
                                ))}
                            </tbody>
                        </table>
                    </div>
                </Show>

                {/* Retention Policies Tab */}
                <Show when={activeTab() === 'retention'}>
                    <div class="governance-table-container">
                        <table class="governance-table">
                            <thead>
                                <tr>
                                    <th>DATA_TYPE</th>
                                    <th>HOT_RETENTION</th>
                                    <th>COLD_RETENTION</th>
                                    <th>AUTO_PURGE</th>
                                    <th>LEGAL_HOLD</th>
                                </tr>
                            </thead>
                            <tbody>
                                <For each={policies()}>
                                    {(p) => (
                                        <tr>
                                            <td style="font-weight: 700;">{p.data_type.toUpperCase()}</td>
                                            <td>{p.hot_days} DAYS</td>
                                            <td>{p.cold_days} DAYS ({(p.cold_days / 365).toFixed(1)}Y)</td>
                                            <td>
                                                <span class={`status-pill ${p.auto_purge ? 'online' : 'offline'}`}>{p.auto_purge ? 'ON' : 'OFF'}</span>
                                            </td>
                                            <td>
                                                <Show when={p.legal_hold} fallback="—">
                                                    <span class="status-pill degraded">HELD</span>
                                                </Show>
                                            </td>
                                        </tr>
                                    )}
                                </For>
                            </tbody>
                        </table>
                    </div>
                </Show>

                {/* Legal Holds Tab */}
                <Show when={activeTab() === 'legal'}>
                    <div style="display: grid; gap: 16px;">
                        <For each={legalHolds()}>
                            {(hold) => (
                                <div class="legal-hold-card">
                                    <div class="alert-card-header">
                                        <div class="alert-subject">{hold.data_type.toUpperCase()} (ID: {hold.id})</div>
                                        <span class={`status-pill ${hold.active ? 'degraded' : 'online'}`}>
                                            {hold.active ? 'ACTIVE_HOLD' : 'RELEASED'}
                                        </span>
                                    </div>
                                    <div class="alert-output">
                                        PERIOD: {hold.start_date} — {hold.end_date}<br />
                                        REASON: {hold.reason.toUpperCase()}<br />
                                        OFFICER: {hold.applied_by}
                                    </div>
                                </div>
                            )}
                        </For>
                        <button class="fb-icon-btn" style="width: auto; height: 40px; border-style: dashed;">
                            + INITIATE_NEW_LEGAL_HOLD
                        </button>
                    </div>
                </Show>

                {/* Disaster Recovery Tab */}
                <Show when={activeTab() === 'disaster'}>
                    <div style="display: grid; gap: 24px;">
                        <div style="display: grid; grid-template-columns: repeat(3, 1fr); gap: 1px; background: var(--glass-border); border: 1px solid var(--glass-border);">
                            {[
                                { mode: 'normal', label: 'NORMAL_OPERATION', desc: 'SYSTEM_RW_ACCESS_ENABLED' },
                                { mode: 'air_gap', label: 'AIR_GAP_MODE', desc: 'NETWORK_EGRESS_DISABLED' },
                                { mode: 'read_only', label: 'KILL_SWITCH', desc: 'READ_ONLY_FORENSICS_ONLY' },
                            ].map(m => (
                                <div
                                    class="stat-card"
                                    style={obsMetrics().mode === m.mode ? 'background: var(--bg-hover); border: 1px solid var(--accent-primary);' : ''}
                                >
                                    <div class="stat-label">{m.label}</div>
                                    <div class="host-id">{m.desc}</div>
                                    <Show when={obsMetrics().mode === m.mode}>
                                        <div class="status-pill online" style="margin-top: 12px; display: inline-block;">ACTIVE_SET</div>
                                    </Show>
                                </div>
                            ))}
                        </div>

                        <div class="governance-table-container">
                            <header class="alert-header">
                                <div class="stat-label">ENCRYPTED_SNAPSHOTS</div>
                                <button class="fb-icon-btn" style="width: auto; padding: 0 12px;">CREATE_SNAPSHOT</button>
                            </header>
                            <table class="governance-table">
                                <thead>
                                    <tr>
                                        <th>SNAPSHOT_ID</th>
                                        <th>CREATED_AT</th>
                                        <th>SIZE</th>
                                        <th>SHA256</th>
                                    </tr>
                                </thead>
                                <tbody>
                                    <For each={snapshots()}>
                                        {(snap) => (
                                            <tr>
                                                <td>{snap.id}</td>
                                                <td>{snap.created_at}</td>
                                                <td>{(snap.size_bytes / 1024 / 1024).toFixed(1)} MB</td>
                                                <td class="host-id" style="font-size: 9px;">{snap.sha256}</td>
                                            </tr>
                                        )}
                                    </For>
                                </tbody>
                            </table>
                        </div>
                    </div>
                </Show>
            </main>
        </div>
    );
};
