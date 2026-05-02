#!/usr/bin/env bash
# OBLIVRA server uninstall script.
#
# Stops + disables the service, removes binaries and the systemd unit.
# DELIBERATELY preserves /var/lib/oblivra and /etc/oblivra/oblivra.env so
# operators don't accidentally destroy audit-grade evidence. Pass --purge
# to wipe data + config + user/group as well.
set -euo pipefail

if [[ $EUID -ne 0 ]]; then
  echo "uninstall.sh: must run as root" >&2
  exit 1
fi

PURGE=0
for a in "$@"; do
  [[ "$a" == "--purge" ]] && PURGE=1
done

echo "==> stopping service"
systemctl disable --now oblivra.service 2>/dev/null || true

echo "==> removing systemd unit"
rm -f /etc/systemd/system/oblivra.service
systemctl daemon-reload

echo "==> removing binaries"
rm -rf /opt/oblivra
rm -f /usr/local/bin/oblivra-cli /usr/local/bin/oblivra-verify /usr/local/bin/oblivra-smoke

if [[ $PURGE -eq 1 ]]; then
  echo "==> --purge: wiping data dir + config + user"
  rm -rf /var/lib/oblivra /etc/oblivra
  userdel oblivra 2>/dev/null || true
  groupdel oblivra 2>/dev/null || true
  echo "✓ purged."
else
  cat <<'NOTE'

✓ binaries removed.
  /var/lib/oblivra and /etc/oblivra/oblivra.env are PRESERVED.
  Pass --purge to wipe them too (DESTROYS the audit chain irrecoverably).

NOTE
fi
