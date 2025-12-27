package ibctesting

import (
	"encoding/json"

	dbm "github.com/cosmos/cosmos-db"

	"cosmossdk.io/log"
	storetypes "cosmossdk.io/store/types"
	cmtproto "github.com/cometbft/cometbft/proto/tendermint/types"
	"github.com/cosmos/cosmos-sdk/baseapp"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"
	capabilitykeeper "github.com/cosmos/ibc-go/modules/capability/keeper"
	corekeeper "github.com/cosmos/ibc-go/v8/modules/core/keeper"
	ibctesting "github.com/cosmos/ibc-go/v8/testing"
	ibctestingtypes "github.com/cosmos/ibc-go/v8/testing/types"

	"github.com/paw-chain/paw/app"
	"github.com/paw-chain/paw/app/ibcutil"
	computetypes "github.com/paw-chain/paw/x/compute/types"
	dextypes "github.com/paw-chain/paw/x/dex/types"
	oracletypes "github.com/paw-chain/paw/x/oracle/types"
)

// Override the default testing app to use PAW's full application.
func init() {
	ibctesting.DefaultTestingAppInit = SetupTestingApp
}

// SetupTestingApp builds a PAW app and genesis state for ibc-go testing harness.
func SetupTestingApp() (ibctesting.TestingApp, map[string]json.RawMessage) {
	db := dbm.NewMemDB()
	app.SetConfig()

	pawApp := app.NewPAWApp(
		log.NewNopLogger(),
		db,
		nil,
		true,
		simtestutil.EmptyAppOptions{},
		baseapp.SetChainID("paw-ibc"),
	)

	ctx := pawApp.BaseApp.NewContext(true).WithBlockHeader(cmtproto.Header{})
	if err := pawApp.ComputeKeeper.BindPort(ctx); err != nil {
		panic(err)
	}
	if err := pawApp.DEXKeeper.BindPort(ctx); err != nil {
		panic(err)
	}
	if err := pawApp.OracleKeeper.BindPort(ctx); err != nil {
		panic(err)
	}
	if !pawApp.IBCKeeper.PortKeeper.IsBound(ctx, computetypes.PortID) {
		panic("compute port not bound")
	}
	if !pawApp.IBCKeeper.PortKeeper.IsBound(ctx, dextypes.PortID) {
		panic("dex port not bound")
	}
	if !pawApp.IBCKeeper.PortKeeper.IsBound(ctx, oracletypes.PortID) {
		panic("oracle port not bound")
	}

	return pawTestingApp{pawApp}, app.ModuleBasics.DefaultGenesis(pawApp.AppCodec())
}

// BindCustomPorts ensures custom module ports are bound in the current chain context.
// Useful for ibctesting paths that create channels before module InitGenesis runs.
func BindCustomPorts(chain *ibctesting.TestChain) {
	pawApp, ok := chain.App.(pawTestingApp)
	if !ok {
		panic("chain app is not pawTestingApp")
	}
	ctx := chain.GetContext()
	if err := pawApp.ComputeKeeper.BindPort(ctx); err != nil {
		panic(err)
	}
	if err := pawApp.DEXKeeper.BindPort(ctx); err != nil {
		panic(err)
	}
	if err := pawApp.OracleKeeper.BindPort(ctx); err != nil {
		panic(err)
	}
}

// BankBalance returns the balance for the given address using the PAW app's bank keeper.
func BankBalance(chain *ibctesting.TestChain, addr sdk.AccAddress, denom string) sdk.Coin {
	pawApp, ok := chain.App.(pawTestingApp)
	if !ok {
		panic("chain app is not pawTestingApp")
	}
	return pawApp.BankKeeper.GetBalance(chain.GetContext(), addr, denom)
}

// pawTestingApp satisfies ibctesting.TestingApp by forwarding to PAWApp.
type pawTestingApp struct{ *app.PAWApp }

func (w pawTestingApp) GetBaseApp() *baseapp.BaseApp                    { return w.BaseApp }
func (w pawTestingApp) GetStakingKeeper() ibctestingtypes.StakingKeeper { return w.StakingKeeper }
func (w pawTestingApp) GetIBCKeeper() *corekeeper.Keeper                { return w.IBCKeeper }
func (w pawTestingApp) GetScopedIBCKeeper() capabilitykeeper.ScopedKeeper {
	return w.ScopedIBCKeeper
}
func (w pawTestingApp) GetTxConfig() client.TxConfig      { return w.TxConfig() }
func (w pawTestingApp) AppCodec() codec.Codec             { return w.PAWApp.AppCodec() }
func (w pawTestingApp) LastCommitID() storetypes.CommitID { return w.BaseApp.LastCommitID() }
func (w pawTestingApp) LastBlockHeight() int64            { return w.BaseApp.LastBlockHeight() }

// GetPAWApp unwraps the underlying PAWApp from a testing chain.
func GetPAWApp(chain *ibctesting.TestChain) *app.PAWApp {
	if w, ok := chain.App.(pawTestingApp); ok {
		return w.PAWApp
	}
	panic("chain app is not pawTestingApp")
}

// AuthorizeModuleChannel registers a port/channel pair with the appropriate module keeper.
func AuthorizeModuleChannel(chain *ibctesting.TestChain, portID, channelID string) {
	pawApp := GetPAWApp(chain)
	ctx := chain.GetContext()

	switch portID {
	case computetypes.PortID:
		if err := ibcutil.AuthorizeChannel(ctx, pawApp.ComputeKeeper, portID, channelID); err != nil {
			panic(err)
		}
	case dextypes.PortID:
		if err := ibcutil.AuthorizeChannel(ctx, pawApp.DEXKeeper, portID, channelID); err != nil {
			panic(err)
		}
	case oracletypes.PortID:
		if err := ibcutil.AuthorizeChannel(ctx, pawApp.OracleKeeper, portID, channelID); err != nil {
			panic(err)
		}
	default:
		panic("unknown port for authorization: " + portID)
	}
}
