# PAW Multi-Validator Testnet Guide

Complete guide for running a 2, 3, or 4 validator PAW testnet with Docker.

## Quick Start (4 Validators)

```bash
# 1. Generate genesis
./scripts/devnet/setup-validators.sh 4

# 2. Start the network
docker compose -f compose/docker-compose.4nodes.yml up -d

# 3. Verify consensus (wait 30 seconds after startup)
sleep 30
curl -s http://localhost:26657/status | jq '.result.sync_info.latest_block_height'

# 4. Shut down
docker compose -f compose/docker-compose.4nodes.yml down -v
```

## Table of Contents

- [Prerequisites](#prerequisites)
- [Network Configurations](#network-configurations)
- [Starting the Testnet](#starting-the-testnet)
- [Monitoring the Network](#monitoring-the-network)
- [Shutting Down](#shutting-down)
- [Common Issues](#common-issues)
- [What NOT to Do](#what-not-to-do)

## Prerequisites

- Docker and Docker Compose installed
- Go 1.24+ (if building pawd binary)
- Python 3 with bech32 module: `pip3 install bech32`
- At least 4GB RAM available for Docker
- Ports available: 26657, 26667, 26677, 26687, 39090-39093, 1317, 1327, 1337, 1347

## Network Configurations

Choose your validator count:

| Validators | Docker Compose File | Genesis Command |
|------------|---------------------|-----------------|
| 2 | `compose/docker-compose.2nodes.yml` | `./scripts/devnet/setup-validators.sh 2` |
| 3 | `compose/docker-compose.3nodes.yml` | `./scripts/devnet/setup-validators.sh 3` |
| 4 | `compose/docker-compose.4nodes.yml` | `./scripts/devnet/setup-validators.sh 4` |

## Starting the Testnet

### Step 1: Clean Previous State (REQUIRED)

**ALWAYS clean before generating new genesis:**

```bash
# Stop any running containers
docker compose -f compose/docker-compose.4nodes.yml down -v

# Remove old genesis and keys
rm -f scripts/devnet/.state/*.json scripts/devnet/.state/*.mnemonic
```

**⚠️ WARNING:** Skipping this step will cause "genesis mismatch" errors.

### Step 2: Generate Genesis

Choose your validator count (2, 3, or 4):

```bash
# For 4 validators
./scripts/devnet/setup-validators.sh 4
```

**Expected output:**
```
=== Setting up 4-validator genesis ===
✓ Initialized base genesis
Setting up node1 validator...
  ✓ Created
...
Converted A9E665739AE529E1535FB7F3F5B18E120A5AC0D0 -> pawvalcons148nx2uu6u557z56lkleltvvwzg994sxs0fu643
...
Added 4 signing info entries
Genesis structure:
  Staking validators: 4
  CometBFT validators: 4
  Bond denom: upaw

=== ✓ Multi-validator genesis complete ===
```

### Step 3: Start Docker Containers

```bash
# For 4 validators
docker compose -f compose/docker-compose.4nodes.yml up -d
```

**Expected output:**
```
 Network compose_pawnet Created
 Volume compose_node1_data Created
 ...
 Container paw-node1 Started
 Container paw-node2 Started
 Container paw-node3 Started
 Container paw-node4 Started
```

### Step 4: Wait for Consensus (IMPORTANT)

**Wait at least 30 seconds** before checking status. Consensus requires:
- All nodes to initialize
- P2P connections to establish
- First block to be proposed and voted on

```bash
# Wait 30 seconds
sleep 30

# Check consensus
curl -s http://localhost:26657/status | jq '.result.sync_info | {latest_block_height, latest_block_time, catching_up}'
```

**Expected output:**
```json
{
  "latest_block_height": "6",
  "latest_block_time": "2025-12-14T02:32:34.260631618Z",
  "catching_up": false
}
```

## Monitoring the Network

### Check All Validator Nodes

```bash
docker ps --filter "name=paw-node"
```

**All nodes should show status `(healthy)` after 1-2 minutes.**

### Verify Block Production

```bash
# Watch blocks being produced (5 second intervals)
watch -n 5 'curl -s http://localhost:26657/status | jq -r ".result.sync_info.latest_block_height"'
```

**Block height should increase every 5 seconds.**

### Check Validator Participation

```bash
# See all active validators
curl -s http://localhost:26657/validators | jq '.result.total'

# Should output: "4" for a 4-validator network
```

### View Node Logs

```bash
# View logs for a specific node
docker logs paw-node1 -f

# Check for errors across all nodes
docker logs paw-node1 2>&1 | grep -i "error\|panic" | tail -20
docker logs paw-node2 2>&1 | grep -i "error\|panic" | tail -20
docker logs paw-node3 2>&1 | grep -i "error\|panic" | tail -20
docker logs paw-node4 2>&1 | grep -i "error\|panic" | tail -20
```

### Access Individual Nodes

| Node | RPC Port | gRPC Port | REST API |
|------|----------|-----------|----------|
| node1 | 26657 | 39090 | 1317 |
| node2 | 26667 | 39091 | 1327 |
| node3 | 26677 | 39092 | 1337 |
| node4 | 26687 | 39093 | 1347 |

```bash
# Query node2
curl -s http://localhost:26667/status | jq '.result.sync_info'

# Query node3
curl -s http://localhost:26677/status | jq '.result.sync_info'
```

## Shutting Down

### Graceful Shutdown (Preserves Data)

```bash
# Stop containers but keep volumes
docker compose -f compose/docker-compose.4nodes.yml down
```

**Use this if you want to restart the network later with the same state.**

### Complete Cleanup (Removes All Data)

```bash
# Stop containers and remove volumes
docker compose -f compose/docker-compose.4nodes.yml down -v
```

**Use this when:**
- Starting a fresh testnet
- Switching validator counts (2 ↔ 3 ↔ 4)
- Troubleshooting consensus issues

### Emergency Stop

```bash
# Force stop all containers
docker stop $(docker ps -q --filter "name=paw-node")
docker rm $(docker ps -aq --filter "name=paw-node")

# Clean volumes
docker volume rm $(docker volume ls -q --filter "name=compose_node")
```

## Common Issues

### Issue: "genesis hash mismatch"

**Cause:** Old genesis files not cleaned before regeneration.

**Solution:**
```bash
docker compose -f compose/docker-compose.4nodes.yml down -v
rm -f scripts/devnet/.state/*.json scripts/devnet/.state/*.mnemonic
./scripts/devnet/setup-validators.sh 4
docker compose -f compose/docker-compose.4nodes.yml up -d
```

### Issue: Nodes stay at height 0

**Symptoms:** `latest_block_height: "0"` after 2+ minutes.

**Cause:** Consensus not achieving >2/3 voting power.

**Solution:**
```bash
# Check all 4 nodes are running
docker ps --filter "name=paw-node"

# Check logs for errors
docker logs paw-node1 2>&1 | tail -50

# Restart the network
docker compose -f compose/docker-compose.4nodes.yml restart
```

### Issue: Port already in use

**Symptoms:** `Error starting userland proxy: listen tcp4 0.0.0.0:26657: bind: address already in use`

**Cause:** Previous containers not fully stopped, or another process using the port.

**Solution:**
```bash
# Find what's using the port
sudo lsof -i :26657

# Stop all PAW containers
docker compose -f compose/docker-compose.4nodes.yml down -v

# If needed, kill the process
sudo kill -9 <PID>
```

### Issue: "unhealthy" container status

**Symptoms:** `docker ps` shows `(unhealthy)` status.

**Cause:** Node took longer than healthcheck timeout to start, or RPC not responding.

**Solution:**
```bash
# Wait 2 minutes - nodes may still be initializing
sleep 120

# Check again
docker ps --filter "name=paw-node"

# If still unhealthy, check logs
docker logs paw-node1 2>&1 | tail -50
```

### Issue: "no validator signing info found"

**Symptoms:** Error in logs: `ERR error in proxyAppConn.FinalizeBlock err="no validator signing info found"`

**Cause:** Genesis file missing signing_infos (should be auto-generated by setup-validators.sh).

**Verification:**
```bash
# Check signing_infos exist in genesis
jq '.app_state.slashing.signing_infos | length' scripts/devnet/.state/genesis.json
# Should output: 4 (for 4 validators)
```

**Solution:** Regenerate genesis with the fixed setup-validators.sh script.

## What NOT to Do

### ❌ DON'T: Skip Cleaning Before Genesis Generation

```bash
# WRONG - will cause genesis hash mismatch
./scripts/devnet/setup-validators.sh 4
docker compose -f compose/docker-compose.4nodes.yml up -d
```

**RIGHT:**
```bash
docker compose -f compose/docker-compose.4nodes.yml down -v
rm -f scripts/devnet/.state/*.json scripts/devnet/.state/*.mnemonic
./scripts/devnet/setup-validators.sh 4
docker compose -f compose/docker-compose.4nodes.yml up -d
```

### ❌ DON'T: Mix Validator Counts

```bash
# WRONG - 4 validator genesis with 2 node docker-compose
./scripts/devnet/setup-validators.sh 4
docker compose -f compose/docker-compose.2nodes.yml up -d
```

**Network will fail because genesis expects 4 validators but only 2 nodes exist.**

### ❌ DON'T: Check Status Immediately After Startup

```bash
# WRONG - nodes haven't initialized yet
docker compose -f compose/docker-compose.4nodes.yml up -d
curl -s http://localhost:26657/status  # Will fail or show height 0
```

**RIGHT:**
```bash
docker compose -f compose/docker-compose.4nodes.yml up -d
sleep 30  # Wait for initialization
curl -s http://localhost:26657/status
```

### ❌ DON'T: Modify Genesis After collect-gentxs

**Once `collect-gentxs` runs, the genesis is finalized. Manual edits will break validation.**

### ❌ DON'T: Use `docker compose down` Without `-v` When Switching Configs

```bash
# WRONG - old data remains
docker compose -f compose/docker-compose.4nodes.yml down
./scripts/devnet/setup-validators.sh 2
docker compose -f compose/docker-compose.2nodes.yml up -d
```

**RIGHT:**
```bash
docker compose -f compose/docker-compose.4nodes.yml down -v  # -v removes volumes
./scripts/devnet/setup-validators.sh 2
docker compose -f compose/docker-compose.2nodes.yml up -d
```

### ❌ DON'T: Edit config.toml or app.toml After Genesis

**The init_node.sh script configures these automatically. Manual edits may break consensus.**

## Troubleshooting Checklist

If the network isn't working:

1. ✅ Cleaned old state before generating genesis
2. ✅ Genesis validator count matches Docker Compose file (2, 3, or 4)
3. ✅ All containers are running (`docker ps`)
4. ✅ Waited at least 30 seconds after startup
5. ✅ No port conflicts (`lsof -i :26657`)
6. ✅ Genesis has correct signing_infos (`jq '.app_state.slashing.signing_infos | length' scripts/devnet/.state/genesis.json`)
7. ✅ All nodes healthy (`docker ps` shows healthy status)

## Advanced: Testing Consensus Recovery

### Simulate Node Failure

```bash
# Stop one validator (network should continue with 3/4)
docker stop paw-node4

# Verify consensus continues
curl -s http://localhost:26657/status | jq '.result.sync_info.latest_block_height'

# Restart the stopped node (it will catch up)
docker start paw-node4
```

**4-validator network can tolerate 1 node failure (3/4 = 75% > 66.67% BFT threshold).**

### Simulate Network Partition

```bash
# Disconnect node4 from network
docker network disconnect compose_pawnet paw-node4

# Network continues with 3/4 validators

# Reconnect
docker network connect compose_pawnet paw-node4
```

## Files and Directories

### Generated State

- `scripts/devnet/.state/genesis.json` - Final genesis file
- `scripts/devnet/.state/node1.priv_validator_key.json` - Validator 1 consensus key
- `scripts/devnet/.state/node2.priv_validator_key.json` - Validator 2 consensus key
- `scripts/devnet/.state/node3.priv_validator_key.json` - Validator 3 consensus key
- `scripts/devnet/.state/node4.priv_validator_key.json` - Validator 4 consensus key
- `scripts/devnet/.state/node*_validator.mnemonic` - Validator mnemonics

**⚠️ NEVER commit these files to version control - they contain private keys.**

### Docker Volumes

- `compose_node1_data` - Node 1 blockchain data
- `compose_node2_data` - Node 2 blockchain data
- `compose_node3_data` - Node 3 blockchain data
- `compose_node4_data` - Node 4 blockchain data

## Support

If you encounter issues not covered here:

1. Check logs: `docker logs paw-node1 2>&1 | grep -i error`
2. Verify genesis structure: `jq '.app_state.staking.validators | length' scripts/devnet/.state/genesis.json`
3. Check network connectivity: `docker network inspect compose_pawnet`
4. Review the detailed error in HANDOFF.md or create a new issue

## Summary

**Working Setup:**
```bash
# Clean → Generate → Start → Wait → Verify → Use → Shutdown
docker compose -f compose/docker-compose.4nodes.yml down -v
rm -f scripts/devnet/.state/*.json scripts/devnet/.state/*.mnemonic
./scripts/devnet/setup-validators.sh 4
docker compose -f compose/docker-compose.4nodes.yml up -d
sleep 30
curl -s http://localhost:26657/status | jq '.result.sync_info'
# ... use the network ...
docker compose -f compose/docker-compose.4nodes.yml down -v
```

**The key to success: Always clean before regenerating genesis, and always wait 30 seconds after startup.**
