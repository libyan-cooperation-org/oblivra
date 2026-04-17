import { subscribe } from '@lib/bridge';

export interface DiagnosticsSnapshot {
  captured_at: string;
  event_bus: {
    dropped_events: number;
    rate_limit_active: boolean;
    lag_estimate_ms: number;
  };
  ingest: {
    current_eps: number;
    target_eps: number;
    percent_of_target: number;
    buffer_fill_pct: number;
    dropped_total: number;
    worker_count: number;
  };
  runtime: {
    goroutines: number;
    heap_alloc_mb: number;
    heap_sys_mb: number;
    gc_pause_ns: number;
    gc_count: number;
    num_cpu: number;
    go_version: string;
  };
  query: {
    last_query_ms: number;
    avg_query_ms: number;
    p99_query_ms: number;
    slow_query_count: number;
    total_queries: number;
  };
  health_grade: string;
}

export class DiagnosticsStore {
  snapshot = $state<DiagnosticsSnapshot | null>(null);
  connected = $state(false);

  constructor() {
    this.init();
  }

  private init() {
    subscribe<DiagnosticsSnapshot>('diagnostics:snapshot', (snap) => {
      this.snapshot = snap;
      this.connected = true;
    });

    subscribe<{status: string, message: string}>('health_status_changed', (data) => {
      console.warn(`[Diagnostics] Health Status Changed: ${data.status} - ${data.message}`);
      // We could trigger a toast here, but the snapshot already updates the grade.
    });
  }

  get healthGrade() {
    return this.snapshot?.health_grade ?? 'PENDING';
  }

  get eps() {
    return this.snapshot?.ingest.current_eps ?? 0;
  }

  get bufferFill() {
    return this.snapshot?.ingest.buffer_fill_pct ?? 0;
  }
}

export const diagnosticsStore = new DiagnosticsStore();
