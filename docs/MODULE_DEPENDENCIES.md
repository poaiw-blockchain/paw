# PAW Module Dependencies

## Module Overview

| Module | Description | IBC Port |
|--------|-------------|----------|
| **compute** | Distributed compute job management with ZK proof verification | `compute` |
| **dex** | Decentralized exchange with AMM pools and multi-hop routing | `dex` |
| **oracle** | Price oracle with validator voting and geographic diversity | `oracle` |
| **shared** | Common utilities: nonce management, IBC packet handling, ABCI errors |

## Dependency Graph

```
                    ┌─────────────────────────────────────────┐
                    │           Cosmos SDK Core               │
                    │  (auth, bank, staking, slashing, gov)   │
                    └─────────────────────────────────────────┘
                              │
        ┌─────────────────────┼─────────────────────┐
        ▼                     ▼                     ▼
  ┌──────────┐         ┌───────────┐         ┌───────────┐
  │  ORACLE  │◄────────│    DEX    │         │  COMPUTE  │
  │          │ prices  │           │         │           │
  └──────────┘         └───────────┘         └───────────┘
        │                    │                     │
        └────────────────────┼─────────────────────┘
                             ▼
                    ┌─────────────────┐
                    │   IBC Keeper    │
                    │  (cross-chain)  │
                    └─────────────────┘
```

## Compile-Time Dependencies (expected_keepers.go)

### DEX Module
```go
// Required keeper interfaces
AccountKeeper: GetAccount(ctx, addr)
BankKeeper:    SpendableCoins, SendCoinsFromAccountToModule, SendCoinsFromModuleToAccount
```

### Oracle Module
```go
AccountKeeper: GetAccount(ctx, addr)
BankKeeper:    SpendableCoins, SendCoinsFromAccountToModule, SendCoinsFromModuleToAccount
StakingKeeper: GetAllValidators(ctx)  // For validator-weighted voting
```

### Compute Module
```go
AccountKeeper: GetAccount(ctx, addr)
BankKeeper:    GetBalance, SpendableCoins, SendCoins, SendCoinsFromAccountToModule,
               SendCoinsFromModuleToAccount, MintCoins, BurnCoins
```

## Runtime Dependencies (app.go Wiring)

### Keeper Initialization Order
1. **Core SDK**: Params → Account → Bank → Staking → Slashing
2. **IBC**: Capability → IBCKeeper → TransferKeeper
3. **PAW Custom**: Oracle → DEX → Compute (Oracle prices first)

### Keeper Constructor Dependencies

| Module | Constructor Dependencies |
|--------|-------------------------|
| DEX | `bankKeeper`, `ibcKeeper`, `portKeeper`, `scopedKeeper` |
| Compute | `bankKeeper`, `accountKeeper`, `stakingKeeper`, `slashingKeeper`, `ibcKeeper`, `portKeeper`, `scopedKeeper` |
| Oracle | `bankKeeper`, `stakingKeeper`, `slashingKeeper`, `ibcKeeper`, `portKeeper`, `scopedKeeper` |

### Genesis Init Order (app.go:520-544)
```
capability → auth → bank → distr → staking → slashing → gov → mint →
crisis → genutil → evidence → feegrant → params → upgrade → vesting →
consensus → ibc → transfer → oracle → dex → compute
```

Oracle initializes before DEX so prices are available for DEX operations.

### Begin/End Block Order
- **BeginBlock**: `oracle → dex → compute` (Oracle provides fresh prices first)
- **EndBlock**: `oracle → dex → compute` (Oracle finalizes, then DEX cleanup)

## Cross-Module Hooks (ARCH-2)

### OracleHooks → DEX
```go
AfterPriceAggregated(asset, price, blockHeight)  // DEX can react to price updates
OnCircuitBreakerTriggered(reason)                // DEX pauses price-sensitive ops
```

### DexHooks → External
```go
AfterSwap(poolID, sender, tokenIn, tokenOut, amountIn, amountOut)
AfterPoolCreated(poolID, tokenA, tokenB, creator)
AfterLiquidityChanged(poolID, provider, deltaA, deltaB, isAdd)
OnCircuitBreakerTriggered(reason)
```

### ComputeHooks → External
```go
AfterJobCompleted(requestID, provider, result)
AfterJobFailed(requestID, reason)
AfterProviderRegistered(provider, stake)
AfterProviderSlashed(provider, slashAmount, reason)
OnCircuitBreakerTriggered(reason)
```

## IBC Dependencies

All three PAW modules implement `IBCModule` interface:
- **Port binding**: Each module binds its own port via `scopedKeeper`
- **Shared utilities**: `x/shared/ibc` provides `ChannelOperation` tracking
- **Authorized channels**: Configured per-module in params

### IBC Router Registration (app.go:431-447)
```go
ibcRouter.AddRoute(ibctransfertypes.ModuleName, transferModule)
ibcRouter.AddRoute(computetypes.PortID, computeIBCModule)
ibcRouter.AddRoute(dextypes.PortID, dexIBCModule)
ibcRouter.AddRoute(oracletypes.PortID, oracleIBCModule)
```

## Shared Utilities

`x/shared/` provides module-agnostic components:

| Package | Purpose |
|---------|---------|
| `nonce` | Replay attack prevention for IBC packets |
| `ibc` | `ChannelOperation` struct for pending packet tracking |
| `abci` | Standardized error handling |

## Module Account Permissions (app.go:1325-1338)

| Module | Permissions |
|--------|-------------|
| dex | Minter, Burner (LP tokens) |
| compute | Minter, Burner (escrow management) |
| oracle | None (read-only price feeds) |

## AnteHandler Integration

Custom AnteHandler (app.go:609-625) receives:
- `ComputeKeeper` - Job validation
- `DEXKeeper` - Swap validation
- `OracleKeeper` - Price data validation
