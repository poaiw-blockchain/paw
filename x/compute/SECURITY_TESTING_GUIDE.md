# Comprehensive Security Testing Guide: x/compute Module

**Purpose**: Ensure fortress-level security through systematic attack scenario testing
**Target**: 100% coverage of all identified attack vectors
**Methodology**: White-box + Black-box + Chaos testing

---

## Testing Philosophy

### Multi-Layer Testing Approach:
1. **Unit Tests**: Individual security function validation
2. **Integration Tests**: End-to-end attack scenario simulation
3. **Chaos Tests**: Random failure injection
4. **Load Tests**: Performance under attack conditions
5. **Adversarial Tests**: Malicious actor simulations

---

## Attack Scenario Test Matrix

### 1. Escrow Security Tests

#### 1.1 Double-Spending Attack
```go
func TestEscrowDoubleSpendingPrevention(t *testing.T) {
    // Setup: Create locked escrow
    // Attack: Attempt to release payment twice
    // Expected: Second release fails with "already released" error
    // Verify: Only one payment transaction executed
    // Verify: Nonce prevents duplicate release
}
```

#### 1.2 Reentrancy Attack
```go
func TestEscrowReentrancyProtection(t *testing.T) {
    // Setup: Malicious provider with reentrant contract
    // Attack: Attempt to call ReleaseEscrow recursively
    // Expected: State updated before external call
    // Expected: Subsequent calls fail due to status check
    // Verify: No double payment occurred
}
```

#### 1.3 Fund Locking Attack
```go
func TestEscrowTimeoutRefund(t *testing.T) {
    // Setup: Create request with escrow
    // Attack: Provider goes offline, never completes
    // Action: Fast-forward time past timeout
    // Action: Call ProcessExpiredEscrows
    // Expected: Automatic refund to requester
    // Verify: Funds returned, escrow status = REFUNDED
}
```

#### 1.4 Challenge Period Bypass
```go
func TestEscrowChallengePeriodEnforcement(t *testing.T) {
    // Setup: Complete request, initiate release
    // Attack: Attempt immediate release before challenge period ends
    // Expected: Error "challenge period active until..."
    // Action: Fast-forward past challenge period
    // Expected: Release succeeds
}
```

#### 1.5 State Inconsistency Attack
```go
func TestEscrowAtomicStateTransitions(t *testing.T) {
    // Setup: Multiple concurrent release attempts
    // Attack: Race condition on status update
    // Expected: Only one succeeds, others fail
    // Verify: State consistency maintained
    // Verify: No orphaned funds or states
}
```

#### 1.6 Payment Theft via Early Release
```go
func TestEscrowPreventEarlyRelease(t *testing.T) {
    // Setup: Request in ASSIGNED status
    // Attack: Attempt to release before completion
    // Expected: Error "escrow cannot be released in status ASSIGNED"
    // Verify: Funds remain locked
}
```

---

### 2. Rate Limiting Tests

#### 2.1 Burst Capacity Exhaustion
```go
func TestRateLimitBurstExhaustion(t *testing.T) {
    // Setup: Account with burst allowance of 20
    // Attack: Submit 21 requests rapidly
    // Expected: First 20 succeed, 21st fails
    // Expected: Error "burst capacity depleted"
    // Action: Wait for token refill
    // Expected: Can submit again
}
```

#### 2.2 Hourly Limit Bypass Attempt
```go
func TestRateLimitHourlyEnforcement(t *testing.T) {
    // Setup: Account with 100 requests/hour limit
    // Attack: Submit 101 requests within one hour
    // Expected: 101st request fails
    // Expected: Error "maximum 100 requests per hour reached"
    // Action: Advance time by 1 hour
    // Expected: Counter resets, can submit again
}
```

#### 2.3 Daily Limit Bypass Attempt
```go
func TestRateLimitDailyEnforcement(t *testing.T) {
    // Setup: Account with 500 requests/day limit
    // Attack: Submit 501 requests in one day
    // Expected: 501st fails
    // Action: Advance to next day
    // Expected: Counter resets
}
```

#### 2.4 Token Refill Gaming
```go
func TestRateLimitTokenRefillCorrectness(t *testing.T) {
    // Setup: Depleted token bucket
    // Attack: Repeatedly check with 1-second advances
    // Expected: Exactly 1 token per second refill
    // Verify: Cannot exceed max tokens
    // Verify: Refill calculation accurate
}
```

#### 2.5 Sybil Attack via Multiple Accounts
```go
func TestRateLimitPerAccountEnforcement(t *testing.T) {
    // Setup: Create 100 different accounts
    // Attack: Each submits 100 requests (10,000 total)
    // Expected: All succeed (limits are per-account)
    // Note: This is expected behavior - Sybil resistance
    //       comes from stake requirements, not rate limits
}
```

---

### 3. Resource Quota Tests

#### 3.1 CPU Quota Enforcement
```go
func TestQuotaCPULimitEnforcement(t *testing.T) {
    // Setup: Account with 100 CPU cores quota
    // Attack: Submit request requiring 101 cores
    // Expected: Error "CPU quota exceeded"
    // Attack: Submit 10 requests of 10 cores each
    // Expected: First 10 succeed, 11th fails
}
```

#### 3.2 Memory Quota Enforcement
```go
func TestQuotaMemoryLimitEnforcement(t *testing.T) {
    // Setup: Account with 100GB memory quota
    // Attack: Submit request requiring 101GB
    // Expected: Error "memory quota exceeded"
    // Verify: Quota tracking accurate
}
```

#### 3.3 Concurrent Request Limit
```go
func TestQuotaConcurrentRequestLimit(t *testing.T) {
    // Setup: Account with 10 concurrent request limit
    // Attack: Submit 11 concurrent requests
    // Expected: 11th fails
    // Action: Complete one request
    // Expected: Can submit another
}
```

#### 3.4 Resource Release on Failure
```go
func TestQuotaResourceReleaseOnFailure(t *testing.T) {
    // Setup: Submit request, allocate resources
    // Action: Request fails
    // Expected: Resources released back to quota
    // Verify: Quota correctly updated
}
```

#### 3.5 Quota Underflow Prevention
```go
func TestQuotaUnderflowProtection(t *testing.T) {
    // Setup: Quota with 0 current usage
    // Attack: Attempt to release more resources than allocated
    // Expected: Quota floors at 0, no underflow
    // Verify: No negative values
}
```

---

### 4. Provider Capacity Tests

#### 4.1 Provider Overload Prevention
```go
func TestProviderCapacityEnforcement(t *testing.T) {
    // Setup: Provider with 50 concurrent request capacity
    // Attack: Assign 51 requests to provider
    // Expected: 51st assignment fails
    // Expected: Error "provider at capacity"
}
```

#### 4.2 Resource Overcommitment Prevention
```go
func TestProviderResourceOvercommitmentPrevention(t *testing.T) {
    // Setup: Provider with 100 CPU cores
    // Attack: Assign requests totaling 101 cores
    // Expected: Last request fails
    // Expected: Error "provider CPU capacity exceeded"
}
```

#### 4.3 Provider Load Balancing
```go
func TestProviderLoadBasedSelection(t *testing.T) {
    // Setup: Two providers, one with 50% load, one with 10%
    // Action: Submit multiple requests
    // Expected: Lower-load provider selected more often
    // Verify: Load factor in selection score works
}
```

---

### 5. Reputation System Tests

#### 5.1 Bayesian Scoring Accuracy
```go
func TestReputationBayesianScoring(t *testing.T) {
    // Setup: New provider (no history)
    // Action: 10 successes, 0 failures
    // Expected: Reliability ≈ 0.92 (Beta(11,1) mean)
    // Action: 1 failure
    // Expected: Reliability ≈ 0.83 (Beta(11,2) mean)
    // Verify: Mathematical correctness
}
```

#### 5.2 Reputation Decay Application
```go
func TestReputationTimeDecay(t *testing.T) {
    // Setup: Provider with 1.0 reliability
    // Action: Advance time 100 days with no activity
    // Expected: Reliability decays toward 0.5
    // Verify: Decay rate = 1% per day
    // Verify: Converges to neutral
}
```

#### 5.3 Multi-Dimensional Scoring
```go
func TestReputationMultiDimensionalScoring(t *testing.T) {
    // Setup: Provider with varied performance
    // - 100% reliability (1.0)
    // - 50% accuracy (0.5)
    // - 75% speed (0.75)
    // - 80% availability (0.8)
    // Expected: Overall = 0.4×1.0 + 0.3×0.5 + 0.2×0.75 + 0.1×0.8
    //                   = 0.4 + 0.15 + 0.15 + 0.08 = 0.78 (78/100)
}
```

#### 5.4 Historical Performance Tracking
```go
func TestReputationHistoricalRecordManagement(t *testing.T) {
    // Setup: Provider with 100 performance records
    // Action: Add 1 more record
    // Expected: Oldest record dropped
    // Verify: Always exactly 100 records
    // Verify: Most recent data preserved
}
```

#### 5.5 Reputation Gaming Prevention
```go
func TestReputationGamingPrevention(t *testing.T) {
    // Setup: Malicious provider
    // Attack: Submit many trivial requests to self to boost reputation
    // Expected: Request value weighting prevents gaming
    // Expected: Decay reduces stale high reputation
    // Verify: Cannot reach perfect score easily
}
```

---

### 6. Provider Selection Tests

#### 6.1 Determinism Prevention
```go
func TestProviderSelectionRandomness(t *testing.T) {
    // Setup: 3 providers with identical reputation/stake
    // Action: Submit 100 requests
    // Expected: Non-deterministic distribution
    // Expected: Each provider gets 20-40 assignments (not 100% to one)
    // Verify: Random factor working
}
```

#### 6.2 Stake Weighting
```go
func TestProviderSelectionStakeWeighting(t *testing.T) {
    // Setup: Provider A with 10x stake of Provider B
    // Setup: Both with same reputation
    // Action: Submit 1000 requests
    // Expected: Provider A selected more frequently
    // Verify: Log-scale stake weighting applied
}
```

#### 6.3 Reputation Threshold Enforcement
```go
func TestProviderSelectionReputationThreshold(t *testing.T) {
    // Setup: Provider with 40% reputation (below 50% minimum)
    // Attack: Try to assign request to low-reputation provider
    // Expected: Provider excluded from selection
    // Expected: Error "no eligible providers"
}
```

#### 6.4 Collusion Prevention
```go
func TestProviderSelectionCollusionPrevention(t *testing.T) {
    // Setup: Colluding providers sharing information
    // Setup: Attempt to predict selection based on request ID
    // Expected: Cannot predict due to block hash entropy
    // Verify: Different blocks yield different selections
}
```

#### 6.5 Preferred Provider Override
```go
func TestProviderSelectionPreferredOverride(t *testing.T) {
    // Setup: Specify preferred provider
    // Condition: Provider is eligible
    // Expected: Preferred provider selected
    // Condition: Provider not eligible (low reputation)
    // Expected: Falls back to automatic selection
}
```

---

### 7. Verification System Tests

#### 7.1 Replay Attack Prevention
```go
func TestVerificationReplayAttackPrevention(t *testing.T) {
    // Setup: Valid verification proof with nonce N
    // Action: Submit result with proof
    // Expected: Succeeds, nonce N recorded
    // Attack: Resubmit same proof (same nonce)
    // Expected: Fails with "replay attack detected"
    // Verify: Event "replay_attack_detected" emitted
}
```

#### 7.2 Signature Verification Enforcement
```go
func TestVerificationSignatureEnforcement(t *testing.T) {
    // Setup: Result with invalid signature
    // Attack: Submit result
    // Expected: Verification score = 0 for signature
    // Expected: Overall score below threshold
    // Expected: Request marked as failed
}
```

#### 7.3 Merkle Proof Validation
```go
func TestVerificationMerkleProofValidation(t *testing.T) {
    // Setup: Valid merkle proof
    // Expected: Score += 15
    // Setup: Invalid merkle proof (wrong hash)
    // Expected: Score += 0
    // Setup: Invalid merkle proof (wrong depth)
    // Expected: Score += 0
}
```

#### 7.4 Mandatory Proof Enforcement
```go
func TestVerificationMandatoryProofEnforcement(t *testing.T) {
    // Setup: Result with empty verification proof
    // Attack: Submit without proof
    // Expected: Should fail or get 0 verification score
    // Note: Current implementation allows empty proofs (VULNERABILITY)
    // TODO: Make proof mandatory
}
```

#### 7.5 Nonce Cleanup Effectiveness
```go
func TestVerificationNonceCleanup(t *testing.T) {
    // Setup: 1000 nonces older than 7 days
    // Setup: 100 nonces newer than 7 days
    // Action: Call CleanupExpiredNonces with 7-day cutoff
    // Expected: 1000 nonces removed
    // Expected: 100 nonces remain
    // Verify: Storage size reduced
}
```

---

### 8. Integration Attack Tests

#### 8.1 Complete Attack Chain
```go
func TestFullAttackChainScenario(t *testing.T) {
    // Sophisticated multi-stage attack
    // 1. Sybil: Create multiple accounts
    // 2. Spam: Submit max allowed requests per account
    // 3. Resource exhaustion: Request max resources
    // 4. Collusion: Coordinate with malicious provider
    // 5. Verify: Attempt result forgery
    // 6. Payment: Attempt double-spending

    // Expected: Each layer catches its attack vector
    // Verify: Defense-in-depth working
}
```

#### 8.2 Race Condition Exploitation
```go
func TestConcurrentAttackRaceConditions(t *testing.T) {
    // Setup: 100 goroutines
    // Attack: Concurrent escrow release attempts
    // Attack: Concurrent quota allocations
    // Attack: Concurrent provider selections
    // Expected: No race conditions
    // Expected: Atomic operations maintained
    // Verify: No data corruption
}
```

#### 8.3 Economic Attack Simulation
```go
func TestEconomicAttackSimulation(t *testing.T) {
    // Setup: Attacker with 10% of total stake
    // Attack: Attempt to monopolize network
    // Attack: Attempt to manipulate reputation
    // Attack: Attempt to front-run requests
    // Expected: Economic incentives prevent attacks
    // Verify: Cost of attack > benefit
}
```

---

### 9. Chaos Engineering Tests

#### 9.1 Random Failure Injection
```go
func TestChaosRandomFailureInjection(t *testing.T) {
    // Setup: Normal operation
    // Chaos: Randomly fail 10% of operations
    // - Bank transfers
    // - State updates
    // - Network calls
    // Expected: Graceful degradation
    // Expected: No permanent state corruption
    // Verify: Recovery on retry
}
```

#### 9.2 Network Partition Simulation
```go
func TestChaosNetworkPartition(t *testing.T) {
    // Setup: Distributed system simulation
    // Chaos: Partition network during escrow release
    // Expected: Timeout and retry
    // Expected: No double-payment
    // Verify: Eventual consistency
}
```

#### 9.3 State Corruption Recovery
```go
func TestChaosStateCorruptionRecovery(t *testing.T) {
    // Setup: Corrupt random state entries
    // Expected: Detection of corruption
    // Expected: Error handling
    // Expected: Catastrophic failure events
    // Verify: Manual recovery path documented
}
```

---

### 10. Load & Performance Tests

#### 10.1 High Load Simulation
```go
func TestLoadHighVolumeRequests(t *testing.T) {
    // Setup: 10,000 concurrent requests
    // Measure: Rate limit overhead
    // Measure: Quota check overhead
    // Measure: Provider selection time
    // Expected: <100ms p99 latency
    // Expected: No memory leaks
}
```

#### 10.2 State Growth Analysis
```go
func TestLoadStateGrowthBounds(t *testing.T) {
    // Setup: Simulate 1 year of operation
    // - 1M requests
    // - 10K providers
    // - Historical data accumulation
    // Measure: Storage size
    // Expected: Bounded growth (nonce cleanup working)
    // Verify: No unbounded data structures
}
```

#### 10.3 Attack Under Load
```go
func TestLoadAttackUnderHighLoad(t *testing.T) {
    // Setup: System under 80% load
    // Attack: Additional malicious requests
    // Expected: Rate limiting still effective
    // Expected: No degradation of security
    // Verify: Attack detected and blocked
}
```

---

## Test Execution Strategy

### Phase 1: Unit Tests (Week 1)
- Implement all unit tests above
- Achieve 90%+ code coverage
- Fix all identified bugs
- Document edge cases

### Phase 2: Integration Tests (Week 2)
- End-to-end flow testing
- Attack scenario simulations
- Cross-module interactions
- State consistency verification

### Phase 3: Chaos Tests (Week 3)
- Random failure injection
- Network partition simulation
- State corruption scenarios
- Recovery testing

### Phase 4: Load Tests (Week 4)
- Performance benchmarking
- Scalability testing
- Memory leak detection
- Resource utilization analysis

### Phase 5: External Audit (Week 5-6)
- Third-party security audit
- Penetration testing
- Code review
- Architecture review

---

## Success Criteria

### Minimum Requirements:
- ✅ 0 critical vulnerabilities
- ✅ 0 high vulnerabilities
- ✅ 95%+ test coverage
- ✅ 100% attack scenarios tested
- ✅ 0 race conditions
- ✅ Bounded resource growth
- ✅ <100ms p99 latency
- ✅ External audit pass

### Excellence Criteria:
- ✅ 100% test coverage
- ✅ Formal verification (escrow state machine)
- ✅ Chaos engineering continuous testing
- ✅ Bug bounty program (no findings)
- ✅ Production incident: 0

---

## Continuous Security Testing

### Automated Testing:
```bash
# Run all security tests
make test-security

# Run specific attack scenarios
make test-escrow-attacks
make test-rate-limiting-attacks
make test-reputation-attacks

# Chaos testing
make test-chaos

# Load testing
make test-load

# Full security suite
make test-security-full
```

### CI/CD Integration:
- Run security tests on every commit
- Block merges with failing security tests
- Automated chaos testing nightly
- Weekly penetration testing
- Monthly security audit

---

## Incident Response

### If Vulnerability Found:
1. **Halt**: Stop testnet/mainnet if critical
2. **Assess**: Determine severity and impact
3. **Patch**: Implement fix
4. **Test**: Full security test suite
5. **Audit**: External review of fix
6. **Deploy**: Coordinated upgrade
7. **Disclose**: Responsible disclosure

### Post-Incident:
- Root cause analysis
- Test case addition
- Documentation update
- Architecture review
- Prevention measures

---

## Conclusion

This testing guide provides a **comprehensive framework** for validating the security of the x/compute module. Systematic execution of these tests will ensure **fortress-level security** suitable for enterprise deployment.

**Next Steps**:
1. Implement all test scenarios
2. Achieve 100% coverage
3. Fix all discovered issues
4. External security audit
5. Production deployment

**Security is not a feature, it's a process.**
