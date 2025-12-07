# Unbounded Order Matching in EndBlocker

---
status: pending
priority: p1
issue_id: "020"
tags: [performance, dex, abci, critical]
dependencies: []
---

## Problem Statement

`MatchAllOrders` in `x/dex/keeper/limit_orders.go` loads ALL open orders and iterates over them every block. With 10,000+ orders, this could exhaust block gas limits and cause block production to halt.

**Why it matters:** Chain halt risk under high order volume.

## Findings

### Source: performance-oracle agent

**Location:** `/home/decri/blockchain-projects/paw/x/dex/keeper/limit_orders.go:1004-1019`

**Code:**
```go
func (k Keeper) MatchAllOrders(ctx context.Context) error {
    orders, err := k.GetOpenOrders(ctx)  // Loads ALL open orders
    if err != nil {
        return err
    }

    for _, order := range orders {  // O(n) iteration
        if err := k.MatchLimitOrder(ctx, order); err != nil {
            sdkCtx.Logger().Error("failed to match order", "order_id", order.ID, "error", err)
        }
    }
    return nil
}
```

**Complexity:** O(n) where n = total open orders

**Impact:**
- At 1,000 orders: ~0.5-1M gas
- At 10,000 orders: ~5-10M gas (approaching block limits)
- At 100,000 orders: Chain halt risk

## Proposed Solutions

### Option A: Batched Order Matching (Recommended)
**Pros:** Prevents chain halt, distributes load
**Cons:** Orders take multiple blocks to match
**Effort:** Medium
**Risk:** Low

```go
const maxOrdersPerBlock = 100
const maxGasForMatching = 5_000_000

func (k Keeper) MatchAllOrders(ctx context.Context) error {
    sdkCtx := sdk.UnwrapSDKContext(ctx)
    startGas := sdkCtx.GasMeter().GasConsumed()
    matched := 0

    iterator := k.getOpenOrdersIterator(ctx)
    defer iterator.Close()

    for ; iterator.Valid(); iterator.Next() {
        if matched >= maxOrdersPerBlock {
            break // Continue next block
        }
        if sdkCtx.GasMeter().GasConsumed() - startGas > maxGasForMatching {
            break // Gas limit reached
        }

        var order LimitOrder
        k.cdc.Unmarshal(iterator.Value(), &order)
        k.MatchLimitOrder(ctx, &order)
        matched++
    }
    return nil
}
```

### Option B: Priority-Based Matching
**Pros:** Important orders matched first
**Cons:** More complex implementation
**Effort:** Large
**Risk:** Medium

## Recommended Action

**Implement Option A** - batch processing with gas limits.

## Technical Details

**Affected Files:**
- `x/dex/keeper/limit_orders.go`
- `x/dex/keeper/abci.go`

**Database Changes:** None

## Acceptance Criteria

- [ ] Maximum orders matched per block is bounded (100)
- [ ] Gas consumption capped at 5M per block for matching
- [ ] Benchmark at 100, 1000, 10000 orders
- [ ] No EndBlocker timeout under high load
- [ ] Orders eventually all get matched (FIFO)

## Work Log

| Date | Action | Notes |
|------|--------|-------|
| 2025-12-07 | Created | Identified by performance-oracle agent |

## Resources

- Related: Issue #008 (oracle ABCI performance)
- Cosmos SDK EndBlocker best practices
