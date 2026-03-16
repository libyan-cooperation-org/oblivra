import { Component, createSignal, onMount, onCleanup, Show, For } from 'solid-js';
import { EventsOn, EventsOff } from '../../../wailsjs/runtime/runtime';

// ── Types mirroring internal/monitoring/diagnostics.go ──────────────────────

interface EventBusDiag {
  dropped_events: number;
  rate_limit_active: boolean;
  lag_estimate_ms: number;
}

interface IngestDiag {
  current_eps: number;
  target_eps: number;
  percent_of_target: number;
  buffer_fill_pct: number;
  dropped_total: number;
  worker_count: number;
}

interface RuntimeDiag {
  goroutines: number;
  heap_alloc_mb: number;
  heap_sys_mb: number;
  gc_pause_ns: number;
  gc_count: number;
  num_cpu: number;
  go_version: string;
}

interface QueryDiag {
  last_query_ms: number;
  avg_query_ms: number;
  p99_query_ms: number;
  slow_query_count: number;
  total_queries: number;
}

interface DiagnosticsSnapshot {
  captured_at: string;
  event_bus: EventBusDiag;
  ingest: IngestDiag;
  runtime: RuntimeDiag;
  query: QueryDiag;
  health_grade: string;
}

interface DiagnosticsModalProps {
  onClose: () => void;
}

// ── Grade helpers ─────────────────────────────────────────────────────────────

const gradeColor = (grade: string) => {
  switch (grade) {
    case 'A': return '#3fb950';
    case 'B': return '#d29922';
    case 'C': return '#f0883e';
    default:  return '#f85149'; // DEGRADED
  }
};

const gradeLabel = (grade: string) => {
  switch (grade) {
    case 'A': return 'Nominal';
    case 'B': return 'Minor Stress';
    case 'C': return 'Moderate Load';
    default:  return 'DEGRADED';
  }
};

// ── Sub-components ────────────────────────────────────────────────────────────

const GaugeBar: Component<{ label: string; value: number; max: number; unit?: string; warn?: number; crit?: number }> = (p) => {
  const pct = () => Math.min(100, (p.value / p.max) * 100);
  const color = () => {
    if (p.crit !== undefined && p.value >= p.crit) return '#f85149';
    if (p.warn !== undefined && p.value >= p.warn) return '#d29922';
    return '#3fb950';
  };
  return (
    <div style="margin-bottom: 10px;">
      <div style="display: flex; justify-content: space-between; font-size: 11px; margin-bottom: 3px; font-family: var(--font-mono);">
        <span style="color: var(--text-secondary);">{p.label}</span>
        <span style={`color: ${color()};`}>{p.value.toFixed(p.unit === 'ms' ? 1 : 0)}{p.unit ?? ''}</span>
      </div>
      <div style="height: 4px; background: rgba(255,255,255,0.08); border-radius: 2px; overflow: hidden;">
        <div style={`height: 100%; width: ${pct()}%; background: ${color()}; transition: width 0.4s ease, background 0.3s;`} />
      </div>
    </div>
  );
};

const StatRow: Component<{ label: string; value: string | number; accent?: boolean }> = (p) => (
  <div style="display: flex; justify-content: space-between; align-items: center; padding: 4px 0; border-bottom: 1px solid rgba(255,255,255,0.04); font-size: 11px; font-family: var(--font-mono);">
    <span style="color: var(--text-muted); text-transform: uppercase; letter-spacing: 0.5px;">{p.label}</span>
    <span style={p.accent ? 'color: var(--accent-primary); font-weight: 700;' : 'color: var(--text-primary);'}>{p.value}</span>
  </div>
);

const SectionHeader: Component<{ title: string; icon: string }> = (p) => (
  <div style="display: flex; align-items: center; gap: 8px; margin: 16px 0 10px; padding-bottom: 6px; border-bottom: 1px solid rgba(255,255,255,0.08);">
    <span style="font-size: 14px;">{p.icon}</span>
    <span style="font-size: 10px; font-weight: 800; letter-spacing: 2px; text-transform: uppercase; color: var(--text-secondary); font-family: var(--font-mono);">{p.title}</span>
  </div>
);

// ── Main Modal ────────────────────────────────────────────────────────────────

export const DiagnosticsModal: Component<DiagnosticsModalProps> = (props) => {
  const [snap, setSnap] = createSignal<DiagnosticsSnapshot | null>(null);
  const [lastUpdate, setLastUpdate] = createSignal<Date | null>(null);

  onMount(() => {
    EventsOn('diagnostics:snapshot', (data: DiagnosticsSnapshot) => {
      setSnap(data);
      setLastUpdate(new Date());
    });
  });

  onCleanup(() => {
    EventsOff('diagnostics:snapshot');
  });

  return (
    <div
      style="position: fixed; inset: 0; background: rgba(0,0,0,0.7); z-index: 9999; display: flex; align-items: center; justify-content: center; backdrop-filter: blur(4px);"
      onClick={props.onClose}
    >
      <div
        style="background: var(--surface-1); border: 1px solid var(--border-primary); border-radius: 10px; width: 780px; max-width: 95vw; max-height: 90vh; overflow: hidden; display: flex; flex-direction: column; box-shadow: 0 24px 80px rgba(0,0,0,0.6);"
        onClick={(e) => e.stopPropagation()}
      >
        {/* ── Header ── */}
        <div style="display: flex; justify-content: space-between; align-items: center; padding: 16px 20px; border-bottom: 1px solid var(--border-primary); background: var(--surface-2);">
          <div style="display: flex; align-items: center; gap: 12px;">
            <span style="font-size: 18px;">📡</span>
            <div>
              <div style="font-family: var(--font-mono); font-size: 13px; font-weight: 800; letter-spacing: 1px; text-transform: uppercase;">Platform Diagnostics</div>
              <div style="font-size: 10px; color: var(--text-muted); margin-top: 2px; font-family: var(--font-mono);">
                Real-time subsystem health · Zero-latency feedback
              </div>
            </div>
          </div>
          <div style="display: flex; align-items: center; gap: 16px;">
            <Show when={snap()}>
              <div style="text-align: center;">
                <div style={`font-size: 28px; font-weight: 900; font-family: var(--font-mono); color: ${gradeColor(snap()!.health_grade)};`}>
                  {snap()!.health_grade}
                </div>
                <div style={`font-size: 9px; letter-spacing: 1px; text-transform: uppercase; color: ${gradeColor(snap()!.health_grade)};`}>
                  {gradeLabel(snap()!.health_grade)}
                </div>
              </div>
            </Show>
            <button
              onClick={props.onClose}
              style="background: transparent; border: 1px solid var(--border-primary); color: var(--text-muted); padding: 6px 10px; border-radius: 5px; cursor: pointer; font-family: var(--font-mono); font-size: 11px;"
            >✕ CLOSE</button>
          </div>
        </div>

        {/* ── Body ── */}
        <div style="flex: 1; overflow-y: auto; padding: 20px; display: grid; grid-template-columns: 1fr 1fr; gap: 24px;">
          <Show when={!snap()}>
            <div style="grid-column: 1 / -1; text-align: center; padding: 60px; color: var(--text-muted); font-family: var(--font-mono); font-size: 12px; letter-spacing: 1px;">
              <div style="font-size: 32px; margin-bottom: 12px; opacity: 0.3;">📡</div>
              WAITING FOR DIAGNOSTICS STREAM...
              <div style="font-size: 10px; margin-top: 8px; opacity: 0.5;">Backend broadcasts every 2 seconds</div>
            </div>
          </Show>

          <Show when={snap()}>
            {/* ── LEFT COLUMN ── */}
            <div>
              {/* Ingest Pipeline */}
              <SectionHeader title="Ingest Pipeline" icon="⚡" />
              <GaugeBar
                label="Current EPS"
                value={snap()!.ingest.current_eps}
                max={snap()!.ingest.target_eps || 50000}
                warn={snap()!.ingest.target_eps * 0.5}
                crit={snap()!.ingest.target_eps * 0.2}
              />
              <GaugeBar
                label="Buffer Fill"
                value={snap()!.ingest.buffer_fill_pct}
                max={100}
                unit="%"
                warn={50}
                crit={80}
              />
              <StatRow label="Target EPS" value={snap()!.ingest.target_eps.toLocaleString()} />
              <StatRow label="EPS % of Target" value={`${snap()!.ingest.percent_of_target.toFixed(1)}%`} accent={snap()!.ingest.percent_of_target >= 80} />
              <StatRow label="Dropped Events" value={snap()!.ingest.dropped_total.toLocaleString()} />
              <StatRow label="Workers" value={snap()!.ingest.worker_count} />

              {/* Event Bus */}
              <SectionHeader title="Event Bus" icon="🔀" />
              <Show when={snap()!.event_bus.rate_limit_active}>
                <div style="background: rgba(248,81,73,0.1); border: 1px solid rgba(248,81,73,0.4); border-radius: 5px; padding: 8px 12px; font-size: 11px; color: #f85149; font-family: var(--font-mono); margin-bottom: 10px; letter-spacing: 0.5px;">
                  ⚠ RATE LIMIT ACTIVE — events are being dropped
                </div>
              </Show>
              <StatRow label="Dropped Events" value={snap()!.event_bus.dropped_events.toLocaleString()} />
              <StatRow label="Estimated Lag" value={`${snap()!.event_bus.lag_estimate_ms.toFixed(0)} ms`} />
              <Show when={snap()!.event_bus.lag_estimate_ms > 500}>
                <div style="font-size: 10px; color: #d29922; font-family: var(--font-mono); margin-top: 4px; padding: 6px 10px; background: rgba(210,153,34,0.08); border-radius: 4px;">
                  ⚠ DATA MAY BE UP TO {(snap()!.event_bus.lag_estimate_ms / 1000).toFixed(1)}s DELAYED
                </div>
              </Show>
            </div>

            {/* ── RIGHT COLUMN ── */}
            <div>
              {/* Query Subsystem */}
              <SectionHeader title="Query Performance" icon="🗄️" />
              <GaugeBar
                label="Last Query"
                value={snap()!.query.last_query_ms}
                max={2000}
                unit=" ms"
                warn={500}
                crit={1000}
              />
              <GaugeBar
                label="P99 Latency"
                value={snap()!.query.p99_query_ms}
                max={2000}
                unit=" ms"
                warn={500}
                crit={1000}
              />
              <StatRow label="Avg Query" value={`${snap()!.query.avg_query_ms.toFixed(1)} ms`} />
              <StatRow label="Slow Queries (>500ms)" value={snap()!.query.slow_query_count.toLocaleString()} />
              <StatRow label="Total Queries" value={snap()!.query.total_queries.toLocaleString()} />

              {/* Go Runtime */}
              <SectionHeader title="Go Runtime" icon="⚙️" />
              <GaugeBar
                label="Heap Memory"
                value={snap()!.runtime.heap_alloc_mb}
                max={snap()!.runtime.heap_sys_mb || 512}
                unit=" MB"
                warn={snap()!.runtime.heap_sys_mb * 0.7}
                crit={snap()!.runtime.heap_sys_mb * 0.9}
              />
              <GaugeBar
                label="Goroutines"
                value={snap()!.runtime.goroutines}
                max={10000}
                warn={5000}
                crit={8000}
              />
              <StatRow label="GC Pause" value={`${(snap()!.runtime.gc_pause_ns / 1e6).toFixed(2)} ms`} />
              <StatRow label="GC Cycles" value={snap()!.runtime.gc_count.toLocaleString()} />
              <StatRow label="CPU Cores" value={snap()!.runtime.num_cpu} />
              <StatRow label="Go Version" value={snap()!.runtime.go_version} />
            </div>
          </Show>
        </div>

        {/* ── Footer ── */}
        <div style="padding: 10px 20px; border-top: 1px solid var(--border-primary); display: flex; justify-content: space-between; align-items: center; background: var(--surface-2);">
          <div style="font-size: 10px; color: var(--text-muted); font-family: var(--font-mono);">
            <Show when={lastUpdate()}>
              LAST UPDATE: {lastUpdate()!.toLocaleTimeString()} · Interval: 2s
            </Show>
            <Show when={!lastUpdate()}>
              STREAM: CONNECTING...
            </Show>
          </div>
          <Show when={snap()}>
            <div style="font-size: 10px; color: var(--text-muted); font-family: var(--font-mono);">
              SNAPSHOT: {snap()!.captured_at.replace('T', ' ').substring(0, 19)} UTC
            </div>
          </Show>
        </div>
      </div>
    </div>
  );
};
