package types

import (
	"cosmossdk.io/math"
)

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params: DefaultParams(),
		Pools:  []Pool{},
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	// Validate parameters
	if err := gs.Params.Validate(); err != nil {
		return err
	}

	// Validate pools
	seenPoolIDs := make(map[uint64]bool)
	seenTokenPairs := make(map[string]bool)

	for i, pool := range gs.Pools {
		// Validate pool ID is unique
		if seenPoolIDs[pool.Id] {
			return ErrInvalidGenesis.Wrapf("duplicate pool ID %d at index %d", pool.Id, i)
		}
		seenPoolIDs[pool.Id] = true

		// Validate pool ID is non-zero
		if pool.Id == 0 {
			return ErrInvalidGenesis.Wrapf("pool ID cannot be zero at index %d", i)
		}

		// Validate token denominations
		if pool.TokenA == "" {
			return ErrInvalidGenesis.Wrapf("empty token_a in pool %d", pool.Id)
		}
		if pool.TokenB == "" {
			return ErrInvalidGenesis.Wrapf("empty token_b in pool %d", pool.Id)
		}
		if pool.TokenA == pool.TokenB {
			return ErrInvalidGenesis.Wrapf("token_a and token_b must be different in pool %d", pool.Id)
		}

		// Validate token pair uniqueness (ensure TokenA < TokenB lexicographically)
		tokenPair := pool.TokenA + "-" + pool.TokenB
		if pool.TokenA > pool.TokenB {
			tokenPair = pool.TokenB + "-" + pool.TokenA
		}
		if seenTokenPairs[tokenPair] {
			return ErrInvalidGenesis.Wrapf("duplicate token pair %s in pool %d", tokenPair, pool.Id)
		}
		seenTokenPairs[tokenPair] = true

		// Validate reserves are positive
		if pool.ReserveA.IsNil() || pool.ReserveA.IsNegative() || pool.ReserveA.IsZero() {
			return ErrInvalidGenesis.Wrapf("reserve_a must be positive in pool %d", pool.Id)
		}
		if pool.ReserveB.IsNil() || pool.ReserveB.IsNegative() || pool.ReserveB.IsZero() {
			return ErrInvalidGenesis.Wrapf("reserve_b must be positive in pool %d", pool.Id)
		}

		// Validate total shares are positive
		if pool.TotalShares.IsNil() || pool.TotalShares.IsNegative() || pool.TotalShares.IsZero() {
			return ErrInvalidGenesis.Wrapf("total_shares must be positive in pool %d", pool.Id)
		}

		// Validate reserves meet minimum liquidity requirement
		if pool.ReserveA.LT(gs.Params.MinLiquidity) {
			return ErrInvalidGenesis.Wrapf("reserve_a %s below min liquidity %s in pool %d",
				pool.ReserveA.String(), gs.Params.MinLiquidity.String(), pool.Id)
		}
		if pool.ReserveB.LT(gs.Params.MinLiquidity) {
			return ErrInvalidGenesis.Wrapf("reserve_b %s below min liquidity %s in pool %d",
				pool.ReserveB.String(), gs.Params.MinLiquidity.String(), pool.Id)
		}

		// Validate creator address
		if pool.Creator == "" {
			return ErrInvalidGenesis.Wrapf("empty creator in pool %d", pool.Id)
		}

		// Validate constant product invariant: shares <= sqrt(reserveA * reserveB)
		// This ensures the AMM math is consistent
		product := pool.ReserveA.Mul(pool.ReserveB)
		// Approximate square root check
		expectedShares := product.Quo(pool.TotalShares)
		if expectedShares.LT(pool.TotalShares.Quo(math.NewInt(2))) {
			return ErrInvalidGenesis.Wrapf("invalid shares-to-reserves ratio in pool %d", pool.Id)
		}
	}

	return nil
}
