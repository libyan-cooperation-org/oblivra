/**
 * PanelLauncher v3
 *
 * Fixes:
 *  - All panel imports are now lazy (no eager top-level imports)
 *  - Fixed import paths: SIEMPanel, FleetDashboard, CommandCenter
 *  - Removed dead `lazyImport` variable (was never called)
 *  - panelCount is now reactive (reads signal in JSX, not at init time)
 *  - snapshots list is wrapped in a reactive signal so saves reflect immediately
 */

import { Component, For, createSignal, Show } from 'solid-js';
import { usePanelManager, cascadePos } from './PanelManager';

// ── Lazy panel helper ─────────────────────────────────────────────────────────
// Returns a content factory: a function that, when called inside a component,
// kicks off the dynamic import and renders the result.
function lazy(importFn: () => Promise<any>, exportKey: string) {
    return () => {
        const [Comp, setComp] = createSignal<any>(null);
        importFn()
            .then(m => setComp(() => m[exportKey] ?? m.default ?? Object.values(m)[0]))
            .catch(err => console.error('[PanelLauncher] lazy import failed:', err));
        return <Show when={Comp()}>{(C) => <C />}</Show>;
    };
}

interface LauncherEntry {
    id: string;
    label: string;
    icon: string;
    kbd?: string;
    spawn: () => void;
}

export const PanelLauncher: Component = () => {
    const { openPanel, isPanelOpen, openPanelCount, saveSnapshot, listSnapshots, loadSnapshot } = usePanelManager();
    const [showSnaps, setShowSnaps] = createSignal(false);
    const [snapName,  setSnapName]  = createSignal('');
    // Reactive snapshot list — updated whenever we save, so <For> re-renders
    const [snapList, setSnapList] = createSignal<string[]>(listSnapshots());

    const refreshSnaps = () => setSnapList(listSnapshots());

    const mk = (
        id: string, title: string, icon: string,
        content: () => any,
        size = { w: 960, h: 620 },
        kbd?: string,
    ): LauncherEntry => ({
        id, label: title, icon, kbd,
        spawn: () => openPanel({ id, title, icon, defaultPos: cascadePos({ x: 100, y: 80 }), defaultSize: size, content }),
    });

    const entries: LauncherEntry[] = [
        mk('terminal',  'Terminal',      '🖥️', lazy(() => import('../terminal/TerminalLayout'),    'TerminalLayout'), { w: 920,  h: 560 }, 'T'),
        mk('dashboard', 'Dashboard',     '📊', lazy(() => import('../dashboard/Dashboard'),         'Dashboard'),      { w: 1100, h: 700 }, 'D'),
        mk('siem',      'SIEM',          '🛡️', lazy(() => import('../siem/SIEMPanel'),              'SIEMPanel'),      { w: 1200, h: 750 }, 'S'),
        mk('fleet',     'Fleet',         '🖧', lazy(() => import('../fleet/FleetDashboard'),         'FleetDashboard'), { w: 1100, h: 680 }, 'F'),
        mk('incidents', 'Incidents',     '🚨', lazy(() => import('../incident/CommandCenter'),       'CommandCenter'),  { w: 1000, h: 650 }, 'I'),
        mk('soc',       'SOC Workspace', '🗖', lazy(() => import('../soc/SOCWorkspace'),             'SOCWorkspace'),   { w: 1400, h: 900 }),
    ];

    return (
        <div style={{ display: 'flex', 'align-items': 'center', gap: '2px', position: 'relative', 'max-width': '480px', 'overflow-x': 'auto' }}>
            {/* Count badge — reactive: reads signal in JSX */}
            <Show when={openPanelCount() > 0}>
                <span style={{
                    'font-family': 'var(--font-mono)', 'font-size': '9px', 'font-weight': 800,
                    color: 'var(--accent-primary)', background: 'rgba(87,139,255,0.12)',
                    'border-radius': '3px', padding: '1px 5px', 'margin-right': '4px',
                    'flex-shrink': 0,
                }}>
                    {openPanelCount()} open
                </span>
            </Show>

            <For each={entries}>
                {(entry) => (
                    <button
                        title={`${entry.label}${entry.kbd ? `  (Ctrl+Shift+${entry.kbd})` : ''}`}
                        onClick={entry.spawn}
                        style={{
                            background: isPanelOpen(entry.id) ? 'rgba(87,139,255,0.12)' : 'transparent',
                            border: `1px solid ${isPanelOpen(entry.id) ? 'var(--accent-primary)' : 'var(--border-primary)'}`,
                            'border-radius': '3px', padding: '3px 7px', cursor: 'pointer',
                            display: 'flex', 'align-items': 'center', gap: '4px', 'flex-shrink': 0,
                            'font-family': 'var(--font-mono)', 'font-size': '9px', 'font-weight': 700,
                            'text-transform': 'uppercase', 'letter-spacing': '0.5px',
                            color: isPanelOpen(entry.id) ? 'var(--accent-primary)' : 'var(--text-muted)',
                            transition: 'all 0.12s', '-webkit-app-region': 'no-drag',
                        }}
                        onMouseOver={(e) => {
                            if (!isPanelOpen(entry.id)) {
                                (e.currentTarget as HTMLElement).style.background = 'var(--surface-2)';
                                (e.currentTarget as HTMLElement).style.color = 'var(--text-primary)';
                            }
                        }}
                        onMouseOut={(e) => {
                            if (!isPanelOpen(entry.id)) {
                                (e.currentTarget as HTMLElement).style.background = 'transparent';
                                (e.currentTarget as HTMLElement).style.color = 'var(--text-muted)';
                            }
                        }}
                    >
                        <span>{entry.icon}</span>
                        <span>{entry.label}</span>
                    </button>
                )}
            </For>

            {/* Snapshot button */}
            <button
                title="Save / restore panel workspace snapshots"
                onClick={() => { refreshSnaps(); setShowSnaps(v => !v); }}
                style={{
                    background: showSnaps() ? 'var(--surface-2)' : 'transparent',
                    border: '1px solid var(--border-primary)', 'border-radius': '3px',
                    padding: '3px 7px', cursor: 'pointer', 'font-size': '11px',
                    color: 'var(--text-muted)', 'margin-left': '2px', 'flex-shrink': 0,
                    '-webkit-app-region': 'no-drag',
                }}
            >💾</button>

            <Show when={showSnaps()}>
                <div style={{
                    position: 'absolute', top: '100%', right: 0,
                    background: 'var(--surface-2)', border: '1px solid var(--border-secondary)',
                    'border-radius': '5px', padding: '8px', 'min-width': '220px',
                    'box-shadow': 'var(--shadow-lg)', 'z-index': 9999, 'margin-top': '4px',
                }}>
                    <div style={{ 'font-family': 'var(--font-mono)', 'font-size': '9px', 'font-weight': 800, color: 'var(--text-muted)', 'text-transform': 'uppercase', 'letter-spacing': '1px', 'margin-bottom': '8px' }}>
                        Workspace Snapshots
                    </div>

                    {/* Save row */}
                    <div style={{ display: 'flex', gap: '4px', 'margin-bottom': '8px' }}>
                        <input
                            placeholder="Snapshot name…"
                            value={snapName()}
                            onInput={(e) => setSnapName(e.currentTarget.value)}
                            onKeyDown={(e) => {
                                if (e.key === 'Enter' && snapName()) {
                                    saveSnapshot(snapName());
                                    setSnapName('');
                                    refreshSnaps();
                                }
                            }}
                            style={{
                                flex: 1, background: 'var(--surface-3)', border: '1px solid var(--border-primary)',
                                'border-radius': '3px', padding: '4px 8px', color: 'var(--text-primary)',
                                'font-family': 'var(--font-mono)', 'font-size': '11px', outline: 'none',
                            }}
                        />
                        <button
                            onClick={() => {
                                if (snapName()) {
                                    saveSnapshot(snapName());
                                    setSnapName('');
                                    refreshSnaps();
                                }
                            }}
                            style={{
                                background: 'var(--accent-primary)', border: 'none', 'border-radius': '3px',
                                padding: '4px 10px', cursor: 'pointer', color: '#000',
                                'font-family': 'var(--font-mono)', 'font-size': '10px', 'font-weight': 800,
                            }}
                        >Save</button>
                    </div>

                    {/* Saved list — reads reactive snapList signal */}
                    <For each={snapList()}>
                        {(name) => (
                            <div style={{ display: 'flex', 'align-items': 'center', gap: '4px', 'margin-bottom': '4px' }}>
                                <button
                                    onClick={() => { loadSnapshot(name); setShowSnaps(false); }}
                                    style={{
                                        flex: 1, background: 'var(--surface-3)', border: '1px solid var(--border-primary)',
                                        'border-radius': '3px', padding: '5px 8px', cursor: 'pointer',
                                        'font-family': 'var(--font-mono)', 'font-size': '11px', color: 'var(--text-secondary)',
                                        'text-align': 'left',
                                    }}
                                >▶ {name}</button>
                            </div>
                        )}
                    </For>
                    <Show when={snapList().length === 0}>
                        <div style={{ 'font-size': '10px', color: 'var(--text-muted)', 'font-family': 'var(--font-mono)' }}>No snapshots yet</div>
                    </Show>
                </div>

                {/* Click-away */}
                <div style={{ position: 'fixed', inset: 0, 'z-index': 9998 }} onPointerDown={() => setShowSnaps(false)} />
            </Show>
        </div>
    );
};
