import { Component, createSignal, onMount, Show } from 'solid-js';
import * as DisasterService from '../../wailsjs/go/app/DisasterService';

// ── Helpers ──────────────────────────────────────────────────────────────────
const modeColor = (mode: string | undefined) => {
    switch (mode) {
        case 'read_only': return 'var(--alert-critical)';
        case 'air_gap':   return 'var(--alert-medium)';
        default:          return 'var(--alert-low)';
    }
};

const modeLabel = (mode: string | undefined) =>
    mode ? mode.replace('_', '-').toUpperCase() : 'NOMINAL';

// ── Shared card wrapper ───────────────────────────────────────────────────────
const Card: Component<{ accent?: string; children: any }> = (props) => (
    <section style={{
        background: 'var(--surface-1)',
        border: '1px solid var(--border-primary)',
        'border-top': `2px solid ${props.accent ?? 'var(--border-primary)'}`,
        padding: '24px',
        display: 'flex',
        'flex-direction': 'column',
        gap: '16px',
    }}>
        {props.children}
    </section>
);

// ── Section label ─────────────────────────────────────────────────────────────
const SectionLabel: Component<{ children: any }> = (props) => (
    <div style={{
        'font-family': 'var(--font-mono)',
        'font-size': '10px',
        'font-weight': 800,
        color: 'var(--text-muted)',
        'text-transform': 'uppercase',
        'letter-spacing': '1.5px',
    }}>
        {props.children}
    </div>
);

// ── Action button ─────────────────────────────────────────────────────────────
const ActionRow: Component<{
    label: string;
    sub: string;
    active?: boolean;
    danger?: boolean;
    onClick: () => void;
}> = (props) => {
    const borderColor = props.active
        ? (props.danger ? 'var(--alert-critical)' : 'var(--alert-medium)')
        : 'var(--border-primary)';
    const bg = props.active
        ? (props.danger ? 'rgba(220,38,38,0.06)' : 'rgba(245,158,11,0.06)')
        : 'transparent';

    return (
        <button
            onClick={props.onClick}
            style={{
                display: 'flex',
                'flex-direction': 'column',
                gap: '4px',
                'text-align': 'left',
                background: bg,
                border: `1px solid ${borderColor}`,
                'border-left': `3px solid ${borderColor}`,
                padding: '14px 16px',
                cursor: 'pointer',
                width: '100%',
            }}
        >
            <span style={{
                'font-family': 'var(--font-mono)',
                'font-size': '11px',
                'font-weight': 800,
                color: props.active
                    ? (props.danger ? 'var(--alert-critical)' : 'var(--alert-medium)')
                    : 'var(--text-primary)',
                'text-transform': 'uppercase',
                'letter-spacing': '0.5px',
            }}>
                {props.label}
            </span>
            <span style={{
                'font-family': 'var(--font-ui)',
                'font-size': '11px',
                color: 'var(--text-muted)',
            }}>
                {props.sub}
            </span>
        </button>
    );
};

// ── WarMode page ──────────────────────────────────────────────────────────────
export const WarMode: Component = () => {
    const [status, setStatus]       = createSignal<any>(null);
    const [loading, setLoading]     = createSignal(true);
    const [passphrase, setPassphrase] = createSignal('');
    const [exporting, setExporting] = createSignal(false);
    const [lastExport, setLastExport] = createSignal<string | null>(null);
    const [exportErr, setExportErr] = createSignal('');

    const refresh = async () => {
        try {
            const mode = await DisasterService.GetMode();
            setStatus({ mode });
        } catch { /* service may not be ready yet */ }
        finally { setLoading(false); }
    };

    onMount(() => {
        refresh();
        const t = setInterval(refresh, 5_000);
        return () => clearInterval(t);
    });

    const handleExport = async () => {
        if (!passphrase()) { setExportErr('Passphrase required'); return; }
        setExporting(true);
        setExportErr('');
        try {
            const path = await DisasterService.ExportResilienceBundle(passphrase());
            setLastExport(path);
        } catch (err: any) {
            setExportErr(err?.message ?? String(err));
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
        refresh();
    };

    const toggleKillSwitch = async () => {
        if (status()?.mode === 'read_only') {
            await DisasterService.DeactivateKillSwitch();
        } else {
            const reason = prompt('KILL-SWITCH ACTIVATION — State reason for audit log:');
            if (reason) await DisasterService.ActivateKillSwitch(reason);
        }
        refresh();
    };

    const mode = () => status()?.mode as string | undefined;

    return (
        <div style={{
            display: 'flex',
            'flex-direction': 'column',
            height: '100%',
            overflow: 'auto',
            padding: '28px 32px',
            background: 'var(--surface-0)',
        }}>
            {/* ── Header ── */}
            <div style={{
                display: 'flex',
                'justify-content': 'space-between',
                'align-items': 'flex-end',
                'margin-bottom': '24px',
                'padding-bottom': '16px',
                'border-bottom': '1px solid var(--border-primary)',
            }}>
                <div>
                    <div style={{
                        'font-family': 'var(--font-mono)',
                        'font-size': '10px',
                        'font-weight': 700,
                        color: 'var(--text-muted)',
                        'text-transform': 'uppercase',
                        'letter-spacing': '2px',
                        'margin-bottom': '4px',
                    }}>
                        SOVEREIGN RESILIENCE
                    </div>
                    <h1 style={{
                        'font-family': 'var(--font-mono)',
                        'font-size': '20px',
                        'font-weight': 800,
                        color: 'var(--text-primary)',
                        margin: 0,
                        'letter-spacing': '-0.5px',
                    }}>
                        War-Mode & Emergency Isolation
                    </h1>
                </div>

                {/* Live mode badge */}
                <div style={{
                    display: 'flex',
                    'align-items': 'center',
                    gap: '8px',
                    border: `1px solid ${modeColor(mode())}`,
                    padding: '6px 16px',
                    'font-family': 'var(--font-mono)',
                    'font-size': '11px',
                    'font-weight': 800,
                    color: modeColor(mode()),
                    'text-transform': 'uppercase',
                    'letter-spacing': '1px',
                }}>
                    <div style={{
                        width: '6px',
                        height: '6px',
                        background: modeColor(mode()),
                        animation: mode() && mode() !== 'nominal' ? 'wr-pulse 1.2s infinite' : 'none',
                    }} />
                    {loading() ? 'CHECKING...' : modeLabel(mode())}
                </div>
            </div>

            {/* ── Grid ── */}
            <div style={{
                display: 'grid',
                'grid-template-columns': 'repeat(auto-fit, minmax(360px, 1fr))',
                gap: '16px',
            }}>
                {/* NODE ISOLATION */}
                <Card accent={mode() === 'read_only' || mode() === 'air_gap' ? 'var(--alert-critical)' : 'var(--border-primary)'}>
                    <SectionLabel>Node Isolation</SectionLabel>
                    <p style={{ 'font-size': '11px', color: 'var(--text-muted)', margin: 0, 'font-family': 'var(--font-ui)' }}>
                        Sever network ties or enter forensic read-only mode.
                        Both actions are logged to the audit ledger.
                    </p>
                    <div style={{ display: 'flex', 'flex-direction': 'column', gap: '8px' }}>
                        <ActionRow
                            label={mode() === 'air_gap' ? 'Restore Network' : 'Activate Air-Gap'}
                            sub={mode() === 'air_gap' ? 'Re-enable outbound traffic' : 'Kill all outbound network activity immediately'}
                            active={mode() === 'air_gap'}
                            onClick={toggleAirGap}
                        />
                        <ActionRow
                            label={mode() === 'read_only' ? 'Release Kill-Switch' : 'Activate Kill-Switch'}
                            sub={mode() === 'read_only' ? 'Restore read-write ingestion' : 'Freeze all event ingestion — forensic mode only'}
                            active={mode() === 'read_only'}
                            danger
                            onClick={toggleKillSwitch}
                        />
                    </div>
                </Card>

                {/* DEAD-DROP REPLICATION */}
                <Card>
                    <SectionLabel>Dead-Drop Replication</SectionLabel>
                    <p style={{ 'font-size': '11px', color: 'var(--text-muted)', margin: 0, 'font-family': 'var(--font-ui)' }}>
                        Export encrypted state bundle for physical transport to air-gapped clones.
                    </p>
                    <div style={{ display: 'flex', 'flex-direction': 'column', gap: '6px' }}>
                        <label style={{
                            'font-family': 'var(--font-mono)',
                            'font-size': '9px',
                            'font-weight': 800,
                            color: 'var(--text-muted)',
                            'text-transform': 'uppercase',
                            'letter-spacing': '1px',
                        }}>
                            Encryption Passphrase
                        </label>
                        <input
                            type="password"
                            placeholder="Enter passphrase..."
                            value={passphrase()}
                            onInput={(e) => setPassphrase(e.currentTarget.value)}
                            style={{
                                background: 'var(--surface-0)',
                                border: '1px solid var(--border-primary)',
                                color: 'var(--text-primary)',
                                'font-family': 'var(--font-mono)',
                                'font-size': '12px',
                                padding: '8px 12px',
                                outline: 'none',
                                width: '100%',
                            }}
                        />
                    </div>

                    <Show when={exportErr()}>
                        <div style={{
                            'font-family': 'var(--font-mono)',
                            'font-size': '10px',
                            color: 'var(--alert-critical)',
                            padding: '6px 10px',
                            border: '1px solid var(--alert-critical)',
                        }}>
                            {exportErr()}
                        </div>
                    </Show>

                    <button
                        onClick={handleExport}
                        disabled={exporting()}
                        style={{
                            background: exporting() ? 'var(--surface-2)' : 'var(--accent-primary)',
                            border: 'none',
                            color: exporting() ? 'var(--text-muted)' : 'var(--surface-0)',
                            'font-family': 'var(--font-mono)',
                            'font-size': '11px',
                            'font-weight': 800,
                            'text-transform': 'uppercase',
                            'letter-spacing': '1px',
                            padding: '10px',
                            cursor: exporting() ? 'wait' : 'pointer',
                            width: '100%',
                        }}
                    >
                        {exporting() ? 'EXPORTING...' : 'GENERATE RESILIENCE BUNDLE'}
                    </button>

                    <Show when={lastExport()}>
                        <div style={{
                            padding: '10px 12px',
                            background: 'rgba(132,204,22,0.05)',
                            border: '1px solid var(--alert-low)',
                            display: 'flex',
                            'flex-direction': 'column',
                            gap: '4px',
                        }}>
                            <span style={{
                                'font-family': 'var(--font-mono)',
                                'font-size': '9px',
                                'font-weight': 800,
                                color: 'var(--alert-low)',
                                'text-transform': 'uppercase',
                                'letter-spacing': '1px',
                            }}>
                                LAST EXPORT
                            </span>
                            <span style={{
                                'font-family': 'var(--font-mono)',
                                'font-size': '11px',
                                color: 'var(--text-secondary)',
                                'word-break': 'break-all',
                            }}>
                                {lastExport()}
                            </span>
                        </div>
                    </Show>
                </Card>

                {/* OFFLINE UPDATE */}
                <Card>
                    <SectionLabel>Offline Update</SectionLabel>
                    <p style={{ 'font-size': '11px', color: 'var(--text-muted)', margin: 0, 'font-family': 'var(--font-ui)' }}>
                        Apply signed update bundles from physical media.
                        No internet required. Signature verified before apply.
                    </p>
                    <div style={{
                        border: '1px dashed var(--border-primary)',
                        padding: '32px 16px',
                        display: 'flex',
                        'flex-direction': 'column',
                        'align-items': 'center',
                        gap: '12px',
                        color: 'var(--text-muted)',
                    }}>
                        <div style={{
                            'font-family': 'var(--font-mono)',
                            'font-size': '11px',
                            'text-align': 'center',
                        }}>
                            DROP UPDATE BUNDLE (.vbx)
                        </div>
                        <button style={{
                            background: 'transparent',
                            border: '1px solid var(--border-primary)',
                            color: 'var(--text-secondary)',
                            'font-family': 'var(--font-mono)',
                            'font-size': '10px',
                            'font-weight': 700,
                            'text-transform': 'uppercase',
                            'letter-spacing': '0.5px',
                            padding: '6px 16px',
                            cursor: 'pointer',
                        }}>
                            SELECT FILE
                        </button>
                    </div>
                    <div style={{
                        display: 'flex',
                        'justify-content': 'space-between',
                        'font-family': 'var(--font-mono)',
                        'font-size': '10px',
                        color: 'var(--text-muted)',
                        'text-transform': 'uppercase',
                    }}>
                        <span>
                            INTEGRITY:{' '}
                            <span style={{ color: 'var(--alert-low)', 'font-weight': 800 }}>VERIFIED</span>
                        </span>
                        <span>CHANNEL: OFFLINE_ONLY</span>
                    </div>
                </Card>
            </div>

            <style>{`
                @keyframes wr-pulse {
                    0%, 100% { opacity: 1; }
                    50% { opacity: 0.2; }
                }
            `}</style>
        </div>
    );
};
