/**
 * RouteGuard — wraps any route component and blocks access if the
 * current deployment context does not support it.
 *
 * Usage in index.tsx:
 *   <Route path="/terminal" component={() =>
 *     <RouteGuard path="/terminal"><TerminalLayout /></RouteGuard>
 *   } />
 */
import { Component, Show, ParentComponent } from 'solid-js';
import {
    APP_CONTEXT,
    isRouteAvailable,
    routeUnavailableReason,
    IS_DESKTOP,
    IS_HYBRID,
} from './context';

interface RouteGuardProps {
    path: string;
}

export const RouteGuard: ParentComponent<RouteGuardProps> = (props) => {
    const available = isRouteAvailable(props.path);
    const reason = routeUnavailableReason(props.path);

    return (
        <Show
            when={available}
            fallback={<UnavailableScreen path={props.path} reason={reason!} />}
        >
            {props.children}
        </Show>
    );
};

// ── Unavailable screen ────────────────────────────────────────────────────

const contextLabel: Record<string, string> = {
    desktop: 'Desktop',
    browser: 'Browser / Server',
    hybrid:  'Hybrid (Desktop + Server)',
};

const UnavailableScreen: Component<{ path: string; reason: string }> = (props) => (
    <div style={{
        display: 'flex',
        'flex-direction': 'column',
        'align-items': 'center',
        'justify-content': 'center',
        height: '100%',
        gap: '20px',
        padding: '48px',
        background: 'var(--surface-0)',
        'text-align': 'center',
    }}>
        {/* Icon */}
        <div style={{
            width: '48px',
            height: '48px',
            'border-radius': '8px',
            background: 'rgba(245,139,0,0.12)',
            border: '1px solid rgba(245,139,0,0.3)',
            display: 'flex',
            'align-items': 'center',
            'justify-content': 'center',
            'font-size': '22px',
        }}>
            ⊘
        </div>

        {/* Title */}
        <div style={{
            'font-family': 'var(--font-ui)',
            'font-size': '15px',
            'font-weight': '700',
            color: 'var(--text-heading)',
        }}>
            Not available in {contextLabel[APP_CONTEXT]} mode
        </div>

        {/* Reason */}
        <div style={{
            'font-family': 'var(--font-ui)',
            'font-size': '13px',
            color: 'var(--text-muted)',
            'max-width': '480px',
            'line-height': '1.6',
        }}>
            {props.reason}
        </div>

        {/* Hint for desktop users who want fleet features */}
        <Show when={IS_DESKTOP}>
            <div style={{
                'font-family': 'var(--font-mono)',
                'font-size': '11px',
                color: 'var(--text-muted)',
                background: 'var(--surface-2)',
                border: '1px solid var(--border-primary)',
                padding: '10px 16px',
                'border-radius': 'var(--radius-sm)',
                'max-width': '520px',
            }}>
                To access server-only features, configure a remote OBLIVRA server in{' '}
                <span style={{ color: 'var(--accent-primary)' }}>Settings → Server Connection</span>
                {' '}to enable Hybrid mode.
            </div>
        </Show>

        {/* Hint for browser users who want desktop features */}
        <Show when={!IS_DESKTOP && !IS_HYBRID}>
            <div style={{
                'font-family': 'var(--font-mono)',
                'font-size': '11px',
                color: 'var(--text-muted)',
                background: 'var(--surface-2)',
                border: '1px solid var(--border-primary)',
                padding: '10px 16px',
                'border-radius': 'var(--radius-sm)',
                'max-width': '520px',
            }}>
                Download and run the{' '}
                <span style={{ color: 'var(--accent-primary)' }}>Oblivra Desktop binary</span>
                {' '}for local PTY terminal, OS keychain, and direct SFTP access.
            </div>
        </Show>

        {/* Current context badge */}
        <div style={{
            display: 'flex',
            'align-items': 'center',
            gap: '6px',
            'font-family': 'var(--font-mono)',
            'font-size': '9px',
            'font-weight': '700',
            'text-transform': 'uppercase',
            'letter-spacing': '1px',
            color: 'var(--text-muted)',
            opacity: '0.5',
        }}>
            <span style={{
                width: '6px',
                height: '6px',
                'border-radius': '50%',
                background: IS_DESKTOP ? '#5cc05c' : IS_HYBRID ? '#f58b00' : '#0099e0',
                display: 'inline-block',
            }} />
            Running in {contextLabel[APP_CONTEXT]} mode · Route: {props.path}
        </div>
    </div>
);
