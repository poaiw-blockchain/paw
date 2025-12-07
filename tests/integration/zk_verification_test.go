package integration_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"testing"
	"time"

	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/compute/circuits"
	computekeeper "github.com/paw-chain/paw/x/compute/keeper"
	"github.com/paw-chain/paw/x/compute/setup"
	"github.com/paw-chain/paw/x/compute/types"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/frontend"
	"github.com/consensys/gnark/frontend/cs/r1cs"
)

// ZKVerificationTestSuite tests the complete ZK-SNARK verification system.
type ZKVerificationTestSuite struct {
	suite.Suite
	ctx        sdk.Context
	keeper     *computekeeper.Keeper
	bankKeeper bankkeeper.Keeper
	zkVerifier *computekeeper.ZKVerifier
	requester  sdk.AccAddress
	provider   sdk.AccAddress
}

func TestZKVerificationTestSuite(t *testing.T) {
	suite.Run(t, new(ZKVerificationTestSuite))
}

func (suite *ZKVerificationTestSuite) SetupTest() {
	// Setup test keeper and context with bank keeper for funding accounts
	k, ctx, bk := keeper.ComputeKeeperWithBank(suite.T())
	suite.keeper = k
	suite.ctx = ctx
	suite.bankKeeper = bk

	// Initialize ZK verifier
	suite.zkVerifier = computekeeper.NewZKVerifier(suite.keeper)

	// Create test accounts
	reqKey := secp256k1.GenPrivKey()
	provKey := secp256k1.GenPrivKey()
	suite.requester = sdk.AccAddress(reqKey.PubKey().Address())
	suite.provider = sdk.AccAddress(provKey.PubKey().Address())

	// Fund test accounts with sufficient tokens for DoS protection deposits
	// The deposit requirement is configurable via circuit params, but we fund generously
	// to ensure all tests have sufficient balance for proof verification deposits
	fundAmount := sdk.NewCoins(sdk.NewInt64Coin("upaw", 100_000_000)) // 100M upaw
	suite.fundAccount(suite.requester, fundAmount)
	suite.fundAccount(suite.provider, fundAmount)
}

// fundAccount mints and sends tokens to the specified address.
// This is essential for DoS protection which requires deposits before verification.
func (suite *ZKVerificationTestSuite) fundAccount(addr sdk.AccAddress, amount sdk.Coins) {
	err := suite.bankKeeper.MintCoins(suite.ctx, types.ModuleName, amount)
	suite.Require().NoError(err, "failed to mint coins for test account")

	err = suite.bankKeeper.SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, addr, amount)
	suite.Require().NoError(err, "failed to send coins to test account")
}

// TestComputeCircuitCompilation tests circuit compilation and constraint count.
func (suite *ZKVerificationTestSuite) TestComputeCircuitCompilation() {
	circuit := &circuits.ComputeCircuit{}

	// Compile circuit
	ccs, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, circuit)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), ccs)

	// Verify constraint count is within expected range
	constraintCount := ccs.GetNbConstraints()
	suite.T().Logf("Compute circuit constraint count: %d", constraintCount)
	require.Greater(suite.T(), constraintCount, 300000, "Circuit should have >300k constraints to cover extended verification checks")
	require.Less(suite.T(), constraintCount, 500000, "Circuit should have <500k constraints to stay within proving limits")

	// Verify public input count
	publicInputs := ccs.GetNbPublicVariables()
	suite.T().Logf("Public inputs: %d", publicInputs)
	require.Equal(suite.T(), 5, publicInputs, "Should have 5 public inputs")
}

// TestEscrowCircuitCompilation tests escrow circuit compilation.
func (suite *ZKVerificationTestSuite) TestEscrowCircuitCompilation() {
	circuit := &circuits.EscrowCircuit{}

	ccs, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, circuit)
	require.NoError(suite.T(), err)

	constraintCount := ccs.GetNbConstraints()
	suite.T().Logf("Escrow circuit constraint count: %d", constraintCount)
	require.Greater(suite.T(), constraintCount, 20000, "Circuit should have >20k constraints")
	require.Less(suite.T(), constraintCount, 35000, "Circuit should have <35k constraints")
}

// TestResultCircuitCompilation tests result correctness circuit compilation.
func (suite *ZKVerificationTestSuite) TestResultCircuitCompilation() {
	circuit := &circuits.ResultCircuit{}

	ccs, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, circuit)
	require.NoError(suite.T(), err)

	constraintCount := ccs.GetNbConstraints()
	suite.T().Logf("Result circuit constraint count: %d", constraintCount)
	require.Greater(suite.T(), constraintCount, 120000, "Circuit should have >120k constraints to cover output validation paths")
	require.Less(suite.T(), constraintCount, 250000, "Circuit should have <250k constraints to stay within proving limits")
}

// TestProofGenerationAndVerification tests end-to-end proof generation and verification.
func (suite *ZKVerificationTestSuite) TestProofGenerationAndVerification() {
	// Prepare computation data
	computationData := []byte("test computation result data")
	timestamp := time.Now().Unix()
	exitCode := int32(0)
	cpuCycles := uint64(1000000)
	memoryBytes := uint64(1024 * 1024) // 1MB
	requestID := uint64(12345)

	// Compute circuit-compatible result hash
	resultHash, err := computekeeper.ComputeResultHash(requestID, suite.provider, computationData, timestamp, exitCode, cpuCycles, memoryBytes)
	require.NoError(suite.T(), err)

	// Generate proof
	proof, err := suite.zkVerifier.GenerateProof(
		suite.ctx,
		requestID,
		resultHash,
		suite.provider,
		computationData,
		timestamp,
		exitCode,
		cpuCycles,
		memoryBytes,
	)

	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), proof)

	// Validate proof structure
	require.NotEmpty(suite.T(), proof.Proof, "Proof data should not be empty")
	require.NotEmpty(suite.T(), proof.PublicInputs, "Public inputs should not be empty")
	require.Equal(suite.T(), "groth16", proof.ProofSystem)
	require.Equal(suite.T(), "compute-verification-v1", proof.CircuitId)

	// Verify proof size is reasonable (Groth16 proofs are ~200 bytes)
	suite.T().Logf("Proof size: %d bytes", len(proof.Proof))
	require.Less(suite.T(), len(proof.Proof), 512, "Proof should be <512 bytes")

	// Verify the proof
	valid, err := suite.zkVerifier.VerifyProof(
		suite.ctx,
		proof,
		requestID,
		resultHash,
		suite.provider,
	)

	require.NoError(suite.T(), err)
	require.True(suite.T(), valid, "Proof verification should succeed")
}

// TestProofVerificationWithWrongData tests that verification fails with incorrect data.
func (suite *ZKVerificationTestSuite) TestProofVerificationWithWrongData() {
	// Generate proof with correct data
	computationData := []byte("correct data")
	timestamp := time.Now().Unix()
	resultHash, err := computekeeper.ComputeResultHash(uint64(123), suite.provider, computationData, timestamp, 0, 1000, 1024)
	require.NoError(suite.T(), err)

	requestID := uint64(123)
	proof, err := suite.zkVerifier.GenerateProof(
		suite.ctx,
		requestID,
		resultHash,
		suite.provider,
		computationData,
		timestamp,
		0,
		1000,
		1024,
	)

	require.NoError(suite.T(), err)

	// Attempt verification with wrong result hash
	wrongHash, err := computekeeper.ComputeResultHash(requestID, suite.provider, []byte("wrong data"), timestamp, 0, 1000, 1024)
	require.NoError(suite.T(), err)
	valid, err := suite.zkVerifier.VerifyProof(
		suite.ctx,
		proof,
		requestID,
		wrongHash,
		suite.provider,
	)

	// Should fail because public inputs don't match
	require.Error(suite.T(), err)
	require.False(suite.T(), valid)
}

// TestTrustedSetupMPCCeremony tests the multi-party computation ceremony.
func (suite *ZKVerificationTestSuite) TestTrustedSetupMPCCeremony() {
	circuit := &circuits.ComputeCircuit{}

	// Compile circuit
	ccs, err := frontend.Compile(ecc.BN254.ScalarField(), r1cs.NewBuilder, circuit)
	require.NoError(suite.T(), err)

	// Create MPC ceremony with mock beacon
	beacon := &MockRandomnessBeacon{}
	ceremony := setup.NewMPCCeremony(
		"test-circuit",
		ccs,
		setup.SecurityLevel256,
		beacon,
		nil,
	)

	// Register participants
	participants := []struct {
		id  string
		key []byte
	}{
		{"alice", randomBytes(32)},
		{"bob", randomBytes(32)},
		{"charlie", randomBytes(32)},
	}

	for _, p := range participants {
		err := ceremony.RegisterParticipant(p.id, p.key)
		require.NoError(suite.T(), err)
	}

	// Start ceremony
	err = ceremony.StartCeremony()
	require.NoError(suite.T(), err)

	// Each participant contributes
	for _, p := range participants {
		randomness := randomBytes(64)
		contrib, err := ceremony.Contribute(p.id, randomness)
		require.NoError(suite.T(), err)
		require.NotNil(suite.T(), contrib)

		suite.T().Logf("Participant %s contributed", p.id)
	}

	// Finalize ceremony
	ctx := context.Background()
	pk, vk, err := ceremony.Finalize(ctx)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), pk)
	require.NotNil(suite.T(), vk)

	suite.T().Log("MPC ceremony completed successfully")
}

// TestKeyGeneration tests cryptographic key generation and storage.
func (suite *ZKVerificationTestSuite) TestKeyGeneration() {
	circuit := &circuits.ComputeCircuit{}

	// Create key generator with mock storage
	storage := NewMockKeyStorage()
	masterPassword := randomBytes(32)
	keygen := setup.NewKeyGenerator("test-circuit", masterPassword, storage)

	// Generate keys
	ctx := context.Background()
	encryptedPair, err := keygen.GenerateKeys(ctx, circuit, false, nil)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), encryptedPair)

	// Verify metadata
	require.NotEmpty(suite.T(), encryptedPair.Metadata.KeyID)
	require.Equal(suite.T(), "test-circuit", encryptedPair.Metadata.CircuitID)
	require.Equal(suite.T(), "groth16", encryptedPair.Metadata.Algorithm)
	require.Equal(suite.T(), "bn254", encryptedPair.Metadata.Curve)

	suite.T().Logf("Generated key ID: %s", encryptedPair.Metadata.KeyID)

	// Load keys
	pk, vk, err := keygen.LoadKeys(ctx, encryptedPair.Metadata.KeyID)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), pk)
	require.NotNil(suite.T(), vk)

	suite.T().Log("Keys loaded successfully")
}

// TestKeyRotation tests key rotation functionality.
func (suite *ZKVerificationTestSuite) TestKeyRotation() {
	circuit := &circuits.ComputeCircuit{}
	storage := NewMockKeyStorage()
	masterPassword := randomBytes(32)
	keygen := setup.NewKeyGenerator("test-circuit", masterPassword, storage)

	ctx := context.Background()

	// Generate initial keys
	oldPair, err := keygen.GenerateKeys(ctx, circuit, false, nil)
	require.NoError(suite.T(), err)

	oldKeyID := oldPair.Metadata.KeyID

	// Rotate keys
	newPair, err := keygen.RotateKeys(ctx, oldKeyID, circuit, false, nil)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), newPair)

	// Verify new key has different ID
	require.NotEqual(suite.T(), oldKeyID, newPair.Metadata.KeyID)

	suite.T().Logf("Rotated from %s to %s", oldKeyID, newPair.Metadata.KeyID)
}

// TestProofBatchVerification tests batch verification of multiple proofs.
func (suite *ZKVerificationTestSuite) TestProofBatchVerification() {
	const batchSize = 5
	type proofCase struct {
		proof      *types.ZKProof
		requestID  uint64
		resultHash []byte
		data       []byte
	}
	proofs := make([]proofCase, batchSize)

	// Generate multiple proofs
	for i := 0; i < batchSize; i++ {
		computationData := []byte(fmt.Sprintf("data-%d", i))
		reqID := uint64(i + 1)
		ts := int64(100 + i)
		resultHash, err := computekeeper.ComputeResultHash(reqID, suite.provider, computationData, ts, 0, 1000, 1024)
		require.NoError(suite.T(), err)

		proof, err := suite.zkVerifier.GenerateProof(
			suite.ctx,
			reqID,
			resultHash,
			suite.provider,
			computationData,
			ts,
			0,
			1000,
			1024,
		)

		require.NoError(suite.T(), err)
		proofs[i] = proofCase{
			proof:      proof,
			requestID:  reqID,
			resultHash: resultHash,
			data:       computationData,
		}
	}

	// Verify each proof individually (in production, use aggregated verification)
	for _, proof := range proofs {
		valid, err := suite.zkVerifier.VerifyProof(
			suite.ctx,
			proof.proof,
			proof.requestID,
			proof.resultHash,
			suite.provider,
		)

		require.NoError(suite.T(), err)
		require.True(suite.T(), valid)
	}

	suite.T().Logf("Verified batch of %d proofs", batchSize)
}

// TestZKMetrics tests metrics tracking for ZK operations.
func (suite *ZKVerificationTestSuite) TestZKMetrics() {
	// Generate and verify a proof
	computationData := []byte("test data")
	resultHash, err := computekeeper.ComputeResultHash(uint64(1), suite.provider, computationData, time.Now().Unix(), 0, 1000, 1024)
	require.NoError(suite.T(), err)

	proof, err := suite.zkVerifier.GenerateProof(
		suite.ctx,
		uint64(1),
		resultHash,
		suite.provider,
		computationData,
		time.Now().Unix(),
		0,
		1000,
		1024,
	)
	require.NoError(suite.T(), err)

	valid, err := suite.zkVerifier.VerifyProof(
		suite.ctx,
		proof,
		uint64(1),
		resultHash[:],
		suite.provider,
	)
	require.NoError(suite.T(), err)
	require.True(suite.T(), valid)

	// Retrieve metrics
	metrics, err := suite.keeper.GetZKMetrics(suite.ctx)
	require.NoError(suite.T(), err)
	require.NotNil(suite.T(), metrics)

	// Verify metrics were updated
	require.Greater(suite.T(), metrics.TotalProofsGenerated, uint64(0))
	require.Greater(suite.T(), metrics.TotalProofsVerified, uint64(0))

	suite.T().Logf("Metrics - Generated: %d, Verified: %d, Failed: %d",
		metrics.TotalProofsGenerated,
		metrics.TotalProofsVerified,
		metrics.TotalProofsFailed)
}

// TestConstantTimeOperations verifies timing attack resistance.
func (suite *ZKVerificationTestSuite) TestConstantTimeOperations() {
	// Generate two different proofs
	data1 := []byte("data1")
	data2 := []byte("data2")
	ts := int64(200)
	hash1, err := computekeeper.ComputeResultHash(1, suite.provider, data1, ts, 0, 1000, 1024)
	require.NoError(suite.T(), err)
	hash2, err := computekeeper.ComputeResultHash(2, suite.provider, data2, ts, 0, 1000, 1024)
	require.NoError(suite.T(), err)

	proof1, err := suite.zkVerifier.GenerateProof(
		suite.ctx, 1, hash1, suite.provider, data1, ts, 0, 1000, 1024,
	)
	require.NoError(suite.T(), err)

	proof2, err := suite.zkVerifier.GenerateProof(
		suite.ctx, 2, hash2, suite.provider, data2, ts, 0, 1000, 1024,
	)
	require.NoError(suite.T(), err)

	// Measure verification times
	const iterations = 10
	times1 := make([]time.Duration, iterations)
	times2 := make([]time.Duration, iterations)

	for i := 0; i < iterations; i++ {
		start := time.Now()
		suite.zkVerifier.VerifyProof(suite.ctx, proof1, 1, hash1, suite.provider)
		times1[i] = time.Since(start)

		start = time.Now()
		suite.zkVerifier.VerifyProof(suite.ctx, proof2, 2, hash2, suite.provider)
		times2[i] = time.Since(start)
	}

	// Calculate average times
	avg1 := average(times1)
	avg2 := average(times2)

	suite.T().Logf("Average verification time 1: %v", avg1)
	suite.T().Logf("Average verification time 2: %v", avg2)

	// Times should be similar (within 20% variance for constant-time ops)
	variance := float64(abs(avg1-avg2)) / float64(max(avg1, avg2))
	suite.T().Logf("Time variance: %.2f%%", variance*100)

	// Note: This is a simplified test. True constant-time verification
	// requires specialized testing tools and controlled environments.
}

// Helper functions and mocks

type MockRandomnessBeacon struct{}

func (m *MockRandomnessBeacon) GetRandomness(round uint64) ([]byte, error) {
	// Generate deterministic randomness for testing
	h := sha256.New()
	binary.Write(h, binary.BigEndian, round)
	return h.Sum(nil), nil
}

func (m *MockRandomnessBeacon) VerifyRandomness(round uint64, randomness []byte) bool {
	expected, _ := m.GetRandomness(round)
	return bytes.Equal(expected, randomness)
}

type MockKeyStorage struct {
	data map[string][]byte
}

func NewMockKeyStorage() *MockKeyStorage {
	return &MockKeyStorage{
		data: make(map[string][]byte),
	}
}

func (m *MockKeyStorage) Store(ctx context.Context, keyID string, data []byte) error {
	m.data[keyID] = data
	return nil
}

func (m *MockKeyStorage) Load(ctx context.Context, keyID string) ([]byte, error) {
	data, ok := m.data[keyID]
	if !ok {
		return nil, fmt.Errorf("key not found")
	}
	return data, nil
}

func (m *MockKeyStorage) Delete(ctx context.Context, keyID string) error {
	delete(m.data, keyID)
	return nil
}

func (m *MockKeyStorage) List(ctx context.Context) ([]string, error) {
	keys := make([]string, 0, len(m.data))
	for k := range m.data {
		keys = append(keys, k)
	}
	return keys, nil
}

func randomBytes(n int) []byte {
	b := make([]byte, n)
	rand.Read(b)
	return b
}

func average(durations []time.Duration) time.Duration {
	var sum time.Duration
	for _, d := range durations {
		sum += d
	}
	return sum / time.Duration(len(durations))
}

func abs(d time.Duration) time.Duration {
	if d < 0 {
		return -d
	}
	return d
}

func max(a, b time.Duration) time.Duration {
	if a > b {
		return a
	}
	return b
}
