# ZK Proof Verification Gas Exhaustion DoS

---
status: pending
priority: p1
issue_id: "004"
tags: [security, compute, zk, dos, critical]
dependencies: []
---

## Problem Statement

ZK proof verification in `x/compute/keeper/zk_verification.go:424-544` consumes gas proportionally to proof/key size BEFORE validation. Attackers can submit large invalid proofs to exhaust block gas limits, blocking legitimate compute jobs.

**Why it matters:** Economic attack - cheap to submit invalid proofs, expensive to verify them.

## Findings

### Source: security-sentinel agent

**Location:** `/home/decri/blockchain-projects/paw/x/compute/keeper/zk_verification.go:424-544`

```go
// Line 424-478: Gas consumed before actual verification
sdkCtx.GasMeter().ConsumeGas(500, "zk_proof_validation_setup")
// ...
vkGas := uint64(len(circuitParams.VerifyingKey.VkData)/32) + 1000
sdkCtx.GasMeter().ConsumeGas(vkGas, "zk_verifying_key_deserialization")
// ...
proofGas := uint64(len(zkProof.Proof)/32) + 1000
sdkCtx.GasMeter().ConsumeGas(proofGas, "zk_proof_deserialization")
// ...
// THEN verification happens at line 524
err = groth16.Verify(proof, vk, publicWitness)
```

**Attack Scenario:**
1. Attacker creates maximum-size invalid proof
2. Submits to mempool with high gas limit
3. Validation consumes significant block gas
4. Legitimate transactions fail to fit in block
5. Cost: tx fee only; Damage: blocked compute operations

## Proposed Solutions

### Option A: Upfront Deposit with Refund (Recommended)
**Pros:** Economic disincentive for attacks
**Cons:** UX friction for legitimate users
**Effort:** Medium
**Risk:** Low

```go
func (k Keeper) VerifyZKProof(ctx context.Context, proof ZKProof) error {
    sdkCtx := sdk.UnwrapSDKContext(ctx)

    // 1. SIZE CHECK BEFORE ANY GAS CONSUMPTION
    if len(proof.Proof) > MaxProofSize {
        return ErrProofTooLarge
    }

    // 2. REQUIRE DEPOSIT
    deposit := sdk.NewCoin("upaw", VerificationDepositAmount)
    if err := k.bankKeeper.SendCoinsFromAccountToModule(sdkCtx, submitter, types.ModuleName, sdk.NewCoins(deposit)); err != nil {
        return ErrInsufficientDeposit
    }

    // 3. VERIFY
    valid, err := k.verifyProofInternal(ctx, proof)

    // 4. REFUND IF VALID
    if valid {
        k.bankKeeper.SendCoinsFromModuleToAccount(sdkCtx, types.ModuleName, submitter, sdk.NewCoins(deposit))
    }
    // Invalid proofs: deposit is slashed/burned

    return err
}
```

### Option B: Rate Limiting per Address
**Pros:** Simpler, no economic mechanism
**Cons:** Sybil attack possible with multiple addresses
**Effort:** Small
**Risk:** Medium

### Option C: Proof Size Limits Only
**Pros:** Simplest fix
**Cons:** Doesn't fully prevent DoS, just limits severity
**Effort:** Small
**Risk:** Medium

## Recommended Action

**Implement Option A** with:
1. Strict proof size limits (check BEFORE gas consumption)
2. Deposit requirement for proof submission
3. Deposit refund on valid proof
4. Deposit slash on invalid proof (burned or sent to fee collector)

## Technical Details

**Affected Files:**
- `x/compute/keeper/zk_verification.go`
- `x/compute/types/params.go` (add MaxProofSize, VerificationDepositAmount)

**Database Changes:** None

## Acceptance Criteria

- [ ] Proof size validated BEFORE any gas consumption
- [ ] Maximum proof size defined in params (governance adjustable)
- [ ] Deposit mechanism implemented
- [ ] Valid proofs get deposit refunded
- [ ] Invalid proofs have deposit slashed
- [ ] Test: oversized proof rejected immediately with minimal gas
- [ ] Test: invalid proof slashes deposit
- [ ] Test: valid proof refunds deposit

## Work Log

| Date | Action | Notes |
|------|--------|-------|
| 2025-12-05 | Created | Identified by security-sentinel agent |

## Resources

- Related: ROADMAP_PRODUCTION.md CRITICAL-2 (ZK verification - may partially address this)
- Gas economics in Cosmos SDK
- Trail of Bits ZK security recommendations
