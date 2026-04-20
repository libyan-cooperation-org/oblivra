# API Reference

Oblivra exposes a REST API for headless deployments, external integrations, and log ingestion from third-party tools.

Base URL: `http://localhost:8080/api/v1`

All endpoints require a Bearer token unless noted.

---

## Authentication

```http
Authorization: Bearer <api-token>
```

Generate tokens in the desktop app:
**Settings → API Tokens → + New Token**

Tokens are RBAC-scoped. Minimum required scopes are noted per endpoint.

---

## Events / Ingestion

### `POST /siem/ingest`

Submit a single event to the ingestion pipeline.

**Scope required:** `siem.write`

**Request:**

```json
{
  "host_id": "web-01",
  "event_type": "failed_login",
  "source_ip": "10.0.0.1",
  "user": "root",
  "raw_log": "Failed password for root from 10.0.0.1 port 22 ssh2",
  "timestamp": "2026-03-16T14:00:00Z"
}
```

**Response `200`:**

```json
{ "accepted": true, "pipeline_depth": 42 }
```

**Batch ingestion:**

```http
POST /siem/ingest/batch
```

```json
{
  "events": [
    { "host_id": "web-01", "event_type": "syslog", "raw_log": "..." },
    { "host_id": "web-02", "event_type": "syslog", "raw_log": "..." }
  ]
}
```

Max 1,000 events per batch. Use the syslog listener (UDP/TCP 1514) for higher-throughput streaming.

---

## Search

### `POST /siem/search`

Execute a log search query.

**Scope required:** `siem.read`

**Request:**

```json
{
  "query": "{host=\"web-01\"} |= \"error\"",
  "mode": "logql",
  "limit": 100,
  "offset": 0
}
```

**Modes:** `logql` | `lucene` | `sql`

**Response `200`:**

```json
{
  "results": [
    {
      "id": "evt-abc123",
      "host_id": "web-01",
      "event_type": "syslog",
      "raw_log": "2026-03-16T14:00:00Z web-01 nginx: error ...",
      "source_ip": "10.0.0.1",
      "timestamp": "2026-03-16T14:00:00Z"
    }
  ],
  "total": 1,
  "elapsed_ms": 12
}
```

---

## Hosts

### `GET /hosts`

List all configured hosts.

**Scope required:** `hosts.read`

**Response `200`:**

```json
{
  "hosts": [
    {
      "id": "host-abc123",
      "label": "Production Web",
      "hostname": "10.0.0.1",
      "port": 22,
      "username": "ubuntu",
      "category": "Production",
      "tags": ["linux", "nginx"],
      "is_favorite": false,
      "connection_count": 42,
      "last_connected": "2026-03-16T14:00:00Z"
    }
  ]
}
```

### `POST /hosts`

Add a new host.

**Scope required:** `hosts.write`

```json
{
  "label": "DB Server",
  "hostname": "10.0.0.5",
  "port": 22,
  "username": "postgres",
  "category": "Database",
  "tags": ["linux", "postgres"]
}
```

### `DELETE /hosts/{id}`

Remove a host by ID.

---

## Alerts

### `GET /alerts`

Retrieve alert history.

**Scope required:** `alerts.read`

**Query parameters:**

| Param | Default | Description |
|---|---|---|
| `limit` | `100` | Max results |
| `severity` | all | Filter: `critical\|high\|medium\|low` |
| `since` | 24h ago | ISO 8601 timestamp |

**Response `200`:**

```json
{
  "alerts": [
    {
      "id": "alert-xyz",
      "rule_id": "windows_shadow_copy_deletion",
      "rule_name": "Shadow Copy Deletion",
      "severity": "critical",
      "entity": "10.0.0.5",
      "triggered_at": "2026-03-16T14:32:00Z",
      "mitre_tactics": ["TA0040"],
      "mitre_techniques": ["T1490"],
      "notified": true
    }
  ],
  "total": 1
}
```

### `GET /alerts/rules`

List all active detection rules.

**Response `200`:**

```json
{
  "rules": [
    {
      "id": "builtin-4",
      "name": "Failed SSH Login",
      "severity": "medium",
      "type": "threshold",
      "threshold": 1,
      "window_sec": 60,
      "mitre_tactics": ["TA0001"],
      "mitre_techniques": ["T1110"]
    }
  ],
  "count": 82
}
```

### `POST /alerts/rules/reload`

Trigger a manual Sigma rule reload.

**Scope required:** `alerts.write`

**Response `200`:**

```json
{ "rules_loaded": 2543 }
```

---

## Compliance

### `GET /compliance/packs`

List available compliance packs.

**Response `200`:**

```json
{
  "packs": [
    { "id": "pci-dss", "name": "PCI-DSS v4.0", "version": "4.0", "control_count": 254 },
    { "id": "nist-800-53", "name": "NIST 800-53 Rev 5", "version": "5", "control_count": 1000 },
    { "id": "iso-27001", "name": "ISO 27001:2022", "version": "2022", "control_count": 114 }
  ]
}
```

### `GET /compliance/packs/{id}/evaluate`

Evaluate a compliance pack and return pass/fail results.

**Response `200`:**

```json
{
  "pack_id": "pci-dss",
  "pass_rate": 0.87,
  "passed": 221,
  "failed": 33,
  "controls": [
    { "id": "pci-1.1", "name": "Network segmentation", "status": "pass" },
    { "id": "pci-8.3", "name": "MFA for all access", "status": "fail", "reason": "No MFA users found" }
  ]
}
```

---

## System

### `GET /health`

System health check. No authentication required — safe for load balancer probes.

**Response `200`:**

```json
{
  "status": "ok",
  "vault": "unlocked",
  "goroutines": 48,
  "heap_mb": 124.3,
  "uptime": "2h15m"
}
```

### `GET /debug/metrics`

Prometheus metrics in text exposition format.

```
# HELP oblivra_goroutines Current goroutine count
# TYPE oblivra_goroutines gauge
oblivra_goroutines 48
oblivra_heap_bytes 130285568
oblivra_gc_count 12
```

### `GET /debug/pprof/`

Go pprof profiling endpoints. Localhost only.

Available profiles: `/profile`, `/heap`, `/goroutine`, `/trace`, `/cmdline`

```bash
go tool pprof http://localhost:6060/debug/pprof/heap
```

---

## Error Responses

All errors return standard JSON:

```json
{
  "error": "vault is locked",
  "code": "VAULT_LOCKED",
  "http_status": 503
}
```

| Code | HTTP | Meaning |
|---|---|---|
| `UNAUTHORIZED` | 401 | Missing or invalid Bearer token |
| `FORBIDDEN` | 403 | Token lacks required scope |
| `VAULT_LOCKED` | 503 | Vault not yet unlocked |
| `PIPELINE_FULL` | 429 | Ingestion buffer full — back off |
| `NOT_FOUND` | 404 | Resource does not exist |
| `INTERNAL_ERROR` | 500 | Unexpected server error |

---

## Rate Limiting

The API applies tiered rate limiting to protect platform stability:

- Global limit: **12,000 req/min** (200 req/sec)
- Per-IP limit: **300 req/min** (5 req/sec burst)
- Search endpoint: **100 req/min** (queries are expensive)
- Ingest endpoint: **50,000 events/min** (use syslog UDP for higher throughput)

Rate limit headers:

```
X-RateLimit-Limit: 1000
X-RateLimit-Remaining: 997
X-RateLimit-Reset: 1710590460
```

---

## Syslog Ingestion (UDP/TCP)

For high-throughput log forwarding, bypass the HTTP API and write directly to the syslog listener:

**UDP (fire-and-forget, max throughput):**

```
<your-server-ip>:1514 UDP
```

**TCP (reliable delivery):**

```
<your-server-ip>:1514 TCP
```

Supported formats are auto-detected: RFC5424, RFC3164, CEF, JSON, plain text.

Example rsyslog config:

```
# /etc/rsyslog.d/oblivra.conf
*.* @<oblivra-ip>:1514   # UDP
*.* @@<oblivra-ip>:1514  # TCP
```

Example Fluentd config:

```yaml
<match **>
  @type syslog
  host <oblivra-ip>
  port 1514
  transport udp
</match>
```
