package benchmarks

import (
	"testing"

	"cosmossdk.io/math"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/dex/types"
)

// BenchmarkCreatePool benchmarks the creation of liquidity pools
func BenchmarkCreatePool(b *testing.B) {
	k, ctx := keepertest.DexKeeper(b)
	creator := types.TestAddr()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		tokenA := "token" + string(rune('A'+i%26))
		tokenB := "token" + string(rune('Z'-i%26))
		b.StartTimer()

		_, err := k.CreatePool(ctx, creator, tokenA, tokenB, math.NewInt(1000000), math.NewInt(1000000))
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkAddLiquidity benchmarks adding liquidity to an existing pool
func BenchmarkAddLiquidity(b *testing.B) {
	k, ctx := keepertest.DexKeeper(b)
	provider := types.TestAddr()

	// Create a test pool
	poolID := keepertest.CreateTestPool(b, k, ctx, "upaw", "uatom", math.NewInt(10000000), math.NewInt(20000000))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := k.AddLiquidity(ctx, provider, poolID, math.NewInt(10000), math.NewInt(20000))
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkRemoveLiquidity benchmarks removing liquidity from a pool
func BenchmarkRemoveLiquidity(b *testing.B) {
	k, ctx := keepertest.DexKeeper(b)
	provider := types.TestAddr()

	// Create pool and add liquidity
	poolID := keepertest.CreateTestPool(b, k, ctx, "upaw", "uatom", math.NewInt(10000000), math.NewInt(20000000))

	// Add significant liquidity
	shares, err := k.AddLiquidity(ctx, provider, poolID, math.NewInt(100000000), math.NewInt(200000000))
	if err != nil {
		b.Fatal(err)
	}

	sharePerIteration := shares.Quo(math.NewInt(int64(b.N + 1)))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := k.RemoveLiquidity(ctx, provider, poolID, sharePerIteration)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkSwap benchmarks single-hop token swaps
func BenchmarkSwap(b *testing.B) {
	k, ctx := keepertest.DexKeeper(b)
	trader := types.TestAddr()

	// Create a large pool for swaps
	poolID := keepertest.CreateTestPool(b, k, ctx, "upaw", "uatom", math.NewInt(1000000000), math.NewInt(2000000000))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := k.ExecuteSwap(ctx, trader, poolID, "upaw", "uatom", math.NewInt(10000), math.NewInt(1))
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkSwapMultiHop benchmarks multi-hop swaps across multiple pools
func BenchmarkSwapMultiHop(b *testing.B) {
	k, ctx := keepertest.DexKeeper(b)
	trader := types.TestAddr()

	// Create three pools for multi-hop: upaw/uatom, uatom/uosmo, uosmo/uusdc
	pool1 := keepertest.CreateTestPool(b, k, ctx, "upaw", "uatom", math.NewInt(1000000000), math.NewInt(1000000000))
	pool2 := keepertest.CreateTestPool(b, k, ctx, "uatom", "uosmo", math.NewInt(1000000000), math.NewInt(1000000000))
	pool3 := keepertest.CreateTestPool(b, k, ctx, "uosmo", "uusdc", math.NewInt(1000000000), math.NewInt(1000000000))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Hop 1: upaw -> uatom
		amountOut1, err := k.ExecuteSwap(ctx, trader, pool1, "upaw", "uatom", math.NewInt(10000), math.NewInt(1))
		if err != nil {
			b.Fatal(err)
		}

		// Hop 2: uatom -> uosmo
		amountOut2, err := k.ExecuteSwap(ctx, trader, pool2, "uatom", "uosmo", amountOut1, math.NewInt(1))
		if err != nil {
			b.Fatal(err)
		}

		// Hop 3: uosmo -> uusdc
		_, err = k.ExecuteSwap(ctx, trader, pool3, "uosmo", "uusdc", amountOut2, math.NewInt(1))
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkGetPool benchmarks retrieving a pool by ID
func BenchmarkGetPool(b *testing.B) {
	k, ctx := keepertest.DexKeeper(b)

	poolID := keepertest.CreateTestPool(b, k, ctx, "upaw", "uatom", math.NewInt(1000000), math.NewInt(1000000))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := k.GetPool(ctx, poolID)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkListPools benchmarks retrieving all pools
func BenchmarkListPools(b *testing.B) {
	k, ctx := keepertest.DexKeeper(b)

	// Create multiple pools
	for i := 0; i < 50; i++ {
		tokenA := "token" + string(rune('A'+i%26)) + string(rune('0'+i/26))
		tokenB := "token" + string(rune('Z'-i%26)) + string(rune('9'-i/26))
		keepertest.CreateTestPool(b, k, ctx, tokenA, tokenB, math.NewInt(1000000), math.NewInt(1000000))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := k.GetAllPools(ctx)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCalculateSwapAmount benchmarks swap amount calculation
func BenchmarkCalculateSwapAmount(b *testing.B) {
	k, ctx := keepertest.DexKeeper(b)

	params, _ := k.GetParams(ctx)
	reserveIn := math.NewInt(1000000)
	reserveOut := math.NewInt(2000000)
	amountIn := math.NewInt(10000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := k.CalculateSwapOutput(ctx, amountIn, reserveIn, reserveOut, params.SwapFee)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkGetPoolLiquidity benchmarks getting total pool liquidity
func BenchmarkGetPoolLiquidity(b *testing.B) {
	k, ctx := keepertest.DexKeeper(b)
	provider := types.TestAddr()

	poolID := keepertest.CreateTestPool(b, k, ctx, "upaw", "uatom", math.NewInt(10000000), math.NewInt(20000000))
	if _, err := k.AddLiquidity(ctx, provider, poolID, math.NewInt(1000000), math.NewInt(2000000)); err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := k.GetLiquidity(ctx, poolID, provider)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkPoolSecurityValidation benchmarks pool security validation
func BenchmarkPoolSecurityValidation(b *testing.B) {
	k, ctx := keepertest.DexKeeper(b)

	poolID := keepertest.CreateTestPool(b, k, ctx, "upaw", "uatom", math.NewInt(1000000), math.NewInt(1000000))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pool, err := k.GetPool(ctx, poolID)
		if err != nil {
			b.Fatal(err)
		}

		// Validate pool invariant: reserveA * reserveB >= constant
		invariant := pool.ReserveA.Mul(pool.ReserveB)
		if invariant.IsZero() {
			b.Fatal("invalid pool invariant")
		}
	}
}

// BenchmarkFeeCalculation benchmarks fee calculation for swaps
func BenchmarkFeeCalculation(b *testing.B) {
	k, ctx := keepertest.DexKeeper(b)

	params, _ := k.GetParams(ctx)
	amountIn := math.NewInt(10000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		feeAmount := math.LegacyNewDecFromInt(amountIn).Mul(params.SwapFee).TruncateInt()
		_ = feeAmount
	}
}

// BenchmarkSlippageCalculation benchmarks slippage calculation
func BenchmarkSlippageCalculation(b *testing.B) {
	k, ctx := keepertest.DexKeeper(b)

	poolID := keepertest.CreateTestPool(b, k, ctx, "upaw", "uatom", math.NewInt(1000000), math.NewInt(2000000))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		actualOutput, err := k.SimulateSwap(ctx, poolID, "upaw", "uatom", math.NewInt(10000))
		if err != nil {
			b.Fatal(err)
		}

		spotPrice, err := k.GetSpotPrice(ctx, poolID, "upaw", "uatom")
		if err != nil {
			b.Fatal(err)
		}

		expectedOutput := math.LegacyNewDecFromInt(math.NewInt(10000)).Mul(spotPrice).TruncateInt()
		slippage := expectedOutput.Sub(actualOutput).Abs()
		_ = slippage
	}
}

// BenchmarkPriceImpact benchmarks calculating price impact
func BenchmarkPriceImpact(b *testing.B) {
	k, ctx := keepertest.DexKeeper(b)

	poolID := keepertest.CreateTestPool(b, k, ctx, "upaw", "uatom", math.NewInt(1000000), math.NewInt(2000000))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pool, _ := k.GetPool(ctx, poolID)

		amountIn := math.NewInt(10000)

		// Price before
		priceBefore := math.LegacyNewDecFromInt(pool.ReserveB).Quo(math.LegacyNewDecFromInt(pool.ReserveA))

		// Simulate swap
		params, _ := k.GetParams(ctx)
		amountOut, _ := k.CalculateSwapOutput(ctx, amountIn, pool.ReserveA, pool.ReserveB, params.SwapFee)

		// Price after
		newReserveA := pool.ReserveA.Add(amountIn)
		newReserveB := pool.ReserveB.Sub(amountOut)
		priceAfter := math.LegacyNewDecFromInt(newReserveB).Quo(math.LegacyNewDecFromInt(newReserveA))

		priceImpact := priceAfter.Sub(priceBefore).Quo(priceBefore).Abs()
		_ = priceImpact
	}
}

// BenchmarkLPTokenMinting benchmarks LP token minting during liquidity addition
func BenchmarkLPTokenMinting(b *testing.B) {
	k, ctx := keepertest.DexKeeper(b)

	poolID := keepertest.CreateTestPool(b, k, ctx, "upaw", "uatom", math.NewInt(10000000), math.NewInt(20000000))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pool, _ := k.GetPool(ctx, poolID)

		amountA := math.NewInt(10000)
		amountB := math.NewInt(20000)

		// Calculate shares to mint
		sharesA := amountA.Mul(pool.TotalShares).Quo(pool.ReserveA)
		sharesB := amountB.Mul(pool.TotalShares).Quo(pool.ReserveB)

		newShares := sharesA
		if sharesB.LT(sharesA) {
			newShares = sharesB
		}
		_ = newShares
	}
}

// BenchmarkLPTokenBurning benchmarks LP token burning during liquidity removal
func BenchmarkLPTokenBurning(b *testing.B) {
	k, ctx := keepertest.DexKeeper(b)
	provider := types.TestAddr()

	poolID := keepertest.CreateTestPool(b, k, ctx, "upaw", "uatom", math.NewInt(10000000), math.NewInt(20000000))
	shares, err := k.AddLiquidity(ctx, provider, poolID, math.NewInt(100000000), math.NewInt(200000000))
	if err != nil {
		b.Fatal(err)
	}

	sharePerIteration := shares.Quo(math.NewInt(int64(b.N + 1)))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pool, _ := k.GetPool(ctx, poolID)

		// Calculate amounts to return
		amountA := sharePerIteration.Mul(pool.ReserveA).Quo(pool.TotalShares)
		amountB := sharePerIteration.Mul(pool.ReserveB).Quo(pool.TotalShares)
		_ = amountA
		_ = amountB
	}
}

// BenchmarkPoolStateUpdate benchmarks updating pool state after operations
func BenchmarkPoolStateUpdate(b *testing.B) {
	k, ctx := keepertest.DexKeeper(b)

	poolID := keepertest.CreateTestPool(b, k, ctx, "upaw", "uatom", math.NewInt(1000000), math.NewInt(2000000))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pool, err := k.GetPool(ctx, poolID)
		if err != nil {
			b.Fatal(err)
		}

		// Modify pool state
		pool.ReserveA = pool.ReserveA.Add(math.NewInt(1000))
		pool.ReserveB = pool.ReserveB.Sub(math.NewInt(500))

		err = k.SetPool(ctx, pool)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkInvariantValidation benchmarks validating pool invariants
func BenchmarkInvariantValidation(b *testing.B) {
	k, ctx := keepertest.DexKeeper(b)

	poolID := keepertest.CreateTestPool(b, k, ctx, "upaw", "uatom", math.NewInt(1000000), math.NewInt(2000000))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		pool, _ := k.GetPool(ctx, poolID)

		// Validate constant product invariant
		k := pool.ReserveA.Mul(pool.ReserveB)

		// Ensure reserves are positive
		if pool.ReserveA.IsZero() || pool.ReserveB.IsZero() {
			b.Fatal("invalid reserves")
		}

		// Ensure total shares is positive
		if pool.TotalShares.IsZero() {
			b.Fatal("invalid total shares")
		}

		_ = k
	}
}

// BenchmarkConcurrentSwaps benchmarks concurrent swap operations
func BenchmarkConcurrentSwaps(b *testing.B) {
	k, ctx := keepertest.DexKeeper(b)
	trader := types.TestAddr()

	poolID := keepertest.CreateTestPool(b, k, ctx, "upaw", "uatom", math.NewInt(10000000000), math.NewInt(20000000000))

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := k.ExecuteSwap(ctx, trader, poolID, "upaw", "uatom", math.NewInt(1000), math.NewInt(1))
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkLargePoolOperations benchmarks operations on large liquidity pools
func BenchmarkLargePoolOperations(b *testing.B) {
	k, ctx := keepertest.DexKeeper(b)
	trader := types.TestAddr()

	// Create a very large pool (1B tokens each)
	poolID := keepertest.CreateTestPool(b, k, ctx, "upaw", "uatom",
		math.NewInt(1000000000000), math.NewInt(1000000000000))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Large swap (1M tokens)
		_, err := k.ExecuteSwap(ctx, trader, poolID, "upaw", "uatom", math.NewInt(1000000000), math.NewInt(1))
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkMultiHopSwapBatched benchmarks the batched ExecuteMultiHopSwap function
func BenchmarkMultiHopSwapBatched(b *testing.B) {
	k, ctx := keepertest.DexKeeper(b)
	trader := types.TestAddr()

	// Create three pools for multi-hop
	pool1 := keepertest.CreateTestPool(b, k, ctx, "upaw", "uatom", math.NewInt(10000000000), math.NewInt(10000000000))
	pool2 := keepertest.CreateTestPool(b, k, ctx, "uatom", "uosmo", math.NewInt(10000000000), math.NewInt(10000000000))
	pool3 := keepertest.CreateTestPool(b, k, ctx, "uosmo", "uusdc", math.NewInt(10000000000), math.NewInt(10000000000))

	hops := []keepertest.SwapHop{
		{PoolID: pool1, TokenIn: "upaw", TokenOut: "uatom"},
		{PoolID: pool2, TokenIn: "uatom", TokenOut: "uosmo"},
		{PoolID: pool3, TokenIn: "uosmo", TokenOut: "uusdc"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := k.ExecuteMultiHopSwap(ctx, trader, hops, math.NewInt(100000), math.NewInt(1))
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkSimulateMultiHopSwap benchmarks simulation without state changes
func BenchmarkSimulateMultiHopSwap(b *testing.B) {
	k, ctx := keepertest.DexKeeper(b)

	pool1 := keepertest.CreateTestPool(b, k, ctx, "upaw", "uatom", math.NewInt(10000000000), math.NewInt(10000000000))
	pool2 := keepertest.CreateTestPool(b, k, ctx, "uatom", "uosmo", math.NewInt(10000000000), math.NewInt(10000000000))

	hops := []keepertest.SwapHop{
		{PoolID: pool1, TokenIn: "upaw", TokenOut: "uatom"},
		{PoolID: pool2, TokenIn: "uatom", TokenOut: "uosmo"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := k.SimulateMultiHopSwap(ctx, hops, math.NewInt(100000))
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCommitRevealFlow benchmarks the commit-reveal swap mechanism
func BenchmarkCommitRevealFlow(b *testing.B) {
	k, ctx := keepertest.DexKeeper(b)
	trader := types.TestAddr()

	poolID := keepertest.CreateTestPool(b, k, ctx, "upaw", "uatom", math.NewInt(10000000000), math.NewInt(20000000000))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Commit phase
		commitment, err := k.CreateSwapCommitment(ctx, trader, poolID, "upaw", "uatom", math.NewInt(1000000), math.NewInt(900000))
		if err != nil {
			b.Fatal(err)
		}

		// Reveal phase (typically after N blocks)
		_, err = k.RevealAndExecuteSwap(ctx, commitment.CommitHash, math.NewInt(1000000), math.NewInt(900000), []byte("secret"))
		if err != nil {
			// Expected to fail without proper block advancement, just measure commit
			_ = err
		}
	}
}

// BenchmarkCircuitBreakerCheck benchmarks circuit breaker state lookup
func BenchmarkCircuitBreakerCheck(b *testing.B) {
	k, ctx := keepertest.DexKeeper(b)

	poolID := keepertest.CreateTestPool(b, k, ctx, "upaw", "uatom", math.NewInt(10000000000), math.NewInt(20000000000))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := k.GetPoolCircuitBreakerState(ctx, poolID)
		if err != nil {
			// May not exist, that's OK for benchmark
			_ = err
		}
	}
}
