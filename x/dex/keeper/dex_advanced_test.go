package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/dex/keeper"
	"github.com/paw-chain/paw/x/dex/types"
)

// TestValidatePoolCreation tests pool creation validation
func TestValidatePoolCreation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		tokenA    string
		tokenB    string
		initialA  math.Int
		initialB  math.Int
		setup     func(*keeper.Keeper, sdk.Context)
		expectErr bool
		errMsg    string
	}{
		{
			name:      "valid pool creation",
			tokenA:    "upaw",
			tokenB:    "uatom",
			initialA:  math.NewInt(100_000_000),
			initialB:  math.NewInt(100_000_000),
			expectErr: false,
		},
		{
			name:      "deposit below minimum - tokenA",
			tokenA:    "upaw",
			tokenB:    "uatom",
			initialA:  math.NewInt(50_000_000),
			initialB:  math.NewInt(100_000_000),
			expectErr: true,
			errMsg:    "initial deposit must be at least",
		},
		{
			name:      "deposit below minimum - tokenB",
			tokenA:    "upaw",
			tokenB:    "uatom",
			initialA:  math.NewInt(100_000_000),
			initialB:  math.NewInt(50_000_000),
			expectErr: true,
			errMsg:    "initial deposit must be at least",
		},
		{
			name:      "duplicate pool",
			tokenA:    "upaw",
			tokenB:    "usdc",
			initialA:  math.NewInt(100_000_000),
			initialB:  math.NewInt(100_000_000),
			setup: func(k *keeper.Keeper, ctx sdk.Context) {
				// Create existing pool
				_, err := k.CreatePool(ctx, types.TestAddr(), "upaw", "usdc",
					math.NewInt(100_000_000), math.NewInt(100_000_000))
				require.NoError(t, err)
			},
			expectErr: true,
			errMsg:    "pool already exists",
		},
		{
			name:      "identical tokens",
			tokenA:    "upaw",
			tokenB:    "upaw",
			initialA:  math.NewInt(100_000_000),
			initialB:  math.NewInt(100_000_000),
			expectErr: true,
			errMsg:    "identical tokens",
		},
	}

	for _, tc := range tests {
		tc := tc // capture range variable
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			k, ctx := keepertest.DexKeeper(t)
			creator := types.TestAddr()

			if tc.setup != nil {
				tc.setup(k, ctx)
			}

			err := k.ValidatePoolCreation(ctx, creator, tc.tokenA, tc.tokenB, tc.initialA, tc.initialB)

			if tc.expectErr {
				require.Error(t, err)
				if tc.errMsg != "" {
					require.Contains(t, err.Error(), tc.errMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestValidatePriceImpact tests price impact validation for MEV protection
func TestValidatePriceImpact(t *testing.T) {
	t.Parallel()

	k, _ := keepertest.DexKeeper(t)

	tests := []struct {
		name       string
		amountIn   math.Int
		reserveIn  math.Int
		reserveOut math.Int
		amountOut  math.Int
		expectErr  bool
	}{
		{
			name:       "small swap - acceptable impact",
			amountIn:   math.NewInt(100),
			reserveIn:  math.NewInt(10000),
			reserveOut: math.NewInt(10000),
			amountOut:  math.NewInt(99),
			expectErr:  false,
		},
		{
			name:       "medium swap - acceptable impact",
			amountIn:   math.NewInt(500),
			reserveIn:  math.NewInt(10000),
			reserveOut: math.NewInt(10000),
			amountOut:  math.NewInt(476),
			expectErr:  false,
		},
		{
			name:       "large swap - high impact",
			amountIn:   math.NewInt(5000),
			reserveIn:  math.NewInt(10000),
			reserveOut: math.NewInt(10000),
			amountOut:  math.NewInt(2000), // 60% slippage
			expectErr:  true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := k.ValidatePriceImpact(tc.amountIn, tc.reserveIn, tc.reserveOut, tc.amountOut)

			if tc.expectErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), "price impact")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestValidateSwapSize tests swap size validation for MEV protection
func TestValidateSwapSize(t *testing.T) {
	t.Parallel()

	k, _ := keepertest.DexKeeper(t)

	tests := []struct {
		name      string
		amountIn  math.Int
		reserveIn math.Int
		expectErr bool
	}{
		{
			name:      "small swap - 1%",
			amountIn:  math.NewInt(100),
			reserveIn: math.NewInt(10000),
			expectErr: false,
		},
		{
			name:      "medium swap - 5%",
			amountIn:  math.NewInt(500),
			reserveIn: math.NewInt(10000),
			expectErr: false,
		},
		{
			name:      "large swap - 10%",
			amountIn:  math.NewInt(1000),
			reserveIn: math.NewInt(10000),
			expectErr: false,
		},
		{
			name:      "excessive swap - 15%",
			amountIn:  math.NewInt(1500),
			reserveIn: math.NewInt(10000),
			expectErr: true,
		},
		{
			name:      "huge swap - 50%",
			amountIn:  math.NewInt(5000),
			reserveIn: math.NewInt(10000),
			expectErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			err := k.ValidateSwapSize(tc.amountIn, tc.reserveIn)

			if tc.expectErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), "swap size")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestDetectJITLiquidity tests JIT (just-in-time) liquidity detection
func TestDetectJITLiquidity(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		shares    math.Int
		expectErr bool
	}{
		{
			name:      "first liquidity action - no detection",
			shares:    math.NewInt(1000000),
			expectErr: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			k, ctx := keepertest.DexKeeper(t)

			// Create test pool first
			poolID := keepertest.CreateTestPool(t, k, ctx, "upaw", "uusdt",
				math.NewInt(1000000), math.NewInt(1000000))

			provider := sdk.AccAddress([]byte("provider1__________"))

			err := k.DetectJITLiquidity(ctx, poolID, provider, tc.shares)

			if tc.expectErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), "just-in-time liquidity")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestDetectSandwichAttack tests sandwich attack detection
func TestDetectSandwichAttack(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		amountIn  math.Int
		expectErr bool
	}{
		{
			name:      "normal swap - no detection",
			amountIn:  math.NewInt(10000),
			expectErr: false,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			k, ctx := keepertest.DexKeeper(t)

			// Create test pool first
			poolID := keepertest.CreateTestPool(t, k, ctx, "upaw", "uatom",
				math.NewInt(10_000_000), math.NewInt(10_000_000))

			trader := sdk.AccAddress([]byte("trader1____________"))

			err := k.DetectSandwichAttack(ctx, poolID, trader, tc.amountIn)

			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestGetPoolFeeTier tests fee tier retrieval
func TestGetPoolFeeTier(t *testing.T) {
	t.Parallel()

	k, ctx := keepertest.DexKeeper(t)

	// Create pool first
	poolID := keepertest.CreateTestPool(t, k, ctx, "upaw", "uatom",
		math.NewInt(1000000), math.NewInt(1000000))

	tests := []struct {
		name      string
		poolID    uint64
		expectErr bool
	}{
		{
			name:      "existing pool - returns default tier",
			poolID:    poolID,
			expectErr: false,
		},
		{
			name:      "non-existent pool - returns default tier",
			poolID:    999,
			expectErr: false, // GetPoolFeeTier returns default tier even for non-existent pools
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			feeTier, err := k.GetPoolFeeTier(ctx, tc.poolID)

			if tc.expectErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				require.NotNil(t, feeTier)
				require.Equal(t, "standard", feeTier.Name)
			}
		})
	}
}

// TestSetPoolFeeTier tests setting fee tiers
func TestSetPoolFeeTier(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		tier      string
		expectErr bool
	}{
		{
			name:      "set to standard tier",
			tier:      "standard",
			expectErr: false,
		},
		{
			name:      "set to low tier",
			tier:      "low",
			expectErr: false,
		},
		{
			name:      "set to high tier",
			tier:      "high",
			expectErr: false,
		},
		{
			name:      "invalid tier",
			tier:      "invalid",
			expectErr: true,
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			k, ctx := keepertest.DexKeeper(t)

			// Create pool first
			poolID := keepertest.CreateTestPool(t, k, ctx, "upaw", "uatom",
				math.NewInt(1000000), math.NewInt(1000000))

			err := k.SetPoolFeeTier(ctx, poolID, tc.tier)

			if tc.expectErr {
				require.Error(t, err)
				require.Contains(t, err.Error(), "invalid fee tier")
			} else {
				require.NoError(t, err)

				// Verify tier was set
				feeTier, err := k.GetPoolFeeTier(ctx, poolID)
				require.NoError(t, err)
				require.Equal(t, tc.tier, feeTier.Name)
			}
		})
	}
}

// TestFeeTierValues tests that fee tier values are correct
func TestFeeTierValues(t *testing.T) {
	t.Parallel()

	k, ctx := keepertest.DexKeeper(t)

	// Create pool
	poolID := keepertest.CreateTestPool(t, k, ctx, "upaw", "uatom",
		math.NewInt(1000000), math.NewInt(1000000))

	tests := []struct {
		name         string
		tier         string
		expectedFee  math.LegacyDec
	}{
		{
			name:        "standard tier - 0.3%",
			tier:        "standard",
			expectedFee: math.LegacyNewDecWithPrec(3, 3),
		},
		{
			name:        "low tier - 0.05%",
			tier:        "low",
			expectedFee: math.LegacyNewDecWithPrec(5, 4),
		},
		{
			name:        "high tier - 1%",
			tier:        "high",
			expectedFee: math.LegacyNewDecWithPrec(1, 2),
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			err := k.SetPoolFeeTier(ctx, poolID, tc.tier)
			require.NoError(t, err)

			feeTier, err := k.GetPoolFeeTier(ctx, poolID)
			require.NoError(t, err)
			require.True(t, tc.expectedFee.Equal(feeTier.SwapFee),
				"expected swap fee %s, got %s", tc.expectedFee, feeTier.SwapFee)
		})
	}
}

// TestCheckFlashLoanProtection tests flash loan protection
func TestCheckFlashLoanProtection(t *testing.T) {
	t.Parallel()

	k, ctx := keepertest.DexKeeper(t)

	// Create pool
	poolID := keepertest.CreateTestPool(t, k, ctx, "upaw", "uatom",
		math.NewInt(1000000), math.NewInt(1000000))

	provider := sdk.AccAddress([]byte("provider1__________"))

	// First action should be allowed
	err := k.CheckFlashLoanProtection(ctx, poolID, provider)
	require.NoError(t, err)
}

// TestValidateTokenDenom tests token denomination validation via ValidatePoolCreation
func TestValidateTokenDenom(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		tokenA    string
		tokenB    string
		expectErr bool
		errMsg    string
	}{
		{
			name:      "valid cosmos denom",
			tokenA:    "upaw",
			tokenB:    "uatom",
			expectErr: false,
		},
		{
			name:      "valid ibc denom",
			tokenA:    "upaw",
			tokenB:    "ibc/27394FB092D2ECCD56123C74F36E4C1F926001CEADA9CA97EA622B25F41E5EB2",
			expectErr: false,
		},
		{
			name:      "empty denom",
			tokenA:    "",
			tokenB:    "uatom",
			expectErr: true,
			errMsg:    "invalid token A",
		},
	}

	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			k, ctx := keepertest.DexKeeper(t)
			creator := types.TestAddr()

			err := k.ValidatePoolCreation(ctx, creator, tc.tokenA, tc.tokenB,
				math.NewInt(100_000_000), math.NewInt(100_000_000))

			if tc.expectErr {
				require.Error(t, err)
				if tc.errMsg != "" {
					require.Contains(t, err.Error(), tc.errMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestPoolCreationCooldown tests that cooldown between pool creations is enforced
func TestPoolCreationCooldown(t *testing.T) {
	t.Parallel()

	k, ctx := keepertest.DexKeeper(t)
	creator := types.TestAddr()

	// First pool creation should succeed
	err := k.ValidatePoolCreation(ctx, creator, "upaw", "uatom",
		math.NewInt(100_000_000), math.NewInt(100_000_000))
	require.NoError(t, err)

	// Immediate second creation should fail due to cooldown
	err = k.ValidatePoolCreation(ctx, creator, "upaw", "osmo",
		math.NewInt(100_000_000), math.NewInt(100_000_000))
	require.Error(t, err)
	require.Contains(t, err.Error(), "must wait")
}

// TestMinPoolCreationDeposit tests minimum deposit requirement
func TestMinPoolCreationDeposit(t *testing.T) {
	t.Parallel()

	k, ctx := keepertest.DexKeeper(t)
	creator := types.TestAddr()

	// Test below minimum deposit for tokenA
	err := k.ValidatePoolCreation(ctx, creator, "upaw", "uatom",
		math.NewInt(1), math.NewInt(100_000_000))
	require.Error(t, err)
	require.Contains(t, err.Error(), "initial deposit must be at least")

	// Test below minimum deposit for tokenB
	err = k.ValidatePoolCreation(ctx, creator, "upaw", "uatom",
		math.NewInt(100_000_000), math.NewInt(1))
	require.Error(t, err)
	require.Contains(t, err.Error(), "initial deposit must be at least")
}
