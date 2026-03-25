import { Component, For, Show } from 'solid-js';

interface HistogramBin {
    count: number;
    heightPct: number;
    timeLabel: string;
}

interface HistogramProps {
    data: HistogramBin[];
    color?: string;
    height?: number;
    class?: string;
}

export const Histogram: Component<HistogramProps> = (props) => {
    const color = () => props.color || 'var(--accent-primary)';
    
    return (
        <div 
            class={`ob-histogram ${props.class || ''}`}
            style={{
                height: `${props.height || 100}px`,
                display: 'flex',
                'align-items': 'flex-end',
                gap: '2px',
                padding: '0 0 var(--gap-sm) 0',
                'border-bottom': '1px solid var(--border-primary)',
                width: '100%',
                'flex-shrink': 0,
                position: 'relative'
            }}
        >
            <Show when={props.data.length === 0}>
                <div style="position: absolute; inset: 0; display: flex; align-items: center; justify-content: center; color: var(--text-muted); font-size: 11px; font-family: var(--font-mono);">
                    NO_TIMELINE_DATA_AVAILABLE
                </div>
            </Show>
            
            <For each={props.data}>
                {(bin) => (
                    <div 
                        title={`Time: ${bin.timeLabel} | Count: ${bin.count}`}
                        style={{
                            flex: 1,
                            background: bin.count > 0 ? color() : 'transparent',
                            opacity: 0.8,
                            height: `${bin.heightPct}%`,
                            'min-height': bin.count > 0 ? '4px' : '0',
                            cursor: 'pointer',
                            transition: 'opacity 0.2s, height 0.6s ease-out'
                        }}
                        onMouseEnter={e => e.currentTarget.style.opacity = '1'}
                        onMouseLeave={e => e.currentTarget.style.opacity = '0.8'}
                    />
                )}
            </For>
        </div>
    );
};
