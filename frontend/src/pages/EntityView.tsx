import { Component, createSignal, Show, For, onMount } from 'solid-js';
import { useParams, useNavigate } from '@solidjs/router';
import { GetHostEvents, GetRiskScoreByHost, SearchHostEvents } from '../../wailsjs/go/services/SIEMService';
import { ListIncidents } from '../../wailsjs/go/services/AlertingService';
import { GetProfiles } from '../../wailsjs/go/services/UEBAService';
import { ListHosts } from '../../wailsjs/go/services/HostService';
import { GetEntityEnrichment } from '../../wailsjs/go/services/AnalyticsService';
import { database, ueba, services } from '../../wailsjs/go/models';
import { Sparkline } from '../components/ui/Sparkline';

// ── Types ──────────────────────────────────────────────────────────────────
export type EntityType = 'host' | 'user' | 'ip' | 'domain';

interface EntityViewProps {
    entityType?: EntityType;
    entityId?: string;
    onClose?: () => void;
    params?: any;
    location?: any;
    data?: any;
}

// ── Styles (Premium UI) ───────────────────────────────────────────────────
const glassStyle = {
    background: 'rgba(255, 255, 255, 0.03)',
    'backdrop-filter': 'blur(12px)',
    '-webkit-backdrop-filter': 'blur(12px)',
    border: '1px solid rgba(255, 255, 255, 0.08)',
    'border-radius': '16px',
    'box-shadow': '0 8px 32px 0 rgba(0, 0, 0, 0.37)',
};

const cardHeaderStyle = {
    'font-size': '12px',
    'font-weight': '700',
    'text-transform': 'uppercase',
    'letter-spacing': '1px',
    color: 'rgba(255, 255, 255, 0.5)',
    'margin-bottom': '12px',
    display: 'flex',
    'align-items': 'center',
    gap: '8px'
};

const sevColor = (s: string) =>
    s === 'critical' ? '#ff4d4d'
  : s === 'high'     ? '#ff8533'
  : s === 'medium'   ? '#ffcc00'
  : s === 'low'      ? '#33ff77'
  : 'rgba(255,255,255,0.4)';

// ── Components ────────────────────────────────────────────────────────────

const RiskGauge: Component<{ score: number }> = (props) => {
    const color = () => props.score >= 80 ? '#ff4d4d' : props.score >= 50 ? '#ffcc00' : '#33ff77';
    return (
        <div style={{ position: 'relative', width: '80px', height: '80px' }}>
            <svg viewBox="0 0 36 36" style={{ width: '100%', height: '100%', transform: 'rotate(-90deg)' }}>
                <path
                    d="M18 2.0845 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831"
                    fill="none"
                    stroke="rgba(255,255,255,0.1)"
                    stroke-width="3"
                />
                <path
                    d="M18 2.0845 a 15.9155 15.9155 0 0 1 0 31.831 a 15.9155 15.9155 0 0 1 0 -31.831"
                    fill="none"
                    stroke={color()}
                    stroke-width="3"
                    stroke-dasharray={`${props.score}, 100`}
                    style={{ transition: 'stroke-dasharray 0.6s ease' }}
                />
            </svg>
            <div style={{
                position: 'absolute', top: '50%', left: '50%', transform: 'translate(-50%, -50%)',
                'font-size': '20px', 'font-weight': '800', 'font-family': 'var(--font-mono)', color: color()
            }}>
                {props.score}
            </div>
        </div>
    );
};

export const EntityView: Component<EntityViewProps> = (props) => {
    const params = useParams();
    const navigate = useNavigate();

    const entityType = () => (props.entityType ?? params.type ?? 'host') as EntityType;
    const entityId   = () => props.entityId ?? params.id ?? '';

    const [loading, setLoading] = createSignal(true);
    const [enrichment, setEnrichment] = createSignal<services.EntityEnrichment | null>(null);
    const [events, setEvents] = createSignal<database.HostEvent[]>([]);
    const [incidents, setIncidents] = createSignal<database.Incident[]>([]);
    const [riskScore, setRiskScore] = createSignal(0);
    const [uebaProfile, setUebaProfile] = createSignal<ueba.EntityProfile | null>(null);
    const [host, setHost] = createSignal<database.Host | null>(null);

    const load = async () => {
        setLoading(true);
        try {
            const id = entityId();
            const type = entityType();

            // Load all relevant data
            const [enr, evts, incs, hostList, profiles] = await Promise.all([
                GetEntityEnrichment(id, type),
                type === 'host' ? GetHostEvents(id, 50) : SearchHostEvents(`${type}:"${id}"`, 50),
                ListIncidents('', 100),
                type === 'host' ? ListHosts() : Promise.resolve([]),
                GetProfiles(),
            ]);

            setEnrichment(enr);
            setEvents(evts ?? []);
            
            // Filter incidents
            const filteredIncs = (incs ?? []).filter((inc: database.Incident) => (inc.group_key ?? '').includes(id));
            setIncidents(filteredIncs);

            if (type === 'host') {
                const found = hostList.find((h: database.Host) => h.id === id || h.hostname === id);
                setHost(found ?? null);
                const score = await GetRiskScoreByHost(id);
                setRiskScore(score || 0);
            }

            const profile = (profiles ?? []).find((p: ueba.EntityProfile) => p.id === id || p.id.includes(id));
            setUebaProfile(profile ?? null);
            if (!riskScore() && profile) {
                setRiskScore(Math.round((profile.risk_score || 0) * 100));
            }

        } catch (e) {
            console.error("Failed to load entity data", e);
        } finally {
            setLoading(false);
        }
    };

    onMount(load);

    return (
        <div style={{
            display: 'flex', 'flex-direction': 'column', height: '100%',
            background: 'linear-gradient(135deg, #0a0a0c 0%, #16161a 100%)',
            color: '#fff', overflow: 'hidden', 'font-family': 'var(--font-ui)'
        }}>
            {/* Header Area */}
            <div style={{
                padding: '24px 32px', display: 'flex', 'align-items': 'center', 'justify-content': 'space-between',
                'border-bottom': '1px solid rgba(255,255,255,0.05)'
            }}>
                <div style={{ display: 'flex', 'align-items': 'center', gap: '20px' }}>
                    <button
                        onClick={() => navigate(-1)}
                        style={{
                            background: 'rgba(255,255,255,0.05)', border: '1px solid rgba(255,255,255,0.1)',
                            width: '40px', height: '40px', 'border-radius': '50%', color: '#fff',
                            cursor: 'pointer', display: 'flex', 'align-items': 'center', 'justify-content': 'center'
                        }}
                    >←</button>
                    <div>
                        <div style={{ 'font-size': '10px', 'text-transform': 'uppercase', 'letter-spacing': '2px', color: '#00aaff', 'margin-bottom': '4px' }}>
                            INVESTIGATION UNIT // {entityType()}
                        </div>
                        <div style={{ 'font-size': '32px', 'font-weight': '800', 'letter-spacing': '-0.5px' }}>
                            {host()?.label || entityId()}
                        </div>
                    </div>
                </div>

                <div style={{ display: 'flex', 'align-items': 'center', gap: '24px' }}>
                    <RiskGauge score={riskScore()} />
                    <div style={{ display: 'flex', gap: '8px' }}>
                        <button style={{ ...glassStyle, padding: '10px 20px', 'font-weight': '700', 'font-size': '13px', cursor: 'pointer' }}>⚡ ACTIONS</button>
                        <button style={{ ...glassStyle, padding: '10px 20px', 'font-weight': '700', 'font-size': '13px', cursor: 'pointer', background: 'rgba(0,170,255,0.2)', 'border-color': 'rgba(0,170,255,0.4)' }}>🛡 MONITOR</button>
                    </div>
                </div>
            </div>

            <Show when={loading()}>
              <div style={{ flex: 1, display: 'flex', 'align-items': 'center', 'justify-content': 'center', 'font-family': 'var(--font-mono)', opacity: 0.5 }}>
                  INITIALIZING FORENSIC ENVIRONMENT...
              </div>
            </Show>

            <Show when={!loading()}>
                <div style={{ flex: 1, overflow: 'auto', padding: '32px', display: 'grid', 'grid-template-columns': '350px 1fr 300px', gap: '24px' }}>
                    
                    {/* Left Column: Context & Enrichment */}
                    <div style={{ display: 'flex', 'flex-direction': 'column', gap: '24px' }}>
                        
                        {/* GeoIP Card */}
                        <div style={{ ...glassStyle, padding: '20px' }}>
                            <div style={cardHeaderStyle}>🌐 GEOLOCATION CONTEXT</div>
                            <div style={{ display: 'flex', 'align-items': 'center', gap: '16px', 'margin-bottom': '16px' }}>
                                <div style={{ width: '48px', height: '48px', background: 'rgba(255,255,255,0.05)', 'border-radius': '8px', display: 'flex', 'align-items': 'center', 'justify-content': 'center', 'font-size': '24px' }}>
                                    🗺️
                                </div>
                                <div>
                                    <div style={{ 'font-size': '16px', 'font-weight': '700' }}>{enrichment()?.location || 'Unknown'}</div>
                                    <div style={{ 'font-size': '12px', opacity: 0.5 }}>{enrichment()?.asn || 'Internal Network'}</div>
                                </div>
                            </div>
                            <div style={{ height: '120px', background: 'rgba(0,0,0,0.3)', 'border-radius': '8px', border: '1px solid rgba(255,255,255,0.05)', display: 'flex', 'align-items': 'center', 'justify-content': 'center', color: 'rgba(255,255,255,0.2)', 'font-size': '11px', 'text-transform': 'uppercase' }}>
                                [ MAP VISUALIZATION PENDING ]
                            </div>
                        </div>

                        {/* Threat Intel Card */}
                        <div style={{ ...glassStyle, padding: '20px', border: enrichment()?.ioc_match ? `1px solid ${sevColor(enrichment()!.severity)}66` : undefined }}>
                            <div style={cardHeaderStyle}>🛡️ THREAT INTELLIGENCE</div>
                            <Show when={enrichment()?.ioc_match} fallback={
                                <div style={{ color: 'rgba(255,255,255,0.4)', 'font-size': '12px' }}>
                                    Zero IOC matches found in current threat landscape.
                                </div>
                            }>
                                <div style={{ display: 'flex', 'flex-direction': 'column', gap: '12px' }}>
                                    <div style={{ color: sevColor(enrichment()!.severity), 'font-weight': '800', 'font-size': '14px' }}>
                                        ⚠️ {enrichment()?.severity.toUpperCase()} MATCH: {enrichment()?.ioc_source}
                                    </div>
                                    <div style={{ 'font-size': '12px', opacity: 0.8, 'line-height': '1.5' }}>
                                        {enrichment()?.ioc_desc}
                                    </div>
                                </div>
                            </Show>
                        </div>

                        {/* Risk Trend */}
                        <div style={{ ...glassStyle, padding: '20px' }}>
                            <div style={cardHeaderStyle}>📈 RISK VELOCITY</div>
                            <div style={{ height: '60px', 'margin-top': '10px' }}>
                                <Sparkline 
                                    data={uebaProfile()?.risk_history?.map(p => p.score) || [10, 12, 11, 15, 14, 18, 22, 20]} 
                                    color={riskScore() > 70 ? '#ff4d4d' : '#00aaff'} 
                                    height={60}
                                />
                            </div>
                        </div>
                    </div>

                    {/* Middle Column: Activity Timeline */}
                    <div style={{ ...glassStyle, padding: '0', overflow: 'hidden', display: 'flex', 'flex-direction': 'column' }}>
                        <div style={{ padding: '20px', ...cardHeaderStyle, 'margin-bottom': '0', 'border-bottom': '1px solid rgba(255,255,255,0.05)' }}>
                            🕒 ACTIVITY CHRONOLOGY
                        </div>
                        <div style={{ flex: 1, overflow: 'auto', padding: '24px' }}>
                            <div style={{ 'border-left': '2px solid rgba(255,255,255,0.05)', 'margin-left': '12px', padding: '0 0 0 24px', display: 'flex', 'flex-direction': 'column', gap: '32px' }}>
                                <For each={events().slice(0, 20)}>
                                    {(evt) => (
                                        <div style={{ position: 'relative' }}>
                                            <div style={{ 
                                                position: 'absolute', left: '-33px', top: '4px', width: '16px', height: '16px', 
                                                border: '2px solid #16161a', 'border-radius': '50%', 
                                                background: evt.event_type.includes('FAIL') ? '#ff4d4d' : '#00aaff',
                                                'box-shadow': `0 0 8px ${evt.event_type.includes('FAIL') ? '#ff4d4d' : '#00aaff'}`
                                            }} />
                                            <div style={{ 'font-size': '10px', 'font-family': 'var(--font-mono)', opacity: 0.4, 'margin-bottom': '4px' }}>
                                                {new Date(evt.timestamp).toLocaleString()}
                                            </div>
                                            <div style={{ 'font-weight': '700', 'font-size': '14px', 'margin-bottom': '4px' }}>
                                                {evt.event_type}
                                            </div>
                                            <div style={{ 'font-size': '12px', opacity: 0.6, 'font-family': 'var(--font-mono)', 'background': 'rgba(255,255,255,0.02)', padding: '8px', 'border-radius': '4px' }}>
                                                {evt.raw_log.length > 150 ? evt.raw_log.substring(0, 150) + '...' : evt.raw_log}
                                            </div>
                                        </div>
                                    )}
                                </For>
                            </div>
                        </div>
                    </div>

                    {/* Right Column: Alerts & Related Entities */}
                    <div style={{ display: 'flex', 'flex-direction': 'column', gap: '24px' }}>
                        <div style={{ ...glassStyle, padding: '20px' }}>
                            <div style={cardHeaderStyle}>🚨 RELEVANT INCIDENTS</div>
                            <div style={{ display: 'flex', 'flex-direction': 'column', gap: '12px' }}>
                                <For each={incidents().slice(0, 5)} fallback={
                                    <div style={{ opacity: 0.4, 'font-size': '12px', 'text-align': 'center', padding: '20px' }}>NO ACTIVE INCIDENTS</div>
                                }>
                                    {(inc) => (
                                        <div style={{ 
                                            padding: '12px', background: 'rgba(255,255,255,0.03)', 'border-radius': '8px', 
                                            border: `1px solid ${sevColor(inc.severity)}33`,
                                            cursor: 'pointer'
                                        }} onClick={() => navigate(`/alerts?id=${inc.id}`)}>
                                            <div style={{ 'font-size': '10px', 'font-weight': '800', color: sevColor(inc.severity), 'margin-bottom': '4px' }}>{inc.severity.toUpperCase()}</div>
                                            <div style={{ 'font-size': '12px', 'font-weight': '600' }}>{inc.title}</div>
                                        </div>
                                    )}
                                </For>
                            </div>
                        </div>

                        <div style={{ ...glassStyle, padding: '20px' }}>
                            <div style={cardHeaderStyle}>🧬 UEBA SIGNALS</div>
                            <div style={{ display: 'flex', 'flex-direction': 'column', gap: '8px' }}>
                                <For each={Object.entries(uebaProfile()?.features || {}).slice(0, 5)}>
                                    {([key, val]) => (
                                        <div style={{ display: 'flex', 'justify-content': 'space-between', 'font-size': '11px' }}>
                                            <span style={{ opacity: 0.5 }}>{key}</span>
                                            <span style={{ color: '#00aaff', 'font-family': 'var(--font-mono)' }}>{Number(val).toFixed(2)}</span>
                                        </div>
                                    )}
                                </For>
                            </div>
                        </div>
                    </div>
                </div>
            </Show>
        </div>
    );
};

export default EntityView;
