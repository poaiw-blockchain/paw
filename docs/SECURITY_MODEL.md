# PAW Blockchain Security Model

## Executive Summary

This document describes the comprehensive security architecture of the PAW blockchain, covering threat models, security assumptions, defense mechanisms, cryptographic primitives, access control, economic security, and incident response across all core modules (x/compute, x/dex, x/oracle).

**Security Philosophy**: Defense-in-depth with fail-safe defaults. The system prioritizes safety over liveness—it halts rather than accepting potentially malicious data.

---

## 1. Threat Model

### 1.1 Adversary Capabilities

PAW assumes attackers with the following capabilities:

- **Byzantine Validators**: Up to 33% of validators may be malicious or compromised (BFT assumption)
- **Network-Level Attacks**: BGP hijacking, eclipse attacks, regional network partitions
- **Economic Attacks**: MEV extraction, front-running, sandwich attacks, flash loans
- **Protocol Manipulation**: Oracle data poisoning, Sybil attacks, collusion
- **Smart Contract Exploits**: Reentrancy, overflow/underflow, precision attacks
- **DoS Attacks**: Transaction spam, state bloat, computational exhaustion

### 1.2 Attacks NOT Considered

The following are out of scope:

- **Social Engineering**: Phishing of validator keys (operational security)
- **Physical Attacks**: Hardware tampering, datacenter compromise
- **Cryptographic Breaks**: SHA-256 or secp256k1 collision attacks
- **Quantum Computing**: Post-quantum cryptography (planned for future)
- **Nation-State Censorship**: Regulatory shutdown of all validators

---

## 2. Security Assumptions

### 2.1 Cryptographic Assumptions

- **Hash Security**: SHA-256 is collision-resistant and preimage-resistant
- **Signature Security**: secp256k1 ECDSA provides 128-bit security
- **ZK Proofs**: Groth16 on BN254 curve is sound and zero-knowledge
- **Randomness**: Block hashes and VRF provide sufficient entropy

### 2.2 Network Assumptions

- **Synchrony**: Messages delivered within known time bounds (6s block time)
- **Geographic Diversity**: Minimum 3 distinct regions for validator distribution
- **IP Diversity**: Maximum 2 validators per IP address, 3 per ASN
- **Channel Security**: IBC channels are authenticated and authorized

### 2.3 Economic Assumptions

- **Rational Actors**: Validators maximize profit and avoid losses
- **Attack Cost**: Byzantine attack requires >34% stake (≥3 of 7 validators minimum)
- **Slashing Deterrence**: Slash fractions (0.01%-1%) exceed dishonest profit
- **Market Efficiency**: Oracle prices converge to real market values

---

## 3. Defense Mechanisms by Module

### 3.1 x/compute (ZK Compute Verification)

#### Threat: Malicious Computation Results

**Defense**:
- **Groth16 ZK Proofs**: All results require cryptographic proof of correct execution
- **Verifying Key Management**: Circuit verification keys stored in params (governance-controlled)
- **Gas Metering**: Proof verification costs 500,000+ gas (DoS prevention)
- **Deposit Requirement**: Providers stake tokens before registration; slashed for invalid proofs

**Implementation**: `/x/compute/keeper/verification.go`, `/proto/paw/compute/v1/zk_proof.proto`

#### Threat: Escrow Fund Theft

**Defense**:
- **Two-Phase Commit**: Atomic bank transfer + state update using CacheContext
- **Challenge Period**: 24-hour delay before release (dispute window)
- **Timeout Protection**: Automatic refund after expiry (prevents permanent lock)
- **Nonce Tracking**: Sequential nonces prevent double-spending race conditions

**Implementation**: `/x/compute/keeper/escrow.go` (lines 67-98)

**Key Code**:
```go
cacheCtx, writeFn := sdkCtx.CacheContext()
// Phase 1: Bank transfer
k.bankKeeper.SendCoinsFromAccountToModule(cacheCtx, requester, types.ModuleName, coins)
// Phase 2: Store state atomically
k.SetEscrowStateIfNotExists(cacheCtx, escrowState)
// Phase 3: Create timeout index
k.setEscrowTimeoutIndex(cacheCtx, requestID, expiresAt)
writeFn() // Atomic commit - all or nothing
```

#### Threat: Provider DoS Attacks

**Defense**:
- **Rate Limiting**: Token bucket (10 req/sec, 100 burst) per client IP
- **Circuit Breakers**: Emergency pause mechanism per provider or globally
- **Stake Requirements**: Minimum provider stake (prevents Sybil spam)
- **Reputation Decay**: Slashing on invalid submissions reduces reputation score

**Implementation**: `/x/compute/keeper/ratelimit.go`, `/x/compute/keeper/circuit_breaker.go`

---

### 3.2 x/dex (Decentralized Exchange)

#### Threat: MEV/Sandwich Attacks

**Defense**:
- **Commit-Reveal Scheme**: Large swaps (>5% of pool) require 2-step process
  - **Commit Phase**: Submit SHA256(poolID||tokenIn||tokenOut||amounts||salt||trader)
  - **Reveal Phase**: Minimum 2 blocks later, reveal preimage and execute swap
- **Deposit Mechanism**: 1M upaw deposit required for commit (returned on valid reveal)
- **Expiry Enforcement**: Commitments expire after 50 blocks; deposit forfeited to protocol

**Implementation**: `/x/dex/keeper/commit_reveal.go` (lines 94-122)

**Key Code**:
```go
func ComputeSwapCommitmentHash(poolID uint64, tokenIn, tokenOut string,
    amountIn, minAmountOut math.Int, salt []byte, trader sdk.AccAddress) []byte {
    h := sha256.New()
    h.Write(poolID bytes) || h.Write(tokenIn) || h.Write(tokenOut)
    h.Write(amountIn.String()) || h.Write(minAmountOut.String())
    h.Write(salt) || h.Write(trader.Bytes())
    return h.Sum(nil) // 32-byte commitment hash
}
```

#### Threat: Flash Loan Price Manipulation

**Defense**:
- **TWAP Oracle Integration**: Prices sourced from x/oracle (not DEX reserves)
- **Multi-Block Delay**: Oracle prices require ≥1 block between updates
- **Circuit Breakers**: 50% price deviation triggers emergency halt
- **Slippage Protection**: User-defined `minAmountOut` enforced on-chain

**Implementation**: `/x/dex/keeper/swap.go`, oracle integration in `/x/dex/keeper/oracle_integration.go`

#### Threat: Liquidity Pool Draining

**Defense**:
- **Overflow Protection**: SafeMath (cosmossdk.io/math) prevents integer overflow
- **Invariant Checks**: Constant product formula `x * y = k` enforced after every swap
- **Precision Limits**: Decimal precision capped to prevent dust attacks
- **LP Share Security**: Burns prevent share inflation exploits

**Implementation**: `/x/dex/keeper/overflow_protection.go`, `/x/dex/keeper/invariants.go`

---

### 3.3 x/oracle (Price Oracle)

#### Threat: Oracle Manipulation (Eclipse/Sybil/Collusion)

**Defense Layers**:

1. **Byzantine Fault Tolerance**
   - Minimum 7 active validators (tolerates 2 Byzantine)
   - Stake concentration limit: No validator >20% of total voting power
   - Formula: `n ≥ 3f + 1` where f = Byzantine faults

2. **Geographic Diversity**
   - Minimum 3 distinct regions (North America, Europe, Asia/Pacific)
   - GeoIP verification via MaxMind database (if available)
   - Runtime enforcement: Reject validator if region concentration >40%
   - Herfindahl-Hirschman Index (HHI) diversity score >0.40

3. **IP/ASN Diversity**
   - Maximum 2 validators per IP address (prevents datacenter concentration)
   - Maximum 3 validators per ASN (prevents single ISP control)
   - Sybil resistance via minimum stake (1M tokens) + age (1000 blocks)

4. **Flash Loan Resistance**
   - Minimum 1 block between price submissions (breaks atomicity)
   - 50% price deviation triggers circuit breaker
   - Statistical outlier detection (3σ from median)

5. **Data Quality Controls**
   - Price bounds: [0.000001, 1,000,000,000] (prevents data poisoning)
   - Staleness check: Reject data >100 blocks old (~10 minutes)
   - Median aggregation: Byzantine-resistant (no averaging)

**Implementation**: `/x/oracle/keeper/security.go` (lines 30-90), `/x/oracle/keeper/cryptoeconomics.go`

**Key Constants**:
```go
const (
    MinValidatorsForSecurity = 7            // BFT: 7 validators tolerate 2 Byzantine
    MinGeographicRegions = 3                // Region diversity: prevent nation-state control
    MinBlocksBetweenSubmissions = 1         // Flash loan prevention
    MaxDataStalenessBlocks = 100            // 10-minute max staleness
    MaxSubmissionsPerWindow = 10            // Rate limit: 10 per 100 blocks
    RateLimitWindow = 100                   // Rate limit window
)
```

#### Threat: Collusion Among Validators

**Defense**:
- **Schelling Point Mechanism**: Median price is focal point for honest coordination
- **Reputation System**: Outlier penalties reduce reputation score
- **Slashing**: 0.01%-1% slash fraction for dishonest submissions
- **Cryptoeconomic Incentives**: Nash equilibrium analysis ensures honesty is profitable

**Implementation**: `/x/oracle/keeper/cryptoeconomics.go` (lines 68-115, 226-278)

**Key Formula**:
```
Attack Expected Value = P(success) * profit - P(failure) * stake * slash_fraction
System secure when: Attack EV < 0
```

---

## 4. Cryptographic Primitives

### 4.1 Zero-Knowledge Proofs (x/compute)

- **Proving System**: Groth16 (succinct, constant-size proofs)
- **Curve**: BN254 (optimal pairing-friendly curve)
- **Circuit**: Custom circuit for compute verification
- **Proof Size**: ~200 bytes (constant, regardless of computation size)
- **Verification Gas**: ~500,000 gas (prevents DoS)
- **Public Inputs**: `requestID || resultHash || providerAddress`

**Security Properties**:
- **Soundness**: Malicious prover cannot forge valid proof
- **Zero-Knowledge**: Proof reveals nothing about private computation
- **Completeness**: Honest computation always produces valid proof

### 4.2 Commit-Reveal Hash (x/dex)

- **Hash Function**: SHA-256 (256-bit collision resistance)
- **Preimage**: `poolID || tokenIn || tokenOut || amountIn || minAmountOut || salt || trader`
- **Salt**: 32-byte random value (prevents rainbow table attacks)
- **Commitment Storage**: On-chain in module state
- **Reveal Window**: 2-50 blocks (prevents expiry DoS)

### 4.3 IBC Channel Authentication

- **Signature Scheme**: IBC packet signatures (CometBFT consensus)
- **Channel Whitelist**: Port/Channel pairs authorized via governance
- **Packet Nonces**: Sequential nonces prevent replay attacks
- **Timeout Protection**: Packets expire if not relayed within timeout

---

## 5. Access Control Model

### 5.1 Role-Based Access Control (RBAC)

| Role | Permissions | Authorization |
|------|-------------|---------------|
| **User** | Submit requests, commit swaps, query data | Any account with balance |
| **Provider** | Register, submit results, earn rewards | Staked ≥ MinProviderStake |
| **Validator** | Submit oracle prices, vote on disputes | Bonded validator set |
| **Governance** | Update params, authorize channels, emergency pause | x/gov multisig (67% approval) |
| **Circuit Breaker Admin** | Emergency halt specific module | Governance or designated admin |

### 5.2 Parameter Governance

**Mutable (via Governance)**:
- Stake requirements (`MinProviderStake`)
- Slashing fractions (`SlashFraction`)
- Timeout periods (`EscrowReleaseDelaySeconds`)
- Rate limits (`MaxSubmissionsPerWindow`)
- IBC authorized channels

**Immutable (Hard-Coded)**:
- `MinValidatorsForSecurity` (7) – changing breaks BFT guarantees
- `MinGeographicRegions` (3) – prevents nation-state censorship
- Cryptographic primitives (SHA-256, Groth16, BN254)

**Rationale**: Security-critical constants are immutable to prevent governance attacks. See `/docs/design/SECURITY_PARAMETER_GOVERNANCE.md`.

### 5.3 IBC Channel Authorization

**Mechanism**: Whitelist-based authorization for cross-chain messages

**Implementation**: `/app/ibcutil/channel_authorization.go`

**Enforcement Points**:
1. **OnRecvPacket**: Check `IsAuthorizedChannel(portID, channelID)` before processing
2. **Fail-Safe**: Return error if unauthorized (prevents malicious packets)
3. **Governance Updates**: Add channels via `AuthorizeChannel` governance proposal

**Example**:
```go
func (am AppModule) OnRecvPacket(ctx sdk.Context, packet channeltypes.Packet, ...) {
    if !ibcutil.IsAuthorizedChannel(ctx, am.keeper, packet.DestinationPort, packet.DestinationChannel) {
        return channeltypes.NewErrorAcknowledgement(types.ErrUnauthorizedChannel)
    }
    // Process packet...
}
```

---

## 6. Economic Security

### 6.1 Slashing Mechanisms

| Violation | Module | Slash Fraction | Detection |
|-----------|--------|----------------|-----------|
| Invalid ZK Proof | x/compute | 0.1% | Proof verification failure |
| Outlier Oracle Price | x/oracle | 0.01% | Statistical outlier (3σ) |
| Missed Oracle Submission | x/oracle | 0.0001% | No submission in window |
| Dispute Loss (Provider) | x/compute | 1% | Validator vote >50% against |
| Collusion (Oracle) | x/oracle | 1% | Manual governance decision |

**Total Slash Cap**: Maximum 5% per validator per day (prevents cascade failures)

### 6.2 Staking Requirements

- **Compute Provider**: Minimum 1M upaw (~$1000 equivalent at genesis)
- **Oracle Validator**: Minimum 1M tokens + bonded validator status
- **DEX Swap Commit**: 1M upaw deposit (refundable)

**Rationale**: Sybil resistance via meaningful economic stake

### 6.3 Game-Theoretic Security

**Nash Equilibrium Analysis** (x/oracle):

- **Honest Strategy EV**: `block_rewards + oracle_fees - slashing_risk`
- **Dishonest Strategy EV**: `attack_profit - stake * slash_fraction - reputation_loss`
- **Equilibrium Condition**: Honest EV ≥ Dishonest EV for all validators

**Attack Cost Calculation**:
```
Byzantine Attack Cost = 34% of total validator stake (≥3 of 7 validators)
Attack Success Probability = e^(-0.3 * num_validators)
Expected Attack Value = P(success) * profit - (1 - P(success)) * attack_cost
```

**Implementation**: `/x/oracle/keeper/cryptoeconomics.go` (lines 226-278)

---

## 7. Incident Response & Circuit Breakers

### 7.1 Circuit Breaker Triggers

**Automatic Triggers**:
1. **Oracle**: 50% price deviation in single block
2. **DEX**: Invariant violation (x * y ≠ k after swap)
3. **Compute**: >10 consecutive invalid proof submissions

**Manual Triggers** (Governance):
- Critical vulnerability discovery
- Smart contract exploit detection
- Coordinated validator attack

### 7.2 Emergency Pause Mechanism

**Scope Levels**:
- **Global**: Halt all transactions chain-wide (x/crisis)
- **Module-Level**: Pause x/compute, x/dex, or x/oracle independently
- **Provider-Level**: Blacklist specific compute provider
- **Channel-Level**: Revoke IBC channel authorization

**Recovery Process**:
1. **Detection**: Monitoring alerts or manual report
2. **Circuit Break**: Governance/admin triggers emergency pause
3. **Investigation**: Identify root cause, assess impact
4. **Mitigation**: Patch vulnerability, restore state if needed
5. **Recovery**: Resume operations after governance vote (67% approval)

**Auto-Recovery**: Oracle circuit breaker auto-recovers after 100 blocks if no new triggers

**Implementation**:
- `/x/compute/keeper/circuit_breaker.go`
- `/x/oracle/keeper/security.go` (lines 246-331)
- `/x/dex/keeper/circuit_breaker.go`

### 7.3 Rate Limiting

**Per-Client Limits** (gRPC queries):
- **Rate**: 10 requests/second
- **Burst**: 100 requests
- **Algorithm**: Token bucket with automatic cleanup

**Per-Validator Limits** (oracle submissions):
- **Global**: 10 submissions per 100 blocks
- **Per-Asset**: 5 submissions per 100 blocks
- **Enforcement**: State tracking with automatic pruning

**Implementation**: `/x/compute/keeper/ratelimit.go`, `/x/oracle/keeper/security.go` (lines 405-471)

---

## 8. Security Testing & Validation

### 8.1 Static Analysis

- **Slither**: Solidity/CosmWasm static analysis (none found – pure Go modules)
- **Go Vet**: Standard Go linting (passed)
- **Gosec**: Security-focused linting (passed with exceptions for crypto.rand)

### 8.2 Dynamic Analysis

- **Fuzzing**: Overflow protection (`/x/dex/keeper/overflow_fuzz_test.go`)
- **Property Testing**: Invariant checks (`/x/*/keeper/invariants_test.go`)
- **Integration Tests**: IBC cross-chain scenarios (`/tests/ibc/*_test.go`)

### 8.3 Formal Verification (Planned)

- **TLA+ Specification**: Escrow two-phase commit, commit-reveal scheme
- **Coq Proofs**: ZK circuit soundness, oracle aggregation correctness

---

## 9. Security Audit Trail

### 9.1 Completed Audits

**Internal Security Reviews**:
- ZK Proof Verification (x/compute) – December 2024
- Oracle Byzantine Tolerance (x/oracle) – December 2024
- Commit-Reveal MEV Protection (x/dex) – December 2024

**Findings Summary**:
- **Critical**: 0
- **High**: 2 (overflow protection, escrow race condition) – **FIXED**
- **Medium**: 4 (rate limiting, geographic diversity) – **FIXED**
- **Low**: 8 (event emissions, documentation) – **FIXED**

### 9.2 Pending External Audits

**Target Firms**:
- Trail of Bits (Smart Contract Security)
- NCC Group (Cryptographic Review)
- Informal Systems (IBC/Tendermint Security)

**Scope**: Full codebase (x/compute, x/dex, x/oracle, IBC modules)

**Budget**: $150,000 allocated (Q1 2025)

---

## 10. Operational Security

### 10.1 Key Management

- **Validator Keys**: HSM or encrypted keystore (operator responsibility)
- **Governance Keys**: Multisig (3-of-5 council members)
- **IBC Relayer Keys**: Separate from validator keys (principle of least privilege)

### 10.2 Monitoring & Alerting

**Metrics Tracked** (Prometheus/Grafana):
- Oracle diversity score (geographic, IP, ASN)
- Stake concentration (Herfindahl index)
- Circuit breaker status (active/inactive)
- Escrow balance vs locked amount
- Rate limit violations per client
- Outlier frequency per validator

**Alerting Thresholds**:
- Geographic diversity <3 regions → **CRITICAL**
- Stake concentration >20% → **HIGH**
- Circuit breaker active >1 hour → **HIGH**
- Escrow balance mismatch → **CRITICAL**

### 10.3 Incident Runbooks

Located in `/docs/operations/CIRCUIT_BREAKER_OPERATIONS.md`:
- **Oracle Manipulation Response**
- **DEX Exploit Containment**
- **Compute Provider Blacklisting**
- **Emergency Governance Proposal**

---

## 11. Compliance & Standards

### 11.1 Cosmos IBC Security (ICS)

- **ICS-20**: Token Transfer (fungible tokens)
- **ICS-27**: Interchain Accounts (cross-chain governance)
- **Channel Authorization**: Custom whitelist enforcement
- **Packet Timeout**: Prevents fund lockup on failed relays

**Compliance Status**: See `/docs/security/ICS_COMPLIANCE_CHECKLIST.md`

### 11.2 Industry Best Practices

- **OWASP Top 10**: Smart contract vulnerabilities mitigated
- **CWE Top 25**: Memory safety (Go runtime), integer overflow (SafeMath)
- **NIST Cybersecurity Framework**: Identify, Protect, Detect, Respond, Recover

---

## 12. Known Limitations & Future Work

### 12.1 Current Limitations

1. **GeoIP Accuracy**: Depends on MaxMind database (VPN/proxy evasion possible)
2. **Governance Attacks**: 67% quorum can modify most parameters
3. **Quantum Resistance**: secp256k1 vulnerable to Shor's algorithm
4. **Front-Running**: MEV still possible for swaps <5% of pool reserves

### 12.2 Roadmap

**Q1 2025**:
- External security audit (Trail of Bits)
- TLA+ formal spec for escrow mechanism
- Enhanced GeoIP with ASN database

**Q2 2025**:
- Zero-knowledge oracle prices (privacy-preserving)
- Threshold ECDSA for distributed validator keys
- MEV-resistant block building (proposer-builder separation)

**Q3-Q4 2025**:
- Post-quantum cryptography migration (Dilithium, Kyber)
- Cross-chain oracle aggregation (Chainlink, Band Protocol integration)
- Formal verification of critical paths (Coq proofs)

---

## Conclusion

The PAW blockchain implements a defense-in-depth security model with:
- **Cryptographic Guarantees**: ZK proofs, BFT consensus, commit-reveal schemes
- **Economic Incentives**: Slashing, staking, game-theoretic equilibrium
- **Operational Safeguards**: Circuit breakers, rate limiting, access control
- **Geographic Decentralization**: Multi-region validator diversity
- **Fail-Safe Defaults**: Halt on uncertainty, reject unauthorized packets

**Security Contact**: security@paw-chain.com
**Bug Bounty**: See `/docs/BUG_BOUNTY.md` (up to $50,000 for critical vulnerabilities)

---

## Appendix A: Security Parameter Reference

See `/docs/PARAMETER_REFERENCE.md` for complete list of configurable security parameters.

## Appendix B: Threat Modeling Diagrams

See `/docs/security/ATTACK_TREES.md` for visual threat models.

## Appendix C: Cryptographic Specifications

See `/docs/implementation/zk/GROTH16_INTEGRATION.md` for ZK proof details.
