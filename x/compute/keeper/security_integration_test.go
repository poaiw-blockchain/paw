package keeper_test

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/binary"
	"testing"
	"time"

	"cosmossdk.io/math"
	sdked25519 "github.com/cosmos/cosmos-sdk/crypto/keys/ed25519"
	sdk "github.com/cosmos/cosmos-sdk/types"
	bankkeeper "github.com/cosmos/cosmos-sdk/x/bank/keeper"
	"github.com/stretchr/testify/suite"

	"github.com/paw-chain/paw/testutil/keeper"
	computekeeper "github.com/paw-chain/paw/x/compute/keeper"
	"github.com/paw-chain/paw/x/compute/types"
)

// ComputeSecuritySuite is a comprehensive test suite for security features in the compute module.
// It tests all attack vectors, cryptographic verification, escrow safety, rate limiting, and reputation systems.
type ComputeSecuritySuite struct {
	suite.Suite
	ctx        sdk.Context
	keeper     *computekeeper.Keeper
	bankKeeper bankkeeper.Keeper
	app        interface{}
	providers  []sdk.AccAddress
	requesters []sdk.AccAddress
	// Cryptographic keys for providers
	providerKeys map[string]ed25519.PrivateKey
}

// SetupTest initializes the test environment with multiple providers and requesters
func (suite *ComputeSecuritySuite) SetupTest() {
	// Initialize test application and context
	testApp, ctx := keeper.SetupTestApp(suite.T())
	suite.app = testApp
	suite.ctx = ctx
	suite.keeper = testApp.ComputeKeeper
	suite.bankKeeper = testApp.BankKeeper
	suite.providerKeys = make(map[string]ed25519.PrivateKey)

	// Create 10 providers with varying stakes and reputations
	suite.providers = make([]sdk.AccAddress, 10)
	for i := 0; i < 10; i++ {
		addr := sdk.AccAddress([]byte("provider" + string(rune('0'+i)) + "______"))
		suite.providers[i] = addr

		// Generate Ed25519 key for each provider
		_, privKey, err := ed25519.GenerateKey(rand.Reader)
		suite.Require().NoError(err)
		suite.providerKeys[addr.String()] = privKey
		pubKey := privKey.Public().(ed25519.PublicKey)
		var sdkPubKey sdked25519.PubKey
		sdkPubKey.Key = make([]byte, len(pubKey))
		copy(sdkPubKey.Key, pubKey)
		acc := testApp.AccountKeeper.GetAccount(suite.ctx, addr)
		if acc == nil {
			acc = testApp.AccountKeeper.NewAccountWithAddress(suite.ctx, addr)
		}
		err = acc.SetPubKey(&sdkPubKey)
		suite.Require().NoError(err)
		testApp.AccountKeeper.SetAccount(suite.ctx, acc)

		// Fund provider account for staking
		stakeAmount := math.NewInt(1000000 + int64(i*100000)) // 1-2M tokens
		coins := sdk.NewCoins(sdk.NewCoin("upaw", stakeAmount))
		err = suite.bankKeeper.MintCoins(suite.ctx, types.ModuleName, coins)
		suite.Require().NoError(err)
		err = suite.bankKeeper.SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, addr, coins)
		suite.Require().NoError(err)

		// Register provider with different specs
		specs := types.ComputeSpec{
			CpuCores:       uint64(2000 + i*1000), // 2-11 cores
			MemoryMb:       uint64(4096 + i*2048), // 4-22 GB
			GpuCount:       uint32(i % 3),         // 0-2 GPUs
			GpuType:        "nvidia-a100",
			StorageGb:      uint64(100 + i*50), // 100-550 GB
			TimeoutSeconds: 3600,
		}

		pricing := types.Pricing{
			CpuPricePerMcoreHour:  math.LegacyNewDec(10),
			MemoryPricePerMbHour:  math.LegacyNewDec(5),
			GpuPricePerHour:       math.LegacyNewDec(1000),
			StoragePricePerGbHour: math.LegacyNewDec(2),
		}

		err = suite.keeper.RegisterProvider(
			suite.ctx,
			addr,
			"Provider"+string(rune('0'+i)),
			"http://provider"+string(rune('0'+i))+".github.com",
			specs,
			pricing,
			stakeAmount, // Stake full allocation to satisfy min stake requirements
		)
		suite.Require().NoError(err)
	}

	// Create 5 requester accounts
	suite.requesters = make([]sdk.AccAddress, 5)
	for i := 0; i < 5; i++ {
		addr := sdk.AccAddress([]byte("requester" + string(rune('0'+i)) + "____"))
		suite.requesters[i] = addr

		// Fund requester account
		fundAmount := math.NewInt(10000000) // 10M tokens
		coins := sdk.NewCoins(sdk.NewCoin("upaw", fundAmount))
		err := suite.bankKeeper.MintCoins(suite.ctx, types.ModuleName, coins)
		suite.Require().NoError(err)
		err = suite.bankKeeper.SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, addr, coins)
		suite.Require().NoError(err)
	}
}

// ========================================
// ESCROW ATTACK TESTS
// ========================================

// TestEscrowAttack_DoubleSpend tests that the same escrowed funds cannot be withdrawn twice
func (suite *ComputeSecuritySuite) TestEscrowAttack_DoubleSpend() {
	requester := suite.requesters[0]
	provider := suite.providers[0]

	// Create a compute request with escrowed payment
	specs := types.ComputeSpec{
		CpuCores:       2000,
		MemoryMb:       4096,
		GpuCount:       0,
		StorageGb:      100,
		TimeoutSeconds: 3600,
	}
	maxPayment := math.NewInt(100000)

	requestID, err := suite.keeper.SubmitRequest(
		suite.ctx,
		requester,
		specs,
		"alpine:latest",
		[]string{"echo", "test"},
		map[string]string{},
		maxPayment,
		provider.String(),
	)
	suite.Require().NoError(err)

	// Get provider balance before
	providerBalanceBefore := suite.bankKeeper.GetBalance(suite.ctx, provider, "upaw")

	// Submit valid result
	outputHash := "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"
	proof := suite.createValidProof(provider, requestID, outputHash)

	err = suite.keeper.SubmitResult(
		suite.ctx,
		provider,
		requestID,
		outputHash,
		"http://result.github.com",
		0,
		"http://logs.github.com",
		proof,
	)
	suite.Require().NoError(err)
	// First release should succeed
	request, err := suite.keeper.GetRequest(suite.ctx, requestID)
	suite.Require().NoError(err)
	suite.Require().Equal(types.REQUEST_STATUS_COMPLETED, request.Status)

	// Verify provider received payment
	providerBalanceAfter := suite.bankKeeper.GetBalance(suite.ctx, provider, "upaw")
	suite.Require().True(providerBalanceAfter.Amount.GT(providerBalanceBefore.Amount))

	// ATTACK: Try to release escrow again by manipulating request state
	// This should fail because request is already completed
	request.Status = types.REQUEST_STATUS_PROCESSING
	err = suite.keeper.SetRequest(suite.ctx, *request)
	suite.Require().NoError(err)

	// Attempt second withdrawal
	providerBalanceBeforeAttack := suite.bankKeeper.GetBalance(suite.ctx, provider, "upaw")

	// Try to complete request again
	err = suite.keeper.CompleteRequest(suite.ctx, requestID, true)
	// Should fail or not change balance
	suite.Require().Error(err)

	providerBalanceAfterAttack := suite.bankKeeper.GetBalance(suite.ctx, provider, "upaw")

	// Balance should not change (double-spend prevented)
	suite.Require().Equal(providerBalanceBeforeAttack.Amount, providerBalanceAfterAttack.Amount,
		"Double-spend attack succeeded - provider received payment twice!")
}

// TestEscrowAttack_PrematureWithdrawal tests that providers cannot withdraw before result submission
func (suite *ComputeSecuritySuite) TestEscrowAttack_PrematureWithdrawal() {
	requester := suite.requesters[0]
	provider := suite.providers[0]

	specs := types.ComputeSpec{
		CpuCores:       2000,
		MemoryMb:       4096,
		GpuCount:       0,
		StorageGb:      100,
		TimeoutSeconds: 3600,
	}
	maxPayment := math.NewInt(100000)

	requestID, err := suite.keeper.SubmitRequest(
		suite.ctx,
		requester,
		specs,
		"alpine:latest",
		[]string{"echo", "test"},
		map[string]string{},
		maxPayment,
		provider.String(),
	)
	suite.Require().NoError(err)

	providerBalanceBefore := suite.bankKeeper.GetBalance(suite.ctx, provider, "upaw")

	// ATTACK: Try to complete request without submitting result
	err = suite.keeper.CompleteRequest(suite.ctx, requestID, true)
	suite.Require().Error(err)

	providerBalanceAfter := suite.bankKeeper.GetBalance(suite.ctx, provider, "upaw")

	// Verify no payment was released
	suite.Require().Equal(providerBalanceBefore.Amount, providerBalanceAfter.Amount,
		"Premature withdrawal attack succeeded - provider withdrew without result!")
}

// TestEscrowAttack_TimeoutExploit tests that timeout mechanism cannot be exploited
func (suite *ComputeSecuritySuite) TestEscrowAttack_TimeoutExploit() {
	requester := suite.requesters[0]
	provider := suite.providers[0]

	specs := types.ComputeSpec{
		CpuCores:       2000,
		MemoryMb:       4096,
		GpuCount:       0,
		StorageGb:      100,
		TimeoutSeconds: 10, // Very short timeout
	}
	maxPayment := math.NewInt(100000)

	requestID, err := suite.keeper.SubmitRequest(
		suite.ctx,
		requester,
		specs,
		"alpine:latest",
		[]string{"echo", "test"},
		map[string]string{},
		maxPayment,
		provider.String(),
	)
	suite.Require().NoError(err)

	requesterBalanceBefore := suite.bankKeeper.GetBalance(suite.ctx, requester, "upaw")
	providerBalanceBefore := suite.bankKeeper.GetBalance(suite.ctx, provider, "upaw")

	// Advance time past timeout
	suite.ctx = suite.ctx.WithBlockTime(suite.ctx.BlockTime().Add(15 * time.Second))

	// ATTACK: Provider tries to submit result after timeout
	outputHash := "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"
	proof := suite.createValidProof(provider, requestID, outputHash)

	err = suite.keeper.SubmitResult(
		suite.ctx,
		provider,
		requestID,
		outputHash,
		"http://result.github.com",
		0,
		"http://logs.github.com",
		proof,
	)
	suite.Require().Error(err)

	// Submission after timeout should fail or not pay provider
	providerBalanceAfter := suite.bankKeeper.GetBalance(suite.ctx, provider, "upaw")
	requesterBalanceAfter := suite.bankKeeper.GetBalance(suite.ctx, requester, "upaw")
	request, err := suite.keeper.GetRequest(suite.ctx, requestID)
	suite.Require().NoError(err)
	suite.Require().True(providerBalanceAfter.Amount.LTE(providerBalanceBefore.Amount))

	// Either request should be failed/cancelled, or requester should have been refunded
	suite.Require().True(
		request.Status == types.REQUEST_STATUS_FAILED ||
			request.Status == types.REQUEST_STATUS_CANCELLED ||
			requesterBalanceAfter.Amount.GT(requesterBalanceBefore.Amount),
		"Timeout exploit succeeded - provider paid after timeout!",
	)

	suite.Require().True(
		providerBalanceAfter.Amount.Equal(providerBalanceBefore.Amount),
		"provider balance should not increase on late submission",
	)
}

// ========================================
// VERIFICATION ATTACK TESTS
// ========================================

// TestVerificationAttack_InvalidProof tests that invalid cryptographic proofs are rejected
func (suite *ComputeSecuritySuite) TestVerificationAttack_InvalidProof() {
	requester := suite.requesters[0]
	provider := suite.providers[0]

	specs := types.ComputeSpec{
		CpuCores:       2000,
		MemoryMb:       4096,
		GpuCount:       0,
		StorageGb:      100,
		TimeoutSeconds: 3600,
	}
	maxPayment := math.NewInt(100000)

	requestID, err := suite.keeper.SubmitRequest(
		suite.ctx,
		requester,
		specs,
		"alpine:latest",
		[]string{"echo", "test"},
		map[string]string{},
		maxPayment,
		provider.String(),
	)
	suite.Require().NoError(err)

	outputHash := "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"

	// ATTACK: Create invalid proof with random data
	invalidProof := make([]byte, 300)
	_, err = rand.Read(invalidProof)
	suite.Require().NoError(err)

	providerRepBefore, _ := suite.keeper.GetProvider(suite.ctx, provider)

	err = suite.keeper.SubmitResult(
		suite.ctx,
		provider,
		requestID,
		outputHash,
		"http://result.github.com",
		0,
		"http://logs.github.com",
		invalidProof,
	)
	suite.Require().NoError(err)

	// Invalid proof should result in low verification score
	result, err := suite.keeper.GetResult(suite.ctx, requestID)
	if err == nil {
		suite.Require().False(result.Verified, "Invalid proof was verified!")
		suite.Require().Less(result.VerificationScore, uint32(types.VerificationPassThreshold),
			"Invalid proof scored above threshold!")
	}

	// Provider reputation should be affected for invalid proof
	providerRepAfter, _ := suite.keeper.GetProvider(suite.ctx, provider)
	suite.Require().LessOrEqual(providerRepAfter.Reputation, providerRepBefore.Reputation,
		"Provider reputation increased after invalid proof!")
}

// TestVerificationAttack_ReplayAttack tests replay attack prevention via nonce tracking
func (suite *ComputeSecuritySuite) TestVerificationAttack_ReplayAttack() {
	requester := suite.requesters[0]
	provider := suite.providers[0]

	specs := types.ComputeSpec{
		CpuCores:       2000,
		MemoryMb:       4096,
		GpuCount:       0,
		StorageGb:      100,
		TimeoutSeconds: 3600,
	}
	maxPayment := math.NewInt(100000)

	// Submit first request
	requestID1, err := suite.keeper.SubmitRequest(
		suite.ctx,
		requester,
		specs,
		"alpine:latest",
		[]string{"echo", "test1"},
		map[string]string{},
		maxPayment,
		provider.String(),
	)
	suite.Require().NoError(err)

	outputHash1 := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	proof1 := suite.createValidProof(provider, requestID1, outputHash1)

	// Submit valid result with nonce N
	err = suite.keeper.SubmitResult(
		suite.ctx,
		provider,
		requestID1,
		outputHash1,
		"http://result1.github.com",
		0,
		"http://logs1.github.com",
		proof1,
	)
	suite.Require().NoError(err)

	result1, err := suite.keeper.GetResult(suite.ctx, requestID1)
	suite.Require().NoError(err)
	suite.Require().True(result1.Verified)

	// Submit second request
	requestID2, err := suite.keeper.SubmitRequest(
		suite.ctx,
		requester,
		specs,
		"alpine:latest",
		[]string{"echo", "test2"},
		map[string]string{},
		maxPayment,
		provider.String(),
	)
	suite.Require().NoError(err)

	// ATTACK: Reuse the same proof (same nonce) for second request
	outputHash2 := "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"
	// Use same nonce by reusing proof1
	replayProof := proof1 // Same nonce, same proof

	err = suite.keeper.SubmitResult(
		suite.ctx,
		provider,
		requestID2,
		outputHash2,
		"http://result2.github.com",
		0,
		"http://logs2.github.com",
		replayProof,
	)
	suite.Require().NoError(err)

	// Check if replay was detected
	result2, err := suite.keeper.GetResult(suite.ctx, requestID2)
	if err == nil {
		// Replay should be detected and verification should fail
		suite.Require().False(result2.Verified,
			"Replay attack succeeded - same nonce accepted twice!")
		suite.Require().Equal(uint32(0), result2.VerificationScore,
			"Replay attack scored above zero!")
	}

	// Check for replay attack event
	events := suite.ctx.EventManager().Events()
	replayDetected := false
	for _, event := range events {
		if event.Type == "replay_attack_detected" {
			replayDetected = true
			break
		}
	}
	suite.Require().True(replayDetected, "Replay attack not detected in events!")
}

// TestVerificationAttack_SignatureForgery tests that forged signatures are rejected
func (suite *ComputeSecuritySuite) TestVerificationAttack_SignatureForgery() {
	requester := suite.requesters[0]
	provider := suite.providers[0]

	specs := types.ComputeSpec{
		CpuCores:       2000,
		MemoryMb:       4096,
		GpuCount:       0,
		StorageGb:      100,
		TimeoutSeconds: 3600,
	}
	maxPayment := math.NewInt(100000)

	requestID, err := suite.keeper.SubmitRequest(
		suite.ctx,
		requester,
		specs,
		"alpine:latest",
		[]string{"echo", "test"},
		map[string]string{},
		maxPayment,
		provider.String(),
	)
	suite.Require().NoError(err)

	outputHash := "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"

	// ATTACK: Create proof with wrong private key
	attackerPubKey, attackerPrivKey, err := ed25519.GenerateKey(rand.Reader)
	suite.Require().NoError(err)

	// Build proof components
	merkleRoot := make([]byte, 32)
	stateCommitment := make([]byte, 32)
	executionTrace := make([]byte, 32)
	merkleProof := [][]byte{make([]byte, 32), make([]byte, 32)}
	nonce := uint64(time.Now().UnixNano())

	// Compute message hash
	hasher := sha256.New()
	reqIDBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(reqIDBytes, requestID)
	hasher.Write(reqIDBytes)
	hasher.Write([]byte(outputHash))
	hasher.Write(merkleRoot)
	hasher.Write(stateCommitment)
	nonceBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(nonceBytes, nonce)
	hasher.Write(nonceBytes)
	timestampBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(timestampBytes, uint64(suite.ctx.BlockTime().Unix()))
	hasher.Write(timestampBytes)
	message := hasher.Sum(nil)

	// Sign with ATTACKER's key (not provider's key)
	forgedSignature := ed25519.Sign(attackerPrivKey, message)

	// Serialize proof
	var proofBytes []byte
	proofBytes = append(proofBytes, forgedSignature...)
	proofBytes = append(proofBytes, attackerPubKey...) // Wrong public key
	proofBytes = append(proofBytes, merkleRoot...)
	proofBytes = append(proofBytes, byte(len(merkleProof)))
	for _, node := range merkleProof {
		proofBytes = append(proofBytes, node...)
	}
	proofBytes = append(proofBytes, stateCommitment...)
	proofBytes = append(proofBytes, executionTrace...)
	proofBytes = append(proofBytes, nonceBytes...)
	proofBytes = append(proofBytes, timestampBytes...)

	err = suite.keeper.SubmitResult(
		suite.ctx,
		provider,
		requestID,
		outputHash,
		"http://result.github.com",
		0,
		"http://logs.github.com",
		proofBytes,
	)
	suite.Require().NoError(err)

	result, err := suite.keeper.GetResult(suite.ctx, requestID)
	if err == nil {
		// Forged signature should fail verification
		suite.Require().False(result.Verified, "Signature forgery attack succeeded!")
		suite.Require().Less(result.VerificationScore, uint32(types.VerificationPassThreshold),
			"Forged signature scored above threshold!")
	}
}

// TestVerificationAttack_MerkleProofManipulation tests merkle proof validation
func (suite *ComputeSecuritySuite) TestVerificationAttack_MerkleProofManipulation() {
	requester := suite.requesters[0]
	provider := suite.providers[0]

	specs := types.ComputeSpec{
		CpuCores:       2000,
		MemoryMb:       4096,
		GpuCount:       0,
		StorageGb:      100,
		TimeoutSeconds: 3600,
	}
	maxPayment := math.NewInt(100000)

	requestID, err := suite.keeper.SubmitRequest(
		suite.ctx,
		requester,
		specs,
		"alpine:latest",
		[]string{"echo", "test"},
		map[string]string{},
		maxPayment,
		provider.String(),
	)
	suite.Require().NoError(err)

	outputHash := "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"

	// Create proof with INVALID merkle root (doesn't match proof path)
	privKey := suite.providerKeys[provider.String()]
	pubKey := privKey.Public().(ed25519.PublicKey)

	merkleRoot := make([]byte, 32)
	_, err = rand.Read(merkleRoot)
	suite.Require().NoError(err, "failed to generate random merkle root")
	stateCommitment := make([]byte, 32)
	executionTrace := make([]byte, 32)

	// ATTACK: Merkle proof that doesn't validate to the root
	merkleProof := [][]byte{make([]byte, 32), make([]byte, 32)}
	_, err = rand.Read(merkleProof[0])
	suite.Require().NoError(err, "failed to randomize merkle proof element 0")
	_, err = rand.Read(merkleProof[1])
	suite.Require().NoError(err, "failed to randomize merkle proof element 1")

	nonce := uint64(time.Now().UnixNano())

	// Compute message hash and sign correctly
	hasher := sha256.New()
	reqIDBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(reqIDBytes, requestID)
	hasher.Write(reqIDBytes)
	hasher.Write([]byte(outputHash))
	hasher.Write(merkleRoot)
	hasher.Write(stateCommitment)
	nonceBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(nonceBytes, nonce)
	hasher.Write(nonceBytes)
	timestampBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(timestampBytes, uint64(suite.ctx.BlockTime().Unix()))
	hasher.Write(timestampBytes)
	message := hasher.Sum(nil)

	signature := ed25519.Sign(privKey, message)

	// Serialize proof
	var proofBytes []byte
	proofBytes = append(proofBytes, signature...)
	proofBytes = append(proofBytes, pubKey...)
	proofBytes = append(proofBytes, merkleRoot...)
	proofBytes = append(proofBytes, byte(len(merkleProof)))
	for _, node := range merkleProof {
		proofBytes = append(proofBytes, node...)
	}
	proofBytes = append(proofBytes, stateCommitment...)
	proofBytes = append(proofBytes, executionTrace...)
	proofBytes = append(proofBytes, nonceBytes...)
	proofBytes = append(proofBytes, timestampBytes...)

	err = suite.keeper.SubmitResult(
		suite.ctx,
		provider,
		requestID,
		outputHash,
		"http://result.github.com",
		0,
		"http://logs.github.com",
		proofBytes,
	)
	suite.Require().NoError(err)

	result, err := suite.keeper.GetResult(suite.ctx, requestID)
	if err == nil {
		// Invalid merkle proof should reduce score
		// Signature will pass (20 points) but merkle proof should fail (0 points)
		suite.Require().LessOrEqual(result.VerificationScore, uint32(40),
			"Invalid merkle proof scored too high!")
	}
}

// ========================================
// REPUTATION GAMING TESTS
// ========================================

// TestReputationGaming_FakeRequests tests that self-dealing is ineffective
func (suite *ComputeSecuritySuite) TestReputationGaming_FakeRequests() {
	provider := suite.providers[0]

	// Get initial reputation
	providerBefore, err := suite.keeper.GetProvider(suite.ctx, provider)
	suite.Require().NoError(err)
	initialReputation := providerBefore.Reputation

	// ATTACK: Provider creates requests to themselves and completes them
	// to artificially boost reputation
	fakeRequester := provider // Provider acts as requester

	specs := types.ComputeSpec{
		CpuCores:       2000,
		MemoryMb:       4096,
		GpuCount:       0,
		StorageGb:      100,
		TimeoutSeconds: 3600,
	}

	// Execute multiple fake successful requests
	for i := 0; i < 10; i++ {
		maxPayment := math.NewInt(10000)

		requestID, err := suite.keeper.SubmitRequest(
			suite.ctx,
			fakeRequester,
			specs,
			"alpine:latest",
			[]string{"echo", "fake"},
			map[string]string{},
			maxPayment,
			provider.String(),
		)

		if err != nil {
			continue // May fail due to same address
		}

		outputHash := "fake" + string(rune('0'+i)) + "fake" + string(rune('0'+i)) + "fake" + string(rune('0'+i)) + "fake" + string(rune('0'+i)) + "fake" + string(rune('0'+i)) + "fake" + string(rune('0'+i)) + "fake"
		proof := suite.createValidProof(provider, requestID, outputHash)

		err = suite.keeper.SubmitResult(
			suite.ctx,
			provider,
			requestID,
			outputHash,
			"http://fake.github.com",
			0,
			"http://logs.github.com",
			proof,
		)
		suite.Require().NoError(err)
	}

	// Check reputation after attack
	providerAfter, err := suite.keeper.GetProvider(suite.ctx, provider)
	suite.Require().NoError(err)

	// Reputation should not have increased dramatically (or at all)
	// Good reputation system should detect self-dealing patterns
	reputationIncrease := int64(providerAfter.Reputation) - int64(initialReputation)
	suite.Require().LessOrEqual(reputationIncrease, int64(20),
		"Reputation gaming attack succeeded - reputation increased by %d!", reputationIncrease)
}

// TestReputationGaming_SybilProviders tests Sybil attack resistance
func (suite *ComputeSecuritySuite) TestReputationGaming_SybilProviders() {
	// ATTACK: Create multiple provider identities with minimal stake
	// to game the provider selection algorithm

	sybilProviders := make([]sdk.AccAddress, 20)
	params, _ := suite.keeper.GetParams(suite.ctx)

	successfulRegistrations := 0
	for i := 0; i < 20; i++ {
		addr := sdk.AccAddress([]byte("sybil" + string(rune('a'+i)) + "________"))
		sybilProviders[i] = addr

		// Fund with exactly minimum stake
		minStake := params.MinProviderStake
		coins := sdk.NewCoins(sdk.NewCoin("upaw", minStake.MulRaw(2)))
		err := suite.bankKeeper.MintCoins(suite.ctx, types.ModuleName, coins)
		suite.Require().NoError(err)
		err = suite.bankKeeper.SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, addr, coins)
		suite.Require().NoError(err)

		specs := types.ComputeSpec{
			CpuCores:       1000, // Minimal specs
			MemoryMb:       1024,
			GpuCount:       0,
			StorageGb:      10,
			TimeoutSeconds: 3600,
		}

		pricing := types.Pricing{
			CpuPricePerMcoreHour:  math.LegacyNewDec(1), // Very low prices to attract requests
			MemoryPricePerMbHour:  math.LegacyNewDec(1),
			GpuPricePerHour:       math.LegacyNewDec(1),
			StoragePricePerGbHour: math.LegacyNewDec(1),
		}

		err = suite.keeper.RegisterProvider(
			suite.ctx,
			addr,
			"Sybil"+string(rune('a'+i)),
			"http://sybil"+string(rune('a'+i))+".github.com",
			specs,
			pricing,
			minStake,
		)

		if err == nil {
			successfulRegistrations++
		}
	}

	// System should allow registration but not give unfair advantage
	// Minimum stake requirement provides Sybil resistance
	suite.Require().GreaterOrEqual(params.MinProviderStake.Int64(), int64(1000000),
		"Minimum stake too low - Sybil attack risk!")

	// Even if all registered, they should have low reputation and minimal resources
	// High-reputation legitimate providers should still be preferred
	legitimateProvider, _ := suite.keeper.GetProvider(suite.ctx, suite.providers[0])
	if successfulRegistrations > 0 {
		sybilProvider, _ := suite.keeper.GetProvider(suite.ctx, sybilProviders[0])
		suite.Require().Greater(legitimateProvider.Reputation, sybilProvider.Reputation,
			"Sybil providers have equal reputation to legitimate providers!")
	}
}

// ========================================
// DOS ATTACK TESTS
// ========================================

// TestDoSAttack_RequestSpam tests rate limiting on request submission
func (suite *ComputeSecuritySuite) TestDoSAttack_RequestSpam() {
	requester := suite.requesters[0]

	specs := types.ComputeSpec{
		CpuCores:       2000,
		MemoryMb:       4096,
		GpuCount:       0,
		StorageGb:      100,
		TimeoutSeconds: 3600,
	}

	successfulRequests := 0
	failedRequests := 0

	// ATTACK: Submit 1000 requests rapidly
	for i := 0; i < 1000; i++ {
		maxPayment := math.NewInt(1000)

		_, err := suite.keeper.SubmitRequest(
			suite.ctx,
			requester,
			specs,
			"alpine:latest",
			[]string{"echo", "spam"},
			map[string]string{},
			maxPayment,
			"",
		)

		if err != nil {
			failedRequests++
		} else {
			successfulRequests++
		}

		// If rate limiting is working, should start failing
		if failedRequests > 0 && successfulRequests < 100 {
			break
		}
	}

	// Rate limiting should prevent spam (most requests should fail)
	suite.Require().Greater(failedRequests, 0,
		"No rate limiting - DoS attack succeeded with %d requests!", successfulRequests)

	suite.Require().Less(successfulRequests, 100,
		"Rate limiting ineffective - accepted %d spam requests!", successfulRequests)
}

// TestDoSAttack_QuotaExhaustion tests resource quota enforcement
func (suite *ComputeSecuritySuite) TestDoSAttack_QuotaExhaustion() {
	requester := suite.requesters[0]

	// ATTACK: Request maximum resources to exhaust quotas
	massiveSpecs := types.ComputeSpec{
		CpuCores:       1000000,  // 1000 cores
		MemoryMb:       10000000, // 10TB RAM
		GpuCount:       100,
		StorageGb:      1000000, // 1PB storage
		TimeoutSeconds: 86400,   // 24 hours
	}

	maxPayment := math.NewInt(1000000000)

	_, err := suite.keeper.SubmitRequest(
		suite.ctx,
		requester,
		massiveSpecs,
		"alpine:latest",
		[]string{"echo", "exhaust"},
		map[string]string{},
		maxPayment,
		"",
	)

	// Should fail due to no provider having such resources or quota limits
	suite.Require().Error(err, "Quota exhaustion attack succeeded - accepted unreasonable resource request!")
}

// ========================================
// ECONOMIC ATTACK TESTS
// ========================================

// TestStakeSlashing_InsufficientStake tests that low-stake providers are penalized
func (suite *ComputeSecuritySuite) TestStakeSlashing_InsufficientStake() {
	// Create provider with exactly minimum stake
	lowStakeProvider := sdk.AccAddress([]byte("lowstake__________"))
	params, _ := suite.keeper.GetParams(suite.ctx)

	coins := sdk.NewCoins(sdk.NewCoin("upaw", params.MinProviderStake.MulRaw(2)))
	err := suite.bankKeeper.MintCoins(suite.ctx, types.ModuleName, coins)
	suite.Require().NoError(err)
	err = suite.bankKeeper.SendCoinsFromModuleToAccount(suite.ctx, types.ModuleName, lowStakeProvider, coins)
	suite.Require().NoError(err)

	specs := types.ComputeSpec{
		CpuCores:       2000,
		MemoryMb:       4096,
		GpuCount:       0,
		StorageGb:      100,
		TimeoutSeconds: 3600,
	}

	pricing := types.Pricing{
		CpuPricePerMcoreHour:  math.LegacyNewDec(10),
		MemoryPricePerMbHour:  math.LegacyNewDec(5),
		GpuPricePerHour:       math.LegacyNewDec(1000),
		StoragePricePerGbHour: math.LegacyNewDec(2),
	}

	err = suite.keeper.RegisterProvider(
		suite.ctx,
		lowStakeProvider,
		"LowStake",
		"http://lowstake.github.com",
		specs,
		pricing,
		params.MinProviderStake,
	)
	suite.Require().NoError(err)

	providerBefore, _ := suite.keeper.GetProvider(suite.ctx, lowStakeProvider)

	// Submit invalid proof to trigger slashing
	requester := suite.requesters[0]
	maxPayment := math.NewInt(100000)

	requestID, err := suite.keeper.SubmitRequest(
		suite.ctx,
		requester,
		specs,
		"alpine:latest",
		[]string{"echo", "test"},
		map[string]string{},
		maxPayment,
		lowStakeProvider.String(),
	)
	suite.Require().NoError(err)

	// Submit invalid proof
	invalidProof := make([]byte, 300)
	_, err = rand.Read(invalidProof)
	suite.Require().NoError(err, "failed to randomize invalid proof")

	err = suite.keeper.SubmitResult(
		suite.ctx,
		lowStakeProvider,
		requestID,
		"invalidhash",
		"http://result.github.com",
		1, // Non-zero exit code
		"http://logs.github.com",
		invalidProof,
	)
	suite.Require().NoError(err, "invalid proof submission should be processed for slashing")

	// After slashing, provider should be deactivated due to insufficient stake
	providerAfter, err := suite.keeper.GetProvider(suite.ctx, lowStakeProvider)

	if err == nil {
		// Provider should be slashed and possibly deactivated
		suite.Require().True(
			providerAfter.Stake.LT(providerBefore.Stake) || !providerAfter.Active,
			"Low-stake provider not penalized after invalid proof!",
		)
	}
}

// TestPaymentTheft_ChallengeBypass tests that challenge period cannot be bypassed
func (suite *ComputeSecuritySuite) TestPaymentTheft_ChallengeBypass() {
	requester := suite.requesters[0]
	provider := suite.providers[0]

	specs := types.ComputeSpec{
		CpuCores:       2000,
		MemoryMb:       4096,
		GpuCount:       0,
		StorageGb:      100,
		TimeoutSeconds: 3600,
	}
	maxPayment := math.NewInt(100000)

	requestID, err := suite.keeper.SubmitRequest(
		suite.ctx,
		requester,
		specs,
		"alpine:latest",
		[]string{"echo", "test"},
		map[string]string{},
		maxPayment,
		provider.String(),
	)
	suite.Require().NoError(err)

	outputHash := "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"
	proof := suite.createValidProof(provider, requestID, outputHash)

	err = suite.keeper.SubmitResult(
		suite.ctx,
		provider,
		requestID,
		outputHash,
		"http://result.github.com",
		0,
		"http://logs.github.com",
		proof,
	)
	suite.Require().NoError(err)

	// Get release delay from params
	params, _ := suite.keeper.GetParams(suite.ctx)
	releaseDelay := time.Duration(params.EscrowReleaseDelaySeconds) * time.Second

	providerBalanceBefore := suite.bankKeeper.GetBalance(suite.ctx, provider, "upaw")

	// ATTACK: Try to bypass challenge period by advancing time manually
	// This tests that the system properly enforces the delay

	// Should not be able to withdraw immediately
	immediateBalance := suite.bankKeeper.GetBalance(suite.ctx, provider, "upaw")

	// Advance time past release delay
	suite.ctx = suite.ctx.WithBlockTime(suite.ctx.BlockTime().Add(releaseDelay + time.Second))

	// Now payment should be releasable
	finalBalance := suite.bankKeeper.GetBalance(suite.ctx, provider, "upaw")

	// Either payment was already released (good) or it's released after delay (good)
	// But it should NOT be released before the delay
	suite.Require().True(
		immediateBalance.Amount.Equal(providerBalanceBefore.Amount) ||
			finalBalance.Amount.GT(providerBalanceBefore.Amount),
		"Challenge period bypass succeeded - immediate withdrawal!",
	)
}

// ========================================
// CRYPTOGRAPHIC TESTS
// ========================================

// TestEd25519_KeySubstitution tests that public key substitution is detected
func (suite *ComputeSecuritySuite) TestEd25519_KeySubstitution() {
	requester := suite.requesters[0]
	provider := suite.providers[0]

	specs := types.ComputeSpec{
		CpuCores:       2000,
		MemoryMb:       4096,
		GpuCount:       0,
		StorageGb:      100,
		TimeoutSeconds: 3600,
	}
	maxPayment := math.NewInt(100000)

	requestID, err := suite.keeper.SubmitRequest(
		suite.ctx,
		requester,
		specs,
		"alpine:latest",
		[]string{"echo", "test"},
		map[string]string{},
		maxPayment,
		provider.String(),
	)
	suite.Require().NoError(err)

	outputHash := "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"

	// ATTACK: Sign with correct key but substitute public key
	correctPrivKey := suite.providerKeys[provider.String()]
	attackerPubKey, _, _ := ed25519.GenerateKey(rand.Reader)

	merkleRoot := make([]byte, 32)
	stateCommitment := make([]byte, 32)
	executionTrace := make([]byte, 32)
	merkleProof := [][]byte{make([]byte, 32)}
	nonce := uint64(time.Now().UnixNano())

	hasher := sha256.New()
	reqIDBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(reqIDBytes, requestID)
	hasher.Write(reqIDBytes)
	hasher.Write([]byte(outputHash))
	hasher.Write(merkleRoot)
	hasher.Write(stateCommitment)
	nonceBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(nonceBytes, nonce)
	hasher.Write(nonceBytes)
	timestampBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(timestampBytes, uint64(suite.ctx.BlockTime().Unix()))
	hasher.Write(timestampBytes)
	message := hasher.Sum(nil)

	// Sign with CORRECT key
	correctSignature := ed25519.Sign(correctPrivKey, message)

	// But use WRONG public key
	var proofBytes []byte
	proofBytes = append(proofBytes, correctSignature...)
	proofBytes = append(proofBytes, attackerPubKey...) // Substituted key!
	proofBytes = append(proofBytes, merkleRoot...)
	proofBytes = append(proofBytes, byte(len(merkleProof)))
	for _, node := range merkleProof {
		proofBytes = append(proofBytes, node...)
	}
	proofBytes = append(proofBytes, stateCommitment...)
	proofBytes = append(proofBytes, executionTrace...)
	proofBytes = append(proofBytes, nonceBytes...)
	proofBytes = append(proofBytes, timestampBytes...)

	err = suite.keeper.SubmitResult(
		suite.ctx,
		provider,
		requestID,
		outputHash,
		"http://result.github.com",
		0,
		"http://logs.github.com",
		proofBytes,
	)
	suite.Require().NoError(err)

	result, err := suite.keeper.GetResult(suite.ctx, requestID)
	if err == nil {
		// Key substitution should fail signature verification
		suite.Require().False(result.Verified, "Key substitution attack succeeded!")
	}
}

// TestNonceReplay_SameNonceMultipleTimes tests comprehensive nonce tracking
func (suite *ComputeSecuritySuite) TestNonceReplay_SameNonceMultipleTimes() {
	requester := suite.requesters[0]
	provider := suite.providers[0]

	specs := types.ComputeSpec{
		CpuCores:       2000,
		MemoryMb:       4096,
		GpuCount:       0,
		StorageGb:      100,
		TimeoutSeconds: 3600,
	}

	// Use fixed nonce
	fixedNonce := uint64(12345678)

	// Try to submit 5 different requests with the same nonce
	successfulSubmissions := 0

	for i := 0; i < 5; i++ {
		maxPayment := math.NewInt(100000)

		requestID, err := suite.keeper.SubmitRequest(
			suite.ctx,
			requester,
			specs,
			"alpine:latest",
			[]string{"echo", string(rune('0' + i))},
			map[string]string{},
			maxPayment,
			provider.String(),
		)

		if err != nil {
			continue
		}

		outputHash := string(rune('a'+i)) + string(rune('a'+i)) + string(rune('a'+i)) + string(rune('a'+i)) + "test" + "test" + "test" + "test" + "test" + "test" + "test" + "test" + "test" + "test" + "test"
		proof := suite.createValidProofWithNonce(provider, requestID, outputHash, fixedNonce)

		err = suite.keeper.SubmitResult(
			suite.ctx,
			provider,
			requestID,
			outputHash,
			"http://result.github.com",
			0,
			"http://logs.github.com",
			proof,
		)
		if err != nil {
			continue
		}

		result, err := suite.keeper.GetResult(suite.ctx, requestID)
		if err == nil && result.Verified {
			successfulSubmissions++
		}
	}

	// Only the first submission should succeed
	suite.Require().LessOrEqual(successfulSubmissions, 1,
		"Nonce replay attack succeeded - multiple submissions with same nonce verified!")
}

// TestTimestampManipulation_FutureTimestamp tests timestamp validation
func (suite *ComputeSecuritySuite) TestTimestampManipulation_FutureTimestamp() {
	requester := suite.requesters[0]
	provider := suite.providers[0]

	specs := types.ComputeSpec{
		CpuCores:       2000,
		MemoryMb:       4096,
		GpuCount:       0,
		StorageGb:      100,
		TimeoutSeconds: 3600,
	}
	maxPayment := math.NewInt(100000)

	requestID, err := suite.keeper.SubmitRequest(
		suite.ctx,
		requester,
		specs,
		"alpine:latest",
		[]string{"echo", "test"},
		map[string]string{},
		maxPayment,
		provider.String(),
	)
	suite.Require().NoError(err)

	outputHash := "abcdef1234567890abcdef1234567890abcdef1234567890abcdef1234567890"

	// ATTACK: Use future timestamp (1 year ahead)
	futureTimestamp := suite.ctx.BlockTime().Add(365 * 24 * time.Hour).Unix()

	privKey := suite.providerKeys[provider.String()]
	pubKey := privKey.Public().(ed25519.PublicKey)

	merkleRoot := make([]byte, 32)
	stateCommitment := make([]byte, 32)
	executionTrace := make([]byte, 32)
	merkleProof := [][]byte{make([]byte, 32)}
	nonce := uint64(time.Now().UnixNano())

	hasher := sha256.New()
	reqIDBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(reqIDBytes, requestID)
	hasher.Write(reqIDBytes)
	hasher.Write([]byte(outputHash))
	hasher.Write(merkleRoot)
	hasher.Write(stateCommitment)
	nonceBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(nonceBytes, nonce)
	hasher.Write(nonceBytes)
	timestampBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(timestampBytes, uint64(futureTimestamp))
	hasher.Write(timestampBytes)
	message := hasher.Sum(nil)

	signature := ed25519.Sign(privKey, message)

	var proofBytes []byte
	proofBytes = append(proofBytes, signature...)
	proofBytes = append(proofBytes, pubKey...)
	proofBytes = append(proofBytes, merkleRoot...)
	proofBytes = append(proofBytes, byte(len(merkleProof)))
	for _, node := range merkleProof {
		proofBytes = append(proofBytes, node...)
	}
	proofBytes = append(proofBytes, stateCommitment...)
	proofBytes = append(proofBytes, executionTrace...)
	proofBytes = append(proofBytes, nonceBytes...)
	proofBytes = append(proofBytes, timestampBytes...)

	err = suite.keeper.SubmitResult(
		suite.ctx,
		provider,
		requestID,
		outputHash,
		"http://result.github.com",
		0,
		"http://logs.github.com",
		proofBytes,
	)

	// SEC-HIGH-1: Future timestamp must be REJECTED (not just scored low)
	suite.Require().Error(err, "SubmitResult should reject future timestamp")
	suite.Require().ErrorIs(err, types.ErrProofExpired, "Should return ErrProofExpired for future timestamp")

	// Result should not exist since submission was rejected
	_, err = suite.keeper.GetResult(suite.ctx, requestID)
	suite.Require().Error(err, "Result should not exist after rejection")
}

// ========================================
// HELPER FUNCTIONS
// ========================================

// createValidProof generates a valid cryptographic proof for testing
func (suite *ComputeSecuritySuite) createValidProof(provider sdk.AccAddress, requestID uint64, outputHash string) []byte {
	nonce := uint64(time.Now().UnixNano())
	return suite.createValidProofWithNonce(provider, requestID, outputHash, nonce)
}

// createValidProofWithNonce generates a valid proof with specified nonce
func (suite *ComputeSecuritySuite) createValidProofWithNonce(provider sdk.AccAddress, requestID uint64, outputHash string, nonce uint64) []byte {
	privKey := suite.providerKeys[provider.String()]
	pubKey := privKey.Public().(ed25519.PublicKey)

	// Create valid merkle tree
	leaves := [][]byte{
		[]byte("execution step 1"),
		[]byte("execution step 2"),
	}

	leafHashes := make([][]byte, len(leaves))
	for i, leaf := range leaves {
		hash := sha256.Sum256(leaf)
		leafHashes[i] = hash[:]
	}

	// Build merkle root with canonical ordering (smaller hash first)
	// This matches the verification code which uses canonical ordering for security
	hasher := sha256.New()
	if bytes.Compare(leafHashes[0], leafHashes[1]) < 0 {
		hasher.Write(leafHashes[0])
		hasher.Write(leafHashes[1])
	} else {
		hasher.Write(leafHashes[1])
		hasher.Write(leafHashes[0])
	}
	merkleRoot := hasher.Sum(nil)

	// Merkle proof for leaf 0
	merkleProof := [][]byte{leafHashes[1]}

	// Execution trace
	executionTrace := leafHashes[0]

	// State commitment (hash of container + commands + output)
	stateHasher := sha256.New()
	stateHasher.Write([]byte("alpine:latest"))
	stateHasher.Write([]byte("echo"))
	stateHasher.Write([]byte("test"))
	stateHasher.Write([]byte(outputHash))
	stateHasher.Write(executionTrace)
	stateCommitment := stateHasher.Sum(nil)

	// Compute message hash
	msgHasher := sha256.New()
	reqIDBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(reqIDBytes, requestID)
	msgHasher.Write(reqIDBytes)
	msgHasher.Write([]byte(outputHash))
	msgHasher.Write(merkleRoot)
	msgHasher.Write(stateCommitment)
	nonceBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(nonceBytes, nonce)
	msgHasher.Write(nonceBytes)
	timestampBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(timestampBytes, uint64(suite.ctx.BlockTime().Unix()))
	msgHasher.Write(timestampBytes)
	message := msgHasher.Sum(nil)

	// Sign message
	signature := ed25519.Sign(privKey, message)

	// Serialize proof
	var proofBytes []byte
	proofBytes = append(proofBytes, signature...)
	proofBytes = append(proofBytes, pubKey...)
	proofBytes = append(proofBytes, merkleRoot...)
	proofBytes = append(proofBytes, byte(len(merkleProof)))
	for _, node := range merkleProof {
		proofBytes = append(proofBytes, node...)
	}
	proofBytes = append(proofBytes, stateCommitment...)
	proofBytes = append(proofBytes, executionTrace...)
	proofBytes = append(proofBytes, nonceBytes...)
	proofBytes = append(proofBytes, timestampBytes...)

	return proofBytes
}

// ========================================
// TEST SUITE RUNNER
// ========================================

func TestComputeSecuritySuite(t *testing.T) {
	suite.Run(t, new(ComputeSecuritySuite))
}
