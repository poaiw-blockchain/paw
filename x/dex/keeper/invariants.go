package keeper

import (
	"fmt"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/dex/types"
)

// RegisterInvariants registers all DEX module invariants
func RegisterInvariants(ir sdk.InvariantRegistry, k Keeper) {
	ir.RegisterRoute(types.ModuleName, "pool-reserves",
		PoolReservesInvariant(k))
	ir.RegisterRoute(types.ModuleName, "liquidity-shares",
		LiquiditySharesInvariant(k))
	ir.RegisterRoute(types.ModuleName, "module-balance",
		ModuleBalanceInvariant(k))
	ir.RegisterRoute(types.ModuleName, "constant-product",
		ConstantProductInvariant(k))
}

// AllInvariants runs all invariants of the DEX module
func AllInvariants(k Keeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		res, stop := PoolReservesInvariant(k)(ctx)
		if stop {
			return res, stop
		}
		res, stop = LiquiditySharesInvariant(k)(ctx)
		if stop {
			return res, stop
		}
		res, stop = ModuleBalanceInvariant(k)(ctx)
		if stop {
			return res, stop
		}
		return ConstantProductInvariant(k)(ctx)
	}
}

// PoolReservesInvariant checks that all pools have positive reserves
func PoolReservesInvariant(k Keeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		var (
			broken bool
			msg    string
			issues []string
		)

		pools, err := k.GetAllPools(ctx)
		if err != nil {
			broken = true
			msg = fmt.Sprintf("error getting pools: %v", err)
			return sdk.FormatInvariant(
				types.ModuleName, "pool-reserves",
				msg,
			), broken
		}

		for _, pool := range pools {
			// Check reserves are positive
			if pool.ReserveA.IsZero() || pool.ReserveA.IsNegative() {
				broken = true
				issues = append(issues, fmt.Sprintf(
					"pool %d has invalid reserve A: %s",
					pool.Id, pool.ReserveA,
				))
			}

			if pool.ReserveB.IsZero() || pool.ReserveB.IsNegative() {
				broken = true
				issues = append(issues, fmt.Sprintf(
					"pool %d has invalid reserve B: %s",
					pool.Id, pool.ReserveB,
				))
			}

			// Check total shares are positive
			if pool.TotalShares.IsZero() || pool.TotalShares.IsNegative() {
				broken = true
				issues = append(issues, fmt.Sprintf(
					"pool %d has invalid total shares: %s",
					pool.Id, pool.TotalShares,
				))
			}

			// Check token denoms are not empty
			if pool.TokenA == "" || pool.TokenB == "" {
				broken = true
				issues = append(issues, fmt.Sprintf(
					"pool %d has empty token denoms",
					pool.Id,
				))
			}

			// Check tokens are ordered
			if pool.TokenA > pool.TokenB {
				broken = true
				issues = append(issues, fmt.Sprintf(
					"pool %d has incorrectly ordered tokens: %s > %s",
					pool.Id, pool.TokenA, pool.TokenB,
				))
			}
		}

		if len(issues) > 0 {
			msg = fmt.Sprintf("%d invalid pool reserves:\n", len(issues))
			for _, issue := range issues {
				msg += fmt.Sprintf("  - %s\n", issue)
			}
		}

		return sdk.FormatInvariant(
			types.ModuleName, "pool-reserves",
			msg,
		), broken
	}
}

// LiquiditySharesInvariant checks that the sum of all liquidity shares equals pool total shares
func LiquiditySharesInvariant(k Keeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		var (
			broken bool
			msg    string
			issues []string
		)

		pools, err := k.GetAllPools(ctx)
		if err != nil {
			broken = true
			msg = fmt.Sprintf("error getting pools: %v", err)
			return sdk.FormatInvariant(
				types.ModuleName, "liquidity-shares",
				msg,
			), broken
		}

		for _, pool := range pools {
			// Sum all liquidity shares for this pool
			totalShares := math.ZeroInt()

			store := k.getStore(ctx)
			iter := storetypes.KVStorePrefixIterator(store, LiquidityKeyPrefix)
			defer iter.Close()

			for ; iter.Valid(); iter.Next() {
				// Check if this share belongs to the current pool
				key := iter.Key()
				if len(key) < 10 { // prefix(2) + poolID(8) - keys are namespaced with 2-byte prefix
					continue
				}

				poolID := sdk.BigEndianToUint64(key[2:10])
				if poolID != pool.Id {
					continue
				}

				var shares math.Int
				if err := shares.Unmarshal(iter.Value()); err != nil {
					broken = true
					issues = append(issues, fmt.Sprintf(
						"pool %d: error unmarshaling shares: %v",
						pool.Id, err,
					))
					continue
				}

				totalShares = totalShares.Add(shares)
			}

			// Check if sum matches pool total shares
			if !totalShares.Equal(pool.TotalShares) {
				broken = true
				issues = append(issues, fmt.Sprintf(
					"pool %d: sum of liquidity shares (%s) != pool total shares (%s), difference: %s",
					pool.Id, totalShares, pool.TotalShares, totalShares.Sub(pool.TotalShares),
				))
			}
		}

		if len(issues) > 0 {
			msg = fmt.Sprintf("%d liquidity share mismatches:\n", len(issues))
			for _, issue := range issues {
				msg += fmt.Sprintf("  - %s\n", issue)
			}
		}

		return sdk.FormatInvariant(
			types.ModuleName, "liquidity-shares",
			msg,
		), broken
	}
}

// ModuleBalanceInvariant checks that module account holds all pool reserves
func ModuleBalanceInvariant(k Keeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		var (
			broken bool
			msg    string
		)

		// Get module account balance
		moduleAddr := sdk.AccAddress([]byte(types.ModuleName))
		moduleBalances := k.bankKeeper.GetAllBalances(ctx, moduleAddr)

		// Calculate total required reserves from all pools
		requiredBalances := sdk.NewCoins()

		pools, err := k.GetAllPools(ctx)
		if err != nil {
			broken = true
			msg = fmt.Sprintf("error getting pools: %v", err)
			return sdk.FormatInvariant(
				types.ModuleName, "module-balance",
				msg,
			), broken
		}

		for _, pool := range pools {
			// Add reserves for token A
			coinA := sdk.NewCoin(pool.TokenA, pool.ReserveA)
			requiredBalances = requiredBalances.Add(coinA)

			// Add reserves for token B
			coinB := sdk.NewCoin(pool.TokenB, pool.ReserveB)
			requiredBalances = requiredBalances.Add(coinB)
		}

		// Check if module has sufficient balance
		for _, required := range requiredBalances {
			moduleBalance := moduleBalances.AmountOf(required.Denom)
			if moduleBalance.LT(required.Amount) {
				broken = true
				msg += fmt.Sprintf(
					"insufficient module balance for %s: has %s, needs %s\n",
					required.Denom, moduleBalance, required.Amount,
				)
			}
		}

		return sdk.FormatInvariant(
			types.ModuleName, "module-balance",
			msg,
		), broken
	}
}

// ConstantProductInvariant checks that the constant product formula k = x * y is maintained
// Note: This invariant allows for slight increases in k due to fees, but never decreases
func ConstantProductInvariant(k Keeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		var (
			broken bool
			msg    string
			issues []string
		)

		pools, err := k.GetAllPools(ctx)
		if err != nil {
			broken = true
			msg = fmt.Sprintf("error getting pools: %v", err)
			return sdk.FormatInvariant(
				types.ModuleName, "constant-product",
				msg,
			), broken
		}

		for _, pool := range pools {
			// Calculate k = x * y
			// We check that reserves maintain a reasonable relationship to shares
			// k should be approximately equal to (totalShares)^2

			// Calculate actual k
			product := pool.ReserveA.Mul(pool.ReserveB)

			// Calculate expected k from shares
			expectedK := pool.TotalShares.Mul(pool.TotalShares)

			// The actual product should be close to expected (within 10% for fees)
			// Convert to Dec for comparison
			productDec := math.LegacyNewDecFromInt(product)
			expectedDec := math.LegacyNewDecFromInt(expectedK)

			if expectedDec.IsZero() {
				continue // Skip empty pools
			}

			// Calculate ratio
			ratio := productDec.Quo(expectedDec)

			// Ratio should be between 0.999 and 1.1
			// k should NEVER decrease (only increase from fees)
			// - minRatio: 99.9% prevents fund extraction through precision manipulation
			// - maxRatio: 110% allows reasonable fee accumulation
			minRatio := math.LegacyNewDecWithPrec(999, 3) // 0.999 (99.9%)
			maxRatio := math.LegacyNewDecWithPrec(11, 1)  // 1.1 (110%)

			if ratio.LT(minRatio) || ratio.GT(maxRatio) {
				broken = true
				issues = append(issues, fmt.Sprintf(
					"pool %d has invalid constant product: k=%s, shares^2=%s, ratio=%s",
					pool.Id, product, expectedK, ratio,
				))
			}
		}

		if len(issues) > 0 {
			msg = fmt.Sprintf("%d constant product violations:\n", len(issues))
			for _, issue := range issues {
				msg += fmt.Sprintf("  - %s\n", issue)
			}
		}

		return sdk.FormatInvariant(
			types.ModuleName, "constant-product",
			msg,
		), broken
	}
}
