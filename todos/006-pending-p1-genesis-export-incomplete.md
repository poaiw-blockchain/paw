# Genesis Export/Import Incomplete - Data Loss Risk

---
status: pending
priority: p1
issue_id: "006"
tags: [data-integrity, dex, genesis, critical]
dependencies: []
---

## Problem Statement

Critical genesis export logic in `x/dex/keeper/genesis.go:59-86, 117-152` is **commented out**. Circuit breaker states and liquidity positions are NOT exported, meaning chain restarts lose this critical state.

**Why it matters:** Chain cannot be properly restored from genesis export - LP positions lost, security features disabled.

## Findings

### Source: architecture-strategist & data-integrity-guardian agents

**Location:** `/home/decri/blockchain-projects/paw/x/dex/keeper/genesis.go:59-86`

```go
// Initialize circuit breaker states
/*
    for _, cbState := range genState.CircuitBreakerStates {
        if err := k.SetCircuitBreakerState(ctx, cbState.PoolId, ...); err != nil {
            return fmt.Errorf("failed to set circuit breaker state for pool %d: %w", cbState.PoolId, err)
        }
    }
*/

// Initialize liquidity positions
/*
    for _, liqPos := range genState.LiquidityPositions {
        // ...
    }
*/
```

**Also in ExportGenesis:**
```go
return &types.GenesisState{
    Params:          params,
    Pools:           pools,
    NextPoolId:      nextPoolID,
    PoolTwapRecords: twapRecords,
    // CircuitBreakerStates:  cbStates,  // COMMENTED OUT
    // LiquidityPositions:    liqPositions,  // COMMENTED OUT
}, nil
```

**Data Loss Scenario:**
1. Chain running with circuit breakers active on Pool #5
2. LPs have $10M in positions across pools
3. Upgrade or emergency requires genesis export/restart
4. Export runs - circuit breakers and LP positions NOT included
5. Chain restarts:
   - All LP positions GONE (users lose all liquidity)
   - Circuit breakers DISABLED (paused pools become active)
   - Security exploits possible during unsafe state

## Proposed Solutions

### Option A: Uncomment and Complete Implementation (Recommended)
**Pros:** Fixes the issue properly
**Cons:** None
**Effort:** Small
**Risk:** Low

```go
// In InitGenesis:
for _, cbState := range genState.CircuitBreakerStates {
    if err := k.SetCircuitBreakerState(ctx, cbState.PoolId, cbState); err != nil {
        return fmt.Errorf("failed to set circuit breaker state for pool %d: %w", cbState.PoolId, err)
    }
}

for _, liqPos := range genState.LiquidityPositions {
    provider, err := sdk.AccAddressFromBech32(liqPos.Provider)
    if err != nil {
        return fmt.Errorf("invalid provider address %s: %w", liqPos.Provider, err)
    }
    if err := k.SetLiquidity(ctx, liqPos.PoolId, provider, liqPos.Shares); err != nil {
        return fmt.Errorf("failed to set liquidity position: %w", err)
    }
}

// In ExportGenesis:
cbStates := []types.CircuitBreakerState{}
// ... iterate and collect
liqPositions := []types.LiquidityPosition{}
// ... iterate and collect

return &types.GenesisState{
    Params:               params,
    Pools:                pools,
    NextPoolId:           nextPoolID,
    PoolTwapRecords:      twapRecords,
    CircuitBreakerStates: cbStates,
    LiquidityPositions:   liqPositions,
}, nil
```

### Option B: Add Migration Path for Existing State
**Pros:** Handles chains already running with missing exports
**Cons:** More complex
**Effort:** Medium
**Risk:** Medium

## Recommended Action

**Implement Option A immediately** - this is a data integrity critical issue.

## Technical Details

**Affected Files:**
- `x/dex/keeper/genesis.go`
- `x/dex/types/genesis.go` (ensure fields exist in GenesisState)

**Database Changes:** None (data already exists, just not exported)

## Acceptance Criteria

- [ ] Circuit breaker states exported in genesis
- [ ] Circuit breaker states imported on InitGenesis
- [ ] Liquidity positions exported in genesis
- [ ] Liquidity positions imported on InitGenesis
- [ ] Add genesis round-trip test: create state → export → import → verify equal
- [ ] Add invariant: sum(LP shares) == pool.TotalShares for each pool

## Work Log

| Date | Action | Notes |
|------|--------|-------|
| 2025-12-05 | Created | Identified by architecture-strategist and data-integrity-guardian agents |

## Resources

- Cosmos SDK Genesis handling best practices
- Related: data-integrity-guardian flagged liquidity conservation invariant missing
