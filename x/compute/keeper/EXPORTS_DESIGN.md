# Test Helper Exports Design

## Problem

Go's test visibility rules:
- `*_test.go` files are only visible to tests in the SAME package
- External test packages (e.g., `compute_test`) cannot access functions in `keeper/export_test.go`
- `ibc_module_test.go` is in package `compute_test`, so it couldn't call `keeper.TrackPendingOperationForTest`

## Solution

Split test helpers into two files:

1. **exports.go** (NEW) - For external test packages
   - No `_test.go` suffix, so it compiles into the main keeper package
   - Functions are exported (capitalized) and accessible from ANY test package
   - Used by: `ibc_module_test.go` and other external tests

2. **export_test.go** (UPDATED) - For internal keeper tests only
   - Keeps `_test.go` suffix for internal-only helpers
   - Only visible to `keeper_test` package tests
   - Used by: tests in the keeper package itself

## File Contents

### exports.go
Functions needed by external test packages:
- `TrackPendingOperationForTest()` - Track IBC channel operations
- `RecordNonceUsageForTesting()` - Record nonce usage
- `CheckReplayAttackForTesting()` - Check replay attacks
- `AuthorizeComputeChannelForTest()` - Authorize IBC channels
- `GetAuthority()` - Get module authority
- `GetDisputeForTesting()` - Get disputes
- `GetBankKeeper()` - Access bank keeper

### export_test.go
Functions only for internal keeper tests:
- `SetSlashRecordForTest()` - Set slash records
- `SetAppealForTest()` - Set appeals
- `VerifyIBCZKProofForTest()` - Verify ZK proofs
- `VerifyAttestationsForTest()` - Verify attestations
- `GetValidatorPublicKeysForTest()` - Get validator keys
- `GetStoreKeyForTesting()` - Access store key

## Usage

```go
// From external test package (compute_test)
keeper.TrackPendingOperationForTest(k, ctx, keeper.ChannelOperation{...})

// From internal keeper test package (keeper_test)
k.SetSlashRecordForTest(ctx, record)
```

## Verification

All tests pass:
- `go build ./x/compute/...` - Compiles successfully
- `go test ./x/compute/...` - All tests pass
