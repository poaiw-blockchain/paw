package keeper

import (
	"encoding/json"
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/dex/types"
)

// PriceObservation stores a price observation at a specific time
type PriceObservation struct {
	BlockHeight int64
	Timestamp   int64
	Price       math.LegacyDec // Price of TokenA in terms of TokenB
	ReserveA    math.Int
	ReserveB    math.Int
}

// RecordPrice records the current price for TWAP calculation
func (k Keeper) RecordPrice(ctx sdk.Context, poolId uint64) error {
	pool := k.GetPool(ctx, poolId)
	if pool == nil {
		return types.ErrPoolNotFound
	}

	// Calculate current price (reserveB / reserveA)
	price := math.LegacyNewDecFromInt(pool.ReserveB).Quo(math.LegacyNewDecFromInt(pool.ReserveA))

	// Get or create price observations array
	observations := k.GetPriceObservations(ctx, poolId)

	// Add new observation
	observation := PriceObservation{
		BlockHeight: ctx.BlockHeight(),
		Timestamp:   ctx.BlockTime().Unix(),
		Price:       price,
		ReserveA:    pool.ReserveA,
		ReserveB:    pool.ReserveB,
	}

	observations = append(observations, observation)

	// Keep only last 100 observations (configurable)
	maxObservations := 100
	if len(observations) > maxObservations {
		observations = observations[len(observations)-maxObservations:]
	}

	// Store observations
	k.SetPriceObservations(ctx, poolId, observations)

	return nil
}

// GetTWAP calculates the Time-Weighted Average Price over a time window
// windowSeconds is the time window in seconds (e.g., 3600 for 1 hour)
func (k Keeper) GetTWAP(ctx sdk.Context, poolId uint64, windowSeconds int64) (math.LegacyDec, error) {
	observations := k.GetPriceObservations(ctx, poolId)

	if len(observations) == 0 {
		return math.LegacyZeroDec(), types.ErrPoolNotFound.Wrap("no price observations available")
	}

	currentTime := ctx.BlockTime().Unix()
	startTime := currentTime - windowSeconds

	// Filter observations within the time window
	var relevantObservations []PriceObservation
	for _, obs := range observations {
		if obs.Timestamp >= startTime {
			relevantObservations = append(relevantObservations, obs)
		}
	}

	if len(relevantObservations) == 0 {
		// If no observations in window, use the most recent one
		return observations[len(observations)-1].Price, nil
	}

	// Calculate time-weighted average
	var weightedSum math.LegacyDec
	var totalWeight int64

	for i := 0; i < len(relevantObservations); i++ {
		var timeWeight int64
		if i == len(relevantObservations)-1 {
			// Last observation: weight from its time to current time
			timeWeight = currentTime - relevantObservations[i].Timestamp
		} else {
			// Other observations: weight from its time to next observation time
			timeWeight = relevantObservations[i+1].Timestamp - relevantObservations[i].Timestamp
		}

		if timeWeight > 0 {
			priceWeighted := relevantObservations[i].Price.MulInt64(timeWeight)
			weightedSum = weightedSum.Add(priceWeighted)
			totalWeight += timeWeight
		}
	}

	if totalWeight == 0 {
		return relevantObservations[len(relevantObservations)-1].Price, nil
	}

	twap := weightedSum.QuoInt64(totalWeight)
	return twap, nil
}

// GetPriceObservations retrieves price observations for a pool
func (k Keeper) GetPriceObservations(ctx sdk.Context, poolId uint64) []PriceObservation {
	store := ctx.KVStore(k.storeKey)
	key := types.GetPriceObservationsKey(poolId)
	bz := store.Get(key)

	if bz == nil {
		return []PriceObservation{}
	}

	var observations []PriceObservation
	if err := json.Unmarshal(bz, &observations); err != nil {
		// If unmarshal fails, return empty array
		return []PriceObservation{}
	}
	return observations
}

// SetPriceObservations stores price observations for a pool
func (k Keeper) SetPriceObservations(ctx sdk.Context, poolId uint64, observations []PriceObservation) {
	store := ctx.KVStore(k.storeKey)
	key := types.GetPriceObservationsKey(poolId)
	bz, err := json.Marshal(&observations)
	if err != nil {
		// Should never happen with simple struct
		panic(fmt.Sprintf("failed to marshal price observations: %v", err))
	}
	store.Set(key, bz)
}

// GetSpotPrice returns the current spot price for a pool
func (k Keeper) GetSpotPrice(ctx sdk.Context, poolId uint64) (math.LegacyDec, error) {
	pool := k.GetPool(ctx, poolId)
	if pool == nil {
		return math.LegacyZeroDec(), types.ErrPoolNotFound
	}

	// Spot price = reserveB / reserveA
	if pool.ReserveA.IsZero() {
		return math.LegacyZeroDec(), types.ErrInsufficientLiquidity
	}

	price := math.LegacyNewDecFromInt(pool.ReserveB).Quo(math.LegacyNewDecFromInt(pool.ReserveA))
	return price, nil
}

// CheckPriceDeviation checks if current price deviates too much from TWAP
// This can detect price manipulation attempts
func (k Keeper) CheckPriceDeviation(ctx sdk.Context, poolId uint64, maxDeviationPercent math.LegacyDec) (bool, error) {
	// Get spot price
	spotPrice, err := k.GetSpotPrice(ctx, poolId)
	if err != nil {
		return false, err
	}

	// Get 1-hour TWAP
	twap, err := k.GetTWAP(ctx, poolId, 3600) // 1 hour
	if err != nil {
		return false, err
	}

	if twap.IsZero() {
		return false, nil
	}

	// Calculate deviation percentage
	deviation := spotPrice.Sub(twap).Quo(twap).Abs()

	// Check if deviation exceeds threshold
	exceeds := deviation.GT(maxDeviationPercent)

	if exceeds {
		// Log warning event
		ctx.EventManager().EmitEvent(
			sdk.NewEvent(
				"price_deviation_detected",
				sdk.NewAttribute("pool_id", fmt.Sprintf("%d", poolId)),
				sdk.NewAttribute("spot_price", spotPrice.String()),
				sdk.NewAttribute("twap", twap.String()),
				sdk.NewAttribute("deviation", deviation.String()),
			),
		)
	}

	return exceeds, nil
}

// ValidateSwapAgainstTWAP validates that a swap doesn't cause excessive price impact
func (k Keeper) ValidateSwapAgainstTWAP(ctx sdk.Context, poolId uint64, amountIn, amountOut math.Int, tokenIn, tokenOut string) error {
	// Get TWAP
	twap, err := k.GetTWAP(ctx, poolId, 3600) // 1 hour TWAP
	if err != nil {
		// If TWAP not available, allow swap (initial bootstrap period)
		return nil
	}

	// Calculate implied price from this swap
	impliedPrice := math.LegacyNewDecFromInt(amountOut).Quo(math.LegacyNewDecFromInt(amountIn))

	// Adjust implied price if tokens are reversed
	pool := k.GetPool(ctx, poolId)
	if pool != nil && tokenIn == pool.TokenB {
		impliedPrice = math.LegacyOneDec().Quo(impliedPrice)
	}

	// Calculate deviation from TWAP
	if !twap.IsZero() {
		deviation := impliedPrice.Sub(twap).Quo(twap).Abs()

		// Allow up to 10% deviation from TWAP
		maxDeviation := math.LegacyNewDecWithPrec(10, 2) // 10%

		if deviation.GT(maxDeviation) {
			return types.ErrInvalidSlippage.Wrapf(
				"swap price deviates too much from TWAP: deviation %.2f%%, max allowed 10%%",
				deviation.MulInt64(100).MustFloat64(),
			)
		}
	}

	return nil
}
