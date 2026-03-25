import { Component, Show, JSX, createSignal, onMount, onCleanup, For } from 'solid-js';

/* ═══════════════════════════════════════════════════════════════
   OBLIVRA Skeleton — Loading placeholder
   Wraps .ob-skeleton from components.css
   ═══════════════════════════════════════════════════════════════ */

interface SkeletonProps {
    width?: string;
    height?: string;
    variant?: 'text' | 'block';
    count?: number;
    class?: string;
}

export const Skeleton: Component<SkeletonProps> = (props) => {
    const cls = () => {
        const base = 'ob-skeleton';
        if (props.variant === 'block') return `${base} ob-skeleton-block`;
        return `${base} ob-skeleton-text`;
    };

    const count = props.count || 1;
    return (
        <>
            {Array.from({ length: count }).map(() => (
                <div
                    class={`${cls()} ${props.class || ''}`}
                    style={`${props.width ? `width: ${props.width};` : ''}${props.height ? `height: ${props.height};` : ''}`}
                />
            ))}
        </>
    );
};

/* ═══════════════════════════════════════════════════════════════
   OBLIVRA LoadingState — Centered loading message
   re-uses existing .ob-loading class
   ═══════════════════════════════════════════════════════════════ */

interface LoadingStateProps {
    message?: string;
}

export const LoadingState: Component<LoadingStateProps> = (props) => (
    <div class="ob-loading">{props.message || 'LOADING…'}</div>
);

/* ═══════════════════════════════════════════════════════════════
   OBLIVRA ErrorState — Error display with retry
   re-uses existing .ob-load-error class
   ═══════════════════════════════════════════════════════════════ */

interface ErrorStateProps {
    message: string;
    onRetry?: () => void;
}

export const ErrorState: Component<ErrorStateProps> = (props) => (
    <div class="ob-load-error">
        <span>{props.message}</span>
        <Show when={props.onRetry}>
            <button onClick={props.onRetry}>RETRY</button>
        </Show>
    </div>
);

/* ═══════════════════════════════════════════════════════════════
   OBLIVRA RelativeTime — Human-readable relative timestamp
   ═══════════════════════════════════════════════════════════════ */

export function formatRelativeTime(date: Date | string | number): string {
    const now = Date.now();
    const then = new Date(date).getTime();
    const diff = now - then;

    if (diff < 0) return 'just now';
    if (diff < 60_000) return `${Math.floor(diff / 1000)}s ago`;
    if (diff < 3_600_000) return `${Math.floor(diff / 60_000)}m ago`;
    if (diff < 86_400_000) return `${Math.floor(diff / 3_600_000)}h ago`;
    if (diff < 604_800_000) return `${Math.floor(diff / 86_400_000)}d ago`;
    return new Date(date).toISOString().slice(0, 10);
}

export function formatTimestamp(date: Date | string | number): string {
    return new Date(date).toISOString().replace('T', ' ').slice(0, 19);
}

/* ═══════════════════════════════════════════════════════════════
   OBLIVRA KeyValue — Inline key-value display (for detail panels)
   ═══════════════════════════════════════════════════════════════ */

interface KeyValueProps {
    label: string;
    value: string | number | JSX.Element;
    mono?: boolean;
}

export const KeyValue: Component<KeyValueProps> = (props) => (
    <div style="display: flex; justify-content: space-between; align-items: center; padding: 6px 0; border-bottom: 1px solid var(--border-subtle);">
        <span style="font-size: 11px; color: var(--text-muted); text-transform: uppercase; letter-spacing: 0.4px; font-weight: 600;">
            {props.label}
        </span>
        <span
            style={`font-size: 12px; color: var(--text-primary);${props.mono !== false ? ' font-family: var(--font-mono); font-size: 11px;' : ''}`}
        >
            {props.value}
        </span>
    </div>
);

/* ═══════════════════════════════════════════════════════════════
   OBLIVRA CountBadge — Numeric count indicator
   ═══════════════════════════════════════════════════════════════ */

interface CountBadgeProps {
    count: number;
    color?: string;
}

export const CountBadge: Component<CountBadgeProps> = (props) => (
    <Show when={props.count > 0}>
        <span style={`
            font-family: var(--font-mono);
            font-size: 9px;
            font-weight: 800;
            padding: 1px 5px;
            border-radius: var(--radius-full);
            background: ${props.color || 'var(--accent-primary)'};
            color: var(--surface-0);
            line-height: 1.3;
            min-width: 16px;
            text-align: center;
            display: inline-block;
        `}>
            {props.count > 99 ? '99+' : props.count}
        </span>
    </Show>
);
