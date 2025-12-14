package keeper

import (
	"fmt"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/oracle/types"
)

// RegisterInvariants registers all oracle module invariants
func RegisterInvariants(ir sdk.InvariantRegistry, k Keeper) {
	ir.RegisterRoute(types.ModuleName, "price-validity",
		PriceValidityInvariant(k))
	ir.RegisterRoute(types.ModuleName, "validator-consistency",
		ValidatorConsistencyInvariant(k))
	ir.RegisterRoute(types.ModuleName, "submission-freshness",
		SubmissionFreshnessInvariant(k))
	ir.RegisterRoute(types.ModuleName, "price-aggregation",
		PriceAggregationInvariant(k))
}

// AllInvariants runs all invariants of the oracle module
func AllInvariants(k Keeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		res, stop := PriceValidityInvariant(k)(ctx)
		if stop {
			return res, stop
		}
		res, stop = ValidatorConsistencyInvariant(k)(ctx)
		if stop {
			return res, stop
		}
		res, stop = SubmissionFreshnessInvariant(k)(ctx)
		if stop {
			return res, stop
		}
		return PriceAggregationInvariant(k)(ctx)
	}
}

// PriceValidityInvariant checks that all prices are positive and within reasonable bounds
func PriceValidityInvariant(k Keeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		var (
			broken bool
			msg    string
			issues []string
		)

		store := ctx.KVStore(k.storeKey)
		iter := storetypes.KVStorePrefixIterator(store, PriceKeyPrefix)
		defer iter.Close()

		for ; iter.Valid(); iter.Next() {
			var price types.Price
			if err := k.cdc.Unmarshal(iter.Value(), &price); err != nil {
				broken = true
				issues = append(issues, fmt.Sprintf(
					"error unmarshaling price: %v",
					err,
				))
				continue
			}

			// Check price is positive
			if price.Price.IsZero() || price.Price.IsNegative() {
				broken = true
				issues = append(issues, fmt.Sprintf(
					"asset %s has invalid price: %s",
					price.Asset, price.Price,
				))
			}

			// Check price is not unreasonably large (sanity check)
			// Max price is 1 trillion (10^12)
			maxPrice := math.LegacyNewDec(1_000_000_000_000)
			if price.Price.GT(maxPrice) {
				broken = true
				issues = append(issues, fmt.Sprintf(
					"asset %s has unreasonably large price: %s",
					price.Asset, price.Price,
				))
			}

			// Check asset name is not empty
			if price.Asset == "" {
				broken = true
				issues = append(issues, "found price with empty asset name")
			}

			// Check block time is not in the future if set
			if price.BlockTime > 0 && price.BlockTime > ctx.BlockTime().Unix() {
				broken = true
				issues = append(issues, fmt.Sprintf(
					"asset %s has future update time: %d > %d",
					price.Asset, price.BlockTime, ctx.BlockTime().Unix(),
				))
			}
		}

		if len(issues) > 0 {
			msg = fmt.Sprintf("%d invalid prices:\n", len(issues))
			for _, issue := range issues {
				msg += fmt.Sprintf("  - %s\n", issue)
			}
		}

		return sdk.FormatInvariant(
			types.ModuleName, "price-validity",
			msg,
		), broken
	}
}

// ValidatorConsistencyInvariant checks that validator oracle data is consistent
func ValidatorConsistencyInvariant(k Keeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		var (
			broken bool
			msg    string
			issues []string
		)

		store := ctx.KVStore(k.storeKey)
		iter := storetypes.KVStorePrefixIterator(store, ValidatorOracleKeyPrefix)
		defer iter.Close()

		for ; iter.Valid(); iter.Next() {
			var validatorOracle types.ValidatorOracle
			if err := k.cdc.Unmarshal(iter.Value(), &validatorOracle); err != nil {
				broken = true
				issues = append(issues, fmt.Sprintf(
					"error unmarshaling validator oracle: %v",
					err,
				))
				continue
			}

			// Check validator address is not empty
			if validatorOracle.ValidatorAddr == "" {
				broken = true
				issues = append(issues, "found validator oracle with empty address")
				continue
			}

			// Verify validator exists in staking module
			valAddr, err := sdk.ValAddressFromBech32(validatorOracle.ValidatorAddr)
			if err != nil {
				broken = true
				issues = append(issues, fmt.Sprintf(
					"invalid validator address: %s",
					validatorOracle.ValidatorAddr,
				))
				continue
			}

			if _, err := k.stakingKeeper.GetValidator(ctx, valAddr); err != nil {
				broken = true
				issues = append(issues, fmt.Sprintf(
					"validator %s not found in staking module",
					validatorOracle.ValidatorAddr,
				))
				continue
			}
		}

		if len(issues) > 0 {
			msg = fmt.Sprintf("%d validator consistency issues:\n", len(issues))
			for _, issue := range issues {
				msg += fmt.Sprintf("  - %s\n", issue)
			}
		}

		return sdk.FormatInvariant(
			types.ModuleName, "validator-consistency",
			msg,
		), broken
	}
}

// SubmissionFreshnessInvariant checks that validator price submissions are recent
func SubmissionFreshnessInvariant(k Keeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		var (
			broken bool
			msg    string
			issues []string
		)

		params, err := k.GetParams(ctx)
		if err != nil {
			broken = true
			msg = fmt.Sprintf("error getting params: %v", err)
			return sdk.FormatInvariant(
				types.ModuleName, "submission-freshness",
				msg,
			), broken
		}

		// Check validator price submissions
		store := ctx.KVStore(k.storeKey)
		iter := storetypes.KVStorePrefixIterator(store, ValidatorPriceKeyPrefix)
		defer iter.Close()

		// Use vote period as freshness window (in blocks)
		maxAgeBlocks := int64(params.VotePeriod)

		for ; iter.Valid(); iter.Next() {
			var validatorPrice types.ValidatorPrice
			if err := k.cdc.Unmarshal(iter.Value(), &validatorPrice); err != nil {
				broken = true
				issues = append(issues, fmt.Sprintf(
					"error unmarshaling validator price: %v",
					err,
				))
				continue
			}

			// Check submission is not too old (warning, not critical)
			ageBlocks := ctx.BlockHeight() - validatorPrice.BlockHeight
			if ageBlocks > maxAgeBlocks*2 { // Allow 2x vote period before breaking invariant
				broken = true
				issues = append(issues, fmt.Sprintf(
					"validator %s has stale %s price: age=%d, max=%d",
					validatorPrice.ValidatorAddr, validatorPrice.Asset,
					ageBlocks, maxAgeBlocks*2,
				))
			}

			// Check price is positive
			if validatorPrice.Price.IsZero() || validatorPrice.Price.IsNegative() {
				broken = true
				issues = append(issues, fmt.Sprintf(
					"validator %s has invalid %s price: %s",
					validatorPrice.ValidatorAddr, validatorPrice.Asset,
					validatorPrice.Price,
				))
			}

			// Check block height is not in the future
			if validatorPrice.BlockHeight > ctx.BlockHeight() {
				broken = true
				issues = append(issues, fmt.Sprintf(
					"validator %s has future block height for %s: %d > %d",
					validatorPrice.ValidatorAddr, validatorPrice.Asset,
					validatorPrice.BlockHeight, ctx.BlockHeight(),
				))
			}
		}

		if len(issues) > 0 {
			msg = fmt.Sprintf("%d submission freshness issues:\n", len(issues))
			for _, issue := range issues {
				msg += fmt.Sprintf("  - %s\n", issue)
			}
		}

		return sdk.FormatInvariant(
			types.ModuleName, "submission-freshness",
			msg,
		), broken
	}
}

// PriceAggregationInvariant checks that aggregated prices are consistent with validator submissions
func PriceAggregationInvariant(k Keeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		var (
			broken bool
			msg    string
			issues []string
		)

		store := ctx.KVStore(k.storeKey)
		priceIter := storetypes.KVStorePrefixIterator(store, PriceKeyPrefix)
		defer priceIter.Close()

		for ; priceIter.Valid(); priceIter.Next() {
			var price types.Price
			if err := k.cdc.Unmarshal(priceIter.Value(), &price); err != nil {
				continue
			}

			// Get all validator submissions for this asset
			var submissions []math.LegacyDec
			var totalPower int64

			valIter := storetypes.KVStorePrefixIterator(store, ValidatorPriceKeyPrefix)
			defer valIter.Close()

			for ; valIter.Valid(); valIter.Next() {
				var valPrice types.ValidatorPrice
				if err := k.cdc.Unmarshal(valIter.Value(), &valPrice); err != nil {
					continue
				}

				if valPrice.Asset != price.Asset {
					continue
				}

				// Get validator power
				valAddr, err := sdk.ValAddressFromBech32(valPrice.ValidatorAddr)
				if err != nil {
					continue
				}

				validator, err := k.stakingKeeper.GetValidator(ctx, valAddr)
				if err != nil {
					continue
				}

				power := validator.GetConsensusPower(k.stakingKeeper.PowerReduction(ctx))
				totalPower += power

				submissions = append(submissions, valPrice.Price)
			}

			// If we have submissions, check that aggregated price is reasonable
			if len(submissions) > 0 {
				// Calculate simple median for comparison
				if len(submissions) >= 3 {
					// Sort submissions to find median
					sortedPrices := make([]math.LegacyDec, len(submissions))
					copy(sortedPrices, submissions)

					// Simple bubble sort for small arrays
					for i := 0; i < len(sortedPrices); i++ {
						for j := i + 1; j < len(sortedPrices); j++ {
							if sortedPrices[i].GT(sortedPrices[j]) {
								sortedPrices[i], sortedPrices[j] = sortedPrices[j], sortedPrices[i]
							}
						}
					}

					median := sortedPrices[len(sortedPrices)/2]

					// Aggregated price should be within 50% of median
					diff := price.Price.Sub(median).Abs()
					threshold := median.Mul(math.LegacyNewDecWithPrec(5, 1)) // 50%

					if diff.GT(threshold) {
						broken = true
						issues = append(issues, fmt.Sprintf(
							"asset %s aggregated price %s deviates significantly from median %s (diff: %s)",
							price.Asset, price.Price, median, diff,
						))
					}
				}
			}
		}

		if len(issues) > 0 {
			msg = fmt.Sprintf("%d price aggregation issues:\n", len(issues))
			for _, issue := range issues {
				msg += fmt.Sprintf("  - %s\n", issue)
			}
		}

		return sdk.FormatInvariant(
			types.ModuleName, "price-aggregation",
			msg,
		), broken
	}
}
