# PAW Validator Operator Guide

Comprehensive operational guide for running and maintaining a PAW blockchain validator, from initial setup through ongoing operations, monitoring, and incident response.

## Table of Contents

1. [Introduction](#introduction)
2. [Becoming a Validator](#becoming-a-validator)
3. [Setting Up a Validator Node](#setting-up-a-validator-node)
4. [Key Management Best Practices](#key-management-best-practices)
5. [Monitoring Your Validator](#monitoring-your-validator)
6. [Commission and Rewards](#commission-and-rewards)
7. [Slashing Conditions and Avoidance](#slashing-conditions-and-avoidance)
8. [Unbonding and Migration](#unbonding-and-migration)
9. [Jailed Validator Recovery](#jailed-validator-recovery)
10. [Governance Participation](#governance-participation)

---

## Introduction

### What is a Validator?

Validators are critical infrastructure operators on the PAW blockchain responsible for:

- **Block Production**: Proposing and validating new blocks using CometBFT consensus
- **Network Security**: Participating in Byzantine Fault Tolerant consensus
- **Governance**: Voting on protocol upgrades and parameter changes
- **State Transitions**: Executing and validating transactions across all modules

Validators earn rewards from transaction fees and block rewards, but also face slashing penalties for downtime and byzantine behavior.

### Validator Economics Overview

| Metric | Typical Range | Notes |
|--------|--------------|-------|
| **Minimum Self-Delegation** | 1,000,000 upaw | Set during validator creation |
| **Commission Rate** | 5-20% | Your percentage of delegator rewards |
| **Maximum Commission** | 20-100% | Upper limit you can charge |
| **Commission Change Rate** | 1-5% per day | Maximum daily adjustment |
| **Unbonding Period** | 21 days | Time to withdraw staked tokens |
| **Downtime Jail Threshold** | 5% missed blocks | ~2,500 blocks in 50,000 block window |
| **Downtime Slash** | 0.01% | Penalty for exceeding downtime threshold |
| **Double-Sign Slash** | 5% | Penalty for byzantine behavior |

### Validator States

```
┌─────────────────────────────────────────────────────────────┐
│ BONDED (Active Set)                                         │
│ - Voting power > 0                                          │
│ - Participates in consensus                                 │
│ - Earns rewards                                             │
│ - Subject to slashing                                       │
└─────────────────────────────────────────────────────────────┘
                          │
         Unbond / Insufficient Stake / Jailed
                          ▼
┌─────────────────────────────────────────────────────────────┐
│ UNBONDING (Transitioning)                                   │
│ - No longer participates in consensus                       │
│ - Funds locked for unbonding period                         │
│ - Still subject to slashing for past behavior               │
│ - Duration: 21 days (configurable)                          │
└─────────────────────────────────────────────────────────────┘
                          │
              After Unbonding Period
                          ▼
┌─────────────────────────────────────────────────────────────┐
│ UNBONDED (Inactive)                                         │
│ - No participation in consensus                             │
│ - Funds withdrawable                                        │
│ - No slashing risk                                          │
│ - Can re-bond at any time                                   │
└─────────────────────────────────────────────────────────────┘
```

---

## Becoming a Validator

### Prerequisites

#### Hardware Requirements

| Component | Minimum (Testnet) | Recommended (Mainnet) | Notes |
|-----------|-------------------|----------------------|-------|
| **CPU** | 4 cores (x86_64) | 8 cores / 16 threads | AVX2 support recommended |
| **RAM** | 8 GB | 32 GB | More for state-sync/snapshots |
| **Storage** | 250 GB SSD | 1 TB NVMe SSD | Low-latency critical |
| **Network** | 100 Mbps symmetric | 1 Gbps dedicated | Low latency < 50ms |
| **OS** | Ubuntu 22.04 LTS | Ubuntu 22.04 LTS | Or equivalent systemd Linux |

**Cloud Provider Recommendations:**
- **AWS**: c6i.2xlarge (8 vCPU, 16 GB RAM) or larger
- **GCP**: c2-standard-8 (8 vCPU, 32 GB RAM) or larger
- **Azure**: F8s_v2 (8 vCPU, 16 GB RAM) or larger
- **Bare Metal**: Preferred for maximum performance and sovereignty

#### Network Requirements

```bash
# Required open ports:
# 26656 - P2P (required for all peers)
# 26657 - RPC (only for sentries/monitoring, NOT public)
# 26658 - Remote Signer (only from tmkms, heavily firewalled)
# 1317  - REST API (optional, for monitoring)
# 26660 - Prometheus metrics (optional, from monitoring only)

# Firewall configuration (ufw example):
sudo ufw default deny incoming
sudo ufw default allow outgoing
sudo ufw allow from <sentry-ip> to any port 26656 proto tcp
sudo ufw allow from <tmkms-ip> to any port 26658 proto tcp
sudo ufw allow from <monitoring-ip> to any port 26660 proto tcp
sudo ufw deny 26657  # No public RPC access
sudo ufw enable
```

**Network Architecture Best Practices:**
```
Internet
    │
    ▼
┌─────────────┐       ┌─────────────┐
│   Sentry    │◄─────►│   Sentry    │  (Public P2P nodes)
│   Node 1    │       │   Node 2    │
└─────────────┘       └─────────────┘
       │                     │
       └──────┬──────────────┘
              │ (private network)
              ▼
       ┌─────────────┐
       │  Validator  │  (Private, firewalled)
       │    Node     │
       └─────────────┘
              │
              ▼
       ┌─────────────┐
       │   tmkms     │  (Air-gapped or HSM)
       │   Signer    │
       └─────────────┘
```

#### Staking Requirements

```bash
# Check current validator set and minimum stake
pawd query staking validators --output json | \
  jq -r '.validators[] | "\(.description.moniker): \(.tokens)"' | \
  sort -t: -k2 -rn | head -20

# Estimate minimum competitive stake
pawd query staking pool

# Typical mainnet requirements:
# - Minimum self-delegation: 1,000,000 upaw (1 PAW)
# - Competitive stake: Top 100 validator set varies
# - Recommended initial: 10,000,000+ upaw (10+ PAW)
```

### Validator Creation Process

#### Step 1: Prepare Infrastructure

```bash
# Install dependencies
sudo apt update && sudo apt upgrade -y
sudo apt install -y build-essential git curl jq unzip systemd

# Install Go 1.21+
curl -L https://go.dev/dl/go1.21.6.linux-amd64.tar.gz -o /tmp/go.tar.gz
sudo rm -rf /usr/local/go && sudo tar -C /usr/local -xzf /tmp/go.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin:$HOME/go/bin' >> ~/.bashrc
source ~/.bashrc

# Verify Go installation
go version
```

#### Step 2: Build and Install pawd

```bash
# Clone repository
cd ~
git clone https://github.com/paw-chain/paw.git
cd paw

# Checkout latest stable release
git fetch --tags
git checkout $(git describe --tags `git rev-list --tags --max-count=1`)

# Build binary
make build

# Install globally
sudo install -m 0755 build/pawd /usr/local/bin/pawd

# Verify installation
pawd version
```

#### Step 3: Initialize Node

```bash
# Set chain ID (testnet)
CHAIN_ID="paw-testnet-1"

# Initialize node with unique moniker
pawd init "<your-validator-name>" --chain-id $CHAIN_ID

# Download genesis file (testnet)
curl https://raw.githubusercontent.com/paw-chain/paw/main/networks/paw-testnet-1/genesis.json \
  > ~/.paw/config/genesis.json
curl https://raw.githubusercontent.com/paw-chain/paw/main/networks/paw-testnet-1/genesis.sha256 \
  > /tmp/genesis.sha256
cd ~/.paw/config && sha256sum -c /tmp/genesis.sha256

# Verify genesis checksum
sha256sum ~/.paw/config/genesis.json
# Compare with official checksum from documentation
```

#### Step 4: Configure Node

```bash
# Edit config.toml
vi ~/.paw/config/config.toml

# Key settings:
# - persistent_peers: Add seed nodes and persistent peers
# - private_peer_ids: Add sentry node IDs (for sentry architecture)
# - addr_book_strict: false (for private networks)
# - max_num_inbound_peers / max_num_outbound_peers: Adjust based on capacity

# Edit app.toml
vi ~/.paw/config/app.toml

# Key settings:
# - minimum-gas-prices: "0.001upaw"
# - pruning: "default" or "custom"
# - index-events: ["tx.hash", "tx.height", ...] for query requirements
```

**Recommended config.toml settings:**
```toml
# Connection settings
persistent_peers = "<peer-id>@<ip>:26656,<peer-id>@<ip>:26656"
max_num_inbound_peers = 40
max_num_outbound_peers = 10
seeds = ""

# Consensus settings
timeout_propose = "3s"
timeout_propose_delta = "500ms"
timeout_prevote = "1s"
timeout_prevote_delta = "500ms"
timeout_precommit = "1s"
timeout_precommit_delta = "500ms"
timeout_commit = "5s"

# Mempool settings
size = 5000
cache_size = 10000

# State sync (for fast sync)
enable = false  # Set true if syncing from snapshot

# Monitoring
prometheus = true
prometheus_listen_addr = ":26660"
```

**Recommended app.toml settings:**
```toml
# Minimum gas prices
minimum-gas-prices = "0.001upaw"

# Pruning (keep last 100,000 blocks)
pruning = "custom"
pruning-keep-recent = "100000"
pruning-interval = "10"

# State sync snapshots
snapshot-interval = 1000
snapshot-keep-recent = 2

# API (disable for validators, enable for sentries)
[api]
enable = false
swagger = false

# gRPC
[grpc]
enable = true
address = "0.0.0.0:9090"

# State
[state-sync]
snapshot-interval = 1000
snapshot-keep-recent = 2
```

#### Step 5: Sync the Node

**Option A: Sync from Genesis (Slow)**

```bash
# Start node
pawd start --home ~/.paw

# Monitor sync progress
pawd status | jq '.SyncInfo'
# Wait until catching_up: false (may take hours/days)
```

**Option B: State Sync (Fast - Recommended)**

```bash
# Configure state sync in config.toml
LATEST_HEIGHT=$(curl -s https://rpc.paw.network/block | jq -r '.result.block.header.height')
BLOCK_HEIGHT=$((LATEST_HEIGHT - 2000))
TRUST_HASH=$(curl -s "https://rpc.paw.network/block?height=$BLOCK_HEIGHT" | jq -r '.result.block_id.hash')

# Edit config.toml
sed -i.bak -E "s|^(enable[[:space:]]+=[[:space:]]+).*$|\1true| ; \
s|^(rpc_servers[[:space:]]+=[[:space:]]+).*$|\1\"https://rpc.paw.network:443,https://rpc2.paw.network:443\"| ; \
s|^(trust_height[[:space:]]+=[[:space:]]+).*$|\1$BLOCK_HEIGHT| ; \
s|^(trust_hash[[:space:]]+=[[:space:]]+).*$|\1\"$TRUST_HASH\"|" ~/.paw/config/config.toml

# Start node (will state sync)
pawd start --home ~/.paw

# Monitor - should sync in minutes instead of hours
pawd status | jq '.SyncInfo'
```

**Option C: Snapshot Restore (Fastest)**

```bash
# Download snapshot from trusted source
wget https://snapshots.paw.network/paw-mainnet-latest.tar.gz

# Stop node
sudo systemctl stop pawd

# Reset data
pawd tendermint unsafe-reset-all --home ~/.paw --keep-addr-book

# Extract snapshot
tar -xzf paw-mainnet-latest.tar.gz -C ~/.paw/data

# Start node
sudo systemctl start pawd

# Verify sync
pawd status | jq '.SyncInfo.catching_up'
# Should return false immediately
```

#### Step 6: Create Validator Keys

**CRITICAL SECURITY:** See [VALIDATOR_KEY_MANAGEMENT.md](./VALIDATOR_KEY_MANAGEMENT.md) for comprehensive key generation procedures.

**Summary for production validators:**

```bash
# Generate operator key (use Ledger for mainnet)
pawd keys add validator-operator --ledger

# Or for testnet (software keyring):
pawd keys add validator-operator --keyring-backend os

# Backup mnemonic IMMEDIATELY (write on paper, store in safe)
# NEVER share mnemonic with anyone
# NEVER store mnemonic digitally on internet-connected device

# Get operator address
OPERATOR_ADDRESS=$(pawd keys show validator-operator -a)
echo "Operator Address: $OPERATOR_ADDRESS"

# Consensus key is generated during 'pawd init'
# For mainnet, migrate to HSM/tmkms (see key management guide)

# Get consensus public key
CONSENSUS_PUBKEY=$(pawd tendermint show-validator)
echo "Consensus PubKey: $CONSENSUS_PUBKEY"
```

#### Step 7: Fund Operator Account

```bash
# Receive tokens to operator address
# Mainnet: Acquire PAW through exchanges, OTC, or foundation
# Testnet: Use faucet

# Verify balance
pawd query bank balances $OPERATOR_ADDRESS

# Ensure sufficient funds for:
# - Validator creation (self-delegation amount)
# - Transaction fees (keep extra ~1000 upaw)
```

#### Step 8: Create Validator

```bash
# Create validator transaction
pawd tx staking create-validator \
  --from validator-operator \
  --amount 10000000upaw \
  --pubkey "$CONSENSUS_PUBKEY" \
  --moniker "<your-validator-name>" \
  --identity "<keybase-identity>" \
  --website "https://your-website.com" \
  --security-contact "security@your-domain.com" \
  --details "Detailed description of your validator operation" \
  --chain-id paw-mainnet-1 \
  --commission-rate 0.10 \
  --commission-max-rate 0.20 \
  --commission-max-change-rate 0.01 \
  --min-self-delegation 1000000 \
  --gas auto \
  --gas-adjustment 1.5 \
  --gas-prices 0.001upaw \
  --broadcast-mode block

# Verify validator creation
pawd query staking validator $(pawd keys show validator-operator --bech val -a)

# Check validator status
pawd query staking validators --output json | \
  jq -r '.validators[] | select(.description.moniker=="<your-validator-name>")'
```

**Parameter Explanations:**

- `--amount`: Initial self-delegation (cannot go below this without unbonding validator)
- `--moniker`: Public validator name (visible in explorers)
- `--identity`: Keybase.io PGP fingerprint (for verified logo/identity)
- `--commission-rate`: Initial commission (5-20% typical)
- `--commission-max-rate`: Maximum commission you can ever charge
- `--commission-max-change-rate`: Maximum daily commission change (prevents sudden increases)
- `--min-self-delegation`: Minimum self-stake required to remain validator

**Creating Keybase Identity (for verified logo):**

```bash
# Install keybase
curl --remote-name https://prerelease.keybase.io/keybase_amd64.deb
sudo dpkg -i keybase_amd64.deb

# Create account and upload validator logo
keybase login
keybase pgp gen --multi

# Get 16-char identity
keybase id | grep key
# Use this as --identity parameter
```

---

## Setting Up a Validator Node

### Production Deployment Architecture

**Recommended Sentry Architecture:**

```
                    Internet
                       │
        ┌──────────────┼──────────────┐
        │              │              │
        ▼              ▼              ▼
   ┌─────────┐    ┌─────────┐    ┌─────────┐
   │ Sentry  │    │ Sentry  │    │ Sentry  │
   │ Node 1  │    │ Node 2  │    │ Node 3  │
   │ Public  │    │ Public  │    │ Public  │
   └─────────┘    └─────────┘    └─────────┘
        │              │              │
        └──────────────┼──────────────┘
                       │ Private VPN/VLAN
                       ▼
                ┌─────────────┐
                │  Validator  │
                │    Node     │
                │  (Private)  │
                └─────────────┘
                       │
                       │ Firewalled
                       ▼
                ┌─────────────┐
                │   tmkms     │
                │   Signer    │
                │ (Air-gapped │
                │  or HSM)    │
                └─────────────┘
```

### Systemd Service Configuration

**Create validator systemd service:**

```bash
sudo tee /etc/systemd/system/pawd.service > /dev/null <<EOF
[Unit]
Description=PAW Validator Node
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=validator
Group=validator
WorkingDirectory=/home/validator
Environment="PAW_HOME=/home/validator/.paw"

ExecStart=/usr/local/bin/pawd start \\
  --home \${PAW_HOME} \\
  --minimum-gas-prices 0.001upaw

Restart=on-failure
RestartSec=10
LimitNOFILE=65535

# Security hardening
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=full
ProtectHome=false

[Install]
WantedBy=multi-user.target
EOF

# Create dedicated user
sudo useradd -m -s /bin/bash validator
sudo chown -R validator:validator /home/validator/.paw

# Enable and start service
sudo systemctl daemon-reload
sudo systemctl enable pawd
sudo systemctl start pawd

# Monitor service
sudo systemctl status pawd
sudo journalctl -u pawd -f
```

### Cosmovisor Setup (Recommended for Automatic Upgrades)

```bash
# Install Cosmovisor
go install cosmossdk.io/tools/cosmovisor/cmd/cosmovisor@latest

# Setup directory structure
export DAEMON_HOME=$HOME/.paw
export DAEMON_NAME=pawd

mkdir -p $DAEMON_HOME/cosmovisor/genesis/bin
mkdir -p $DAEMON_HOME/cosmovisor/upgrades

# Copy current binary
cp $(which pawd) $DAEMON_HOME/cosmovisor/genesis/bin/

# Create Cosmovisor systemd service
sudo tee /etc/systemd/system/cosmovisor.service > /dev/null <<EOF
[Unit]
Description=Cosmovisor for PAW Validator
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
User=validator
Group=validator
WorkingDirectory=/home/validator

Environment="DAEMON_NAME=pawd"
Environment="DAEMON_HOME=/home/validator/.paw"
Environment="DAEMON_RESTART_AFTER_UPGRADE=true"
Environment="DAEMON_ALLOW_DOWNLOAD_BINARIES=false"
Environment="UNSAFE_SKIP_BACKUP=false"

ExecStart=/home/validator/go/bin/cosmovisor run start \\
  --home /home/validator/.paw \\
  --minimum-gas-prices 0.001upaw

Restart=on-failure
RestartSec=10
LimitNOFILE=65535

[Install]
WantedBy=multi-user.target
EOF

# Enable and start Cosmovisor
sudo systemctl daemon-reload
sudo systemctl enable cosmovisor
sudo systemctl start cosmovisor
```

---

## Key Management Best Practices

**CRITICAL:** Proper key management is the most important security aspect of validator operations. Compromise or loss of keys can result in permanent slashing and fund loss.

### Three Types of Validator Keys

| Key Type | Purpose | Storage | Compromise Impact | Rotation |
|----------|---------|---------|-------------------|----------|
| **Consensus Key** | Sign blocks and votes | HSM/tmkms (hot) | CRITICAL - Double-sign slashing (5%) | NEVER (requires validator recreation) |
| **Operator Key** | Staking operations | Ledger (cold) | HIGH - Unauthorized staking changes | IMPOSSIBLE (address is permanent) |
| **Node Key** | P2P identity | File (hot) | LOW - P2P impersonation | Anytime (no slashing risk) |

### Consensus Key Security

**For Production Mainnet Validators:**

1. **NEVER store consensus key on the validator node itself**
2. **ALWAYS use HSM or tmkms remote signer**
3. **Generate keys on air-gapped machines**
4. **Maintain encrypted backups in multiple physical locations**

**Quick Setup (see VALIDATOR_KEY_MANAGEMENT.md for full details):**

```bash
# Install tmkms with YubiHSM support
cargo install tmkms --features=yubihsm --locked

# Initialize tmkms configuration
tmkms init /etc/tmkms

# Generate key on YubiHSM
tmkms yubihsm keys generate 1 -b

# Configure validator for remote signer
# Edit ~/.paw/config/config.toml:
priv_validator_laddr = "tcp://0.0.0.0:26658"

# Start tmkms
tmkms start -c /etc/tmkms/tmkms.toml

# Firewall: Only allow tmkms IP to port 26658
sudo ufw allow from <tmkms-ip> to any port 26658 proto tcp
```

### Operator Key Security

**For Production Mainnet Validators:**

```bash
# Use Ledger hardware wallet
pawd keys add validator-operator --ledger

# For multi-sig (institutional validators):
pawd keys add validator-multisig \
  --multisig operator1,operator2,operator3 \
  --multisig-threshold 2

# Backup mnemonic:
# - Write on paper (never digital)
# - Store in fireproof safe + bank vault
# - Test recovery quarterly
```

### Key Backup Verification

**Quarterly Drill (MANDATORY):**

```bash
# Drill 1: Decrypt consensus key backup
gpg --decrypt priv_validator_key.json.gpg
# Verify checksum, then re-encrypt

# Drill 2: Recover operator key from mnemonic
pawd keys delete validator-operator-test
pawd keys add validator-operator-test --recover
# Enter mnemonic from paper backup
# Verify address matches

# Drill 3: Test tmkms failover
# Stop tmkms, verify validator stops signing
# Start tmkms, verify signing resumes
```

**See [VALIDATOR_KEY_MANAGEMENT.md](./VALIDATOR_KEY_MANAGEMENT.md) for comprehensive procedures.**

---

## Monitoring Your Validator

### Essential Metrics to Monitor

| Metric Category | Key Indicators | Alert Threshold | Impact |
|----------------|----------------|-----------------|---------|
| **Consensus Participation** | Missed blocks, voting power | >1% missed blocks | Jailing, slashing |
| **Node Health** | Sync status, peer count | Catching up = true | No block signing |
| **System Resources** | CPU, RAM, disk I/O | >80% utilization | Performance degradation |
| **Network** | Latency, bandwidth | >100ms to peers | Slow consensus |
| **Signing** | Block signatures, double-sign | Any double-sign | 5% slashing |

### Prometheus + Grafana Setup

**Install Prometheus:**

```bash
# Download and install Prometheus
wget https://github.com/prometheus/prometheus/releases/download/v2.45.0/prometheus-2.45.0.linux-amd64.tar.gz
tar -xzf prometheus-2.45.0.linux-amd64.tar.gz
sudo mv prometheus-2.45.0.linux-amd64 /opt/prometheus

# Create Prometheus configuration
sudo tee /opt/prometheus/prometheus.yml > /dev/null <<EOF
global:
  scrape_interval: 15s
  evaluation_interval: 15s

scrape_configs:
  - job_name: 'paw-validator'
    static_configs:
      - targets: ['localhost:26660']
        labels:
          instance: 'validator-1'

  - job_name: 'node-exporter'
    static_configs:
      - targets: ['localhost:9100']
EOF

# Create systemd service
sudo tee /etc/systemd/system/prometheus.service > /dev/null <<EOF
[Unit]
Description=Prometheus
After=network.target

[Service]
Type=simple
User=prometheus
ExecStart=/opt/prometheus/prometheus \\
  --config.file=/opt/prometheus/prometheus.yml \\
  --storage.tsdb.path=/opt/prometheus/data

Restart=on-failure

[Install]
WantedBy=multi-user.target
EOF

sudo systemctl daemon-reload
sudo systemctl enable --now prometheus
```

**Install Grafana:**

```bash
# Add Grafana repository
sudo apt-get install -y software-properties-common
sudo add-apt-repository "deb https://packages.grafana.com/oss/deb stable main"
wget -q -O - https://packages.grafana.com/gpg.key | sudo apt-key add -
sudo apt-get update
sudo apt-get install grafana

# Start Grafana
sudo systemctl enable --now grafana-server

# Access Grafana at http://localhost:3000
# Default credentials: admin/admin
```

**Import Cosmos Validator Dashboard:**

1. Login to Grafana (http://localhost:3000)
2. Add Prometheus data source (http://localhost:9090)
3. Import dashboard ID: 11036 (Cosmos Validator Monitor)
4. Customize panels for PAW-specific metrics

### Critical Alerts to Configure

```yaml
# Prometheus alerting rules
# File: /opt/prometheus/alert.rules.yml

groups:
  - name: validator_alerts
    interval: 30s
    rules:
      # Critical: Validator not signing
      - alert: ValidatorNotSigning
        expr: increase(tendermint_consensus_validator_missed_blocks[10m]) > 10
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "Validator missing blocks"
          description: "Validator has missed {{ $value }} blocks in the last 10 minutes"

      # Critical: Node not syncing
      - alert: NodeNotSyncing
        expr: tendermint_consensus_fast_syncing == 1
        for: 15m
        labels:
          severity: critical
        annotations:
          summary: "Node is catching up"
          description: "Node has been syncing for more than 15 minutes"

      # Warning: Low peer count
      - alert: LowPeerCount
        expr: tendermint_p2p_peers < 5
        for: 10m
        labels:
          severity: warning
        annotations:
          summary: "Low number of peers"
          description: "Node has {{ $value }} peers (threshold: 5)"

      # Critical: High disk usage
      - alert: HighDiskUsage
        expr: (node_filesystem_avail_bytes{mountpoint="/"} / node_filesystem_size_bytes{mountpoint="/"}) < 0.1
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: "Disk space low"
          description: "Less than 10% disk space remaining"

      # Critical: tmkms disconnected
      - alert: TmkmsDisconnected
        expr: up{job="tmkms"} == 0
        for: 2m
        labels:
          severity: critical
        annotations:
          summary: "tmkms signer offline"
          description: "Remote signer has been down for 2 minutes"
```

### Manual Monitoring Commands

```bash
# Check validator status
pawd status | jq '.'

# Check if catching up
pawd status | jq '.SyncInfo.catching_up'

# Get current block height
pawd status | jq '.SyncInfo.latest_block_height'

# Check validator info
pawd query staking validator $(pawd keys show validator-operator --bech val -a)

# Check signing info (missed blocks)
pawd query slashing signing-info $(pawd tendermint show-validator)

# Check validator voting power
pawd query staking validators --output json | \
  jq -r '.validators[] | select(.description.moniker=="<your-moniker>") | .tokens'

# Check peer count
curl -s localhost:26657/net_info | jq '.result.n_peers'

# Check recent blocks signed
pawd query block | jq '.block.last_commit.signatures[] | select(.validator_address != null)'
```

### Log Monitoring

```bash
# Real-time logs
sudo journalctl -u pawd -f

# Filter for errors
sudo journalctl -u pawd | grep -i error

# Filter for consensus issues
sudo journalctl -u pawd | grep -i "consensus"

# Export logs for analysis
sudo journalctl -u pawd --since "1 hour ago" > validator-logs.txt
```

---

## Commission and Rewards

### Understanding Validator Economics

**Reward Sources:**

1. **Block Rewards**: Inflation-based rewards distributed to validators and delegators
2. **Transaction Fees**: Fees from all transactions in each block
3. **Module-Specific Fees**: DEX trading fees, oracle submission fees, compute fees

**Reward Distribution:**

```
Total Block Reward (100%)
        │
        ├─► Community Pool (2%) - Governance-controlled treasury
        │
        └─► Validator Set (98%)
                │
                ├─► Your Validator (proportional to voting power)
                │      │
                │      ├─► Commission (5-20% of delegator rewards)
                │      │
                │      └─► Self-Delegation Rewards (from your own stake)
                │
                └─► Other Validators (proportional to their voting power)
```

### Setting Commission

**Initial Commission (at validator creation):**

```bash
# Conservative approach (build trust before raising)
--commission-rate 0.05         # 5% initial
--commission-max-rate 0.20     # Can raise to max 20%
--commission-max-change-rate 0.01  # Max 1% increase per day

# Competitive approach
--commission-rate 0.10         # 10% initial
--commission-max-rate 0.20
--commission-max-change-rate 0.02  # Max 2% increase per day
```

**Changing Commission:**

```bash
# Increase commission (example: 5% → 7%)
pawd tx staking edit-validator \
  --commission-rate 0.07 \
  --from validator-operator \
  --chain-id paw-mainnet-1 \
  --gas auto \
  --gas-prices 0.001upaw \
  --broadcast-mode block

# LIMITATIONS:
# - Cannot exceed commission-max-rate (set at creation)
# - Cannot increase more than commission-max-change-rate per day
# - Cannot decrease below 0
# - Changes are IMMEDIATE (delegators can react immediately)

# Check current commission
pawd query staking validator $(pawd keys show validator-operator --bech val -a) | \
  jq '.commission'
```

**Commission Best Practices:**

| Practice | Recommendation | Rationale |
|----------|---------------|-----------|
| **Start Low** | 5% for new validators | Build trust and attract delegators |
| **Communicate Changes** | Announce 1 week before increases | Allow delegators to make informed decisions |
| **Justify Increases** | Explain infrastructure improvements | Transparency builds trust |
| **Gradual Increases** | 1-2% per change, quarterly | Avoid shocking delegators |
| **Competitive Analysis** | Monitor top validators' rates | Stay competitive (5-15% typical) |

### Claiming Rewards

**Validator Operator Rewards:**

```bash
# Check pending rewards (commission + self-delegation)
pawd query distribution commission $(pawd keys show validator-operator --bech val -a)

# Withdraw commission rewards
pawd tx distribution withdraw-rewards $(pawd keys show validator-operator --bech val -a) \
  --from validator-operator \
  --commission \
  --chain-id paw-mainnet-1 \
  --gas auto \
  --gas-prices 0.001upaw \
  --broadcast-mode block

# Withdraw self-delegation rewards
pawd tx distribution withdraw-rewards $(pawd keys show validator-operator --bech val -a) \
  --from validator-operator \
  --chain-id paw-mainnet-1 \
  --gas auto \
  --gas-prices 0.001upaw \
  --broadcast-mode block

# Withdraw all rewards at once
pawd tx distribution withdraw-all-rewards \
  --from validator-operator \
  --chain-id paw-mainnet-1 \
  --gas auto \
  --gas-prices 0.001upaw \
  --broadcast-mode block
```

**Automatic Reward Compounding:**

```bash
# Script: auto-compound-rewards.sh
#!/bin/bash
set -e

VALIDATOR_OPERATOR="validator-operator"
VALIDATOR_ADDRESS=$(pawd keys show $VALIDATOR_OPERATOR --bech val -a)
CHAIN_ID="paw-mainnet-1"

# Withdraw rewards
pawd tx distribution withdraw-rewards $VALIDATOR_ADDRESS \
  --from $VALIDATOR_OPERATOR \
  --commission \
  --chain-id $CHAIN_ID \
  --gas auto \
  --gas-prices 0.001upaw \
  --yes

sleep 10

# Re-delegate rewards (keep some for fees)
BALANCE=$(pawd query bank balances $(pawd keys show $VALIDATOR_OPERATOR -a) \
  --output json | jq -r '.balances[] | select(.denom=="upaw") | .amount')

# Reserve 10000 upaw for fees
DELEGATE_AMOUNT=$((BALANCE - 10000))

if [ $DELEGATE_AMOUNT -gt 1000000 ]; then
  pawd tx staking delegate $VALIDATOR_ADDRESS ${DELEGATE_AMOUNT}upaw \
    --from $VALIDATOR_OPERATOR \
    --chain-id $CHAIN_ID \
    --gas auto \
    --gas-prices 0.001upaw \
    --yes
fi

# Schedule daily via cron:
# 0 0 * * * /home/validator/auto-compound-rewards.sh >> /var/log/auto-compound.log 2>&1
```

### Reward Calculations

**Estimated Annual Percentage Rate (APR):**

```bash
# Query inflation rate
INFLATION=$(pawd query mint inflation | tr -d '"')

# Query bonded ratio
BONDED_TOKENS=$(pawd query staking pool | jq -r '.bonded_tokens')
TOTAL_SUPPLY=$(pawd query bank total | jq -r '.supply[] | select(.denom=="upaw") | .amount')
BONDED_RATIO=$(echo "scale=4; $BONDED_TOKENS / $TOTAL_SUPPLY" | bc)

# Estimate validator APR
# APR ≈ (Inflation / Bonded_Ratio) * (1 - Community_Tax)
# Assume 2% community tax
VALIDATOR_APR=$(echo "scale=4; ($INFLATION / $BONDED_RATIO) * 0.98" | bc)

echo "Estimated Validator APR: ${VALIDATOR_APR}%"

# Delegator APR = Validator APR * (1 - Commission)
# If your commission is 10%:
DELEGATOR_APR=$(echo "scale=4; $VALIDATOR_APR * 0.90" | bc)
echo "Estimated Delegator APR: ${DELEGATOR_APR}%"
```

---

## Slashing Conditions and Avoidance

### Slashing Events

PAW blockchain implements two types of slashing:

#### 1. Downtime Slashing

**Trigger Conditions:**
- **Signed Blocks Window**: 50,000 blocks (~3.5 days at 6s block time)
- **Missed Block Threshold**: 5% of window (2,500 blocks)
- **Penalty**: 0.01% of bonded stake
- **Additional Penalty**: Validator jailed for 10 minutes

**Calculation:**
```
If validator misses > 2,500 blocks in any rolling 50,000 block window:
  → Slash 0.01% of stake
  → Jail validator (must unjail manually)
```

**Avoidance Strategies:**

| Strategy | Implementation | Effectiveness |
|----------|---------------|---------------|
| **High Uptime SLA** | Target 99.9%+ uptime | Essential |
| **Redundant Infrastructure** | Active/standby validators (DO NOT DOUBLE-SIGN) | High risk if misconfigured |
| **Monitoring & Alerts** | Alert on >100 missed blocks | Early warning system |
| **Automated Recovery** | Systemd auto-restart, health checks | Reduces downtime |
| **Horcrux (Advanced)** | Distributed signing across nodes | Complex, requires expertise |

**Monitoring Downtime:**

```bash
# Check current signing window status
pawd query slashing signing-info $(pawd tendermint show-validator)

# Output interpretation:
# {
#   "address": "pawvalcons1...",
#   "start_height": "0",
#   "index_offset": "12345",          # Blocks since last reset
#   "jailed_until": "1970-01-01...",  # If not jailed, shows epoch 0
#   "tombstoned": false,               # True if permanently slashed
#   "missed_blocks_counter": "150"     # Current missed blocks
# }

# Calculate distance to jail:
# Threshold: 2500 missed blocks
# Current: 150 missed blocks
# Remaining buffer: 2350 blocks

# Automate monitoring
watch -n 60 'pawd query slashing signing-info $(pawd tendermint show-validator) | jq ".missed_blocks_counter"'
```

#### 2. Double-Sign Slashing

**Trigger Conditions:**
- Validator signs two different blocks at the same height
- Typically caused by running multiple validator instances with same key
- **Penalty**: 5% of bonded stake
- **Additional Penalty**: Validator permanently tombstoned (cannot unjail)

**Avoidance Strategies:**

| Practice | Implementation | Critical Level |
|----------|---------------|----------------|
| **NEVER Duplicate Consensus Key** | Use tmkms with single HSM | CRITICAL |
| **Single Validator Instance** | Never run multiple pawd with same priv_validator_key | CRITICAL |
| **State File Management** | Ensure priv_validator_state.json is synced | HIGH |
| **Failover Coordination** | If using standby, ensure PRIMARY fully stopped | CRITICAL |
| **tmkms Double-Sign Protection** | tmkms maintains state to prevent double-sign | HIGH |

**Double-Sign Detection:**

```bash
# Check if validator is tombstoned
pawd query slashing signing-info $(pawd tendermint show-validator) | jq '.tombstoned'

# If true, validator is PERMANENTLY disabled
# ONLY OPTION: Create new validator with new consensus key

# Monitor for double-sign evidence
pawd query evidence

# Check validator status
pawd query staking validator $(pawd keys show validator-operator --bech val -a) | jq '.jailed'
```

**CRITICAL WARNING: Running Duplicate Validators**

```
❌ NEVER DO THIS:

Primary Validator (IP: 1.2.3.4)
    └─► priv_validator_key.json (consensus key A)

Backup Validator (IP: 5.6.7.8) - RUNNING SIMULTANEOUSLY
    └─► priv_validator_key.json (SAME consensus key A)

RESULT: Double-sign slashing, 5% stake loss, tombstoned

✅ CORRECT APPROACH:

Primary Validator (IP: 1.2.3.4)
    └─► tmkms (HSM with consensus key A)
            └─► State file prevents double-sign

If failover needed:
    1. FULLY STOP primary validator
    2. VERIFY primary is stopped (wait 2+ block times)
    3. Start standby validator
    4. Connect to SAME tmkms (state prevents double-sign)
```

### Tombstoning Recovery

**If double-sign slashing occurs:**

```bash
# 1. IMMEDIATELY STOP VALIDATOR
sudo systemctl stop pawd
sudo systemctl stop tmkms

# 2. Verify validator is tombstoned
pawd query slashing signing-info $(pawd tendermint show-validator)
# tombstoned: true

# 3. Calculate losses
pawd query staking validator $(pawd keys show validator-operator --bech val -a) | jq '.tokens'
# You will have lost 5% of bonded stake

# 4. ONLY OPTION: Create new validator
# - Generate NEW consensus key (never reuse compromised key)
# - Create new validator with create-validator transaction
# - Communicate to delegators (reputation damage likely severe)
# - Investigate root cause thoroughly

# 5. Old validator CANNOT be recovered
# Tombstoned validators are permanently disabled
```

### Jail Recovery

See [Jailed Validator Recovery](#jailed-validator-recovery) section below.

---

## Unbonding and Migration

### Unbonding Process

**Unbonding Timeline:**

```
T+0: Unbond Transaction Submitted
  └─► Validator moves to UNBONDING state
       │
       ├─► No longer participates in consensus
       ├─► No longer earns rewards
       ├─► Still subject to slashing for past behavior
       └─► Delegators can still unbond from you
            │
            ▼
T+21 days: Unbonding Period Completes
  └─► Validator moves to UNBONDED state
       │
       ├─► Funds fully withdrawable
       ├─► No slashing risk
       └─► Can re-bond at any time
```

### Self-Unbonding

```bash
# Check current delegation
pawd query staking delegation \
  $(pawd keys show validator-operator -a) \
  $(pawd keys show validator-operator --bech val -a)

# Unbond tokens from your validator
pawd tx staking unbond \
  $(pawd keys show validator-operator --bech val -a) \
  5000000upaw \
  --from validator-operator \
  --chain-id paw-mainnet-1 \
  --gas auto \
  --gas-prices 0.001upaw \
  --broadcast-mode block

# IMPORTANT: If unbonding reduces stake below min-self-delegation,
# your validator will be AUTOMATICALLY UNBONDED ENTIRELY

# Check unbonding status
pawd query staking unbonding-delegation \
  $(pawd keys show validator-operator -a) \
  $(pawd keys show validator-operator --bech val -a)

# Wait 21 days, then funds are automatically returned to operator account
```

### Validator Migration (Transferring to New Infrastructure)

**Scenario:** Moving validator to new servers without downtime

**Safe Migration Procedure:**

```bash
# STEP 1: Prepare new infrastructure
# - Provision new server
# - Install pawd
# - Configure firewall
# - DO NOT start validator yet

# STEP 2: Sync new node (without signing)
# On NEW node:
pawd init <same-moniker> --chain-id paw-mainnet-1
# Copy genesis.json from old node
# Configure config.toml (peers, etc.)
# DO NOT copy priv_validator_key.json
pawd start  # Let it sync without signing

# STEP 3: Wait for full sync
pawd status | jq '.SyncInfo.catching_up'
# Wait until: false

# STEP 4: Stop OLD validator (coordinated downtime)
# On OLD node:
sudo systemctl stop pawd
sudo systemctl stop tmkms

# STEP 5: Transfer signing capability
# Option A: Move tmkms to point to new validator IP
# Edit /etc/tmkms/tmkms.toml:
# [[validator]]
# addr = "tcp://<NEW_VALIDATOR_IP>:26658"

# Option B: Copy priv_validator_key.json (NOT RECOMMENDED for mainnet)
# scp old-node:~/.paw/config/priv_validator_key.json new-node:~/.paw/config/
# scp old-node:~/.paw/data/priv_validator_state.json new-node:~/.paw/data/

# STEP 6: Start NEW validator
# On NEW node:
sudo systemctl start pawd
# OR (if using tmkms):
sudo systemctl restart tmkms
sudo systemctl start pawd

# STEP 7: Verify signing
pawd status | jq '.ValidatorInfo'
tail -f /var/log/syslog | grep pawd

# STEP 8: Monitor for 24 hours
# - Check missed blocks
# - Verify rewards accumulating
# - Monitor logs for errors

# STEP 9: Decommission OLD node
# - Backup any remaining data
# - DELETE priv_validator_key.json (prevent accidental double-sign)
# - Shutdown server
```

**Migration Downtime:**
- Expected: 5-15 minutes
- Plan for: 30 minutes maximum
- Missed blocks during migration: 50-150 (well below jail threshold)

### Retiring a Validator

**Planned Shutdown:**

```bash
# 1. Announce to delegators (1-2 weeks notice)
# - Social media
# - Validator website
# - Discord/Telegram

# 2. Stop accepting new delegations (optional)
# Edit validator details to indicate retirement

# 3. Unbond all stake
pawd tx staking unbond \
  $(pawd keys show validator-operator --bech val -a) \
  $(pawd query staking delegation $(pawd keys show validator-operator -a) $(pawd keys show validator-operator --bech val -a) | jq -r '.balance.amount')upaw \
  --from validator-operator \
  --chain-id paw-mainnet-1 \
  --gas auto \
  --gas-prices 0.001upaw \
  --broadcast-mode block

# 4. Wait 21 days for unbonding period

# 5. After unbonding complete, stop validator
sudo systemctl stop pawd
sudo systemctl disable pawd

# 6. Securely wipe consensus keys
shred -vfz -n 10 ~/.paw/config/priv_validator_key.json
shred -vfz -n 10 ~/.paw/data/priv_validator_state.json

# 7. Withdraw all funds
pawd tx bank send validator-operator <destination-address> <amount>upaw \
  --from validator-operator \
  --chain-id paw-mainnet-1 \
  --gas auto \
  --gas-prices 0.001upaw \
  --broadcast-mode block
```

---

## Jailed Validator Recovery

### Understanding Jailing

**Jailing is a temporary penalty for downtime.**

- Validator stops earning rewards
- Validator does not participate in consensus
- Delegations remain bonded (do not automatically unbond)
- Must manually unjail to resume

**Common Causes:**
- Server downtime (hardware failure, network outage)
- Software bugs causing crashes
- Missed upgrade coordination
- Resource exhaustion (disk full, OOM)

### Checking Jail Status

```bash
# Check if validator is jailed
pawd query staking validator $(pawd keys show validator-operator --bech val -a) | jq '.jailed'
# true = jailed, false = active

# Check signing info
pawd query slashing signing-info $(pawd tendermint show-validator)

# Example output:
# {
#   "address": "pawvalcons1...",
#   "jailed_until": "2025-12-07T12:34:56.789Z",  # Must wait until this time
#   "tombstoned": false,                         # If true, CANNOT unjail
#   "missed_blocks_counter": "2501"              # Exceeded threshold
# }
```

### Unjailing Procedure

**Step 1: Fix Underlying Issue**

```bash
# Identify root cause
sudo journalctl -u pawd --since "2 hours ago" | grep -i error

# Common fixes:
# - Restart crashed service: sudo systemctl restart pawd
# - Free disk space: df -h && sudo journalctl --vacuum-time=7d
# - Restore network connectivity
# - Update to latest binary (if missed upgrade)

# Verify node is healthy
pawd status | jq '.SyncInfo.catching_up'
# Should return: false
```

**Step 2: Wait for Jail Period**

```bash
# Check when you can unjail
pawd query slashing signing-info $(pawd tendermint show-validator) | jq '.jailed_until'

# Typical jail duration: 10 minutes
# Must wait until this timestamp before unjailing
```

**Step 3: Submit Unjail Transaction**

```bash
# Unjail your validator
pawd tx slashing unjail \
  --from validator-operator \
  --chain-id paw-mainnet-1 \
  --gas auto \
  --gas-prices 0.001upaw \
  --broadcast-mode block

# Verify unjailed
pawd query staking validator $(pawd keys show validator-operator --bech val -a) | jq '.jailed'
# Should return: false

# Verify signing resumed
pawd status | jq '.ValidatorInfo'
# Should show voting_power > 0
```

**Step 4: Monitor Recovery**

```bash
# Monitor missed blocks counter (should stop increasing)
watch -n 60 'pawd query slashing signing-info $(pawd tendermint show-validator) | jq ".missed_blocks_counter"'

# Verify earning rewards again
pawd query distribution commission $(pawd keys show validator-operator --bech val -a)

# Check validator status
pawd query staking validators --output json | \
  jq -r '.validators[] | select(.description.moniker=="<your-moniker>") | {status, jailed, tokens}'
```

### Preventing Future Jailing

**Operational Improvements:**

| Prevention Strategy | Implementation | Priority |
|--------------------|----------------|----------|
| **High Availability** | Redundant power, network, hardware | High |
| **Monitoring & Alerts** | Alert on >100 missed blocks | Critical |
| **Automated Recovery** | Systemd auto-restart, health checks | High |
| **Capacity Planning** | Monitor disk, ensure 30%+ free | Medium |
| **Upgrade Coordination** | Join validator channels, test upgrades | Critical |
| **Backup Infrastructure** | Standby server (with extreme caution) | Advanced |

**Automated Recovery Script:**

```bash
#!/bin/bash
# File: /usr/local/bin/auto-unjail.sh
# Cron: */15 * * * * /usr/local/bin/auto-unjail.sh >> /var/log/auto-unjail.log 2>&1

set -e

OPERATOR_KEY="validator-operator"
CHAIN_ID="paw-mainnet-1"

# Check if jailed
JAILED=$(pawd query staking validator $(pawd keys show $OPERATOR_KEY --bech val -a) | jq -r '.jailed')

if [ "$JAILED" == "true" ]; then
    echo "[$(date)] Validator is jailed, attempting unjail..."

    # Attempt unjail
    pawd tx slashing unjail \
      --from $OPERATOR_KEY \
      --chain-id $CHAIN_ID \
      --gas auto \
      --gas-prices 0.001upaw \
      --yes

    if [ $? -eq 0 ]; then
        echo "[$(date)] Unjail transaction submitted successfully"
        # Send alert (email, Telegram, PagerDuty, etc.)
        # curl -X POST https://api.telegram.org/bot<TOKEN>/sendMessage \
        #   -d chat_id=<CHAT_ID> \
        #   -d text="Validator was jailed and has been unjailed automatically"
    else
        echo "[$(date)] Unjail transaction FAILED"
        # Send critical alert
    fi
else
    echo "[$(date)] Validator is active, no action needed"
fi
```

---

## Governance Participation

### Validator Governance Responsibilities

As a validator, you have a **fiduciary duty** to:

1. **Vote on all proposals** - Delegators inherit your vote if they don't vote themselves
2. **Research proposals thoroughly** - Understand technical and economic implications
3. **Communicate reasoning** - Transparency builds delegator trust
4. **Represent delegator interests** - Balance chain health with stakeholder needs

### Governance Process

```
┌─────────────────────────────────────────────────────────────┐
│ 1. DEPOSIT PERIOD (7 days)                                  │
│    - Proposal submitted with initial deposit                │
│    - Community deposits until MinDeposit reached            │
│    - If MinDeposit not reached: proposal rejected           │
└─────────────────────────────────────────────────────────────┘
                          │
              MinDeposit Reached (10,000 PAW)
                          ▼
┌─────────────────────────────────────────────────────────────┐
│ 2. VOTING PERIOD (7 days)                                   │
│    - Validators and delegators vote                         │
│    - Options: Yes, No, NoWithVeto, Abstain                  │
│    - Delegators can override validator vote                 │
└─────────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│ 3. TALLYING                                                  │
│    - Quorum: 33.4% of bonded stake must vote                │
│    - Pass Threshold: >50% of participating stake votes Yes  │
│    - Veto Threshold: <33.4% of participating stake NoWithVeto│
└─────────────────────────────────────────────────────────────┘
                          │
                          ▼
┌─────────────────────────────────────────────────────────────┐
│ 4. EXECUTION                                                 │
│    - If passed: Proposal executes automatically             │
│    - If failed: No action taken                             │
│    - If vetoed: Deposit burned                              │
└─────────────────────────────────────────────────────────────┘
```

### Viewing Active Proposals

```bash
# List all proposals
pawd query gov proposals

# Get proposal details
pawd query gov proposal <proposal-id>

# Check proposal tally
pawd query gov tally <proposal-id>

# Check your vote
pawd query gov vote <proposal-id> $(pawd keys show validator-operator -a)
```

### Voting on Proposals

```bash
# Vote YES
pawd tx gov vote <proposal-id> yes \
  --from validator-operator \
  --chain-id paw-mainnet-1 \
  --gas auto \
  --gas-prices 0.001upaw \
  --broadcast-mode block

# Vote NO
pawd tx gov vote <proposal-id> no \
  --from validator-operator \
  --chain-id paw-mainnet-1 \
  --gas auto \
  --gas-prices 0.001upaw \
  --broadcast-mode block

# Vote NO WITH VETO (strong disagreement, burns deposit if passes)
pawd tx gov vote <proposal-id> no_with_veto \
  --from validator-operator \
  --chain-id paw-mainnet-1 \
  --gas auto \
  --gas-prices 0.001upaw \
  --broadcast-mode block

# Vote ABSTAIN (counted for quorum, but neutral)
pawd tx gov vote <proposal-id> abstain \
  --from validator-operator \
  --chain-id paw-mainnet-1 \
  --gas auto \
  --gas-prices 0.001upaw \
  --broadcast-mode block
```

### Submitting Proposals

```bash
# Text Proposal (general governance)
pawd tx gov submit-proposal \
  --title "Proposal Title" \
  --description "Detailed description of the proposal" \
  --type Text \
  --deposit 10000000upaw \
  --from validator-operator \
  --chain-id paw-mainnet-1 \
  --gas auto \
  --gas-prices 0.001upaw \
  --broadcast-mode block

# Parameter Change Proposal
pawd tx gov submit-proposal param-change proposal.json \
  --from validator-operator \
  --deposit 10000000upaw \
  --chain-id paw-mainnet-1 \
  --gas auto \
  --gas-prices 0.001upaw \
  --broadcast-mode block

# proposal.json example:
{
  "title": "Increase Block Size",
  "description": "Increase max block size to improve throughput",
  "changes": [
    {
      "subspace": "baseapp",
      "key": "BlockParams",
      "value": "{\"max_bytes\":\"500000\",\"max_gas\":\"-1\"}"
    }
  ],
  "deposit": "10000000upaw"
}

# Software Upgrade Proposal (see UPGRADE_PROCEDURES.md)
pawd tx gov submit-proposal software-upgrade v1.2.0 \
  --title "Upgrade to v1.2.0" \
  --description "Consensus upgrade with new features" \
  --upgrade-height 1000000 \
  --upgrade-info '{"binaries":{...}}' \
  --deposit 10000000upaw \
  --from validator-operator \
  --chain-id paw-mainnet-1 \
  --gas auto \
  --gas-prices 0.001upaw \
  --broadcast-mode block
```

### Vote Delegation

**Delegators can override your vote:**

```bash
# As a delegator (not validator operator):
pawd tx gov vote <proposal-id> yes \
  --from delegator-account \
  --chain-id paw-mainnet-1 \
  --gas auto \
  --gas-prices 0.001upaw \
  --broadcast-mode block

# This delegator's vote overrides your validator vote for their stake
```

**Implications:**
- If you don't vote, delegators inherit your abstention (hurts quorum)
- If you vote, delegators who don't vote inherit your choice
- If delegators vote, their choice overrides yours for their stake only
- **ALWAYS VOTE** - Delegators expect active governance participation

### Governance Best Practices

| Best Practice | Implementation | Impact |
|---------------|----------------|--------|
| **Vote on Every Proposal** | Set calendar reminders, automate alerts | Delegator trust |
| **Research Thoroughly** | Read forum discussions, technical specs | Informed decisions |
| **Communicate Publicly** | Tweet/blog your vote reasoning | Transparency |
| **Engage in Discussion** | Participate in governance forums | Community building |
| **Test Upgrades** | Run testnet before voting yes on upgrades | Network stability |
| **Consider Economic Impact** | Model parameter changes | Protect delegators |

---

## Appendix: Quick Reference

### Essential Commands

```bash
# Validator status
pawd query staking validator $(pawd keys show validator-operator --bech val -a)

# Signing info (missed blocks)
pawd query slashing signing-info $(pawd tendermint show-validator)

# Check if jailed
pawd query staking validator $(pawd keys show validator-operator --bech val -a) | jq '.jailed'

# Unjail
pawd tx slashing unjail --from validator-operator --chain-id paw-mainnet-1 --gas auto --gas-prices 0.001upaw --yes

# Withdraw rewards
pawd tx distribution withdraw-rewards $(pawd keys show validator-operator --bech val -a) --from validator-operator --commission --chain-id paw-mainnet-1 --gas auto --gas-prices 0.001upaw --yes

# Edit validator
pawd tx staking edit-validator --website "https://example.com" --details "Updated description" --from validator-operator --chain-id paw-mainnet-1 --gas auto --gas-prices 0.001upaw --yes

# Change commission
pawd tx staking edit-validator --commission-rate 0.08 --from validator-operator --chain-id paw-mainnet-1 --gas auto --gas-prices 0.001upaw --yes

# Vote on proposal
pawd tx gov vote <proposal-id> yes --from validator-operator --chain-id paw-mainnet-1 --gas auto --gas-prices 0.001upaw --yes
```

### Monitoring Checklist

**Daily:**
- [ ] Check missed blocks (`pawd query slashing signing-info`)
- [ ] Verify node syncing (`pawd status | jq '.SyncInfo.catching_up'`)
- [ ] Monitor disk space (`df -h`)
- [ ] Review logs for errors (`sudo journalctl -u pawd --since "1 hour ago" | grep -i error`)

**Weekly:**
- [ ] Review active governance proposals
- [ ] Check pending rewards
- [ ] Verify backup integrity
- [ ] Review Grafana dashboards

**Monthly:**
- [ ] Withdraw and compound rewards
- [ ] Review validator performance metrics
- [ ] Check for software updates
- [ ] Audit security configurations

**Quarterly:**
- [ ] Test disaster recovery procedures
- [ ] Review commission competitiveness
- [ ] Conduct key backup verification drill
- [ ] Update documentation

### Support Resources

- **Official Documentation**: https://docs.paw-chain.org
- **Validator Discord**: https://discord.gg/DBHTc2QV
- **Technical Support**: validators@paw-chain.org
- **Security Issues**: security@paw-chain.org
- **Governance Forum**: https://forum.paw-chain.org

### Related Documentation

- [VALIDATOR_QUICKSTART.md](./guides/VALIDATOR_QUICKSTART.md) - Initial setup guide
- [VALIDATOR_KEY_MANAGEMENT.md](./VALIDATOR_KEY_MANAGEMENT.md) - Comprehensive key security
- [UPGRADE_PROCEDURES.md](./UPGRADE_PROCEDURES.md) - Upgrade execution guide
- [DISASTER_RECOVERY.md](./DISASTER_RECOVERY.md) - Incident response and recovery

---

**Last Updated:** 2025-12-07
**Version:** 1.0
**Maintainer:** PAW Blockchain Validator Operations Team
