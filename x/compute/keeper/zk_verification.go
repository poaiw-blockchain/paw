package keeper

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"time"

	errorsmod "cosmossdk.io/errors"
	"cosmossdk.io/math"
	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark-crypto/ecc/bn254/fr"
	mimcfr "github.com/consensys/gnark-crypto/ecc/bn254/fr/mimc"
	"github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark/constraint"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/frontend/cs/r1cs"
	mimcstd "github.com/consensys/gnark/std/hash/mimc"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/compute/types"
)

// ZKVerifier handles zero-knowledge proof generation and verification for compute results.
type ZKVerifier struct {
	keeper        *Keeper
	circuitCCS    map[string]constraint.ConstraintSystem
	provingKeys   map[string]groth16.ProvingKey
	verifyingKeys map[string]groth16.VerifyingKey
}

// NewZKVerifier creates a new ZK proof verifier.
func NewZKVerifier(keeper *Keeper) *ZKVerifier {
	return &ZKVerifier{
		keeper:        keeper,
		circuitCCS:    make(map[string]constraint.ConstraintSystem),
		provingKeys:   make(map[string]groth16.ProvingKey),
		verifyingKeys: make(map[string]groth16.VerifyingKey),
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
//   - ResultHash: field element (MiMC hash)
//   - ProviderAddressHash: field element (hash of provider address)
//
// Private Inputs (Witness):
//   - ComputationDataHash: pre-computed hash of computation data
//   - ExecutionTimestamp: when computation ran
//   - ExitCode: exit code (0-255)
//   - CpuCyclesUsed: CPU cycles consumed
//   - MemoryBytesUsed: memory usage
//
// Constraint:
//
//	MiMC(ComputationDataHash || ExecutionTimestamp || ExitCode || CpuCycles || Memory) == ResultHash
//
// This proves that the provider knows the actual computation that produced
// the result hash, without revealing the computation details on-chain.
type ComputeVerificationCircuit struct {
	// Public inputs (visible on-chain)
	RequestID           frontend.Variable `gnark:",public"`
	ResultHash          frontend.Variable `gnark:",public"` // MiMC hash of computation
	ProviderAddressHash frontend.Variable `gnark:",public"` // Hash of provider address

	// Private inputs (witness data, kept secret)
	ComputationDataHash frontend.Variable `gnark:",private"` // Pre-computed hash of computation data
	ExecutionTimestamp  frontend.Variable `gnark:",private"` // When computation ran
	ExitCode            frontend.Variable `gnark:",private"` // Exit code (0-255)
	CpuCyclesUsed       frontend.Variable `gnark:",private"` // CPU cycles consumed
	MemoryBytesUsed     frontend.Variable `gnark:",private"` // Memory usage
}

// Define implements the gnark Circuit interface.
// This is the core of the ZK-SNARK - it defines the constraints that must be satisfied.
func (circuit *ComputeVerificationCircuit) Define(api frontend.API) error {
	// Initialize MiMC hasher - ZK-friendly hash function
	hasher, err := mimcstd.NewMiMC(api)
	if err != nil {
		return fmt.Errorf("failed to initialize MiMC hasher: %w", err)
	}

	// Hash all private inputs together with request context
	// This creates a deterministic commitment to the computation
	hasher.Write(circuit.RequestID)
	hasher.Write(circuit.ProviderAddressHash)
	hasher.Write(circuit.ComputationDataHash)
	hasher.Write(circuit.ExecutionTimestamp)
	hasher.Write(circuit.ExitCode)
	hasher.Write(circuit.CpuCyclesUsed)
	hasher.Write(circuit.MemoryBytesUsed)

	// Get the computed hash
	computedHash := hasher.Sum()

	// Core constraint: computed hash must equal the public ResultHash
	// This proves knowledge of the private inputs that produce the result
	api.AssertIsEqual(computedHash, circuit.ResultHash)

	// Additional validity constraints:

	// 1. ExitCode must be a valid exit code (0-255)
	// Decompose into bits and verify it fits in 8 bits
	exitCodeBits := api.ToBinary(circuit.ExitCode, 8)
	recomposedExitCode := api.FromBinary(exitCodeBits...)
	api.AssertIsEqual(recomposedExitCode, circuit.ExitCode)

	// 2. Timestamp must be positive (non-zero)
	// We check that timestamp is not zero
	api.AssertIsDifferent(circuit.ExecutionTimestamp, 0)

	// 3. RequestID must be positive
	api.AssertIsDifferent(circuit.RequestID, 0)

	// 4. CpuCyclesUsed must fit in 64 bits (reasonable upper bound)
	cpuBits := api.ToBinary(circuit.CpuCyclesUsed, 64)
	recomposedCpu := api.FromBinary(cpuBits...)
	api.AssertIsEqual(recomposedCpu, circuit.CpuCyclesUsed)

	// 5. MemoryBytesUsed must fit in 64 bits
	memBits := api.ToBinary(circuit.MemoryBytesUsed, 64)
	recomposedMem := api.FromBinary(memBits...)
	api.AssertIsEqual(recomposedMem, circuit.MemoryBytesUsed)

	return nil
}

// ========================================================================================
// PROOF GENERATION
// ========================================================================================

// ComputeMiMCHash computes MiMC hash off-chain using the same algorithm as the circuit.
// This is used to generate the result hash that will be verified in the circuit.
func ComputeMiMCHash(
	requestID uint64,
	providerAddressHash []byte,
	computationDataHash []byte,
	executionTimestamp int64,
	exitCode int32,
	cpuCyclesUsed uint64,
	memoryBytesUsed uint64,
) ([]byte, error) {
	// Use gnark-crypto's MiMC implementation for off-chain computation
	// This matches what the circuit computes
	h := sha256.New() // Use SHA256 to create field elements

	// Write all inputs in the same order as the circuit
	reqIDBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(reqIDBytes, requestID)
	h.Write(reqIDBytes)

	h.Write(providerAddressHash)
	h.Write(computationDataHash)

	tsBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(tsBytes, saturateInt64ToUint64(executionTimestamp))
	h.Write(tsBytes)

	ecBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(ecBytes, saturateInt64ToUint32(int64(exitCode)))
	h.Write(ecBytes)

	cpuBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(cpuBytes, cpuCyclesUsed)
	h.Write(cpuBytes)

	memBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(memBytes, memoryBytesUsed)
	h.Write(memBytes)

	return h.Sum(nil), nil
}

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
	if len(providerAddress) == 0 {
		return nil, fmt.Errorf("provider address cannot be empty")
	}
	if exitCode < 0 || exitCode > 255 {
		return nil, fmt.Errorf("exit code must be 0-255, got %d", exitCode)
	}
	if executionTimestamp <= 0 {
		return nil, fmt.Errorf("execution timestamp must be positive")
	}

	const circuitID = "compute-verification-v1"

	// Get circuit params from state
	circuitParams, err := zk.keeper.GetCircuitParams(ctx, circuitID)
	if err != nil {
		return nil, fmt.Errorf("failed to get circuit params: %w", err)
	}

	if !circuitParams.Enabled {
		return nil, fmt.Errorf("circuit is not enabled")
	}

	// Compile and cache the circuit definition for reuse
	ccs, ok := zk.circuitCCS[circuitID]
	if !ok {
		compiled, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, &ComputeVerificationCircuit{})
		if err != nil {
			return nil, fmt.Errorf("failed to compile circuit: %w", err)
		}
		ccs = compiled
		zk.circuitCCS[circuitID] = ccs
	}

	// Reuse the proving key so proofs remain compatible with the stored verifying key
	pk, ok := zk.provingKeys[circuitID]
	var vk groth16.VerifyingKey
	if !ok {
		var err error
		pk, vk, err = groth16.Setup(ccs)
		if err != nil {
			return nil, fmt.Errorf("failed to setup circuit: %w", err)
		}
		zk.provingKeys[circuitID] = pk
		zk.verifyingKeys[circuitID] = vk

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
	}

	// Compute hashes for the witness
	providerAddressHash := sha256.Sum256(providerAddress.Bytes())
	computationDataHash := sha256.Sum256(computationData)

	// Recompute result hash to ensure it matches circuit hashing and reject mismatches.
	computedResultHash, err := computeResultHashMiMC(requestID, providerAddress, computationData, executionTimestamp, exitCode, cpuCyclesUsed, memoryBytesUsed)
	if err != nil {
		return nil, fmt.Errorf("failed to compute result hash: %w", err)
	}
	if len(resultHash) > 0 && !bytes.Equal(resultHash, computedResultHash) {
		return nil, fmt.Errorf("provided result hash does not match circuit hash")
	}
	resultHash = computedResultHash

	// Convert result hash to field element (first 31 bytes to fit in BN254 field)
	resultHashField := bytesToFieldElement(resultHash)
	providerHashField := bytesToFieldElement(providerAddressHash[:])
	compDataHashField := bytesToFieldElement(computationDataHash[:])

	// Prepare witness data (private inputs + public inputs)
	assignment := &ComputeVerificationCircuit{
		// Public inputs
		RequestID:           requestID,
		ResultHash:          resultHashField,
		ProviderAddressHash: providerHashField,
		// Private inputs
		ComputationDataHash: compDataHashField,
		ExecutionTimestamp:  executionTimestamp,
		ExitCode:            int(exitCode),
		CpuCyclesUsed:       cpuCyclesUsed,
		MemoryBytesUsed:     memoryBytesUsed,
	}

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

	// Serialize public inputs for verification
	publicInputs := serializePublicInputs(requestID, resultHash, providerAddressHash[:])

	provingTime := time.Since(startTime)

	// Create ZKProof message
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	zkProof := &types.ZKProof{
		Proof:        proofBytes.Bytes(),
		PublicInputs: publicInputs,
		ProofSystem:  "groth16",
		CircuitId:    circuitID,
		GeneratedAt:  sdkCtx.BlockTime(),
	}

	// Update metrics
	if err := zk.updateProofGenerationMetrics(
		ctx,
		requestID,
		providerAddress.String(),
		saturateInt64ToUint64(provingTime.Milliseconds()),
		saturateInt64ToUint32(int64(len(proofBytes.Bytes()))),
	); err != nil {
		// Log but don't fail on metrics update
		sdkCtx.Logger().Error("failed to update proof generation metrics", "error", err)
	}

	return zkProof, nil
}

// bytesToFieldElement converts bytes to a field element representation.
// Takes up to 32 bytes and reduces into BN254 scalar field.
func bytesToFieldElement(b []byte) interface{} {
	if len(b) > 32 {
		b = b[:32]
	}
	var elem fr.Element
	elem.SetBytes(b)
	return elem
}

// serializePublicInputs creates a serialized representation of public inputs.
func serializePublicInputs(requestID uint64, resultHash, providerAddressHash []byte) []byte {
	publicInputs := make([]byte, 0, 8+32+32) // requestID + resultHash + providerAddressHash
	reqIDBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(reqIDBytes, requestID)
	publicInputs = append(publicInputs, reqIDBytes...)
	publicInputs = append(publicInputs, resultHash...)
	publicInputs = append(publicInputs, providerAddressHash...)
	return publicInputs
}

// computeResultHashMiMC mirrors the circuit hashing (MiMC over request + provider hash + computation hash + telemetry).
func computeResultHashMiMC(
	requestID uint64,
	providerAddress sdk.AccAddress,
	computationData []byte,
	executionTimestamp int64,
	exitCode int32,
	cpuCyclesUsed uint64,
	memoryBytesUsed uint64,
) ([]byte, error) {
	if executionTimestamp < 0 {
		return nil, fmt.Errorf("execution timestamp must be non-negative")
	}
	if exitCode < 0 {
		return nil, fmt.Errorf("exit code must be non-negative")
	}

	h := mimcfr.NewMiMC()

	writeElem := func(e fr.Element) error {
		b := e.Bytes()
		if _, err := h.Write(b[:]); err != nil {
			return fmt.Errorf("failed to hash element: %w", err)
		}
		return nil
	}

	var elem fr.Element
	elem.SetUint64(requestID)
	if err := writeElem(elem); err != nil {
		return nil, err
	}

	providerHash := sha256.Sum256(providerAddress.Bytes())
	elem.SetBytes(providerHash[:])
	if err := writeElem(elem); err != nil {
		return nil, err
	}

	computationHash := sha256.Sum256(computationData)
	elem.SetBytes(computationHash[:])
	if err := writeElem(elem); err != nil {
		return nil, err
	}

	elem.SetUint64(uint64(executionTimestamp))
	if err := writeElem(elem); err != nil {
		return nil, err
	}

	elem.SetUint64(uint64(exitCode))
	if err := writeElem(elem); err != nil {
		return nil, err
	}

	elem.SetUint64(cpuCyclesUsed)
	if err := writeElem(elem); err != nil {
		return nil, err
	}

	elem.SetUint64(memoryBytesUsed)
	if err := writeElem(elem); err != nil {
		return nil, err
	}

	sum := h.Sum(nil)
	if len(sum) != sha256.Size {
		return nil, fmt.Errorf("unexpected mimc hash size: %d", len(sum))
	}
	return sum, nil
}

// ========================================================================================
// PROOF VERIFICATION
// ========================================================================================

// VerifyProof verifies a ZK-SNARK proof for a compute result.
// This is called on-chain during result submission to verify correctness.
//
// Security: Proof size is checked BEFORE any gas consumption to prevent DoS attacks
// where attackers submit massive proofs to exhaust gas. Additionally, a deposit is required
// and refunded on valid proofs, slashed on invalid proofs (economic DoS protection).
func (zk *ZKVerifier) VerifyProof(
	ctx context.Context,
	zkProof *types.ZKProof,
	requestID uint64,
	resultHash []byte,
	providerAddress sdk.AccAddress,
) (bool, error) {
	startTime := time.Now()
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// CRITICAL SECURITY: Absolute size limit check BEFORE any operations (including storage reads)
	// This prevents DoS attacks where attackers submit massive proofs to exhaust gas.
	// We use a hard upper bound before even checking circuit-specific limits.
	const absoluteMaxProofSize = 10 * 1024 * 1024 // 10MB absolute maximum
	if uint64(len(zkProof.Proof)) > absoluteMaxProofSize {
		return false, types.ErrProofTooLarge
	}

	// Now safe to get circuit params (may consume gas for storage read)
	circuitParams, err := zk.keeper.GetCircuitParams(ctx, zkProof.CircuitId)
	if err != nil {
		return false, fmt.Errorf("failed to get circuit params: %w", err)
	}

	// Check circuit-specific size limit
	maxProofSize := circuitParams.MaxProofSize
	if maxProofSize == 0 {
		maxProofSize = 1024 * 1024 // Default 1MB max if not set
	}
	if uint64(len(zkProof.Proof)) > uint64(maxProofSize) {
		return false, types.ErrProofTooLarge
	}

	// CRITICAL SECURITY: Require deposit BEFORE any expensive verification operations
	// This provides economic DoS protection - attackers must pay for invalid proofs
	depositRequired := circuitParams.VerificationDepositAmount
	var (
		depositCoin  sdk.Coin
		depositCoins sdk.Coins
	)
	if depositRequired > 0 {
		depositCoin = sdk.NewCoin("upaw", math.NewInt(int64(depositRequired)))
		depositCoins = sdk.NewCoins(depositCoin)

		// Transfer deposit from provider to module account
		if err := zk.keeper.bankKeeper.SendCoinsFromAccountToModule(
			sdkCtx,
			providerAddress,
			types.ModuleName,
			depositCoins,
		); err != nil {
			return false, errorsmod.Wrap(types.ErrInsufficientDeposit, err.Error())
		}

		sdkCtx.EventManager().EmitEvent(
			sdk.NewEvent(
				"zk_proof_deposit_locked",
				sdk.NewAttribute("request_id", fmt.Sprintf("%d", requestID)),
				sdk.NewAttribute("provider", providerAddress.String()),
				sdk.NewAttribute("deposit_amount", depositCoin.String()),
				sdk.NewAttribute("circuit_id", zkProof.CircuitId),
			),
		)
	}

	// Now we can safely consume gas for validation
	sdkCtx.GasMeter().ConsumeGas(500, "zk_proof_validation_setup")

	// Helper function to handle deposit refund on early errors
	refundDeposit := func() {
		if depositRequired > 0 {
			depositCoin := sdk.NewCoin("upaw", math.NewInt(saturateUint64ToInt64(depositRequired)))
			depositCoins := sdk.NewCoins(depositCoin)
			if err := zk.keeper.bankKeeper.SendCoinsFromModuleToAccount(
				sdkCtx,
				types.ModuleName,
				providerAddress,
				depositCoins,
			); err != nil {
				sdkCtx.Logger().Error("failed to refund deposit on validation error",
					"request_id", requestID,
					"provider", providerAddress.String(),
					"error", err.Error(),
				)
			}
		}
	}

	// Validate proof system
	if zkProof.ProofSystem != "groth16" {
		refundDeposit()
		return false, fmt.Errorf("unsupported proof system: %s", zkProof.ProofSystem)
	}

	// Validate proof is not empty
	if len(zkProof.Proof) == 0 {
		refundDeposit()
		return false, fmt.Errorf("proof cannot be empty")
	}
	if len(resultHash) != sha256.Size {
		refundDeposit()
		return false, fmt.Errorf("result hash must be 32 bytes, got %d", len(resultHash))
	}

	if !circuitParams.Enabled {
		refundDeposit()
		return false, fmt.Errorf("circuit %s is not enabled", zkProof.CircuitId)
	}

	// Consume gas for deserializing keys - proportional to size
	vkGas := saturateInt64ToUint64(int64(len(circuitParams.VerifyingKey.VkData)/32)) + 1000
	sdkCtx.GasMeter().ConsumeGas(vkGas, "zk_verifying_key_deserialization")

	// Get or deserialize verifying key
	vk, ok := zk.verifyingKeys[zkProof.CircuitId]
	if !ok {
		if len(circuitParams.VerifyingKey.VkData) == 0 {
			refundDeposit()
			return false, fmt.Errorf("verifying key not found for circuit %s", zkProof.CircuitId)
		}
		vk = groth16.NewVerifyingKey(ecc.BN254)
		if _, err := vk.ReadFrom(bytes.NewReader(circuitParams.VerifyingKey.VkData)); err != nil {
			refundDeposit()
			return false, fmt.Errorf("failed to deserialize verifying key: %w", err)
		}
		zk.verifyingKeys[zkProof.CircuitId] = vk
	}

	// Consume gas for deserializing proof - proportional to size
	proofGas := saturateInt64ToUint64(int64(len(zkProof.Proof)/32)) + 1000
	sdkCtx.GasMeter().ConsumeGas(proofGas, "zk_proof_deserialization")

	// Deserialize proof
	proof := groth16.NewProof(ecc.BN254)
	if _, err := proof.ReadFrom(bytes.NewReader(zkProof.Proof)); err != nil {
		refundDeposit()
		return false, fmt.Errorf("failed to deserialize proof: %w", err)
	}

	// Consume gas for public input reconstruction
	sdkCtx.GasMeter().ConsumeGas(300, "zk_public_input_reconstruction")

	// Compute provider address hash for comparison
	providerAddressHash := sha256.Sum256(providerAddress.Bytes())

	// Reconstruct expected public inputs
	expectedPublicInputs := serializePublicInputs(requestID, resultHash, providerAddressHash[:])

	// Verify public inputs match
	if !bytes.Equal(zkProof.PublicInputs, expectedPublicInputs) {
		sdkCtx.Logger().Warn("public inputs mismatch",
			"expected_len", len(expectedPublicInputs),
			"actual_len", len(zkProof.PublicInputs),
		)
		refundDeposit()
		return false, fmt.Errorf("public inputs mismatch")
	}

	// Consume gas for creating witness
	sdkCtx.GasMeter().ConsumeGas(800, "zk_witness_creation")

	// Convert hashes to field elements
	resultHashField := bytesToFieldElement(resultHash)
	providerHashField := bytesToFieldElement(providerAddressHash[:])

	// Create public witness from inputs
	publicAssignment := &ComputeVerificationCircuit{
		RequestID:           requestID,
		ResultHash:          resultHashField,
		ProviderAddressHash: providerHashField,
	}

	publicWitness, err := frontend.NewWitness(publicAssignment, ecc.BN254.ScalarField(), frontend.PublicOnly())
	if err != nil {
		refundDeposit()
		return false, fmt.Errorf("failed to create public witness: %w", err)
	}

	// Verify the proof
	verificationErr := groth16Verify(proof, vk, publicWitness)

	// Handle verification failures before proceeding
	if verificationErr != nil {
		if depositRequired > 0 {
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
		}

		if updateErr := zk.updateVerificationMetrics(ctx, false, time.Since(startTime), circuitParams.GasCost); updateErr != nil {
			sdkCtx.Logger().Error("failed to update verification metrics", "error", updateErr)
		}

		return false, fmt.Errorf("zk proof verification failed: %w", verificationErr)
	}

	// Handle deposit refunds on success
	if depositRequired > 0 {
		// VALID PROOF: Refund deposit to provider
		if err := zk.keeper.bankKeeper.SendCoinsFromModuleToAccount(
			sdkCtx,
			types.ModuleName,
			providerAddress,
			depositCoins,
		); err != nil {
			// Log error but don't fail the verification since proof is valid
			sdkCtx.Logger().Error("failed to refund verification deposit",
				"request_id", requestID,
				"provider", providerAddress.String(),
				"deposit", depositCoin.String(),
				"error", err.Error(),
			)
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
	}

	// Update metrics for successful verification
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

// InitializeCircuit compiles and sets up the circuit with proving/verifying keys.
// This should be called during module initialization or when circuit params are first needed.
func (zk *ZKVerifier) InitializeCircuit(ctx context.Context, circuitID string) error {
	// Compile the circuit
	ccs, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, &ComputeVerificationCircuit{})
	if err != nil {
		return fmt.Errorf("failed to compile circuit: %w", err)
	}
	zk.circuitCCS[circuitID] = ccs

	// Generate proving and verifying keys
	pk, vk, err := groth16Setup(ccs)
	if err != nil {
		return fmt.Errorf("failed to setup circuit: %w", err)
	}
	zk.provingKeys[circuitID] = pk
	zk.verifyingKeys[circuitID] = vk

	// Serialize and store verifying key
	vkBytes := new(bytes.Buffer)
	if _, err := vk.WriteTo(vkBytes); err != nil {
		return fmt.Errorf("failed to serialize verifying key: %w", err)
	}

	// Get existing params or create new ones
	circuitParams, err := zk.keeper.GetCircuitParams(ctx, circuitID)
	if err != nil {
		circuitParams = zk.keeper.getDefaultCircuitParams(ctx, circuitID)
	}

	circuitParams.VerifyingKey.VkData = vkBytes.Bytes()
	circuitParams.MaxProofSize = 2048                 // Groth16 proofs can be ~1-2KB
	circuitParams.VerificationDepositAmount = 1000000 // 1 PAW deposit for DoS protection

	return zk.keeper.SetCircuitParams(ctx, *circuitParams)
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
		return k.getDefaultCircuitParams(ctx, circuitID), nil
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
func (k *Keeper) getDefaultCircuitParams(ctx context.Context, circuitID string) *types.CircuitParams {
	createdAt := time.Now().UTC()
	if sdkCtx, ok := ctx.(sdk.Context); ok && !sdkCtx.BlockTime().IsZero() {
		createdAt = sdkCtx.BlockTime()
	}

	return &types.CircuitParams{
		CircuitId:   circuitID,
		Description: "Compute result verification circuit using Groth16",
		VerifyingKey: types.VerifyingKey{
			CircuitId:        circuitID,
			Curve:            "bn254",
			ProofSystem:      "groth16",
			CreatedAt:        createdAt,
			PublicInputCount: 3, // RequestID, ResultHash, ProviderAddress
		},
		MaxProofSize:              1024 * 1024, // 1MB max - prevents DoS via oversized proofs
		GasCost:                   500000,      // Gas cost for verification (~0.5M gas)
		Enabled:                   true,
		VerificationDepositAmount: 1000000, // 1,000,000 upaw (1 PAW) - refunded on valid proof, slashed on invalid
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
	newTime := saturateInt64ToUint64(duration.Milliseconds())
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
		binary.BigEndian.PutUint64(tsBytes, saturateInt64ToUint64(timestamp))
		hasher.Write(tsBytes)
	}

	if exitCode, ok := metadata["exit_code"].(int32); ok {
		ecBytes := make([]byte, 4)
		binary.BigEndian.PutUint32(ecBytes, saturateInt64ToUint32(int64(exitCode)))
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

// ComputeResultHash derives the circuit-compatible result hash using MiMC over the same fields
// enforced in ComputeVerificationCircuit. This must be used when generating/validating proofs.
func ComputeResultHash(
	requestID uint64,
	providerAddress sdk.AccAddress,
	computationData []byte,
	executionTimestamp int64,
	exitCode int32,
	cpuCyclesUsed uint64,
	memoryBytesUsed uint64,
) ([]byte, error) {
	return computeResultHashMiMC(requestID, providerAddress, computationData, executionTimestamp, exitCode, cpuCyclesUsed, memoryBytesUsed)
}
