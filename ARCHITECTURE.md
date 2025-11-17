# PAW Blockchain Architecture

## Overview

PAW is a Layer-1 blockchain built on the Cosmos SDK framework with Tendermint BFT consensus. This document provides a comprehensive overview of the system architecture, module interactions, and key design decisions.

## System Architecture

### High-Level Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                        Client Layer                              │
│  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌──────────┐        │
│  │  Mobile  │  │ Desktop  │  │    Web   │  │   CLI    │        │
│  │  Wallet  │  │  Wallet  │  │  Wallet  │  │  pawd    │        │
│  └────┬─────┘  └────┬─────┘  └────┬─────┘  └────┬─────┘        │
└───────┼─────────────┼─────────────┼─────────────┼───────────────┘
        │             │             │             │
        └─────────────┴─────────────┴─────────────┘
                          │
        ┌─────────────────▼─────────────────────────────────────┐
        │            API Gateway Layer                          │
        │  ┌──────────────┐  ┌──────────────┐                  │
        │  │   REST API   │  │   gRPC API   │                  │
        │  │  (port 1317) │  │  (port 9090) │                  │
        │  └──────┬───────┘  └──────┬───────┘                  │
        └─────────┼──────────────────┼──────────────────────────┘
                  │                  │
        ┌─────────▼──────────────────▼──────────────────────────┐
        │              Application Layer                         │
        │  ┌──────────────────────────────────────────────────┐ │
        │  │         PAW Application (app.go)                 │ │
        │  │  - Module Manager                                │ │
        │  │  - Message Routing                               │ │
        │  │  - Transaction Processing                        │ │
        │  │  - State Management                              │ │
        │  └──────────────────────────────────────────────────┘ │
        └────────────────────────────────────────────────────────┘
                            │
        ┌───────────────────▼────────────────────────────────────┐
        │              Module Layer                              │
        │  ┌──────────┐  ┌──────────┐  ┌──────────┐            │
        │  │   Bank   │  │   DEX    │  │  Compute │            │
        │  │  Module  │  │  Module  │  │  Module  │            │
        │  └──────────┘  └──────────┘  └──────────┘            │
        │  ┌──────────┐  ┌──────────┐  ┌──────────┐            │
        │  │  Oracle  │  │ Staking  │  │   Gov    │            │
        │  │  Module  │  │  Module  │  │  Module  │            │
        │  └──────────┘  └──────────┘  └──────────┘            │
        └────────────────────────────────────────────────────────┘
                            │
        ┌───────────────────▼────────────────────────────────────┐
        │             Consensus Layer                            │
        │  ┌──────────────────────────────────────────────────┐ │
        │  │     Tendermint BFT Consensus Engine              │ │
        │  │  - Block Proposal & Validation                   │ │
        │  │  - BFT-DPoS with 4-second finality               │ │
        │  │  - Byzantine Fault Tolerance (33% adversary)     │ │
        │  └──────────────────────────────────────────────────┘ │
        └────────────────────────────────────────────────────────┘
                            │
        ┌───────────────────▼────────────────────────────────────┐
        │              Storage Layer                             │
        │  ┌──────────────────────────────────────────────────┐ │
        │  │         IAVL+ Merkle Tree Store                  │ │
        │  │  - Account State                                 │ │
        │  │  - Module State (DEX, Oracle, Compute)           │ │
        │  │  - Cryptographic Proofs                          │ │
        │  └──────────────────────────────────────────────────┘ │
        │  ┌──────────────────────────────────────────────────┐ │
        │  │          LevelDB/RocksDB Backend                 │ │
        │  └──────────────────────────────────────────────────┘ │
        └────────────────────────────────────────────────────────┘
                            │
        ┌───────────────────▼────────────────────────────────────┐
        │              Network Layer                             │
        │  ┌──────────────────────────────────────────────────┐ │
        │  │         P2P Communication (CometBFT)             │ │
        │  │  - Peer Discovery                                │ │
        │  │  - Block Broadcasting                            │ │
        │  │  - State Synchronization                         │ │
        │  │  - Mempool Transaction Gossip                    │ │
        │  └──────────────────────────────────────────────────┘ │
        └────────────────────────────────────────────────────────┘
```

## Core Modules

### 1. Bank Module

**Purpose**: Handles token transfers and account balances

**Key Functions**:
- Transfer tokens between accounts
- Query account balances
- Multi-send transactions
- Supply tracking and denomination metadata

**State Storage**:
- Account balances (key: `0x02 | address | denom`)
- Total supply (key: `0x00 | denom`)
- Denomination metadata

### 2. DEX Module

**Purpose**: Native decentralized exchange with liquidity pools

**Key Features**:
- Automated Market Maker (AMM) pools
- Atomic swaps with slippage protection
- Liquidity provision and LP tokens
- Circuit breaker protection
- MEV protection mechanisms
- Flash loan prevention
- TWAP (Time-Weighted Average Price) oracles

**State Storage**:
- Liquidity pools (key: `0x01 | pool_id`)
- LP token balances (key: `0x02 | address | pool_id`)
- Circuit breaker state (key: `0x03 | pool_id`)
- TWAP data (key: `0x04 | pool_id | timestamp`)

**Architecture**:
```
┌────────────────────────────────────────────────┐
│              DEX Module                        │
│  ┌──────────────────────────────────────────┐ │
│  │         Message Handler                  │ │
│  │  - CreatePool                            │ │
│  │  - AddLiquidity                          │ │
│  │  - RemoveLiquidity                       │ │
│  │  - Swap                                  │ │
│  └────────────┬─────────────────────────────┘ │
│               │                                │
│  ┌────────────▼─────────────────────────────┐ │
│  │         Keeper Layer                     │ │
│  │  ┌────────────┐  ┌────────────┐          │ │
│  │  │   Pool     │  │  Circuit   │          │ │
│  │  │  Manager   │  │  Breaker   │          │ │
│  │  └────────────┘  └────────────┘          │ │
│  │  ┌────────────┐  ┌────────────┐          │ │
│  │  │    MEV     │  │    TWAP    │          │ │
│  │  │ Protection │  │   Oracle   │          │ │
│  │  └────────────┘  └────────────┘          │ │
│  └──────────────────────────────────────────┘ │
└────────────────────────────────────────────────┘
```

### 3. Oracle Module

**Purpose**: Decentralized price feeds for DeFi operations

**Key Features**:
- Validator-based price submissions
- Median aggregation (outlier resistant)
- Automatic slashing for inaccurate submissions
- Rate limiting and freshness checks
- Confidence intervals and price deviation tracking

**State Storage**:
- Price feeds (key: `0x01 | asset_id`)
- Validator submissions (key: `0x02 | asset_id | validator`)
- Submission timestamps (key: `0x03 | validator | asset_id`)

**Price Aggregation Flow**:
```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│ Validator 1 │     │ Validator 2 │ ... │ Validator N │
│  Price: 100 │     │  Price: 102 │     │  Price: 99  │
└──────┬──────┘     └──────┬──────┘     └──────┬──────┘
       │                   │                    │
       └───────────────────┼────────────────────┘
                           │
                  ┌────────▼────────┐
                  │  Oracle Keeper  │
                  │                 │
                  │  1. Collect     │
                  │  2. Filter      │
                  │     Outliers    │
                  │  3. Calculate   │
                  │     Median      │
                  │  4. Store       │
                  └────────┬────────┘
                           │
                  ┌────────▼────────┐
                  │  Aggregated     │
                  │  Price: 100     │
                  │  Confidence: 95%│
                  └─────────────────┘
```

### 4. Compute Module

**Purpose**: Secure API key aggregation and compute task routing

**Key Features**:
- Provider registration with staking
- Compute request submission
- Result verification and settlement
- TEE (Trusted Execution Environment) integration
- Task status tracking

**State Storage**:
- Compute tasks (key: `0x03 | task_id`)
- Providers (key: `0x04 | provider_address`)
- Compute requests (key: `0x05 | request_id`)

**Compute Request Flow**:
```
┌─────────────┐
│   Requester │
│  (submits)  │
└──────┬──────┘
       │
       │ 1. RequestCompute(api_url, max_fee)
       │
┌──────▼──────────────────────────────────────┐
│         Compute Module Keeper               │
│  - Validate request                         │
│  - Generate request_id                      │
│  - Store in pending state                   │
└──────┬──────────────────────────────────────┘
       │
       │ 2. Task Assignment
       │
┌──────▼──────────┐
│  Compute        │
│  Provider       │
│  (TEE-enabled)  │
│  - Execute task │
│  - Submit result│
└──────┬──────────┘
       │
       │ 3. SubmitResult(request_id, result)
       │
┌──────▼──────────────────────────────────────┐
│         Compute Module Keeper               │
│  - Verify provider                          │
│  - Validate result                          │
│  - Mark as completed                        │
│  - Distribute rewards                       │
└─────────────────────────────────────────────┘
```

### 5. Staking Module

**Purpose**: Validator set management and delegation

**Key Functions**:
- Validator registration
- Token delegation/undelegation
- Reward distribution
- Slashing for misbehavior

### 6. Governance Module

**Purpose**: On-chain governance and parameter updates

**Key Functions**:
- Proposal submission
- Voting (yes/no/abstain/veto)
- Parameter change proposals
- Software upgrade coordination

## Data Flow

### Transaction Lifecycle

```
1. Transaction Submission
   ┌─────────┐
   │  Client │
   └────┬────┘
        │
        ▼
   ┌─────────┐
   │ Mempool │ (Validation & Ordering)
   └────┬────┘
        │
        ▼
2. Block Proposal
   ┌──────────┐
   │ Proposer │ (Select transactions)
   └────┬─────┘
        │
        ▼
3. Consensus
   ┌───────────┐
   │ Validators│ (Vote on block)
   └────┬──────┘
        │
        ▼
4. Execution
   ┌─────────────┐
   │ Application │ (Process messages)
   └────┬────────┘
        │
        ▼
5. State Update
   ┌──────────┐
   │   Store  │ (Commit to IAVL tree)
   └──────────┘
```

### DEX Swap Flow

```
1. User submits swap transaction
   MsgSwap{
     trader: "paw1xxx...",
     token_in: "1000uapaw",
     token_out_min: "900upaw",
     pool_id: "pool-1"
   }
   ↓
2. DEX Keeper validates
   - Check pool exists
   - Verify circuit breaker not triggered
   - Check MEV protection
   ↓
3. Calculate swap amounts
   - Query pool reserves
   - Apply constant product formula (x * y = k)
   - Calculate output with fees
   ↓
4. Execute swap
   - Transfer token_in from trader to pool
   - Transfer token_out from pool to trader
   - Update pool reserves
   - Update TWAP oracle
   ↓
5. Emit events
   EventSwap{
     pool_id: "pool-1",
     trader: "paw1xxx...",
     token_in: "1000uapaw",
     token_out: "950upaw",
     fee: "3upaw"
   }
```

## Key Design Decisions

### 1. Cosmos SDK Framework

**Rationale**:
- Battle-tested framework with extensive tooling
- Modular architecture enables rapid development
- IBC compatibility for future interoperability
- Strong community and ecosystem support

**Tradeoffs**:
- (+) Mature codebase with security audits
- (+) Built-in modules reduce development time
- (-) Learning curve for Cosmos-specific patterns
- (-) Limited flexibility in consensus modifications

### 2. Tendermint BFT-DPoS Consensus

**Rationale**:
- Immediate finality (no reorgs)
- Proven Byzantine fault tolerance
- 4-second block times for fast UX
- Energy efficient (no mining)

**Parameters**:
- Block time: 4 seconds
- Byzantine tolerance: 33% adversarial validators
- Validator set: 4-100 nodes (governable)

### 3. Native DEX Implementation

**Rationale**:
- Eliminates smart contract vulnerabilities
- Lower gas costs for trading
- Built-in MEV protection
- Circuit breaker for crisis management

**Security Features**:
- Price impact limits (circuit breaker)
- Flashloan prevention via same-block checks
- Minimum liquidity requirements
- Transaction ordering protection

### 4. Oracle Design - Median Aggregation

**Rationale**:
- Outlier resistant (vs mean/average)
- No single point of failure
- Cryptographic verification of submissions
- Automatic slashing incentivizes accuracy

**Alternative Considered**:
- Chainlink-style external oracle: Rejected due to dependency on external infrastructure
- Band Protocol integration: Rejected to maintain sovereignty

### 5. TEE-Based Compute Plane

**Rationale**:
- Hardware-level security for API keys
- Minute-level accounting for cost efficiency
- Time-limited proxy tokens prevent key extraction
- Verifiable computation results

**Security Model**:
- API keys never touch blockchain state
- Encrypted communication channels
- Automatic key destruction post-execution
- Proof of TEE attestation required

## State Management

### IAVL Tree Structure

```
State Root Hash
│
├─ Bank Module State
│  ├─ Balances: 0x02 | address | denom → amount
│  └─ Supply: 0x00 | denom → total_supply
│
├─ DEX Module State
│  ├─ Pools: 0x01 | pool_id → Pool{reserves, lp_supply}
│  ├─ LP Tokens: 0x02 | address | pool_id → lp_amount
│  ├─ Circuit Breakers: 0x03 | pool_id → BreakerState
│  └─ TWAP: 0x04 | pool_id | timestamp → price
│
├─ Oracle Module State
│  ├─ Price Feeds: 0x01 | asset_id → PriceFeed
│  ├─ Submissions: 0x02 | asset_id | validator → Price
│  └─ Timestamps: 0x03 | validator | asset_id → time
│
├─ Compute Module State
│  ├─ Tasks: 0x03 | task_id → ComputeTask
│  ├─ Providers: 0x04 | provider → Provider
│  └─ Requests: 0x05 | request_id → ComputeRequest
│
└─ Staking Module State
   ├─ Validators: 0x21 | operator_addr → Validator
   └─ Delegations: 0x31 | delegator | validator → Delegation
```

## Network Topology

```
┌───────────────────────────────────────────────────────┐
│                  Validator Network                    │
│  ┌──────────┐    ┌──────────┐    ┌──────────┐        │
│  │Validator │◄───►│Validator │◄───►│Validator │        │
│  │    1     │    │    2     │    │    3     │        │
│  └─────┬────┘    └─────┬────┘    └─────┬────┘        │
│        │               │               │              │
│        └───────────────┼───────────────┘              │
└────────────────────────┼──────────────────────────────┘
                         │
                         │ Sentry Node Architecture
                         │
        ┌────────────────┼────────────────┐
        │                │                │
┌───────▼─────┐  ┌───────▼─────┐  ┌──────▼──────┐
│   Sentry    │  │   Sentry    │  │   Sentry    │
│   Node 1    │  │   Node 2    │  │   Node 3    │
└───────┬─────┘  └───────┬─────┘  └──────┬──────┘
        │                │                │
        │         ┌──────┴────────┐       │
        │         │               │       │
    ┌───▼────┐ ┌──▼───┐    ┌────▼──┐ ┌──▼────┐
    │ Public │ │Public│    │Public │ │Public │
    │  Node  │ │ Node │    │ Node  │ │ Node  │
    └────────┘ └──────┘    └───────┘ └───────┘
```

## Security Architecture

### Multi-Layer Security

1. **Network Layer**
   - DDoS protection via rate limiting
   - Sentry node architecture protects validators
   - P2P encryption (LibP2P)

2. **Consensus Layer**
   - Byzantine fault tolerance (33% adversary)
   - Validator slashing for double-signing
   - Tendermint BFT guarantees

3. **Application Layer**
   - Input validation on all messages
   - Authorization checks (ante handlers)
   - Cryptographic signature verification

4. **Module Layer**
   - Circuit breakers in DEX
   - Oracle slashing for bad data
   - TEE attestation for compute

5. **State Layer**
   - Merkle proofs for state verification
   - Immutable state transitions
   - Cryptographic hashing (SHA256)

## Performance Considerations

### Optimization Strategies

1. **Transaction Throughput**
   - Target: 1000+ tx/s
   - Mempool optimization
   - Efficient state access patterns
   - Batching where possible

2. **State Growth Management**
   - State pruning for non-archive nodes
   - Efficient key-value encoding
   - IAVL tree optimization

3. **Query Performance**
   - Indexed queries for common patterns
   - Caching for hot data
   - gRPC streaming for large datasets

## Upgrade Path

### Planned Enhancements

**Phase 2: Optimization**
- IAVL v2 migration
- State sync improvements
- Query optimization

**Phase 3: Intelligence Layer**
- zkML proof integration
- Enhanced privacy (zk-SNARKs)
- Sharding support

**Phase 4: Interoperability**
- IBC enablement
- Cross-chain DEX
- Bridge integrations

## Monitoring & Observability

### Key Metrics

1. **Chain Metrics**
   - Block time (target: 4s)
   - Block size
   - Transaction throughput
   - Validator uptime

2. **DEX Metrics**
   - Pool liquidity
   - Trade volume
   - Circuit breaker triggers
   - Slippage rates

3. **Oracle Metrics**
   - Price deviation
   - Submission rates
   - Slash events
   - Confidence intervals

4. **Compute Metrics**
   - Active providers
   - Task completion rate
   - Average execution time
   - Resource utilization

### Telemetry

```
Application
  │
  ├─ Prometheus Exporter (port 26660)
  │  └─ Metrics: block_time, tx_count, pool_tvl, etc.
  │
  └─ Structured Logging
     └─ Levels: info, warn, error, debug
```

## References

- [Cosmos SDK Documentation](https://docs.cosmos.network)
- [Tendermint Core](https://docs.tendermint.com)
- [IAVL Tree Specification](https://github.com/cosmos/iavl)
- [PAW Whitepaper](PAW%20Extensive%20whitepaper%20.md)

---

**Document Version**: 1.0
**Last Updated**: November 2025
**Maintainer**: PAW Development Team
