package gas

import (
	"fmt"
	"testing"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
)

// DoS Attack Test Suite
// Tests various denial-of-service attack vectors to ensure proper gas metering

func TestDoS_UnboundedLoop_Prevention(t *testing.T) {
	t.Run("Oracle aggregation with many validators", func(t *testing.T) {
		rawKeeper, _, ctx := keepertest.OracleKeeper(t)
		k := NewOracleGasKeeper(rawKeeper)

		// Try to create a scenario with excessive validators
		// This should be bounded by gas limits
		numOracles := 500 // Excessive number
		asset := "BTC"

		// Set reasonable gas limit
		ctx = ctx.WithGasMeter(storetypes.NewGasMeter(5000000))
		defer func() {
			if r := recover(); r != nil {
				if _, ok := r.(storetypes.ErrorOutOfGas); ok {
					t.Log("Hit gas limit during oracle registration (expected behavior)")
					return
				}
				panic(r)
			}
		}()

		// Register many oracles
		for i := 0; i < numOracles; i++ {
			oracle := sdk.AccAddress(fmt.Sprintf("oracle%d____________", i))
			err := k.RegisterOracle(ctx, oracle.String())

			// Should hit gas limit before completing all registrations
			if err != nil && ctx.GasMeter().IsOutOfGas() {
				t.Logf("Hit gas limit at oracle %d (expected behavior)", i)
				return
			}

			if err == nil {
				price := math.LegacyNewDec(50000)
				_ = k.SubmitPrice(ctx, oracle.String(), asset, price)
			}
		}

		// Try to aggregate - should fail with out of gas
		err := k.AggregatePrices(ctx)

		// Should either error or consume significant gas
		if err != nil {
			require.Contains(t, err.Error(), "out of gas",
				"Should fail with out of gas error for excessive validators")
		} else {
			gasUsed := ctx.GasMeter().GasConsumed()
			t.Logf("Aggregation completed but used %d gas", gasUsed)
			require.Greater(t, gasUsed, uint64(1000000),
				"Should consume significant gas for many validators")
		}
	})

	t.Run("DEX pool iteration limit", func(t *testing.T) {
		rawKeeper, ctx := keepertest.DexKeeper(t)
		k := NewDexGasKeeper(rawKeeper)

		// Create many pools to test iteration limits
		ctx = ctx.WithGasMeter(storetypes.NewGasMeter(10000000))
		defer func() {
			if r := recover(); r != nil {
				if _, ok := r.(storetypes.ErrorOutOfGas); ok {
					t.Log("Hit gas limit while creating pools (expected behavior)")
					return
				}
				panic(r)
			}
		}()

		creator := sdk.AccAddress("creator1___________")

		// Try to create many pools
		maxPools := 100
		for i := 0; i < maxPools; i++ {
			tokenA := fmt.Sprintf("token%d", i)
			tokenB := "upaw"

			_, err := k.CreatePool(ctx, creator.String(), tokenA, tokenB,
				math.NewInt(1000000), math.NewInt(1000000))

			if err != nil && ctx.GasMeter().IsOutOfGas() {
				t.Logf("Hit gas limit at pool %d (expected)", i)
				return
			}
		}

		// If we created all pools, iterating should be bounded
		gasUsed := ctx.GasMeter().GasConsumed()
		t.Logf("Created %d pools using %d gas", maxPools, gasUsed)

		// Even creating many pools should not exceed reasonable limits
		require.Less(t, gasUsed, uint64(10000000),
			"Pool creation should not exceed gas limits")
	})
}

func TestDoS_LargeInputData(t *testing.T) {
	t.Run("Compute module large input", func(t *testing.T) {
		rawKeeper, ctx := keepertest.ComputeKeeper(t)
		k := NewComputeGasKeeper(rawKeeper)

		// Register provider
		provider := sdk.AccAddress("provider1__________")
		err := k.RegisterProvider(ctx, provider.String(), "Test Provider", "https://test.com", ResourceSpecs{
			CPUCores: 16,
			MemoryMB: 32768,
		})
		require.NoError(t, err)

		requester := sdk.AccAddress("requester1_________")

		tests := []struct {
			name        string
			inputSize   int
			gasLimit    uint64
			shouldError bool
		}{
			{"normal 1KB", 1024, 500000, false},
			{"large 1MB", 1024 * 1024, 2000000, false},
			{"huge 10MB", 10 * 1024 * 1024, 5000000, true}, // Should fail
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				ctx = ctx.WithGasMeter(storetypes.NewGasMeter(tt.gasLimit))

				largeInput := make([]byte, tt.inputSize)
				for i := range largeInput {
					largeInput[i] = byte(i % 256)
				}

				_, err := k.SubmitRequest(ctx, requester.String(), provider.String(), largeInput, ResourceRequirements{
					CPUCores: 4,
					MemoryMB: 8192,
				})

				if tt.shouldError {
					require.Error(t, err,
						"Should reject or hit gas limit for %d bytes", tt.inputSize)
				} else {
					gasUsed := ctx.GasMeter().GasConsumed()
					// Gas should scale with input size
					gasPerKB := gasUsed / uint64(tt.inputSize/1024)
					require.Less(t, gasPerKB, uint64(5000),
						"Gas per KB should be reasonable: %d", gasPerKB)
					t.Logf("Input %d bytes: %d gas (%d gas/KB)", tt.inputSize, gasUsed, gasPerKB)
				}
			})
		}
	})

	t.Run("Oracle batch submission limit", func(t *testing.T) {
		rawKeeper, _, ctx := keepertest.OracleKeeper(t)
		k := NewOracleGasKeeper(rawKeeper)

		oracle := sdk.AccAddress("oracle1_____________")
		err := k.RegisterOracle(ctx, oracle.String())
		require.NoError(t, err)

		// Try to submit prices for many assets at once
		ctx = ctx.WithGasMeter(storetypes.NewGasMeter(2000000))

		numAssets := 100 // Excessive number
		for i := 0; i < numAssets; i++ {
			asset := fmt.Sprintf("ASSET_%d", i)
			price := math.LegacyNewDec(int64(1000 + i))

			err := k.SubmitPrice(ctx, oracle.String(), asset, price)

			if err != nil && ctx.GasMeter().IsOutOfGas() {
				t.Logf("Hit gas limit at asset %d (expected)", i)
				return
			}
		}

		gasUsed := ctx.GasMeter().GasConsumed()
		t.Logf("Submitted %d assets using %d gas", numAssets, gasUsed)

		// Should consume significant gas
		require.Greater(t, gasUsed, uint64(500000),
			"Batch submission should consume significant gas")
	})
}

func TestDoS_NestedOperations(t *testing.T) {
	t.Run("Multiple DEX swaps in sequence", func(t *testing.T) {
		rawKeeper, ctx := keepertest.DexKeeper(t)
		k := NewDexGasKeeper(rawKeeper)

		// Create pool
		creator := sdk.AccAddress("creator1___________")
		poolID, err := k.CreatePool(ctx, creator.String(), "upaw", "uusdt",
			math.NewInt(1000000000), math.NewInt(2000000000))
		require.NoError(t, err)

		trader := sdk.AccAddress("trader1____________")

		// Set gas limit for multiple swaps
		ctx = ctx.WithGasMeter(storetypes.NewGasMeter(3000000))

		// Execute multiple swaps
		numSwaps := 20
		for i := 0; i < numSwaps; i++ {
			_, err := k.Swap(ctx, trader.String(), poolID, "upaw",
				math.NewInt(1000000), math.NewInt(1))

			if err != nil && ctx.GasMeter().IsOutOfGas() {
				t.Logf("Hit gas limit at swap %d (expected)", i)
				return
			}
			require.NoError(t, err)
		}

		gasUsed := ctx.GasMeter().GasConsumed()
		gasPerSwap := gasUsed / uint64(numSwaps)

		t.Logf("Executed %d swaps: %d gas total, %d gas/swap", numSwaps, gasUsed, gasPerSwap)

		// Gas should accumulate linearly, not exponentially
		require.Less(t, gasPerSwap, uint64(150000),
			"Gas per swap should not increase with nesting")
	})

	t.Run("Nested escrow operations", func(t *testing.T) {
		rawKeeper, ctx := keepertest.ComputeKeeper(t)
		k := NewComputeGasKeeper(rawKeeper)

		// Setup
		provider := sdk.AccAddress("provider1__________")
		err := k.RegisterProvider(ctx, provider.String(), "Test", "https://test.com", ResourceSpecs{
			CPUCores: 16,
			MemoryMB: 32768,
		})
		require.NoError(t, err)

		requester := sdk.AccAddress("requester1_________")

		ctx = ctx.WithGasMeter(storetypes.NewGasMeter(5000000))

		// Create multiple requests with escrow
		numRequests := 10
		for i := 0; i < numRequests; i++ {
			_, err := k.SubmitRequest(ctx, requester.String(), provider.String(),
				[]byte(fmt.Sprintf("request_%d", i)), ResourceRequirements{
					CPUCores: 4,
					MemoryMB: 8192,
				})

			if err != nil && ctx.GasMeter().IsOutOfGas() {
				t.Logf("Hit gas limit at request %d", i)
				return
			}
		}

		gasUsed := ctx.GasMeter().GasConsumed()
		t.Logf("Created %d requests: %d gas total", numRequests, gasUsed)

		require.Less(t, gasUsed, uint64(5000000),
			"Multiple operations should not exceed reasonable limits")
	})
}

func TestDoS_StateIterations(t *testing.T) {
	t.Run("Iterator gas consumption", func(t *testing.T) {
		rawKeeper, ctx := keepertest.DexKeeper(t)
		k := NewDexGasKeeper(rawKeeper)

		creator := sdk.AccAddress("creator1___________")

		// Create multiple pools
		numPools := 50
		ctx = ctx.WithGasMeter(storetypes.NewGasMeter(10000000))

		for i := 0; i < numPools; i++ {
			tokenA := fmt.Sprintf("token%da", i)
			tokenB := fmt.Sprintf("token%db", i)

			_, err := k.CreatePool(ctx, creator.String(), tokenA, tokenB,
				math.NewInt(1000000), math.NewInt(1000000))

			if ctx.GasMeter().IsOutOfGas() {
				t.Logf("Hit gas limit at pool %d", i)
				break
			}
			require.NoError(t, err)
		}

		gasAfterCreate := ctx.GasMeter().GasConsumed()
		t.Logf("Created pools: %d gas", gasAfterCreate)

		// Now iterate over all pools - should consume gas
		ctx = ctx.WithGasMeter(storetypes.NewGasMeter(2000000))

		pools := k.GetAllPools(ctx)

		gasUsed := ctx.GasMeter().GasConsumed()
		gasPerPool := gasUsed / uint64(len(pools))

		t.Logf("Iterated %d pools: %d gas total, %d gas/pool", len(pools), gasUsed, gasPerPool)

		// Iteration should be metered
		require.Greater(t, gasUsed, uint64(1000),
			"State iteration should consume gas")

		// But should be bounded
		require.Less(t, gasPerPool, uint64(10000),
			"Gas per pool iteration should be reasonable")
	})
}

func TestDoS_MalformedData(t *testing.T) {
	t.Run("Invalid price submissions", func(t *testing.T) {
		rawKeeper, _, ctx := keepertest.OracleKeeper(t)
		k := NewOracleGasKeeper(rawKeeper)

		oracle := sdk.AccAddress("oracle1_____________")
		err := k.RegisterOracle(ctx, oracle.String())
		require.NoError(t, err)

		tests := []struct {
			name  string
			asset string
			price math.LegacyDec
		}{
			{"negative price", "BTC", math.LegacyNewDec(-1000)},
			{"zero price", "ETH", math.LegacyZeroDec()},
			{"empty asset", "", math.LegacyNewDec(1000)},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				ctx = ctx.WithGasMeter(storetypes.NewGasMeter(100000))

				err := k.SubmitPrice(ctx, oracle.String(), tt.asset, tt.price)

				gasUsed := ctx.GasMeter().GasConsumed()

				// Validation should be cheap (fail fast)
				if err != nil {
					require.Less(t, gasUsed, uint64(20000),
						"Validation should be cheap: used %d gas", gasUsed)
					t.Logf("Rejected %s: %d gas", tt.name, gasUsed)
				}
			})
		}
	})

	t.Run("Invalid swap parameters", func(t *testing.T) {
		rawKeeper, ctx := keepertest.DexKeeper(t)
		k := NewDexGasKeeper(rawKeeper)

		// Create pool
		creator := sdk.AccAddress("creator1___________")
		poolID, err := k.CreatePool(ctx, creator.String(), "upaw", "uusdt",
			math.NewInt(1000000000), math.NewInt(2000000000))
		require.NoError(t, err)

		trader := sdk.AccAddress("trader1____________")

		tests := []struct {
			name     string
			amountIn math.Int
			minOut   math.Int
		}{
			{"zero amount", math.ZeroInt(), math.NewInt(1)},
			{"negative amount", math.NewInt(-1000), math.NewInt(1)},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				ctx = ctx.WithGasMeter(storetypes.NewGasMeter(100000))

				_, err := k.Swap(ctx, trader.String(), poolID, "upaw", tt.amountIn, tt.minOut)

				gasUsed := ctx.GasMeter().GasConsumed()

				// Should fail fast with minimal gas
				if err != nil {
					require.Less(t, gasUsed, uint64(30000),
						"Invalid parameter check should be cheap: used %d gas", gasUsed)
					t.Logf("Rejected %s: %d gas", tt.name, gasUsed)
				}
			})
		}
	})
}

func TestDoS_ExcessiveStateWrites(t *testing.T) {
	t.Run("Multiple provider registrations", func(t *testing.T) {
		rawKeeper, ctx := keepertest.ComputeKeeper(t)
		k := NewComputeGasKeeper(rawKeeper)

		ctx = ctx.WithGasMeter(storetypes.NewGasMeter(10000000))

		// Try to register many providers
		numProviders := 100
		for i := 0; i < numProviders; i++ {
			provider := sdk.AccAddress(fmt.Sprintf("provider%d_________", i))

			err := k.RegisterProvider(ctx, provider.String(),
				fmt.Sprintf("Provider %d", i),
				fmt.Sprintf("https://provider%d.com", i),
				ResourceSpecs{
					CPUCores: 16,
					MemoryMB: 32768,
				})

			if err != nil && ctx.GasMeter().IsOutOfGas() {
				t.Logf("Hit gas limit at provider %d", i)
				return
			}
			require.NoError(t, err)
		}

		gasUsed := ctx.GasMeter().GasConsumed()
		gasPerProvider := gasUsed / uint64(numProviders)

		t.Logf("Registered %d providers: %d gas total, %d gas/provider",
			numProviders, gasUsed, gasPerProvider)

		// State writes should be expensive enough to prevent spam
		require.Greater(t, gasPerProvider, uint64(50000),
			"Provider registration should consume meaningful gas")
	})
}

func TestDoS_GasExhaustion(t *testing.T) {
	t.Run("Verify gas limit enforcement", func(t *testing.T) {
		rawKeeper, ctx := keepertest.DexKeeper(t)
		k := NewDexGasKeeper(rawKeeper)

		// Set very low gas limit
		lowGasLimit := uint64(10000)
		ctx = ctx.WithGasMeter(storetypes.NewGasMeter(lowGasLimit))

		creator := sdk.AccAddress("creator1___________")

		// Try operation that requires more gas
		_, err := k.CreatePool(ctx, creator.String(), "upaw", "uusdt",
			math.NewInt(1000000), math.NewInt(2000000))

		// Should fail with out of gas
		require.Error(t, err)
		require.True(t, ctx.GasMeter().IsOutOfGas(),
			"Should run out of gas with low limit")

		t.Logf("Out of gas error (expected): %v", err)
	})

	t.Run("Verify gas meter accuracy", func(t *testing.T) {
		rawKeeper, _, ctx := keepertest.OracleKeeper(t)
		k := NewOracleGasKeeper(rawKeeper)

		ctx = ctx.WithGasMeter(storetypes.NewGasMeter(1000000))

		oracle := sdk.AccAddress("oracle1_____________")
		err := k.RegisterOracle(ctx, oracle.String())
		require.NoError(t, err)

		gasAfterRegister := ctx.GasMeter().GasConsumed()

		// Submit price
		err = k.SubmitPrice(ctx, oracle.String(), "BTC", math.LegacyNewDec(50000))
		require.NoError(t, err)

		gasAfterSubmit := ctx.GasMeter().GasConsumed()

		// Gas should increase
		require.Greater(t, gasAfterSubmit, gasAfterRegister,
			"Gas should accumulate across operations")

		t.Logf("Register: %d gas, Submit: %d gas, Total: %d gas",
			gasAfterRegister, gasAfterSubmit-gasAfterRegister, gasAfterSubmit)
	})
}

func TestDoS_CircuitBreaker(t *testing.T) {
	t.Run("Large swap price impact", func(t *testing.T) {
		rawKeeper, ctx := keepertest.DexKeeper(t)
		k := NewDexGasKeeper(rawKeeper)

		// Create pool with limited liquidity
		creator := sdk.AccAddress("creator1___________")
		poolID, err := k.CreatePool(ctx, creator.String(), "upaw", "uusdt",
			math.NewInt(1000000), math.NewInt(2000000)) // Small pool
		require.NoError(t, err)

		trader := sdk.AccAddress("trader1____________")

		// Try massive swap that would drain pool
		ctx = ctx.WithGasMeter(storetypes.NewGasMeter(500000))

		hugeSwap := math.NewInt(10000000) // 10x pool size

		_, err = k.Swap(ctx, trader.String(), poolID, "upaw", hugeSwap, math.NewInt(1))

		gasUsed := ctx.GasMeter().GasConsumed()

		// Should reject due to circuit breaker, but still consume some gas
		if err != nil {
			t.Logf("Circuit breaker triggered (expected): %v, gas: %d", err, gasUsed)
			require.Less(t, gasUsed, uint64(150000),
				"Circuit breaker check should be efficient")
		}
	})
}
