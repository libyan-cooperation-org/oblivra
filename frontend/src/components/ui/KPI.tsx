import { Component, Show, JSX } from 'solid-js';
import { Sparkline } from './Sparkline';

/* ═══════════════════════════════════════════════════════════════
   OBLIVRA KPI Card — Splunk dashboard metric tile
   Wraps .ob-kpi from components.css
   ═══════════════════════════════════════════════════════════════ */

interface KPIProps {
    label: string;
    value: string | number;
    subtitle?: string;
    delta?: number | null;
    deltaLabel?: string;
    color?: string;
    sparkData?: number[];
    sparkColor?: string;
    onClick?: () => void;
    class?: string;
}

export const KPI: Component<KPIProps> = (props) => {
    const deltaClass = () => {
        if (props.delta == null) return '';
        return props.delta >= 0 ? 'up' : 'down';
    };

    const deltaText = () => {
        if (props.delta == null) return '';
        const sign = props.delta >= 0 ? '+' : '';
        const label = props.deltaLabel || '';
        return `${sign}${props.delta}${label ? ' ' + label : ''}`;
    };

    return (
        <div
            class={`ob-kpi ${props.class || ''}`}
            onClick={props.onClick}
            style={props.onClick ? 'cursor: pointer;' : undefined}
        >
            <div class="ob-kpi-label">{props.label}</div>
            <div
                class="ob-kpi-value"
                style={props.color ? `color: ${props.color};` : undefined}
            >
                {props.value}
            </div>
            <Show when={props.subtitle}>
                <div style="font-size: 9px; color: var(--text-muted); font-family: var(--font-mono); margin-top: -2px; font-weight: 700; text-transform: uppercase;">
                    {props.subtitle}
                </div>
            </Show>
            <Show when={props.delta != null}>
                <div class={`ob-kpi-delta ${deltaClass()}`}>
                    {deltaText()}
                </div>
            </Show>
            <Show when={props.sparkData && props.sparkData.length > 1}>
                <div style="margin-top: 4px;">
                    <Sparkline
                        data={props.sparkData!}
                        color={props.sparkColor || 'var(--accent-primary)'}
                        height={24}
                    />
                </div>
            </Show>
        </div>
    );
};

/* ═══════════════════════════════════════════════════════════════
   KPI Grid — Stat grid layout
   Wraps .ob-stat-grid from components.css
   ═══════════════════════════════════════════════════════════════ */

interface KPIGridProps {
    cols?: 2 | 3 | 4 | 5;
    children: JSX.Element;
    class?: string;
}

export const KPIGrid: Component<KPIGridProps> = (props) => (
    <div class={`ob-stat-grid ob-stat-grid-${props.cols || 4} ${props.class || ''}`}>
        {props.children}
    </div>
);
