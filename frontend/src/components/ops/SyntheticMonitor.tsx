import { Component, createSignal, For, onCleanup, onMount, Show } from 'solid-js';
import { AddProbe, GetResults } from '../../../wailsjs/go/app/SyntheticService';
import { monitoring } from '../../../wailsjs/go/models';
import { Card, Badge, Button } from '../ui/TacticalComponents';

export const SyntheticMonitor: Component = () => {
    const [results, setResults] = createSignal<monitoring.ProbeResult[]>([]);
    const [name, setName] = createSignal('');
    const [target, setTarget] = createSignal('');
    const [type, setType] = createSignal('http');

    const refresh = async () => {
        const res = await GetResults();
        setResults(res || []);
    };

    onMount(() => {
        refresh();
        const interval = setInterval(refresh, 5000);
        onCleanup(() => clearInterval(interval));
    });

    const handleAdd = async () => {
        if (!name() || !target()) return;
        await AddProbe(name(), type(), target(), 60);
        setName('');
        setTarget('');
        refresh();
    };

    return (
        <div class="synthetic-monitor">
            <div class="probe-controls">
                <input class="ops-input" placeholder="Probe Name" value={name()} onInput={e => setName(e.currentTarget.value)} />
                <input class="ops-input" placeholder="Target (e.g. http://node-a:8080)" value={target()} onInput={e => setTarget(e.currentTarget.value)} />
                <select class="ops-select" value={type()} onChange={e => setType(e.currentTarget.value)}>
                    <option value="http">HTTP</option>
                    <option value="tcp">TCP</option>
                </select>
                <Button variant="primary" size="sm" onClick={handleAdd}>ADD PROBE</Button>
            </div>

            <div class="probe-list" style="margin-top: 20px; display: grid; grid-template-columns: repeat(auto-fill, minmax(300px, 1fr)); gap: 12px;">
                <For each={results()}>
                    {(r) => (
                        <Card variant="raised" padding="16px">
                            <div style="display: flex; justify-content: space-between; align-items: flex-start;">
                                <div style="display: flex; flex-direction: column;">
                                    <span style="font-weight: 800; font-size: 14px; letter-spacing: 0.5px;">{r.probe_id}</span>
                                    <span style="font-size: 11px; color: var(--text-muted); font-family: var(--font-mono);">{r.latency / 1e6}ms latency</span>
                                </div>
                                <Badge severity={r.status === 'up' ? 'success' : 'error'}>
                                    {r.status.toUpperCase()}
                                </Badge>
                            </div>
                            <Show when={r.error}>
                                <div style="margin-top: 8px; font-size: 11px; color: var(--alert-critical); font-family: var(--font-mono); overflow-wrap: break-word;">
                                    ERR: {r.error}
                                </div>
                            </Show>
                        </Card>
                    )}
                </For>
            </div>
        </div>
    );
};
