package keeper

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/paw-chain/paw/x/compute/types"
)

func TestMigrator_Migrate1to2(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Seed minimal provider/request data to exercise migration paths
	provider := sdk.AccAddress([]byte("migrate_provider"))
	require.NoError(t, k.SetProvider(ctx, types.Provider{
		Address:    provider.String(),
		Active:     true,
		Stake:      math.NewInt(1000),
		Reputation: 120, // intentionally above 100 to trigger normalization
	}))
	require.NoError(t, k.setActiveProviderIndex(ctx, provider, true))

	migrator := NewMigrator(*k)
	err := migrator.Migrate1to2(sdkCtx)
	require.NoError(t, err)
}
