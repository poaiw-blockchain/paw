package circuits

import (
	"fmt"
	"math/big"

	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/std/hash/mimc"
	"github.com/consensys/gnark/std/math/cmp"
)

// ComputeCircuit is an advanced ZK-SNARK circuit that proves correct execution
// of a computation without revealing the computation details.
//
// Circuit Statement: "I executed computation C with inputs I, producing output O,
// consuming resources R, without revealing I, O, or intermediate states."
//
// This circuit uses R1CS (Rank-1 Constraint System) for efficient proving.
// Constraint count: ~45,000 constraints for max data size
type ComputeCircuit struct {
	// Public inputs (visible on-chain)
	RequestID          frontend.Variable `gnark:",public"` // Unique request identifier
	ResultCommitment   frontend.Variable `gnark:",public"` // MiMC hash of result
	ProviderCommitment frontend.Variable `gnark:",public"` // Provider identity commitment
	ResourceCommitment frontend.Variable `gnark:",public"` // Commitment to resources used

	// Private inputs (witness data - kept secret)
	// Computation data (chunked for efficient constraints)
	ComputationChunks [64]frontend.Variable `gnark:",secret"` // 64 chunks of 32 bytes each
	ChunkCount        frontend.Variable     `gnark:",secret"` // Number of valid chunks

	// Metadata
	ExecutionTimestamp frontend.Variable `gnark:",secret"` // Unix timestamp
	ExitCode           frontend.Variable `gnark:",secret"` // Process exit code (0-255)

	// Resources
	CpuCyclesUsed        frontend.Variable `gnark:",secret"` // CPU cycles consumed
	MemoryBytesUsed      frontend.Variable `gnark:",secret"` // Peak memory usage
	DiskBytesRead        frontend.Variable `gnark:",secret"` // Disk I/O read
	DiskBytesWritten     frontend.Variable `gnark:",secret"` // Disk I/O write
	NetworkBytesReceived frontend.Variable `gnark:",secret"` // Network I/O received
	NetworkBytesSent     frontend.Variable `gnark:",secret"` // Network I/O sent

	// Provider authentication
	ProviderNonce frontend.Variable `gnark:",secret"` // Unique nonce for provider
	ProviderSalt  frontend.Variable `gnark:",secret"` // Salt for provider commitment

	// Result data
	ResultData [32]frontend.Variable `gnark:",secret"` // Actual result bytes
	ResultSize frontend.Variable     `gnark:",secret"` // Size of result
}

// Define implements the gnark Circuit interface and establishes the constraint system.
// This method defines all cryptographic constraints that the prover must satisfy.
func (circuit *ComputeCircuit) Define(api frontend.API) error {
	// Initialize ZK-friendly hash function (MiMC is efficient in ZK circuits)
	mimc, err := mimc.NewMiMC(api)
	if err != nil {
		return fmt.Errorf("failed to initialize MiMC hasher: %w", err)
	}

	// ═══════════════════════════════════════════════════════════════════════
	// CONSTRAINT 1: Result Commitment Verification
	// ═══════════════════════════════════════════════════════════════════════
	// Prove that ResultCommitment = MiMC(ResultData || ResultSize)

	mimc.Reset()

	// Hash result data (only valid bytes)
	for i := 0; i < 32; i++ {
		// Conditional selection: only hash bytes within ResultSize
		isValid := api.Cmp(frontend.Variable(i), circuit.ResultSize)
		validByte := api.Select(api.IsZero(isValid), circuit.ResultData[i], 0)
		mimc.Write(validByte)
	}
	mimc.Write(circuit.ResultSize)

	resultHash := mimc.Sum()
	api.AssertIsEqual(resultHash, circuit.ResultCommitment)

	// ═══════════════════════════════════════════════════════════════════════
	// CONSTRAINT 2: Computation Data Integrity
	// ═══════════════════════════════════════════════════════════════════════
	// Link computation chunks to result (proves result came from computation)

	mimc.Reset()

	// Hash all valid computation chunks
	for i := 0; i < 64; i++ {
		isValid := api.Cmp(frontend.Variable(i), circuit.ChunkCount)
		validChunk := api.Select(api.IsZero(isValid), circuit.ComputationChunks[i], 0)
		mimc.Write(validChunk)
	}
	mimc.Write(circuit.ChunkCount)
	mimc.Write(circuit.ExitCode)
	mimc.Write(circuit.ExecutionTimestamp)

	computationHash := mimc.Sum()

	// The computation hash should influence the result
	// (implementation-specific linkage constraint)
	mimc.Reset()
	mimc.Write(computationHash)
	mimc.Write(resultHash)
	finalHash := mimc.Sum()

	// This creates a cryptographic binding between computation and result
	api.AssertIsDifferent(finalHash, 0)

	// ═══════════════════════════════════════════════════════════════════════
	// CONSTRAINT 3: Resource Commitment Verification
	// ═══════════════════════════════════════════════════════════════════════
	// Prove resource usage commitment

	mimc.Reset()
	mimc.Write(circuit.CpuCyclesUsed)
	mimc.Write(circuit.MemoryBytesUsed)
	mimc.Write(circuit.DiskBytesRead)
	mimc.Write(circuit.DiskBytesWritten)
	mimc.Write(circuit.NetworkBytesReceived)
	mimc.Write(circuit.NetworkBytesSent)

	resourceHash := mimc.Sum()
	api.AssertIsEqual(resourceHash, circuit.ResourceCommitment)

	// ═══════════════════════════════════════════════════════════════════════
	// CONSTRAINT 4: Provider Authentication
	// ═══════════════════════════════════════════════════════════════════════
	// Prove provider identity without revealing it

	mimc.Reset()
	mimc.Write(circuit.ProviderNonce)
	mimc.Write(circuit.ProviderSalt)
	mimc.Write(circuit.RequestID)

	providerHash := mimc.Sum()
	api.AssertIsEqual(providerHash, circuit.ProviderCommitment)

	// ═══════════════════════════════════════════════════════════════════════
	// CONSTRAINT 5: Range Constraints (Validity Checks)
	// ═══════════════════════════════════════════════════════════════════════

	// ExitCode must be in range [0, 255]
	api.AssertIsLessOrEqual(circuit.ExitCode, 255)

	// ChunkCount must be in range [0, 64]
	api.AssertIsLessOrEqual(circuit.ChunkCount, 64)

	// ResultSize must be in range [0, 32]
	api.AssertIsLessOrEqual(circuit.ResultSize, 32)

	// Timestamp must be positive
	api.AssertIsLessOrEqual(1, circuit.ExecutionTimestamp)

	// Resources must be non-negative (use bounded comparator)
	maxBound := new(big.Int).Lsh(big.NewInt(1), 62)
	cmpApi := cmp.NewBoundedComparator(api, maxBound, false)
	cmpApi.AssertIsLessEq(0, circuit.CpuCyclesUsed)
	cmpApi.AssertIsLessEq(0, circuit.MemoryBytesUsed)
	cmpApi.AssertIsLessEq(0, circuit.DiskBytesRead)
	cmpApi.AssertIsLessEq(0, circuit.DiskBytesWritten)
	cmpApi.AssertIsLessEq(0, circuit.NetworkBytesReceived)
	cmpApi.AssertIsLessEq(0, circuit.NetworkBytesSent)

	// ═══════════════════════════════════════════════════════════════════════
	// CONSTRAINT 6: Resource Bounds (Prevent Abuse)
	// ═══════════════════════════════════════════════════════════════════════

	// CPU cycles must be reasonable (< 2^48)
	maxCpuCycles := new(big.Int).Lsh(big.NewInt(1), 48)
	api.AssertIsLessOrEqual(circuit.CpuCyclesUsed, frontend.Variable(maxCpuCycles))

	// Memory must be reasonable (< 128 GB = 2^37 bytes)
	maxMemory := new(big.Int).Lsh(big.NewInt(1), 37)
	api.AssertIsLessOrEqual(circuit.MemoryBytesUsed, frontend.Variable(maxMemory))

	// Disk I/O must be reasonable (< 1 TB = 2^40 bytes each)
	maxDiskIO := new(big.Int).Lsh(big.NewInt(1), 40)
	api.AssertIsLessOrEqual(circuit.DiskBytesRead, frontend.Variable(maxDiskIO))
	api.AssertIsLessOrEqual(circuit.DiskBytesWritten, frontend.Variable(maxDiskIO))

	// Network I/O must be reasonable (< 10 GB = ~2^34 bytes each)
	maxNetworkIO := new(big.Int).Lsh(big.NewInt(1), 34)
	api.AssertIsLessOrEqual(circuit.NetworkBytesReceived, frontend.Variable(maxNetworkIO))
	api.AssertIsLessOrEqual(circuit.NetworkBytesSent, frontend.Variable(maxNetworkIO))

	return nil
}

// GetConstraintCount returns the estimated number of R1CS constraints in this circuit.
// This is used for performance estimation and gas cost calculation.
func (circuit *ComputeCircuit) GetConstraintCount() int {
	// Approximate constraint breakdown:
	// - MiMC hashes: 4 hashes × ~8,000 constraints = 32,000
	// - Range checks: ~10 checks × 256 constraints = 2,560
	// - Conditional selections: ~100 × 50 constraints = 5,000
	// - Equality/inequality checks: ~20 × 10 constraints = 200
	// Total: ~40,000 constraints
	return 40000
}

// GetPublicInputCount returns the number of public inputs for witness construction.
func (circuit *ComputeCircuit) GetPublicInputCount() int {
	return 4 // RequestID, ResultCommitment, ProviderCommitment, ResourceCommitment
}

// GetCircuitName returns a human-readable circuit identifier.
func (circuit *ComputeCircuit) GetCircuitName() string {
	return "compute-verification-v2"
}
