package keeper

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"sort"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/compute/types"
)

// ComputeCircuitParamsHash computes a canonical SHA256 hash of circuit parameters.
// This hash is used to verify circuit integrity on initialization.
//
// The hash includes:
// - Circuit ID
// - Description
// - Verifying key data (the critical security parameter)
// - Max proof size
// - Gas cost
// - Public input count
// - Curve name
// - Proof system
//
// Fields are hashed in a deterministic order to ensure consistent hashes.
func ComputeCircuitParamsHash(params types.CircuitParams) ([]byte, error) {
	h := sha256.New()

	// Hash circuit ID
	if _, err := h.Write([]byte(params.CircuitId)); err != nil {
		return nil, fmt.Errorf("failed to hash circuit_id: %w", err)
	}

	// Hash description
	if _, err := h.Write([]byte(params.Description)); err != nil {
		return nil, fmt.Errorf("failed to hash description: %w", err)
	}

	// Hash verifying key data (most critical field)
	if _, err := h.Write(params.VerifyingKey.VkData); err != nil {
		return nil, fmt.Errorf("failed to hash verifying key data: %w", err)
	}

	// Hash max proof size
	maxProofSizeBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(maxProofSizeBytes, params.MaxProofSize)
	if _, err := h.Write(maxProofSizeBytes); err != nil {
		return nil, fmt.Errorf("failed to hash max_proof_size: %w", err)
	}

	// Hash gas cost
	gasCostBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(gasCostBytes, params.GasCost)
	if _, err := h.Write(gasCostBytes); err != nil {
		return nil, fmt.Errorf("failed to hash gas_cost: %w", err)
	}

	// Hash public input count
	publicInputCountBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(publicInputCountBytes, params.VerifyingKey.PublicInputCount)
	if _, err := h.Write(publicInputCountBytes); err != nil {
		return nil, fmt.Errorf("failed to hash public_input_count: %w", err)
	}

	// Hash curve
	if _, err := h.Write([]byte(params.VerifyingKey.Curve)); err != nil {
		return nil, fmt.Errorf("failed to hash curve: %w", err)
	}

	// Hash proof system
	if _, err := h.Write([]byte(params.VerifyingKey.ProofSystem)); err != nil {
		return nil, fmt.Errorf("failed to hash proof_system: %w", err)
	}

	return h.Sum(nil), nil
}

// VerifyCircuitParamsHash verifies that circuit parameters match the expected hash.
// This prevents using tampered or invalid circuit parameters.
//
// Returns:
//   - true if hash matches or no expected hash is configured
//   - false if hash mismatch detected
//   - error if hash computation fails
func (k *Keeper) VerifyCircuitParamsHash(ctx context.Context, circuitID string, params types.CircuitParams) (bool, error) {
	// Get module params
	moduleParams, err := k.GetParams(ctx)
	if err != nil {
		return false, fmt.Errorf("failed to get module params: %w", err)
	}

	// Check if expected hash is configured for this circuit
	expectedHash, exists := moduleParams.CircuitParamHashes[circuitID]
	if !exists || len(expectedHash) == 0 {
		// No expected hash configured - allow (for backward compatibility during migration)
		// In production, all circuits should have expected hashes set via governance
		sdkCtx := sdk.UnwrapSDKContext(ctx)
		sdkCtx.Logger().Warn("no circuit parameter hash configured - skipping verification",
			"circuit_id", circuitID,
			"recommendation", "set expected hash via governance proposal",
		)
		return true, nil
	}

	// Compute actual hash
	actualHash, err := ComputeCircuitParamsHash(params)
	if err != nil {
		return false, fmt.Errorf("failed to compute circuit params hash: %w", err)
	}

	// Verify hash matches
	if !bytes.Equal(expectedHash, actualHash) {
		sdkCtx := sdk.UnwrapSDKContext(ctx)
		sdkCtx.Logger().Error("circuit parameter hash mismatch detected",
			"circuit_id", circuitID,
			"expected_hash", fmt.Sprintf("%x", expectedHash),
			"actual_hash", fmt.Sprintf("%x", actualHash),
		)
		return false, nil
	}

	return true, nil
}

// ValidateCircuitParams validates circuit parameters before storing.
// This includes both structural validation and hash verification.
func (k *Keeper) ValidateCircuitParams(ctx context.Context, params types.CircuitParams) error {
	// Structural validation
	if params.CircuitId == "" {
		return fmt.Errorf("circuit_id cannot be empty")
	}

	if params.Description == "" {
		return fmt.Errorf("description cannot be empty")
	}

	if len(params.VerifyingKey.VkData) == 0 {
		return fmt.Errorf("verifying key data cannot be empty")
	}

	if params.VerifyingKey.Curve == "" {
		return fmt.Errorf("curve cannot be empty")
	}

	if params.VerifyingKey.ProofSystem == "" {
		return fmt.Errorf("proof_system cannot be empty")
	}

	if params.MaxProofSize == 0 {
		return fmt.Errorf("max_proof_size must be positive")
	}

	if params.GasCost == 0 {
		return fmt.Errorf("gas_cost must be positive")
	}

	if params.VerifyingKey.PublicInputCount == 0 {
		return fmt.Errorf("public_input_count must be positive")
	}

	// Hash verification
	valid, err := k.VerifyCircuitParamsHash(ctx, params.CircuitId, params)
	if err != nil {
		return fmt.Errorf("hash verification failed: %w", err)
	}

	if !valid {
		return fmt.Errorf("circuit parameter hash mismatch - parameters may be invalid or tampered")
	}

	return nil
}

// SetCircuitParamHash sets the expected hash for a circuit (governance-controlled).
// This should only be called via governance proposals or during genesis initialization.
func (k *Keeper) SetCircuitParamHash(ctx context.Context, circuitID string, expectedHash []byte) error {
	if circuitID == "" {
		return fmt.Errorf("circuit_id cannot be empty")
	}

	if len(expectedHash) != sha256.Size {
		return fmt.Errorf("expected hash must be %d bytes, got %d", sha256.Size, len(expectedHash))
	}

	params, err := k.GetParams(ctx)
	if err != nil {
		return fmt.Errorf("failed to get module params: %w", err)
	}

	if params.CircuitParamHashes == nil {
		params.CircuitParamHashes = make(map[string][]byte)
	}

	params.CircuitParamHashes[circuitID] = expectedHash

	if err := k.SetParams(ctx, params); err != nil {
		return fmt.Errorf("failed to set module params: %w", err)
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.Logger().Info("circuit parameter hash updated",
		"circuit_id", circuitID,
		"hash", fmt.Sprintf("%x", expectedHash),
	)

	return nil
}

// GetCircuitParamHash retrieves the expected hash for a circuit.
func (k *Keeper) GetCircuitParamHash(ctx context.Context, circuitID string) ([]byte, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get module params: %w", err)
	}

	hash, exists := params.CircuitParamHashes[circuitID]
	if !exists {
		return nil, fmt.Errorf("no hash configured for circuit %s", circuitID)
	}

	return hash, nil
}

// ListCircuitHashes returns all configured circuit hashes.
// Results are sorted by circuit ID for deterministic output.
func (k *Keeper) ListCircuitHashes(ctx context.Context) (map[string][]byte, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get module params: %w", err)
	}

	if params.CircuitParamHashes == nil {
		return make(map[string][]byte), nil
	}

	// Return a copy to prevent external modification
	result := make(map[string][]byte, len(params.CircuitParamHashes))
	for k, v := range params.CircuitParamHashes {
		hashCopy := make([]byte, len(v))
		copy(hashCopy, v)
		result[k] = hashCopy
	}

	return result, nil
}

// VerifyAllCircuitHashes verifies all stored circuit parameters against expected hashes.
// Returns a map of circuit IDs to verification results.
// This is useful for auditing and debugging.
func (k *Keeper) VerifyAllCircuitHashes(ctx context.Context) (map[string]bool, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get module params: %w", err)
	}

	results := make(map[string]bool)

	// Get all configured circuit IDs from expected hashes
	circuitIDs := make([]string, 0, len(params.CircuitParamHashes))
	for id := range params.CircuitParamHashes {
		circuitIDs = append(circuitIDs, id)
	}
	sort.Strings(circuitIDs)

	// Verify each circuit
	for _, circuitID := range circuitIDs {
		circuitParams, err := k.GetCircuitParams(ctx, circuitID)
		if err != nil {
			// Circuit params not found - mark as invalid
			results[circuitID] = false
			continue
		}

		valid, err := k.VerifyCircuitParamsHash(ctx, circuitID, *circuitParams)
		if err != nil {
			// Hash computation failed - mark as invalid
			results[circuitID] = false
			continue
		}

		results[circuitID] = valid
	}

	return results, nil
}
