# API Migration Notes for PAW Blockchain v1.0

**Status**: Documentation Only - No Action Required for v1.0
**Date**: 2025-11-25

## Overview

This document tracks Cosmos SDK API usage in PAW v1.0 that may require migration in future versions. These APIs are currently **functional and stable** but are marked as "Legacy" or "beta" in upstream SDK.

**IMPORTANT**: These are informational notes only. The v1.0 codebase is clean and production-ready. These items are tracked for potential future optimization, not as blockers.

---

## 1. LegacyDec API Usage

### Status: FUNCTIONAL - Monitor for Deprecation

**API**: `cosmossdk.io/math.LegacyDec`
**Usage**: 100+ occurrences across 37 files
**Introduced**: Cosmos SDK v0.50+
**Status**: Stable but marked "Legacy"

### Background

Cosmos SDK introduced `LegacyDec` when transitioning decimal types. Despite the "Legacy" name, this is currently the **recommended stable decimal type** for:
- Fixed-point arithmetic
- Financial calculations
- Price calculations
- Fee computations

The "Legacy" designation indicates:
- Maintains SDK 0.47.x API compatibility
- Will eventually be replaced by a new decimal type
- Still fully supported and maintained

### Files Using LegacyDec

**Fuzz Tests** (34 occurrences):
- `tests/fuzz/dex_fuzz.go`
- `tests/fuzz/proto_fuzz.go`
- `tests/fuzz/safemath_fuzz.go`

**Module Keepers**:
- `x/dex/keeper/*.go` (pool calculations, swap math, fees)
- `x/oracle/keeper/*.go` (price aggregation, TWAP, median calculations)
- `x/compute/keeper/*.go` (cost calculations, escrow amounts)

**SDK & Helpers**:
- `sdk/go/helpers/helpers.go`
- `simapp/params.go`
- `simapp/state.go`
- `app/params.go`

### Example Usage

```go
// Price calculation
fee, err := math.LegacyNewDecFromStr(feeStr)
if err != nil || fee.IsNegative() || fee.GTE(math.LegacyOneDec()) {
    return errors.New("invalid fee")
}

// Pool math
k := math.LegacyNewDec(reserveA).Mul(math.LegacyNewDec(reserveB))
```

### Recommendation for Future

**v1.0**: ✅ Keep as-is - LegacyDec is the correct choice
**v1.1+**: Monitor Cosmos SDK for:
- Introduction of new stable decimal type
- Deprecation warnings for LegacyDec
- Migration guides from SDK team

**When to Migrate**: Only when Cosmos SDK officially deprecates LegacyDec and provides a stable replacement with migration path.

---

## 2. v1beta1 Governance API

### Status: FUNCTIONAL - Evaluate for v1 Migration

**API**: `github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1`
**Usage**: 6 occurrences across 6 files
**Alternative**: `github.com/cosmos/cosmos-sdk/x/gov/types/v1`

### Background

Cosmos SDK governance module has both:
- **v1beta1**: Legacy API, maintained for compatibility
- **v1**: Current stable API with improved features

PAW v1.0 uses v1beta1 in some integration points while other areas use v1.

### Files Using v1beta1

1. **app/app.go:82**
   ```go
   govv1beta1 "github.com/cosmos/cosmos-sdk/x/gov/types/v1beta1"
   ```
   - Used for parameter change proposal handlers
   - Can be migrated to v1 API

2. **explorer/indexer/internal/indexer/indexer.go:381**
   - Proposal type handling in indexer
   - Can be updated when migrating to v1

3. **status/backend/pkg/health/monitor.go:105**
4. **status/backend/pkg/metrics/metrics.go:178**
   - Monitoring/metrics collection
   - Non-critical, can be updated

5. **faucet/backend/pkg/faucet/faucet.go:131, 189, 194, 201**
   - Faucet service governance queries
   - Can be migrated to v1 queries

6. **tests/integration/wallet_integration_test.go:682**
   - Integration test assertions
   - Update when migrating core code

### Recommendation for Future

**v1.0**: ✅ Ship as-is - v1beta1 is still fully supported
**v1.1**: Consider migrating to v1 API for:
- Improved proposal features
- Better type safety
- Alignment with latest SDK best practices

**Migration Effort**: Low - mostly import path changes and minor API adjustments

---

## 3. Protocol Version Management

### Status: CORRECT IMPLEMENTATION - No Action Needed

**Pattern**: Version checking in P2P protocol
**Files**: `p2p/protocol/messages.go`

### Code Examples

```go
if m.ProtocolVersion != CurrentProtocolVersion {
    return fmt.Errorf("unsupported protocol version: %d", m.ProtocolVersion)
}
```

### Assessment

✅ **This is NOT backwards compatibility** - this is proper protocol versioning.

Protocol version checks are:
- Essential for P2P network compatibility
- Standard practice in all blockchain implementations
- Required for future protocol upgrades
- Part of the v1.0 protocol specification

**Action**: None - this is correct implementation.

---

## 4. Test Generator Template TODOs

### Status: INTENTIONAL DESIGN - No Action Needed

**File**: `scripts/coverage_tools/go_test_generator.go`
**Pattern**: Template generates TODO comments

### Example

```go
{{ range .ParamNames }}
    {{ . }}: nil, // TODO: Add test value
{{ end }}
want: nil, // TODO: Add expected value
```

### Assessment

✅ **This is INTENTIONAL** - the script generates test templates with placeholder TODOs.

This is proper test generation design:
- Reminds developers to fill in test values
- Standard practice for code generators
- Not actual incomplete code

**Action**: None - document that generated tests need manual completion.

---

## Migration Priority Matrix

| Item | Current Status | Migration Priority | Estimated Effort | Target Version |
|------|---------------|-------------------|-----------------|----------------|
| LegacyDec | ✅ Stable | Low - Monitor only | Medium | When SDK deprecates |
| v1beta1 Gov API | ✅ Functional | Low | Low | v1.1 optional |
| Protocol Versioning | ✅ Correct | None | N/A | N/A |
| Test Generator | ✅ Intentional | None | N/A | N/A |

---

## Monitoring Recommendations

### For LegacyDec:

Monitor Cosmos SDK release notes for:
- Deprecation warnings
- New decimal type introduction
- Migration guides
- Breaking changes in math package

**Subscribe to**: Cosmos SDK  releases, ADR discussions

### For v1beta1 Governance:

Monitor for:
- Official deprecation announcements
- Removal timelines
- Migration tooling

**Current Assessment**: v1beta1 still widely used in ecosystem, no immediate pressure to migrate.

---

## v1.0 Decision Summary

**All "legacy" API usage in PAW v1.0 is intentional and appropriate:**

1. **LegacyDec**: Current recommended approach for decimal math
2. **v1beta1**: Still fully supported, migration is optional optimization
3. **Protocol versioning**: Correct implementation, not legacy code
4. **Test generators**: Intentional design pattern

**No migration required for v1.0 release.**

---

## References

- [Cosmos SDK v0.50 Release Notes](https://github.com/cosmos/cosmos-sdk/releases/tag/v0.50.0)
- [Cosmos SDK Math Package](https://pkg.go.dev/cosmossdk.io/math)
- [Governance Module Documentation](https://docs.cosmos.network/main/modules/gov)

---

**Document Version**: 1.0
**Last Updated**: 2025-11-25
**Next Review**: Before v1.1 planning
