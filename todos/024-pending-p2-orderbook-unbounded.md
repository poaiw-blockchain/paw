# GetOrderBook Loads Entire Order Set

---
status: pending
priority: p2
issue_id: "024"
tags: [performance, dex, queries, high]
dependencies: ["009"]
---

## Problem Statement

`GetOrderBook` loads ALL orders for a pool without pagination. Popular pools with thousands of orders will cause query timeouts and memory exhaustion.

**Why it matters:** Query nodes become unstable, users experience timeouts.

## Findings

### Source: performance-oracle agent

**Location:** `/home/decri/blockchain-projects/paw/x/dex/keeper/limit_orders.go:945-963`

**Code:**
```go
func (k Keeper) GetOrderBook(ctx context.Context, poolID uint64) (buyOrders, sellOrders []*LimitOrder, err error) {
    orders, err := k.GetOrdersByPool(ctx, poolID)  // Loads ALL orders
    if err != nil {
        return nil, nil, err
    }

    for _, order := range orders {  // Filters in memory
        if order.Status != OrderStatusOpen && order.Status != OrderStatusPartial {
            continue
        }
        if order.OrderType == OrderTypeBuy {
            buyOrders = append(buyOrders, order)
        } else {
            sellOrders = append(sellOrders, order)
        }
    }
    return buyOrders, sellOrders, nil
}
```

**Complexity:** O(n) where n = total orders in pool (unbounded)

**Impact:**
- Memory: ~500 bytes per order x 10,000 orders = 5MB per query
- CPU: O(n) deserialization + filtering
- Network: Unbounded response size

## Proposed Solutions

### Option A: Add Index and Limit Parameter (Recommended)
**Pros:** Efficient, standard practice
**Cons:** Requires index
**Effort:** Medium
**Risk:** Low

```go
// Add index for open orders by pool and status
// Key: 0x09 + poolID + status + orderID

func (k Keeper) GetOrderBook(ctx context.Context, poolID uint64, limit int) (buyOrders, sellOrders []*LimitOrder, err error) {
    if limit == 0 || limit > 100 {
        limit = 50  // Default safe limit
    }

    buyOrders = k.getOrdersByPoolAndStatus(ctx, poolID, OrderStatusOpen, OrderTypeBuy, limit)
    sellOrders = k.getOrdersByPoolAndStatus(ctx, poolID, OrderStatusOpen, OrderTypeSell, limit)

    return buyOrders, sellOrders, nil
}
```

## Recommended Action

**Implement Option A** - add secondary index and limit parameter.

## Technical Details

**Affected Files:**
- `x/dex/keeper/limit_orders.go`
- `x/dex/types/keys.go` (add new key prefix)

**Database Changes:** Add index for open orders by pool/status

## Acceptance Criteria

- [ ] GetOrderBook accepts limit parameter
- [ ] Default limit is 50, max is 100
- [ ] Index created for efficient filtering
- [ ] Benchmark: query time < 100ms with 10,000 orders
- [ ] Memory usage bounded

## Work Log

| Date | Action | Notes |
|------|--------|-------|
| 2025-12-07 | Created | Identified by performance-oracle agent |

## Resources

- Related: Issue #009 (pagination missing)
- Cosmos SDK pagination patterns
