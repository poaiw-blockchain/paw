package keeper

import (
	"context"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/paw-chain/paw/x/dex/types"
)

// Task 124: Price Oracle Integration for Pool Valuation

// OracleKeeper defines the interface for oracle module integration
type OracleKeeper interface {
	GetPrice(ctx context.Context, denom string) (math.LegacyDec, error)
	GetPriceWithTimestamp(ctx context.Context, denom string) (math.LegacyDec, int64, error)
}

// GetPoolValueUSD calculates the total USD value of a pool using oracle prices
func (k Keeper) GetPoolValueUSD(ctx context.Context, poolID uint64, oracleKeeper OracleKeeper) (math.LegacyDec, error) {
	// Get pool
	pool, err := k.GetPool(ctx, poolID)
	if err != nil {
		return math.LegacyZeroDec(), err
	}

	// Get price for token A
	priceA, err := oracleKeeper.GetPrice(ctx, pool.TokenA)
	if err != nil {
		return math.LegacyZeroDec(), types.ErrOraclePrice.Wrapf("failed to get price for %s: %v", pool.TokenA, err)
	}

	// Get price for token B
	priceB, err := oracleKeeper.GetPrice(ctx, pool.TokenB)
	if err != nil {
		return math.LegacyZeroDec(), types.ErrOraclePrice.Wrapf("failed to get price for %s: %v", pool.TokenB, err)
	}

	// Calculate USD values
	valueA := math.LegacyNewDecFromInt(pool.ReserveA).Mul(priceA)
	valueB := math.LegacyNewDecFromInt(pool.ReserveB).Mul(priceB)

	totalValue := valueA.Add(valueB)

	return totalValue, nil
}

// ValidatePoolPrice validates pool price against oracle price (arbitrage detection)
func (k Keeper) ValidatePoolPrice(ctx context.Context, poolID uint64, oracleKeeper OracleKeeper, maxDeviationPercent math.LegacyDec) error {
	// Get pool
	pool, err := k.GetPool(ctx, poolID)
	if err != nil {
		return err
	}

	// Get oracle prices
	priceA, timestampA, err := oracleKeeper.GetPriceWithTimestamp(ctx, pool.TokenA)
	if err != nil {
		return types.ErrOraclePrice.Wrapf("failed to get price for %s: %v", pool.TokenA, err)
	}

	priceB, timestampB, err := oracleKeeper.GetPriceWithTimestamp(ctx, pool.TokenB)
	if err != nil {
		return types.ErrOraclePrice.Wrapf("failed to get price for %s: %v", pool.TokenB, err)
	}

	// Check price freshness (within last 60 seconds)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	currentTime := sdkCtx.BlockTime().Unix()

	if currentTime-timestampA > 60 || currentTime-timestampB > 60 {
		return types.ErrOraclePrice.Wrap("oracle prices are stale")
	}

	// DIVISION BY ZERO PROTECTION: Validate oracle prices before division
	if priceB.IsZero() {
		return types.ErrOraclePrice.Wrap("oracle price for token B is zero")
	}

	// DIVISION BY ZERO PROTECTION: Validate pool reserves before division
	if pool.ReserveA.IsZero() || pool.ReserveB.IsZero() {
		return types.ErrInsufficientLiquidity.Wrap("pool has zero reserves")
	}

	// Calculate oracle-based price ratio
	oracleRatio := priceA.Quo(priceB)

	// Calculate pool-based price ratio
	poolRatio := math.LegacyNewDecFromInt(pool.ReserveA).Quo(math.LegacyNewDecFromInt(pool.ReserveB))

	// Calculate deviation
	var deviation math.LegacyDec
	if oracleRatio.GT(poolRatio) {
		// DIVISION BY ZERO PROTECTION: Check oracleRatio is non-zero
		if oracleRatio.IsZero() {
			return types.ErrOraclePrice.Wrap("oracle ratio is zero")
		}
		deviation = oracleRatio.Sub(poolRatio).Quo(oracleRatio)
	} else {
		// DIVISION BY ZERO PROTECTION: Check poolRatio is non-zero
		if poolRatio.IsZero() {
			return types.ErrInvalidPoolState.Wrap("pool ratio is zero")
		}
		deviation = poolRatio.Sub(oracleRatio).Quo(poolRatio)
	}

	// Check if deviation exceeds threshold
	if deviation.GT(maxDeviationPercent) {
		return types.ErrPriceDeviation.Wrapf(
			"pool price deviates %s%% from oracle price, exceeds max %s%%",
			deviation.Mul(math.LegacyNewDec(100)),
			maxDeviationPercent.Mul(math.LegacyNewDec(100)),
		)
	}

	return nil
}

// GetFairPoolPrice returns the oracle-based fair price for the pool
func (k Keeper) GetFairPoolPrice(ctx context.Context, poolID uint64, oracleKeeper OracleKeeper) (math.LegacyDec, error) {
	// Get pool
	pool, err := k.GetPool(ctx, poolID)
	if err != nil {
		return math.LegacyZeroDec(), err
	}

	// Get oracle prices
	priceA, err := oracleKeeper.GetPrice(ctx, pool.TokenA)
	if err != nil {
		return math.LegacyZeroDec(), types.ErrOraclePrice.Wrapf("failed to get price for %s: %v", pool.TokenA, err)
	}

	priceB, err := oracleKeeper.GetPrice(ctx, pool.TokenB)
	if err != nil {
		return math.LegacyZeroDec(), types.ErrOraclePrice.Wrapf("failed to get price for %s: %v", pool.TokenB, err)
	}

	// DIVISION BY ZERO PROTECTION: Validate price before division
	if priceB.IsZero() {
		return math.LegacyZeroDec(), types.ErrOraclePrice.Wrap("oracle price for token B is zero")
	}

	// Return ratio of token A to token B
	return priceA.Quo(priceB), nil
}

// GetLPTokenValueUSD calculates the USD value of LP tokens
func (k Keeper) GetLPTokenValueUSD(ctx context.Context, poolID uint64, shares math.Int, oracleKeeper OracleKeeper) (math.LegacyDec, error) {
	// Get pool
	pool, err := k.GetPool(ctx, poolID)
	if err != nil {
		return math.LegacyZeroDec(), err
	}

	if pool.TotalShares.IsZero() {
		return math.LegacyZeroDec(), types.ErrInvalidPoolState.Wrap("pool has zero total shares")
	}

	// Calculate total pool value
	totalValue, err := k.GetPoolValueUSD(ctx, poolID, oracleKeeper)
	if err != nil {
		return math.LegacyZeroDec(), err
	}

	// Calculate share of total value
	sharePercentage := math.LegacyNewDecFromInt(shares).Quo(math.LegacyNewDecFromInt(pool.TotalShares))
	lpValue := totalValue.Mul(sharePercentage)

	return lpValue, nil
}

// DetectArbitrageOpportunity detects if there's an arbitrage opportunity
func (k Keeper) DetectArbitrageOpportunity(ctx context.Context, poolID uint64, oracleKeeper OracleKeeper, minProfitPercent math.LegacyDec) (bool, math.LegacyDec, error) {
	// Get pool
	pool, err := k.GetPool(ctx, poolID)
	if err != nil {
		return false, math.LegacyZeroDec(), err
	}

	// Get oracle-based fair price
	fairPrice, err := k.GetFairPoolPrice(ctx, poolID, oracleKeeper)
	if err != nil {
		return false, math.LegacyZeroDec(), err
	}

	// DIVISION BY ZERO PROTECTION: Validate pool reserves before division
	if pool.ReserveA.IsZero() || pool.ReserveB.IsZero() {
		return false, math.LegacyZeroDec(), types.ErrInsufficientLiquidity.Wrap("pool has zero reserves")
	}

	// Get current pool price
	poolPrice := math.LegacyNewDecFromInt(pool.ReserveA).Quo(math.LegacyNewDecFromInt(pool.ReserveB))

	// Calculate potential profit percentage
	profitPercent := math.LegacyZeroDec()
	if fairPrice.GT(poolPrice) {
		// DIVISION BY ZERO PROTECTION: Check poolPrice is non-zero
		if poolPrice.IsZero() {
			return false, math.LegacyZeroDec(), types.ErrInvalidPoolState.Wrap("pool price is zero")
		}
		// Arbitrage: buy token A from pool, sell at oracle price
		profitPercent = fairPrice.Sub(poolPrice).Quo(poolPrice)
	} else if poolPrice.GT(fairPrice) {
		// DIVISION BY ZERO PROTECTION: Check fairPrice is non-zero
		if fairPrice.IsZero() {
			return false, math.LegacyZeroDec(), types.ErrOraclePrice.Wrap("fair price is zero")
		}
		// Arbitrage: buy token A at oracle price, sell to pool
		profitPercent = poolPrice.Sub(fairPrice).Quo(fairPrice)
	}
	// If fairPrice equals poolPrice, profitPercent remains zero

	hasOpportunity := profitPercent.GT(minProfitPercent)

	return hasOpportunity, profitPercent, nil
}

// ValidateSwapWithOracle validates a swap against oracle prices
func (k Keeper) ValidateSwapWithOracle(ctx context.Context, poolID uint64, tokenIn, tokenOut string, amountIn, expectedOut math.Int, oracleKeeper OracleKeeper) error {
	// Get oracle prices
	priceIn, err := oracleKeeper.GetPrice(ctx, tokenIn)
	if err != nil {
		return types.ErrOraclePrice.Wrapf("failed to get price for %s: %v", tokenIn, err)
	}

	priceOut, err := oracleKeeper.GetPrice(ctx, tokenOut)
	if err != nil {
		return types.ErrOraclePrice.Wrapf("failed to get price for %s: %v", tokenOut, err)
	}

	// Calculate expected output based on oracle prices
	valueIn := math.LegacyNewDecFromInt(amountIn).Mul(priceIn)
	oracleExpectedOut := valueIn.Quo(priceOut).TruncateInt()

	// Allow up to 5% deviation from oracle price (accounting for fees and slippage)
	maxDeviation := math.LegacyNewDecWithPrec(5, 2)
	tolerance := math.LegacyNewDecFromInt(oracleExpectedOut).Mul(maxDeviation).TruncateInt()

	// Calculate acceptable range with underflow protection
	var minAcceptable math.Int
	if oracleExpectedOut.GT(tolerance) {
		minAcceptable = oracleExpectedOut.Sub(tolerance)
	} else {
		minAcceptable = math.ZeroInt()
	}

	maxAcceptable := oracleExpectedOut.Add(tolerance)

	// Check if expected output is within acceptable range
	if expectedOut.LT(minAcceptable) || expectedOut.GT(maxAcceptable) {
		return types.ErrPriceDeviation.Wrapf(
			"swap output %s deviates from oracle-expected %s (range: %s to %s)",
			expectedOut, oracleExpectedOut, minAcceptable, maxAcceptable,
		)
	}

	return nil
}
