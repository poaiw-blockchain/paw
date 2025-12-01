package network

import (
	"context"
	"fmt"
	"testing"

	"cosmossdk.io/math"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client/flags"
	"github.com/cosmos/cosmos-sdk/client/tx"
	servertypes "github.com/cosmos/cosmos-sdk/server/types"
	"github.com/cosmos/cosmos-sdk/testutil/network"
	sdk "github.com/cosmos/cosmos-sdk/types"
	moduletestutil "github.com/cosmos/cosmos-sdk/types/module/testutil"

	"github.com/paw-chain/paw/app"
)

// Re-export common test network types so test suites can rely on local package paths.
type (
	Config    = network.Config
	Network   = network.Network
	Validator = network.Validator
)

// WaitForNextBlock waits for one more block to be committed.
func WaitForNextBlock(n *Network, ctx context.Context) (int64, error) {
	if n == nil {
		return 0, fmt.Errorf("network is nil")
	}
	h, err := n.LatestHeight()
	if err != nil {
		return 0, err
	}
	return n.WaitForHeight(h + 1)
}

// BroadcastTx sends a transaction and waits for inclusion using the underlying network helpers.
func BroadcastTx(n *Network, ctx context.Context, msgs ...sdk.Msg) (*sdk.TxResponse, error) {
	if n == nil {
		return nil, fmt.Errorf("network is nil")
	}
	val := n.Validators[0]
	clientCtx := val.ClientCtx
	clientCtx = clientCtx.WithBroadcastMode(flags.BroadcastSync)

	factory := tx.Factory{}.
		WithChainID(n.Config.ChainID).
		WithTxConfig(clientCtx.TxConfig).
		WithKeybase(val.ClientCtx.Keyring).
		WithGasAdjustment(1.2).
		WithGasPrices(val.AppConfig.MinGasPrices)

	clientCtx = clientCtx.WithFromAddress(val.Address)
	clientCtx = clientCtx.WithFromName(val.Moniker)

	unsigned := clientCtx.TxConfig.NewTxBuilder()
	if err := unsigned.SetMsgs(msgs...); err != nil {
		return nil, err
	}

	if err := tx.Sign(ctx, factory, val.Moniker, unsigned, true); err != nil {
		return nil, err
	}

	bytes, err := clientCtx.TxConfig.TxEncoder()(unsigned.GetTx())
	if err != nil {
		return nil, err
	}

	return clientCtx.BroadcastTx(bytes)
}

// New spins up a Cosmos SDK in-process network using the PAW defaults.
func New(t *testing.T, dir string, cfg Config) (*Network, error) {
	return network.New(t, dir, cfg)
}

// DefaultConfig returns a network.Config wired to the PAW app with sane defaults.
func DefaultConfig() network.Config {
	app.SetConfig()

	const chainID = "paw-localnet"
	encCfg := app.MakeEncodingConfig()
	cfg := network.DefaultConfig(func() network.TestFixture {
		return network.TestFixture{
			AppConstructor: func(val network.ValidatorI) servertypes.Application {
				db := dbm.NewMemDB()
				return app.NewPAWApp(
					val.GetCtx().Logger,
					db,
					nil,
					true,
					val.GetCtx().Viper,
					baseapp.SetChainID(chainID),
				)
			},
			GenesisState: app.ModuleBasics.DefaultGenesis(encCfg.Codec),
			EncodingConfig: moduletestutil.TestEncodingConfig{
				InterfaceRegistry: encCfg.InterfaceRegistry,
				Codec:             encCfg.Codec,
				TxConfig:          encCfg.TxConfig,
				Amino:             encCfg.Amino,
			},
		}
	})

	// Override chain-specific defaults
	cfg.MinGasPrices = sdk.NewDecCoinFromDec(app.BondDenom, math.LegacyMustNewDecFromStr("0.001")).String() // 0.001upaw
	cfg.TimeoutCommit = 2_000_000_000                                                                       // 2s
	cfg.ChainID = chainID
	cfg.NumValidators = 2

	return cfg
}
