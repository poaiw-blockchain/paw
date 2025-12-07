# PAW Blockchain Upgrade Procedures

Comprehensive guide for planning, executing, and verifying blockchain upgrades for validators, node operators, and developers on the PAW network.

## Table of Contents

1. [Overview](#overview)
2. [Types of Upgrades](#types-of-upgrades)
3. [Pre-Upgrade Checklist](#pre-upgrade-checklist)
4. [Governance-Based Upgrades](#governance-based-upgrades)
5. [Manual Upgrade Procedure](#manual-upgrade-procedure)
6. [Cosmovisor Automatic Upgrades](#cosmovisor-automatic-upgrades)
7. [Emergency Upgrades](#emergency-upgrades)
8. [Rollback Procedures](#rollback-procedures)
9. [Post-Upgrade Verification](#post-upgrade-verification)
10. [Troubleshooting](#troubleshooting)

---

## Overview

### Upgrade Philosophy

The PAW blockchain upgrade process prioritizes:

1. **Safety First**: Comprehensive testing and validation before production deployment
2. **Transparency**: Clear communication with validators and the community
3. **Coordination**: Synchronized upgrades across the validator set
4. **Reversibility**: Well-documented rollback procedures for critical failures
5. **Minimal Downtime**: Optimized procedures for rapid execution

### Upgrade Coordination

Blockchain upgrades require consensus among validators and careful coordination to ensure network continuity. All validators must upgrade their binaries at the same block height to maintain consensus.

**Coordination Channels:**
- **Validator Discord**: Real-time coordination during upgrades
- **Governance Forum**: Proposal discussion and planning
- **Telegram**: Community announcements
- **Twitter**: Public status updates
- **Email**: Critical security notifications

### Key Terminology

| Term | Definition |
|------|------------|
| **Consensus-Breaking** | Changes that affect consensus requiring all validators to upgrade |
| **Non-Consensus** | Changes that don't affect consensus (bug fixes, optimizations) |
| **Upgrade Height** | Specific block height where the upgrade activates |
| **State Migration** | Process of transforming blockchain state during an upgrade |
| **Cosmovisor** | Automatic upgrade management tool for Cosmos SDK chains |
| **Genesis Export** | Snapshot of blockchain state at a specific height |
| **Halt Height** | Block height where the chain stops to perform upgrade |

---

## Types of Upgrades

### 1. Soft Fork (Non-Consensus Upgrades)

**Characteristics:**
- Does not affect consensus rules
- Backward compatible with previous versions
- Validators can upgrade gradually

**Examples:**
- Performance optimizations
- Bug fixes that don't affect state
- API improvements
- Logging enhancements
- Documentation updates

**Process:**
```
Developer Release → Validator Notification → Gradual Rollout → Monitoring
```

**No governance proposal required** - validators upgrade at their convenience.

**Upgrade Procedure:**
```bash
# Download new binary
wget https://github.com/decristofaroj/paw/releases/download/v1.0.1/pawd

# Verify checksum
sha256sum pawd
# Compare with published checksum

# Stop node
sudo systemctl stop pawd

# Replace binary
sudo cp pawd /usr/local/bin/pawd
sudo chmod +x /usr/local/bin/pawd

# Verify version
pawd version

# Restart node
sudo systemctl start pawd

# Monitor logs
journalctl -u pawd -f
```

---

### 2. Hard Fork (Consensus-Breaking Upgrades)

**Characteristics:**
- Changes consensus rules or state structure
- NOT backward compatible
- Requires ALL validators to upgrade simultaneously
- Requires governance proposal approval

**Examples:**
- Module consensus version changes
- Store structure modifications
- Transaction format changes
- Consensus parameter adjustments
- New module additions
- State machine logic changes

**Process:**
```
Development → Testing → Governance Proposal → Voting → Coordinated Upgrade → Verification
```

**Governance approval required** (50% yes votes, 33.4% quorum).

**Timeline:**
- Proposal submission: Day 0
- Discussion period: Days 0-7
- Voting period: Days 0-7
- Preparation period: Days 7-14
- Upgrade execution: Day 14 (approximately)

---

### 3. State Migration Upgrade

**Characteristics:**
- Transforms blockchain state during upgrade
- Includes migration logic to update data structures
- May involve data cleanup or reorganization
- Requires careful testing and validation

**Examples:**
- Changing data models
- Adding new state fields
- Removing deprecated state
- Recalculating derived values
- Reorganizing store keys

**Migration Code Structure:**
```go
// Example migration from v1 to v2
func (m Migrator) Migrate1to2(ctx sdk.Context) error {
    // Iterate over existing state
    store := ctx.KVStore(m.keeper.storeKey)
    iterator := sdk.KVStorePrefixIterator(store, []byte{})
    defer iterator.Close()

    for ; iterator.Valid(); iterator.Next() {
        var oldData OldDataStructure
        m.cdc.MustUnmarshal(iterator.Value(), &oldData)

        // Transform to new structure
        newData := NewDataStructure{
            ExistingField: oldData.ExistingField,
            NewField:      calculateNewField(oldData),
        }

        // Save new structure
        store.Set(iterator.Key(), m.cdc.MustMarshal(&newData))
    }

    return nil
}
```

---

## Pre-Upgrade Checklist

### For Validators (1-2 Weeks Before Upgrade)

#### System Preparation

- [ ] **Review upgrade documentation** - Read upgrade guide thoroughly
- [ ] **Verify hardware requirements** - Ensure sufficient resources
  - CPU: Check if additional cores needed
  - RAM: Verify memory requirements
  - Disk: Ensure 2x current state size free (for backups)
  - Network: Test bandwidth and latency
- [ ] **Test upgrade on local node** - Simulate upgrade locally
- [ ] **Join validator coordination channel** - Discord/Telegram
- [ ] **Schedule maintenance window** - Coordinate with delegators
- [ ] **Notify delegators** - Announce planned downtime

#### Backup Strategy

- [ ] **Backup validator keys**
  ```bash
  # Backup priv_validator_key.json
  cp ~/.paw/config/priv_validator_key.json ~/paw-backup/

  # Encrypt backup
  gpg -c ~/paw-backup/priv_validator_key.json

  # Store in secure location (encrypted USB, password manager)
  ```

- [ ] **Export blockchain state**
  ```bash
  # Export genesis at current height
  pawd export > pre-upgrade-state-$(date +%Y%m%d).json

  # Verify export integrity
  jq '.app_state' pre-upgrade-state-*.json > /dev/null && echo "Valid" || echo "Invalid"

  # Compress for storage
  gzip pre-upgrade-state-*.json

  # Calculate checksum
  sha256sum pre-upgrade-state-*.json.gz
  ```

- [ ] **Backup database (optional but recommended)**
  ```bash
  # Stop node for consistent backup
  sudo systemctl stop pawd

  # Backup entire data directory
  tar -czf paw-data-backup-$(date +%Y%m%d).tar.gz ~/.paw/data/

  # Restart node
  sudo systemctl start pawd

  # Store backup on separate disk/system
  rsync -avz paw-data-backup-*.tar.gz backup-server:/backups/
  ```

- [ ] **Backup configuration**
  ```bash
  # Backup all config files
  tar -czf paw-config-backup-$(date +%Y%m%d).tar.gz ~/.paw/config/
  ```

#### Binary Preparation

- [ ] **Download new binary from official release**
  ```bash
  # Download from GitHub releases
  wget https://github.com/decristofaroj/paw/releases/download/v1.1.0/pawd-v1.1.0-linux-amd64

  # Download checksum
  wget https://github.com/decristofaroj/paw/releases/download/v1.1.0/SHA256SUMS
  ```

- [ ] **Verify binary checksum**
  ```bash
  # Verify SHA256
  sha256sum -c SHA256SUMS 2>&1 | grep pawd-v1.1.0-linux-amd64
  # Should output: pawd-v1.1.0-linux-amd64: OK

  # Alternative manual check
  sha256sum pawd-v1.1.0-linux-amd64
  # Compare with published checksum
  ```

- [ ] **Verify binary signature (if provided)**
  ```bash
  # Import signing key
  gpg --keyserver keyserver.ubuntu.com --recv-keys SIGNING_KEY_ID

  # Verify signature
  gpg --verify pawd-v1.1.0-linux-amd64.asc pawd-v1.1.0-linux-amd64
  ```

- [ ] **Test new binary**
  ```bash
  # Make executable
  chmod +x pawd-v1.1.0-linux-amd64

  # Check version
  ./pawd-v1.1.0-linux-amd64 version
  # Should output: v1.1.0

  # Test help command
  ./pawd-v1.1.0-linux-amd64 --help
  ```

#### Cosmovisor Setup (Recommended)

- [ ] **Install Cosmovisor** (if not already installed)
  ```bash
  go install cosmossdk.io/tools/cosmovisor/cmd/cosmovisor@latest

  # Verify installation
  cosmovisor version
  ```

- [ ] **Configure Cosmovisor directory structure**
  ```bash
  # Setup environment variables
  export DAEMON_HOME=$HOME/.paw
  export DAEMON_NAME=pawd

  # Create directory structure
  mkdir -p $DAEMON_HOME/cosmovisor/genesis/bin
  mkdir -p $DAEMON_HOME/cosmovisor/upgrades/v1.1.0/bin

  # Copy current binary to genesis
  cp $(which pawd) $DAEMON_HOME/cosmovisor/genesis/bin/

  # Copy new binary to upgrade directory
  cp pawd-v1.1.0-linux-amd64 $DAEMON_HOME/cosmovisor/upgrades/v1.1.0/bin/pawd
  chmod +x $DAEMON_HOME/cosmovisor/upgrades/v1.1.0/bin/pawd
  ```

- [ ] **Verify Cosmovisor configuration**
  ```bash
  # List cosmovisor directories
  tree $DAEMON_HOME/cosmovisor

  # Should show:
  # cosmovisor/
  # ├── genesis/
  # │   └── bin/
  # │       └── pawd
  # └── upgrades/
  #     └── v1.1.0/
  #         └── bin/
  #             └── pawd
  ```

#### Monitoring Setup

- [ ] **Ensure monitoring is operational**
  - Prometheus metrics exporter running
  - Grafana dashboards configured
  - Alert rules enabled

- [ ] **Setup upgrade-specific alerts**
  ```yaml
  # Add to prometheus alerts
  - alert: UpgradeHeight
    expr: tendermint_consensus_height > UPGRADE_HEIGHT - 100
    annotations:
      summary: "Approaching upgrade height"
  ```

- [ ] **Prepare monitoring checklist**
  - Block height tracking
  - Validator signing status
  - Peer connections
  - Memory/CPU usage
  - Disk I/O

---

### For Node Operators

- [ ] **Review upgrade guide** - Understand changes
- [ ] **Backup node data** - State export and database backup
- [ ] **Update monitoring dashboards** - Add upgrade metrics
- [ ] **Schedule maintenance window** - Coordinate with API users
- [ ] **Communicate downtime** - Notify dependent services
- [ ] **Prepare rollback plan** - Document rollback steps
- [ ] **Test RPC endpoints** - Verify endpoints after upgrade

---

### For Developers and Integrators

- [ ] **Review API changes** - Check for breaking changes
- [ ] **Update client libraries** - Upgrade SDK versions
- [ ] **Test against upgraded testnet** - Validate integrations
- [ ] **Update application code** - Handle new features/changes
- [ ] **Prepare user communications** - Notify users of downtime
- [ ] **Review migration guides** - Understand data structure changes
- [ ] **Update documentation** - Reflect new API versions

---

## Governance-Based Upgrades

### Upgrade Proposal Process

#### 1. Proposal Submission

```bash
# Prepare upgrade info JSON with binary download URLs
cat > upgrade-info.json <<EOF
{
  "binaries": {
    "linux/amd64": "https://github.com/decristofaroj/paw/releases/download/v1.1.0/pawd-v1.1.0-linux-amd64?checksum=sha256:CHECKSUM_HERE",
    "linux/arm64": "https://github.com/decristofaroj/paw/releases/download/v1.1.0/pawd-v1.1.0-linux-arm64?checksum=sha256:CHECKSUM_HERE",
    "darwin/amd64": "https://github.com/decristofaroj/paw/releases/download/v1.1.0/pawd-v1.1.0-darwin-amd64?checksum=sha256:CHECKSUM_HERE"
  }
}
EOF

# Submit software upgrade proposal
pawd tx gov submit-proposal software-upgrade v1.1.0 \
  --title "PAW v1.1.0 Upgrade: Enhanced Security and State Validation" \
  --description "$(cat upgrade-description.md)" \
  --upgrade-height 1500000 \
  --upgrade-info "$(cat upgrade-info.json)" \
  --deposit 10000000upaw \
  --from validator \
  --chain-id paw-1 \
  --gas auto \
  --gas-adjustment 1.3 \
  --yes

# Note the proposal ID from output
```

**Proposal Description Template** (`upgrade-description.md`):
```markdown
# PAW v1.1.0 Upgrade Proposal

## Summary
This proposal seeks approval for the v1.1.0 upgrade of the PAW blockchain, introducing critical security enhancements and state validation mechanisms.

## Changes
- Enhanced security validation across all modules
- Automated state consistency checks
- Improved oracle price feed validation
- DEX pool invariant enforcement
- Compute module rate limiting

## Upgrade Details
- **Upgrade Name**: v1.1.0
- **Upgrade Height**: 1,500,000 (approximately 2025-12-20 14:00 UTC)
- **Estimated Downtime**: 15-30 minutes
- **Breaking Changes**: None (backward-compatible migration)

## Documentation
- Upgrade Guide: https://docs.paw.com/upgrades/v1.1.0
- Testing Results: https://github.com/decristofaroj/paw/releases/tag/v1.1.0
- Rollback Procedure: https://docs.paw.com/upgrades/ROLLBACK.md

## Timeline
- Proposal submission: 2025-12-05
- Discussion period: 2025-12-05 to 2025-12-12 (7 days)
- Voting period: 2025-12-05 to 2025-12-12 (7 days)
- Preparation period: 2025-12-12 to 2025-12-20 (8 days)
- Upgrade execution: 2025-12-20 at height 1,500,000

## Testing
The upgrade has been tested on:
- Local development networks (100+ upgrade simulations)
- Public testnet (paw-testnet-1, upgraded at height 500,000)
- Load testing (validated with 1000 TPS)

Vote YES to approve this upgrade.
```

#### 2. Discussion Period (7 Days)

**For Proposers:**
- Monitor community feedback on governance forum
- Answer questions in Discord/Telegram
- Address concerns raised by validators
- Update proposal details if needed (via forum posts)

**For Validators:**
- Review technical changes
- Test upgrade on local testnet
- Discuss concerns with development team
- Evaluate impact on operations

#### 3. Voting Period (7 Days)

```bash
# Query proposal details
pawd query gov proposal <proposal-id>

# Query current vote tally
pawd query gov tally <proposal-id>

# View your vote (if already voted)
pawd query gov vote <proposal-id> $(pawd keys show validator -a)

# Cast your vote
pawd tx gov vote <proposal-id> yes \
  --from validator \
  --chain-id paw-1 \
  --gas auto \
  --yes

# Vote options: yes, no, abstain, no_with_veto
```

**Voting Thresholds:**
- **Quorum**: 33.4% of bonded tokens must vote
- **Pass Threshold**: 50% of votes must be "yes"
- **Veto Threshold**: Less than 33.4% "no_with_veto"

#### 4. Proposal Outcome

```bash
# Check if proposal passed
pawd query gov proposal <proposal-id> | jq '.status'

# Possible statuses:
# - PROPOSAL_STATUS_VOTING_PERIOD: Still in voting
# - PROPOSAL_STATUS_PASSED: Approved, upgrade will execute
# - PROPOSAL_STATUS_REJECTED: Not enough yes votes
# - PROPOSAL_STATUS_FAILED: Did not meet quorum
```

**If Passed:**
- Upgrade will automatically activate at specified height
- Validators must prepare binaries before upgrade height
- Chain will halt at upgrade height for binary swap

**If Rejected:**
- No upgrade occurs
- Proposers can submit revised proposal
- Deposit may be returned or burned (depending on veto votes)

---

## Manual Upgrade Procedure

For validators **not using Cosmovisor**, follow this manual procedure:

### Step 1: Monitor Upgrade Height

```bash
# Query scheduled upgrade plan
pawd query upgrade plan

# Output will show:
# name: v1.1.0
# height: 1500000
# info: {...}

# Monitor current block height
watch -n 5 'pawd status | jq ".SyncInfo.latest_block_height"'

# Calculate time to upgrade (6-second blocks)
# Time = (upgrade_height - current_height) * 6 seconds
```

**Set up alerts** to notify you when approaching upgrade height:
```bash
# Simple alert script
UPGRADE_HEIGHT=1500000
while true; do
  CURRENT=$(pawd status | jq -r '.SyncInfo.latest_block_height')
  REMAINING=$((UPGRADE_HEIGHT - CURRENT))

  if [ $REMAINING -lt 100 ]; then
    echo "ALERT: Only $REMAINING blocks until upgrade!"
    # Send notification (email, Telegram, etc.)
  fi

  sleep 30
done
```

---

### Step 2: Chain Halts at Upgrade Height

**What Happens:**
- At block height 1,500,000, the chain will automatically halt
- Validators will log: `ERR UPGRADE "v1.1.0" NEEDED at height: 1500000`
- No new blocks will be produced until validators upgrade

**Verify Halt:**
```bash
# Check node logs
journalctl -u pawd -f | grep UPGRADE

# Example log output:
# ERR UPGRADE "v1.1.0" NEEDED at height: 1500000
# Module=main module=consensus

# Check status (will show chain is halted)
pawd status | jq '.SyncInfo'
```

**Expected Behavior:**
- Node process continues running but stops producing blocks
- RPC endpoints may return errors
- Sync status shows catching_up = false at halt height

---

### Step 3: Stop Node

```bash
# Stop the validator node
sudo systemctl stop pawd

# Verify process stopped
ps aux | grep pawd
# Should return no results

# Check last block height
pawd status --node tcp://localhost:26657 2>&1 || echo "Node stopped"
```

**Alternative Stop Methods:**

If systemd service stop hangs:
```bash
# Find PID
PID=$(pgrep pawd)

# Graceful stop with SIGTERM
kill -TERM $PID

# Wait 10 seconds
sleep 10

# Force kill if still running
if ps -p $PID > /dev/null; then
  kill -9 $PID
fi
```

---

### Step 4: Backup Current State

**Critical: Always backup before replacing binary**

```bash
# Backup current binary
cp $(which pawd) ~/pawd-backup-$(date +%Y%m%d)

# Backup data directory (optional but recommended)
tar -czf ~/.paw-data-backup-$(date +%Y%m%d).tar.gz ~/.paw/data/

# Export state at halt height
pawd export --height 1500000 > state-height-1500000.json

# Verify export
jq '.app_state' state-height-1500000.json > /dev/null && echo "Valid" || echo "Invalid"

# Compress state export
gzip state-height-1500000.json
```

---

### Step 5: Install New Binary

```bash
# Verify new binary is ready
ls -lh pawd-v1.1.0-linux-amd64
chmod +x pawd-v1.1.0-linux-amd64

# Verify checksum one more time
sha256sum pawd-v1.1.0-linux-amd64

# Replace system binary
sudo cp pawd-v1.1.0-linux-amd64 /usr/local/bin/pawd

# Verify installation
pawd version
# Should output: v1.1.0

# Verify binary works
pawd --help
```

**Binary Installation Locations:**

| Installation Method | Binary Path |
|---------------------|-------------|
| System-wide (recommended) | `/usr/local/bin/pawd` |
| User-local | `~/go/bin/pawd` |
| Systemd service | Path specified in systemd unit file |

**Verify Systemd Service Configuration:**
```bash
# Check which binary systemd uses
sudo systemctl cat pawd | grep ExecStart

# If using hardcoded path, update service file
sudo nano /etc/systemd/system/pawd.service

# Reload systemd
sudo systemctl daemon-reload
```

---

### Step 6: Start Node with New Binary

```bash
# Start the node
sudo systemctl start pawd

# Monitor startup logs
journalctl -u pawd -f

# Watch for successful migration messages:
# - "applying upgrade v1.1.0 at height 1500000"
# - "running migrations for module dex from 1 to 2"
# - "running migrations for module oracle from 1 to 2"
# - "successfully migrated module dex to version 2"
# - "upgrade v1.1.0 applied successfully"
```

**Expected Startup Sequence:**
1. Load chain state from disk
2. Detect upgrade plan at halt height
3. Execute migration handlers
4. Update consensus versions
5. Resume block production
6. Sync with network

**Startup Time:**
- Small chains: 1-5 minutes
- Large chains: 5-15 minutes
- Migration complexity may add 5-10 minutes

---

### Step 7: Verify Upgrade Success

```bash
# Check node is running
sudo systemctl status pawd

# Verify version
pawd version
# Should output: v1.1.0

# Check sync status
pawd status | jq '.SyncInfo'

# Verify new blocks are being produced
watch -n 2 'pawd status | jq ".SyncInfo.latest_block_height"'
# Height should increment every 6 seconds

# Check validator signing status
pawd query slashing signing-info $(pawd tendermint show-validator)

# Verify validator is in active set
pawd query staking validator $(pawd keys show validator --bech val -a) | jq '.status'
# Should output: "BOND_STATUS_BONDED"
```

**Health Check Script:**
```bash
#!/bin/bash
# upgrade-health-check.sh

echo "=== PAW Node Health Check ==="

# Check version
VERSION=$(pawd version)
echo "Version: $VERSION"

# Check sync status
SYNC_INFO=$(pawd status | jq '.SyncInfo')
HEIGHT=$(echo $SYNC_INFO | jq -r '.latest_block_height')
CATCHING_UP=$(echo $SYNC_INFO | jq -r '.catching_up')

echo "Block Height: $HEIGHT"
echo "Catching Up: $CATCHING_UP"

# Check validator status
VAL_ADDR=$(pawd keys show validator --bech val -a 2>/dev/null)
if [ -n "$VAL_ADDR" ]; then
  VAL_STATUS=$(pawd query staking validator $VAL_ADDR | jq -r '.status')
  echo "Validator Status: $VAL_STATUS"

  # Check signing info
  CONS_PUBKEY=$(pawd tendermint show-validator)
  SIGNING_INFO=$(pawd query slashing signing-info $CONS_PUBKEY)
  MISSED_BLOCKS=$(echo $SIGNING_INFO | jq -r '.missed_blocks_counter')
  echo "Missed Blocks: $MISSED_BLOCKS"
fi

# Check peers
PEERS=$(pawd status | jq -r '.SyncInfo.peer_count // .NodeInfo.peers // 0')
echo "Peer Count: $PEERS"

echo "=== Health Check Complete ==="
```

---

## Cosmovisor Automatic Upgrades

### Overview

**Cosmovisor** is a process manager for Cosmos SDK binaries that automates upgrades. It monitors for governance-approved upgrade plans and automatically swaps binaries at the upgrade height.

**Benefits:**
- Automatic binary switching at upgrade height
- No manual intervention required during upgrade
- Reduced risk of missing upgrade window
- Automatic backup of previous binaries
- Support for auto-download of new binaries (optional)

**How It Works:**
```
1. Cosmovisor monitors for upgrade plan in state
2. At upgrade height, chain halts
3. Cosmovisor detects upgrade name (e.g., "v1.1.0")
4. Cosmovisor switches symlink to new binary
5. Cosmovisor restarts node with new binary
6. Migration executes automatically
7. Chain resumes block production
```

---

### Initial Cosmovisor Setup

#### 1. Install Cosmovisor

```bash
# Install latest version
go install cosmossdk.io/tools/cosmovisor/cmd/cosmovisor@latest

# Verify installation
cosmovisor version

# Add to PATH if needed
echo 'export PATH=$PATH:$(go env GOPATH)/bin' >> ~/.bashrc
source ~/.bashrc
```

#### 2. Configure Environment Variables

```bash
# Add to ~/.bashrc or ~/.profile
cat >> ~/.bashrc <<EOF

# Cosmovisor Configuration
export DAEMON_NAME=pawd
export DAEMON_HOME=\$HOME/.paw
export DAEMON_ALLOW_DOWNLOAD_BINARIES=false  # Set to true for auto-download
export DAEMON_RESTART_AFTER_UPGRADE=true
export UNSAFE_SKIP_BACKUP=false  # Keep backups of previous versions
EOF

# Reload environment
source ~/.bashrc
```

**Environment Variable Reference:**

| Variable | Default | Description |
|----------|---------|-------------|
| `DAEMON_NAME` | (required) | Name of the binary (e.g., "pawd") |
| `DAEMON_HOME` | (required) | Home directory for the daemon (e.g., "$HOME/.paw") |
| `DAEMON_ALLOW_DOWNLOAD_BINARIES` | `false` | Auto-download binaries from upgrade proposal |
| `DAEMON_RESTART_AFTER_UPGRADE` | `true` | Auto-restart after upgrade completes |
| `DAEMON_POLL_INTERVAL` | `300ms` | How often to check for upgrades |
| `UNSAFE_SKIP_BACKUP` | `false` | Skip automatic backup of data directory |
| `DAEMON_PREUPGRADE_MAX_RETRIES` | `0` | Retries for pre-upgrade script |

#### 3. Setup Directory Structure

```bash
# Create cosmovisor directories
mkdir -p $DAEMON_HOME/cosmovisor/genesis/bin
mkdir -p $DAEMON_HOME/cosmovisor/upgrades

# Copy current binary to genesis
cp $(which pawd) $DAEMON_HOME/cosmovisor/genesis/bin/

# Verify structure
tree $DAEMON_HOME/cosmovisor
# Expected output:
# cosmovisor/
# ├── genesis/
# │   └── bin/
# │       └── pawd
# └── upgrades/
```

#### 4. Create Systemd Service for Cosmovisor

```bash
# Create systemd service file
sudo tee /etc/systemd/system/cosmovisor-pawd.service > /dev/null <<EOF
[Unit]
Description=Cosmovisor PAW Daemon
After=network-online.target

[Service]
User=$USER
ExecStart=$(which cosmovisor) run start
Restart=always
RestartSec=3
LimitNOFILE=65535
Environment="DAEMON_NAME=pawd"
Environment="DAEMON_HOME=$HOME/.paw"
Environment="DAEMON_ALLOW_DOWNLOAD_BINARIES=false"
Environment="DAEMON_RESTART_AFTER_UPGRADE=true"
Environment="UNSAFE_SKIP_BACKUP=false"

[Install]
WantedBy=multi-user.target
EOF

# Reload systemd
sudo systemctl daemon-reload

# Enable service
sudo systemctl enable cosmovisor-pawd

# Start service
sudo systemctl start cosmovisor-pawd

# Check status
sudo systemctl status cosmovisor-pawd
```

---

### Preparing for an Upgrade with Cosmovisor

#### 1. Create Upgrade Directory

```bash
# For upgrade named "v1.1.0"
mkdir -p $DAEMON_HOME/cosmovisor/upgrades/v1.1.0/bin
```

**Note:** Upgrade name MUST match exactly what's in the governance proposal.

#### 2. Add New Binary

**Option A: Manual Placement**

```bash
# Download new binary
wget https://github.com/decristofaroj/paw/releases/download/v1.1.0/pawd-v1.1.0-linux-amd64

# Verify checksum
sha256sum pawd-v1.1.0-linux-amd64

# Copy to upgrade directory
cp pawd-v1.1.0-linux-amd64 $DAEMON_HOME/cosmovisor/upgrades/v1.1.0/bin/pawd
chmod +x $DAEMON_HOME/cosmovisor/upgrades/v1.1.0/bin/pawd

# Verify binary
$DAEMON_HOME/cosmovisor/upgrades/v1.1.0/bin/pawd version
# Should output: v1.1.0
```

**Option B: Auto-Download (if enabled)**

If `DAEMON_ALLOW_DOWNLOAD_BINARIES=true`, Cosmovisor will automatically download the binary from the URL in the upgrade proposal.

```bash
# Verify auto-download configuration
echo $DAEMON_ALLOW_DOWNLOAD_BINARIES

# Monitor logs for download activity
journalctl -u cosmovisor-pawd -f | grep download
```

**Security Warning:** Auto-download is convenient but introduces risk. The binary will be downloaded from URLs in the governance proposal, which could be compromised. For production validators, manual verification is recommended.

#### 3. Verify Upgrade Preparation

```bash
# Check directory structure
tree $DAEMON_HOME/cosmovisor

# Should show:
# cosmovisor/
# ├── current -> genesis  (symlink)
# ├── genesis/
# │   └── bin/
# │       └── pawd
# └── upgrades/
#     └── v1.1.0/
#         └── bin/
#             └── pawd

# Verify current binary
$DAEMON_HOME/cosmovisor/current/bin/pawd version

# Verify upgrade binary
$DAEMON_HOME/cosmovisor/upgrades/v1.1.0/bin/pawd version
```

#### 4. Optional: Pre-Upgrade Script

Create a script to run before the upgrade (e.g., for state backup):

```bash
# Create pre-upgrade script
cat > $DAEMON_HOME/cosmovisor/upgrades/v1.1.0/pre-upgrade.sh <<'EOF'
#!/bin/bash
# Pre-upgrade backup script for v1.1.0

set -e

echo "Running pre-upgrade backup..."

# Export state
pawd export > $DAEMON_HOME/pre-upgrade-v1.1.0-state.json

# Backup database
tar -czf $DAEMON_HOME/pre-upgrade-v1.1.0-data.tar.gz $DAEMON_HOME/data/

echo "Pre-upgrade backup complete"
EOF

# Make executable
chmod +x $DAEMON_HOME/cosmovisor/upgrades/v1.1.0/pre-upgrade.sh
```

---

### Automatic Upgrade Execution

**During the Upgrade:**

1. **Cosmovisor monitors block height**
   ```
   No manual action needed - Cosmovisor watches chain state
   ```

2. **At upgrade height, chain halts**
   ```
   Cosmovisor detects: "UPGRADE v1.1.0 NEEDED at height: 1500000"
   ```

3. **Pre-upgrade script runs** (if configured)
   ```
   Executes: $DAEMON_HOME/cosmovisor/upgrades/v1.1.0/pre-upgrade.sh
   ```

4. **Cosmovisor switches binary**
   ```
   Updates symlink: current -> upgrades/v1.1.0
   ```

5. **Node restarts with new binary**
   ```
   Cosmovisor executes: cosmovisor run start
   ```

6. **Migration executes automatically**
   ```
   New binary applies migration handlers
   ```

7. **Chain resumes**
   ```
   Block production continues at height 1,500,001
   ```

**Monitor Upgrade Progress:**

```bash
# Watch Cosmovisor logs
journalctl -u cosmovisor-pawd -f

# Expected log sequence:
# - "Detected upgrade plan: v1.1.0 at height 1500000"
# - "Stopping current binary"
# - "Running pre-upgrade script"
# - "Switching to upgrade binary: v1.1.0"
# - "Restarting daemon"
# - "Upgrade v1.1.0 applied successfully"
```

---

### Post-Upgrade with Cosmovisor

```bash
# Verify Cosmovisor is using new binary
cosmovisor version
# or
$DAEMON_HOME/cosmovisor/current/bin/pawd version

# Check symlink points to upgrade
ls -la $DAEMON_HOME/cosmovisor/current
# Should show: current -> upgrades/v1.1.0

# Verify chain health
pawd status | jq '.SyncInfo'

# Check validator status
pawd query staking validator $(pawd keys show validator --bech val -a)
```

---

### Cosmovisor Troubleshooting

**Issue: Cosmovisor doesn't detect upgrade**

```bash
# Check upgrade plan is in state
pawd query upgrade plan

# Verify DAEMON_NAME matches
echo $DAEMON_NAME

# Check Cosmovisor logs
journalctl -u cosmovisor-pawd -n 100
```

**Issue: Binary not found at upgrade height**

```bash
# Verify upgrade directory exists
ls -la $DAEMON_HOME/cosmovisor/upgrades/v1.1.0/bin/pawd

# Check binary is executable
chmod +x $DAEMON_HOME/cosmovisor/upgrades/v1.1.0/bin/pawd

# Verify binary version
$DAEMON_HOME/cosmovisor/upgrades/v1.1.0/bin/pawd version
```

**Issue: Node doesn't restart after upgrade**

```bash
# Check DAEMON_RESTART_AFTER_UPGRADE is set
echo $DAEMON_RESTART_AFTER_UPGRADE

# Manually restart if needed
sudo systemctl restart cosmovisor-pawd

# Check for errors in logs
journalctl -u cosmovisor-pawd -n 50
```

---

## Emergency Upgrades

### When Emergency Upgrades Are Needed

Emergency upgrades bypass normal governance when there is an **immediate security threat** or **critical chain-halting bug**.

**Valid Emergency Scenarios:**
- Critical consensus bug causing chain halt
- Security vulnerability actively being exploited
- State corruption affecting network integrity
- Byzantine attack in progress

**Emergency Procedure:**

#### 1. Core Team Identifies Critical Issue

```bash
# Example: Critical security vulnerability discovered
# Core team prepares emergency patch release
```

#### 2. Validator Coordination (Off-Chain)

```
- Emergency announcement via Discord/Telegram
- Validator conference call if needed
- Explanation of threat and required action
- Distribution of emergency binary
```

#### 3. Coordinated Manual Upgrade

```bash
# All validators coordinate to stop at agreed height
# Example: Current height is 1,000,000, agree to stop at 1,000,100

# Stop node at coordinated height
# (Monitor block height and stop manually)

# Install emergency binary
sudo cp pawd-emergency-fix /usr/local/bin/pawd

# Verify checksum (provided by core team)
sha256sum /usr/local/bin/pawd

# Coordinate restart time
# All validators restart simultaneously at agreed time
sudo systemctl start pawd
```

#### 4. Post-Emergency Governance

```bash
# After emergency is resolved, submit retroactive governance proposal
# explaining the emergency and actions taken

pawd tx gov submit-proposal text \
  --title "Emergency Upgrade v1.0.1 Post-Mortem" \
  --description "$(cat emergency-explanation.md)" \
  --deposit 10000000upaw \
  --from validator \
  --yes
```

**Emergency Upgrade Example Timeline:**

| Time | Action |
|------|--------|
| T+0 | Security vulnerability discovered |
| T+1h | Core team prepares emergency fix |
| T+2h | Validators notified via Discord |
| T+3h | Emergency conference call |
| T+4h | Coordinated chain halt |
| T+4h 15m | Validators install emergency binary |
| T+4h 30m | Coordinated restart |
| T+5h | Chain resumes normal operation |
| T+24h | Retroactive governance proposal submitted |

**Security Considerations:**
- Emergency binary checksums must be verified
- Multiple core team members should sign off
- Public announcement after vulnerability is patched
- Incident report published for transparency

---

## Rollback Procedures

### When to Consider Rollback

Rollback should **only** be used in critical situations:

1. **Chain Halt**: Chain fails to produce blocks after upgrade
2. **Consensus Failure**: Validators cannot reach consensus
3. **State Corruption**: Upgrade causes data integrity issues
4. **Critical Bug**: Upgrade introduces severe bugs affecting operations
5. **Security Vulnerability**: New version introduces security risks

**DO NOT rollback for:**
- Minor bugs that don't affect consensus
- Individual validator configuration errors
- Temporary network connectivity issues
- Issues that can be patched quickly

---

### Rollback Prerequisites

Before initiating rollback:

- [ ] **Achieve validator consensus** - Minimum 67% voting power agreement
- [ ] **Establish coordination channel** - Discord/Telegram for real-time communication
- [ ] **Verify backups available** - All validators have pre-upgrade state backups
- [ ] **Document rollback reason** - Clear explanation for community
- [ ] **Set coordinated rollback time** - All validators rollback simultaneously

---

### Rollback Procedure

#### Phase 1: Preparation

```bash
# 1. Establish validator coordination
# - Create emergency Discord channel
# - Get confirmation from validators representing 67%+ voting power
# - Agree on rollback time (e.g., "in 30 minutes at 14:00 UTC")

# 2. Verify backup availability
ls -lh pre-upgrade-state-*.json
ls -lh ~/.paw-pre-upgrade-backup/

# 3. Verify backup integrity
jq '.app_state' pre-upgrade-state-*.json > /dev/null && echo "Valid" || echo "Invalid"
```

#### Phase 2: Coordinated Rollback

```bash
# At agreed time, ALL validators execute simultaneously:

# 1. Stop node
sudo systemctl stop pawd

# 2. Backup failed upgrade state (for analysis)
cp -r ~/.paw ~/.paw-failed-upgrade-backup-$(date +%Y%m%d-%H%M%S)

# 3. Restore previous binary
sudo cp ~/pawd-backup-* /usr/local/bin/pawd

# If using Cosmovisor:
rm $DAEMON_HOME/cosmovisor/current
ln -s $DAEMON_HOME/cosmovisor/genesis $DAEMON_HOME/cosmovisor/current

# Verify binary version
pawd version
# Should show previous version (e.g., v1.0.0)

# 4. Reset to pre-upgrade state
pawd tendermint unsafe-reset-all --home ~/.paw

# 5. Restore state from backup
gunzip pre-upgrade-state-*.json.gz
cp pre-upgrade-state-*.json ~/.paw/config/genesis.json

# Verify genesis hash matches other validators
jq -S -c '.app_state' ~/.paw/config/genesis.json | shasum -a 256

# 6. At coordinated restart time, all validators start:
sudo systemctl start pawd

# Monitor startup
journalctl -u pawd -f
```

#### Phase 3: Verification

```bash
# Verify chain is producing blocks
watch -n 2 'pawd status | jq ".SyncInfo.latest_block_height"'

# Verify validator is active
pawd query staking validator $(pawd keys show validator --bech val -a) | jq '.status'

# Verify state consistency
pawd export > post-rollback-state.json
diff <(jq -S '.app_state' pre-upgrade-state.json) \
     <(jq -S '.app_state' post-rollback-state.json)

# Test basic operations
pawd query bank balances $(pawd keys show -a validator)
```

**For detailed rollback procedures, see:** [`docs/upgrades/ROLLBACK.md`](./upgrades/ROLLBACK.md)

---

## Post-Upgrade Verification

### 1. Node Health Checks

```bash
# System health
pawd status | jq '.'

# Expected output fields:
# - NodeInfo.version: "v1.1.0"
# - SyncInfo.latest_block_height: Incrementing
# - SyncInfo.catching_up: false
# - ValidatorInfo.address: Your validator address
# - ValidatorInfo.voting_power: Your voting power

# Block production
watch -n 5 'pawd status | jq ".SyncInfo.latest_block_height"'
# Height should increase every ~6 seconds

# Peer connections
pawd status | jq '.NodeInfo.other.rpc_address, .SyncInfo'
pawd net_info | jq '.n_peers'
# Should have multiple peers (10-50+ typical)
```

---

### 2. Validator Status Verification

```bash
# Check validator is bonded and active
VAL_ADDR=$(pawd keys show validator --bech val -a)
pawd query staking validator $VAL_ADDR | jq '{
  status: .status,
  tokens: .tokens,
  delegator_shares: .delegator_shares,
  jailed: .jailed
}'

# Expected output:
# - status: "BOND_STATUS_BONDED"
# - tokens: Your total stake
# - jailed: false

# Check signing info
CONS_PUBKEY=$(pawd tendermint show-validator)
pawd query slashing signing-info $CONS_PUBKEY | jq '{
  address: .address,
  start_height: .start_height,
  missed_blocks_counter: .missed_blocks_counter,
  jailed_until: .jailed_until
}'

# missed_blocks_counter should be low (< 10)
# jailed_until should be "1970-01-01T00:00:00Z" (not jailed)

# Verify validator is signing recent blocks
pawd query block | jq '.block.last_commit.signatures[] | select(.validator_address != null)'
# Should see your validator address in signatures
```

---

### 3. Consensus Verification

```bash
# Check consensus state
pawd status | jq '.SyncInfo.latest_block_hash, .ValidatorInfo'

# Verify voting power matches expectation
pawd query staking validator $VAL_ADDR | jq '.tokens'
pawd query staking pool | jq '.bonded_tokens'

# Calculate voting power percentage
# voting_power_percentage = (validator_tokens / total_bonded_tokens) * 100

# Check if validator is in active set (top 150 validators)
pawd query staking validators --limit 150 | jq '.validators[] | select(.operator_address == "'$VAL_ADDR'")'
```

---

### 4. IBC Channel Verification (If Applicable)

```bash
# List all IBC channels
pawd query ibc channel channels

# Check specific channel status
pawd query ibc channel end transfer channel-0

# Expected state: "STATE_OPEN"

# Verify channel version matches upgrade
pawd query ibc channel end transfer channel-0 | jq '.version'

# Test IBC transfer (on testnet only)
pawd tx ibc-transfer transfer transfer channel-0 \
  cosmos1... 1000upaw \
  --from validator \
  --chain-id paw-1 \
  --yes

# Query pending packets
pawd query ibc channel unreceived-packets transfer channel-0
pawd query ibc channel unreceived-acks transfer channel-0
```

---

### 5. Module-Specific Verification

#### DEX Module

```bash
# Verify DEX pools are operational
pawd query dex list-pools

# Check specific pool state
pawd query dex pool 1

# Test swap query (doesn't execute)
pawd query dex estimate-swap 1 1000upaw tokenbtoken

# Check limit orders
pawd query dex list-limit-orders
```

#### Oracle Module

```bash
# Verify oracle price feeds
pawd query oracle list-prices

# Check specific price
pawd query oracle price usd:paw

# Verify validator price submissions
pawd query oracle validator-prices $VAL_ADDR

# Check oracle parameters
pawd query oracle params
```

#### Compute Module

```bash
# List registered providers
pawd query compute list-providers

# Check provider status
pawd query compute provider <provider-address>

# List compute requests
pawd query compute list-requests --status pending
```

---

### 6. Transaction Testing

```bash
# Test basic bank transfer
pawd tx bank send validator \
  $(pawd keys show -a test-account) \
  1000upaw \
  --from validator \
  --chain-id paw-1 \
  --gas auto \
  --yes

# Query transaction
TX_HASH=<hash-from-above>
pawd query tx $TX_HASH

# Test staking delegation
pawd tx staking delegate $VAL_ADDR 1000upaw \
  --from validator \
  --chain-id paw-1 \
  --gas auto \
  --yes

# Test governance vote (if proposal exists)
pawd tx gov vote 1 yes \
  --from validator \
  --chain-id paw-1 \
  --yes
```

---

### 7. Performance Metrics

```bash
# Check block time
# Average should be ~6 seconds
pawd status | jq '.SyncInfo.latest_block_time'

# Monitor transaction throughput
# Query prometheus metrics if configured
curl http://localhost:26660/metrics | grep tendermint_consensus

# Check memory usage
ps aux | grep pawd
# Validator node typically uses 2-8 GB RAM

# Check disk I/O
iostat -x 1 10

# Network bandwidth
iftop -i eth0
```

---

### 8. Log Analysis

```bash
# Check for errors in logs
journalctl -u pawd --since "1 hour ago" | grep -i error

# Check for warnings
journalctl -u pawd --since "1 hour ago" | grep -i warn

# Verify migration success
journalctl -u pawd | grep -i migration

# Expected log entries:
# - "applying upgrade v1.1.0 at height 1500000"
# - "running migrations for module dex from 1 to 2"
# - "successfully migrated module dex to version 2"
# - "upgrade v1.1.0 applied successfully"
```

---

### 9. API Endpoint Testing

```bash
# Test RPC endpoints
curl http://localhost:26657/status | jq '.'
curl http://localhost:26657/net_info | jq '.'
curl http://localhost:26657/validators | jq '.'

# Test API endpoints
curl http://localhost:1317/cosmos/staking/v1beta1/validators | jq '.'
curl http://localhost:1317/cosmos/bank/v1beta1/balances/$VAL_ADDR | jq '.'

# Test GRPC (using grpcurl)
grpcurl -plaintext localhost:9090 list
grpcurl -plaintext localhost:9090 cosmos.staking.v1beta1.Query/Validators
```

---

### 10. Security Validation

```bash
# Verify no exposed private keys
sudo find ~/.paw -name "priv_validator_key.json" -exec ls -la {} \;
# Should only be in ~/.paw/config/ with 600 permissions

# Check firewall rules
sudo ufw status
# Should block all ports except:
# - 26656 (P2P)
# - 26657 (RPC, if needed)
# - 1317 (API, if needed)
# - 9090 (gRPC, if needed)

# Verify node is not exposing sensitive endpoints
curl http://public-ip:26657/status 2>&1 | grep -i "refused" && echo "Good: RPC not exposed" || echo "WARNING: RPC exposed"

# Check validator key security
ls -la ~/.paw/config/priv_validator_key.json
# Should be: -rw------- (600 permissions)

# Verify no test keys in production
pawd keys list
# Should NOT include keys named "test", "alice", "bob", etc.
```

---

### Post-Upgrade Checklist

After completing verification:

- [ ] Node is producing blocks
- [ ] Validator is signing blocks (check recent blocks)
- [ ] Validator is in active set (bonded status)
- [ ] Missed blocks counter is low (< 10)
- [ ] Peer connections are healthy (> 10 peers)
- [ ] IBC channels are open (if applicable)
- [ ] Module queries return expected results
- [ ] Transaction submission works
- [ ] RPC/API endpoints functional
- [ ] Monitoring dashboards updated
- [ ] No errors in logs
- [ ] Delegators notified of successful upgrade
- [ ] Upgrade success reported to coordination channel

---

## Troubleshooting

### Common Upgrade Issues

#### Issue 1: Chain Halted but Node Won't Start with New Binary

**Symptoms:**
```
ERR UPGRADE "v1.1.0" NEEDED at height: 1500000
Node fails to start after binary replacement
```

**Diagnosis:**
```bash
# Check if binary is correct version
pawd version

# Check binary permissions
ls -la $(which pawd)

# Check upgrade info in state
pawd query upgrade plan
```

**Solutions:**

1. **Verify binary checksum**
   ```bash
   sha256sum $(which pawd)
   # Compare with official checksum
   ```

2. **Check binary compatibility**
   ```bash
   # Verify binary architecture
   file $(which pawd)
   # Should show: ELF 64-bit LSB executable, x86-64

   # Test binary
   pawd version
   pawd --help
   ```

3. **Review startup logs**
   ```bash
   journalctl -u pawd -n 100 --no-pager
   # Look for specific error messages
   ```

4. **Clear upgrade plan (if binary is wrong version)**
   ```bash
   # This requires manual state editing - DANGEROUS
   # Only do this if coordinated with other validators
   pawd unsafe-reset-all
   # Then restore from pre-upgrade backup
   ```

---

#### Issue 2: Migration Fails During Upgrade

**Symptoms:**
```
ERR migration failed for module dex from version 1 to 2
panic: migration error
```

**Diagnosis:**
```bash
# Check migration logs
journalctl -u pawd | grep -i migration

# Check state at upgrade height
pawd export --height 1500000 > state-debug.json
jq '.app_state.dex' state-debug.json
```

**Solutions:**

1. **Verify state integrity before upgrade**
   ```bash
   # Export and validate state
   pawd export --height 1499999 > pre-migration-state.json
   jq '.app_state' pre-migration-state.json > /dev/null && echo "Valid"
   ```

2. **Check for state corruption**
   ```bash
   # Look for invalid data in module state
   jq '.app_state.dex.pools' state-debug.json
   # Verify all pool data is valid
   ```

3. **Rollback if migration consistently fails**
   ```bash
   # See Rollback Procedures section
   # Coordinate with validators to rollback to pre-upgrade state
   ```

---

#### Issue 3: Validator Missing Blocks After Upgrade

**Symptoms:**
```
Validator not appearing in block signatures
Missed blocks counter increasing rapidly
```

**Diagnosis:**
```bash
# Check validator status
pawd query staking validator $VAL_ADDR | jq '.status, .jailed'

# Check signing info
pawd query slashing signing-info $(pawd tendermint show-validator)

# Check node sync status
pawd status | jq '.SyncInfo'
```

**Solutions:**

1. **Verify node is synced**
   ```bash
   # If catching_up is true, wait for sync
   pawd status | jq '.SyncInfo.catching_up'

   # Monitor sync progress
   watch -n 5 'pawd status | jq ".SyncInfo.latest_block_height"'
   ```

2. **Check validator key**
   ```bash
   # Verify priv_validator_key.json is present
   ls -la ~/.paw/config/priv_validator_key.json

   # Check consensus public key matches
   pawd tendermint show-validator
   pawd query staking validator $VAL_ADDR | jq '.consensus_pubkey'
   ```

3. **Restart node**
   ```bash
   sudo systemctl restart pawd
   journalctl -u pawd -f
   ```

4. **Check for double signing protection**
   ```bash
   # If you restored from backup, priv_validator_state.json may be old
   # This can trigger double-sign protection

   # Check state file
   cat ~/.paw/data/priv_validator_state.json

   # If height is far behind, you may need to carefully update it
   # CAUTION: Incorrect updates can cause double signing
   ```

---

#### Issue 4: Cosmovisor Doesn't Switch Binary

**Symptoms:**
```
Upgrade height reached but Cosmovisor still uses old binary
Chain halted but no automatic restart
```

**Diagnosis:**
```bash
# Check Cosmovisor is running
ps aux | grep cosmovisor

# Check upgrade directory
ls -la $DAEMON_HOME/cosmovisor/upgrades/v1.1.0/bin/

# Check Cosmovisor logs
journalctl -u cosmovisor-pawd -n 100
```

**Solutions:**

1. **Verify upgrade name matches exactly**
   ```bash
   # Check governance proposal upgrade name
   pawd query upgrade plan | jq '.name'

   # Check directory name
   ls $DAEMON_HOME/cosmovisor/upgrades/

   # Names must match EXACTLY (case-sensitive)
   ```

2. **Verify binary exists and is executable**
   ```bash
   ls -la $DAEMON_HOME/cosmovisor/upgrades/v1.1.0/bin/pawd
   # Should show: -rwxr-xr-x

   chmod +x $DAEMON_HOME/cosmovisor/upgrades/v1.1.0/bin/pawd
   ```

3. **Check environment variables**
   ```bash
   systemctl cat cosmovisor-pawd | grep Environment
   # Verify DAEMON_NAME and DAEMON_HOME are correct
   ```

4. **Manual fallback**
   ```bash
   # Manually switch binary and restart
   sudo systemctl stop cosmovisor-pawd

   cp $DAEMON_HOME/cosmovisor/upgrades/v1.1.0/bin/pawd /usr/local/bin/pawd

   sudo systemctl start cosmovisor-pawd
   ```

---

#### Issue 5: State Export Fails

**Symptoms:**
```
pawd export > state.json
Error: failed to export state
```

**Diagnosis:**
```bash
# Check node status
pawd status

# Check disk space
df -h ~/.paw

# Check logs for errors
journalctl -u pawd | grep -i export
```

**Solutions:**

1. **Ensure node is stopped**
   ```bash
   sudo systemctl stop pawd
   pawd export > state.json
   ```

2. **Export at specific height**
   ```bash
   pawd export --height 1500000 > state.json
   ```

3. **Check for database corruption**
   ```bash
   # Try database integrity check
   pawd tendermint unsafe-reset-all --keep-addr-book
   # This will reset state - only use if you have backups
   ```

4. **Use database backup instead**
   ```bash
   # If state export fails, use raw database backup
   tar -czf data-backup.tar.gz ~/.paw/data/
   ```

---

#### Issue 6: IBC Channels Broken After Upgrade

**Symptoms:**
```
IBC transfers failing
Channels showing as "STATE_CLOSED"
Timeout errors on IBC packets
```

**Diagnosis:**
```bash
# Check channel status
pawd query ibc channel channels

# Check specific channel
pawd query ibc channel end transfer channel-0

# Check pending packets
pawd query ibc channel unreceived-packets transfer channel-0
```

**Solutions:**

1. **Verify counterparty chain has also upgraded**
   ```bash
   # Query counterparty chain status
   # Both chains must be on compatible IBC versions
   ```

2. **Check for hung packets**
   ```bash
   # Query unreceived packets
   pawd query ibc channel unreceived-packets transfer channel-0

   # Relay pending packets
   hermes clear packets --chain paw-1 --port transfer --channel channel-0
   ```

3. **Re-establish channel if necessary**
   ```bash
   # Close old channel (requires counterparty coordination)
   pawd tx ibc channel close transfer channel-0 \
     --from validator \
     --yes

   # Create new channel connection
   # (Requires IBC relayer)
   ```

---

#### Issue 7: Performance Degradation After Upgrade

**Symptoms:**
```
Slower block times
High memory usage
Increased CPU utilization
```

**Diagnosis:**
```bash
# Check resource usage
top -p $(pgrep pawd)
ps aux | grep pawd
df -h ~/.paw

# Check block times
pawd status | jq '.SyncInfo.latest_block_time'

# Monitor Prometheus metrics
curl http://localhost:26660/metrics | grep tendermint
```

**Solutions:**

1. **Check for excessive logging**
   ```bash
   # Reduce log level if too verbose
   # Edit ~/.paw/config/config.toml
   [log]
   level = "main:info,state:info"  # Instead of "debug"

   sudo systemctl restart pawd
   ```

2. **Optimize database settings**
   ```bash
   # Edit ~/.paw/config/app.toml
   [state-sync]
   snapshot-interval = 1000  # Adjust snapshot frequency
   snapshot-keep-recent = 2  # Reduce kept snapshots
   ```

3. **Monitor and tune Go GC**
   ```bash
   # Adjust GOGC environment variable
   # Edit systemd service
   Environment="GOGC=100"  # Default, lower = more frequent GC

   sudo systemctl daemon-reload
   sudo systemctl restart pawd
   ```

4. **Check for memory leaks**
   ```bash
   # Monitor memory over time
   watch -n 10 'ps aux | grep pawd | grep -v grep'

   # If memory continually grows, report to developers
   ```

---

### Getting Help

If issues persist after troubleshooting:

1. **Gather diagnostic information**
   ```bash
   # Create diagnostic report
   cat > diagnostic-report.txt <<EOF
   Node Version: $(pawd version)
   Block Height: $(pawd status | jq -r '.SyncInfo.latest_block_height')
   Catching Up: $(pawd status | jq -r '.SyncInfo.catching_up')
   Validator Status: $(pawd query staking validator $VAL_ADDR | jq -r '.status')
   Peers: $(pawd status | jq -r '.NodeInfo.other.n_peers // 0')

   Recent Logs:
   $(journalctl -u pawd -n 50 --no-pager)
   EOF
   ```

2. **Contact support channels**
   - **Discord**: Validator support channel
   - **Telegram**: @paw_validators
   - **Email**: support@paw-chain.org
   - **GitHub Issues**: For bug reports

3. **Provide diagnostic report**
   - Share diagnostic-report.txt (NEVER share private keys)
   - Include upgrade name and height
   - Describe exact error messages
   - List steps already attempted

---

## Additional Resources

### Documentation

- **Main Documentation**: [docs.paw.com](https://docs.paw.com)
- **Validator Setup Guide**: [`VALIDATOR_OPERATOR_GUIDE.md`](./VALIDATOR_OPERATOR_GUIDE.md)
- **Validator Key Management**: [`VALIDATOR_KEY_MANAGEMENT.md`](./VALIDATOR_KEY_MANAGEMENT.md)
- **Disaster Recovery**: [`DISASTER_RECOVERY.md`](./DISASTER_RECOVERY.md)
- **Rollback Procedures**: [`docs/upgrades/ROLLBACK.md`](./upgrades/ROLLBACK.md)
- **Upgrade Proposal Template**: [`docs/upgrades/UPGRADE_PROPOSAL_TEMPLATE.md`](./upgrades/UPGRADE_PROPOSAL_TEMPLATE.md)

### Tools

- **Cosmovisor**: [Cosmos SDK Cosmovisor Docs](https://docs.cosmos.network/main/build/tooling/cosmovisor)
- **Cosmos SDK**: [Cosmos SDK Documentation](https://docs.cosmos.network/)
- **CometBFT**: [CometBFT Documentation](https://docs.cometbft.com/)
- **IBC**: [IBC Protocol Specification](https://github.com/cosmos/ibc)

### Community

- **Discord**: https://discord.gg/paw-chain (Validator channel)
- **Telegram**: https://t.me/paw_validators
- **Forum**: https://forum.paw-chain.org/c/governance
- **Twitter**: [@paw_chain](https://twitter.com/paw_chain)
- **GitHub**: https://github.com/decristofaroj/paw

### Support

- **Technical Support**: support@paw-chain.org
- **Security Issues**: security@paw-chain.org (PGP key available)
- **Validator Coordination**: validators@paw-chain.org

---

## Appendix: Upgrade Checklist Summary

### Pre-Upgrade (1-2 Weeks Before)

- [ ] Read upgrade documentation thoroughly
- [ ] Verify hardware requirements
- [ ] Test upgrade on local node
- [ ] Backup validator keys (encrypted)
- [ ] Export blockchain state
- [ ] Backup database
- [ ] Download and verify new binary
- [ ] Setup Cosmovisor (recommended)
- [ ] Join validator coordination channel
- [ ] Notify delegators of planned downtime
- [ ] Schedule maintenance window

### During Upgrade (Upgrade Day)

- [ ] Monitor block height approaching upgrade
- [ ] Verify new binary is ready
- [ ] Watch for chain halt at upgrade height
- [ ] Stop node (if manual upgrade)
- [ ] Backup current state
- [ ] Install new binary
- [ ] Restart node / let Cosmovisor auto-upgrade
- [ ] Monitor logs for successful migration
- [ ] Verify block production resumes

### Post-Upgrade (Within 24 Hours)

- [ ] Verify node is producing blocks
- [ ] Check validator signing status
- [ ] Verify validator is in active set
- [ ] Test RPC/API endpoints
- [ ] Verify IBC channels (if applicable)
- [ ] Test module functionality
- [ ] Monitor performance metrics
- [ ] Check logs for errors
- [ ] Report status to coordination channel
- [ ] Update monitoring dashboards
- [ ] Notify delegators of successful upgrade
- [ ] Document any issues encountered

---

**Document Version:** 1.0
**Last Updated:** 2025-12-07
**Maintainer:** PAW Core Development Team
**Status:** Production Ready

---

**IMPORTANT:** This document should be reviewed and updated with each upgrade to incorporate lessons learned and reflect the latest best practices. Validators are encouraged to contribute improvements based on their operational experience.
