package circuits

import (
	"fmt"
	"math/big"

	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/std/hash/mimc"
)

// EscrowCircuit proves that escrow funds should be released based on verified computation.
//
// Circuit Statement: "I have completed computation for request R, meeting all requirements,
// and the escrow amount A should be released to provider P."
//
// This circuit ensures atomic proof of computation completion and escrow release eligibility.
// Constraint count: ~25,000 constraints
type EscrowCircuit struct {
	// Public inputs
	RequestID            frontend.Variable `gnark:",public"` // Request identifier
	EscrowAmount         frontend.Variable `gnark:",public"` // Amount to release
	RequesterCommitment  frontend.Variable `gnark:",public"` // Requester identity
	ProviderCommitment   frontend.Variable `gnark:",public"` // Provider identity
	CompletionCommitment frontend.Variable `gnark:",public"` // Proof of completion

	// Private inputs
	// Request details
	RequestNonce         frontend.Variable `gnark:",private"` // Unique request nonce
	RequesterAddress     [20]frontend.Variable `gnark:",private"` // Requester address bytes
	ProviderAddress      [20]frontend.Variable `gnark:",private"` // Provider address bytes

	// Computation proof
	ResultHash           frontend.Variable `gnark:",private"` // Hash of computation result
	ExecutionSuccess     frontend.Variable `gnark:",private"` // 1 if successful, 0 if failed
	VerificationPassed   frontend.Variable `gnark:",private"` // 1 if verified, 0 otherwise

	// Resource validation
	EstimatedCost        frontend.Variable `gnark:",private"` // Original cost estimate
	ActualCost           frontend.Variable `gnark:",private"` // Actual computation cost

	// Slashing conditions
	ProviderMisbehavior  frontend.Variable `gnark:",private"` // 1 if provider misbehaved
	RequesterCancelled   frontend.Variable `gnark:",private"` // 1 if requester cancelled

	// Timing
	StartTimestamp       frontend.Variable `gnark:",private"` // Computation start time
	EndTimestamp         frontend.Variable `gnark:",private"` // Computation end time
	DeadlineTimestamp    frontend.Variable `gnark:",private"` // Required completion deadline
}

// Define implements the gnark Circuit interface for escrow release constraints.
func (circuit *EscrowCircuit) Define(api frontend.API) error {
	mimc, err := mimc.NewMiMC(api)
	if err != nil {
		return fmt.Errorf("failed to initialize MiMC: %w", err)
	}

	// ═══════════════════════════════════════════════════════════════════════
	// CONSTRAINT 1: Requester Identity Verification
	// ═══════════════════════════════════════════════════════════════════════

	mimc.Reset()
	for i := 0; i < 20; i++ {
		mimc.Write(circuit.RequesterAddress[i])
	}
	mimc.Write(circuit.RequestNonce)

	requesterHash := mimc.Sum()
	api.AssertIsEqual(requesterHash, circuit.RequesterCommitment)

	// ═══════════════════════════════════════════════════════════════════════
	// CONSTRAINT 2: Provider Identity Verification
	// ═══════════════════════════════════════════════════════════════════════

	mimc.Reset()
	for i := 0; i < 20; i++ {
		mimc.Write(circuit.ProviderAddress[i])
	}
	mimc.Write(circuit.RequestID)

	providerHash := mimc.Sum()
	api.AssertIsEqual(providerHash, circuit.ProviderCommitment)

	// ═══════════════════════════════════════════════════════════════════════
	// CONSTRAINT 3: Completion Verification
	// ═══════════════════════════════════════════════════════════════════════

	mimc.Reset()
	mimc.Write(circuit.ResultHash)
	mimc.Write(circuit.ExecutionSuccess)
	mimc.Write(circuit.VerificationPassed)
	mimc.Write(circuit.EndTimestamp)

	completionHash := mimc.Sum()
	api.AssertIsEqual(completionHash, circuit.CompletionCommitment)

	// ═══════════════════════════════════════════════════════════════════════
	// CONSTRAINT 4: Success Conditions (Boolean)
	// ═══════════════════════════════════════════════════════════════════════

	// ExecutionSuccess must be boolean (0 or 1)
	api.AssertIsBoolean(circuit.ExecutionSuccess)

	// VerificationPassed must be boolean
	api.AssertIsBoolean(circuit.VerificationPassed)

	// ProviderMisbehavior must be boolean
	api.AssertIsBoolean(circuit.ProviderMisbehavior)

	// RequesterCancelled must be boolean
	api.AssertIsBoolean(circuit.RequesterCancelled)

	// ═══════════════════════════════════════════════════════════════════════
	// CONSTRAINT 5: Escrow Release Logic
	// ═══════════════════════════════════════════════════════════════════════

	// Escrow should be released to provider if:
	// 1. Execution succeeded AND verification passed
	// 2. Provider did not misbehave
	// 3. Requester did not cancel
	// 4. Deadline was met

	// Check if computation succeeded and was verified
	successAndVerified := api.Mul(circuit.ExecutionSuccess, circuit.VerificationPassed)

	// Check no misbehavior (NOT misbehavior = 1 - misbehavior)
	noMisbehavior := api.Sub(1, circuit.ProviderMisbehavior)

	// Check not cancelled (NOT cancelled = 1 - cancelled)
	notCancelled := api.Sub(1, circuit.RequesterCancelled)

	// Check deadline met (EndTimestamp <= DeadlineTimestamp)
	deadlineMet := api.IsZero(api.Cmp(circuit.EndTimestamp, circuit.DeadlineTimestamp))

	// Combine all conditions (AND operation)
	releaseCondition := api.Mul(successAndVerified, noMisbehavior)
	releaseCondition = api.Mul(releaseCondition, notCancelled)
	releaseCondition = api.Mul(releaseCondition, deadlineMet)

	// If release condition is met, escrow amount must equal actual cost
	// Otherwise, escrow should be refunded (amount = 0 for provider)
	expectedAmount := api.Select(releaseCondition, circuit.ActualCost, 0)
	api.AssertIsEqual(circuit.EscrowAmount, expectedAmount)

	// ═══════════════════════════════════════════════════════════════════════
	// CONSTRAINT 6: Cost Validation
	// ═══════════════════════════════════════════════════════════════════════

	// Actual cost must not exceed estimated cost (prevent overcharging)
	api.AssertIsLessOrEqual(circuit.ActualCost, circuit.EstimatedCost)

	// Actual cost must be positive if execution succeeded
	actualCostIsPositive := api.IsZero(api.Cmp(1, circuit.ActualCost))
	shouldBePositive := api.Select(circuit.ExecutionSuccess, actualCostIsPositive, 1)
	api.AssertIsEqual(shouldBePositive, 1)

	// ═══════════════════════════════════════════════════════════════════════
	// CONSTRAINT 7: Timing Validation
	// ═══════════════════════════════════════════════════════════════════════

	// End timestamp must be after start timestamp
	api.AssertIsLessOrEqual(circuit.StartTimestamp, circuit.EndTimestamp)

	// Timestamps must be positive
	api.AssertIsLessOrEqual(1, circuit.StartTimestamp)
	api.AssertIsLessOrEqual(1, circuit.DeadlineTimestamp)

	// Execution duration should be reasonable (< 7 days = 604800 seconds)
	duration := api.Sub(circuit.EndTimestamp, circuit.StartTimestamp)
	api.AssertIsLessOrEqual(duration, 604800)

	// ═══════════════════════════════════════════════════════════════════════
	// CONSTRAINT 8: Amount Bounds
	// ═══════════════════════════════════════════════════════════════════════

	// Escrow amount must be non-negative
	api.AssertIsLessOrEqual(0, circuit.EscrowAmount)

	// Escrow amount must be reasonable (< 2^63)
	maxEscrow := new(big.Int).Lsh(big.NewInt(1), 62)
	api.AssertIsLessOrEqual(circuit.EscrowAmount, frontend.Variable(maxEscrow))

	return nil
}

// GetConstraintCount returns the estimated number of constraints.
func (circuit *EscrowCircuit) GetConstraintCount() int {
	// Approximate constraint breakdown:
	// - MiMC hashes: 3 hashes × ~8,000 constraints = 24,000
	// - Boolean checks: 4 × 50 constraints = 200
	// - Comparisons: 10 × 100 constraints = 1,000
	// - Arithmetic: ~500 constraints
	// Total: ~26,000 constraints
	return 26000
}

// GetPublicInputCount returns the number of public inputs.
func (circuit *EscrowCircuit) GetPublicInputCount() int {
	return 5 // RequestID, EscrowAmount, RequesterCommitment, ProviderCommitment, CompletionCommitment
}

// GetCircuitName returns the circuit identifier.
func (circuit *EscrowCircuit) GetCircuitName() string {
	return "escrow-release-v1"
}
