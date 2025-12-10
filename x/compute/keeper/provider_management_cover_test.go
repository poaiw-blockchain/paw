package keeper

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/compute/types"
)

func TestApplyReputationDecayToAllOldEntries(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx).WithBlockTime(time.Now().Add(48 * time.Hour))

	provider := sdk.AccAddress([]byte("rep_decay_provider"))
	initialRep := types.ProviderReputation{
		Provider:           provider.String(),
		ReliabilityScore:   1,
		SpeedScore:         1,
		AccuracyScore:      1,
		AvailabilityScore:  1,
		OverallScore:       100,
		LastDecayTimestamp: time.Now().Add(-72 * time.Hour),
	}
	require.NoError(t, k.SetProviderReputation(sdkCtx, initialRep))

	require.NoError(t, k.ApplyReputationDecayToAll(sdkCtx))

	updated, err := k.GetProviderReputation(sdkCtx, provider)
	require.NoError(t, err)
	require.Less(t, updated.ReliabilityScore, initialRep.ReliabilityScore)
	require.Equal(t, sdkCtx.BlockTime(), updated.LastDecayTimestamp)
}

func TestMonitorProviderAvailabilityPenalizesInactive(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx).WithBlockTime(time.Now())

	provider := sdk.AccAddress([]byte("availability_provider"))
	require.NoError(t, k.SetProvider(sdkCtx, types.Provider{
		Address: provider.String(),
		Active:  true,
	}))
	require.NoError(t, k.SetProviderReputation(sdkCtx, types.ProviderReputation{
		Provider:          provider.String(),
		AvailabilityScore: 1,
	}))

	// Write performance metrics with last_active older than 1h
	store := sdkCtx.KVStore(k.storeKey)
	metrics := map[string]interface{}{
		"last_active": float64(sdkCtx.BlockTime().Add(-2 * time.Hour).Unix()),
	}
	bz, err := json.Marshal(metrics)
	require.NoError(t, err)
	store.Set([]byte("perf_"+provider.String()), bz)

	require.NoError(t, k.MonitorProviderAvailability(sdkCtx))

	rep, err := k.GetProviderReputation(sdkCtx, provider)
	require.NoError(t, err)
	require.Less(t, rep.AvailabilityScore, 1.0)
}
