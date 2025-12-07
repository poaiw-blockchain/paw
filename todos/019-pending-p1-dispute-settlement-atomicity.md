# Dispute Settlement Non-Atomic with Escrow Operations

---
status: pending
priority: p1
issue_id: "019"
tags: [data-integrity, compute, escrow, critical]
dependencies: ["001"]
---

## Problem Statement

`SettleDisputeOutcome` in `x/compute/keeper/dispute.go` performs escrow operations using `_` (ignore errors), creating potential for inconsistent state where dispute is resolved but escrow operation fails silently.

**Why it matters:** State divergence between escrow records and actual fund locations, breaking invariants.

## Findings

### Source: data-integrity-guardian agent

**Location:** `/home/decri/blockchain-projects/paw/x/compute/keeper/dispute.go:564-653`

**Code Evidence:**
```go
// dispute.go:630-632
if !escrowAmount.IsZero() {
    _ = k.RefundEscrow(ctx, request.Id, "provider_fault")  // ERROR IGNORED
}

// dispute.go:635-637
if !escrowAmount.IsZero() {
    _ = k.ReleaseEscrow(ctx, request.Id, true)  // ERROR IGNORED
}
```

**State Corruption Scenario:**
1. Dispute #42, RequestID #100, Resolution: SLASH_PROVIDER
2. Escrow: 1000 upaw LOCKED for request #100
3. Slash provider stake: 500 upaw burned (succeeds)
4. Call RefundEscrow(100, "provider_fault")
   - Bank transfer: module → requester (succeeds)
   - SetEscrowState fails (disk full / cosmos DB error)
5. Error ignored due to `_ = k.RefundEscrow(...)`
6. Function returns nil (success)
7. Result: Funds returned but escrow state still shows LOCKED
8. EscrowBalanceInvariant will fail

## Proposed Solutions

### Option A: Proper Error Propagation (Recommended)
**Pros:** Simple, follows CEI pattern
**Cons:** None
**Effort:** Small
**Risk:** Low

```go
// Replace all `_ = k.RefundEscrow(...)` with proper error handling:
if !escrowAmount.IsZero() {
    if err := k.RefundEscrow(ctx, request.Id, "provider_fault"); err != nil {
        return fmt.Errorf("failed to refund escrow during dispute settlement: %w", err)
    }
}

// Similar for ReleaseEscrow
if !escrowAmount.IsZero() {
    if err := k.ReleaseEscrow(ctx, request.Id, true); err != nil {
        return fmt.Errorf("failed to release escrow during dispute settlement: %w", err)
    }
}
```

## Recommended Action

**Implement Option A** - propagate all errors properly.

## Technical Details

**Affected Files:**
- `x/compute/keeper/dispute.go`

**Database Changes:** None

## Acceptance Criteria

- [ ] All escrow operations in SettleDisputeOutcome return errors properly
- [ ] No `_ =` error ignoring in dispute settlement code
- [ ] Add test: mock escrow failure, verify dispute settlement fails atomically
- [ ] Transaction atomicity maintained - either all succeed or all fail

## Work Log

| Date | Action | Notes |
|------|--------|-------|
| 2025-12-07 | Created | Identified by data-integrity-guardian agent |

## Resources

- Related: Issue #001 (escrow state inconsistency)
- Cosmos SDK transaction atomicity
