package simulation

import (
	"testing"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/codec"
	codectypes "github.com/cosmos/cosmos-sdk/codec/types"
	simtypes "github.com/cosmos/cosmos-sdk/types/simulation"
	"github.com/stretchr/testify/require"

	"github.com/paw-chain/paw/x/compute/keeper"
	"github.com/paw-chain/paw/x/compute/types"
)

// Smoke-test that WeightedOperations builds without panics and returns operations.
// Returned operations aren't executed here, so zero-value dependencies are acceptable.
func TestWeightedOperationsBuild(t *testing.T) {
	appParams := simtypes.AppParams{}
	interfaceRegistry := codectypes.NewInterfaceRegistry()
	types.RegisterInterfaces(interfaceRegistry)
	reg := codec.NewProtoCodec(interfaceRegistry)

	var (
		txGen client.TxConfig // nil is fine for construction
		k     keeper.Keeper   // zero value
		ak    types.AccountKeeper
		bk    types.BankKeeper
	)

	ops := WeightedOperations(appParams, reg, txGen, k, ak, bk)
	require.NotEmpty(t, ops)
}
