# Missing Pagination in Critical Queries

---
status: pending
priority: p2
issue_id: "009"
tags: [performance, queries, scalability, architecture]
dependencies: []
---

## Problem Statement

Multiple query endpoints return unbounded result sets without pagination. As state grows, queries will fail with memory exhaustion, causing node instability.

**Why it matters:** Query nodes become unstable, users experience timeouts and failures.

## Findings

### Source: architecture-strategist & performance-oracle agents

**Critical Queries Without Pagination:**
1. `QueryAllPools` - DEX pools (could return thousands)
2. `QueryPriceSnapshots` - Oracle snapshots (grows every block)
3. `QueryValidatorPrices` - One per validator per price feed
4. `QueryAllRequests` - Compute requests (unbounded)
5. `QueryAllProviders` - Compute providers (unbounded)
6. `GetAllValidatorOracles` - Used in ABCI (O(n) every block)

**Example from `x/oracle/keeper/validator.go:88-96`:**
```go
func (k Keeper) GetAllValidatorOracles(ctx context.Context) ([]types.ValidatorOracle, error) {
    validatorOracles := []types.ValidatorOracle{}
    err := k.IterateValidatorOracles(ctx, func(vo types.ValidatorOracle) bool {
        validatorOracles = append(validatorOracles, vo) // Unbounded append
        return false
    })
    return validatorOracles, err
}
```

**Impact:**
- Memory exhaustion on query nodes
- Slow response times degrade UX
- Applications can't fetch complete state
- Used in ABCI paths = consensus slowdown

## Proposed Solutions

### Option A: Add Cosmos SDK Pagination (Recommended)
**Pros:** Standard pattern, well-tested
**Cons:** API breaking change
**Effort:** Medium
**Risk:** Low

```go
func (k QueryServer) Pools(
    ctx context.Context,
    req *types.QueryPoolsRequest,
) (*types.QueryPoolsResponse, error) {
    if req == nil {
        return nil, status.Error(codes.InvalidArgument, "empty request")
    }

    sdkCtx := sdk.UnwrapSDKContext(ctx)
    store := prefix.NewStore(sdkCtx.KVStore(k.storeKey), types.PoolKeyPrefix)

    var pools []types.Pool
    pageRes, err := query.Paginate(store, req.Pagination, func(key, value []byte) error {
        var pool types.Pool
        if err := k.cdc.Unmarshal(value, &pool); err != nil {
            return err
        }
        pools = append(pools, pool)
        return nil
    })
    if err != nil {
        return nil, status.Error(codes.Internal, err.Error())
    }

    return &types.QueryPoolsResponse{
        Pools:      pools,
        Pagination: pageRes,
    }, nil
}
```

### Option B: Add Streaming API
**Pros:** Handles very large datasets
**Cons:** More complex client handling
**Effort:** Large
**Risk:** Medium

## Recommended Action

**Implement Option A** for all query endpoints:
- Default page size: 100
- Max page size: 1000
- Use keyset pagination where possible

## Technical Details

**Affected Files:**
- `x/dex/keeper/query_server.go`
- `x/oracle/keeper/query_server.go`
- `x/compute/keeper/query_server.go`
- All `*_test.go` files for queries

**Proto Changes Required:**
```protobuf
message QueryPoolsRequest {
    cosmos.base.query.v1beta1.PageRequest pagination = 1;
}

message QueryPoolsResponse {
    repeated Pool pools = 1;
    cosmos.base.query.v1beta1.PageResponse pagination = 2;
}
```

## Acceptance Criteria

- [ ] All "GetAll" queries have pagination
- [ ] Default page size = 100
- [ ] Max page size = 1000
- [ ] Pagination tests added
- [ ] Benchmark: query 10,000 pools with pagination
- [ ] No memory spikes on large queries

## Work Log

| Date | Action | Notes |
|------|--------|-------|
| 2025-12-05 | Created | Identified by architecture-strategist and performance-oracle agents |

## Resources

- Cosmos SDK query pagination: https://docs.cosmos.network/main/build/building-modules/query
- Related: TEST-2 in ROADMAP (query server tests)
