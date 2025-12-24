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

	// Seed minimal provider data to exercise migration paths
	// The migration's main job is to normalize provider reputations above 100
	provider := sdk.AccAddress([]byte("migrate_provider"))
	require.NoError(t, k.SetProvider(ctx, types.Provider{
		Address:                provider.String(),
		Active:                 true,
		Stake:                  math.NewInt(1000),
		Reputation:             120, // intentionally above 100 to trigger normalization
		Moniker:                "test-provider",
		Endpoint:               "http://test.com",
		TotalRequestsCompleted: 10,
		TotalRequestsFailed:    1,
		AvailableSpecs: types.ComputeSpec{
			CpuCores:       1000,
			MemoryMb:       1024,
			StorageGb:      10,
			TimeoutSeconds: 3600,
		},
		Pricing: types.Pricing{
			CpuPricePerMcoreHour:  math.LegacyMustNewDecFromStr("0.001"),
			MemoryPricePerMbHour:  math.LegacyMustNewDecFromStr("0.0001"),
			GpuPricePerHour:       math.LegacyMustNewDecFromStr("0.1"),
			StoragePricePerGbHour: math.LegacyMustNewDecFromStr("0.00001"),
		},
	}))
	require.NoError(t, k.setActiveProviderIndex(ctx, provider, true))

	// Run migration
	migrator := NewMigrator(*k)
	err := migrator.Migrate1to2(sdkCtx)
	require.NoError(t, err)

	// Verify provider reputation was normalized to 100
	migratedProvider, err := k.GetProvider(ctx, provider)
	require.NoError(t, err)
	require.Equal(t, uint32(100), migratedProvider.Reputation, "reputation should be capped at 100")
}
