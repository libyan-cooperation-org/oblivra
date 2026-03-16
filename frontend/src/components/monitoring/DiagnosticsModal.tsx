import { Component, createSignal, onMount, onCleanup, Show, For } from 'solid-js';
import { GetSnapshot } from '../../../wailsjs/go/services/DiagnosticsService';
import { monitoring } from '../../../wailsjs/go/models';
import { EventsOn, EventsOff } from '../../../wailsjs/runtime/runtime';

interface DiagnosticsModalProps {
    onClose: () => void;
}

type Snap = monitoring.DiagnosticsSnapshot;

// ── helpers ──────────────────────────────────────────────────────────────────

const gradeColor = (g: string) => {
    switch (g) {
        case 'A': return '#5cc05c';
        case 'B': return '#0099e0';
        case 'C': return '#f5c518';
        case 'D': return '#f58b00';
        default:  return '#e04040';
    }
};

const fmt1 = (n: number) => n?.toFixed(1) ?? '—';
const fmtInt = (n: number) => (n ?? 0).toLocaleString();
const fmtMs = (ns: number) => ns > 0 ? `${(ns / 1_000_000).toFixed(2)} ms` : '—';

function Bar(props: { pct: number; color?: string }) {
    const c = props.color ?? '#0099e0';
    const w = Math.min(100, Math.max(0, props.pct ?? 0));
    return (
        <div style={{
            height: '4px', background: 'var(--surface-3)',
            'border-radius': '2px', overflow: 'hidden', flex: '1',
        }}>
            <div style={{ width: `${w}%`, height: '100%', background: c, 'border-radius': '2px', transition: 'width 0.4s ease' }} />
        </div>
    );
}

function Row(props: { label: string; value: string; mono?: boolean; accent?: string }) {
    return (
        <div style={{ display: 'flex', 'justify-content': 'space-between', 'align-items': 'center', padding: '5px 0', 'border-bottom': '1px solid var(--border-subtle)' }}>
            <span style={{ 'font-size': '11px', color: 'var(--text-muted)', 'font-family': 'var(--font-ui)' }}>{props.label}</span>
            <span style={{
                'font-size': '12px',
                'font-family': props.mono ? 'var(--font-mono)' : 'var(--font-ui)',
                'font-weight': '600',
                color: props.accent ?? 'var(--text-primary)',
            }}>{props.value}</span>
        </div>
    );
}

function Section(props: { title: string; children: any }) {
    return (
        <div style={{ 'margin-bottom': '16px' }}>
            <div style={{
                'font-size': '9px', 'font-weight': '700', 'text-transform': 'uppercase',
                'letter-spacing': '1px', color: 'var(--text-muted)',
                'padding-bottom': '6px', 'border-bottom': '1px solid var(--border-primary)',
                'margin-bottom': '6px', 'font-family': 'var(--font-ui)',
            }}>{props.title}</div>
            {props.children}
        </div>
    );
}

// ── component ─────────────────────────────────────────────────────────────────

export const DiagnosticsModal: Component<DiagnosticsModalProps> = (props) => {
    const [snap, setSnap] = createSignal<Snap | null>(null);
    const [age, setAge] = createSignal(0); // seconds since last update
    let ageTimer: ReturnType<typeof setInterval>;

    const load = async () => {
        try {
            const s = await GetSnapshot();
            setSnap(s);
            setAge(0);
        } catch (e) {
            console.error('[DiagnosticsModal] GetSnapshot failed:', e);
        }
    };

    onMount(() => {
        load();

        // Poll every 2s (matches the backend broadcast interval)
        const pollTimer = setInterval(load, 2000);

        // Also listen for push events from backend broadcast
        EventsOn('diagnostics:snapshot', (data: Snap) => {
            setSnap(data);
            setAge(0);
        });

        // Tick age counter every second
        ageTimer = setInterval(() => setAge(a => a + 1), 1000);

        onCleanup(() => {
            clearInterval(pollTimer);
            clearInterval(ageTimer);
            EventsOff('diagnostics:snapshot');
        });
    });

    const handleBackdrop = (e: MouseEvent) => {
        if ((e.target as HTMLElement).classList.contains('diag-overlay')) {
            props.onClose();
        }
    };

    return (
        <div
            class="diag-overlay"
            onClick={handleBackdrop}
            style={{
                position: 'fixed', inset: '0',
                background: 'rgba(0,0,0,0.6)',
                display: 'flex', 'align-items': 'flex-start', 'justify-content': 'flex-end',
                padding: '48px 16px 16px',
                'z-index': '8000',
                animation: 'fade-in 0.15s ease-out',
            }}
        >
            <div style={{
                width: '360px',
                background: 'var(--surface-1)',
                border: '1px solid var(--border-secondary)',
                'border-radius': 'var(--radius-lg)',
                'box-shadow': 'var(--shadow-xl)',
                display: 'flex',
                'flex-direction': 'column',
                overflow: 'hidden',
                animation: 'slide-down 0.15s ease-out',
                'max-height': 'calc(100vh - 64px)',
            }}>
                {/* Header */}
                <div style={{
                    display: 'flex', 'align-items': 'center', 'justify-content': 'space-between',
                    padding: '12px 16px',
                    background: 'var(--surface-2)',
                    'border-bottom': '1px solid var(--border-primary)',
                    'flex-shrink': '0',
                }}>
                    <div style={{ display: 'flex', 'align-items': 'center', gap: '10px' }}>
                        <span style={{ 'font-size': '12px', 'font-weight': '700', color: 'var(--text-heading)', 'font-family': 'var(--font-ui)', 'text-transform': 'uppercase', 'letter-spacing': '0.5px' }}>
                            Platform Diagnostics
                        </span>
                        <Show when={snap()}>
                            <span style={{
                                'font-size': '20px', 'font-weight': '800',
                                color: gradeColor(snap()!.health_grade),
                                'font-family': 'var(--font-mono)',
                                'line-height': '1',
                            }}>
                                {snap()!.health_grade}
                            </span>
                        </Show>
                    </div>
                    <div style={{ display: 'flex', 'align-items': 'center', gap: '8px' }}>
                        <span style={{ 'font-size': '10px', color: 'var(--text-muted)', 'font-family': 'var(--font-mono)' }}>
                            {age() === 0 ? 'live' : `${age()}s ago`}
                        </span>
                        <button
                            onClick={props.onClose}
                            style={{ background: 'none', border: 'none', color: 'var(--text-muted)', cursor: 'pointer', 'font-size': '18px', padding: '0', 'line-height': '1' }}
                            onMouseEnter={e => e.currentTarget.style.color = 'var(--text-primary)'}
                            onMouseLeave={e => e.currentTarget.style.color = 'var(--text-muted)'}
                        >×</button>
                    </div>
                </div>

                {/* Body */}
                <div style={{ flex: '1', 'overflow-y': 'auto', padding: '14px 16px' }}>
                    <Show when={!snap()} fallback={
                        <div style={{ color: 'var(--text-muted)', 'font-size': '12px', 'text-align': 'center', padding: '24px 0' }}>
                            Loading…
                        </div>
                    }>
                        {/* Ingest */}
                        <Section title="Ingest Pipeline">
                            <Row label="Events / sec" value={fmtInt(snap()!.ingest.current_eps)} mono accent={snap()!.ingest.current_eps > 0 ? '#5cc05c' : 'var(--text-primary)'} />
                            <div style={{ display: 'flex', 'align-items': 'center', gap: '8px', padding: '5px 0' }}>
                                <span style={{ 'font-size': '11px', color: 'var(--text-muted)', 'white-space': 'nowrap', 'font-family': 'var(--font-ui)' }}>
                                    Buffer {fmt1(snap()!.ingest.buffer_fill_pct)}%
                                </span>
                                <Bar
                                    pct={snap()!.ingest.buffer_fill_pct}
                                    color={snap()!.ingest.buffer_fill_pct > 80 ? '#e04040' : snap()!.ingest.buffer_fill_pct > 50 ? '#f58b00' : '#0099e0'}
                                />
                                <span style={{ 'font-size': '11px', color: 'var(--text-muted)', 'font-family': 'var(--font-mono)', 'white-space': 'nowrap' }}>
                                    {fmtInt(snap()!.ingest.dropped_total)} dropped
                                </span>
                            </div>
                            <Row label="Workers" value={String(snap()!.ingest.worker_count)} mono />
                            <Row label="Target EPS" value={fmtInt(snap()!.ingest.target_eps)} mono />
                        </Section>

                        {/* Runtime */}
                        <Section title="Go Runtime">
                            <Row label="Goroutines" value={fmtInt(snap()!.runtime.goroutines)} mono
                                accent={snap()!.runtime.goroutines > 800 ? '#f58b00' : snap()!.runtime.goroutines > 1500 ? '#e04040' : 'var(--text-primary)'}
                            />
                            <Row label="Heap Alloc" value={`${fmt1(snap()!.runtime.heap_alloc_mb)} MB`} mono />
                            <Row label="Heap Sys" value={`${fmt1(snap()!.runtime.heap_sys_mb)} MB`} mono />
                            <Row label="GC Pause" value={fmtMs(snap()!.runtime.gc_pause_ns)} mono />
                            <Row label="GC Cycles" value={fmtInt(snap()!.runtime.gc_count)} mono />
                            <Row label="CPUs" value={String(snap()!.runtime.num_cpu)} mono />
                            <Row label="Go" value={snap()!.runtime.go_version} mono />
                        </Section>

                        {/* Event Bus */}
                        <Section title="Event Bus">
                            <Row label="Dropped Events" value={fmtInt(snap()!.event_bus.dropped_events)} mono
                                accent={snap()!.event_bus.dropped_events > 0 ? '#f58b00' : 'var(--text-primary)'}
                            />
                            <Row label="Rate Limited" value={snap()!.event_bus.rate_limit_active ? 'YES' : 'No'} mono
                                accent={snap()!.event_bus.rate_limit_active ? '#e04040' : '#5cc05c'}
                            />
                            <Row label="Lag Estimate" value={snap()!.event_bus.lag_estimate_ms > 0 ? `${fmtInt(snap()!.event_bus.lag_estimate_ms)} ms` : '< 1 ms'} mono />
                        </Section>

                        {/* Query */}
                        <Section title="Database Queries">
                            <Row label="Last Query" value={`${fmt1(snap()!.query.last_query_ms)} ms`} mono />
                            <Row label="Avg Latency" value={`${fmt1(snap()!.query.avg_query_ms)} ms`} mono />
                            <Row label="P99 Latency" value={`${fmt1(snap()!.query.p99_query_ms)} ms`} mono
                                accent={snap()!.query.p99_query_ms > 200 ? '#f58b00' : 'var(--text-primary)'}
                            />
                            <Row label="Slow Queries" value={fmtInt(snap()!.query.slow_query_count)} mono
                                accent={snap()!.query.slow_query_count > 0 ? '#f58b00' : 'var(--text-primary)'}
                            />
                            <Row label="Total Queries" value={fmtInt(snap()!.query.total_queries)} mono />
                        </Section>

                        {/* Captured at */}
                        <div style={{ 'font-size': '10px', color: 'var(--text-muted)', 'font-family': 'var(--font-mono)', 'text-align': 'right', 'padding-top': '4px' }}>
                            {snap()!.captured_at ? new Date(snap()!.captured_at).toLocaleTimeString() : ''}
                        </div>
                    </Show>
                </div>
            </div>

            <style>{`
                @keyframes slide-down {
                    from { opacity: 0; transform: translateY(-8px); }
                    to   { opacity: 1; transform: translateY(0); }
                }
            `}</style>
        </div>
    );
};
