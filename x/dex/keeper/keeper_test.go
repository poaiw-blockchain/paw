package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/dex/keeper"
	"github.com/paw-chain/paw/x/dex/types"
)

type KeeperTestSuite struct {
	suite.Suite
	keeper keeper.Keeper
	ctx    sdk.Context
}

func (suite *KeeperTestSuite) SetupTest() {
	suite.keeper, suite.ctx = keepertest.DexKeeper(suite.T())
}

func TestKeeperTestSuite(t *testing.T) {
	suite.Run(t, new(KeeperTestSuite))
}

// TestCreatePool validates pool creation with valid parameters
func TestCreatePool(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	tests := []struct {
		name    string
		msg     *types.MsgCreatePool
		wantErr bool
		errMsg  string
	}{
		{
			name: "valid pool creation",
			msg: &types.MsgCreatePool{
				Creator: "paw1creator",
				TokenA:  "upaw",
				TokenB:  "uusdt",
				AmountA: math.NewInt(1000000),
				AmountB: math.NewInt(1000000),
			},
			wantErr: false,
		},
		{
			name: "same token pool",
			msg: &types.MsgCreatePool{
				Creator: "paw1creator",
				TokenA:  "upaw",
				TokenB:  "upaw",
				AmountA: math.NewInt(1000000),
				AmountB: math.NewInt(1000000),
			},
			wantErr: true,
			errMsg:  "tokens must be different",
		},
		{
			name: "zero amount",
			msg: &types.MsgCreatePool{
				Creator: "paw1creator",
				TokenA:  "upaw",
				TokenB:  "uusdt",
				AmountA: math.NewInt(0),
				AmountB: math.NewInt(1000000),
			},
			wantErr: true,
			errMsg:  "amounts must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resp, err := k.CreatePool(ctx, tt.msg)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					require.Contains(t, err.Error(), tt.errMsg)
				}
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.Greater(t, resp.PoolId, uint64(0))

				// Verify pool exists and has correct initial state
				pool, found := k.GetPool(ctx, resp.PoolId)
				require.True(t, found)
				require.Equal(t, tt.msg.TokenA, pool.TokenA)
				require.Equal(t, tt.msg.TokenB, pool.TokenB)
				require.Equal(t, tt.msg.AmountA, pool.ReserveA)
				require.Equal(t, tt.msg.AmountB, pool.ReserveB)
			}
		})
	}
}

// TestSwap validates AMM swap formula (x * y = k)
func TestSwap(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	// Create pool with 1000 PAW and 2000 USDT (1 PAW = 2 USDT)
	poolId := keepertest.CreateTestPool(t, k, ctx, "upaw", "uusdt",
		math.NewInt(1000000000), math.NewInt(2000000000))

	tests := []struct {
		name           string
		poolId         uint64
		tokenIn        string
		amountIn       sdk.Int
		minAmountOut   sdk.Int
		wantErr        bool
		validateOutput bool
	}{
		{
			name:           "swap PAW for USDT",
			poolId:         poolId,
			tokenIn:        "upaw",
			amountIn:       math.NewInt(100000000), // 100 PAW
			minAmountOut:   math.NewInt(180000000), // expect ~180-190 USDT (with 0.3% fee)
			wantErr:        false,
			validateOutput: true,
		},
		{
			name:           "swap USDT for PAW",
			poolId:         poolId,
			tokenIn:        "uusdt",
			amountIn:       math.NewInt(200000000), // 200 USDT
			minAmountOut:   math.NewInt(90000000),  // expect ~90-95 PAW
			wantErr:        false,
			validateOutput: true,
		},
		{
			name:         "slippage too high",
			poolId:       poolId,
			tokenIn:      "upaw",
			amountIn:     math.NewInt(100000000),
			minAmountOut: math.NewInt(300000000), // unrealistic expectation
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Get pool state before swap
			poolBefore, found := k.GetPool(ctx, tt.poolId)
			require.True(t, found)

			constantProduct := poolBefore.ReserveA.Mul(poolBefore.ReserveB)

			msg := &types.MsgSwap{
				Trader:       "paw1trader",
				PoolId:       tt.poolId,
				TokenIn:      tt.tokenIn,
				AmountIn:     tt.amountIn,
				MinAmountOut: tt.minAmountOut,
			}

			resp, err := k.Swap(ctx, msg)

			if tt.wantErr {
				require.Error(t, err)
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.True(t, resp.AmountOut.GT(sdk.ZeroInt()))
				require.True(t, resp.AmountOut.GTE(tt.minAmountOut))

				if tt.validateOutput {
					// Verify constant product formula (with fee)
					poolAfter, found := k.GetPool(ctx, tt.poolId)
					require.True(t, found)

					newConstantProduct := poolAfter.ReserveA.Mul(poolAfter.ReserveB)
					// After fees, constant product should increase slightly
					require.True(t, newConstantProduct.GTE(constantProduct))
				}
			}
		})
	}
}

// TestAddLiquidity validates liquidity provision
func TestAddLiquidity(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	// Create initial pool
	poolId := keepertest.CreateTestPool(t, k, ctx, "upaw", "uusdt",
		math.NewInt(1000000000), math.NewInt(2000000000))

	tests := []struct {
		name    string
		poolId  uint64
		amountA sdk.Int
		amountB sdk.Int
		wantErr bool
		errMsg  string
	}{
		{
			name:    "add proportional liquidity",
			poolId:  poolId,
			amountA: math.NewInt(100000000), // 100 PAW
			amountB: math.NewInt(200000000), // 200 USDT (maintains 1:2 ratio)
			wantErr: false,
		},
		{
			name:    "add with ratio mismatch",
			poolId:  poolId,
			amountA: math.NewInt(100000000), // 100 PAW
			amountB: math.NewInt(100000000), // 100 USDT (wrong ratio)
			wantErr: true,
			errMsg:  "ratio mismatch",
		},
		{
			name:    "zero liquidity",
			poolId:  poolId,
			amountA: math.NewInt(0),
			amountB: math.NewInt(0),
			wantErr: true,
			errMsg:  "amounts must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			poolBefore, found := k.GetPool(ctx, tt.poolId)
			require.True(t, found)

			msg := &types.MsgAddLiquidity{
				Provider: "paw1provider",
				PoolId:   tt.poolId,
				AmountA:  tt.amountA,
				AmountB:  tt.amountB,
			}

			resp, err := k.AddLiquidity(ctx, msg)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					require.Contains(t, err.Error(), tt.errMsg)
				}
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.True(t, resp.LiquidityTokens.GT(sdk.ZeroInt()))

				// Verify reserves increased
				poolAfter, found := k.GetPool(ctx, tt.poolId)
				require.True(t, found)
				require.Equal(t, poolBefore.ReserveA.Add(tt.amountA), poolAfter.ReserveA)
				require.Equal(t, poolBefore.ReserveB.Add(tt.amountB), poolAfter.ReserveB)
			}
		})
	}
}

// TestRemoveLiquidity validates liquidity withdrawal
func TestRemoveLiquidity(t *testing.T) {
	k, ctx := keepertest.DexKeeper(t)

	// Create pool and add liquidity
	poolId := keepertest.CreateTestPool(t, k, ctx, "upaw", "uusdt",
		math.NewInt(1000000000), math.NewInt(2000000000))

	tests := []struct {
		name            string
		poolId          uint64
		liquidityTokens sdk.Int
		minAmountA      sdk.Int
		minAmountB      sdk.Int
		wantErr         bool
		errMsg          string
	}{
		{
			name:            "remove 10% liquidity",
			poolId:          poolId,
			liquidityTokens: math.NewInt(100000000), // 10% of initial
			minAmountA:      math.NewInt(90000000),  // expect ~100M PAW
			minAmountB:      math.NewInt(180000000), // expect ~200M USDT
			wantErr:         false,
		},
		{
			name:            "insufficient liquidity tokens",
			poolId:          poolId,
			liquidityTokens: math.NewInt(10000000000), // more than exists
			minAmountA:      math.NewInt(0),
			minAmountB:      math.NewInt(0),
			wantErr:         true,
			errMsg:          "insufficient liquidity",
		},
		{
			name:            "zero liquidity",
			poolId:          poolId,
			liquidityTokens: math.NewInt(0),
			minAmountA:      math.NewInt(0),
			minAmountB:      math.NewInt(0),
			wantErr:         true,
			errMsg:          "must be positive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			poolBefore, found := k.GetPool(ctx, tt.poolId)
			require.True(t, found)

			msg := &types.MsgRemoveLiquidity{
				Provider:        "paw1provider",
				PoolId:          tt.poolId,
				LiquidityTokens: tt.liquidityTokens,
				MinAmountA:      tt.minAmountA,
				MinAmountB:      tt.minAmountB,
			}

			resp, err := k.RemoveLiquidity(ctx, msg)

			if tt.wantErr {
				require.Error(t, err)
				if tt.errMsg != "" {
					require.Contains(t, err.Error(), tt.errMsg)
				}
				require.Nil(t, resp)
			} else {
				require.NoError(t, err)
				require.NotNil(t, resp)
				require.True(t, resp.AmountA.GTE(tt.minAmountA))
				require.True(t, resp.AmountB.GTE(tt.minAmountB))

				// Verify reserves decreased proportionally
				poolAfter, found := k.GetPool(ctx, tt.poolId)
				require.True(t, found)
				require.True(t, poolAfter.ReserveA.LT(poolBefore.ReserveA))
				require.True(t, poolAfter.ReserveB.LT(poolBefore.ReserveB))
			}
		})
	}
}
