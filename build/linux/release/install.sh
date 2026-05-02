#!/usr/bin/env bash
# OBLIVRA server install script (Linux, systemd-based distros).
# Run from inside the unpacked tarball as root.
#
# Splunk-UF-style customisation — every path is overridable by env
# var so an operator can install into /opt/security/oblivra,
# /apps/oblivra, or wherever their layout requires:
#
#   INSTALL_DIR=/opt/oblivra            # binaries + sigma rules
#   ETC_DIR=/etc/oblivra                # config (oblivra.env)
#   DATA_DIR=/var/lib/oblivra           # audit log, WAL, parquet, vault
#   USER=oblivra GROUP=oblivra          # service identity
#   SERVICE_NAME=oblivra                # systemd unit name + service path
#   UNIT_PATH=/etc/systemd/system/oblivra.service
#   ADD_SYMLINKS=1                      # 1 = add /usr/local/bin symlinks (default)
#
# Idempotent — re-running upgrades binaries + unit, preserves env file
# + data dir.
set -euo pipefail

if [[ $EUID -ne 0 ]]; then
  echo "install.sh: must run as root (try: sudo $0)" >&2
  exit 1
fi

INSTALL_DIR="${INSTALL_DIR:-/opt/oblivra}"
ETC_DIR="${ETC_DIR:-/etc/oblivra}"
DATA_DIR="${DATA_DIR:-/var/lib/oblivra}"
USER="${USER:-oblivra}"
GROUP="${GROUP:-oblivra}"
SERVICE_NAME="${SERVICE_NAME:-oblivra}"
UNIT_PATH="${UNIT_PATH:-/etc/systemd/system/${SERVICE_NAME}.service}"
ADD_SYMLINKS="${ADD_SYMLINKS:-1}"
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

echo "==> using:"
echo "    INSTALL_DIR=$INSTALL_DIR"
echo "    ETC_DIR=$ETC_DIR"
echo "    DATA_DIR=$DATA_DIR"
echo "    USER=$USER  GROUP=$GROUP"
echo "    UNIT_PATH=$UNIT_PATH"

echo "==> creating system user/group"
if ! getent group "$GROUP" >/dev/null 2>&1; then
  groupadd --system "$GROUP"
fi
if ! id "$USER" >/dev/null 2>&1; then
  useradd --system --gid "$GROUP" --no-create-home --shell /usr/sbin/nologin "$USER"
fi

echo "==> installing binaries to $INSTALL_DIR"
install -d -m 0755 "$INSTALL_DIR"
for bin in oblivra-server oblivra-cli oblivra-verify oblivra-agent oblivra-migrate oblivra-smoke; do
  if [[ -f "$SCRIPT_DIR/$bin" ]]; then
    install -m 0755 "$SCRIPT_DIR/$bin" "$INSTALL_DIR/$bin"
  fi
done

if [[ "$ADD_SYMLINKS" == "1" ]]; then
  echo "==> adding /usr/local/bin symlinks"
  ln -sf "$INSTALL_DIR/oblivra-cli"    /usr/local/bin/oblivra-cli
  ln -sf "$INSTALL_DIR/oblivra-verify" /usr/local/bin/oblivra-verify
  ln -sf "$INSTALL_DIR/oblivra-smoke"  /usr/local/bin/oblivra-smoke
fi

echo "==> installing sigma rule pack"
install -d -m 0755 "$INSTALL_DIR/sigma"
cp -r "$SCRIPT_DIR/sigma/." "$INSTALL_DIR/sigma/"
find "$INSTALL_DIR/sigma" -type f -exec chmod 0644 {} \;
find "$INSTALL_DIR/sigma" -type d -exec chmod 0755 {} \;

echo "==> creating data dir $DATA_DIR"
install -d -m 0700 -o "$USER" -g "$GROUP" "$DATA_DIR"

echo "==> creating config dir $ETC_DIR"
install -d -m 0750 -o root -g "$GROUP" "$ETC_DIR"
if [[ ! -e "$ETC_DIR/oblivra.env" ]]; then
  install -m 0640 -o root -g "$GROUP" "$SCRIPT_DIR/oblivra.env.example" "$ETC_DIR/oblivra.env"
  # Default OBLIVRA_DATA_DIR in the env to whatever we used above so a
  # custom DATA_DIR= just works without editing.
  sed -i "s#^OBLIVRA_DATA_DIR=.*#OBLIVRA_DATA_DIR=$DATA_DIR#" "$ETC_DIR/oblivra.env"
  cat <<WARN

  ⚠  $ETC_DIR/oblivra.env was created from the example.
     EDIT IT before starting the service:
        - replace OBLIVRA_API_KEYS with a real bearer token
        - replace OBLIVRA_AUDIT_KEY with a 32-byte hex secret
          (openssl rand -hex 32)

WARN
else
  echo "    (existing $ETC_DIR/oblivra.env preserved)"
fi

echo "==> installing systemd unit"
# Generate the unit on the fly so custom INSTALL_DIR / DATA_DIR /
# USER / GROUP / ETC_DIR all flow into the right place.
cat > "$UNIT_PATH" <<UNIT
[Unit]
Description=OBLIVRA — sovereign log-driven security platform
Documentation=https://github.com/libyan-cooperation-org/oblivra
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=$USER
Group=$GROUP
WorkingDirectory=$INSTALL_DIR
EnvironmentFile=$ETC_DIR/oblivra.env
ExecStart=$INSTALL_DIR/oblivra-server
ExecReload=/bin/kill -HUP \$MAINPID
Restart=always
RestartSec=5

LimitNOFILE=65536
ProtectSystem=strict
ProtectHome=true
PrivateTmp=true
PrivateDevices=true
NoNewPrivileges=true
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectControlGroups=true
RestrictNamespaces=true
RestrictRealtime=true
RestrictSUIDSGID=true
LockPersonality=true
MemoryDenyWriteExecute=true
SystemCallArchitectures=native
SystemCallFilter=@system-service
SystemCallFilter=~@privileged @resources

ReadWritePaths=$DATA_DIR

[Install]
WantedBy=multi-user.target
UNIT
chmod 0644 "$UNIT_PATH"
systemctl daemon-reload

cat <<NEXT

✓ install complete.

Next steps:
  1. sudoedit $ETC_DIR/oblivra.env
  2. sudo systemctl enable --now $SERVICE_NAME.service
  3. sudo systemctl status $SERVICE_NAME.service
  4. curl -s http://127.0.0.1:8080/healthz

Logs:   journalctl -u $SERVICE_NAME -f
Data:   $DATA_DIR
Config: $ETC_DIR/oblivra.env
Bins:   $INSTALL_DIR

To install elsewhere on a future host, set env vars before running:
  sudo INSTALL_DIR=/opt/security/oblivra DATA_DIR=/srv/oblivra ./install.sh

NEXT
