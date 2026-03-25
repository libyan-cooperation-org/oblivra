import { Component, For, Show, JSX, createSignal, splitProps } from 'solid-js';

/* ═══════════════════════════════════════════════════════════════
   OBLIVRA Table — Dense Splunk-style data table
   Wraps .ob-table-wrap + .ob-table from components.css
   ═══════════════════════════════════════════════════════════════ */

export type SortDir = 'asc' | 'desc' | null;

export interface Column<T = any> {
    key: string;
    label: string;
    width?: string;
    mono?: boolean;
    sortable?: boolean;
    render?: (row: T, idx: number) => JSX.Element;
}

interface TableProps<T = any> {
    columns: Column<T>[];
    data: T[];
    striped?: boolean;
    selectedId?: string | null;
    idKey?: string;
    onRowClick?: (row: T, idx: number) => void;
    onSort?: (key: string, dir: SortDir) => void;
    emptyText?: string;
    class?: string;
    stickyHeader?: boolean;
}

export const Table: Component<TableProps> = (props) => {
    const [sortKey, setSortKey] = createSignal<string | null>(null);
    const [sortDir, setSortDir] = createSignal<SortDir>(null);

    const handleSort = (col: Column) => {
        if (!col.sortable) return;
        let nextDir: SortDir = 'asc';
        if (sortKey() === col.key) {
            nextDir = sortDir() === 'asc' ? 'desc' : sortDir() === 'desc' ? null : 'asc';
        }
        setSortKey(nextDir ? col.key : null);
        setSortDir(nextDir);
        props.onSort?.(col.key, nextDir);
    };

    const sortIndicator = (col: Column) => {
        if (!col.sortable) return '';
        if (sortKey() !== col.key) return ' ↕';
        return sortDir() === 'asc' ? ' ↑' : ' ↓';
    };

    return (
        <div class={`ob-table-wrap ${props.class || ''}`}>
            <table class={`ob-table${props.striped ? ' striped' : ''}`}>
                <thead>
                    <tr>
                        <For each={props.columns}>
                            {(col) => (
                                <th
                                    style={col.width ? { width: col.width, 'min-width': col.width } : undefined}
                                    onClick={() => handleSort(col)}
                                    class={col.sortable ? 'sortable' : ''}
                                >
                                    {col.label}{sortIndicator(col)}
                                </th>
                            )}
                        </For>
                    </tr>
                </thead>
                <tbody>
                    <Show when={props.data.length > 0} fallback={
                        <tr>
                            <td
                                colSpan={props.columns.length}
                                style="text-align: center; padding: 32px; color: var(--text-muted); font-size: 12px;"
                            >
                                {props.emptyText || 'NO DATA'}
                            </td>
                        </tr>
                    }>
                        <For each={props.data}>
                            {(row, idx) => {
                                const idKey = props.idKey || 'id';
                                const isSelected = () => props.selectedId != null && (row as any)[idKey] === props.selectedId;
                                return (
                                    <tr
                                        class={isSelected() ? 'selected' : ''}
                                        onClick={() => props.onRowClick?.(row, idx())}
                                        style={props.onRowClick ? { cursor: 'pointer' } : undefined}
                                    >
                                        <For each={props.columns}>
                                            {(col) => (
                                                <td class={col.mono ? 'mono' : ''}>
                                                    {col.render
                                                        ? col.render(row, idx())
                                                        : String((row as any)[col.key] ?? '')}
                                                </td>
                                            )}
                                        </For>
                                    </tr>
                                );
                            }}
                        </For>
                    </Show>
                </tbody>
            </table>
        </div>
    );
};
