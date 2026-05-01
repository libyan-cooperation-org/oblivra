# API client snippets

Minimal, copy-pasteable clients for the OBLIVRA REST surface. The full
schema lives in [`docs/openapi.yaml`](../openapi.yaml).

## Authentication

Every example assumes:

- `OBLIVRA_ADDR` — base URL (e.g. `https://oblivra.internal`)
- `OBLIVRA_TOKEN` — bearer token

Set both before running the snippet.

---

## Bash

```bash
# Send one event
curl -fsS -X POST "$OBLIVRA_ADDR/api/v1/siem/ingest" \
  -H "Authorization: Bearer $OBLIVRA_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"hostId":"web-01","eventType":"failed_login","message":"Failed password for root"}'

# Search the last 50 events with severity error
curl -fsSG "$OBLIVRA_ADDR/api/v1/siem/search" \
  -H "Authorization: Bearer $OBLIVRA_TOKEN" \
  --data-urlencode 'q=severity:error' \
  --data-urlencode 'limit=50' | jq .

# Open a case
curl -fsS -X POST "$OBLIVRA_ADDR/api/v1/cases" \
  -H "Authorization: Bearer $OBLIVRA_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"title":"Auth burst on web-01","hostId":"web-01","fromUnix":1714521600,"toUnix":1714525200}'

# Verify the audit chain
curl -fsS "$OBLIVRA_ADDR/api/v1/audit/verify" \
  -H "Authorization: Bearer $OBLIVRA_TOKEN" | jq .
```

---

## PowerShell

```powershell
$base    = $env:OBLIVRA_ADDR
$headers = @{ Authorization = "Bearer $($env:OBLIVRA_TOKEN)" }

# Send one event
$body = @{
  hostId    = "web-01"
  eventType = "failed_login"
  message   = "Failed password for root"
} | ConvertTo-Json
Invoke-RestMethod -Method POST -Uri "$base/api/v1/siem/ingest" -Headers $headers -ContentType "application/json" -Body $body

# Search
Invoke-RestMethod -Method GET -Uri "$base/api/v1/siem/search?q=severity:error&limit=50" -Headers $headers

# Open a case
$case = @{
  title    = "Auth burst on web-01"
  hostId   = "web-01"
  fromUnix = 1714521600
  toUnix   = 1714525200
} | ConvertTo-Json
Invoke-RestMethod -Method POST -Uri "$base/api/v1/cases" -Headers $headers -ContentType "application/json" -Body $case
```

---

## Python (stdlib only — no dependencies)

```python
import json
import os
import urllib.request

BASE  = os.environ["OBLIVRA_ADDR"].rstrip("/")
TOKEN = os.environ["OBLIVRA_TOKEN"]

def call(method: str, path: str, payload=None):
    req = urllib.request.Request(
        BASE + path,
        method=method,
        headers={
            "Authorization": f"Bearer {TOKEN}",
            "Content-Type": "application/json",
        },
        data=json.dumps(payload).encode() if payload is not None else None,
    )
    with urllib.request.urlopen(req, timeout=30) as r:
        return json.loads(r.read())

# Send one event
print(call("POST", "/api/v1/siem/ingest", {
    "hostId": "web-01",
    "eventType": "failed_login",
    "message": "Failed password for root",
}))

# Search
print(call("GET", "/api/v1/siem/search?q=severity:error&limit=50"))

# Open a case
print(call("POST", "/api/v1/cases", {
    "title": "Auth burst on web-01",
    "hostId": "web-01",
    "fromUnix": 1714521600,
    "toUnix":   1714525200,
}))
```

---

## Go

```go
package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
)

func main() {
	base  := os.Getenv("OBLIVRA_ADDR")
	token := os.Getenv("OBLIVRA_TOKEN")

	body, _ := json.Marshal(map[string]any{
		"hostId":    "web-01",
		"eventType": "failed_login",
		"message":   "Failed password for root",
	})
	req, _ := http.NewRequest("POST", base+"/api/v1/siem/ingest", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	fmt.Println("status:", resp.Status)
}
```

---

## Splunk SPL — forwarding events to OBLIVRA

OBLIVRA accepts the canonical Splunk HEC envelope at
`POST /services/collector/event` (Phase 41 compatibility). Existing
`outputs.conf` only needs the URL changed:

```ini
# $SPLUNK_HOME/etc/system/local/outputs.conf
[httpout]
httpEventCollectorToken = <oblivra-agent-token>
uri = https://oblivra.internal/services/collector
```

Splunk's UF forwards bytes-for-bytes — OBLIVRA maps Splunk's
`Authorization: Splunk <token>` header to our standard bearer pipeline,
so no token rewriting is needed.

---

## OpenTelemetry Collector — `otlphttp/json` exporter

```yaml
exporters:
  otlphttp/oblivra:
    endpoint: https://oblivra.internal
    encoding: json
    headers:
      Authorization: "Bearer ${OBLIVRA_TOKEN}"

service:
  pipelines:
    logs:
      receivers: [filelog]
      exporters: [otlphttp/oblivra]
```

OBLIVRA accepts the JSON-encoded OTLP/HTTP shape at `POST /v1/logs`. The
`otlphttp/proto` (protobuf) variant is intentionally not supported —
configure the exporter with `encoding: json`.

---

## Webhook signature verification (Bash)

OBLIVRA signs every webhook body with HMAC-SHA256. Verify in your
receiver:

```bash
#!/usr/bin/env bash
# Reads the body from stdin, the signature from $X_OBLIVRA_SIGNATURE,
# and the shared secret from $WEBHOOK_SECRET.
set -euo pipefail
body=$(cat)
expected=$(printf '%s' "$body" | openssl dgst -sha256 -hmac "$WEBHOOK_SECRET" -binary | base64)
if [[ "$expected" != "$X_OBLIVRA_SIGNATURE" ]]; then
  echo "signature mismatch" >&2
  exit 1
fi
echo "verified"
```
