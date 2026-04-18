// TypeScript interfaces for external log sources

export interface LogSource {
    id: string;
    name: string;
    type: string;
    url: string;
    enabled: boolean;
    api_key?: string;
    username?: string;
    password?: string;
    index?: string;
    org_id?: string;
    tls_skip_verify?: boolean;
    tags?: string[];
}

export interface LogResult {
    timestamp: string;
    source: string;
    host: string;
    message: string;
    level: string;
    fields?: Record<string, string>;
}

export interface ConnectionResult {
    ok: boolean;
    message: string;
}

// Alert types
export interface AlertTrigger {
    id: string;
    name: string;
    pattern: string;
    severity: 'critical' | 'high' | 'medium' | 'low';
    enabled: boolean;
}

export interface AlertEvent {
    timestamp: string;
    trigger_id: string;
    name: string;
    severity: string;
    host: string;
    session_id: string;
    log_line: string;
    sent: boolean;
}

// Notification types
export interface NotificationConfig {
    smtp_host: string;
    smtp_port: number;
    smtp_username: string;
    smtp_password: string;
    to_email: string;
    telegram_token: string;
    telegram_chat_id: string;
    twilio_sid: string;
    twilio_token: string;
    twilio_from: string;
    to_phone: string;
    enable_email: boolean;
    enable_telegram: boolean;
    enable_sms: boolean;
    enable_whatsapp: boolean;
}

// Severity helpers
export const SEVERITY_ORDER: Record<string, number> = {
    critical: 0, error: 1, warn: 2, warning: 2, info: 3, debug: 4, trace: 5
};

export const SEVERITY_COLORS: Record<string, string> = {
    critical: '#ff2d55', error: '#f85149', warn: '#d29922', warning: '#d29922',
    info: '#3fb950', debug: '#58a6ff', trace: '#8b949e'
};
