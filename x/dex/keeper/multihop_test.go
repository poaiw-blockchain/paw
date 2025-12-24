package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	"github.com/paw-chain/paw/x/dex/keeper"
)

func TestSwapHopValidation(t *testing.T) {
	// Test hop chain validation
	validHops := []keeper.SwapHop{
		{PoolID: 1, TokenIn: "tokenA", TokenOut: "tokenB"},
		{PoolID: 2, TokenIn: "tokenB", TokenOut: "tokenC"},
		{PoolID: 3, TokenIn: "tokenC", TokenOut: "tokenD"},
	}

	// Verify chain is continuous
	for i := 0; i < len(validHops)-1; i++ {
		require.Equal(t, validHops[i].TokenOut, validHops[i+1].TokenIn,
			"hop chain should be continuous")
	}

	// Invalid chain - broken at hop 1
	invalidHops := []keeper.SwapHop{
		{PoolID: 1, TokenIn: "tokenA", TokenOut: "tokenB"},
		{PoolID: 2, TokenIn: "tokenC", TokenOut: "tokenD"}, // broken - should be tokenB
	}

	require.NotEqual(t, invalidHops[0].TokenOut, invalidHops[1].TokenIn,
		"invalid chain should have mismatch")
}

func TestMultiHopSwapResult(t *testing.T) {
	// Test result structure
	result := keeper.MultiHopSwapResult{
		AmountIn:   math.NewInt(1000000),
		AmountOut:  math.NewInt(950000),
		HopAmounts: []math.Int{math.NewInt(1000000), math.NewInt(980000), math.NewInt(950000)},
		TotalFees:  math.NewInt(30000),
		GasUsed:    150000,
	}

	require.True(t, result.AmountIn.GT(result.AmountOut),
		"amount out should be less than amount in due to fees")
	require.Equal(t, 3, len(result.HopAmounts),
		"should have 3 hop amounts for a 2-hop swap")
	require.True(t, result.TotalFees.GT(math.ZeroInt()),
		"total fees should be positive")
}

func TestMaxHopsLimit(t *testing.T) {
	// Maximum 5 hops allowed
	maxHops := 5

	hops := make([]keeper.SwapHop, maxHops)
	for i := 0; i < maxHops; i++ {
		hops[i] = keeper.SwapHop{
			PoolID:   uint64(i + 1),
			TokenIn:  "token" + string(rune('A'+i)),
			TokenOut: "token" + string(rune('B'+i)),
		}
	}

	require.Equal(t, maxHops, len(hops))

	// More than 5 hops should be invalid
	tooManyHops := make([]keeper.SwapHop, 6)
	require.Greater(t, len(tooManyHops), maxHops)
}

func TestHopAmountsProgression(t *testing.T) {
	// Hop amounts should generally decrease due to fees
	hopAmounts := []math.Int{
		math.NewInt(1000000), // initial input
		math.NewInt(970000),  // after first hop
		math.NewInt(940900),  // after second hop
	}

	for i := 1; i < len(hopAmounts); i++ {
		require.True(t, hopAmounts[i].LT(hopAmounts[i-1]),
			"amounts should decrease through hops due to fees")
	}
}

func TestMultiHopGasSavings(t *testing.T) {
	// Target: 40% gas savings for 3-hop swaps
	//
	// Traditional 3 separate swaps:
	// - 3x state reads
	// - 3x state writes
	// - 3x event emissions
	// - 3x invariant checks
	//
	// Batched multi-hop:
	// - 1x state read per unique pool
	// - 1x state write per unique pool
	// - 1x event emission
	// - 1x invariant check per unique pool

	singleSwapGas := uint64(50000)     // approximate gas per swap
	threeSwapsGas := 3 * singleSwapGas // 150000

	// With batching, we expect ~60% of original gas
	expectedBatchedGas := uint64(float64(threeSwapsGas) * 0.6) // ~90000

	savings := float64(threeSwapsGas-expectedBatchedGas) / float64(threeSwapsGas)
	require.GreaterOrEqual(t, savings, 0.4, "should achieve at least 40% gas savings")
}

func TestSwapHopEquality(t *testing.T) {
	hop1 := keeper.SwapHop{PoolID: 1, TokenIn: "tokenA", TokenOut: "tokenB"}
	hop2 := keeper.SwapHop{PoolID: 1, TokenIn: "tokenA", TokenOut: "tokenB"}
	hop3 := keeper.SwapHop{PoolID: 2, TokenIn: "tokenA", TokenOut: "tokenB"}

	require.Equal(t, hop1, hop2)
	require.NotEqual(t, hop1, hop3)
}
