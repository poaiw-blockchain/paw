# Escrow State Not Exported in Genesis

---
status: pending
priority: p1
issue_id: "018"
tags: [data-integrity, compute, genesis, critical]
dependencies: ["006"]
---

## Problem Statement

The `ExportGenesis` function in `x/compute/keeper/genesis.go` does NOT export `EscrowState` records, but `InitGenesis` expects to restore them. This will cause **catastrophic data loss** during chain upgrades or genesis export/import operations.

**Why it matters:** All escrowed funds will be permanently locked after any chain restart that uses genesis export/import.

## Findings

### Source: data-integrity-guardian agent

**Location:** `/home/decri/blockchain-projects/paw/x/compute/keeper/genesis.go:149-261`

**Missing Exports:**
- `EscrowState` records (stored under key prefix `0x20`)
- Escrow timeout indexes (key prefix `0x21`)
- Escrow nonce counter (key `0x22`)

**Data Loss Scenario:**
1. Chain running with 50 active compute requests, each with escrowed funds
2. Governance initiates chain upgrade via genesis export/import
3. Genesis export captures: providers, requests, results, disputes, appeals
4. Genesis export MISSING: all escrow state records
5. Chain restarts with genesis import
6. All escrow funds remain locked in module account
7. No escrow metadata exists to track which requests own which funds
8. Requests show `EscrowedAmount` but `GetEscrowState()` returns "not found"
9. **Permanent fund lockup** - escrow cannot be released or refunded

## Proposed Solutions

### Option A: Add Escrow Export/Import (Recommended)
**Pros:** Complete fix, preserves all escrow state
**Cons:** Requires proto changes
**Effort:** Medium
**Risk:** Low

```go
// In ExportGenesis, add before return statement:
var escrowStates []types.EscrowState
store := k.getStore(ctx)
escrowIter := storetypes.KVStorePrefixIterator(store, EscrowStateKeyPrefix)
defer escrowIter.Close()
for ; escrowIter.Valid(); escrowIter.Next() {
    var escrow types.EscrowState
    if err := k.cdc.Unmarshal(escrowIter.Value(), &escrow); err != nil {
        return nil, fmt.Errorf("failed to unmarshal escrow state: %w", err)
    }
    escrowStates = append(escrowStates, escrow)
}

// In InitGenesis, add restoration:
for _, escrow := range data.EscrowStates {
    if err := k.SetEscrowState(ctx, escrow); err != nil {
        return fmt.Errorf("failed to set escrow state %d: %w", escrow.RequestId, err)
    }
    if escrow.Status == types.ESCROW_STATUS_LOCKED {
        if err := k.setEscrowTimeoutIndex(ctx, escrow.RequestId, escrow.ExpiresAt); err != nil {
            return fmt.Errorf("failed to restore timeout index: %w", err)
        }
    }
}
```

## Recommended Action

**Implement Option A immediately** - this is a data integrity critical issue.

## Technical Details

**Affected Files:**
- `x/compute/keeper/genesis.go`
- `x/compute/types/genesis.go`
- `proto/paw/compute/v1/state.proto` (add EscrowState to GenesisState)

**Proto Changes Required:**
```protobuf
message GenesisState {
  // ... existing fields
  repeated EscrowState escrow_states = 12;
  uint64 next_escrow_nonce = 13;
}
```

## Acceptance Criteria

- [ ] EscrowState records exported in genesis
- [ ] Escrow timeout indexes exported
- [ ] NextEscrowNonce counter exported
- [ ] InitGenesis restores all escrow state
- [ ] Add genesis round-trip test with active escrows
- [ ] Add invariant: module balance >= sum(locked escrows)

## Work Log

| Date | Action | Notes |
|------|--------|-------|
| 2025-12-07 | Created | Identified by data-integrity-guardian agent |

## Resources

- Related: Issue #006 (DEX genesis incomplete)
- Cosmos SDK genesis handling best practices
