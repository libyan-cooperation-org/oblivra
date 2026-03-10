import { Component, For, createSignal } from 'solid-js';
import '../../styles/log_detail.css';

const extractFields = (message: string) => {
    const fields: { key: string, value: string }[] = [];
    const regex = /(\w+)=("[^"]*"|\S+)/g;
    let match;
    while ((match = regex.exec(message)) !== null) {
        fields.push({ key: match[1], value: match[2].replace(/"/g, '') });
    }
    return fields;
};

interface LogDetailProps {
    log: {
        output?: string,
        message?: string,
        timestamp?: string | number,
        host?: string,
        hostname?: string,
        session_id?: string,
        level?: string
    };
    onAddFilter: (key: string, value: string, operator: '=' | '!=') => void;
}

export const LogDetail: Component<LogDetailProps> = (props) => {
    const fields = () => extractFields(props.log.output || '');
    const [expanded, setExpanded] = createSignal(false);

    return (
        <div class="log-entry">
            <div class="log-summary" onClick={() => setExpanded(!expanded())}>
                <span class="timestamp">{new Date(props.log.timestamp || Date.now()).toLocaleTimeString()}</span>
                <span class="host">{props.log.host}</span>
                <span class="message">{props.log.output}</span>
            </div>

            {expanded() && (
                <div class="log-details">
                    <h4>Extracted Fields</h4>
                    <div class="fields-grid">
                        <For each={fields()}>
                            {(field) => (
                                <div class="field-item">
                                    <span class="key">{field.key}</span>
                                    <span class="value">{field.value}</span>

                                    <div class="field-actions">
                                        <button
                                            title="Add to search"
                                            onClick={(e) => { e.stopPropagation(); props.onAddFilter(field.key, field.value, '='); }}
                                        >🔍+</button>
                                        <button
                                            title="Exclude from search"
                                            onClick={(e) => { e.stopPropagation(); props.onAddFilter(field.key, field.value, '!='); }}
                                        >🔍-</button>
                                    </div>
                                </div>
                            )}
                        </For>
                        <div class="field-item">
                            <span class="key">host</span>
                            <span class="value">{props.log.host || '—'}</span>
                            <div class="field-actions">
                                <button onClick={(e) => { e.stopPropagation(); props.onAddFilter('host', props.log.host || '', '='); }}>🔍+</button>
                            </div>
                        </div>
                        <div class="field-item">
                            <span class="key">session</span>
                            <span class="value">{props.log.session_id || '—'}</span>
                            <div class="field-actions">
                                <button onClick={(e) => { e.stopPropagation(); props.onAddFilter('session', props.log.session_id || '', '='); }}>🔍+</button>
                            </div>
                        </div>
                    </div>
                </div>
            )}
        </div>
    );
};
