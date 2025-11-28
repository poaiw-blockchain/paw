# Architecture

Technical architecture of PAW Blockchain.

## System Overview

```
┌─────────────────────────────────────────────────┐
│              PAW Blockchain                      │
├─────────────────────────────────────────────────┤
│  Application Layer (Cosmos SDK Modules)         │
│  ├─ Bank    ├─ Staking  ├─ Governance          │
│  ├─ DEX     ├─ Oracle   ├─ Compute             │
├─────────────────────────────────────────────────┤
│  Consensus Layer (Tendermint BFT-DPoS)          │
│  ├─ Block Proposal  ├─ Voting  ├─ Commit       │
├─────────────────────────────────────────────────┤
│  Networking Layer (P2P)                          │
│  ├─ Peer Discovery  ├─ Gossip  ├─ Sync         │
├─────────────────────────────────────────────────┤
│  Storage Layer (IAVL Trees + LevelDB)           │
│  ├─ State   ├─ Blocks   ├─ Transactions        │
└─────────────────────────────────────────────────┘
```

## Core Components

### Consensus Engine

- **Type**: Tendermint BFT-DPoS
- **Block Time**: 4 seconds
- **Finality**: Immediate (1 block)
- **Validator Set**: 4-100 validators
- **Byzantine Tolerance**: Up to 1/3 malicious nodes

### State Machine

Built on Cosmos SDK with custom modules:

**Bank Module**
- Token transfers
- Multi-send operations
- Supply tracking

**Staking Module**
- Validator management
- Delegation/undelegation
- Reward distribution
- Slashing

**DEX Module**
- AMM liquidity pools
- Atomic swaps
- Fee distribution
- Circuit breakers

**Oracle Module**
- Price feeds
- Median aggregation
- Update frequency: 1 minute

**Governance Module**
- Proposal submission
- Voting mechanism
- Parameter changes
- Treasury management

## Data Flow

```
User Transaction → Wallet → RPC Node → Mempool
→ Block Proposal → Consensus → State Update
→ Block Commit → State Root → Next Block
```

## Network Topology

### Public Network

```
Users
  ↓
Load Balancer
  ↓
RPC Nodes (Public)
  ↓
Sentry Nodes
  ↓
Validator Nodes (Private)
```

### State Synchronization

1. **Full Sync**: Download and verify all blocks
2. **State Sync**: Snapshot at specific height
3. **Fast Sync**: Download headers, verify blocks

## Security Model

### Cryptography

- **Signing**: secp256k1 (ECDSA)
- **Hashing**: SHA-256
- **Address Format**: Bech32 (paw prefix)
- **Keys**: BIP39 mnemonics, BIP44 derivation

### Slashing

| Violation | Penalty |
|-----------|---------|
| Double Signing | 5% slash + jail |
| Downtime | 0.01% slash + jail |

### Circuit Breakers

- Single trade >10% of pool triggers halt
- Price movement >20% in one block triggers review
- Governance can emergency pause

## Performance

### Throughput

- **Theoretical Max**: 10,000 tx/s
- **Current Capacity**: 1,000+ tx/s
- **Average Load**: 50-100 tx/s

### Latency

- **Block Time**: 4 seconds
- **Finality**: 4 seconds (1 block)
- **DEX Swap**: Sub-second execution

## Storage

### State Structure

```
AppState
├─ Accounts (balances, sequences)
├─ Validators (status, power, commission)
├─ Delegations (delegator → validator)
├─ DEX Pools (reserves, liquidity)
└─ Governance (proposals, votes)
```

### Database

- **Type**: LevelDB (default) or RocksDB
- **State Tree**: IAVL (Immutable AVL Tree)
- **Pruning**: Configurable retention

## Upgrade Mechanism

### Coordinated Upgrades

1. Governance proposal submitted
2. Community votes
3. If passed, upgrade height set
4. Validators upgrade binary
5. Chain automatically upgrades at height

### Emergency Upgrades

Guardian DAO can fast-track critical fixes.

## Interoperability

### IBC Support

- **Protocol**: IBC v3
- **Channels**: Unordered/Ordered
- **Packet Timeouts**: Configurable
- **Relayers**: Hermes, Go Relayer

### Bridges

- Ethereum bridge (planned)
- Bitcoin bridge (planned)
- Cross-chain DEX aggregation

## Monitoring & Observability

### Metrics

- Prometheus endpoints
- Custom module metrics
- Validator performance
- Network health

### Logging

- Structured JSON logs
- Configurable levels
- Module-specific loggers

---

**Next:** [Tokenomics](/reference/tokenomics) →
