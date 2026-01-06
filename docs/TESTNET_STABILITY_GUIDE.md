# PAW Testnet Stability Guide

**Version**: 1.0
**Date**: 2026-01-06
**Status**: CRITICAL - Must follow for stable testnet

---

## Root Cause Analysis: Why Previous Testnets Failed

### Failure Pattern Identified

After 4 failed testnet attempts, the following issues were identified:

| Issue | Evidence | Impact |
|-------|----------|--------|
| **Single validator operation** | Only 1 of 4 genesis validators running | Chain halts if single validator goes offline |
| **Duplicate processes** | Two `pawd` instances running from different directories | Potential consensus conflicts |
| **Genesis/runtime mismatch** | Genesis has 4 validators, only 1 active at runtime | Loss of 75% voting power |
| **No persistent peers** | `persistent_peers = ""` in config.toml | Validators can't find each other |
| **No seed nodes** | `seeds = ""` in config.toml | New nodes can't bootstrap |
| **Archive vs active conflict** | `.paw-archive/` and `.paw/` both have data | Confusing state management |
| **Node downtime** | Monitoring tools (cosmos-exporter, explorer) failed when node offline | Appeared to be pruning issue but was actually availability issue |

### Pruning Myth Debunked

**Previous assumption**: Aggressive pruning caused monitoring/explorer failures.

**Actual finding**: After analyzing [cosmos-exporter source code](https://github.com/solarlabsteam/cosmos-exporter) and the PAW explorer:

1. **cosmos-exporter** queries ONLY current state via gRPC:
   - `Validators()`, `SigningInfos()`, `AllBalances()`, `DelegationTotalRewards()`
   - **None use historical block height headers**

2. **PAW explorer** queries:
   - `/block?height=X` → queries **blockstore.db** (NOT affected by pruning)
   - `/tx_search` → queries **tx_index.db** (NOT affected by pruning)
   - Current validator info → current state only

3. **What pruning actually affects**:
   - Only IAVL state queries with `x-cosmos-block-height` header
   - Example: "What was account X's balance at block 1000?"
   - **Monitoring tools don't make these queries**

**The errors were "connection refused" because the node was down, not pruning.**

### Why Single-Validator Testnets Fail

**CometBFT Consensus Requirement**: Requires >2/3 (67%) of voting power to produce blocks.

Your genesis file has 4 validators with equal stake (25% each):
- node1: 250,000,000,000 upaw (25%)
- node2: 250,000,000,000 upaw (25%)
- node3: 250,000,000,000 upaw (25%)
- node4: 250,000,000,000 upaw (25%)

**With only 1 validator running**: 25% voting power < 67% required = **CHAIN HALTS**

---

## The Fix: Multi-Validator Testnet Architecture

### Option A: True Multi-Validator Setup (Recommended)

Deploy 4 validators across 2 servers:

```
┌─────────────────────────────────────────────────────────────┐
│                    PAW Testnet Architecture                  │
├─────────────────────────────────────────────────────────────┤
│                                                              │
│  paw-testnet (54.39.103.49)      services-testnet (139.99)  │
│  ┌─────────────────────────┐    ┌─────────────────────────┐ │
│  │  Validator 1 (Primary)  │◄──►│  Validator 3            │ │
│  │  25% voting power       │    │  25% voting power       │ │
│  │  Port: 26656            │    │  Port: 27656            │ │
│  ├─────────────────────────┤    ├─────────────────────────┤ │
│  │  Validator 2            │◄──►│  Validator 4            │ │
│  │  25% voting power       │    │  25% voting power       │ │
│  │  Port: 26756            │    │  Port: 27756            │ │
│  └─────────────────────────┘    └─────────────────────────┘ │
│                                                              │
│  Total: 4 validators = 100% voting power                    │
│  Fault tolerance: Can lose 1 validator (75% > 67%)         │
└─────────────────────────────────────────────────────────────┘
```

### Option B: Single-Validator Genesis (Simpler, Less Resilient)

If you want a single-validator testnet for simplicity:

1. Create new genesis with only 1 validator
2. That validator has 100% voting power
3. Chain never halts due to missing validators
4. **Downside**: Zero fault tolerance

---

## Pre-Launch Checklist

### Phase 1: Infrastructure Verification

```bash
# Run on EACH server before launch

# 1. Verify binary exists and version
~/.paw/cosmovisor/genesis/bin/pawd version

# 2. Check no stale processes
pkill -f pawd
ps aux | grep pawd  # Should show nothing

# 3. Verify disk space (need >50GB free)
df -h /

# 4. Check time synchronization (CRITICAL for consensus)
timedatectl status
# Ensure NTP is active and time is synced

# 5. Verify network connectivity between validators
ping -c 3 10.10.0.2  # From services-testnet
ping -c 3 10.10.0.4  # From paw-testnet
```

### Phase 2: Genesis Configuration

```bash
# 1. Verify genesis hash matches across ALL nodes
sha256sum ~/.paw/config/genesis.json

# 2. Verify chain-id
jq -r '.chain_id' ~/.paw/config/genesis.json
# Expected: paw-testnet-1

# 3. Count validators in genesis
jq '.app_state.staking.validators | length' ~/.paw/config/genesis.json
# Should match number of validators you're running

# 4. Verify genesis time is in the past (or coordinate start time)
jq -r '.genesis_time' ~/.paw/config/genesis.json
```

### Phase 3: Node Configuration

**config.toml - CRITICAL SETTINGS**:

```toml
# Peer configuration (MUST SET THESE)
[p2p]
persistent_peers = "node1_id@10.10.0.2:26656,node2_id@10.10.0.2:26756,node3_id@10.10.0.4:27656,node4_id@10.10.0.4:27756"

# Enable peer exchange for discovery
pex = true

# Protect against double-signing after restart
double_sign_check_height = 10

# Reasonable timeouts
dial_timeout = "3s"
handshake_timeout = "20s"
```

**app.toml - ESSENTIAL SETTINGS**:

```toml
[api]
enable = true
address = "tcp://0.0.0.0:1317"

[grpc]
enable = true
address = "0.0.0.0:9091"

# State sync snapshots (for recovery)
[state-sync]
snapshot-interval = 1000
snapshot-keep-recent = 2
```

### Phase 4: Validator Key Management

```bash
# On each validator, verify keys exist
ls -la ~/.paw/config/priv_validator_key.json
ls -la ~/.paw/data/priv_validator_state.json

# CRITICAL: Ensure priv_validator_state.json height is correct
cat ~/.paw/data/priv_validator_state.json
# For fresh start: height should be "0"
# For restart: height should match or be below latest block
```

---

## Coordinated Launch Procedure

### Step 1: Clean Slate (All Nodes)

```bash
# Stop any running processes
pkill -f pawd
sleep 5

# Verify stopped
ps aux | grep pawd

# Reset data (ONLY for fresh start, NOT for restart)
# WARNING: This deletes all chain data!
~/.paw/cosmovisor/genesis/bin/pawd tendermint unsafe-reset-all --home ~/.paw
```

### Step 2: Copy Genesis (All Nodes)

```bash
# Ensure all nodes have IDENTICAL genesis
scp ~/.paw/config/genesis.json paw-testnet:~/.paw/config/
scp ~/.paw/config/genesis.json services-testnet:~/.paw/config/

# Verify hashes match
ssh paw-testnet "sha256sum ~/.paw/config/genesis.json"
ssh services-testnet "sha256sum ~/.paw/config/genesis.json"
```

### Step 3: Configure Peers (All Nodes)

```bash
# Get node IDs
NODE1_ID=$(ssh paw-testnet "~/.paw/cosmovisor/genesis/bin/pawd tendermint show-node-id --home ~/.paw")
NODE2_ID=$(ssh services-testnet "~/.paw/cosmovisor/genesis/bin/pawd tendermint show-node-id --home ~/.paw")

# Set persistent peers on each node
PEERS="${NODE1_ID}@10.10.0.2:26656,${NODE2_ID}@10.10.0.4:27657"

# Update config.toml on each node
ssh paw-testnet "sed -i 's/persistent_peers = \"\"/persistent_peers = \"${PEERS}\"/' ~/.paw/config/config.toml"
ssh services-testnet "sed -i 's/persistent_peers = \"\"/persistent_peers = \"${PEERS}\"/' ~/.paw/config/config.toml"
```

### Step 4: Start Validators (Coordinated)

```bash
# Start primary validator first
ssh paw-testnet "nohup ~/.paw/cosmovisor/genesis/bin/pawd start --home ~/.paw > ~/.paw/logs/node.log 2>&1 &"

# Wait 10 seconds for it to initialize
sleep 10

# Start secondary validator
ssh services-testnet "nohup ~/.paw/cosmovisor/genesis/bin/pawd start --home ~/.paw > ~/.paw/logs/node.log 2>&1 &"

# Wait for nodes to connect
sleep 30
```

### Step 5: Verify Health

```bash
# Check block height is advancing
ssh paw-testnet "curl -s http://127.0.0.1:26657/status | jq '.result.sync_info.latest_block_height'"

# Check peer count (should be > 0)
ssh paw-testnet "curl -s http://127.0.0.1:26657/net_info | jq '.result.n_peers'"

# Check validators are signing
ssh paw-testnet "curl -s http://127.0.0.1:26657/consensus_state | jq '.result.round_state.validators'"
```

---

## Monitoring Setup (Required)

### Minimum Monitoring

```bash
# Create simple health check script
cat > /usr/local/bin/paw-health-check.sh << 'EOF'
#!/bin/bash
HEIGHT=$(curl -s http://127.0.0.1:26657/status | jq -r '.result.sync_info.latest_block_height')
PEERS=$(curl -s http://127.0.0.1:26657/net_info | jq -r '.result.n_peers')
CATCHING_UP=$(curl -s http://127.0.0.1:26657/status | jq -r '.result.sync_info.catching_up')

echo "Height: $HEIGHT | Peers: $PEERS | Catching Up: $CATCHING_UP"

if [ "$CATCHING_UP" = "true" ]; then
    echo "WARNING: Node is catching up"
fi

if [ "$PEERS" -lt 1 ]; then
    echo "CRITICAL: No peers connected!"
fi
EOF
chmod +x /usr/local/bin/paw-health-check.sh

# Add to crontab for every 5 minutes
echo "*/5 * * * * /usr/local/bin/paw-health-check.sh >> /var/log/paw-health.log 2>&1" | crontab -
```

### Prometheus Metrics

Enable in `config.toml`:
```toml
[instrumentation]
prometheus = true
prometheus_listen_addr = ":26660"
```

### Critical Alerts

Set up alerts for:
1. Block height not advancing for > 60 seconds
2. Peer count drops to 0
3. Validator missing > 10 blocks
4. Disk usage > 80%
5. Memory usage > 90%

---

## Recovery Procedures

### Chain Halted - Single Validator Down

```bash
# 1. Identify which validator is down
curl -s http://paw-testnet:26657/consensus_state | jq '.result.round_state.validators'

# 2. Start the missing validator
ssh <missing-validator> "~/.paw/cosmovisor/genesis/bin/pawd start --home ~/.paw"

# 3. Chain should resume within ~30 seconds
```

### Chain Halted - Consensus Failure

```bash
# 1. Stop all validators
ssh paw-testnet "pkill -f pawd"
ssh services-testnet "pkill -f pawd"

# 2. Check for state mismatch
ssh paw-testnet "cat ~/.paw/data/priv_validator_state.json"
ssh services-testnet "cat ~/.paw/data/priv_validator_state.json"

# 3. If heights differ significantly, rollback the higher one
~/.paw/cosmovisor/genesis/bin/pawd rollback --home ~/.paw

# 4. Restart all validators
```

### AppHash Mismatch (Non-Determinism Bug)

```bash
# 1. STOP immediately - this is a bug in the application
pkill -f pawd

# 2. Check logs for the mismatch
grep -i "apphash" ~/.paw/logs/node.log

# 3. DO NOT restart until the bug is fixed
# 4. May need to export state and restart from a known-good height

# Export genesis from a known-good state
~/.paw/cosmovisor/genesis/bin/pawd export --height <last-good-height> > export.json
```

---

## Testnet Parameter Recommendations

For faster iteration on testnet, consider these parameter changes:

| Parameter | Mainnet Value | Testnet Value | Reason |
|-----------|---------------|---------------|--------|
| `unbonding_time` | 21 days | 1 hour | Faster testing |
| `voting_period` | 2 days | 10 minutes | Quick governance |
| `max_deposit_period` | 2 days | 10 minutes | Quick deposits |
| `downtime_jail_duration` | 10 minutes | 1 minute | Quick recovery |
| `signed_blocks_window` | 100 | 50 | Faster detection |

---

## Community-Preferred Node Architecture

Based on [Cosmos SDK documentation](https://docs.cosmos.network/main/user/run-node/interact-node) and industry standards:

| Node Type | Pruning Setting | Purpose | Your Infrastructure |
|-----------|-----------------|---------|---------------------|
| **Validator** | `pruning=custom`, `keep-recent=100` | Consensus, minimal disk | paw-testnet primary ✓ |
| **API/RPC Node** | `pruning=default` | Serve queries | services-testnet secondary ✓ |
| **Archive Node** | `pruning=nothing` | Full history for indexers | paw-testnet:26687 ✓ |
| **Indexer** | PostgreSQL | Historical queries | services-testnet PostgreSQL ✓ |

**Your current architecture is correct.** The issue was node availability, not architecture.

### Recommended Pruning Settings (Keep Current)

```toml
# app.toml - Validator Node (paw-testnet)
pruning = "custom"
pruning-keep-recent = "100"
pruning-interval = "10"
iavl-disable-fastnode = false  # CRITICAL: Must be false
```

This configuration:
- ✓ Works with cosmos-exporter (current state only)
- ✓ Works with block explorer (blocks stored separately)
- ✓ Works with Grafana/Prometheus metrics
- ✓ Minimizes disk usage
- ✗ Does NOT support historical balance queries (use PostgreSQL indexer instead)

---

## Critical Success Factors

### Must Have

1. **Multiple validators running** (minimum 3 for fault tolerance)
2. **Persistent peers configured** (validators can find each other)
3. **Identical genesis across all nodes** (same hash)
4. **Synchronized time** (NTP active)
5. **Health monitoring** (know when things break)
6. **systemd auto-restart** (node must stay online for monitoring)

### Should Have

1. **State sync enabled** (faster recovery)
2. **Prometheus metrics** (track performance)
3. **Log aggregation** (debug issues)
4. **Automated restarts** (systemd with restart=always)
5. **Backup validator keys** (disaster recovery)

### Nice to Have

1. **Sentry node architecture** (DDoS protection)
2. **Grafana dashboards** (visual monitoring)
3. **PagerDuty/Slack alerts** (immediate notification)
4. **Automated backup scripts** (state preservation)

---

## Quick Reference Commands

```bash
# Check if chain is producing blocks
curl -s http://127.0.0.1:26657/status | jq '.result.sync_info'

# Check peer connections
curl -s http://127.0.0.1:26657/net_info | jq '.result.n_peers'

# Check validator set
curl -s http://127.0.0.1:26657/validators | jq '.result.validators'

# Check consensus state
curl -s http://127.0.0.1:26657/consensus_state | jq '.result.round_state."height/round/step"'

# Watch logs in real-time
tail -f ~/.paw/logs/node.log

# Get node ID (for peer config)
~/.paw/cosmovisor/genesis/bin/pawd tendermint show-node-id --home ~/.paw

# Safe rollback (if consensus stuck)
~/.paw/cosmovisor/genesis/bin/pawd rollback --home ~/.paw
```

---

## Next Steps for This Restart

1. **Choose architecture**: Multi-validator (recommended) or single-validator
2. **Clean up server**: Remove `.paw-archive/`, ensure only one `.paw/` directory
3. **Generate new genesis** OR **run all 4 genesis validators**
4. **Configure persistent peers** on all nodes
5. **Enable monitoring** before launch
6. **Coordinate start** with all validators
7. **Verify health** within 5 minutes of launch

---

**Document Owner**: PAW Core Team
**Review Date**: 2026-01-06
**Next Review**: After successful testnet launch
