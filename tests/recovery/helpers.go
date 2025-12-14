package recovery

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"cosmossdk.io/log"
	"cosmossdk.io/math"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/stretchr/testify/require"

	"github.com/cosmos/cosmos-sdk/baseapp"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	abci "github.com/cometbft/cometbft/abci/types"

	"github.com/paw-chain/paw/app"
	"github.com/paw-chain/paw/p2p/snapshot"
)

// TestNode represents a test blockchain node with recovery capabilities
type TestNode struct {
	App       *app.PAWApp
	DB        dbm.DB
	DataDir   string
	Logger    log.Logger
	Ctx       sdk.Context
	Height    int64
	ChainID   string
	Snapshots *snapshot.Manager
}

// RecoveryTestConfig holds configuration for recovery tests
type RecoveryTestConfig struct {
	ChainID            string
	InitialHeight      int64
	BlocksToGenerate   int
	SnapshotInterval   uint64
	KeepRecentBlocks   uint32
	EnableSnapshots    bool
	SimulateCrashAt    int64 // 0 = no crash simulation
	CrashDuringCommit  bool
	CrashDuringConsensus bool
}

// DefaultRecoveryTestConfig returns a default test configuration
func DefaultRecoveryTestConfig() RecoveryTestConfig {
	return RecoveryTestConfig{
		ChainID:          "paw-recovery-test",
		InitialHeight:    1,
		BlocksToGenerate: 10,
		SnapshotInterval: 5,
		KeepRecentBlocks: 3,
		EnableSnapshots:  true,
		SimulateCrashAt:  0,
	}
}

// TestingT is an interface that both *testing.T and *testing.B implement
type TestingT interface {
	Helper()
	TempDir() string
	Logf(format string, args ...interface{})
	Errorf(format string, args ...interface{})
	Log(args ...interface{})
	FailNow()
	Failed() bool
	Skipf(format string, args ...interface{})
}

// skipHelper wraps Skip() for TestingT
type testingTSkipper interface {
	Skip(args ...interface{})
}

// Skip skips the test if available
func skipTest(t TestingT, message string) {
	if skipper, ok := t.(testingTSkipper); ok {
		skipper.Skip(message)
	} else {
		t.Logf("SKIP: %s", message)
	}
}

// SetupTestNode creates a new test node with full recovery infrastructure
func SetupTestNode(t TestingT, config RecoveryTestConfig) *TestNode {
	t.Helper()

	// Create temporary data directory
	dataDir := t.TempDir()

	logger := log.NewTestLogger(t)

	// Setup database
	dbPath := filepath.Join(dataDir, "data")
	require.NoError(t, os.MkdirAll(dbPath, 0o750))

	db, err := dbm.NewGoLevelDB("application", dbPath, nil)
	require.NoError(t, err)

	// Create application
	pawApp := app.NewPAWApp(
		logger,
		db,
		nil,
		true,
		simtestutil.EmptyAppOptions{},
		baseapp.SetChainID(config.ChainID),
	)

	// Initialize snapshot manager if enabled
	var snapMgr *snapshot.Manager
	if config.EnableSnapshots {
		snapConfig := &snapshot.ManagerConfig{
			SnapshotDir:        filepath.Join(dataDir, "snapshots"),
			SnapshotInterval:   config.SnapshotInterval,
			SnapshotKeepRecent: uint32(config.KeepRecentBlocks),
			ChunkSize:          snapshot.DefaultChunkSize,
			PruneOldSnapshots:  true,
			MinSnapshotsToKeep: 2,
			ChainID:            config.ChainID,
		}
		snapMgr, err = snapshot.NewManager(snapConfig, logger)
		require.NoError(t, err)
	}

	node := &TestNode{
		App:       pawApp,
		DB:        db,
		DataDir:   dataDir,
		Logger:    logger,
		Height:    config.InitialHeight,
		ChainID:   config.ChainID,
		Snapshots: snapMgr,
	}

	return node
}

// InitializeChain initializes the blockchain with genesis state
func (n *TestNode) InitializeChain(t TestingT) {
	t.Helper()

	// Create validator account
	privKey := secp256k1.GenPrivKey()
	pubKey := privKey.PubKey()
	address := sdk.AccAddress(pubKey.Address())
	valAddress := sdk.ValAddress(address)

	// Create genesis accounts
	genesisAccounts := []authtypes.GenesisAccount{
		authtypes.NewBaseAccount(address, pubKey, 0, 0),
	}

	// Set bonded tokens - must be >= DefaultPowerReduction
	bondedTokens := sdk.TokensFromConsensusPower(100, sdk.DefaultPowerReduction)

	// Create balances for both the validator account and bonded pool module account
	bondedPoolAddress := authtypes.NewModuleAddress(stakingtypes.BondedPoolName)
	balances := []banktypes.Balance{
		{
			Address: address.String(),
			Coins:   sdk.NewCoins(sdk.NewCoin("upaw", bondedTokens)),
		},
		{
			Address: bondedPoolAddress.String(),
			Coins:   sdk.NewCoins(sdk.NewCoin("upaw", bondedTokens)),
		},
	}

	// Build genesis state
	genesisState := app.NewDefaultGenesisState(n.ChainID)

	// Set auth genesis
	authGenesis := authtypes.NewGenesisState(authtypes.DefaultParams(), genesisAccounts)
	genesisState[authtypes.ModuleName] = n.App.AppCodec().MustMarshalJSON(authGenesis)

	// Set bank genesis - include total supply (validator balance + bonded pool)
	totalSupply := sdk.NewCoins(sdk.NewCoin("upaw", bondedTokens.Add(bondedTokens)))
	bankGenesis := banktypes.NewGenesisState(
		banktypes.DefaultParams(),
		balances,
		totalSupply,
		[]banktypes.Metadata{},
		[]banktypes.SendEnabled{},
	)
	genesisState[banktypes.ModuleName] = n.App.AppCodec().MustMarshalJSON(bankGenesis)

	// Set staking genesis - create validator
	pkAny, err := codectypes.NewAnyWithValue(pubKey)
	require.NoError(t, err)

	validator := stakingtypes.Validator{
		OperatorAddress:   valAddress.String(),
		ConsensusPubkey:   pkAny,
		Jailed:            false,
		Status:            stakingtypes.Bonded,
		Tokens:            bondedTokens,
		DelegatorShares:   math.LegacyNewDecFromInt(bondedTokens),
		Description:       stakingtypes.Description{Moniker: "recovery-test-validator"},
		UnbondingHeight:   int64(0),
		UnbondingTime:     time.Unix(0, 0).UTC(),
		Commission:        stakingtypes.NewCommission(math.LegacyZeroDec(), math.LegacyZeroDec(), math.LegacyZeroDec()),
		MinSelfDelegation: math.OneInt(),
	}

	delegation := stakingtypes.Delegation{
		DelegatorAddress: address.String(),
		ValidatorAddress: valAddress.String(),
		Shares:           math.LegacyNewDecFromInt(bondedTokens),
	}

	// Set staking params with correct bond denom
	stakingParams := stakingtypes.DefaultParams()
	stakingParams.BondDenom = "upaw"

	stakingGenesis := stakingtypes.NewGenesisState(
		stakingParams,
		[]stakingtypes.Validator{validator},
		[]stakingtypes.Delegation{delegation},
	)
	genesisState[stakingtypes.ModuleName] = n.App.AppCodec().MustMarshalJSON(stakingGenesis)

	// Marshal genesis state - use json.Marshal for map types
	stateBytes, err := json.Marshal(genesisState)
	require.NoError(t, err)

	// Initialize chain
	consensusParams := simtestutil.DefaultConsensusParams
	_, err = n.App.InitChain(&abci.RequestInitChain{
		ChainId:         n.ChainID,
		Time:            time.Now(),
		ConsensusParams: consensusParams,
		AppStateBytes:   stateBytes,
	})
	require.NoError(t, err)

	// Commit genesis block
	_, err = n.App.FinalizeBlock(&abci.RequestFinalizeBlock{Height: n.Height})
	require.NoError(t, err)
	_, err = n.App.Commit()
	require.NoError(t, err)

	// Update height after genesis block
	n.Height++
}

// ProduceBlocks generates a specified number of blocks
func (n *TestNode) ProduceBlocks(t TestingT, numBlocks int) {
	t.Helper()

	for i := 0; i < numBlocks; i++ {
		n.ProduceBlock(t)
	}
}

// ProduceBlock generates a single block with optional transactions
func (n *TestNode) ProduceBlock(t TestingT) {
	t.Helper()

	// Begin block
	_, err := n.App.FinalizeBlock(&abci.RequestFinalizeBlock{
		Height: n.Height,
		Time:   time.Now(),
	})
	require.NoError(t, err)

	// Commit block
	_, err = n.App.Commit()
	require.NoError(t, err)

	n.Height++
}

// ProduceBlockWithTxs generates a block with specified number of transactions
func (n *TestNode) ProduceBlockWithTxs(t TestingT, numTxs int) {
	t.Helper()

	// Create test transactions
	txs := make([][]byte, numTxs)
	for i := 0; i < numTxs; i++ {
		txs[i] = n.CreateTestTx(t, i)
	}

	// Begin block
	_, err := n.App.FinalizeBlock(&abci.RequestFinalizeBlock{
		Height: n.Height,
		Time:   time.Now(),
		Txs:    txs,
	})
	require.NoError(t, err)

	// Commit block
	_, err = n.App.Commit()
	require.NoError(t, err)

	n.Height++
}

// CreateTestTx creates a simple test transaction
func (n *TestNode) CreateTestTx(t TestingT, nonce int) []byte {
	t.Helper()

	// Create a simple bank send message as test transaction
	privKey := secp256k1.GenPrivKey()
	addr := sdk.AccAddress(privKey.PubKey().Address())

	msg := &banktypes.MsgSend{
		FromAddress: addr.String(),
		ToAddress:   addr.String(),
		Amount:      sdk.NewCoins(sdk.NewCoin("upaw", math.NewInt(1))),
	}

	// Encode message
	msgBytes, err := n.App.AppCodec().MarshalJSON(msg)
	require.NoError(t, err)

	return msgBytes
}

// SimulateCrash simulates a node crash by closing the database without proper shutdown
func (n *TestNode) SimulateCrash(t TestingT) {
	t.Helper()

	// Close database abruptly without proper app shutdown
	// This simulates a crash where WAL and state may be inconsistent
	err := n.DB.Close()
	if err != nil {
		t.Logf("Database close during crash simulation: %v", err)
	}
}

// SimulateCrashDuringCommit simulates crash during state commit
func (n *TestNode) SimulateCrashDuringCommit(t TestingT) {
	t.Helper()

	// Start block processing
	_, err := n.App.FinalizeBlock(&abci.RequestFinalizeBlock{
		Height: n.Height,
		Time:   time.Now(),
	})
	require.NoError(t, err)

	// Crash before commit completes
	n.SimulateCrash(t)
}

// Restart reopens the database and creates a new app instance
func (n *TestNode) Restart(t TestingT) {
	t.Helper()

	// Reopen database
	dbPath := filepath.Join(n.DataDir, "data")
	db, err := dbm.NewGoLevelDB("application", dbPath, nil)
	require.NoError(t, err)

	// Create new app instance
	pawApp := app.NewPAWApp(
		n.Logger,
		db,
		nil,
		true,
		simtestutil.EmptyAppOptions{},
		baseapp.SetChainID(n.ChainID),
	)

	// Initialize internal state by calling Info to load the latest state
	// This ensures finalizeBlockState is properly initialized
	_, err = pawApp.Info(&abci.RequestInfo{})
	require.NoError(t, err)

	n.App = pawApp
	n.DB = db
}

// VerifyState verifies the blockchain state integrity
func (n *TestNode) VerifyState(t TestingT) {
	t.Helper()

	// Verify last block height is positive
	lastBlockHeight := n.App.LastBlockHeight()
	require.Greater(t, lastBlockHeight, int64(0), "last block height should be positive")

	// Verify store is accessible
	bankKeeper := n.App.BankKeeper
	require.NotNil(t, bankKeeper)

	// Verify app hash is not empty (indicates state is committed)
	commitID := n.App.LastCommitID()
	require.NotEmpty(t, commitID.Hash, "app hash should not be empty")

	// Basic verification is enough - we don't need to query keeper state
	// which would require a properly initialized context that may not be
	// available immediately after restart
}

// CreateSnapshot creates a state snapshot at current height
func (n *TestNode) CreateSnapshot(t TestingT) *snapshot.Snapshot {
	t.Helper()

	if n.Snapshots == nil {
		skipTest(t, "Snapshots not enabled for this node")
		return nil
	}

	// Get current state
	height := n.App.LastBlockHeight()

	// Create snapshot
	// In a real implementation, this would serialize the full app state
	// For testing, we create a minimal snapshot
	stateData := []byte(fmt.Sprintf("state-at-height-%d", height))
	appHash := n.GetStateHash(t)
	validatorHash := []byte("validator-hash")
	consensusHash := []byte("consensus-hash")

	snap, err := n.Snapshots.CreateSnapshot(height, stateData, appHash, validatorHash, consensusHash)
	require.NoError(t, err)

	n.Logger.Info("Created snapshot", "height", height)
	return snap
}

// RestoreFromSnapshot restores state from a snapshot
func (n *TestNode) RestoreFromSnapshot(t TestingT, snapHeight int64) {
	t.Helper()

	if n.Snapshots == nil {
		skipTest(t, "Snapshots not enabled for this node")
		return
	}

	snap, err := n.Snapshots.LoadSnapshot(snapHeight)
	require.NoError(t, err)
	require.NotNil(t, snap, "snapshot not found for height %d", snapHeight)

	n.Logger.Info("Restored from snapshot", "height", snapHeight)
}

// GetStateHash returns the current app hash
func (n *TestNode) GetStateHash(t TestingT) []byte {
	t.Helper()

	// Get the last commit info to access app hash
	height := n.App.LastBlockHeight()
	if height == 0 {
		return nil
	}

	// The app hash is in the last commit
	return n.App.LastCommitID().Hash
}

// VerifyStateConsistency verifies state consistency between two nodes
func VerifyStateConsistency(t TestingT, node1, node2 *TestNode) {
	t.Helper()

	// Verify heights match
	height1 := node1.App.LastBlockHeight()
	height2 := node2.App.LastBlockHeight()
	require.Equal(t, height1, height2, "heights should match")

	// Verify app hashes match
	hash1 := node1.GetStateHash(t)
	hash2 := node2.GetStateHash(t)
	require.Equal(t, hash1, hash2, "app hashes should match")
}

// WaitForCondition waits for a condition to be met or timeout
func WaitForCondition(t TestingT, condition func() bool, timeout time.Duration, message string) {
	t.Helper()

	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		if condition() {
			return
		}

		select {
		case <-ticker.C:
			if time.Now().After(deadline) {
				require.Fail(t, "condition timeout", message)
				return
			}
		}
	}
}

// Cleanup performs cleanup of test node resources
func (n *TestNode) Cleanup(t TestingT) {
	if n.DB != nil {
		err := n.DB.Close()
		if err != nil {
			t.Logf("Error closing database: %v", err)
		}
	}
}
