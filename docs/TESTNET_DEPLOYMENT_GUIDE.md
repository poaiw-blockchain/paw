# PAW Testnet Deployment Guide

This guide covers deploying and managing the PAW blockchain testnet.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Quick Start (Local 4-Validator Testnet)](#quick-start-local-4-validator-testnet)
- [Step-by-Step Setup](#step-by-step-setup)
- [Verification Procedures](#verification-procedures)
- [Common Issues and Solutions](#common-issues-and-solutions)
- [Adding External Validators](#adding-external-validators)

## Prerequisites

### Software Requirements

- Docker and Docker Compose
- Go 1.24+
- jq (for JSON processing)
- curl or wget

### Hardware Requirements (per validator)

- CPU: 4 cores minimum, 8 cores recommended
- RAM: 8GB minimum, 16GB recommended
- Disk: 100GB SSD minimum
- Network: 100 Mbps minimum

## Quick Start (Local 4-Validator Testnet)

This starts a fully functional 4-validator network on a single machine using Docker:

```bash
# 1. Clean any existing state
docker compose -f compose/docker-compose.4nodes.yml down -v
rm -f scripts/devnet/.state/*.json scripts/devnet/.state/*.mnemonic scripts/devnet/.state/*.id

# 2. Build the binary
go build -o pawd ./cmd/pawd

# 3. Generate 4-validator genesis
./scripts/devnet/setup-validators.sh 4

# 4. Start the network
docker compose -f compose/docker-compose.4nodes.yml up -d

# 5. Wait for network to stabilize (60 seconds)
sleep 60

# 6. Run smoke tests
COMPOSE_FILE=compose/docker-compose.4nodes.yml PAW_SMOKE_KEEP_STACK=1 ./scripts/devnet/smoke_tests.sh

# 7. Verify validators are signing
curl -s http://localhost:26657/block | jq '.result.block.last_commit.signatures | map(select(.signature)) | length'
# Should return: 4
```

## Step-by-Step Setup

### 1. Generate Multi-Validator Genesis

The `setup-validators.sh` script creates a genesis file with multiple validators:

```bash
./scripts/devnet/setup-validators.sh <NUM_VALIDATORS>
```

Where `NUM_VALIDATORS` can be 2, 3, or 4.

This script:
- Creates validator keys for each node
- Creates test accounts (smoke-trader, smoke-counterparty)
- Generates gentx for each validator
- Collects gentxs into a single genesis
- Adds validator signing info to prevent slashing errors
- Saves all keys and mnemonics to `scripts/devnet/.state/`

**Important files created:**
- `scripts/devnet/.state/genesis.json` - The multi-validator genesis
- `scripts/devnet/.state/node{1,2,3,4}_validator.mnemonic` - Validator key mnemonics
- `scripts/devnet/.state/node{1,2,3,4}.priv_validator_key.json` - Validator signing keys
- `scripts/devnet/.state/smoke-trader.mnemonic` - Test account mnemonic
- `scripts/devnet/.state/smoke-counterparty.mnemonic` - Test account mnemonic

### 2. Docker Compose Configuration

The `compose/docker-compose.4nodes.yml` file defines the 4-node network:

**Port mappings:**
- Node 1: RPC 26657, gRPC 39090, API 1317
- Node 2: RPC 26667, gRPC 39091, API 1327
- Node 3: RPC 26677, gRPC 39092, API 1337
- Node 4: RPC 26687, gRPC 39093, API 1347

Each container:
- Builds `pawd` binary on startup (or copies if pre-built)
- Runs `scripts/devnet/init_node.sh` to initialize the node
- Automatically discovers and connects to peer nodes
- Restores keys from mnemonics

### 3. Starting the Network

```bash
docker compose -f compose/docker-compose.4nodes.yml up -d
```

Monitor startup:
```bash
# Check container status
docker ps | grep paw-node

# Follow logs for a specific node
docker logs -f paw-node1

# Check all nodes are at the same height
curl -s http://localhost:26657/status | jq '.result.sync_info.latest_block_height'
```

### 4. Running Tests

**Smoke Tests:**
```bash
COMPOSE_FILE=compose/docker-compose.4nodes.yml PAW_SMOKE_KEEP_STACK=1 ./scripts/devnet/smoke_tests.sh
```

The smoke tests verify:
- RPC and API endpoints are accessible
- Bank transfers work
- DEX pool creation works
- DEX swaps execute successfully

**Individual test phases:**
```bash
# Run specific phases only
PAW_SMOKE_PHASES=bank,dex ./scripts/devnet/smoke_tests.sh
```

Available phases: `setup`, `bank`, `dex`, `swap`, `summary`

## Verification Procedures

### Check Validator Set

```bash
# Query active validators
curl -s http://localhost:26657/validators | jq '.result.validators | length'
# Should return: 4

# Get validator details
docker exec paw-node1 pawd query staking validators --home /root/.paw/node1
```

### Verify Consensus

```bash
# Check all validators are signing blocks
curl -s http://localhost:26657/block | jq '.result.block.last_commit.signatures | map(select(.signature)) | length'
# Should return: 4

# Check node is not catching up
curl -s http://localhost:26657/status | jq '.result.sync_info.catching_up'
# Should return: false
```

### Check Peer Connectivity

```bash
# View connected peers
curl -s http://localhost:26657/net_info | jq '.result.n_peers'
# Should return: 3 (for a 4-node network, each node connects to 3 peers)
```

### Verify Transactions

```bash
# Get test account addresses
TRADER=$(docker exec paw-node1 pawd keys show smoke-trader --keyring-backend test --home /root/.paw/node1 | awk '/address:/ {print $2}')

# Check balance
curl -s "http://localhost:1317/cosmos/bank/v1beta1/balances/${TRADER}" | jq '.balances'

# Send a transaction
docker exec paw-node1 pawd tx bank send smoke-trader <recipient> 1000000upaw \
  --chain-id paw-devnet \
  --keyring-backend test \
  --home /root/.paw/node1 \
  --yes \
  --fees 5000upaw
```

## Common Issues and Solutions

### Issue: Containers exit immediately

**Solution:** Check logs for errors:
```bash
docker logs paw-node1
cat scripts/devnet/.state/init_node1.log
```

Common causes:
- Genesis file not found or invalid
- Port conflicts
- Insufficient resources

### Issue: Nodes can't find peers

**Solution:** Ensure all node IDs are generated before startup:
```bash
ls scripts/devnet/.state/node*.id
# Should see: node1.id, node2.id, node3.id, node4.id
```

If missing, the init script will wait up to 10 seconds for peer IDs to appear.

### Issue: Prometheus server error

This is a warning, not an error. The telemetry server port may be in use. To suppress:
```bash
# The smoke tests filter this out automatically
docker exec paw-node1 pawd query dex pools --home /root/.paw/node1 2>&1 | grep -v "prometheus"
```

### Issue: "deadline must be set" for swaps

**Solution:** The DEX swap command requires a `--deadline` flag:
```bash
pawd tx dex swap 1 upaw 100000 ufoo 1 \
  --deadline 300 \  # 300 seconds from now
  --from trader \
  --yes
```

### Issue: Keys not found in keyring

**Solution:** Check mnemonics exist and restore keys:
```bash
# List mnemonics
ls scripts/devnet/.state/*.mnemonic

# Restore a key manually
docker exec -it paw-node1 pawd keys recover smoke-trader \
  --keyring-backend test \
  --home /root/.paw/node1 \
  < scripts/devnet/.state/smoke-trader.mnemonic
```

### Issue: High disk usage

**Solution:** The init logs can grow large. Clean them periodically:
```bash
# Truncate init logs
truncate -s 0 scripts/devnet/.state/init_*.log

# Or remove old backups
rm -rf scripts/devnet/.state.bak-*
```

## Adding External Validators

To allow external validators to join the network:

### 1. Share Network Artifacts

Publish these files from `networks/paw-devnet/`:
- `genesis.json` - The genesis file
- `genesis.sha256` - Checksum for verification
- `peers.txt` - Peer connection information

### 2. External Validator Setup

The external validator should:

```bash
# 1. Download artifacts
wget https://your-domain.com/paw-devnet/genesis.json
wget https://your-domain.com/paw-devnet/genesis.sha256

# 2. Verify genesis
sha256sum -c genesis.sha256

# 3. Initialize node
pawd init <moniker> --chain-id paw-devnet

# 4. Replace genesis
cp genesis.json ~/.paw/config/genesis.json

# 5. Configure persistent peers in ~/.paw/config/config.toml
# persistent_peers = "NODE_ID@YOUR_PUBLIC_IP:26656,..."

# 6. Start node
pawd start
```

### 3. Firewall Configuration

Ensure these ports are accessible:
- 26656 (P2P)
- 26657 (RPC, optional for external access)
- 9090 (gRPC, optional for external access)

### 4. Create Validator

Once the external node is synced:

```bash
# Create validator
pawd tx staking create-validator \
  --amount=1000000000upaw \
  --pubkey=$(pawd tendermint show-validator) \
  --moniker="<your-moniker>" \
  --chain-id=paw-devnet \
  --commission-rate="0.10" \
  --commission-max-rate="0.20" \
  --commission-max-change-rate="0.01" \
  --min-self-delegation="1" \
  --from=<your-key> \
  --fees=5000upaw
```

## Maintenance

### Restart Network

```bash
docker compose -f compose/docker-compose.4nodes.yml restart
```

### Stop Network (preserve state)

```bash
docker compose -f compose/docker-compose.4nodes.yml stop
```

### Clean Restart (wipe state)

```bash
docker compose -f compose/docker-compose.4nodes.yml down -v
# Then regenerate genesis and restart as in Quick Start
```

### Backup State

```bash
# Backup genesis and keys
tar -czf paw-devnet-backup-$(date +%Y%m%d).tar.gz \
  scripts/devnet/.state/genesis.json \
  scripts/devnet/.state/*.mnemonic \
  scripts/devnet/.state/*.priv_validator_key.json

# Backup docker volumes (while containers are stopped)
docker compose -f compose/docker-compose.4nodes.yml stop
tar -czf paw-devnet-volumes-$(date +%Y%m%d).tar.gz \
  -C /var/lib/docker/volumes \
  compose_node1_data compose_node2_data compose_node3_data compose_node4_data
```

## Monitoring

### Check Network Health

```bash
# Current block height
curl -s http://localhost:26657/status | jq '.result.sync_info.latest_block_height'

# Block time
curl -s http://localhost:26657/status | jq '.result.sync_info.latest_block_time'

# Validator power distribution
curl -s http://localhost:26657/validators | jq '.result.validators[] | {address, voting_power}'

# Total peers across all nodes
docker exec paw-node1 curl -s http://localhost:26657/net_info | jq '.result.n_peers'
```

### Performance Metrics

```bash
# Transaction throughput (approximate)
HEIGHT1=$(curl -s http://localhost:26657/status | jq -r '.result.sync_info.latest_block_height')
sleep 60
HEIGHT2=$(curl -s http://localhost:26657/status | jq -r '.result.sync_info.latest_block_height')
echo "Blocks per minute: $((HEIGHT2 - HEIGHT1))"
```

## Next Steps

- Set up monitoring with Prometheus/Grafana (see roadmap)
- Configure IBC channels to other testnets
- Deploy smart contracts
- Test governance proposals
- Perform load testing

## Support

For issues or questions:
1. Check the logs: `docker logs paw-node<N>`
2. Review init logs: `cat scripts/devnet/.state/init_node<N>.log`
3. Consult the main README and documentation
4. Open an issue on GitHub
