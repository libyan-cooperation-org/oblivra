import { Component, Show, For, createSignal } from 'solid-js';
import { Button } from './TacticalComponents';
import '../../styles/modal.css';

export interface ModalAction {
    label: string;
    onClick: () => void;
    type?: 'primary' | 'secondary' | 'danger';
}

export interface ModalOptions {
    title: string;
    message: string;
    confirmText?: string;
    cancelText?: string;
    type?: 'info' | 'warning' | 'error' | 'danger';
    onConfirm: (inputValue?: string) => void;
    onCancel: () => void;
    actions?: ModalAction[];
    showInput?: boolean;
    inputPlaceholder?: string;
    initialValue?: string;
}

const [currentModal, setCurrentModal] = createSignal<ModalOptions | null>(null);
const [inputValue, setInputValue] = createSignal('');

export function showModal(options: ModalOptions) {
    setInputValue(options.initialValue || '');
    setCurrentModal(options);
}

export const ModalSystem: Component = () => {
    const handleConfirm = () => {
        currentModal()?.onConfirm(inputValue());
        setCurrentModal(null);
    };

    const handleCancel = () => {
        currentModal()?.onCancel();
        setCurrentModal(null);
    };

    return (
        <Show when={currentModal()}>
            {(modal) => (
                <div class="modal-overlay" onClick={handleCancel}>
                    <div class="modal-glass" onClick={(e) => e.stopPropagation()}>
                        <div class="modal-header">
                            <span class={`modal-type-icon ${modal().type || 'info'}`}>
                                {modal().type === 'danger' || modal().type === 'error' ? '⚠' : 'ℹ'}
                            </span>
                            <h3 class="modal-title">{modal().title}</h3>
                        </div>
                        <div class="modal-body">
                            <p class="modal-message">{modal().message}</p>
                            <Show when={modal().showInput}>
                                <input
                                    type="text"
                                    class="modal-input"
                                    placeholder={modal().inputPlaceholder || 'Enter value...'}
                                    value={inputValue()}
                                    onInput={(e) => setInputValue(e.currentTarget.value)}
                                    onKeyDown={(e) => {
                                        if (e.key === 'Enter') handleConfirm();
                                        if (e.key === 'Escape') handleCancel();
                                    }}
                                    autofocus
                                />
                            </Show>
                        </div>
                        <div class="modal-footer">
                            <Button variant="ghost" onClick={handleCancel}>
                                {modal().cancelText || 'Cancel'}
                            </Button>
                            <Show when={modal().actions}>
                                <For each={modal().actions}>
                                    {(action: ModalAction) => (
                                        <Button
                                            variant={action.type === 'danger' ? 'danger' : (action.type === 'primary' ? 'primary' : 'secondary')}
                                            onClick={() => { action.onClick(); setCurrentModal(null); }}
                                        >
                                            {action.label}
                                        </Button>
                                    )}
                                </For>
                            </Show>
                            <Show when={!modal().actions || modal().actions!.length === 0}>
                                <Button
                                    variant={modal().type === 'danger' || modal().type === 'error' ? 'danger' : 'primary'}
                                    onClick={handleConfirm}
                                >
                                    {modal().confirmText || 'Confirm'}
                                </Button>
                            </Show>
                        </div>
                    </div>
                </div>
            )}
        </Show>
    );
};
