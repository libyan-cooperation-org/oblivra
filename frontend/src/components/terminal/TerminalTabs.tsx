import { Component, For } from 'solid-js';

interface TabInfo {
    id: string;
    title: string;
    hostLabel: string;
    status: 'active' | 'closed' | 'error';
    isRecording?: boolean;
}

interface TerminalTabsProps {
    tabs: TabInfo[];
    activeTabId: string | null;
    onSelect: (id: string) => void;
    onClose: (id: string) => void;
    onNewTab: () => void;
}

const statusIcon = (status: string) => {
    switch (status) {
        case 'active': return '🟢';
        case 'error': return '🔴';
        case 'closed': return '⚫';
        default: return '⚪';
    }
};

export const TerminalTabs: Component<TerminalTabsProps> = (props) => (
    <div class="terminal-tabs">
        <div class="tabs-scroll">
            <For each={props.tabs}>
                {(tab) => (
                    <div
                        class={`tab${props.activeTabId === tab.id ? ' active' : ''}`}
                        onClick={() => props.onSelect(tab.id)}
                        draggable={true}
                        title={`${tab.hostLabel} — ${tab.status}`}
                    >
                        <span class="tab-status">{statusIcon(tab.status)}</span>
                        <span class="tab-title">{tab.title || tab.hostLabel}</span>
                        {tab.isRecording && (
                            <span class="tab-rec-indicator" title="Internal Recording Active">
                                <span class="rec-dot"></span>
                                REC
                            </span>
                        )}
                        <button
                            id={`tab-close-${tab.id}`}
                            class="tab-close"
                            onClick={(e) => { e.stopPropagation(); props.onClose(tab.id); }}
                            title="Close session"
                        >
                            ✕
                        </button>
                    </div>
                )}
            </For>
        </div>
        <button
            id="btn-new-tab"
            class="tab-new"
            onClick={() => props.onNewTab()}
            title="New connection (Ctrl+T)"
        >
            +
        </button>
    </div>
);
