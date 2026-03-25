import { Component, JSX } from 'solid-js';

/* ═══════════════════════════════════════════════════════════════
   OBLIVRA Badge — Severity + status pills
   Wraps .ob-badge + .sev-* from components.css
   ═══════════════════════════════════════════════════════════════ */

export type BadgeColor = 'green' | 'red' | 'yellow' | 'orange' | 'blue' | 'gray';
export type SeverityLevel = 'critical' | 'high' | 'medium' | 'low' | 'info' | 'success' | 'warning' | 'danger' | 'neutral';

interface BadgeProps {
    color?: BadgeColor | string;
    severity?: SeverityLevel;
    size?: 'sm' | 'md' | 'lg';
    children: JSX.Element;
    class?: string;
}

const sevToClass: Record<string, string> = {
    critical: 'sev-critical',
    high:     'sev-high',
    medium:   'sev-medium',
    low:      'sev-low',
    info:     'sev-info',
    success:  'ob-badge-green',
    warning:  'ob-badge-yellow',
    danger:   'ob-badge-red',
    neutral:  'ob-badge-gray',
};

const colorToClass: Record<string, string> = {
    green:  'ob-badge-green',
    red:    'ob-badge-red',
    yellow: 'ob-badge-yellow',
    orange: 'ob-badge-orange',
    blue:   'ob-badge-blue',
    gray:   'ob-badge-gray',
};

export const Badge: Component<BadgeProps> = (props) => {
    const cls = () => {
        let base = 'ob-badge';
        if (props.size === 'sm') base += ' ob-badge-sm';
        if (props.size === 'lg') base += ' ob-badge-lg';
        
        if (props.severity && sevToClass[props.severity]) {
            return `${base} ${sevToClass[props.severity]}`;
        }
        if (props.color && colorToClass[props.color]) {
            return `${base} ${colorToClass[props.color]}`;
        }
        return `${base} ob-badge-gray`;
    };

    return (
        <span class={`${cls()} ${props.class || ''}`}>
            {props.children}
        </span>
    );
};

/* ═══════════════════════════════════════════════════════════════
   Severity Map — normalize severity string to typed levels
   ═══════════════════════════════════════════════════════════════ */

export function normalizeSeverity(sev: string | undefined | null): SeverityLevel {
    const s = (sev || '').toLowerCase().trim();
    if (s === 'critical' || s === 'crit') return 'critical';
    if (s === 'high' || s === 'danger') return 'high';
    if (s === 'medium' || s === 'med' || s === 'moderate' || s === 'warning' || s === 'warn') return 'medium';
    if (s === 'low' || s === 'success' || s === 'info') return 'low';
    return 'info';
}

/* ═══════════════════════════════════════════════════════════════
   Status Dot
   Wraps .ob-status-dot from components.css
   ═══════════════════════════════════════════════════════════════ */

type StatusLevel = 'online' | 'warn' | 'offline';

interface StatusDotProps {
    status: StatusLevel;
    class?: string;
}

const statusToClass: Record<StatusLevel, string> = {
    online:  'ob-status-online',
    warn:    'ob-status-warn',
    offline: 'ob-status-offline',
};

export const StatusDot: Component<StatusDotProps> = (props) => (
    <span class={`ob-status-dot ${statusToClass[props.status]} ${props.class || ''}`} />
);
