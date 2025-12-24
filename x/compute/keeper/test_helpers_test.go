package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	authtypes "github.com/cosmos/cosmos-sdk/x/auth/types"
	"github.com/stretchr/testify/require"

	"github.com/paw-chain/paw/x/compute/types"
)

// Helper function to fund accounts in compute module tests
func fundTestAccount(t testing.TB, k interface{ GetBankKeeper() types.BankKeeper }, ctx sdk.Context, addr sdk.AccAddress, denom string, amount math.Int) {
	// Get bank keeper
	bankKeeper := k.GetBankKeeper()

	// Mint coins to module account first
	moduleAddr := authtypes.NewModuleAddress(types.ModuleName)
	coins := sdk.NewCoins(sdk.NewCoin(denom, amount))

	err := bankKeeper.MintCoins(ctx, types.ModuleName, coins)
	require.NoError(t, err)

	// Transfer to target address
	err = bankKeeper.SendCoins(ctx, moduleAddr, addr, coins)
	require.NoError(t, err)
}
