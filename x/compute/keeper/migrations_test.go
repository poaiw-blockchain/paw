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
	provider := sdk.AccAddress([]byte("migrate_provider"))
	require.NoError(t, k.SetProvider(ctx, types.Provider{
		Address:    provider.String(),
		Active:     true,
		Stake:      math.NewInt(1000),
		Reputation: 120, // intentionally above 100 to trigger normalization
		Moniker:    "test-provider",
		Endpoint:   "http://test.com",
		AvailableSpecs: types.ComputeSpec{
			CpuCores:       1000,
			MemoryMb:       1024,
			StorageGb:      10,
			TimeoutSeconds: 3600,
		},
		Pricing: types.Pricing{
			CpuPricePerMcoreHour:  math.LegacyMustNewDecFromStr("0.001"),
			MemoryPricePerMbHour:  math.LegacyMustNewDecFromStr("0.0001"),
			StoragePricePerGbHour: math.LegacyMustNewDecFromStr("0.00001"),
		},
	}))
	require.NoError(t, k.setActiveProviderIndex(ctx, provider, true))

	// Create a valid request with all required fields using proper API
	requester := sdk.AccAddress([]byte("migrate_requester"))
	requestID, err := k.SubmitRequest(ctx, requester,
		types.ComputeSpec{CpuCores: 500, MemoryMb: 512, StorageGb: 5, TimeoutSeconds: 1800},
		"test:latest",
		[]string{"echo", "test"},
		map[string]string{},
		math.NewInt(100000),
		"",
	)
	require.NoError(t, err)
	require.Greater(t, requestID, uint64(0))

	migrator := NewMigrator(*k)
	err = migrator.Migrate1to2(sdkCtx)
	require.NoError(t, err)
}
