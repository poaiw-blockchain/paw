# ADR-006: Compute Verification & ZK Proofs

## Status
Accepted

## Context
Off-chain compute providers execute arbitrary workloads. Verification ensures:
- Correct execution
- No result manipulation
- Provider accountability

## Decision

### Verification Layers

1. **Ed25519 Signatures**: Provider signs results
   - All 8 low-order points rejected (small subgroup attack prevention)
   - Explicit key registration required (no trust-on-first-use)

2. **Groth16 ZK-SNARK Proofs**: Optional cryptographic verification
   - Proves execution without revealing computation
   - BN254 curve for efficient pairing operations

3. **Deterministic Execution Traces**: Hash-based verification
   - Output hash matches expected computation
   - Execution trace provides audit trail

### Nonce Management (Replay Prevention)
```go
// Two-phase nonce pattern:
// 1. Reserve nonce BEFORE verification
// 2. Mark as "used" AFTER verification completes
NonceStatusReserved = 0x01
NonceStatusUsed     = 0x02
```

### Provider Registration
- Providers must call `RegisterSigningKey()` before submitting results
- Key updates require proof of old key ownership
- Active provider status validated on each submission

### Dispute Resolution
- Requesters can file disputes with deposit
- Validators vote on disputes
- Automated slashing on verified misbehavior

## Consequences

**Positive:**
- Multiple verification options (speed vs. security)
- Replay attacks impossible
- Clear accountability chain

**Negative:**
- ZK proof verification is gas-intensive
- Provider onboarding requires explicit registration

## References
- [Groth16 Paper](https://eprint.iacr.org/2016/260)
- [Ed25519 Low-Order Points](https://cr.yp.to/ecdh/curve25519-20060209.pdf)
