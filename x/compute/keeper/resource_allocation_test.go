package keeper

import (
	"testing"

	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"crypto/ed25519"
	"github.com/stretchr/testify/require"

	"github.com/paw-chain/paw/x/compute/types"
)

// Covers ReleaseResources and provider resource allocate/release paths.
func TestResourceAllocationLifecycle(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	account := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())

	specs := types.ComputeSpec{
		CpuCores:  2,
		MemoryMb:  1024,
		GpuCount:  1,
		StorageGb: 10,
	}

	// Allocate then release quota
	require.NoError(t, k.AllocateResources(ctx, account, specs))
	quota, err := k.GetResourceQuota(ctx, account)
	require.NoError(t, err)
	require.Equal(t, uint64(1), quota.CurrentRequests)
	require.Equal(t, uint64(2), quota.CurrentCpu)

	require.NoError(t, k.ReleaseResources(ctx, account, specs))
	quota, err = k.GetResourceQuota(ctx, account)
	require.NoError(t, err)
	require.Equal(t, uint64(0), quota.CurrentRequests)
	require.Equal(t, uint64(0), quota.CurrentCpu)

	// Provider load tracker allocate/release
	provider := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
	err = k.SetProviderLoadTracker(ctx, types.ProviderLoadTracker{
		Provider:              provider.String(),
		MaxConcurrentRequests: 5,
		CurrentRequests:       0,
		TotalCpuCores:         8,
		UsedCpuCores:          0,
		TotalMemoryMb:         4096,
		UsedMemoryMb:          0,
		TotalGpus:             2,
		UsedGpus:              0,
	})
	require.NoError(t, err)

	require.NoError(t, k.AllocateProviderResources(ctx, provider, specs))
	tracker, err := k.GetProviderLoadTracker(ctx, provider)
	require.NoError(t, err)
	require.Equal(t, uint64(1), tracker.CurrentRequests)
	require.Equal(t, uint64(2), tracker.UsedCpuCores)

	require.NoError(t, k.ReleaseProviderResources(ctx, provider, specs))
	tracker, err = k.GetProviderLoadTracker(ctx, provider)
	require.NoError(t, err)
	require.Equal(t, uint64(0), tracker.CurrentRequests)
	require.Equal(t, uint64(0), tracker.UsedCpuCores)
}

// Covers decrementTotalProviderCount guard against underflow.
func TestDecrementTotalProviderCount(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	require.Equal(t, uint64(0), k.GetTotalProviderCount(ctx))

	k.decrementTotalProviderCount(ctx)
	require.Equal(t, uint64(0), k.GetTotalProviderCount(ctx))

	k.incrementTotalProviderCount(ctx)
	require.Equal(t, uint64(1), k.GetTotalProviderCount(ctx))
	k.decrementTotalProviderCount(ctx)
	require.Equal(t, uint64(0), k.GetTotalProviderCount(ctx))
}

// Covers msg_server RegisterSigningKey and keeper Has/GetRegisteredSigningKey.
func TestMsgServerRegisterSigningKey(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	provider := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
	providerStr := provider.String()

	require.NoError(t, k.SetProvider(ctx, types.Provider{
		Address:  providerStr,
		Moniker:  "provider",
		Endpoint: "http://p",
		Stake:    sdkmath.NewInt(1),
		Active:   true,
	}))

	edPriv := ed25519.NewKeyFromSeed([]byte("01234567890123456789012345678901"))
	pub := edPriv.Public().(ed25519.PublicKey)

	msgSrv := msgServer{Keeper: *k}
	_, err := msgSrv.RegisterSigningKey(ctx, &types.MsgRegisterSigningKey{
		Provider:   providerStr,
		PublicKey:  pub,
		OldKeySignature: nil,
	})
	require.NoError(t, err)

	require.True(t, k.HasRegisteredSigningKey(ctx, provider))
	require.NotNil(t, k.GetRegisteredSigningKey(ctx, provider))
}
