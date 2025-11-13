package integration

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	dbm "github.com/cometbft/cometbft-db"
	"github.com/cometbft/cometbft/libs/log"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"

	"cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/baseapp"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	cryptotypes "github.com/cosmos/cosmos-sdk/crypto/types"
	"github.com/cosmos/cosmos-sdk/server"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	abci "github.com/cometbft/cometbft/abci/types"

	"github.com/paw-chain/paw/app"
)

// Network represents a multi-node test network
type Network struct {
	T          *testing.T
	Validators []*Validator
	Config     NetworkConfig
}

// NetworkConfig holds the configuration for test networks
type NetworkConfig struct {
	NumValidators int
	NumFullNodes  int
	ChainID       string
	BondDenom     string
	MinGasPrices  string
	AccountTokens math.Int
	StakingTokens math.Int
	BondedTokens  math.Int
	TimeoutCommit time.Duration
}

// Validator represents a network validator
type Validator struct {
	Index       int
	App         *app.PAWApp
	Ctx         sdk.Context
	ClientCtx   interface{} // client.Context
	Address     sdk.AccAddress
	ValAddress  sdk.ValAddress
	PubKey      cryptotypes.PubKey
	Moniker     string
	RPCAddress  string
	P2PAddress  string
	APIAddress  string
	GRPCAddress string
}

// DefaultNetworkConfig returns a default network configuration
func DefaultNetworkConfig() NetworkConfig {
	return NetworkConfig{
		NumValidators: 4,
		NumFullNodes:  0,
		ChainID:       "paw-test-1",
		BondDenom:     "upaw",
		MinGasPrices:  "0upaw",
		AccountTokens: sdk.TokensFromConsensusPower(1000, sdk.DefaultPowerReduction),
		StakingTokens: sdk.TokensFromConsensusPower(500, sdk.DefaultPowerReduction),
		BondedTokens:  sdk.TokensFromConsensusPower(100, sdk.DefaultPowerReduction),
		TimeoutCommit: 500 * time.Millisecond,
	}
}

// New creates a new multi-node test network
func New(t *testing.T, cfg NetworkConfig) *Network {
	t.Helper()

	network := &Network{
		T:          t,
		Validators: make([]*Validator, cfg.NumValidators),
		Config:     cfg,
	}

	// Create validators
	for i := 0; i < cfg.NumValidators; i++ {
		network.Validators[i] = network.createValidator(i)
	}

	return network
}

// createValidator creates a single validator node
func (n *Network) createValidator(index int) *Validator {
	db := dbm.NewMemDB()

	logger := log.NewNopLogger()
	if testing.Verbose() {
		// Create a writer that writes to the test log
		writer := &testWriter{t: n.T}
		logger = log.NewTMLogger(log.NewSyncWriter(writer))
	}

	// Create app options for testing
	appOpts := &mockAppOptions{}

	pawApp := app.NewPAWApp(
		logger,
		db,
		nil,
		true,
		appOpts,
		baseapp.SetChainID(n.Config.ChainID),
	)

	// Create validator account
	privKey := secp256k1.GenPrivKey()
	pubKey := privKey.PubKey()
	address := sdk.AccAddress(pubKey.Address())
	valAddress := sdk.ValAddress(address)

	ctx := pawApp.BaseApp.NewContext(false, tmproto.Header{
		ChainID: n.Config.ChainID,
		Height:  1,
		Time:    time.Now(),
	})

	validator := &Validator{
		Index:       index,
		App:         pawApp,
		Ctx:         ctx,
		Address:     address,
		ValAddress:  valAddress,
		PubKey:      pubKey,
		Moniker:     fmt.Sprintf("validator-%d", index),
		RPCAddress:  fmt.Sprintf("tcp://0.0.0.0:%d", 26657+index),
		P2PAddress:  fmt.Sprintf("tcp://0.0.0.0:%d", 26656+index),
		APIAddress:  fmt.Sprintf("tcp://0.0.0.0:%d", 1317+index),
		GRPCAddress: fmt.Sprintf("0.0.0.0:%d", 9090+index),
	}

	return validator
}

// InitChain initializes the network chain
func (n *Network) InitChain(t *testing.T) {
	t.Helper()

	// Create genesis state
	genesisState := n.createGenesisState(t)
	stateBytes, err := json.MarshalIndent(genesisState, "", "  ")
	require.NoError(t, err)

	// Initialize each validator
	for _, val := range n.Validators {
		_, err := val.App.InitChain(
			&abci.RequestInitChain{
				Time:          time.Now(),
				ChainId:       n.Config.ChainID,
				InitialHeight: 1,
				Validators:    []abci.ValidatorUpdate{},
				AppStateBytes: stateBytes,
			},
		)
		require.NoError(t, err)
		val.App.Commit()
	}
}

// createGenesisState creates the genesis state for the network
func (n *Network) createGenesisState(t *testing.T) map[string]json.RawMessage {
	t.Helper()

	encCfg := app.MakeEncodingConfig()
	genesisState := app.NewDefaultGenesisState(n.Config.ChainID)

	// Create accounts and balances
	accounts := make([]authtypes.GenesisAccount, 0, len(n.Validators))
	balances := make([]banktypes.Balance, 0, len(n.Validators))

	for _, val := range n.Validators {
		baseAcc := authtypes.NewBaseAccount(val.Address, val.PubKey, uint64(val.Index), 0)
		accounts = append(accounts, baseAcc)

		balances = append(balances, banktypes.Balance{
			Address: val.Address.String(),
			Coins:   sdk.NewCoins(sdk.NewCoin(n.Config.BondDenom, n.Config.AccountTokens)),
		})
	}

	// Update auth genesis
	authGenesis := authtypes.NewGenesisState(authtypes.DefaultParams(), accounts)
	genesisState[authtypes.ModuleName] = encCfg.Codec.MustMarshalJSON(authGenesis)

	// Update bank genesis
	totalSupply := sdk.NewCoins()
	for _, balance := range balances {
		totalSupply = totalSupply.Add(balance.Coins...)
	}

	bankGenesis := banktypes.NewGenesisState(
		banktypes.DefaultParams(),
		balances,
		totalSupply,
		[]banktypes.Metadata{},
		[]banktypes.SendEnabled{},
	)
	genesisState[banktypes.ModuleName] = encCfg.Codec.MustMarshalJSON(bankGenesis)

	// Create validators
	validators := make([]stakingtypes.Validator, 0, len(n.Validators))
	delegations := make([]stakingtypes.Delegation, 0, len(n.Validators))

	for _, val := range n.Validators {
		// Convert PubKey directly to Any
		pkAny, err := codectypes.NewAnyWithValue(val.PubKey)
		require.NoError(t, err)

		validator := stakingtypes.Validator{
			OperatorAddress:   val.ValAddress.String(),
			ConsensusPubkey:   pkAny,
			Jailed:            false,
			Status:            stakingtypes.Bonded,
			Tokens:            n.Config.BondedTokens,
			DelegatorShares:   math.LegacyNewDecFromInt(n.Config.BondedTokens),
			Description:       stakingtypes.Description{Moniker: val.Moniker},
			UnbondingHeight:   int64(0),
			UnbondingTime:     time.Unix(0, 0).UTC(),
			Commission:        stakingtypes.NewCommission(math.LegacyZeroDec(), math.LegacyZeroDec(), math.LegacyZeroDec()),
			MinSelfDelegation: math.OneInt(),
		}

		validators = append(validators, validator)

		delegations = append(delegations, stakingtypes.Delegation{
			DelegatorAddress: val.Address.String(),
			ValidatorAddress: val.ValAddress.String(),
			Shares:           math.LegacyNewDecFromInt(n.Config.BondedTokens),
		})
	}

	stakingGenesis := stakingtypes.NewGenesisState(
		stakingtypes.DefaultParams(),
		validators,
		delegations,
	)
	genesisState[stakingtypes.ModuleName] = encCfg.Codec.MustMarshalJSON(stakingGenesis)

	return genesisState
}

// Cleanup performs cleanup tasks
func (n *Network) Cleanup() {
	for _, val := range n.Validators {
		val.App.Close()
	}
}

// WaitForHeight waits for the network to reach a specific height
func (n *Network) WaitForHeight(height int64) error {
	return n.WaitForHeightWithTimeout(height, 10*time.Second)
}

// WaitForHeightWithTimeout waits for height with timeout
func (n *Network) WaitForHeightWithTimeout(height int64, timeout time.Duration) error {
	ticker := time.NewTicker(n.Config.TimeoutCommit)
	defer ticker.Stop()

	timeoutTimer := time.NewTimer(timeout)
	defer timeoutTimer.Stop()

	for {
		select {
		case <-timeoutTimer.C:
			return fmt.Errorf("timeout waiting for height %d", height)
		case <-ticker.C:
			if n.Validators[0].Ctx.BlockHeight() >= height {
				return nil
			}
		}
	}
}

// LatestHeight returns the latest block height
func (n *Network) LatestHeight() int64 {
	return n.Validators[0].Ctx.BlockHeight()
}

// testWriter wraps testing.T to implement io.Writer
type testWriter struct {
	t *testing.T
}

func (tw *testWriter) Write(p []byte) (n int, err error) {
	tw.t.Log(string(p))
	return len(p), nil
}

// mockAppOptions implements server.AppOptions for testing
type mockAppOptions struct{}

func (m *mockAppOptions) Get(key string) interface{} {
	return nil
}
