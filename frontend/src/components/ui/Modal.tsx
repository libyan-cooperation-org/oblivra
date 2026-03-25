import { Component, Show, For, createSignal, JSX } from 'solid-js';

/* ═══════════════════════════════════════════════════════════════
   OBLIVRA Modal — Standard modal dialog
   Wraps .ob-modal from components.css
   ═══════════════════════════════════════════════════════════════ */

interface ModalProps {
    open: boolean;
    onClose: () => void;
    title: string;
    children: JSX.Element;
    actions?: JSX.Element;
    width?: string;
}

export const Modal: Component<ModalProps> = (props) => (
    <Show when={props.open}>
        <div class="ob-modal-overlay" onClick={(e) => {
            if (e.target === e.currentTarget) props.onClose();
        }}>
            <div class="ob-modal" style={props.width ? `min-width: ${props.width}; max-width: ${props.width};` : undefined}>
                <div class="ob-modal-header">
                    <h2>{props.title}</h2>
                    <button
                        class="ob-btn ob-btn-ghost ob-btn-icon"
                        onClick={props.onClose}
                        title="Close"
                    >
                        ✕
                    </button>
                </div>
                <div class="ob-modal-body">
                    {props.children}
                </div>
                <Show when={props.actions}>
                    <div class="ob-modal-actions">
                        {props.actions}
                    </div>
                </Show>
            </div>
        </div>
    </Show>
);

/* ═══════════════════════════════════════════════════════════════
   OBLIVRA Dropdown — Context menu / action dropdown
   Wraps .ob-dropdown from components.css
   ═══════════════════════════════════════════════════════════════ */

interface DropdownItem {
    id: string;
    label: string;
    danger?: boolean;
    divider?: boolean;
    icon?: JSX.Element;
}

interface DropdownProps {
    items: DropdownItem[];
    onSelect: (id: string) => void;
    trigger: JSX.Element;
    class?: string;
}

export const Dropdown: Component<DropdownProps> = (props) => {
    const [open, setOpen] = createSignal(false);

    return (
        <div style="position: relative; display: inline-block;" class={props.class || ''}>
            <div onClick={() => setOpen(!open())}>
                {props.trigger}
            </div>
            <Show when={open()}>
                <div
                    class="ob-dropdown"
                    style="top: 100%; right: 0; margin-top: 4px;"
                    onMouseLeave={() => setOpen(false)}
                >
                    <For each={props.items}>
                        {(item) => (
                            <Show
                                when={!item.divider}
                                fallback={<div class="ob-dropdown-divider" />}
                            >
                                <button
                                    class={`ob-dropdown-item${item.danger ? ' danger' : ''}`}
                                    onClick={() => {
                                        props.onSelect(item.id);
                                        setOpen(false);
                                    }}
                                >
                                    <Show when={item.icon}>{item.icon}</Show>
                                    {item.label}
                                </button>
                            </Show>
                        )}
                    </For>
                </div>
            </Show>
        </div>
    );
};

/* ═══════════════════════════════════════════════════════════════
   OBLIVRA Form — Typed form components
   Wraps .ob-form from components.css
   ═══════════════════════════════════════════════════════════════ */

interface FormFieldProps {
    label: string;
    help?: string;
    children: JSX.Element;
    class?: string;
}

export const FormField: Component<FormFieldProps> = (props) => (
    <div class={`ob-form-field ${props.class || ''}`}>
        <label>{props.label}</label>
        {props.children}
        <Show when={props.help}>
            <span class="ob-form-help">{props.help}</span>
        </Show>
    </div>
);

interface FormRowProps {
    children: JSX.Element;
}

export const FormRow: Component<FormRowProps> = (props) => (
    <div class="ob-form-row">{props.children}</div>
);
