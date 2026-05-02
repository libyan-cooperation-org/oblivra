#!/usr/bin/env bash
# OBLIVRA server uninstall script.
#
# Stops + disables the service, removes binaries and the systemd unit.
# DELIBERATELY preserves the data dir + env file so operators don't
# accidentally destroy audit-grade evidence. Pass --purge to wipe data
# + config + user/group as well.
#
# Honours the same env vars as install.sh so you can uninstall a
# custom-prefix install:
#   INSTALL_DIR=/opt/security/oblivra DATA_DIR=/srv/oblivra ./uninstall.sh
set -euo pipefail

if [[ $EUID -ne 0 ]]; then
  echo "uninstall.sh: must run as root" >&2
  exit 1
fi

INSTALL_DIR="${INSTALL_DIR:-/opt/oblivra}"
ETC_DIR="${ETC_DIR:-/etc/oblivra}"
DATA_DIR="${DATA_DIR:-/var/lib/oblivra}"
USER="${USER:-oblivra}"
GROUP="${GROUP:-oblivra}"
SERVICE_NAME="${SERVICE_NAME:-oblivra}"
UNIT_PATH="${UNIT_PATH:-/etc/systemd/system/${SERVICE_NAME}.service}"

PURGE=0
for a in "$@"; do
  [[ "$a" == "--purge" ]] && PURGE=1
done

echo "==> stopping service"
systemctl disable --now "$SERVICE_NAME.service" 2>/dev/null || true

echo "==> removing systemd unit"
rm -f "$UNIT_PATH"
systemctl daemon-reload

echo "==> removing binaries"
rm -rf "$INSTALL_DIR"
rm -f /usr/local/bin/oblivra-cli /usr/local/bin/oblivra-verify /usr/local/bin/oblivra-smoke

if [[ $PURGE -eq 1 ]]; then
  echo "==> --purge: wiping $DATA_DIR + $ETC_DIR + user/group $USER/$GROUP"
  rm -rf "$DATA_DIR" "$ETC_DIR"
  userdel "$USER" 2>/dev/null || true
  groupdel "$GROUP" 2>/dev/null || true
  echo "✓ purged."
else
  cat <<NOTE

✓ binaries removed.
  $DATA_DIR and $ETC_DIR/oblivra.env are PRESERVED.
  Pass --purge to wipe them too (DESTROYS the audit chain irrecoverably).

NOTE
fi
