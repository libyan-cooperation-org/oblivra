# Enterprise Deployment Guide

Sovereign Terminal is designed primarily as a standalone desktop application, but it can also be deployed in headless mode within data centers or cloud environments as a central SIEM ingestion point, API backend, or Raft cluster participant.

This guide details best practices for deploying the headless Go backend in an enterprise setting.

## Infrastructure Architecture

When running headlessly, Sovereign Terminal provides:
1.  **SIEM Ingestion Layer**: Accepting events through the REST API (`/api/v1/siem/ingest`) and EventBus WebSocket.
2.  **Analytics Engine**: Real-time correlation, UEBA ML scoring, and alert generation.
3.  **Distributed State**: Synchronization of identities, vaults, and configuration across multiple instances via the Raft (`:8443`) cluster.

### 1. Single Node Deployment

The most straightforward way to run Sovereign Terminal is a single Docker container via `docker-compose.yml`:

```bash
docker-compose up -d
```

**Requirements:**
*   Ensure the host machine maps the volumes `sovereign_data` and `sovereign_logs`.
*   The SQLite/DuckDB WAL format relies on disk synchronization. SSDs or NVMe drives are highly recommended for the data mount.

### 2. High Availability Cluster (Experimental)

Sovereign Terminal supports Hashicorp Raft consensus for state synchronization (identities, RBAC, vault). The embedded database (DuckDB) handles SIEM data locally per node, while critical application configurations are synchronized via Raft across the network.

**Bootstrapping the Cluster:**
1. Start the first node. It will elect itself leader.
2. Start additional nodes configuration pointing to the origin `--raft-join=node1:8443`.
3. Nodes should maintain a quorum (3 or 5 nodes recommended to tolerate n-1/2 failures).

### 3. API Security & Access Management

Because the API routes sensitive SIEM events and enables plugin execution, it must be secured in enterprise environments.

**TLS / SSL Configuration**
By default, the Sovereign API listens on HTTP `:8080`. For production deployments:
*   Place the container behind a reverse proxy (e.g., Nginx, Envoy, or Traefik) that handles TLS termination.
*   Configure the reverse proxy to restrict access to trusted internal IP ranges if the UI relies on WebSockets for live streaming.

**Bearer Token Authentication**
All API endpoints (except documentation) require Bearer Authentication.
*   Generate API tokens via the desktop root interface or the CLI (`oblivra-cli`).
*   Inject long-lived tokens securely into third-party ingestion scripts via environment variables.
*   Tokens are RBAC-aligned and should be generated with minimal privileges via Sovereign's Vault system.

### 4. Plugin Orchestration

Plugins executed via the `Sandbox` interface (Lua or WebAssembly) possess access to internal host events and API abstractions.
1. Mount enterprise-approved plugins into the `/root/.oblivrashell/plugins` directory.
2. Ensure manifest definitions `manifest.json` do not grant overly permissive capabilities (e.g., `os.execute` or excessive CPU thresholds).

**Example Security Review:**
```json
// Restrict an ingestion plugin's capabilities inside manifest.json
"permissions": [
    "siem.write",
    "events.publish"
],
```

### 5. Diagnostics & Troubleshooting

*   **Logs**: Audit and execution logs are tracked internally and flushed to the `sovereign_logs` volume.
*   **Trust Verification Endpoint**: For health checks and intrusion detection, query the `/api/v1/debug/trust` endpoint. It returns cryptographic hashes and anomalies of the running Go binary and current memory profile.
