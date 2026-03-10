import { Component, Show, JSX } from 'solid-js';
import '../../styles/empty-state.css';

interface EmptyStateProps {
    icon: string;
    title: string;
    description: string;
    action?: string;
    onAction?: () => void;
    compact?: boolean;
    children?: JSX.Element;
}

export const EmptyState: Component<EmptyStateProps> = (props) => {
    return (
        <div class={`empty-state-container ${props.compact ? 'compact' : ''}`}>
            <div class="empty-state-icon-ring">
                <span class="empty-state-icon">{props.icon}</span>
            </div>
            <h3 class="empty-state-title">{props.title}</h3>
            <p class="empty-state-desc">{props.description}</p>
            <Show when={props.action && props.onAction}>
                <button class="empty-state-action" onClick={props.onAction}>
                    {props.action}
                </button>
            </Show>
            <Show when={props.children}>
                {props.children}
            </Show>
        </div>
    );
};
