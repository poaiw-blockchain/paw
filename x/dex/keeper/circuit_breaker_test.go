package keeper_test

import (
	"testing"
	"time"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/dex/keeper"
	"github.com/paw-chain/paw/x/dex/types"
)

func TestCircuitBreakerConfig(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	// Test default config
	defaultConfig := keeper.DefaultCircuitBreakerConfig()
	require.Equal(t, math.LegacyNewDecWithPrec(10, 2), defaultConfig.Threshold1Min)
	require.Equal(t, math.LegacyNewDecWithPrec(20, 2), defaultConfig.Threshold5Min)
	require.Equal(t, math.LegacyNewDecWithPrec(25, 2), defaultConfig.Threshold15Min)
	require.Equal(t, math.LegacyNewDecWithPrec(30, 2), defaultConfig.Threshold1Hour)
	require.Equal(t, int64(600), defaultConfig.CooldownPeriod)
	require.True(t, defaultConfig.EnableGradualResume)

	// Test set and get config
	customConfig := keeper.CircuitBreakerConfig{
		Threshold1Min:       math.LegacyNewDecWithPrec(15, 2), // 15%
		Threshold5Min:       math.LegacyNewDecWithPrec(25, 2), // 25%
		Threshold15Min:      math.LegacyNewDecWithPrec(30, 2), // 30%
		Threshold1Hour:      math.LegacyNewDecWithPrec(40, 2), // 40%
		CooldownPeriod:      1200,                             // 20 minutes
		EnableGradualResume: false,
		ResumeVolumeFactor:  math.LegacyNewDecWithPrec(3, 1), // 30%
	}

	err := k.SetCircuitBreakerConfig(ctx, customConfig)
	require.NoError(t, err)

	retrievedConfig := k.GetCircuitBreakerConfig(ctx)
	require.Equal(t, customConfig.Threshold1Min, retrievedConfig.Threshold1Min)
	require.Equal(t, customConfig.Threshold5Min, retrievedConfig.Threshold5Min)
	require.Equal(t, customConfig.CooldownPeriod, retrievedConfig.CooldownPeriod)
}

func TestCircuitBreakerValidation(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	tests := []struct {
		name      string
		config    keeper.CircuitBreakerConfig
		expectErr bool
	}{
		{
			name: "valid config",
			config: keeper.CircuitBreakerConfig{
				Threshold1Min:       math.LegacyNewDecWithPrec(10, 2),
				Threshold5Min:       math.LegacyNewDecWithPrec(20, 2),
				Threshold15Min:      math.LegacyNewDecWithPrec(25, 2),
				Threshold1Hour:      math.LegacyNewDecWithPrec(30, 2),
				CooldownPeriod:      600,
				EnableGradualResume: true,
				ResumeVolumeFactor:  math.LegacyNewDecWithPrec(5, 1),
			},
			expectErr: false,
		},
		{
			name: "negative threshold",
			config: keeper.CircuitBreakerConfig{
				Threshold1Min:       math.LegacyNewDecWithPrec(-10, 2),
				Threshold5Min:       math.LegacyNewDecWithPrec(20, 2),
				Threshold15Min:      math.LegacyNewDecWithPrec(25, 2),
				Threshold1Hour:      math.LegacyNewDecWithPrec(30, 2),
				CooldownPeriod:      600,
				EnableGradualResume: false,
				ResumeVolumeFactor:  math.LegacyNewDecWithPrec(5, 1),
			},
			expectErr: true,
		},
		{
			name: "thresholds not in ascending order",
			config: keeper.CircuitBreakerConfig{
				Threshold1Min:       math.LegacyNewDecWithPrec(30, 2),
				Threshold5Min:       math.LegacyNewDecWithPrec(20, 2),
				Threshold15Min:      math.LegacyNewDecWithPrec(25, 2),
				Threshold1Hour:      math.LegacyNewDecWithPrec(10, 2),
				CooldownPeriod:      600,
				EnableGradualResume: false,
				ResumeVolumeFactor:  math.LegacyNewDecWithPrec(5, 1),
			},
			expectErr: true,
		},
		{
			name: "negative cooldown period",
			config: keeper.CircuitBreakerConfig{
				Threshold1Min:       math.LegacyNewDecWithPrec(10, 2),
				Threshold5Min:       math.LegacyNewDecWithPrec(20, 2),
				Threshold15Min:      math.LegacyNewDecWithPrec(25, 2),
				Threshold1Hour:      math.LegacyNewDecWithPrec(30, 2),
				CooldownPeriod:      -100,
				EnableGradualResume: false,
				ResumeVolumeFactor:  math.LegacyNewDecWithPrec(5, 1),
			},
			expectErr: true,
		},
		{
			name: "invalid resume volume factor",
			config: keeper.CircuitBreakerConfig{
				Threshold1Min:       math.LegacyNewDecWithPrec(10, 2),
				Threshold5Min:       math.LegacyNewDecWithPrec(20, 2),
				Threshold15Min:      math.LegacyNewDecWithPrec(25, 2),
				Threshold1Hour:      math.LegacyNewDecWithPrec(30, 2),
				CooldownPeriod:      600,
				EnableGradualResume: true,
				ResumeVolumeFactor:  math.LegacyNewDecWithPrec(150, 2), // 150% > 100%
			},
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := k.SetCircuitBreakerConfig(ctx, tt.config)
			if tt.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestCircuitBreakerTrip(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	// Create a pool
	poolId := uint64(1)
	initialPrice := math.LegacyNewDecWithPrec(100, 0) // 100

	// Trip the circuit breaker
	reason := "Test trip"
	k.TripCircuitBreaker(ctx, poolId, reason, initialPrice)

	// Verify state
	state := k.GetCircuitBreakerState(ctx, poolId)
	require.True(t, state.IsTripped)
	require.Equal(t, reason, state.TripReason)
	require.Equal(t, poolId, state.PoolId)
	require.Equal(t, initialPrice, state.PriceAtTrip)
	require.Equal(t, ctx.BlockHeight(), state.TrippedAt)

	// Verify circuit breaker is active
	isTripped := k.IsCircuitBreakerTripped(ctx, poolId)
	require.True(t, isTripped)
}

func TestCircuitBreakerResume(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	poolId := uint64(1)
	initialPrice := math.LegacyNewDecWithPrec(100, 0)

	// Trip the circuit breaker
	k.TripCircuitBreaker(ctx, poolId, "Test trip", initialPrice)

	// Try to resume before cooldown - should fail
	err := k.ResumeTrading(ctx, poolId, false)
	require.Error(t, err)

	// Resume with governance override - should succeed
	err = k.ResumeTrading(ctx, poolId, true)
	require.NoError(t, err)

	// Verify state
	state := k.GetCircuitBreakerState(ctx, poolId)
	require.False(t, state.IsTripped)

	isTripped := k.IsCircuitBreakerTripped(ctx, poolId)
	require.False(t, isTripped)
}

func TestCircuitBreakerAutomaticResume(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	// Set short cooldown period
	config := keeper.DefaultCircuitBreakerConfig()
	config.CooldownPeriod = 10 // 10 seconds
	err := k.SetCircuitBreakerConfig(ctx, config)
	require.NoError(t, err)

	poolId := uint64(1)
	initialPrice := math.LegacyNewDecWithPrec(100, 0)

	// Trip the circuit breaker
	k.TripCircuitBreaker(ctx, poolId, "Test trip", initialPrice)

	// Verify it's tripped
	require.True(t, k.IsCircuitBreakerTripped(ctx, poolId))

	// Move time forward past cooldown
	newTime := ctx.BlockTime().Add(15 * time.Second)
	newCtx := ctx.WithBlockTime(newTime)

	// Check if circuit breaker is still tripped (should auto-resume)
	isTripped := k.IsCircuitBreakerTripped(newCtx, poolId)
	require.False(t, isTripped)
}

func TestPriceVolatilityDetection(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	// Create a test pool
	creator := "paw1test123"
	tokenA := "token_a"
	tokenB := "token_b"
	amountA := math.NewInt(1000000)
	amountB := math.NewInt(1000000)

	poolId, err := k.CreatePool(ctx, creator, tokenA, tokenB, amountA, amountB)
	require.NoError(t, err)

	// Record initial price
	err = k.RecordPrice(ctx, poolId)
	require.NoError(t, err)

	// Simulate price movement by updating pool reserves
	pool := k.GetPool(ctx, poolId)
	require.NotNil(t, pool)

	// Simulate a 15% price change (should trip 1min threshold at 10%)
	pool.ReserveB = pool.ReserveB.MulRaw(115).QuoRaw(100) // +15%
	k.SetPool(ctx, *pool)

	// Record new price
	err = k.RecordPrice(ctx, poolId)
	require.NoError(t, err)

	// Move time forward 30 seconds (within 1 minute window)
	newTime := ctx.BlockTime().Add(30 * time.Second)
	newCtx := ctx.WithBlockTime(newTime).WithBlockHeight(ctx.BlockHeight() + 1)

	// Check volatility - should trip circuit breaker
	err = k.DetectPriceVolatility(newCtx, poolId)
	require.Error(t, err)
	require.Contains(t, err.Error(), "circuit breaker tripped")

	// Verify circuit breaker is tripped
	state := k.GetCircuitBreakerState(newCtx, poolId)
	require.True(t, state.IsTripped)
}

func TestSwapVolumeLimit(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	// Set config with gradual resume
	config := keeper.DefaultCircuitBreakerConfig()
	config.EnableGradualResume = true
	config.ResumeVolumeFactor = math.LegacyNewDecWithPrec(2, 1) // 20%
	err := k.SetCircuitBreakerConfig(ctx, config)
	require.NoError(t, err)

	// Create a test pool
	creator := "paw1test123"
	poolId, err := k.CreatePool(ctx, creator, "token_a", "token_b",
		math.NewInt(1000000), math.NewInt(1000000))
	require.NoError(t, err)

	// Trip and resume with gradual mode
	k.TripCircuitBreaker(ctx, poolId, "Test", math.LegacyNewDec(1))
	err = k.ResumeTrading(ctx, poolId, true)
	require.NoError(t, err)

	// Try a large swap (should fail due to volume limit)
	largeAmount := math.NewInt(300000) // 30% of reserves
	err = k.CheckSwapVolumeLimit(ctx, poolId, largeAmount)
	require.Error(t, err)
	require.Contains(t, err.Error(), "gradual resume limit")

	// Try a smaller swap (should succeed)
	smallAmount := math.NewInt(100000) // 10% of reserves
	err = k.CheckSwapVolumeLimit(ctx, poolId, smallAmount)
	require.NoError(t, err)
}

func TestCheckCircuitBreakerInSwap(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	// Create a pool
	creator := "paw1test123"
	tokenA := "token_a"
	tokenB := "token_b"
	amountA := math.NewInt(1000000)
	amountB := math.NewInt(1000000)

	poolId, err := k.CreatePool(ctx, creator, tokenA, tokenB, amountA, amountB)
	require.NoError(t, err)

	// Trip circuit breaker
	k.TripCircuitBreaker(ctx, poolId, "Manual trip for testing", math.LegacyNewDec(1))

	// Try to swap - should fail
	trader := "paw1trader"
	_, err = k.Swap(ctx, trader, poolId, tokenA, tokenB, math.NewInt(1000), math.NewInt(1))
	require.Error(t, err)
	require.Contains(t, err.Error(), "circuit breaker")

	// Resume trading
	err = k.ResumeTrading(ctx, poolId, true)
	require.NoError(t, err)

	// Try to swap again - should succeed (assuming other conditions are met)
	// Note: This may still fail due to balance issues in test, but NOT due to circuit breaker
	_, err = k.Swap(ctx, trader, poolId, tokenA, tokenB, math.NewInt(1000), math.NewInt(1))
	if err != nil {
		require.NotContains(t, err.Error(), "circuit breaker")
	}
}

func TestGetAllCircuitBreakerStates(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	// Create multiple pools and trip some circuit breakers
	pool1Price := math.LegacyNewDecWithPrec(100, 0)
	pool2Price := math.LegacyNewDecWithPrec(200, 0)

	k.TripCircuitBreaker(ctx, 1, "Pool 1 tripped", pool1Price)
	k.TripCircuitBreaker(ctx, 2, "Pool 2 tripped", pool2Price)

	// Get all states
	states := k.GetAllCircuitBreakerStates(ctx)
	require.Len(t, states, 2)

	// Verify states
	foundPool1 := false
	foundPool2 := false
	for _, state := range states {
		if state.PoolId == 1 {
			foundPool1 = true
			require.True(t, state.IsTripped)
			require.Equal(t, "Pool 1 tripped", state.TripReason)
		}
		if state.PoolId == 2 {
			foundPool2 = true
			require.True(t, state.IsTripped)
			require.Equal(t, "Pool 2 tripped", state.TripReason)
		}
	}
	require.True(t, foundPool1)
	require.True(t, foundPool2)
}
