export interface DecisionCardData {
    id: string;
    title: string;
    severity: 'P0' | 'P1' | 'P2' | 'P3';
    confidence: number;          // 0–100
    recommendedAction: string;
    impactSeconds: number;       // seconds until impact (countdown)
    source?: string;
    host?: string;
    mitre?: string;
    timestamp?: string;
    dataConfidence?: 'Verified' | 'Derived' | 'Untrusted';
    isChained?: boolean; // True if part of a tamper-evident event chain
}
