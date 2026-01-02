# PAW Blockchain - External Validator Onboarding Guide

**Version:** 1.1
**Last Updated:** 2026-01-12
**Network:** paw-testnet-1
**Status:** Public Testnet Open for External Validators

---

## Table of Contents

1. [Welcome](#welcome)
2. [Prerequisites](#prerequisites)
3. [Step-by-Step Setup](#step-by-step-setup)
4. [Network Information](#network-information)
5. [Validator Registration](#validator-registration)
6. [Post-Setup Checklist](#post-setup-checklist)
7. [Getting Testnet Tokens](#getting-testnet-tokens)
8. [Monitoring and Maintenance](#monitoring-and-maintenance)
9. [Common Issues](#common-issues)
10. [Support and Community](#support-and-community)

---

## Welcome

Thank you for your interest in running a validator on the PAW blockchain! This guide will walk you through everything needed to set up and operate a validator node on our public testnet.

### What You'll Learn

- How to provision and configure validator infrastructure
- How to join the PAW testnet network
- How to register as a validator and start earning rewards
- Best practices for security, monitoring, and maintenance

### Timeline Estimate

- **Basic setup:** 2-4 hours
- **Security hardening:** 1-2 hours
- **Monitoring setup:** 1-2 hours
- **Total:** 4-8 hours for complete production-ready setup

---

## Prerequisites

### Required Skills

- **Linux system administration:** Comfortable with command line, package management
- **Basic networking:** Understanding of IP addresses, ports, firewalls
- **Docker/systemd experience:** Managing services and containers
- **Git familiarity:** Cloning repositories, checking out versions

### Hardware Requirements

See [VALIDATOR_HARDWARE_REQUIREMENTS.md](./VALIDATOR_HARDWARE_REQUIREMENTS.md) for complete details.

**Testnet Minimum:**
```
CPU:     4 cores @ 2.5+ GHz (x86_64, AVX2 support)
RAM:     16 GB DDR4
Storage: 200 GB NVMe SSD (IOPS: 3000+ read, 1500+ write)
Network: 100 Mbps symmetric, <100ms latency to other validators
OS:      Ubuntu 22.04 LTS (recommended)
```

**Mainnet Recommended:**
```
CPU:     8+ cores @ 3.0+ GHz
RAM:     32 GB ECC
Storage: 500 GB+ NVMe SSD (enterprise grade)
Network: 1 Gbps symmetric, <50ms latency, DDoS protection
```

### Software Requirements

- Go 1.24+ (for building `pawd`)
- Git
- Make
- jq (JSON processor)
- curl

### Access Requirements

- SSH access to server with sudo privileges
- Static IP address or dynamic DNS
- Firewall access to configure ports

---

## Step-by-Step Setup

### Step 1: Provision Infrastructure

**Cloud Provider Options:**

| Provider | Instance Type | Monthly Cost | Notes |
|----------|--------------|--------------|-------|
| **AWS** | c6i.2xlarge (8 vCPU, 16GB) | ~$250 | EBS gp3 500GB |
| **GCP** | c2-standard-8 (8 vCPU, 32GB) | ~$280 | Balanced persistent disk |
| **Azure** | F8s_v2 (8 vCPU, 16GB) | ~$300 | Premium SSD |
| **Hetzner** | CCX23 (8 vCPU, 32GB) | ~$60 | Excellent value, EU-based |
| **DigitalOcean** | c-8 (8 vCPU, 16GB) | ~$160 | Simple management |

**Bare Metal:** Preferred for maximum sovereignty and performance

**Firewall Rules:**
```bash
# Required open ports:
# 26656 - P2P (TCP, all validators/sentries)
# 26657 - RPC (TCP, localhost or monitoring only, NOT public)
# 1317  - REST API (TCP, optional, localhost/monitoring only)
# 26660 - Prometheus metrics (TCP, monitoring server only)

sudo ufw default deny incoming
sudo ufw default allow outgoing
sudo ufw allow 22/tcp        # SSH (restrict to your IP in production)
sudo ufw allow 26656/tcp     # P2P
sudo ufw enable
```

### Step 2: Install Dependencies

```bash
# Update system
sudo apt update && sudo apt upgrade -y

# Install build dependencies
sudo apt install -y build-essential git curl jq unzip systemd wget

# Install Go 1.24.11
cd /tmp
wget https://go.dev/dl/go1.24.11.linux-amd64.tar.gz
sudo rm -rf /usr/local/go
sudo tar -C /usr/local -xzf go1.24.11.linux-amd64.tar.gz

# Add Go to PATH
cat <<'EOF' >> ~/.bashrc
export GOROOT=/usr/local/go
export GOPATH=$HOME/go
export PATH=$PATH:$GOROOT/bin:$GOPATH/bin
EOF
source ~/.bashrc

# Verify Go installation
go version
# Expected: go version go1.24.11 linux/amd64
```

### Step 3: Build PAW Binary

```bash
# Clone repository
cd ~
git clone https://github.com/paw-chain/paw.git
cd paw

# Checkout latest stable release (or specific version)
git fetch --tags
LATEST_TAG=$(git describe --tags `git rev-list --tags --max-count=1`)
git checkout $LATEST_TAG

# Build binary
make build

# Install globally
sudo install -m 0755 build/pawd /usr/local/bin/pawd

# Verify installation
pawd version
# Expected output with version, commit, build date
```

### Step 4: Initialize Node

```bash
# Set environment variables
export CHAIN_ID="paw-testnet-1"
export MONIKER="<your-validator-name>"  # Choose unique name

# Initialize node (creates ~/.paw directory structure)
pawd init "$MONIKER" --chain-id $CHAIN_ID

# Expected output:
# - Created ~/.paw/config/genesis.json (placeholder)
# - Created ~/.paw/config/config.toml
# - Created ~/.paw/config/app.toml
# - Generated priv_validator_key.json (consensus key)
# - Generated node_key.json (P2P key)
```

### Step 5: Download Network Configuration

```bash
# Download official genesis file
curl -L https://raw.githubusercontent.com/paw-chain/paw/main/networks/paw-testnet-1/genesis.json \
  > ~/.paw/config/genesis.json

# Download genesis checksum for verification
curl -L https://raw.githubusercontent.com/paw-chain/paw/main/networks/paw-testnet-1/genesis.sha256 \
  > /tmp/genesis.sha256

# Verify genesis integrity
cd ~/.paw/config
sha256sum -c /tmp/genesis.sha256
# Expected: genesis.json: OK

# Download persistent peers list
curl -L https://raw.githubusercontent.com/paw-chain/paw/main/networks/paw-testnet-1/peers.txt \
  > /tmp/peers.txt

# Extract peer list (format: node-id@ip:port,node-id@ip:port,...)
PERSISTENT_PEERS=$(cat /tmp/peers.txt | tr '\n' ',' | sed 's/,$//')
echo "Persistent peers: $PERSISTENT_PEERS"
```

### Step 6: Configure Node

```bash
# Edit config.toml
vi ~/.paw/config/config.toml
```

**Key settings to update:**

```toml
#######################################################
###           P2P Configuration Options             ###
#######################################################

# Address to listen for incoming connections
laddr = "tcp://0.0.0.0:26656"

# Comma separated list of seed nodes
seeds = ""

# Comma separated list of nodes to keep persistent connections to
persistent_peers = "<paste-from-peers.txt>"

# Maximum number of inbound peers
max_num_inbound_peers = 40

# Maximum number of outbound peers
max_num_outbound_peers = 10

# Enable peer exchange reactor (PEX)
pex = true

#######################################################
###         Consensus Configuration Options         ###
#######################################################

# Timeout settings (defaults are fine for testnet)
timeout_propose = "3s"
timeout_propose_delta = "500ms"
timeout_prevote = "1s"
timeout_prevote_delta = "500ms"
timeout_precommit = "1s"
timeout_precommit_delta = "500ms"
timeout_commit = "5s"

#######################################################
###          Instrumentation Configuration          ###
#######################################################

# Enable Prometheus metrics
prometheus = true
prometheus_listen_addr = ":26660"
```

**Edit app.toml:**

```bash
vi ~/.paw/config/app.toml
```

```toml
#######################################################
###         Base Configuration Options              ###
#######################################################

# Minimum gas prices (anti-spam)
minimum-gas-prices = "0.001upaw"

#######################################################
###          Pruning Configuration Options          ###
#######################################################

# Pruning strategy: default, nothing, everything, custom
pruning = "custom"
pruning-keep-recent = "100000"  # Keep last 100k blocks
pruning-interval = "10"

#######################################################
###         State Sync Snapshot Configuration       ###
#######################################################

# Snapshot interval (blocks)
snapshot-interval = 1000

# Number of recent snapshots to keep
snapshot-keep-recent = 2

#######################################################
###                  API Configuration              ###
#######################################################

[api]
# Disable public API for validators (security)
enable = false
swagger = false

[grpc]
# Enable gRPC for local queries
enable = true
address = "127.0.0.1:9090"
```

### Step 7: Create Validator Key

**CRITICAL SECURITY:** For testnet, we'll use software keys. For mainnet, ALWAYS use hardware wallets (Ledger) or HSM. See [VALIDATOR_KEY_MANAGEMENT.md](./VALIDATOR_KEY_MANAGEMENT.md).

```bash
# Create validator operator key
pawd keys add validator-operator --keyring-backend os

# IMPORTANT: Backup the 24-word mnemonic phrase immediately!
# Write it down on paper and store in a safe location
# This is the ONLY way to recover your account

# Get operator address
OPERATOR_ADDR=$(pawd keys show validator-operator -a --keyring-backend os)
echo "Operator address: $OPERATOR_ADDR"

# Get consensus public key (for create-validator transaction)
CONSENSUS_PUBKEY=$(pawd tendermint show-validator)
echo "Consensus pubkey: $CONSENSUS_PUBKEY"
```

**Backup Checklist:**
- [ ] Mnemonic written on paper (2+ copies)
- [ ] Mnemonic stored in fireproof safe
- [ ] priv_validator_key.json backed up (encrypted)
- [ ] node_key.json backed up
- [ ] Backup locations documented

### Step 8: Sync the Node

**Option A: Sync from Genesis (Slow)**

```bash
# Start node in foreground (for testing)
pawd start --home ~/.paw

# Monitor sync progress (in another terminal)
pawd status | jq '.SyncInfo'

# Wait until "catching_up": false
# This may take hours depending on chain length
```

**Option B: State Sync (Fast - Recommended)**

Use RPC endpoints listed in `docs/TESTNET_QUICK_REFERENCE.md`:

```bash
# Set RPC endpoints from the quick reference
RPC_1="https://<rpc-endpoint-1>"
RPC_2="https://<rpc-endpoint-2>"

# Get latest block height and trust hash
LATEST_HEIGHT=$(curl -s ${RPC_1}/block | jq -r .result.block.header.height)
BLOCK_HEIGHT=$((LATEST_HEIGHT - 2000))
TRUST_HASH=$(curl -s "${RPC_1}/block?height=$BLOCK_HEIGHT" | jq -r .result.block_id.hash)

# Update config.toml
sed -i.bak -E "s|^(enable[[:space:]]+=[[:space:]]+).*$|\1true| ; \
s|^(rpc_servers[[:space:]]+=[[:space:]]+).*$|\1\"${RPC_1},${RPC_2}\"| ; \
s|^(trust_height[[:space:]]+=[[:space:]]+).*$|\1$BLOCK_HEIGHT| ; \
s|^(trust_hash[[:space:]]+=[[:space:]]+).*$|\1\"$TRUST_HASH\"|" ~/.paw/config/config.toml

# Start node (will state sync)
pawd start --home ~/.paw
```

**Option C: Snapshot Restore (Fastest)**

If a snapshot is published (see `networks/paw-testnet-1/STATUS.md`), follow the provided download and extraction instructions before restarting the node.

### Step 9: Create systemd Service

```bash
# Create dedicated user
sudo useradd -m -s /bin/bash validator
sudo chown -R validator:validator ~/.paw

# Copy home directory if you initialized as different user
# sudo cp -r ~/.paw /home/validator/
# sudo chown -R validator:validator /home/validator/.paw

# Create systemd service
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

# Reload systemd
sudo systemctl daemon-reload

# Enable service (start on boot)
sudo systemctl enable pawd

# Start service
sudo systemctl start pawd

# Check status
sudo systemctl status pawd

# View logs
sudo journalctl -u pawd -f
```

### Step 10: Request Testnet Tokens

See [Getting Testnet Tokens](#getting-testnet-tokens) section below.

You'll need:
- Your operator address (from Step 7)
- Proof of node operation (systemctl status output)

Typical faucet allocation: 10,000,000 upaw (10 PAW)

### Step 11: Wait for Sync

```bash
# Check sync status
pawd status | jq '.SyncInfo.catching_up'

# If true: still syncing
# If false: fully synced, ready to create validator

# Check current block height
pawd status | jq '.SyncInfo.latest_block_height'

# Compare with network height (use RPC endpoint from quick reference)
RPC_1="https://<rpc-endpoint>"
curl -s ${RPC_1}/status | jq '.result.sync_info.latest_block_height'
```

**Sync is complete when:**
- `catching_up: false`
- `latest_block_height` matches network height
- Node is actively receiving blocks (height increasing)

---

## Network Information

- Chain ID: `paw-testnet-1`
- Denomination: `upaw`
- Artifacts: `networks/paw-testnet-1/` (`genesis.json`, `genesis.sha256`, `peers.txt`, templates, STATUS.md)
- Live endpoints and faucet: see `docs/TESTNET_QUICK_REFERENCE.md` and `networks/paw-testnet-1/STATUS.md`.

---

## Validator Registration

Once your node is fully synced and you have testnet tokens:

### Step 1: Verify Prerequisites

```bash
# 1. Node fully synced
pawd status | jq '.SyncInfo.catching_up'
# Expected: false

# 2. Sufficient balance
pawd query bank balances $OPERATOR_ADDR
# Expected: At least 1,000,000 upaw for self-delegation + gas fees

# 3. Consensus key available
pawd tendermint show-validator
# Should output your consensus public key
```

### Step 2: Create Validator Transaction

```bash
# Set parameters
MONIKER="<your-validator-name>"
OPERATOR_ADDR=$(pawd keys show validator-operator -a --keyring-backend os)
CONSENSUS_PUBKEY=$(pawd tendermint show-validator)
SELF_DELEGATION="1000000upaw"  # 1 PAW minimum
COMMISSION_RATE="0.10"  # 10%
COMMISSION_MAX_RATE="0.20"  # 20% max
COMMISSION_MAX_CHANGE_RATE="0.01"  # Max 1% change per day
MIN_SELF_DELEGATION="1000000"  # 1 PAW minimum

# Optional: Setup identity (Keybase.io)
IDENTITY=""  # Leave empty or add keybase.io 16-char identity
WEBSITE="https://your-validator-website.com"
SECURITY_CONTACT="security@your-domain.com"
DETAILS="Your validator description"

# Create validator
pawd tx staking create-validator \
  --from validator-operator \
  --amount $SELF_DELEGATION \
  --pubkey "$CONSENSUS_PUBKEY" \
  --moniker "$MONIKER" \
  --identity "$IDENTITY" \
  --website "$WEBSITE" \
  --security-contact "$SECURITY_CONTACT" \
  --details "$DETAILS" \
  --chain-id paw-testnet-1 \
  --commission-rate $COMMISSION_RATE \
  --commission-max-rate $COMMISSION_MAX_RATE \
  --commission-max-change-rate $COMMISSION_MAX_CHANGE_RATE \
  --min-self-delegation $MIN_SELF_DELEGATION \
  --gas auto \
  --gas-adjustment 1.5 \
  --gas-prices 0.001upaw \
  --keyring-backend os \
  --broadcast-mode block

# Expected output: Transaction hash and success message
```

### Step 3: Verify Validator Creation

```bash
# Get validator address
VALOPER_ADDR=$(pawd keys show validator-operator --bech val -a --keyring-backend os)

# Query validator
pawd query staking validator $VALOPER_ADDR

# Check validator appears in validator set
pawd query staking validators --output json | \
  jq -r '.validators[] | select(.description.moniker=="'$MONIKER'")'

# Check if bonded (active)
pawd query staking validator $VALOPER_ADDR | jq '.status'
# Expected: "BOND_STATUS_BONDED" (if in active set)
```

### Step 4: Verify Signing

```bash
# Check validator is signing blocks
pawd query slashing signing-info $(pawd tendermint show-validator)

# Monitor missed blocks counter
# Should remain low (<10)

# Check voting power
pawd query staking validator $VALOPER_ADDR | jq '.tokens'
```

---

## Post-Setup Checklist

### Security Hardening

- [ ] Firewall configured (only necessary ports open)
- [ ] SSH key-based authentication (password auth disabled)
- [ ] Fail2ban installed and configured
- [ ] priv_validator_key.json backed up (encrypted)
- [ ] Mnemonic phrase backed up (paper, multiple locations)
- [ ] OS updated and security patches applied
- [ ] Sentry node architecture considered (see [SENTRY_ARCHITECTURE.md](./SENTRY_ARCHITECTURE.md))
- [ ] HSM/tmkms planned for mainnet (see [VALIDATOR_KEY_MANAGEMENT.md](./VALIDATOR_KEY_MANAGEMENT.md))

### Monitoring Setup

- [ ] Prometheus metrics enabled (`config.toml`: `prometheus = true`)
- [ ] Grafana dashboard configured
- [ ] Alert rules configured (missed blocks, downtime)
- [ ] Uptime monitoring (external service like UptimeRobot)
- [ ] Log aggregation setup (Loki, ELK, or CloudWatch)
- [ ] Daily automated backups of keys

See [OBSERVABILITY.md](./OBSERVABILITY.md) and [DASHBOARDS_GUIDE.md](./DASHBOARDS_GUIDE.md) for complete setup.

### Operational Readiness

- [ ] Documentation of node IP, ports, and credentials
- [ ] Emergency contact list documented
- [ ] Runbook for common scenarios (restart, upgrade, unjail)
- [ ] Communication channel documented for validator announcements and incident coordination
- [ ] GitHub watch enabled for PAW repository (releases/upgrades)
- [ ] Calendar reminders for governance voting

### Performance Optimization

- [ ] Database tuning applied (LevelDB cache size)
- [ ] Filesystem optimized (noatime mount option)
- [ ] Resource limits tuned (ulimit, systemd)
- [ ] Pruning strategy configured
- [ ] State sync snapshots enabled

---

## Getting Testnet Tokens

Use the faucet listed in `docs/TESTNET_QUICK_REFERENCE.md` (and `networks/paw-testnet-1/STATUS.md` when available). Request only what you need for validator creation and fees.

### Verify Token Receipt

```bash
# Check balance
pawd query bank balances $OPERATOR_ADDR

# Expected output:
# balances:
# - amount: "10000000"
#   denom: upaw
```

---

## Monitoring and Maintenance

### Daily Checks

```bash
# Check node is running
sudo systemctl status pawd

# Check sync status
pawd status | jq '.SyncInfo.catching_up'

# Check missed blocks
pawd query slashing signing-info $(pawd tendermint show-validator)

# Check voting power
pawd query staking validator $VALOPER_ADDR | jq '.tokens'

# Check if jailed
pawd query staking validator $VALOPER_ADDR | jq '.jailed'
```

### Weekly Maintenance

- Review logs for errors: `sudo journalctl -u pawd --since "7 days ago" | grep -i error`
- Check disk space: `df -h`
- Update system packages: `sudo apt update && sudo apt upgrade -y`
- Review active governance proposals: `pawd query gov proposals`
- Withdraw and re-stake rewards (compound)

### Governance Participation

```bash
# List active proposals
pawd query gov proposals --status voting_period

# Get proposal details
pawd query gov proposal <proposal-id>

# Vote on proposal
pawd tx gov vote <proposal-id> yes \
  --from validator-operator \
  --chain-id paw-testnet-1 \
  --gas auto \
  --gas-prices 0.001upaw \
  --keyring-backend os \
  --yes
```

### Upgrade Procedures

When a chain upgrade is announced:

1. **Monitor release announcements** (GitHub releases and `networks/paw-testnet-1/STATUS.md`) for coordination details
2. **Read upgrade proposal** carefully
3. **Download new binary** or build from source (new version tag)
4. **Test on separate testnet** if possible
5. **Schedule upgrade** (coordinate with other validators)
6. **Execute upgrade** at designated block height (using Cosmovisor recommended)

See [UPGRADE_PROCEDURES.md](./UPGRADE_PROCEDURES.md) for detailed process.

---

## Common Issues

### Issue: Node Not Syncing

**Symptoms:**
- `catching_up: true` for extended period
- `latest_block_height` not increasing

**Diagnosis:**
```bash
# Check logs
sudo journalctl -u pawd -n 100 --no-pager

# Check peer count
pawd status | jq '.SyncInfo.latest_block_height'
curl -s localhost:26657/net_info | jq '.result.n_peers'
```

**Fixes:**
1. Verify `persistent_peers` configured correctly in `config.toml`
2. Check firewall allows port 26656
3. Verify genesis file matches network: `sha256sum ~/.paw/config/genesis.json`
4. Restart node: `sudo systemctl restart pawd`
5. Try state sync or snapshot restore

---

### Issue: Validator Jailed

**Symptoms:**
- Validator status shows `"jailed": true`
- Not earning rewards

**Diagnosis:**
```bash
pawd query staking validator $VALOPER_ADDR | jq '.jailed'
pawd query slashing signing-info $(pawd tendermint show-validator)
```

**Cause:** Missed too many blocks (downtime threshold exceeded)

**Fix:**
```bash
# 1. Ensure node is healthy and synced
pawd status | jq '.SyncInfo.catching_up'

# 2. Wait for jail period (typically 10 minutes)
pawd query slashing signing-info $(pawd tendermint show-validator) | jq '.jailed_until'

# 3. Unjail validator
pawd tx slashing unjail \
  --from validator-operator \
  --chain-id paw-testnet-1 \
  --gas auto \
  --gas-prices 0.001upaw \
  --keyring-backend os \
  --yes

# 4. Verify unjailed
pawd query staking validator $VALOPER_ADDR | jq '.jailed'
```

**Prevention:**
- Setup monitoring and alerts
- Ensure high uptime (99.9%+)
- Use sentry node architecture
- Have standby infrastructure ready

---

### Issue: Out of Disk Space

**Symptoms:**
- Node crashes with disk I/O errors
- `df -h` shows /root or /home at 100%

**Fix:**
```bash
# 1. Check disk usage
df -h
du -sh ~/.paw/data/*

# 2. Stop node
sudo systemctl stop pawd

# 3. Prune old data (if pruning enabled)
# Database compaction happens automatically

# 4. Clean logs
sudo journalctl --vacuum-time=7d

# 5. Expand disk (if cloud)
# AWS: Resize EBS volume + extend filesystem
# GCP: Resize persistent disk

# 6. Restart node
sudo systemctl start pawd
```

**Prevention:**
- Monitor disk usage with alerts (at 80%)
- Configure aggressive pruning (`pruning = "custom"`, `pruning-keep-recent = "1000"`)
- Provision adequate storage from start

---

### Issue: High Memory Usage / OOM Kills

**Symptoms:**
- Node crashes randomly
- `dmesg` shows "Out of memory: Killed process pawd"

**Diagnosis:**
```bash
# Check memory usage
free -h
top -o %MEM
```

**Fix:**
```bash
# 1. Increase RAM (resize instance)

# 2. Optimize config
# Edit ~/.paw/config/app.toml
# Reduce LevelDB cache:
# block-cache-size = 4194304  # 4 GB instead of 8 GB

# 3. Enable swap (temporary)
sudo fallocate -l 8G /swapfile
sudo chmod 600 /swapfile
sudo mkswap /swapfile
sudo swapon /swapfile

# 4. Restart node
sudo systemctl restart pawd
```

**Prevention:**
- Provision adequate RAM (32 GB recommended for mainnet)
- Monitor memory usage
- Optimize database caching based on available RAM

---

## Support and Community

- Issues and operational questions: https://github.com/paw-chain/paw/issues
- Security reports: follow `SECURITY.md` for responsible disclosure.
- When requesting help, include `pawd version`, sync status (`pawd status | jq '.SyncInfo'`), and recent logs (`journalctl -u pawd -n 200`).

### Validator Resources

- **Hardware:** [VALIDATOR_HARDWARE_REQUIREMENTS.md](./VALIDATOR_HARDWARE_REQUIREMENTS.md)
- **Key Management:** [VALIDATOR_KEY_MANAGEMENT.md](./VALIDATOR_KEY_MANAGEMENT.md)
- **Economics:** [VALIDATOR_ECONOMICS.md](./VALIDATOR_ECONOMICS.md)
- **Observability:** [OBSERVABILITY.md](./OBSERVABILITY.md) and [DASHBOARDS_GUIDE.md](./DASHBOARDS_GUIDE.md)
- **Operations:** [VALIDATOR_OPERATOR_GUIDE.md](./VALIDATOR_OPERATOR_GUIDE.md)
- **Sentry Topology:** [SENTRY_ARCHITECTURE.md](./SENTRY_ARCHITECTURE.md)

---

## Next Steps

Now that your validator is set up:

1. Complete observability setup: [OBSERVABILITY.md](./OBSERVABILITY.md) and [DASHBOARDS_GUIDE.md](./DASHBOARDS_GUIDE.md).
2. Review operational runbooks: [VALIDATOR_OPERATOR_GUIDE.md](./VALIDATOR_OPERATOR_GUIDE.md) and [SENTRY_ARCHITECTURE.md](./SENTRY_ARCHITECTURE.md) if exposing public endpoints.
3. Keep peers and templates current from `networks/paw-testnet-1/`.
4. Participate in governance once proposals are live.
5. Test backups and key recovery quarterly.

---

## Feedback

We're constantly improving this documentation. If you found errors, have suggestions, or want to contribute:

- **Open an issue:** https://github.com/paw-chain/paw/issues
- **Submit a PR:** https://github.com/paw-chain/paw/pulls
- **Email feedback:** docs@paw.network

Thank you for becoming a PAW validator.

---

**Last Updated:** 2026-01-12
**Version:** 1.1
**Maintainer:** PAW Validator Operations Team
**License:** Apache 2.0
