package keeper

import (
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/dex/types"
)

// RegisterInvariants registers all DEX invariants
func RegisterInvariants(ir sdk.InvariantRegistry, k Keeper) {
	ir.RegisterRoute(types.ModuleName, "pool-reserves", PoolReservesInvariant(k))
	ir.RegisterRoute(types.ModuleName, "pool-shares", PoolSharesInvariant(k))
	ir.RegisterRoute(types.ModuleName, "positive-reserves", PositiveReservesInvariant(k))
	ir.RegisterRoute(types.ModuleName, "module-account-balance", ModuleAccountBalanceInvariant(k))
	ir.RegisterRoute(types.ModuleName, "constant-product", ConstantProductInvariant(k))
}

// AllInvariants runs all invariants of the DEX module
func AllInvariants(k Keeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		res, stop := PoolReservesInvariant(k)(ctx)
		if stop {
			return res, stop
		}

		res, stop = PoolSharesInvariant(k)(ctx)
		if stop {
			return res, stop
		}

		res, stop = PositiveReservesInvariant(k)(ctx)
		if stop {
			return res, stop
		}

		res, stop = ModuleAccountBalanceInvariant(k)(ctx)
		if stop {
			return res, stop
		}

		return ConstantProductInvariant(k)(ctx)
	}
}

// PoolReservesInvariant checks that pool reserves match module account balances
func PoolReservesInvariant(k Keeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		var (
			msg   string
			count int
		)

		pools := k.GetAllPools(ctx)
		for _, pool := range pools {
			// Get module account balance for each token
			moduleAddr := k.GetModuleAddress()
			balanceA := k.bankKeeper.GetBalance(ctx, moduleAddr, pool.TokenA)
			balanceB := k.bankKeeper.GetBalance(ctx, moduleAddr, pool.TokenB)

			// Check if module has sufficient balance for pool reserves
			// Note: Multiple pools can use the same token, so we check >= not ==
			if balanceA.Amount.LT(pool.ReserveA) {
				count++
				msg += fmt.Sprintf("pool %d: module balance for %s (%s) < reserve (%s)\n",
					pool.Id, pool.TokenA, balanceA.Amount.String(), pool.ReserveA.String())
			}

			if balanceB.Amount.LT(pool.ReserveB) {
				count++
				msg += fmt.Sprintf("pool %d: module balance for %s (%s) < reserve (%s)\n",
					pool.Id, pool.TokenB, balanceB.Amount.String(), pool.ReserveB.String())
			}
		}

		broken := count != 0
		return sdk.FormatInvariant(
			types.ModuleName, "pool-reserves",
			fmt.Sprintf("found %d pools with reserve > module balance\n%s", count, msg),
		), broken
	}
}

// PoolSharesInvariant checks that total shares are consistent with issued shares
func PoolSharesInvariant(k Keeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		var (
			msg   string
			count int
		)

		pools := k.GetAllPools(ctx)
		for _, pool := range pools {
			// Total shares must be positive
			if pool.TotalShares.IsNil() || pool.TotalShares.LTE(math.ZeroInt()) {
				count++
				msg += fmt.Sprintf("pool %d: total shares is nil or non-positive (%s)\n",
					pool.Id, pool.TotalShares.String())
			}

			// Shares should be reasonable relative to reserves
			// This prevents overflow attacks where shares >> reserves
			maxShares := pool.ReserveA.Mul(pool.ReserveB)
			if !maxShares.IsZero() && pool.TotalShares.GT(maxShares) {
				count++
				msg += fmt.Sprintf("pool %d: shares (%s) > reserve_a * reserve_b (%s)\n",
					pool.Id, pool.TotalShares.String(), maxShares.String())
			}
		}

		broken := count != 0
		return sdk.FormatInvariant(
			types.ModuleName, "pool-shares",
			fmt.Sprintf("found %d pools with invalid shares\n%s", count, msg),
		), broken
	}
}

// PositiveReservesInvariant checks that all pool reserves are positive
func PositiveReservesInvariant(k Keeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		var (
			msg   string
			count int
		)

		pools := k.GetAllPools(ctx)
		params := k.GetParams(ctx)

		for _, pool := range pools {
			// Reserve A must be positive and meet minimum
			if pool.ReserveA.IsNil() || pool.ReserveA.LTE(math.ZeroInt()) {
				count++
				msg += fmt.Sprintf("pool %d: reserve_a is nil or non-positive (%s)\n",
					pool.Id, pool.ReserveA.String())
			} else if pool.ReserveA.LT(params.MinLiquidity) {
				count++
				msg += fmt.Sprintf("pool %d: reserve_a (%s) < min liquidity (%s)\n",
					pool.Id, pool.ReserveA.String(), params.MinLiquidity.String())
			}

			// Reserve B must be positive and meet minimum
			if pool.ReserveB.IsNil() || pool.ReserveB.LTE(math.ZeroInt()) {
				count++
				msg += fmt.Sprintf("pool %d: reserve_b is nil or non-positive (%s)\n",
					pool.Id, pool.ReserveB.String())
			} else if pool.ReserveB.LT(params.MinLiquidity) {
				count++
				msg += fmt.Sprintf("pool %d: reserve_b (%s) < min liquidity (%s)\n",
					pool.Id, pool.ReserveB.String(), params.MinLiquidity.String())
			}

			// Tokens must be different
			if pool.TokenA == pool.TokenB {
				count++
				msg += fmt.Sprintf("pool %d: token_a == token_b (%s)\n",
					pool.Id, pool.TokenA)
			}

			// Pool ID must be positive
			if pool.Id == 0 {
				count++
				msg += fmt.Sprintf("pool has zero ID\n")
			}
		}

		broken := count != 0
		return sdk.FormatInvariant(
			types.ModuleName, "positive-reserves",
			fmt.Sprintf("found %d pools with invalid reserves\n%s", count, msg),
		), broken
	}
}

// ModuleAccountBalanceInvariant checks that module account has sufficient balance
func ModuleAccountBalanceInvariant(k Keeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		var (
			msg   string
			count int
		)

		// Calculate total reserves needed per token
		totalReserves := make(map[string]math.Int)
		pools := k.GetAllPools(ctx)

		for _, pool := range pools {
			// Sum up reserve A
			if existing, ok := totalReserves[pool.TokenA]; ok {
				totalReserves[pool.TokenA] = existing.Add(pool.ReserveA)
			} else {
				totalReserves[pool.TokenA] = pool.ReserveA
			}

			// Sum up reserve B
			if existing, ok := totalReserves[pool.TokenB]; ok {
				totalReserves[pool.TokenB] = existing.Add(pool.ReserveB)
			} else {
				totalReserves[pool.TokenB] = pool.ReserveB
			}
		}

		// Check module account has sufficient balance for each token
		moduleAddr := k.GetModuleAddress()
		for denom, requiredAmount := range totalReserves {
			balance := k.bankKeeper.GetBalance(ctx, moduleAddr, denom)
			if balance.Amount.LT(requiredAmount) {
				count++
				msg += fmt.Sprintf("token %s: module balance (%s) < total reserves (%s)\n",
					denom, balance.Amount.String(), requiredAmount.String())
			}
		}

		broken := count != 0
		return sdk.FormatInvariant(
			types.ModuleName, "module-account-balance",
			fmt.Sprintf("found %d tokens with insufficient module balance\n%s", count, msg),
		), broken
	}
}

// ConstantProductInvariant checks the constant product formula for AMM
func ConstantProductInvariant(k Keeper) sdk.Invariant {
	return func(ctx sdk.Context) (string, bool) {
		var (
			msg   string
			count int
		)

		pools := k.GetAllPools(ctx)
		for _, pool := range pools {
			// Calculate k = reserve_a * reserve_b
			k := pool.ReserveA.Mul(pool.ReserveB)

			// K must be positive and non-zero
			if k.IsNil() || k.LTE(math.ZeroInt()) {
				count++
				msg += fmt.Sprintf("pool %d: k (reserve_a * reserve_b) is nil or non-positive\n", pool.Id)
				continue
			}

			// Shares should approximate sqrt(k)
			// shares^2 should be close to k (within reasonable margin)
			sharesSquared := pool.TotalShares.Mul(pool.TotalShares)

			// Allow 10x deviation (shares can be less than sqrt(k) for precision)
			minK := k.Quo(math.NewInt(100))
			maxK := k.Mul(math.NewInt(100))

			if sharesSquared.LT(minK) || sharesSquared.GT(maxK) {
				count++
				msg += fmt.Sprintf("pool %d: shares^2 (%s) not consistent with k (%s)\n",
					pool.Id, sharesSquared.String(), k.String())
			}
		}

		broken := count != 0
		return sdk.FormatInvariant(
			types.ModuleName, "constant-product",
			fmt.Sprintf("found %d pools with invalid constant product\n%s", count, msg),
		), broken
	}
}
