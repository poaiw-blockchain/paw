# Cross-Module Integration Guide

This document describes how the DEX, Oracle, and Compute modules interact within the PAW blockchain.

## Overview

The PAW blockchain features three primary custom modules that work together to provide a complete DeFi ecosystem:

- **DEX Module**: Automated market maker (AMM) for token swaps and liquidity provision
- **Oracle Module**: Decentralized price feed system with Byzantine fault tolerance
- **Compute Module**: Zero-knowledge proof verification for off-chain computations

## Module Dependency Graph

```
┌─────────────────┐
│  Compute Module │
│  (Standalone)   │
└─────────────────┘

┌─────────────────┐         ┌─────────────────┐
│  Oracle Module  │────────>│   DEX Module    │
│  (Price Feeds)  │ Prices  │ (Price Consumer)│
└─────────────────┘         └─────────────────┘
```

**Key Relationships:**
- DEX → Oracle: One-way dependency (DEX consumes Oracle prices)
- Compute: Independent module with IBC packet capabilities
- All modules share IBC infrastructure via `app/ibcutil`

---

## 1. DEX ↔ Oracle Integration

### 1.1 Interface Definition

The DEX module defines an `OracleKeeper` interface for price data consumption:

**Location**: `x/dex/keeper/oracle_integration.go`

```go
type OracleKeeper interface {
    GetPrice(ctx context.Context, denom string) (math.LegacyDec, error)
    GetPriceWithTimestamp(ctx context.Context, denom string) (math.LegacyDec, int64, error)
}
```

### 1.2 Integration Points

#### A. Pool Valuation (USD)

**Function**: `GetPoolValueUSD(ctx, poolID, oracleKeeper)`

Calculates total USD value of a liquidity pool by:
1. Fetching pool reserves (tokenA, tokenB)
2. Querying oracle prices for both tokens
3. Computing: `valueUSD = (reserveA * priceA) + (reserveB * priceB)`

**Use Cases**:
- Pool analytics and rankings
- Total Value Locked (TVL) calculations
- Risk assessment for large positions

**Error Handling**:
- `ErrOraclePrice`: Oracle unavailable or stale price
- Wraps oracle errors with context about which token failed

---

#### B. Price Arbitrage Detection

**Function**: `ValidatePoolPrice(ctx, poolID, oracleKeeper, maxDeviationPercent)`

Compares pool price ratio against oracle prices to detect arbitrage opportunities:

1. **Fetch Oracle Prices** with timestamps
2. **Freshness Check**: Prices must be < 60 seconds old
3. **Calculate Ratios**:
   - Oracle ratio: `priceA / priceB`
   - Pool ratio: `reserveA / reserveB`
4. **Deviation Check**: `|oracleRatio - poolRatio| / oracleRatio`
5. **Threshold**: Fail if deviation > `maxDeviationPercent`

**Security Features**:
- Division-by-zero protection on all ratios
- Timestamp validation prevents stale data
- Configurable deviation thresholds

**Error Codes**:
- `ErrOraclePrice`: Price stale or unavailable
- `ErrPriceDeviation`: Pool price too far from oracle (manipulation alert)
- `ErrInsufficientLiquidity`: Pool has zero reserves

---

#### C. Fair Pool Price Calculation

**Function**: `GetFairPoolPrice(ctx, poolID, oracleKeeper)`

Returns the theoretically correct exchange rate based on oracle data:

```
fairPrice = oraclePrice(tokenA) / oraclePrice(tokenB)
```

**Applications**:
- Setting initial pool ratios
- Rebalancing strategies
- Arbitrage calculators

---

#### D. LP Token Valuation

**Function**: `GetLPTokenValueUSD(ctx, poolID, shares, oracleKeeper)`

Calculates the USD value of LP shares:

1. Get total pool value in USD (using oracle prices)
2. Calculate share percentage: `shares / totalShares`
3. Return: `poolValueUSD * sharePercentage`

**Use Cases**:
- User portfolio tracking
- Collateral valuation in lending protocols
- Tax reporting and accounting

---

#### E. Arbitrage Opportunity Detection

**Function**: `DetectArbitrageOpportunity(ctx, poolID, oracleKeeper, minProfitPercent)`

Identifies profitable arbitrage scenarios:

**Algorithm**:
```
fairPrice = oracle price ratio
poolPrice = pool reserve ratio

if fairPrice > poolPrice:
    profitPercent = (fairPrice - poolPrice) / poolPrice
    direction = "Buy from pool, sell at oracle price"
else if poolPrice > fairPrice:
    profitPercent = (poolPrice - fairPrice) / fairPrice
    direction = "Buy at oracle price, sell to pool"

hasOpportunity = (profitPercent > minProfitPercent)
```

**Returns**: `(hasOpportunity bool, profitPercent Dec, error)`

---

#### F. Swap Validation Against Oracle

**Function**: `ValidateSwapWithOracle(ctx, poolID, tokenIn, tokenOut, amountIn, expectedOut, oracleKeeper)`

Validates swap output is within acceptable range of oracle-based calculation:

1. **Oracle-Based Calculation**:
   ```
   valueIn = amountIn * oraclePrice(tokenIn)
   expectedOut_oracle = valueIn / oraclePrice(tokenOut)
   ```

2. **Tolerance**: ±5% deviation allowed (accounts for fees + slippage)

3. **Validation**:
   ```
   acceptable range = [expectedOut_oracle * 0.95, expectedOut_oracle * 1.05]
   valid = (expectedOut in acceptable range)
   ```

**Purpose**: Prevents MEV attacks and swap manipulation

---

### 1.3 TWAP Integration

Both modules implement Time-Weighted Average Price (TWAP) calculations for flash-loan resistance.

#### Oracle TWAP Implementation

**Location**: `x/oracle/keeper/twap_advanced.go`

**Methods Available**:
1. **Standard TWAP**: Simple time-weighted average
2. **Volume-Weighted TWAP** (VWTWAP): Weights by validator count
3. **Exponential TWAP** (EWMA): Recent prices weighted higher
4. **Trimmed TWAP**: Outlier-resistant (removes top/bottom 10%)
5. **Kalman Filter TWAP**: Adaptive noise filtering

**Key Function**: `CalculateTWAPMultiMethod(ctx, asset)`

Returns a map of all TWAP methods with confidence metrics.

#### DEX TWAP Implementation

**Location**: `x/dex/keeper/twap.go`

**Uniswap V2 Style TWAP**:
- Cumulative price tracking: `cumulativePrice += currentPrice * timeDelta`
- TWAP query: `(cumulativeNow - cumulativePast) / timeDelta`
- Updates on every swap (O(1) computation)

**Key Function**: `UpdateCumulativePriceOnSwap(ctx, poolID, price0, price1)`

**Security Benefits**:
- Manipulation requires sustained attack over entire TWAP window
- No iteration needed (gas-efficient)
- Cannot be manipulated within single block

#### Cross-Module TWAP Usage

DEX pools can optionally use Oracle TWAP for:
- Initial pool pricing
- Circuit breaker triggers
- Arbitrage threshold calculations

**Example Integration**:
```go
// Get Oracle's robust TWAP
oracleTWAP, err := oracleKeeper.CalculateTWAP(ctx, "BTC")

// Compare with DEX pool price
poolPrice := dexKeeper.GetSpotPrice(ctx, poolID)

// Trigger circuit breaker if deviation > 10%
if abs(oracleTWAP - poolPrice) / oracleTWAP > 0.10 {
    dexKeeper.ActivateCircuitBreaker(ctx, poolID)
}
```

---

### 1.4 Dependency Injection

The Oracle keeper is injected into the DEX module at application initialization:

**Location**: `app/app.go` (lines ~250-300)

```go
// Initialize Oracle keeper first
app.OracleKeeper = oraclekeeper.NewKeeper(...)

// Initialize DEX keeper with Oracle keeper reference
app.DexKeeper = dexkeeper.NewKeeper(
    cdc,
    keys[dextypes.StoreKey],
    app.BankKeeper,
    app.IBCKeeper,
    // Oracle keeper injected here for price feeds
)
```

**Important**: DEX does NOT receive the OracleKeeper directly in its constructor. Instead:
- Oracle functions are called on-demand via interface
- DEX remains decoupled from Oracle implementation details
- Functions accept `OracleKeeper` interface as parameter

---

## 2. Compute Module (Standalone)

### 2.1 Module Independence

The Compute module operates independently without direct dependencies on DEX or Oracle:

**Keeper Structure** (`x/compute/keeper/keeper.go`):
```go
type Keeper struct {
    storeKey       storetypes.StoreKey
    cdc            codec.BinaryCodec
    bankKeeper     bankkeeper.Keeper
    accountKeeper  accountkeeper.AccountKeeper
    stakingKeeper  *stakingkeeper.Keeper
    slashingKeeper slashingkeeper.Keeper
    ibcKeeper      *ibckeeper.Keeper
    portKeeper     *portkeeper.Keeper
    // NO Oracle or DEX keepers
}
```

### 2.2 Core Capabilities

#### A. Zero-Knowledge Proof Verification

**Circuit Manager** (`x/compute/keeper/keeper.go:186-280`):

```go
// Lazy initialization to avoid startup cost
func (k *Keeper) GetCircuitManager() *CircuitManager

// Three circuit types supported:
1. ComputeCircuit - Generic computation verification
2. EscrowCircuit - Payment release verification
3. ResultCircuit - Result correctness verification
```

**Key Functions**:
- `VerifyComputeProofWithCircuitManager(ctx, proofData, requestID, ...)`
- `VerifyEscrowProofWithCircuitManager(ctx, proofData, requestID, ...)`
- `VerifyResultProofWithCircuitManager(ctx, proofData, requestID, ...)`

**Security Features**:
- Proof size limits (prevent DoS)
- Deposit-based verification (refunded on valid proof, slashed on invalid)
- Replay attack prevention via nonce tracking
- Proof expiration timestamps

---

#### B. Provider Reputation System

**Location**: `x/compute/keeper/reputation.go`

Tracks compute provider performance without external module dependencies:

- Stake-weighted reputation scoring
- Automatic slashing for failed verifications
- Provider capacity management
- Geographic diversity (optional, not enforced)

---

#### C. Request Lifecycle Management

**Location**: `x/compute/keeper/request.go`

Full lifecycle without cross-module calls:

1. **Request Creation**: Escrow locked in bank module
2. **Provider Assignment**: Based on reputation + capacity
3. **Result Submission**: ZK proof verification
4. **Escrow Release**: Automatic on valid proof
5. **Dispute Resolution**: On-chain governance voting

---

### 2.3 IBC Capabilities

All three modules share IBC authorization infrastructure:

**Shared Interface** (`app/ibcutil/channel_authorization.go`):

```go
type ChannelStore interface {
    GetAuthorizedChannels(ctx) ([]AuthorizedChannel, error)
    SetAuthorizedChannels(ctx, []AuthorizedChannel) error
}
```

**Implementation** (each module):
- `keeper.GetAuthorizedChannels(ctx)` - Fetch from module params
- `keeper.SetAuthorizedChannels(ctx, channels)` - Update via governance
- `keeper.IsAuthorizedChannel(ctx, portID, channelID)` - Packet validation

**Nonce Tracking** (replay prevention):
- DEX: `ErrInvalidNonce` (code 31)
- Oracle: `ErrInvalidNonce` (code 50)
- Compute: `ErrInvalidNonce` (code 84)

All modules use `NonceRetentionBlocks` parameter for cleanup.

---

## 3. Common Infrastructure

### 3.1 IBC Authorization

**Location**: `app/ibcutil/channel_authorization.go`

Shared helper functions prevent code duplication:

```go
// Check if channel is authorized
func IsAuthorizedChannel(ctx, store, portID, channelID) bool

// Add channel to allowlist
func AuthorizeChannel(ctx, store, portID, channelID) error

// Validate and set entire allowlist
func SetAuthorizedChannelsWithValidation(ctx, store, channels) error
```

**Benefits**:
- Consistent authorization logic across modules
- Governance-approved channel lists
- Protection against unauthorized relayers
- Easy multi-module channel management

---

### 3.2 Circuit Breakers

All modules implement emergency pause functionality:

**Error Codes**:
- `ErrCircuitBreakerTriggered` - Module operations paused
- `ErrCircuitBreakerAlreadyOpen` - Already activated
- `ErrCircuitBreakerAlreadyClosed` - Already deactivated

**Triggers** (module-specific):
- **DEX**: Large price swings, unusual volume, flash loan detection
- **Oracle**: Insufficient validator participation, data poisoning, Sybil attacks
- **Compute**: Provider mass slashing, verification failures spike

**Recovery**:
- Automatic after cooldown period (configurable per module)
- Manual via governance proposal
- Logged via telemetry for monitoring

---

### 3.3 Metrics and Telemetry

**Location** (per module):
- `x/dex/keeper/metrics.go`
- `x/oracle/keeper/metrics.go`
- `x/compute/keeper/metrics.go`

**Shared Patterns**:
- Prometheus-compatible metrics
- Latency histograms for operations
- Counter metrics for events
- Gauge metrics for state values

**Cross-Module Monitoring**:
```go
// DEX metrics include oracle price deviation
dexMetrics.OraclePriceDeviation.Observe(deviation)

// Oracle metrics track DEX query patterns
oracleMetrics.PriceQueries.With("consumer", "dex").Inc()
```

---

## 4. Security Considerations

### 4.1 Oracle Price Manipulation

**DEX Defenses**:
1. **Freshness Checks**: Reject prices older than 60 seconds
2. **Deviation Limits**: Circuit breaker if oracle diverges > configured threshold
3. **Fallback**: DEX can operate without oracle (loses validation features)
4. **TWAP**: Use Oracle's TWAP instead of spot price for critical operations

**Oracle Defenses**:
1. **Validator Diversity**: Minimum 67% voting power required
2. **Outlier Filtering**: Statistical outlier removal before aggregation
3. **Geographic Distribution**: Prevent regional price manipulation
4. **Flash Loan Detection**: Multi-block TWAP prevents single-block attacks

---

### 4.2 Cross-Module Attack Vectors

#### Vector: Oracle Poisoning → DEX Manipulation

**Attack**: Submit false oracle prices to trigger DEX arbitrage or circuit breaker

**Mitigations**:
1. **Stake-Weighted Voting**: Requires controlling >33% stake to manipulate
2. **Slashing**: Invalid prices result in 1% stake slashing
3. **Multi-Method TWAP**: 5 different TWAP methods must agree
4. **DEX Validation**: Independent TWAP acts as sanity check
5. **Circuit Breaker**: Automatic pause on suspicious oracle behavior

#### Vector: Compute Module → Resource Exhaustion

**Attack**: Submit computationally expensive ZK proofs to DoS validators

**Mitigations**:
1. **Proof Size Limits**: Max proof size defined in circuit params
2. **Deposit Requirements**: Expensive proofs require larger deposits
3. **Rate Limiting**: Per-account request quotas
4. **Lazy Initialization**: Circuits compiled on first use, cached
5. **Timeout Enforcement**: Verification must complete within timeout

---

### 4.3 IBC Security

All modules implement identical IBC security patterns:

1. **Channel Authorization**: Governance-approved channel list
2. **Nonce Tracking**: Prevent replay attacks
3. **Packet Validation**: Strict message format checks
4. **Timeout Handling**: Refund on timeout, no state corruption
5. **Acknowledgement Validation**: Verify counterparty responses

**Shared Error Codes**:
- `ErrUnauthorizedChannel` (DEX: 92, Oracle: 60, Compute: 85)
- `ErrInvalidNonce` (DEX: 31, Oracle: 50, Compute: 84)
- `ErrInvalidAck` (DEX: 91, Oracle: 90, Compute: 83)

---

## 5. Usage Examples

### Example 1: Safe Swap with Oracle Validation

```go
// User submits swap
tokenIn := "upaw"
tokenOut := "ubtc"
amountIn := sdk.NewInt(1000000)

// DEX calculates expected output
expectedOut := dexKeeper.CalculateSwapOutput(ctx, poolID, amountIn)

// Validate against oracle (optional but recommended)
err := dexKeeper.ValidateSwapWithOracle(
    ctx, poolID, tokenIn, tokenOut,
    amountIn, expectedOut, oracleKeeper,
)
if err != nil {
    // Oracle validation failed - potential manipulation
    // Option 1: Reject swap
    return err

    // Option 2: Reduce slippage tolerance
    expectedOut = expectedOut.Mul(0.95)

    // Option 3: Use TWAP instead of spot price
    oracleTWAP := oracleKeeper.CalculateTWAP(ctx, "BTC")
    // ... recalculate
}

// Execute swap if validation passes
err = dexKeeper.ExecuteSwap(ctx, poolID, tokenIn, tokenOut, amountIn, expectedOut)
```

---

### Example 2: Pool Analytics with Oracle Prices

```go
// Get all pools
pools, _ := dexKeeper.GetAllPools(ctx)

type PoolAnalytics struct {
    PoolID      uint64
    TVL         sdk.Dec  // Total Value Locked (USD)
    Arbitrage   bool
    Profit      sdk.Dec
}

analytics := []PoolAnalytics{}

for _, pool := range pools {
    // Calculate USD value
    tvl, _ := dexKeeper.GetPoolValueUSD(ctx, pool.Id, oracleKeeper)

    // Check for arbitrage
    hasArb, profit, _ := dexKeeper.DetectArbitrageOpportunity(
        ctx, pool.Id, oracleKeeper,
        sdk.NewDecWithPrec(1, 2), // 1% minimum profit
    )

    analytics = append(analytics, PoolAnalytics{
        PoolID:    pool.Id,
        TVL:       tvl,
        Arbitrage: hasArb,
        Profit:    profit,
    })
}

// Sort by TVL
sort.Slice(analytics, func(i, j int) bool {
    return analytics[i].TVL.GT(analytics[j].TVL)
})
```

---

### Example 3: Compute Request with Escrow

```go
// Submit compute request (fully independent)
requestMsg := computetypes.MsgSubmitRequest{
    Requester:   userAddress,
    Provider:    providerAddress,
    Image:       "compute/ml-model:v1.0",
    InputData:   []byte("..."),
    Escrow:      sdk.NewCoins(sdk.NewCoin("upaw", sdk.NewInt(100000))),
    Timeout:     3600, // 1 hour
}

// Compute module handles everything internally
requestID, err := computeKeeper.SubmitRequest(ctx, requestMsg)

// Provider submits result with ZK proof
resultMsg := computetypes.MsgSubmitResult{
    RequestId:  requestID,
    Provider:   providerAddress,
    ResultData: []byte("..."),
    Proof:      zkProof,
}

// Automatic verification and escrow release
err = computeKeeper.SubmitResult(ctx, resultMsg)
// If proof valid: escrow released to provider
// If proof invalid: provider slashed, escrow refunded
```

---

## 6. Future Integration Opportunities

### 6.1 Compute-Enhanced Oracle

**Concept**: Use Compute module to verify oracle price sources

```go
// Oracle validators submit price + ZK proof of source authenticity
type ValidatedPriceSubmission struct {
    Price         sdk.Dec
    Proof         []byte  // ZK proof of API signature verification
    SourceHash    []byte  // Hash of price source (e.g., Coinbase API)
}

// Compute module verifies proof before oracle accepts submission
computeKeeper.VerifyPriceSourceProof(ctx, proof, sourceHash)
```

**Benefits**:
- Prevents validators from submitting fabricated prices
- Cryptographic proof of legitimate data source
- Enhanced censorship resistance

---

### 6.2 Oracle-Priced Compute Market

**Concept**: Use Oracle to price compute resources dynamically

```go
// Query oracle for current cloud compute costs
cloudCost := oracleKeeper.GetPrice(ctx, "cloud-compute-usd-per-hour")

// Adjust compute provider prices based on market
basePrice := sdk.NewDec(1000) // 1000 upaw base
marketAdjustedPrice := basePrice.Mul(cloudCost)

// Providers can't overcharge relative to market
if providerPrice.GT(marketAdjustedPrice.Mul(sdk.NewDecWithPrec(110, 2))) {
    return errors.New("price exceeds market rate by >10%")
}
```

**Benefits**:
- Prevents compute provider price gouging
- Aligns on-chain compute costs with off-chain market
- Creates arbitrage opportunities for efficient providers

---

### 6.3 DEX with Verifiable Execution

**Concept**: Use Compute module to verify complex DEX operations

```go
// Large multi-hop swap with ZK proof of optimal routing
type VerifiedSwapRequest struct {
    Path         []uint64  // Pool IDs in optimal route
    AmountIn     sdk.Int
    MinAmountOut sdk.Int
    Proof        []byte    // ZK proof that this is optimal route
}

// Compute module verifies routing algorithm execution
isOptimal := computeKeeper.VerifyRoutingProof(ctx, proof, path)

// DEX executes swap only if proof valid
if isOptimal {
    dexKeeper.ExecuteMultiHopSwap(ctx, path, amountIn, minAmountOut)
}
```

**Benefits**:
- Prevents MEV from suboptimal routing
- Trustless verification of complex calculations
- Enhanced capital efficiency

---

## 7. Module Communication Summary

| Source Module | Target Module | Communication Method | Data Flow |
|---------------|---------------|---------------------|-----------|
| DEX           | Oracle        | Interface calls     | Price queries (read-only) |
| Oracle        | DEX           | None (passive)      | Prices available for query |
| Compute       | DEX/Oracle    | None                | Independent operation |
| DEX           | Compute       | None                | No direct integration yet |
| Oracle        | Compute       | None                | No direct integration yet |
| All           | IBC           | Packet relay        | Cross-chain messages |
| All           | Bank          | Send/Escrow         | Token transfers |
| All           | Staking       | Validator queries   | Voting power, bonded status |
| All           | Slashing      | Slash operations    | Penalty enforcement |

---

## 8. Developer Quick Reference

### Adding Oracle Integration to New Module

1. **Define Interface**:
   ```go
   type OracleKeeper interface {
       GetPrice(ctx, denom) (sdk.Dec, error)
   }
   ```

2. **Add to Keeper Constructor** (optional, or pass as parameter):
   ```go
   type Keeper struct {
       oracleKeeper OracleKeeper  // If persistent
   }
   ```

3. **Use in Functions**:
   ```go
   func (k Keeper) DoSomething(ctx, oracle OracleKeeper) error {
       price, err := oracle.GetPrice(ctx, "upaw")
       // ... use price
   }
   ```

4. **Inject in app.go**:
   ```go
   app.NewModuleKeeper = NewKeeper(
       app.OracleKeeper,  // Pass oracle keeper
   )
   ```

---

### Adding IBC to New Module

1. **Implement ChannelStore**:
   ```go
   func (k Keeper) GetAuthorizedChannels(ctx) ([]ibcutil.AuthorizedChannel, error)
   func (k Keeper) SetAuthorizedChannels(ctx, channels []ibcutil.AuthorizedChannel) error
   ```

2. **Use Shared Helpers**:
   ```go
   import "github.com/paw-chain/paw/app/ibcutil"

   if !ibcutil.IsAuthorizedChannel(ctx, k, portID, channelID) {
       return ErrUnauthorizedChannel
   }
   ```

3. **Add Nonce Tracking**:
   ```go
   // In packet send
   nonce := k.GetNextNonce(ctx, channelID)
   packet.Nonce = nonce
   k.IncrementNonce(ctx, channelID)

   // In packet receive
   if !k.ValidateNonce(ctx, packet.SourceChannel, packet.Nonce) {
       return ErrInvalidNonce
   }
   ```

---

## Conclusion

The PAW blockchain's three custom modules are designed with clear separation of concerns:

- **DEX**: Consumes oracle prices for validation, operates independently otherwise
- **Oracle**: Provides price feeds, unaware of consumers
- **Compute**: Fully independent, focused on ZK verification

This architecture provides:
- **Modularity**: Modules can be upgraded independently
- **Security**: Clear trust boundaries, no circular dependencies
- **Flexibility**: Easy to add new integrations without core changes
- **Auditability**: Isolated codebases reduce attack surface

Future enhancements will focus on optional integrations that enhance functionality without creating hard dependencies.
