# Property Test Failures - Wallet Crypto

---
status: pending
priority: p2
issue_id: "015"
tags: [testing, wallet, crypto, bug]
dependencies: []
---

## Problem Statement

Property test failure artifacts are committed to git, indicating unresolved cryptographic test failures in the wallet module.

**Why it matters:** BIP32 derivation or encryption issues could compromise user funds.

## Findings

### Source: git-history-analyzer agent

**Failure Artifacts in Git:**
```
testBIP32PathProperties-20251130102956-282891.fail
testDerivationProperties-20251130102956-282891.fail
testEncryptionProperties-20251130102956-282891.fail
```

**Date:** 2025-11-30

**Affected Areas:**
1. BIP32 path properties - HD wallet derivation
2. Derivation properties - Key derivation from mnemonic
3. Encryption properties - Wallet encryption/decryption

**Implications:**
- Property tests found edge cases that break expectations
- Cryptographic operations may produce incorrect results
- Failure artifacts should never be committed to git

## Proposed Solutions

### Option A: Investigate and Fix (Recommended)
**Pros:** Resolves underlying issues
**Cons:** May reveal complex bugs
**Effort:** Medium-Large
**Risk:** Low

**Steps:**
1. Reproduce failures using saved artifacts
2. Identify edge cases that break properties
3. Fix implementation or adjust property bounds
4. Remove failure artifacts from git
5. Add `.fail` to `.gitignore`

### Option B: Adjust Property Bounds
**Pros:** Quick if properties too strict
**Cons:** May hide real bugs
**Effort:** Small
**Risk:** High

## Recommended Action

**Implement Option A** - investigate root cause before dismissing.

## Technical Details

**Affected Files:**
- `wallet/*/test/*_test.go` (property tests)
- Failure artifacts (to be deleted)
- `.gitignore` (add `*.fail`)

**Failure Reproduction:**
```bash
# Load and reproduce failure
go test -run TestBIP32PathProperties -args -seed <seed-from-file>
```

## Acceptance Criteria

- [ ] Reproduce each failure locally
- [ ] Root cause identified for each failure
- [ ] Implementation fixed OR property adjusted with justification
- [ ] All property tests pass
- [ ] Failure artifacts removed from git
- [ ] `*.fail` added to .gitignore
- [ ] Add fuzz testing duration to CI

## Work Log

| Date | Action | Notes |
|------|--------|-------|
| 2025-12-05 | Created | Identified by git-history-analyzer agent |

## Resources

- Go fuzzing documentation
- BIP32 specification
- Failure artifacts contain seeds to reproduce
