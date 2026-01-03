# Validator Key Backup Procedures

## Overview

Secure encrypted backup of validator keys to Cloudflare R2 with GPG AES-256 encryption.

## Files Backed Up

**Critical (encrypted)**:
- `priv_validator_key.json` - Validator signing key
- `node_key.json` - Node P2P identity key
- `keyring-*/` - Account keys (if using file/test backend)

**Config (encrypted)**:
- `config.toml`, `app.toml`, `client.toml`
- `genesis.json`

## Quick Start

```bash
cd ~/blockchain-projects/paw/scripts/backup

# 1. Setup passphrase (one-time)
./setup-backup-passphrase.sh

# 2. Run manual backup
./validator-backup.sh aura aura-testnet-artifacts
./validator-backup.sh paw paw-testnet-artifacts

# 3. Install daily cron (optional)
./validator-backup-cron.sh --install
```

## Scripts

| Script | Purpose |
|--------|---------|
| `setup-backup-passphrase.sh` | Create/manage encryption passphrase |
| `validator-backup.sh` | Manual backup to R2 |
| `validator-restore.sh` | Restore from R2 backup |
| `validator-backup-cron.sh` | Automated daily backups |

## Passphrase Security

**Storage Locations** (use multiple):
1. Hardware password manager (1Password, Bitwarden)
2. Encrypted offline USB
3. Paper copy in safe/deposit box

**Fingerprint**: First 16 chars of SHA256 hash - use to verify correct passphrase without exposing it.

```bash
# Show fingerprint
./setup-backup-passphrase.sh --show-fingerprint

# Verify passphrase
./setup-backup-passphrase.sh --verify
```

## Restore Procedure

```bash
# List available backups
./validator-restore.sh --list aura-testnet-artifacts

# Restore (interactive)
./validator-restore.sh aura-testnet-artifacts validator-keys-aura-20240101-120000.tar.gz.gpg

# Dry run (preview only)
DRY_RUN=1 ./validator-restore.sh aura-testnet-artifacts <backup-file>
```

## R2 Bucket Structure

```
{chain}-testnet-artifacts/
  backups/
    validator-keys/
      validator-keys-{chain}-{YYYYMMDD-HHMMSS}.tar.gz.gpg
      {YYYY}/{MM}/
        validator-keys-{chain}-{YYYYMMDD-HHMMSS}.tar.gz.gpg
```

## Retention Policy

- **R2**: 30 days (configurable in cron script)
- **Local**: 7 days, stored in `~/.validator-backups/{chain}/`

## Monitoring

Logs: `~/.validator-backups/logs/backup.log`

Optional Slack notifications via `SLACK_WEBHOOK_URL` env var.

## Emergency Recovery

If passphrase is lost, backups are unrecoverable. Always maintain multiple copies of:
1. The passphrase itself
2. The passphrase fingerprint (for verification)
