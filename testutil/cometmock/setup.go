package cometmock

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"cosmossdk.io/log"
	dbm "github.com/cosmos/cosmos-db"
	abci "github.com/cometbft/cometbft/abci/types"
	"github.com/cometbft/cometbft/crypto/ed25519"
	tmproto "github.com/cometbft/cometbft/proto/tendermint/types"
	tmtypes "github.com/cometbft/cometbft/types"

	"github.com/cosmos/cosmos-sdk/baseapp"
	sdk "github.com/cosmos/cosmos-sdk/types"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"

	"github.com/paw-chain/paw/app"
)

// CometMockApp wraps the PAW app with CometMock for testing
type CometMockApp struct {
	*app.PAWApp
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

	pawApp := app.NewPAWApp(
		log.NewNopLogger(),
		db,
		nil,
		true,
		simtestutil.EmptyAppOptions{},
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
	ctx := pawApp.BaseApp.NewContext(false)

	mockApp := &CometMockApp{
		PAWApp:       pawApp,
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

	_, err := m.PAWApp.BeginBlocker(m.ctx.WithBlockHeader(header))
	if err != nil {
		panic(err)
	}

	m.ctx = m.PAWApp.BaseApp.NewContext(false)
}

// EndBlock simulates ending the current block
func (m *CometMockApp) EndBlock() {
	_, err := m.PAWApp.EndBlocker(m.ctx)
	if err != nil {
		panic(err)
	}

	_, err = m.PAWApp.Commit()
	if err != nil {
		panic(err)
	}
}

// DeliverTx simulates delivering a transaction
func (m *CometMockApp) DeliverTx(tx []byte) (*abci.ExecTxResult, error) {
	// Note: RequestDeliverTx is deprecated in newer versions
	// For now, we'll use a simplified approach
	res := m.PAWApp.BaseApp.DeliverTx(abci.RequestDeliverTx{Tx: tx})
	return &abci.ExecTxResult{
		Code: res.Code,
		Data: res.Data,
		Log:  res.Log,
	}, nil
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
