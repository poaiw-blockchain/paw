# PAW Blockchain Disaster Recovery Plan

**Version:** 1.0
**Last Updated:** 2025-11-14
**Document Owner:** Infrastructure & Operations Team

## Table of Contents

1. [Overview](#overview)
2. [Backup Procedures](#backup-procedures)
3. [Recovery Procedures](#recovery-procedures)
4. [RTO and RPO Targets](#rto-and-rpo-targets)
5. [Backup Verification](#backup-verification)
6. [Failover Procedures](#failover-procedures)
7. [Testing & Drills](#testing--drills)

---

## Overview

### Purpose

This Disaster Recovery (DR) Plan ensures business continuity for the PAW blockchain in the event of catastrophic failures, natural disasters, or major incidents. The plan defines backup strategies, recovery procedures, and failover mechanisms to minimize downtime and data loss.

### Disaster Categories

**Category 1: Single Node Failure**

- Individual validator node hardware failure
- Corrupt database on single node
- Single datacenter outage
- **Impact:** Minimal if validators >67% online

**Category 2: Multiple Node Failure**

- Multiple validator failures simultaneously
- Regional outage affecting multiple validators
- Coordinated attack on validators
- **Impact:** Significant; may affect consensus

**Category 3: Complete Network Failure**

- Global outage affecting all validators
- Critical software bug halting chain
- Catastrophic network event
- **Impact:** Critical; full chain halt

**Category 4: Data Corruption**

- State database corruption
- Invalid chain state
- Blockchain fork requiring rollback
- **Impact:** Critical; may require chain recovery

### Disaster Response Principles

1. **Safety First:** Never compromise key security even during emergencies
2. **Data Integrity:** Preserve blockchain state integrity above all
3. **Transparency:** Communicate openly with validators and users
4. **Coordinated Recovery:** Synchronize recovery across validators
5. **Learn and Improve:** Update DR plan after every incident

---

## Backup Procedures

### 1. Validator Key Backups

#### 1.1 Private Validator Keys

**Critical Files:**

```
~/.paw/config/priv_validator_key.json
~/.paw/data/priv_validator_state.json
```

**Backup Frequency:** Once at creation, then after any key rotation
**Storage:** Offline, encrypted, geographically distributed

#### Backup Procedure

```bash
#!/bin/bash
# backup-validator-keys.sh

# Set variables
BACKUP_DIR="/secure/offline/storage"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
NODE_HOME="$HOME/.paw"
ENCRYPTION_KEY="$1"  # GPG key ID

# Validate encryption key provided
if [ -z "$ENCRYPTION_KEY" ]; then
    echo "Error: GPG key ID required"
    echo "Usage: $0 <gpg-key-id>"
    exit 1
fi

# Create backup directory
mkdir -p "$BACKUP_DIR/validator_keys_$TIMESTAMP"

# Copy validator keys
cp "$NODE_HOME/config/priv_validator_key.json" \
   "$BACKUP_DIR/validator_keys_$TIMESTAMP/"
cp "$NODE_HOME/data/priv_validator_state.json" \
   "$BACKUP_DIR/validator_keys_$TIMESTAMP/"

# Copy node key
cp "$NODE_HOME/config/node_key.json" \
   "$BACKUP_DIR/validator_keys_$TIMESTAMP/"

# Create checksum file
cd "$BACKUP_DIR/validator_keys_$TIMESTAMP"
sha256sum * > checksums.txt

# Encrypt the entire directory
cd "$BACKUP_DIR"
tar czf - "validator_keys_$TIMESTAMP" | \
    gpg --encrypt --recipient "$ENCRYPTION_KEY" \
    > "validator_keys_$TIMESTAMP.tar.gz.gpg"

# Securely delete unencrypted files
shred -vfz -n 10 "validator_keys_$TIMESTAMP"/*
rm -rf "validator_keys_$TIMESTAMP"

echo "Validator keys backed up and encrypted: validator_keys_$TIMESTAMP.tar.gz.gpg"
echo "Store this file in geographically separate locations"
echo "SHA256: $(sha256sum validator_keys_$TIMESTAMP.tar.gz.gpg)"
```

#### Backup Storage Strategy

**3-2-1 Rule:**

- **3** copies of data (primary + 2 backups)
- **2** different storage media types
- **1** copy off-site

**Storage Locations:**

1. **Primary:** Hardware Security Module (HSM) or hardware wallet
2. **Backup 1:** Encrypted USB drive in fireproof safe (Location A)
3. **Backup 2:** Encrypted backup in bank safety deposit box (Location B)
4. **Backup 3:** Encrypted cloud storage with MFA (as last resort)

**Security Requirements:**

- AES-256 encryption minimum
- Passphrase stored separately from backup
- Multi-signature access for production keys
- Documented recovery procedures
- Regular test restores (quarterly)

#### 1.2 Account Keys

**Files:**

```
~/.paw/keyring-file/
~/.paw/keyring-test/
```

**Backup Procedure:**

```bash
#!/bin/bash
# backup-account-keys.sh

BACKUP_DIR="/secure/backups/account_keys"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
KEYRING_BACKEND="file"  # or "test" for testnet

# Export keys
mkdir -p "$BACKUP_DIR/$TIMESTAMP"

# List all keys
KEYS=$(pawd keys list --keyring-backend $KEYRING_BACKEND --output json | jq -r '.[].name')

# Export each key
for KEY in $KEYS; do
    echo "Exporting key: $KEY"
    pawd keys export $KEY \
        --keyring-backend $KEYRING_BACKEND \
        > "$BACKUP_DIR/$TIMESTAMP/$KEY.key"
done

# Create encrypted archive
tar czf - -C "$BACKUP_DIR" "$TIMESTAMP" | \
    gpg --encrypt --recipient $GPG_KEY_ID \
    > "$BACKUP_DIR/account_keys_$TIMESTAMP.tar.gz.gpg"

# Clean up unencrypted files
shred -vfz -n 10 "$BACKUP_DIR/$TIMESTAMP"/*
rm -rf "$BACKUP_DIR/$TIMESTAMP"

echo "Account keys backed up: account_keys_$TIMESTAMP.tar.gz.gpg"
```

### 2. Node Data Backups

#### 2.1 Full Chain State Backup

**Purpose:** Complete blockchain data for full node recovery
**Frequency:** Weekly full backups + daily incrementals
**Retention:** 30 days for full backups, 7 days for incrementals

```bash
#!/bin/bash
# backup-chain-data.sh

set -e

# Configuration
NODE_HOME="$HOME/.paw"
BACKUP_BASE="/mnt/backups/paw"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
RETENTION_DAYS=30
S3_BUCKET="s3://paw-backups"  # Optional cloud backup

# Get current block height
BLOCK_HEIGHT=$(curl -s http://localhost:26657/status | jq -r '.result.sync_info.latest_block_height')

echo "Starting backup at block height: $BLOCK_HEIGHT"

# Create backup directory
BACKUP_DIR="$BACKUP_BASE/full_$TIMESTAMP"
mkdir -p "$BACKUP_DIR"

# Stop node (optional - for consistent backup)
# systemctl stop pawd

# Backup chain data
echo "Backing up chain data..."
rsync -av --progress \
    "$NODE_HOME/data/" \
    "$BACKUP_DIR/data/"

# Backup configuration
echo "Backing up configuration..."
rsync -av --progress \
    "$NODE_HOME/config/" \
    "$BACKUP_DIR/config/"

# Create manifest
cat > "$BACKUP_DIR/manifest.json" <<EOF
{
    "timestamp": "$TIMESTAMP",
    "block_height": $BLOCK_HEIGHT,
    "backup_type": "full",
    "node_version": "$(pawd version)",
    "chain_id": "$(cat $NODE_HOME/config/genesis.json | jq -r '.chain_id')"
}
EOF

# Create checksum
cd "$BACKUP_DIR"
find . -type f -exec sha256sum {} \; > checksums.txt

# Compress backup
cd "$BACKUP_BASE"
echo "Compressing backup..."
tar czf "full_$TIMESTAMP.tar.gz" "full_$TIMESTAMP"

# Calculate final checksum
sha256sum "full_$TIMESTAMP.tar.gz" > "full_$TIMESTAMP.tar.gz.sha256"

# Optional: Upload to S3
if [ -n "$S3_BUCKET" ]; then
    echo "Uploading to S3..."
    aws s3 cp "full_$TIMESTAMP.tar.gz" "$S3_BUCKET/full/"
    aws s3 cp "full_$TIMESTAMP.tar.gz.sha256" "$S3_BUCKET/full/"
fi

# Restart node if it was stopped
# systemctl start pawd

# Cleanup old backups
echo "Cleaning up old backups..."
find "$BACKUP_BASE" -name "full_*.tar.gz" -mtime +$RETENTION_DAYS -delete
find "$BACKUP_BASE" -name "full_*" -type d -mtime +$RETENTION_DAYS -exec rm -rf {} +

echo "Backup completed: full_$TIMESTAMP.tar.gz"
echo "Block height: $BLOCK_HEIGHT"
echo "Size: $(du -h full_$TIMESTAMP.tar.gz | cut -f1)"
```

#### 2.2 Incremental State Backup

**Purpose:** Capture changes since last full backup
**Frequency:** Daily
**Method:** Using rsync with hard links for space efficiency

```bash
#!/bin/bash
# backup-chain-incremental.sh

set -e

BACKUP_BASE="/mnt/backups/paw"
NODE_HOME="$HOME/.paw"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
LATEST_FULL=$(ls -t $BACKUP_BASE/full_*.tar.gz | head -1)

if [ -z "$LATEST_FULL" ]; then
    echo "Error: No full backup found. Run full backup first."
    exit 1
fi

# Extract timestamp from latest full backup
FULL_TIMESTAMP=$(basename $LATEST_FULL .tar.gz | cut -d_ -f2-)

BACKUP_DIR="$BACKUP_BASE/incremental_$TIMESTAMP"
REFERENCE_DIR="$BACKUP_BASE/full_$FULL_TIMESTAMP"

# Create incremental backup using hard links
mkdir -p "$BACKUP_DIR"

# If reference doesn't exist, extract it
if [ ! -d "$REFERENCE_DIR" ]; then
    tar xzf "$LATEST_FULL" -C "$BACKUP_BASE"
fi

# Incremental backup with rsync
rsync -av --progress \
    --link-dest="$REFERENCE_DIR/data" \
    "$NODE_HOME/data/" \
    "$BACKUP_DIR/data/"

# Get current height
BLOCK_HEIGHT=$(curl -s http://localhost:26657/status | jq -r '.result.sync_info.latest_block_height')

# Create manifest
cat > "$BACKUP_DIR/manifest.json" <<EOF
{
    "timestamp": "$TIMESTAMP",
    "block_height": $BLOCK_HEIGHT,
    "backup_type": "incremental",
    "reference_backup": "full_$FULL_TIMESTAMP"
}
EOF

echo "Incremental backup completed: incremental_$TIMESTAMP"
```

#### 2.3 Application State Snapshots

**Purpose:** Quick state snapshots for rapid recovery
**Frequency:** Every 1000 blocks (configurable)
**Storage:** Local SSD + remote storage

```bash
#!/bin/bash
# create-state-snapshot.sh

NODE_HOME="$HOME/.paw"
SNAPSHOT_DIR="/mnt/snapshots/paw"
INTERVAL=1000  # Snapshot every 1000 blocks

# Get current height
CURRENT_HEIGHT=$(curl -s http://localhost:26657/status | \
    jq -r '.result.sync_info.latest_block_height')

# Check if we should create snapshot
LAST_SNAPSHOT=$(ls -t $SNAPSHOT_DIR/snapshot_*.tar.gz 2>/dev/null | head -1)
if [ -n "$LAST_SNAPSHOT" ]; then
    LAST_HEIGHT=$(basename $LAST_SNAPSHOT .tar.gz | grep -oP '\d+$')
    if [ $((CURRENT_HEIGHT - LAST_HEIGHT)) -lt $INTERVAL ]; then
        echo "Not yet time for snapshot (current: $CURRENT_HEIGHT, last: $LAST_HEIGHT)"
        exit 0
    fi
fi

echo "Creating snapshot at height $CURRENT_HEIGHT"

# Create snapshot using pawd
pawd snapshots create \
    --home "$NODE_HOME" \
    --output "$SNAPSHOT_DIR/snapshot_${CURRENT_HEIGHT}.tar.gz"

# Prune old snapshots (keep last 10)
ls -t $SNAPSHOT_DIR/snapshot_*.tar.gz | tail -n +11 | xargs -r rm

echo "Snapshot created: snapshot_${CURRENT_HEIGHT}.tar.gz"
```

### 3. Configuration File Backups

#### 3.1 Critical Configuration Files

**Files to back up:**

```
~/.paw/config/config.toml          # Tendermint config
~/.paw/config/app.toml             # Application config
~/.paw/config/client.toml          # Client config
~/.paw/config/genesis.json         # Genesis file
~/.paw/config/addrbook.json        # Address book (optional)
```

**Backup Procedure:**

```bash
#!/bin/bash
# backup-configs.sh

NODE_HOME="$HOME/.paw"
BACKUP_DIR="/etc/paw/config_backups"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
GIT_REPO="/etc/paw/config_repo"  # Optional git-based versioning

mkdir -p "$BACKUP_DIR"

# Create backup
CONFIG_BACKUP="$BACKUP_DIR/config_$TIMESTAMP"
mkdir -p "$CONFIG_BACKUP"

# Copy configs
cp "$NODE_HOME/config/config.toml" "$CONFIG_BACKUP/"
cp "$NODE_HOME/config/app.toml" "$CONFIG_BACKUP/"
cp "$NODE_HOME/config/client.toml" "$CONFIG_BACKUP/"
cp "$NODE_HOME/config/genesis.json" "$CONFIG_BACKUP/"

# Create manifest
cat > "$CONFIG_BACKUP/manifest.txt" <<EOF
Backup Date: $(date -u)
Node Version: $(pawd version)
Chain ID: $(cat $NODE_HOME/config/genesis.json | jq -r '.chain_id')
Moniker: $(grep ^moniker $NODE_HOME/config/config.toml | cut -d'=' -f2 | tr -d ' "')
EOF

# Compress
tar czf "$BACKUP_DIR/config_$TIMESTAMP.tar.gz" -C "$BACKUP_DIR" "config_$TIMESTAMP"
rm -rf "$CONFIG_BACKUP"

# Optional: Version control with git
if [ -d "$GIT_REPO" ]; then
    cd "$GIT_REPO"
    cp "$NODE_HOME/config/"*.toml .
    cp "$NODE_HOME/config/genesis.json" .
    git add .
    git commit -m "Config backup $TIMESTAMP" || true
    git push || true
fi

# Cleanup old backups (keep 90 days)
find "$BACKUP_DIR" -name "config_*.tar.gz" -mtime +90 -delete

echo "Configuration backed up: config_$TIMESTAMP.tar.gz"
```

#### 3.2 Automated Configuration Backup via Cron

```cron
# Add to /etc/cron.d/paw-backups

# Configuration backup - daily at 2 AM
0 2 * * * paw /usr/local/bin/backup-configs.sh >> /var/log/paw/config-backup.log 2>&1

# Validator keys backup - weekly on Sunday at 3 AM
0 3 * * 0 paw /usr/local/bin/backup-validator-keys.sh $GPG_KEY_ID >> /var/log/paw/key-backup.log 2>&1

# Full chain data backup - weekly on Sunday at 1 AM
0 1 * * 0 paw /usr/local/bin/backup-chain-data.sh >> /var/log/paw/chain-backup.log 2>&1

# Incremental backup - daily at 4 AM
0 4 * * * paw /usr/local/bin/backup-chain-incremental.sh >> /var/log/paw/incremental-backup.log 2>&1

# Snapshot check - every 6 hours
0 */6 * * * paw /usr/local/bin/create-state-snapshot.sh >> /var/log/paw/snapshot.log 2>&1
```

### 4. Database Backups

#### 4.1 Application Database

If using external database for API/indexer:

```bash
#!/bin/bash
# backup-postgres.sh

DB_NAME="paw_indexer"
DB_USER="paw"
DB_HOST="localhost"
BACKUP_DIR="/mnt/backups/postgres"
TIMESTAMP=$(date +%Y%m%d_%H%M%S)

# Full backup
pg_dump -h $DB_HOST -U $DB_USER -d $DB_NAME -Fc \
    -f "$BACKUP_DIR/paw_db_$TIMESTAMP.dump"

# Compress
gzip "$BACKUP_DIR/paw_db_$TIMESTAMP.dump"

# Upload to S3 (optional)
aws s3 cp "$BACKUP_DIR/paw_db_$TIMESTAMP.dump.gz" \
    s3://paw-backups/database/

# Cleanup old backups (keep 14 days)
find "$BACKUP_DIR" -name "paw_db_*.dump.gz" -mtime +14 -delete
```

---

## Recovery Procedures

### Recovery Procedure 1: Single Validator Failure

**Scenario:** One validator node has hardware failure or corruption
**RTO:** 1-2 hours
**RPO:** 0 (no data loss)

#### Prerequisites

- Backup validator keys available
- Access to recent chain state backup or snapshot
- New hardware provisioned (or existing backup validator)

#### Recovery Steps

**Step 1: Provision New Infrastructure (0-30 minutes)**

```bash
# On new server
# 1. Install dependencies
sudo apt update && sudo apt install -y build-essential jq curl wget

# 2. Install pawd
wget https://github.com/paw-chain/paw/releases/download/v1.0.0/pawd-linux-amd64
sudo mv pawd-linux-amd64 /usr/local/bin/pawd
sudo chmod +x /usr/local/bin/pawd

# 3. Verify installation
pawd version

# 4. Create service user
sudo useradd -m -s /bin/bash paw

# 5. Initialize node
sudo -u paw pawd init <moniker> --chain-id paw-mainnet-1 --home /home/paw/.paw
```

**Step 2: Restore Configuration (30-45 minutes)**

```bash
# Restore from backup
sudo -u paw mkdir -p /home/paw/.paw/config

# Copy genesis file
sudo -u paw wget -O /home/paw/.paw/config/genesis.json \
    https://raw.githubusercontent.com/paw-chain/networks/main/mainnet/genesis.json

# Restore configuration backup
cd /home/paw/.paw/config
sudo -u paw tar xzf /path/to/config_backup.tar.gz --strip-components=1

# Verify genesis hash
sha256sum /home/paw/.paw/config/genesis.json
# Should match: <expected_hash>
```

**Step 3: Restore Validator Keys (45-50 minutes)**

```bash
# CRITICAL: Ensure old validator is completely stopped before proceeding
# Double-signing will result in slashing!

# Decrypt and extract validator keys
gpg --decrypt /secure/storage/validator_keys_TIMESTAMP.tar.gz.gpg | \
    tar xz -C /tmp/

# Verify checksums
cd /tmp/validator_keys_TIMESTAMP
sha256sum -c checksums.txt

# Restore keys
sudo -u paw cp priv_validator_key.json /home/paw/.paw/config/
sudo -u paw cp priv_validator_state.json /home/paw/.paw/data/
sudo -u paw cp node_key.json /home/paw/.paw/config/

# Set permissions
sudo chmod 600 /home/paw/.paw/config/priv_validator_key.json
sudo chmod 600 /home/paw/.paw/data/priv_validator_state.json

# Securely delete temporary files
shred -vfz -n 10 /tmp/validator_keys_TIMESTAMP/*
rm -rf /tmp/validator_keys_TIMESTAMP
```

**Step 4: Restore Chain Data (50-90 minutes)**

**Option A: From Snapshot (Fastest)**

```bash
# Download latest snapshot
SNAPSHOT_URL="https://snapshots.paw.network/latest.tar.gz"
wget -O /tmp/snapshot.tar.gz $SNAPSHOT_URL

# Verify checksum
wget -O /tmp/snapshot.tar.gz.sha256 ${SNAPSHOT_URL}.sha256
sha256sum -c /tmp/snapshot.tar.gz.sha256

# Extract to data directory
sudo -u paw tar xzf /tmp/snapshot.tar.gz -C /home/paw/.paw/data/

# Cleanup
rm /tmp/snapshot.tar.gz*
```

**Option B: From Full Backup**

```bash
# Restore from backup
sudo -u paw tar xzf /mnt/backups/paw/full_TIMESTAMP.tar.gz \
    -C /home/paw/.paw/ --strip-components=1

# Node will sync remaining blocks from network
```

**Option C: State Sync (Fastest, but requires trusted peers)**

```bash
# Edit config.toml
sudo -u paw nano /home/paw/.paw/config/config.toml

# Enable state sync
[statesync]
enable = true
rpc_servers = "https://rpc1.paw.network:26657,https://rpc2.paw.network:26657"
trust_height = <recent_height>
trust_hash = "<block_hash>"
trust_period = "168h0m0s"

# Get trust height and hash from RPC
LATEST_HEIGHT=$(curl -s https://rpc1.paw.network:26657/block | jq -r '.result.block.header.height')
TRUST_HEIGHT=$((LATEST_HEIGHT - 1000))
TRUST_HASH=$(curl -s https://rpc1.paw.network:26657/block?height=$TRUST_HEIGHT | jq -r '.result.block_id.hash')

echo "trust_height = $TRUST_HEIGHT"
echo "trust_hash = \"$TRUST_HASH\""
```

**Step 5: Configure and Start Node (90-100 minutes)**

```bash
# Configure seeds and persistent peers
sudo -u paw nano /home/paw/.paw/config/config.toml

# Update:
seeds = "seed1.paw.network:26656,seed2.paw.network:26656"
persistent_peers = "<peer_list>"

# Create systemd service
sudo tee /etc/systemd/system/pawd.service > /dev/null <<EOF
[Unit]
Description=PAW Blockchain Daemon
After=network-online.target

[Service]
User=paw
ExecStart=/usr/local/bin/pawd start --home /home/paw/.paw
Restart=always
RestartSec=3
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
EOF

# Enable and start service
sudo systemctl daemon-reload
sudo systemctl enable pawd
sudo systemctl start pawd

# Monitor startup
sudo journalctl -u pawd -f
```

**Step 6: Verify Recovery (100-120 minutes)**

```bash
# Check node status
curl http://localhost:26657/status | jq

# Verify catching up
curl http://localhost:26657/status | jq '.result.sync_info.catching_up'
# Should be false when fully synced

# Check validator is signing
curl http://localhost:26657/status | jq '.result.validator_info'

# Monitor blocks signed
pawd query slashing signing-info \
    $(pawd tendermint show-validator) | \
    jq '.missed_blocks_counter'

# Check peers
curl http://localhost:26657/net_info | jq '.result.n_peers'

# Verify consensus participation
tail -f /home/paw/.paw/logs/consensus.log | grep "signed"
```

**Recovery Checklist:**

- [ ] New server provisioned and secured
- [ ] Pawd binary installed and verified
- [ ] Genesis file matches network
- [ ] Configuration restored
- [ ] Validator keys restored (AFTER old node stopped)
- [ ] Chain data restored or synced
- [ ] Node started and syncing
- [ ] Validator signing blocks
- [ ] No slashing penalties
- [ ] Monitoring configured
- [ ] Peers connected
- [ ] Post-recovery documentation completed

---

### Recovery Procedure 2: Multiple Validator Failure

**Scenario:** Multiple validators (but <33%) offline simultaneously
**RTO:** 2-4 hours
**RPO:** 0 (no data loss)
**Impact:** Network continues but with reduced security margin

#### Assessment Phase (0-15 minutes)

```bash
# Assess situation
# 1. How many validators are down?
curl https://rpc.paw.network:26657/validators | jq '.result.total'

# 2. What percentage of voting power?
curl https://rpc.paw.network:26657/validators | \
    jq '[.result.validators[] | select(.voting_power)] |
    map(.voting_power | tonumber) | add'

# 3. Is consensus still progressing?
watch -n 1 'curl -s http://localhost:26657/status | jq .result.sync_info.latest_block_height'
```

#### Triage Decision Tree

```
Are >67% validators online?
│
├─ YES → Network operational
│         └─ Recover failed validators using Single Validator Procedure
│         └─ Prioritize by voting power (highest first)
│         └─ Coordinate recovery to avoid simultaneous key loading
│
└─ NO → Network halted
         └─ Proceed to Complete Network Failure recovery
         └─ Coordinate with all validators
```

#### Coordinated Recovery (if >67% online)

**Step 1: Establish Coordination (0-20 minutes)**

```bash
# Set up coordination channel
# - Slack: #paw-validator-recovery
# - Telegram: @PAWValidatorRecovery
# - Discord: #validator-emergency

# Create recovery schedule
cat > recovery_schedule.txt <<EOF
Validator Recovery Schedule
Generated: $(date -u)

Priority 1 (Largest Voting Power):
- Validator A: Start 12:00 UTC (Operator: Alice)
- Validator B: Start 12:30 UTC (Operator: Bob)

Priority 2:
- Validator C: Start 13:00 UTC (Operator: Carol)
- Validator D: Start 13:30 UTC (Operator: Dave)

Coordination:
- Each validator: announce when starting recovery
- Each validator: announce when signing blocks
- Monitor for any double-sign events
EOF
```

**Step 2: Sequential Recovery**

- Recover validators ONE AT A TIME
- Wait for each validator to start signing before proceeding to next
- Use Single Validator Recovery Procedure for each
- Maintain constant communication

**Step 3: Monitor Recovery**

```bash
#!/bin/bash
# monitor-validator-recovery.sh

while true; do
    clear
    echo "=== Validator Recovery Status ==="
    echo "Time: $(date -u)"
    echo

    # Current block height
    HEIGHT=$(curl -s http://localhost:26657/status | jq -r '.result.sync_info.latest_block_height')
    echo "Block Height: $HEIGHT"

    # Active validators
    ACTIVE=$(curl -s http://localhost:26657/validators | jq '.result.total')
    echo "Active Validators: $ACTIVE"

    # Total voting power
    TOTAL_VP=$(curl -s http://localhost:26657/validators | \
        jq '[.result.validators[].voting_power | tonumber] | add')
    echo "Total Voting Power: $TOTAL_VP"

    # Blocks per minute
    sleep 60
    NEW_HEIGHT=$(curl -s http://localhost:26657/status | jq -r '.result.sync_info.latest_block_height')
    BPM=$((NEW_HEIGHT - HEIGHT))
    echo "Blocks/min: $BPM (normal: ~10)"

    sleep 5
done
```

---

### Recovery Procedure 3: Complete Network Failure

**Scenario:** All or >33% of validators offline (consensus halted)
**RTO:** 4-12 hours
**RPO:** 0-100 blocks (depending on backup recency)
**Severity:** CRITICAL

#### Phase 1: Emergency Assessment (0-30 minutes)

**Immediate Actions:**

```bash
# 1. Confirm total network halt
curl -s https://rpc.paw.network:26657/status | jq '.result.sync_info'
# Note the last block height and time

# 2. Contact all validators
# Use emergency contact list
# Establish coordination bridge (Zoom/Discord)

# 3. Identify last known good state
LAST_HEIGHT=$(curl -s https://rpc.paw.network:26657/status | \
    jq -r '.result.sync_info.latest_block_height')
echo "Last block height: $LAST_HEIGHT"

# 4. Determine cause
# - Software bug?
# - Coordinated attack?
# - Infrastructure failure?
# - Network partition?

# 5. Verify validator status
# Create spreadsheet tracking:
# - Validator moniker
# - Contact status
# - Last block signed
# - Data integrity status
# - Recovery readiness
```

#### Phase 2: Coordination (30 minutes - 2 hours)

**Emergency Validator Call Checklist:**

- [ ] All validators contacted
- [ ] Consensus on recovery plan
- [ ] Identify validator with highest integrity state
- [ ] Determine if rollback needed
- [ ] Set synchronized restart time
- [ ] Assign roles (coordinator, monitors, communicators)

**Communications:**

```markdown
# Public Status Update Template

CRITICAL: PAW NETWORK HALTED

Status: The PAW blockchain has halted consensus at block [HEIGHT]
Time: [TIMESTAMP UTC]
Cause: Under investigation

Impact:

- Transactions are not being processed
- All funds remain secure
- No user action required

Recovery:
Validator coordination in progress. Estimated recovery: [TIMEFRAME]

Next Update: [TIME]
```

#### Phase 3: Recovery Decision

**Option A: Standard Restart (No Rollback)**

_Use when:_

- Chain state is valid
- All validators agree on last block
- No consensus failure or corruption

```bash
# Coordinated restart procedure
# All validators execute simultaneously at agreed time

# 1. Verify data integrity
pawd validate-genesis ~/.paw/config/genesis.json

# 2. Check last block hash consistency
curl http://localhost:26657/block?height=$LAST_HEIGHT | \
    jq '.result.block_id.hash'
# All validators must have same hash

# 3. Synchronized start
# At agreed time (e.g., 15:00:00 UTC):
sudo systemctl start pawd

# 4. Monitor consensus formation
watch -n 1 'curl -s http://localhost:26657/status | jq .result.sync_info'
```

**Option B: Rollback and Restart**

_Use when:_

- State corruption detected
- Invalid block produced
- Need to revert to last known good state

```bash
# 1. Identify rollback target
ROLLBACK_HEIGHT=12345  # Last known good height

# 2. All validators: Stop nodes
sudo systemctl stop pawd

# 3. Backup current state
cp -r ~/.paw/data ~/.paw/data.backup_$(date +%s)

# 4. Rollback to target height
pawd rollback --hard --height $ROLLBACK_HEIGHT --home ~/.paw

# 5. Verify consistency
curl http://localhost:26657/block?height=$ROLLBACK_HEIGHT | \
    jq '.result.block_id.hash'
# All validators must agree on this hash

# 6. Coordinated restart
# Wait for coordinator signal
sudo systemctl start pawd
```

**Option C: Genesis Restart (Last Resort)**

_Use only when:_

- State is irrecoverably corrupted
- Consensus cannot be achieved
- Critical bug requires chain reset

**WARNING:** This creates a new chain. Requires governance approval.

```bash
# 1. Export final state
pawd export --height $LAST_GOOD_HEIGHT > chain_state_export.json

# 2. Create new genesis
# Preserve account balances and critical state
python3 scripts/create_genesis_from_export.py \
    chain_state_export.json > new_genesis.json

# 3. Validators: Reset data
pawd unsafe-reset-all --home ~/.paw

# 4. Install new genesis
cp new_genesis.json ~/.paw/config/genesis.json

# 5. Verify genesis hash
sha256sum ~/.paw/config/genesis.json
# All validators must have identical hash

# 6. Coordinated start
sudo systemctl start pawd

# 7. Submit gentxs and collect
# Follow standard chain initialization
```

#### Phase 4: Recovery Execution (2-6 hours)

**Pre-Start Checklist:**

- [ ] All validators ready
- [ ] Recovery method agreed
- [ ] Sync time established (use NTP)
- [ ] Monitoring ready
- [ ] Communication channels open
- [ ] Rollback plan if recovery fails

**Execution:**

```bash
#!/bin/bash
# coordinated-start.sh

TARGET_TIME="2025-11-14T15:00:00Z"
CURRENT_TIME=$(date -u +%s)
TARGET_TIMESTAMP=$(date -d "$TARGET_TIME" +%s)
SLEEP_SECONDS=$((TARGET_TIMESTAMP - CURRENT_TIME))

echo "Coordinated start at: $TARGET_TIME"
echo "Sleeping for $SLEEP_SECONDS seconds..."

# Countdown
while [ $SLEEP_SECONDS -gt 0 ]; do
    echo "Starting in $SLEEP_SECONDS seconds..."
    sleep 10
    SLEEP_SECONDS=$((SLEEP_SECONDS - 10))
done

# Start node
echo "STARTING NODE NOW"
sudo systemctl start pawd

# Monitor
journalctl -u pawd -f
```

#### Phase 5: Post-Recovery Validation (6-12 hours)

```bash
# Validation checklist script
#!/bin/bash
# validate-recovery.sh

echo "=== PAW Network Recovery Validation ==="
echo

# 1. Check consensus
echo "1. Checking consensus..."
STATUS=$(curl -s http://localhost:26657/status)
CATCHING_UP=$(echo $STATUS | jq -r '.result.sync_info.catching_up')
HEIGHT=$(echo $STATUS | jq -r '.result.sync_info.latest_block_height')

if [ "$CATCHING_UP" = "false" ]; then
    echo "✓ Consensus active, height: $HEIGHT"
else
    echo "✗ Node still catching up"
fi

# 2. Check validators signing
echo "2. Checking validator participation..."
VALIDATORS=$(curl -s http://localhost:26657/validators | jq '.result.total')
echo "✓ Active validators: $VALIDATORS"

# 3. Check block production
echo "3. Monitoring block production..."
for i in {1..5}; do
    HEIGHT=$(curl -s http://localhost:26657/status | jq -r '.result.sync_info.latest_block_height')
    echo "  Block: $HEIGHT"
    sleep 6
done

# 4. Test transaction
echo "4. Testing transaction submission..."
pawd tx bank send test1 test2 100upaw --gas auto --yes
if [ $? -eq 0 ]; then
    echo "✓ Transactions processing"
else
    echo "✗ Transaction failed"
fi

# 5. Check for slashing events
echo "5. Checking for slashing..."
pawd query slashing signing-infos --limit 100

echo
echo "Recovery validation complete"
```

**Post-Recovery Checklist:**

- [ ] Consensus restored
- [ ] All validators signing
- [ ] Block production normal (6s avg)
- [ ] Transactions processing
- [ ] No double-sign slashing
- [ ] Peer count normal
- [ ] Monitoring operational
- [ ] Public status updated
- [ ] Post-mortem scheduled

---

### Recovery Procedure 4: Data Corruption

**Scenario:** Chain state corruption on single or multiple nodes
**RTO:** 2-6 hours
**RPO:** 0-1000 blocks

#### Corruption Detection

```bash
# Signs of corruption:
# - Database errors in logs
# - Node crashes with "database corrupted" message
# - Inconsistent block hashes between nodes
# - Unexpected state transitions
# - Panic errors from state machine

# Verify corruption
journalctl -u pawd | grep -i "corrupt\|panic\|fatal"

# Check database integrity (if using goleveldb)
pawd inspect-db --home ~/.paw
```

#### Recovery Options

**Option 1: Restore from Backup**

```bash
# 1. Stop node
sudo systemctl stop pawd

# 2. Backup corrupted data
mv ~/.paw/data ~/.paw/data.corrupted_$(date +%s)

# 3. Restore from latest backup
tar xzf /mnt/backups/paw/full_TIMESTAMP.tar.gz \
    -C ~/.paw/ data/

# 4. Restart and catch up
sudo systemctl start pawd

# Node will sync remaining blocks from network
```

**Option 2: State Sync from Healthy Peers**

```bash
# 1. Stop node
sudo systemctl stop pawd

# 2. Reset corrupted state
pawd unsafe-reset-all --home ~/.paw --keep-addr-book

# 3. Configure state sync
nano ~/.paw/config/config.toml

[statesync]
enable = true
rpc_servers = "https://rpc1.paw.network:26657,https://rpc2.paw.network:26657"
trust_height = <height>
trust_hash = "<hash>"

# 4. Start node
sudo systemctl start pawd
```

**Option 3: Complete Resync**

```bash
# Last resort: Download entire chain from scratch

# 1. Stop node
sudo systemctl stop pawd

# 2. Backup keys (if not backed up)
cp ~/.paw/config/priv_validator_key.json /secure/location/
cp ~/.paw/data/priv_validator_state.json /secure/location/

# 3. Reset all data
pawd unsafe-reset-all --home ~/.paw

# 4. Restore keys
cp /secure/location/priv_validator_key.json ~/.paw/config/
cp /secure/location/priv_validator_state.json ~/.paw/data/

# 5. Start from genesis
sudo systemctl start pawd

# Will take several hours to days depending on chain height
```

---

## RTO and RPO Targets

### Recovery Time Objective (RTO)

Maximum acceptable downtime for each scenario:

| Scenario                   | RTO Target  | Notes                            |
| -------------------------- | ----------- | -------------------------------- |
| Single validator failure   | 1-2 hours   | Network continues operating      |
| Multiple validator failure | 2-4 hours   | If >67% still online             |
| Complete network failure   | 4-12 hours  | Requires coordination            |
| Data corruption (single)   | 2-6 hours   | Depends on backup availability   |
| Regional outage            | 6-12 hours  | May require datacenter migration |
| Catastrophic loss          | 12-48 hours | Multiple simultaneous failures   |

### Recovery Point Objective (RPO)

Maximum acceptable data loss:

| Data Type       | RPO Target             | Backup Frequency                   |
| --------------- | ---------------------- | ---------------------------------- |
| Validator keys  | 0 (no loss acceptable) | Offline backup at creation         |
| Chain state     | 0-1000 blocks          | Continuous + snapshots/1000 blocks |
| Configuration   | 0-24 hours             | Daily automated backup             |
| Account keys    | 0 (no loss acceptable) | Manual backup after changes        |
| Application DB  | 0-6 hours              | Every 6 hours                      |
| Monitoring data | 24 hours               | Not critical for recovery          |

### SLA Commitments

**Network Availability Target:** 99.9% uptime

- **Allowed downtime:** ~8.76 hours/year
- **Monthly:** ~43 minutes

**Recovery Performance Metrics:**

| Metric                          | Target       |
| ------------------------------- | ------------ |
| Mean Time To Detect (MTTD)      | < 5 minutes  |
| Mean Time To Acknowledge (MTTA) | < 15 minutes |
| Mean Time To Contain (MTTC)     | < 30 minutes |
| Mean Time To Recover (MTTR)     | < 4 hours    |

---

## Backup Verification

### Verification Procedures

#### 1. Validator Key Backup Verification

**Frequency:** Quarterly
**Procedure:**

```bash
#!/bin/bash
# verify-validator-keys.sh

BACKUP_FILE="$1"
GPG_KEY="$2"

echo "Verifying validator key backup: $BACKUP_FILE"

# 1. Decrypt backup
gpg --decrypt "$BACKUP_FILE" | tar xz -C /tmp/verify_keys

# 2. Verify checksums
cd /tmp/verify_keys/validator_keys_*
if sha256sum -c checksums.txt; then
    echo "✓ Checksums valid"
else
    echo "✗ Checksum verification failed"
    exit 1
fi

# 3. Verify key format
if jq -e . priv_validator_key.json > /dev/null 2>&1; then
    echo "✓ Key file format valid"
else
    echo "✗ Invalid key file format"
    exit 1
fi

# 4. Extract validator address
VAL_ADDR=$(jq -r '.address' priv_validator_key.json)
echo "Validator Address: $VAL_ADDR"

# 5. Cleanup
cd /
shred -vfz -n 10 /tmp/verify_keys/validator_keys_*/*
rm -rf /tmp/verify_keys

echo "✓ Backup verification successful"
```

#### 2. Chain Data Backup Verification

**Frequency:** Monthly
**Procedure:**

```bash
#!/bin/bash
# verify-chain-backup.sh

BACKUP_FILE="$1"
TEST_DIR="/tmp/backup_verify_$(date +%s)"

echo "Verifying chain data backup: $BACKUP_FILE"

# 1. Extract to test directory
mkdir -p "$TEST_DIR"
tar xzf "$BACKUP_FILE" -C "$TEST_DIR"

# 2. Verify checksums
cd "$TEST_DIR/full_"*
if [ -f checksums.txt ]; then
    sha256sum -c checksums.txt
    if [ $? -eq 0 ]; then
        echo "✓ Checksums valid"
    else
        echo "✗ Checksum verification failed"
        exit 1
    fi
fi

# 3. Verify manifest
if [ -f manifest.json ]; then
    BACKUP_HEIGHT=$(jq -r '.block_height' manifest.json)
    echo "Backup block height: $BACKUP_HEIGHT"
else
    echo "✗ Manifest missing"
    exit 1
fi

# 4. Verify genesis
if [ -f config/genesis.json ]; then
    GENESIS_HASH=$(sha256sum config/genesis.json | cut -d' ' -f1)
    echo "Genesis hash: $GENESIS_HASH"
    # Compare with known good hash
    EXPECTED_HASH="<insert expected genesis hash>"
    if [ "$GENESIS_HASH" = "$EXPECTED_HASH" ]; then
        echo "✓ Genesis file valid"
    else
        echo "⚠ Genesis hash mismatch"
    fi
fi

# 5. Test restore (optional - resource intensive)
# Would actually start a node from this backup

# Cleanup
rm -rf "$TEST_DIR"

echo "✓ Backup verification complete"
```

#### 3. Test Recovery Drill

**Frequency:** Quarterly
**Purpose:** Ensure recovery procedures work and team is trained

```bash
#!/bin/bash
# recovery-drill.sh
# Simulates validator recovery on testnet

DRILL_LOG="/var/log/paw/recovery_drill_$(date +%Y%m%d_%H%M%S).log"

echo "=== PAW Recovery Drill ===" | tee -a $DRILL_LOG
echo "Start: $(date -u)" | tee -a $DRILL_LOG

# 1. Stop testnet validator
echo "1. Stopping testnet validator..." | tee -a $DRILL_LOG
systemctl stop pawd-testnet
STOP_TIME=$(date +%s)

# 2. Simulate failure - delete data
echo "2. Simulating data loss..." | tee -a $DRILL_LOG
rm -rf ~/.paw-testnet/data/*

# 3. Restore from backup
echo "3. Restoring from backup..." | tee -a $DRILL_LOG
BACKUP=$(ls -t /mnt/backups/testnet/full_*.tar.gz | head -1)
tar xzf "$BACKUP" -C ~/.paw-testnet/ --strip-components=1
RESTORE_TIME=$(date +%s)
RESTORE_DURATION=$((RESTORE_TIME - STOP_TIME))

# 4. Restart validator
echo "4. Restarting validator..." | tee -a $DRILL_LOG
systemctl start pawd-testnet

# 5. Wait for sync
echo "5. Waiting for sync..." | tee -a $DRILL_LOG
while true; do
    CATCHING_UP=$(curl -s http://localhost:26657/status | jq -r '.result.sync_info.catching_up')
    if [ "$CATCHING_UP" = "false" ]; then
        break
    fi
    sleep 5
done
SYNC_TIME=$(date +%s)
SYNC_DURATION=$((SYNC_TIME - RESTORE_TIME))

# 6. Verify signing
echo "6. Verifying signing..." | tee -a $DRILL_LOG
sleep 30  # Wait for some blocks
SIGNED=$(journalctl -u pawd-testnet --since "30 seconds ago" | grep -c "signed")
if [ $SIGNED -gt 0 ]; then
    echo "✓ Validator signing blocks" | tee -a $DRILL_LOG
else
    echo "✗ Validator not signing" | tee -a $DRILL_LOG
fi

# Summary
echo | tee -a $DRILL_LOG
echo "=== Recovery Drill Summary ===" | tee -a $DRILL_LOG
echo "Restore Duration: $RESTORE_DURATION seconds" | tee -a $DRILL_LOG
echo "Sync Duration: $SYNC_DURATION seconds" | tee -a $DRILL_LOG
echo "Total Recovery Time: $((SYNC_TIME - STOP_TIME)) seconds" | tee -a $DRILL_LOG
echo "Status: SUCCESS" | tee -a $DRILL_LOG
echo "End: $(date -u)" | tee -a $DRILL_LOG
```

---

## Failover Procedures

### Active-Passive Validator Failover

**Setup:** Hot standby validator ready to take over

#### Prerequisites

- Secondary validator infrastructure provisioned
- Syncing chain data continuously
- Validator keys in secure backup (NOT loaded on standby)
- Automated monitoring and alerting

#### Failover Trigger Conditions

Automatic failover if primary validator:

- Offline for >5 minutes
- Missing >10 consecutive blocks
- Hardware failure detected
- Network unreachable

#### Failover Procedure

```bash
#!/bin/bash
# automatic-failover.sh
# Run on monitoring server

PRIMARY_RPC="http://primary-validator:26657"
STANDBY_HOST="standby-validator"
ALERT_WEBHOOK="https://hooks.slack.com/services/YOUR/WEBHOOK"
FAILOVER_THRESHOLD=10  # Missed blocks before failover

while true; do
    # Check primary health
    STATUS=$(curl -s --max-time 5 "$PRIMARY_RPC/status" 2>/dev/null)

    if [ $? -ne 0 ]; then
        # Primary unreachable
        echo "Primary validator unreachable"

        # Alert team
        curl -X POST $ALERT_WEBHOOK -H 'Content-Type: application/json' \
            -d '{"text":"PRIMARY VALIDATOR UNREACHABLE - Initiating failover"}'

        # Trigger failover
        ssh $STANDBY_HOST '/usr/local/bin/activate-validator.sh'

        # Wait before checking again
        sleep 300
    else
        # Check if signing
        LATEST_BLOCK_TIME=$(echo $STATUS | jq -r '.result.sync_info.latest_block_time')
        BLOCK_AGE=$(( $(date +%s) - $(date -d "$LATEST_BLOCK_TIME" +%s) ))

        if [ $BLOCK_AGE -gt 60 ]; then
            echo "Primary validator not signing (last block: ${BLOCK_AGE}s ago)"

            # Alert
            curl -X POST $ALERT_WEBHOOK -H 'Content-Type: application/json' \
                -d "{\"text\":\"Primary validator stalled - Initiating failover\"}"

            # Trigger failover
            ssh $STANDBY_HOST '/usr/local/bin/activate-validator.sh'

            sleep 300
        fi
    fi

    sleep 10
done
```

#### Standby Activation Script

```bash
#!/bin/bash
# activate-validator.sh
# Run on standby validator

set -e

LOCKFILE="/var/lock/validator-failover"
NODE_HOME="/home/paw/.paw"

# Prevent concurrent execution
if [ -f "$LOCKFILE" ]; then
    echo "Failover already in progress"
    exit 1
fi

touch "$LOCKFILE"

echo "=== VALIDATOR FAILOVER INITIATED ==="
echo "Time: $(date -u)"

# 1. Verify primary is really down (safety check)
echo "1. Verifying primary failure..."
if timeout 10 curl -s http://primary-validator:26657/status > /dev/null; then
    echo "WARNING: Primary appears to be online. Aborting failover."
    rm "$LOCKFILE"
    exit 1
fi

# 2. Ensure standby is synced
echo "2. Checking standby sync status..."
CATCHING_UP=$(curl -s http://localhost:26657/status | jq -r '.result.sync_info.catching_up')
if [ "$CATCHING_UP" != "false" ]; then
    echo "ERROR: Standby not fully synced. Cannot failover."
    rm "$LOCKFILE"
    exit 1
fi

# 3. Stop standby node
echo "3. Stopping standby node..."
systemctl stop pawd

# 4. Load validator keys from secure backup
echo "4. Loading validator keys..."
# This should prompt for passphrase or use HSM
gpg --decrypt /secure/backup/validator_keys.tar.gz.gpg | \
    tar xz -C /tmp/

# Verify checksums
cd /tmp/validator_keys_*
sha256sum -c checksums.txt || exit 1

# Install keys
cp priv_validator_key.json "$NODE_HOME/config/"
cp priv_validator_state.json "$NODE_HOME/data/"
chmod 600 "$NODE_HOME/config/priv_validator_key.json"
chmod 600 "$NODE_HOME/data/priv_validator_state.json"

# Secure cleanup
shred -vfz -n 10 /tmp/validator_keys_*/*
rm -rf /tmp/validator_keys_*

# 5. Start validator
echo "5. Starting validator..."
systemctl start pawd

# 6. Monitor activation
echo "6. Monitoring activation..."
sleep 30  # Wait for participation

# Check if signing
SIGNED=$(journalctl -u pawd --since "30 seconds ago" | grep -c "signed" || true)
if [ $SIGNED -gt 0 ]; then
    echo "✓ Validator activated and signing"

    # Alert success
    curl -X POST $ALERT_WEBHOOK -H 'Content-Type: application/json' \
        -d '{"text":"✓ Standby validator activated successfully"}'
else
    echo "✗ Validator not signing yet"

    # Alert potential issue
    curl -X POST $ALERT_WEBHOOK -H 'Content-Type: application/json' \
        -d '{"text":"⚠ Standby activated but not signing - investigate"}'
fi

rm "$LOCKFILE"
echo "=== FAILOVER COMPLETE ==="
```

### Geographic Failover

For regional outages (e.g., datacenter failure):

```bash
# Geographic redundancy setup
# Primary: US-East
# Secondary: EU-West
# Tertiary: Asia-Pacific

# Automated DNS failover using health checks
# Route53 / CloudFlare health checks monitor:
# - RPC endpoint availability
# - Recent block height
# - Validator signing status

# If primary region fails:
# 1. DNS automatically routes to secondary region
# 2. Monitoring triggers failover activation
# 3. Secondary validator loads keys and activates
# 4. Team notified of failover
# 5. Primary region recovered in background
```

---

## Testing & Drills

### Quarterly Recovery Drill Schedule

**Q1 - January 15:**

- Test: Single validator recovery
- Environment: Testnet
- Duration: 4 hours
- Participants: All operations team

**Q2 - April 15:**

- Test: Multi-validator coordination
- Environment: Devnet
- Duration: 6 hours
- Participants: Ops team + validator partners

**Q3 - July 15:**

- Test: Complete network recovery
- Environment: Dedicated test environment
- Duration: 8 hours
- Participants: Full team

**Q4 - October 15:**

- Test: Failover procedures
- Environment: Testnet
- Duration: 4 hours
- Participants: Ops team

### Drill Execution Checklist

- [ ] Schedule drill 2 weeks in advance
- [ ] Notify all participants
- [ ] Prepare test environment
- [ ] Document baseline metrics
- [ ] Execute drill scenario
- [ ] Monitor and document issues
- [ ] Time all recovery steps
- [ ] Hold post-drill review
- [ ] Update procedures based on learnings
- [ ] Share results with team

### Drill Metrics to Track

| Metric                  | Target   | Actual | Status |
| ----------------------- | -------- | ------ | ------ |
| Time to detect failure  | < 5 min  |        |        |
| Time to coordinate team | < 15 min |        |        |
| Time to begin recovery  | < 30 min |        |        |
| Time to restore service | < 2 hrs  |        |        |
| Backup restore time     | < 30 min |        |        |
| Full sync time          | < 90 min |        |        |
| Communication clarity   | 9/10     |        |        |
| Procedure accuracy      | 100%     |        |        |

---

## Document Control

**Version History:**

| Version | Date       | Author              | Changes         |
| ------- | ---------- | ------------------- | --------------- |
| 1.0     | 2025-11-14 | Infrastructure Team | Initial release |

**Review Schedule:** Semi-annually or after major incidents

**Next Review Date:** 2026-05-14

**Approval:**

- CTO: ********\_******** Date: ****\_****
- Infrastructure Lead: ********\_******** Date: ****\_****

**Related Documents:**

- INCIDENT_RESPONSE_PLAN.md
- SECURITY_RUNBOOK.md
- MONITORING_QUICKREF.md
- Validator Operations Guide

---

**IMPORTANT:** This is a living document. After each recovery drill or actual incident, update procedures based on lessons learned. Regular testing ensures procedures remain effective and team remains prepared.
