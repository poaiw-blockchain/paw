package keeper

// This file exports private keeper methods for testing purposes.
// This is a standard Go testing pattern for white-box testing.

import (
	"context"
	"fmt"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/oracle/types"
)

// Exported for testing: median calculation
func (k Keeper) CalculateMedian(prices []sdkmath.LegacyDec) sdkmath.LegacyDec {
	return k.calculateMedian(prices)
}

// Exported for testing: MAD calculation
func (k Keeper) CalculateMAD(prices []sdkmath.LegacyDec, median sdkmath.LegacyDec) sdkmath.LegacyDec {
	return k.calculateMAD(prices, median)
}

// Exported for testing: IQR calculation
func (k Keeper) CalculateIQR(prices []sdkmath.LegacyDec) (q1, q3, iqr sdkmath.LegacyDec) {
	return k.calculateIQR(prices)
}

// Exported for testing: weighted median calculation
func (k Keeper) CalculateWeightedMedian(validatorPrices []types.ValidatorPrice) (sdkmath.LegacyDec, error) {
	return k.calculateWeightedMedian(validatorPrices)
}

// Exported for testing: outlier severity classification
func (k Keeper) ClassifyOutlierSeverity(price, median, mad, threshold sdkmath.LegacyDec) (OutlierSeverity, sdkmath.LegacyDec) {
	return k.classifyOutlierSeverity(price, median, mad, threshold)
}

// Exported for testing: IQR outlier detection
func (k Keeper) IsIQROutlier(price, q1, q3, iqr, volatility sdkmath.LegacyDec) bool {
	return k.isIQROutlier(price, q1, q3, iqr, volatility)
}

// Exported for testing: Grubbs' test
func (k Keeper) GrubbsTest(prices []sdkmath.LegacyDec, testPrice sdkmath.LegacyDec, alpha float64) bool {
	return k.grubbsTest(prices, testPrice, alpha)
}

// Exported for testing: volatility calculation
func (k Keeper) CalculateVolatility(ctx context.Context, asset string, window int) sdkmath.LegacyDec {
	return k.calculateVolatility(ctx, asset, window)
}

// Exported for testing: outlier detection and filtering
func (k Keeper) DetectAndFilterOutliers(ctx context.Context, asset string, prices []types.ValidatorPrice) (*FilteredPriceData, error) {
	return k.detectAndFilterOutliers(ctx, asset, prices)
}

// Exported for testing: voting power calculation
func (k Keeper) CalculateVotingPower(ctx context.Context, validatorPrices []types.ValidatorPrice) (int64, []types.ValidatorPrice, error) {
	return k.calculateVotingPower(ctx, validatorPrices)
}

// Exported for testing: cached price helpers
func (k Keeper) StoreCachedPrice(ctx sdk.Context, symbol, chainID string, price *CrossChainPriceData) {
	k.storeCachedPrice(ctx, symbol, chainID, price)
}

func (k Keeper) ClearCachedPrice(ctx sdk.Context, symbol, chainID string) {
	store := ctx.KVStore(k.storeKey)
	key := []byte(fmt.Sprintf("cached_price_%s_%s", chainID, symbol))
	store.Delete(key)
}
