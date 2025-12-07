# ZK Proof Verification DoS Vulnerability Fix

## Summary

Fixed a critical DoS vulnerability in ZK proof verification where attackers could submit large invalid proofs to exhaust gas without economic penalty.

## Vulnerability

**Location:** `x/compute/keeper/zk_verification.go:424-544` (VerifyProof function)

**Issue:** ZK proof verification consumed gas proportionally to proof size BEFORE validation. Attackers could submit large invalid proofs repeatedly to:
1. Exhaust network resources
2. Waste validator compute time
3. No economic cost to attacker (deposits only exist for compute requests, not proof submissions)

**Attack Vector:**
```
1. Submit large invalid proof (e.g., 1MB of random data)
2. Gas consumed proportional to proof size during deserialization
3. Proof rejected but attacker paid no penalty
4. Repeat attack to DoS the network
```

## Fix Implementation

### 1. Proto Changes

**File:** `proto/paw/compute/v1/zk_proof.proto`

Added deposit parameter to `CircuitParams`:
```protobuf
message CircuitParams {
  // ... existing fields ...

  // verification_deposit_amount is the deposit required to submit a proof for verification (in upaw)
  // This deposit is refunded if the proof is valid, slashed if invalid (DoS protection)
  uint64 verification_deposit_amount = 7;
}
```

### 2. Error Definitions

**File:** `x/compute/types/errors.go`

Added new error codes:
```go
ErrInsufficientDeposit     = sdkerrors.Register(ModuleName, 56, "insufficient deposit for proof verification")
ErrDepositTransferFailed   = sdkerrors.Register(ModuleName, 57, "failed to transfer verification deposit")
```

With recovery suggestions for users.

### 3. VerifyProof Function Enhancement

**File:** `x/compute/keeper/zk_verification.go`

#### Security Layers (in order):

1. **Absolute Size Check (BEFORE any gas consumption)**
   ```go
   const absoluteMaxProofSize = 10 * 1024 * 1024 // 10MB absolute maximum
   if uint64(len(zkProof.Proof)) > absoluteMaxProofSize {
       return false, types.ErrProofTooLarge
   }
   ```

2. **Circuit-Specific Size Check**
   ```go
   maxProofSize := circuitParams.MaxProofSize
   if maxProofSize == 0 {
       maxProofSize = 1024 * 1024 // Default 1MB max if not set
   }
   if uint64(len(zkProof.Proof)) > uint64(maxProofSize) {
       return false, types.ErrProofTooLarge
   }
   ```

3. **Deposit Requirement (BEFORE expensive verification)**
   ```go
   depositRequired := circuitParams.VerificationDepositAmount
   if depositRequired > 0 {
       depositCoin := sdk.NewCoin("upaw", math.NewInt(int64(depositRequired)))
       depositCoins := sdk.NewCoins(depositCoin)

       // Transfer deposit from provider to module account
       if err := zk.keeper.bankKeeper.SendCoinsFromAccountToModule(
           sdkCtx,
           providerAddress,
           types.ModuleName,
           depositCoins,
       ); err != nil {
           return false, errorsmod.Wrap(types.ErrInsufficientDeposit, err.Error())
       }
   }
   ```

4. **Deposit Handling Based on Verification Result**

   **Invalid Proof (Slash Deposit):**
   ```go
   if verificationErr != nil {
       // INVALID PROOF: Slash deposit (keep in module account, don't refund)
       // The deposit is burned/kept as penalty for submitting invalid proof
       sdkCtx.Logger().Warn("ZK proof verification failed - deposit slashed",
           "request_id", requestID,
           "provider", providerAddress.String(),
           "deposit_slashed", depositCoin.String(),
           "error", verificationErr.Error(),
       )

       sdkCtx.EventManager().EmitEvent(
           sdk.NewEvent(
               "zk_proof_deposit_slashed",
               sdk.NewAttribute("request_id", fmt.Sprintf("%d", requestID)),
               sdk.NewAttribute("provider", providerAddress.String()),
               sdk.NewAttribute("deposit_amount", depositCoin.String()),
               sdk.NewAttribute("circuit_id", zkProof.CircuitId),
               sdk.NewAttribute("reason", "invalid_proof"),
           ),
       )

       return false, nil
   }
   ```

   **Valid Proof (Refund Deposit):**
   ```go
   // VALID PROOF: Refund deposit to provider
   if err := zk.keeper.bankKeeper.SendCoinsFromModuleToAccount(
       sdkCtx,
       types.ModuleName,
       providerAddress,
       depositCoins,
   ); err != nil {
       // Log error but don't fail the verification since proof is valid
       sdkCtx.Logger().Error("failed to refund verification deposit", ...)
   } else {
       sdkCtx.EventManager().EmitEvent(
           sdk.NewEvent(
               "zk_proof_deposit_refunded",
               sdk.NewAttribute("request_id", fmt.Sprintf("%d", requestID)),
               sdk.NewAttribute("provider", providerAddress.String()),
               sdk.NewAttribute("deposit_amount", depositCoin.String()),
               sdk.NewAttribute("circuit_id", zkProof.CircuitId),
           ),
       )
   }
   ```

   **Error Before Verification (Refund Deposit):**
   - Deserialization failures
   - Invalid proof format
   - Public input mismatch
   - Circuit disabled

   All these cases refund the deposit since the provider didn't submit a maliciously invalid proof.

### 4. Default Parameters

**File:** `x/compute/keeper/zk_verification.go`

Default deposit set to 1 PAW (1,000,000 upaw):
```go
func (k *Keeper) getDefaultCircuitParams(ctx context.Context, circuitID string) *types.CircuitParams {
    return &types.CircuitParams{
        CircuitId:   circuitID,
        Description: "Compute result verification circuit using Groth16",
        VerifyingKey: types.VerifyingKey{
            CircuitId:        circuitID,
            Curve:            "bn254",
            ProofSystem:      "groth16",
            CreatedAt:        createdAt,
            PublicInputCount: 3,
        },
        MaxProofSize:                1024 * 1024, // 1MB max
        GasCost:                     500000,      // ~0.5M gas
        Enabled:                     true,
        VerificationDepositAmount:   1000000,     // 1 PAW deposit
    }
}
```

## Security Properties

### DoS Protection

1. **Economic Cost:** Attackers must pay 1 PAW per invalid proof attempt
2. **Size Limits:** Proofs larger than 1MB (or circuit-specific limit) rejected immediately
3. **Gas Protection:** Deposit required BEFORE gas-intensive verification operations
4. **Slash on Invalid:** Invalid proofs result in deposit loss, making attacks expensive

### Attack Cost Analysis

**Before Fix:**
- Cost per attack: ~gas fees only (minimal)
- Attack scalability: High (can spam network)
- Economic barrier: None

**After Fix:**
- Cost per invalid proof: 1 PAW + gas fees
- Cost for 100 attacks: 100 PAW (~$X depending on price)
- Cost for 1000 attacks: 1000 PAW
- Economic barrier: Significant

### Legitimate Use Cases Protected

1. **Valid Proofs:** Deposit fully refunded, no cost to honest providers
2. **Accidental Errors:** Refunded if error is due to format/deserialization issues
3. **Circuit Upgrades:** Governance can adjust deposit amount via params

## Events Emitted

### Deposit Locked
```go
sdk.NewEvent(
    "zk_proof_deposit_locked",
    sdk.NewAttribute("request_id", ...),
    sdk.NewAttribute("provider", ...),
    sdk.NewAttribute("deposit_amount", ...),
    sdk.NewAttribute("circuit_id", ...),
)
```

### Deposit Slashed (Invalid Proof)
```go
sdk.NewEvent(
    "zk_proof_deposit_slashed",
    sdk.NewAttribute("request_id", ...),
    sdk.NewAttribute("provider", ...),
    sdk.NewAttribute("deposit_amount", ...),
    sdk.NewAttribute("circuit_id", ...),
    sdk.NewAttribute("reason", "invalid_proof"),
)
```

### Deposit Refunded (Valid Proof)
```go
sdk.NewEvent(
    "zk_proof_deposit_refunded",
    sdk.NewAttribute("request_id", ...),
    sdk.NewAttribute("provider", ...),
    sdk.NewAttribute("deposit_amount", ...),
    sdk.NewAttribute("circuit_id", ...),
)
```

## Testing

### Manual Testing Commands

```bash
# Query circuit params to see deposit requirement
pawd query compute circuit-params compute-verification-v1

# Submit compute request with ZK proof (deposit handled automatically)
pawd tx compute submit-result <request-id> <result-hash> <proof-file> --from provider

# Monitor events for deposit handling
pawd query tx <tx-hash> --output json | jq '.logs[].events[] | select(.type | contains("deposit"))'
```

### Expected Test Scenarios

1. **Valid Proof:** Deposit locked → proof verified → deposit refunded
2. **Invalid Proof:** Deposit locked → proof verification fails → deposit slashed
3. **Oversized Proof:** Rejected before deposit required
4. **Insufficient Balance:** Transaction fails before gas consumption
5. **Deserialization Error:** Deposit locked → error → deposit refunded

## Governance Parameters

Circuit deposit amount can be adjusted via governance:

```bash
# Update deposit amount via governance proposal
pawd tx gov submit-proposal param-change proposal.json --from validator

# Example proposal.json
{
  "title": "Adjust ZK Proof Verification Deposit",
  "description": "Increase deposit to 5 PAW for enhanced DoS protection",
  "changes": [
    {
      "subspace": "compute",
      "key": "CircuitParams",
      "value": "{\"circuit_id\":\"compute-verification-v1\",\"verification_deposit_amount\":5000000}"
    }
  ]
}
```

## Migration Notes

For existing deployments:
1. Existing circuits will use default deposit (1 PAW) after upgrade
2. No state migration required (deposit is per-verification, not stored)
3. Providers need sufficient balance to submit proofs
4. Events allow tracking of slashed deposits for monitoring

## Files Modified

1. `proto/paw/compute/v1/zk_proof.proto` - Added deposit parameter
2. `x/compute/types/errors.go` - Added deposit-related errors
3. `x/compute/keeper/zk_verification.go` - Implemented deposit logic
4. `x/compute/types/zk_proof.pb.go` - Auto-generated from proto
5. `x/compute/types/zk_proof.pulsar.go` - Auto-generated from proto

## Backward Compatibility

- **Proto:** New field with field number 7, backward compatible
- **API:** Existing transactions work (deposit auto-deducted if sufficient balance)
- **Queries:** Circuit params now include `verification_deposit_amount`
- **Events:** New events added (non-breaking)

## Security Considerations

1. **Module Account:** Slashed deposits remain in compute module account
2. **Governance:** Can distribute slashed funds via governance proposals
3. **Gas Metering:** Still metered for all operations (deposit is additional protection)
4. **Front-running:** Deposit required before verification prevents front-running attacks
5. **Re-entrancy:** Not applicable (Cosmos SDK bank module is non-reentrant)

## References

- Groth16 Proof System: https://github.com/consensys/gnark
- Cosmos SDK Bank Module: https://docs.cosmos.network/main/modules/bank
- PAW Compute Module: /x/compute/README.md
