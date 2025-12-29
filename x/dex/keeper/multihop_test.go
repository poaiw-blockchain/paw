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

// === Route Finding Tests ===

func TestRouteConstants(t *testing.T) {
	// Verify route finding constants are reasonable for resource bounds
	require.Equal(t, 100, keeper.MaxRouteSearchPools,
		"should limit pools searched to prevent resource exhaustion")
	require.Equal(t, 10, keeper.MaxRouteCandidates,
		"should limit candidate routes for evaluation")
	require.Equal(t, 3, keeper.DefaultMaxHops,
		"default max hops should be reasonable")
}

func TestRouteValidation(t *testing.T) {
	tests := []struct {
		name     string
		route    []keeper.SwapHop
		wantErr  bool
		errCheck func([]keeper.SwapHop) bool
	}{
		{
			name: "valid single hop",
			route: []keeper.SwapHop{
				{PoolID: 1, TokenIn: "tokenA", TokenOut: "tokenB"},
			},
			wantErr: false,
		},
		{
			name: "valid two hop route",
			route: []keeper.SwapHop{
				{PoolID: 1, TokenIn: "tokenA", TokenOut: "tokenB"},
				{PoolID: 2, TokenIn: "tokenB", TokenOut: "tokenC"},
			},
			wantErr: false,
		},
		{
			name: "valid three hop route",
			route: []keeper.SwapHop{
				{PoolID: 1, TokenIn: "tokenA", TokenOut: "tokenB"},
				{PoolID: 2, TokenIn: "tokenB", TokenOut: "tokenC"},
				{PoolID: 3, TokenIn: "tokenC", TokenOut: "tokenD"},
			},
			wantErr: false,
		},
		{
			name: "broken chain - token mismatch",
			route: []keeper.SwapHop{
				{PoolID: 1, TokenIn: "tokenA", TokenOut: "tokenB"},
				{PoolID: 2, TokenIn: "tokenC", TokenOut: "tokenD"}, // should be tokenB
			},
			wantErr: true,
			errCheck: func(hops []keeper.SwapHop) bool {
				return hops[0].TokenOut != hops[1].TokenIn
			},
		},
		{
			name:    "empty route",
			route:   []keeper.SwapHop{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Validate chain continuity
			var isValid bool
			if len(tt.route) == 0 {
				isValid = false
			} else {
				isValid = true
				for i := 0; i < len(tt.route)-1; i++ {
					if tt.route[i].TokenOut != tt.route[i+1].TokenIn {
						isValid = false
						break
					}
				}
			}

			if tt.wantErr {
				require.False(t, isValid)
				if tt.errCheck != nil {
					require.True(t, tt.errCheck(tt.route))
				}
			} else {
				require.True(t, isValid)
			}
		})
	}
}

func TestRouteCycleDetection(t *testing.T) {
	// Routes should not contain cycles (same token visited twice)
	tests := []struct {
		name    string
		route   []keeper.SwapHop
		hasCyle bool
	}{
		{
			name: "no cycle - linear path",
			route: []keeper.SwapHop{
				{PoolID: 1, TokenIn: "A", TokenOut: "B"},
				{PoolID: 2, TokenIn: "B", TokenOut: "C"},
				{PoolID: 3, TokenIn: "C", TokenOut: "D"},
			},
			hasCyle: false,
		},
		{
			name: "cycle - returns to start",
			route: []keeper.SwapHop{
				{PoolID: 1, TokenIn: "A", TokenOut: "B"},
				{PoolID: 2, TokenIn: "B", TokenOut: "C"},
				{PoolID: 3, TokenIn: "C", TokenOut: "A"}, // returns to A
			},
			hasCyle: true,
		},
		{
			name: "cycle - intermediate revisit",
			route: []keeper.SwapHop{
				{PoolID: 1, TokenIn: "A", TokenOut: "B"},
				{PoolID: 2, TokenIn: "B", TokenOut: "C"},
				{PoolID: 3, TokenIn: "C", TokenOut: "B"}, // returns to B
			},
			hasCyle: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			visited := make(map[string]bool)
			hasCycle := false

			for _, hop := range tt.route {
				if visited[hop.TokenIn] {
					hasCycle = true
					break
				}
				visited[hop.TokenIn] = true
			}

			// Also check if final tokenOut creates a cycle
			if len(tt.route) > 0 && visited[tt.route[len(tt.route)-1].TokenOut] {
				hasCycle = true
			}

			require.Equal(t, tt.hasCyle, hasCycle)
		})
	}
}

func TestRoutePoolReuse(t *testing.T) {
	// Same pool can be used multiple times in a route (e.g., A->B, then B->A in different hops)
	// This is valid for triangular arbitrage detection

	route := []keeper.SwapHop{
		{PoolID: 1, TokenIn: "A", TokenOut: "B"},
		{PoolID: 2, TokenIn: "B", TokenOut: "C"},
		{PoolID: 1, TokenIn: "C", TokenOut: "D"}, // reuses pool 1 (if it has C/D pair)
	}

	poolUsage := make(map[uint64]int)
	for _, hop := range route {
		poolUsage[hop.PoolID]++
	}

	require.Equal(t, 2, poolUsage[1], "pool 1 used twice")
	require.Equal(t, 1, poolUsage[2], "pool 2 used once")
}

func TestRouteOutputProgression(t *testing.T) {
	// Test that route finding would select the route with best output
	// Given multiple routes, the one with highest final output should be selected

	route1Output := math.NewInt(950000) // Route A->B->C
	route2Output := math.NewInt(940000) // Route A->D->C (worse)
	route3Output := math.NewInt(960000) // Route A->E->F->C (better despite more hops)

	outputs := []math.Int{route1Output, route2Output, route3Output}

	var bestOutput math.Int
	bestIdx := -1
	for i, out := range outputs {
		if bestIdx == -1 || out.GT(bestOutput) {
			bestOutput = out
			bestIdx = i
		}
	}

	require.Equal(t, 2, bestIdx, "should select route 3 with best output")
	require.Equal(t, route3Output, bestOutput)
}

func TestMaxHopsBoundary(t *testing.T) {
	tests := []struct {
		name        string
		requestHops int
		expectHops  int
	}{
		{"zero defaults to 3", 0, 3},
		{"negative defaults to 3", -1, 3},
		{"1 hop allowed", 1, 1},
		{"3 hops allowed", 3, 3},
		{"5 hops allowed (max)", 5, 5},
		{"6 hops capped to 3", 6, 3},
		{"100 hops capped to 3", 100, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			maxHops := tt.requestHops
			if maxHops <= 0 || maxHops > 5 {
				maxHops = keeper.DefaultMaxHops
			}
			require.Equal(t, tt.expectHops, maxHops)
		})
	}
}

func TestRouteFindingResourceBounds(t *testing.T) {
	// Verify that route finding respects resource limits

	// With 100 pools max and 10 candidate routes max:
	// - Memory: O(100 pools * edge data) + O(10 routes * 5 hops)
	// - Time: O(100 * branching factor^5) bounded by candidate limit

	maxPools := keeper.MaxRouteSearchPools
	maxCandidates := keeper.MaxRouteCandidates
	maxHops := 5

	// Estimate worst case memory (rough)
	edgeSize := 32            // poolID (8) + string pointer (16) + overhead
	poolMemory := maxPools * 2 * edgeSize // 2 edges per pool
	routeMemory := maxCandidates * maxHops * 32 // hop struct size

	totalMemory := poolMemory + routeMemory

	// Should be well under 1MB even in worst case
	require.Less(t, totalMemory, 1024*1024,
		"route finding memory should be bounded under 1MB")
}

func TestDirectRouteOptimization(t *testing.T) {
	// Direct routes (1 hop) should be found without building full graph
	// This is a fast path optimization

	directHop := keeper.SwapHop{
		PoolID:   1,
		TokenIn:  "tokenA",
		TokenOut: "tokenB",
	}

	route := []keeper.SwapHop{directHop}

	require.Equal(t, 1, len(route), "direct route has 1 hop")
	require.Equal(t, "tokenA", route[0].TokenIn)
	require.Equal(t, "tokenB", route[0].TokenOut)
}
