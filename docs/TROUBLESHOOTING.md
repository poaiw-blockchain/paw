# PAW Blockchain - Troubleshooting Guide

**Version:** 1.0
**Last Updated:** 2025-12-07
**Audience:** Node Operators, Validators, Developers, Users

---

## Table of Contents

1. [Node Startup Issues](#node-startup-issues)
2. [Synchronization Problems](#synchronization-problems)
3. [Consensus Issues](#consensus-issues)
4. [Memory and Disk Issues](#memory-and-disk-issues)
5. [P2P Connectivity Problems](#p2p-connectivity-problems)
6. [IBC Channel Issues](#ibc-channel-issues)
7. [DEX Module Errors](#dex-module-errors)
8. [Oracle Module Errors](#oracle-module-errors)
9. [Compute Module Errors](#compute-module-errors)
10. [Transaction Failures](#transaction-failures)
11. [Performance Issues](#performance-issues)
12. [Log Analysis](#log-analysis)
13. [Emergency Procedures](#emergency-procedures)

---

## Node Startup Issues

### Issue: Node won't start - Port already in use

**Symptoms:**
```
ERR Error listen tcp 127.0.0.1:26657: bind: address already in use
```

**Diagnosis:**
```bash
# Check if ports are in use
lsof -i :26656  # P2P port
lsof -i :26657  # RPC port
lsof -i :9090   # gRPC port
lsof -i :1317   # REST API port

# Alternative using netstat
netstat -tuln | grep -E '26656|26657|9090|1317'
```

**Solutions:**

1. **Kill existing pawd process:**
   ```bash
   # Find the process
   ps aux | grep pawd

   # Kill it (replace PID)
   kill -9 <PID>
   ```

2. **Use different ports:**
   ```bash
   ./build/pawd start \
     --p2p.laddr tcp://0.0.0.0:26666 \
     --rpc.laddr tcp://127.0.0.1:26667 \
     --grpc.address 0.0.0.0:9091 \
     --api.address tcp://0.0.0.0:1327
   ```

3. **Use custom data directory with different config:**
   ```bash
   ./build/pawd start --home ~/.paw-node2
   ```

---

### Issue: Genesis validation fails

**Symptoms:**
```
ERR Error: genesis file is invalid: error unmarshaling genesis state
ERR Error: genesis checksum mismatch
```

**Diagnosis:**
```bash
# Validate genesis file
./build/pawd genesis validate-genesis

# Check genesis file integrity
sha256sum ~/.paw/config/genesis.json

# Pretty-print to check for malformed JSON
jq . ~/.paw/config/genesis.json
```

**Solutions:**

1. **Re-download genesis file:**
   ```bash
   # Backup current genesis
   cp ~/.paw/config/genesis.json ~/.paw/config/genesis.json.backup

   # Download official genesis
   wget https://raw.githubusercontent.com/decristofaroj/paw/main/networks/testnet/genesis.json \
     -O ~/.paw/config/genesis.json

   # Verify checksum (get from official docs)
   sha256sum ~/.paw/config/genesis.json
   ```

2. **For localnet, regenerate genesis:**
   ```bash
   # Remove existing data
   ./build/pawd tendermint unsafe-reset-all

   # Re-initialize
   ./build/pawd init mynode --chain-id paw-localnet-1

   # Follow setup steps from DEPLOYMENT_QUICKSTART.md
   ```

---

### Issue: Missing or corrupted validator key

**Symptoms:**
```
ERR Error: failed to load private validator file
ERR Error: priv_validator_key.json: no such file or directory
```

**Diagnosis:**
```bash
# Check if validator key exists
ls -la ~/.paw/config/priv_validator_key.json

# Check if node key exists
ls -la ~/.paw/config/node_key.json
```

**Solutions:**

1. **For new node (no existing validator):**
   ```bash
   # Validator key is created during 'pawd init'
   ./build/pawd init mynode --chain-id paw-testnet-1
   ```

2. **If you have a backup:**
   ```bash
   # Restore from backup
   cp /path/to/backup/priv_validator_key.json ~/.paw/config/

   # Verify permissions
   chmod 600 ~/.paw/config/priv_validator_key.json
   ```

3. **If key is lost (CRITICAL):**
   ```
   ⚠️  WARNING: Lost validator keys cannot be recovered!

   - Your validator is permanently offline
   - Delegators should redelegate to active validators
   - You must create a NEW validator with a new key
   - Previous validator cannot be reactivated
   ```

---

### Issue: Database version mismatch

**Symptoms:**
```
ERR Error: database version mismatch
ERR Error: incompatible database version: expected X, got Y
```

**Solutions:**

1. **For minor version upgrades:**
   ```bash
   # Usually automatic, but check upgrade documentation
   cat docs/upgrades/vX.Y.Z.md
   ```

2. **For major version upgrades:**
   ```bash
   # Export state from old version
   ./pawd-old export > exported_state.json

   # Initialize new version with exported state
   ./pawd-new init mynode --chain-id paw-mainnet-1 --recover
   ./pawd-new genesis migrate exported_state.json > new_genesis.json
   mv new_genesis.json ~/.paw/config/genesis.json
   ```

3. **Nuclear option - full resync:**
   ```bash
   # ⚠️  WARNING: Deletes all blockchain data!
   ./build/pawd tendermint unsafe-reset-all

   # Re-sync from network or state sync
   # Follow steps in DEPLOYMENT_QUICKSTART.md
   ```

---

## Synchronization Problems

### Issue: Node stuck syncing / very slow sync

**Symptoms:**
```
INFO committed state height=12345 module=state
# Block height not increasing or very slow
```

**Diagnosis:**
```bash
# Check sync status
./build/pawcli status | jq '.SyncInfo'

# Compare local vs network height
LOCAL=$(./build/pawcli status | jq -r '.SyncInfo.latest_block_height')
NETWORK=$(curl -s https://rpc.testnet.paw.network/status | jq -r '.result.sync_info.latest_block_height')
echo "Local: $LOCAL | Network: $NETWORK | Behind: $((NETWORK - LOCAL)) blocks"

# Check peer quality
curl -s http://localhost:26657/net_info | jq '.result.peers[] | {moniker, send_queue, recv_queue}'
```

**Solutions:**

1. **Enable state sync (fastest):**
   ```bash
   # Stop node
   pkill pawd

   # Configure state sync
   RPC="https://rpc.testnet.paw.network:443"
   LATEST_HEIGHT=$(curl -s $RPC/block | jq -r .result.block.header.height)
   TRUST_HEIGHT=$((LATEST_HEIGHT - 1000))
   TRUST_HASH=$(curl -s "$RPC/block?height=$TRUST_HEIGHT" | jq -r .result.block_id.hash)

   sed -i.bak -E "s|^(enable[[:space:]]+=[[:space:]]+).*$|\1true|" ~/.paw/config/config.toml
   sed -i.bak -E "s|^(rpc_servers[[:space:]]+=[[:space:]]+).*$|\1\"$RPC,$RPC\"|" ~/.paw/config/config.toml
   sed -i.bak -E "s|^(trust_height[[:space:]]+=[[:space:]]+).*$|\1$TRUST_HEIGHT|" ~/.paw/config/config.toml
   sed -i.bak -E "s|^(trust_hash[[:space:]]+=[[:space:]]+).*$|\1\"$TRUST_HASH\"|" ~/.paw/config/config.toml

   # Reset data and restart
   ./build/pawd tendermint unsafe-reset-all
   ./build/pawd start
   ```

2. **Improve peer connections:**
   ```bash
   # Add more persistent peers
   nano ~/.paw/config/config.toml
   # Update: persistent_peers = "node1@ip1:26656,node2@ip2:26656,..."

   # Increase max peers
   sed -i 's/max_num_outbound_peers = .*/max_num_outbound_peers = 20/' ~/.paw/config/config.toml
   sed -i 's/max_num_inbound_peers = .*/max_num_inbound_peers = 50/' ~/.paw/config/config.toml

   # Restart node
   pkill pawd && ./build/pawd start
   ```

3. **Check resource constraints:**
   ```bash
   # Monitor CPU and I/O during sync
   top -p $(pgrep pawd)
   iotop -p $(pgrep pawd)

   # If disk I/O is bottleneck:
   # - Use NVMe SSD instead of SATA SSD
   # - Ensure database is on fastest disk
   # - Disable unnecessary services
   ```

---

### Issue: State sync fails

**Symptoms:**
```
ERR Error: snapshot restoration failed
ERR Error: no available snapshots
ERR Error: failed to verify snapshot
```

**Diagnosis:**
```bash
# Check if RPC servers are providing snapshots
curl -s https://rpc.testnet.paw.network:443/status | jq '.result.sync_info'

# Check state sync config
grep -A 10 "\[state-sync\]" ~/.paw/config/config.toml
```

**Solutions:**

1. **Try different RPC servers:**
   ```bash
   # Use multiple RPC endpoints
   RPC1="https://rpc1.testnet.paw.network:443"
   RPC2="https://rpc2.testnet.paw.network:443"

   sed -i.bak -E "s|^(rpc_servers[[:space:]]+=[[:space:]]+).*$|\1\"$RPC1,$RPC2\"|" \
     ~/.paw/config/config.toml
   ```

2. **Adjust trust height:**
   ```bash
   # Try with more recent trust height
   LATEST_HEIGHT=$(curl -s $RPC/block | jq -r .result.block.header.height)
   TRUST_HEIGHT=$((LATEST_HEIGHT - 500))  # Closer to current height

   # Update config and retry
   ```

3. **Fallback to full sync:**
   ```bash
   # Disable state sync
   sed -i 's/enable = true/enable = false/' ~/.paw/config/config.toml

   # This will sync from genesis (slow but reliable)
   ./build/pawd start
   ```

---

### Issue: Catching up is false but node is behind

**Symptoms:**
```json
{
  "catching_up": false,
  "latest_block_height": "123456"
}
```
But network is at height 150000.

**Diagnosis:**
```bash
# Check if node has peers
curl http://localhost:26657/net_info | jq '.result.n_peers'

# Check if consensus is working
tail -f ~/.paw/paw.log | grep -E "committed|height"
```

**Solutions:**

1. **Restart node to reconnect:**
   ```bash
   pkill pawd
   ./build/pawd start
   ```

2. **Force peer discovery:**
   ```bash
   # Add seed nodes
   SEEDS="seed1@seed.testnet.paw.network:26656"
   sed -i "s/^seeds = .*/seeds = \"$SEEDS\"/" ~/.paw/config/config.toml

   pkill pawd && ./build/pawd start
   ```

3. **Reset consensus state (preserves data):**
   ```bash
   # This doesn't delete blocks, only consensus state
   ./build/pawd tendermint unsafe-reset-all --keep-addr-book
   ./build/pawd start
   ```

---

## Consensus Issues

### Issue: Validator not signing blocks

**Symptoms:**
```
WARN This node is a validator but not in the active set
INFO Validator missed signing block height=X
```

**Diagnosis:**
```bash
# Check validator status
./build/pawcli query staking validator $(./build/pawd keys show validator --bech val -a --keyring-backend test)

# Check if jailed
./build/pawcli query slashing signing-info $(./build/pawd tendermint show-validator)

# Check voting power
./build/pawcli status | jq '.ValidatorInfo.voting_power'
```

**Solutions:**

1. **If voting power is 0:**
   ```bash
   # Check if bonded
   # Status should be "BOND_STATUS_BONDED"

   # If unbonded, need more stake
   # Ensure total delegation meets active set threshold
   ```

2. **If jailed:**
   ```bash
   # Unjail validator
   ./build/pawcli tx slashing unjail \
     --from validator \
     --chain-id paw-testnet-1 \
     --keyring-backend test \
     --fees 1000upaw \
     --yes

   # Wait for next block, check status
   ./build/pawcli query slashing signing-info $(./build/pawd tendermint show-validator)
   ```

3. **If node is out of active set:**
   ```bash
   # Check active set size
   ./build/pawcli query staking params | jq '.max_validators'

   # Check your ranking
   ./build/pawcli query staking validators --limit 200 | jq '.validators[] | {moniker, tokens}' | sort -k2 -rn

   # Need to increase stake to enter active set
   ```

---

### Issue: Double sign evidence

**Symptoms:**
```
ERR CONSENSUS FAILURE!!! err="duplicate vote"
ERR Evidence detected: DoubleSignEvidence
```

**CRITICAL - This is extremely serious!**

**Causes:**
- Running same validator key on multiple nodes
- Accidentally starting old validator after migration
- Corrupted consensus state

**Immediate Actions:**
```bash
# 1. STOP ALL VALIDATOR NODES IMMEDIATELY
pkill pawd

# 2. Identify which node is legitimate
# Check node_id and latest block height on each instance

# 3. Disable validator key on ALL BUT ONE node
# On nodes you want to disable:
chmod 000 ~/.paw/config/priv_validator_key.json

# 4. On the ONE legitimate node:
./build/pawd start

# 5. You will be slashed 5% and tombstoned
# Validator is permanently banned - CANNOT unjail
# Must create new validator with new key
```

**Prevention:**
- Never run same validator key on multiple nodes simultaneously
- Use remote signer (tmkms) for production
- Implement proper migration procedures
- Monitor for duplicate signing

---

### Issue: Stuck at height, not producing blocks

**Symptoms:**
```
INFO This node is a validator addr=XXXXX
INFO Committed state but no new blocks
```

**Diagnosis:**
```bash
# Check if majority of validators are online
./build/pawcli query staking validators | jq '.validators[] | select(.status=="BOND_STATUS_BONDED") | .description.moniker' | wc -l

# Check network consensus
curl http://localhost:26657/consensus_state | jq
```

**Solutions:**

1. **If network-wide issue:**
   ```
   - Coordinate with other validators on Discord/Telegram
   - Check if upgrade is scheduled
   - Wait for 2/3+ validators to come online
   ```

2. **If only your node:**
   ```bash
   # Check if you're on correct chain
   ./build/pawcli status | jq -r '.NodeInfo.network'

   # Ensure correct genesis
   sha256sum ~/.paw/config/genesis.json

   # Reset and resync
   ./build/pawd tendermint unsafe-reset-all
   ./build/pawd start
   ```

---

## Memory and Disk Issues

### Issue: Out of memory / OOM killer

**Symptoms:**
```
kernel: Out of memory: Killed process <PID> (pawd)
ERR Error: signal: killed
```

**Diagnosis:**
```bash
# Check memory usage
free -h
top -p $(pgrep pawd)

# Check for memory leaks
ps aux | grep pawd | awk '{print $6}'  # RSS memory in KB
```

**Solutions:**

1. **Increase system memory:**
   ```bash
   # Check current RAM
   free -h

   # Upgrade server RAM (recommended: 32GB for mainnet)
   ```

2. **Optimize node configuration:**
   ```bash
   # Reduce cache sizes
   nano ~/.paw/config/app.toml

   # Set:
   # iavl-cache-size = 781250  (default is higher)

   # Reduce max connections
   nano ~/.paw/config/config.toml
   # max_num_inbound_peers = 20
   # max_num_outbound_peers = 10
   ```

3. **Enable swap (emergency only):**
   ```bash
   # Create 8GB swap file
   sudo fallocate -l 8G /swapfile
   sudo chmod 600 /swapfile
   sudo mkswap /swapfile
   sudo swapon /swapfile

   # Make permanent
   echo '/swapfile none swap sw 0 0' | sudo tee -a /etc/fstab

   # Note: Swap is slower, should still upgrade RAM
   ```

---

### Issue: Disk full / no space left

**Symptoms:**
```
ERR Error: write /root/.paw/data: no space left on device
ERR Error: failed to commit batch to disk
```

**Diagnosis:**
```bash
# Check disk usage
df -h ~/.paw

# Check which directories are largest
du -sh ~/.paw/*
du -sh ~/.paw/data/*
```

**Solutions:**

1. **Clean up old data:**
   ```bash
   # Prune old state (if state sync is configured)
   # This requires restart

   # Enable pruning in app.toml
   nano ~/.paw/config/app.toml

   # Set:
   # pruning = "custom"
   # pruning-keep-recent = "100"
   # pruning-interval = "10"

   # Restart node
   pkill pawd && ./build/pawd start
   ```

2. **Rotate logs:**
   ```bash
   # Configure log rotation
   sudo nano /etc/logrotate.d/pawd

   # Add:
   /var/log/pawd.log {
       daily
       rotate 7
       compress
       delaycompress
       missingok
       notifempty
   }

   # Test
   sudo logrotate -f /etc/logrotate.d/pawd
   ```

3. **Increase disk space:**
   ```bash
   # Expand volume (cloud provider dependent)
   # For AWS: Modify volume in console, then:
   sudo growpart /dev/nvme0n1 1
   sudo resize2fs /dev/nvme0n1p1

   # Verify
   df -h
   ```

4. **Move data directory to larger disk:**
   ```bash
   # Stop node
   pkill pawd

   # Copy data to new disk
   sudo rsync -av ~/.paw/ /mnt/large-disk/paw/

   # Create symlink
   mv ~/.paw ~/.paw.backup
   ln -s /mnt/large-disk/paw ~/.paw

   # Restart
   ./build/pawd start
   ```

---

### Issue: Database corruption

**Symptoms:**
```
ERR Error: database is corrupted
ERR Error: checksum mismatch
panic: database corruption
```

**Diagnosis:**
```bash
# Check data directory
ls -lah ~/.paw/data/

# Check for filesystem errors
sudo dmesg | grep -i error
```

**Solutions:**

1. **Restore from backup (if available):**
   ```bash
   pkill pawd

   # Restore data directory
   rm -rf ~/.paw/data
   tar -xzf /backup/paw-data-YYYY-MM-DD.tar.gz -C ~/.paw/

   ./build/pawd start
   ```

2. **Resync from network:**
   ```bash
   # Nuclear option
   ./build/pawd tendermint unsafe-reset-all

   # Use state sync for faster recovery
   # Follow state sync setup in DEPLOYMENT_QUICKSTART.md

   ./build/pawd start
   ```

3. **Contact other validators for snapshot:**
   ```
   - Ask in validator chat for recent snapshot
   - Download and extract to ~/.paw/data/
   - Verify with your trusted validators
   ```

---

## P2P Connectivity Problems

### Issue: No peers / cannot connect to network

**Symptoms:**
```
INFO No peers available
WARN Dialing failed: connection refused
```

**Diagnosis:**
```bash
# Check peer count
curl http://localhost:26657/net_info | jq '.result.n_peers'

# Check listening address
curl http://localhost:26657/status | jq '.result.node_info.listen_addr'

# Test outbound connectivity
nc -zv <peer-ip> 26656
```

**Solutions:**

1. **Check firewall:**
   ```bash
   # Ensure P2P port is open
   sudo ufw status
   sudo ufw allow 26656/tcp

   # Test from external machine
   nc -zv <your-ip> 26656
   ```

2. **Add persistent peers:**
   ```bash
   # Get peer list from documentation or community
   PEERS="node1@ip1:26656,node2@ip2:26656,node3@ip3:26656"

   sed -i "s/^persistent_peers = .*/persistent_peers = \"$PEERS\"/" \
     ~/.paw/config/config.toml

   # Restart
   pkill pawd && ./build/pawd start
   ```

3. **Add seed nodes:**
   ```bash
   SEEDS="seed1@seed.testnet.paw.network:26656"

   sed -i "s/^seeds = .*/seeds = \"$SEEDS\"/" ~/.paw/config/config.toml

   pkill pawd && ./build/pawd start
   ```

4. **Check if behind NAT:**
   ```bash
   # If behind NAT, configure external address
   EXTERNAL_IP=$(curl -s ifconfig.me)

   sed -i "s/^external_address = .*/external_address = \"$EXTERNAL_IP:26656\"/" \
     ~/.paw/config/config.toml
   ```

---

### Issue: Peers keep disconnecting

**Symptoms:**
```
WARN Stopping peer for error: read timeout
INFO Dialed peer, marking as persistent
INFO Peer disconnected
```

**Diagnosis:**
```bash
# Check peer status
curl http://localhost:26657/net_info | jq '.result.peers[] | {moniker, send_queue, recv_queue, is_outbound}'

# Monitor disconnections
tail -f ~/.paw/paw.log | grep -E "peer|disconnect"
```

**Solutions:**

1. **Increase timeouts:**
   ```bash
   nano ~/.paw/config/config.toml

   # Increase:
   # timeout_commit = "5s" -> "10s"
   # timeout_propose = "3s" -> "5s"
   # flush_throttle_timeout = "100ms" -> "200ms"
   ```

2. **Reduce send/recv rates (if network is slow):**
   ```bash
   nano ~/.paw/config/config.toml

   # Set:
   # send_rate = 5120000  (5MB/s)
   # recv_rate = 5120000  (5MB/s)
   ```

3. **Check for network issues:**
   ```bash
   # Test network quality to peers
   mtr <peer-ip>  # Shows packet loss and latency

   # If high packet loss, switch to peers with better connectivity
   ```

---

## IBC Channel Issues

### Issue: IBC transfer stuck / not completing

**Symptoms:**
```
Transaction sent but tokens not received on destination chain
```

**Diagnosis:**
```bash
# Check IBC channels
./build/pawcli query ibc channel channels

# Check channel state
./build/pawcli query ibc channel end transfer <channel-id>

# Check packet commitment
./build/pawcli query ibc channel packet-commitments transfer <channel-id>
```

**Solutions:**

1. **Verify relayer is running:**
   ```
   - Check if relayer is active between chains
   - Verify relayer configuration
   - Check relayer logs for errors
   ```

2. **Manually relay packets (if you run relayer):**
   ```bash
   # Using Hermes relayer
   hermes tx packet-recv paw-testnet-1 transfer <channel-id>
   hermes tx packet-ack paw-testnet-1 transfer <channel-id>
   ```

3. **Check timeout height:**
   ```bash
   # If packet timed out, may need to refund
   ./build/pawcli tx ibc-transfer timeout-refund \
     --from <account> \
     --packet-sequence <seq> \
     --channel <channel-id>
   ```

---

### Issue: IBC channel not authorized

**Symptoms:**
```
ERR Error: unauthorized IBC channel
```

**Solutions:**

1. **Check authorized channels:**
   ```bash
   # DEX module
   ./build/pawcli query dex params | jq '.authorized_channels'

   # Oracle module
   ./build/pawcli query oracle params | jq '.authorized_channels'

   # Compute module
   ./build/pawcli query compute params | jq '.authorized_channels'
   ```

2. **Authorize channel via governance:**
   ```
   - Submit governance proposal to add channel
   - Wait for voting period
   - Coordinate with validators to vote
   - Channel authorized after proposal passes
   ```

---

## DEX Module Errors

### Error: Insufficient liquidity

**Error Code:** `ErrInsufficientLiquidity`

**Meaning:** Pool doesn't have enough tokens for requested swap

**Solutions:**
```bash
# Check pool reserves
./build/pawcli query dex pool <pool-id>

# Reduce swap amount
# Try swapping smaller amounts

# Or add liquidity first
./build/pawcli tx dex add-liquidity <pool-id> <amount-a> <amount-b> \
  --from <account> \
  --fees 1000upaw
```

---

### Error: Slippage too high

**Error Code:** `ErrSlippageTooHigh`

**Meaning:** Price moved beyond your tolerance during execution

**Solutions:**
```bash
# Increase slippage tolerance
# Use --slippage-tolerance flag (default 0.5%)

./build/pawcli tx dex swap <pool-id> <token-in> <amount-in> <min-amount-out> \
  --from <account> \
  --slippage-tolerance 1.0  # 1%

# Or split large swaps into smaller ones
# Or wait for better pool liquidity
```

---

### Error: Price impact too high

**Error Code:** `ErrPriceImpactTooHigh`

**Meaning:** Swap would move price by >5%, indicating large trade vs pool size

**Solutions:**
```bash
# Split swap into multiple smaller swaps over time
# Use limit orders instead (if available)
# Wait for pool to grow larger
# Use multiple pools if available
```

---

### Error: Circuit breaker triggered

**Error Code:** `ErrCircuitBreakerTriggered`

**Meaning:** Automatic safety mechanism activated due to anomaly

**Recovery:**
```
- Circuit breaker automatically resets after cooldown period (typically 1 hour)
- No manual intervention needed
- Check status:
  ./build/pawcli query dex circuit-breaker-status

- This is expected security behavior during:
  - Large price swings
  - High volume spikes
  - Suspicious activity patterns
```

---

### Error: Pool not found

**Error Code:** `ErrPoolNotFound`

**Solutions:**
```bash
# List all pools
./build/pawcli query dex pools

# Create new pool if needed
./build/pawcli tx dex create-pool <token-a> <token-b> <amount-a> <amount-b> \
  --from <account> \
  --fees 1000upaw
```

---

## Oracle Module Errors

### Error: Price not found

**Error Code:** `ErrPriceNotFound`

**Meaning:** No price data available for requested asset

**Solutions:**
```bash
# Check available assets
./build/pawcli query oracle prices

# Wait for next vote period (typically 30 seconds)

# If asset is not tracked:
# Submit governance proposal to add asset to oracle
```

---

### Error: Price expired

**Error Code:** `ErrPriceExpired`

**Meaning:** Cached price data is too old (stale)

**Solutions:**
```bash
# Check latest price update time
./build/pawcli query oracle price <asset>

# Wait for validators to submit fresh prices

# Check validator participation
./build/pawcli query oracle miss-counter

# If many validators missing, contact validator community
```

---

### Error: Insufficient votes

**Error Code:** `ErrInsufficientVotes`

**Meaning:** Not enough validators submitted prices (need >66%)

**Solutions:**
```
- Wait for more validators to submit
- Check network connectivity
- Verify validators are running oracle feeders
- Contact validator operators if persistent
```

---

### Error: Feeder not authorized

**Error Code:** `ErrFeederNotAuthorized`

**Meaning:** Feeder address not delegated by validator

**Solutions:**
```bash
# Validator must delegate feeder address
./build/pawcli tx oracle delegate-feeder <feeder-address> \
  --from validator \
  --chain-id paw-testnet-1 \
  --fees 1000upaw

# Verify delegation
./build/pawcli query oracle feeder-delegation <validator-address>
```

---

## Compute Module Errors

### Error: Provider not found

**Error Code:** `ErrProviderNotFound`

**Solutions:**
```bash
# Register as compute provider
./build/pawcli tx compute register-provider \
  --endpoint "https://compute.example.com" \
  --stake 1000000upaw \
  --from <account> \
  --fees 1000upaw

# List available providers
./build/pawcli query compute providers
```

---

### Error: Insufficient escrow

**Error Code:** `ErrInsufficientEscrow`

**Meaning:** Not enough tokens deposited for computation cost

**Solutions:**
```bash
# Query provider pricing
./build/pawcli query compute provider <provider-address>

# Calculate required amount:
# base_price + (cpu_cores * cpu_price) + (memory_gb * memory_price)

# Submit request with sufficient escrow
./build/pawcli tx compute submit-request \
  --container "alpine:latest" \
  --escrow 5000000upaw \
  --provider <provider-address> \
  --from <account>
```

---

### Error: Verification failed

**Error Code:** `ErrVerificationFailed`

**Meaning:** Computation result failed cryptographic verification

**Solutions:**
```bash
# Check verification logs
./build/pawcli query compute request <request-id>

# Provider may be penalized
# Submit new request with different provider

# Report to validators if suspicious
```

---

### Error: Rate limit exceeded

**Error Code:** `ErrRateLimitExceeded`

**Meaning:** Too many compute requests in time window

**Solutions:**
```bash
# Check rate limit params
./build/pawcli query compute params | jq '.rate_limit'

# Wait for rate limit reset (typically per hour)

# Upgrade account tier if available

# Batch multiple operations into single request
```

---

## Transaction Failures

### Error: Insufficient fees

**Symptoms:**
```
ERR Error: insufficient fees
```

**Solutions:**
```bash
# Check minimum gas prices
./build/pawcli query params subspace baseapp MinGasPrices

# Increase fees
./build/pawcli tx bank send <from> <to> <amount> \
  --fees 2000upaw  # Increased from 1000upaw

# Or use gas-prices flag
--gas-prices 0.002upaw
```

---

### Error: Account sequence mismatch

**Symptoms:**
```
ERR Error: account sequence mismatch, expected X, got Y
```

**Cause:** Transaction sent with wrong sequence number (nonce)

**Solutions:**
```bash
# Query current sequence
./build/pawcli query auth account <address>

# Let CLI auto-detect sequence
./build/pawcli tx bank send <from> <to> <amount> \
  --sequence auto

# Or explicitly set correct sequence
--sequence <correct-number>

# If multiple transactions, wait for previous to confirm
```

---

### Error: Transaction timeout

**Symptoms:**
```
ERR Error: timed out waiting for tx to be included in a block
```

**Solutions:**
```bash
# Increase timeout
export TIMEOUT=60s

# Check if mempool is full
curl http://localhost:26657/num_unconfirmed_txs

# Increase gas price to prioritize
./build/pawcli tx bank send <from> <to> <amount> \
  --fees 5000upaw  # Higher fees = higher priority
```

---

## Performance Issues

### Issue: RPC queries are slow

**Diagnosis:**
```bash
# Test query latency
time ./build/pawcli query bank balances <address>

# Check RPC endpoint load
curl http://localhost:26657/health
```

**Solutions:**

1. **Optimize node configuration:**
   ```bash
   nano ~/.paw/config/config.toml

   # Increase cache size
   # iavl-cache-size = 1562500

   # Enable query caching
   # max-open-connections = 1000
   ```

2. **Use dedicated query node:**
   ```
   - Separate consensus node from query node
   - Point queries to query-only node
   - Query node can use pruning to save space
   ```

3. **Add read replicas:**
   ```
   - Run multiple query nodes behind load balancer
   - Distribute query load
   ```

---

### Issue: High CPU usage

**Diagnosis:**
```bash
# Monitor CPU
top -p $(pgrep pawd)
mpstat -P ALL 1 10

# Profile node
go tool pprof http://localhost:26657/debug/pprof/profile
```

**Solutions:**

1. **Optimize consensus parameters:**
   ```bash
   nano ~/.paw/config/config.toml

   # Increase block time if acceptable
   # timeout_commit = "7s"  # Default 5s
   ```

2. **Reduce peer count:**
   ```bash
   # Fewer peers = less consensus overhead
   sed -i 's/max_num_inbound_peers = .*/max_num_inbound_peers = 20/' \
     ~/.paw/config/config.toml
   ```

3. **Upgrade hardware:**
   ```
   - Use CPU with higher single-core performance
   - Blockchain is often single-threaded for consensus
   ```

---

## Log Analysis

### Enable detailed logging

```bash
# Temporary (current session)
./build/pawd start --log_level debug

# Permanent
nano ~/.paw/config/config.toml
# log_level = "debug"  # info|debug|error
```

---

### Filter logs by module

```bash
# View specific module
tail -f ~/.paw/paw.log | grep "module=dex"
tail -f ~/.paw/paw.log | grep "module=oracle"
tail -f ~/.paw/paw.log | grep "module=compute"

# View errors only
tail -f ~/.paw/paw.log | grep "ERR"

# View consensus
tail -f ~/.paw/paw.log | grep "module=consensus"
```

---

### Common log patterns

**Healthy node:**
```
INF committed state app_hash=... height=12345 module=state
INF indexed block height=12345 module=txindex
INF Timed out dur=4976 height=12345 module=consensus round=0 step=1
INF received proposal height=12346 module=consensus
```

**Sync in progress:**
```
INF Executed block height=10000 module=state
INF committed state height=10000 module=state
# Heights increasing rapidly
```

**Network issues:**
```
WARN Dialing failed attempts=5 module=p2p
ERR Error on broadcastTxCommit err="timed out waiting"
```

**Consensus issues:**
```
ERR CONSENSUS FAILURE!!! err="..."
WARN This node is a validator but not in the active set
```

---

## Emergency Procedures

### Emergency: Chain halt (entire network stopped)

**Symptoms:**
- No blocks being produced across entire network
- All validators show same height

**Actions:**
1. **Join validator chat immediately**
2. **Do NOT restart your node or reset state**
3. **Coordinate with other validators for emergency upgrade or patch**
4. **Wait for instructions from core team**

---

### Emergency: Validator under attack

**Symptoms:**
- DDoS on RPC/P2P ports
- High network traffic
- Connection exhaustion

**Immediate Actions:**

1. **Enable sentry node architecture:**
   ```bash
   # Configure validator to only connect to sentry nodes
   nano ~/.paw/config/config.toml

   # On validator:
   pex = false  # Disable peer exchange
   persistent_peers = "<sentry1-id>@sentry1-private-ip:26656,<sentry2-id>@sentry2-private-ip:26656"

   # On sentries:
   pex = true
   private_peer_ids = "<validator-node-id>"  # Keep validator hidden
   ```

2. **Implement rate limiting:**
   ```bash
   # At firewall level (iptables)
   sudo iptables -A INPUT -p tcp --dport 26656 -m limit --limit 25/minute --limit-burst 100 -j ACCEPT
   sudo iptables -A INPUT -p tcp --dport 26656 -j DROP
   ```

3. **Use DDoS protection service:**
   ```
   - Cloudflare (for RPC/API)
   - AWS Shield
   - Sentry nodes with different IPs
   ```

---

### Emergency: State corruption across network

**Symptoms:**
- Multiple validators reporting state corruption
- Consensus halt at specific height
- Merkle root mismatches

**Actions:**
1. **Stop your validator immediately**
2. **Export state at last known good height**
   ```bash
   ./build/pawd export --height <last-good-height> > exported_state.json
   ```
3. **Join emergency validator call**
4. **Await coordinated recovery plan**

---

### Emergency: Slash event detected

**Symptoms:**
```
ERR Validator slashed for double signing
ERR Validator slashed for downtime
```

**Actions:**

1. **For double signing (CRITICAL):**
   ```bash
   # IMMEDIATELY stop ALL validator instances
   pkill pawd

   # Investigate which instance double signed
   # Check logs and node IDs

   # Validator is tombstoned - cannot recover
   # Must create new validator
   ```

2. **For downtime:**
   ```bash
   # Check why node was down
   # Fix underlying issue

   # Unjail validator
   ./build/pawcli tx slashing unjail \
     --from validator \
     --chain-id paw-testnet-1 \
     --fees 1000upaw

   # Monitor to prevent future downtime
   ```

---

## Getting Help

### Community Resources

- **Discord:** #node-support, #validator-support channels
- **GitHub Issues:** https://github.com/decristofaroj/paw/issues
- **Forum:** https://forum.paw.network
- **Documentation:** https://docs.paw.network

### When Reporting Issues

Include:
1. **Node version:** `./build/pawd version`
2. **Chain ID and height:** `./build/pawcli status | jq '.NodeInfo.network, .SyncInfo.latest_block_height'`
3. **Error logs:** Last 50 lines with error
4. **System info:** OS, CPU, RAM, Disk
5. **Configuration:** Relevant config sections
6. **Steps to reproduce:** What triggered the issue

### Security Issues

**DO NOT report security vulnerabilities publicly!**

- **Email:** security@paw.network (PGP key in docs/security/)
- **Bug Bounty:** See docs/BUG_BOUNTY.md

---

**Document Version:** 1.0
**Last Review:** 2025-12-07
**Next Review:** 2026-03-07
