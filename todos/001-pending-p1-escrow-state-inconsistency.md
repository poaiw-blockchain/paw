# Escrow State Inconsistency Vulnerability

---
status: pending
priority: p1
issue_id: "001"
tags: [security, compute, escrow, critical]
dependencies: []
---

## Problem Statement

The escrow release flow in `x/compute/keeper/escrow.go:201-211` has a catastrophic failure scenario where escrow state is marked as RELEASED but the payment transfer fails, causing permanent fund lock.

**Why it matters:** Users could lose funds permanently with no recovery path except governance intervention.

## Findings

### Source: security-sentinel agent

**Location:** `/home/decri/blockchain-projects/paw/x/compute/keeper/escrow.go:201-211`

```go
escrowState.Status = types.ESCROW_STATUS_RELEASED
escrowState.ReleasedAt = &now
escrowState.ReleaseAttempts++

if err := k.SetEscrowState(ctx, *escrowState); err != nil {
    return fmt.Errorf("failed to update escrow state: %w", err)
}

// CRITICAL: State already committed above
coins := sdk.NewCoins(sdk.NewCoin("upaw", escrowState.Amount))
if err := k.bankKeeper.SendCoinsFromModuleToAccount(sdkCtx, types.ModuleName, provider, coins); err != nil {
    // Payment failed but state shows RELEASED - funds locked forever
    k.recordCatastrophicFailure(ctx, requestID, provider, escrowState.Amount, "state updated but payment failed")
    return fmt.Errorf("failed to release payment: %w", err)
}
```

**Data Corruption Scenario:**
1. Escrow state updated to RELEASED and committed to store
2. Bank transfer fails (module account issue, state corruption, etc.)
3. Escrow shows RELEASED but provider never received funds
4. Module account still holds funds but they're marked as released
5. No automatic recovery - requires governance proposal

## Proposed Solutions

### Option A: Revert State on Transfer Failure (Recommended)
**Pros:** Simple, handles failure gracefully
**Cons:** Requires careful error handling
**Effort:** Small
**Risk:** Low

```go
// Save original state for rollback
originalStatus := escrowState.Status

// Attempt transfer FIRST (before state change)
coins := sdk.NewCoins(sdk.NewCoin("upaw", escrowState.Amount))
if err := k.bankKeeper.SendCoinsFromModuleToAccount(sdkCtx, types.ModuleName, provider, coins); err != nil {
    return fmt.Errorf("failed to release payment: %w", err)
}

// ONLY update state after successful transfer
escrowState.Status = types.ESCROW_STATUS_RELEASED
escrowState.ReleasedAt = &now
if err := k.SetEscrowState(ctx, *escrowState); err != nil {
    // Transfer succeeded but state update failed - this is also bad
    // but funds are at least with the correct party
    return fmt.Errorf("CRITICAL: payment sent but state update failed: %w", err)
}
```

### Option B: Two-Phase Commit with Recovery Job
**Pros:** More robust, handles all failure scenarios
**Cons:** More complex, requires background job
**Effort:** Medium
**Risk:** Medium

### Option C: Idempotent Release with Finalization Flag
**Pros:** Prevents double-release, allows retry
**Cons:** Adds complexity
**Effort:** Medium
**Risk:** Low

## Recommended Action

**Implement Option A** - Reorder operations to perform transfer before state update. This follows the checks-effects-interactions pattern properly.

## Technical Details

**Affected Files:**
- `x/compute/keeper/escrow.go`

**Database Changes:** None required

## Acceptance Criteria

- [ ] Transfer happens BEFORE state is marked as RELEASED
- [ ] If transfer fails, escrow remains in LOCKED state
- [ ] If state update fails after transfer, error is logged but funds are safe
- [ ] Add test: mock bank transfer failure, verify escrow stays LOCKED
- [ ] Add test: mock state update failure, verify recovery path

## Work Log

| Date | Action | Notes |
|------|--------|-------|
| 2025-12-05 | Created | Identified by security-sentinel agent |

## Resources

- Related: data-integrity-guardian also flagged this issue
- Pattern: Checks-Effects-Interactions (CEI)
- Cosmos SDK best practice: Transfer funds before updating state
