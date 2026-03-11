/**
 * WindowFrame v2
 *
 * Now uses the shared PanelTitleBar primitive (#22).
 * Replaced all raw Tailwind gray-XXX classes with CSS variables (#21),
 * so the sovereign theme propagates correctly.
 */

import { Component, JSX } from 'solid-js';
import { PanelTitleBar } from '../layout/PanelTitleBar';

interface WindowFrameProps {
    title: string;
    status?: string;
    onClose?: () => void;
    onMinimise?: () => void;
    onMaximise?: () => void;
    onPopout?: () => void;
    children: JSX.Element;
}

export const WindowFrame: Component<WindowFrameProps> = (props) => {
    return (
        <div style={{
            height: '100%', width: '100%',
            display: 'flex', 'flex-direction': 'column',
            background: 'var(--surface-0)',
            border: '1px solid var(--border-primary)',
            position: 'relative', overflow: 'hidden',
        }}>
            {/* Corner decorators */}
            <div style={{ position: 'absolute', top: 0, left: 0, width: '8px', height: '8px', 'border-top': '2px solid rgba(87,139,255,0.35)', 'border-left': '2px solid rgba(87,139,255,0.35)', 'z-index': 20, 'pointer-events': 'none' }} />
            <div style={{ position: 'absolute', bottom: 0, right: 0, width: '8px', height: '8px', 'border-bottom': '2px solid rgba(87,139,255,0.2)', 'border-right': '2px solid rgba(87,139,255,0.2)', 'z-index': 20, 'pointer-events': 'none' }} />

            <PanelTitleBar
                title={props.title}
                status={props.status}
                onClose={props.onClose}
                onMinimise={props.onMinimise}
                onMaximise={props.onMaximise}
                onPopout={props.onPopout}
            />

            {/* Content */}
            <div style={{ flex: 1, position: 'relative', overflow: 'hidden', background: 'var(--surface-0)' }}>
                {props.children}

                {/* Subtle grid overlay */}
                <div style={{
                    position: 'absolute', inset: 0, 'pointer-events': 'none',
                    opacity: 0.018,
                    background: 'linear-gradient(rgba(255,255,255,0.05) 1px, transparent 1px), linear-gradient(90deg, rgba(255,255,255,0.05) 1px, transparent 1px)',
                    'background-size': '24px 24px',
                }} />
            </div>
        </div>
    );
};
