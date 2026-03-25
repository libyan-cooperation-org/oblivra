import { Component, JSX, splitProps, For, Show } from 'solid-js';

/* ═══════════════════════════════════════════════════════════════
   OBLIVRA Form Inputs — Standard form controls
   Wraps .ob-input, .ob-select, .ob-textarea from components.css
   ═══════════════════════════════════════════════════════════════ */

interface InputProps extends JSX.InputHTMLAttributes<HTMLInputElement> {
    mono?: boolean;
}

export const Input: Component<InputProps> = (props) => {
    const [local, rest] = splitProps(props, ['mono', 'class']);
    return (
        <input
            class={`ob-input ${local.mono ? 'ob-input-mono' : ''} ${local.class || ''}`}
            {...rest}
            style={{
                'font-family': local.mono ? 'var(--font-mono)' : 'var(--font-ui)'
            }}
        />
    );
};

interface SelectOption {
    label: string | number;
    value: string | number;
}

interface SelectProps extends Omit<JSX.SelectHTMLAttributes<HTMLSelectElement>, 'onChange'> {
    options?: (string | number | SelectOption)[];
    onChange?: (value: string) => void;
    class?: string;
    variant?: 'default' | 'outline' | 'ghost';
}

export const Select: Component<SelectProps> = (props) => {
    const [local, rest] = splitProps(props, ['options', 'onChange', 'class', 'variant', 'children']);
    
    const formattedOptions = () => local.options?.map(opt => 
        (typeof opt === 'object' ? opt : { label: String(opt), value: String(opt) })
    ) || [];

    return (
        <div class="ob-select-wrap" style="position: relative; display: flex; align-items: center;">
            <select
                class={`ob-select ${local.class || ''}`}
                {...rest}
                onChange={(e) => local.onChange?.(e.currentTarget.value)}
                style={{
                    appearance: 'none',
                    background: 'var(--surface-3)',
                    border: '1px solid var(--border-primary)',
                    'border-radius': 'var(--radius-sm)',
                    color: 'var(--accent-primary)',
                    'font-family': 'var(--font-mono)',
                    'font-size': '11px',
                    'font-weight': '700',
                    padding: '4px 28px 4px 10px',
                    cursor: 'pointer',
                    outline: 'none'
                }}
            >
                <Show when={local.children}>{local.children}</Show>
                <For each={formattedOptions()}>
                    {(opt) => (
                        <option value={opt.value} style="background: var(--surface-2); color: var(--text-primary);">
                            {opt.label}
                        </option>
                    )}
                </For>
            </select>
            <div style="position: absolute; right: 8px; pointer-events: none; color: var(--accent-primary); font-size: 8px;">▼</div>
        </div>
    );
};

export const TextArea: Component<JSX.TextareaHTMLAttributes<HTMLTextAreaElement>> = (props) => {
    const [local, rest] = splitProps(props, ['class']);
    return (
        <textarea
            class={`ob-textarea ob-input ${local.class || ''}`}
            {...rest}
        />
    );
};
