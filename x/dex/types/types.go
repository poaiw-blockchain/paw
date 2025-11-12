package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Pool represents a liquidity pool for token pairs
type Pool struct {
	Id          uint64  `json:"id"`
	TokenA      string  `json:"token_a"`
	TokenB      string  `json:"token_b"`
	ReserveA    sdk.Int `json:"reserve_a"`
	ReserveB    sdk.Int `json:"reserve_b"`
	TotalShares sdk.Int `json:"total_shares"`
	Creator     string  `json:"creator"`
}

// NewPool creates a new liquidity pool
func NewPool(id uint64, tokenA, tokenB string, reserveA, reserveB sdk.Int, creator string) Pool {
	return Pool{
		Id:          id,
		TokenA:      tokenA,
		TokenB:      tokenB,
		ReserveA:    reserveA,
		ReserveB:    reserveB,
		TotalShares: reserveA.Mul(reserveB), // Initial shares = sqrt(reserveA * reserveB) approximated
		Creator:     creator,
	}
}

// Validate performs basic validation of pool parameters
func (p Pool) Validate() error {
	if p.Id == 0 {
		return fmt.Errorf("pool id cannot be zero")
	}
	if p.TokenA == "" || p.TokenB == "" {
		return fmt.Errorf("token denoms cannot be empty")
	}
	if p.TokenA == p.TokenB {
		return fmt.Errorf("token denoms must be different")
	}
	if p.ReserveA.IsNegative() || p.ReserveA.IsZero() {
		return fmt.Errorf("reserve A must be positive")
	}
	if p.ReserveB.IsNegative() || p.ReserveB.IsZero() {
		return fmt.Errorf("reserve B must be positive")
	}
	if p.TotalShares.IsNegative() || p.TotalShares.IsZero() {
		return fmt.Errorf("total shares must be positive")
	}
	if p.Creator == "" {
		return fmt.Errorf("creator address cannot be empty")
	}
	return nil
}

// GetTokenPair returns the ordered token pair
func (p Pool) GetTokenPair() (string, string) {
	if p.TokenA < p.TokenB {
		return p.TokenA, p.TokenB
	}
	return p.TokenB, p.TokenA
}
