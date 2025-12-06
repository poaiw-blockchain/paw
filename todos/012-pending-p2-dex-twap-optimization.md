# DEX TWAP Update Performance - O(n) Every Block

---
status: pending
priority: p2
issue_id: "012"
tags: [performance, dex, abci, scalability]
dependencies: []
---

## Problem Statement

BeginBlocker iterates ALL pools to update TWAPs in `x/dex/keeper/abci.go:224-287`. With 1000 pools, this consumes ~30M gas per block.

**Why it matters:** BeginBlocker may exceed block gas limit as pool count grows.

## Findings

### Source: performance-oracle agent

**Location:** `/home/decri/blockchain-projects/paw/x/dex/keeper/abci.go:224-287`

```go
// Line 230: Unbounded iteration every block
return k.IteratePools(ctx, func(pool types.Pool) bool {
    // ... TWAP calculation for EVERY pool
    if err := k.SetPoolTWAP(ctx, *record); err != nil {
        sdkCtx.Logger().Error("failed to persist TWAP", "pool_id", pool.Id, "error", err)
    }
    return false // Never stops early
})
```

**Performance Impact:**
- 1000 pools × ~30,000 gas/pool = 30M gas per block
- Growing linearly with pool count
- Competes with user transactions for block space

## Proposed Solutions

### Option A: Lazy TWAP Updates (Recommended)
**Pros:** Only update pools that are queried
**Cons:** First query after gap is slower
**Effort:** Medium
**Risk:** Low

```go
func (k Keeper) GetPoolTWAP(ctx context.Context, poolID uint64) (types.TWAPRecord, error) {
    record, err := k.getStoredTWAP(ctx, poolID)
    if err != nil {
        return types.TWAPRecord{}, err
    }

    // Check if stale (not updated recently)
    currentHeight := sdk.UnwrapSDKContext(ctx).BlockHeight()
    if record.LastUpdateHeight < currentHeight - 1 {
        // Update TWAP lazily
        record = k.calculateTWAPUpdate(ctx, poolID, record)
        k.SetPoolTWAP(ctx, record)
    }

    return record, nil
}

// BeginBlocker only updates active pools
func (k Keeper) UpdateTWAPs(ctx context.Context) error {
    // Only update pools that had swaps in last block
    activePoolIDs := k.GetPoolsWithRecentActivity(ctx)
    for _, poolID := range activePoolIDs {
        k.updatePoolTWAP(ctx, poolID)
    }
    return nil
}
```

### Option B: Batched Updates
**Pros:** Spreads load across blocks
**Cons:** TWAP slightly stale for some pools
**Effort:** Small
**Risk:** Low

### Option C: Remove BeginBlocker TWAP
**Pros:** Simplest
**Cons:** All TWAP calculations on-demand
**Effort:** Small
**Risk:** Medium

## Recommended Action

**Implement Option A** - lazy TWAP with activity-based updates.

## Technical Details

**Affected Files:**
- `x/dex/keeper/abci.go`
- `x/dex/keeper/twap.go` (add lazy calculation)

**Database Changes:** Add pool activity tracking

## Acceptance Criteria

- [ ] BeginBlocker only updates active pools
- [ ] Lazy TWAP calculation on query
- [ ] Benchmark: 1000 pools with 10 active = ~300k gas (not 30M)
- [ ] TWAP accuracy maintained for active pools
- [ ] Tests verify TWAP correctness with lazy updates

## Work Log

| Date | Action | Notes |
|------|--------|-------|
| 2025-12-05 | Created | Identified by performance-oracle agent |

## Resources

- Uniswap V3 TWAP approach (per-observation storage)
- Related: Limit order matching also has O(n) issues
