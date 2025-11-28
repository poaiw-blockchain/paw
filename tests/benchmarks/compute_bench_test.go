package benchmarks

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/compute/types"
)

// BenchmarkJobSubmission benchmarks submitting compute jobs
func BenchmarkJobSubmission(b *testing.B) {
	k, ctx := keepertest.ComputeKeeper(b)

	// Register a provider
	provider := sdk.AccAddress("provider_____________")
	err := k.RegisterProvider(
		ctx,
		provider,
		"test-provider",
		"http://provider:8080",
		types.ComputeSpec{
			CpuCores:       4,
			MemoryMb:       8192,
			StorageGb:      100,
			GpuCount:       0,
			GpuType:        "",
			TimeoutSeconds: 3600,
		},
		types.Pricing{
			CpuPricePerMcoreHour:  math.LegacyNewDec(100),
			MemoryPricePerMbHour:  math.LegacyNewDec(10),
			GpuPricePerHour:       math.LegacyZeroDec(),
			StoragePricePerGbHour: math.LegacyNewDec(5),
		},
		math.NewInt(10000000),
	)
	if err != nil {
		b.Fatal(err)
	}

	requester := sdk.AccAddress("requester___________")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := k.SubmitRequest(
			ctx,
			requester,
			types.ComputeSpec{
				CpuCores:       2,
				MemoryMb:       4096,
				StorageGb:      10,
				GpuCount:       0,
				GpuType:        "",
				TimeoutSeconds: 1800,
			},
			"ubuntu:22.04",
			[]string{"python", "script.py"},
			map[string]string{"ENV": "test"},
			math.NewInt(100000),
			"",
		)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkResultVerification benchmarks verifying compute results
func BenchmarkResultVerification(b *testing.B) {
	k, ctx := keepertest.ComputeKeeper(b)

	// Setup: Register provider and submit request
	provider := sdk.AccAddress("provider_____________")
	err := k.RegisterProvider(
		ctx,
		provider,
		"test-provider",
		"http://provider:8080",
		types.ComputeSpec{
			CpuCores:       4,
			MemoryMb:       8192,
			StorageGb:      100,
			GpuCount:       0,
			GpuType:        "",
			TimeoutSeconds: 3600,
		},
		types.Pricing{
			CpuPricePerMcoreHour:  math.LegacyNewDec(100),
			MemoryPricePerMbHour:  math.LegacyNewDec(10),
			GpuPricePerHour:       math.LegacyZeroDec(),
			StoragePricePerGbHour: math.LegacyNewDec(5),
		},
		math.NewInt(10000000),
	)
	if err != nil {
		b.Fatal(err)
	}

	requester := sdk.AccAddress("requester___________")

	// Create multiple requests to verify
	requestIDs := make([]uint64, b.N)
	for i := 0; i < b.N; i++ {
		requestID, err := k.SubmitRequest(
			ctx,
			requester,
			types.ComputeSpec{
				CpuCores:       2,
				MemoryMb:       4096,
				StorageGb:      10,
				GpuCount:       0,
				GpuType:        "",
				TimeoutSeconds: 1800,
			},
			"ubuntu:22.04",
			[]string{"python", "script.py"},
			map[string]string{"ENV": "test"},
			math.NewInt(100000),
			"",
		)
		if err != nil {
			b.Fatal(err)
		}
		requestIDs[i] = requestID
	}

	// Create a valid verification proof (simplified for benchmarking)
	verificationProof := make([]byte, 200)
	for i := 0; i < 200; i++ {
		verificationProof[i] = byte(i % 256)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := k.SubmitResult(
			ctx,
			provider,
			requestIDs[i],
			"0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			"http://results.github.com/output.tar.gz",
			0,
			"http://results.github.com/logs.txt",
			verificationProof,
		)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkProviderSelection benchmarks finding suitable providers
func BenchmarkProviderSelection(b *testing.B) {
	k, ctx := keepertest.ComputeKeeper(b)

	// Register multiple providers with different specs
	for i := 0; i < 10; i++ {
		provider := sdk.AccAddress([]byte{byte(i), 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})
		err := k.RegisterProvider(
			ctx,
			provider,
			"provider-"+string(rune('0'+i)),
			"http://provider:8080",
			types.ComputeSpec{
				CpuCores:       uint32(4 + i),
				MemoryMb:       uint32(8192 + i*1024),
				StorageGb:      uint32(100 + i*10),
				GpuCount:       0,
				GpuType:        "",
				TimeoutSeconds: 3600,
			},
			types.Pricing{
				CpuPricePerMcoreHour:  math.LegacyNewDec(int64(100 + i*10)),
				MemoryPricePerMbHour:  math.LegacyNewDec(int64(10 + i)),
				GpuPricePerHour:       math.LegacyZeroDec(),
				StoragePricePerGbHour: math.LegacyNewDec(int64(5 + i)),
			},
			math.NewInt(10000000),
		)
		if err != nil {
			b.Fatal(err)
		}
	}

	specs := types.ComputeSpec{
		CpuCores:       2,
		MemoryMb:       4096,
		StorageGb:      10,
		GpuCount:       0,
		GpuType:        "",
		TimeoutSeconds: 1800,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := k.FindSuitableProvider(ctx, specs, "")
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkEscrowOperations benchmarks escrow locking and releasing
func BenchmarkEscrowOperations(b *testing.B) {
	k, ctx := keepertest.ComputeKeeper(b)

	requester := sdk.AccAddress("requester___________")
	provider := sdk.AccAddress("provider_____________")

	// Register provider first
	err := k.RegisterProvider(
		ctx,
		provider,
		"test-provider",
		"http://provider:8080",
		types.ComputeSpec{
			CpuCores:       4,
			MemoryMb:       8192,
			StorageGb:      100,
			GpuCount:       0,
			GpuType:        "",
			TimeoutSeconds: 3600,
		},
		types.Pricing{
			CpuPricePerMcoreHour:  math.LegacyNewDec(100),
			MemoryPricePerMbHour:  math.LegacyNewDec(10),
			GpuPricePerHour:       math.LegacyZeroDec(),
			StoragePricePerGbHour: math.LegacyNewDec(5),
		},
		math.NewInt(10000000),
	)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		requestID := uint64(i + 1)
		b.StartTimer()

		// Lock escrow
		err := k.LockEscrow(ctx, requester, provider, math.NewInt(100000), requestID, 3600)
		if err != nil {
			b.Fatal(err)
		}

		// Release escrow immediately (governance override)
		err = k.ReleaseEscrow(ctx, requestID, true)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkProviderRegistration benchmarks registering compute providers
func BenchmarkProviderRegistration(b *testing.B) {
	k, ctx := keepertest.ComputeKeeper(b)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		provider := sdk.AccAddress([]byte{byte(i), byte(i >> 8), byte(i >> 16), byte(i >> 24), 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})
		err := k.RegisterProvider(
			ctx,
			provider,
			"provider-"+string(rune('0'+i%10)),
			"http://provider:8080",
			types.ComputeSpec{
				CpuCores:       4,
				MemoryMb:       8192,
				StorageGb:      100,
				GpuCount:       0,
				GpuType:        "",
				TimeoutSeconds: 3600,
			},
			types.Pricing{
				CpuPricePerMcoreHour:  math.LegacyNewDec(100),
				MemoryPricePerMbHour:  math.LegacyNewDec(10),
				GpuPricePerHour:       math.LegacyZeroDec(),
				StoragePricePerGbHour: math.LegacyNewDec(5),
			},
			math.NewInt(10000000),
		)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkProviderLookup benchmarks retrieving provider information
func BenchmarkProviderLookup(b *testing.B) {
	k, ctx := keepertest.ComputeKeeper(b)

	provider := sdk.AccAddress("provider_____________")
	err := k.RegisterProvider(
		ctx,
		provider,
		"test-provider",
		"http://provider:8080",
		types.ComputeSpec{
			CpuCores:       4,
			MemoryMb:       8192,
			StorageGb:      100,
			GpuCount:       0,
			GpuType:        "",
			TimeoutSeconds: 3600,
		},
		types.Pricing{
			CpuPricePerMcoreHour:  math.LegacyNewDec(100),
			MemoryPricePerMbHour:  math.LegacyNewDec(10),
			GpuPricePerHour:       math.LegacyZeroDec(),
			StoragePricePerGbHour: math.LegacyNewDec(5),
		},
		math.NewInt(10000000),
	)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := k.GetProvider(ctx, provider)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkCostEstimation benchmarks estimating compute costs
func BenchmarkCostEstimation(b *testing.B) {
	k, ctx := keepertest.ComputeKeeper(b)

	provider := sdk.AccAddress("provider_____________")
	err := k.RegisterProvider(
		ctx,
		provider,
		"test-provider",
		"http://provider:8080",
		types.ComputeSpec{
			CpuCores:       4,
			MemoryMb:       8192,
			StorageGb:      100,
			GpuCount:       0,
			GpuType:        "",
			TimeoutSeconds: 3600,
		},
		types.Pricing{
			CpuPricePerMcoreHour:  math.LegacyNewDec(100),
			MemoryPricePerMbHour:  math.LegacyNewDec(10),
			GpuPricePerHour:       math.LegacyZeroDec(),
			StoragePricePerGbHour: math.LegacyNewDec(5),
		},
		math.NewInt(10000000),
	)
	if err != nil {
		b.Fatal(err)
	}

	specs := types.ComputeSpec{
		CpuCores:       2,
		MemoryMb:       4096,
		StorageGb:      10,
		GpuCount:       0,
		GpuType:        "",
		TimeoutSeconds: 1800,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, err := k.EstimateCost(ctx, provider, specs)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkProviderReputationUpdate benchmarks updating provider reputation
func BenchmarkProviderReputationUpdate(b *testing.B) {
	k, ctx := keepertest.ComputeKeeper(b)

	provider := sdk.AccAddress("provider_____________")
	err := k.RegisterProvider(
		ctx,
		provider,
		"test-provider",
		"http://provider:8080",
		types.ComputeSpec{
			CpuCores:       4,
			MemoryMb:       8192,
			StorageGb:      100,
			GpuCount:       0,
			GpuType:        "",
			TimeoutSeconds: 3600,
		},
		types.Pricing{
			CpuPricePerMcoreHour:  math.LegacyNewDec(100),
			MemoryPricePerMbHour:  math.LegacyNewDec(10),
			GpuPricePerHour:       math.LegacyZeroDec(),
			StoragePricePerGbHour: math.LegacyNewDec(5),
		},
		math.NewInt(10000000),
	)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		success := i%2 == 0 // Alternate between success and failure
		err := k.UpdateProviderReputation(ctx, provider, success)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkRequestCancellation benchmarks cancelling compute requests
func BenchmarkRequestCancellation(b *testing.B) {
	k, ctx := keepertest.ComputeKeeper(b)

	// Setup: Register provider
	provider := sdk.AccAddress("provider_____________")
	err := k.RegisterProvider(
		ctx,
		provider,
		"test-provider",
		"http://provider:8080",
		types.ComputeSpec{
			CpuCores:       4,
			MemoryMb:       8192,
			StorageGb:      100,
			GpuCount:       0,
			GpuType:        "",
			TimeoutSeconds: 3600,
		},
		types.Pricing{
			CpuPricePerMcoreHour:  math.LegacyNewDec(100),
			MemoryPricePerMbHour:  math.LegacyNewDec(10),
			GpuPricePerHour:       math.LegacyZeroDec(),
			StoragePricePerGbHour: math.LegacyNewDec(5),
		},
		math.NewInt(10000000),
	)
	if err != nil {
		b.Fatal(err)
	}

	requester := sdk.AccAddress("requester___________")

	// Create requests to cancel
	requestIDs := make([]uint64, b.N)
	for i := 0; i < b.N; i++ {
		requestID, err := k.SubmitRequest(
			ctx,
			requester,
			types.ComputeSpec{
				CpuCores:       2,
				MemoryMb:       4096,
				StorageGb:      10,
				GpuCount:       0,
				GpuType:        "",
				TimeoutSeconds: 1800,
			},
			"ubuntu:22.04",
			[]string{"python", "script.py"},
			map[string]string{"ENV": "test"},
			math.NewInt(100000),
			"",
		)
		if err != nil {
			b.Fatal(err)
		}
		requestIDs[i] = requestID
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		err := k.CancelRequest(ctx, requester, requestIDs[i])
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkIterateActiveProviders benchmarks iterating over active providers
func BenchmarkIterateActiveProviders(b *testing.B) {
	k, ctx := keepertest.ComputeKeeper(b)

	// Register multiple providers
	for i := 0; i < 50; i++ {
		provider := sdk.AccAddress([]byte{byte(i), 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0})
		err := k.RegisterProvider(
			ctx,
			provider,
			"provider-"+string(rune('0'+i%10)),
			"http://provider:8080",
			types.ComputeSpec{
				CpuCores:       4,
				MemoryMb:       8192,
				StorageGb:      100,
				GpuCount:       0,
				GpuType:        "",
				TimeoutSeconds: 3600,
			},
			types.Pricing{
				CpuPricePerMcoreHour:  math.LegacyNewDec(100),
				MemoryPricePerMbHour:  math.LegacyNewDec(10),
				GpuPricePerHour:       math.LegacyZeroDec(),
				StoragePricePerGbHour: math.LegacyNewDec(5),
			},
			math.NewInt(10000000),
		)
		if err != nil {
			b.Fatal(err)
		}
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		count := 0
		err := k.IterateActiveProviders(ctx, func(provider types.Provider) (bool, error) {
			count++
			return false, nil
		})
		if err != nil {
			b.Fatal(err)
		}
	}
}
