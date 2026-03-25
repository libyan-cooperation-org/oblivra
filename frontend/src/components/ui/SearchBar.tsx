import { Component, JSX, Show } from 'solid-js';

/* ═══════════════════════════════════════════════════════════════
   OBLIVRA SearchBar — Splunk-style query input with action button
   Wraps .ob-search-bar from components.css
   ═══════════════════════════════════════════════════════════════ */

interface SearchBarProps {
    value: string;
    onInput: (value: string) => void;
    onSubmit?: () => void;
    placeholder?: string;
    buttonText?: string;
    class?: string;
    prefix?: JSX.Element;
    suffix?: JSX.Element;
}

export const SearchBar: Component<SearchBarProps> = (props) => {
    const handleKeyDown = (e: KeyboardEvent) => {
        if (e.key === 'Enter') {
            e.preventDefault();
            props.onSubmit?.();
        }
    };

    return (
        <div class={`ob-search-bar ${props.class || ''}`} style="display: flex; align-items: stretch; gap: 0;">
            <Show when={props.prefix}>
                <div class="ob-search-prefix" style="display: flex; align-items: center; padding: 0 var(--gap-md); border-right: 1px solid var(--border-primary); background: var(--surface-2); border-left: 1px solid var(--border-primary); border-top: 1px solid var(--border-primary); border-bottom: 1px solid var(--border-primary); border-top-left-radius: var(--radius-sm); border-bottom-left-radius: var(--radius-sm);">
                    {props.prefix}
                </div>
            </Show>
            <div style="flex: 1; position: relative; display: flex;">
                <input
                    type="text"
                    value={props.value}
                    onInput={(e) => props.onInput(e.currentTarget.value)}
                    onKeyDown={handleKeyDown}
                    placeholder={props.placeholder || 'Search…'}
                    style={{
                        flex: '1',
                        background: 'var(--surface-3)',
                        border: '1px solid var(--border-primary)',
                        'border-left': props.prefix ? 'none' : '1px solid var(--border-primary)',
                        color: 'var(--text-primary)',
                        'font-family': 'var(--font-mono)',
                        'font-size': '12px',
                        padding: '8px 12px',
                        outline: 'none',
                        'border-top-left-radius': props.prefix ? '0' : 'var(--radius-sm)',
                        'border-bottom-left-radius': props.prefix ? '0' : 'var(--radius-sm)'
                    }}
                />
            </div>
            <Show when={props.suffix}>
                <div class="ob-search-suffix" style="display: flex; align-items: center; border-top: 1px solid var(--border-primary); border-bottom: 1px solid var(--border-primary); background: var(--surface-2); border-right: 1px solid var(--border-primary);">
                    {props.suffix}
                </div>
            </Show>
            <button 
                class="ob-search-btn" 
                onClick={() => props.onSubmit?.()}
                style={{
                    padding: '0 24px',
                    'font-family': 'var(--font-mono)',
                    'font-size': '11px',
                    'font-weight': 800,
                    'text-transform': 'uppercase',
                    'letter-spacing': '1px'
                }}
            >
                {props.buttonText || 'Search'}
            </button>
        </div>
    );
};

/* ═══════════════════════════════════════════════════════════════
   OBLIVRA FilterBar — Toolbar with filter buttons
   Wraps .ob-toolbar from components.css
   ═══════════════════════════════════════════════════════════════ */

interface ToolbarProps {
    children: JSX.Element;
    class?: string;
}

export const Toolbar: Component<ToolbarProps> = (props) => (
    <div class={`ob-toolbar ${props.class || ''}`}>
        {props.children}
    </div>
);

export const ToolbarSpacer: Component = () => (
    <div class="ob-toolbar-spacer" />
);

/* ═══════════════════════════════════════════════════════════════
   OBLIVRA Tabs
   Wraps .ob-tabs + .ob-tab from components.css
   ═══════════════════════════════════════════════════════════════ */

interface Tab {
    id: string;
    label: string;
    badge?: number;
}

interface TabBarProps {
    tabs: Tab[];
    active: string;
    onSelect: (id: string) => void;
    class?: string;
}

export const TabBar: Component<TabBarProps> = (props) => (
    <div class={`ob-tabs ${props.class || ''}`}>
        {props.tabs.map((tab) => (
            <button
                class={`ob-tab${props.active === tab.id ? ' active' : ''}`}
                onClick={() => props.onSelect(tab.id)}
            >
                {tab.label}
                {tab.badge != null && tab.badge > 0 && (
                    <span style="margin-left: 6px; font-family: var(--font-mono); font-size: 9px; background: var(--surface-3); padding: 1px 5px; border-radius: var(--radius-full); border: 1px solid var(--border-primary);">
                        {tab.badge}
                    </span>
                )}
            </button>
        ))}
    </div>
);
