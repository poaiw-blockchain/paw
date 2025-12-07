# Division by Zero Risk in Liquidity Calculations

---
status: pending
priority: p2
issue_id: "022"
tags: [security, dex, math, high]
dependencies: []
---

## Problem Statement

The `AddLiquidity` function calculates optimal amounts with division operations that lack explicit zero checks before division. While `ValidatePoolState` checks reserves are positive, there's a race condition window where reserves could be zero.

**Why it matters:** Division by zero causes panic, halting the chain.

## Findings

### Source: security-sentinel agent

**Location:** `/home/decri/blockchain-projects/paw/x/dex/keeper/liquidity.go:60-61, 76-77`

**Code:**
```go
// Line 60-61
optimalAmountB := amountA.Mul(pool.ReserveB).Quo(pool.ReserveA)
optimalAmountA := amountB.Mul(pool.ReserveA).Quo(pool.ReserveB)

// Line 76-77
sharesA := finalAmountA.Mul(pool.TotalShares).Quo(pool.ReserveA)
sharesB := finalAmountB.Mul(pool.TotalShares).Quo(pool.ReserveB)
```

**Attack Scenario:**
1. Pool created with minimal liquidity
2. Attacker front-runs next `AddLiquidity` with reserve drain
3. Division by zero causes panic
4. Chain halts

## Proposed Solutions

### Option A: Explicit Zero Checks (Recommended)
**Pros:** Simple, defensive
**Cons:** Slight overhead
**Effort:** Small
**Risk:** Low

```go
func (k Keeper) AddLiquidity(...) (math.Int, error) {
    // ... existing code ...

    // EXPLICIT ZERO CHECKS BEFORE DIVISION
    if pool.ReserveA.IsZero() || pool.ReserveB.IsZero() {
        return math.ZeroInt(), types.ErrInsufficientLiquidity.Wrap("pool reserves cannot be zero")
    }

    if pool.TotalShares.IsZero() {
        return math.ZeroInt(), types.ErrInvalidPoolState.Wrap("pool shares cannot be zero")
    }

    // NOW safe to divide
    optimalAmountB := amountA.Mul(pool.ReserveB).Quo(pool.ReserveA)
    // ...
}
```

## Recommended Action

**Implement Option A** - add explicit zero checks before all division operations.

## Technical Details

**Affected Files:**
- `x/dex/keeper/liquidity.go`
- `x/dex/keeper/swap.go` (check for similar issues)

**Database Changes:** None

## Acceptance Criteria

- [ ] Zero checks added before every Quo() operation in liquidity code
- [ ] Zero checks added before every Quo() operation in swap code
- [ ] Test: attempt operation with zero reserves, verify graceful error
- [ ] No panic possible from division operations

## Work Log

| Date | Action | Notes |
|------|--------|-------|
| 2025-12-07 | Created | Identified by security-sentinel agent |

## Resources

- math.Int Quo() will panic on division by zero
- Defense in depth principle
