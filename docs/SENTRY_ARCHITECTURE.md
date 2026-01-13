# PAW Sentry Node Architecture

Complete guide to sentry nodes for production-like testnet deployments.

## What Are Sentry Nodes?

**Sentry nodes** are non-validator full nodes that act as a protective layer between validators and the public network. They provide:

- **Validator protection**: Hide validators from direct public exposure
- **DDoS mitigation**: Sentries absorb attack traffic
- **Network redundancy**: Multiple entry points to the network
- **Load distribution**: Spread RPC/API load across sentries

### Validators vs Sentries

| Feature | Validator Node | Sentry Node |
|---------|---------------|-------------|
| Signs blocks | ✅ Yes | ❌ No |
| Requires staked tokens | ✅ Yes | ❌ No |
| Has `priv_validator_key.json` | ✅ Yes | ❌ No (removed) |
| Connects to validators | ✅ Via P2P mesh | ✅ Via persistent_peers |
| Accepts public connections | ⚠️ Not recommended | ✅ Yes (PEX enabled) |
| Max inbound peers | 40 (default) | 100 (higher for public) |
| Purpose | Consensus & signing | Relay & protection |

## Architecture Diagram

```
                    Public Network
                         │
         ┌───────────────┼───────────────┐
         │               │               │
    ┌────▼────┐     ┌────▼────┐         │
    │ Sentry1 │◄────┤ Sentry2 │         │
    │ (PEX)   │     │ (PEX)   │         │
    └────┬────┘     └────┬────┘         │
         │               │               │
    ┌────┴───────────────┴────┐          │
    │                          │          │
    │  Private Validator Zone  │          │
    │                          │          │
    │  ┌──────┐  ┌──────┐     │          │
    │  │Node1 │──│Node2 │     │          │
    │  └──┬───┘  └───┬──┘     │          │
    │     │          │         │          │
    │  ┌──┴───┐  ┌──┴──┐      │          │
    │  │Node3 │──│Node4│      │          │
    │  └──────┘  └─────┘      │          │
    │                          │          │
    └──────────────────────────┘          │
                                          │
                              Direct Access Blocked
```

## Network Topology

### IP Addressing (Docker Network)

**Subnet:** 172.22.0.0/24

**Validators** (Private):
- node1: 172.22.0.10
- node2: 172.22.0.11
- node3: 172.22.0.12
- node4: 172.22.0.13

**Sentries** (Public-facing):
- sentry1: 172.22.0.20
- sentry2: 172.22.0.21

### Port Mappings

**Validators** (limited external exposure):
- node1: RPC :26657, gRPC :39090, REST :1317
- node2: RPC :26667, gRPC :39091, REST :1327
- node3: RPC :26677, gRPC :39092, REST :1337
- node4: RPC :26687, gRPC :39093, REST :1347

**Sentries** (public entry points):
- sentry1: RPC :30658, P2P :30656, gRPC :39094, REST :1357
- sentry2: RPC :30668, P2P :30666, gRPC :39095, REST :1367

## Quick Start

### 1. Generate Genesis (Same as Before)

```bash
docker compose -f compose/docker-compose.4nodes-with-sentries.yml down -v
rm -f scripts/devnet/.state/*.json scripts/devnet/.state/*.mnemonic
./scripts/devnet/setup-validators.sh 4
```

### 2. Start Network (Validators + Sentries)

```bash
docker compose -f compose/docker-compose.4nodes-with-sentries.yml up -d
```

### 3. Wait for Consensus + Sync

```bash
sleep 60  # Validators start first, sentries sync after
```

### 4. Verify Validators

```bash
# Check validators are producing blocks
curl -s http://localhost:26657/status | jq '.result.sync_info.latest_block_height'

# All 4 validators signing
curl -s http://localhost:26657/validators | jq '.result.total'
```

### 5. Verify Sentries

```bash
# Sentry1 catching up
curl -s http://localhost:30658/status | jq '.result.sync_info'

# Sentry2 catching up
curl -s http://localhost:30668/status | jq '.result.sync_info'

# Check sentry peer connections (should see 4 validators + 1 other sentry)
curl -s http://localhost:30658/net_info | jq '.result.n_peers'
```

## P2P Configuration

### Validator Configuration

Validators connect to:
- Other validators (P2P mesh)
- Sentries (for redundancy)

**Config:** `persistent_peers` includes all nodes

**Security note:** In production, validators should use `private_peer_ids` to hide from sentries' peer exchange.

### Sentry Configuration

Sentries connect to:
- **All validators** via `persistent_peers`
- **Other sentries** via `persistent_peers`
- **Public peers** via PEX (peer exchange)

**Key settings in `config.toml`:**
```toml
pex = true                        # Enable peer exchange (accept public peers)
max_num_inbound_peers = 100       # Higher limit for public connections
max_num_outbound_peers = 20       # Actively discover new peers
persistent_peers = "<validator1>,<validator2>,<validator3>,<validator4>,<sentry2>"
```

## Testing Scenarios

### 1. Basic Connectivity Test

```bash
# Query validator via sentry1
curl -s http://localhost:30658/status | jq '.result.node_info.network'

# Query validator via sentry2
curl -s http://localhost:30668/status | jq '.result.node_info.network'

# Both should return: "paw-devnet"
```

### 2. Load Distribution Test

```bash
# Send transactions via different entry points
# Via sentry1
pawd tx bank send validator <recipient> 1000upaw \
  --node http://localhost:30658 \
  --chain-id paw-devnet \
  --keyring-backend test \
  --yes

# Via sentry2
pawd tx bank send validator <recipient> 1000upaw \
  --node http://localhost:30668 \
  --chain-id paw-devnet \
  --keyring-backend test \
  --yes
```

### 3. Sentry Failure Resilience

```bash
# Stop sentry1
docker stop paw-sentry1

# Network should still be accessible via sentry2
curl -s http://localhost:30668/status | jq '.result.sync_info.latest_block_height'

# Validators should still be reaching consensus (check via node1)
curl -s http://localhost:26657/status | jq '.result.sync_info.latest_block_height'

# Restart sentry1
docker start paw-sentry1

# Wait for sync
sleep 30

# Verify sentry1 caught up
curl -s http://localhost:30658/status | jq '.result.sync_info.catching_up'
# Should return: false
```

### 4. Validator Isolation Test

```bash
# Sentries should have connections to validators
curl -s http://localhost:30658/net_info | jq '.result.peers[] | select(.node_info.moniker | test("node[1-4]"))'

# Public clients should NOT see validators directly (in production with private_peer_ids)
# They should only see sentries
```

### 5. PEX Discovery Test

```bash
# Check sentry1's address book (accumulated via PEX)
docker exec paw-sentry1 cat /root/.paw/sentry1/config/addrbook.json | jq '.addrs | length'

# Should grow over time as sentries discover peers
```

## Real-World Testing Scenarios

### Scenario 1: Public Testnet Simulation

**Setup:** Expose sentry RPC ports to internet (or local network)

```bash
# Docker Compose: map sentry ports to host
# Users connect ONLY to sentries, never validators
pawd status --node http://<your-ip>:30658
```

**Benefits:**
- Validators remain hidden
- Sentries handle all public traffic
- DDoS attacks hit sentries, not validators

### Scenario 2: Geographic Distribution

**Setup:** Run sentries in different regions/data centers

```bash
# Sentry1: US East
# Sentry2: Europe
# Validators: Protected region
```

**Benefits:**
- Reduced latency for users in different regions
- Redundancy across geographic zones
- Validators can be in secure facility

### Scenario 3: High-Frequency Trading Endpoint

**Setup:** Use sentries as dedicated API endpoints for trading bots

```bash
# Configure sentry with higher rate limits
# Point trading bots to sentry RPC/gRPC
# Validators protected from excessive query load
```

### Scenario 4: Network Partition Testing

```bash
# Simulate network split: isolate sentry1 from validators
docker network disconnect pawnet paw-sentry1

# Validators should still reach consensus (3/4 BFT)
# Sentry2 still serves clients

# Reconnect
docker network connect pawnet paw-sentry1

# Sentry1 resyncs via fast-sync
```

## Advanced Configuration

### Production Hardening

**Validators** (`config.toml`):
```toml
# Hide validators from public peer discovery
private_peer_ids = "<sentry1_id>,<sentry2_id>"

# Only connect to sentries (isolate from public)
persistent_peers = "<sentry1>,<sentry2>"
unconditional_peers = "<sentry1>,<sentry2>"

# Disable PEX
pex = false

# Reduce peer limits
max_num_inbound_peers = 10
max_num_outbound_peers = 5
```

**Sentries** (`config.toml`):
```toml
# Accept public connections
pex = true

# Higher limits for public traffic
max_num_inbound_peers = 200
max_num_outbound_peers = 50

# Always connect to validators
persistent_peers = "<validator1>,<validator2>,<validator3>,<validator4>"
unconditional_peers = "<validator1>,<validator2>,<validator3>,<validator4>"
```

### Rate Limiting (App-Level)

**For sentries serving public APIs**, consider:
- Nginx reverse proxy with rate limiting
- Traefik with rate limit middleware
- Custom rate limiting in application layer

### Monitoring Recommendations

**Validators:**
- Block production rate
- Peer count (should be stable)
- Memory/CPU usage
- Disk I/O (state growth)

**Sentries:**
- Sync status (should stay in sync)
- Peer count (should be higher)
- RPC request rate
- Network bandwidth (higher than validators)

**Alerts:**
- Sentry sync lag > 100 blocks
- Sentry peer count < 5
- Validator peer count ≠ expected (sentries + validators)

## Common Issues

### Sentry Not Syncing

**Symptom:** Sentry stuck at old block height

**Diagnosis:**
```bash
docker logs paw-sentry1 2>&1 | grep -i error
curl -s http://localhost:30658/status | jq '.result.sync_info'
```

**Fixes:**
1. Verify persistent_peers configured correctly
2. Check validators are producing blocks
3. Ensure genesis matches validators
4. Restart sentry: `docker restart paw-sentry1`

### Sentry Has No Peers

**Symptom:** `n_peers` = 0

**Diagnosis:**
```bash
curl -s http://localhost:30658/net_info | jq '.result.n_peers'
docker logs paw-sentry1 2>&1 | grep "persistent peer"
```

**Fixes:**
1. Wait for validators to fully start (30-60s)
2. Verify node IDs exist: `ls scripts/devnet/.state/*.id`
3. Check network connectivity: `docker network inspect pawnet`

### Priv Validator Key Error on Sentry

**Symptom:** `ERR priv_val_key.json not found`

**Root cause:** Sentry should NOT have priv_validator_key

**Fix:** This is NORMAL - sentries don't validate. Ignore this error on sentries.

### Sentry Can't Connect to Validators

**Symptom:** `failed to connect to seed` errors

**Diagnosis:**
```bash
# Check if validators are running
docker ps --filter "label=paw.role=validator"

# Test connectivity from sentry
docker exec paw-sentry1 ping paw-node1
```

**Fixes:**
1. Ensure validators started before sentries
2. Verify Docker network: `docker network inspect pawnet`
3. Check firewall rules (if applicable)

## Security Considerations

### Validator Protection

✅ **DO:**
- Keep validators in private network segment
- Only expose sentries to public internet
- Use `private_peer_ids` to hide validators
- Monitor validator peer connections
- Use VPN/firewall for validator access

❌ **DON'T:**
- Expose validator RPC ports to internet
- Run validators without sentries in production
- Allow direct validator connections from unknown peers
- Skip sentry monitoring

### Sentry Hardening

✅ **DO:**
- Run sentries on separate infrastructure from validators
- Use rate limiting on sentry APIs
- Monitor for DDoS attacks
- Keep sentries updated
- Log all RPC access

❌ **DON'T:**
- Run sentries on same hardware as validators
- Allow unlimited API requests
- Skip security updates
- Trust sentry nodes with sensitive keys

## File Locations

```
paw/
├── compose/
│   └── docker-compose.4nodes-with-sentries.yml    # 4 validators + 2 sentries
├── scripts/
│   └── devnet/
│       ├── setup-validators.sh                    # Genesis generation
│       ├── init_node.sh                           # Validator initialization
│       ├── init_sentry.sh                         # Sentry initialization
│       └── .state/
│           ├── genesis.json                       # Shared genesis
│           ├── node*.id                           # Validator node IDs
│           ├── sentry*.id                         # Sentry node IDs
│           └── *.priv_validator_key.json          # Validator keys only
└── docs/
    ├── SENTRY_ARCHITECTURE.md                     # This file
    ├── MULTI_VALIDATOR_TESTNET.md                 # Validator guide
    └── TESTNET_QUICK_REFERENCE.md                 # Quick commands
```

## Migration from 4-Node to Sentry Setup

### From Existing 4-Validator Network

**Current:** Running `docker-compose.4nodes.yml`

**Steps:**
1. Stop current network (preserves blockchain data):
   ```bash
   docker compose -f compose/docker-compose.4nodes.yml down
   ```

2. Start with sentries (uses same genesis):
   ```bash
   docker compose -f compose/docker-compose.4nodes-with-sentries.yml up -d
   ```

3. Wait for sync:
   ```bash
   sleep 60
   ```

4. Verify:
   ```bash
   # Validators continuing from previous height
   curl -s http://localhost:26657/status | jq '.result.sync_info.latest_block_height'

   # Sentries catching up
   curl -s http://localhost:30658/status | jq '.result.sync_info.catching_up'
   ```

**Note:** Blockchain state is preserved because volumes are reused.

### Fresh Start with Sentries

**Recommended for clean testing:**
```bash
docker compose -f compose/docker-compose.4nodes-with-sentries.yml down -v
rm -f scripts/devnet/.state/*.json scripts/devnet/.state/*.mnemonic
./scripts/devnet/setup-validators.sh 4
docker compose -f compose/docker-compose.4nodes-with-sentries.yml up -d
sleep 60
```

## Performance Expectations

### Sync Times

**Validators** (from genesis):
- Reach consensus: ~30 seconds
- First block: ~40 seconds
- Stable production: ~60 seconds

**Sentries** (from genesis with 4 validators already running):
- Start sync: ~10 seconds
- Catch up (if < 100 blocks behind): ~30 seconds
- Catch up (if > 1000 blocks behind): ~5 minutes

### Resource Usage

**Validators:**
- CPU: 5-10% per node
- RAM: 500-800 MB per node
- Disk: 1-2 GB (grows with chain length)
- Network: 1-5 Mbps

**Sentries:**
- CPU: 5-15% (higher for RPC load)
- RAM: 500-900 MB
- Disk: 1-2 GB (same as validators)
- Network: 5-20 Mbps (higher for public traffic)

## Quick Reference

### Start Sentry Testnet
```bash
docker compose -f compose/docker-compose.4nodes-with-sentries.yml down -v
rm -f scripts/devnet/.state/*.json scripts/devnet/.state/*.mnemonic
./scripts/devnet/setup-validators.sh 4
docker compose -f compose/docker-compose.4nodes-with-sentries.yml up -d
sleep 60
```

### Check Status
```bash
# Validators
for port in 26657 26667 26677 26687; do
  echo "Node on :$port"
  curl -s http://localhost:$port/status | jq '.result.sync_info.latest_block_height'
done

# Sentries
for port in 30658 30668; do
  echo "Sentry on :$port"
  curl -s http://localhost:$port/status | jq '.result.sync_info'
done
```

### Stop Network
```bash
# Keep data
docker compose -f compose/docker-compose.4nodes-with-sentries.yml down

# Remove all data
docker compose -f compose/docker-compose.4nodes-with-sentries.yml down -v
```

## Support

For issues with sentry setup:
1. Check logs: `docker logs paw-sentry1 2>&1 | tail -100`
2. Verify peer connections: `curl -s http://localhost:30658/net_info | jq '.result.n_peers'`
3. Review this guide's troubleshooting section
4. Check main testnet docs: [MULTI_VALIDATOR_TESTNET.md](MULTI_VALIDATOR_TESTNET.md)

---

## Live Testnet Sentry

The PAW testnet has a live sentry node deployed:

| Property | Value |
|----------|-------|
| Node ID | `ce6afbda0a4443139ad14d2b856cca586161f00d` |
| P2P | `139.99.149.160:12056` |
| Server | services-testnet |

**External nodes should connect to the sentry**, not directly to validators:
```bash
persistent_peers = "ce6afbda0a4443139ad14d2b856cca586161f00d@139.99.149.160:12056"
```

For production sentry configuration files, see:
- [`infra/sentry/config-sentry.toml`](../infra/sentry/config-sentry.toml)
- [`infra/sentry/app-sentry.toml`](../infra/sentry/app-sentry.toml)
- [`infra/sentry/SENTRY_ARCHITECTURE.md`](../infra/sentry/SENTRY_ARCHITECTURE.md)

---

**Remember:** Sentries protect validators by handling public traffic. Always use sentries for production deployments.
