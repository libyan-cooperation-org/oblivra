import { Component, Show, JSX } from 'solid-js';

/* ═══════════════════════════════════════════════════════════════
   OBLIVRA Panel — Inner panel with header strip
   Wraps .ob-panel-header / .ob-panel-title from components.css
   ═══════════════════════════════════════════════════════════════ */

interface PanelProps {
    title?: string;
    subtitle?: string;
    actions?: JSX.Element;
    children: JSX.Element;
    class?: string;
    noPadding?: boolean;
}

export const Panel: Component<PanelProps> = (props) => (
    <div
        class={`${props.class || ''}`}
        style="display: flex; flex-direction: column; background: var(--surface-1); border: 1px solid var(--border-primary); border-radius: var(--radius-md); overflow: hidden;"
    >
        <Show when={props.title || props.actions}>
            <div class="ob-panel-header">
                <div style="display: flex; align-items: center; gap: 8px;">
                    <Show when={props.title}>
                        <span class="ob-panel-title">{props.title}</span>
                    </Show>
                    <Show when={props.subtitle}>
                        <span style="font-size: 10px; color: var(--text-muted); font-family: var(--font-mono);">
                            {props.subtitle}
                        </span>
                    </Show>
                </div>
                <Show when={props.actions}>
                    <div class="ob-panel-actions">{props.actions}</div>
                </Show>
            </div>
        </Show>
        <div style={props.noPadding ? undefined : "padding: 12px 16px;"}>
            {props.children}
        </div>
    </div>
);

/* ═══════════════════════════════════════════════════════════════
   OBLIVRA Section Header — Uppercase section divider
   Wraps .ob-section-header from components.css
   ═══════════════════════════════════════════════════════════════ */

interface SectionHeaderProps {
    children: JSX.Element;
    class?: string;
}

export const SectionHeader: Component<SectionHeaderProps> = (props) => (
    <div class={`ob-section-header ${props.class || ''}`}>
        {props.children}
    </div>
);
