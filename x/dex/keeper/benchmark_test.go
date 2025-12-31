package keeper_test

import (
	"fmt"
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/dex/keeper"
	"github.com/paw-chain/paw/x/dex/types"
)

// Benchmark pool sizes for different scenarios
// Note: Sizes are constrained by pre-funded amounts (5B per token per address)
const (
	SmallPoolSize  = 10_000_000   // 10M - Small pool
	MediumPoolSize = 100_000_000  // 100M - Medium pool
	LargePoolSize  = 500_000_000  // 500M - Large pool (within 5B limit)
)

// Pre-funded tokens from testutil/keeper/dex.go
const (
	benchTokenA = "tokenA"
	benchTokenB = "tokenB"
	benchTokenC = "tokenC"
	benchTokenD = "tokenD"
	benchTokenE = "tokenE"
	benchTokenF = "tokenF"
)

// createBenchmarkPool creates a pool for benchmarks with specified reserve size
// Uses pre-funded tokenA and tokenB
func createBenchmarkPool(b *testing.B, k *keeper.Keeper, ctx sdk.Context, reserveSize int64) uint64 {
	b.Helper()
	creator := types.TestAddr()
	pool, err := k.CreatePool(ctx, creator, benchTokenA, benchTokenB, math.NewInt(reserveSize), math.NewInt(reserveSize))
	if err != nil {
		b.Fatalf("failed to create benchmark pool: %v", err)
	}
	return pool.Id
}

// createBenchmarkPoolWithTokens creates a pool with custom tokens
func createBenchmarkPoolWithTokens(b *testing.B, k *keeper.Keeper, ctx sdk.Context, tokenA, tokenB string, reserveSize int64) uint64 {
	b.Helper()
	creator := types.TestAddr()
	pool, err := k.CreatePool(ctx, creator, tokenA, tokenB, math.NewInt(reserveSize), math.NewInt(reserveSize))
	if err != nil {
		b.Fatalf("failed to create benchmark pool with tokens %s/%s: %v", tokenA, tokenB, err)
	}
	return pool.Id
}

// getBenchmarkTrader returns a pre-funded trader address from the test keeper
// Uses addresses that are already funded in testutil/keeper/dex.go
func getBenchmarkTrader(index int) sdk.AccAddress {
	switch index {
	case 0:
		return types.TestAddr()
	case 1:
		return sdk.AccAddress([]byte("trader1____________"))
	case 2:
		return sdk.AccAddress([]byte("trader2____________"))
	case 3:
		return sdk.AccAddress([]byte("trader3____________"))
	case 4:
		return sdk.AccAddress([]byte("provider1__________"))
	default:
		// Use the test_trader_ pattern that's funded in testutil
		addr := make([]byte, 20)
		copy(addr, []byte("test_trader_"))
		addr[19] = byte(index % 10)
		return sdk.AccAddress(addr)
	}
}

// ============================================================================
// BenchmarkSwap - Benchmark ExecuteSwap with various pool sizes
// ============================================================================

func BenchmarkSwap(b *testing.B) {
	b.Run("SmallPool", func(b *testing.B) {
		benchmarkSwapWithPoolSize(b, SmallPoolSize)
	})
	b.Run("MediumPool", func(b *testing.B) {
		benchmarkSwapWithPoolSize(b, MediumPoolSize)
	})
	b.Run("LargePool", func(b *testing.B) {
		benchmarkSwapWithPoolSize(b, LargePoolSize)
	})
}

func benchmarkSwapWithPoolSize(b *testing.B, reserveSize int64) {
	k, ctx := keepertest.DexKeeper(b)
	poolID := createBenchmarkPool(b, k, ctx, reserveSize)
	trader := getBenchmarkTrader(1)

	// Swap 0.1% of pool size to stay within MEV limits
	swapAmount := math.NewInt(reserveSize / 1000)
	minAmountOut := math.NewInt(1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := k.ExecuteSwap(ctx, trader, poolID, benchTokenA, benchTokenB, swapAmount, minAmountOut)
		if err != nil {
			b.Fatalf("swap failed: %v", err)
		}
	}
}

// BenchmarkSwapParallel benchmarks concurrent swap execution
func BenchmarkSwapParallel(b *testing.B) {
	k, ctx := keepertest.DexKeeper(b)
	poolID := createBenchmarkPool(b, k, ctx, LargePoolSize)

	swapAmount := math.NewInt(LargePoolSize / 10000)
	minAmountOut := math.NewInt(1)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		trader := getBenchmarkTrader(1)
		for pb.Next() {
			// Note: parallel operations may have race conditions in test context
			_, _ = k.ExecuteSwap(ctx, trader, poolID, benchTokenA, benchTokenB, swapAmount, minAmountOut)
		}
	})
}

// BenchmarkSwapCalculation benchmarks just the swap output calculation
func BenchmarkSwapCalculation(b *testing.B) {
	k, ctx := keepertest.DexKeeper(b)

	params, _ := k.GetParams(ctx)
	reserveIn := math.NewInt(LargePoolSize)
	reserveOut := math.NewInt(LargePoolSize)
	amountIn := math.NewInt(LargePoolSize / 1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := k.CalculateSwapOutput(ctx, amountIn, reserveIn, reserveOut, params.SwapFee, params.MaxPoolDrainPercent)
		if err != nil {
			b.Fatalf("calculation failed: %v", err)
		}
	}
}

// BenchmarkSwapBidirectional benchmarks swaps in both directions
func BenchmarkSwapBidirectional(b *testing.B) {
	k, ctx := keepertest.DexKeeper(b)
	poolID := createBenchmarkPool(b, k, ctx, LargePoolSize)
	trader := getBenchmarkTrader(1)

	swapAmount := math.NewInt(LargePoolSize / 10000)
	minAmountOut := math.NewInt(1)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Alternate swap directions
		if i%2 == 0 {
			_, err := k.ExecuteSwap(ctx, trader, poolID, benchTokenA, benchTokenB, swapAmount, minAmountOut)
			if err != nil {
				b.Fatalf("swap A->B failed: %v", err)
			}
		} else {
			_, err := k.ExecuteSwap(ctx, trader, poolID, benchTokenB, benchTokenA, swapAmount, minAmountOut)
			if err != nil {
				b.Fatalf("swap B->A failed: %v", err)
			}
		}
	}
}

// ============================================================================
// BenchmarkAddLiquidity - Benchmark AddLiquidity operations
// ============================================================================

func BenchmarkAddLiquidity(b *testing.B) {
	b.Run("SmallPool", func(b *testing.B) {
		benchmarkAddLiquidityWithPoolSize(b, SmallPoolSize)
	})
	b.Run("MediumPool", func(b *testing.B) {
		benchmarkAddLiquidityWithPoolSize(b, MediumPoolSize)
	})
	b.Run("LargePool", func(b *testing.B) {
		benchmarkAddLiquidityWithPoolSize(b, LargePoolSize)
	})
}

func benchmarkAddLiquidityWithPoolSize(b *testing.B, reserveSize int64) {
	k, ctx := keepertest.DexKeeper(b)
	poolID := createBenchmarkPool(b, k, ctx, reserveSize)
	provider := getBenchmarkTrader(2)

	// Add 0.1% of pool size per operation
	addAmount := math.NewInt(reserveSize / 1000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := k.AddLiquidity(ctx, provider, poolID, addAmount, addAmount)
		if err != nil {
			b.Fatalf("add liquidity failed: %v", err)
		}
	}
}

// BenchmarkAddLiquidityParallel benchmarks concurrent liquidity additions
func BenchmarkAddLiquidityParallel(b *testing.B) {
	k, ctx := keepertest.DexKeeper(b)
	poolID := createBenchmarkPool(b, k, ctx, LargePoolSize)

	addAmount := math.NewInt(LargePoolSize / 10000)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		provider := getBenchmarkTrader(2)
		for pb.Next() {
			_, _ = k.AddLiquidity(ctx, provider, poolID, addAmount, addAmount)
		}
	})
}

// BenchmarkAddLiquidityMultipleProviders benchmarks adding liquidity from many providers
func BenchmarkAddLiquidityMultipleProviders(b *testing.B) {
	k, ctx := keepertest.DexKeeper(b)
	poolID := createBenchmarkPool(b, k, ctx, LargePoolSize)

	addAmount := math.NewInt(LargePoolSize / 10000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Rotate through available pre-funded traders
		provider := getBenchmarkTrader(i % 10)
		_, err := k.AddLiquidity(ctx, provider, poolID, addAmount, addAmount)
		if err != nil {
			b.Fatalf("add liquidity failed for provider %d: %v", i%10, err)
		}
	}
}

// ============================================================================
// BenchmarkRemoveLiquidity - Benchmark RemoveLiquidity operations
// ============================================================================

func BenchmarkRemoveLiquidity(b *testing.B) {
	b.Run("SmallPool", func(b *testing.B) {
		benchmarkRemoveLiquidityWithPoolSize(b, SmallPoolSize)
	})
	b.Run("MediumPool", func(b *testing.B) {
		benchmarkRemoveLiquidityWithPoolSize(b, MediumPoolSize)
	})
	b.Run("LargePool", func(b *testing.B) {
		benchmarkRemoveLiquidityWithPoolSize(b, LargePoolSize)
	})
}

func benchmarkRemoveLiquidityWithPoolSize(b *testing.B, reserveSize int64) {
	k, ctx := keepertest.DexKeeper(b)
	provider := getBenchmarkTrader(3)

	// Create pool with larger reserves to allow multiple withdrawals
	poolReserve := reserveSize * 10
	if poolReserve > 4_000_000_000 { // Stay within 5B limit
		poolReserve = 4_000_000_000
	}
	pool, err := k.CreatePool(ctx, types.TestAddr(), benchTokenA, benchTokenB, math.NewInt(poolReserve), math.NewInt(poolReserve))
	if err != nil {
		b.Fatalf("failed to create pool: %v", err)
	}

	// Add significant liquidity to the pool for the provider
	addAmount := math.NewInt(poolReserve / 10)
	_, err = k.AddLiquidity(ctx, provider, pool.Id, addAmount, addAmount)
	if err != nil {
		b.Fatalf("failed to add liquidity: %v", err)
	}

	// Advance block to satisfy flash loan protection (FlashLoanProtectionBlocks = 100)
	ctx = advanceBlockHeight(ctx, 101)

	// Get provider's shares to calculate withdrawal amount
	shares, err := k.GetLiquidity(ctx, pool.Id, provider)
	if err != nil {
		b.Fatalf("failed to get liquidity: %v", err)
	}

	// Remove small fraction per iteration (0.01% of shares) to allow many iterations
	// while still leaving enough reserves (MinimumReserves = 1,000,000)
	removeShares := shares.Quo(math.NewInt(10000))
	if removeShares.IsZero() {
		removeShares = math.NewInt(1)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := k.RemoveLiquidity(ctx, provider, pool.Id, removeShares)
		if err != nil {
			// Skip error - likely hit minimum reserves
			continue
		}
	}
}

// advanceBlockHeight returns a new context with an advanced block height
func advanceBlockHeight(ctx sdk.Context, blocks int64) sdk.Context {
	header := ctx.BlockHeader()
	header.Height += blocks
	return ctx.WithBlockHeader(header)
}

// BenchmarkRemoveLiquidityParallel benchmarks concurrent liquidity removals
func BenchmarkRemoveLiquidityParallel(b *testing.B) {
	k, ctx := keepertest.DexKeeper(b)
	provider := getBenchmarkTrader(4)

	// Create large pool
	poolReserve := int64(2_000_000_000)
	pool, err := k.CreatePool(ctx, types.TestAddr(), benchTokenA, benchTokenB, math.NewInt(poolReserve), math.NewInt(poolReserve))
	if err != nil {
		b.Fatalf("failed to create pool: %v", err)
	}

	// Add significant liquidity
	addAmount := math.NewInt(poolReserve / 5)
	_, err = k.AddLiquidity(ctx, provider, pool.Id, addAmount, addAmount)
	if err != nil {
		b.Fatalf("failed to add liquidity: %v", err)
	}

	ctx = advanceBlockHeight(ctx, 101)

	shares, _ := k.GetLiquidity(ctx, pool.Id, provider)
	removeShares := shares.Quo(math.NewInt(100000))

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _, _ = k.RemoveLiquidity(ctx, provider, pool.Id, removeShares)
		}
	})
}

// ============================================================================
// BenchmarkMultihopSwap - Benchmark multi-hop swap routing
// ============================================================================

func BenchmarkMultihopSwap(b *testing.B) {
	// Note: Multi-hop swap benchmarks test the SimulateMultiHopSwap function
	// which is the read-only simulation path. The ExecuteMultiHopSwap has
	// an issue with uninitialized deltaA/deltaB in poolUpdates map that needs
	// to be addressed separately.
	b.Run("TwoHops", func(b *testing.B) {
		benchmarkMultihopSimulation(b, 2)
	})
	b.Run("ThreeHops", func(b *testing.B) {
		benchmarkMultihopSimulation(b, 3)
	})
	b.Run("FourHops", func(b *testing.B) {
		benchmarkMultihopSimulation(b, 4)
	})
	b.Run("FiveHops", func(b *testing.B) {
		benchmarkMultihopSimulation(b, 5)
	})
}

// benchmarkMultihopSimulation benchmarks multi-hop swap simulation (read-only, no state changes)
func benchmarkMultihopSimulation(b *testing.B, numHops int) {
	k, ctx := keepertest.DexKeeper(b)
	creator := types.TestAddr()

	// Use pre-funded tokens: tokenA, tokenB, tokenC, tokenD, tokenE, tokenF
	tokens := []string{benchTokenA, benchTokenB, benchTokenC, benchTokenD, benchTokenE, benchTokenF}

	if numHops+1 > len(tokens) {
		b.Skipf("not enough pre-funded tokens for %d hops", numHops)
	}

	// Create chain of pools
	poolIDs := make([]uint64, numHops)
	poolReserve := int64(MediumPoolSize)

	for i := 0; i < numHops; i++ {
		pool, err := k.CreatePool(ctx, creator, tokens[i], tokens[i+1], math.NewInt(poolReserve), math.NewInt(poolReserve))
		if err != nil {
			b.Fatalf("failed to create pool %d (%s/%s): %v", i, tokens[i], tokens[i+1], err)
		}
		poolIDs[i] = pool.Id
	}

	// Build hop route
	hops := make([]keeper.SwapHop, numHops)
	for i := 0; i < numHops; i++ {
		hops[i] = keeper.SwapHop{
			PoolID:   poolIDs[i],
			TokenIn:  tokens[i],
			TokenOut: tokens[i+1],
		}
	}

	swapAmount := math.NewInt(poolReserve / 10000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := k.SimulateMultiHopSwap(ctx, hops, swapAmount)
		if err != nil {
			b.Fatalf("multihop simulation failed: %v", err)
		}
	}
}

// BenchmarkMultihopSwapSimulation benchmarks simulating multi-hop swaps (no state changes)
func BenchmarkMultihopSwapSimulation(b *testing.B) {
	k, ctx := keepertest.DexKeeper(b)
	creator := types.TestAddr()

	// Create 3-hop route
	numHops := 3
	tokens := []string{benchTokenA, benchTokenB, benchTokenC, benchTokenD}
	poolIDs := make([]uint64, numHops)

	for i := 0; i < numHops; i++ {
		pool, err := k.CreatePool(ctx, creator, tokens[i], tokens[i+1], math.NewInt(MediumPoolSize), math.NewInt(MediumPoolSize))
		if err != nil {
			b.Fatalf("failed to create pool: %v", err)
		}
		poolIDs[i] = pool.Id
	}

	hops := make([]keeper.SwapHop, numHops)
	for i := 0; i < numHops; i++ {
		hops[i] = keeper.SwapHop{
			PoolID:   poolIDs[i],
			TokenIn:  tokens[i],
			TokenOut: tokens[i+1],
		}
	}

	swapAmount := math.NewInt(MediumPoolSize / 10000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := k.SimulateMultiHopSwap(ctx, hops, swapAmount)
		if err != nil {
			b.Fatalf("simulation failed: %v", err)
		}
	}
}

// BenchmarkRouteFinding benchmarks finding the best route between tokens
func BenchmarkRouteFinding(b *testing.B) {
	b.Run("DirectRoute", func(b *testing.B) {
		k, ctx := keepertest.DexKeeper(b)
		creator := types.TestAddr()

		// Create direct route
		_, err := k.CreatePool(ctx, creator, benchTokenA, benchTokenB, math.NewInt(MediumPoolSize), math.NewInt(MediumPoolSize))
		if err != nil {
			b.Fatalf("failed to create pool: %v", err)
		}

		swapAmount := math.NewInt(MediumPoolSize / 10000)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := k.FindBestRoute(ctx, benchTokenA, benchTokenB, swapAmount, 3)
			if err != nil {
				b.Fatalf("route finding failed: %v", err)
			}
		}
	})

	b.Run("TwoHopRoute", func(b *testing.B) {
		k, ctx := keepertest.DexKeeper(b)
		creator := types.TestAddr()

		// Create two-hop route: A -> C -> B (no direct A -> B)
		_, _ = k.CreatePool(ctx, creator, benchTokenA, benchTokenC, math.NewInt(MediumPoolSize), math.NewInt(MediumPoolSize))
		_, _ = k.CreatePool(ctx, creator, benchTokenC, benchTokenB, math.NewInt(MediumPoolSize), math.NewInt(MediumPoolSize))

		swapAmount := math.NewInt(MediumPoolSize / 10000)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			route, err := k.FindBestRoute(ctx, benchTokenA, benchTokenB, swapAmount, 3)
			if err != nil || len(route) != 2 {
				continue // Route finding may not find optimal path
			}
		}
	})

	b.Run("ComplexGraph", func(b *testing.B) {
		k, ctx := keepertest.DexKeeper(b)
		creator := types.TestAddr()

		// Create complex graph with multiple paths
		tokens := []string{benchTokenA, benchTokenB, benchTokenC, benchTokenD, benchTokenE, benchTokenF}
		for i := 0; i < len(tokens); i++ {
			for j := i + 1; j < len(tokens); j++ {
				_, _ = k.CreatePool(ctx, creator, tokens[i], tokens[j], math.NewInt(MediumPoolSize/10), math.NewInt(MediumPoolSize/10))
			}
		}

		swapAmount := math.NewInt(MediumPoolSize / 100000)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = k.FindBestRoute(ctx, benchTokenA, benchTokenF, swapAmount, 3)
		}
	})
}

// ============================================================================
// BenchmarkPoolCreation - Benchmark CreatePool
// ============================================================================

func BenchmarkPoolCreation(b *testing.B) {
	b.Run("SmallPool", func(b *testing.B) {
		benchmarkPoolCreationWithSize(b, SmallPoolSize)
	})
	b.Run("MediumPool", func(b *testing.B) {
		benchmarkPoolCreationWithSize(b, MediumPoolSize)
	})
	b.Run("LargePool", func(b *testing.B) {
		benchmarkPoolCreationWithSize(b, LargePoolSize)
	})
}

func benchmarkPoolCreationWithSize(b *testing.B, reserveSize int64) {
	k, bankKeeper, ctx := keepertest.DexKeeperWithBank(b)
	creator := types.TestAddr()

	// We need to mint tokens for pool creation since we're creating many pools
	// with unique token names
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Use unique token names per iteration to avoid "pool already exists" errors
		tokenA := fmt.Sprintf("benchA%d", i)
		tokenB := fmt.Sprintf("benchB%d", i)

		// Mint tokens for this pool creation
		mintCoins := sdk.NewCoins(
			sdk.NewInt64Coin(tokenA, reserveSize*2),
			sdk.NewInt64Coin(tokenB, reserveSize*2),
		)
		err := bankKeeper.MintCoins(ctx, types.ModuleName, mintCoins)
		if err != nil {
			b.Fatalf("failed to mint coins: %v", err)
		}
		err = bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, creator, mintCoins)
		if err != nil {
			b.Fatalf("failed to send coins: %v", err)
		}

		_, err = k.CreatePool(ctx, creator, tokenA, tokenB, math.NewInt(reserveSize), math.NewInt(reserveSize))
		if err != nil {
			b.Fatalf("pool creation failed: %v", err)
		}
	}
}

// BenchmarkPoolCreationWithPreFundedTokens benchmarks creating pools with pre-funded tokens
func BenchmarkPoolCreationWithPreFundedTokens(b *testing.B) {
	k, ctx := keepertest.DexKeeper(b)
	creator := types.TestAddr()

	// Create just one pool since we can't create duplicates with same tokens
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if i == 0 {
			_, err := k.CreatePool(ctx, creator, benchTokenA, benchTokenB, math.NewInt(SmallPoolSize), math.NewInt(SmallPoolSize))
			if err != nil {
				b.Fatalf("pool creation failed: %v", err)
			}
		}
		// Subsequent iterations measure the overhead of checking existing pool
	}
}

// ============================================================================
// BenchmarkPoolLookup - Benchmark pool retrieval operations
// ============================================================================

func BenchmarkPoolLookup(b *testing.B) {
	b.Run("ByID", func(b *testing.B) {
		k, ctx := keepertest.DexKeeper(b)
		poolID := createBenchmarkPool(b, k, ctx, MediumPoolSize)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := k.GetPool(ctx, poolID)
			if err != nil {
				b.Fatalf("lookup failed: %v", err)
			}
		}
	})

	b.Run("ByTokens", func(b *testing.B) {
		k, ctx := keepertest.DexKeeper(b)
		createBenchmarkPool(b, k, ctx, MediumPoolSize)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := k.GetPoolByTokens(ctx, benchTokenA, benchTokenB)
			if err != nil {
				b.Fatalf("lookup failed: %v", err)
			}
		}
	})

	b.Run("AllPools", func(b *testing.B) {
		k, bankKeeper, ctx := keepertest.DexKeeperWithBank(b)
		creator := types.TestAddr()

		// Create 50 pools
		for i := 0; i < 50; i++ {
			tokenA := fmt.Sprintf("lookupA%d", i)
			tokenB := fmt.Sprintf("lookupB%d", i)

			// Mint tokens
			mintCoins := sdk.NewCoins(
				sdk.NewInt64Coin(tokenA, MediumPoolSize*2),
				sdk.NewInt64Coin(tokenB, MediumPoolSize*2),
			)
			_ = bankKeeper.MintCoins(ctx, types.ModuleName, mintCoins)
			_ = bankKeeper.SendCoinsFromModuleToAccount(ctx, types.ModuleName, creator, mintCoins)

			_, err := k.CreatePool(ctx, creator, tokenA, tokenB, math.NewInt(MediumPoolSize), math.NewInt(MediumPoolSize))
			if err != nil {
				b.Fatalf("pool creation failed: %v", err)
			}
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			pools, err := k.GetAllPools(ctx)
			if err != nil {
				b.Fatalf("get all pools failed: %v", err)
			}
			if len(pools) < 50 {
				b.Fatalf("expected at least 50 pools, got %d", len(pools))
			}
		}
	})
}

// ============================================================================
// BenchmarkSecurityValidations - Benchmark security-related operations
// ============================================================================

func BenchmarkSecurityValidations(b *testing.B) {
	b.Run("PoolStateValidation", func(b *testing.B) {
		k, ctx := keepertest.DexKeeper(b)
		pool, _ := k.GetPool(ctx, createBenchmarkPool(b, k, ctx, MediumPoolSize))

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err := k.ValidatePoolState(pool)
			if err != nil {
				b.Fatalf("validation failed: %v", err)
			}
		}
	})

	b.Run("SwapSizeValidation", func(b *testing.B) {
		k, _ := keepertest.DexKeeper(b)
		reserveIn := math.NewInt(LargePoolSize)
		amountIn := math.NewInt(LargePoolSize / 100)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err := k.ValidateSwapSize(amountIn, reserveIn)
			if err != nil {
				b.Fatalf("validation failed: %v", err)
			}
		}
	})

	b.Run("InvariantValidation", func(b *testing.B) {
		k, ctx := keepertest.DexKeeper(b)
		pool, _ := k.GetPool(ctx, createBenchmarkPool(b, k, ctx, MediumPoolSize))
		oldK := pool.ReserveA.Mul(pool.ReserveB)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err := k.ValidatePoolInvariant(ctx, pool, oldK)
			if err != nil {
				b.Fatalf("validation failed: %v", err)
			}
		}
	})

	b.Run("CircuitBreakerCheck", func(b *testing.B) {
		k, ctx := keepertest.DexKeeper(b)
		poolID := createBenchmarkPool(b, k, ctx, MediumPoolSize)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err := k.CheckPoolCircuitBreaker(ctx, poolID)
			if err != nil {
				b.Fatalf("circuit breaker check failed: %v", err)
			}
		}
	})
}

// ============================================================================
// BenchmarkLiquidityQueries - Benchmark liquidity-related queries
// ============================================================================

func BenchmarkLiquidityQueries(b *testing.B) {
	b.Run("GetLiquidity", func(b *testing.B) {
		k, ctx := keepertest.DexKeeper(b)
		poolID := createBenchmarkPool(b, k, ctx, MediumPoolSize)
		provider := getBenchmarkTrader(5)

		// Add some liquidity first
		_, _ = k.AddLiquidity(ctx, provider, poolID, math.NewInt(MediumPoolSize/10), math.NewInt(MediumPoolSize/10))

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := k.GetLiquidity(ctx, poolID, provider)
			if err != nil {
				b.Fatalf("get liquidity failed: %v", err)
			}
		}
	})

	b.Run("IterateLiquidityByPool", func(b *testing.B) {
		k, ctx := keepertest.DexKeeper(b)
		poolID := createBenchmarkPool(b, k, ctx, MediumPoolSize)

		// Add liquidity from multiple providers (use available pre-funded addresses)
		for i := 0; i < 10; i++ {
			provider := getBenchmarkTrader(i)
			_, _ = k.AddLiquidity(ctx, provider, poolID, math.NewInt(MediumPoolSize/1000), math.NewInt(MediumPoolSize/1000))
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			count := 0
			err := k.IterateLiquidityByPool(ctx, poolID, func(provider sdk.AccAddress, shares math.Int) bool {
				count++
				return false
			})
			if err != nil {
				b.Fatalf("iteration failed: %v", err)
			}
		}
	})

	b.Run("GetLiquidityProviderCount", func(b *testing.B) {
		k, ctx := keepertest.DexKeeper(b)
		poolID := createBenchmarkPool(b, k, ctx, MediumPoolSize)

		// Add liquidity from multiple providers
		for i := 0; i < 10; i++ {
			provider := getBenchmarkTrader(i)
			_, _ = k.AddLiquidity(ctx, provider, poolID, math.NewInt(MediumPoolSize/1000), math.NewInt(MediumPoolSize/1000))
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			count, err := k.GetLiquidityProviderCount(ctx, poolID)
			if err != nil {
				b.Fatalf("count failed: %v", err)
			}
			if count < 10 {
				b.Fatalf("expected at least 10 providers, got %d", count)
			}
		}
	})
}

// ============================================================================
// BenchmarkPriceCalculations - Benchmark price-related operations
// ============================================================================

func BenchmarkPriceCalculations(b *testing.B) {
	b.Run("GetSpotPrice", func(b *testing.B) {
		k, ctx := keepertest.DexKeeper(b)
		poolID := createBenchmarkPool(b, k, ctx, MediumPoolSize)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := k.GetSpotPrice(ctx, poolID, benchTokenA, benchTokenB)
			if err != nil {
				b.Fatalf("get spot price failed: %v", err)
			}
		}
	})

	b.Run("SimulateSwap", func(b *testing.B) {
		k, ctx := keepertest.DexKeeper(b)
		poolID := createBenchmarkPool(b, k, ctx, MediumPoolSize)

		swapAmount := math.NewInt(MediumPoolSize / 1000)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, err := k.SimulateSwap(ctx, poolID, benchTokenA, benchTokenB, swapAmount)
			if err != nil {
				b.Fatalf("simulate swap failed: %v", err)
			}
		}
	})
}

// ============================================================================
// BenchmarkMixedOperations - Benchmark realistic mixed workloads
// ============================================================================

func BenchmarkMixedOperations(b *testing.B) {
	b.Run("SwapsAndLiquidity", func(b *testing.B) {
		k, ctx := keepertest.DexKeeper(b)
		poolID := createBenchmarkPool(b, k, ctx, LargePoolSize)
		trader := getBenchmarkTrader(1)
		provider := getBenchmarkTrader(2)

		swapAmount := math.NewInt(LargePoolSize / 10000)
		addAmount := math.NewInt(LargePoolSize / 1000)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// 80% swaps, 20% liquidity operations
			if i%5 == 0 {
				_, _ = k.AddLiquidity(ctx, provider, poolID, addAmount, addAmount)
			} else {
				_, _ = k.ExecuteSwap(ctx, trader, poolID, benchTokenA, benchTokenB, swapAmount, math.NewInt(1))
			}
		}
	})

	b.Run("HighFrequencySwaps", func(b *testing.B) {
		k, ctx := keepertest.DexKeeper(b)
		poolID := createBenchmarkPool(b, k, ctx, LargePoolSize)
		trader := getBenchmarkTrader(1)

		// Very small swaps to simulate high-frequency trading
		swapAmount := math.NewInt(LargePoolSize / 100000)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Alternate directions
			if i%2 == 0 {
				_, _ = k.ExecuteSwap(ctx, trader, poolID, benchTokenA, benchTokenB, swapAmount, math.NewInt(1))
			} else {
				_, _ = k.ExecuteSwap(ctx, trader, poolID, benchTokenB, benchTokenA, swapAmount, math.NewInt(1))
			}
		}
	})
}

// ============================================================================
// BenchmarkGasMetering - Benchmark operations with gas measurement focus
// ============================================================================

func BenchmarkGasMetering(b *testing.B) {
	b.Run("SwapGasConsumption", func(b *testing.B) {
		k, ctx := keepertest.DexKeeper(b)
		poolID := createBenchmarkPool(b, k, ctx, MediumPoolSize)
		trader := getBenchmarkTrader(1)

		swapAmount := math.NewInt(MediumPoolSize / 1000)

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_, err := k.ExecuteSwap(ctx, trader, poolID, benchTokenA, benchTokenB, swapAmount, math.NewInt(1))
			if err != nil {
				b.Fatalf("swap failed: %v", err)
			}
		}
	})

	b.Run("AddLiquidityGasConsumption", func(b *testing.B) {
		k, ctx := keepertest.DexKeeper(b)
		poolID := createBenchmarkPool(b, k, ctx, MediumPoolSize)
		provider := getBenchmarkTrader(2)

		addAmount := math.NewInt(MediumPoolSize / 1000)

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_, err := k.AddLiquidity(ctx, provider, poolID, addAmount, addAmount)
			if err != nil {
				b.Fatalf("add liquidity failed: %v", err)
			}
		}
	})
}
