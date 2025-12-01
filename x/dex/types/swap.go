package types

import (
	"cosmossdk.io/math"
)

// SwapResult defines the result of a swap operation
type SwapResult struct {
	AmountIn     math.Int
	AmountOut    math.Int
	Route        string
	Slippage     math.LegacyDec
	PriceImpact  math.LegacyDec
	Fee          math.Int
	NewPoolPrice math.LegacyDec
}
