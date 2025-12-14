# PAW Blockchain - Test Fixing Plan

**Version**: 1.0
**Date**: 2025-12-14
**Status**: Ready for Execution
**Target**: Achieve >95% test pass rate

## Overview

This plan provides a step-by-step roadmap to fix all failing tests in the PAW blockchain codebase. The plan is divided into three phases with clear priorities, dependencies, and time estimates.

**Current State**: 70.5% pass rate (31/44 packages passing)
**Target State**: >95% pass rate (42+/44 packages passing)
**Total Effort**: 20-36 hours of focused development

## Phase 1: Unblock Testing (CRITICAL)

**Goal**: Fix all compilation errors to enable test execution
**Duration**: 12-20 hours
**Priority**: CRITICAL - Must complete before any other testing

### Task 1.1: Fix p2p/discovery Compilation

**Package**: `github.com/paw-chain/paw/p2p/discovery`
**Status**: Build failed
**Priority**: CRITICAL
**Estimated Time**: 2-4 hours

**Errors to Fix**:
1. NewAddressBook signature mismatch (expects 3 args, gets 1)
2. PeerAddr struct fields removed (IP, AddedAt)
3. AddressBook.IsBad method removed

**Action Plan**:
```bash
# Files to modify:
- p2p/discovery/discovery_advanced_test.go

# Steps:
1. Update NewAddressBook calls to match new signature:
   OLD: book := NewAddressBook(logger)
   NEW: book := NewAddressBook(config, dbDir, logger)

2. Update PeerAddr struct literals:
   - Remove IP field (use Address field instead)
   - Remove AddedAt field (use Timestamp or remove)

3. Replace IsBad method calls:
   - Find replacement method (possibly IsBlacklisted or GetScore)
   - Update test logic accordingly

4. Run: go test ./p2p/discovery/... -v
5. Verify all tests pass or identify remaining issues
```

**Dependencies**: None
**Risk**: Low - straightforward API update

---

### Task 1.2: Fix p2p/protocol Compilation

**Package**: `github.com/paw-chain/paw/p2p/protocol`
**Status**: Build failed
**Priority**: CRITICAL
**Estimated Time**: 2-4 hours

**Errors to Fix**:
1. BlockMessage struct fields removed (PrevHash, Timestamp)
2. PingMessage type undefined
3. PongMessage type undefined

**Action Plan**:
```bash
# Files to modify:
- p2p/protocol/handlers_integration_test.go

# Steps:
1. Update BlockMessage struct literals:
   - Remove PrevHash field
   - Remove Timestamp field
   - Verify what fields are actually needed for tests

2. Find PingMessage/PongMessage replacements:
   - Check if moved to different package
   - Check if renamed (e.g., HeartbeatMessage)
   - Update imports and type references

3. Review test logic:
   - Ensure tests still validate correct behavior
   - Update assertions if message structure changed

4. Run: go test ./p2p/protocol/... -v
5. Verify all tests pass
```

**Dependencies**: None
**Risk**: Low - struct field updates

---

### Task 1.3: Fix p2p/security Compilation

**Package**: `github.com/paw-chain/paw/p2p/security`
**Status**: Build failed
**Priority**: CRITICAL
**Estimated Time**: 1-2 hours

**Errors to Fix**:
1. NewRateLimiter function undefined (2 occurrences)
2. Unused variable peerID

**Action Plan**:
```bash
# Files to modify:
- p2p/security/security_test.go

# Steps:
1. Find NewRateLimiter location:
   - Check if moved to different package
   - Check if renamed
   - Add correct import

2. Fix unused variable:
   - Line 281: Remove peerID or use it
   - Likely incomplete refactoring

3. Run: go test ./p2p/security/... -v
4. Verify all tests pass
```

**Dependencies**: None
**Risk**: Very low - simple import fix

---

### Task 1.4: Fix p2p/reputation Tests

**Package**: `github.com/paw-chain/paw/p2p/reputation`
**Status**: Tests failing
**Priority**: HIGH
**Estimated Time**: 2-4 hours

**Failing Tests** (6 total):
- TestGetDiversePeers
- TestMaliciousPeerDetection/oversized_messages
- TestScoreDecay
- TestSubnetLimits
- TestWhitelist

**Errors**:
- "1" is not greater than or equal to "3" - off-by-one error
- Should be true - assertion failure
- "55" is not less than "55" - edge case (should be <=)
- "invalid peer address" does not contain "subnet" - wrong error message

**Action Plan**:
```bash
# Files to analyze:
- p2p/reputation/reputation_test.go
- p2p/reputation/manager.go

# Steps:
1. Fix TestGetDiversePeers:
   - Check peer selection algorithm
   - Fix count comparison (likely >= instead of >)

2. Fix TestMaliciousPeerDetection:
   - Review malicious peer detection logic
   - Ensure oversized messages are properly flagged

3. Fix TestScoreDecay:
   - Review score decay calculation
   - Fix edge case: score == threshold (use <= not <)

4. Fix TestSubnetLimits:
   - Review subnet limit validation
   - Fix error message to include "subnet"

5. Fix TestWhitelist:
   - Review whitelist logic
   - Fix boolean assertion

6. Run: go test ./p2p/reputation/... -v
7. Verify all tests pass
```

**Dependencies**: Tasks 1.1-1.3 (to ensure full P2P suite compiles)
**Risk**: Medium - requires logic fixes, not just API updates

---

### Task 1.5: Fix DEX Flash Loan Protection Tests

**Package**: `github.com/paw-chain/paw/x/dex/keeper`
**Status**: Tests failing
**Priority**: HIGH
**Estimated Time**: 4-6 hours

**Failing Tests** (9 total):
- TestFlashLoanProtection_AddAndRemoveNextBlock
- TestFlashLoanProtection_MultipleProviders
- TestFlashLoanProtection_PartialRemoval
- TestFlashLoanProtection_IntegrationWithReentrancy
- TestFlashLoanProtection_AddMultipleThenRemove
- TestFlashLoanProtection_SimulatedAttackScenario
- TestFlashLoanProtection_ErrorMessageQuality
- TestFlashLoanProtection_MultipleAddRemoveCycles
- TestFlashLoanProtection_EdgeCases/different_pools_same_provider

**Root Cause**: Flash loan protection logic fails when liquidity is added/removed in quick succession (same or next block)

**Action Plan**:
```bash
# Files to analyze:
- x/dex/keeper/liquidity.go
- x/dex/keeper/liquidity_secure.go
- x/dex/keeper/security.go

# Investigation:
1. Review flash loan protection algorithm:
   - How is block height tracked per provider?
   - What happens when provider adds liquidity at block N, removes at block N+1?
   - Is the 10-block delay properly enforced?

2. Analyze failing scenarios:
   - Single provider, multiple add/remove cycles
   - Multiple providers in same pool
   - Different pools, same provider
   - Interaction with reentrancy protection

3. Fix implementation:
   - Ensure per-provider, per-pool tracking
   - Handle edge case: remove immediately after add
   - Handle edge case: multiple providers same address
   - Proper error messages

4. Run: go test ./x/dex/keeper/... -run FlashLoan -v
5. Verify all flash loan tests pass
6. Run full DEX test suite to ensure no regressions
```

**Dependencies**: None
**Risk**: Medium-High - security-critical code, must be correct

---

## Phase 2: Critical Functionality (HIGH)

**Goal**: Fix tests for critical blockchain features
**Duration**: 8-16 hours
**Priority**: HIGH - Required before testnet/mainnet

### Task 2.1: Fix Recovery Test Validator Setup

**Package**: `github.com/paw-chain/paw/tests/recovery`
**Status**: All tests failing (0% pass rate)
**Priority**: HIGH
**Estimated Time**: 4-8 hours

**Failing Tests**: 29 total (all recovery tests)

**Error**: "Received unexpected error" at helpers.go:199

**Root Cause**: Validator genesis setup - validator keys not properly initialized

**Action Plan**:
```bash
# Files to analyze:
- tests/recovery/helpers.go:199
- tests/recovery/crash_recovery_test.go
- tests/recovery/snapshot_test.go
- tests/recovery/wal_replay_test.go

# Investigation:
1. Review validator initialization in helpers.go:
   - How are validator keys generated?
   - Are they added to genesis?
   - Is the validator set properly configured?

2. Compare with working tests:
   - Check tests/integration setup
   - Compare genesis configuration
   - Check validator key management

3. Fix validator setup:
   - Generate proper validator keys
   - Add to genesis validators
   - Ensure proper voting power
   - Initialize priv_validator_key.json

4. Run: go test ./tests/recovery/... -run TestBasicCrashRecovery -v
5. If passing, run full suite: go test ./tests/recovery/... -v
6. Verify all recovery tests pass
```

**Dependencies**: None
**Risk**: Medium - complex setup, but well-documented pattern

---

### Task 2.2: Fix Simulation Nil Pointer Dereference

**Package**: `github.com/paw-chain/paw/tests/simulation`
**Status**: Tests failing (panic)
**Priority**: HIGH
**Estimated Time**: 2-4 hours

**Failing Tests**:
- TestMultiSeedFullSimulation/seed=2
- TestMultiSeedFullSimulation/seed=3

**Error**: 
```
panic: runtime error: invalid memory address or nil pointer dereference
[signal SIGSEGV: segmentation violation code=0x1 addr=0x18 pc=0x334e8f5]
```

**Action Plan**:
```bash
# Files to analyze:
- tests/simulation/simulation_test.go

# Investigation:
1. Get full stack trace:
   - Run: go test ./tests/simulation/... -v 2>&1 | grep -A 20 "panic"
   - Identify exact line causing panic

2. Find nil pointer:
   - Check which variable is nil at crash point
   - Trace back to initialization
   - Identify missing initialization

3. Fix initialization:
   - Add nil check before dereference
   - Properly initialize pointer
   - Add validation

4. Run: go test ./tests/simulation/... -v
5. Test with multiple seeds to ensure consistency
```

**Dependencies**: None
**Risk**: Low - straightforward nil check

---

### Task 2.3: Fix IBC Type Assertions

**Package**: `github.com/paw-chain/paw/tests/ibc`
**Status**: Tests failing
**Priority**: MEDIUM-HIGH
**Estimated Time**: 2-4 hours

**Failing Tests**:
- TestDEXCrossChainTestSuite/TestMultiHopSwap

**Error**: "Elements should be the same type" at dex_cross_chain_test.go:243

**Action Plan**:
```bash
# Files to analyze:
- tests/ibc/dex_cross_chain_test.go:243
- x/dex/ibc_module.go
- x/shared/ibc/packet.go

# Investigation:
1. Review line 243 in test:
   - What types are being compared?
   - What is the expected type?
   - What is the actual type?

2. Trace packet handling:
   - How are multi-hop packets structured?
   - Are types consistent across hops?
   - Is there a type conversion issue?

3. Fix type handling:
   - Ensure consistent packet types
   - Add proper type assertions
   - Add validation

4. Run: go test ./tests/ibc/... -run TestMultiHopSwap -v
5. Verify multi-hop swap works
6. Run full IBC test suite
```

**Dependencies**: None
**Risk**: Medium - IBC complexity, but localized issue

---

## Phase 3: Optimization & Expansion (ONGOING)

**Goal**: Improve test infrastructure and coverage
**Duration**: Ongoing
**Priority**: MEDIUM-LOW

### Task 3.1: Add Race Detection to CI

**Priority**: MEDIUM
**Estimated Time**: 2-3 hours

**Action Plan**:
```bash
# Modify CI configuration to run race detector
# Add to .github/workflows/ or equivalent

# Steps:
1. Add race detector to critical packages:
   - go test -race ./x/dex/keeper/...
   - go test -race ./x/oracle/keeper/...
   - go test -race ./x/compute/keeper/...
   - go test -race ./p2p/...

2. Configure failure notifications

3. Add to PR checks

4. Document race detector findings
```

**Dependencies**: Phase 1 complete (P2P compiles)
**Risk**: Low

---

### Task 3.2: Optimize Test Runtime

**Priority**: LOW
**Estimated Time**: 4-8 hours

**Current Runtime**: ~780 seconds (~13 minutes)
**Target Runtime**: ~300-420 seconds (~5-7 minutes)

**Action Plan**:
```bash
# Parallelization opportunities:
1. Integration tests (currently 494s):
   - Run independent test suites in parallel
   - Use t.Parallel() for safe tests
   - Split into smaller packages

2. Circuit compilation (currently 69s):
   - Cache compiled circuits
   - Load from cache in tests
   - Only compile once per test run

3. Test suite optimization:
   - Identify redundant tests
   - Combine similar test cases
   - Use table-driven tests
```

**Dependencies**: None
**Risk**: Low

---

### Task 3.3: Expand Test Coverage

**Priority**: MEDIUM
**Estimated Time**: Ongoing

**Coverage Gaps**:
1. P2P snapshot functionality (no tests)
2. Flash loan attack scenarios (expand)
3. Byzantine fault scenarios (expand)
4. Governance edge cases (expand)

**Action Plan**:
```bash
# For each gap:
1. Identify critical scenarios
2. Write test cases
3. Implement tests
4. Run and verify
5. Update coverage metrics
```

**Dependencies**: None
**Risk**: Low

---

## Task Assignments

| Phase | Task | Package | Priority | Time | Assignee |
|-------|------|---------|----------|------|----------|
| 1 | 1.1 | p2p/discovery | CRITICAL | 2-4h | TBD |
| 1 | 1.2 | p2p/protocol | CRITICAL | 2-4h | TBD |
| 1 | 1.3 | p2p/security | CRITICAL | 1-2h | TBD |
| 1 | 1.4 | p2p/reputation | HIGH | 2-4h | TBD |
| 1 | 1.5 | x/dex/keeper | HIGH | 4-6h | TBD |
| 2 | 2.1 | tests/recovery | HIGH | 4-8h | TBD |
| 2 | 2.2 | tests/simulation | HIGH | 2-4h | TBD |
| 2 | 2.3 | tests/ibc | MEDIUM-HIGH | 2-4h | TBD |
| 3 | 3.1 | CI | MEDIUM | 2-3h | TBD |
| 3 | 3.2 | Optimization | LOW | 4-8h | TBD |
| 3 | 3.3 | Coverage | MEDIUM | Ongoing | TBD |

## Dependencies Graph

```
Phase 1: All tasks can run in parallel EXCEPT:
  - Task 1.4 should wait for 1.1-1.3 (optional, for full P2P suite)

Phase 2: All tasks can run in parallel
  - No dependencies on Phase 1 (except compilation fixes)

Phase 3: Can start anytime
  - Task 3.1 should wait for Task 1.1-1.3 (P2P compilation fixed)
```

## Success Criteria

### Phase 1 Success
- [ ] All P2P packages compile successfully
- [ ] p2p/reputation tests pass (100%)
- [ ] x/dex/keeper flash loan tests pass (100%)
- [ ] No compilation errors in test suite
- [ ] Pass rate: >85%

### Phase 2 Success
- [ ] Recovery tests pass (100%)
- [ ] Simulation tests pass (100%)
- [ ] IBC tests pass (100%)
- [ ] No panics in test suite
- [ ] Pass rate: >95%

### Phase 3 Success
- [ ] Race detector integrated into CI
- [ ] Test runtime reduced to <7 minutes
- [ ] Coverage increased in identified gaps
- [ ] All tests documented

## Timeline

**Week 1** (Target: Phase 1 complete)
- Day 1-2: Tasks 1.1, 1.2, 1.3 (P2P compilation)
- Day 3: Task 1.4 (p2p/reputation)
- Day 4-5: Task 1.5 (DEX flash loans)

**Week 2** (Target: Phase 2 complete)
- Day 1-2: Task 2.1 (Recovery tests)
- Day 3: Task 2.2 (Simulation)
- Day 4: Task 2.3 (IBC)
- Day 5: Full test suite run, regression check

**Week 3+** (Target: Phase 3 ongoing)
- Optimization and expansion tasks
- Continuous improvement

## Risk Mitigation

### High Risk Tasks

**Task 1.5: DEX Flash Loan Protection**
- **Risk**: Security-critical code, errors could enable attacks
- **Mitigation**: 
  - Peer review all changes
  - Run security-focused test scenarios
  - Consider external security audit
  - Test with mainnet fork if available

**Task 2.1: Recovery Test Validator Setup**
- **Risk**: Complex setup, may uncover deeper issues
- **Mitigation**:
  - Review working examples first
  - Document setup process
  - Test incrementally (single validator first)

### Medium Risk Tasks

**Task 1.4: p2p/reputation**
- **Risk**: Logic bugs could affect peer selection
- **Mitigation**:
  - Review algorithm thoroughly
  - Test edge cases
  - Compare with reference implementations

**Task 2.3: IBC Type Assertions**
- **Risk**: IBC is complex, changes could break cross-chain
- **Mitigation**:
  - Test with actual IBC relayer
  - Verify packet serialization
  - Test on testnets before mainnet

## Rollback Plan

If a fix causes regressions:

1. **Immediate**:
   - Revert commit
   - Document issue
   - Re-analyze problem

2. **Investigation**:
   - Run full test suite on revert
   - Identify what broke
   - Determine if fix was correct

3. **Retry**:
   - Develop alternative fix
   - Test more thoroughly
   - Apply with increased review

## Monitoring & Reporting

### Daily Standup
- What was completed yesterday?
- What will be completed today?
- Any blockers?

### Weekly Status Report
- Tasks completed
- Current pass rate
- Blockers and risks
- Timeline adjustments

### Metrics to Track
- Test pass rate (%)
- Number of failing tests
- Test runtime (seconds)
- Race conditions found
- Coverage percentage

## Post-Completion Tasks

After achieving >95% pass rate:

1. **Documentation**:
   - Update test documentation
   - Document any workarounds
   - Create test writing guidelines

2. **CI/CD Integration**:
   - Enable automated test runs
   - Set up PR checks
   - Configure failure notifications

3. **Maintenance Plan**:
   - Schedule regular test reviews
   - Plan coverage expansion
   - Monitor for flaky tests

4. **Knowledge Transfer**:
   - Team training on test suite
   - Document common issues
   - Create troubleshooting guide

---

## Appendix A: Quick Reference Commands

```bash
# Run full test suite
go test ./... -v -timeout 30m 2>&1 | tee test_results.txt

# Run specific package
go test ./x/dex/keeper/... -v

# Run specific test
go test ./x/dex/keeper/... -run TestFlashLoan -v

# Run with race detector
go test -race ./x/dex/keeper/...

# Run with coverage
go test -cover ./x/dex/keeper/...

# Run parallel tests
go test ./... -parallel 8

# Clean cache (if needed)
go clean -cache -testcache
```

## Appendix B: Test Status Tracking

Create `TEST_STATUS.csv` for tracking:

```csv
Package,Status,Passing,Failing,Time,Owner,Notes
p2p/discovery,BLOCKED,0,N/A,N/A,TBD,Build failed
p2p/protocol,BLOCKED,0,N/A,N/A,TBD,Build failed
p2p/security,BLOCKED,0,N/A,N/A,TBD,Build failed
p2p/reputation,FAILING,80%,6,0.2s,TBD,Edge cases
x/dex/keeper,PARTIAL,95%,9,7.2s,TBD,Flash loans
tests/recovery,FAILING,0%,29,4.5s,TBD,Validator setup
tests/simulation,FAILING,0%,2,0.2s,TBD,Nil pointer
tests/ibc,PARTIAL,90%,1,7.0s,TBD,Type assertion
```

---

**Document Version**: 1.0
**Last Updated**: 2025-12-14
**Next Review**: After Phase 1 completion
**Owner**: Development Team Lead
**Status**: Ready for Execution
