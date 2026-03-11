/**
 * PanelTitleBar — shared primitive used by BOTH DraggablePanel and
 * the GoldenLayout WindowFrame.  Single source of truth for the
 * sovereign title-bar chrome.
 *
 * Props:
 *   title          — panel label (uppercased automatically)
 *   icon           — optional emoji / SVG string
 *   status         — optional badge text, e.g. "LIVE"
 *   dragging       — pass true while the panel is being dragged
 *   maximised      — controls the icon shown on the maximise button
 *   minimised      — controls the icon shown on the minimise button
 *   onPointerDown  — forwarded to the title bar div for drag initiation
 *   onDblClick     — forwarded (toggle maximise on double-click)
 *   onClose / onMinimise / onMaximise / onPopout — button handlers
 */

import { Component, Show, JSX } from 'solid-js';

export interface PanelTitleBarProps {
    title: string;
    icon?: string;
    status?: string;
    dragging?: boolean;
    maximised?: boolean;
    minimised?: boolean;
    onPointerDown?: JSX.EventHandlerUnion<HTMLDivElement, PointerEvent>;
    onDblClick?: JSX.EventHandlerUnion<HTMLDivElement, MouseEvent>;
    onClose?: () => void;
    onMinimise?: () => void;
    onMaximise?: () => void;
    onPopout?: () => void;
    /** Extra JSX rendered in the right slot (before pop-out) */
    rightSlot?: JSX.Element;
}

export const PanelTitleBar: Component<PanelTitleBarProps> = (props) => {
    return (
        <div
            class="sov-titlebar"
            onPointerDown={props.onPointerDown}
            onDblClick={props.onDblClick}
            style={{
                display: 'flex',
                'align-items': 'center',
                gap: '8px',
                padding: '0 10px',
                height: '36px',
                'flex-shrink': 0,
                background: 'var(--surface-1)',
                'border-bottom': '1px solid var(--border-primary)',
                cursor: props.maximised ? 'default' : props.dragging ? 'grabbing' : 'grab',
                'user-select': 'none',
                position: 'relative',
            }}
        >
            {/* Corner accent */}
            <div style={{
                position: 'absolute', top: 0, left: 0,
                width: '8px', height: '8px',
                'border-top': '2px solid rgba(87,139,255,0.35)',
                'border-left': '2px solid rgba(87,139,255,0.35)',
            }} />
            <div style={{
                position: 'absolute', top: 0, right: 0,
                width: '8px', height: '8px',
                'border-top': '2px solid rgba(87,139,255,0.35)',
                'border-right': '2px solid rgba(87,139,255,0.35)',
            }} />

            {/* Traffic-light buttons */}
            <div style={{ display: 'flex', gap: '6px', 'flex-shrink': 0, '-webkit-app-region': 'no-drag' }}>
                <TLBtn color="#ff5f57" title="Close"   onClick={props.onClose}    label="×" />
                <TLBtn color="#febc2e" title={props.minimised ? 'Restore' : 'Minimise'} onClick={props.onMinimise} label="–" />
                <TLBtn color="#28c840" title={props.maximised ? 'Restore' : 'Maximise'} onClick={props.onMaximise} label="⤢" />
            </div>

            {/* Vertical accent bar */}
            <div style={{ width: '3px', height: '14px', background: 'var(--accent-primary)', opacity: '0.6', 'border-radius': '2px', 'flex-shrink': 0 }} />

            {/* Icon */}
            <Show when={props.icon}>
                <span style={{ 'font-size': '13px', opacity: '0.75', 'flex-shrink': 0 }}>{props.icon}</span>
            </Show>

            {/* Title */}
            <span style={{
                flex: 1,
                'font-family': 'var(--font-mono)',
                'font-size': '10px',
                'font-weight': 800,
                'text-transform': 'uppercase',
                'letter-spacing': '1.5px',
                color: 'var(--text-muted)',
                overflow: 'hidden',
                'white-space': 'nowrap',
                'text-overflow': 'ellipsis',
            }}>
                {props.title}
            </span>

            {/* Status badge */}
            <Show when={props.status}>
                <span style={{
                    'font-family': 'var(--font-mono)',
                    'font-size': '8px',
                    'font-weight': 800,
                    'text-transform': 'uppercase',
                    'letter-spacing': '1px',
                    color: 'var(--accent-primary)',
                    background: 'rgba(87,139,255,0.12)',
                    padding: '2px 6px',
                    'border-radius': '2px',
                    'flex-shrink': 0,
                }}>
                    {props.status}
                </span>
            </Show>

            {/* Right slot */}
            <Show when={props.rightSlot}>
                {props.rightSlot}
            </Show>

            {/* Pop-out */}
            <Show when={props.onPopout}>
                <button
                    title="Pop out to new window"
                    onClick={(e) => { e.stopPropagation(); props.onPopout!(); }}
                    style={{
                        background: 'transparent',
                        border: '1px solid var(--border-primary)',
                        color: 'var(--text-muted)',
                        'border-radius': '3px',
                        padding: '2px 6px',
                        cursor: 'pointer',
                        'font-size': '9px',
                        'font-family': 'var(--font-mono)',
                        'font-weight': 700,
                        'text-transform': 'uppercase',
                        'letter-spacing': '0.5px',
                        'flex-shrink': 0,
                        '-webkit-app-region': 'no-drag',
                    }}
                >⎋ POP</button>
            </Show>
        </div>
    );
};

// ── Traffic light helper ─────────────────────────────────────────────────────
const TLBtn: Component<{ color: string; title: string; onClick?: () => void; label: string }> = (p) => {
    return (
        <button
            title={p.title}
            onClick={(e) => { e.stopPropagation(); p.onClick?.(); }}
            style={{
                width: '13px',
                height: '13px',
                'border-radius': '50%',
                background: p.color,
                border: 'none',
                cursor: 'pointer',
                'font-size': '0',
                padding: 0,
                'flex-shrink': 0,
                transition: 'filter 0.1s',
                '-webkit-app-region': 'no-drag',
            }}
            onMouseOver={(e) => (e.currentTarget as HTMLElement).style.filter = 'brightness(1.25)'}
            onMouseOut={(e)  => (e.currentTarget as HTMLElement).style.filter = ''}
        >
            {p.label}
        </button>
    );
};
