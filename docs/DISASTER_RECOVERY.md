# PAW Blockchain Disaster Recovery Procedures

**Version:** 1.0
**Last Updated:** 2025-12-06
**Criticality:** CRITICAL - Review quarterly, test annually

---

## Table of Contents

1. [Overview](#overview)
2. [Backup Strategy](#backup-strategy)
3. [Node Recovery Scenarios](#node-recovery-scenarios)
4. [State Sync Recovery](#state-sync-recovery)
5. [Data Restore Procedures](#data-restore-procedures)
6. [Testing Recovery](#testing-recovery)
7. [Emergency Contacts and Escalation](#emergency-contacts-and-escalation)

---

## Overview

This document provides comprehensive disaster recovery procedures for PAW blockchain nodes. It covers backup strategies, recovery scenarios, and step-by-step restoration procedures.

**Critical principle:** Validator private keys cannot be recovered once lost. Prevention through secure backup is the ONLY protection.

**Recovery Time Objectives (RTO):**
- Non-validator node: < 4 hours
- Validator node: < 1 hour (to prevent slashing)

**Recovery Point Objectives (RPO):**
- Maximum acceptable data loss: Latest snapshot (24 hours max)
- State sync can recover to latest consensus state

---

## Backup Strategy

### 1. Daily Automated Snapshots

#### Local Snapshot Creation

Create daily snapshots of the node data directory:

```bash
#!/bin/bash
# File: /usr/local/bin/paw-backup.sh
# Schedule via cron: 0 2 * * * /usr/local/bin/paw-backup.sh

set -euo pipefail

# Configuration
PAW_HOME="${PAW_HOME:-$HOME/.paw}"
BACKUP_DIR="/var/backups/paw"
RETENTION_DAYS=7
DATE=$(date +%Y%m%d-%H%M%S)
BACKUP_NAME="paw-backup-${DATE}"

# Create backup directory
mkdir -p "${BACKUP_DIR}"

echo "[$(date)] Starting PAW blockchain backup..."

# Stop node for consistent snapshot (optional - comment out for running backup)
# systemctl stop pawd

# Create compressed archive
# Exclude tx_index.db and blockstore.db as they can be rebuilt
tar -czf "${BACKUP_DIR}/${BACKUP_NAME}.tar.gz" \
    -C "${PAW_HOME}" \
    --exclude='data/tx_index.db' \
    --exclude='data/blockstore.db' \
    --exclude='data/cs.wal' \
    --exclude='data/evidence.db' \
    config/ \
    data/application.db \
    data/state.db \
    data/snapshots/ \
    data/priv_validator_state.json

# Restart node if stopped
# systemctl start pawd

# Calculate checksum
sha256sum "${BACKUP_DIR}/${BACKUP_NAME}.tar.gz" > "${BACKUP_DIR}/${BACKUP_NAME}.tar.gz.sha256"

echo "[$(date)] Backup created: ${BACKUP_NAME}.tar.gz"

# Remove old backups (local retention)
find "${BACKUP_DIR}" -name "paw-backup-*.tar.gz*" -mtime +${RETENTION_DAYS} -delete

echo "[$(date)] Backup completed successfully"
```

Make the script executable:
```bash
chmod +x /usr/local/bin/paw-backup.sh
```

Add to crontab (runs daily at 2 AM):
```bash
crontab -e
# Add:
0 2 * * * /usr/local/bin/paw-backup.sh >> /var/log/paw-backup.log 2>&1
```

#### What Gets Backed Up

| Component | Backed Up | Reason |
|-----------|-----------|--------|
| `config/app.toml` | YES | Node configuration |
| `config/config.toml` | YES | CometBFT configuration |
| `config/genesis.json` | YES | Chain genesis state |
| `config/node_key.json` | YES | P2P identity |
| `config/priv_validator_key.json` | YES | **CRITICAL** - Validator signing key |
| `data/application.db` | YES | Application state |
| `data/state.db` | YES | Consensus state |
| `data/snapshots/` | YES | State sync snapshots |
| `data/priv_validator_state.json` | YES | Prevents double-signing |
| `data/tx_index.db` | NO | Can be rebuilt via replay |
| `data/blockstore.db` | NO | Can be recovered via state sync |
| `data/cs.wal` | NO | Write-ahead log, regenerated |
| `data/evidence.db` | NO | Evidence can be recovered |

### 2. External Backup to Cloud Storage

#### AWS S3 Backup

```bash
#!/bin/bash
# File: /usr/local/bin/paw-s3-backup.sh

set -euo pipefail

# Configuration
AWS_BUCKET="s3://paw-blockchain-backups"
AWS_REGION="us-east-1"
BACKUP_DIR="/var/backups/paw"
RETENTION_DAYS=30

# Find latest backup
LATEST_BACKUP=$(ls -t "${BACKUP_DIR}"/paw-backup-*.tar.gz | head -1)

if [[ -z "${LATEST_BACKUP}" ]]; then
    echo "ERROR: No backup file found in ${BACKUP_DIR}"
    exit 1
fi

echo "[$(date)] Uploading to S3: ${LATEST_BACKUP}"

# Upload with server-side encryption
aws s3 cp "${LATEST_BACKUP}" \
    "${AWS_BUCKET}/$(basename ${LATEST_BACKUP})" \
    --region "${AWS_REGION}" \
    --storage-class STANDARD_IA \
    --server-side-encryption AES256

# Upload checksum
aws s3 cp "${LATEST_BACKUP}.sha256" \
    "${AWS_BUCKET}/$(basename ${LATEST_BACKUP}).sha256" \
    --region "${AWS_REGION}"

echo "[$(date)] Upload completed"

# Cleanup old S3 backups
aws s3 ls "${AWS_BUCKET}/" | \
    grep "paw-backup-" | \
    awk '{print $4}' | \
    while read backup; do
        BACKUP_DATE=$(echo "$backup" | grep -oP '\d{8}')
        DAYS_OLD=$(( ($(date +%s) - $(date -d "$BACKUP_DATE" +%s)) / 86400 ))
        if [[ $DAYS_OLD -gt $RETENTION_DAYS ]]; then
            echo "Deleting old backup: $backup (${DAYS_OLD} days old)"
            aws s3 rm "${AWS_BUCKET}/${backup}"
            aws s3 rm "${AWS_BUCKET}/${backup}.sha256" 2>/dev/null || true
        fi
    done
```

Schedule after local backup:
```bash
# Crontab entry
0 3 * * * /usr/local/bin/paw-s3-backup.sh >> /var/log/paw-s3-backup.log 2>&1
```

#### Google Cloud Storage (GCS) Backup

```bash
#!/bin/bash
# File: /usr/local/bin/paw-gcs-backup.sh

set -euo pipefail

# Configuration
GCS_BUCKET="gs://paw-blockchain-backups"
BACKUP_DIR="/var/backups/paw"
RETENTION_DAYS=30

# Find latest backup
LATEST_BACKUP=$(ls -t "${BACKUP_DIR}"/paw-backup-*.tar.gz | head -1)

if [[ -z "${LATEST_BACKUP}" ]]; then
    echo "ERROR: No backup file found in ${BACKUP_DIR}"
    exit 1
fi

echo "[$(date)] Uploading to GCS: ${LATEST_BACKUP}"

# Upload with lifecycle management
gsutil -m cp "${LATEST_BACKUP}" \
    "${GCS_BUCKET}/$(basename ${LATEST_BACKUP})"

gsutil -m cp "${LATEST_BACKUP}.sha256" \
    "${GCS_BUCKET}/$(basename ${LATEST_BACKUP}).sha256"

echo "[$(date)] Upload completed"

# Cleanup old backups
gsutil ls "${GCS_BUCKET}/paw-backup-*.tar.gz" | \
    while read backup; do
        BACKUP_DATE=$(echo "$backup" | grep -oP '\d{8}')
        DAYS_OLD=$(( ($(date +%s) - $(date -d "$BACKUP_DATE" +%s)) / 86400 ))
        if [[ $DAYS_OLD -gt $RETENTION_DAYS ]]; then
            echo "Deleting old backup: $backup (${DAYS_OLD} days old)"
            gsutil rm "$backup"
            gsutil rm "${backup}.sha256" 2>/dev/null || true
        fi
    done
```

### 3. Validator Key Security

**CRITICAL: Validator keys are the most sensitive component.**

#### Secure Backup of Validator Key

```bash
#!/bin/bash
# File: /usr/local/bin/paw-validator-key-backup.sh
# RUN ONCE during initial setup, store securely offline

set -euo pipefail

PAW_HOME="${PAW_HOME:-$HOME/.paw}"
VALIDATOR_KEY="${PAW_HOME}/config/priv_validator_key.json"
BACKUP_DEST="/secure/offline/storage"  # CHANGE THIS to secure location

if [[ ! -f "${VALIDATOR_KEY}" ]]; then
    echo "ERROR: Validator key not found at ${VALIDATOR_KEY}"
    exit 1
fi

# Create encrypted backup
gpg --symmetric --cipher-algo AES256 \
    --output "${BACKUP_DEST}/priv_validator_key.json.gpg" \
    "${VALIDATOR_KEY}"

echo "Validator key backed up to: ${BACKUP_DEST}/priv_validator_key.json.gpg"
echo "CRITICAL: Store this file offline in multiple secure locations"
echo "CRITICAL: Store the encryption password separately and securely"
```

#### Validator Key Best Practices

1. **Multiple encrypted copies** in different physical locations
2. **Never** store unencrypted on cloud storage
3. **Never** commit to version control
4. **Test decryption** during recovery drills
5. **Document** who has access and where copies are stored

### 4. Backup Retention Policy

| Backup Type | Retention | Storage | Purpose |
|-------------|-----------|---------|---------|
| Local daily | 7 days | Local disk | Fast recovery |
| S3/GCS daily | 30 days | Cloud (IA) | Medium-term recovery |
| Monthly snapshot | 12 months | Cloud (Glacier/Archive) | Long-term compliance |
| Validator key | Indefinite | Offline encrypted | Critical key recovery |

---

## Node Recovery Scenarios

### Scenario 1: Complete Node Failure

**Symptoms:**
- Server hardware failure
- Complete data loss
- Node unreachable

**Recovery Steps:**

1. **Provision new hardware/VM**
   ```bash
   # Ensure system requirements:
   # - CPU: 4+ cores
   # - RAM: 16GB+
   # - Disk: 500GB+ SSD
   # - Network: 100Mbps+
   ```

2. **Install dependencies**
   ```bash
   sudo apt update
   sudo apt install -y build-essential git

   # Install Go 1.21+
   wget https://go.dev/dl/go1.21.6.linux-amd64.tar.gz
   sudo rm -rf /usr/local/go
   sudo tar -C /usr/local -xzf go1.21.6.linux-amd64.tar.gz
   export PATH=$PATH:/usr/local/go/bin
   ```

3. **Build pawd binary**
   ```bash
   git clone git@github.com:decristofaroj/paw.git
   cd paw
   go build -o pawd ./cmd/...
   sudo mv pawd /usr/local/bin/
   ```

4. **Restore from backup** (see [Data Restore Procedures](#data-restore-procedures))

5. **Verify configuration**
   ```bash
   pawd config chain-id
   pawd config node tcp://localhost:26657
   ```

6. **Start node**
   ```bash
   pawd start
   ```

### Scenario 2: Corrupted Database

**Symptoms:**
- Node crashes on startup
- Database errors in logs
- "panic: corrupted database" errors

**Recovery Options:**

#### Option A: Restore from Backup (Faster for recent backups)

```bash
# Stop node
sudo systemctl stop pawd

# Backup corrupted data (for analysis)
mv ~/.paw/data ~/.paw/data.corrupted.$(date +%Y%m%d)

# Restore from latest backup (see Data Restore Procedures)
# ... restoration steps ...

# Start node
sudo systemctl start pawd
```

#### Option B: State Sync Recovery (Faster for older backups)

```bash
# Stop node
sudo systemctl stop pawd

# Remove corrupted data (keep config)
rm -rf ~/.paw/data

# Initialize fresh data directory
pawd init $(cat ~/.paw/config/config.toml | grep moniker | cut -d'"' -f2) --recover

# Configure state sync (see State Sync Recovery section)
# ... state sync configuration ...

# Start node
sudo systemctl start pawd
```

#### Option C: Full Resync (Last resort, slowest)

```bash
# Stop node
sudo systemctl stop pawd

# Backup and remove data
mv ~/.paw/data ~/.paw/data.corrupted.$(date +%Y%m%d)
mkdir -p ~/.paw/data

# Copy genesis
cp ~/.paw/config/genesis.json ~/.paw/data/

# Start node (will sync from block 0)
sudo systemctl start pawd

# Monitor sync progress
pawd status | jq '.SyncInfo'
```

### Scenario 3: Lost Validator Key (PREVENTION ONLY)

**CRITICAL: Validator private keys CANNOT be recovered once lost.**

#### Prevention Measures

1. **Multiple encrypted backups** in different physical locations
2. **Hardware Security Module (HSM)** for production validators
3. **Tmkms (Tendermint KMS)** for remote signing
4. **Regular backup verification** (quarterly)

#### If Key is Lost

**There is NO recovery.** You must:

1. Accept loss of validator identity
2. Create new validator with new key
3. Migrate delegations (if possible)
4. May face slashing for downtime

**This is why backup prevention is CRITICAL.**

### Scenario 4: Network Partition / Stale Chain State

**Symptoms:**
- Node stuck at old block height
- No peer connections
- "height regression" errors

**Recovery Steps:**

1. **Check peer connectivity**
   ```bash
   pawd status | jq '.SyncInfo.latest_block_height'
   curl -s http://localhost:26657/net_info | jq '.result.n_peers'
   ```

2. **Update persistent peers**
   ```bash
   # Edit config.toml
   nano ~/.paw/config/config.toml

   # Update persistent_peers with known good nodes
   persistent_peers = "node1@ip1:26656,node2@ip2:26656"
   ```

3. **Reset node to safe height**
   ```bash
   # Find a recent safe height (e.g., 1000 blocks back)
   CURRENT_HEIGHT=$(pawd status | jq -r '.SyncInfo.latest_block_height')
   SAFE_HEIGHT=$((CURRENT_HEIGHT - 1000))

   # Rollback
   pawd rollback --hard --unsafe-rollback-height $SAFE_HEIGHT
   ```

4. **Restart and resync**
   ```bash
   sudo systemctl restart pawd

   # Monitor
   journalctl -u pawd -f
   ```

---

## State Sync Recovery

State sync allows a node to quickly sync to the current chain state without replaying all historical blocks.

### When to Use State Sync

- **Node far behind** (>10,000 blocks)
- **Fresh node setup**
- **After database corruption** (if backup is old)
- **Fast validator replacement**

### Configuring State Sync

#### Step 1: Find State Sync RPC Endpoints

```bash
# Public RPC endpoints (example - replace with actual PAW endpoints)
RPC1="https://rpc1.paw-testnet.io:443"
RPC2="https://rpc2.paw-testnet.io:443"
```

#### Step 2: Get Trust Height and Hash

```bash
# Query a recent block (at least 1000 blocks back for safety)
LATEST_HEIGHT=$(curl -s $RPC1/block | jq -r '.result.block.header.height')
TRUST_HEIGHT=$((LATEST_HEIGHT - 1000))
TRUST_HASH=$(curl -s "$RPC1/block?height=$TRUST_HEIGHT" | jq -r '.result.block_id.hash')

echo "Trust Height: $TRUST_HEIGHT"
echo "Trust Hash: $TRUST_HASH"
```

#### Step 3: Configure State Sync in config.toml

```bash
# Edit config.toml
nano ~/.paw/config/config.toml
```

Update the `[statesync]` section:

```toml
[statesync]
enable = true

# RPC servers to fetch snapshots from
rpc_servers = "https://rpc1.paw-testnet.io:443,https://rpc2.paw-testnet.io:443"

# Trust height and hash from Step 2
trust_height = 12345000  # Replace with actual TRUST_HEIGHT
trust_hash = "ABC123..."  # Replace with actual TRUST_HASH

# Trust period (should be less than unbonding period)
trust_period = "168h0m0s"  # 7 days

# Snapshot discovery
discovery_time = "15s"
chunk_request_timeout = "10s"
chunk_fetchers = "4"
```

#### Step 4: Reset Data and Start State Sync

```bash
# Stop node
sudo systemctl stop pawd

# IMPORTANT: Backup priv_validator_state.json to prevent double-signing
cp ~/.paw/data/priv_validator_state.json ~/priv_validator_state.json.backup

# Reset data directory (keeps config)
pawd tendermint unsafe-reset-all --keep-addr-book

# Restore priv_validator_state.json
cp ~/priv_validator_state.json.backup ~/.paw/data/priv_validator_state.json

# Start node (will begin state sync)
sudo systemctl start pawd

# Monitor progress
journalctl -u pawd -f
```

#### Step 5: Verify State Sync Completion

```bash
# Check sync status
pawd status | jq '.SyncInfo'

# Should show:
# - catching_up: false (once complete)
# - latest_block_height: close to current chain height

# Check peers
curl -s http://localhost:26657/net_info | jq '.result.n_peers'
```

### State Sync Troubleshooting

| Problem | Solution |
|---------|----------|
| "snapshot not found" | RPC servers don't have snapshots; try different RPCs |
| "chunk verification failed" | Trust hash/height incorrect; recalculate |
| "no connected peers" | Check firewall, persistent_peers configuration |
| "state sync stalled" | Increase chunk_fetchers, try different RPC servers |

---

## Data Restore Procedures

### Step-by-Step Restore from Backup

#### Prerequisites

- Latest backup file (`paw-backup-YYYYMMDD-HHMMSS.tar.gz`)
- Corresponding checksum file (`.sha256`)
- pawd binary installed
- Node stopped

#### Procedure

1. **Stop the node**
   ```bash
   sudo systemctl stop pawd
   # or if running manually:
   pkill -SIGTERM pawd
   ```

2. **Download backup from S3 (if needed)**
   ```bash
   # List available backups
   aws s3 ls s3://paw-blockchain-backups/ | grep paw-backup

   # Download latest (or specific backup)
   aws s3 cp s3://paw-blockchain-backups/paw-backup-20251206-020000.tar.gz \
       /tmp/paw-restore.tar.gz

   aws s3 cp s3://paw-blockchain-backups/paw-backup-20251206-020000.tar.gz.sha256 \
       /tmp/paw-restore.tar.gz.sha256
   ```

   Or from GCS:
   ```bash
   gsutil ls gs://paw-blockchain-backups/ | grep paw-backup

   gsutil cp gs://paw-blockchain-backups/paw-backup-20251206-020000.tar.gz \
       /tmp/paw-restore.tar.gz

   gsutil cp gs://paw-blockchain-backups/paw-backup-20251206-020000.tar.gz.sha256 \
       /tmp/paw-restore.tar.gz.sha256
   ```

3. **Verify backup integrity**
   ```bash
   cd /tmp
   sha256sum -c paw-restore.tar.gz.sha256

   # Should output:
   # paw-restore.tar.gz: OK
   ```

   If checksum fails:
   ```bash
   echo "ERROR: Backup file corrupted!"
   echo "Try downloading again or use a different backup"
   exit 1
   ```

4. **Backup current state (if salvageable)**
   ```bash
   PAW_HOME="${PAW_HOME:-$HOME/.paw}"

   # Create emergency backup of current state
   TIMESTAMP=$(date +%Y%m%d-%H%M%S)
   sudo mv "$PAW_HOME" "${PAW_HOME}.emergency-${TIMESTAMP}"

   echo "Current state backed up to: ${PAW_HOME}.emergency-${TIMESTAMP}"
   ```

5. **Extract backup**
   ```bash
   # Create fresh data directory
   mkdir -p "$PAW_HOME"

   # Extract backup
   tar -xzf /tmp/paw-restore.tar.gz -C "$PAW_HOME"

   echo "Backup extracted to: $PAW_HOME"
   ```

6. **Verify restored files**
   ```bash
   # Check critical files exist
   for file in \
       config/app.toml \
       config/config.toml \
       config/genesis.json \
       config/priv_validator_key.json \
       data/priv_validator_state.json; do

       if [[ ! -f "$PAW_HOME/$file" ]]; then
           echo "ERROR: Missing critical file: $file"
           exit 1
       else
           echo "OK: $file"
       fi
   done
   ```

7. **Set correct permissions**
   ```bash
   # Ensure correct ownership
   chown -R $(whoami):$(whoami) "$PAW_HOME"

   # Secure validator key
   chmod 600 "$PAW_HOME/config/priv_validator_key.json"
   chmod 600 "$PAW_HOME/data/priv_validator_state.json"
   ```

8. **Verify configuration**
   ```bash
   # Check chain ID matches
   CHAIN_ID=$(grep 'chain-id' "$PAW_HOME/config/client.toml" | cut -d'"' -f2)
   echo "Chain ID: $CHAIN_ID"

   # Verify genesis hash
   jq -S -c . "$PAW_HOME/config/genesis.json" | shasum -a 256
   # Compare with known genesis hash
   ```

9. **Start node**
   ```bash
   sudo systemctl start pawd

   # Monitor startup
   journalctl -u pawd -f
   ```

10. **Validate chain state after recovery**
    ```bash
    # Wait for node to start (may take 30-60 seconds)
    sleep 60

    # Check status
    pawd status | jq '.SyncInfo'

    # Verify block height is progressing
    HEIGHT1=$(pawd status | jq -r '.SyncInfo.latest_block_height')
    sleep 10
    HEIGHT2=$(pawd status | jq -r '.SyncInfo.latest_block_height')

    if [[ $HEIGHT2 -gt $HEIGHT1 ]]; then
        echo "SUCCESS: Node is syncing (height: $HEIGHT1 -> $HEIGHT2)"
    else
        echo "WARNING: Node may be stuck at height $HEIGHT1"
        echo "Check logs: journalctl -u pawd -n 100"
    fi
    ```

11. **Rejoin network (if validator)**
    ```bash
    # Check validator status
    pawd query staking validator $(pawd keys show validator --bech val -a)

    # If jailed, unjail after recovery
    pawd tx slashing unjail --from validator --chain-id paw-testnet-1
    ```

### Validating Restored Data

After restoration, verify:

```bash
#!/bin/bash
# Validation checklist

PAW_HOME="${PAW_HOME:-$HOME/.paw}"

echo "=== PAW Node Restoration Validation ==="
echo ""

# 1. Check process is running
if pgrep -x pawd > /dev/null; then
    echo "[OK] pawd process is running"
else
    echo "[FAIL] pawd process not running"
fi

# 2. Check RPC is responding
if curl -s http://localhost:26657/status > /dev/null; then
    echo "[OK] RPC endpoint responding"
else
    echo "[FAIL] RPC endpoint not responding"
fi

# 3. Check sync status
CATCHING_UP=$(pawd status 2>/dev/null | jq -r '.SyncInfo.catching_up')
if [[ "$CATCHING_UP" == "false" ]]; then
    echo "[OK] Node is fully synced"
elif [[ "$CATCHING_UP" == "true" ]]; then
    echo "[INFO] Node is catching up (expected after restore)"
else
    echo "[FAIL] Cannot determine sync status"
fi

# 4. Check block height progression
HEIGHT1=$(pawd status 2>/dev/null | jq -r '.SyncInfo.latest_block_height')
sleep 10
HEIGHT2=$(pawd status 2>/dev/null | jq -r '.SyncInfo.latest_block_height')

if [[ "$HEIGHT2" -gt "$HEIGHT1" ]]; then
    echo "[OK] Block height progressing ($HEIGHT1 -> $HEIGHT2)"
else
    echo "[FAIL] Block height not progressing (stuck at $HEIGHT1)"
fi

# 5. Check peer connections
NUM_PEERS=$(curl -s http://localhost:26657/net_info 2>/dev/null | jq -r '.result.n_peers')
if [[ "$NUM_PEERS" -gt 0 ]]; then
    echo "[OK] Connected to $NUM_PEERS peers"
else
    echo "[FAIL] No peer connections"
fi

# 6. Check validator signing (if validator)
if [[ -f "$PAW_HOME/config/priv_validator_key.json" ]]; then
    VAL_ADDR=$(jq -r '.address' "$PAW_HOME/config/priv_validator_key.json")
    if [[ -n "$VAL_ADDR" ]]; then
        echo "[OK] Validator key present (address: $VAL_ADDR)"

        # Check if signing
        SIGNING=$(pawd status 2>/dev/null | jq -r '.ValidatorInfo.VotingPower')
        if [[ "$SIGNING" != "0" ]] && [[ "$SIGNING" != "null" ]]; then
            echo "[OK] Validator has voting power: $SIGNING"
        else
            echo "[WARN] Validator has no voting power (may be jailed)"
        fi
    fi
fi

echo ""
echo "=== Validation Complete ==="
```

---

## Testing Recovery

### Quarterly Recovery Drills

**Schedule:** Test disaster recovery procedures every 3 months.

#### Test Procedure

1. **Select test environment**
   - Use non-production node
   - Or create test VM for drill

2. **Simulate failure**
   ```bash
   # Stop node
   sudo systemctl stop pawd

   # Destroy data
   rm -rf ~/.paw/data
   ```

3. **Execute recovery**
   - Follow [Data Restore Procedures](#data-restore-procedures)
   - Time the process
   - Document any issues

4. **Validate recovery**
   - Run validation checklist
   - Verify node rejoins network
   - Check validator signing (if applicable)

5. **Document results**
   ```bash
   # Record in recovery log
   cat >> /var/log/paw-recovery-tests.log <<EOF

   === Recovery Drill $(date) ===
   Test Type: Full restore from backup
   Backup Date: 2024-12-03
   Start Time: $(date +%H:%M:%S)
   End Time: $(date +%H:%M:%S)
   Duration: 45 minutes
   Result: SUCCESS
   Issues: None
   Notes: State sync completed in 30 minutes, validator rejoined network
   Tester: operator@example.com
   EOF
   ```

#### Test Scenarios to Cover

| Scenario | Frequency | Purpose |
|----------|-----------|---------|
| Local backup restore | Quarterly | Verify backup integrity |
| S3/GCS restore | Semi-annually | Test cloud backup retrieval |
| State sync recovery | Quarterly | Validate state sync configuration |
| Validator key restore | Annually | Verify key backup encryption |
| Full node rebuild | Annually | Test complete disaster recovery |

### Documentation of Test Results

Create test result template:

```markdown
# Recovery Test Result

**Date:** YYYY-MM-DD
**Test Type:** [Local Restore / Cloud Restore / State Sync / Full Rebuild]
**Tester:** [Name/Email]

## Test Environment

- Node Type: [Validator / Full Node / Archive]
- OS: [Ubuntu 22.04 / etc]
- pawd Version: [commit hash / tag]

## Test Steps

1. [Step taken]
2. [Step taken]
3. ...

## Results

- **Start Time:** HH:MM:SS
- **End Time:** HH:MM:SS
- **Total Duration:** XX minutes
- **Status:** [SUCCESS / FAILED / PARTIAL]

## Metrics

- Time to restore data: XX minutes
- Time to sync: XX minutes
- Validator downtime: XX minutes
- Data loss: [None / X blocks]

## Issues Encountered

1. [Issue description and resolution]
2. ...

## Improvements Identified

1. [Process improvement]
2. [Documentation update needed]
3. ...

## Sign-off

Tested by: ___________________
Reviewed by: ___________________
Date: ___________________
```

---

## Emergency Contacts and Escalation

### Team Contact List Template

**CRITICAL: Update this section with actual team contacts.**

```markdown
# PAW Blockchain Emergency Contacts

Last Updated: YYYY-MM-DD

## Primary On-Call

| Role | Name | Phone | Email | Telegram |
|------|------|-------|-------|----------|
| Primary Operator | [Name] | [+1-xxx-xxx-xxxx] | [email] | [@handle] |
| Backup Operator | [Name] | [+1-xxx-xxx-xxxx] | [email] | [@handle] |

## Technical Team

| Role | Name | Contact | Availability |
|------|------|---------|--------------|
| Lead Developer | [Name] | [email / phone] | [timezone / hours] |
| DevOps Lead | [Name] | [email / phone] | [timezone / hours] |
| Security Lead | [Name] | [email / phone] | [timezone / hours] |

## Escalation Path

Level 1 (0-15 min):
- Primary on-call operator
- Automated monitoring alerts

Level 2 (15-30 min):
- Backup operator
- DevOps lead

Level 3 (30-60 min):
- Lead developer
- Management notification

Level 4 (60+ min):
- All hands on deck
- External support (if contracted)

## External Resources

| Resource | Contact | Purpose |
|----------|---------|---------|
| Cloud Provider Support | [AWS / GCP support] | Infrastructure issues |
| Security Audit Firm | [Email / Phone] | Security incidents |
| Legal Counsel | [Email / Phone] | Legal matters |
```

### Escalation Procedures

#### Severity Levels

| Severity | Description | Response Time | Escalation |
|----------|-------------|---------------|------------|
| **Critical** | Validator down, network halt, security breach | < 15 minutes | Immediate Level 3 |
| **High** | Full node down, sync issues, backup failures | < 1 hour | Level 2 after 30 min |
| **Medium** | Performance degradation, non-critical errors | < 4 hours | Level 2 if unresolved |
| **Low** | Monitoring alerts, minor issues | < 24 hours | Level 1 only |

#### Incident Response Steps

1. **Detection** (0-5 min)
   - Alert received (monitoring, user report, etc.)
   - Initial assessment of severity
   - Log incident start time

2. **Notification** (5-15 min)
   - Contact primary on-call
   - Create incident channel (Slack/Discord)
   - Begin incident log

3. **Assessment** (15-30 min)
   - Identify root cause
   - Determine recovery path
   - Estimate downtime

4. **Escalation** (if needed)
   - Follow escalation path based on severity
   - Notify stakeholders
   - Request additional resources

5. **Recovery** (variable)
   - Execute recovery procedures
   - Monitor progress
   - Document steps taken

6. **Validation** (post-recovery)
   - Verify system health
   - Confirm all services operational
   - Check for data integrity

7. **Post-Mortem** (within 48 hours)
   - Document incident timeline
   - Identify root cause
   - Create action items to prevent recurrence

#### Incident Communication Template

```markdown
# Incident Report: [Brief Description]

**Incident ID:** INC-YYYYMMDD-NNN
**Severity:** [Critical / High / Medium / Low]
**Status:** [Investigating / Recovering / Resolved]
**Started:** YYYY-MM-DD HH:MM UTC
**Resolved:** YYYY-MM-DD HH:MM UTC (if resolved)

## Summary

[Brief description of the incident]

## Impact

- Affected Components: [Validator / Full nodes / Network]
- User Impact: [Description]
- Estimated Downtime: XX minutes

## Timeline (UTC)

- HH:MM - Incident detected
- HH:MM - Primary on-call notified
- HH:MM - Root cause identified
- HH:MM - Recovery initiated
- HH:MM - Service restored
- HH:MM - Incident closed

## Root Cause

[Detailed technical explanation]

## Resolution

[Steps taken to resolve]

## Action Items

1. [ ] [Preventive measure]
2. [ ] [Documentation update]
3. [ ] [Process improvement]

## Responders

- [Name] (Primary)
- [Name] (Support)

---
Report prepared by: [Name]
Date: YYYY-MM-DD
```

---

## Appendix

### Useful Commands Reference

```bash
# Node status
pawd status | jq

# Check sync progress
watch -n 5 'pawd status | jq .SyncInfo'

# View logs
journalctl -u pawd -f

# Check disk space
df -h ~/.paw

# Check validator status
pawd query staking validator $(pawd keys show validator --bech val -a)

# List backups
ls -lh /var/backups/paw/

# S3 backups
aws s3 ls s3://paw-blockchain-backups/

# GCS backups
gsutil ls gs://paw-blockchain-backups/

# Verify backup integrity
sha256sum -c backup.tar.gz.sha256

# Rollback to height
pawd rollback --hard --unsafe-rollback-height [HEIGHT]

# Reset tendermint state
pawd tendermint unsafe-reset-all --keep-addr-book
```

### Additional Resources

- [Cosmos SDK Documentation](https://docs.cosmos.network/)
- [CometBFT Operator Guide](https://docs.cometbft.com/v0.37/operator/)
- [PAW GitHub Repository](https://github.com/decristofaroj/paw)
- [PAW Production Monitoring](docs/PROD_MONITORING.md)

---

**End of Disaster Recovery Procedures**

**Remember:** The best recovery is the one you never need. Maintain robust backups, test regularly, and monitor proactively.
