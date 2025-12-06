# Dead Code Removal - ~3,000 Lines

---
status: pending
priority: p2
issue_id: "011"
tags: [code-quality, simplification, maintenance]
dependencies: []
---

## Problem Statement

The codebase contains approximately 3,000 lines of unused or unnecessary code that increases maintenance burden and obscures the actual implementation.

**Why it matters:** Technical debt, harder to understand codebase, potential security surface.

## Findings

### Source: code-simplicity-reviewer agent

**Dead Code Summary:**

| Category | Files | Lines |
|----------|-------|-------|
| Duplicate SafeMath | 2 files | 240 |
| Triplicate Nonce | 3 files | 220 |
| Backup/Recovery (test-only) | 3 files | 1,483 |
| Unused Panic Recovery | 1 file | 64 |
| Unused Monitoring | 1 file | 311 |
| Minimal-use Audit Trail | 1 file | 279 |
| Unwired Rate Limiting | 1 file | 286 |
| Error Wrapper | 1 file | 12 |
| Unused Key Helpers | Partial | ~80 |
| **TOTAL** | **12+ files** | **~2,975 lines** |

### Detailed Findings:

**1. Duplicate SafeMath (~240 lines):**
- `x/dex/keeper/safemath.go`
- `x/compute/keeper/safemath.go`
- Issue: `math.Int` already provides overflow protection

**2. Triplicate Nonce Management (~220 lines):**
- `x/dex/keeper/nonce.go`
- `x/oracle/keeper/nonce.go`
- `x/compute/keeper/nonce.go`
- Solution: Create shared `pkg/nonce/manager.go`

**3. Backup/Recovery (1,483 lines - test-only):**
- `x/dex/keeper/backup.go`
- `x/oracle/keeper/state_recovery.go`
- `x/compute/keeper/state_recovery.go`
- Issue: File I/O in consensus layer, only used in tests
- Solution: Use genesis export/import instead

**4. Unused Panic Recovery (64 lines):**
- `x/compute/keeper/panic_recovery.go`
- Issue: Zero usage, Cosmos SDK handles panics at ABCI boundary

**5. Unused Monitoring (311 lines):**
- `x/compute/keeper/monitoring.go`
- Issue: Zero production usage, overlaps with SDK telemetry

**6. Minimal-use Audit Trail (279 lines):**
- `x/compute/keeper/audit_trail.go`
- Issue: Used once, events already provide audit trail

**7. Unwired Rate Limiting (286 lines):**
- `x/compute/keeper/ratelimit.go`
- Issue: Infrastructure exists but interceptors never registered

## Proposed Solutions

### Option A: Phased Removal (Recommended)
**Pros:** Lower risk, verifiable at each step
**Cons:** Takes longer
**Effort:** Medium
**Risk:** Low

**Phase 1:** Remove clearly unused (panic recovery, monitoring, audit trail)
**Phase 2:** Consolidate duplicates (SafeMath, nonce)
**Phase 3:** Remove backup/recovery (replace with genesis patterns)

### Option B: Single Cleanup PR
**Pros:** Done quickly
**Cons:** Higher risk, harder to review
**Effort:** Small
**Risk:** Medium

## Recommended Action

**Implement Option A** in phases with tests at each step.

## Technical Details

**Files to Delete:**
- `x/compute/keeper/panic_recovery.go`
- `x/compute/keeper/monitoring.go`
- `x/compute/keeper/audit_trail.go`
- `x/compute/keeper/ratelimit.go` (or complete wiring)
- `x/dex/keeper/backup.go`
- `x/oracle/keeper/state_recovery.go`
- `x/compute/keeper/state_recovery.go`
- `x/dex/keeper/safemath.go`
- `x/compute/keeper/safemath.go`
- `x/dex/keeper/errors.go`

**Files to Consolidate:**
- Nonce management → `pkg/nonce/`

## Acceptance Criteria

- [ ] Remove unused panic recovery (64 lines)
- [ ] Remove unused monitoring (311 lines)
- [ ] Remove minimal-use audit trail (279 lines)
- [ ] Complete rate limiting wiring OR remove (286 lines)
- [ ] Remove backup/recovery, document genesis alternative (1,483 lines)
- [ ] Consolidate SafeMath (240 lines)
- [ ] Consolidate nonce management (220 lines)
- [ ] All existing tests pass
- [ ] Code coverage maintained

## Work Log

| Date | Action | Notes |
|------|--------|-------|
| 2025-12-05 | Created | Identified by code-simplicity-reviewer agent |

## Resources

- Cosmos SDK math package
- Genesis export/import patterns
- SDK telemetry documentation
