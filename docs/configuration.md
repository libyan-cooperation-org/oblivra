# Configuration Reference

OblivraShell uses a combination of environment variables, CLI flags, and dynamic platform paths to configure itself. This document outlines the available configuration surfaces for both the user CLI and the headless/agent binaries.

## 1. Environment Variables

The core platform relies on the host OS environment to resolve data directories and SSH configurations.

| Variable | Usage | Default Fallback |
| :--- | :--- | :--- |
| `USER` | Used by the SSH Client (`cmd/cli`, `internal/ssh`) as the default login username if none is specified. | Current OS user |
| `HOME` | Used by `discovery_service` to resolve `~/.ssh/` filepaths on Unix systems. | Current OS profile path |
| `SSH_AUTH_SOCK` | Used by the SSH authentication engine (`internal/ssh/auth.go`) to communicate with `ssh-agent`. | None |

### Platform-Specific Path Resolution
The app stores isolated data, logs, and configuration state dynamically depending on the OS environment.

**Windows:**
*   **Config Dir:** Resolves to `%APPDATA%\oblivrashell`
*   **Data/Vault Dir:** Resolves to `%LOCALAPPDATA%\oblivrashell\data`
*   **Log Dir:** Resolves to `%LOCALAPPDATA%\oblivrashell\logs`

**Linux / macOS:**
*   **Config Dir:** `$XDG_CONFIG_HOME/oblivrashell` (falls back to `~/.config/oblivrashell`)
*   **Data/Vault Dir:** `$XDG_DATA_HOME/oblivrashell` (falls back to `~/.local/share/oblivrashell`)
*   **Log/State Dir:** `$XDG_STATE_HOME/oblivrashell` (falls back to `~/.local/state/oblivrashell`)

---

## 2. Agent CLI Flags (`cmd/agent`)

The remote telemetry agent is configured entirely via CLI flags upon startup.

| Flag | Default | Description |
| :--- | :--- | :--- |
| `--server` | `localhost:8443` | OBLIVRA Server address (host:port) |
| `--data-dir`| OS-specific data dir | Local data directory for Write-Ahead Log (WAL) and caching |
| `--interval` | `30` | Collection interval in seconds |
| `--fim` | `false` | Enable File Integrity Monitoring module |
| `--syslog` | `true` | Enable Syslog forwarding module |
| `--metrics`| `true` | Enable system hardware metrics collection |
| `--eventlog`| `false` | Enable Windows Event Log collection |
| `--tls-cert`| `""` | Path to TLS client certificate (for mTLS environments) |
| `--tls-key` | `""` | Path to TLS client key (for mTLS environments) |
| `--tls-ca`  | `""` | Path to Custom CA certificate for server verification |
| `--version` | `false` | Print version and exit |

---

## 3. Sovereign Terminal CLI Flags (`cmd/cli`)

The user-facing CLI manages manual SSH connections, executions, and database lookups. 

### `connect`
Connect to an SSH host or an alias stored in the database.
*   `-u <user>`: SSH Username
*   `-p <port>`: SSH port (default: `22`)
*   `-i <file>`: Identity file (private key path)
*   `-J <host>`: Jump host specifier (`user@host:port`)

### `exec`
Execute a command in parallel across one or more hosts.
*   `--hosts`: Comma-separated list of host names or database IDs.
*   `--tag`: Execute on all database hosts matching this tag.
*   `--timeout`: Execution timeout in seconds (default: `30`).

### `list`
Query the internal SQLite database for resources.
*   `--tag`: Filter listed resources by tag.
*   `--format`: Output format, accepts `table`, `json`, or `csv` (default: `table`).
*   *(Positional Resource)*: `hosts`, `sessions`, or `tags`.

### `tunnel`
Establish and manage port forwarding tunnels.
*   `-L <spec>`: Local forward (`local_port:remote_host:remote_port`)
*   `-R <spec>`: Remote forward (`remote_port:local_host:local_port`)
*   `-D <port>`: Dynamic SOCKS proxy on `local_port`
