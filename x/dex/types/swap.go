package types

import (
	"cosmossdk.io/math"
)

// SwapResult defines the result of a swap operation
type SwapResult struct {
	AmountOut      math.Int
	PriceImpact    math.LegacyDec
	Fee            math.Int
	NewPoolPrice   math.LegacyDec
}
