# PAW Testnet Sentry Node Architecture

## Overview

Sentry nodes protect validators from DDoS attacks by acting as a public-facing shield. All public traffic (RPC, REST, gRPC, P2P) routes through sentry nodes, never directly to validators.

## Architecture

```
                         Internet
                            │
                      [Cloudflare]
                            │
                     [nginx on paw-testnet]
                            │
              ┌─────────────┴─────────────┐
              │    WireGuard VPN          │
              │      (10.10.0.x)          │
              │                           │
       [sentry1:12057]                    │
    services-testnet                      │
       (10.10.0.4)                        │
              │                           │
    ┌─────────┴─────────┐                 │
    │                   │                 │
[val3:11857]      [val4:11957]    [val1:11657]  [val2:11757]
services-testnet  services-testnet  paw-testnet   paw-testnet
```

## How Sentries Affect Each Blockchain Aspect

### 1. RPC/REST/gRPC Queries

**Before Sentry (Direct Access)**:
- Public queries hit validators directly
- Validators exposed to DDoS attacks
- High load on consensus-critical nodes

**After Sentry (Protected)**:
- All public queries route through sentry
- Validators hidden from public network
- Sentry absorbs query load and attacks

**Configuration**:
```nginx
# nginx routes to sentry as primary
upstream paw_testnet_rpc_backend {
    server 10.10.0.4:12057 weight=10;    # Sentry (primary)
    server 127.0.0.1:11657 backup;        # Validator (fallback only)
}
```

**Latency Impact**: +1-5ms per hop (negligible for most applications)

### 2. Metrics and Monitoring

**Collect From Both Sentries AND Validators**:

| Metric | Source | Reason |
|--------|--------|--------|
| `consensus_height` | Validator | Authoritative chain state |
| `consensus_rounds` | Validator | Consensus participation |
| `p2p_peers` | Both | Different network views |
| `mempool_size` | Sentry | Transactions arrive here first |
| `p2p_message_*_bytes` | Sentry | Public network traffic |

**Prometheus Configuration**:
```yaml
scrape_configs:
  - job_name: 'paw-validators'
    static_configs:
      - targets: ['10.10.0.2:11660', '10.10.0.2:11760']  # val1, val2
      - targets: ['10.10.0.4:11860', '10.10.0.4:11960']  # val3, val4
  - job_name: 'paw-sentries'
    static_configs:
      - targets: ['10.10.0.4:12060']  # sentry1
```

**Key Alerts**:
- Sentry `p2p_peers < 5`: Network isolation
- Validator `consensus_rounds > 1`: Consensus delay
- Sentry down: Automatic failover to backup validators

### 3. Consensus and Block Production

**Sentries DO NOT participate in consensus.** They are full nodes without validator keys.

**Transaction Flow**:
```
User -> Sentry (CheckTx) -> Validator Mempool -> Block Proposal -> Consensus -> Block
```

**Consensus Message Relay**:
1. Validator signs prevote/precommit
2. Validator sends to sentry via private peer connection
3. Sentry gossips to public network
4. Other validators receive via their sentries

**Latency Overhead**: 2-10ms added to consensus round (acceptable)

### 4. State Sync and Snapshots

**Recommendation**: Sentries CAN serve snapshots, but dedicated nodes are better for high traffic.

**Current Configuration** (sentry):
```toml
# app.toml
[state-sync]
snapshot-interval = 0      # Disabled on primary sentry
snapshot-keep-recent = 0
```

**For State Sync Serving**:
- Enable on a dedicated full node, not sentry
- Avoids resource contention with consensus relay

### 5. Peer Discovery (PEX)

**Validator** (`pex = false`):
- No peer exchange
- Connects ONLY to trusted sentries
- Cannot be isolated by malicious peers

**Sentry** (`pex = true`):
- Active peer discovery
- `private_peer_ids` hides validators
- `unconditional_peer_ids` ensures validator connection

**Critical Sentry Config**:
```toml
[p2p]
pex = true
private_peer_ids = "val1_id,val2_id,val3_id,val4_id"
unconditional_peer_ids = "val1_id,val2_id,val3_id,val4_id"
```

### 6. Transaction Broadcasting

**Flow Through Sentry**:
1. User submits tx to sentry RPC
2. Sentry performs `CheckTx` validation
3. Valid tx added to sentry mempool
4. Sentry gossips to validators
5. Validator adds to mempool for block inclusion

**Latency**: ~10-100ms (client to sentry) + ~1-10ms (sentry to validator)

### 7. Security

**Attacks Mitigated**:
- DDoS: Sentry absorbs traffic, validators unaffected
- Eclipse attacks: Validators only connect to trusted sentries
- IP discovery: Validators have no public IP

**New Considerations**:
- Compromised sentry: Can see transactions, but cannot sign
- All sentries down: Validators isolated (mitigate with redundancy)

## Port Configuration

| Service | Sentry Port | Validator Ports |
|---------|-------------|-----------------|
| P2P | 12056 | 11656, 11756, 11856, 11956 |
| RPC | 12057 | 11657, 11757, 11857, 11957 |
| gRPC | 12090 | 11090, 11190, 11290, 11390 |
| REST | 12017 | 11317, 11417, 11517, 11617 |
| Prometheus | 12060 | 11660, 11760, 11860, 11960 |

## Nginx Routing

All public endpoints route through sentry:

| Endpoint | Route |
|----------|-------|
| testnet-rpc.poaiw.org | -> 10.10.0.4:12057 (sentry RPC) |
| testnet-api.poaiw.org | -> 10.10.0.4:12017 (sentry REST) |
| testnet-grpc.poaiw.org | -> 10.10.0.4:12090 (sentry gRPC) |

Validators are backup-only (used if sentry fails).

## Monitoring Commands

```bash
# Check sentry health
./scripts/testnet/health-sentry.sh

# Check validator health
./scripts/testnet/health-validators.sh

# Check sentry peer connections
ssh services-testnet 'curl -s http://127.0.0.1:12057/net_info | jq ".result.peers[].node_info.moniker"'

# Check public endpoint routing
curl -s https://testnet-rpc.poaiw.org/status | jq '.result.node_info.moniker'
# Should show "paw-sentry-1"
```

## Failover Behavior

1. **Sentry down**: nginx fails over to backup validators
2. **Validator down**: Consensus continues with 3/4 validators
3. **VPN down**: Sentry cannot reach val1/val2, partial connectivity

## Best Practices

1. **Multiple sentries**: Add sentry2 on paw-testnet for redundancy
2. **Geographic distribution**: Sentries in different regions
3. **Monitoring**: Alert on sentry disconnection from validators
4. **Rate limiting**: Protect sentries from abuse
5. **Separate snapshot nodes**: Don't overload sentries with state sync
