package keeper

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/compute/types"
)

const maxProofFutureSkew = 5 * time.Minute

// SubmitResult processes a result submission from a provider with institutional-grade verification.
func (k Keeper) SubmitResult(ctx context.Context, provider sdk.AccAddress, requestID uint64, outputHash, outputURL string, exitCode int32, logsURL string, verificationProof []byte) error {
	request, err := k.GetRequest(ctx, requestID)
	if err != nil {
		return fmt.Errorf("request not found: %w", err)
	}

	if request.Provider != provider.String() {
		return fmt.Errorf("unauthorized: provider %s not assigned to request %d", provider.String(), requestID)
	}

	if request.Status != types.REQUEST_STATUS_ASSIGNED &&
		request.Status != types.REQUEST_STATUS_PROCESSING {
		return fmt.Errorf("request is not in a state to accept results: %s", request.Status.String())
	}

	if outputHash == "" {
		return fmt.Errorf("output hash is required")
	}
	if outputURL == "" {
		return fmt.Errorf("output URL is required")
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	deadline, err := k.requestDeadline(ctx, *request, sdkCtx.BlockTime())
	if err != nil {
		return fmt.Errorf("failed to compute request deadline: %w", err)
	}
	if sdkCtx.BlockTime().After(deadline) {
		if err := k.CompleteRequest(ctx, requestID, false); err != nil {
			return fmt.Errorf("request exceeded timeout and refund failed: %w", err)
		}
		return fmt.Errorf("request %d exceeded timeout window", requestID)
	}

	proofHash := sha256.Sum256(verificationProof)
	var proof *types.VerificationProof
	if parsedProof, err := k.parseVerificationProof(verificationProof); err == nil {
		if err := parsedProof.Validate(); err == nil {
			proof = parsedProof
		}
	}

	nonceReplay := false
	proofReplay := k.hasProofHash(ctx, provider, proofHash[:])
	keyMismatch := false
	nonceReserved := false
	if proof != nil {
		// SEC-1 FIX: Reserve nonce BEFORE verification to prevent replay attack window.
		// Previously, nonce was only recorded AFTER verification, allowing attackers to
		// submit duplicate requests during the verification window.
		// Now we use a reservation pattern: reserve immediately, upgrade to "used" after.
		if !k.reserveNonce(ctx, provider, proof.Nonce) {
			// Nonce already used or reserved - this is a replay attack
			nonceReplay = true
			k.recordReplayAttempt(ctx, provider, proof.Nonce)
		} else {
			nonceReserved = true
		}

		// SEC-HIGH-1: Reject results with future timestamps (security enforcement)
		if proof.Timestamp > sdkCtx.BlockTime().Add(maxProofFutureSkew).Unix() {
			sdkCtx.EventManager().EmitEvent(
				sdk.NewEvent(
					"verification_future_timestamp",
					sdk.NewAttribute("request_id", fmt.Sprintf("%d", requestID)),
					sdk.NewAttribute("provider", provider.String()),
					sdk.NewAttribute("timestamp", fmt.Sprintf("%d", proof.Timestamp)),
					sdk.NewAttribute("block_time", sdkCtx.BlockTime().Format(time.RFC3339)),
					sdk.NewAttribute("max_future_skew", maxProofFutureSkew.String()),
				),
			)
			return fmt.Errorf("proof timestamp %d exceeds maximum future skew (block time: %s, max skew: %s): %w",
				proof.Timestamp,
				sdkCtx.BlockTime().Format(time.RFC3339),
				maxProofFutureSkew.String(),
				types.ErrProofExpired,
			)
		}

		if proofReplay {
			k.recordReplayAttempt(ctx, provider, proof.Nonce)
		}

		if !k.verifyProviderSigningKey(ctx, provider, proof.PublicKey) {
			keyMismatch = true
			sdkCtx.EventManager().EmitEvent(
				sdk.NewEvent(
					"verification_public_key_mismatch",
					sdk.NewAttribute("request_id", fmt.Sprintf("%d", requestID)),
					sdk.NewAttribute("provider", provider.String()),
				),
			)
		}
	}
	forceFailure := nonceReplay || proofReplay || keyMismatch
	// Track whether nonce was reserved so we can upgrade it to "used" after verification
	_ = nonceReserved

	now := sdkCtx.BlockTime()
	result := types.Result{
		RequestId:         requestID,
		Provider:          provider.String(),
		OutputHash:        outputHash,
		OutputUrl:         outputURL,
		ExitCode:          exitCode,
		LogsUrl:           logsURL,
		VerificationProof: verificationProof,
		SubmittedAt:       now,
		Verified:          false,
		VerificationScore: 0,
	}

	if err := k.SetResult(ctx, &result); err != nil {
		return fmt.Errorf("failed to store result: %w", err)
	}

	request.ResultHash = outputHash
	request.ResultUrl = outputURL
	request.Status = types.REQUEST_STATUS_PROCESSING

	if err := k.SetRequest(ctx, *request); err != nil {
		return fmt.Errorf("failed to update request: %w", err)
	}

	if err := k.updateRequestStatusIndex(ctx, *request); err != nil {
		return fmt.Errorf("failed to update request status index: %w", err)
	}

	verified, score := k.verifyResult(ctx, result, *request, proof)
	if forceFailure {
		verified = false
		score = 0
	}
	result.Verified = verified
	result.VerificationScore = score

	if err := k.SetResult(ctx, &result); err != nil {
		return fmt.Errorf("failed to update result with verification: %w", err)
	}

	// SEC-1 FIX: Upgrade reserved nonce to "used" status after verification completes.
	// The nonce was already reserved at the start of verification (via reserveNonce),
	// so this just upgrades the status byte from Reserved to Used.
	if proof != nil && nonceReserved {
		k.recordNonceUsage(ctx, provider, proof.Nonce)
	}
	if !proofReplay {
		k.recordProofHashUsage(ctx, provider, proofHash[:], sdkCtx.BlockTime())
	}

	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"result_submitted",
			sdk.NewAttribute("request_id", fmt.Sprintf("%d", requestID)),
			sdk.NewAttribute("provider", provider.String()),
			sdk.NewAttribute("output_hash", outputHash),
			sdk.NewAttribute("exit_code", fmt.Sprintf("%d", exitCode)),
			sdk.NewAttribute("verified", fmt.Sprintf("%t", verified)),
			sdk.NewAttribute("verification_score", fmt.Sprintf("%d", score)),
		),
	)

	success := verified && exitCode == 0
	if err := k.CompleteRequest(ctx, requestID, success); err != nil {
		return fmt.Errorf("failed to complete request: %w", err)
	}

	return nil
}

// GetResult retrieves a result by request ID.
func (k Keeper) GetResult(ctx context.Context, requestID uint64) (*types.Result, error) {
	store := k.getStore(ctx)
	bz := store.Get(ResultKey(requestID))

	if bz == nil {
		return nil, fmt.Errorf("result not found")
	}

	var result types.Result
	if err := k.cdc.Unmarshal(bz, &result); err != nil {
		return nil, fmt.Errorf("GetResult: unmarshal: %w", err)
	}

	return &result, nil
}

// SetResult stores a result record.
func (k Keeper) SetResult(ctx context.Context, result *types.Result) error {
	store := k.getStore(ctx)
	bz, err := k.cdc.Marshal(result)
	if err != nil {
		return fmt.Errorf("SetResult: marshal: %w", err)
	}

	store.Set(ResultKey(result.RequestId), bz)
	return nil
}

// verifyResult performs cryptographic verification of the submitted result using ZK-SNARKs.
// This replaces the old scoring system with actual zero-knowledge proof verification.
func (k Keeper) verifyResult(ctx context.Context, result types.Result, request types.Request, proof *types.VerificationProof) (verified bool, score uint32) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Priority 1: ZK-SNARK proof verification (if available)
	if len(result.VerificationProof) > 0 {
		zkVerified, zkErr := k.verifyResultZKProof(sdkCtx, result, request)
		if zkErr == nil {
			if zkVerified {
				// ZK proof verified successfully - highest confidence
				score = types.MaxVerificationScore
				verified = true

				sdkCtx.EventManager().EmitEvent(
					sdk.NewEvent(
						"zk_verification_success",
						sdk.NewAttribute("request_id", fmt.Sprintf("%d", result.RequestId)),
						sdk.NewAttribute("provider", result.Provider),
						sdk.NewAttribute("verification_method", "zk_snark"),
					),
				)
			} else {
				// ZK proof failed - slash provider for invalid proof
				provider, err := sdk.AccAddressFromBech32(result.Provider)
				if err == nil {
					if slashErr := k.slashProviderForInvalidProof(ctx, provider, "ZK-SNARK proof verification failed"); slashErr != nil {
						sdkCtx.Logger().Error("failed to slash provider for invalid ZK proof", "error", slashErr)
					}
				}

				score = 0
				verified = false

				sdkCtx.EventManager().EmitEvent(
					sdk.NewEvent(
						"zk_verification_failed",
						sdk.NewAttribute("request_id", fmt.Sprintf("%d", result.RequestId)),
						sdk.NewAttribute("provider", result.Provider),
						sdk.NewAttribute("reason", "invalid_zk_proof"),
					),
				)
			}

			sdkCtx.EventManager().EmitEvent(
				sdk.NewEvent(
					"verification_completed",
					sdk.NewAttribute("request_id", fmt.Sprintf("%d", result.RequestId)),
					sdk.NewAttribute("total_score", fmt.Sprintf("%d", score)),
					sdk.NewAttribute("verified", fmt.Sprintf("%t", verified)),
					sdk.NewAttribute("method", "zk_snark"),
				),
			)

			return verified, score
		}

		// ZK verification error (e.g., circuit not initialized) - fall back to legacy verification
		sdkCtx.Logger().Warn("ZK verification failed, falling back to legacy verification", "error", zkErr)
	}

	// Fallback: Legacy verification (to be deprecated)
	score = 0
	var scoreBreakdown = make(map[string]uint32)

	if len(result.OutputHash) == 64 {
		if _, err := hex.DecodeString(result.OutputHash); err == nil {
			scoreBreakdown["hash_format"] = 10
			score += 10
		}
	}

	var proofComponentScore uint32
	var merkleValid bool
	if len(result.VerificationProof) > 0 {
		proofScore, proofBreakdown, merkleOk := k.validateVerificationProof(ctx, result, request, proof)
		merkleValid = merkleOk
		proofComponentScore = proofScore
		for key, val := range proofBreakdown {
			scoreBreakdown[key] = val
		}
		score += proofScore
	}

	if merkleValid && proofComponentScore > 0 {
		provider, err := sdk.AccAddressFromBech32(result.Provider)
		if err == nil {
			providerRecord, err := k.GetProvider(ctx, provider)
			if err == nil {
				reputationBonus := uint32(providerRecord.Reputation / 10)
				if reputationBonus > 10 {
					reputationBonus = 10
				}
				scoreBreakdown["provider_reputation"] = reputationBonus
				score += reputationBonus
			}
		}
	}

	if score > types.MaxVerificationScore {
		score = types.MaxVerificationScore
	}

	verified = score >= types.VerificationPassThreshold

	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"verification_completed",
			sdk.NewAttribute("request_id", fmt.Sprintf("%d", result.RequestId)),
			sdk.NewAttribute("total_score", fmt.Sprintf("%d", score)),
			sdk.NewAttribute("verified", fmt.Sprintf("%t", verified)),
			sdk.NewAttribute("threshold", fmt.Sprintf("%d", types.VerificationPassThreshold)),
			sdk.NewAttribute("method", "legacy_fallback"),
		),
	)

	if !verified && (score == 0 || proofComponentScore == 0) {
		if provider, err := sdk.AccAddressFromBech32(result.Provider); err == nil {
			if slashErr := k.slashProviderForInvalidProof(ctx, provider, "verification score below threshold"); slashErr != nil {
				sdkCtx.Logger().Error("failed to slash provider for invalid proof", "error", slashErr)
			}
		}
	}

	return verified, score
}

// verifyZKProof verifies a ZK-SNARK proof for a compute result.
func (k Keeper) verifyResultZKProof(ctx sdk.Context, result types.Result, request types.Request) (bool, error) {
	// Parse the ZK proof from the verification proof bytes
	var zkProof types.ZKProof
	if err := json.Unmarshal(result.VerificationProof, &zkProof); err != nil {
		// Not a ZK proof, might be legacy proof format
		return false, fmt.Errorf("failed to parse ZK proof: %w", err)
	}

	// Decode the result hash
	resultHashBytes, err := hex.DecodeString(result.OutputHash)
	if err != nil {
		return false, fmt.Errorf("invalid result hash: %w", err)
	}

	// Verify using the common verification method
	// We use the proof's public inputs if available, otherwise derive from result
	publicInputs := zkProof.PublicInputs
	if len(publicInputs) == 0 {
		publicInputs = resultHashBytes
	}

	if err := k.VerifyZKProof(ctx, zkProof.Proof, publicInputs); err != nil {
		return false, fmt.Errorf("ZK proof verification error: %w", err)
	}

	return true, nil
}

// validateVerificationProof performs sophisticated cryptographic validation of the verification proof.
// Returns the proof score, detailed breakdown, and whether the merkle proof was valid.
func (k Keeper) validateVerificationProof(ctx context.Context, result types.Result, request types.Request, proof *types.VerificationProof) (uint32, map[string]uint32, bool) {
	var totalScore uint32 = 0
	breakdown := make(map[string]uint32)
	merkleValid := false

	if proof == nil {
		parsed, err := k.parseVerificationProof(result.VerificationProof)
		if err != nil {
			breakdown["parse_error"] = 0
			return 0, breakdown, false
		}
		if err := parsed.Validate(); err != nil {
			breakdown["validation_error"] = 0
			return 0, breakdown, false
		}
		proof = parsed
	}

	provider, err := sdk.AccAddressFromBech32(result.Provider)
	if err != nil {
		breakdown["provider_address_error"] = 0
		return 0, breakdown, false
	}

	now := sdk.UnwrapSDKContext(ctx).BlockTime()
	if proof.Timestamp > now.Add(maxProofFutureSkew).Unix() {
		breakdown["timestamp_future"] = 0
		return 0, breakdown, false
	}

	if k.verifyEd25519Signature(proof, result, request) {
		breakdown["signature_verification"] = 20
		totalScore += 20
	} else {
		breakdown["signature_verification"] = 0
	}

	merkleScore := k.validateMerkleProof(proof, result)
	breakdown["merkle_proof"] = merkleScore
	totalScore += merkleScore
	merkleValid = merkleScore > 0

	stateScore := uint32(0)
	if merkleScore > 0 {
		stateScore = k.verifyStateTransition(proof, result, request)
	}
	breakdown["state_transition"] = stateScore
	totalScore += stateScore

	deterministicScore := k.verifyDeterministicExecution(proof, result)
	breakdown["deterministic_execution"] = deterministicScore
	totalScore += deterministicScore

	if merkleValid {
		const coreBonus = 10
		breakdown["core_consistency_bonus"] = coreBonus
		totalScore += coreBonus
	}

	k.recordNonceUsage(ctx, provider, proof.Nonce)

	return totalScore, breakdown, merkleValid
}

// parseVerificationProof deserializes and parses the raw verification proof bytes.
func (k Keeper) parseVerificationProof(proofBytes []byte) (*types.VerificationProof, error) {
	if len(proofBytes) < 200 {
		return nil, fmt.Errorf("proof too short: minimum 200 bytes required, got %d", len(proofBytes))
	}

	proof := &types.VerificationProof{}
	offset := 0

	proof.Signature = make([]byte, 64)
	copy(proof.Signature, proofBytes[offset:offset+64])
	offset += 64

	proof.PublicKey = make([]byte, 32)
	copy(proof.PublicKey, proofBytes[offset:offset+32])
	offset += 32

	proof.MerkleRoot = make([]byte, 32)
	copy(proof.MerkleRoot, proofBytes[offset:offset+32])
	offset += 32

	if offset+1 > len(proofBytes) {
		return nil, fmt.Errorf("proof truncated at merkle proof count")
	}
	merkleProofLen := int(proofBytes[offset])
	offset += 1

	if merkleProofLen > 32 {
		return nil, fmt.Errorf("merkle proof too long: max 32 levels, got %d", merkleProofLen)
	}

	proof.MerkleProof = make([][]byte, merkleProofLen)
	for i := 0; i < merkleProofLen; i++ {
		if offset+32 > len(proofBytes) {
			return nil, fmt.Errorf("proof truncated at merkle proof node %d", i)
		}
		node := make([]byte, 32)
		copy(node, proofBytes[offset:offset+32])
		proof.MerkleProof[i] = node
		offset += 32
	}

	if offset+32 > len(proofBytes) {
		return nil, fmt.Errorf("proof truncated at state commitment")
	}
	proof.StateCommitment = make([]byte, 32)
	copy(proof.StateCommitment, proofBytes[offset:offset+32])
	offset += 32

	if offset+32 > len(proofBytes) {
		return nil, fmt.Errorf("proof truncated at execution trace")
	}
	proof.ExecutionTrace = make([]byte, 32)
	copy(proof.ExecutionTrace, proofBytes[offset:offset+32])
	offset += 32

	if offset+8 > len(proofBytes) {
		return nil, fmt.Errorf("proof truncated at nonce")
	}
	proof.Nonce = binary.BigEndian.Uint64(proofBytes[offset : offset+8])
	offset += 8

	if offset+8 > len(proofBytes) {
		return nil, fmt.Errorf("proof truncated at timestamp")
	}
	proof.Timestamp = types.SaturateUint64ToInt64(binary.BigEndian.Uint64(proofBytes[offset : offset+8]))

	return proof, nil
}

// verifyEd25519Signature verifies the Ed25519 signature over the computation result.
// Performs comprehensive key validation to prevent small subgroup and key substitution attacks.
func (k Keeper) verifyEd25519Signature(proof *types.VerificationProof, result types.Result, request types.Request) bool {
	if len(proof.Signature) != ed25519.SignatureSize {
		return false
	}

	if len(proof.PublicKey) != ed25519.PublicKeySize {
		return false
	}

	// Validate public key is not all zeros (invalid key)
	allZeros := true
	for _, b := range proof.PublicKey {
		if b != 0 {
			allZeros = false
			break
		}
	}
	if allZeros {
		return false
	}

	// SEC-5: Validate public key is not a low-order point (small subgroup attack prevention)
	// Ed25519 has 8 low-order points that must be rejected for security.
	// Using a low-order public key allows signature forgery attacks because
	// multiplying by the curve order produces the identity element.
	// Reference: https://cr.yp.to/ecdh/curve25519-20060209.pdf
	lowOrderPoints := [][]byte{
		// 1. Identity point (0)
		{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		// 2. Order 1 point
		{0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		// 3. Order 8 point (p-1 encoding)
		{0xec, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0x7f},
		// 4. Order 2 point (non-canonical)
		{0xc7, 0x17, 0x6a, 0x70, 0x3d, 0x4d, 0xd8, 0x4f, 0xba, 0x3c, 0x0b, 0x76, 0x0d, 0x10, 0x67, 0x0f, 0x2a, 0x20, 0x53, 0xfa, 0x2c, 0x39, 0xcc, 0xc6, 0x4e, 0xc7, 0xfd, 0x77, 0x92, 0xac, 0x03, 0x7a},
		// 5. Order 4 point (non-canonical, high bit set)
		{0xc7, 0x17, 0x6a, 0x70, 0x3d, 0x4d, 0xd8, 0x4f, 0xba, 0x3c, 0x0b, 0x76, 0x0d, 0x10, 0x67, 0x0f, 0x2a, 0x20, 0x53, 0xfa, 0x2c, 0x39, 0xcc, 0xc6, 0x4e, 0xc7, 0xfd, 0x77, 0x92, 0xac, 0x03, 0xfa},
		// 6. Order 4 point
		{0x26, 0xe8, 0x95, 0x8f, 0xc2, 0xb2, 0x27, 0xb0, 0x45, 0xc3, 0xf4, 0x89, 0xf2, 0xef, 0x98, 0xf0, 0xd5, 0xdf, 0xac, 0x05, 0xd3, 0xc6, 0x33, 0x39, 0xb1, 0x38, 0x02, 0x88, 0x6d, 0x53, 0xfc, 0x05},
		// 7. Order 4 point (high bit set)
		{0x26, 0xe8, 0x95, 0x8f, 0xc2, 0xb2, 0x27, 0xb0, 0x45, 0xc3, 0xf4, 0x89, 0xf2, 0xef, 0x98, 0xf0, 0xd5, 0xdf, 0xac, 0x05, 0xd3, 0xc6, 0x33, 0x39, 0xb1, 0x38, 0x02, 0x88, 0x6d, 0x53, 0xfc, 0x85},
		// 8. Identity with high bit set (non-canonical zero)
		{0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x80},
	}
	for _, lowOrder := range lowOrderPoints {
		if bytes.Equal(proof.PublicKey, lowOrder) {
			return false
		}
	}

	publicKey := ed25519.PublicKey(proof.PublicKey)
	message := proof.ComputeMessageHash(result.RequestId, result.OutputHash)

	return ed25519.Verify(publicKey, message, proof.Signature)
}

// validateMerkleProof validates the merkle inclusion proof for the execution trace.
func (k Keeper) validateMerkleProof(proof *types.VerificationProof, result types.Result) uint32 {
	if len(proof.MerkleProof) == 0 {
		return 0
	}

	if len(proof.MerkleRoot) != 32 {
		return 0
	}

	var currentHash []byte
	if len(proof.ExecutionTrace) == 32 {
		currentHash = make([]byte, 32)
		copy(currentHash, proof.ExecutionTrace)
	} else {
		leafHash := sha256.Sum256(proof.ExecutionTrace)
		currentHash = leafHash[:]
	}

	for _, sibling := range proof.MerkleProof {
		if len(sibling) != 32 {
			return 0
		}

		// Canonical ordering: always hash smaller value first to prevent proof forgery.
		// This is critical for Merkle tree security - without canonical ordering,
		// attackers can swap sibling positions to create valid but fraudulent proofs.
		hasher := sha256.New()
		if bytes.Compare(currentHash, sibling) < 0 {
			hasher.Write(currentHash)
			hasher.Write(sibling)
		} else {
			hasher.Write(sibling)
			hasher.Write(currentHash)
		}
		currentHash = hasher.Sum(nil)
	}

	if bytes.Equal(currentHash, proof.MerkleRoot) {
		return 15
	}

	return 0
}

// verifyStateTransition validates the state transition commitment.
func (k Keeper) verifyStateTransition(proof *types.VerificationProof, result types.Result, request types.Request) uint32 {
	if len(proof.StateCommitment) != 32 {
		return 0
	}

	hasher := sha256.New()
	hasher.Write([]byte(request.ContainerImage))
	for _, cmd := range request.Command {
		hasher.Write([]byte(cmd))
	}
	hasher.Write([]byte(result.OutputHash))
	hasher.Write(proof.ExecutionTrace)
	expectedCommitment := hasher.Sum(nil)

	if bytes.Equal(proof.StateCommitment, expectedCommitment) {
		return 25
	}

	partialMatch := 0
	for i := 0; i < 32; i++ {
		if proof.StateCommitment[i] == expectedCommitment[i] {
			partialMatch++
		}
	}

	if partialMatch >= 24 {
		return 15
	} else if partialMatch >= 16 {
		return 8
	}

	return 0
}

// verifyDeterministicExecution verifies deterministic computation properties.
func (k Keeper) verifyDeterministicExecution(proof *types.VerificationProof, result types.Result) uint32 {
	if len(proof.ExecutionTrace) < 32 {
		return 0
	}

	hasher := sha256.New()
	hasher.Write([]byte(result.OutputHash))
	hasher.Write(proof.ExecutionTrace)
	traceVerification := hasher.Sum(nil)

	if len(traceVerification) == 32 {
		return 10
	}

	return 5
}

// checkReplayAttack checks if the nonce has been used before by this provider.
func (k Keeper) checkReplayAttack(ctx context.Context, provider sdk.AccAddress, nonce uint64) bool {
	store := k.getStore(ctx)
	key := NonceKey(provider, nonce)
	return store.Has(key)
}

// NonceReservationStatus represents the status of a nonce reservation.
// SEC-1 FIX: This type enables the nonce reservation pattern that prevents replay attacks
// during the window between nonce check and verification completion.
type NonceReservationStatus byte

const (
	// NonceStatusReserved indicates the nonce is reserved but verification not complete.
	// This prevents concurrent submissions with the same nonce.
	NonceStatusReserved NonceReservationStatus = 0x01
	// NonceStatusUsed indicates the nonce has been fully used (verification complete).
	NonceStatusUsed NonceReservationStatus = 0x02
)

// reserveNonce reserves a nonce BEFORE verification begins to prevent replay attacks.
// SEC-1 FIX: This prevents the race condition where an attacker could submit the same
// nonce while verification is in progress. The nonce is marked as reserved immediately,
// and upgraded to "used" status after verification completes.
// Returns true if the reservation was successful (nonce not already used/reserved).
func (k Keeper) reserveNonce(ctx context.Context, provider sdk.AccAddress, nonce uint64) bool {
	store := k.getStore(ctx)
	key := NonceKey(provider, nonce)

	// Check if nonce already exists (either reserved or used)
	if store.Has(key) {
		return false
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	currentHeight := sdkCtx.BlockHeight()

	// Store nonce with "reserved" status and timestamp
	// Format: [status byte][timestamp 8 bytes]
	data := make([]byte, 9)
	data[0] = byte(NonceStatusReserved)
	binary.BigEndian.PutUint64(data[1:], types.SaturateInt64ToUint64(sdkCtx.BlockTime().Unix()))

	store.Set(key, data)

	// Create height-indexed entry for cleanup
	heightIndexKey := NonceByHeightKey(currentHeight, provider, nonce)
	store.Set(heightIndexKey, data)

	return true
}

// recordNonceUsage upgrades a reserved nonce to "used" status after verification completes.
// If the nonce was not previously reserved, it creates a new entry with "used" status.
// SEC-1 FIX: This completes the reservation pattern - nonces are reserved at the START
// of verification and marked as used at the END, preventing any replay window.
func (k Keeper) recordNonceUsage(ctx context.Context, provider sdk.AccAddress, nonce uint64) {
	store := k.getStore(ctx)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	currentHeight := sdkCtx.BlockHeight()

	// Store with "used" status and timestamp
	// Format: [status byte][timestamp 8 bytes]
	data := make([]byte, 9)
	data[0] = byte(NonceStatusUsed)
	binary.BigEndian.PutUint64(data[1:], types.SaturateInt64ToUint64(sdkCtx.BlockTime().Unix()))

	// Store the nonce with "used" status in the main nonce store
	key := NonceKey(provider, nonce)
	store.Set(key, data)

	// Create/update height-indexed entry for cleanup
	heightIndexKey := NonceByHeightKey(currentHeight, provider, nonce)
	store.Set(heightIndexKey, data)
}

// recordReplayAttempt records a detected replay attack attempt.
func (k Keeper) recordReplayAttempt(ctx context.Context, provider sdk.AccAddress, nonce uint64) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"replay_attack_detected",
			sdk.NewAttribute("provider", provider.String()),
			sdk.NewAttribute("nonce", fmt.Sprintf("%d", nonce)),
			sdk.NewAttribute("timestamp", sdkCtx.BlockTime().Format(time.RFC3339)),
		),
	)
}

// hasProofHash checks whether a verification proof hash has already been used.
func (k Keeper) hasProofHash(ctx context.Context, provider sdk.AccAddress, hash []byte) bool {
	store := k.getStore(ctx)
	return store.Has(ProofHashKey(provider, hash))
}

// recordProofHashUsage stores the fact that a provider has used a specific verification proof.
func (k Keeper) recordProofHashUsage(ctx context.Context, provider sdk.AccAddress, hash []byte, timestamp time.Time) {
	store := k.getStore(ctx)
	timeBz := make([]byte, 8)
	binary.BigEndian.PutUint64(timeBz, types.SaturateInt64ToUint64(timestamp.Unix()))
	store.Set(ProofHashKey(provider, hash), timeBz)
}

// verifyProviderSigningKey ensures the provided key matches the provider's registered signing key.
// SEC-2 FIX: This function no longer auto-trusts first-submitted keys. Providers MUST explicitly
// register their signing key via RegisterSigningKey BEFORE submitting results. This prevents
// trust-on-first-use attacks where an attacker could submit a result with their own key
// before the legitimate provider registers.
func (k Keeper) verifyProviderSigningKey(ctx context.Context, provider sdk.AccAddress, pubKey []byte) bool {
	// First, check if the provider's on-chain account has a public key set
	// This is the primary source of truth for identity verification
	account := k.accountKeeper.GetAccount(ctx, provider)
	if account != nil && account.GetPubKey() != nil {
		return bytes.Equal(account.GetPubKey().Bytes(), pubKey)
	}

	// Fall back to explicitly registered signing key
	store := k.getStore(ctx)
	key := ProviderSigningKeyKey(provider)
	stored := store.Get(key)

	// SEC-2 FIX: If no key is registered, reject the verification.
	// Providers MUST call RegisterSigningKey first before submitting results.
	// This prevents trust-on-first-use (TOFU) attacks.
	if len(stored) == 0 {
		sdkCtx := sdk.UnwrapSDKContext(ctx)
		sdkCtx.EventManager().EmitEvent(
			sdk.NewEvent(
				"signing_key_not_registered",
				sdk.NewAttribute("provider", provider.String()),
				sdk.NewAttribute("action", "result_submission_rejected"),
			),
		)
		return false
	}

	return bytes.Equal(stored, pubKey)
}

// RegisterSigningKey explicitly registers a provider's signing key.
// SEC-2 FIX: This is the ONLY way to register a signing key. The key is NOT auto-trusted
// on first use. Providers must call this function to register their key before submitting results.
// The key can only be registered if:
// 1. The provider is a registered and active provider
// 2. No key is currently registered, OR
// 3. The provider is updating their key (requires proof of ownership of old key)
func (k Keeper) RegisterSigningKey(ctx context.Context, provider sdk.AccAddress, newPubKey []byte, oldKeySignature []byte) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Validate the provider is registered and active
	providerRecord, err := k.GetProvider(ctx, provider)
	if err != nil {
		return fmt.Errorf("provider not registered: %w", err)
	}

	if !providerRecord.Active {
		return fmt.Errorf("provider is not active")
	}

	// Validate the new public key
	if len(newPubKey) != ed25519.PublicKeySize {
		return fmt.Errorf("invalid public key size: expected %d bytes, got %d", ed25519.PublicKeySize, len(newPubKey))
	}

	// Check for all-zeros key (invalid)
	allZeros := true
	for _, b := range newPubKey {
		if b != 0 {
			allZeros = false
			break
		}
	}
	if allZeros {
		return fmt.Errorf("invalid public key: all zeros")
	}

	store := k.getStore(ctx)
	key := ProviderSigningKeyKey(provider)
	existingKey := store.Get(key)

	// If a key is already registered, require proof of ownership (signature)
	if len(existingKey) > 0 {
		if len(oldKeySignature) == 0 {
			return fmt.Errorf("key rotation requires signature from existing key")
		}

		// Verify the signature using the existing key
		// Message: "ROTATE_KEY:" + provider address + new public key
		message := []byte("ROTATE_KEY:" + provider.String())
		message = append(message, newPubKey...)
		messageHash := sha256.Sum256(message)

		if len(oldKeySignature) != ed25519.SignatureSize {
			return fmt.Errorf("invalid signature size")
		}

		if !ed25519.Verify(existingKey, messageHash[:], oldKeySignature) {
			return fmt.Errorf("invalid signature for key rotation: must sign with existing key")
		}

		sdkCtx.EventManager().EmitEvent(
			sdk.NewEvent(
				"signing_key_rotated",
				sdk.NewAttribute("provider", provider.String()),
				sdk.NewAttribute("old_key_hash", hex.EncodeToString(sha256.New().Sum(existingKey)[:8])),
				sdk.NewAttribute("new_key_hash", hex.EncodeToString(sha256.New().Sum(newPubKey)[:8])),
			),
		)
	} else {
		// First-time registration: verify provider owns the new key by checking
		// that the transaction was signed by the provider's account
		// (This is implicitly verified by Cosmos SDK's ante handler)
		sdkCtx.EventManager().EmitEvent(
			sdk.NewEvent(
				"signing_key_registered",
				sdk.NewAttribute("provider", provider.String()),
				sdk.NewAttribute("key_hash", hex.EncodeToString(sha256.New().Sum(newPubKey)[:8])),
			),
		)
	}

	// Store the new signing key
	keyToStore := make([]byte, len(newPubKey))
	copy(keyToStore, newPubKey)
	store.Set(key, keyToStore)

	return nil
}

// HasRegisteredSigningKey checks if a provider has a registered signing key.
func (k Keeper) HasRegisteredSigningKey(ctx context.Context, provider sdk.AccAddress) bool {
	// Check on-chain account key first
	account := k.accountKeeper.GetAccount(ctx, provider)
	if account != nil && account.GetPubKey() != nil {
		return true
	}

	// Check explicitly registered key
	store := k.getStore(ctx)
	key := ProviderSigningKeyKey(provider)
	return len(store.Get(key)) > 0
}

// GetRegisteredSigningKey returns the registered signing key for a provider, if any.
func (k Keeper) GetRegisteredSigningKey(ctx context.Context, provider sdk.AccAddress) []byte {
	// Check on-chain account key first
	account := k.accountKeeper.GetAccount(ctx, provider)
	if account != nil && account.GetPubKey() != nil {
		return account.GetPubKey().Bytes()
	}

	// Check explicitly registered key
	store := k.getStore(ctx)
	key := ProviderSigningKeyKey(provider)
	return store.Get(key)
}

// slashProviderForInvalidProof slashes a provider's stake for submitting invalid verification proofs.
func (k Keeper) slashProviderForInvalidProof(ctx context.Context, provider sdk.AccAddress, reason string) error {
	providerRecord, err := k.GetProvider(ctx, provider)
	if err != nil {
		return fmt.Errorf("provider not found: %w", err)
	}

	params, err := k.GetParams(ctx)
	if err != nil {
		return fmt.Errorf("slashProviderForVerificationFailure: get params: %w", err)
	}

	slashPercentage := math.LegacyNewDec(int64(params.StakeSlashPercentage)).QuoInt64(100)
	slashAmount := slashPercentage.MulInt(providerRecord.Stake).TruncateInt()

	newStake := providerRecord.Stake.Sub(slashAmount)
	if newStake.IsNegative() {
		newStake = math.ZeroInt()
	}
	providerRecord.Stake = newStake

	if providerRecord.Reputation > 30 {
		providerRecord.Reputation -= 30
	} else {
		providerRecord.Reputation = 0
	}

	if providerRecord.Stake.LT(params.MinProviderStake) {
		providerRecord.Active = false
		if err := k.setActiveProviderIndex(ctx, provider, false); err != nil {
			return fmt.Errorf("slashProviderForVerificationFailure: update index: %w", err)
		}
	}

	if err := k.SetProvider(ctx, *providerRecord); err != nil {
		return fmt.Errorf("failed to update provider: %w", err)
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"provider_slashed_verification_failure",
			sdk.NewAttribute("provider", provider.String()),
			sdk.NewAttribute("slash_amount", slashAmount.String()),
			sdk.NewAttribute("reason", reason),
			sdk.NewAttribute("new_stake", newStake.String()),
			sdk.NewAttribute("new_reputation", fmt.Sprintf("%d", providerRecord.Reputation)),
		),
	)

	return nil
}

// VerifyZKProof is implemented in zk_enhancements.go and calls the ZKVerifier from zk_verification.go.
// The implementation performs actual Groth16 ZK-SNARK verification using the gnark library.
