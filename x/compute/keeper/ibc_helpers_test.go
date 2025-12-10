package keeper

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/paw-chain/paw/x/compute/types"
)

func TestGetComputeChannel(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	channel, err := k.getComputeChannel(sdkCtx, AkashChainID)
	require.NoError(t, err)
	require.Equal(t, "channel-akash", channel)

	store := sdkCtx.KVStore(k.storeKey)
	store.Set([]byte("compute_channel_custom"), []byte("channel-custom"))
	channel, err = k.getComputeChannel(sdkCtx, "custom")
	require.NoError(t, err)
	require.Equal(t, "channel-custom", channel)

	store.Set([]byte("compute_channel_"+RenderChainID), []byte("channel-override"))
	channel, err = k.getComputeChannel(sdkCtx, RenderChainID)
	require.NoError(t, err)
	require.Equal(t, "channel-override", channel)
}

func TestEscrowLifecycle(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	requester := sdk.AccAddress([]byte("escrow_requester"))
	provider := sdk.AccAddress([]byte("escrow_provider"))
	amount := sdk.NewInt64Coin("upaw", 1000)

	require.NoError(t, k.bankKeeper.MintCoins(sdkCtx, types.ModuleName, sdk.NewCoins(amount)))
	require.NoError(t, k.bankKeeper.SendCoinsFromModuleToAccount(sdkCtx, types.ModuleName, requester, sdk.NewCoins(amount)))

	// Lock funds
	require.NoError(t, k.lockEscrow(sdkCtx, requester, amount))
	k.storeEscrow(sdkCtx, "job-1", &CrossChainEscrow{
		JobID:     "job-1",
		Requester: requester.String(),
		Provider:  provider.String(),
		Amount:    amount.Amount,
		Status:    "locked",
		LockedAt:  sdkCtx.BlockTime(),
	})

	initialProviderBal := k.bankKeeper.GetBalance(sdkCtx, provider, "upaw")

	// Release to provider
	require.NoError(t, k.releaseEscrow(sdkCtx, "job-1"))
	escrow := k.getEscrow(sdkCtx, "job-1")
	require.Equal(t, "released", escrow.Status)
	require.NotNil(t, escrow.ReleasedAt)
	finalProviderBal := k.bankKeeper.GetBalance(sdkCtx, provider, "upaw")
	require.Equal(t, initialProviderBal.Amount.Add(amount.Amount), finalProviderBal.Amount)
}

func TestReleaseEscrowFailsWithInvalidProvider(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	escrow := &CrossChainEscrow{
		JobID:     "job-invalid-provider",
		Requester: sdk.AccAddress([]byte("req")).String(),
		Provider:  "not-a-bech32", // force address decode failure
		Amount:    math.NewInt(500),
		Status:    "locked",
		LockedAt:  sdkCtx.BlockTime(),
	}
	k.storeEscrow(sdkCtx, escrow.JobID, escrow)

	err := k.releaseEscrow(sdkCtx, escrow.JobID)
	require.Error(t, err)
	require.Contains(t, err.Error(), "bech32")

	stored := k.getEscrow(sdkCtx, escrow.JobID)
	require.NotNil(t, stored)
	require.Equal(t, "locked", stored.Status)
	require.Nil(t, stored.ReleasedAt)
}

func TestReleaseEscrowFailsWithInsufficientFunds(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Escrow stored but module account not funded for transfer
	escrow := &CrossChainEscrow{
		JobID:     "job-no-funds",
		Requester: sdk.AccAddress([]byte("req-no-funds")).String(),
		Provider:  sdk.AccAddress([]byte("prov-no-funds")).String(),
		Amount:    math.NewInt(10_000),
		Status:    "locked",
		LockedAt:  sdkCtx.BlockTime(),
	}
	k.storeEscrow(sdkCtx, escrow.JobID, escrow)

	err := k.releaseEscrow(sdkCtx, escrow.JobID)
	require.Error(t, err)

	stored := k.getEscrow(sdkCtx, escrow.JobID)
	require.NotNil(t, stored)
	require.Equal(t, "locked", stored.Status)
}

func TestPendingJobTracking(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	k.storePendingJobSubmission(sdkCtx, "channel-0", 1, "job-123")
	require.Equal(t, "job-123", k.getPendingJobSubmission(sdkCtx, 1))
	k.removePendingJobSubmission(sdkCtx, "channel-0", 1)
	require.Equal(t, "", k.getPendingJobSubmission(sdkCtx, 1))
}

func TestCachedProviderStorage(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	provider := &RemoteComputeProvider{
		ChainID:      "chain-X",
		ProviderID:   "p1",
		Address:      "addr1",
		Capabilities: []string{"gpu"},
		PricePerUnit: math.LegacyNewDec(5),
		Reputation:   math.LegacyNewDec(9),
		Active:       true,
		LastSeen:     time.Now(),
	}
	k.storeProvider(sdkCtx, provider)

	result := k.getCachedProviders(sdkCtx, []string{"gpu"}, math.LegacyNewDec(10))
	require.Len(t, result, 1)
	require.Equal(t, "p1", result[0].ProviderID)
}

func TestProgressForStatus(t *testing.T) {
	require.Equal(t, uint32(10), progressForStatus("pending", 0))
	require.Equal(t, uint32(100), progressForStatus("completed", 50))
	require.Equal(t, uint32(0), progressForStatus("failed", 70))
	require.Equal(t, uint32(25), progressForStatus("accepted", 10))
}
