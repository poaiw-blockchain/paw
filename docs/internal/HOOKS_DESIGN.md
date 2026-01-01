# Module Hooks Design (SEC-3.12)

PAW modules use hooks for cross-module notifications. See `x/*/types/hooks.go`.

## Hook Interfaces

| Module | Interface | Callbacks |
|--------|-----------|-----------|
| Oracle | `OracleHooks` | AfterPriceAggregated, AfterPriceSubmitted, OnCircuitBreakerTriggered |
| DEX | `DexHooks` | AfterSwap, AfterPoolCreated, AfterLiquidityChanged, OnCircuitBreakerTriggered |
| Compute | `ComputeHooks` | AfterJobCompleted/Failed, AfterProviderRegistered/Slashed, OnCircuitBreakerTriggered |

## Execution Order (app/app.go)

**BeginBlocker**: mint -> distribution -> slashing -> evidence -> staking -> **oracle** -> **dex** -> **compute**

**EndBlocker**: crisis -> gov -> staking -> feegrant -> **oracle** -> **dex** -> **compute**

**Genesis Init**: **oracle** -> **dex** -> **compute**

## Hook Registration

Via `SetHooks()` on keepers. MultiHooks execute sequentially; first error halts chain.

## Circular Dependency Prevention

**Rule**: Hooks must only call downstream modules.

```
Oracle (upstream) --> DEX --> Compute (downstream)
```

**Forbidden**: DEX hook calling Oracle setter, Compute hook modifying DEX, any self-calls.

## Circuit Breaker

`CircuitBreakerCoordinator` (app/circuit_breaker_coordinator.go) propagates events across modules.

## Best Practices

1. Keep hooks lightweight
2. Return errors early
3. Test hook interactions in integration tests
