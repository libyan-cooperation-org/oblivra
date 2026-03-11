/**
 * usePanelShortcuts v3
 *
 * Fixes:
 *  - Corrected import paths: SIEMPanel, FleetDashboard, CommandCenter
 *  - lazyPanel helper now correctly defined as a JSX-returning component factory
 *  - Import path to PanelManager is correct relative to src/hooks/
 */

import { onMount, onCleanup, createSignal, Show } from 'solid-js';
import { usePanelManager, cascadePos } from '../components/layout/PanelManager';

/**
 * Returns a content-factory function.
 * The returned function is called inside a component context (by PanelManager),
 * so createSignal and Show are valid here.
 */
function lazyPanel(importFn: () => Promise<any>, exportKey?: string) {
    return () => {
        const [Comp, setComp] = createSignal<any>(null);
        importFn()
            .then(m => setComp(() => exportKey ? m[exportKey] : (m.default ?? Object.values(m)[0])))
            .catch(err => console.error('[usePanelShortcuts] lazy import failed:', err));
        return <Show when={Comp()}>{(C) => <C />}</Show>;
    };
}

export function usePanelShortcuts() {
    const { openPanel } = usePanelManager();

    const spawn = (id: string, title: string, icon: string, content: () => any, size = { w: 960, h: 620 }) =>
        openPanel({ id, title, icon, defaultPos: cascadePos({ x: 100, y: 80 }), defaultSize: size, content });

    const map: Record<string, () => void> = {
        // Ctrl+Shift+T — new terminal (unique id so multiple can coexist)
        't': () => spawn(
            `terminal-${Date.now()}`, 'Terminal', '🖥️',
            lazyPanel(() => import('../components/terminal/TerminalLayout'), 'TerminalLayout'),
            { w: 920, h: 560 },
        ),
        // Ctrl+Shift+D — dashboard
        'd': () => spawn(
            'dashboard', 'Dashboard', '📊',
            lazyPanel(() => import('../components/dashboard/Dashboard'), 'Dashboard'),
            { w: 1100, h: 700 },
        ),
        // Ctrl+Shift+S — SIEM (actual export: SIEMPanel in SIEMPanel.tsx)
        's': () => spawn(
            'siem', 'SIEM', '🛡️',
            lazyPanel(() => import('../components/siem/SIEMPanel'), 'SIEMPanel'),
            { w: 1200, h: 750 },
        ),
        // Ctrl+Shift+F — Fleet (actual export: FleetDashboard in FleetDashboard.tsx)
        'f': () => spawn(
            'fleet', 'Fleet', '🖧',
            lazyPanel(() => import('../components/fleet/FleetDashboard'), 'FleetDashboard'),
            { w: 1100, h: 680 },
        ),
        // Ctrl+Shift+I — Incidents (actual export: CommandCenter in CommandCenter.tsx)
        'i': () => spawn(
            'incidents', 'Incidents', '🚨',
            lazyPanel(() => import('../components/incident/CommandCenter'), 'CommandCenter'),
            { w: 1000, h: 650 },
        ),
        // Ctrl+Shift+O — SOC workspace
        'o': () => spawn(
            'soc', 'SOC Workspace', '🗖',
            lazyPanel(() => import('../components/soc/SOCWorkspace'), 'SOCWorkspace'),
            { w: 1400, h: 900 },
        ),
    };

    const handler = (e: KeyboardEvent) => {
        const mod = e.ctrlKey || e.metaKey;
        if (!mod || !e.shiftKey) return;
        const action = map[e.key.toLowerCase()];
        if (action) { e.preventDefault(); action(); }
    };

    onMount(() => window.addEventListener('keydown', handler));
    onCleanup(() => window.removeEventListener('keydown', handler));
}
