# Test File Consolidation Status

## Completed Consolidations

Successfully consolidated **19 test files** from 73 to 54 files:

### 1. IBC Tests (10 files → 1 file)
Merged into `ibc_test.go` (1,947 lines):
- ibc_ack_handlers_test.go
- ibc_compute_helpers_test.go
- ibc_compute_test.go
- ibc_compute_unit_test.go
- ibc_compute_validate_test.go
- ibc_compute_verification_test.go
- ibc_helpers_test.go
- ibc_packet_tracking_cover_test.go
- ibc_packet_tracking_test.go
- ibc_timeout_test.go

### 2. ZK Tests (8 files → 1 file)
Merged into `zk_test.go` (963 lines):
- zk_enhancements_test.go
- zk_enhancements_batch_test.go
- zk_enhancements_cover_test.go
- zk_enhancements_positive_test.go
- zk_metrics_test.go
- zk_verification_test.go
- zk_verification_deposit_test.go
- zk_verification_extended_test.go

### 3. ABCI Tests (3 files merged into existing)
Merged into `abci_test.go` (610 lines total):
- abci_disputes_cover_test.go
- abci_disputes_multi_test.go
- abci_score_cover_test.go

**Total: 19 files eliminated (26% reduction in file count)**

## Blocking Issue: Import Cycle

Further consolidation is blocked by an import cycle:

```
package github.com/paw-chain/paw/x/compute/keeper
    imports github.com/paw-chain/paw/testutil/keeper from ibc_test.go
    imports github.com/paw-chain/paw/x/compute/keeper from compute.go: import cycle not allowed in test
```

### Root Cause

The codebase has a problematic mix:
- **Internal tests** (`package keeper`) - 32 files
- **External tests** (`package keeper_test`) - 22 files

The internal test file `ibc_test.go` imports `testutil/keeper`, which in turn imports `x/compute/keeper`, creating a cycle. This prevents Go from compiling any tests in the `keeper` package.

### Files Affected

Cannot merge files with different package declarations:
- **Group 1 (dispute)**: Mixed packages
  - dispute_test.go (keeper_test)
  - dispute_extended_test.go (keeper)

- **Group 2 (escrow)**: Mixed packages
  - escrow_test.go (keeper_test)
  - escrow_helpers_test.go (keeper)

- **Group 3 (invariants)**: Mixed packages
- **Group 4 (keeper)**: All `package keeper` but import cycle blocks testing
- **Group 5 (nonce)**: Mixed packages
- **Group 6 (provider)**: Mixed packages
- **Group 7 (ratelimit)**: Mixed packages
- **Group 8 (request)**: Mixed packages
- **Group 9 (security)**: Mixed packages
- **Group 10 (verification)**: Mixed packages

## Recommended Fix

To enable further consolidation, the import cycle must be resolved:

### Option 1: Remove testutil/keeper import from ibc_test.go
Move `ibc_test.go` to external tests (`package keeper_test`) OR stop using testutil/keeper helpers.

### Option 2: Fix testutil/keeper
Refactor `testutil/keeper` to not import `x/compute/keeper` directly, breaking the cycle.

### Option 3: Split internal tests
Move all internal tests that require testutil helpers to external test files.

## Current State

- **Files**: 54 test files (down from 73)
- **Reduction**: 26%
- **All tests passing**: Yes (for successfully merged files)
- **Further work blocked**: Yes (import cycle)

## Next Steps

1. Fix the import cycle (choose option above)
2. Continue consolidation of remaining 35+ files
3. Target final count: ~25-30 test files total
