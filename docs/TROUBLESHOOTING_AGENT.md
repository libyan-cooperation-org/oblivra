# OBLIVRA Agent Troubleshooting Guide

Issue: Agent runs but nothing appears in the frontend app

---

## Quick Diagnostic Checklist

1. **Verify agent process is running**
   - **Linux/macOS**: `ps aux | grep oblivra-agent`
   - **Windows**: Task Manager or `Get-Process oblivra-agent`

2. **Check agent logs for errors**
   - **Linux**: `/var/lib/oblivra/agent/agent.log`
   - **Windows**: `C:\ProgramData\oblivra\agent\agent.log`
   - **Custom Path**: `./agent-debug.log` (if specified)

3. **Verify server is listening on port 8443**
   - **Linux/macOS**: `ss -tlnp | grep 8443`
   - **Windows**: `netstat -an | findstr 8443`

4. **Test basic connectivity to server**
   ```bash
   curl -v --cacert ./certs/ca.crt https://localhost:8443/health
   ```

---

## Most Likely Causes & Fixes

### 1. mTLS Certificate Issues (Most Common)
The agent requires mutual TLS by default. Missing or invalid certs will block all communication.
- **Fix**: Provide valid CA, client cert, and key:
  ```bash
  ./oblivra-agent -server=localhost:8443 \
    -tls-ca=./certs/ca.crt \
    -tls-cert=./certs/agent.crt \
    -tls-key=./certs/agent.key \
    -log-json=true
  ```

### 2. WAL Buffering Delay
Events are buffered locally before sending (default: up to 500,000 events). This can delay visibility.
- **Fix for testing**: Force smaller, faster batches:
  ```bash
  ./oblivra-agent -server=localhost:8443 \
    -max-wal-events=100 -max-batch=10 -interval=5
  ```

### 3. Collectors Not Enabled
By default, only syslog and metrics are enabled. FIM/EventLog are OFF.
- **Fix**: Enable at least one collector to generate data:
  ```bash
  ./oblivra-agent -server=localhost:8443 \
    -syslog=true -metrics=true -fim=true
  ```

### 4. Frontend Not Receiving Data
The UI polls `/api/v1/agent/fleet` and uses WebSockets for real-time events.
- **Check browser DevTools (F12) -> Network tab**:
  - Look for `GET /api/v1/agent/fleet` (should return 200 with agent list)
  - Look for WS connection (should show 101 Switching Protocols)
  - If 401/403: Re-authenticate in the UI or check RBAC permissions

### 5. Server Address Mismatch
Default is `localhost:8443`. If your server is remote, specify it:
```bash
./oblivra-agent -server=192.168.1.100:8443 ...
```

---

## Debug Mode: Full Visibility

Run with maximum logging:
```bash
./oblivra-agent \
  -server=localhost:8443 \
  -tls-ca=./certs/ca.crt \
  -tls-cert=./certs/agent.crt \
  -tls-key=./certs/agent.key \
  -log-json=true \
  -log-path=./agent-debug.log \
  -interval=5 \
  -max-batch=10 \
  -max-wal-events=100 \
  -syslog=true \
  -metrics=true \
  -fim=true
```

**Monitor in real-time**:
- **Linux/macOS**: `tail -f ./agent-debug.log`
- **Windows**: `Get-Content .\agent-debug.log -Wait`

### Success Indicators to Look For:
- `"Connected to server: https://localhost:8443"`
- `"batch sent: X events to /api/v1/agent/ingest"`
- `"[config] Applied fleet config: interval=5s collectors=3"`

### Error Patterns to Watch For:
- `"dial tcp ... connect: connection refused"` -> Server not running or wrong IP/port
- `"TLS handshake failed"` or `"certificate signed by unknown authority"` -> Cert mismatch
- `"server returned 401"` -> Authentication/token issue

---

## Frontend-Specific Checks

- **Hard refresh the UI**: Ctrl+Shift+R (or Cmd+Shift+R on Mac)
- **Clear Wails/frontend cache** if applicable: `~/.oblivra/frontend-cache/`
- **Ensure you are logged in** with a role that has `agent:view` or `siem:read` permissions
- **Verify the frontend is actually making API calls** (check Network tab in DevTools)

---

## Next Steps

1. Share sanitized agent logs (remove IPs, certs, tokens) for deeper analysis
2. Verify server-side ingestion logs at `/var/log/oblivra/server.log`
3. Contact repository maintainer (KingKnull/Sanad) with:
   - OS & architecture
   - Exact agent launch command
   - Server version & deployment method
   - Sanitized debug logs

---

> [!CAUTION]
> **Security Note**: Never commit real TLS keys, certificates, or production server addresses to version control or public channels. Use environment variables or a secrets manager for sensitive configuration.
