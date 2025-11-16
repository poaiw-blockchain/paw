package keeper

import (
	"testing"

	"cosmossdk.io/log"
	dbm "github.com/cosmos/cosmos-db"
	"github.com/cosmos/cosmos-sdk/baseapp"
	simtestutil "github.com/cosmos/cosmos-sdk/testutil/sims"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/app"
)

// SetupTestApp initializes a test application with all modules
func SetupTestApp(t *testing.T) (*app.PAWApp, sdk.Context) {
	db := dbm.NewMemDB()
	testApp := app.NewPAWApp(
		log.NewNopLogger(),
		db,
		nil,
		true,
		simtestutil.EmptyAppOptions{},
		baseapp.SetChainID("paw-test-1"),
	)

	ctx := testApp.BaseApp.NewContext(false)

	return testApp, ctx
}
