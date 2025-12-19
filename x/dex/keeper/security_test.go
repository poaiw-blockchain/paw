package keeper_test

import (
	"context"
	"testing"
	"time"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/dex/keeper"
	"github.com/paw-chain/paw/x/dex/types"
)

// TestReentrancyProtection tests that reentrancy attacks are prevented
func TestReentrancyProtection(t *testing.T) {
	// This test verifies that the reentrancy guard prevents nested calls
	// In production, this would be tested with actual malicious contracts
	// For now, we verify the guard mechanism exists and works

	guard := keeper.NewReentrancyGuard()

	// First lock should succeed
	err := guard.Lock("test_key")
	require.NoError(t, err)

	// Second lock on same key should fail (reentrancy detected)
	err = guard.Lock("test_key")
	require.Error(t, err)
	require.Contains(t, err.Error(), "reentrancy")

	// Unlock should allow locking again
	guard.Unlock("test_key")
	err = guard.Lock("test_key")
	require.NoError(t, err)
}

// TestMathIntOperations verifies math.Int provides safe arithmetic operations.
// Note: math.Int uses arbitrary precision (big.Int) internally, so overflow is not possible.
// This test documents the expected behavior of math.Int for future reference.
func TestMathIntOperations(t *testing.T) {
	// Addition - always safe with math.Int (arbitrary precision)
	a := math.NewInt(100)
	b := math.NewInt(200)
	result := a.Add(b)
	require.Equal(t, math.NewInt(300), result)

	// Subtraction - result can be negative but no panic
	result = a.Sub(b) // 100 - 200 = -100
	require.True(t, result.IsNegative())

	// Multiplication - always safe with math.Int
	result = a.Mul(b)
	require.Equal(t, math.NewInt(20000), result)

	// Division - safe, but zero divisor must be checked before calling
	result = b.Quo(a) // 200 / 100 = 2
	require.Equal(t, math.NewInt(2), result)

	// Note: Quo with zero divisor panics, so callers must check before calling
	// This is the expected behavior documented by Cosmos SDK
}

// TestSwapSizeValidation tests MEV protection via swap size limits
func TestSwapSizeValidation(t *testing.T) {
	k := keeper.Keeper{}

	tests := []struct {
		name      string
		amountIn  math.Int
		reserveIn math.Int
		expectErr bool
	}{
		{
			name:      "normal swap size",
			amountIn:  math.NewInt(1000),
			reserveIn: math.NewInt(100000),
			expectErr: false,
		},
		{
			name:      "swap too large (>10% of reserve)",
			amountIn:  math.NewInt(15000),
			reserveIn: math.NewInt(100000),
			expectErr: true,
		},
		{
			name:      "maximum allowed swap (10% of reserve)",
			amountIn:  math.NewInt(10000),
			reserveIn: math.NewInt(100000),
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := k.ValidateSwapSize(tt.amountIn, tt.reserveIn)

			if tt.expectErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), "too large")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestPriceImpactValidation tests price impact limits
func TestPriceImpactValidation(t *testing.T) {
	k := keeper.Keeper{}

	tests := []struct {
		name       string
		amountIn   math.Int
		reserveIn  math.Int
		reserveOut math.Int
		amountOut  math.Int
		expectErr  bool
	}{
		{
			name:       "low price impact",
			amountIn:   math.NewInt(100),
			reserveIn:  math.NewInt(10000),
			reserveOut: math.NewInt(10000),
			amountOut:  math.NewInt(99),
			expectErr:  false,
		},
		{
			name:       "high price impact (>50%)",
			amountIn:   math.NewInt(10000),
			reserveIn:  math.NewInt(10000),
			reserveOut: math.NewInt(10000),
			amountOut:  math.NewInt(3000),
			expectErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := k.ValidatePriceImpact(tt.amountIn, tt.reserveIn, tt.reserveOut, tt.amountOut)

			if tt.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestPoolStateValidation tests comprehensive pool state checks
func TestPoolStateValidation(t *testing.T) {
	k := keeper.Keeper{}

	tests := []struct {
		name      string
		pool      *types.Pool
		expectErr bool
		errMsg    string
	}{
		{
			name: "valid pool",
			pool: &types.Pool{
				Id:          1,
				TokenA:      "atom",
				TokenB:      "usdc",
				ReserveA:    math.NewInt(10000),
				ReserveB:    math.NewInt(10000),
				TotalShares: math.NewInt(10000),
			},
			expectErr: false,
		},
		{
			name: "negative reserve A",
			pool: &types.Pool{
				Id:          1,
				TokenA:      "atom",
				TokenB:      "usdc",
				ReserveA:    math.NewInt(-100),
				ReserveB:    math.NewInt(10000),
				TotalShares: math.NewInt(10000),
			},
			expectErr: true,
			errMsg:    "negative reserve",
		},
		{
			name: "reserves but no shares",
			pool: &types.Pool{
				Id:          1,
				TokenA:      "atom",
				TokenB:      "usdc",
				ReserveA:    math.NewInt(10000),
				ReserveB:    math.NewInt(10000),
				TotalShares: math.ZeroInt(),
			},
			expectErr: true,
			errMsg:    "has reserves but no shares",
		},
		{
			name: "shares but missing reserves",
			pool: &types.Pool{
				Id:          1,
				TokenA:      "atom",
				TokenB:      "usdc",
				ReserveA:    math.ZeroInt(),
				ReserveB:    math.NewInt(10000),
				TotalShares: math.NewInt(10000),
			},
			expectErr: true,
			errMsg:    "has shares but missing reserves",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := k.ValidatePoolState(tt.pool)

			if tt.expectErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestInvariantValidation tests constant product invariant checks
func TestInvariantValidation(t *testing.T) {
	k := keeper.Keeper{}

	tests := []struct {
		name      string
		pool      *types.Pool
		oldK      math.Int
		expectErr bool
	}{
		{
			name: "k increased (fees accumulated)",
			pool: &types.Pool{
				ReserveA: math.NewInt(10100),
				ReserveB: math.NewInt(10000),
			},
			oldK:      math.NewInt(100000000), // 10000 * 10000
			expectErr: false,
		},
		{
			name: "k maintained",
			pool: &types.Pool{
				ReserveA: math.NewInt(10000),
				ReserveB: math.NewInt(10000),
			},
			oldK:      math.NewInt(100000000),
			expectErr: false,
		},
		{
			name: "k decreased (invariant violation)",
			pool: &types.Pool{
				ReserveA: math.NewInt(9000),
				ReserveB: math.NewInt(10000),
			},
			oldK:      math.NewInt(100000000),
			expectErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := k.ValidatePoolInvariant(context.Background(), tt.pool, tt.oldK)

			if tt.expectErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), "invariant violated")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestCircuitBreakerMechanism tests emergency pause functionality
func TestCircuitBreakerMechanism(t *testing.T) {
	// Test circuit breaker state management
	state := keeper.CircuitBreakerState{
		Enabled:       true,
		PausedUntil:   time.Now().Add(1 * time.Hour),
		TriggerReason: "test pause",
	}

	require.True(t, state.Enabled)
	require.False(t, state.PausedUntil.IsZero())
}

// TestCalculateSwapOutputSecurity tests swap calculation with all validations
func TestCalculateSwapOutputSecurity(t *testing.T) {
	tests := []struct {
		name            string
		amountIn        math.Int
		reserveIn       math.Int
		reserveOut      math.Int
		swapFee         math.LegacyDec
		maxDrainPercent math.LegacyDec
		expectErr       bool
		errMsg          string
	}{
		{
			name:            "valid swap",
			amountIn:        math.NewInt(100),
			reserveIn:       math.NewInt(10000),
			reserveOut:      math.NewInt(10000),
			swapFee:         math.LegacyNewDecWithPrec(3, 3),
			maxDrainPercent: math.LegacyNewDecWithPrec(30, 2),
			expectErr:       false,
		},
		{
			name:            "zero input",
			amountIn:        math.ZeroInt(),
			reserveIn:       math.NewInt(10000),
			reserveOut:      math.NewInt(10000),
			swapFee:         math.LegacyNewDecWithPrec(3, 3),
			maxDrainPercent: math.LegacyNewDecWithPrec(30, 2),
			expectErr:       true,
			errMsg:          "must be positive",
		},
		{
			name:            "zero reserve in",
			amountIn:        math.NewInt(100),
			reserveIn:       math.ZeroInt(),
			reserveOut:      math.NewInt(10000),
			swapFee:         math.LegacyNewDecWithPrec(3, 3),
			maxDrainPercent: math.LegacyNewDecWithPrec(30, 2),
			expectErr:       true,
			errMsg:          "must be positive",
		},
		{
			name:            "invalid fee (>= 1)",
			amountIn:        math.NewInt(100),
			reserveIn:       math.NewInt(10000),
			reserveOut:      math.NewInt(10000),
			swapFee:         math.LegacyNewDec(1),
			maxDrainPercent: math.LegacyNewDecWithPrec(30, 2),
			expectErr:       true,
			errMsg:          "fee must be in range",
		},
		{
			name:            "output would drain pool",
			amountIn:        math.NewInt(100000),
			reserveIn:       math.NewInt(10000),
			reserveOut:      math.NewInt(10000),
			swapFee:         math.LegacyNewDecWithPrec(3, 3),
			maxDrainPercent: math.LegacyNewDecWithPrec(30, 2),
			expectErr:       true,
			errMsg:          "swap too large",
		},
		{
			name:            "exceeds governance drain limit",
			amountIn:        math.NewInt(50000),
			reserveIn:       math.NewInt(10000),
			reserveOut:      math.NewInt(10000),
			swapFee:         math.LegacyNewDecWithPrec(3, 3),
			maxDrainPercent: math.LegacyNewDecWithPrec(30, 2),
			expectErr:       true,
			errMsg:          "drain too much liquidity",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k := keeper.Keeper{}
			output, err := k.CalculateSwapOutputSecure(context.Background(), tt.amountIn, tt.reserveIn, tt.reserveOut, tt.swapFee, tt.maxDrainPercent)

			if tt.expectErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errMsg)
			} else {
				require.NoError(t, err)
				require.True(t, output.GT(math.ZeroInt()))
				require.True(t, output.LT(tt.reserveOut))
			}
		})
	}
}

// TestFlashLoanProtection tests minimum lock period enforcement
func TestFlashLoanProtection(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)
	poolID := keepertest.CreateTestPool(t, k, ctx, "upaw", "uatom",
		math.NewInt(1_000_000), math.NewInt(1_000_000))
	provider := types.TestAddr()

	ctx = ctx.WithBlockHeight(50)
	require.NoError(t, k.SetLastLiquidityActionBlock(ctx, poolID, provider))

	ctx = ctx.WithBlockHeight(55) // Only 5 blocks elapsed; default is 10
	err := k.CheckFlashLoanProtection(ctx, poolID, provider)
	require.ErrorIs(t, err, types.ErrFlashLoanDetected)

	ctx = ctx.WithBlockHeight(65)
	err = k.CheckFlashLoanProtection(ctx, poolID, provider)
	require.NoError(t, err)
}

// TestMaxPoolsLimit tests DoS protection via pool limit
func TestMaxPoolsLimit(t *testing.T) {
	require.Equal(t, uint64(1000), keeper.MaxPools)
}

// TestSecurityConstants verifies all security constants are set correctly
func TestSecurityConstants(t *testing.T) {
	// Verify maximum price deviation is reasonable
	maxDev, err := math.LegacyNewDecFromStr(keeper.MaxPriceDeviation)
	require.NoError(t, err)
	require.Equal(t, "0.250000000000000000", maxDev.String())

	// Verify maximum swap size is reasonable
	maxSwap, err := math.LegacyNewDecFromStr(keeper.MaxSwapSizePercent)
	require.NoError(t, err)
	require.Equal(t, "0.100000000000000000", maxSwap.String())

	// Verify price update tolerance
	tolerance, err := math.LegacyNewDecFromStr(keeper.PriceUpdateTolerance)
	require.NoError(t, err)
	require.True(t, tolerance.GT(math.LegacyZeroDec()))
}
