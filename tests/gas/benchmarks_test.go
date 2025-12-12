package gas

import (
	"fmt"
	"testing"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
)

// Gas Benchmarking Suite
// Measures gas consumption across many iterations to establish baselines

func BenchmarkGas_DEX_CreatePool(b *testing.B) {
	rawKeeper, ctx := keepertest.DexKeeper(b)
	k := NewDexGasKeeper(rawKeeper)

	creator := sdk.AccAddress("creator1___________")
	var totalGas uint64

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx = ctx.WithGasMeter(storetypes.NewGasMeter(10000000))

		tokenA := fmt.Sprintf("token%da", i)
		tokenB := fmt.Sprintf("token%db", i)

		_, err := k.CreatePool(ctx, creator.String(), tokenA, tokenB,
			math.NewInt(1000000), math.NewInt(2000000))
		if err != nil {
			b.Fatalf("CreatePool failed: %v", err)
		}

		totalGas += ctx.GasMeter().GasConsumed()
	}

	avgGas := totalGas / uint64(b.N)
	b.ReportMetric(float64(avgGas), "gas/op")
	b.Logf("Average gas per CreatePool: %d", avgGas)
}

func BenchmarkGas_DEX_Swap(b *testing.B) {
	rawKeeper, ctx := keepertest.DexKeeper(b)
	k := NewDexGasKeeper(rawKeeper)

	// Setup: Create pool
	creator := sdk.AccAddress("creator1___________")
	poolID, err := k.CreatePool(ctx, creator.String(), "upaw", "uusdt",
		math.NewInt(1000000000), math.NewInt(2000000000))
	if err != nil {
		b.Fatalf("Setup failed: %v", err)
	}

	trader := sdk.AccAddress("trader1____________")
	var totalGas uint64

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx = ctx.WithGasMeter(storetypes.NewGasMeter(10000000))

		_, err := k.Swap(ctx, trader.String(), poolID, "upaw",
			math.NewInt(1000000), math.NewInt(1))
		if err != nil {
			b.Fatalf("Swap failed: %v", err)
		}

		totalGas += ctx.GasMeter().GasConsumed()
	}

	avgGas := totalGas / uint64(b.N)
	b.ReportMetric(float64(avgGas), "gas/op")
	b.Logf("Average gas per Swap: %d", avgGas)
}

func BenchmarkGas_DEX_AddLiquidity(b *testing.B) {
	rawKeeper, ctx := keepertest.DexKeeper(b)
	k := NewDexGasKeeper(rawKeeper)

	// Setup: Create pool
	creator := sdk.AccAddress("creator1___________")
	poolID, err := k.CreatePool(ctx, creator.String(), "upaw", "uusdt",
		math.NewInt(1000000000), math.NewInt(2000000000))
	if err != nil {
		b.Fatalf("Setup failed: %v", err)
	}

	provider := sdk.AccAddress("provider1__________")
	var totalGas uint64

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx = ctx.WithGasMeter(storetypes.NewGasMeter(10000000))

		_, err := k.AddLiquidity(ctx, provider.String(), poolID,
			math.NewInt(1000000), math.NewInt(2000000), math.NewInt(1))
		if err != nil {
			b.Fatalf("AddLiquidity failed: %v", err)
		}

		totalGas += ctx.GasMeter().GasConsumed()
	}

	avgGas := totalGas / uint64(b.N)
	b.ReportMetric(float64(avgGas), "gas/op")
	b.Logf("Average gas per AddLiquidity: %d", avgGas)
}

func BenchmarkGas_DEX_RemoveLiquidity(b *testing.B) {
	rawKeeper, ctx := keepertest.DexKeeper(b)
	k := NewDexGasKeeper(rawKeeper)

	// Setup: Create pool and add liquidity
	creator := sdk.AccAddress("creator1___________")
	poolID, err := k.CreatePool(ctx, creator.String(), "upaw", "uusdt",
		math.NewInt(1000000000), math.NewInt(2000000000))
	if err != nil {
		b.Fatalf("Setup failed: %v", err)
	}

	provider := sdk.AccAddress("provider1__________")
	lpTokens, err := k.AddLiquidity(ctx, provider.String(), poolID,
		math.NewInt(100000000), math.NewInt(200000000), math.NewInt(1))
	if err != nil {
		b.Fatalf("Setup failed: %v", err)
	}

	removeAmount := lpTokens.Quo(math.NewInt(1000)) // Remove small amount each time
	var totalGas uint64

	b.ResetTimer()
	for i := 0; i < b.N && i < 100; i++ { // Limit iterations
		ctx = ctx.WithGasMeter(storetypes.NewGasMeter(10000000))

		_, _, err := k.RemoveLiquidity(ctx, provider.String(), poolID,
			removeAmount, math.NewInt(1), math.NewInt(1))
		if err != nil {
			b.Fatalf("RemoveLiquidity failed: %v", err)
		}

		totalGas += ctx.GasMeter().GasConsumed()
	}

	avgGas := totalGas / uint64(b.N)
	b.ReportMetric(float64(avgGas), "gas/op")
	b.Logf("Average gas per RemoveLiquidity: %d", avgGas)
}

func BenchmarkGas_Oracle_SubmitPrice(b *testing.B) {
	rawKeeper, _, ctx := keepertest.OracleKeeper(b)
	k := NewOracleGasKeeper(rawKeeper)

	// Setup: Register oracle
	oracle := sdk.AccAddress("oracle1_____________")
	err := k.RegisterOracle(ctx, oracle.String())
	if err != nil {
		b.Fatalf("Setup failed: %v", err)
	}

	var totalGas uint64
	price := math.LegacyNewDec(50000)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx = ctx.WithGasMeter(storetypes.NewGasMeter(10000000))

		asset := fmt.Sprintf("ASSET_%d", i%10) // Rotate through 10 assets

		err := k.SubmitPrice(ctx, oracle.String(), asset, price)
		if err != nil {
			b.Fatalf("SubmitPrice failed: %v", err)
		}

		totalGas += ctx.GasMeter().GasConsumed()
	}

	avgGas := totalGas / uint64(b.N)
	b.ReportMetric(float64(avgGas), "gas/op")
	b.Logf("Average gas per SubmitPrice: %d", avgGas)
}

func BenchmarkGas_Oracle_AggregatePrices(b *testing.B) {
	// Test with different validator counts
	validatorCounts := []int{7, 21, 50, 100}

	for _, numValidators := range validatorCounts {
		b.Run(fmt.Sprintf("validators_%d", numValidators), func(b *testing.B) {
			rawKeeper, _, ctx := keepertest.OracleKeeper(b)
			k := NewOracleGasKeeper(rawKeeper)

			// Setup: Register oracles and submit prices
			asset := "BTC"
			basePrice := math.LegacyNewDec(50000)

			oracles := make([]sdk.AccAddress, numValidators)
			for i := 0; i < numValidators; i++ {
				oracle := sdk.AccAddress(fmt.Sprintf("oracle%d____________", i))
				oracles[i] = oracle

				err := k.RegisterOracle(ctx, oracle.String())
				if err != nil {
					b.Fatalf("Setup failed: %v", err)
				}

				price := basePrice.Add(math.LegacyNewDec(int64(i * 10)))
				err = k.SubmitPrice(ctx, oracle.String(), asset, price)
				if err != nil {
					b.Fatalf("Setup failed: %v", err)
				}
			}

			var totalGas uint64

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				ctx = ctx.WithGasMeter(storetypes.NewGasMeter(10000000))

				err := k.AggregatePrices(ctx)
				if err != nil {
					b.Fatalf("AggregatePrices failed: %v", err)
				}

				totalGas += ctx.GasMeter().GasConsumed()
			}

			avgGas := totalGas / uint64(b.N)
			gasPerValidator := avgGas / uint64(numValidators)

			b.ReportMetric(float64(avgGas), "gas/op")
			b.ReportMetric(float64(gasPerValidator), "gas/validator")
			b.Logf("Average gas per AggregatePrices (%d validators): %d (%.0f gas/validator)",
				numValidators, avgGas, float64(gasPerValidator))
		})
	}
}

func BenchmarkGas_Compute_RegisterProvider(b *testing.B) {
	rawKeeper, ctx := keepertest.ComputeKeeper(b)
	k := NewComputeGasKeeper(rawKeeper)

	var totalGas uint64

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx = ctx.WithGasMeter(storetypes.NewGasMeter(10000000))

		provider := sdk.AccAddress(fmt.Sprintf("provider%d_________", i))

		err := k.RegisterProvider(ctx, provider.String(),
			fmt.Sprintf("Provider %d", i),
			fmt.Sprintf("https://provider%d.com", i),
			ResourceSpecs{
				CPUCores: 16,
				MemoryMB: 32768,
				DiskGB:   1000,
			})
		if err != nil {
			b.Fatalf("RegisterProvider failed: %v", err)
		}

		totalGas += ctx.GasMeter().GasConsumed()
	}

	avgGas := totalGas / uint64(b.N)
	b.ReportMetric(float64(avgGas), "gas/op")
	b.Logf("Average gas per RegisterProvider: %d", avgGas)
}

func BenchmarkGas_Compute_SubmitRequest(b *testing.B) {
	rawKeeper, ctx := keepertest.ComputeKeeper(b)
	k := NewComputeGasKeeper(rawKeeper)

	// Setup: Register provider
	provider := sdk.AccAddress("provider1__________")
	err := k.RegisterProvider(ctx, provider.String(), "Test Provider", "https://test.com",
		ResourceSpecs{
			CPUCores: 16,
			MemoryMB: 32768,
		})
	if err != nil {
		b.Fatalf("Setup failed: %v", err)
	}

	// Test with different input sizes
	inputSizes := []int{1024, 10240, 102400} // 1KB, 10KB, 100KB

	for _, size := range inputSizes {
		b.Run(fmt.Sprintf("input_%dKB", size/1024), func(b *testing.B) {
			requester := sdk.AccAddress("requester1_________")
			input := make([]byte, size)
			var totalGas uint64

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				ctx = ctx.WithGasMeter(storetypes.NewGasMeter(10000000))

				_, err := k.SubmitRequest(ctx, requester.String(), provider.String(),
					input, ResourceRequirements{
						CPUCores: 4,
						MemoryMB: 8192,
					})
				if err != nil {
					b.Fatalf("SubmitRequest failed: %v", err)
				}

				totalGas += ctx.GasMeter().GasConsumed()
			}

			avgGas := totalGas / uint64(b.N)
			gasPerKB := avgGas / uint64(size/1024)

			b.ReportMetric(float64(avgGas), "gas/op")
			b.ReportMetric(float64(gasPerKB), "gas/KB")
			b.Logf("Average gas per SubmitRequest (%dKB): %d (%.0f gas/KB)",
				size/1024, avgGas, float64(gasPerKB))
		})
	}
}

func BenchmarkGas_Compute_SubmitResult(b *testing.B) {
	rawKeeper, ctx := keepertest.ComputeKeeper(b)
	k := NewComputeGasKeeper(rawKeeper)

	// Setup
	provider := sdk.AccAddress("provider1__________")
	err := k.RegisterProvider(ctx, provider.String(), "Test", "https://test.com",
		ResourceSpecs{CPUCores: 16, MemoryMB: 32768})
	if err != nil {
		b.Fatalf("Setup failed: %v", err)
	}

	requester := sdk.AccAddress("requester1_________")

	// Create requests
	requestIDs := make([]string, b.N)
	for i := 0; i < b.N && i < 100; i++ {
		requestID, err := k.SubmitRequest(ctx, requester.String(), provider.String(),
			[]byte("test"), ResourceRequirements{CPUCores: 4, MemoryMB: 8192})
		if err != nil {
			b.Fatalf("Setup failed: %v", err)
		}
		requestIDs[i] = requestID
	}

	result := make([]byte, 1024)
	proof := make([]byte, 256)
	var totalGas uint64

	b.ResetTimer()
	for i := 0; i < b.N && i < len(requestIDs); i++ {
		ctx = ctx.WithGasMeter(storetypes.NewGasMeter(10000000))

		err := k.SubmitResult(ctx, requestIDs[i], provider.String(), result, proof)
		if err != nil {
			b.Fatalf("SubmitResult failed: %v", err)
		}

		totalGas += ctx.GasMeter().GasConsumed()
	}

	avgGas := totalGas / uint64(b.N)
	b.ReportMetric(float64(avgGas), "gas/op")
	b.Logf("Average gas per SubmitResult: %d", avgGas)
}

// Comparison benchmarks

func BenchmarkGas_Comparison_StateRead(b *testing.B) {
	rawKeeper, ctx := keepertest.DexKeeper(b)
	k := NewDexGasKeeper(rawKeeper)

	// Create pool
	creator := sdk.AccAddress("creator1___________")
	poolID, _ := k.CreatePool(ctx, creator.String(), "upaw", "uusdt",
		math.NewInt(1000000), math.NewInt(2000000))

	var totalGas uint64

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx = ctx.WithGasMeter(storetypes.NewGasMeter(100000))

		_, _ = k.GetPool(ctx, poolID)

		totalGas += ctx.GasMeter().GasConsumed()
	}

	avgGas := totalGas / uint64(b.N)
	b.ReportMetric(float64(avgGas), "gas/op")
	b.Logf("Average gas per state read: %d", avgGas)
}

func BenchmarkGas_Comparison_StateWrite(b *testing.B) {
	rawKeeper, ctx := keepertest.ComputeKeeper(b)
	k := NewComputeGasKeeper(rawKeeper)

	var totalGas uint64

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx = ctx.WithGasMeter(storetypes.NewGasMeter(500000))

		provider := sdk.AccAddress(fmt.Sprintf("provider%d_________", i))
		_ = k.RegisterProvider(ctx, provider.String(), "Test", "https://test.com",
			ResourceSpecs{CPUCores: 16, MemoryMB: 32768})

		totalGas += ctx.GasMeter().GasConsumed()
	}

	avgGas := totalGas / uint64(b.N)
	b.ReportMetric(float64(avgGas), "gas/op")
	b.Logf("Average gas per state write: %d", avgGas)
}

// Memory allocation benchmarks

func BenchmarkGas_MemoryAllocation(b *testing.B) {
	sizes := []int{1024, 10240, 102400, 1024000} // 1KB to 1MB

	for _, size := range sizes {
		b.Run(fmt.Sprintf("alloc_%dKB", size/1024), func(b *testing.B) {
			b.ReportAllocs()
			b.ResetTimer()

			for i := 0; i < b.N; i++ {
				data := make([]byte, size)
				_ = data
			}

			b.Logf("Memory allocation for %d bytes", size)
		})
	}
}
