# API Reference

The OBLIVRA REST API exposes external capabilities for headless administration, SIEM querying, alerting, and external agent ingestion.

## Base URL
All API requests should be prefixed with the base URL configured for the `RESTServer` (default: `https://<OBLIVRA-HOST>:<PORT>`).

## Authentication & Security

The API is protected by several security layers enforced by `secureMiddleware` and `APIKeyMiddleware`.

*   **API Keys**: Required for authenticated endpoints. Pass in the `X-API-Key` header or `Authorization: Bearer <Key>`.
*   **Rate Limiting**: The API enforces a strict global rate limit of **20 requests per second** with a burst limit of 50. Exceeding this returns `429 Too Many Requests`.
*   **Request Size**: All `POST` request bodies (e.g., SIEM searches) are capped at **1MB** to prevent JSON decoding OOM Denial of Service.
*   **CORS**: Currently permissive (`Access-Control-Allow-Origin: *`) but enforces standard `OPTIONS` preflight handling.
*   **Headers**: Responses include `X-Content-Type-Options: nosniff`, `X-Frame-Options: DENY`, and `Strict-Transport-Security: max-age=31536000; includeSubDomains`.

---

## 1. System Endpoints

These endpoints provide liveness, readiness, health, and prometheus metrics for the platform.

### GET `/healthz`
Returns the current liveness health of the REST server.

**Response (200 OK):**
```json
{
  "status": "alive",
  "time": "2026-03-01T20:00:00Z"
}
```

### GET `/readyz`
Returns the readiness state of the platform.

**Response (200 OK):**
```json
{
  "status": "ready"
}
```

### GET `/metrics`
Returns system metrics (events per second, total processed, active alerts) formatted for Prometheus scraping.

**Response (200 OK) Text/Plain:**
```text
# HELP oblivra_ingest_eps Current events processed per second
# TYPE oblivra_ingest_eps gauge
oblivra_ingest_eps 450

# HELP oblivra_ingest_total Total events processed
# TYPE oblivra_ingest_total counter
oblivra_ingest_total 1560943

# HELP oblivra_active_alerts Current active security anomalies
# TYPE oblivra_active_alerts gauge
oblivra_active_alerts 3
```

### GET `/api/v1/ingest/status`
Returns the current health and performance metrics of the ingestion pipeline.

**Response (200 OK):**
```json
{
  "buffer_capacity": 50000,
  "buffer_usage": 150,
  "dropped_events": 0,
  "eps": 450,
  "total_processed": 1560943
}
```

---

## 2. SIEM & Alert Endpoints

These endpoints interact directly with the underlying `SIEMStore` (BadgerDB + Bleve).

### POST `/api/v1/siem/search`
Executes a search query against the SIEM database. Can also be called via `GET` using a `?q=` query parameter.

**Request Body (`application/json`):**
```json
{
  "query": "EventType:auth AND Action:failed",
  "filters": {
    "limit": 100
  }
}
```

**Response (200 OK):**
```json
{
  "count": 2,
  "events": [
    {
      "id": "event-1234",
      "timestamp": "2026-03-01T20:15:00Z",
      "data": { "user": "root", "ip": "192.168.1.100" }
    },
    ...
  ],
  "query": "EventType:auth AND Action:failed"
}
```

### GET `/api/v1/alerts`
Retrieves a list of currently active security alerts (queries `EventType:security_alert`).

**Response (200 OK):**
```json
{
  "active_alerts": 1,
  "alerts": [
    {
       "id": "alert-4321",
       "timestamp": "2026-03-01T19:50:00Z",
       "severity": "high",
       "message": "Multiple failed logins from single IP"
    }
  ]
}
```

---

## 3. Agent Endpoints

These endpoints are designed for the external Agent fleet to register, report, and ingest data.

### POST `/api/v1/agent/register`
Logs a new agent into the fleet.

**(Note: Body/Response formats are subject to internal `auth.go` definitions currently out of scope of this document).**

### POST `/api/v1/agent/ingest`
Allows an external authenticated agent to push raw event sets into the ingestion pipeline.

### GET `/api/v1/agent/fleet`
Returns metadata concerning the active fleet of agents managed by the platform.
