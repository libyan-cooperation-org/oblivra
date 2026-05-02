# OBLIVRA — Linux server release

This is the server-only release tarball — designed for deployment on a
Linux box behind a VPN where the operator UI is reached over HTTP from
trusted clients.

## What's in the box

| File | Purpose |
|---|---|
| `oblivra-server`         | Headless server + embedded UI (the main binary) |
| `oblivra-cli`            | REST client (ping, search, alerts, audit, backup verify/diff/restore --dry-run) |
| `oblivra-verify`         | Standalone offline integrity verifier |
| `oblivra-agent`          | Log forwarder; install on every host that ships logs |
| `oblivra-migrate`        | Schema migration runner |
| `oblivra-smoke`          | 43-endpoint go-live gate |
| `sigma/`                 | Built-in detection rule pack (incl. counter-forensic) |
| `oblivra.service`        | systemd unit |
| `oblivra.env.example`    | Config template — copy to `/etc/oblivra/oblivra.env` |
| `Caddyfile.example`      | Optional reverse proxy config (skip if VPN terminates TLS) |
| `install.sh`             | One-shot installer (creates user, drops files, registers unit) |
| `uninstall.sh`           | Reverse of install.sh; `--purge` wipes data too |
| `LICENSE` / `README.md` / `SECURITY.md` | Project docs |

## 60-second install

```bash
sudo ./install.sh
sudoedit /etc/oblivra/oblivra.env       # set OBLIVRA_API_KEYS + OBLIVRA_AUDIT_KEY
sudo systemctl enable --now oblivra.service
curl -s http://127.0.0.1:8080/healthz
```

That's it. The web UI is at `http://<host>:8080/` once the service is up.

## Behind a VPN

Set `OBLIVRA_ADDR=0.0.0.0:8080` in `/etc/oblivra/oblivra.env` so VPN
clients can reach the server directly. You don't need Caddy / nginx —
the VPN is the perimeter and the API requires bearer-token auth on every
request anyway.

If you want browser TLS (some clients flag plain HTTP as "Not Secure"
even on a VPN), drop `Caddyfile.example` into `/etc/caddy/Caddyfile`,
keep `OBLIVRA_ADDR=127.0.0.1:8080`, and let Caddy front the service.

## Day-2 operations

| Task | Command |
|---|---|
| Service status     | `systemctl status oblivra` |
| Tail logs          | `journalctl -u oblivra -f` |
| Verify audit chain | `oblivra-cli audit verify` |
| Smoke test (CI)    | `oblivra-smoke --base http://127.0.0.1:8080 --token $OBLIVRA_TOKEN` |
| Backup snapshot    | `tar -C /var/lib -czf oblivra-$(date +%F).tar.gz oblivra/` |
| Verify a backup    | `oblivra-cli backup verify /path/to/extracted/oblivra` |
| Diff two backups   | `oblivra-cli backup diff snapA snapB` |
| Plan a restore     | `oblivra-cli backup restore --dry-run snapA /var/lib/oblivra` |

## Directory layout (after install)

```
/opt/oblivra/                Binaries + sigma rule pack
/etc/oblivra/oblivra.env     Config (mode 0640, owned root:oblivra)
/var/lib/oblivra/            Data dir (mode 0700, owned oblivra:oblivra)
  ├── audit.log              Merkle-chained audit journal
  ├── ingest.wal             Write-ahead log
  ├── siem_hot.badger/       BadgerDB hot store
  ├── bleve.idx/             Full-text index
  ├── warm/                  Parquet warm tier (WORM)
  └── oblivra.vault          Encrypted vault
/etc/systemd/system/oblivra.service
/usr/local/bin/oblivra-cli, oblivra-verify, oblivra-smoke (symlinks)
```

## Uninstall

```bash
sudo ./uninstall.sh           # removes binaries + unit, KEEPS data
sudo ./uninstall.sh --purge   # also wipes /var/lib/oblivra (DESTROYS audit chain)
```

Without `--purge`, you can reinstall and continue exactly where you left
off — audit chain integrity preserved end-to-end.
