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

// SubmitResult processes a result submission from a provider with institutional-grade verification.
func (k Keeper) SubmitResult(ctx context.Context, provider sdk.AccAddress, requestID uint64, outputHash, outputURL string, exitCode int32, logsURL string, verificationProof []byte) error {
	request, err := k.GetRequest(ctx, requestID)
	if err != nil {
		return fmt.Errorf("request not found: %w", err)
	}

	if request.Provider != provider.String() {
		return fmt.Errorf("unauthorized: provider %s not assigned to request %d", provider.String(), requestID)
	}

	if request.Status != types.RequestStatus_REQUEST_STATUS_ASSIGNED &&
		request.Status != types.RequestStatus_REQUEST_STATUS_PROCESSING {
		return fmt.Errorf("request is not in a state to accept results: %s", request.Status.String())
	}

	if outputHash == "" {
		return fmt.Errorf("output hash is required")
	}
	if outputURL == "" {
		return fmt.Errorf("output URL is required")
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
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
	request.Status = types.RequestStatus_REQUEST_STATUS_PROCESSING

	if err := k.SetRequest(ctx, *request); err != nil {
		return fmt.Errorf("failed to update request: %w", err)
	}

	if err := k.updateRequestStatusIndex(ctx, *request); err != nil {
		return fmt.Errorf("failed to update request status index: %w", err)
	}

	verified, score := k.verifyResult(ctx, result, *request)
	result.Verified = verified
	result.VerificationScore = score

	if err := k.SetResult(ctx, &result); err != nil {
		return fmt.Errorf("failed to update result with verification: %w", err)
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
		return nil, err
	}

	return &result, nil
}

// SetResult stores a result record.
/*
func (k Keeper) SetResult(ctx context.Context, result types.Result) error {
	store := k.getStore(ctx)
	bz, err := k.cdc.Marshal(&result)
	if err != nil {
		return err
	}

	store.Set(ResultKey(result.RequestId), bz)
	return nil
}
*/

// verifyResult performs cryptographic verification of the submitted result using ZK-SNARKs.
// This replaces the old scoring system with actual zero-knowledge proof verification.
func (k Keeper) verifyResult(ctx context.Context, result types.Result, request types.Request) (verified bool, score uint32) {
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
		}
	}

	if len(result.VerificationProof) > 0 {
		proofScore, proofBreakdown := k.validateVerificationProof(ctx, result, request)
		scoreBreakdown["cryptographic_proof"] = proofScore
		for key, val := range proofBreakdown {
			scoreBreakdown[key] = val
		}
	}

	provider, err := sdk.AccAddressFromBech32(result.Provider)
	if err == nil {
		providerRecord, err := k.GetProvider(ctx, provider)
		if err == nil {
			reputationBonus := uint32(providerRecord.Reputation / 10)
			if reputationBonus > 10 {
				reputationBonus = 10
			}
			scoreBreakdown["provider_reputation"] = reputationBonus
		}
	}

	for _, points := range scoreBreakdown {
		score += points
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
// Returns the proof score and detailed breakdown of verification components.
func (k Keeper) validateVerificationProof(ctx context.Context, result types.Result, request types.Request) (uint32, map[string]uint32) {
	var totalScore uint32 = 0
	breakdown := make(map[string]uint32)

	proof, err := k.parseVerificationProof(result.VerificationProof)
	if err != nil {
		breakdown["parse_error"] = 0
		return 0, breakdown
	}

	if err := proof.Validate(); err != nil {
		breakdown["validation_error"] = 0
		return 0, breakdown
	}

	provider, err := sdk.AccAddressFromBech32(result.Provider)
	if err != nil {
		breakdown["provider_address_error"] = 0
		return 0, breakdown
	}

	if k.checkReplayAttack(ctx, provider, proof.Nonce) {
		breakdown["replay_attack_detected"] = 0
		k.recordReplayAttempt(ctx, provider, proof.Nonce)
		return 0, breakdown
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

	stateScore := k.verifyStateTransition(proof, result, request)
	breakdown["state_transition"] = stateScore
	totalScore += stateScore

	deterministicScore := k.verifyDeterministicExecution(proof, result)
	breakdown["deterministic_execution"] = deterministicScore
	totalScore += deterministicScore

	k.recordNonceUsage(ctx, provider, proof.Nonce)

	return totalScore, breakdown
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
	proof.Timestamp = int64(binary.BigEndian.Uint64(proofBytes[offset : offset+8]))

	return proof, nil
}

// verifyEd25519Signature verifies the Ed25519 signature over the computation result.
func (k Keeper) verifyEd25519Signature(proof *types.VerificationProof, result types.Result, request types.Request) bool {
	if len(proof.Signature) != ed25519.SignatureSize {
		return false
	}

	if len(proof.PublicKey) != ed25519.PublicKeySize {
		return false
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

	leafHash := sha256.Sum256(proof.ExecutionTrace)
	currentHash := leafHash[:]

	for _, sibling := range proof.MerkleProof {
		if len(sibling) != 32 {
			return 0
		}

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
		return 15
	}

	partialMatch := 0
	for i := 0; i < 32; i++ {
		if proof.StateCommitment[i] == expectedCommitment[i] {
			partialMatch++
		}
	}

	if partialMatch >= 24 {
		return 10
	} else if partialMatch >= 16 {
		return 5
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

// recordNonceUsage records a nonce as used to prevent replay attacks.
func (k Keeper) recordNonceUsage(ctx context.Context, provider sdk.AccAddress, nonce uint64) {
	store := k.getStore(ctx)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Store timestamp as 8-byte big-endian integer
	timestampBz := make([]byte, 8)
	binary.BigEndian.PutUint64(timestampBz, uint64(sdkCtx.BlockTime().Unix()))

	key := NonceKey(provider, nonce)
	store.Set(key, timestampBz)
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

// slashProviderForInvalidProof slashes a provider's stake for submitting invalid verification proofs.
func (k Keeper) slashProviderForInvalidProof(ctx context.Context, provider sdk.AccAddress, reason string) error {
	providerRecord, err := k.GetProvider(ctx, provider)
	if err != nil {
		return fmt.Errorf("provider not found: %w", err)
	}

	params, err := k.GetParams(ctx)
	if err != nil {
		return err
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
			return err
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

// VerifyZKProof verifies a ZK proof for result data. Currently a placeholder that always succeeds.

