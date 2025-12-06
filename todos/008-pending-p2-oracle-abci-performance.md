# Oracle ABCI O(n×m) Performance Bottleneck

---
status: pending
priority: p2
issue_id: "008"
tags: [performance, oracle, abci, scalability]
dependencies: []
---

## Problem Statement

The `CleanupOldOutlierHistoryGlobal` function in `x/oracle/keeper/abci.go:115-166` executes a nested loop over validators and assets every block, resulting in O(n×m) complexity. This will exceed block gas limits as the network grows.

**Why it matters:** With 100 validators × 50 assets = 5,000 storage operations per block.

## Findings

### Source: performance-oracle agent

**Location:** `/home/decri/blockchain-projects/paw/x/oracle/keeper/abci.go:115-166`

```go
// Lines 141-152: Nested iteration every block
for _, vo := range validatorOracles {
    for _, price := range prices {
        if err := k.CleanupOldOutlierHistory(ctx, vo.ValidatorAddr, price.Asset, minHeight); err != nil {
            // ...continues processing ALL validator-asset pairs
        }
        cleanedCount++
    }
}
```

**Scalability Impact:**
- 100 validators × 50 assets = 5,000 operations/block
- At 1s block time = 300,000 operations/minute
- Gas grows quadratically with validator set
- **EndBlocker will timeout with large validator sets**

## Proposed Solutions

### Option A: Batched Cleanup (Recommended)
**Pros:** Spreads load across blocks
**Cons:** Slightly stale data
**Effort:** Medium
**Risk:** Low

```go
func (k Keeper) CleanupOldOutlierHistoryBatched(ctx context.Context) error {
    sdkCtx := sdk.UnwrapSDKContext(ctx)
    blockHeight := sdkCtx.BlockHeight()

    // Only run every 10 blocks
    if blockHeight % 10 != 0 {
        return nil
    }

    // Process subset of validators per block using cursor
    cursor := k.GetCleanupCursor(ctx)
    validators, err := k.GetValidatorOraclesPaginated(ctx, cursor, 10) // 10 validators per block
    if err != nil {
        return err
    }

    // ... cleanup for this batch only
    k.SetCleanupCursor(ctx, nextCursor)
    return nil
}
```

### Option B: Height-Indexed Deletion
**Pros:** O(1) per height cleanup
**Cons:** Requires schema change
**Effort:** Large
**Risk:** Medium

### Option C: Background Goroutine
**Pros:** Non-blocking
**Cons:** State consistency concerns
**Effort:** Medium
**Risk:** High

## Recommended Action

**Implement Option A** - batch processing with cursor to spread cleanup across blocks.

## Technical Details

**Affected Files:**
- `x/oracle/keeper/abci.go`
- `x/oracle/keeper/validator.go` (add pagination)

**Database Changes:** Add cleanup cursor storage

## Acceptance Criteria

- [ ] Cleanup runs every 10 blocks (configurable)
- [ ] Process 10 validators per batch
- [ ] Cursor tracks cleanup progress
- [ ] Benchmark: measure gas at 100, 500, 1000 validators
- [ ] No EndBlocker timeout under load

## Work Log

| Date | Action | Notes |
|------|--------|-------|
| 2025-12-05 | Created | Identified by performance-oracle agent |

## Resources

- Related issues in ProcessSlashWindows (same pattern)
- Cosmos SDK pagination patterns
