# PAW Blockchain - Deployment Quick Start Guide

**Version:** 1.0
**Last Updated:** 2025-12-07
**Audience:** Developers, DevOps Engineers, Node Operators

---

## Table of Contents

1. [Prerequisites](#prerequisites)
2. [Building from Source](#building-from-source)
3. [Running a Local Node](#running-a-local-node)
4. [Connecting to Testnet](#connecting-to-testnet)
5. [Basic Configuration](#basic-configuration)
6. [Verification and Testing](#verification-and-testing)
7. [Next Steps](#next-steps)

---

## Prerequisites

### System Requirements

**Minimum Specifications (Development/Testnet):**
- **CPU:** 4 cores (2.0+ GHz)
- **RAM:** 8 GB
- **Storage:** 100 GB SSD
- **Network:** 10 Mbps up/down
- **OS:** Linux (Ubuntu 22.04 LTS recommended), macOS, Windows (WSL2)

**Recommended Specifications (Production/Mainnet):**
- **CPU:** 8+ cores (3.0+ GHz)
- **RAM:** 32 GB
- **Storage:** 500 GB NVMe SSD (with room for growth)
- **Network:** 100 Mbps dedicated connection
- **OS:** Ubuntu 22.04 LTS (production standard)

### Software Dependencies

#### Required

```bash
# Go 1.23+ (required for building from source)
wget https://go.dev/dl/go1.23.1.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.23.1.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
echo 'export GOPATH=$HOME/go' >> ~/.bashrc
echo 'export GOBIN=$GOPATH/bin' >> ~/.bashrc
source ~/.bashrc
go version  # Verify installation: go version go1.23.1 linux/amd64

# Git (for cloning repository)
sudo apt update
sudo apt install -y git

# Build essentials
sudo apt install -y build-essential gcc make

# jq (for JSON processing in scripts)
sudo apt install -y jq
```

#### Optional (for development)

```bash
# buf CLI (for Protocol Buffer code generation)
go install github.com/bufbuild/buf/cmd/buf@latest

# golangci-lint (for code quality)
go install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.55.2

# Docker and Docker Compose (for containerized development)
sudo apt install -y docker.io docker-compose
sudo usermod -aG docker $USER
```

### Firewall Configuration

If running a node that accepts incoming connections:

```bash
# Allow P2P port (CometBFT)
sudo ufw allow 26656/tcp

# Allow RPC port (optional, for local queries only - DO NOT expose publicly)
# sudo ufw allow 26657/tcp  # ONLY on localhost or trusted networks

# Enable firewall
sudo ufw enable
```

**Security Note:** Never expose RPC (26657) or gRPC (9090) ports publicly without authentication.

---

## Building from Source

### 1. Clone the Repository

```bash
# Clone PAW blockchain repository
git clone https://github.com/paw-chain/paw.git
cd paw

# Verify you're on the latest stable release
git fetch --tags
git checkout $(git describe --tags `git rev-list --tags --max-count=1`)

# Or use main branch for development
git checkout main
```

### 2. Install Go Dependencies

```bash
# Download and verify Go module dependencies
go mod download
go mod verify

# Optional: Tidy dependencies
go mod tidy
```

### 3. Build the Binary

**Standard build:**

```bash
# Build pawd (daemon) and pawcli (CLI)
make build

# Binaries will be in ./build/
./build/pawd version
./build/pawcli version
```

**Optimized production build:**

```bash
# Build with size optimization
go build -ldflags="-s -w" -o ./build/pawd ./cmd/pawd
go build -ldflags="-s -w" -o ./build/pawcli ./cmd/pawcli
```

**Install system-wide (optional):**

```bash
# Install to $GOBIN (typically ~/go/bin)
make install

# Verify installation
pawd version
pawcli version

# Ensure $GOBIN is in PATH
echo $PATH | grep -q $GOBIN || echo 'export PATH=$PATH:$GOBIN' >> ~/.bashrc
```

### 4. Verify Build

```bash
# Check binary functionality
./build/pawd version --long

# Example output:
# name: paw
# server_name: pawd
# version: v0.1.0
# commit: a1b2c3d4e5f6...
# build_tags: netgo,ledger
# go: go1.23.1

# Verify binary architecture
file ./build/pawd
# Output: ./build/pawd: ELF 64-bit LSB executable, x86-64...
```

---

## Running a Local Node

### Single-Node Local Network

This section covers running a single validator node on your local machine for development and testing.

#### 1. Initialize Node

```bash
# Set chain ID and moniker (node name)
CHAIN_ID="paw-localnet-1"
MONIKER="my-local-node"

# Initialize node configuration and genesis file
./build/pawd init $MONIKER --chain-id $CHAIN_ID

# Configuration files created:
# ~/.paw/config/config.toml     (CometBFT configuration)
# ~/.paw/config/app.toml         (Application configuration)
# ~/.paw/config/genesis.json     (Chain genesis state)
# ~/.paw/config/node_key.json    (P2P node identity)
# ~/.paw/config/priv_validator_key.json  (Validator consensus key)
```

#### 2. Create Validator Key

```bash
# Create a key for the validator (stored in keyring)
./build/pawd keys add validator --keyring-backend test

# IMPORTANT: Save the mnemonic phrase shown!
# Example output:
# - address: paw1abc...xyz
#   name: validator
#   pubkey: '{"@type":"/cosmos.crypto.secp256k1.PubKey","key":"A..."}'
#   type: local
#
# **Important** write this mnemonic phrase in a safe place.
# It is the only way to recover your account if you ever forget your password.
#
# word1 word2 word3 ... word24
```

**Keyring Backends:**
- `test`: Unencrypted, for development only
- `file`: Encrypted, requires password
- `os`: OS native keyring (recommended for production)

#### 3. Fund Genesis Account

```bash
# Add genesis account with initial tokens
./build/pawd genesis add-genesis-account validator 1000000000000upaw --keyring-backend test

# upaw = micro-PAW (1 PAW = 1,000,000 upaw)
# 1000000000000 upaw = 1,000,000 PAW
```

#### 4. Create Genesis Transaction

```bash
# Create gentx (genesis transaction) to make this node a validator
./build/pawd genesis gentx validator 500000000000upaw \
  --chain-id $CHAIN_ID \
  --moniker $MONIKER \
  --commission-rate 0.10 \
  --commission-max-rate 0.20 \
  --commission-max-change-rate 0.01 \
  --min-self-delegation 1 \
  --keyring-backend test

# This delegates 500,000 PAW to the validator
# Remaining 500,000 PAW stays liquid for testing
```

#### 5. Collect Genesis Transactions

```bash
# Add gentx to genesis file
./build/pawd genesis collect-gentxs

# Validate final genesis file
./build/pawd genesis validate-genesis

# Genesis file validated: ~/.paw/config/genesis.json
```

#### 6. Configure Node (Optional)

```bash
# Enable API server (REST endpoints)
sed -i 's/enable = false/enable = true/' ~/.paw/config/app.toml

# Enable Prometheus metrics
sed -i 's/prometheus = false/prometheus = true/' ~/.paw/config/config.toml

# Allow CORS for local development (ONLY for local dev!)
sed -i 's/cors_allowed_origins = \[\]/cors_allowed_origins = ["*"]/' ~/.paw/config/config.toml

# Set minimum gas prices (prevent spam)
sed -i 's/minimum-gas-prices = ""/minimum-gas-prices = "0.001upaw"/' ~/.paw/config/app.toml
```

#### 7. Start the Node

```bash
# Start blockchain node
./build/pawd start

# Node will start producing blocks immediately
# Console output shows block production:
# INF committed state app_hash=... height=1 module=state
# INF indexed block height=1 module=txindex
# INF committed state app_hash=... height=2 module=state
```

**Run in background:**

```bash
# Using nohup
nohup ./build/pawd start > paw.log 2>&1 &

# Using systemd (recommended for production - see production guide)
```

#### 8. Verify Node is Running

```bash
# Query node status (in a new terminal)
./build/pawcli status

# Check current block height
./build/pawcli query block

# Query your account balance
./build/pawcli query bank balances $(./build/pawd keys show validator -a --keyring-backend test)
```

### Quick Local Network Script

For rapid development iteration, use the provided script:

```bash
# Clean, initialize, and start local network
./scripts/localnet-start.sh

# This script:
# 1. Removes old data (~/.paw)
# 2. Initializes fresh node
# 3. Creates validator key
# 4. Adds genesis account
# 5. Creates gentx
# 6. Validates genesis
# 7. Starts node
```

---

## Connecting to Testnet

### Join Existing Testnet

#### 1. Initialize Node

```bash
# Testnet configuration
CHAIN_ID="paw-mvp-1"
MONIKER="my-testnet-node"

# Initialize with testnet chain ID
./build/pawd init $MONIKER --chain-id $CHAIN_ID
```

#### 2. Download Official Genesis

```bash
# Download verified genesis file from official source
wget https://raw.githubusercontent.com/paw-chain/paw/main/networks/paw-mvp-1/genesis.json \
  -O ~/.paw/config/genesis.json

# Verify genesis checksum (published on GitHub)
EXPECTED_HASH="<official-genesis-sha256>"
ACTUAL_HASH=$(sha256sum ~/.paw/config/genesis.json | awk '{print $1}')

if [ "$ACTUAL_HASH" != "$EXPECTED_HASH" ]; then
  echo "ERROR: Genesis checksum mismatch!"
  exit 1
fi

echo "Genesis verified successfully"
```

**Critical Security:** Always verify genesis file integrity before joining a network.

#### 3. Configure Peers

```bash
# Add persistent peers (published on GitHub or testnet documentation)
PEERS="node1-id@node1.testnet.paw.network:26656,node2-id@node2.testnet.paw.network:26656"

# Update config.toml
sed -i "s/^persistent_peers = .*/persistent_peers = \"$PEERS\"/" ~/.paw/config/config.toml

# Optional: Add seed nodes
SEEDS="seed1-id@seed1.testnet.paw.network:26656"
sed -i "s/^seeds = .*/seeds = \"$SEEDS\"/" ~/.paw/config/config.toml
```

**Finding Peers:**
- Official documentation: https://docs.paw.network/testnet/peers
- GitHub: https://github.com/paw-chain/paw/tree/main/networks/paw-mvp-1
- Community Discord: #testnet-peers channel

#### 4. Configure State Sync (Recommended)

State sync allows fast synchronization without downloading full blockchain history.

```bash
# Get latest snapshot height and hash from RPC endpoint
RPC="https://rpc.testnet.paw.network:443"

LATEST_HEIGHT=$(curl -s $RPC/block | jq -r .result.block.header.height)
TRUST_HEIGHT=$((LATEST_HEIGHT - 1000))  # Trust height 1000 blocks back
TRUST_HASH=$(curl -s "$RPC/block?height=$TRUST_HEIGHT" | jq -r .result.block_id.hash)

# Update config.toml with state sync settings
sed -i.bak -E "s|^(enable[[:space:]]+=[[:space:]]+).*$|\1true|" ~/.paw/config/config.toml
sed -i.bak -E "s|^(rpc_servers[[:space:]]+=[[:space:]]+).*$|\1\"$RPC,$RPC\"|" ~/.paw/config/config.toml
sed -i.bak -E "s|^(trust_height[[:space:]]+=[[:space:]]+).*$|\1$TRUST_HEIGHT|" ~/.paw/config/config.toml
sed -i.bak -E "s|^(trust_hash[[:space:]]+=[[:space:]]+).*$|\1\"$TRUST_HASH\"|" ~/.paw/config/config.toml

echo "State sync configured: height=$TRUST_HEIGHT hash=$TRUST_HASH"
```

**State Sync Benefits:**
- Sync time: ~10 minutes vs. days for full sync
- Disk usage: Minimal (only recent state)
- Limitations: No historical block/transaction queries

#### 5. Set Minimum Gas Price

```bash
# Prevent spam transactions
sed -i 's/minimum-gas-prices = ""/minimum-gas-prices = "0.001upaw"/' ~/.paw/config/app.toml
```

#### 6. Start Node and Sync

```bash
# Start node
./build/pawd start

# Monitor sync progress (in another terminal)
./build/pawcli status | jq '.SyncInfo'

# Output:
# {
#   "latest_block_hash": "ABC123...",
#   "latest_app_hash": "DEF456...",
#   "latest_block_height": "123456",
#   "latest_block_time": "2025-12-07T10:30:00Z",
#   "catching_up": false  # false = fully synced
# }
```

**Sync Time Estimates:**
- State sync: 5-15 minutes
- Full sync from genesis: 12-48 hours (depending on network age)

#### 7. Verify Synchronization

```bash
# Check if node is caught up
CATCHING_UP=$(./build/pawcli status | jq -r '.SyncInfo.catching_up')

if [ "$CATCHING_UP" = "false" ]; then
  echo "Node is fully synced!"
else
  echo "Node is still syncing..."
fi

# Check current vs. network block height
LOCAL_HEIGHT=$(./build/pawcli status | jq -r '.SyncInfo.latest_block_height')
NETWORK_HEIGHT=$(curl -s https://rpc.testnet.paw.network/status | jq -r '.result.sync_info.latest_block_height')

echo "Local: $LOCAL_HEIGHT | Network: $NETWORK_HEIGHT"
```

---

## Basic Configuration

### Configuration Files Overview

PAW nodes use two primary configuration files:

#### 1. config.toml (CometBFT Consensus)

Location: `~/.paw/config/config.toml`

**Key settings:**

```toml
# Network configuration
[p2p]
laddr = "tcp://0.0.0.0:26656"       # P2P listen address
persistent_peers = ""                # Comma-separated peer list
seeds = ""                           # Seed nodes for peer discovery
max_num_inbound_peers = 40           # Max incoming connections
max_num_outbound_peers = 10          # Max outgoing connections

# RPC server configuration
[rpc]
laddr = "tcp://127.0.0.1:26657"     # RPC listen (localhost ONLY)
cors_allowed_origins = []            # CORS origins (restrict in production)

# Consensus configuration
[consensus]
timeout_commit = "5s"                # Block time
create_empty_blocks = true           # Produce blocks even when no txs

# Monitoring
[instrumentation]
prometheus = true                    # Enable Prometheus metrics
prometheus_listen_addr = ":26660"    # Metrics endpoint
```

**Security Best Practices:**
- Keep RPC on `127.0.0.1` (localhost only)
- Use firewall rules to restrict RPC access
- Limit CORS origins in production
- Enable TLS for public endpoints

#### 2. app.toml (Application Configuration)

Location: `~/.paw/config/app.toml`

**Key settings:**

```toml
# Base configuration
minimum-gas-prices = "0.001upaw"     # Minimum gas price (prevents spam)

# API server (REST endpoints)
[api]
enable = true                        # Enable REST API
address = "tcp://0.0.0.0:1317"      # API listen address
enabled-unsafe-cors = false          # CORS (false in production)

# gRPC server
[grpc]
enable = true                        # Enable gRPC
address = "0.0.0.0:9090"            # gRPC listen address

# State sync (for serving state sync to other nodes)
[state-sync]
snapshot-interval = 1000             # Snapshot every 1000 blocks
snapshot-keep-recent = 2             # Keep 2 recent snapshots
```

### Environment Variables

Configure node behavior via environment variables:

```bash
# Node data directory (default: ~/.paw)
export PAW_HOME="$HOME/.paw"

# Chain ID
export PAW_CHAIN_ID="paw-mvp-1"

# Custom ports (useful for running multiple nodes)
export PAW_RPC_PORT="26657"
export PAW_P2P_PORT="26656"
export PAW_GRPC_PORT="9090"
export PAW_REST_PORT="1317"

# Keyring backend
export PAW_KEYRING_BACKEND="test"  # test|file|os

# Network selection
export PAW_NETWORK="testnet"       # mainnet|testnet|devnet
```

**Using custom data directory:**

```bash
# All pawd commands accept --home flag
./build/pawd init mynode --home /custom/path --chain-id paw-test
./build/pawd start --home /custom/path
```

### Port Reference

| Service | Default Port | Protocol | Purpose |
|---------|--------------|----------|---------|
| P2P | 26656 | TCP | CometBFT peer-to-peer |
| RPC | 26657 | HTTP/WebSocket | CometBFT RPC (queries, txs) |
| Prometheus | 26660 | HTTP | Metrics export |
| gRPC | 9090 | gRPC | Cosmos SDK gRPC queries |
| REST API | 1317 | HTTP | Cosmos SDK REST API |

**Multiple Nodes on Same Machine:**

```bash
# Node 1: Default ports
./build/pawd start --home ~/.paw/node1

# Node 2: Custom ports
./build/pawd start --home ~/.paw/node2 \
  --p2p.laddr tcp://0.0.0.0:26666 \
  --rpc.laddr tcp://127.0.0.1:26667 \
  --grpc.address 0.0.0.0:9091 \
  --api.address tcp://0.0.0.0:1327
```

---

## Verification and Testing

### Health Checks

#### 1. Node Status

```bash
# Basic status check
./build/pawcli status

# Formatted output
./build/pawcli status | jq

# Check specific values
./build/pawcli status | jq -r '.NodeInfo.network'      # Chain ID
./build/pawcli status | jq -r '.SyncInfo.latest_block_height'  # Current height
./build/pawcli status | jq -r '.ValidatorInfo.voting_power'    # Voting power
```

#### 2. RPC Endpoint Health

```bash
# HTTP health check
curl http://localhost:26657/health

# Response: {"jsonrpc":"2.0","id":-1,"result":{}}

# Status endpoint
curl http://localhost:26657/status | jq

# Net info (peer count)
curl http://localhost:26657/net_info | jq '.result.n_peers'
```

#### 3. REST API Health

```bash
# Node info
curl http://localhost:1317/cosmos/base/tendermint/v1beta1/node_info | jq

# Latest block
curl http://localhost:1317/cosmos/base/tendermint/v1beta1/blocks/latest | jq
```

### Basic Operations

#### Query Blockchain State

```bash
# Get latest block
./build/pawcli query block

# Query account balance
ADDRESS=$(./build/pawd keys show validator -a --keyring-backend test)
./build/pawcli query bank balances $ADDRESS

# Query validators
./build/pawcli query staking validators

# Query transaction by hash
./build/pawcli query tx <TX_HASH>
```

#### Send Transaction

```bash
# Send tokens to another address
./build/pawcli tx bank send validator paw1recipient... 1000000upaw \
  --chain-id paw-localnet-1 \
  --keyring-backend test \
  --fees 1000upaw \
  --yes

# Delegate to validator
./build/pawcli tx staking delegate <validator-address> 1000000upaw \
  --from validator \
  --chain-id paw-localnet-1 \
  --keyring-backend test \
  --fees 1000upaw \
  --yes
```

### Monitoring

#### View Logs

```bash
# If running in foreground: logs output to console

# If running in background with nohup:
tail -f paw.log

# Filter for errors
tail -f paw.log | grep ERR

# Filter for specific module
tail -f paw.log | grep module=dex
```

#### Prometheus Metrics

```bash
# Check Prometheus endpoint
curl http://localhost:26660/metrics

# Key metrics to monitor:
# - tendermint_consensus_height: Current block height
# - tendermint_consensus_validators: Active validator count
# - tendermint_mempool_size: Pending transaction count
# - tendermint_p2p_peers: Connected peer count
```

### Common Issues and Solutions

#### Issue: Node won't start

```bash
# Check logs for errors
./build/pawd start 2>&1 | head -n 50

# Common causes:
# 1. Port already in use
lsof -i :26656  # Check if P2P port is taken

# 2. Invalid genesis file
./build/pawd genesis validate-genesis

# 3. Corrupted state
# Nuclear option: reset data (WARNING: deletes blockchain state)
./build/pawd tendermint unsafe-reset-all
```

#### Issue: Cannot connect to peers

```bash
# Check firewall
sudo ufw status

# Test connectivity to peer
nc -zv <peer-ip> 26656

# Check peer configuration
grep "persistent_peers" ~/.paw/config/config.toml

# View connection attempts
./build/pawcli status | jq '.NodeInfo.network'
curl http://localhost:26657/net_info | jq '.result.peers'
```

#### Issue: Sync is slow

```bash
# Check if state sync is enabled
grep "enable = true" ~/.paw/config/config.toml -A 5 | grep state-sync

# Verify peer quality
curl http://localhost:26657/net_info | jq '.result.peers[] | {moniker, send_queue, recv_queue}'

# Increase peer count
sed -i 's/max_num_outbound_peers = .*/max_num_outbound_peers = 20/' ~/.paw/config/config.toml
```

---

## Next Steps

### For Developers

1. **Explore Module APIs**
   - DEX: `docs/CLI_DEX.md`
   - Compute: `x/compute/README.md`
   - Oracle: `x/oracle/README.md`

2. **Run Integration Tests**
   ```bash
   make test-integration
   ```

3. **Enable Development Tools**
   ```bash
   make install-tools
   make install-hooks
   ```

### For Node Operators

1. **Production Deployment**
   - Read: `docs/DEPLOYMENT_PRODUCTION.md`
   - Configure systemd service
   - Set up monitoring (Prometheus/Grafana)
   - Implement backup strategy

2. **Become a Validator**
   - Read: `docs/guides/VALIDATOR_QUICKSTART.md`
   - Review: `docs/VALIDATOR_KEY_MANAGEMENT.md`
   - Understand slashing conditions
   - Set up monitoring and alerts

3. **Security Hardening**
   - Review: `docs/SECURITY_TESTING_RECOMMENDATIONS.md`
   - Configure firewall rules
   - Implement key management best practices
   - Set up disaster recovery procedures

### For Testnet Participants

1. **Get Testnet Tokens**
   - Faucet: https://faucet.testnet.paw.network
   - Community: Discord #testnet-faucet channel

2. **Experiment with Features**
   - DEX: Trade tokens, provide liquidity
   - Compute: Submit computation jobs
   - Oracle: Query price feeds

3. **Report Issues**
   - GitHub: https://github.com/paw-chain/paw/issues
   - Discord: #testnet-support

---

## Additional Resources

- **Official Documentation:** https://docs.paw.network
- **GitHub Repository:** https://github.com/paw-chain/paw
- **Technical Specification:** `docs/TECHNICAL_SPECIFICATION.md`
- **Whitepaper:** `docs/WHITEPAPER.md`
- **Cosmos SDK Docs:** https://docs.cosmos.network
- **CometBFT Docs:** https://docs.cometbft.com

---

**Questions or Issues?**

- GitHub Issues: https://github.com/paw-chain/paw/issues
- Discord Community: https://discord.gg/DBHTc2QV
- Developer Forum: https://forum.paw.network

---

**Document Version:** 1.0
**Last Review:** 2025-12-07
**Next Review:** 2026-03-07
