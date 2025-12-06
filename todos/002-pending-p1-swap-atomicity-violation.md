# DEX Swap Atomicity Violation

---
status: pending
priority: p1
issue_id: "002"
tags: [security, dex, swap, critical, data-integrity]
dependencies: []
---

## Problem Statement

In `x/dex/keeper/swap.go:92-128`, pool reserves are updated and committed to storage BEFORE token transfers complete. If the second transfer fails, pool state becomes permanently inconsistent with actual token balances, violating the constant product invariant (k = x * y).

**Why it matters:** AMM invariant violation means LP funds can be drained or pools become unusable.

## Findings

### Source: data-integrity-guardian & security-sentinel agents

**Location:** `/home/decri/blockchain-projects/paw/x/dex/keeper/swap.go:92-128`

```go
// Lines 92-98: POOL STATE UPDATED FIRST
if isTokenAIn {
    pool.ReserveA = pool.ReserveA.Add(amountInAfterFee)
    pool.ReserveB = pool.ReserveB.Sub(amountOut)
} else {
    pool.ReserveB = pool.ReserveB.Add(amountInAfterFee)
    pool.ReserveA = pool.ReserveA.Sub(amountOut)
}

// Line 101: COMMITTED TO STORAGE
if err := k.SetPool(ctx, pool); err != nil {
    return math.ZeroInt(), err
}

// Lines 110-127: TRANSFERS HAPPEN AFTER COMMIT
if err := k.bankKeeper.SendCoins(sdkCtx, trader, moduleAddr, sdk.NewCoins(coinIn)); err != nil {
    return math.ZeroInt(), fmt.Errorf("failed to transfer input tokens: %w", err)
}
if err := k.bankKeeper.SendCoins(sdkCtx, moduleAddr, trader, sdk.NewCoins(coinOut)); err != nil {
    return math.ZeroInt(), fmt.Errorf("failed to transfer output tokens: %w", err)
}
```

**Data Corruption Scenario:**
1. Pool reserves updated: ReserveA = 1000, ReserveB = 900
2. State committed to storage
3. First transfer succeeds (trader → module)
4. Second transfer fails (module → trader) - e.g., module doesn't have enough
5. Pool state shows ReserveB = 900 but module still holds the tokens
6. Constant product invariant k = 1000 × 900 is violated permanently
7. Future swaps calculate wrong outputs based on incorrect reserves

## Proposed Solutions

### Option A: Transfer First, Then Update State (Recommended)
**Pros:** Simple, follows CEI pattern
**Cons:** None significant
**Effort:** Small
**Risk:** Low

```go
// 1. VALIDATE (already done above)

// 2. TRANSFER INPUT TOKENS FIRST
if err := k.bankKeeper.SendCoins(sdkCtx, trader, moduleAddr, sdk.NewCoins(coinIn)); err != nil {
    return math.ZeroInt(), fmt.Errorf("failed to transfer input tokens: %w", err)
}

// 3. TRANSFER OUTPUT TOKENS SECOND
if err := k.bankKeeper.SendCoins(sdkCtx, moduleAddr, trader, sdk.NewCoins(coinOut)); err != nil {
    // CRITICAL: First transfer succeeded but second failed
    // Need to revert first transfer
    _ = k.bankKeeper.SendCoins(sdkCtx, moduleAddr, trader, sdk.NewCoins(coinIn))
    return math.ZeroInt(), fmt.Errorf("failed to transfer output tokens: %w", err)
}

// 4. ONLY NOW UPDATE POOL STATE (after all transfers succeeded)
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
```

### Option B: Use Cosmos SDK's Transient Store for Rollback
**Pros:** More robust rollback mechanism
**Cons:** More complex implementation
**Effort:** Medium
**Risk:** Medium

## Recommended Action

**Implement Option A** - Reorder to perform ALL transfers before ANY state updates.

## Technical Details

**Affected Files:**
- `x/dex/keeper/swap.go`

**Database Changes:** None required

## Acceptance Criteria

- [ ] All token transfers complete BEFORE pool state is updated
- [ ] If any transfer fails, no state is changed
- [ ] Add test: mock second transfer failure, verify pool state unchanged
- [ ] Add test: verify k = x * y invariant holds after all operations
- [ ] Run existing swap tests to ensure no regressions

## Work Log

| Date | Action | Notes |
|------|--------|-------|
| 2025-12-05 | Created | Identified by data-integrity-guardian and security-sentinel agents |

## Resources

- Related: Issue #001 (escrow state inconsistency) - same pattern
- Uniswap V2: Uses similar CEI pattern
- Cosmos SDK: Bank keeper transfers are atomic within themselves
