package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/dex/types"
)

func TestValidateSwapSize_Errors(t *testing.T) {
	k, _ := keepertest.DexKeeper(t)

	// Boundary: exactly at 10% should pass
	require.NoError(t, k.ValidateSwapSize(math.NewInt(100_000), math.NewInt(1_000_000)))

	// Exceeds max drain percent (default 10% → amount > 100_000 of 1_000_000)
	err := k.ValidateSwapSize(math.NewInt(150_000), math.NewInt(1_000_000))
	require.ErrorIs(t, err, types.ErrSwapTooLarge)
}

func TestSafeCalculateSwapOutput_Errors(t *testing.T) {
	k, _ := keepertest.DexKeeper(t)

	// Zero reserves
	_, err := k.SafeCalculateSwapOutput(
		nil,
		math.NewInt(10_000),
		math.NewInt(0),
		math.NewInt(1_000),
		math.LegacyZeroDec(),
	)
	require.ErrorIs(t, err, types.ErrInsufficientLiquidity)

	// Oversized amount should still respect liquidity invariant
	_, err = k.SafeCalculateSwapOutput(
		nil,
		math.NewInt(1_000_000_000),
		math.NewInt(1),
		math.NewInt(1),
		math.LegacyZeroDec(),
	)
	require.ErrorIs(t, err, types.ErrInsufficientLiquidity)

}

func TestDistributeProtocolFees_Events(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx).WithEventManager(sdk.NewEventManager())
	ctx = sdkCtx
	store := sdkCtx.KVStore(k.GetStoreKey())

	// No fees: should be no panic and no events emitted
	require.NoError(t, k.DistributeProtocolFees(ctx))
	require.Equal(t, 0, len(sdkCtx.EventManager().Events()))

	// With fees: event emitted and keys cleared
	bz, _ := math.NewInt(100).Marshal()
	store.Set(types.GetProtocolFeeKey("upaw"), bz)
	require.NoError(t, k.DistributeProtocolFees(ctx))
	require.Equal(t, 1, len(sdkCtx.EventManager().Events()))
	require.Nil(t, store.Get(types.GetProtocolFeeKey("upaw")))
}
