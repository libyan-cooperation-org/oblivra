<!--
  OBLIVRA — DataTable (Svelte 5)
  Sortable data table used across SIEM, fleet, compliance, etc.
-->
<script lang="ts" generics="T extends Record<string, any>">
  import type { Snippet } from 'svelte';

  interface Column<T> {
    key: keyof T & string;
    label: string;
    width?: string;
    align?: 'left' | 'center' | 'right';
    sortable?: boolean;
    render?: Snippet<[{ value: any; row: T; index: number }]>;
  }

  interface Props {
    columns: Column<T>[];
    data: T[];
    rowKey?: keyof T & string;
    emptyMessage?: string;
    compact?: boolean;
    striped?: boolean;
    onRowClick?: (row: T, index: number) => void;
    render?: Snippet<[{ value: any; col: Column<T>; row: T; index: number }]>;
  }

  let {
    columns,
    data,
    rowKey,
    emptyMessage = 'No data',
    compact = false,
    striped = false,
    onRowClick,
    render: globalRender,
  }: Props = $props();

  let sortKey = $state<string | null>(null);
  let sortDir = $state<'asc' | 'desc'>('asc');

  function handleSort(key: string) {
    if (sortKey === key) {
      sortDir = sortDir === 'asc' ? 'desc' : 'asc';
    } else {
      sortKey = key;
      sortDir = 'asc';
    }
  }

  const sortedData = $derived.by(() => {
    if (!sortKey) return data;
    const dir = sortDir === 'asc' ? 1 : -1;
    return [...data].sort((a, b) => {
      const aVal = a[sortKey!] ?? '';
      const bVal = b[sortKey!] ?? '';
      if (typeof aVal === 'number' && typeof bVal === 'number') return (aVal - bVal) * dir;
      return String(aVal).localeCompare(String(bVal)) * dir;
    });
  });
</script>

<div class="w-full overflow-auto border border-border-primary rounded-sm">
  <table class="w-full border-collapse">
    <!-- Header -->
    <thead>
      <tr class="border-b border-border-primary bg-surface-1">
        {#each columns as col}
          <th
            class="font-[var(--font-mono)] text-[10px] font-bold uppercase tracking-wider text-text-muted text-left whitespace-nowrap select-none
              {compact ? 'px-2.5 py-1.5' : 'px-3 py-2'}
              {col.align === 'right' ? 'text-right' : col.align === 'center' ? 'text-center' : 'text-left'}
              {col.sortable !== false ? 'cursor-pointer hover:text-text-secondary hover:bg-surface-2 transition-colors duration-fast focus:outline-hidden focus:bg-surface-2' : ''}"
            style={col.width ? `width: ${col.width}` : ''}
            onclick={() => col.sortable !== false && handleSort(col.key)}
            onkeydown={(e) => col.sortable !== false && (e.key === 'Enter' || e.key === ' ') && (e.preventDefault(), handleSort(col.key))}
            role={col.sortable !== false ? "button" : undefined}
            tabindex={col.sortable !== false ? 0 : undefined}
            aria-sort={sortKey === col.key ? (sortDir === 'asc' ? 'ascending' : 'descending') : undefined}
          >
            <span class="inline-flex items-center gap-1">
              {col.label}
              {#if sortKey === col.key}
                <span class="text-accent text-[8px]" aria-hidden="true">{sortDir === 'asc' ? '▲' : '▼'}</span>
              {/if}
            </span>
          </th>
        {/each}
      </tr>
    </thead>

    <!-- Body -->
    <tbody>
      {#if sortedData.length === 0}
        <tr>
          <td
            colspan={columns.length}
            class="text-center text-text-muted text-xs py-8 font-[var(--font-ui)]"
          >
            {emptyMessage}
          </td>
        </tr>
      {:else}
        {#each sortedData as row, i (rowKey ? (row[rowKey] as string) : i)}
          <tr
            class="border-b border-border-subtle transition-colors duration-fast
              {onRowClick ? 'cursor-pointer hover:bg-surface-3/50 focus:bg-surface-3/50 focus:outline-hidden' : ''}
              {striped && i % 2 === 1 ? 'bg-surface-1/30' : ''}
              {!onRowClick ? 'hover:bg-surface-3/20' : ''}"
            onclick={() => onRowClick?.(row, i)}
            onkeydown={(e) => onRowClick && (e.key === 'Enter' || e.key === ' ') && (e.preventDefault(), onRowClick(row, i))}
            role={onRowClick ? "button" : undefined}
            tabindex={onRowClick ? 0 : undefined}
          >
            {#each columns as col}
              <td
                class="text-text-primary text-xs font-[var(--font-ui)]
                  {compact ? 'px-2.5 py-1' : 'px-3 py-2'}
                  {col.align === 'right' ? 'text-right' : col.align === 'center' ? 'text-center' : 'text-left'}"
              >
                {#if globalRender}
                  {@render globalRender({ value: row[col.key], col, row, index: i })}
                {:else if col.render}
                  {@render col.render({ value: row[col.key], row, index: i })}
                {:else}
                  {row[col.key] ?? '—'}
                {/if}
              </td>
            {/each}
          </tr>
        {/each}
      {/if}
    </tbody>
  </table>
</div>
