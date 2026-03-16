import { Component, createSignal, For } from 'solid-js';
import { SignData } from '../../../wailsjs/go/services/ComplianceService';
import { EmptyState } from '../ui/EmptyState';
import '../../styles/workflow.css';

interface ApprovalRequest {
    id: string;
    title: string;
    requester: string;
    host: string;
    snippet: string;
    severity: 'high' | 'medium';
    timestamp: Date;
}

const ComplianceWorkflow: Component = () => {
    // Mock pending requests for the demonstration/phase 6 completion
    const [requests, setRequests] = createSignal<ApprovalRequest[]>([
        {
            id: 'req-1',
            title: 'Kernel Memory Patch (Hotfix)',
            requester: 'System Automaton (AI)',
            host: 'prod-db-01',
            snippet: 'sysctl -w vm.max_map_count=262144',
            severity: 'high',
            timestamp: new Date()
        },
        {
            id: 'req-2',
            title: 'Log Rotation Expansion',
            requester: 'Compliance-Bot',
            host: 'web-front-04',
            snippet: 'sed -i "s/rotate 7/rotate 30/g" /etc/logrotate.d/nginx',
            severity: 'medium',
            timestamp: new Date()
        }
    ]);

    const handleApprove = async (req: ApprovalRequest) => {
        try {
            // Digitally sign the approval token
            const payload = JSON.stringify({
                action: 'APPROVE',
                request_id: req.id,
                snippet: req.snippet,
                approver: 'Admin-Current'
            });

            const result = await SignData(payload);
            console.log("Approval Signed and Logic Validated:", result);

            // Remove from local list as if processed
            setRequests(requests().filter(r => r.id !== req.id));
            alert(`Snippet Approved & Digitally Signed via Vault Hash: ${result.signature.substring(0, 12)}...`);
        } catch (err) {
            console.error("Signing failed:", err);
        }
    };

    const handleReject = (id: string) => {
        setRequests(requests().filter(r => r.id !== id));
    };

    return (
        <div class="compliance-workflow">
            <div class="workflow-header">
                <h2>
                    <span>Digital Authorization Workflow</span>
                    <span class="modifier-count">{requests().length} Pending</span>
                </h2>
                <div class="heatmap-legend">
                    <div class="legend-item">
                        <span class="legend-dot" style="background: #10b981;"></span>
                        <span>Vault-Backed Signatures Active</span>
                    </div>
                </div>
            </div>

            <div class="request-list">
                <For each={requests()}>
                    {(req) => (
                        <div class="request-card">
                            <div class="request-meta">
                                <div class="request-info">
                                    <div class="request-title">{req.title}</div>
                                    <div class="request-subtitle">
                                        Requested by {req.requester} for {req.host} • {req.timestamp.toLocaleTimeString()}
                                    </div>
                                </div>
                                <span class={`severity-badge sev-${req.severity}`}>
                                    {req.severity} Risk
                                </span>
                            </div>

                            <div class="snippet-preview">
                                {req.snippet}
                            </div>

                            <div class="action-row">
                                <button class="btn-reject" onClick={() => handleReject(req.id)}>Reject</button>
                                <button class="btn-approve" onClick={() => handleApprove(req)}>Approve & Sign</button>
                            </div>
                        </div>
                    )}
                </For>

                {requests().length === 0 && (
                    <EmptyState
                        icon="✅"
                        title="All Operations Authorized"
                        description="Fleet compliance status is nominal. Pending requests will appear here."
                        compact
                    />
                )}
            </div>
        </div>
    );
};

export default ComplianceWorkflow;
