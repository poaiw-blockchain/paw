# DEX Swap Reentrancy Risk via Bank Keeper Hooks

---
status: pending
priority: p1
issue_id: "007"
tags: [security, dex, reentrancy, critical]
dependencies: ["002"]
---

## Problem Statement

While checks-effects-interactions pattern is attempted in `x/dex/keeper/swap.go:106-127`, pool state is updated AFTER token transfer in. If `bankKeeper.SendCoins` has hooks that call back into DEX, reentrancy is possible.

**Why it matters:** Pool drainage, LP fund loss, AMM invariant violation.

## Findings

### Source: security-sentinel agent

**Location:** `/home/decri/blockchain-projects/paw/x/dex/keeper/swap.go:106-127`

```go
// Line 92-103: Pool state updated
if isTokenAIn {
    pool.ReserveA = pool.ReserveA.Add(amountInAfterFee)
    pool.ReserveB = pool.ReserveB.Sub(amountOut)
} else {
    pool.ReserveB = pool.ReserveB.Add(amountInAfterFee)
    pool.ReserveA = pool.ReserveA.Sub(amountOut)
}

if err := k.SetPool(ctx, pool); err != nil {
    return math.ZeroInt(), err
}

// Line 106-127: External calls AFTER pool update
if err := k.bankKeeper.SendCoins(sdkCtx, trader, moduleAddr, sdk.NewCoins(coinIn)); err != nil {
    return math.ZeroInt(), fmt.Errorf("failed to transfer input tokens: %w", err)
}
```

**Attack Scenario (if bank hooks exist):**
1. Attacker initiates swap: 100 TokenA → TokenB
2. Pool state updated: ReserveA += 100, ReserveB -= X
3. `bankKeeper.SendCoins` called for input transfer
4. Bank hook triggers (e.g., token callback, module hook)
5. Hook calls back into DEX swap with manipulated state
6. Second swap executes on already-modified reserves
7. Attacker extracts more than entitled

**Note:** Reentrancy guards exist (`x/dex/keeper/security.go:76-98`) but need verification they're applied to all entry points.

## Proposed Solutions

### Option A: Verify Reentrancy Guard Coverage (Recommended)
**Pros:** Guards already implemented, just need verification
**Cons:** None
**Effort:** Small
**Risk:** Low

```go
// Verify in swap.go that guard is applied:
func (k Keeper) ExecuteSwap(ctx context.Context, poolID uint64, ...) (math.Int, error) {
    return k.WithReentrancyGuard(ctx, poolID, "swap", func() error {
        // ALL swap logic inside guard
    })
}
```

### Option B: Add Explicit In-Progress Flag
**Pros:** Defense in depth
**Cons:** Slightly more gas
**Effort:** Small
**Risk:** Low

### Option C: Reorder to CEI Pattern
**Pros:** Eliminates the vulnerability entirely
**Cons:** Requires careful refactoring
**Effort:** Medium
**Risk:** Low

## Recommended Action

1. **Verify existing reentrancy guard coverage** (Option A)
2. **Reorder operations** (related to issue #002)
3. **Audit all DEX entry points** for guard application

## Technical Details

**Affected Files:**
- `x/dex/keeper/swap.go`
- `x/dex/keeper/security.go`
- `x/dex/keeper/msg_server.go` (verify guard on message handlers)

**Database Changes:** None

## Acceptance Criteria

- [ ] Verify `WithReentrancyGuard` wraps ExecuteSwap
- [ ] Verify guard covers AddLiquidity, RemoveLiquidity
- [ ] Test: attempt recursive call during swap, verify blocked
- [ ] Audit bank module for hook mechanisms
- [ ] Document which operations are guard-protected

## Work Log

| Date | Action | Notes |
|------|--------|-------|
| 2025-12-05 | Created | Identified by security-sentinel agent |

## Resources

- x/dex/keeper/security.go - existing reentrancy guard implementation
- Uniswap V2 reentrancy protection pattern
- OpenZeppelin ReentrancyGuard
