# OBLIVRA Architecture: Desktop vs. Browser Context

## Overview

OBLIVRA runs as a single codebase across two deployment models. The `APP_CONTEXT`
constant (detected at module load in `frontend/src/core/context.ts`) is the single
source of truth for every routing decision and feature gate.

| Context | Detected by | Use case |
|---|---|---|
| `desktop` | `window.__WAILS__` present, no remote server | Individual engineer / admin |
| `browser` | No `window.__WAILS__` | SOC team on enterprise server |
| `hybrid` | `window.__WAILS__` + `localStorage:oblivra:remote_server` set | Desktop binary + remote OBLIVRA server |

---

## Context Detection

```typescript
// frontend/src/core/context.ts
export const APP_CONTEXT: AppContext = detectContext();
export const IS_DESKTOP = APP_CONTEXT === 'desktop';
export const IS_BROWSER  = APP_CONTEXT === 'browser';
export const IS_HYBRID   = APP_CONTEXT === 'hybrid';
```

`detectContext()` reads:
1. `window.__WAILS__` — injected by the Wails WebView host at startup
2. `localStorage:oblivra:remote_server` — set by the operator in Settings

---

## Route Availability Matrix

| Route | Desktop | Browser | Hybrid |
|---|:---:|:---:|:---:|
| `/dashboard` | ✅ | ✅ | ✅ |
| `/siem` | ✅ | ✅ | ✅ |
| `/alerts` | ✅ | ✅ | ✅ |
| `/compliance` | ✅ | ✅ | ✅ |
| `/vault` | ✅ | ✅ | ✅ |
| `/mitre-heatmap` | ✅ | ✅ | ✅ |
| `/terminal` | ✅ | ❌ | ✅ |
| `/tunnels` | ✅ | ❌ | ✅ |
| `/recordings` | ✅ | ❌ | ✅ |
| `/snippets` | ✅ | ❌ | ✅ |
| `/notes` | ✅ | ❌ | ✅ |
| `/sync` | ✅ | ❌ | ✅ |
| `/offline-update` | ✅ | ❌ | ✅ |
| `/agents` | ❌ | ✅ | ✅ |
| `/fleet` | ❌ | ✅ | ✅ |
| `/identity` | ❌ | ✅ | ✅ |
| `/soc` | ❌ | ✅ | ✅ |

Enforced by the `RouteGuard` component in `frontend/src/core/RouteGuard.tsx`.

---

## Service Capabilities

```typescript
getServiceCapabilities() → {
    localTerminal:    IS_DESKTOP || IS_HYBRID,
    osKeychain:       IS_DESKTOP,
    localSftp:        IS_DESKTOP || IS_HYBRID,
    enterpriseAuth:   IS_BROWSER || IS_HYBRID,
    agentFleet:       IS_BROWSER || IS_HYBRID,
    clustering:       IS_BROWSER || IS_HYBRID,
    socCollaboration: IS_BROWSER || IS_HYBRID,
    remoteServer:     IS_HYBRID,
    remoteServerUrl:  string | null,
}
```

---

## Hybrid Mode

The most powerful configuration. The desktop binary acts as a thick client
while connected to a remote OBLIVRA server:

- Local PTY, OS keychain, direct SFTP (via Wails)
- Enterprise fleet management, agent console, identity (via remote server API)
- Entity pivot: click an IP in the local terminal → entity page loads from remote SIEM context

### Enabling hybrid mode

```typescript
configureHybridMode('https://siem.yourorg.com:8080');
// Writes to localStorage:oblivra:remote_server and reloads the page.
// APP_CONTEXT becomes 'hybrid' after reload.
```

### Disabling

```typescript
disconnectHybridMode();
// Clears the server URL and reloads. APP_CONTEXT returns to 'desktop'.
```

The `ContextBadge` component in the status bar provides a UI for this.

---

## Bridge Behavior by Context

| | Desktop | Browser | Hybrid |
|---|---|---|---|
| `initBridge()` | Waits for `window.runtime` (max 2s) | Resolves immediately | Waits for `window.runtime` |
| `subscribe(event, cb)` | Wails `EventsOn` | No-op (browser uses WebSocket) | Wails `EventsOn` |
| Wails IPC calls | Available | Not available | Available |

---

## Go Backend Service Registration

The backend service container already supports selective service startup:

```go
// container.go — services only started if platform supports them
if mode == ServerMode {
    c.Register(NewIdentityService(...))  // Enterprise Auth
    c.Register(NewClusterService(...))   // Raft/Scaling
}
if mode == DesktopMode {
    c.Register(NewLocalTerminalService(...)) // Direct PTY
    c.Register(NewKeychainService(...))      // OS Secrets
}
```

This is wired at startup from `DeploymentMode` which reads the same
`OBLIVRA_MODE` environment variable that the server sets.
