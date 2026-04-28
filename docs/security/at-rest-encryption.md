# OBLIVRA — At-Rest Encryption Deployment Guide

This document tells the operator how to encrypt every byte OBLIVRA writes to disk, including the parts the application itself does not encrypt natively.

## What OBLIVRA Encrypts Natively

| Data path | Encryption | Notes |
| --------- | ---------- | ----- |
| Vault (credentials, fleet secret, agent identity) | AES-256-GCM with Argon2 KDF | Unlocked via passphrase; locked on idle / restart |
| Settings table values flagged sensitive | AES-256-GCM | Per-row, key wrapped by vault |
| Agent → server transit | TLS 1.3 | Server cert via `internal/security/CertificateManager` |
| Server → agent control channel | TLS 1.3 + HMAC fleet auth | Per-payload signature |
| WAL on the agent (if FIM enabled) | XChaCha20-Poly1305 | Keys derived from agent identity |

## What OBLIVRA Does *Not* Encrypt Natively

| Data path | Why it's plaintext | Operator action |
| --------- | ------------------ | --------------- |
| Bleve full-text index (`<data-dir>/bleve/`) | Bleve has no native at-rest encryption layer | **Use full-disk encryption (below)** |
| BadgerDB hot-tier event store (`<data-dir>/badger/`) | Performance — Badger has encryption but adds 10-15% I/O cost per page | **Use full-disk encryption (below)** |
| Parquet warm-tier files (`<data-dir>/parquet/`) | Plaintext columnar files for analyst tooling compatibility | **Use full-disk encryption (below)** |
| Audit logs (`audit_logs` table in SQLite) | Tamper-evident chain, not confidentiality | Out of threat model unless operator turns on optional row-level encrypt |
| Recordings (`recordings/*.cast`) | Asciinema cast format — replay tooling needs plaintext | **Use full-disk encryption (below)** |
| Backups (`backups/*.tar.gz`) | Created by `disasterservice.ExportResilienceBundle` | **Use the passphrase parameter** when calling `ExportResilienceBundle(passphrase)` — that path *is* encrypted |

The audit's recommendation: **deploy OBLIVRA on an encrypted volume**. This is the operationally normal answer for SIEMs and brings every plaintext path above under one consistent encryption layer.

---

## Linux: dm-crypt / LUKS

### One-time setup

```sh
# 1. Create a 200 GB encrypted volume backed by a sparse file (or use a real
#    block device — recommended for production):
sudo dd if=/dev/zero of=/var/oblivra-secure.img bs=1G count=200
sudo cryptsetup luksFormat /var/oblivra-secure.img
# Use a strong passphrase OR --key-file with a HSM-backed key

# 2. Open the volume
sudo cryptsetup luksOpen /var/oblivra-secure.img oblivra-data

# 3. Create the filesystem
sudo mkfs.ext4 /dev/mapper/oblivra-data

# 4. Mount
sudo mkdir -p /var/lib/oblivra
sudo mount /dev/mapper/oblivra-data /var/lib/oblivra
sudo chown oblivra:oblivra /var/lib/oblivra
```

### Boot-time auto-unlock options

Pick one based on threat model:

- **Network-bound disk encryption (NBDE) via Tang/Clevis** — recommended for fleet. Disk only opens when a Tang server in the same VPC is reachable. Theft of the box is useless without the network.
- **TPM 2.0 binding** — disk only opens when sealed against the TPM's measured boot state. Tampering with bootloader or kernel breaks unsealing.
- **Manual passphrase** — most secure, requires operator presence at every reboot. Use only for offline / air-gapped deployments.

### Add to OBLIVRA service

```ini
# /etc/systemd/system/oblivra.service
[Unit]
After=oblivra-secure.mount
Requires=oblivra-secure.mount

[Service]
Environment="OBLIVRA_DATA_DIR=/var/lib/oblivra"
ExecStart=/usr/local/bin/oblivra-server
```

---

## Windows: BitLocker

### One-time setup (PowerShell, Admin)

```powershell
# 1. Choose / create the volume that will host C:\ProgramData\oblivra
#    Recommended: a dedicated NTFS volume (D: or similar)

# 2. Enable BitLocker with a TPM-backed key
Enable-BitLocker -MountPoint "D:" `
  -EncryptionMethod XtsAes256 `
  -UsedSpaceOnly `
  -TpmProtector

# 3. Add a recovery key (keep this in your secrets manager / safe)
Add-BitLockerKeyProtector -MountPoint "D:" -RecoveryPasswordProtector

# 4. Move OBLIVRA data dir to D:
Stop-Service oblivra
robocopy C:\ProgramData\oblivra D:\oblivra /MIR
# Update OBLIVRA_DATA_DIR registry / environment
```

### Verification

```powershell
Get-BitLockerVolume -MountPoint "D:"
# VolumeStatus should be "FullyEncrypted"
# ProtectionStatus should be "On"
```

---

## macOS: FileVault + Encrypted APFS Volume

```sh
# 1. Enable FileVault for the boot volume (System Settings → Privacy & Security)

# 2. Create a dedicated encrypted APFS volume for OBLIVRA data:
diskutil apfs addVolume disk1 APFS oblivra-data -reserve 200g
diskutil apfs encryptVolume /Volumes/oblivra-data -user disk -passphrase "$(security find-generic-password ...)"

# 3. Symlink (or update OBLIVRA_DATA_DIR):
ln -sf /Volumes/oblivra-data /var/lib/oblivra
```

---

## Containers / Kubernetes

If deploying via Helm:

```yaml
# values.yaml — encrypted PVC
persistence:
  enabled: true
  storageClass: encrypted-ssd     # storage class with at-rest encryption
  size: 200Gi
  annotations:
    volume.beta.kubernetes.io/storage-class: encrypted-ssd

# StorageClass example (AWS):
parameters:
  type: gp3
  encrypted: "true"
  kmsKeyId: arn:aws:kms:...
```

The KMS key MUST be customer-managed, not AWS-managed, to satisfy SOC 2 § 6.5.

---

## Verification Checklist

After deployment:

- [ ] `oblivra-server` runs from a path on the encrypted volume
- [ ] `du -sh <data-dir>` returns reasonable numbers (volume mounted, not empty)
- [ ] On a clean reboot WITHOUT the unlock key/passphrase, `oblivra-server.service` fails to start (the data dir is unreachable)
- [ ] Backups (`disasterservice.ExportResilienceBundle`) ALSO use a strong passphrase — full-disk encryption only protects in-place data, not exfiltrated files
- [ ] Recovery keys / Tang server / TPM are documented in your runbook with operator escalation contacts

---

## Why Not Native Bleve Encryption?

The audit specifically asks about Bleve at-rest encryption. Three reasons we recommend FDE over a custom Bleve wrapper:

1. **Bleve has no encryption hook.** The kvstore interface accepts a custom backend, but every existing encrypted-kvstore implementation has known integrity gaps (no AEAD on the value space).
2. **A wrapper layer breaks Bleve internals.** Bleve mmap's segment files for query speed; a userspace cipher layer costs 30-50% query throughput.
3. **FDE is the operationally normal answer.** Every SIEM in this category (Splunk, Elastic, Wazuh) recommends FDE rather than per-store encryption. Auditors accept it as compliance-equivalent.

A future Stage-N improvement: ship an optional `OBLIVRA_BLEVE_ENCRYPT=true` mode that wraps Bleve in a userspace AES-GCM layer, accepting the throughput cost for operators who can't deploy FDE. Tracked as a roadmap item, not a current shipping capability.

---

**Last reviewed:** 2026-04-27 · For questions, contact `security@oblivra.io`
