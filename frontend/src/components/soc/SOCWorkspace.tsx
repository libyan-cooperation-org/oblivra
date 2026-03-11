/**
 * SOCWorkspace v2
 *
 * Fixes:
 *  - GoldenLayout render() now uses createRoot for proper Solid cleanup (#23)
 *  - CSS variables instead of raw Tailwind gray-XXX (#21)
 *  - WindowFrame uses shared PanelTitleBar via new import (#22)
 *  - Minimised panel state persisted to localStorage (#13)
 *  - Layout preset save/load UI (#5)
 *  - Popout window CSS injection fixed — injects <style> text, not cloned nodes (#2)
 */

import {
    Component, onMount, onCleanup, createSignal,
    createResource, For, Show, createRoot,
} from 'solid-js';
import { render } from 'solid-js/web';
import { GoldenLayout, LayoutConfig } from 'golden-layout';
import 'golden-layout/dist/css/goldenlayout-base.css';
import 'golden-layout/dist/css/themes/goldenlayout-dark-theme.css';
import { TerminalView } from '../terminal/Terminal';
import { WindowFrame } from './WindowFrame';
import { AlertInvestigationPanel }   from './panels/AlertInvestigationPanel';
import { LogSearchPanel }             from './panels/LogSearchPanel';
import { TimelineInvestigationPanel } from './panels/TimelineInvestigationPanel';
import { MitreAttackPanel }           from './panels/MitreAttackPanel';
import { HardwareTrustPanel }         from './panels/HardwareTrustPanel';
import { ThreatGraphPanel }           from './panels/ThreatGraphPanel';
import { GetAllMetrics }    from '../../../wailsjs/go/app/MetricsService';
import { GetAllHealth }     from '../../../wailsjs/go/app/HealthService';
import {
    ActivateAirGapMode, ActivateKillSwitch,
    DeactivateKillSwitch, TriggerNuclearDestruction, GetMode,
} from '../../../wailsjs/go/app/DisasterService';

// ── Layout storage helpers ────────────────────────────────────────────────────
const LAYOUT_KEY   = 'sov-soc-layout';
const MINIMISE_KEY = 'sov-soc-minimised';

function saveLayout(layout: GoldenLayout) {
    try { localStorage.setItem(LAYOUT_KEY, JSON.stringify(layout.saveLayout())); } catch { /* ignore */ }
}
function loadLayout(): LayoutConfig | null {
    try { const r = localStorage.getItem(LAYOUT_KEY); return r ? JSON.parse(r) : null; } catch { return null; }
}
function savePreset(name: string, layout: GoldenLayout) {
    try { localStorage.setItem(`sov-soc-preset-${name}`, JSON.stringify(layout.saveLayout())); } catch { /* ignore */ }
}
function loadPreset(name: string): LayoutConfig | null {
    try { const r = localStorage.getItem(`sov-soc-preset-${name}`); return r ? JSON.parse(r) : null; } catch { return null; }
}
function listPresets(): string[] {
    try { return Object.keys(localStorage).filter(k => k.startsWith('sov-soc-preset-')).map(k => k.replace('sov-soc-preset-', '')); } catch { return []; }
}
function loadMinimised(): Array<{id: string; title: string; componentType: string}> {
    try { const r = localStorage.getItem(MINIMISE_KEY); return r ? JSON.parse(r) : []; } catch { return []; }
}
function saveMinimised(panels: Array<{id: string; title: string; componentType: string}>) {
    try { localStorage.setItem(MINIMISE_KEY, JSON.stringify(panels)); } catch { /* ignore */ }
}

// ── Default layout ────────────────────────────────────────────────────────────
const DEFAULT_LAYOUT: LayoutConfig = {
    settings: { showPopoutIcon: true, showMaximiseIcon: true, showCloseIcon: true },
    root: {
        type: 'row',
        content: [
            { type: 'column', width: 20, content: [
                { type: 'component', componentType: 'Alerts',   title: 'ALERT INVESTIGATION' },
                { type: 'component', componentType: 'Mitre',    title: 'MITRE ATT&CK' },
            ]},
            { type: 'column', width: 50, content: [
                { type: 'component', componentType: 'Terminal', title: 'SSH TERMINAL' },
                { type: 'component', componentType: 'Logs',     title: 'SIEM LOG SEARCH' },
            ]},
            { type: 'column', width: 30, content: [
                { type: 'component', componentType: 'Trust',    title: 'PLATFORM INTEGRITY' },
                { type: 'component', componentType: 'Graph',    title: 'THREAT GRAPH' },
                { type: 'component', componentType: 'Timeline', title: 'EVENT TIMELINE' },
            ]},
        ],
    },
};

// ── War-mode CSS ──────────────────────────────────────────────────────────────
const WAR_MODE_STYLE = `
.war-mode-active { background: #0d0303 !important; }
.war-mode-active .sov-titlebar { background: rgba(127,29,29,0.4) !important; border-color: rgba(127,29,29,0.6) !important; }
@keyframes war-aberration { 0%{transform:translate(0,0)} 25%{transform:translate(1px,-1px)} 50%{transform:translate(-1px,1px)} 75%{transform:translate(1px,1px)} 100%{transform:translate(0,0)} }
.war-mode-aberration { animation: war-aberration 0.18s infinite; }
`;

// ── Component ─────────────────────────────────────────────────────────────────
export const SOCWorkspace: Component = () => {
    let containerRef: HTMLDivElement | undefined;
    let layout: GoldenLayout | undefined;
    // Track Solid roots created inside GL containers so we can dispose them
    const solidRoots: Array<() => void> = [];

    const [warMode,      setWarMode]      = createSignal(false);
    const [presetName,   setPresetName]   = createSignal('');
    const [showPresets,  setShowPresets]  = createSignal(false);
    const [minimisedPanels, setMinimisedPanels] = createSignal(loadMinimised());

    const [disasterMode, { refetch: refetchMode }] = createResource(GetMode);

    const [metrics] = createResource(async () => {
        try {
            const [m, health] = await Promise.all([GetAllMetrics(), GetAllHealth()]);
            return {
                latency:   m.find((x: any) => x.Name === 'system.latency')?.value || 12,
                eps:       m.find((x: any) => x.Name === 'ingest.eps')?.value || 4204,
                nodes:     Object.keys(health).length || 1,
                integrity: Object.values(health).every((h: any) => h.CPU < 90) ? 'Optimal' : 'Degraded',
            };
        } catch { return { latency: 0, eps: 0, nodes: 0, integrity: 'Offline' }; }
    });

    // ── register a GL component safely using createRoot ─────────────────────
    const registerPanel = (name: string, title: string, status: string, jsx: () => any) => {
        layout?.registerComponentFactoryFunction(name, (container) => {
            const el = document.createElement('div');
            el.style.cssText = 'height:100%;width:100%;';
            container.getElement().appendChild(el);

            // Use createRoot so this Solid tree is properly tracked and disposable
            const dispose = createRoot((d) => {
                render(() => (
                    <WindowFrame
                        title={title}
                        status={status}
                        onClose={() => container.close()}
                        onMinimise={() => {
                            const panel = { id: `${name}-${Date.now()}`, title, componentType: name };
                            setMinimisedPanels(prev => {
                                const next = [...prev, panel];
                                saveMinimised(next);
                                return next;
                            });
                            container.close();
                        }}
                        onMaximise={() => (container as any).toggleMaximise?.()}
                        onPopout={() => (container as any).popout?.()}
                    >
                        {jsx()}
                    </WindowFrame>
                ), el);
                return d;
            });
            solidRoots.push(dispose);
        });
    };

    onMount(() => {
        if (!containerRef) return;

        const config = loadLayout() ?? DEFAULT_LAYOUT;
        layout = new GoldenLayout(containerRef);

        registerPanel('Alerts',   'Alert Investigation',   'CRITICAL',   () => <AlertInvestigationPanel />);
        registerPanel('Logs',     'SIEM Log Search',       'LIVE',       () => <LogSearchPanel />);
        registerPanel('Timeline', 'Event Timeline',        'ANALYTIC',   () => <TimelineInvestigationPanel />);
        registerPanel('Mitre',    'MITRE ATT&CK Matrix',   'COVERAGE',   () => <MitreAttackPanel />);
        registerPanel('Trust',    'Platform Integrity',    'ATTESTED',   () => <HardwareTrustPanel />);
        registerPanel('Graph',    'Multi-Hop Threat Graph','CORRELATED', () => <ThreatGraphPanel />);

        // Terminal panel
        layout.registerComponentFactoryFunction('Terminal', (container) => {
            const el = document.createElement('div');
            el.style.cssText = 'height:100%;width:100%;';
            container.getElement().appendChild(el);
            const dispose = createRoot((d) => {
                render(() => (
                    <WindowFrame
                        title="SSH Response Terminal"
                        status="CONNECTED"
                        onClose={() => container.close()}
                        onMinimise={() => {
                            const p = { id: 'Terminal', title: 'SSH Response Terminal', componentType: 'Terminal' };
                            setMinimisedPanels(prev => { const n = [...prev, p]; saveMinimised(n); return n; });
                            container.close();
                        }}
                        onMaximise={() => (container as any).toggleMaximise?.()}
                        onPopout={() => (container as any).popout?.()}
                    >
                        <TerminalView sessionId="soc-session" onData={() => {}} onResize={() => {}} />
                    </WindowFrame>
                ), el);
                return d;
            });
            solidRoots.push(dispose);
        });

        layout.loadLayout(config);

        layout.on('stateChanged', () => saveLayout(layout!));

        // ── Fix popout CSS injection (#2) ───────────────────────────────────
        layout.on('windowOpened', (newWin: any) => {
            const extWin = newWin.getGlWindow?.() ?? newWin;
            if (!extWin?.document) return;
            // Collect all inline <style> text from current document
            const allCss = Array.from(document.styleSheets).map(sheet => {
                try { return Array.from(sheet.cssRules).map(r => r.cssText).join('\n'); } catch { return ''; }
            }).join('\n');
            const style = extWin.document.createElement('style');
            style.textContent = allCss;
            extWin.document.head.appendChild(style);
            extWin.document.body.style.cssText = 'background:var(--surface-0,#111318);color:var(--text-primary,#e8eaf2);overflow:hidden;';
        });

        const onResize = () => (layout as any)?.updateSize?.();
        window.addEventListener('resize', onResize);
        onCleanup(() => {
            window.removeEventListener('resize', onResize);
            solidRoots.forEach(d => d());
            layout?.destroy();
        });
    });

    // ── Disaster controls ─────────────────────────────────────────────────────
    const handleNuclear = async () => {
        if (confirm('WARNING: NUCLEAR DESTRUCTION INITIATED.\nThis will cryptographically wipe the vault.\nPROCEED?')) {
            await TriggerNuclearDestruction();
            window.location.reload();
        }
    };
    const handleAirGap  = async () => { await ActivateAirGapMode();  refetchMode(); };
    const handleKillSwitch = async () => {
        disasterMode() === 'KILL_SWITCH' ? await DeactivateKillSwitch() : await ActivateKillSwitch('Manual tactical override');
        refetchMode();
    };

    const restorePanel = (panel: { id: string; title: string; componentType: string }) => {
        layout?.rootItem?.contentItems[0]?.addChild({
            type: 'component', title: panel.title,
            componentType: panel.componentType, componentState: {},
        } as any);
        setMinimisedPanels(prev => { const n = prev.filter(p => p.id !== panel.id); saveMinimised(n); return n; });
    };

    const resetLayout = () => {
        try { localStorage.removeItem(LAYOUT_KEY); } catch { /* ignore */ }
        window.location.reload();
    };

    return (
        <>
            <style>{WAR_MODE_STYLE}</style>

            <div
                class={warMode() ? 'war-mode-active' : ''}
                style={{
                    display: 'flex', 'flex-direction': 'column',
                    height: '100%', width: '100%', overflow: 'hidden',
                    background: 'var(--surface-0)', color: 'var(--text-primary)',
                    position: 'relative',
                }}
            >
                {/* Header */}
                <header style={{
                    display: 'flex', 'align-items': 'center', 'justify-content': 'space-between',
                    padding: '0 12px', height: '40px', 'flex-shrink': 0,
                    background: warMode() ? 'rgba(127,29,29,0.4)' : 'var(--surface-1)',
                    'border-bottom': `1px solid ${warMode() ? 'rgba(127,29,29,0.6)' : 'var(--border-primary)'}`,
                    'user-select': 'none',
                }}>
                    <div style={{ display: 'flex', 'align-items': 'center', gap: '12px' }}>
                        <span style={{ 'font-family': 'var(--font-mono)', 'font-size': '11px', 'font-weight': 800, color: 'var(--text-primary)', 'text-transform': 'uppercase', 'letter-spacing': '2px' }}>
                            SOVEREIGN // SOC TERMINAL
                        </span>
                        <div style={{ display: 'flex', 'align-items': 'center', gap: '6px', 'border-left': '1px solid var(--border-primary)', 'padding-left': '12px' }}>
                            <span style={{ width: '6px', height: '6px', 'border-radius': '50%', background: metrics()?.integrity === 'Optimal' ? 'var(--status-online)' : 'var(--status-offline)', display: 'inline-block' }} />
                            <span style={{ 'font-family': 'var(--font-mono)', 'font-size': '9px', 'font-weight': 700, color: 'var(--text-muted)', 'text-transform': 'uppercase', 'letter-spacing': '1px' }}>
                                {disasterMode() || 'NORMAL'}
                            </span>
                        </div>
                    </div>

                    <div style={{ display: 'flex', 'align-items': 'center', gap: '6px' }}>
                        {/* Layout presets */}
                        <div style={{ position: 'relative' }}>
                            <button onClick={() => setShowPresets(v => !v)} style={headerBtn()}>Presets</button>
                            <Show when={showPresets()}>
                                <div style={{
                                    position: 'absolute', top: '100%', right: 0, 'margin-top': '4px',
                                    background: 'var(--surface-2)', border: '1px solid var(--border-secondary)',
                                    'border-radius': '4px', padding: '6px', 'min-width': '200px',
                                    'box-shadow': 'var(--shadow-lg)', 'z-index': 9999,
                                }}>
                                    <div style={{ 'font-family': 'var(--font-mono)', 'font-size': '9px', 'font-weight': 800, color: 'var(--text-muted)', 'text-transform': 'uppercase', 'margin-bottom': '6px', 'letter-spacing': '1px' }}>
                                        Layout Presets
                                    </div>
                                    <div style={{ display: 'flex', gap: '4px', 'margin-bottom': '6px' }}>
                                        <input
                                            placeholder="Preset name…"
                                            value={presetName()}
                                            onInput={(e) => setPresetName(e.currentTarget.value)}
                                            style={{ flex: 1, background: 'var(--surface-3)', border: '1px solid var(--border-primary)', 'border-radius': '3px', padding: '4px 6px', color: 'var(--text-primary)', 'font-family': 'var(--font-mono)', 'font-size': '10px', outline: 'none' }}
                                        />
                                        <button
                                            onClick={() => { if (presetName() && layout) { savePreset(presetName(), layout); setPresetName(''); } }}
                                            style={{ background: 'var(--accent-primary)', border: 'none', 'border-radius': '3px', padding: '4px 8px', cursor: 'pointer', color: '#000', 'font-family': 'var(--font-mono)', 'font-size': '10px', 'font-weight': 800 }}
                                        >Save</button>
                                    </div>
                                    <For each={listPresets()}>
                                        {(name) => (
                                            <button
                                                onClick={() => { const c = loadPreset(name); if (c && layout) { layout.loadLayout(c); setShowPresets(false); } }}
                                                style={{ display: 'block', width: '100%', background: 'var(--surface-3)', border: '1px solid var(--border-primary)', 'border-radius': '3px', padding: '5px 8px', cursor: 'pointer', 'font-family': 'var(--font-mono)', 'font-size': '11px', color: 'var(--text-secondary)', 'text-align': 'left', 'margin-bottom': '3px' }}
                                            >▶ {name}</button>
                                        )}
                                    </For>
                                    <Show when={listPresets().length === 0}>
                                        <div style={{ 'font-size': '10px', color: 'var(--text-muted)', 'font-family': 'var(--font-mono)' }}>No presets saved</div>
                                    </Show>
                                </div>
                                {/* click-away */}
                                <div style={{ position: 'fixed', inset: 0, 'z-index': 9998 }} onPointerDown={() => setShowPresets(false)} />
                            </Show>
                        </div>

                        <button onClick={handleAirGap}     style={headerBtn(disasterMode() === 'AIR_GAP'    ? 'var(--accent-primary)' : undefined)}>Air-Gap</button>
                        <button onClick={handleKillSwitch} style={headerBtn(disasterMode() === 'KILL_SWITCH' ? 'var(--alert-medium)'   : undefined)}>Kill-Switch</button>
                        <button onClick={handleNuclear}    style={{ ...headerBtn('var(--alert-critical)'), 'box-shadow': '0 0 12px rgba(240,64,64,0.25)' }}>Nuclear</button>

                        <div style={{ width: '1px', height: '16px', background: 'var(--border-primary)', margin: '0 4px' }} />

                        <button
                            onClick={() => setWarMode(v => !v)}
                            style={headerBtn(warMode() ? 'var(--alert-critical)' : undefined)}
                        >
                            {warMode() ? 'WAR MODE ●' : 'WAR MODE'}
                        </button>

                        <button onClick={resetLayout} title="Reset layout" style={{ background: 'transparent', border: 'none', color: 'var(--text-muted)', cursor: 'pointer', padding: '4px', 'font-size': '14px' }}>↺</button>
                    </div>
                </header>

                {/* GoldenLayout container */}
                <div
                    ref={containerRef}
                    id="sov-gl-container"
                    class={warMode() ? 'war-mode-aberration' : ''}
                    style={{
                        flex: 1, width: '100%', position: 'relative',
                        opacity: disasterMode() === 'KILL_SWITCH' ? '0.4' : '1',
                        filter: disasterMode() === 'KILL_SWITCH' ? 'grayscale(1)' : 'none',
                        'pointer-events': disasterMode() === 'KILL_SWITCH' ? 'none' : 'auto',
                        transition: 'opacity 0.4s, filter 0.4s',
                    }}
                >
                    <Show when={disasterMode() === 'KILL_SWITCH'}>
                        <div style={{
                            position: 'absolute', inset: 0, 'z-index': 50,
                            display: 'flex', 'align-items': 'center', 'justify-content': 'center',
                            background: 'rgba(0,0,0,0.6)', 'backdrop-filter': 'blur(4px)',
                        }}>
                            <div style={{ color: 'var(--alert-medium)', 'font-family': 'var(--font-mono)', 'font-size': '32px', 'font-weight': 800, 'text-transform': 'uppercase', 'letter-spacing': '4px', border: '3px solid var(--alert-medium)', padding: '24px 40px' }}>
                                SYSTEM KILL-SWITCH ACTIVE
                            </div>
                        </div>
                    </Show>
                </div>

                {/* Minimised dock */}
                <Show when={minimisedPanels().length > 0}>
                    <div style={{
                        height: '32px', background: 'var(--surface-1)',
                        'border-top': '1px solid var(--border-primary)',
                        display: 'flex', 'align-items': 'center', padding: '0 8px', gap: '6px',
                        'overflow-x': 'auto', 'flex-shrink': 0,
                    }}>
                        <span style={{ 'font-family': 'var(--font-mono)', 'font-size': '8px', 'font-weight': 800, color: 'var(--text-muted)', 'text-transform': 'uppercase', 'letter-spacing': '1px', 'padding-right': '8px', 'border-right': '1px solid var(--border-primary)', 'white-space': 'nowrap' }}>
                            MINIMISED
                        </span>
                        <For each={minimisedPanels()}>
                            {(panel) => (
                                <button
                                    onClick={() => restorePanel(panel)}
                                    style={{ background: 'var(--surface-2)', border: '1px solid var(--border-primary)', 'border-radius': '3px', padding: '4px 10px', cursor: 'pointer', display: 'flex', 'align-items': 'center', gap: '6px', 'font-family': 'var(--font-mono)', 'font-size': '10px', color: 'var(--text-primary)', transition: 'all 0.12s', 'white-space': 'nowrap' }}
                                >
                                    <span style={{ color: 'var(--accent-primary)' }}>■</span> {panel.title}
                                </button>
                            )}
                        </For>
                    </div>
                </Show>

                {/* Footer */}
                <footer style={{
                    height: '26px', background: 'var(--surface-1)', 'border-top': '1px solid var(--border-primary)',
                    display: 'flex', 'align-items': 'center', padding: '0 12px', 'justify-content': 'space-between',
                    'font-family': 'var(--font-mono)', 'font-size': '9px', color: 'var(--text-muted)',
                    'text-transform': 'uppercase', 'letter-spacing': '0.5px', 'flex-shrink': 0,
                }}>
                    <div style={{ display: 'flex', gap: '20px' }}>
                        <span><span style={{ opacity: '0.4' }}>LATENCY </span><span style={{ color: 'var(--status-online)', 'font-weight': 800 }}>{metrics()?.latency}ms</span></span>
                        <span><span style={{ opacity: '0.4' }}>EPS </span><span style={{ color: 'var(--accent-primary)', 'font-weight': 800 }}>{metrics()?.eps}</span></span>
                        <span><span style={{ opacity: '0.4' }}>NODES </span><span style={{ 'font-weight': 700 }}>{metrics()?.nodes}</span></span>
                    </div>
                    <div style={{ display: 'flex', gap: '20px' }}>
                        <span><span style={{ opacity: '0.4' }}>INTEGRITY </span><span style={{ color: metrics()?.integrity === 'Optimal' ? 'var(--status-online)' : 'var(--status-offline)', 'font-weight': 800 }}>{metrics()?.integrity === 'Optimal' ? 'VERIFIED' : 'DEGRADED'}</span></span>
                        <span><span style={{ opacity: '0.4' }}>VAULT </span><span style={{ color: 'var(--alert-medium)', 'font-weight': 700 }}>UNLOCKED</span></span>
                    </div>
                </footer>
            </div>
        </>
    );
};

// Helper for header buttons
function headerBtn(accentColor?: string) {
    return {
        background: accentColor ? `${accentColor}22` : 'transparent',
        border: `1px solid ${accentColor ?? 'var(--border-primary)'}`,
        color: accentColor ?? 'var(--text-muted)',
        'border-radius': '3px', padding: '3px 8px', cursor: 'pointer',
        'font-family': 'var(--font-mono)', 'font-size': '9px', 'font-weight': '800',
        'text-transform': 'uppercase', 'letter-spacing': '0.5px',
        transition: 'all 0.12s',
    } as Record<string, string>;
}
