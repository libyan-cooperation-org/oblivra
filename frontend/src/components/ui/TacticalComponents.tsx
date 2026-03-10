import { Component, JSX, splitProps } from 'solid-js';

/* ── Tactical Card ── */
interface CardProps extends JSX.HTMLAttributes<HTMLDivElement> {
    variant?: 'flat' | 'raised' | 'outline';
    padding?: string;
}

export const Card: Component<CardProps> = (props) => {
    const [local, rest] = splitProps(props, ['variant', 'padding', 'children', 'class', 'style']);

    const getVariantStyle = () => {
        switch (local.variant) {
            case 'raised': return 'background: var(--bg-secondary); border: 1px solid var(--border-secondary); box-shadow: var(--shadow-md);';
            case 'outline': return 'background: transparent; border: 1px solid var(--border-secondary);';
            default: return 'background: var(--bg-surface); border: 1px solid transparent;';
        }
    };

    return (
        <div
            class={`tactical-card ${local.class || ''}`}
            style={`${getVariantStyle()} padding: ${local.padding || '1rem'}; border-radius: var(--radius-sm); ${local.style || ''}`}
            {...rest}
        >
            {local.children}
        </div>
    );
};

/* ── Tactical Badge ── */
interface BadgeProps extends JSX.HTMLAttributes<HTMLSpanElement> {
    severity?: 'info' | 'success' | 'warning' | 'error' | 'neutral';
}

export const Badge: Component<BadgeProps> = (props) => {
    const [local, rest] = splitProps(props, ['severity', 'children', 'class', 'style']);

    const getColor = () => {
        switch (local.severity) {
            case 'success': return 'var(--status-online)';
            case 'warning': return 'var(--status-degraded)';
            case 'error': return 'var(--status-offline)';
            case 'info': return 'var(--accent-secondary)';
            default: return 'var(--text-muted)';
        }
    };

    return (
        <span
            class={`tactical-badge ${local.class || ''}`}
            style={`
                display: inline-flex;
                align-items: center;
                padding: 2px 8px;
                font-size: 10px;
                font-family: var(--font-mono);
                font-weight: 800;
                text-transform: uppercase;
                letter-spacing: 0.5px;
                border-radius: var(--radius-xs);
                background: ${getColor()}1A;
                color: ${getColor()};
                border: 1px solid ${getColor()}33;
                ${local.style || ''}
            `}
            {...rest}
        >
            {local.children}
        </span>
    );
};

/* ── Tactical Button ── */
interface ButtonProps extends JSX.ButtonHTMLAttributes<HTMLButtonElement> {
    variant?: 'primary' | 'secondary' | 'ghost' | 'danger';
    size?: 'sm' | 'md' | 'lg';
}

export const Button: Component<ButtonProps> = (props) => {
    const [local, rest] = splitProps(props, ['variant', 'size', 'children', 'class', 'style']);

    const getVariantStyles = () => {
        switch (local.variant) {
            case 'primary': return 'background: var(--accent-secondary); color: #fff; border: 1px solid var(--accent-secondary);';
            case 'danger': return 'background: var(--status-offline); color: #fff; border: 1px solid var(--status-offline);';
            case 'ghost': return 'background: transparent; color: var(--text-secondary); border: 1px solid var(--border-secondary);';
            default: return 'background: var(--bg-tertiary); color: var(--text-primary); border: 1px solid var(--border-secondary);';
        }
    };

    const getSizeStyles = () => {
        switch (local.size) {
            case 'sm': return 'padding: 4px 10px; font-size: 11px;';
            case 'lg': return 'padding: 10px 20px; font-size: 14px;';
            default: return 'padding: 6px 14px; font-size: 12px;';
        }
    };

    return (
        <button
            class={`tactical-button transition-all ${local.class || ''}`}
            style={`
                display: inline-flex;
                align-items: center;
                justify-content: center;
                font-family: var(--font-ui);
                font-weight: 600;
                cursor: pointer;
                border-radius: var(--radius-sm);
                ${getVariantStyles()}
                ${getSizeStyles()}
                ${local.style || ''}
            `}
            {...rest}
        >
            {local.children}
        </button>
    );
};
