package cometmock

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	dbm "github.com/cometbft/cometbft-db"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/crypto/ed25519"
	"github.com/cometbft/cometbft/libs/log"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	tmtypes "github.com/cometbft/cometbft/types"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/app"
)

// CometMockApp wraps the PAW app with CometMock for testing
type CometMockApp struct {
	*app.App
	ctx          sdk.Context
	blockHeight  int64
	blockTime    time.Time
	validators   []*tmtypes.Validator
	validatorSet *tmtypes.ValidatorSet
	proposer     *tmtypes.Validator
}

// CometMockConfig contains configuration for CometMock
type CometMockConfig struct {
	NumValidators    int
	BlockTime        time.Duration
	ChainID          string
	InitialHeight    int64
	AccountAddresses []sdk.AccAddress
	GenesisAccounts  []app.GenesisAccount
}

// DefaultCometMockConfig returns default configuration for CometMock
func DefaultCometMockConfig() CometMockConfig {
	return CometMockConfig{
		NumValidators: 4,
		BlockTime:     5 * time.Second,
		ChainID:       "paw-test-1",
		InitialHeight: 1,
	}
}

// SetupCometMock initializes a new CometMock instance for testing
func SetupCometMock(t *testing.T, config CometMockConfig) *CometMockApp {
	t.Helper()

	db := dbm.NewMemDB()
	encCfg := app.MakeEncodingConfig()

	pawApp := app.NewApp(
		log.NewNopLogger(),
		db,
		nil,
		true,
		map[int64]bool{},
		app.DefaultNodeHome,
		0,
		encCfg,
		app.GetEnabledProposals(),
		baseapp.SetChainID(config.ChainID),
	)

	// Create validator set
	validators := createValidators(config.NumValidators)
	validatorSet := tmtypes.NewValidatorSet(validators)
	proposer := validators[0]

	// Initialize the chain
	genesisState := app.NewDefaultGenesisState(encCfg.Codec)
	stateBytes, err := encCfg.Codec.MarshalJSON(genesisState)
	require.NoError(t, err)

	// Initialize the blockchain
	_, err = pawApp.InitChain(
		&abci.RequestInitChain{
			Time:            time.Now(),
			ChainId:         config.ChainID,
			InitialHeight:   config.InitialHeight,
			ConsensusParams: DefaultConsensusParams(),
			Validators:      []abci.ValidatorUpdate{},
			AppStateBytes:   stateBytes,
		},
	)
	require.NoError(t, err)

	// Create initial context
	header := tmproto.Header{
		ChainID:         config.ChainID,
		Height:          config.InitialHeight,
		Time:            time.Now(),
		ProposerAddress: proposer.Address,
	}

	ctx := pawApp.BaseApp.NewContext(false, header)

	mockApp := &CometMockApp{
		App:          pawApp,
		ctx:          ctx,
		blockHeight:  config.InitialHeight,
		blockTime:    time.Now(),
		validators:   validators,
		validatorSet: validatorSet,
		proposer:     proposer,
	}

	return mockApp
}

// BeginBlock simulates beginning a new block
func (m *CometMockApp) BeginBlock(txs [][]byte) {
	m.blockHeight++
	m.blockTime = m.blockTime.Add(5 * time.Second)

	header := tmproto.Header{
		ChainID:         m.ctx.ChainID(),
		Height:          m.blockHeight,
		Time:            m.blockTime,
		ProposerAddress: m.proposer.Address,
		LastBlockId: tmproto.BlockID{
			Hash: []byte("mock_block_hash"),
		},
	}

	_, err := m.App.BeginBlocker(m.ctx.WithBlockHeader(header))
	if err != nil {
		panic(err)
	}

	m.ctx = m.App.BaseApp.NewContext(false, header)
}

// EndBlock simulates ending the current block
func (m *CometMockApp) EndBlock() {
	_, err := m.App.EndBlocker(m.ctx)
	if err != nil {
		panic(err)
	}

	_, err = m.App.Commit()
	if err != nil {
		panic(err)
	}
}

// DeliverTx simulates delivering a transaction
func (m *CometMockApp) DeliverTx(tx []byte) (*abci.ExecTxResult, error) {
	req := &abci.RequestDeliverTx{Tx: tx}
	res, err := m.App.BaseApp.DeliverTx(req)
	return res, err
}

// Context returns the current SDK context
func (m *CometMockApp) Context() sdk.Context {
	return m.ctx
}

// Height returns the current block height
func (m *CometMockApp) Height() int64 {
	return m.blockHeight
}

// Time returns the current block time
func (m *CometMockApp) Time() time.Time {
	return m.blockTime
}

// NextBlock advances to the next block with given transactions
func (m *CometMockApp) NextBlock(txs ...[]byte) {
	m.BeginBlock(txs)

	for _, tx := range txs {
		_, err := m.DeliverTx(tx)
		if err != nil {
			// Log error but continue
			m.ctx.Logger().Error("transaction delivery failed", "error", err)
		}
	}

	m.EndBlock()
}

// NextBlocks advances multiple blocks
func (m *CometMockApp) NextBlocks(n int) {
	for i := 0; i < n; i++ {
		m.NextBlock()
	}
}

// createValidators creates a set of mock validators
func createValidators(num int) []*tmtypes.Validator {
	validators := make([]*tmtypes.Validator, num)
	for i := 0; i < num; i++ {
		privKey := ed25519.GenPrivKey()
		pubKey := privKey.PubKey()
		validators[i] = tmtypes.NewValidator(pubKey, 100)
	}
	return validators
}

// DefaultConsensusParams returns default consensus parameters for testing
func DefaultConsensusParams() *tmproto.ConsensusParams {
	return &tmproto.ConsensusParams{
		Block: &tmproto.BlockParams{
			MaxBytes: 200000,
			MaxGas:   2000000,
		},
		Evidence: &tmproto.EvidenceParams{
			MaxAgeNumBlocks: 302400,
			MaxAgeDuration:  504 * time.Hour,
			MaxBytes:        10000,
		},
		Validator: &tmproto.ValidatorParams{
			PubKeyTypes: []string{
				tmtypes.ABCIPubKeyTypeEd25519,
			},
		},
		Version: &tmproto.VersionParams{
			App: 1,
		},
	}
}
