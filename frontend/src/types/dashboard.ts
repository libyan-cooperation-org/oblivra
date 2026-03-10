export type WidgetType = 'line' | 'bar' | 'pie' | 'metric' | 'table' | 'log-stream';

export interface Widget {
    id: string;
    title: string;
    type: WidgetType;
    query: string;
    source: string; // can be "analytics", "osquery", or an external source ID
    refreshInterval: number; // seconds (0 = manual)
    layout: { x: number; y: number; w: number; h: number };
}

export interface Dashboard {
    id: string;
    name: string;
    widgets: Widget[];
}
