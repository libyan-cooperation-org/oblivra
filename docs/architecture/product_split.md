# OBLIVRA Architecture: Desktop vs. Browser Split

OBLIVRA is designed as a unified codebase that serves two distinct deployment models: the **Desktop Power Tool** and the **Enterprise Security Platform**.

## Core Philosophy

| Context | User | Mental Model | Network | Scale |
| :--- | :--- | :--- | :--- | :--- |
| **Desktop App (Wails)** | Individual Engineer/Admin | "My terminal, my vault, my connections" | Local or direct SSH | 1–50 hosts |
| **Browser (Server)** | SOC Team / Enterprise | "Our alerts, our investigations, our fleet" | Server-mediated | 50–100,000+ endpoints |

---

## Desktop-Exclusive Features
*Requires Wails/OS Runtime access.*

### Terminal & Shell
- **Local PTY terminal**: Spawns local shell processes directly.
- **SSH agent forwarding**: Reads local `SSH_AUTH_SOCK`.
- **SSH config parser**: Handles `~/.ssh/config` natively.
- **Low-latency rendering**: Native keypress handling in the terminal grid.

### OS & Filesystem Integration
- **OS Keychain**: Native integration with Windows Credential Manager, macOS Keychain, and Linux Secret Service.
- **Local SFTP Path**: Direct drag-and-drop from/to the local desktop filesystem.
- **Personal Workspace**: Local layout persistence and theme preferences.

---

## Browser-Exclusive Features
*Requires Server Deployment + Multi-User Access (Clustered Backend).*

### SOC Operations & Collaboration
- **Triage & Incident Hub**: Multi-analyst shared queues and "War Room" communication.
- **Investigation Notebooks**: Collaborative editing (CRDT) for shared case documentation.
- **SOC Metrics**: Global MTTR/MTTD tracking and analyst performance leaderboards.

### Enterprise Administration
- **Fleet Management**: Mass-push configurations to 100,000+ agents.
- **RBAC & Identity**: OIDC/SAML/MFA configuration and tenant management (MSSP mode).
- **Compliance Center**: Organizational-wide posture scoring and evidence generation.

---

## Hybrid Mode: The Tactical Sweet Spot

The Desktop App's most powerful configuration is **Hybrid Mode**. It functions as a local power tool (direct SSH, native PTY) while acting as a "thick client" for a remote OBLIVRA Server.

- **Analyst Workflow**: Click an IP in a local terminal → Entity Page opens with full remote SIEM context → Click "Investigate" → Remote Case opens.
- **Performance**: High-speed native terminal interaction + Enterprise-scale data indexing.

---

## Route Availability Matrix

| Route | Desktop | Browser | Notes |
| :--- | :---: | :---: | :--- |
| `/terminal` | ✅ | ✅* | Browser requires server SSH proxy |
| `/workspace` | ✅ | ❌ | Desktop-only concept |
| `/fleet` | ❌ | ✅ | Server-only; impossible at local scale |
| `/identity` | ❌ | ✅ | Managed at enterprise level |
| `/siem` | ✅ | ✅ | Local scope vs. Clustered scope |
| `/entity/:type/:id` | ✅ | ✅ | Deep entity context (local or remote) |

---

## Technical Implementation

### Frontend Context Detection
```typescript
// Detected at startup — Wails injects this, browser doesn't
export const APP_CONTEXT: 'desktop' | 'browser' = 
  (window as any).__WAILS__ ? 'desktop' : 'browser';
```

### Backend Service Registration
```go
func (c *Container) RegisterServices(mode DeploymentMode) {
    if mode == ServerMode {
        c.Register(NewIdentityService(...)) // Enterprise Auth
        c.Register(NewClusterService(...))  // Raft/Scaling
    }
    if mode == DesktopMode {
        c.Register(NewLocalTerminalService(...)) // Direct PTY
        c.Register(NewKeychainService(...))      // OS Secrets
    }
}
```
