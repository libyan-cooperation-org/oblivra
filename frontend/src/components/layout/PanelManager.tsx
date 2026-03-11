/**
 * PanelManager v2
 *
 * Changes from v1:
 *  - Module-level signal replaced with a proper SolidJS context (#24)
 *  - openPanel() supports onDuplicate callback (wires up ContextMenu item)
 *  - saveSnapshot() / restoreSnapshot() for named workspace snapshots (#20)
 *  - StatusBar helper: openPanelCount() reactive accessor
 *  - Alt+Tab cycling across all open panels (#14)
 */

import {
    Component, createContext, useContext, createSignal,
    For, JSX, ParentComponent, onMount, onCleanup,
} from 'solid-js';
import { DraggablePanel, panelFocusFns } from './DraggablePanel';

// ── Panel definition ─────────────────────────────────────────────────────────
export interface PanelDef {
    id: string;
    title: string;
    icon?: string;
    status?: string;
    content: () => JSX.Element;
    defaultPos?:  { x: number; y: number };
    defaultSize?: { w: number; h: number };
    minSize?:     { w: number; h: number };
}

// ── Snapshot type ─────────────────────────────────────────────────────────────
interface PanelSnapshot {
    name: string;
    panels: Array<Omit<PanelDef, 'content'>>;
}

// ── Context ───────────────────────────────────────────────────────────────────
interface PanelManagerCtx {
    panels:          () => PanelDef[];
    openPanel:       (def: PanelDef) => void;
    closePanel:      (id: string) => void;
    isPanelOpen:     (id: string) => boolean;
    openPanelCount:  () => number;
    saveSnapshot:    (name: string) => void;
    loadSnapshot:    (name: string) => void;
    listSnapshots:   () => string[];
    deleteSnapshot:  (name: string) => void;
}

const Ctx = createContext<PanelManagerCtx>();

export function usePanelManager(): PanelManagerCtx {
    const ctx = useContext(Ctx);
    if (!ctx) throw new Error('usePanelManager must be used inside <PanelManagerProvider>');
    return ctx;
}

// ── Cascade offset ────────────────────────────────────────────────────────────
let _spawnCount = 0;
const CASCADE = 36;
export function cascadePos(base?: { x: number; y: number }) {
    const off = (_spawnCount++ % 8) * CASCADE;
    return { x: (base?.x ?? 100) + off, y: (base?.y ?? 80) + off };
}

// ── Safe storage ──────────────────────────────────────────────────────────────
function snapKey(name: string) { return `sov-snapshot-${name}`; }
function listSnapNames(): string[] {
    try {
        return Object.keys(localStorage)
            .filter(k => k.startsWith('sov-snapshot-'))
            .map(k => k.replace('sov-snapshot-', ''));
    } catch { return []; }
}

// ── Provider ──────────────────────────────────────────────────────────────────
export const PanelManagerProvider: ParentComponent = (props) => {
    const [panels, setPanels] = createSignal<PanelDef[]>([]);
    // content factories keyed by panel id for active panels
    const contentFactories = new Map<string, () => JSX.Element>();
    // type registry keyed by a stable "type key" (e.g. 'terminal', 'dashboard')
    // so snapshots can restore even after the originating panel was closed.
    const typeRegistry = new Map<string, () => JSX.Element>();

    const openPanel = (def: PanelDef) => {
        if (panels().find(p => p.id === def.id)) return;
        contentFactories.set(def.id, def.content);
        // Derive type key: strip timestamp suffixes like '-1234567890'
        const typeKey = def.id.replace(/-\d+$/, '');
        typeRegistry.set(typeKey, def.content);
        setPanels(prev => [...prev, def]);
    };

    const closePanel = (id: string) => {
        contentFactories.delete(id);
        setPanels(prev => prev.filter(p => p.id !== id));
    };

    const isPanelOpen  = (id: string) => panels().some(p => p.id === id);
    const openPanelCount = () => panels().length;

    // ── Snapshots ─────────────────────────────────────────────────────────────
    const saveSnapshot = (name: string) => {
        const snapshot: PanelSnapshot = {
            name,
            panels: panels().map(({ content: _c, ...rest }) => rest),
        };
        try { localStorage.setItem(snapKey(name), JSON.stringify(snapshot)); } catch { /* ignore */ }
    };

    const loadSnapshot = (name: string) => {
        try {
            const raw = localStorage.getItem(snapKey(name));
            if (!raw) return;
            const snap = JSON.parse(raw) as PanelSnapshot;
            snap.panels.forEach(p => {
                // Try exact id match first, then type key match
                const typeKey = p.id.replace(/-\d+$/, '');
                const factory = contentFactories.get(p.id) ?? typeRegistry.get(typeKey);
                if (factory) openPanel({ ...p, content: factory });
            });
        } catch { /* ignore */ }
    };

    const listSnapshots = listSnapNames;
    const deleteSnapshot = (name: string) => {
        try { localStorage.removeItem(snapKey(name)); } catch { /* ignore */ }
    };

    // ── Alt+Tab cycling ───────────────────────────────────────────────────────
    onMount(() => {
        let tabIdx = 0;
        const onKey = (e: KeyboardEvent) => {
            if (e.altKey && e.key === 'Tab') {
                e.preventDefault();
                const ids = [...panelFocusFns.keys()];
                if (ids.length === 0) return;
                tabIdx = (tabIdx + (e.shiftKey ? -1 : 1) + ids.length) % ids.length;
                panelFocusFns.get(ids[tabIdx])?.();
            }
        };
        window.addEventListener('keydown', onKey);
        onCleanup(() => window.removeEventListener('keydown', onKey));
    });

    const ctx: PanelManagerCtx = {
        panels, openPanel, closePanel, isPanelOpen,
        openPanelCount, saveSnapshot, loadSnapshot,
        listSnapshots, deleteSnapshot,
    };

    return (
        <Ctx.Provider value={ctx}>
            {props.children}
            <PanelManagerLayer />
        </Ctx.Provider>
    );
};

// ── Render layer ──────────────────────────────────────────────────────────────
/**
 * Rendered automatically by PanelManagerProvider — do NOT also drop
 * PanelManagerLayer in AppLayout anymore.
 */
const PanelManagerLayer: Component = () => {
    const { panels, closePanel, openPanel } = usePanelManager();
    return (
        <For each={panels()}>
            {(def) => (
                <DraggablePanel
                    id={def.id}
                    title={def.title}
                    icon={def.icon}
                    status={def.status}
                    defaultPos={def.defaultPos}
                    defaultSize={def.defaultSize}
                    minSize={def.minSize}
                    onClose={() => closePanel(def.id)}
                    onDuplicate={() => openPanel({
                        ...def,
                        id: `${def.id}-${Date.now()}`,
                        defaultPos: { x: (def.defaultPos?.x ?? 100) + 40, y: (def.defaultPos?.y ?? 80) + 40 },
                    })}
                >
                    {def.content()}
                </DraggablePanel>
            )}
        </For>
    );
};
