package keeper

import (
	"strings"
	"testing"
	"time"

	sdkmath "cosmossdk.io/math"
	"github.com/cosmos/cosmos-sdk/crypto/keys/secp256k1"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/paw-chain/paw/x/compute/types"
)

func TestCircuitBreakerLifecycle(t *testing.T) {
	k, ctx := setupKeeperForTest(t)

	require.False(t, k.IsCircuitBreakerOpen(ctx))

	err := k.OpenCircuitBreaker(ctx, "ops", "maintenance")
	require.NoError(t, err)
	require.True(t, k.IsCircuitBreakerOpen(ctx))

	enabled, reason, actor := k.GetCircuitBreakerState(ctx)
	require.True(t, enabled)
	require.Equal(t, "maintenance", reason)
	require.Equal(t, "ops", actor)

	err = k.OpenCircuitBreaker(ctx, "ops", "duplicate")
	require.ErrorIs(t, err, types.ErrCircuitBreakerAlreadyOpen)

	err = k.CheckCircuitBreaker(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "maintenance")
	require.Contains(t, err.Error(), "ops")

	err = k.CloseCircuitBreaker(ctx, "ops", "resume")
	require.NoError(t, err)
	require.False(t, k.IsCircuitBreakerOpen(ctx))

	err = k.CloseCircuitBreaker(ctx, "ops", "already-closed")
	require.ErrorIs(t, err, types.ErrCircuitBreakerAlreadyClosed)
}

func TestProviderCircuitBreakerLifecycle(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	providerAddr := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String()

	require.False(t, k.IsProviderCircuitBreakerOpen(ctx, providerAddr))

	err := k.OpenProviderCircuitBreaker(ctx, providerAddr, "ops", "investigation")
	require.NoError(t, err)
	require.True(t, k.IsProviderCircuitBreakerOpen(ctx, providerAddr))

	err = k.OpenProviderCircuitBreaker(ctx, providerAddr, "ops", "duplicate")
	require.ErrorIs(t, err, types.ErrCircuitBreakerAlreadyOpen)

	err = k.CheckProviderCircuitBreaker(ctx, providerAddr)
	require.Error(t, err)
	require.Contains(t, err.Error(), "provider")

	err = k.CloseProviderCircuitBreaker(ctx, providerAddr, "ops", "resolved")
	require.NoError(t, err)
	require.False(t, k.IsProviderCircuitBreakerOpen(ctx, providerAddr))

	err = k.CloseProviderCircuitBreaker(ctx, providerAddr, "ops", "already-closed")
	require.ErrorIs(t, err, types.ErrCircuitBreakerAlreadyClosed)
}

func TestProviderCircuitBreakerRespectsGlobal(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	providerAddr := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address()).String()

	require.NoError(t, k.OpenCircuitBreaker(ctx, "ops", "global pause"))
	err := k.CheckProviderCircuitBreaker(ctx, providerAddr)
	require.Error(t, err)
	require.True(t, strings.Contains(err.Error(), "global check"))
	require.True(t, strings.Contains(err.Error(), "global pause"))

	require.NoError(t, k.CloseCircuitBreaker(ctx, "ops", "resume"))
}

func TestJobCancellationLifecycle(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	ctx = ctx.WithBlockTime(time.Unix(1_700_000_000, 0))

	jobID := "job-123"
	err := k.CancelJob(ctx, jobID, "ops", "faulty result")
	require.NoError(t, err)

	require.True(t, k.IsJobCancelled(ctx, jobID))

	metadata, found := k.GetJobCancellation(ctx, jobID)
	require.True(t, found)
	require.Equal(t, jobID, metadata["job_id"])
	require.Equal(t, "ops", metadata["actor"])
	require.Equal(t, "faulty result", metadata["reason"])
	require.NotEmpty(t, metadata["timestamp"])

	k.ClearJobCancellation(ctx, jobID)
	require.False(t, k.IsJobCancelled(ctx, jobID))
	_, found = k.GetJobCancellation(ctx, jobID)
	require.False(t, found)
}

func TestReputationOverrideLifecycle(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	ctx = ctx.WithBlockTime(time.Unix(1_700_000_000, 0))

	providerAddr := sdk.AccAddress(secp256k1.GenPrivKey().PubKey().Address())
	providerAddrStr := providerAddr.String()

	err := k.SetReputationOverride(ctx, providerAddrStr, 80, "gov", "temporary downgrade")
	require.NoError(t, err)

	overrideScore, ok := k.GetReputationOverride(ctx, providerAddrStr)
	require.True(t, ok)
	require.Equal(t, int64(80), overrideScore)

	score, ok := k.GetReputationWithOverride(ctx, providerAddrStr)
	require.True(t, ok)
	require.Equal(t, int64(80), score)

	k.ClearReputationOverride(ctx, providerAddrStr)
	_, ok = k.GetReputationOverride(ctx, providerAddrStr)
	require.False(t, ok)

	err = k.SetProvider(ctx, types.Provider{
		Address:  providerAddrStr,
		Moniker:  "provider-1",
		Endpoint: "http://provider.test",
		Stake:    sdkmath.NewInt(1),
		Active:   true,
	})
	require.NoError(t, err)

	err = k.SetProviderReputation(ctx, types.ProviderReputation{
		Provider:     providerAddrStr,
		OverallScore: 55,
	})
	require.NoError(t, err)

	score, ok = k.GetReputationWithOverride(ctx, providerAddrStr)
	require.True(t, ok)
	require.Equal(t, int64(55), score)
}
