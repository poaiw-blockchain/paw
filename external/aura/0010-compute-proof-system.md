# RFC-0010: Compute Proof System

- **Author(s):** Compute & Security Team
- **Status:** Draft
- **Created:** 2025-11-12
- **Target Release:** Community Testnet

## Summary

Define a cryptographically verifiable compute proof system combining TEE (Trusted Execution Environment) attestations as the primary mechanism, optimistic fraud proofs as fallback enforcement, and ZK proofs for future privacy-preserving computation. This enables trustless AI task execution with verifiable results while maintaining economic security through challenge-response mechanisms.

## Motivation & Goals

- Enable verifiable off-chain compute for AI tasks without requiring users to trust individual compute agents.
- Leverage hardware-based security (TEE) for efficient, real-time verification.
- Provide economic deterrents through fraud proof challenges and slashing.
- Support future privacy-preserving computation via zero-knowledge proofs.
- Integrate with existing fraud detection infrastructure for proactive security.

## Detailed Design

### Proof Types

#### 1. TEE Attestation (Primary)

Remote attestation from Intel SGX, AMD SEV-SNP, or AWS Nitro Enclaves provides cryptographic proof that code executed in a verified enclave.

**Quote Structure:**
```
{
  "task_id": "uuid-v4",
  "input_hash": "sha256(task_input)",
  "output_hash": "sha256(task_output)",
  "timestamp": "unix_timestamp_ms",
  "enclave_measurement": "sha256(enclave_code)",
  "nonce": "random_32_bytes",
  "platform": "sgx|sev|nitro"
}
```

**Attestation Components:**
- **Quote**: Platform-specific attestation report (DCAP quote for SGX, attestation document for Nitro)
- **Signature**: Signed by TEE platform root key
- **Certificate Chain**: Validates signing key back to manufacturer root-of-trust
- **Measurement**: Hash of enclave binary, verified against on-chain whitelist

**Security Properties:**
- Freshness: Nonce prevents replay attacks
- Integrity: Measurement ensures correct code execution
- Authenticity: Signature chain validates genuine TEE hardware

#### 2. Fraud Proof (Fallback)

Challenge mechanism for disputing incorrect results when TEE attestation is unavailable or suspected compromised.

**Challenge Period:** 24 hours from proof submission

**Challenge Workflow:**
1. Challenger stakes bond (1,000 PAW minimum)
2. Challenger submits claim: `{task_id, correct_output_hash, evidence_url}`
3. Arbitration period: 72 hours for resolution
4. Resolution via:
   - Re-execution in trusted TEE by validator set
   - Deterministic re-computation on-chain (for simple tasks)
   - Governance vote (for edge cases)

**Slashing:**
- If fraud proven: Compute agent loses stake (minimum 10,000 PAW)
- If frivolous challenge: Challenger loses bond
- Slashed funds: 50% to challenger, 50% to community pool

**Challenge Bond Calculation:**
```
bond = max(1000, task_fee * 10)
```

#### 3. ZK Proof (Phase 2)

Zero-knowledge proofs for privacy-preserving compute, enabling verification without revealing inputs/outputs.

**Use Cases:**
- Private ML inference
- Confidential data processing
- Regulatory compliance (GDPR, HIPAA)

**Proof System:** PlonK/Groth16 via gnark or similar, specifics deferred to Phase 2 RFC.

### Task Lifecycle

```
┌──────────────┐
│ 1. Submit    │  User posts task + fee to mempool
└──────┬───────┘
       │
       ▼
┌──────────────┐
│ 2. Assignment│  Compute agent claims task (stake verified)
└──────┬───────┘
       │
       ▼
┌──────────────┐
│ 3. Execution │  Agent runs in TEE, generates attestation
└──────┬───────┘
       │
       ▼
┌──────────────┐
│ 4. Proof Sub │  Attestation + output posted on-chain
└──────┬───────┘
       │
       ▼
┌──────────────┐
│ 5. Challenge │  24-hour challenge period
│    Period    │  (fraud detector monitors)
└──────┬───────┘
       │
       ▼
┌──────────────┐
│ 6. Settlement│  Fee released if no successful challenges
└──────────────┘
```

### On-Chain Verification

**Smart Contract Verification Steps:**

1. **Signature Verification**
   - Extract attestation signature from quote
   - Verify against TEE platform certificate chain
   - Check certificate validity (not expired/revoked)

2. **Enclave Measurement Validation**
   - Extract `enclave_measurement` from quote
   - Compare against on-chain whitelist: `approved_enclaves[]`
   - Reject if not whitelisted

3. **Freshness Check**
   - `current_time - timestamp < 300` (5 minutes)
   - Prevents replay attacks

4. **Task Integrity**
   - Verify `task_id` matches on-chain task
   - Confirm `input_hash` matches submitted task input
   - Validate agent has sufficient stake

5. **Output Storage**
   - Store `output_hash` on-chain
   - Output content stored on IPFS/Arweave (referenced by hash)

**Verification Pseudocode:**
```python
def verify_tee_attestation(task_id, attestation):
    # 1. Extract quote and signature
    quote = decode_attestation(attestation.quote)

    # 2. Verify signature chain
    if not verify_certificate_chain(
        attestation.certificate_chain,
        PLATFORM_ROOT_KEYS[quote.platform]
    ):
        return False

    # 3. Verify quote signature
    if not verify_signature(
        quote,
        attestation.signature,
        attestation.certificate_chain[-1]
    ):
        return False

    # 4. Check enclave measurement
    if quote.enclave_measurement not in APPROVED_ENCLAVES:
        return False

    # 5. Freshness check
    if block.timestamp - quote.timestamp > 300:
        return False

    # 6. Task integrity
    task = load_task(task_id)
    if sha256(task.input) != quote.input_hash:
        return False

    # 7. Nonce uniqueness (prevent replay)
    if quote.nonce in used_nonces:
        return False
    used_nonces.add(quote.nonce)

    return True
```

### Fraud Detection Integration

**Automated Monitoring via `fraud_detector.py`:**

```python
# fraud_detector.py integration points

class ComputeProofMonitor:
    def analyze_proof(self, task_id, proof):
        """Analyze submitted proof for fraud indicators"""
        risk_score = 0.0

        # Agent behavior analysis
        agent_history = self.get_agent_history(proof.agent_address)
        if agent_history.failure_rate > 0.1:
            risk_score += 0.3

        # Output plausibility check
        if self.is_output_implausible(task, proof.output_hash):
            risk_score += 0.4

        # Timing analysis (too fast = suspicious)
        expected_time = self.estimate_compute_time(task)
        actual_time = proof.timestamp - task.submitted_at
        if actual_time < expected_time * 0.1:
            risk_score += 0.3

        # Pattern detection
        if self.detect_copy_paste(proof.output_hash):
            risk_score += 0.5

        return risk_score

    def auto_challenge(self, task_id, proof, risk_score):
        """Automatically challenge if high confidence fraud"""
        if risk_score > 0.9:
            # Post challenge with protocol funds
            return submit_fraud_challenge(
                task_id=task_id,
                evidence=self.generate_evidence(proof),
                challenger=FRAUD_DETECTOR_ADDRESS
            )
```

**Alert Thresholds:**
- `risk_score > 0.9`: Auto-challenge
- `risk_score > 0.7`: Flag for manual review
- `risk_score > 0.5`: Enhanced monitoring

### Proof Format

**Complete Proof Submission:**
```json
{
  "version": "1.0",
  "task_id": "550e8400-e29b-41d4-a716-446655440000",
  "agent_address": "paw1qy3rkz6t9s4lvw6fjhmxjxz5sxvj0xq9",
  "attestation": {
    "platform": "sgx",
    "quote": "AwACAAAAAAAKAA...(base64)",
    "signature": "MEUCIQDx7...(base64)",
    "certificate_chain": [
      "MIIEoTCCA...(base64)",
      "MIICjzCCA...(base64)"
    ]
  },
  "task_data": {
    "input_hash": "e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855",
    "output_hash": "ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad",
    "parameters": {
      "model": "llama-3.1-70b",
      "max_tokens": 1000,
      "temperature": 0.7
    }
  },
  "output_url": "ipfs://QmYwAPJzv5CZsnA636s8L4Xs1nN1eGN3N",
  "timestamp": 1731427200000,
  "nonce": "c7a8f9e2d1b3a4f5e6d7c8b9a0f1e2d3"
}
```

### Enclave Governance

**Whitelist Management:**
- Initial whitelist: Governance proposal with security audit
- Updates: 7-day voting period, 66% threshold
- Emergency removal: Validator set can remove compromised enclaves (50% threshold + 24hr timelock)

**Enclave Metadata:**
```json
{
  "measurement": "sha256...",
  "version": "1.2.0",
  "audit_report": "ipfs://...",
  "approved_date": "2025-11-01",
  "expiry_date": "2026-11-01",
  "min_stake": 10000000000
}
```

## Security Considerations

### Threat Model

**1. TEE Side-Channel Attacks**
- **Spectre/Meltdown variants**: Require enclave updates when disclosed
- **Cache timing attacks**: Enclave code must use constant-time operations
- **Mitigation**: Regular security audits, rapid patch deployment via governance

**2. Enclave Compromise**
- **Supply chain attacks**: Certificate chain validation prevents unauthorized enclaves
- **Extraction attacks**: Physical security assumed (agents responsible for hardware security)
- **Mitigation**: Multiple TEE platform support reduces single point of failure

**3. Attestation Replay**
- **Attack**: Resubmit old attestation for new task
- **Mitigation**: Nonce uniqueness check, timestamp freshness, task_id binding

**4. Economic Attacks**

**Sybil Attacks:**
- Create many agents, accept tasks, submit invalid proofs
- **Mitigation**: Minimum stake (10,000 PAW), reputation system

**Griefing Attacks:**
- Spam frivolous challenges to tie up compute agents
- **Mitigation**: Challenge bonds, slashing for false challenges

**Collusion:**
- Agent + challenger collude to split slashed funds
- **Mitigation**: Randomized validator re-execution, reputation tracking

### Attestation Freshness

**Time Synchronization:**
- Agents must sync with NTP servers
- Acceptable drift: ±30 seconds
- Verification uses block timestamp (BFT consensus time)

**Nonce Management:**
- 32-byte cryptographically random nonce per task
- On-chain storage: Bloom filter for recent nonces (last 10,000 tasks)
- Prevents replay while bounding state growth

### Privacy Leakage

**Attestation Metadata:**
- Task parameters may reveal sensitive info
- **Mitigation**: Support encrypted task inputs (Phase 2)
- Output stored off-chain; only hash on-chain

**Agent Tracking:**
- Agent address linkable across tasks
- **Mitigation**: Support key rotation, privacy-preserving agent selection (Phase 2)

## Validation Plan

### Phase 1: Testnet

**Week 1-2: Verification Logic Audit**
- External security review of on-chain verification contract
- Focus: Signature validation, nonce management, timestamp handling
- Deliverable: Audit report with remediation plan

**Week 3-4: TEE Pentesting**
- Side-channel attack simulation
- Enclave binary reverse engineering attempts
- Physical security assessment (if applicable)
- Deliverable: Pentest report, updated enclave code

**Week 5-6: Economic Attack Simulation**
- Sybil attack with low-stake agents
- Frivolous challenge spam
- Collusion scenarios
- Deliverable: Parameter tuning recommendations

### Phase 2: Mainnet Preparation

**Integration Testing:**
- 100+ tasks across different agent implementations
- Verify fraud detector auto-challenge accuracy
- Test enclave update governance flow

**Performance Benchmarks:**
- Verification gas costs (target: <100k gas per attestation)
- Challenge resolution time (target: <24 hours median)
- Fraud detection false positive rate (target: <1%)

**Chaos Engineering:**
- Network partitions during proof submission
- Validator Byzantine behavior
- TEE platform certificate expiry

## Backwards Compatibility

- New proof module; no breaking changes to existing modules
- Fraud detector updated to integrate compute proof monitoring
- Existing identity/VC modules unaffected

## Open Questions

1. **Multi-TEE Support**: Should we require proofs from multiple TEE types (e.g., SGX + Nitro) for high-value tasks?
2. **Stake Scaling**: Should minimum stake scale with task value? Formula: `min_stake = max(10000, task_fee * 100)`?
3. **Proof Aggregation**: Can we batch-verify multiple attestations to reduce gas costs?
4. **Enclave Diversity**: Minimum number of approved enclave versions to prevent centralization?
5. **Challenge Resolution**: Should we support multi-tier arbitration (validator re-execution, then governance)?

## References

- Intel SGX Remote Attestation: https://software.intel.com/content/www/us/en/develop/topics/software-guard-extensions.html
- AWS Nitro Enclaves: https://aws.amazon.com/ec2/nitro/nitro-enclaves/
- Optimistic Rollup Fraud Proofs: Arbitrum/Optimism documentation
- TEE Security Research: https://github.com/ayeks/SGX-hardware
