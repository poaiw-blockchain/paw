# P3-SEC-3: Finalization Flag Check Implementation

## Summary
Added finalization flag check to RequestStatusInvariant to prevent double-settlement edge cases.

## Changes Made

### 1. invariants.go (lines 238-247)
Added check in `RequestStatusInvariant`:
- Verifies all COMPLETED requests have IsFinalized=true flag set
- Prevents double-settlement by catching missing finalization markers
- Returns error: "request %d is completed but not finalized (missing settlement flag)"

### 2. invariants_test.go (lines 56-285)
Added 6 comprehensive test cases:
- Detects completed request without finalization flag (fail case)
- Passes with properly finalized completed request
- Passes with multiple completed requests all finalized
- Detects one unfinalized among multiple requests
- Passes with pending/processing requests (no finalization needed)
- Verifies edge cases and error messages

### 3. keeper_capability_test.go (line 58)
Updated registered invariants list to include "escrow-state-consistency"

## Security Impact
- **Risk Mitigated**: Double-settlement attacks where funds are released twice
- **Detection**: Invariant breaks if completed request missing finalization flag
- **Prevention**: Ensures settlement atomicity guarantee

## Test Results
All tests pass (16/16 invariant tests):
```
PASS: TestRequestStatusInvariant (6 subtests)
PASS: All Invariant tests
```

## Related Code
- Finalization tracking: `isRequestFinalized()` (request.go:589)
- Finalization setter: `markRequestFinalized()` (request.go:594)
- Key storage: `RequestFinalizedKey()` (keys.go:216)
