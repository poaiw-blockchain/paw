# Flash Loan Protection Insufficient

---
status: pending
priority: p2
issue_id: "014"
tags: [security, dex, flash-loan, mev]
dependencies: []
---

## Problem Statement

Flash loan protection in `x/dex/keeper/pool_secure.go:143-146` only records block number but doesn't prevent same-block add→swap→remove attacks.

**Why it matters:** Flash loan attacks, MEV exploitation, price manipulation.

## Findings

### Source: security-sentinel agent

**Location:** `/home/decri/blockchain-projects/paw/x/dex/keeper/pool_secure.go:143-146`

```go
// Line 143-146: Only records block, doesn't prevent same-block actions
if err := k.SetLastLiquidityActionBlock(ctx, poolID, creator); err != nil {
    return nil, err
}
```

**Attack Scenario (within single block):**
1. Add huge liquidity to pool (becomes LP)
2. Execute large swap at favorable price (moves price)
3. Arbitrage the price difference on another pool/chain
4. Remove liquidity (in same block!)
5. Profit from price manipulation

**Current Protection:**
- Records block number of liquidity action
- Checks block on removal
- BUT: if add and remove in same block, check passes

## Proposed Solutions

### Option A: Multi-Block Lock Period (Recommended)
**Pros:** Simple, effective
**Cons:** Legitimate LPs must wait
**Effort:** Small
**Risk:** Low

```go
const MinLiquidityLockBlocks = 10 // Configurable via params

func (k Keeper) RemoveLiquidity(ctx context.Context, poolID uint64, provider sdk.AccAddress, shares math.Int) error {
    sdkCtx := sdk.UnwrapSDKContext(ctx)
    currentBlock := sdkCtx.BlockHeight()

    lastActionBlock, err := k.GetLastLiquidityActionBlock(ctx, poolID, provider)
    if err != nil {
        return err
    }

    // ENFORCE MINIMUM LOCK PERIOD
    if currentBlock - lastActionBlock < MinLiquidityLockBlocks {
        return types.ErrFlashLoanProtection.Wrapf(
            "must wait %d blocks, only %d elapsed",
            MinLiquidityLockBlocks, currentBlock - lastActionBlock)
    }

    // ... proceed with removal
}
```

### Option B: Timelock with Withdrawal Queue
**Pros:** More sophisticated, supports large withdrawals
**Cons:** Complex UX
**Effort:** Large
**Risk:** Medium

### Option C: Early Withdrawal Fee
**Pros:** Discourages flash loans economically
**Cons:** Legitimate LPs pay fee
**Effort:** Small
**Risk:** Low

## Recommended Action

**Implement Option A** with governance-adjustable lock period.

## Technical Details

**Affected Files:**
- `x/dex/keeper/pool_secure.go`
- `x/dex/types/params.go` (add MinLiquidityLockBlocks)

**Database Changes:** None (already tracks last action block)

## Acceptance Criteria

- [ ] Minimum 10 block lock period between add and remove
- [ ] Lock period configurable via governance
- [ ] Test: add liquidity block N, remove block N, verify rejection
- [ ] Test: add liquidity block N, remove block N+10, verify success
- [ ] Existing tests updated for new behavior

## Work Log

| Date | Action | Notes |
|------|--------|-------|
| 2025-12-05 | Created | Identified by security-sentinel agent |

## Resources

- Related: ROADMAP HIGH-3 (Flash Loan Protection) - marked completed but may be incomplete
- Uniswap V2: No flash loan protection (by design)
- Balancer: Implements flash loan fees
