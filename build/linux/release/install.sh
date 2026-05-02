#!/usr/bin/env bash
# OBLIVRA server install script (Linux, systemd-based distros).
# Run from inside the unpacked tarball as root.
#
# What it does:
#   1. Creates the `oblivra` system user/group (no shell, no home).
#   2. Installs binaries under /opt/oblivra (mode 0755).
#   3. Installs sigma rule pack under /opt/oblivra/sigma (mode 0644).
#   4. Creates /etc/oblivra/oblivra.env from the example (mode 0600).
#   5. Creates /var/lib/oblivra (mode 0700, owned by oblivra:oblivra).
#   6. Drops the systemd unit at /etc/systemd/system/oblivra.service.
#   7. Reloads systemd; does NOT enable/start so you can edit the env file first.
#
# Idempotent — re-running on an existing install upgrades the binaries
# and unit but leaves the env file and data dir untouched.
set -euo pipefail

if [[ $EUID -ne 0 ]]; then
  echo "install.sh: must run as root (try: sudo $0)" >&2
  exit 1
fi

INSTALL_DIR=/opt/oblivra
ETC_DIR=/etc/oblivra
DATA_DIR=/var/lib/oblivra
UNIT_PATH=/etc/systemd/system/oblivra.service
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "==> creating system user/group"
if ! id oblivra >/dev/null 2>&1; then
  useradd --system --no-create-home --shell /usr/sbin/nologin oblivra
fi

echo "==> installing binaries to $INSTALL_DIR"
install -d -m 0755 "$INSTALL_DIR"
install -m 0755 "$SCRIPT_DIR/oblivra-server"  "$INSTALL_DIR/oblivra-server"
install -m 0755 "$SCRIPT_DIR/oblivra-cli"     "$INSTALL_DIR/oblivra-cli"
install -m 0755 "$SCRIPT_DIR/oblivra-verify"  "$INSTALL_DIR/oblivra-verify"
install -m 0755 "$SCRIPT_DIR/oblivra-agent"   "$INSTALL_DIR/oblivra-agent"
install -m 0755 "$SCRIPT_DIR/oblivra-migrate" "$INSTALL_DIR/oblivra-migrate"
install -m 0755 "$SCRIPT_DIR/oblivra-smoke"   "$INSTALL_DIR/oblivra-smoke"

# Optional: drop convenience symlinks into /usr/local/bin so operators
# can run the CLI without an absolute path.
ln -sf "$INSTALL_DIR/oblivra-cli"    /usr/local/bin/oblivra-cli
ln -sf "$INSTALL_DIR/oblivra-verify" /usr/local/bin/oblivra-verify
ln -sf "$INSTALL_DIR/oblivra-smoke"  /usr/local/bin/oblivra-smoke

echo "==> installing sigma rule pack"
install -d -m 0755 "$INSTALL_DIR/sigma"
cp -r "$SCRIPT_DIR/sigma/." "$INSTALL_DIR/sigma/"
find "$INSTALL_DIR/sigma" -type f -exec chmod 0644 {} \;
find "$INSTALL_DIR/sigma" -type d -exec chmod 0755 {} \;

echo "==> creating data dir $DATA_DIR"
install -d -m 0700 -o oblivra -g oblivra "$DATA_DIR"

echo "==> creating config dir $ETC_DIR"
install -d -m 0750 -o root -g oblivra "$ETC_DIR"
if [[ ! -e "$ETC_DIR/oblivra.env" ]]; then
  install -m 0640 -o root -g oblivra "$SCRIPT_DIR/oblivra.env.example" "$ETC_DIR/oblivra.env"
  cat <<'WARN'

  ⚠  /etc/oblivra/oblivra.env was created from the example.
     EDIT IT before starting the service:
        - replace OBLIVRA_API_KEYS with a real bearer token
        - replace OBLIVRA_AUDIT_KEY with a 32-byte hex secret
          (openssl rand -hex 32)

WARN
else
  echo "    (existing oblivra.env preserved)"
fi

echo "==> installing systemd unit"
install -m 0644 "$SCRIPT_DIR/oblivra.service" "$UNIT_PATH"
systemctl daemon-reload

cat <<'NEXT'

✓ install complete.

Next steps:
  1. sudoedit /etc/oblivra/oblivra.env
  2. sudo systemctl enable --now oblivra.service
  3. sudo systemctl status oblivra.service
  4. curl -s http://127.0.0.1:8080/healthz

Behind a VPN: bind OBLIVRA_ADDR=0.0.0.0:8080 in oblivra.env so VPN
clients can reach it directly. Otherwise leave it at 127.0.0.1:8080
and front with Caddy / nginx for TLS termination.

Logs:   journalctl -u oblivra -f
Data:   /var/lib/oblivra
Config: /etc/oblivra/oblivra.env

NEXT
