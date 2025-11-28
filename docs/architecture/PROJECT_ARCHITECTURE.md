# PAW Blockchain - Production Architecture Specification

**Version**: 3.0
**Date**: 2025-01-24
**Status**: OFFICIAL ARCHITECTURE - PURE BLOCKCHAIN ONLY
**Classification**: Production-Ready Blueprint

---

## ⚠️ CRITICAL ARCHITECTURE DECISION ⚠️

**PAW IS A PURE BLOCKCHAIN - NOT A CLIENT-SERVER APPLICATION**

**ALL business logic MUST be in blockchain modules (x/*/keeper/)**
**The api/ directory is DELETED - it was a mock implementation mistake**
**This is non-negotiable for crypto community acceptance**

---

## ARCHITECTURAL PRINCIPLES

This document defines the official, production-grade architecture for the PAW blockchain project as a pure Cosmos SDK blockchain with zero centralized components.

### Core Tenets:
1. **Single Source of Truth**: All state resides on blockchain
2. **Standard Cosmos SDK Patterns**: No custom workarounds
3. **Separation of Concerns**: Clean boundaries between layers
4. **Production Quality**: No mock implementations in production code
5. **Scalability First**: Multi-node consensus from day one

---

## SYSTEM ARCHITECTURE

```
┌─────────────────────────────────────────────────────────────────┐
│                     CLIENT APPLICATIONS                          │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐ ┌──────────┐          │
│  │   CLI    │ │   Web    │ │  Mobile  │ │  Browser │          │
│  │  (pawd)  │ │    UI    │ │   App    │ │Extension │          │
│  └────┬─────┘ └────┬─────┘ └────┬─────┘ └────┬─────┘          │
└───────┼────────────┼────────────┼────────────┼─────────────────┘
        │            │            │            │
        └────────────┴────────────┴────────────┘
                     │
                     │ gRPC/REST
                     ▼
┌─────────────────────────────────────────────────────────────────┐
│              ❌ NO API GATEWAY - DELETED ❌                       │
│                                                                   │
│  The previous api/ directory has been PERMANENTLY DELETED        │
│  It contained mock implementations that violate blockchain       │
│  principles and crypto community standards.                      │
│                                                                   │
│  Clients connect DIRECTLY to blockchain node:                   │
│  ├─ gRPC: localhost:9090                                         │
│  ├─ REST: localhost:1317 (auto-generated from gRPC)             │
│  └─ Tendermint RPC: localhost:26657                             │
│                                                                   │
│  ALL business logic is in x/*/keeper/ modules                   │
└────────────────────────┬────────────────────────────────────────┘
                         │
                         │ gRPC (localhost:9090)
                         ▼
┌─────────────────────────────────────────────────────────────────┐
│                    COSMOS SDK BLOCKCHAIN NODE                    │
│                         (pawd daemon)                            │
│                                                                   │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │                    GRPC/REST SERVERS                        │ │
│  │  ┌──────────────┐  ┌──────────────┐  ┌─────────────────┐  │ │
│  │  │ gRPC Server  │  │ REST Gateway │  │  Tendermint RPC │  │ │
│  │  │  (Port 9090) │  │ (Port 1317)  │  │   (Port 26657)  │  │ │
│  │  └──────────────┘  └──────────────┘  └─────────────────┘  │ │
│  └────────────────────────────────────────────────────────────┘ │
│                                                                   │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │                  APPLICATION LAYER (app.go)                 │ │
│  │                                                              │ │
│  │  ┌────────────┐  ┌────────────┐  ┌────────────┐           │ │
│  │  │   Router   │  │  Tx Codec  │  │ Query Codec│           │ │
│  │  └────────────┘  └────────────┘  └────────────┘           │ │
│  └────────────────────────────────────────────────────────────┘ │
│                                                                   │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │                   BUSINESS LOGIC (Modules)                  │ │
│  │                                                              │ │
│  │  ┌──────────────────────────────────────────────────────┐  │ │
│  │  │  Cosmos SDK Standard Modules                         │  │ │
│  │  │  • auth      (accounts, signatures)                  │  │ │
│  │  │  • bank      (token transfers)                       │  │ │
│  │  │  • staking   (validator delegation)                  │  │ │
│  │  │  • gov       (governance proposals)                  │  │ │
│  │  │  • distribution (rewards)                            │  │ │
│  │  │  • slashing  (validator penalties)                   │  │ │
│  │  └──────────────────────────────────────────────────────┘  │ │
│  │                                                              │ │
│  │  ┌──────────────────────────────────────────────────────┐  │ │
│  │  │  IBC Modules (PHASE 1 - TO BE IMPLEMENTED)           │  │ │
│  │  │  • ibc-core     (light clients, connections)         │  │ │
│  │  │  • ibc-transfer (cross-chain tokens)                 │  │ │
│  │  │  • capability   (port authorization)                 │  │ │
│  │  └──────────────────────────────────────────────────────┘  │ │
│  │                                                              │ │
│  │  ┌──────────────────────────────────────────────────────┐  │ │
│  │  │  Smart Contract Module (PHASE 2 - BLOCKED BY IBC)    │  │ │
│  │  │  • wasm         (CosmWasm v3.0.1)                    │  │ │
│  │  │    - Contract upload (governance-only)               │  │ │
│  │  │    - Contract instantiation                          │  │ │
│  │  │    - Contract execution                              │  │ │
│  │  │    - Contract queries                                │  │ │
│  │  │    - Contract migration                              │  │ │
│  │  └──────────────────────────────────────────────────────┘  │ │
│  │                                                              │ │
│  │  ┌──────────────────────────────────────────────────────┐  │ │
│  │  │  PAW Custom Modules (TO BE IMPLEMENTED)              │  │ │
│  │  │                                                        │  │ │
│  │  │  • x/dex (Decentralized Exchange)                    │  │ │
│  │  │    Keeper Methods:                                    │  │ │
│  │  │    - CreatePool(tokenA, tokenB, amounts)             │  │ │
│  │  │    - AddLiquidity(poolID, amounts) → shares          │  │ │
│  │  │    - RemoveLiquidity(poolID, shares) → amounts       │  │ │
│  │  │    - Swap(poolID, tokenIn, tokenOut, amount)         │  │ │
│  │  │    - QueryPool(poolID) → Pool                        │  │ │
│  │  │    - QueryAllPools(pagination) → []Pool              │  │ │
│  │  │                                                        │  │ │
│  │  │  • x/oracle (Price Oracle)                           │  │ │
│  │  │    Keeper Methods:                                    │  │ │
│  │  │    - RegisterOracle(validator, stake)                │  │ │
│  │  │    - SubmitPrice(asset, price, timestamp, sig)       │  │ │
│  │  │    - AggregatePrice(asset) → consensus price         │  │ │
│  │  │    - SlashOracle(validator, reason)                  │  │ │
│  │  │    - RewardOracle(validator, amount)                 │  │ │
│  │  │                                                        │  │ │
│  │  │  • x/compute (Off-chain Compute Verification)        │  │ │
│  │  │    Keeper Methods:                                    │  │ │
│  │  │    - RegisterProvider(address, endpoint, stake)      │  │ │
│  │  │    - RequestCompute(apiURL, maxFee)                  │  │ │
│  │  │    - SubmitResult(requestID, result, proof)          │  │ │
│  │  │    - VerifyResult(requestID) → bool                  │  │ │
│  │  │    - SlashProvider(address, reason)                  │  │ │
│  │  └──────────────────────────────────────────────────────┘  │ │
│  └────────────────────────────────────────────────────────────┘ │
│                                                                   │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │                 STATE STORAGE (IAVL Trees)                  │ │
│  │                                                              │ │
│  │  Persistent Key-Value Store with Merkle Proofs             │ │
│  │  • Account balances                                         │ │
│  │  • Pool reserves                                            │ │
│  │  • Oracle prices                                            │ │
│  │  • Compute requests                                         │ │
│  │  • Smart contract state                                     │ │
│  │  • Validator set                                            │ │
│  └────────────────────────────────────────────────────────────┘ │
│                                                                   │
│  ┌────────────────────────────────────────────────────────────┐ │
│  │              CONSENSUS ENGINE (CometBFT)                    │ │
│  │                                                              │ │
│  │  Byzantine Fault Tolerant Consensus                         │ │
│  │  • Block proposal and voting                                │ │
│  │  • Transaction ordering                                     │ │
│  │  • State finalization                                       │ │
│  │  • P2P networking                                           │ │
│  └────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────┘
```

---

## DIRECTORY STRUCTURE

### Production Blockchain Repository (Core)

```
paw/
├── app/                          # Application wiring
│   ├── app.go                    # Main app construction
│   ├── encoding.go               # Codec configuration
│   └── export.go                 # Genesis export
│
├── cmd/
│   └── pawd/                     # Daemon binary
│       ├── main.go               # Entry point
│       └── cmd/                  # CLI commands
│           ├── root.go
│           ├── init.go
│           ├── start.go
│           ├── keys.go
│           ├── tx.go
│           └── query.go
│
├── x/                            # Custom modules
│   ├── dex/
│   │   ├── keeper/               # Business logic
│   │   │   ├── keeper.go         # Keeper struct
│   │   │   ├── msg_server.go    # Transaction handlers
│   │   │   ├── query_server.go  # Query handlers
│   │   │   ├── pool.go           # Pool management
│   │   │   ├── swap.go           # Swap execution
│   │   │   ├── liquidity.go     # Liquidity operations
│   │   │   ├── fees.go           # Fee distribution
│   │   │   └── invariants.go    # Invariant checks
│   │   ├── types/                # Data structures
│   │   │   ├── codec.go
│   │   │   ├── keys.go
│   │   │   ├── msgs.go
│   │   │   ├── events.go
│   │   │   └── errors.go
│   │   └── module.go             # Module interface
│   │
│   ├── oracle/
│   │   ├── keeper/
│   │   │   ├── keeper.go
│   │   │   ├── msg_server.go
│   │   │   ├── query_server.go
│   │   │   ├── price.go          # Price submission
│   │   │   ├── aggregation.go   # Price consensus
│   │   │   ├── slashing.go      # Oracle slashing
│   │   │   └── rewards.go       # Oracle rewards
│   │   ├── types/
│   │   └── module.go
│   │
│   └── compute/
│       ├── keeper/
│       │   ├── keeper.go
│       │   ├── msg_server.go
│       │   ├── query_server.go
│       │   ├── provider.go      # Provider management
│       │   ├── request.go       # Request handling
│       │   └── verification.go  # Result verification
│       ├── types/
│       └── module.go
│
├── proto/                        # Protobuf definitions
│   └── paw/
│       ├── dex/v1/
│       │   ├── tx.proto          # Transaction messages
│       │   ├── query.proto       # Query requests
│       │   └── dex.proto         # State types
│       ├── oracle/v1/
│       └── compute/v1/
│
├── testutil/                     # Test utilities
│   ├── keeper/                   # Keeper test helpers
│   └── integration/              # Integration helpers
│
├── tests/                        # Test suites
│   ├── integration/              # Integration tests
│   ├── e2e/                      # End-to-end tests
│   ├── security/                 # Security tests
│   └── benchmarks/               # Performance tests
│
├── docs/                         # Documentation
│   ├── architecture/             # ADRs and design docs
│   ├── spec/                     # Module specifications
│   └── guides/                   # User guides
│
├── scripts/                      # Automation scripts
│   ├── protocgen.sh              # Protobuf code generation
│   ├── init-devnet.sh            # Local network setup
│   └── test-coverage.sh          # Coverage reporting
│
├── Dockerfile                    # Production image
├── docker-compose.yml            # Local development
├── Makefile                      # Build automation
├── go.mod                        # Go dependencies
└── README.md                     # Project overview
```

### Excluded from Core Repository (Separate Repos)

**NOT in paw/ repository**:
- ❌ api/ (separate repository: paw-api-gateway)
- ❌ wallet/ (separate repository: paw-wallet)
- ❌ sdk/ (separate repository: paw-sdk)
- ❌ docs/portal/ (separate repository: paw-docs)
- ❌ external/ (third-party code, do not maintain)
- ❌ playground/ (separate demo repository)

---

## MODULE SPECIFICATIONS

### x/dex Module

**Purpose**: Decentralized exchange with AMM pools

**State**:
```protobuf
message Pool {
  uint64 id = 1;
  string token_a = 2;
  string token_b = 3;
  string reserve_a = 4 [(cosmos_proto.scalar) = "cosmos.Int"];
  string reserve_b = 5 [(cosmos_proto.scalar) = "cosmos.Int"];
  string total_shares = 6 [(cosmos_proto.scalar) = "cosmos.Int"];
  string swap_fee = 7 [(cosmos_proto.scalar) = "cosmos.Dec"];
  string protocol_fee = 8 [(cosmos_proto.scalar) = "cosmos.Dec"];
}
```

**Messages**:
- MsgCreatePool
- MsgAddLiquidity
- MsgRemoveLiquidity
- MsgSwap

**Queries**:
- QueryPool
- QueryAllPools
- QueryPoolLiquidity
- QuerySwapEstimate

**Keeper Responsibilities**:
- Pool creation with initial liquidity
- Liquidity provision and LP token minting
- Liquidity removal and LP token burning
- Token swaps using constant product formula (x * y = k)
- Fee collection and distribution
- Invariant checking (k value maintenance)
- Event emission for indexing

### x/oracle Module

**Purpose**: Decentralized price oracle with validator consensus

**State**:
```protobuf
message PriceFeed {
  string asset = 1;
  string price = 2 [(cosmos_proto.scalar) = "cosmos.Dec"];
  int64 timestamp = 3;
  string validator = 4 [(cosmos_proto.scalar) = "cosmos.AddressString"];
  bytes signature = 5;
}

message AggregatedPrice {
  string asset = 1;
  string price = 2 [(cosmos_proto.scalar) = "cosmos.Dec"];
  int64 timestamp = 3;
  uint32 submission_count = 4;
}
```

**Messages**:
- MsgRegisterOracle
- MsgSubmitPrice
- MsgUpdateParams

**Queries**:
- QueryPrice
- QueryPriceHistory
- QueryOracleInfo
- QueryAllOracles

**Keeper Responsibilities**:
- Oracle registration with minimum stake
- Price submission from validators
- Price aggregation using median or TWAP
- Oracle slashing for incorrect submissions
- Oracle reward distribution
- Event emission for price updates

### x/compute Module

**Purpose**: Off-chain computation verification with TEE providers

**State**:
```protobuf
message Provider {
  string address = 1 [(cosmos_proto.scalar) = "cosmos.AddressString"];
  string endpoint = 2;
  string stake = 3 [(cosmos_proto.scalar) = "cosmos.Int"];
  bool active = 4;
  uint32 success_count = 5;
  uint32 failure_count = 6;
}

message ComputeRequest {
  uint64 id = 1;
  string requester = 2 [(cosmos_proto.scalar) = "cosmos.AddressString"];
  string api_url = 3;
  string max_fee = 4 [(cosmos_proto.scalar) = "cosmos.Int"];
  RequestStatus status = 5;
  string result = 6;
  string provider = 7 [(cosmos_proto.scalar) = "cosmos.AddressString"];
}
```

**Messages**:
- MsgRegisterProvider
- MsgRequestCompute
- MsgSubmitResult
- MsgUpdateParams

**Queries**:
- QueryProvider
- QueryAllProviders
- QueryRequest
- QueryRequestsByProvider

**Keeper Responsibilities**:
- Provider registration with stake locking
- Compute request creation with fee escrow
- Provider selection (stake-weighted random)
- Result submission and verification
- Fee distribution to providers
- Provider slashing for failures
- Timeout and retry handling

---

## DATA FLOW PATTERNS

### Transaction Flow

```
1. User Action
   ↓
2. Client Signs Transaction
   ↓
3. [Optional: API Gateway validates and routes]
   ↓
4. Blockchain Node receives transaction
   ↓
5. Node validates (signature, fee, nonce)
   ↓
6. Transaction enters mempool
   ↓
7. Proposer includes in block
   ↓
8. Validators vote on block
   ↓
9. Block finalized (2/3+ validators agree)
   ↓
10. Transaction executed in order
    ↓
11. Keeper method processes business logic
    ↓
12. State updated atomically
    ↓
13. Events emitted
    ↓
14. Receipt returned to client
```

### Query Flow

```
1. User Query Request
   ↓
2. [Optional: API Gateway caches/routes]
   ↓
3. Blockchain Node receives query
   ↓
4. Query router dispatches to module
   ↓
5. Keeper reads current state
   ↓
6. Response formatted (JSON/Protobuf)
   ↓
7. Returned to client
```

---

## SECURITY ARCHITECTURE

### Authentication & Authorization

**Blockchain Level**:
- Transaction signature verification (secp256k1)
- Account nonce for replay protection
- Message authorization via signers
- Module authority enforcement

**API Gateway Level** (if used):
- JWT for session management
- API key validation
- Rate limiting per user/IP
- Request validation and sanitization

### Access Control

**Module Permissions**:
```go
// Only governance can update module parameters
if msg.Authority != k.authority {
    return ErrUnauthorized
}

// Only pool creator or governance can update pool
if msg.Sender != pool.Creator && msg.Sender != k.authority {
    return ErrUnauthorized
}
```

**CosmWasm Permissions**:
- Upload: Governance-only (AllowNobody)
- Instantiate: Per-code configuration
- Execute: Contract logic + admin
- Migrate: Admin-only

### Resource Limits

**Gas Metering**:
- All operations consume gas
- Smart contract queries: 3,000,000 gas limit
- Smart contract execution: 10,000,000 gas per tx
- Block gas limit: TBD based on network capacity

**Rate Limiting** (API Gateway):
- Per-IP: 1000 requests/minute
- Per-user: 100 requests/minute
- Per-endpoint: Varies by operation cost

**Size Limits**:
- Smart contract code: 800KB max
- Transaction size: 1MB max
- Block size: TBD

---

## DEPLOYMENT ARCHITECTURE

### Single Node (Development)

```
┌────────────────────────┐
│  Developer Machine     │
│  ┌──────────────────┐  │
│  │  pawd            │  │
│  │  (validator)     │  │
│  │  Port: 26657     │  │
│  │  gRPC: 9090      │  │
│  │  REST: 1317      │  │
│  └──────────────────┘  │
└────────────────────────┘
```

### Devnet (Testing)

```
┌─────────────┐  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐
│ Validator 1 │  │ Validator 2 │  │ Validator 3 │  │ Validator 4 │
│  Stake: 25% │  │  Stake: 25% │  │  Stake: 25% │  │  Stake: 25% │
│  Port:26656 │  │  Port:26657 │  │  Port:26658 │  │  Port:26659 │
└─────────────┘  └─────────────┘  └─────────────┘  └─────────────┘
       │                │                │                │
       └────────────────┴────────────────┴────────────────┘
                             │
                    P2P Network (Tendermint)
```

### Production (Mainnet)

```
                        ┌──────────────────┐
                        │   Load Balancer  │
                        │   (API Gateway)  │
                        └────────┬─────────┘
                                 │
         ┌───────────────────────┼───────────────────────┐
         │                       │                       │
    ┌────▼────┐            ┌────▼────┐            ┌────▼────┐
    │ gRPC    │            │ gRPC    │            │ gRPC    │
    │ Endpoint│            │ Endpoint│            │ Endpoint│
    └────┬────┘            └────┬────┘            └────┬────┘
         │                      │                      │
    ┌────▼────────┐        ┌───▼─────────┐       ┌────▼────────┐
    │ Validator 1 │        │ Validator 2 │       │ Validator 3 │
    │  (Active)   │        │  (Active)   │       │  (Active)   │
    │ 30% stake   │◄──────►│ 25% stake   │◄─────►│ 20% stake   │
    └─────────────┘        └─────────────┘       └─────────────┘
          ▲                      ▲                      ▲
          │                      │                      │
          │        P2P Network (CometBFT)              │
          │                      │                      │
    ┌─────▼─────┐          ┌────▼─────┐          ┌────▼─────┐
    │ Sentinel  │          │ Sentinel │          │ Sentinel │
    │  Node 1   │          │  Node 2  │          │  Node 3  │
    │ (Backup)  │          │ (Backup) │          │ (Backup) │
    └───────────┘          └──────────┘          └──────────┘

Additional:
- Sentry nodes (DDoS protection)
- Archive nodes (full history)
- Monitoring (Prometheus + Grafana)
- Alerting (PagerDuty)
```

---

## INTEGRATION PATTERNS

### Client SDK Integration

**Go SDK**:
```go
import (
    "github.com/paw-chain/paw/x/dex/types"
    "github.com/cosmos/cosmos-sdk/client"
)

// Create client
clientCtx := client.Context{}.
    WithNodeURI("tcp://localhost:26657").
    WithChainID("paw-1")

// Query pool
queryClient := types.NewQueryClient(clientCtx)
pool, err := queryClient.Pool(ctx, &types.QueryPoolRequest{
    PoolId: 1,
})

// Create transaction
msg := types.NewMsgSwap(...)
tx, err := clientCtx.BroadcastTx(txBytes)
```

**JavaScript SDK**:
```javascript
import { SigningStargateClient } from "@cosmjs/stargate";

// Connect to blockchain
const client = await SigningStargateClient.connectWithSigner(
  "http://localhost:26657",
  wallet
);

// Query pool
const pool = await client.queryContractSmart(poolAddress, {
  pool_info: {}
});

// Execute swap
const result = await client.execute(
  senderAddress,
  dexAddress,
  { swap: { ... } },
  "auto"
);
```

### Smart Contract Integration

**CosmWasm Contract calling DEX**:
```rust
use cosmwasm_std::{Deps, DepsMut, Env, MessageInfo, Response, to_json_binary};
use paw_dex::msg::ExecuteMsg as DexMsg;

pub fn execute_arbitrage(
    deps: DepsMut,
    env: Env,
    info: MessageInfo,
) -> Result<Response, ContractError> {
    // Call DEX swap from contract
    let swap_msg = DexMsg::Swap {
        pool_id: 1,
        token_in: "token_a".to_string(),
        amount_in: Uint128::new(1000),
        min_amount_out: Uint128::new(900),
    };

    let cosmos_msg = CosmosMsg::Wasm(WasmMsg::Execute {
        contract_addr: DEX_CONTRACT_ADDR.to_string(),
        msg: to_json_binary(&swap_msg)?,
        funds: vec![],
    });

    Ok(Response::new().add_message(cosmos_msg))
}
```

---

## OBSERVABILITY

### Logging

**Structured Logging** (zerolog):
```go
log.Info().
    Str("module", "dex").
    Uint64("pool_id", poolID).
    Str("swap_amount", amount.String()).
    Msg("swap executed")
```

**Log Levels**:
- ERROR: Critical failures
- WARN: Recoverable issues
- INFO: Important events
- DEBUG: Detailed traces (dev only)

### Metrics

**Prometheus Metrics**:
```
# Transaction metrics
paw_tx_total{module="dex",type="swap"} 1234
paw_tx_failed_total{module="dex",error="insufficient_liquidity"} 56

# Pool metrics
paw_dex_pool_count 42
paw_dex_tvl_usd 1500000

# Oracle metrics
paw_oracle_price{asset="paw_usd"} 10.50
paw_oracle_submissions_total{validator="val1"} 5432

# System metrics
paw_block_height 123456
paw_validator_count 150
paw_tx_per_second 847
```

### Tracing

**Distributed Tracing** (Jaeger):
- Trace transactions from client to consensus
- Trace query paths
- Identify performance bottlenecks

### Dashboards

**Grafana Dashboards**:
- Blockchain Health (block time, validator status)
- Transaction Metrics (TPS, success rate, latency)
- DEX Metrics (volume, TVL, pool utilization)
- Oracle Metrics (price updates, validator participation)
- System Metrics (CPU, memory, disk, network)

---

## UPGRADE STRATEGY

### Chain Upgrades

**Coordinated Upgrades** (using gov module):
1. Governance proposal for upgrade
2. Voting period (7 days)
3. If passed, upgrade height set
4. Validators update binary before height
5. Chain halts at height
6. Validators restart with new binary
7. Chain resumes with new logic

**State Migration**:
```go
func (app *App) RegisterUpgradeHandlers() {
    app.UpgradeKeeper.SetUpgradeHandler(
        "v2.0.0",
        func(ctx sdk.Context, plan upgrade.Plan, vm module.VersionMap) (module.VersionMap, error) {
            // Migrate state from v1 to v2
            return app.mm.RunMigrations(ctx, app.configurator, vm)
        },
    )
}
```

### Smart Contract Upgrades

**Immutable Contracts**:
- Contract code cannot be changed after upload
- Use proxy pattern for upgradeable logic
- Admin can migrate to new code if configured

**Governance Upgrades**:
- Only governance can upload new contracts
- Community voting on contract changes
- Transparent upgrade process

---

## DISASTER RECOVERY

### Backup Strategy

**State Backups**:
- Daily full state snapshots
- Hourly incremental backups
- Retention: 30 days rolling

**Validator Key Backups**:
- Encrypted offline storage
- Hardware security modules (HSM)
- Multi-signature recovery

### Recovery Procedures

**Node Failure**:
1. Sentinel node takes over
2. Failed node replaced
3. State synced from peers

**Chain Halt**:
1. Identify root cause
2. Coordinate fix with validators
3. Restart chain at last good block

**State Corruption**:
1. Restore from backup
2. Replay transactions since backup
3. Verify state consistency

---

## COMPLIANCE & GOVERNANCE

### Upgrade Governance

**Proposal Types**:
- Software upgrade
- Parameter change
- Pool creation (if restricted)
- Contract upload (governance-only)

**Voting**:
- Voting power based on stake
- Quorum: 33.4% of total stake
- Threshold: 50% yes votes
- Veto: 33.4% no with veto cancels

### Emergency Procedures

**Circuit Breaker**:
- Governance can pause modules
- Prevents further transactions
- Allows time to fix vulnerabilities

**Security Incidents**:
1. Identify exploit
2. Coordinate emergency proposal
3. Expedited voting (24h)
4. Deploy fix
5. Resume operations

---

## PERFORMANCE TARGETS

### Transaction Throughput

- **Target**: 1000 transactions per second
- **Block Time**: 6 seconds average
- **Finality**: 2 blocks (~12 seconds)

### Query Performance

- **Simple queries**: <100ms (p95)
- **Complex queries**: <500ms (p95)
- **WebSocket latency**: <50ms (p95)

### Scalability

- **Validators**: 150+ active validators supported
- **Nodes**: Unlimited sentinel/sentry nodes
- **Storage**: Pruning keeps ~6 months history

---

## SECURITY AUDIT REQUIREMENTS

### Pre-Production

- [ ] Internal security review
- [ ] Automated security scanning (GoSec, CodeQL)
- [ ] External security audit (minimum 2 firms)
- [ ] All HIGH/CRITICAL findings resolved
- [ ] Penetration testing
- [ ] Formal verification of critical invariants

### Post-Production

- [ ] Bug bounty program active
- [ ] Quarterly security audits
- [ ] Continuous monitoring
- [ ] Incident response plan tested

---

**END OF ARCHITECTURE SPECIFICATION**

This document is the authoritative reference for PAW blockchain architecture. All implementation must conform to these specifications.

**Revisions**:
- v1.0: Initial architecture (deleted - outdated)
- v2.0: Production architecture post-cleanup (current)

**Approval Required**: Lead architect, security team, core developers

**Next Review**: After Phase 1 (IBC Integration) completion
