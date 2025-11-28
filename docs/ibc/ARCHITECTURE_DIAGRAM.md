# PAW Blockchain - IBC Architecture Diagrams

## Overview Architecture

```
┌───────────────────────────────────────────────────────────────────────────────┐
│                           PAW BLOCKCHAIN APPLICATION                           │
│                                                                                │
│  ┌─────────────────────────────────────────────────────────────────────────┐  │
│  │                          app/app.go (Main App)                           │  │
│  │                                                                          │  │
│  │  ┌────────────────────────────────────────────────────────────────────┐ │  │
│  │  │                      IBC ROUTER                                     │ │  │
│  │  │                                                                     │ │  │
│  │  │  Routes:                                                            │ │  │
│  │  │  • transfer    -> TransferIBCModule                                 │ │  │
│  │  │  • compute     -> ComputeIBCModule                                  │ │  │
│  │  │  • dex         -> DEXIBCModule                                      │ │  │
│  │  │  • oracle      -> OracleIBCModule                                   │ │  │
│  │  └────────────────────────────────────────────────────────────────────┘ │  │
│  │                                                                          │  │
│  │  ┌────────────────────────────────────────────────────────────────────┐ │  │
│  │  │               Capability Keeper (IBC Security)                      │ │  │
│  │  │                                                                     │ │  │
│  │  │  Scoped Keepers:                                                    │ │  │
│  │  │  • ScopedIBCKeeper                                                  │ │  │
│  │  │  • ScopedTransferKeeper                                             │ │  │
│  │  │  • ScopedComputeKeeper    (Port: "compute")                         │ │  │
│  │  │  • ScopedDEXKeeper        (Port: "dex")                             │ │  │
│  │  │  • ScopedOracleKeeper     (Port: "oracle")                          │ │  │
│  │  └────────────────────────────────────────────────────────────────────┘ │  │
│  └─────────────────────────────────────────────────────────────────────────┘  │
│                                                                                │
│  ┌──────────────┐   ┌──────────────┐   ┌──────────────┐   ┌──────────────┐  │
│  │   Compute    │   │     DEX      │   │    Oracle    │   │   Transfer   │  │
│  │   Module     │   │    Module    │   │    Module    │   │    Module    │  │
│  │              │   │              │   │              │   │              │  │
│  │ ┌──────────┐ │   │ ┌──────────┐ │   │ ┌──────────┐ │   │ ┌──────────┐ │  │
│  │ │IBC Module│ │   │ │IBC Module│ │   │ │IBC Module│ │   │ │IBC Module│ │  │
│  │ └────┬─────┘ │   │ └────┬─────┘ │   │ └────┬─────┘ │   │ └────┬─────┘ │  │
│  │      │       │   │      │       │   │      │       │   │      │       │  │
│  │ ┌────▼─────┐ │   │ ┌────▼─────┐ │   │ ┌────▼─────┐ │   │ ┌────▼─────┐ │  │
│  │ │ Keeper   │ │   │ │ Keeper   │ │   │ │ Keeper   │ │   │ │ Keeper   │ │  │
│  │ │          │ │   │ │          │ │   │ │          │ │   │ │          │ │  │
│  │ │OnRecv    │ │   │ │OnRecv    │ │   │ │OnRecv    │ │   │ │OnRecv    │ │  │
│  │ │OnAck     │ │   │ │OnAck     │ │   │ │OnAck     │ │   │ │OnAck     │ │  │
│  │ │OnTimeout │ │   │ │OnTimeout │ │   │ │OnTimeout │ │   │ │OnTimeout │ │  │
│  │ └──────────┘ │   │ └──────────┘ │   │ └──────────┘ │   │ └──────────┘ │  │
│  └──────────────┘   └──────────────┘   └──────────────┘   └──────────────┘  │
│                                                                                │
└────────────────────────────────────────┬───────────────────────────────────────┘
                                         │
                                         │ IBC Protocol
                                         │
                     ┌───────────────────┼───────────────────┐
                     │                   │                   │
          ┌──────────▼──────────┐  ┌─────▼──────┐  ┌────────▼────────┐
          │   Akash Network     │  │  Osmosis   │  │ Band Protocol   │
          │  (Compute Provider) │  │    DEX     │  │     Oracle      │
          └─────────────────────┘  └────────────┘  └─────────────────┘
```

---

## Compute Module IBC Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        COMPUTE MODULE IBC ARCHITECTURE                       │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│  x/compute/ibc_module.go                                                     │
│                                                                              │
│  IBCModule                                                                   │
│  ├─ OnChanOpenInit()      - Validate ORDERED channel                        │
│  ├─ OnChanOpenTry()       - Confirm channel establishment                   │
│  ├─ OnChanOpenAck()       - Finalize channel opening                        │
│  ├─ OnChanOpenConfirm()   - Confirm channel ready                           │
│  ├─ OnChanCloseInit()     - Reject user-initiated close                     │
│  ├─ OnChanCloseConfirm()  - Handle channel closure                          │
│  │                                                                           │
│  ├─ OnRecvPacket()        ──┐                                               │
│  │                          │                                               │
│  │  Routes to:              │                                               │
│  │  • handleJobResult()          - Process computation results              │
│  │  • handleDiscoverProviders()  - Return provider list                     │
│  │  • handleSubmitJob()          - Accept job submission                    │
│  │  • handleJobStatusQuery()     - Return job status                        │
│  │                          │                                               │
│  ├─ OnAcknowledgementPacket() ─> Keeper.OnAcknowledgementPacket()           │
│  └─ OnTimeoutPacket()         ─> Keeper.OnTimeoutPacket()                   │
└───────────────────────────┬─────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│  x/compute/keeper/ibc_compute.go                                             │
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │  OUTBOUND OPERATIONS (Send to Remote Chain)                         │   │
│  ├─────────────────────────────────────────────────────────────────────┤   │
│  │  DiscoverRemoteProviders()                                           │   │
│  │    ├─ Get compute channel for target chain                           │   │
│  │    ├─ Create DiscoverProvidersPacket                                 │   │
│  │    ├─ Send via sendComputeIBCPacket()                                │   │
│  │    └─ Store pending discovery                                        │   │
│  │                                                                       │   │
│  │  SubmitCrossChainJob()                                                │   │
│  │    ├─ Validate job size                                               │   │
│  │    ├─ Lock escrow (via bankKeeper)                                    │   │
│  │    ├─ Create Merkle escrow proof                                      │   │
│  │    ├─ Create SubmitJobPacket                                          │   │
│  │    ├─ Send via sendComputeIBCPacket()                                 │   │
│  │    └─ Store job record                                                │   │
│  │                                                                       │   │
│  │  QueryCrossChainJobStatus()                                           │   │
│  │    ├─ Get job from store                                              │   │
│  │    ├─ Create JobStatusPacket                                          │   │
│  │    └─ Send via sendComputeIBCPacket()                                 │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │  INBOUND OPERATIONS (Receive from Remote Chain)                     │   │
│  ├─────────────────────────────────────────────────────────────────────┤   │
│  │  OnRecvPacket()                                                      │   │
│  │    ├─ Parse packet type                                               │   │
│  │    └─ Route to handleJobResult()                                      │   │
│  │         ├─ Get job from store                                         │   │
│  │         ├─ Verify result hash (SHA-256, constant-time)                │   │
│  │         ├─ Verify ZK proof (Groth16 on BN254)                         │   │
│  │         │   ├─ Deserialize proof                                      │   │
│  │         │   ├─ Validate curve points                                  │   │
│  │         │   └─ Perform pairing check                                  │   │
│  │         ├─ Verify attestations (2/3+ threshold)                       │   │
│  │         │   ├─ Get validator public keys                              │   │
│  │         │   ├─ Verify each ECDSA signature (secp256k1)                │   │
│  │         │   └─ Check threshold met                                    │   │
│  │         ├─ Update job status to "completed"                           │   │
│  │         ├─ Release escrow to provider                                 │   │
│  │         └─ Return success acknowledgement                             │   │
│  │                                                                       │   │
│  │  OnAcknowledgementPacket()                                            │   │
│  │    ├─ Parse acknowledgement                                           │   │
│  │    └─ Route based on original packet type:                            │   │
│  │         ├─ handleDiscoverProvidersAck()  - Cache providers            │   │
│  │         ├─ handleSubmitJobAck()          - Update job status          │   │
│  │         └─ handleJobStatusAck()          - Update local cache         │   │
│  │                                                                       │   │
│  │  OnTimeoutPacket()                                                    │   │
│  │    ├─ Parse packet type                                               │   │
│  │    └─ Route to handler:                                               │   │
│  │         ├─ DiscoverProviders: Remove pending                          │   │
│  │         ├─ SubmitJob: Refund escrow to requester                      │   │
│  │         └─ JobStatus: Non-critical, ignore                            │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │  SECURITY & VERIFICATION                                            │   │
│  ├─────────────────────────────────────────────────────────────────────┤   │
│  │  createEscrowProof()                                                  │   │
│  │    ├─ Serialize escrow data                                           │   │
│  │    ├─ Compute leaf hash (SHA-256)                                     │   │
│  │    ├─ Build Merkle path                                               │   │
│  │    └─ Return proof bytes                                              │   │
│  │                                                                       │   │
│  │  verifyZKProof()                                                      │   │
│  │    ├─ Deserialize Groth16 proof (BN254)                               │   │
│  │    ├─ Validate proof structure                                        │   │
│  │    │   ├─ Check A is on curve                                         │   │
│  │    │   ├─ Check B is on curve                                         │   │
│  │    │   └─ Check C is on curve                                         │   │
│  │    └─ Perform pairing check:                                          │   │
│  │        e(A, B) = e(α, β) · e(C, δ) · e(pub, γ)                        │   │
│  │                                                                       │   │
│  │  verifyAttestations()                                                 │   │
│  │    ├─ Check attestation count ≥ 2/3 threshold                         │   │
│  │    ├─ For each attestation:                                           │   │
│  │    │   ├─ Parse public key (33/65 bytes)                              │   │
│  │    │   ├─ Verify ECDSA signature (constant-time)                      │   │
│  │    │   └─ Increment valid count                                       │   │
│  │    └─ Check threshold met                                             │   │
│  │                                                                       │   │
│  │  lockEscrow() / refundEscrow() / releaseEscrow()                      │   │
│  │    └─ Interact with BankKeeper for fund transfers                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## DEX Module IBC Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                          DEX MODULE IBC ARCHITECTURE                         │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│  x/dex/ibc_module.go                                                         │
│                                                                              │
│  IBCModule                                                                   │
│  ├─ OnChanOpenInit()      - Validate UNORDERED channel                      │
│  ├─ OnChanOpenTry()       - Confirm channel establishment                   │
│  ├─ OnChanOpenAck()       - Finalize channel opening                        │
│  ├─ OnChanOpenConfirm()   - Confirm channel ready                           │
│  ├─ OnChanCloseInit()     - Reject user-initiated close                     │
│  ├─ OnChanCloseConfirm()  - Handle channel closure                          │
│  │                                                                           │
│  ├─ OnRecvPacket()        ──┐                                               │
│  │                          │                                               │
│  │  Routes to:              │                                               │
│  │  • handleQueryPools()         - Return pool information                  │
│  │  • handleExecuteSwap()        - Execute local swap                       │
│  │  • handleCrossChainSwap()     - Multi-hop swap routing                   │
│  │  • handlePoolUpdate()         - Process state sync                       │
│  │                          │                                               │
│  ├─ OnAcknowledgementPacket() ─> Keeper.OnAcknowledgementPacket()           │
│  └─ OnTimeoutPacket()         ─> Keeper.OnTimeoutPacket()                   │
└───────────────────────────┬─────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│  x/dex/keeper/ibc_aggregation.go                                             │
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │  LIQUIDITY AGGREGATION                                              │   │
│  ├─────────────────────────────────────────────────────────────────────┤   │
│  │  QueryCrossChainPools()                                              │   │
│  │    ├─ For each target chain:                                         │   │
│  │    │   ├─ Get IBC connection                                         │   │
│  │    │   ├─ Create QueryPoolsPacket                                    │   │
│  │    │   ├─ Send via sendIBCPacket()                                   │   │
│  │    │   └─ Store pending query                                        │   │
│  │    └─ Return cached pools (async)                                    │   │
│  │                                                                       │   │
│  │  FindBestCrossChainRoute()                                            │   │
│  │    ├─ Query local pools                                               │   │
│  │    ├─ Query remote pools (via QueryCrossChainPools)                   │   │
│  │    ├─ Combine all pools                                               │   │
│  │    └─ Run routing algorithm (find optimal path)                       │   │
│  │         ├─ Consider liquidity depth                                   │   │
│  │         ├─ Compare fees                                               │   │
│  │         └─ Calculate expected slippage                                │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │  CROSS-CHAIN SWAP EXECUTION                                         │   │
│  ├─────────────────────────────────────────────────────────────────────┤   │
│  │  ExecuteCrossChainSwap()                                             │   │
│  │    ├─ Validate route                                                  │   │
│  │    ├─ For each step in route:                                         │   │
│  │    │   ├─ If local chain:                                             │   │
│  │    │   │   └─ executeLocalSwap()                                      │   │
│  │    │   └─ If remote chain:                                            │   │
│  │    │       └─ executeRemoteSwap()                                     │   │
│  │    │           ├─ Create ExecuteSwapPacket                            │   │
│  │    │           ├─ Send via IBC                                        │   │
│  │    │           └─ Wait for acknowledgement                            │   │
│  │    ├─ Calculate final slippage                                        │   │
│  │    └─ Verify max slippage not exceeded                                │   │
│  │                                                                       │   │
│  │  executeLocalSwap()                                                   │   │
│  │    ├─ Get pool from state                                             │   │
│  │    ├─ Calculate output amount                                         │   │
│  │    ├─ Execute swap                                                    │   │
│  │    └─ Update pool reserves                                            │   │
│  │                                                                       │   │
│  │  executeRemoteSwap()                                                  │   │
│  │    ├─ Lock input tokens                                               │   │
│  │    ├─ Create swap packet                                              │   │
│  │    ├─ Send via IBC                                                    │   │
│  │    └─ Store pending swap                                              │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │  PACKET HANDLERS                                                    │   │
│  ├─────────────────────────────────────────────────────────────────────┤   │
│  │  OnAcknowledgementPacket()                                           │   │
│  │    ├─ Parse acknowledgement                                           │   │
│  │    └─ Route based on packet type:                                     │   │
│  │         ├─ QueryPools: Cache pool data                                │   │
│  │         └─ ExecuteSwap: Process swap result                           │   │
│  │             ├─ Update balances                                        │   │
│  │             ├─ Transfer output tokens                                 │   │
│  │             └─ Emit swap completed event                              │   │
│  │                                                                       │   │
│  │  OnTimeoutPacket()                                                    │   │
│  │    ├─ Parse packet type                                               │   │
│  │    └─ Route to handler:                                               │   │
│  │         ├─ QueryPools: Remove pending query                           │   │
│  │         └─ ExecuteSwap: Refund locked tokens                          │   │
│  │             ├─ Get pending swap                                       │   │
│  │             ├─ Return tokens to sender                                │   │
│  │             └─ Emit timeout event                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │  STATE MANAGEMENT                                                   │   │
│  ├─────────────────────────────────────────────────────────────────────┤   │
│  │  getCachedPools()         - Retrieve cached pool info                │   │
│  │  storePendingQuery()      - Track async pool queries                 │   │
│  │  getLocalPools()          - Query local DEX pools                    │   │
│  │  findOptimalRoute()       - Routing algorithm implementation         │   │
│  │  calculateSlippage()      - Slippage calculation                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## Oracle Module IBC Flow

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                        ORACLE MODULE IBC ARCHITECTURE                        │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│  x/oracle/ibc_module.go                                                      │
│                                                                              │
│  IBCModule                                                                   │
│  ├─ OnChanOpenInit()      - Validate UNORDERED channel                      │
│  ├─ OnChanOpenTry()       - Confirm channel establishment                   │
│  ├─ OnChanOpenAck()       - Finalize channel opening                        │
│  ├─ OnChanOpenConfirm()   - Confirm channel ready                           │
│  ├─ OnChanCloseInit()     - Reject user-initiated close                     │
│  ├─ OnChanCloseConfirm()  - Handle channel closure                          │
│  │                                                                           │
│  ├─ OnRecvPacket()        ──┐                                               │
│  │                          │                                               │
│  │  Routes to:              │                                               │
│  │  • handleSubscribePrices()    - Subscribe to price feeds                 │
│  │  • handleQueryPrice()         - Return current price                     │
│  │  • handlePriceUpdate()        - Process price update                     │
│  │  • handleOracleHeartbeat()    - Process liveness signal                  │
│  │                          │                                               │
│  ├─ OnAcknowledgementPacket() ─> Keeper.OnAcknowledgementPacket()           │
│  └─ OnTimeoutPacket()         ─> Keeper.OnTimeoutPacket()                   │
└───────────────────────────┬─────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│  x/oracle/keeper/ibc_prices.go                                               │
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │  ORACLE SOURCE MANAGEMENT                                           │   │
│  ├─────────────────────────────────────────────────────────────────────┤   │
│  │  RegisterCrossChainOracleSource()                                    │   │
│  │    ├─ Create CrossChainOracleSource                                  │   │
│  │    │   ├─ Set chainID, oracleType (band/slinky/uma)                 │   │
│  │    │   ├─ Initialize reputation = 1.0                                │   │
│  │    │   └─ Set active = true                                          │   │
│  │    ├─ Store in KV store                                               │   │
│  │    └─ Emit registration event                                         │   │
│  │                                                                       │   │
│  │  getCrossChainOracleSource()                                         │   │
│  │    ├─ Load from KV store                                              │   │
│  │    └─ Check if active                                                 │   │
│  │                                                                       │   │
│  │  updateOracleQueryStats()                                             │   │
│  │    ├─ Increment total queries                                         │   │
│  │    ├─ Increment successful queries                                    │   │
│  │    └─ Recalculate reputation score                                    │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │  PRICE SUBSCRIPTION & QUERIES                                       │   │
│  ├─────────────────────────────────────────────────────────────────────┤   │
│  │  SubscribeToCrossChainPrices()                                       │   │
│  │    ├─ For each source chain:                                          │   │
│  │    │   ├─ Get oracle source                                           │   │
│  │    │   ├─ Validate active status                                      │   │
│  │    │   ├─ Create SubscribePricesPacket                                │   │
│  │    │   ├─ Send via sendOracleIBCPacket()                              │   │
│  │    │   └─ Store subscription                                          │   │
│  │    └─ Wait for periodic price updates                                 │   │
│  │                                                                       │   │
│  │  QueryCrossChainPrice()                                               │   │
│  │    ├─ Get oracle source                                               │   │
│  │    ├─ Validate active status                                          │   │
│  │    ├─ Create QueryPricePacket                                         │   │
│  │    ├─ Send via sendOracleIBCPacket()                                  │   │
│  │    ├─ Update query stats                                              │   │
│  │    ├─ Store pending query                                             │   │
│  │    └─ Return cached price (async)                                     │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │  PRICE AGGREGATION & CONSENSUS                                      │   │
│  ├─────────────────────────────────────────────────────────────────────┤   │
│  │  AggregateCrossChainPrices()                                         │   │
│  │    ├─ Collect prices from all sources                                 │   │
│  │    │   ├─ Local oracle prices                                         │   │
│  │    │   └─ Remote oracle prices (from cache)                           │   │
│  │    ├─ Filter stale prices (> 5 minutes old)                           │   │
│  │    ├─ Detect and remove outliers                                      │   │
│  │    │   ├─ Calculate median                                            │   │
│  │    │   └─ Remove prices > 10% deviation                               │   │
│  │    ├─ Check Byzantine fault tolerance                                 │   │
│  │    │   └─ Require 2/3+ sources in agreement                           │   │
│  │    ├─ Calculate weighted average                                      │   │
│  │    │   └─ Weight by source reputation                                 │   │
│  │    ├─ Calculate confidence score                                      │   │
│  │    └─ Return AggregatedCrossChainPrice                                │   │
│  │         ├─ Weighted price                                             │   │
│  │         ├─ Median price                                               │   │
│  │         ├─ Source list                                                │   │
│  │         ├─ Confidence (0.0-1.0)                                       │   │
│  │         └─ ByzantineSafe flag                                         │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │  PACKET HANDLERS                                                    │   │
│  ├─────────────────────────────────────────────────────────────────────┤   │
│  │  OnRecvPacket() - handlePriceUpdate()                                │   │
│  │    ├─ Parse PriceUpdatePacket                                         │   │
│  │    ├─ Validate each price:                                            │   │
│  │    │   ├─ Check symbol not empty                                      │   │
│  │    │   ├─ Check price > 0                                             │   │
│  │    │   └─ Check confidence in [0, 1]                                  │   │
│  │    ├─ Store prices in cache                                           │   │
│  │    ├─ Update source last heartbeat                                    │   │
│  │    ├─ Emit price update events                                        │   │
│  │    └─ Return success acknowledgement                                  │   │
│  │                                                                       │   │
│  │  OnAcknowledgementPacket()                                            │   │
│  │    ├─ Parse acknowledgement                                           │   │
│  │    └─ Route based on packet type:                                     │   │
│  │         ├─ SubscribePrices: Confirm subscription                      │   │
│  │         └─ QueryPrice: Cache price data                               │   │
│  │             ├─ Store in local cache                                   │   │
│  │             └─ Update oracle reputation                               │   │
│  │                                                                       │   │
│  │  OnTimeoutPacket()                                                    │   │
│  │    ├─ Parse packet type                                               │   │
│  │    └─ Route to handler:                                               │   │
│  │         ├─ SubscribePrices: Mark subscription failed                  │   │
│  │         ├─ QueryPrice: Use cached value                               │   │
│  │         └─ Decrease oracle reputation                                 │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐   │
│  │  STATE MANAGEMENT                                                   │   │
│  ├─────────────────────────────────────────────────────────────────────┤   │
│  │  storeSubscription()         - Track price subscriptions             │   │
│  │  storePendingPriceQuery()    - Track async queries                   │   │
│  │  getCachedPrice()            - Retrieve cached price                 │   │
│  │  sendOracleIBCPacket()       - Send IBC packet helper                │   │
│  └─────────────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────────────┘
```

---

## IBC Packet Lifecycle

```
┌───────────────────────────────────────────────────────────────────────────┐
│                        IBC PACKET LIFECYCLE FLOW                           │
└───────────────────────────────────────────────────────────────────────────┘

                    ┌──────────────────────────────────┐
                    │    Source Chain (PAW)            │
                    └──────────────────────────────────┘
                                    │
                    ┌───────────────▼──────────────────┐
                    │  1. Create Packet                 │
                    │     - PacketData (JSON)           │
                    │     - Source port/channel         │
                    │     - Dest port/channel           │
                    │     - Timeout height/timestamp    │
                    └───────────────┬──────────────────┘
                                    │
                    ┌───────────────▼──────────────────┐
                    │  2. Send via IBCKeeper           │
                    │     - Validate packet             │
                    │     - Increment sequence          │
                    │     - Store commitment            │
                    │     - Emit SendPacket event       │
                    └───────────────┬──────────────────┘
                                    │
                    ┌───────────────▼──────────────────┐
                    │  3. Relayer picks up packet      │
                    │     - Monitor SendPacket events   │
                    │     - Create MsgRecvPacket        │
                    │     - Submit to dest chain        │
                    └───────────────┬──────────────────┘
                                    │
                    ┌───────────────▼──────────────────┐
                    │  Destination Chain               │
                    └───────────────┬──────────────────┘
                                    │
                    ┌───────────────▼──────────────────┐
                    │  4. OnRecvPacket()               │
                    │     - Verify packet proofs        │
                    │     - Route to module             │
                    │     - Process packet data         │
                    │     - Execute business logic      │
                    │     - Return acknowledgement      │
                    └───────────────┬──────────────────┘
                                    │
                    ┌───────────────▼──────────────────┐
                    │  5. Write Acknowledgement        │
                    │     - Store ack in state          │
                    │     - Emit WriteAck event         │
                    └───────────────┬──────────────────┘
                                    │
                    ┌───────────────▼──────────────────┐
                    │  6. Relayer picks up ack         │
                    │     - Monitor WriteAck events     │
                    │     - Create MsgAcknowledgement   │
                    │     - Submit to source chain      │
                    └───────────────┬──────────────────┘
                                    │
                    ┌───────────────▼──────────────────┐
                    │  Source Chain (PAW)              │
                    └───────────────┬──────────────────┘
                                    │
                    ┌───────────────▼──────────────────┐
                    │  7. OnAcknowledgementPacket()    │
                    │     - Verify ack proofs           │
                    │     - Route to module             │
                    │     - Process acknowledgement     │
                    │     - Update state                │
                    │     - Delete packet commitment    │
                    └───────────────┬──────────────────┘
                                    │
                    ┌───────────────▼──────────────────┐
                    │  8. Packet Complete              │
                    │     - Emit AckPacket event        │
                    │     - Transaction finalized       │
                    └──────────────────────────────────┘

                         ──── TIMEOUT PATH ────

                    ┌──────────────────────────────────┐
                    │  Timeout Condition Met           │
                    │  (Height or Timestamp exceeded)   │
                    └───────────────┬──────────────────┘
                                    │
                    ┌───────────────▼──────────────────┐
                    │  OnTimeoutPacket()               │
                    │     - Verify timeout proof        │
                    │     - Route to module             │
                    │     - Refund/Rollback             │
                    │     - Delete packet commitment    │
                    └───────────────┬──────────────────┘
                                    │
                    ┌───────────────▼──────────────────┐
                    │  Timeout Handled                 │
                    │     - Emit TimeoutPacket event    │
                    │     - User funds returned         │
                    └──────────────────────────────────┘
```

---

## Complete System Integration

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                     PAW BLOCKCHAIN IBC ECOSYSTEM                             │
└─────────────────────────────────────────────────────────────────────────────┘

┌─────────────────────────────────────────────────────────────────────────────┐
│                               PAW CHAIN                                      │
│                                                                              │
│  ┌──────────────────────────────────────────────────────────────────────┐  │
│  │                          Application                                  │  │
│  │  ┌────────────┐  ┌────────────┐  ┌────────────┐  ┌────────────┐    │  │
│  │  │  Compute   │  │    DEX     │  │   Oracle   │  │  Transfer  │    │  │
│  │  │   Module   │  │   Module   │  │   Module   │  │   Module   │    │  │
│  │  └─────┬──────┘  └─────┬──────┘  └─────┬──────┘  └─────┬──────┘    │  │
│  │        │               │               │               │            │  │
│  │        └───────────────┴───────────────┴───────────────┘            │  │
│  │                              │                                       │  │
│  │                    ┌─────────▼─────────┐                            │  │
│  │                    │   IBC Router       │                            │  │
│  │                    └─────────┬─────────┘                            │  │
│  │                              │                                       │  │
│  │                    ┌─────────▼─────────┐                            │  │
│  │                    │   IBC Keeper       │                            │  │
│  │                    └─────────┬─────────┘                            │  │
│  └──────────────────────────────┼─────────────────────────────────────┘  │
│                                 │                                         │
│  ┌──────────────────────────────▼─────────────────────────────────────┐  │
│  │                       Consensus Engine                              │  │
│  │                       (CometBFT)                                    │  │
│  └──────────────────────────────┬─────────────────────────────────────┘  │
└─────────────────────────────────┼───────────────────────────────────────┘
                                  │
                                  │  IBC Protocol
                                  │  (via Relayers)
                                  │
        ┌─────────────────────────┼─────────────────────────┐
        │                         │                         │
        │                         │                         │
┌───────▼────────┐      ┌─────────▼────────┐      ┌────────▼────────┐
│ Akash Network  │      │    Osmosis       │      │ Band Protocol   │
├────────────────┤      ├──────────────────┤      ├─────────────────┤
│                │      │                  │      │                 │
│ Port: compute  │      │   Port: dex      │      │  Port: oracle   │
│                │      │                  │      │                 │
│ • Job Execution│      │ • Liquidity Pools│      │ • Price Feeds   │
│ • TEE Support  │      │ • Token Swaps    │      │ • Data Oracles  │
│ • GPU Compute  │      │ • AMM Protocol   │      │ • Aggregation   │
└────────────────┘      └──────────────────┘      └─────────────────┘
        │                         │                         │
        └─────────────────────────┼─────────────────────────┘
                                  │
                    ┌─────────────▼─────────────┐
                    │       Hermes Relayer       │
                    │                            │
                    │  • Packet forwarding       │
                    │  • Event monitoring        │
                    │  • Proof generation        │
                    │  • Multi-chain support     │
                    └────────────────────────────┘
```

---

**Generated:** 2025-11-25
**Document Version:** 1.0
**Status:** Complete IBC Integration Architecture
