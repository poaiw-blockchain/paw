# Missing Access Control on DeletePool

---
status: pending
priority: p2
issue_id: "025"
tags: [security, dex, access-control, high]
dependencies: []
---

## Problem Statement

The `DeletePool` function has NO authorization check. Any caller can delete empty pools, potentially breaking pool indexing and griefing pool creators.

**Why it matters:** Unauthorized pool deletion, griefing attacks.

## Findings

### Source: security-sentinel agent

**Location:** `/home/decri/blockchain-projects/paw/x/dex/keeper/pool_secure.go:254-285`

**Code:**
```go
func (k Keeper) DeletePool(ctx context.Context, poolID uint64) error {
    pool, err := k.GetPool(ctx, poolID)
    if err != nil {
        return types.ErrPoolNotFound.Wrapf("pool %d not found", poolID)
    }
    // NO AUTHORIZATION CHECK HERE

    // Verify pool is empty
    if !pool.ReserveA.IsZero() || !pool.ReserveB.IsZero() || !pool.TotalShares.IsZero() {
        return types.ErrInvalidPoolState.Wrap("cannot delete pool with active liquidity")
    }
    // ... deletion logic
}
```

**Impact:**
- Any user can delete empty pools
- Breaks pool indexing
- Disrupts price oracles that rely on pool existence
- Griefing pool creators

## Proposed Solutions

### Option A: Governance-Only Deletion (Recommended)
**Pros:** Secure, follows Cosmos SDK patterns
**Cons:** Requires governance proposal for deletion
**Effort:** Small
**Risk:** Low

```go
func (k Keeper) DeletePool(ctx context.Context, poolID uint64, authority string) error {
    // GOVERNANCE-ONLY CHECK
    if authority != k.authority {
        return errorsmod.Wrapf(govtypes.ErrInvalidSigner,
            "invalid authority; expected %s, got %s", k.authority, authority)
    }

    pool, err := k.GetPool(ctx, poolID)
    // ... rest of function
}
```

### Option B: Creator + Governance
**Pros:** Creator can also delete their empty pools
**Cons:** More complex
**Effort:** Medium
**Risk:** Low

## Recommended Action

**Implement Option A** - governance-only deletion.

## Technical Details

**Affected Files:**
- `x/dex/keeper/pool_secure.go`
- `x/dex/keeper/msg_server.go` (update to pass authority)

**Database Changes:** None

## Acceptance Criteria

- [ ] DeletePool requires governance authority
- [ ] Test: non-authority caller rejected
- [ ] Test: governance can delete empty pool
- [ ] Update msg_server.go to pass authority parameter

## Work Log

| Date | Action | Notes |
|------|--------|-------|
| 2025-12-07 | Created | Identified by security-sentinel agent |

## Resources

- Cosmos SDK authority pattern
- govtypes.ErrInvalidSigner
