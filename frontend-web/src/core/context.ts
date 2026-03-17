/**
 * OBLIVRA — Web Context
 *
 * This version of context.ts is tailored for the OBLIVRA Web Dashboard.
 * It defaults to 'browser' context but retains the shared interface.
 */

export type AppContext = 'desktop' | 'browser' | 'hybrid';

function detectContext(): AppContext {
    // In the dedicated web dashboard, we are almost always in 'browser' mode.
    // However, if we're running in a Wails environment (e.g. for testing or 
    // unified builds), we detect it here.
    const isWails = !!(window as any).__WAILS__ || !!(window as any).runtime;

    if (!isWails) return 'browser';

    const remoteServer = localStorage.getItem('oblivra:remote_server');
    if (remoteServer && remoteServer.trim() !== '') return 'hybrid';

    return 'desktop';
}

export const APP_CONTEXT: AppContext = detectContext();

export const IS_DESKTOP = APP_CONTEXT === 'desktop';
export const IS_BROWSER  = APP_CONTEXT === 'browser';
export const IS_HYBRID   = APP_CONTEXT === 'hybrid';

export interface ServiceCapabilities {
    localTerminal: boolean;
    osKeychain: boolean;
    localSftp: boolean;
    enterpriseAuth: boolean;
    agentFleet: boolean;
    clustering: boolean;
    socCollaboration: boolean;
    remoteServer: boolean;
    remoteServerUrl: string | null;
}

export function getServiceCapabilities(): ServiceCapabilities {
    const remoteUrl = localStorage.getItem('oblivra:remote_server') ?? null;

    return {
        localTerminal:   IS_DESKTOP || IS_HYBRID,
        osKeychain:      IS_DESKTOP,
        localSftp:       IS_DESKTOP || IS_HYBRID,
        enterpriseAuth:  IS_BROWSER || IS_HYBRID,
        agentFleet:      IS_BROWSER || IS_HYBRID,
        clustering:      IS_BROWSER || IS_HYBRID,
        socCollaboration:IS_BROWSER || IS_HYBRID,
        remoteServer:    IS_HYBRID,
        remoteServerUrl: IS_HYBRID ? remoteUrl : null,
    };
}
