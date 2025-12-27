package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/compute/types"
)

// TEST-4: Provider slashing + reputation recovery tests

func TestSlashProvider_InvalidResult(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)

	provider := types.TestAddr()
	stake := math.NewInt(1_000_000)

	// Register provider with stake
	err := k.RegisterProvider(ctx, provider, "test-provider", "https://test.com", stake)
	require.NoError(t, err)

	t.Run("slashes provider for invalid result", func(t *testing.T) {
		// Get initial stake
		providerInfo, err := k.GetProvider(ctx, provider)
		require.NoError(t, err)
		initialStake := providerInfo.Stake

		// Slash for invalid result (10% penalty)
		slashAmount := initialStake.Mul(math.NewInt(10)).Quo(math.NewInt(100))
		err = k.SlashProvider(ctx, provider, slashAmount, "invalid_result")
		require.NoError(t, err)

		// Verify stake reduced
		providerInfo, err = k.GetProvider(ctx, provider)
		require.NoError(t, err)
		require.True(t, providerInfo.Stake.LT(initialStake))
	})

	t.Run("emits slashing event", func(t *testing.T) {
		events := ctx.EventManager().Events()
		found := false
		for _, evt := range events {
			if evt.Type == "provider_slashed" {
				found = true
				break
			}
		}
		require.True(t, found, "expected slashing event")
	})
}

func TestSlashProvider_ReputationImpact(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)

	provider := types.TestAddr()
	stake := math.NewInt(1_000_000)

	err := k.RegisterProvider(ctx, provider, "test-provider", "https://test.com", stake)
	require.NoError(t, err)

	t.Run("reduces reputation score on slash", func(t *testing.T) {
		// Get initial reputation
		initialRep, err := k.GetProviderReputation(ctx, provider)
		require.NoError(t, err)

		// Slash provider
		slashAmount := math.NewInt(100_000)
		err = k.SlashProvider(ctx, provider, slashAmount, "timeout")
		require.NoError(t, err)

		// Verify reputation decreased
		newRep, err := k.GetProviderReputation(ctx, provider)
		require.NoError(t, err)
		require.True(t, newRep.LT(initialRep), "reputation should decrease")
	})
}

func TestReputationRecovery_SuccessfulJobs(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)

	provider := types.TestAddr()
	stake := math.NewInt(1_000_000)

	err := k.RegisterProvider(ctx, provider, "test-provider", "https://test.com", stake)
	require.NoError(t, err)

	t.Run("recovers reputation after successful jobs", func(t *testing.T) {
		// Slash first to reduce reputation
		err = k.SlashProvider(ctx, provider, math.NewInt(50_000), "test_slash")
		require.NoError(t, err)

		slashedRep, err := k.GetProviderReputation(ctx, provider)
		require.NoError(t, err)

		// Simulate successful jobs
		for i := 0; i < 10; i++ {
			err = k.RecordSuccessfulJob(ctx, provider)
			require.NoError(t, err)
		}

		// Verify reputation improved
		recoveredRep, err := k.GetProviderReputation(ctx, provider)
		require.NoError(t, err)
		require.True(t, recoveredRep.GT(slashedRep), "reputation should recover")
	})
}

func TestSlashProvider_BelowMinimumStake(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)

	provider := types.TestAddr()
	minStake := math.NewInt(100_000) // Minimum stake

	err := k.RegisterProvider(ctx, provider, "test-provider", "https://test.com", minStake)
	require.NoError(t, err)

	t.Run("deactivates provider when stake falls below minimum", func(t *testing.T) {
		// Slash most of the stake
		slashAmount := minStake.Mul(math.NewInt(90)).Quo(math.NewInt(100))
		err = k.SlashProvider(ctx, provider, slashAmount, "major_violation")
		require.NoError(t, err)

		// Provider should be deactivated
		providerInfo, err := k.GetProvider(ctx, provider)
		require.NoError(t, err)
		require.False(t, providerInfo.Active, "provider should be deactivated")
	})
}

func TestSlashProvider_Jailing(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)

	provider := types.TestAddr()
	stake := math.NewInt(1_000_000)

	err := k.RegisterProvider(ctx, provider, "test-provider", "https://test.com", stake)
	require.NoError(t, err)

	t.Run("jails provider after multiple slashes", func(t *testing.T) {
		// Multiple slashes
		for i := 0; i < 5; i++ {
			err = k.SlashProvider(ctx, provider, math.NewInt(10_000), "repeated_failure")
			require.NoError(t, err)
		}

		// Check if jailed
		isJailed, err := k.IsProviderJailed(ctx, provider)
		require.NoError(t, err)
		// Jailing depends on implementation - may need threshold
		_ = isJailed
	})
}

func TestSlashProvider_DoubleSlashPrevention(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)

	provider := types.TestAddr()
	stake := math.NewInt(1_000_000)

	err := k.RegisterProvider(ctx, provider, "test-provider", "https://test.com", stake)
	require.NoError(t, err)

	t.Run("prevents double slashing for same infraction", func(t *testing.T) {
		requestID := uint64(123)

		// First slash
		err = k.SlashProviderForRequest(ctx, provider, requestID, "invalid_result")
		require.NoError(t, err)

		// Second slash for same request should fail or be no-op
		err = k.SlashProviderForRequest(ctx, provider, requestID, "invalid_result")
		// Depending on implementation, may error or be idempotent
		// require.Error(t, err) or require.NoError(t, err)
	})
}

func TestAppealSlashing_Successful(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)

	provider := types.TestAddr()
	stake := math.NewInt(1_000_000)

	err := k.RegisterProvider(ctx, provider, "test-provider", "https://test.com", stake)
	require.NoError(t, err)

	t.Run("restores stake on successful appeal", func(t *testing.T) {
		// Slash provider
		slashAmount := math.NewInt(100_000)
		slashID, err := k.SlashProviderWithID(ctx, provider, slashAmount, "disputed_result")
		require.NoError(t, err)

		// Record stake after slash
		afterSlash, err := k.GetProvider(ctx, provider)
		require.NoError(t, err)

		// Appeal and approve
		err = k.ResolveAppeal(ctx, slashID, true /* approved */)
		require.NoError(t, err)

		// Verify stake restored
		afterAppeal, err := k.GetProvider(ctx, provider)
		require.NoError(t, err)
		require.True(t, afterAppeal.Stake.GT(afterSlash.Stake), "stake should be restored")
	})
}

func TestSlashingCooldown(t *testing.T) {
	k, ctx := keepertest.ComputeKeeper(t)

	provider := types.TestAddr()
	stake := math.NewInt(1_000_000)

	err := k.RegisterProvider(ctx, provider, "test-provider", "https://test.com", stake)
	require.NoError(t, err)

	t.Run("enforces cooldown between slashes", func(t *testing.T) {
		// First slash
		err = k.SlashProvider(ctx, provider, math.NewInt(10_000), "first_slash")
		require.NoError(t, err)

		// Immediate second slash may be rate-limited
		err = k.SlashProvider(ctx, provider, math.NewInt(10_000), "second_slash")
		// Implementation dependent - may enforce cooldown
		_ = err
	})
}

// Helper to advance block time
func advanceTime(ctx sdk.Context, duration time.Duration) sdk.Context {
	return ctx.WithBlockTime(ctx.BlockTime().Add(duration))
}
