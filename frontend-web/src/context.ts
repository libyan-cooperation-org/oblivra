/**
 * OBLIVRA — Web Context
 * Detects whether running inside Wails Desktop or browser.
 * Pure TypeScript — no framework dependency.
 */

export type AppContext = 'desktop' | 'browser' | 'hybrid';

function detectContext(): AppContext {
  const isWails = !!(window as any).__WAILS__ || !!(window as any).runtime;
  if (!isWails) return 'browser';
  const remoteServer = localStorage.getItem('oblivra:remote_server');
  if (remoteServer && remoteServer.trim() !== '') return 'hybrid';
  return 'desktop';
}

export const APP_CONTEXT: AppContext = detectContext();
export const IS_DESKTOP = APP_CONTEXT === 'desktop';
export const IS_BROWSER = APP_CONTEXT === 'browser';
export const IS_HYBRID  = APP_CONTEXT === 'hybrid';

// Legacy helpers used by services/api.ts
export const isDesktop = (): boolean => IS_DESKTOP;
export const isWeb     = (): boolean => IS_BROWSER;

export interface ServiceCapabilities {
  localTerminal:    boolean;
  osKeychain:       boolean;
  localSftp:        boolean;
  enterpriseAuth:   boolean;
  agentFleet:       boolean;
  clustering:       boolean;
  socCollaboration: boolean;
  remoteServer:     boolean;
  remoteServerUrl:  string | null;
}

export function getServiceCapabilities(): ServiceCapabilities {
  const remoteUrl = localStorage.getItem('oblivra:remote_server') ?? null;
  return {
    localTerminal:    IS_DESKTOP || IS_HYBRID,
    osKeychain:       IS_DESKTOP,
    localSftp:        IS_DESKTOP || IS_HYBRID,
    enterpriseAuth:   IS_BROWSER || IS_HYBRID,
    agentFleet:       IS_BROWSER || IS_HYBRID,
    clustering:       IS_BROWSER || IS_HYBRID,
    socCollaboration: IS_BROWSER || IS_HYBRID,
    remoteServer:     IS_HYBRID,
    remoteServerUrl:  IS_HYBRID ? remoteUrl : null,
  };
}
