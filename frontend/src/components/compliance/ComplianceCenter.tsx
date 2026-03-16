import { Component, createSignal, onMount, For, Show, createResource } from 'solid-js';
import {
    GenerateReport,
    ListReports,
    ListCompliancePacks,
    EvaluatePack,
    ExportReportPDF
} from '../../../wailsjs/go/services/ComplianceService';
import { GetBiasLogs } from '../../../wailsjs/go/services/GovernanceService';
import { compliance } from '../../../wailsjs/go/models';
import { EmptyState } from '../ui/EmptyState';
import { PolicyVerifier } from './PolicyVerifier';
import '../../styles/compliance.css';

export const ComplianceCenter: Component = () => {
    const [activeTab, setActiveTab] = createSignal<'reports' | 'governance' | 'verification'>('reports');
    const [reports, setReports] = createSignal<compliance.ComplianceReport[]>([]);
    const [selectedReport, setSelectedReport] = createSignal<compliance.ComplianceReport | null>(null);
    const [, setLoading] = createSignal(false);

    const [generating, setGenerating] = createSignal(false);
    const [reportType, setReportType] = createSignal('soc2');

    // Governance State
    const [packs] = createResource(ListCompliancePacks);
    const [evalResults, setEvalResults] = createSignal<Record<string, compliance.PackResult>>({});
    // evaluating is unused, removing
    const [biasLogs, { refetch: refetchBiasLogs }] = createResource(async () => {
        try {
            return await GetBiasLogs();
        } catch (e) {
            return [];
        }
    });

    const loadReports = async () => {
        setLoading(true);
        try {
            const res = await ListReports();
            setReports(res || []);
            if (res && res.length > 0 && !selectedReport()) {
                setSelectedReport(res[0]);
            }
        } catch (err) {
            console.error("Failed to list reports:", err);
        } finally {
            setLoading(false);
        }
    };

    const handleGenerate = async () => {
        setGenerating(true);
        try {
            const end = Math.floor(Date.now() / 1000);
            const start = end - (30 * 24 * 60 * 60);
            const report = await GenerateReport(reportType(), start, end);
            if (report) {
                setReports([report, ...reports()]);
                setSelectedReport(report);
            }
        } catch (err) {
            console.error("Report generation failed:", err);
        } finally {
            setGenerating(false);
        }
    };

    const handleEvaluate = async (packId: string) => {
        try {
            const result = await EvaluatePack(packId);
            setEvalResults(prev => ({ ...prev, [packId]: result }));
        } catch (err) {
            console.error(`Evaluation failed for ${packId}:`, err);
        } finally {
            // setEvaluating(null);
        }
    };

    const handleExportPDF = async (report: compliance.ComplianceReport) => {
        try {
            const start = Math.floor(new Date(report.period_start.toString()).getTime() / 1000);
            const end = Math.floor(new Date(report.period_end.toString()).getTime() / 1000);
            const path = await ExportReportPDF(report.type, start, end);
            alert(`Report exported to: ${path}`);
        } catch (err) {
            console.error("PDF export failed:", err);
        }
    };

    onMount(() => {
        loadReports();
    });

    return (
        <div class="compliance-container">
            <header class="compliance-header">
                <div class="header-title">
                    <h1>Compliance Center</h1>
                    <p>Formal security audits and governance reporting</p>
                </div>
                <div class="header-actions">
                    <div class="compliance-tabs">
                        <button
                            class={`tab-btn ${activeTab() === 'reports' ? 'active' : ''}`}
                            onClick={() => setActiveTab('reports')}
                        >
                            Audits & Reports
                        </button>
                        <button
                            class={`tab-btn ${activeTab() === 'governance' ? 'active' : ''}`}
                            onClick={() => setActiveTab('governance')}
                        >
                            Governance Dashboard
                        </button>
                        <button
                            class={`tab-btn ${activeTab() === 'verification' ? 'active' : ''}`}
                            onClick={() => setActiveTab('verification')}
                        >
                            Policy Verification
                        </button>
                    </div>
                </div>
            </header>

            <Show when={activeTab() === 'reports'}>
                <div class="compliance-layout">
                    <aside class="reports-sidebar glass-panel">
                        <div class="sidebar-header">Previous Audits</div>
                        <div class="reports-list">
                            <For each={reports()}>
                                {(report) => (
                                    <button
                                        class={`report-item ${selectedReport()?.id === report.id ? 'active' : ''}`}
                                        onClick={() => setSelectedReport(report)}
                                    >
                                        <div class="report-meta">
                                            <span class="report-type">{report.type.toUpperCase()}</span>
                                            <span class="report-date">{new Date(report.generated_at.toString()).toLocaleDateString()}</span>
                                        </div>
                                        <h4 style="margin: 0 0 8px 0; font-size: 14px;">{report.title}</h4>
                                        <div style="font-size: 11px; margin-bottom: 15px; color: var(--text-muted); opacity: 0.8;">
                                            Period: {new Date(report.period_start.toString()).toLocaleDateString()} - {new Date(report.period_end.toString()).toLocaleDateString()}
                                        </div>
                                    </button>
                                )}
                            </For>
                        </div>
                        <div class="sidebar-footer" style="padding: 16px; border-top: 1px solid var(--glass-border);">
                            <select
                                class="siem-select"
                                style="width: 100%; margin-bottom: 12px;"
                                value={reportType()}
                                onChange={(e) => setReportType(e.currentTarget.value)}
                            >
                                <option value="soc2">SOC2 Type II</option>
                                <option value="pci-dss">PCI-DSS v4.0</option>
                                <option value="hipaa">HIPAA / HITECH</option>
                                <option value="general">General Security Audit</option>
                            </select>
                            <button
                                class="btn btn-primary"
                                style="width: 100%;"
                                onClick={handleGenerate}
                                disabled={generating()}
                            >
                                {generating() ? 'Generating...' : '✨ New Audit'}
                            </button>
                        </div>
                    </aside>

                    <main class="report-viewer">
                        <Show when={selectedReport()} fallback={
                            <EmptyState
                                icon="⚖️"
                                title="Audit Readiness"
                                description="Select a report to view detailed findings or generate a new one to verify current security posture."
                            />
                        }>
                            <div class="report-content glass-panel">
                                <div class="report-header-actions" style="display: flex; justify-content: flex-end; gap: 12px; margin-bottom: -20px;">
                                    <button class="btn btn-secondary btn-sm" onClick={() => handleExportPDF(selectedReport()!)}>
                                        📄 Export PDF
                                    </button>
                                </div>
                                <div class="report-hero">
                                    <div class="hero-main">
                                        <span class="badge">{selectedReport()!.type.toUpperCase()} Compliant</span>
                                        <h2>{selectedReport()!.title}</h2>
                                        <p class="period">Period: {new Date(selectedReport()!.period_start.toString()).toLocaleDateString()} - {new Date(selectedReport()!.period_end.toString()).toLocaleDateString()}</p>
                                    </div>
                                    <div class="score-widget">
                                        <div class="score-value">{selectedReport()!.summary.compliance_score.toFixed(0)}%</div>
                                        <div class="score-label">Compliance Score</div>
                                    </div>
                                </div>

                                <div class="stats-row">
                                    <div class="stat-card">
                                        <span class="label">Total Sessions</span>
                                        <span class="value">{selectedReport()!.summary.total_sessions}</span>
                                    </div>
                                    <div class="stat-card">
                                        <span class="label">Unique Hosts</span>
                                        <span class="value">{selectedReport()!.summary.unique_hosts}</span>
                                    </div>
                                    <div class="stat-card danger">
                                        <span class="label">Critical Findings</span>
                                        <span class="value">{selectedReport()!.summary.critical_findings}</span>
                                    </div>
                                </div>

                                <div class="report-sections">
                                    <h3>Detailed Findings</h3>
                                    <div class="findings-grid">
                                        <For each={selectedReport()!.findings}>
                                            {(finding) => (
                                                <div class={`finding-card ${finding.severity}`}>
                                                    <div class="finding-header">
                                                        <span class="severity-badge">{finding.severity}</span>
                                                        <span class="category">{finding.category}</span>
                                                    </div>
                                                    <div class="finding-title">{finding.title}</div>
                                                    <div class="finding-desc">{finding.description}</div>
                                                    <Show when={finding.remediation}>
                                                        <div class="remediation">
                                                            <strong>Remediation:</strong> {finding.remediation}
                                                        </div>
                                                    </Show>
                                                </div>
                                            )}
                                        </For>
                                    </div>
                                </div>
                            </div>
                        </Show>
                    </main>
                </div>
            </Show>

            <Show when={activeTab() === 'governance'}>
                <div class="governance-layout" style="display: grid; grid-template-columns: 1fr 400px; gap: 24px;">
                    <div class="packs-section">
                        <header class="section-header" style="margin-bottom: 20px;">
                            <h3>COMPLIANCE PACKS</h3>
                            <div class="header-line"></div>
                        </header>
                        <div class="governance-grid" style="display: grid; grid-template-columns: repeat(auto-fill, minmax(350px, 1fr)); gap: 20px;">
                            <For each={packs()}>
                                {(pack) => {
                                    const result = () => evalResults()[pack.id];
                                    return (
                                        <div class="pack-card" onClick={() => handleEvaluate(pack.id)}>
                                            <div class="pack-header">
                                                <div class="pack-info">
                                                    <h4>{pack.name}</h4>
                                                    <span>v{pack.version} • {pack.category}</span>
                                                </div>
                                                <div class="pack-score">
                                                    <div class="score-num">
                                                        {result() ? result()!.score.toFixed(0) + '%' : '--'}
                                                    </div>
                                                </div>
                                            </div>
                                            <div class="pack-progress">
                                                <div class="progress-bar" style={{ width: `${result()?.score || 0}%` }}></div>
                                            </div>
                                            <div class="pack-footer">
                                                <span>{pack.controls.length} Automated Controls</span>
                                                <button class="btn btn-secondary btn-sm">Scan Now</button>
                                            </div>
                                        </div>
                                    );
                                }}
                            </For>
                        </div>
                    </div>

                    <aside class="bias-audit-section glass-panel" style="padding: 20px;">
                        <header class="section-header">
                            <h3>AI DECISION GOVERNANCE</h3>
                            <button class="btn btn-icon btn-sm" onClick={() => refetchBiasLogs()}>🔄</button>
                        </header>
                        <p style="font-size: 11px; color: var(--text-muted); margin-bottom: 20px;">Cryptographic audit trail of human-in-the-loop decisions and model corrections.</p>

                        <div class="bias-logs-list" style={{ display: 'flex', 'flex-direction': 'column', gap: '12px' }}>
                            <For each={biasLogs() || []}>
                                {(entry: any) => (
                                    <div class="bias-entry-card" style={{ background: 'rgba(255,255,255,0.03)', border: '1px solid var(--glass-border)', padding: '12px', 'border-radius': '8px' }}>
                                        <div style={{ display: 'flex', 'justify-content': 'space-between', 'margin-bottom': '8px' }}>
                                            <span style={{ 'font-size': '10px', 'font-weight': '800', color: 'var(--tactical-amber)' }}>FALSE_POSITIVE_MARK</span>
                                            <span style={{ 'font-size': '10px', color: 'var(--text-muted)' }}>{new Date(entry.timestamp.toString()).toLocaleDateString()}</span>
                                        </div>
                                        <div style={{ 'font-family': "'JetBrains Mono', monospace", 'font-size': '12px', 'margin-bottom': '4px' }}>Entity: {entry.anomaly_id}</div>
                                        <div style={{ 'font-size': '11px', color: 'var(--text-secondary)', 'margin-bottom': '8px' }}>Reason: {entry.reason}</div>
                                        <div class="evidence-pill-row" style={{ display: 'flex', 'flex-wrap': 'wrap', gap: '4px' }}>
                                            <For each={entry.evidence || []}>
                                                {(ev: any) => (
                                                    <span style={{ 'font-size': '9px', background: 'rgba(0,0,0,0.2)', padding: '2px 6px', 'border-radius': '4px', border: '1px solid rgba(255,255,255,0.05)' }}>
                                                        {ev.key}: {ev.value}
                                                    </span>
                                                )}
                                            </For>
                                        </div>
                                    </div>
                                )}
                            </For>
                            <Show when={!biasLogs() || (biasLogs()?.length ?? 0) === 0}>
                                <div style="text-align: center; padding: 40px; color: var(--text-muted); font-size: 12px; border: 1px dashed var(--glass-border); border-radius: 8px;">
                                    No bias logs recorded. Decision loop is nominal.
                                </div>
                            </Show>
                        </div>
                    </aside>
                </div>
            </Show>

            <Show when={activeTab() === 'verification'}>
                <PolicyVerifier />
            </Show>
        </div>
    );
};

