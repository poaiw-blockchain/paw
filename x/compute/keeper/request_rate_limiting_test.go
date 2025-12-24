package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/compute/keeper"
	"github.com/paw-chain/paw/x/compute/types"
)

// TestRequestRateLimit_Cooldown tests the cooldown period between requests
func TestRequestRateLimit_Cooldown(t *testing.T) {
	t.Parallel()

	k, ctx := keepertest.ComputeKeeper(t)
	requester := sdk.AccAddress([]byte("test_requester_addr"))

	// Set params with cooldown of 10 blocks
	params := types.DefaultParams()
	params.RequestCooldownBlocks = 10
	require.NoError(t, k.SetParams(ctx, params))

	// First request should succeed
	err := k.CheckRequestRateLimit(ctx, requester)
	require.NoError(t, err)
	k.RecordComputeRequest(ctx, requester)

	// Immediate second request should fail
	err = k.CheckRequestRateLimit(ctx, requester)
	require.Error(t, err)
	require.Contains(t, err.Error(), "must wait")

	// Advance 5 blocks - should still fail
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 5)
	err = k.CheckRequestRateLimit(ctx, requester)
	require.Error(t, err)

	// Advance 5 more blocks (total 10) - should succeed
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 5)
	err = k.CheckRequestRateLimit(ctx, requester)
	require.NoError(t, err)
}

// TestRequestRateLimit_DailyLimit tests the daily request limit
func TestRequestRateLimit_DailyLimit(t *testing.T) {
	t.Parallel()

	k, ctx := keepertest.ComputeKeeper(t)
	requester := sdk.AccAddress([]byte("test_requester_addr"))

	// Set params with daily limit of 3 requests
	params := types.DefaultParams()
	params.MaxRequestsPerAddressPerDay = 3
	params.RequestCooldownBlocks = 0 // Disable cooldown for this test
	require.NoError(t, k.SetParams(ctx, params))

	// Submit 3 requests - all should succeed
	for i := 0; i < 3; i++ {
		err := k.CheckRequestRateLimit(ctx, requester)
		require.NoError(t, err, "request %d should succeed", i+1)
		k.RecordComputeRequest(ctx, requester)

		// Advance block to avoid cooldown issues
		ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)
	}

	// Fourth request should fail
	err := k.CheckRequestRateLimit(ctx, requester)
	require.Error(t, err)
	require.Contains(t, err.Error(), "exceeded maximum")

	// Advance to next day (17280 blocks = ~24 hours)
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 17280)

	// First request in new day should succeed
	err = k.CheckRequestRateLimit(ctx, requester)
	require.NoError(t, err)
}

// TestRequestRateLimit_MultipleUsers tests that rate limits are per-address
func TestRequestRateLimit_MultipleUsers(t *testing.T) {
	t.Parallel()

	k, ctx := keepertest.ComputeKeeper(t)
	requester1 := sdk.AccAddress([]byte("requester_1_address"))
	requester2 := sdk.AccAddress([]byte("requester_2_address"))

	// Set params with daily limit of 2 and cooldown of 5 blocks
	params := types.DefaultParams()
	params.MaxRequestsPerAddressPerDay = 2
	params.RequestCooldownBlocks = 5
	require.NoError(t, k.SetParams(ctx, params))

	// Requester 1 makes 2 requests
	err := k.CheckRequestRateLimit(ctx, requester1)
	require.NoError(t, err)
	k.RecordComputeRequest(ctx, requester1)

	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 10)

	err = k.CheckRequestRateLimit(ctx, requester1)
	require.NoError(t, err)
	k.RecordComputeRequest(ctx, requester1)

	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 10)

	// Requester 1's third request should fail
	err = k.CheckRequestRateLimit(ctx, requester1)
	require.Error(t, err)

	// Requester 2 should still be able to make requests
	err = k.CheckRequestRateLimit(ctx, requester2)
	require.NoError(t, err)
	k.RecordComputeRequest(ctx, requester2)

	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 10)

	err = k.CheckRequestRateLimit(ctx, requester2)
	require.NoError(t, err)
}

// TestRequestRateLimit_ZeroLimitsDisabled tests that zero values disable limits
func TestRequestRateLimit_ZeroLimitsDisabled(t *testing.T) {
	t.Parallel()

	k, ctx := keepertest.ComputeKeeper(t)
	requester := sdk.AccAddress([]byte("test_requester_addr"))

	// Set params with limits disabled (zero values)
	params := types.DefaultParams()
	params.MaxRequestsPerAddressPerDay = 0 // Disabled
	params.RequestCooldownBlocks = 0       // Disabled
	require.NoError(t, k.SetParams(ctx, params))

	// Should be able to make many requests without limits
	for i := 0; i < 10; i++ {
		err := k.CheckRequestRateLimit(ctx, requester)
		require.NoError(t, err)
		k.RecordComputeRequest(ctx, requester)
	}
}

// TestRequestRateLimit_RecordingAccuracy tests request count accuracy
func TestRequestRateLimit_RecordingAccuracy(t *testing.T) {
	t.Parallel()

	k, ctx := keepertest.ComputeKeeper(t)
	requester := sdk.AccAddress([]byte("test_requester_addr"))

	// Set params with daily limit of 5
	params := types.DefaultParams()
	params.MaxRequestsPerAddressPerDay = 5
	params.RequestCooldownBlocks = 0
	require.NoError(t, k.SetParams(ctx, params))

	// Record 3 requests
	for i := 0; i < 3; i++ {
		err := k.CheckRequestRateLimit(ctx, requester)
		require.NoError(t, err)
		k.RecordComputeRequest(ctx, requester)
		ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)
	}

	// Should have 2 remaining
	for i := 0; i < 2; i++ {
		err := k.CheckRequestRateLimit(ctx, requester)
		require.NoError(t, err, "should have %d remaining", 2-i)
		k.RecordComputeRequest(ctx, requester)
		ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)
	}

	// 6th request should fail
	err := k.CheckRequestRateLimit(ctx, requester)
	require.Error(t, err)
}

// TestCleanupOldRequestRateLimitData tests cleanup of old rate limit data
func TestCleanupOldRequestRateLimitData(t *testing.T) {
	t.Parallel()

	k, ctx := keepertest.ComputeKeeper(t)
	requester := sdk.AccAddress([]byte("test_requester_addr"))

	// Set params
	params := types.DefaultParams()
	params.MaxRequestsPerAddressPerDay = 100
	params.RequestCooldownBlocks = 0
	require.NoError(t, k.SetParams(ctx, params))

	// Record some requests at early height (well before the cutoff)
	// If current height is 35100, cutoff would be 35100 - 34560 = 540
	// So we record at height 530-539 which will be in the cleanup range (530-540)
	ctx = ctx.WithBlockHeight(530)
	k.RecordComputeRequest(ctx, requester)
	ctx = ctx.WithBlockHeight(535)
	k.RecordComputeRequest(ctx, requester)

	// Advance past retention period (34,560 blocks = 48 hours)
	// Set to height that makes our old records cleanable
	ctx = ctx.WithBlockHeight(35100)

	// Run cleanup - should clean data from ~540 blocks ago
	err := k.CleanupOldRequestRateLimitData(ctx)
	require.NoError(t, err)

	// Old data should be cleaned up - verify by checking events
	events := ctx.EventManager().Events()
	hasCleanupEvent := false
	for _, event := range events {
		if event.Type == "request_rate_limit_data_cleaned" {
			hasCleanupEvent = true
			break
		}
	}
	require.True(t, hasCleanupEvent, "cleanup event should be emitted")
}

// TestRequestRateLimit_EdgeCaseDayBoundary tests requests across day boundaries
func TestRequestRateLimit_EdgeCaseDayBoundary(t *testing.T) {
	t.Parallel()

	k, ctx := keepertest.ComputeKeeper(t)
	requester := sdk.AccAddress([]byte("test_requester_addr"))

	// Set params with daily limit of 2
	params := types.DefaultParams()
	params.MaxRequestsPerAddressPerDay = 2
	params.RequestCooldownBlocks = 0
	require.NoError(t, k.SetParams(ctx, params))

	// Start at block 17270 (near end of first day)
	ctx = ctx.WithBlockHeight(17270)

	// Make 2 requests in first day
	err := k.CheckRequestRateLimit(ctx, requester)
	require.NoError(t, err)
	k.RecordComputeRequest(ctx, requester)

	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)

	err = k.CheckRequestRateLimit(ctx, requester)
	require.NoError(t, err)
	k.RecordComputeRequest(ctx, requester)

	// Move to next day
	ctx = ctx.WithBlockHeight(17280) // New day starts

	// Should be able to make new requests
	err = k.CheckRequestRateLimit(ctx, requester)
	require.NoError(t, err)
}

// TestRequestRateLimit_CombinedLimits tests cooldown and daily limit together
func TestRequestRateLimit_CombinedLimits(t *testing.T) {
	t.Parallel()

	k, ctx := keepertest.ComputeKeeper(t)
	requester := sdk.AccAddress([]byte("test_requester_addr"))

	// Set params with both limits active
	params := types.DefaultParams()
	params.MaxRequestsPerAddressPerDay = 5
	params.RequestCooldownBlocks = 10
	require.NoError(t, k.SetParams(ctx, params))

	// Make first request
	err := k.CheckRequestRateLimit(ctx, requester)
	require.NoError(t, err)
	k.RecordComputeRequest(ctx, requester)

	// Immediate request should fail due to cooldown
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 1)
	err = k.CheckRequestRateLimit(ctx, requester)
	require.Error(t, err)
	require.Contains(t, err.Error(), "must wait")

	// After cooldown, request should succeed
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 15)
	err = k.CheckRequestRateLimit(ctx, requester)
	require.NoError(t, err)
	k.RecordComputeRequest(ctx, requester)

	// Repeat until daily limit reached
	for i := 2; i < 5; i++ {
		ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 15)
		err = k.CheckRequestRateLimit(ctx, requester)
		require.NoError(t, err)
		k.RecordComputeRequest(ctx, requester)
	}

	// Next request should fail due to daily limit
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 15)
	err = k.CheckRequestRateLimit(ctx, requester)
	require.Error(t, err)
	require.Contains(t, err.Error(), "exceeded maximum")
}

// TestRequestRateLimit_MsgServerIntegration tests integration with msg server
func TestRequestRateLimit_MsgServerIntegration(t *testing.T) {
	t.Parallel()

	k, ctx := keepertest.ComputeKeeper(t)
	ms := keeper.NewMsgServerImpl(*k)
	requester := sdk.AccAddress([]byte("test_requester_addr"))
	provider := sdk.AccAddress([]byte("test_provider_addr_"))

	// Set params with strict limits
	params := types.DefaultParams()
	params.MaxRequestsPerAddressPerDay = 2
	params.RequestCooldownBlocks = 5
	require.NoError(t, k.SetParams(ctx, params))

	// Register a provider that can handle the request specs
	err := k.RegisterProvider(ctx, provider, "test-provider", "http://test.com",
		types.ComputeSpec{CpuCores: 2000, MemoryMb: 4096, StorageGb: 10, TimeoutSeconds: 3600},
		types.Pricing{
			CpuPricePerMcoreHour:  math.LegacyMustNewDecFromStr("0.001"),
			MemoryPricePerMbHour:  math.LegacyMustNewDecFromStr("0.0001"),
			GpuPricePerHour:       math.LegacyMustNewDecFromStr("0.1"),
			StoragePricePerGbHour: math.LegacyMustNewDecFromStr("0.00001"),
		},
		math.NewInt(1000000),
	)
	require.NoError(t, err)

	// Create a valid request message
	msg := &types.MsgSubmitRequest{
		Requester:      requester.String(),
		Specs:          types.ComputeSpec{CpuCores: 1000, MemoryMb: 1024, StorageGb: 1, TimeoutSeconds: 3600},
		ContainerImage: "test:latest",
		Command:        []string{"echo", "test"},
		MaxPayment:     math.NewInt(1000000),
	}

	// First request should succeed
	_, err = ms.SubmitRequest(ctx, msg)
	require.NoError(t, err)

	// Second request without cooldown should fail
	_, err = ms.SubmitRequest(ctx, msg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "must wait")

	// After cooldown, should succeed
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 10)
	_, err = ms.SubmitRequest(ctx, msg)
	require.NoError(t, err)

	// Third request after cooldown should fail (daily limit)
	ctx = ctx.WithBlockHeight(ctx.BlockHeight() + 10)
	_, err = ms.SubmitRequest(ctx, msg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "exceeded maximum")
}
