package keeper

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

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

// CircuitManager manages all ZK circuits for the compute module.
// It provides a unified interface for circuit initialization, proof generation,
// and verification across all circuit types (compute, escrow, result).
type CircuitManager struct {
	keeper *Keeper
	mu     sync.RWMutex

	// Compiled circuits
	computeCircuit constraint.ConstraintSystem
	escrowCircuit  constraint.ConstraintSystem
	resultCircuit  constraint.ConstraintSystem

	// Proving keys (for off-chain proof generation)
	computeProvingKey groth16.ProvingKey
	escrowProvingKey  groth16.ProvingKey
	resultProvingKey  groth16.ProvingKey

	// Verifying keys (for on-chain verification)
	computeVerifyingKey groth16.VerifyingKey
	escrowVerifyingKey  groth16.VerifyingKey
	resultVerifyingKey  groth16.VerifyingKey

	// Circuit metadata
	circuitVersions map[string]string
	initialized     bool
}

// groth16Verify is a function pointer to allow tests to stub verification without
// running heavy cryptographic checks.
var groth16Verify = groth16.Verify
var groth16Setup = groth16.Setup

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

// CircuitType identifies the type of ZK circuit.
type CircuitType string

const (
	CircuitTypeCompute CircuitType = "compute-verification-v2"
	CircuitTypeEscrow  CircuitType = "escrow-release-v1"
	CircuitTypeResult  CircuitType = "result-correctness-v1"
)

// CircuitDefinition provides metadata for circuit initialization.
type CircuitDefinition interface {
	frontend.Circuit
	GetCircuitName() string
	GetPublicInputCount() int
	GetConstraintCount() int
}

// circuitConfig holds initialization config for each circuit type.
type circuitConfig struct {
	circuitType CircuitType
	circuit     CircuitDefinition
	description string
	gasCost     uint64
}

// NewCircuitManager creates a new circuit manager for the keeper.
func NewCircuitManager(keeper *Keeper) *CircuitManager {
	return &CircuitManager{
		keeper:          keeper,
		circuitVersions: make(map[string]string),
	}
}

// Initialize compiles all circuits and generates proving/verifying keys.
// This is an expensive operation (~30-60 seconds) and should be done once during
// module initialization or genesis.
func (cm *CircuitManager) Initialize(ctx context.Context) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if cm.initialized {
		return nil
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.Logger().Info("Initializing ZK circuit manager")

	startTime := time.Now()

	// Initialize compute circuit
	if err := cm.initializeComputeCircuit(ctx); err != nil {
		return fmt.Errorf("failed to initialize compute circuit: %w", err)
	}

	// Initialize escrow circuit
	if err := cm.initializeEscrowCircuit(ctx); err != nil {
		return fmt.Errorf("failed to initialize escrow circuit: %w", err)
	}

	// Initialize result circuit
	if err := cm.initializeResultCircuit(ctx); err != nil {
		return fmt.Errorf("failed to initialize result circuit: %w", err)
	}

	cm.initialized = true

	sdkCtx.Logger().Info("ZK circuit manager initialized",
		"duration", time.Since(startTime).String(),
		"circuits", []string{
			string(CircuitTypeCompute),
			string(CircuitTypeEscrow),
			string(CircuitTypeResult),
		},
	)

	return nil
}

// initializeCircuit is the generic circuit initialization function.
// It compiles the circuit, loads or generates keys, and stores circuit params.
func (cm *CircuitManager) initializeCircuit(ctx context.Context, cfg circuitConfig) (constraint.ConstraintSystem, groth16.ProvingKey, groth16.VerifyingKey, error) {
	circuitID := cfg.circuit.GetCircuitName()

	// Compile circuit to constraint system
	ccs, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, cfg.circuit)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("failed to compile %s circuit: %w", cfg.circuitType, err)
	}

	// Check for existing verifying key in state
	existingParams, err := cm.keeper.GetCircuitParams(ctx, circuitID)
	if err == nil && len(existingParams.VerifyingKey.VkData) > 0 {
		// SECURITY: Verify circuit parameter hash before using existing key
		valid, hashErr := cm.keeper.VerifyCircuitParamsHash(ctx, circuitID, *existingParams)
		if hashErr != nil {
			return nil, nil, nil, fmt.Errorf("failed to verify circuit params hash: %w", hashErr)
		}
		if !valid {
			return nil, nil, nil, fmt.Errorf("circuit parameter hash verification failed for %s - parameters may be invalid or tampered", circuitID)
		}

		// Try to use existing verifying key
		vk := groth16.NewVerifyingKey(ecc.BN254)
		if _, readErr := vk.ReadFrom(bytes.NewReader(existingParams.VerifyingKey.VkData)); readErr == nil {
			// Key loaded successfully, regenerate proving key
			pk, _, setupErr := groth16Setup(ccs)
			if setupErr != nil {
				return nil, nil, nil, fmt.Errorf("failed to regenerate proving key: %w", setupErr)
			}
			cm.circuitVersions[string(cfg.circuitType)] = circuitID
			return ccs, pk, vk, nil
		}
		// Key corrupted, fall through to regenerate
	}

	// Generate new keys and store them
	pk, vk, err := cm.generateAndStoreKeys(ctx, cfg, ccs)
	if err != nil {
		return nil, nil, nil, err
	}

	cm.circuitVersions[string(cfg.circuitType)] = circuitID
	return ccs, pk, vk, nil
}

// generateAndStoreKeys generates new proving/verifying keys and stores circuit params.
func (cm *CircuitManager) generateAndStoreKeys(ctx context.Context, cfg circuitConfig, ccs constraint.ConstraintSystem) (groth16.ProvingKey, groth16.VerifyingKey, error) {
	circuitID := cfg.circuit.GetCircuitName()

	pk, vk, err := groth16Setup(ccs)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to setup %s circuit: %w", cfg.circuitType, err)
	}

	// Serialize verifying key
	vkBytes := new(bytes.Buffer)
	if _, err := vk.WriteTo(vkBytes); err != nil {
		return nil, nil, fmt.Errorf("failed to serialize verifying key: %w", err)
	}

	params := types.CircuitParams{
		CircuitId:   circuitID,
		Description: cfg.description,
		VerifyingKey: types.VerifyingKey{
			CircuitId:        circuitID,
			Curve:            "bn254",
			ProofSystem:      "groth16",
			CreatedAt:        sdk.UnwrapSDKContext(ctx).BlockTime(),
			VkData:           vkBytes.Bytes(),
			PublicInputCount: types.SaturateIntToUint32(cfg.circuit.GetPublicInputCount()),
		},
		MaxProofSize: 2048,
		GasCost:      cfg.gasCost,
		Enabled:      true,
	}

	if err := cm.keeper.SetCircuitParams(ctx, params); err != nil {
		return nil, nil, err
	}

	// SECURITY: Compute and store circuit parameter hash for future verification
	paramHash, err := ComputeCircuitParamsHash(params)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to compute circuit params hash: %w", err)
	}

	if err := cm.keeper.SetCircuitParamHash(ctx, circuitID, paramHash); err != nil {
		return nil, nil, fmt.Errorf("failed to set circuit param hash: %w", err)
	}

	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.Logger().Info("circuit parameters stored with hash verification",
		"circuit_id", circuitID,
		"hash", fmt.Sprintf("%x", paramHash),
	)

	return pk, vk, nil
}

// initializeComputeCircuit compiles the compute verification circuit.
func (cm *CircuitManager) initializeComputeCircuit(ctx context.Context) error {
	cfg := circuitConfig{
		circuitType: CircuitTypeCompute,
		circuit:     &circuits.ComputeCircuit{},
		description: "Compute verification circuit (v2) - proves correct execution of computations",
		gasCost:     500000,
	}
	ccs, pk, vk, err := cm.initializeCircuit(ctx, cfg)
	if err != nil {
		return err
	}
	cm.computeCircuit = ccs
	cm.computeProvingKey = pk
	cm.computeVerifyingKey = vk
	return nil
}

// initializeEscrowCircuit compiles the escrow release circuit.
func (cm *CircuitManager) initializeEscrowCircuit(ctx context.Context) error {
	cfg := circuitConfig{
		circuitType: CircuitTypeEscrow,
		circuit:     &circuits.EscrowCircuit{},
		description: "Escrow release circuit - proves computation completion for fund release",
		gasCost:     450000,
	}
	ccs, pk, vk, err := cm.initializeCircuit(ctx, cfg)
	if err != nil {
		return err
	}
	cm.escrowCircuit = ccs
	cm.escrowProvingKey = pk
	cm.escrowVerifyingKey = vk
	return nil
}

// initializeResultCircuit compiles the result correctness circuit.
func (cm *CircuitManager) initializeResultCircuit(ctx context.Context) error {
	cfg := circuitConfig{
		circuitType: CircuitTypeResult,
		circuit:     &circuits.ResultCircuit{},
		description: "Result correctness circuit - proves result validity with merkle proofs",
		gasCost:     550000,
	}
	ccs, pk, vk, err := cm.initializeCircuit(ctx, cfg)
	if err != nil {
		return err
	}
	cm.resultCircuit = ccs
	cm.resultProvingKey = pk
	cm.resultVerifyingKey = vk
	return nil
}

// VerifyComputeProof verifies a compute verification proof.
func (cm *CircuitManager) VerifyComputeProof(
	ctx sdk.Context,
	proofData []byte,
	publicInputs *ComputePublicInputs,
) (bool, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if !cm.initialized {
		return false, fmt.Errorf("circuit manager not initialized")
	}

	if cm.computeVerifyingKey == nil {
		return false, fmt.Errorf("compute verifying key not available")
	}

	// Consume gas for verification setup
	ctx.GasMeter().ConsumeGas(1000, "zk_compute_proof_setup")

	// Deserialize proof
	proof := groth16.NewProof(ecc.BN254)
	if _, err := proof.ReadFrom(bytes.NewReader(proofData)); err != nil {
		return false, fmt.Errorf("failed to deserialize proof: %w", err)
	}

	// Create public witness
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

	// Consume gas proportional to circuit size
	ctx.GasMeter().ConsumeGas(400000, "zk_compute_proof_verification")

	// Verify the proof
	if err := groth16Verify(proof, cm.computeVerifyingKey, witness); err != nil {
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

// ComputePublicInputs holds the public inputs for compute verification.
type ComputePublicInputs struct {
	RequestID          uint64
	ResultCommitment   interface{} // Field element
	ProviderCommitment interface{} // Field element
	ResourceCommitment interface{} // Field element
}

// VerifyEscrowProof verifies an escrow release proof.
func (cm *CircuitManager) VerifyEscrowProof(
	ctx sdk.Context,
	proofData []byte,
	publicInputs *EscrowPublicInputs,
) (bool, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if !cm.initialized {
		return false, fmt.Errorf("circuit manager not initialized")
	}

	if cm.escrowVerifyingKey == nil {
		return false, fmt.Errorf("escrow verifying key not available")
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

	if err := groth16Verify(proof, cm.escrowVerifyingKey, witness); err != nil {
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

// EscrowPublicInputs holds the public inputs for escrow verification.
type EscrowPublicInputs struct {
	RequestID            uint64
	EscrowAmount         uint64
	RequesterCommitment  interface{}
	ProviderCommitment   interface{}
	CompletionCommitment interface{}
}

// VerifyResultProof verifies a result correctness proof.
func (cm *CircuitManager) VerifyResultProof(
	ctx sdk.Context,
	proofData []byte,
	publicInputs *ResultPublicInputs,
) (bool, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if !cm.initialized {
		return false, fmt.Errorf("circuit manager not initialized")
	}

	if cm.resultVerifyingKey == nil {
		return false, fmt.Errorf("result verifying key not available")
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

	if err := groth16Verify(proof, cm.resultVerifyingKey, witness); err != nil {
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

// ResultPublicInputs holds the public inputs for result verification.
type ResultPublicInputs struct {
	RequestID      uint64
	ResultRootHash interface{}
	InputRootHash  interface{}
	ProgramHash    interface{}
}

// GetCircuitInfo returns information about a specific circuit.
func (cm *CircuitManager) GetCircuitInfo(circuitType CircuitType) (*CircuitInfo, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	var info CircuitInfo
	info.CircuitType = string(circuitType)

	switch circuitType {
	case CircuitTypeCompute:
		circuit := &circuits.ComputeCircuit{}
		info.CircuitID = circuit.GetCircuitName()
		info.ConstraintCount = circuit.GetConstraintCount()
		info.PublicInputCount = circuit.GetPublicInputCount()
		info.Initialized = cm.computeVerifyingKey != nil
	case CircuitTypeEscrow:
		circuit := &circuits.EscrowCircuit{}
		info.CircuitID = circuit.GetCircuitName()
		info.ConstraintCount = circuit.GetConstraintCount()
		info.PublicInputCount = circuit.GetPublicInputCount()
		info.Initialized = cm.escrowVerifyingKey != nil
	case CircuitTypeResult:
		circuit := &circuits.ResultCircuit{}
		info.CircuitID = circuit.GetCircuitName()
		info.ConstraintCount = circuit.GetConstraintCount()
		info.PublicInputCount = circuit.GetPublicInputCount()
		info.Initialized = cm.resultVerifyingKey != nil
	default:
		return nil, fmt.Errorf("unknown circuit type: %s", circuitType)
	}

	return &info, nil
}

// CircuitInfo provides metadata about a circuit.
type CircuitInfo struct {
	CircuitType      string
	CircuitID        string
	ConstraintCount  int
	PublicInputCount int
	Initialized      bool
}

// GetAllCircuitInfo returns information about all circuits.
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

// IsInitialized returns whether the circuit manager has been initialized.
func (cm *CircuitManager) IsInitialized() bool {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.initialized
}

// ExportVerifyingKeys exports all verifying keys for external use (e.g., client-side verification).
func (cm *CircuitManager) ExportVerifyingKeys() (map[string][]byte, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if !cm.initialized {
		return nil, fmt.Errorf("circuit manager not initialized")
	}

	keys := make(map[string][]byte)

	if cm.computeVerifyingKey != nil {
		buf := new(bytes.Buffer)
		if _, err := cm.computeVerifyingKey.WriteTo(buf); err == nil {
			keys[string(CircuitTypeCompute)] = buf.Bytes()
		}
	}

	if cm.escrowVerifyingKey != nil {
		buf := new(bytes.Buffer)
		if _, err := cm.escrowVerifyingKey.WriteTo(buf); err == nil {
			keys[string(CircuitTypeEscrow)] = buf.Bytes()
		}
	}

	if cm.resultVerifyingKey != nil {
		buf := new(bytes.Buffer)
		if _, err := cm.resultVerifyingKey.WriteTo(buf); err == nil {
			keys[string(CircuitTypeResult)] = buf.Bytes()
		}
	}

	return keys, nil
}

// GetVerifyingKey returns the verifying key for the requested circuit.
func (cm *CircuitManager) GetVerifyingKey(ctx context.Context, circuitID string) (*groth16.VerifyingKey, error) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	if !cm.initialized {
		return nil, fmt.Errorf("circuit manager not initialized")
	}

	switch circuitID {
	case (&circuits.ComputeCircuit{}).GetCircuitName():
		if cm.computeVerifyingKey == nil {
			return nil, fmt.Errorf("compute verifying key unavailable")
		}
		vk := cm.computeVerifyingKey
		return &vk, nil
	case (&circuits.EscrowCircuit{}).GetCircuitName():
		if cm.escrowVerifyingKey == nil {
			return nil, fmt.Errorf("escrow verifying key unavailable")
		}
		vk := cm.escrowVerifyingKey
		return &vk, nil
	case (&circuits.ResultCircuit{}).GetCircuitName():
		if cm.resultVerifyingKey == nil {
			return nil, fmt.Errorf("result verifying key unavailable")
		}
		vk := cm.resultVerifyingKey
		return &vk, nil
	default:
		return nil, fmt.Errorf("unknown circuit: %s", circuitID)
	}
}

// GetCircuitStats returns statistics about circuit usage.
func (cm *CircuitManager) GetCircuitStats(ctx context.Context) (*CircuitStats, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := sdkCtx.KVStore(cm.keeper.storeKey)

	statsKey := []byte("circuit_stats_global")
	bz := store.Get(statsKey)

	if bz == nil {
		return &CircuitStats{}, nil
	}

	var stats CircuitStats
	if err := json.Unmarshal(bz, &stats); err != nil {
		return nil, err
	}

	return &stats, nil
}

// CircuitStats tracks circuit usage metrics.
type CircuitStats struct {
	TotalComputeProofs  uint64 `json:"total_compute_proofs"`
	TotalEscrowProofs   uint64 `json:"total_escrow_proofs"`
	TotalResultProofs   uint64 `json:"total_result_proofs"`
	FailedVerifications uint64 `json:"failed_verifications"`
	TotalGasConsumed    uint64 `json:"total_gas_consumed"`
	LastUpdated         int64  `json:"last_updated"`
}
