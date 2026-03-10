// Global Wails API Type Definitions
export interface Violation {
    timestamp: string;
    host_id: string;
    type: string;
    detail: string;
    delta_ms: number;
}

export interface LineageRecord {
    id: string;
    entity_id: string;
    stage: string;
    timestamp: string;
    parent_id?: string;
    proof_hash: string;
}

export interface EvidenceLink {
    type: string;
    description: string;
    confidence: number;
}

export interface Alternative {
    rule_name: string;
    score: number;
    reason: string;
}

export interface DecisionTrace {
    id: string;
    timestamp: string;
    rule_id: string;
    rule_name: string;
    evidence_chain: EvidenceLink[];
    confidence_score: number;
    alternatives: Alternative[];
    explanation: string;
    crypto_proof: string;
}

declare global {
    interface Window {
        go: {
            app: {
                TemporalService: {
                    GetViolations(): Promise<Violation[]>;
                    GetAgentDrift(): Promise<Record<string, number>>;
                };
                LineageService: {
                    GetRecentLineage(limit: number): Promise<LineageRecord[]>;
                    GetStats(): Promise<any>;
                };
                DecisionService: {
                    ListRecentDecisions(limit: number): Promise<DecisionTrace[]>;
                    GetExplanation(id: string): Promise<string>;
                    GetProof(id: string): Promise<string>;
                };
                SettingsService?: any;
            };
        };
    }
}
