import { Component, Show, JSX } from 'solid-js';

/* ═══════════════════════════════════════════════════════════════
   OBLIVRA Page Layout — Standard page chrome
   Wraps .ob-page / .ob-page-header from components.css
   
   Usage:
     <PageLayout
       title="Active Threats"
       subtitle="AGGREGATED_THREAT_DETECTION"
       actions={<button class="ob-btn ob-btn-primary">Deploy</button>}
     >
       {content}
     </PageLayout>
   ═══════════════════════════════════════════════════════════════ */

interface PageLayoutProps {
    title: string;
    subtitle?: string;
    breadcrumb?: { label: string; href?: string }[];
    actions?: JSX.Element;
    children: JSX.Element;
    class?: string;
    noPadding?: boolean;
}

export const PageLayout: Component<PageLayoutProps> = (props) => (
    <div class={`ob-page ${props.class || ''}`}>
        <div class="ob-page-header">
            <div>
                <Show when={props.breadcrumb && props.breadcrumb.length > 0}>
                    <div class="ob-breadcrumb">
                        {props.breadcrumb!.map((crumb, i) => (
                            <>
                                {i > 0 && <span class="ob-breadcrumb-sep">›</span>}
                                {crumb.href
                                    ? <a href={crumb.href}>{crumb.label}</a>
                                    : <span>{crumb.label}</span>
                                }
                            </>
                        ))}
                    </div>
                </Show>
                <div class="ob-page-title">{props.title}</div>
                <Show when={props.subtitle}>
                    <div class="ob-page-subtitle">{props.subtitle}</div>
                </Show>
            </div>
            <Show when={props.actions}>
                <div style="display: flex; gap: 8px; align-items: center;">
                    {props.actions}
                </div>
            </Show>
        </div>
        <div style={props.noPadding ? "flex: 1; min-height: 0; display: flex; flex-direction: column;" : "padding: 16px 20px; flex: 1; min-height: 0; display: flex; flex-direction: column; gap: 16px;"}>
            {props.children}
        </div>
    </div>
);

/* ═══════════════════════════════════════════════════════════════
   OBLIVRA Notice — Alert/notification strip
   Wraps .ob-notice from components.css
   ═══════════════════════════════════════════════════════════════ */

type NoticeLevel = 'info' | 'warn' | 'error' | 'success';

interface NoticeProps {
    level: NoticeLevel;
    children: JSX.Element;
    class?: string;
}

const levelClass: Record<NoticeLevel, string> = {
    info:    'ob-notice-info',
    warn:    'ob-notice-warn',
    error:   'ob-notice-error',
    success: 'ob-notice-success',
};

export const Notice: Component<NoticeProps> = (props) => (
    <div class={`ob-notice ${levelClass[props.level]} ${props.class || ''}`}>
        {props.children}
    </div>
);

/* ═══════════════════════════════════════════════════════════════
   OBLIVRA Code — Code/log display block
   Wraps .ob-code from components.css
   ═══════════════════════════════════════════════════════════════ */

interface CodeBlockProps {
    children: JSX.Element;
    class?: string;
}

export const CodeBlock: Component<CodeBlockProps> = (props) => (
    <pre class={`ob-code ${props.class || ''}`}>{props.children}</pre>
);

/* ═══════════════════════════════════════════════════════════════
   OBLIVRA Progress Bar
   Wraps .ob-progress from components.css
   ═══════════════════════════════════════════════════════════════ */

interface ProgressProps {
    value: number; // 0-100
    color?: 'blue' | 'orange' | 'green' | 'red';
    class?: string;
}

export const Progress: Component<ProgressProps> = (props) => {
    const barClass = () => {
        if (props.color === 'orange') return 'ob-progress-bar orange';
        if (props.color === 'green') return 'ob-progress-bar green';
        if (props.color === 'red') return 'ob-progress-bar red';
        return 'ob-progress-bar';
    };

    return (
        <div class={`ob-progress ${props.class || ''}`}>
            <div class={barClass()} style={`width: ${Math.min(100, Math.max(0, props.value))}%;`} />
        </div>
    );
};
