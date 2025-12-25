# PAW Blockchain Architecture

## System Overview

PAW is a Cosmos SDK-based blockchain optimized for decentralized finance (DeFi) and verifiable computation. Built on Tendermint BFT consensus, PAW integrates three specialized modules that work synergistically to enable trustless trading, off-chain computation, and manipulation-resistant price feeds.

**Core Design Principles:**
- **Security First**: Multi-layer security with circuit breakers, slashing, and cryptographic verification
- **Interoperability**: Native IBC support for cross-chain operations
- **Verifiability**: Zero-knowledge proofs for computation integrity
- **Decentralization**: Validator-based consensus for critical operations

**Technology Stack:**
- Cosmos SDK v0.50+
- Tendermint BFT Consensus
- IBC v8 (Inter-Blockchain Communication)
- Go 1.24+
- Groth16 ZK-SNARKs (gnark framework)

## Module Architecture

### Three-Module Design

```
┌─────────────────────────────────────────────────────────┐
│                    PAW Blockchain                        │
├─────────────────┬─────────────────┬─────────────────────┤
│   x/compute     │     x/dex       │     x/oracle        │
│                 │                 │                     │
│ Off-chain       │ Automated       │ Decentralized       │
│ Computation     │ Market Maker    │ Price Feeds         │
│ Marketplace     │ + Limit Orders  │ (TWAP)              │
│                 │                 │                     │
│ • ZK Proofs     │ • Liquidity     │ • Validator-based   │
│ • Reputation    │   Pools         │ • Geographic        │
│ • Escrow        │ • Swap Routing  │   Diversity         │
│ • IBC Compute   │ • IBC Trading   │ • IBC Price Sync    │
└─────────────────┴─────────────────┴─────────────────────┘
         │                 │                   │
         └─────────────────┼───────────────────┘
                           │
                ┌──────────▼───────────┐
                │   Cosmos SDK Core    │
                ├──────────────────────┤
                │ • Bank   • Staking   │
                │ • Gov    • Slashing  │
                │ • Auth   • IBC       │
                └──────────────────────┘
```

### x/compute: Decentralized Computation Marketplace

**Purpose:** Trustless off-chain computation with on-chain verification.

**Key Components:**
1. **Request Lifecycle Manager**
   - Job submission and escrow
   - Provider matching (by specs, stake, reputation)
   - Timeout and cancellation handling

2. **Multi-Tier Verification System**
   - **Merkle Proofs**: Fast integrity verification
   - **Ed25519 Signatures**: Provider authenticity
   - **ZK-SNARKs (Groth16)**: Computation correctness without revealing private data

3. **Reputation & Slashing**
   - Reputation scoring (0-100 scale)
   - Economic penalties for misbehavior
   - Minimum thresholds for participation

4. **ZK Circuit Architecture**
   ```
   Public Inputs:   RequestID, ResultHash, ProviderAddressHash
   Private Inputs:  ComputationDataHash, Timestamp, ExitCode,
                    CpuCycles, MemoryUsed
   Constraint:      MiMC(inputs) == ResultHash
   ```

5. **Circuit Breaker**
   - Module-level emergency halt
   - Per-provider suspension
   - Governance-controlled recovery

**State Keys:**
- `requests/{requestId}` → Request metadata
- `providers/{address}` → Provider registration + performance
- `results/{requestId}` → Verification data + proofs
- `disputes/{disputeId}` → Governance resolution
- `nonces/{height}` → Replay protection (auto-cleanup)

### x/dex: Automated Market Maker + Order Book

**Purpose:** Decentralized token exchange with hybrid AMM/order book model.

**Key Components:**
1. **Constant Product AMM (x * y = k)**
   - Liquidity pools with configurable fees
   - Three-tier fee distribution (LP, Protocol, Burn)
   - LP token minting/burning

2. **Limit Order Book**
   - Price-time priority matching
   - Partial fills supported
   - GTC and IOC order types

3. **Security Mechanisms**
   - **Flash Loan Protection**: Intra-block state tracking (10-block window)
   - **Slippage Protection**: Minimum output guarantees
   - **Pool Drain Protection**: Max 30% single-transaction limit
   - **Oracle Integration**: Price validation against TWAP

4. **TWAP Oracle**
   - Per-pool price accumulation
   - Configurable lookback window (default: 1000 blocks)
   - Manipulation resistance

**State Keys:**
- `pools/{poolId}` → Pool reserves + shares
- `limit_orders/{orderId}` → Active orders
- `twap/{poolId}/{height}` → Price snapshots
- `lp_shares/{poolId}/{address}` → User positions
- `nonces/{address}` → Replay protection

**Oracle Integration:**
- DEX → Oracle: Uses TWAP for fair pricing
- Oracle → DEX: Validates swap prices, detects arbitrage
- Interface: `OracleKeeper.GetPrice(denom)` and `GetPriceWithTimestamp(denom)`

### x/oracle: Decentralized Price Feeds

**Purpose:** Manipulation-resistant price data for DeFi protocols.

**Key Components:**
1. **Validator-Based Consensus**
   - Stake-weighted price aggregation
   - Median filtering to exclude outliers
   - Vote period batching (default: 30 blocks)

2. **Geographic Distribution**
   - GeoIP verification of validator regions
   - Minimum regional diversity requirement
   - Prevents localized market manipulation

3. **TWAP Accumulator**
   ```
   TWAP = Σ(price_i * duration_i) / total_duration
   ```
   - Resistant to flash attacks
   - Configurable lookback (default: 1000 blocks)
   - Used by DEX for price validation

4. **Slashing Protection**
   - Miss rate tracking (10,000 block window)
   - 1% stake slashing for high miss rates
   - Automatic jailing for severe violations

**State Keys:**
- `prices/{asset}` → Canonical price + metadata
- `validator_prices/{asset}/{validator}` → Individual submissions
- `validator_oracles/{validator}` → Performance tracking
- `price_snapshots/{asset}/{height}` → TWAP data

**Supported Assets:**
- Configurable via governance
- Default: BTC/USD, ETH/USD, ATOM/USD, OSMO/USD
- Extensible to equities, commodities, forex

## Inter-Module Communication

### Data Flow Patterns

#### 1. DEX ↔ Oracle Integration
```
DEX Swap Request
     ↓
Query Oracle Price → Validate against pool price
     ↓
If deviation > 5% → Reject (manipulation protection)
     ↓
Execute Swap → Update pool TWAP
     ↓
Oracle reads DEX TWAP → Aggregate with validator prices
```

**Use Cases:**
- Arbitrage detection
- Pool valuation in USD
- LP token pricing
- Price impact validation

#### 2. Compute ↔ Oracle (Potential)
```
Compute Job Pricing
     ↓
Query Oracle for resource costs
     ↓
Calculate fair payment in native token
     ↓
Adjust provider pricing dynamically
```

#### 3. Cross-Module State Access
- **DEX reads Oracle**: Via `OracleKeeper` interface (GetPrice, GetPriceWithTimestamp)
- **Oracle reads DEX**: Potential future integration for TWAP aggregation
- **Compute isolated**: Currently no direct integration (future: dynamic pricing)

### IBC Integration

All three modules support IBC for cross-chain operations:

**IBC Capabilities:**
```
┌──────────────────────────────────────────────────────┐
│                 IBC Channel Types                     │
├──────────────────┬──────────────────┬────────────────┤
│   compute-1      │     dex-1        │   oracle-1     │
│                  │                  │                │
│ • Job Packets    │ • Swap Packets   │ • Price Feeds  │
│ • Result Packets │ • Liquidity Ops  │ • Consensus    │
│ • Timeout Refund │ • Timeout Refund │ • Sync         │
└──────────────────┴──────────────────┴────────────────┘
```

**Authorization:**
- Whitelist-based channel authorization (`params.AuthorizedChannels`)
- Per-module configuration via governance
- Timeout handling with automatic refunds

**Implementation:**
- Each module implements `IBCModule` interface
- Packet acknowledgment with error handling
- Channel capability management via `ScopedKeeper`

## Security Architecture

### Multi-Layer Defense

#### Layer 1: Cryptographic Verification
1. **Compute Module**
   - Merkle tree verification (output integrity)
   - Ed25519 signatures (provider authenticity)
   - Groth16 ZK-SNARKs (computation correctness)

2. **DEX Module**
   - Nonce-based replay protection
   - Signature verification on all transactions

3. **Oracle Module**
   - Validator signatures on price submissions
   - Stake-weighted consensus

#### Layer 2: Economic Security
1. **Compute Slashing**
   - Reputation: -10% on failure
   - Stake: -1% on fraud
   - Minimum stake: 1 PAW (1,000,000 upaw)

2. **Oracle Slashing**
   - Miss rate > threshold: 1% stake slash
   - Temporary jailing for repeated violations

3. **DEX Fees**
   - Swap fee: 0.3% (0.25% LP + 0.05% protocol)
   - Disincentivizes manipulation via cost

#### Layer 3: Circuit Breakers

**Module-Level:**
```go
// Emergency halt mechanism
OpenCircuitBreaker(actor, reason) → Pause all operations
CloseCircuitBreaker(actor, reason) → Resume (governance-only)
```

**Per-Provider (Compute):**
- Individual provider suspension
- Prevents cascading failures

**Pool-Level (DEX):**
- Max drain protection (30%)
- Flash loan protection (10-block window)

#### Layer 4: Rate Limiting

**Compute Module:**
- Per-address: 10 requests/block
- Global: 100 requests/block
- Dynamic adjustment based on load

**Oracle Module:**
- Validators: 1 submission per vote period
- Duplicate detection per asset

**DEX Module:**
- Implicit: Gas costs and slippage protection

### Commit-Reveal Scheme

**DEX Secure Swaps (ADR-003):**
```
Phase 1: Commit
  User submits hash(swap_details + secret)
  Escrow tokens

Phase 2: Wait
  Minimum wait period (e.g., 2 blocks)

Phase 3: Reveal
  User reveals swap_details + secret
  Verification: hash(revealed) == committed
  Execute swap if valid
```

**Benefits:**
- MEV resistance (sandwich attack prevention)
- Front-running protection
- Censorship resistance

## State Management

### Storage Architecture

**IAVL Tree Structure:**
```
PAW State Root
├── x/compute
│   ├── requests/{id}        [Indexed by ID]
│   ├── providers/{addr}     [Indexed by address]
│   ├── results/{id}
│   ├── disputes/{id}
│   └── nonces/{height}      [Indexed by height for cleanup]
├── x/dex
│   ├── pools/{id}           [Indexed by ID]
│   ├── limit_orders/{id}
│   ├── twap/{pool}/{height}
│   └── lp_shares/{pool}/{addr}
└── x/oracle
    ├── prices/{asset}       [Indexed by denom]
    ├── validator_prices/{asset}/{val}
    ├── validator_oracles/{val}
    └── price_snapshots/{asset}/{height}
```

### State Pruning & Cleanup

**Compute Module:**
- **Nonce Cleanup**: Automatic cleanup of nonces older than retention period
  - Default retention: 17,280 blocks (~24h at 5s blocks)
  - Batched cleanup: 100 blocks per EndBlocker
  - Prevents unbounded state growth

**DEX Module:**
- **TWAP Pruning**: Old snapshots beyond lookback window
- **Expired Orders**: Automatic cleanup of expired limit orders

**Oracle Module:**
- **Price History**: Configurable retention (default: 100,000 blocks)
- **Validator Metrics**: Rolling window (10,000 blocks)

### Indexing Strategy

**Primary Indices:**
- Compute: RequestID, ProviderAddress
- DEX: PoolID, OrderID, UserAddress
- Oracle: Asset, ValidatorAddress

**Secondary Indices:**
- Compute: Status, CreatedAt (for filtering)
- DEX: TokenPair, Price (for order matching)
- Oracle: Timestamp, Region (for geographic diversity)

## Performance & Scalability

### Throughput Characteristics

**Transaction Types:**
- Compute: 50-100 job submissions/sec (limited by verification overhead)
- DEX: 200-500 swaps/sec (limited by state updates)
- Oracle: 10-30 price updates/sec (limited by consensus)

**Bottlenecks:**
1. **Compute**: ZK proof verification (CPU-intensive)
2. **DEX**: IAVL state commits (I/O-intensive)
3. **Oracle**: Validator coordination (network-intensive)

### Optimization Strategies

**Lazy Circuit Compilation:**
- ZK circuits compiled on first use
- Cached for subsequent verifications
- Reduces startup time

**Batched Operations:**
- TWAP updates batched per block
- Nonce cleanup batched (100 blocks/cycle)
- Order matching batched

**Caching:**
- Oracle prices cached per block
- Pool reserves cached during multi-hop swaps
- Provider capabilities cached

## Monitoring & Observability

### Prometheus Metrics

**Compute Module:**
```
paw_compute_provider_count
paw_compute_active_requests
paw_compute_total_completed
paw_compute_provider_reputation{provider}
paw_compute_slashing_events
paw_compute_nonce_cleanups_total
```

**DEX Module:**
```
paw_dex_pool_count
paw_dex_swap_count
paw_dex_total_volume
paw_dex_pool_liquidity{pool_id}
paw_dex_order_count
```

**Oracle Module:**
```
paw_oracle_price_count
paw_oracle_validator_submissions
paw_oracle_miss_rate{validator}
paw_oracle_price_age{asset}
paw_oracle_validator_count{region}
```

### Event Emission

**Critical Events:**
- `compute_result_verified`: Result validation completed
- `dex_swap`: Trade executed
- `oracle_price_aggregated`: Consensus price finalized
- `compute_provider_slashed`: Economic penalty applied
- `circuit_breaker_open/close`: Emergency state changes

## Upgrade & Governance

### Module Governance

**Parameter Updates:**
- Governance proposals for param changes
- Example: Adjust swap fees, slashing percentages, vote periods
- Two-week voting period (default)

**Circuit Breaker Control:**
- Governance-only circuit breaker closure
- Emergency multisig for opening (future)

**IBC Channel Management:**
- Add/remove authorized channels via governance
- Security-critical: requires high quorum

### Upgrade Path

**Cosmos SDK Upgrades:**
- Cosmovisor for automated upgrades
- State migration handlers per module
- Backward-compatible protobuf definitions

**Module Versioning:**
- Current: v1 (paw.compute.v1, paw.dex.v1, paw.oracle.v1)
- Future: v2 with additional features

## Future Enhancements

### Planned Features

**Compute:**
- TEE (Trusted Execution Environment) integration
- Multi-provider redundancy for critical jobs
- GPU workload support
- Distributed storage (IPFS/Arweave)

**DEX:**
- Concentrated liquidity (Uniswap v3 style)
- Multi-hop routing optimization
- Liquidity mining rewards
- Cross-chain aggregation

**Oracle:**
- Advanced outlier detection (DBSCAN)
- Dynamic vote periods (volatility-based)
- Multi-sig oracle for critical assets
- Zero-knowledge price commitments

### Research Areas

**Cross-Module:**
- Fully homomorphic encryption (FHE) for private computation
- Verifiable delay functions (VDF) for fairness
- Cross-chain oracle aggregation
- MEV mitigation strategies

## References

- [Cosmos SDK Documentation](https://docs.cosmos.network/)
- [Tendermint Consensus](https://docs.tendermint.com/)
- [IBC Protocol](https://github.com/cosmos/ibc)
- [Groth16 ZK-SNARKs](https://eprint.iacr.org/2016/260.pdf)
- [Uniswap V2 Whitepaper](https://uniswap.org/whitepaper.pdf)
- [gnark ZK Framework](https://github.com/consensys/gnark)

---

**Document Version:** 1.0
**Last Updated:** 2025-12-25
**Maintainers:** PAW Core Development Team
