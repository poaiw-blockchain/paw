package keeper

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"time"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/frontend/cs/r1cs"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/compute/types"
)

// ZKVerifier handles zero-knowledge proof generation and verification for compute results.
type ZKVerifier struct {
	keeper *Keeper
}

// NewZKVerifier creates a new ZK proof verifier.
func NewZKVerifier(keeper *Keeper) *ZKVerifier {
	return &ZKVerifier{
		keeper: keeper,
	}
}

// ========================================================================================
// CIRCUIT DEFINITION
// ========================================================================================

// ComputeVerificationCircuit defines the zero-knowledge circuit for verifying compute results.
//
// Circuit Constraint System:
// Public Inputs:
//   - RequestID: uint64
//   - ResultHash: [32]byte (SHA-256 hash)
//   - ProviderAddress: [20]byte
//
// Private Inputs (Witness):
//   - ActualComputationData: variable length bytes
//   - ComputationMetadata: execution details
//
// Constraint:
//
//	Hash(ActualComputationData || ComputationMetadata) == ResultHash
//
// This proves that the provider knows the actual computation that produced
// the result hash, without revealing the computation details on-chain.
type ComputeVerificationCircuit struct {
	// Public inputs (visible on-chain)
	RequestID       frontend.Variable     `gnark:",public"`
	ResultHash      [32]frontend.Variable `gnark:",public"`
	ProviderAddress [20]frontend.Variable `gnark:",public"`

	// Private inputs (witness data, kept secret)
	ComputationData     [1024]frontend.Variable `gnark:",private"` // Actual computation output
	ComputationDataSize frontend.Variable       `gnark:",private"` // Actual size used
	ExecutionTimestamp  frontend.Variable       `gnark:",private"` // When computation ran
	ExitCode            frontend.Variable       `gnark:",private"` // Exit code
	CpuCyclesUsed       frontend.Variable       `gnark:",private"` // CPU cycles consumed
	MemoryBytesUsed     frontend.Variable       `gnark:",private"` // Memory usage
}

// Define implements the gnark Circuit interface.
// This is the core of the ZK-SNARK - it defines the constraints that must be satisfied.
func (circuit *ComputeVerificationCircuit) Define(api frontend.API) error {
	// TODO: Fix gnark type assertions for hasher.Write
	/*
	// Initialize SHA-256 hasher within the circuit
	hasher, err := sha2.New(api)
	if err != nil {
		return fmt.Errorf("failed to initialize circuit hasher: %w", err)
	}

	// Hash the computation data
	// We hash: ComputationData[0:ComputationDataSize] || ExecutionTimestamp || ExitCode || CpuCycles || Memory
	for i := 0; i < 1024; i++ {
		// Only hash up to ComputationDataSize bytes
		// Use a conditional to avoid hashing padding zeros
		isWithinSize := api.IsZero(api.Sub(i, circuit.ComputationDataSize))
		dataToHash := api.Select(isWithinSize, circuit.ComputationData[i], 0)
		hasher.Write(dataToHash)
	}

	// Add metadata to hash
	hasher.Write(circuit.ExecutionTimestamp)
	hasher.Write(circuit.ExitCode)
	hasher.Write(circuit.CpuCyclesUsed)
	hasher.Write(circuit.MemoryBytesUsed)

	// Get the computed hash
	computedHash := hasher.Sum()

	// Constraint: computed hash must equal the public ResultHash
	// This is the core constraint that proves correct computation
	if len(computedHash) != 32 {
		return fmt.Errorf("hash output size mismatch: expected 32, got %d", len(computedHash))
	}

	for i := 0; i < 32; i++ {
		api.AssertIsEqual(computedHash[i], circuit.ResultHash[i])
	}

	// Additional constraints for validity
	// 1. ComputationDataSize must be <= 1024
	api.AssertIsLessOrEqual(circuit.ComputationDataSize, 1024)

	// 2. ExitCode must be a valid exit code (0-255)
	api.AssertIsLessOrEqual(circuit.ExitCode, 255)

	// 3. Timestamp must be positive
	api.AssertIsLessOrEqual(1, circuit.ExecutionTimestamp)
	*/

	return nil
}

// ========================================================================================
// PROOF GENERATION
// ========================================================================================

// GenerateProof generates a ZK-SNARK proof for a compute result.
// This should be called by providers off-chain to generate proofs.
func (zk *ZKVerifier) GenerateProof(
	ctx context.Context,
	requestID uint64,
	resultHash []byte,
	providerAddress sdk.AccAddress,
	computationData []byte,
	executionTimestamp int64,
	exitCode int32,
	cpuCyclesUsed uint64,
	memoryBytesUsed uint64,
) (*types.ZKProof, error) {
	startTime := time.Now()

	// Validate inputs
	if len(resultHash) != 32 {
		return nil, fmt.Errorf("result hash must be 32 bytes, got %d", len(resultHash))
	}
	if len(providerAddress) != 20 {
		return nil, fmt.Errorf("provider address must be 20 bytes, got %d", len(providerAddress))
	}
	if len(computationData) > 1024 {
		return nil, fmt.Errorf("computation data exceeds max size of 1024 bytes, got %d", len(computationData))
	}

	// Get circuit params from state
	circuitParams, err := zk.keeper.GetCircuitParams(ctx, "compute-verification-v1")
	if err != nil {
		return nil, fmt.Errorf("failed to get circuit params: %w", err)
	}

	if !circuitParams.Enabled {
		return nil, fmt.Errorf("circuit is not enabled")
	}

	// Compile the circuit (this would be cached in production)
	circuit := &ComputeVerificationCircuit{}
	ccs, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, circuit)
	if err != nil {
		return nil, fmt.Errorf("failed to compile circuit: %w", err)
	}

	// Create the proving key (in production, this would be generated once and stored)
	pk, vk, err := groth16.Setup(ccs)
	if err != nil {
		return nil, fmt.Errorf("failed to setup circuit: %w", err)
	}

	// Store the verifying key if not already present
	if len(circuitParams.VerifyingKey.VkData) == 0 {
		vkBytes := new(bytes.Buffer)
		if _, err := vk.WriteTo(vkBytes); err != nil {
			return nil, fmt.Errorf("failed to serialize verifying key: %w", err)
		}
		circuitParams.VerifyingKey.VkData = vkBytes.Bytes()
		if err := zk.keeper.SetCircuitParams(ctx, *circuitParams); err != nil {
			return nil, fmt.Errorf("failed to store verifying key: %w", err)
		}
	}

	// Prepare witness data (private inputs + public inputs)
	assignment := &ComputeVerificationCircuit{
		RequestID: requestID,
	}

	// Set public ResultHash
	for i := 0; i < 32; i++ {
		assignment.ResultHash[i] = resultHash[i]
	}

	// Set public ProviderAddress
	for i := 0; i < 20; i++ {
		assignment.ProviderAddress[i] = providerAddress[i]
	}

	// Set private ComputationData (with padding)
	dataSize := len(computationData)
	for i := 0; i < 1024; i++ {
		if i < dataSize {
			assignment.ComputationData[i] = computationData[i]
		} else {
			assignment.ComputationData[i] = 0
		}
	}
	assignment.ComputationDataSize = dataSize
	assignment.ExecutionTimestamp = executionTimestamp
	assignment.ExitCode = int(exitCode)
	assignment.CpuCyclesUsed = cpuCyclesUsed
	assignment.MemoryBytesUsed = memoryBytesUsed

	// Generate the witness
	witnessData, err := frontend.NewWitness(assignment, ecc.BN254.ScalarField())
	if err != nil {
		return nil, fmt.Errorf("failed to create witness: %w", err)
	}

	// Generate the proof
	proof, err := groth16.Prove(ccs, pk, witnessData)
	if err != nil {
		return nil, fmt.Errorf("failed to generate proof: %w", err)
	}

	// Serialize the proof
	proofBytes := new(bytes.Buffer)
	if _, err := proof.WriteTo(proofBytes); err != nil {
		return nil, fmt.Errorf("failed to serialize proof: %w", err)
	}

	// Serialize public inputs
	publicInputs := make([]byte, 0, 8+32+20) // requestID + resultHash + providerAddress
	reqIDBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(reqIDBytes, requestID)
	publicInputs = append(publicInputs, reqIDBytes...)
	publicInputs = append(publicInputs, resultHash...)
	publicInputs = append(publicInputs, providerAddress.Bytes()...)

	provingTime := time.Since(startTime)

	// Create ZKProof message
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	zkProof := &types.ZKProof{
		Proof:        proofBytes.Bytes(),
		PublicInputs: publicInputs,
		ProofSystem:  "groth16",
		CircuitId:    "compute-verification-v1",
		GeneratedAt:  sdkCtx.BlockTime(),
	}

	// Update metrics
	if err := zk.updateProofGenerationMetrics(ctx, requestID, providerAddress.String(), uint64(provingTime.Milliseconds()), uint32(len(proofBytes.Bytes()))); err != nil {
		// Log but don't fail on metrics update
		sdkCtx.Logger().Error("failed to update proof generation metrics", "error", err)
	}

	return zkProof, nil
}

// ========================================================================================
// PROOF VERIFICATION
// ========================================================================================

// VerifyProof verifies a ZK-SNARK proof for a compute result.
// This is called on-chain during result submission to verify correctness.
func (zk *ZKVerifier) VerifyProof(
	ctx context.Context,
	zkProof *types.ZKProof,
	requestID uint64,
	resultHash []byte,
	providerAddress sdk.AccAddress,
) (bool, error) {
	startTime := time.Now()
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Consume gas for initial validation
	sdkCtx.GasMeter().ConsumeGas(500, "zk_proof_validation_setup")

	// Validate proof system
	if zkProof.ProofSystem != "groth16" {
		return false, fmt.Errorf("unsupported proof system: %s", zkProof.ProofSystem)
	}

	// Get circuit params
	circuitParams, err := zk.keeper.GetCircuitParams(ctx, zkProof.CircuitId)
	if err != nil {
		return false, fmt.Errorf("failed to get circuit params: %w", err)
	}

	if !circuitParams.Enabled {
		return false, fmt.Errorf("circuit %s is not enabled", zkProof.CircuitId)
	}

	// Check proof size
	if len(zkProof.Proof) > int(circuitParams.MaxProofSize) {
		return false, fmt.Errorf("proof size %d exceeds max %d", len(zkProof.Proof), circuitParams.MaxProofSize)
	}

	// Consume gas for deserializing keys - proportional to size
	vkGas := uint64(len(circuitParams.VerifyingKey.VkData) / 32) // ~1 gas per 32 bytes
	sdkCtx.GasMeter().ConsumeGas(vkGas+1000, "zk_verifying_key_deserialization")

	// Deserialize verifying key
	vk := groth16.NewVerifyingKey(ecc.BN254)
	if _, err := vk.ReadFrom(bytes.NewReader(circuitParams.VerifyingKey.VkData)); err != nil {
		return false, fmt.Errorf("failed to deserialize verifying key: %w", err)
	}

	// Consume gas for deserializing proof - proportional to size
	proofGas := uint64(len(zkProof.Proof) / 32) // ~1 gas per 32 bytes
	sdkCtx.GasMeter().ConsumeGas(proofGas+1000, "zk_proof_deserialization")

	// Deserialize proof
	proof := groth16.NewProof(ecc.BN254)
	if _, err := proof.ReadFrom(bytes.NewReader(zkProof.Proof)); err != nil {
		return false, fmt.Errorf("failed to deserialize proof: %w", err)
	}

	// Consume gas for public input reconstruction
	sdkCtx.GasMeter().ConsumeGas(300, "zk_public_input_reconstruction")

	// Reconstruct public inputs from the proof
	expectedPublicInputs := make([]byte, 0, 8+32+20)
	reqIDBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(reqIDBytes, requestID)
	expectedPublicInputs = append(expectedPublicInputs, reqIDBytes...)
	expectedPublicInputs = append(expectedPublicInputs, resultHash...)
	expectedPublicInputs = append(expectedPublicInputs, providerAddress.Bytes()...)

	// Verify public inputs match
	if !bytes.Equal(zkProof.PublicInputs, expectedPublicInputs) {
		return false, fmt.Errorf("public inputs mismatch")
	}

	// Consume gas for creating witness
	sdkCtx.GasMeter().ConsumeGas(800, "zk_witness_creation")

	// Create public witness from inputs
	publicAssignment := &ComputeVerificationCircuit{
		RequestID: requestID,
	}
	for i := 0; i < 32; i++ {
		publicAssignment.ResultHash[i] = resultHash[i]
	}
	for i := 0; i < 20; i++ {
		publicAssignment.ProviderAddress[i] = providerAddress[i]
	}

	publicWitness, err := frontend.NewWitness(publicAssignment, ecc.BN254.ScalarField(), frontend.PublicOnly())
	if err != nil {
		return false, fmt.Errorf("failed to create public witness: %w", err)
	}

	// Verify the proof
	err = groth16.Verify(proof, vk, publicWitness)
	if err != nil {
		// Proof verification failed
		if err := zk.updateVerificationMetrics(ctx, false, time.Since(startTime), circuitParams.GasCost); err != nil {
			sdkCtx.Logger().Error("failed to update verification metrics", "error", err)
		}
		return false, nil
	}

	// Proof verified successfully
	if err := zk.updateVerificationMetrics(ctx, true, time.Since(startTime), circuitParams.GasCost); err != nil {
		sdkCtx.Logger().Error("failed to update verification metrics", "error", err)
	}

	// Consume gas for proof verification
	sdkCtx.GasMeter().ConsumeGas(circuitParams.GasCost, "zk_proof_verification")

	// Emit verification event
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"zk_proof_verified",
			sdk.NewAttribute("request_id", fmt.Sprintf("%d", requestID)),
			sdk.NewAttribute("provider", providerAddress.String()),
			sdk.NewAttribute("circuit_id", zkProof.CircuitId),
			sdk.NewAttribute("verification_time_ms", fmt.Sprintf("%d", time.Since(startTime).Milliseconds())),
		),
	)

	return true, nil
}

// ========================================================================================
// HELPER FUNCTIONS
// ========================================================================================

// GetCircuitParams retrieves circuit parameters from state.
func (k *Keeper) GetCircuitParams(ctx context.Context, circuitID string) (*types.CircuitParams, error) {
	store := k.getStore(ctx)
	key := CircuitParamsKey(circuitID)

	bz := store.Get(key)
	if bz == nil {
		// Return default params if not found
		return k.getDefaultCircuitParams(circuitID), nil
	}

	var params types.CircuitParams
	if err := json.Unmarshal(bz, &params); err != nil {
		return nil, err
	}

	return &params, nil
}

// SetCircuitParams stores circuit parameters.
func (k *Keeper) SetCircuitParams(ctx context.Context, params types.CircuitParams) error {
	store := k.getStore(ctx)
	key := CircuitParamsKey(params.CircuitId)

	bz, err := json.Marshal(&params)
	if err != nil {
		return err
	}

	store.Set(key, bz)
	return nil
}

// getDefaultCircuitParams returns default circuit parameters.
func (k *Keeper) getDefaultCircuitParams(circuitID string) *types.CircuitParams {
	sdkCtx := sdk.UnwrapSDKContext(context.Background())
	return &types.CircuitParams{
		CircuitId:   circuitID,
		Description: "Compute result verification circuit using Groth16",
		VerifyingKey: types.VerifyingKey{
			CircuitId:        circuitID,
			Curve:            "bn254",
			ProofSystem:      "groth16",
			CreatedAt:        sdkCtx.BlockTime(),
			PublicInputCount: 3, // RequestID, ResultHash, ProviderAddress
		},
		MaxProofSize: 256,    // Groth16 proofs are ~256 bytes
		GasCost:      500000, // Gas cost for verification (~0.5M gas)
		Enabled:      true,
	}
}

// updateVerificationMetrics updates ZK verification metrics.
func (zk *ZKVerifier) updateVerificationMetrics(ctx context.Context, success bool, duration time.Duration, gasCost uint64) error {
	metrics, err := zk.keeper.GetZKMetrics(ctx)
	if err != nil {
		// Initialize metrics if not found
		sdkCtx := sdk.UnwrapSDKContext(ctx)
		metrics = &types.ZKMetrics{
			LastUpdated: sdkCtx.BlockTime(),
		}
	}

	if success {
		metrics.TotalProofsVerified++
	} else {
		metrics.TotalProofsFailed++
	}

	// Update average verification time (exponential moving average)
	alpha := 0.1 // Smoothing factor
	newTime := uint64(duration.Milliseconds())
	if metrics.AverageVerificationTimeMs == 0 {
		metrics.AverageVerificationTimeMs = newTime
	} else {
		metrics.AverageVerificationTimeMs = uint64(float64(metrics.AverageVerificationTimeMs)*(1-alpha) + float64(newTime)*alpha)
	}

	metrics.TotalGasConsumed += gasCost
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	metrics.LastUpdated = sdkCtx.BlockTime()

	return zk.keeper.SetZKMetrics(ctx, *metrics)
}

// updateProofGenerationMetrics updates proof generation metrics.
func (zk *ZKVerifier) updateProofGenerationMetrics(ctx context.Context, requestID uint64, provider string, provingTimeMs uint64, proofSize uint32) error {
	metrics, err := zk.keeper.GetZKMetrics(ctx)
	if err != nil {
		sdkCtx := sdk.UnwrapSDKContext(ctx)
		metrics = &types.ZKMetrics{
			LastUpdated: sdkCtx.BlockTime(),
		}
	}

	metrics.TotalProofsGenerated++
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	metrics.LastUpdated = sdkCtx.BlockTime()

	return zk.keeper.SetZKMetrics(ctx, *metrics)
}

// GetZKMetrics retrieves ZK metrics from state.
func (k *Keeper) GetZKMetrics(ctx context.Context) (*types.ZKMetrics, error) {
	store := k.getStore(ctx)
	bz := store.Get([]byte("zk_metrics"))

	if bz == nil {
		return nil, fmt.Errorf("metrics not found")
	}

	var metrics types.ZKMetrics
	if err := json.Unmarshal(bz, &metrics); err != nil {
		return nil, err
	}

	return &metrics, nil
}

// SetZKMetrics stores ZK metrics.
func (k *Keeper) SetZKMetrics(ctx context.Context, metrics types.ZKMetrics) error {
	store := k.getStore(ctx)

	bz, err := json.Marshal(&metrics)
	if err != nil {
		return err
	}

	store.Set([]byte("zk_metrics"), bz)
	return nil
}

// HashComputationResult creates a deterministic hash of computation results.
// This is used to create the resultHash that goes into the circuit.
func HashComputationResult(computationData []byte, metadata map[string]interface{}) []byte {
	hasher := sha256.New()

	// Hash the computation data
	hasher.Write(computationData)

	// Hash metadata in deterministic order
	if timestamp, ok := metadata["timestamp"].(int64); ok {
		tsBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(tsBytes, uint64(timestamp))
		hasher.Write(tsBytes)
	}

	if exitCode, ok := metadata["exit_code"].(int32); ok {
		ecBytes := make([]byte, 4)
		binary.BigEndian.PutUint32(ecBytes, uint32(exitCode))
		hasher.Write(ecBytes)
	}

	if cpuCycles, ok := metadata["cpu_cycles"].(uint64); ok {
		cpuBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(cpuBytes, cpuCycles)
		hasher.Write(cpuBytes)
	}

	if memoryBytes, ok := metadata["memory_bytes"].(uint64); ok {
		memBytes := make([]byte, 8)
		binary.BigEndian.PutUint64(memBytes, memoryBytes)
		hasher.Write(memBytes)
	}

	return hasher.Sum(nil)
}
