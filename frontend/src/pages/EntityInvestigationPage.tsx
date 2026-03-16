import { Component, createSignal, createEffect, Show, For, onMount, Switch, Match } from 'solid-js';
import { useParams, useNavigate } from '@solidjs/router';
import { GetHostEvents, GetRiskScoreByHost, SearchHostEvents } from '../../wailsjs/go/services/SIEMService';
import { ListIncidents } from '../../wailsjs/go/services/AlertingService';
import { GetProfiles } from '../../wailsjs/go/services/UEBAService';
import { ListHosts } from '../../wailsjs/go/services/HostService';
import { database, ueba } from '../../wailsjs/go/models';

// ── Types ──────────────────────────────────────────────────────────────────

export type EntityType = 'host' | 'user' | 'ip';

interface EntityPageProps {
    // Passed programmatically when opening from within the app.
    // Falls back to URL params when navigated directly.
    entityType?: EntityType;
    entityId?: string;
    onClose?: () => void;
}

// ── Severity helpers ───────────────────────────────────────────────────────

const sevColor = (s: string) =>
    s === 'critical' ? '#e04040'
  : s === 'high'     ? '#f58b00'
  : s === 'medium'   ? '#f5c518'
  : s === 'low'      ? '#5cc05c'
  : 'var(--text-muted)';

const sevBg = (s: string) =>
    s === 'critical' ? 'rgba(224,64,64,0.12)'
  : s === 'high'     ? 'rgba(245,139,0,0.12)'
  : s === 'medium'   ? 'rgba(245,197,24,0.12)'
  : s === 'low'      ? 'rgba(92,192,92,0.12)'
  : 'var(--surface-3)';

const SevBadge: Component<{ severity: string }> = (props) => (
    <span style={{
        display: 'inline-flex', 'align-items': 'center',
        padding: '1px 7px', 'border-radius': '10px',
        'font-size': '10px', 'font-weight': '700',
        'text-transform': 'uppercase', 'letter-spacing': '0.4px',
        'font-family': 'var(--font-ui)',
        color: sevColor(props.severity),
        background: sevBg(props.severity),
        border: `1px solid ${sevColor(props.severity)}40`,
    }}>
        {props.severity}
    </span>
);

const riskColor = (score: number) =>
    score >= 80 ? '#e04040'
  : score >= 60 ? '#f58b00'
  : score >= 40 ? '#f5c518'
  : '#5cc05c';

// ── Reusable section components ───────────────────────────────────────────

const Section: Component<{ title: string; count?: number; children: any; action?: any }> = (props) => (
    <div style={{ 'margin-bottom': '20px' }}>
        <div style={{
            display: 'flex', 'align-items': 'center', 'justify-content': 'space-between',
            'padding-bottom': '8px', 'margin-bottom': '10px',
            'border-bottom': '1px solid var(--border-primary)',
        }}>
            <div style={{ display: 'flex', 'align-items': 'center', gap: '8px' }}>
                <span style={{ 'font-size': '11px', 'font-weight': '700', 'text-transform': 'uppercase', 'letter-spacing': '0.7px', color: 'var(--text-muted)', 'font-family': 'var(--font-ui)' }}>
                    {props.title}
                </span>
                <Show when={props.count !== undefined}>
                    <span style={{ 'font-size': '10px', color: 'var(--text-muted)', background: 'var(--surface-3)', padding: '0 5px', 'border-radius': '8px', 'font-family': 'var(--font-mono)' }}>
                        {props.count}
                    </span>
                </Show>
            </div>
            <Show when={props.action}>{props.action}</Show>
        </div>
        {props.children}
    </div>
);

const KPITile: Component<{ label: string; value: string | number; accent?: string; sub?: string }> = (props) => (
    <div style={{
        background: 'var(--surface-1)', border: '1px solid var(--border-primary)',
        'border-radius': 'var(--radius-md)', padding: '14px 16px',
        display: 'flex', 'flex-direction': 'column', gap: '4px',
    }}>
        <div style={{ 'font-size': '11px', 'font-weight': '600', 'text-transform': 'uppercase', 'letter-spacing': '0.5px', color: 'var(--text-muted)', 'font-family': 'var(--font-ui)' }}>
            {props.label}
        </div>
        <div style={{ 'font-size': '26px', 'font-weight': '700', 'font-family': 'var(--font-mono)', color: props.accent ?? 'var(--text-heading)', 'line-height': '1' }}>
            {props.value}
        </div>
        <Show when={props.sub}>
            <div style={{ 'font-size': '10px', color: 'var(--text-muted)', 'font-family': 'var(--font-ui)' }}>{props.sub}</div>
        </Show>
    </div>
);

const EmptyState: Component<{ message: string }> = (props) => (
    <div style={{ padding: '24px', 'text-align': 'center', color: 'var(--text-muted)', 'font-size': '12px', 'font-family': 'var(--font-ui)' }}>
        {props.message}
    </div>
);

// ── Main Entity Page ───────────────────────────────────────────────────────

export const EntityInvestigationPage: Component<EntityPageProps> = (props) => {
    const params = useParams();
    const navigate = useNavigate();

    const entityType = () => (props.entityType ?? params.type ?? 'host') as EntityType;
    const entityId   = () => props.entityId ?? params.id ?? '';

    const [activeTab, setActiveTab] = createSignal<'overview' | 'events' | 'incidents' | 'ueba'>('overview');
    const [loading, setLoading] = createSignal(true);
    const [error, setError] = createSignal<string | null>(null);

    // Data state
    const [events, setEvents] = createSignal<database.HostEvent[]>([]);
    const [incidents, setIncidents] = createSignal<database.Incident[]>([]);
    const [riskScore, setRiskScore] = createSignal(0);
    const [uebaProfile, setUebaProfile] = createSignal<ueba.EntityProfile | null>(null);
    const [host, setHost] = createSignal<database.Host | null>(null);

    // Load all data for this entity
    const load = async () => {
        setLoading(true);
        setError(null);
        try {
            const id = entityId();
            const type = entityType();

            // Load in parallel
            const [evts, incs, hostList, profiles] = await Promise.allSettled([
                type === 'host'
                    ? GetHostEvents(id, 50)
                    : SearchHostEvents(type === 'user' ? `user:"${id}"` : `source_ip:"${id}"`, 50),
                ListIncidents('', 100),
                type === 'host' ? ListHosts() : Promise.resolve([]),
                GetProfiles(),
            ]);

            // Events
            if (evts.status === 'fulfilled') setEvents(evts.value ?? []);

            // Incidents — filter to those matching this entity
            if (incs.status === 'fulfilled') {
                const all = incs.value ?? [];
                const filtered = all.filter(inc => {
                    const gk = inc.group_key ?? '';
                    return gk === id || gk.includes(id);
                });
                setIncidents(filtered);
            }

            // Host metadata
            if (type === 'host' && hostList.status === 'fulfilled') {
                const found = (hostList.value ?? []).find(h => h.id === id || h.hostname === id || h.label === id);
                setHost(found ?? null);
            }

            // UEBA profile
            if (profiles.status === 'fulfilled') {
                const profile = (profiles.value ?? []).find(p => p.id === id || p.id.includes(id));
                setUebaProfile(profile ?? null);
            }

            // Risk score (host-specific)
            if (type === 'host') {
                try {
                    const rs = await GetRiskScoreByHost(id);
                    setRiskScore(rs ?? 0);
                } catch { setRiskScore(0); }
            } else if (uebaProfile()) {
                setRiskScore(Math.round((uebaProfile()!.risk_score ?? 0) * 100));
            }

        } catch (e: any) {
            setError(e?.message ?? String(e));
        } finally {
            setLoading(false);
        }
    };

    onMount(load);
    createEffect(() => {
        entityId(); // reactive dependency
        load();
    });

    const typeIcon = () => entityType() === 'host' ? '🖥' : entityType() === 'user' ? '👤' : '🌐';
    const typeLabel = () => entityType() === 'host' ? 'Host' : entityType() === 'user' ? 'User' : 'IP Address';

    const displayName = () => {
        if (entityType() === 'host' && host()) {
            return host()!.label || host()!.hostname;
        }
        return entityId();
    };

    const lastEventTime = () => {
        const evs = events();
        if (!evs.length) return 'Never';
        const latest = evs[0]?.timestamp;
        if (!latest) return 'Unknown';
        try { return new Date(latest).toLocaleString(); } catch { return latest; }
    };

    const openIncidents = () => incidents().filter(i => i.status !== 'Closed' && i.status !== 'closed');

    return (
        <div style={{
            display: 'flex', 'flex-direction': 'column',
            height: '100%', background: 'var(--surface-0)', overflow: 'hidden',
        }}>
            {/* ── Header ─────────────────────────────────────────────── */}
            <div style={{
                display: 'flex', 'align-items': 'center', 'justify-content': 'space-between',
                padding: '12px 20px', background: 'var(--surface-1)',
                'border-bottom': '1px solid var(--border-primary)', 'flex-shrink': '0',
            }}>
                <div style={{ display: 'flex', 'align-items': 'center', gap: '12px' }}>
                    {/* Back button */}
                    <Show when={!props.onClose}>
                        <button
                            onClick={() => navigate(-1)}
                            style={{ background: 'none', border: 'none', cursor: 'pointer', color: 'var(--text-muted)', 'font-size': '18px', padding: '0', 'line-height': '1' }}
                        >←</button>
                    </Show>
                    <Show when={props.onClose}>
                        <button
                            onClick={props.onClose}
                            style={{ background: 'none', border: 'none', cursor: 'pointer', color: 'var(--text-muted)', 'font-size': '18px', padding: '0', 'line-height': '1' }}
                        >←</button>
                    </Show>

                    {/* Entity type badge */}
                    <span style={{
                        'font-size': '18px', width: '36px', height: '36px',
                        display: 'flex', 'align-items': 'center', 'justify-content': 'center',
                        background: 'var(--surface-2)', 'border-radius': 'var(--radius-sm)',
                        border: '1px solid var(--border-primary)',
                    }}>{typeIcon()}</span>

                    <div>
                        <div style={{ 'font-size': '16px', 'font-weight': '700', color: 'var(--text-heading)', 'font-family': 'var(--font-ui)' }}>
                            {displayName()}
                        </div>
                        <div style={{ display: 'flex', 'align-items': 'center', gap: '8px', 'margin-top': '2px' }}>
                            <span style={{ 'font-size': '10px', 'font-weight': '700', 'text-transform': 'uppercase', 'letter-spacing': '0.6px', color: '#0099e0', 'font-family': 'var(--font-ui)' }}>
                                {typeLabel()}
                            </span>
                            <Show when={entityType() === 'host' && host()?.hostname && host()!.hostname !== displayName()}>
                                <span style={{ 'font-size': '11px', color: 'var(--text-muted)', 'font-family': 'var(--font-mono)' }}>
                                    {host()!.hostname}:{host()!.port}
                                </span>
                            </Show>
                            <Show when={entityType() === 'host' && host()?.category}>
                                <span style={{ 'font-size': '10px', color: 'var(--text-muted)', background: 'var(--surface-3)', padding: '1px 6px', 'border-radius': '8px', 'font-family': 'var(--font-ui)' }}>
                                    {host()!.category}
                                </span>
                            </Show>
                        </div>
                    </div>
                </div>

                {/* Risk score + actions */}
                <div style={{ display: 'flex', 'align-items': 'center', gap: '12px' }}>
                    <Show when={riskScore() > 0}>
                        <div style={{
                            display: 'flex', 'flex-direction': 'column', 'align-items': 'center',
                            padding: '6px 14px', border: `1px solid ${riskColor(riskScore())}40`,
                            background: `${riskColor(riskScore())}10`, 'border-radius': 'var(--radius-sm)',
                        }}>
                            <span style={{ 'font-size': '18px', 'font-weight': '800', 'font-family': 'var(--font-mono)', color: riskColor(riskScore()), 'line-height': '1' }}>
                                {riskScore()}
                            </span>
                            <span style={{ 'font-size': '9px', 'text-transform': 'uppercase', 'letter-spacing': '0.5px', color: 'var(--text-muted)', 'margin-top': '2px' }}>
                                Risk
                            </span>
                        </div>
                    </Show>

                    <ActionButtons entityType={entityType()} entityId={entityId()} host={host()} navigate={navigate} />

                    <button onClick={load} title="Refresh" style={{ background: 'var(--surface-2)', border: '1px solid var(--border-primary)', 'border-radius': 'var(--radius-sm)', color: 'var(--text-muted)', cursor: 'pointer', padding: '6px 10px', 'font-size': '14px' }}>↺</button>
                </div>
            </div>

            {/* ── Loading / Error ─────────────────────────────────── */}
            <Show when={loading()}>
                <div style={{ padding: '48px', 'text-align': 'center', color: 'var(--text-muted)', 'font-family': 'var(--font-mono)', 'font-size': '11px', 'text-transform': 'uppercase', 'letter-spacing': '1px' }}>
                    Loading entity data…
                </div>
            </Show>
            <Show when={error()}>
                <div style={{ margin: '16px 20px', padding: '10px 14px', background: 'rgba(224,64,64,0.06)', border: '1px solid rgba(224,64,64,0.3)', 'border-radius': 'var(--radius-sm)', color: '#e04040', 'font-size': '12px', 'font-family': 'var(--font-ui)' }}>
                    ⚠ {error()}
                </div>
            </Show>

            <Show when={!loading()}>
                {/* ── KPI strip ───────────────────────────────────── */}
                <div style={{ display: 'grid', 'grid-template-columns': 'repeat(4, 1fr)', gap: '1px', background: 'var(--border-primary)', 'flex-shrink': '0' }}>
                    <KPITile label="Events (50 latest)" value={events().length} />
                    <KPITile label="Open Incidents" value={openIncidents().length} accent={openIncidents().length > 0 ? '#f58b00' : undefined} />
                    <KPITile label="UEBA Observations" value={uebaProfile()?.observations ?? '—'} sub={uebaProfile() ? `Last seen: ${new Date(uebaProfile()!.last_seen).toLocaleDateString()}` : undefined} />
                    <KPITile label="Last Event" value={lastEventTime()} />
                </div>

                {/* ── Tab bar ─────────────────────────────────────── */}
                <div class="ob-tabs" style={{ 'flex-shrink': '0' }}>
                    {(['overview', 'events', 'incidents', 'ueba'] as const).map(tab => (
                        <button
                            class={`ob-tab${activeTab() === tab ? ' active' : ''}`}
                            onClick={() => setActiveTab(tab)}
                        >
                            {tab === 'overview' ? 'Overview'
                             : tab === 'events' ? `Events (${events().length})`
                             : tab === 'incidents' ? `Incidents (${incidents().length})`
                             : 'UEBA Profile'}
                        </button>
                    ))}
                </div>

                {/* ── Tab content ─────────────────────────────────── */}
                <div style={{ flex: '1', 'overflow-y': 'auto', padding: '20px' }}>
                    <Show when={activeTab() === 'overview'}>
                        <OverviewTab
                            entityType={entityType()} entityId={entityId()}
                            host={host()} events={events()} incidents={incidents()}
                            uebaProfile={uebaProfile()} riskScore={riskScore()}
                        />
                    </Show>
                    <Show when={activeTab() === 'events'}>
                        <EventsTab events={events()} entityType={entityType()} />
                    </Show>
                    <Show when={activeTab() === 'incidents'}>
                        <IncidentsTab incidents={incidents()} />
                    </Show>
                    <Show when={activeTab() === 'ueba'}>
                        <UEBATab profile={uebaProfile()} entityId={entityId()} />
                    </Show>
                </div>
            </Show>
        </div>
    );
};

// ── Action buttons ─────────────────────────────────────────────────────────

const ActionButtons: Component<{
    entityType: EntityType;
    entityId: string;
    host: database.Host | null;
    navigate: (path: string) => void;
}> = (props) => {
    const btnStyle = {
        background: 'var(--surface-2)', border: '1px solid var(--border-primary)',
        'border-radius': 'var(--radius-sm)', color: 'var(--text-secondary)',
        'font-family': 'var(--font-ui)', 'font-size': '11px', 'font-weight': '600',
        padding: '6px 12px', cursor: 'pointer', transition: 'all 0.15s',
    };

    return (
        <div style={{ display: 'flex', gap: '8px' }}>
            {/* Hunt — opens threat hunter pre-filtered to this entity */}
            <button
                style={btnStyle}
                onClick={() => props.navigate(`/threat-hunter?entity=${encodeURIComponent(props.entityId)}`)}
                onMouseEnter={e => { (e.currentTarget as HTMLElement).style.borderColor = '#0099e0'; (e.currentTarget as HTMLElement).style.color = '#0099e0'; }}
                onMouseLeave={e => { (e.currentTarget as HTMLElement).style.borderColor = 'var(--border-primary)'; (e.currentTarget as HTMLElement).style.color = 'var(--text-secondary)'; }}
                title="Open Threat Hunter for this entity"
            >
                🔍 Hunt
            </button>

            {/* SIEM search */}
            <button
                style={btnStyle}
                onClick={() => {
                    const q = props.entityType === 'host' ? `host_id:"${props.entityId}"`
                            : props.entityType === 'user' ? `user:"${props.entityId}"`
                            : `source_ip:"${props.entityId}"`;
                    props.navigate(`/siem?q=${encodeURIComponent(q)}`);
                }}
                onMouseEnter={e => { (e.currentTarget as HTMLElement).style.borderColor = '#0099e0'; (e.currentTarget as HTMLElement).style.color = '#0099e0'; }}
                onMouseLeave={e => { (e.currentTarget as HTMLElement).style.borderColor = 'var(--border-primary)'; (e.currentTarget as HTMLElement).style.color = 'var(--text-secondary)'; }}
                title="Search SIEM for this entity"
            >
                📊 SIEM
            </button>

            {/* Connect — host only */}
            <Show when={props.entityType === 'host' && props.host}>
                <button
                    style={{ ...btnStyle, background: 'rgba(92,192,92,0.1)', 'border-color': 'rgba(92,192,92,0.3)', color: '#5cc05c' }}
                    onClick={() => props.navigate(`/terminal`)}
                    title="Open SSH session to this host"
                >
                    ⚡ Connect
                </button>
            </Show>
        </div>
    );
};

// ── Overview tab ───────────────────────────────────────────────────────────

const OverviewTab: Component<{
    entityType: EntityType;
    entityId: string;
    host: database.Host | null;
    events: database.HostEvent[];
    incidents: database.Incident[];
    uebaProfile: ueba.EntityProfile | null;
    riskScore: number;
}> = (props) => {
    // Event type breakdown
    const eventTypes = () => {
        const counts: Record<string, number> = {};
        for (const e of props.events) {
            counts[e.event_type] = (counts[e.event_type] ?? 0) + 1;
        }
        return Object.entries(counts)
            .sort((a, b) => b[1] - a[1])
            .slice(0, 8);
    };

    // Top source IPs
    const topIPs = () => {
        const counts: Record<string, number> = {};
        for (const e of props.events) {
            if (e.source_ip && e.source_ip !== '127.0.0.1') {
                counts[e.source_ip] = (counts[e.source_ip] ?? 0) + 1;
            }
        }
        return Object.entries(counts).sort((a, b) => b[1] - a[1]).slice(0, 5);
    };

    const openInc = () => props.incidents.filter(i => i.status !== 'Closed' && i.status !== 'closed');

    return (
        <div style={{ display: 'grid', 'grid-template-columns': '1fr 1fr', gap: '24px' }}>
            {/* Left column */}
            <div>
                {/* Host metadata */}
                <Show when={props.entityType === 'host' && props.host}>
                    <Section title="Host Details">
                        <div style={{ display: 'flex', 'flex-direction': 'column', gap: '0' }}>
                            <InfoRow label="Hostname" value={props.host!.hostname} mono />
                            <InfoRow label="Port" value={String(props.host!.port)} mono />
                            <InfoRow label="Username" value={props.host!.username} mono />
                            <InfoRow label="Auth Method" value={props.host!.auth_method} />
                            <InfoRow label="Category" value={props.host!.category || '—'} />
                            <InfoRow label="Last Connected" value={props.host!.last_connected_at ? new Date(props.host!.last_connected_at).toLocaleString() : 'Never'} />
                            <InfoRow label="Connection Count" value={String(props.host!.connection_count)} mono />
                            <Show when={props.host!.tags?.length > 0}>
                                <div style={{ display: 'flex', 'justify-content': 'space-between', 'padding': '5px 0', 'border-bottom': '1px solid var(--border-subtle)' }}>
                                    <span style={{ 'font-size': '11px', color: 'var(--text-muted)', 'font-family': 'var(--font-ui)' }}>Tags</span>
                                    <div style={{ display: 'flex', gap: '4px', 'flex-wrap': 'wrap', 'justify-content': 'flex-end' }}>
                                        <For each={props.host!.tags}>
                                            {tag => <span style={{ 'font-size': '10px', color: '#0099e0', background: 'rgba(0,153,224,0.1)', padding: '1px 6px', 'border-radius': '8px' }}>{tag}</span>}
                                        </For>
                                    </div>
                                </div>
                            </Show>
                        </div>
                    </Section>
                </Show>

                {/* Event type breakdown */}
                <Section title="Event Types" count={eventTypes().length}>
                    <Show when={eventTypes().length === 0}>
                        <EmptyState message="No events indexed for this entity yet." />
                    </Show>
                    <For each={eventTypes()}>
                        {([type, count]) => {
                            const max = eventTypes()[0]?.[1] ?? 1;
                            const pct = Math.round((count / max) * 100);
                            return (
                                <div style={{ display: 'flex', 'align-items': 'center', gap: '10px', padding: '5px 0' }}>
                                    <span style={{ 'font-family': 'var(--font-mono)', 'font-size': '11px', color: 'var(--text-secondary)', 'min-width': '160px', 'white-space': 'nowrap', overflow: 'hidden', 'text-overflow': 'ellipsis' }}>{type}</span>
                                    <div style={{ flex: '1', height: '4px', background: 'var(--surface-3)', 'border-radius': '2px', overflow: 'hidden' }}>
                                        <div style={{ width: `${pct}%`, height: '100%', background: '#0099e0', 'border-radius': '2px', transition: 'width 0.4s ease' }} />
                                    </div>
                                    <span style={{ 'font-family': 'var(--font-mono)', 'font-size': '11px', color: 'var(--text-muted)', 'min-width': '28px', 'text-align': 'right' }}>{count}</span>
                                </div>
                            );
                        }}
                    </For>
                </Section>
            </div>

            {/* Right column */}
            <div>
                {/* Open incidents */}
                <Section title="Open Incidents" count={openInc().length}>
                    <Show when={openInc().length === 0}>
                        <div style={{ padding: '12px', background: 'rgba(92,192,92,0.06)', border: '1px solid rgba(92,192,92,0.2)', 'border-radius': 'var(--radius-sm)', 'font-size': '12px', color: '#5cc05c', 'text-align': 'center' }}>
                            ✓ No open incidents
                        </div>
                    </Show>
                    <For each={openInc().slice(0, 5)}>
                        {inc => (
                            <div style={{ padding: '8px 0', 'border-bottom': '1px solid var(--border-subtle)', display: 'flex', 'align-items': 'center', gap: '10px' }}>
                                <SevBadge severity={inc.severity} />
                                <div style={{ flex: '1', 'min-width': '0' }}>
                                    <div style={{ 'font-size': '12px', color: 'var(--text-primary)', 'font-family': 'var(--font-ui)', 'white-space': 'nowrap', overflow: 'hidden', 'text-overflow': 'ellipsis' }}>{inc.title}</div>
                                    <div style={{ 'font-size': '10px', color: 'var(--text-muted)', 'font-family': 'var(--font-mono)', 'margin-top': '2px' }}>
                                        {inc.event_count} events · {inc.status}
                                    </div>
                                </div>
                            </div>
                        )}
                    </For>
                </Section>

                {/* Top source IPs */}
                <Show when={props.entityType !== 'ip' && topIPs().length > 0}>
                    <Section title="Top Source IPs" count={topIPs().length}>
                        <For each={topIPs()}>
                            {([ip, count]) => (
                                <div style={{ display: 'flex', 'justify-content': 'space-between', padding: '5px 0', 'border-bottom': '1px solid var(--border-subtle)' }}>
                                    <a
                                        onClick={() => window.location.hash = `#/entity/ip/${ip}`}
                                        style={{ 'font-family': 'var(--font-mono)', 'font-size': '12px', color: '#0099e0', cursor: 'pointer', 'text-decoration': 'none' }}
                                        onMouseEnter={e => (e.currentTarget as HTMLElement).style.textDecoration = 'underline'}
                                        onMouseLeave={e => (e.currentTarget as HTMLElement).style.textDecoration = 'none'}
                                    >{ip}</a>
                                    <span style={{ 'font-family': 'var(--font-mono)', 'font-size': '11px', color: 'var(--text-muted)' }}>{count} events</span>
                                </div>
                            )}
                        </For>
                    </Section>
                </Show>

                {/* UEBA summary */}
                <Show when={props.uebaProfile}>
                    <Section title="UEBA Summary">
                        <InfoRow label="Risk Score" value={String(Math.round((props.uebaProfile!.risk_score ?? 0) * 100))} mono accent={riskColor(Math.round((props.uebaProfile!.risk_score ?? 0) * 100))} />
                        <InfoRow label="Observations" value={String(props.uebaProfile!.observations)} mono />
                        <InfoRow label="Entity Type" value={props.uebaProfile!.type} />
                        <InfoRow label="Peer Group" value={props.uebaProfile!.peer_group_id || '—'} mono />
                        <InfoRow label="Last Seen" value={new Date(props.uebaProfile!.last_seen).toLocaleString()} />
                    </Section>
                </Show>
            </div>
        </div>
    );
};

// ── Events tab ─────────────────────────────────────────────────────────────

const EventsTab: Component<{ events: database.HostEvent[]; entityType: EntityType }> = (props) => (
    <div>
        <Show when={props.events.length === 0}>
            <EmptyState message="No events found for this entity." />
        </Show>
        <Show when={props.events.length > 0}>
            <div class="ob-table-wrap">
                <table class="ob-table">
                    <thead>
                        <tr>
                            <th>Timestamp</th>
                            <th>Event Type</th>
                            <th>Source IP</th>
                            <th>User</th>
                            <th>Location</th>
                            <th>Raw Log</th>
                        </tr>
                    </thead>
                    <tbody>
                        <For each={props.events}>
                            {evt => (
                                <tr>
                                    <td class="mono" style={{ 'white-space': 'nowrap' }}>
                                        {evt.timestamp ? new Date(evt.timestamp).toLocaleString() : '—'}
                                    </td>
                                    <td>
                                        <span style={{ 'font-family': 'var(--font-mono)', 'font-size': '11px', color: '#0099e0', background: 'rgba(0,153,224,0.08)', padding: '1px 6px', 'border-radius': '4px' }}>
                                            {evt.event_type}
                                        </span>
                                    </td>
                                    <td class="mono">{evt.source_ip || '—'}</td>
                                    <td class="mono">{evt.user || '—'}</td>
                                    <td class="mono">{evt.location || '—'}</td>
                                    <td style={{ 'max-width': '300px', overflow: 'hidden', 'text-overflow': 'ellipsis', 'white-space': 'nowrap', 'font-family': 'var(--font-mono)', 'font-size': '11px', color: 'var(--text-muted)' }} title={evt.raw_log}>
                                        {evt.raw_log}
                                    </td>
                                </tr>
                            )}
                        </For>
                    </tbody>
                </table>
            </div>
        </Show>
    </div>
);

// ── Incidents tab ──────────────────────────────────────────────────────────

const IncidentsTab: Component<{ incidents: database.Incident[] }> = (props) => (
    <div>
        <Show when={props.incidents.length === 0}>
            <EmptyState message="No incidents linked to this entity." />
        </Show>
        <Show when={props.incidents.length > 0}>
            <div class="ob-table-wrap">
                <table class="ob-table">
                    <thead>
                        <tr>
                            <th>Severity</th>
                            <th>Title</th>
                            <th>Status</th>
                            <th>Events</th>
                            <th>First Seen</th>
                            <th>Last Seen</th>
                            <th>Tactics</th>
                        </tr>
                    </thead>
                    <tbody>
                        <For each={props.incidents}>
                            {inc => (
                                <tr>
                                    <td><SevBadge severity={inc.severity} /></td>
                                    <td style={{ 'max-width': '280px' }}>
                                        <div style={{ 'font-weight': '500', color: 'var(--text-primary)', 'font-size': '12px', overflow: 'hidden', 'text-overflow': 'ellipsis', 'white-space': 'nowrap' }} title={inc.title}>
                                            {inc.title}
                                        </div>
                                        <div style={{ 'font-size': '10px', color: 'var(--text-muted)', 'font-family': 'var(--font-mono)', 'margin-top': '1px' }}>{inc.rule_id}</div>
                                    </td>
                                    <td>
                                        <span style={{
                                            'font-size': '10px', padding: '2px 6px', 'border-radius': '8px', 'font-weight': '600',
                                            color: inc.status === 'New' ? '#f58b00' : inc.status === 'Closed' ? '#5cc05c' : '#0099e0',
                                            background: inc.status === 'New' ? 'rgba(245,139,0,0.1)' : inc.status === 'Closed' ? 'rgba(92,192,92,0.1)' : 'rgba(0,153,224,0.1)',
                                        }}>{inc.status}</span>
                                    </td>
                                    <td class="mono">{inc.event_count}</td>
                                    <td class="mono" style={{ 'white-space': 'nowrap', 'font-size': '11px' }}>
                                        {inc.first_seen_at ? new Date(inc.first_seen_at).toLocaleDateString() : '—'}
                                    </td>
                                    <td class="mono" style={{ 'white-space': 'nowrap', 'font-size': '11px' }}>
                                        {inc.last_seen_at ? new Date(inc.last_seen_at).toLocaleDateString() : '—'}
                                    </td>
                                    <td>
                                        <div style={{ display: 'flex', gap: '3px', 'flex-wrap': 'wrap' }}>
                                            <For each={inc.mitre_tactics?.slice(0, 3) ?? []}>
                                                {t => <span style={{ 'font-size': '9px', color: 'var(--text-muted)', background: 'var(--surface-3)', padding: '1px 4px', 'border-radius': '4px', 'font-family': 'var(--font-mono)' }}>{t}</span>}
                                            </For>
                                        </div>
                                    </td>
                                </tr>
                            )}
                        </For>
                    </tbody>
                </table>
            </div>
        </Show>
    </div>
);

// ── UEBA tab ───────────────────────────────────────────────────────────────

const UEBATab: Component<{ profile: ueba.EntityProfile | null; entityId: string }> = (props) => (
    <Show
        when={props.profile}
        fallback={
            <div style={{ padding: '32px', 'text-align': 'center', color: 'var(--text-muted)', 'font-size': '13px', 'font-family': 'var(--font-ui)' }}>
                <div style={{ 'font-size': '24px', 'margin-bottom': '12px', opacity: '0.3' }}>◎</div>
                No UEBA profile for <span style={{ 'font-family': 'var(--font-mono)', color: 'var(--text-secondary)' }}>{props.entityId}</span>.<br />
                <span style={{ 'font-size': '11px', 'margin-top': '6px', display: 'block' }}>
                    UEBA profiles are built from observed behaviour over time. Check back after more events are ingested.
                </span>
            </div>
        }
    >
        <div style={{ display: 'grid', 'grid-template-columns': '1fr 1fr', gap: '24px' }}>
            <div>
                <Section title="Behavioural Baseline">
                    <InfoRow label="Risk Score" value={`${Math.round((props.profile!.risk_score ?? 0) * 100)} / 100`} mono accent={riskColor(Math.round((props.profile!.risk_score ?? 0) * 100))} />
                    <InfoRow label="Observations" value={String(props.profile!.observations)} mono />
                    <InfoRow label="Entity Type" value={props.profile!.type} />
                    <InfoRow label="Peer Group" value={props.profile!.peer_group_id || 'None'} mono />
                    <InfoRow label="Last Seen" value={new Date(props.profile!.last_seen).toLocaleString()} />
                </Section>

                <Show when={Object.keys(props.profile!.features ?? {}).length > 0}>
                    <Section title="Feature Weights">
                        <For each={Object.entries(props.profile!.features ?? {}).sort((a, b) => b[1] - a[1])}>
                            {([feat, val]) => (
                                <div style={{ display: 'flex', 'justify-content': 'space-between', padding: '4px 0', 'border-bottom': '1px solid var(--border-subtle)' }}>
                                    <span style={{ 'font-family': 'var(--font-mono)', 'font-size': '11px', color: 'var(--text-secondary)' }}>{feat}</span>
                                    <span style={{ 'font-family': 'var(--font-mono)', 'font-size': '11px', color: 'var(--text-muted)' }}>{typeof val === 'number' ? val.toFixed(3) : val}</span>
                                </div>
                            )}
                        </For>
                    </Section>
                </Show>
            </div>

            <div>
                <Show when={(props.profile!.risk_history ?? []).length > 0}>
                    <Section title="Risk History">
                        <div style={{ display: 'flex', 'flex-direction': 'column', gap: '4px' }}>
                            <For each={[...(props.profile!.risk_history ?? [])].reverse().slice(0, 15)}>
                                {pt => {
                                    const score = Math.round((pt.score ?? 0) * 100);
                                    return (
                                        <div style={{ display: 'flex', 'align-items': 'center', gap: '10px' }}>
                                            <span style={{ 'font-family': 'var(--font-mono)', 'font-size': '10px', color: 'var(--text-muted)', 'min-width': '130px', 'white-space': 'nowrap' }}>
                                                {new Date(pt.timestamp).toLocaleString()}
                                            </span>
                                            <div style={{ flex: '1', height: '4px', background: 'var(--surface-3)', 'border-radius': '2px', overflow: 'hidden' }}>
                                                <div style={{ width: `${score}%`, height: '100%', background: riskColor(score), 'border-radius': '2px' }} />
                                            </div>
                                            <span style={{ 'font-family': 'var(--font-mono)', 'font-size': '10px', color: riskColor(score), 'min-width': '28px', 'text-align': 'right', 'font-weight': '700' }}>
                                                {score}
                                            </span>
                                        </div>
                                    );
                                }}
                            </For>
                        </div>
                    </Section>
                </Show>
            </div>
        </div>
    </Show>
);

// ── InfoRow helper ─────────────────────────────────────────────────────────

const InfoRow: Component<{ label: string; value: string; mono?: boolean; accent?: string }> = (props) => (
    <div style={{
        display: 'flex', 'justify-content': 'space-between', 'align-items': 'center',
        padding: '5px 0', 'border-bottom': '1px solid var(--border-subtle)',
    }}>
        <span style={{ 'font-size': '11px', color: 'var(--text-muted)', 'font-family': 'var(--font-ui)' }}>{props.label}</span>
        <span style={{
            'font-size': '12px', 'font-weight': '600',
            'font-family': props.mono ? 'var(--font-mono)' : 'var(--font-ui)',
            color: props.accent ?? 'var(--text-primary)',
        }}>{props.value}</span>
    </div>
);

export default EntityInvestigationPage;
