package keeper

import (
	"bytes"
	"context"
	"fmt"
	"sync"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/backend"
	"github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark/backend/witness"
	"github.com/consensys/gnark/constraint"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/frontend/cs/r1cs"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/compute/circuits"
	"github.com/paw-chain/paw/x/compute/types"
)

// Circuit type constants
type CircuitType string

const (
	CircuitTypeCompute CircuitType = "compute-verification-v2"
	CircuitTypeEscrow  CircuitType = "escrow-release-v1"
	CircuitTypeResult  CircuitType = "result-correctness-v1"
)

// circuitDef holds static circuit definition and metadata
type circuitDef struct {
	id              string
	description     string
	gasCost         uint64
	constraintCount int
	publicInputs    int
	newInstance     func() frontend.Circuit
}

// Static circuit definitions (3 fixed circuits)
var (
	computeCircuitDef = circuitDef{
		id:              "compute-verification-v2",
		description:     "Compute verification circuit (v2) - proves correct execution of computations",
		gasCost:         500000,
		constraintCount: 40000,
		publicInputs:    4,
		newInstance:     func() frontend.Circuit { return &circuits.ComputeCircuit{} },
	}

	escrowCircuitDef = circuitDef{
		id:              "escrow-release-v1",
		description:     "Escrow release circuit - proves computation completion for fund release",
		gasCost:         450000,
		constraintCount: 26000,
		publicInputs:    5,
		newInstance:     func() frontend.Circuit { return &circuits.EscrowCircuit{} },
	}

	resultCircuitDef = circuitDef{
		id:              "result-correctness-v1",
		description:     "Result correctness circuit - proves result validity with merkle proofs",
		gasCost:         550000,
		constraintCount: 45000,
		publicInputs:    4,
		newInstance:     func() frontend.Circuit { return &circuits.ResultCircuit{} },
	}
)

// Package-level circuit state (protected by mutex)
var (
	circuitMu       sync.RWMutex
	circuitState    = make(map[string]*circuitKeys)
	circuitsInitialized bool
)

// circuitKeys holds the compiled circuit and keys for a single circuit
type circuitKeys struct {
	ccs constraint.ConstraintSystem
	pk  groth16.ProvingKey
	vk  groth16.VerifyingKey
}

// Function variables for testing
var (
	groth16Verify = groth16.Verify
	groth16Setup  = groth16.Setup
)

// SetGroth16Setup allows tests to stub key generation for fast execution.
func SetGroth16Setup(fn func(constraint.ConstraintSystem) (groth16.ProvingKey, groth16.VerifyingKey, error)) {
	groth16Setup = fn
}

// Groth16SetupFunc exposes the current setup function (useful for restoring after stubs).
func Groth16SetupFunc() func(constraint.ConstraintSystem) (groth16.ProvingKey, groth16.VerifyingKey, error) {
	return groth16Setup
}

// SetGroth16Verify allows tests to stub proof verification.
func SetGroth16Verify(fn func(groth16.Proof, groth16.VerifyingKey, witness.Witness, ...backend.VerifierOption) error) {
	groth16Verify = fn
}

// Groth16VerifyFunc exposes the current verify function (useful for restoring after stubs).
func Groth16VerifyFunc() func(groth16.Proof, groth16.VerifyingKey, witness.Witness, ...backend.VerifierOption) error {
	return groth16Verify
}

// CircuitManager provides a minimal interface for backward compatibility
type CircuitManager struct {
	keeper *Keeper
}

// NewCircuitManager creates a circuit manager instance
func NewCircuitManager(keeper *Keeper) *CircuitManager {
	return &CircuitManager{keeper: keeper}
}

// Initialize compiles all circuits and generates keys (called once at startup)
func (cm *CircuitManager) Initialize(ctx context.Context) error {
	circuitMu.Lock()
	defer circuitMu.Unlock()

	if circuitsInitialized {
		return nil
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.Logger().Info("Initializing ZK circuits")

	// Initialize all three circuits
	defs := []circuitDef{computeCircuitDef, escrowCircuitDef, resultCircuitDef}
	for _, def := range defs {
		if err := initializeCircuit(ctx, cm.keeper, def); err != nil {
			return fmt.Errorf("failed to initialize %s: %w", def.id, err)
		}
	}

	circuitsInitialized = true
	sdkCtx.Logger().Info("ZK circuits initialized", "count", len(defs))
	return nil
}

// initializeCircuit compiles a circuit and generates/loads keys
func initializeCircuit(ctx context.Context, k *Keeper, def circuitDef) error {
	// Compile circuit
	circuit := def.newInstance()
	ccs, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, circuit)
	if err != nil {
		return fmt.Errorf("failed to compile circuit: %w", err)
	}

	// Try to load existing verifying key
	existingParams, err := k.GetCircuitParams(ctx, def.id)
	if err == nil && len(existingParams.VerifyingKey.VkData) > 0 {
		// SECURITY: Verify circuit parameter hash
		valid, hashErr := k.VerifyCircuitParamsHash(ctx, def.id, *existingParams)
		if hashErr == nil && valid {
			// Try to deserialize existing VK
			vk := groth16.NewVerifyingKey(ecc.BN254)
			if _, readErr := vk.ReadFrom(bytes.NewReader(existingParams.VerifyingKey.VkData)); readErr == nil {
				// Regenerate PK (not stored on-chain)
				pk, _, setupErr := groth16Setup(ccs)
				if setupErr != nil {
					return fmt.Errorf("failed to regenerate proving key: %w", setupErr)
				}
				circuitState[def.id] = &circuitKeys{ccs: ccs, pk: pk, vk: vk}
				return nil
			}
		}
	}

	// Generate new keys and store them
	pk, vk, err := groth16Setup(ccs)
	if err != nil {
		return fmt.Errorf("failed to setup circuit: %w", err)
	}

	// Serialize verifying key for storage
	vkBytes := new(bytes.Buffer)
	if _, err := vk.WriteTo(vkBytes); err != nil {
		return fmt.Errorf("failed to serialize verifying key: %w", err)
	}

	params := types.CircuitParams{
		CircuitId:   def.id,
		Description: def.description,
		VerifyingKey: types.VerifyingKey{
			CircuitId:        def.id,
			Curve:            "bn254",
			ProofSystem:      "groth16",
			CreatedAt:        sdk.UnwrapSDKContext(ctx).BlockTime(),
			VkData:           vkBytes.Bytes(),
			PublicInputCount: types.SaturateIntToUint32(def.publicInputs),
		},
		MaxProofSize: 2048,
		GasCost:      def.gasCost,
		Enabled:      true,
	}

	if err := k.SetCircuitParams(ctx, params); err != nil {
		return err
	}

	// SECURITY: Compute and store circuit parameter hash
	paramHash, err := ComputeCircuitParamsHash(params)
	if err != nil {
		return fmt.Errorf("failed to compute circuit params hash: %w", err)
	}

	if err := k.SetCircuitParamHash(ctx, def.id, paramHash); err != nil {
		return fmt.Errorf("failed to set circuit param hash: %w", err)
	}

	circuitState[def.id] = &circuitKeys{ccs: ccs, pk: pk, vk: vk}
	return nil
}

// IsInitialized returns whether circuits have been initialized
func (cm *CircuitManager) IsInitialized() bool {
	circuitMu.RLock()
	defer circuitMu.RUnlock()
	return circuitsInitialized
}

// Public input types for each circuit
type ComputePublicInputs struct {
	RequestID          uint64
	ResultCommitment   interface{}
	ProviderCommitment interface{}
	ResourceCommitment interface{}
}

type EscrowPublicInputs struct {
	RequestID            uint64
	EscrowAmount         uint64
	RequesterCommitment  interface{}
	ProviderCommitment   interface{}
	CompletionCommitment interface{}
}

type ResultPublicInputs struct {
	RequestID      uint64
	ResultRootHash interface{}
	InputRootHash  interface{}
	ProgramHash    interface{}
}

// VerifyComputeProof verifies a compute verification proof
func (cm *CircuitManager) VerifyComputeProof(
	ctx sdk.Context,
	proofData []byte,
	publicInputs *ComputePublicInputs,
) (bool, error) {
	circuitMu.RLock()
	keys := circuitState[computeCircuitDef.id]
	circuitMu.RUnlock()

	if keys == nil || keys.vk == nil {
		return false, fmt.Errorf("compute circuit not initialized")
	}

	ctx.GasMeter().ConsumeGas(1000, "zk_compute_proof_setup")

	proof := groth16.NewProof(ecc.BN254)
	if _, err := proof.ReadFrom(bytes.NewReader(proofData)); err != nil {
		return false, fmt.Errorf("failed to deserialize proof: %w", err)
	}

	assignment := &circuits.ComputeCircuit{
		RequestID:          publicInputs.RequestID,
		ResultCommitment:   publicInputs.ResultCommitment,
		ProviderCommitment: publicInputs.ProviderCommitment,
		ResourceCommitment: publicInputs.ResourceCommitment,
	}

	witness, err := frontend.NewWitness(assignment, ecc.BN254.ScalarField(), frontend.PublicOnly())
	if err != nil {
		return false, fmt.Errorf("failed to create witness: %w", err)
	}

	ctx.GasMeter().ConsumeGas(400000, "zk_compute_proof_verification")

	if err := groth16Verify(proof, keys.vk, witness); err != nil {
		ctx.Logger().Warn("compute proof verification failed",
			"request_id", publicInputs.RequestID,
			"error", err.Error(),
		)
		return false, nil
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"zk_compute_proof_verified",
			sdk.NewAttribute("request_id", fmt.Sprintf("%d", publicInputs.RequestID)),
			sdk.NewAttribute("circuit", string(CircuitTypeCompute)),
		),
	)

	return true, nil
}

// VerifyEscrowProof verifies an escrow release proof
func (cm *CircuitManager) VerifyEscrowProof(
	ctx sdk.Context,
	proofData []byte,
	publicInputs *EscrowPublicInputs,
) (bool, error) {
	circuitMu.RLock()
	keys := circuitState[escrowCircuitDef.id]
	circuitMu.RUnlock()

	if keys == nil || keys.vk == nil {
		return false, fmt.Errorf("escrow circuit not initialized")
	}

	ctx.GasMeter().ConsumeGas(1000, "zk_escrow_proof_setup")

	proof := groth16.NewProof(ecc.BN254)
	if _, err := proof.ReadFrom(bytes.NewReader(proofData)); err != nil {
		return false, fmt.Errorf("failed to deserialize proof: %w", err)
	}

	assignment := &circuits.EscrowCircuit{
		RequestID:            publicInputs.RequestID,
		EscrowAmount:         publicInputs.EscrowAmount,
		RequesterCommitment:  publicInputs.RequesterCommitment,
		ProviderCommitment:   publicInputs.ProviderCommitment,
		CompletionCommitment: publicInputs.CompletionCommitment,
	}

	witness, err := frontend.NewWitness(assignment, ecc.BN254.ScalarField(), frontend.PublicOnly())
	if err != nil {
		return false, fmt.Errorf("failed to create witness: %w", err)
	}

	ctx.GasMeter().ConsumeGas(350000, "zk_escrow_proof_verification")

	if err := groth16Verify(proof, keys.vk, witness); err != nil {
		ctx.Logger().Warn("escrow proof verification failed",
			"request_id", publicInputs.RequestID,
			"error", err.Error(),
		)
		return false, nil
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"zk_escrow_proof_verified",
			sdk.NewAttribute("request_id", fmt.Sprintf("%d", publicInputs.RequestID)),
			sdk.NewAttribute("escrow_amount", fmt.Sprintf("%d", publicInputs.EscrowAmount)),
			sdk.NewAttribute("circuit", string(CircuitTypeEscrow)),
		),
	)

	return true, nil
}

// VerifyResultProof verifies a result correctness proof
func (cm *CircuitManager) VerifyResultProof(
	ctx sdk.Context,
	proofData []byte,
	publicInputs *ResultPublicInputs,
) (bool, error) {
	circuitMu.RLock()
	keys := circuitState[resultCircuitDef.id]
	circuitMu.RUnlock()

	if keys == nil || keys.vk == nil {
		return false, fmt.Errorf("result circuit not initialized")
	}

	ctx.GasMeter().ConsumeGas(1000, "zk_result_proof_setup")

	proof := groth16.NewProof(ecc.BN254)
	if _, err := proof.ReadFrom(bytes.NewReader(proofData)); err != nil {
		return false, fmt.Errorf("failed to deserialize proof: %w", err)
	}

	assignment := &circuits.ResultCircuit{
		RequestID:      publicInputs.RequestID,
		ResultRootHash: publicInputs.ResultRootHash,
		InputRootHash:  publicInputs.InputRootHash,
		ProgramHash:    publicInputs.ProgramHash,
	}

	witness, err := frontend.NewWitness(assignment, ecc.BN254.ScalarField(), frontend.PublicOnly())
	if err != nil {
		return false, fmt.Errorf("failed to create witness: %w", err)
	}

	ctx.GasMeter().ConsumeGas(450000, "zk_result_proof_verification")

	if err := groth16Verify(proof, keys.vk, witness); err != nil {
		ctx.Logger().Warn("result proof verification failed",
			"request_id", publicInputs.RequestID,
			"error", err.Error(),
		)
		return false, nil
	}

	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"zk_result_proof_verified",
			sdk.NewAttribute("request_id", fmt.Sprintf("%d", publicInputs.RequestID)),
			sdk.NewAttribute("circuit", string(CircuitTypeResult)),
		),
	)

	return true, nil
}

// CircuitInfo provides metadata about a circuit
type CircuitInfo struct {
	CircuitType      string
	CircuitID        string
	ConstraintCount  int
	PublicInputCount int
	Initialized      bool
}

// GetCircuitInfo returns information about a specific circuit
func (cm *CircuitManager) GetCircuitInfo(circuitType CircuitType) (*CircuitInfo, error) {
	var def circuitDef

	switch circuitType {
	case CircuitTypeCompute:
		def = computeCircuitDef
	case CircuitTypeEscrow:
		def = escrowCircuitDef
	case CircuitTypeResult:
		def = resultCircuitDef
	default:
		return nil, fmt.Errorf("unknown circuit type: %s", circuitType)
	}

	circuitMu.RLock()
	initialized := circuitState[def.id] != nil && circuitState[def.id].vk != nil
	circuitMu.RUnlock()

	return &CircuitInfo{
		CircuitType:      string(circuitType),
		CircuitID:        def.id,
		ConstraintCount:  def.constraintCount,
		PublicInputCount: def.publicInputs,
		Initialized:      initialized,
	}, nil
}

// GetAllCircuitInfo returns information about all circuits
func (cm *CircuitManager) GetAllCircuitInfo() []*CircuitInfo {
	circuitTypes := []CircuitType{CircuitTypeCompute, CircuitTypeEscrow, CircuitTypeResult}
	infos := make([]*CircuitInfo, 0, len(circuitTypes))

	for _, ct := range circuitTypes {
		if info, err := cm.GetCircuitInfo(ct); err == nil {
			infos = append(infos, info)
		}
	}

	return infos
}

// ExportVerifyingKeys exports all verifying keys for external use
func (cm *CircuitManager) ExportVerifyingKeys() (map[string][]byte, error) {
	circuitMu.RLock()
	defer circuitMu.RUnlock()

	if !circuitsInitialized {
		return nil, fmt.Errorf("circuits not initialized")
	}

	keys := make(map[string][]byte)
	defs := []struct {
		typ CircuitType
		id  string
	}{
		{CircuitTypeCompute, computeCircuitDef.id},
		{CircuitTypeEscrow, escrowCircuitDef.id},
		{CircuitTypeResult, resultCircuitDef.id},
	}

	for _, d := range defs {
		if state := circuitState[d.id]; state != nil && state.vk != nil {
			buf := new(bytes.Buffer)
			if _, err := state.vk.WriteTo(buf); err == nil {
				keys[string(d.typ)] = buf.Bytes()
			}
		}
	}

	return keys, nil
}

// GetVerifyingKey returns the verifying key for a circuit
func (cm *CircuitManager) GetVerifyingKey(ctx context.Context, circuitID string) (*groth16.VerifyingKey, error) {
	circuitMu.RLock()
	defer circuitMu.RUnlock()

	if !circuitsInitialized {
		return nil, fmt.Errorf("circuits not initialized")
	}

	state := circuitState[circuitID]
	if state == nil || state.vk == nil {
		return nil, fmt.Errorf("verifying key unavailable for circuit: %s", circuitID)
	}

	vk := state.vk
	return &vk, nil
}

// CircuitStats tracks circuit usage metrics
type CircuitStats struct {
	TotalComputeProofs  uint64 `json:"total_compute_proofs"`
	TotalEscrowProofs   uint64 `json:"total_escrow_proofs"`
	TotalResultProofs   uint64 `json:"total_result_proofs"`
	FailedVerifications uint64 `json:"failed_verifications"`
	TotalGasConsumed    uint64 `json:"total_gas_consumed"`
	LastUpdated         int64  `json:"last_updated"`
}

// GetCircuitStats returns statistics about circuit usage (stub for compatibility)
func (cm *CircuitManager) GetCircuitStats(ctx context.Context) (*CircuitStats, error) {
	// This could be enhanced to track actual stats if needed
	return &CircuitStats{}, nil
}
