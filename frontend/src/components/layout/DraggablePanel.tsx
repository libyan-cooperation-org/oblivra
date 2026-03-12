/**
 * DraggablePanel v3
 *
 * All fixes applied:
 *  - Single onMount block (no duplicate event listeners)
 *  - Escape only minimises the topmost panel (highest z-index)
 *  - panelStyle() opacity logic is clean and unambiguous
 *  - Uses shared PanelTitleBar
 *  - Open/close animation, snap-to-edges, context menu, always-on-top,
 *    safe localStorage, sov:panel:resize event, Alt+Tab registry
 */

import {
    Component, createSignal, onMount, onCleanup,
    JSX, Show, createEffect,
} from 'solid-js';
import { PanelTitleBar } from './PanelTitleBar';

// ── Panel registry for Alt+Tab cycling ───────────────────────────────────────
export const panelRegistry: Set<string> = new Set();
export const panelFocusFns: Map<string, () => void> = new Map();

interface PanelPos  { x: number; y: number }
interface PanelSize { w: number; h: number }

export interface DraggablePanelProps {
    id: string;
    title: string;
    icon?: string;
    status?: string;
    defaultPos?: PanelPos;
    defaultSize?: PanelSize;
    minSize?: PanelSize;
    onClose?: () => void;
    onDuplicate?: () => void;
    children?: JSX.Element;
    class?: string;
}

// Module-level z-index counter shared across all panels
let _zCounter = 1000;
const bringToFront = () => ++_zCounter;

// ── Safe localStorage with in-memory fallback ─────────────────────────────────
const _memStore: Record<string, string> = {};
function storeGet(key: string): string | null {
    try { return localStorage.getItem(key); } catch { return _memStore[key] ?? null; }
}
function storeSet(key: string, val: string) {
    try { localStorage.setItem(key, val); } catch { _memStore[key] = val; }
}
const SK = (id: string) => `sov-panel-v2-${id}`;

function loadState(id: string, dp: PanelPos, ds: PanelSize) {
    try {
        const raw = storeGet(SK(id));
        if (raw) return JSON.parse(raw) as { pos: PanelPos; size: PanelSize };
    } catch { /* ignore */ }
    return { pos: dp, size: ds };
}
function saveState(id: string, pos: PanelPos, size: PanelSize) {
    try { storeSet(SK(id), JSON.stringify({ pos, size })); } catch { /* ignore */ }
}

// ── Magnetic edge snap (threshold = 20px) ────────────────────────────────────
const SNAP = 20;
function snapPos(x: number, y: number, w: number): PanelPos {
    const sw = window.innerWidth;
    const sh = window.innerHeight;
    const nx = x < SNAP ? 0 : x + w > sw - SNAP ? sw - w : x;
    const ny = y < SNAP ? 0 : y > sh - SNAP - 36 ? sh - 36 : y;
    return { x: nx, y: ny };
}

// ── Component ─────────────────────────────────────────────────────────────────
export const DraggablePanel: Component<DraggablePanelProps> = (props) => {
    const defaults = loadState(
        props.id,
        props.defaultPos  ?? { x: 80, y: 80 },
        props.defaultSize ?? { w: 860, h: 540 },
    );
    const minW = props.minSize?.w ?? 320;
    const minH = props.minSize?.h ?? 200;

    const [pos,       setPos]       = createSignal<PanelPos>(defaults.pos);
    const [size,      setSize]      = createSignal<PanelSize>(defaults.size);
    const [zIndex,    setZIndex]    = createSignal(bringToFront());
    const [minimised, setMinimised] = createSignal(false);
    const [maximised, setMaximised] = createSignal(false);
    const [alwaysTop, setAlwaysTop] = createSignal(false);
    const [opacity,   setOpacity]   = createSignal(1);
    const [mounted,   setMounted]   = createSignal(false);
    const [closing,   setClosing]   = createSignal(false);
    const [dragging,  setDragging]  = createSignal(false);
    const [ctxMenu,   setCtxMenu]   = createSignal<{ x: number; y: number } | null>(null);

    let preMaxPos:  PanelPos  = defaults.pos;
    let preMaxSize: PanelSize = defaults.size;

    // ── Drag state (mutable refs, not signals — no re-render needed) ──────────
    let dragData = { mx: 0, my: 0, px: 0, py: 0 };
    let isDragging = false;
    let resData = { mx: 0, my: 0, w: 0, h: 0 };
    let isResizing = false;

    // ── Pointer handlers ──────────────────────────────────────────────────────
    const onMove = (e: PointerEvent) => {
        if (isDragging) {
            const nx = dragData.px + e.clientX - dragData.mx;
            const ny = dragData.py + e.clientY - dragData.my;
            setPos({
                x: Math.max(-size().w + 80, Math.min(nx, window.innerWidth - 80)),
                y: Math.max(0, Math.min(ny, window.innerHeight - 40)),
            });
        }
        if (isResizing) {
            const nw = Math.max(minW, resData.w + e.clientX - resData.mx);
            const nh = Math.max(minH, resData.h + e.clientY - resData.my);
            setSize({ w: nw, h: nh });
            dispatchResizeEvent();
        }
    };

    const onUp = () => {
        if (isDragging) {
            isDragging = false;
            setDragging(false);
            const snapped = snapPos(pos().x, pos().y, size().w);
            setPos(snapped);
            saveState(props.id, snapped, size());
        }
        if (isResizing) {
            isResizing = false;
            saveState(props.id, pos(), size());
        }
    };

    // ── Single onMount: register all listeners once ───────────────────────────
    onMount(() => {
        requestAnimationFrame(() => setMounted(true));

        panelRegistry.add(props.id);
        panelFocusFns.set(props.id, bringUp);

        window.addEventListener('pointermove', onMove as EventListener);
        window.addEventListener('pointerup',   onUp);

        // Escape only fires on the topmost panel (the one with the highest z-index)
        const onKey = (e: KeyboardEvent) => {
            if (e.key === 'Escape' && !minimised() && zIndex() === _zCounter) {
                setMinimised(true);
            }
        };
        window.addEventListener('keydown', onKey);

        onCleanup(() => {
            panelRegistry.delete(props.id);
            panelFocusFns.delete(props.id);
            window.removeEventListener('pointermove', onMove as EventListener);
            window.removeEventListener('pointerup',   onUp);
            window.removeEventListener('keydown',     onKey);
        });
    });

    // ── Close with exit animation ─────────────────────────────────────────────
    const doClose = () => {
        setClosing(true);
        setTimeout(() => props.onClose?.(), 130);
    };

    // ── Title bar drag initiation ─────────────────────────────────────────────
    const onTitlePointerDown = (e: PointerEvent) => {
        if ((e.target as HTMLElement).closest('button')) return;
        if (maximised()) return;
        isDragging = true;
        setDragging(true);
        dragData = { mx: e.clientX, my: e.clientY, px: pos().x, py: pos().y };
        (e.currentTarget as HTMLElement).setPointerCapture(e.pointerId);
        e.preventDefault();
    };

    // ── SE-corner resize handle ───────────────────────────────────────────────
    const onResizeDown = (e: PointerEvent) => {
        isResizing = true;
        resData = { mx: e.clientX, my: e.clientY, w: size().w, h: size().h };
        (e.currentTarget as HTMLElement).setPointerCapture(e.pointerId);
        e.preventDefault();
        e.stopPropagation();
    };

    // ── Resize event for xterm FitAddon ──────────────────────────────────────
    const dispatchResizeEvent = () => {
        window.dispatchEvent(new CustomEvent('sov:panel:resize', {
            detail: { id: props.id, w: size().w, h: size().h },
        }));
    };
    // Only dispatch after mount — skip the initial reactive run on first render
    createEffect(() => {
        size(); // track signal
        if (mounted()) dispatchResizeEvent();
    });

    // ── Maximise / restore ────────────────────────────────────────────────────
    const toggleMaximise = () => {
        if (maximised()) {
            setPos(preMaxPos);
            setSize(preMaxSize);
            setMaximised(false);
        } else {
            preMaxPos  = pos();
            preMaxSize = size();
            setPos({ x: 0, y: 0 });
            setSize({ w: window.innerWidth, h: window.innerHeight - 28 });
            setMaximised(true);
        }
    };

    // ── Z-index management ────────────────────────────────────────────────────
    const bringUp = () => setZIndex(alwaysTop() ? 9990 : bringToFront());

    createEffect(() => {
        if (alwaysTop()) setZIndex(9990);
    });

    // ── Context menu ──────────────────────────────────────────────────────────
    const onContextMenu = (e: MouseEvent) => {
        e.preventDefault();
        setCtxMenu({ x: e.clientX, y: e.clientY });
    };
    const closeCtx = () => setCtxMenu(null);

    const sendToMonitor2 = () => {
        closeCtx();
        const secondX = (window.screen as any).availWidth ?? window.screen.width;
        window.open(
            window.location.href,
            `sov-popout-${props.id}`,
            `left=${secondX},top=0,width=${size().w},height=${size().h},resizable=yes`,
        );
    };

    const resetSize = () => {
        closeCtx();
        const ds = props.defaultSize ?? { w: 860, h: 540 };
        setSize(ds);
        saveState(props.id, pos(), ds);
    };

    // ── Computed style — explicit, unambiguous opacity logic ──────────────────
    const panelStyle = () => {
        // Animation: panel fades/scales in on mount, fades/scales out on close.
        // User-set opacity (from context menu) only applies while fully visible.
        const isVisible = mounted() && !closing();

        return {
            position:         'fixed',
            left:             `${pos().x}px`,
            top:              `${pos().y}px`,
            width:            `${size().w}px`,
            height:           minimised() ? '36px' : `${size().h}px`,
            'z-index':        String(zIndex()),
            overflow:         'hidden',
            display:          'flex',
            'flex-direction': 'column',
            background:       'var(--surface-0)',
            border:           '1px solid var(--border-primary)',
            'border-radius':  '4px',
            'box-shadow':     '0 24px 80px rgba(0,0,0,0.72), 0 0 0 1px rgba(255,255,255,0.04)',
            opacity:          isVisible ? String(opacity()) : '0',
            transform:        isVisible ? 'scale(1)' : 'scale(0.94)',
            transition:       'transform 130ms cubic-bezier(0.22,1,0.36,1), opacity 130ms ease, height 130ms ease',
        } as Record<string, string>;
    };

    // ── Render ────────────────────────────────────────────────────────────────
    return (
        <>
            <div
                style={panelStyle()}
                class={`sov-draggable-panel ${props.class ?? ''}`}
                onPointerDown={bringUp}
                onContextMenu={onContextMenu}
            >
                <PanelTitleBar
                    title={props.title}
                    icon={props.icon}
                    status={props.status}
                    dragging={dragging()}
                    maximised={maximised()}
                    minimised={minimised()}
                    onPointerDown={onTitlePointerDown as any}
                    onDblClick={() => toggleMaximise()}
                    onClose={doClose}
                    onMinimise={() => setMinimised(m => !m)}
                    onMaximise={toggleMaximise}
                />

                <Show when={!minimised()}>
                    <div style={{ flex: 1, overflow: 'hidden', position: 'relative' }}>
                        {props.children}
                    </div>

                    <Show when={!maximised()}>
                        <div
                            onPointerDown={onResizeDown}
                            style={{
                                position: 'absolute', right: 0, bottom: 0,
                                width: '18px', height: '18px',
                                cursor: 'se-resize',
                                display: 'flex', 'align-items': 'flex-end',
                                'justify-content': 'flex-end',
                                padding: '3px', opacity: '0.35', 'z-index': 9,
                            }}
                        >
                            <svg width="10" height="10" viewBox="0 0 10 10">
                                <path d="M9 1L1 9M9 5L5 9" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" />
                            </svg>
                        </div>
                    </Show>
                </Show>
            </div>

            {/* Right-click context menu */}
            <Show when={ctxMenu()}>
                <ContextMenu
                    x={ctxMenu()!.x}
                    y={ctxMenu()!.y}
                    onClose={closeCtx}
                    items={[
                        { label: '⧉  Duplicate',          action: () => { closeCtx(); props.onDuplicate?.(); } },
                        { label: '⎋  Send to Monitor 2',  action: sendToMonitor2 },
                        { label: '⤢  Reset Size',         action: resetSize },
                        {
                            label: alwaysTop() ? '📌 Unpin Always-on-Top' : '📌 Always on Top',
                            action: () => { closeCtx(); setAlwaysTop(v => !v); },
                        },
                        { label: '◑  Opacity 80%',        action: () => { closeCtx(); setOpacity(0.8); } },
                        { label: '○  Opacity 100%',       action: () => { closeCtx(); setOpacity(1); } },
                        { label: '', separator: true,      action: () => {} },
                        { label: '✕  Close',              action: () => { closeCtx(); doClose(); } },
                    ]}
                />
                <div
                    style={{ position: 'fixed', inset: 0, 'z-index': 19998 }}
                    onPointerDown={closeCtx}
                />
            </Show>
        </>
    );
};

// ── Context Menu ──────────────────────────────────────────────────────────────
interface CtxItem { label: string; action: () => void; separator?: boolean }

const ContextMenu: Component<{ x: number; y: number; onClose: () => void; items: CtxItem[] }> = (p) => (
    <div style={{
        position: 'fixed',
        left: `${Math.min(p.x, window.innerWidth - 210)}px`,
        top:  `${Math.min(p.y, window.innerHeight - 260)}px`,
        'z-index': 19999,
        background: 'var(--surface-2)',
        border: '1px solid var(--border-secondary)',
        'border-radius': '5px',
        padding: '4px',
        'min-width': '200px',
        'box-shadow': 'var(--shadow-lg)',
    }}>
        {p.items.map(item =>
            item.separator
                ? <div style={{ height: '1px', background: 'var(--border-primary)', margin: '3px 0' }} />
                : (
                    <button
                        onClick={item.action}
                        style={{
                            display: 'block', width: '100%', background: 'transparent',
                            border: 'none', 'border-radius': '3px', padding: '7px 10px',
                            'text-align': 'left', cursor: 'pointer',
                            'font-family': 'var(--font-ui)', 'font-size': '12px',
                            color: 'var(--text-secondary)', transition: 'all 0.1s',
                        }}
                        onMouseOver={(e) => {
                            const el = e.currentTarget as HTMLElement;
                            el.style.background = 'var(--surface-3)';
                            el.style.color = 'var(--text-primary)';
                        }}
                        onMouseOut={(e) => {
                            const el = e.currentTarget as HTMLElement;
                            el.style.background = 'transparent';
                            el.style.color = 'var(--text-secondary)';
                        }}
                    >
                        {item.label}
                    </button>
                )
        )}
    </div>
);
