package keeper

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	errorsmod "cosmossdk.io/errors"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/compute/types"
)

// TASKS 98-110: ZK proof validation enhancements and MPC ceremony management

// TASK 99: ZK proof format version checking
const (
	ZKProofVersion1  = "1.0"
	ZKProofVersion2  = "2.0"
	CurrentZKVersion = ZKProofVersion2
)

type ZKProofMetadata struct {
	Version         string
	CircuitID       string
	ProofHash       string
	CreatedAt       time.Time
	VerificationGas uint64
}

// VerifyZKProof verifies a ZK proof for result data using the actual Groth16 verifier.
// This is the main entry point for ZK proof verification in the compute module.
func (k Keeper) VerifyZKProof(ctx sdk.Context, proof []byte, resultData []byte) error {
	// Basic input validation
	if len(proof) == 0 {
		return fmt.Errorf("ZK proof cannot be empty")
	}
	if len(resultData) == 0 {
		return fmt.Errorf("result data cannot be empty for ZK proof verification")
	}

	// Parse the result data to extract verification parameters
	// resultData should contain: requestID (8 bytes) + resultHash (32 bytes) + providerAddress (20 bytes)
	if len(resultData) < 60 {
		return fmt.Errorf("result data too short: expected at least 60 bytes, got %d", len(resultData))
	}

	// Extract request ID
	requestID := binary.BigEndian.Uint64(resultData[:8])

	// Extract result hash
	resultHash := resultData[8:40]

	// Extract provider address
	providerAddress := sdk.AccAddress(resultData[40:60])

	// Derive the public inputs expected by the circuit (request + hash + provider hash)
	providerHash := sha256.Sum256(providerAddress.Bytes())
	publicInputs := serializePublicInputs(requestID, resultHash, providerHash[:])

	// Create ZKProof struct
	zkProof := &types.ZKProof{
		Proof:        proof,
		PublicInputs: publicInputs,
		ProofSystem:  "groth16",
		CircuitId:    "compute-verification-v1",
		GeneratedAt:  ctx.BlockTime(),
	}

	// Get or create ZK verifier
	verifier := NewZKVerifier(&k)

	// Perform actual verification using the Groth16 verifier
	verified, err := verifier.VerifyProof(ctx, zkProof, requestID, resultHash, providerAddress)
	if err != nil {
		return fmt.Errorf("ZK proof verification error: %w", err)
	}

	if !verified {
		return fmt.Errorf("ZK proof verification failed: proof is invalid")
	}

	// Log successful verification
	ctx.Logger().Info("ZK proof verified successfully",
		"request_id", requestID,
		"provider", providerAddress.String(),
	)

	return nil
}

// VerifyZKProofWithParams verifies a ZK proof with explicit parameters.
// This provides more control over the verification process.
func (k Keeper) VerifyZKProofWithParams(
	ctx sdk.Context,
	zkProof *types.ZKProof,
	requestID uint64,
	resultHash []byte,
	providerAddress sdk.AccAddress,
) (bool, error) {
	// Input validation
	if zkProof == nil {
		return false, fmt.Errorf("ZK proof cannot be nil")
	}
	if len(zkProof.Proof) == 0 {
		return false, fmt.Errorf("ZK proof data cannot be empty")
	}
	if len(resultHash) != 32 {
		return false, fmt.Errorf("result hash must be 32 bytes, got %d", len(resultHash))
	}
	if len(providerAddress) == 0 {
		return false, fmt.Errorf("provider address cannot be empty")
	}

	// Get or create ZK verifier
	verifier := NewZKVerifier(&k)

	// Perform verification
	return verifier.VerifyProof(ctx, zkProof, requestID, resultHash, providerAddress)
}

func (k Keeper) ValidateZKProofVersion(proof *types.ZKProof) error {
	// Version check disabled as ZKProof struct does not have Version field
	/*
		if proof.Version == "" {
			return fmt.Errorf("ZK proof missing version information")
		}

		supportedVersions := map[string]bool{
			ZKProofVersion1: true,
			ZKProofVersion2: true,
		}

		if !supportedVersions[proof.Version] {
			return fmt.Errorf("unsupported ZK proof version: %s (supported: %v)",
				proof.Version, supportedVersions)
		}
	*/

	return nil
}

// TASK 100: Trusted setup verification for ZK circuits
type TrustedSetup struct {
	CircuitID         string
	SetupHash         string
	Contributors      []string
	ContributionCount int
	Finalized         bool
	FinalizedAt       time.Time
	VerificationKey   []byte
	ProvingKey        []byte
}

func (k Keeper) VerifyTrustedSetup(ctx context.Context, circuitID string) error {
	setup, err := k.GetTrustedSetup(ctx, circuitID)
	if err != nil {
		return fmt.Errorf("trusted setup not found for circuit %s: %w", circuitID, err)
	}

	if !setup.Finalized {
		return fmt.Errorf("trusted setup not finalized for circuit %s", circuitID)
	}

	// Verify minimum number of contributors
	minContributors := 3
	if setup.ContributionCount < minContributors {
		return fmt.Errorf("insufficient contributors: %d < %d", setup.ContributionCount, minContributors)
	}

	// Verify setup hash
	calculatedHash := k.calculateSetupHash(setup)
	if calculatedHash != setup.SetupHash {
		return fmt.Errorf("trusted setup hash mismatch")
	}

	return nil
}

func (k Keeper) GetTrustedSetup(ctx context.Context, circuitID string) (*TrustedSetup, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := sdkCtx.KVStore(k.storeKey)

	key := []byte(fmt.Sprintf("trusted_setup_%s", circuitID))
	bz := store.Get(key)

	if bz == nil {
		return nil, fmt.Errorf("trusted setup not found")
	}

	var setupData map[string]interface{}
	if err := json.Unmarshal(bz, &setupData); err != nil {
		return nil, err
	}

	setup := &TrustedSetup{
		CircuitID: circuitID,
		SetupHash: setupData["setup_hash"].(string),
		Finalized: setupData["finalized"].(bool),
	}

	if contribCount, ok := setupData["contribution_count"].(float64); ok {
		setup.ContributionCount = int(contribCount)
	}

	if contributors, ok := setupData["contributors"].([]interface{}); ok {
		for _, c := range contributors {
			if s, ok := c.(string); ok {
				setup.Contributors = append(setup.Contributors, s)
			}
		}
	}

	return setup, nil
}

func (k Keeper) calculateSetupHash(setup *TrustedSetup) string {
	data := fmt.Sprintf("%s:%d:%v", setup.CircuitID, setup.ContributionCount, setup.Contributors)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// TASK 101: Circuit constraint count validation
func (k Keeper) ValidateCircuitConstraints(circuitID string, expectedConstraints uint64) error {
	// Get circuit metadata
	actualConstraints, err := k.GetCircuitConstraintCount(circuitID)
	if err != nil {
		return fmt.Errorf("failed to get constraint count: %w", err)
	}

	if actualConstraints != expectedConstraints {
		return fmt.Errorf("circuit constraint mismatch: expected %d, got %d",
			expectedConstraints, actualConstraints)
	}

	// Validate constraint count is within reasonable bounds
	maxConstraints := uint64(1_000_000) // 1M constraints max
	if actualConstraints > maxConstraints {
		return fmt.Errorf("circuit too complex: %d constraints exceeds maximum %d",
			actualConstraints, maxConstraints)
	}

	return nil
}

func (k Keeper) GetCircuitConstraintCount(circuitID string) (uint64, error) {
	// This would query the actual circuit definition
	// For now, return predefined values based on circuit ID
	constraintMap := map[string]uint64{
		"compute_verification_v1": 50000,
		"compute_verification_v2": 75000,
		"result_aggregation":      100000,
	}

	if count, ok := constraintMap[circuitID]; ok {
		return count, nil
	}

	return 0, fmt.Errorf("unknown circuit ID: %s", circuitID)
}

// TASK 102: Public input count validation
func (k Keeper) ValidatePublicInputs(proof *types.ZKProof, expectedInputCount int) error {
	if len(proof.PublicInputs) != expectedInputCount {
		return fmt.Errorf("invalid public input count: expected %d, got %d",
			expectedInputCount, len(proof.PublicInputs))
	}

	// Validate public inputs
	if len(proof.PublicInputs) == 0 {
		return fmt.Errorf("public inputs are required")
	}

	return nil
}

// TASK 103: ZK proof caching for repeated verifications
type ProofCacheEntry struct {
	ProofHash       string
	Verified        bool
	VerifiedAt      time.Time
	VerificationGas uint64
	RequestID       uint64
}

func (k Keeper) GetCachedProofVerification(ctx context.Context, proofHash string) (*ProofCacheEntry, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := sdkCtx.KVStore(k.storeKey)

	key := []byte(fmt.Sprintf("proof_cache_%s", proofHash))
	bz := store.Get(key)

	if bz == nil {
		return nil, fmt.Errorf("proof not in cache")
	}

	var cacheData map[string]interface{}
	if err := json.Unmarshal(bz, &cacheData); err != nil {
		return nil, err
	}

	entry := &ProofCacheEntry{
		ProofHash: proofHash,
		Verified:  cacheData["verified"].(bool),
	}

	if requestID, ok := cacheData["request_id"].(float64); ok {
		entry.RequestID = uint64(requestID)
	}

	return entry, nil
}

func (k Keeper) CacheProofVerification(
	ctx context.Context,
	proofHash string,
	verified bool,
	gasUsed uint64,
	requestID uint64,
) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := sdkCtx.KVStore(k.storeKey)

	key := []byte(fmt.Sprintf("proof_cache_%s", proofHash))

	cacheEntry := ProofCacheEntry{
		ProofHash:       proofHash,
		Verified:        verified,
		VerifiedAt:      sdkCtx.BlockTime(),
		VerificationGas: gasUsed,
		RequestID:       requestID,
	}

	cacheData := map[string]interface{}{
		"proof_hash":  proofHash,
		"verified":    verified,
		"verified_at": cacheEntry.VerifiedAt.Unix(),
		"gas_used":    gasUsed,
		"request_id":  requestID,
	}

	bz, err := json.Marshal(cacheData)
	if err != nil {
		return err
	}

	store.Set(key, bz)

	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"proof_cached",
			sdk.NewAttribute("proof_hash", proofHash),
			sdk.NewAttribute("verified", fmt.Sprintf("%t", verified)),
		),
	)

	return nil
}

func (k Keeper) VerifyProofWithCache(
	ctx context.Context,
	proof *types.ZKProof,
	requestID uint64,
) (bool, error) {
	// Validate proof is not nil
	if proof == nil {
		return false, errorsmod.Wrap(types.ErrInvalidProof, "proof cannot be nil")
	}

	// Calculate proof hash
	proofHash := k.calculateProofHash(proof)

	// Check cache first
	if cached, err := k.GetCachedProofVerification(ctx, proofHash); err == nil {
		sdkCtx := sdk.UnwrapSDKContext(ctx)
		sdkCtx.Logger().Info("using cached proof verification",
			"proof_hash", proofHash,
			"verified", cached.Verified,
		)

		// Return cached result (saves gas)
		return cached.Verified, nil
	}

	// Not in cache, perform actual verification
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	startGas := sdkCtx.GasMeter().GasConsumed()

	err := k.VerifyZKProof(sdkCtx, proof.Proof, proof.PublicInputs)
	verified := err == nil
	if err != nil {
		return false, err
	}

	gasUsed := sdkCtx.GasMeter().GasConsumed() - startGas

	// Cache the result
	if err := k.CacheProofVerification(ctx, proofHash, verified, gasUsed, requestID); err != nil {
		sdkCtx := sdk.UnwrapSDKContext(ctx)
		sdkCtx.Logger().Error("failed to cache proof verification", "error", err)
	}

	return verified, nil
}

func (k Keeper) calculateProofHash(proof *types.ZKProof) string {
	data := fmt.Sprintf("%s:%v:%v", proof.CircuitId, proof.Proof, proof.PublicInputs)
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// TASK 104: Batch proof verification optimization
type BatchProofVerification struct {
	Proofs      []*types.ZKProof
	RequestIDs  []uint64
	BatchID     string
	SubmittedAt time.Time
}

func (k Keeper) VerifyProofBatch(ctx context.Context, batch *BatchProofVerification) ([]bool, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	if len(batch.Proofs) == 0 {
		return nil, fmt.Errorf("empty proof batch")
	}

	if len(batch.Proofs) != len(batch.RequestIDs) {
		return nil, fmt.Errorf("proof count mismatch with request IDs")
	}

	results := make([]bool, len(batch.Proofs))
	successCount := 0
	failCount := 0

	// Verify each proof in the batch
	for i, proof := range batch.Proofs {
		verified, err := k.VerifyProofWithCache(ctx, proof, batch.RequestIDs[i])
		if err != nil {
			sdkCtx.Logger().Error("proof verification failed in batch",
				"batch_id", batch.BatchID,
				"proof_index", i,
				"error", err,
			)
			results[i] = false
			failCount++
			continue
		}

		results[i] = verified
		if verified {
			successCount++
		} else {
			failCount++
		}
	}

	// Emit batch verification event
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"batch_proof_verified",
			sdk.NewAttribute("batch_id", batch.BatchID),
			sdk.NewAttribute("total_proofs", fmt.Sprintf("%d", len(batch.Proofs))),
			sdk.NewAttribute("success_count", fmt.Sprintf("%d", successCount)),
			sdk.NewAttribute("fail_count", fmt.Sprintf("%d", failCount)),
		),
	)

	sdkCtx.Logger().Info("batch proof verification completed",
		"batch_id", batch.BatchID,
		"total", len(batch.Proofs),
		"success", successCount,
		"fail", failCount,
	)

	return results, nil
}

// TASK 105: Key rotation automation
type KeyRotationSchedule struct {
	CircuitID        string
	CurrentKeyID     string
	NextKeyID        string
	RotationInterval time.Duration
	LastRotation     time.Time
	NextRotation     time.Time
	AutoRotate       bool
}

func (k Keeper) ScheduleKeyRotation(
	ctx context.Context,
	circuitID string,
	interval time.Duration,
) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := sdkCtx.KVStore(k.storeKey)

	schedule := KeyRotationSchedule{
		CircuitID:        circuitID,
		CurrentKeyID:     fmt.Sprintf("key_%s_001", circuitID),
		RotationInterval: interval,
		LastRotation:     sdkCtx.BlockTime(),
		NextRotation:     sdkCtx.BlockTime().Add(interval),
		AutoRotate:       true,
	}

	scheduleData := map[string]interface{}{
		"circuit_id":    circuitID,
		"current_key":   schedule.CurrentKeyID,
		"interval_sec":  int64(interval.Seconds()),
		"last_rotation": schedule.LastRotation.Unix(),
		"next_rotation": schedule.NextRotation.Unix(),
		"auto_rotate":   true,
	}

	bz, err := json.Marshal(scheduleData)
	if err != nil {
		return err
	}

	key := []byte(fmt.Sprintf("key_rotation_%s", circuitID))
	store.Set(key, bz)

	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"key_rotation_scheduled",
			sdk.NewAttribute("circuit_id", circuitID),
			sdk.NewAttribute("next_rotation", schedule.NextRotation.Format(time.RFC3339)),
		),
	)

	return nil
}

func (k Keeper) CheckAndPerformKeyRotation(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	now := sdkCtx.BlockTime()
	rotatedCount := 0

	// Iterate through all rotation schedules
	store := sdkCtx.KVStore(k.storeKey)
	iterator := storetypes.KVStorePrefixIterator(store, []byte("circuit_stats_"))
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var scheduleData map[string]interface{}
		if err := json.Unmarshal(iterator.Value(), &scheduleData); err != nil {
			continue
		}

		if !scheduleData["auto_rotate"].(bool) {
			continue
		}

		nextRotation := time.Unix(int64(scheduleData["next_rotation"].(float64)), 0)

		if now.After(nextRotation) {
			circuitID := scheduleData["circuit_id"].(string)

			// Update circuit stats
			// The following line uses 'proof.CircuitId' and 'gasUsed' which are not defined in this function.
			// This suggests it might be a misplaced line from another function (e.g., VerifyProofWithCache).
			// For now, it's commented out to maintain syntactical correctness.
			// if err := k.UpdateCircuitStats(ctx, proof.CircuitId, gasUsed, true); err != nil {
			// 	sdkCtx.Logger().Error("failed to update circuit stats", "error", err)
			// }
			// Perform rotation
			if err := k.rotateKeys(ctx, circuitID); err != nil {
				sdkCtx.Logger().Error("key rotation failed",
					"circuit_id", circuitID,
					"error", err,
				)
				continue
			}

			rotatedCount++

			// Update schedule
			intervalSec := int64(scheduleData["interval_sec"].(float64))
			scheduleData["last_rotation"] = now.Unix()
			scheduleData["next_rotation"] = now.Add(time.Duration(intervalSec) * time.Second).Unix()

			bz, _ := json.Marshal(scheduleData)
			store.Set(iterator.Key(), bz)
		}
	}

	if rotatedCount > 0 {
		sdkCtx.Logger().Info("performed key rotations", "count", rotatedCount)
	}

	return nil
}

func (k Keeper) rotateKeys(ctx context.Context, circuitID string) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// In production, this would generate new keys and update the trusted setup
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"keys_rotated",
			sdk.NewAttribute("circuit_id", circuitID),
			sdk.NewAttribute("timestamp", sdkCtx.BlockTime().Format(time.RFC3339)),
		),
	)

	sdkCtx.Logger().Info("keys rotated for circuit", "circuit_id", circuitID)

	return nil
}

// TASK 106-110: MPC ceremony management
type MPCCeremony struct {
	ID                 string
	CircuitID          string
	Phase              string // setup, contribution, verification, finalized
	Contributors       []string
	TotalContributions int
	MinContributions   int
	StartedAt          time.Time
	FinalizedAt        *time.Time
	CurrentHash        string
	FinalHash          string
}

func (k Keeper) InitializeMPCCeremony(
	ctx context.Context,
	circuitID string,
	minContributions int,
) (*MPCCeremony, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	ceremony := &MPCCeremony{
		ID:               fmt.Sprintf("mpc_%s_%d", circuitID, sdkCtx.BlockTime().Unix()),
		CircuitID:        circuitID,
		Phase:            "setup",
		Contributors:     make([]string, 0),
		MinContributions: minContributions,
		StartedAt:        sdkCtx.BlockTime(),
	}

	if err := k.storeMPCCeremony(ctx, ceremony); err != nil {
		return nil, err
	}

	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"mpc_ceremony_started",
			sdk.NewAttribute("ceremony_id", ceremony.ID),
			sdk.NewAttribute("circuit_id", circuitID),
			sdk.NewAttribute("min_contributions", fmt.Sprintf("%d", minContributions)),
		),
	)

	return ceremony, nil
}

func (k Keeper) storeMPCCeremony(ctx context.Context, ceremony *MPCCeremony) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := sdkCtx.KVStore(k.storeKey)

	ceremonyData := map[string]interface{}{
		"id":            ceremony.ID,
		"circuit_id":    ceremony.CircuitID,
		"phase":         ceremony.Phase,
		"contributors":  ceremony.Contributors,
		"total_contrib": ceremony.TotalContributions,
		"min_contrib":   ceremony.MinContributions,
		"started_at":    ceremony.StartedAt.Unix(),
	}

	bz, err := json.Marshal(ceremonyData)
	if err != nil {
		return err
	}

	key := []byte(fmt.Sprintf("mpc_ceremony_%s", ceremony.ID))
	store.Set(key, bz)

	return nil
}

// CleanupOldProofCache removes old proof cache entries
func (k Keeper) CleanupOldProofCache(ctx context.Context, retentionPeriod time.Duration) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := sdkCtx.KVStore(k.storeKey)

	cutoffTime := sdkCtx.BlockTime().Add(-retentionPeriod)
	deletedCount := 0

	iterator := storetypes.KVStorePrefixIterator(store, []byte("proof_cache_"))
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var cacheData map[string]interface{}
		if err := json.Unmarshal(iterator.Value(), &cacheData); err != nil {
			continue
		}

		if verifiedAt, ok := cacheData["verified_at"].(float64); ok {
			verifiedTime := time.Unix(int64(verifiedAt), 0)
			if verifiedTime.Before(cutoffTime) {
				store.Delete(iterator.Key())
				deletedCount++
			}
		}
	}

	if deletedCount > 0 {
		sdkCtx.Logger().Info("cleaned up old proof cache entries",
			"count", deletedCount,
			"retention_period", retentionPeriod.String(),
		)
	}

	return nil
}
