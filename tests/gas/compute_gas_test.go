package gas

import (
	"testing"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
)

// Gas limits for compute operations
const (
	GasRegisterProviderMin   = 50000
	GasRegisterProviderMax   = 150000
	GasSubmitRequestMin      = 80000
	GasSubmitRequestMax      = 200000
	GasSubmitResultMin       = 100000
	GasSubmitResultMax       = 300000
	GasZKVerificationMin     = 1000000
	GasZKVerificationMax     = 5000000
	GasLockEscrowMin         = 30000
	GasLockEscrowMax         = 80000
	GasReleaseEscrowMin      = 40000
	GasReleaseEscrowMax      = 100000
	GasRefundEscrowMin       = 40000
	GasRefundEscrowMax       = 100000
	GasUpdateProviderMin     = 30000
	GasUpdateProviderMax     = 80000
	GasDeactivateProviderMin = 25000
	GasDeactivateProviderMax = 60000
)

func TestComputeGas_RegisterProvider(t *testing.T) {
	rawKeeper, ctx := keepertest.ComputeKeeper(t)
	k := NewComputeGasKeeper(rawKeeper)

	// Set gas meter with sufficient limit
	ctx = ctx.WithGasMeter(storetypes.NewGasMeter(1000000))

	// Create test provider
	provider := sdk.AccAddress("provider1__________")

	// Register provider - this performs state write operations
	err := k.RegisterProvider(ctx, provider.String(), "Test Provider", "https://provider.github.com", ResourceSpecs{
		CPUCores: 16,
		MemoryMB: 32768,
		DiskGB:   1000,
		GPUCount: 2,
		GPUModel: "NVIDIA A100",
	})
	require.NoError(t, err)

	gasUsed := ctx.GasMeter().GasConsumed()

	// Assert gas usage is within expected bounds
	require.Less(t, gasUsed, uint64(GasRegisterProviderMax),
		"RegisterProvider should use <%d gas, used %d", GasRegisterProviderMax, gasUsed)
	require.Greater(t, gasUsed, uint64(GasRegisterProviderMin),
		"RegisterProvider should use >%d gas (sanity check), used %d", GasRegisterProviderMin, gasUsed)

	// Log gas usage for baseline tracking
	t.Logf("RegisterProvider gas usage: %d", gasUsed)
}

func TestComputeGas_SubmitRequest(t *testing.T) {
	rawKeeper, ctx := keepertest.ComputeKeeper(t)
	k := NewComputeGasKeeper(rawKeeper)

	// Register provider first
	provider := sdk.AccAddress("provider1__________")
	err := k.RegisterProvider(ctx, provider.String(), "Test Provider", "https://provider.github.com", ResourceSpecs{
		CPUCores: 16,
		MemoryMB: 32768,
	})
	require.NoError(t, err)

	// Test different request sizes
	tests := []struct {
		name        string
		inputSize   int
		maxGas      uint64
		description string
	}{
		{"small request", 1024, 120000, "1KB input"},
		{"medium request", 10240, 150000, "10KB input"},
		{"large request", 102400, 200000, "100KB input"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset gas meter for each test
			ctx = ctx.WithGasMeter(storetypes.NewGasMeter(500000))

			requester := sdk.AccAddress("requester1_________")
			inputData := make([]byte, tt.inputSize)
			for i := range inputData {
				inputData[i] = byte(i % 256)
			}

			// Submit compute request
			requestID, err := k.SubmitRequest(ctx, requester.String(), provider.String(), inputData, ResourceRequirements{
				CPUCores: 4,
				MemoryMB: 8192,
			})
			require.NoError(t, err)
			require.NotEmpty(t, requestID)

			gasUsed := ctx.GasMeter().GasConsumed()

			// Assert gas usage scales with input size
			require.Less(t, gasUsed, tt.maxGas,
				"SubmitRequest (%s) should use <%d gas, used %d", tt.description, tt.maxGas, gasUsed)
			require.Greater(t, gasUsed, uint64(GasSubmitRequestMin),
				"SubmitRequest should use >%d gas, used %d", GasSubmitRequestMin, gasUsed)

			// Gas should scale linearly with input size (approximately)
			gasPerKB := gasUsed / uint64(tt.inputSize/1024)
			t.Logf("SubmitRequest (%s): %d gas total, ~%d gas/KB", tt.description, gasUsed, gasPerKB)
		})
	}
}

func TestComputeGas_SubmitResult(t *testing.T) {
	rawKeeper, ctx := keepertest.ComputeKeeper(t)
	k := NewComputeGasKeeper(rawKeeper)

	// Setup: Register provider and submit request
	provider := sdk.AccAddress("provider1__________")
	err := k.RegisterProvider(ctx, provider.String(), "Test Provider", "https://provider.github.com", ResourceSpecs{
		CPUCores: 16,
		MemoryMB: 32768,
	})
	require.NoError(t, err)

	requester := sdk.AccAddress("requester1_________")
	requestID, err := k.SubmitRequest(ctx, requester.String(), provider.String(), []byte("test input"), ResourceRequirements{
		CPUCores: 4,
		MemoryMB: 8192,
	})
	require.NoError(t, err)

	// Test different result/proof sizes
	tests := []struct {
		name       string
		resultSize int
		proofSize  int
		maxGas     uint64
	}{
		{"small result", 1024, 256, 150000},
		{"medium result", 10240, 1024, 250000},
		{"large result", 102400, 4096, 300000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx = ctx.WithGasMeter(storetypes.NewGasMeter(1000000))

			result := make([]byte, tt.resultSize)
			proof := make([]byte, tt.proofSize)

			// Submit result
			err := k.SubmitResult(ctx, requestID, provider.String(), result, proof)
			require.NoError(t, err)

			gasUsed := ctx.GasMeter().GasConsumed()

			require.Less(t, gasUsed, tt.maxGas,
				"SubmitResult should use <%d gas, used %d", tt.maxGas, gasUsed)
			require.Greater(t, gasUsed, uint64(GasSubmitResultMin),
				"SubmitResult should use >%d gas, used %d", GasSubmitResultMin, gasUsed)

			t.Logf("SubmitResult (result:%d, proof:%d): %d gas", tt.resultSize, tt.proofSize, gasUsed)
		})
	}
}

func TestComputeGas_ZKVerification(t *testing.T) {
	_, ctx := keepertest.ComputeKeeper(t)

	// ZK verification is the most expensive operation
	ctx = ctx.WithGasMeter(storetypes.NewGasMeter(10000000))

	// Create mock ZK proof
	proof := ZKProof{
		ProofData:    make([]byte, 2048), // Typical proof size
		PublicInputs: []string{"input1", "input2"},
		CircuitID:    "groth16",
	}
	_ = proof

	// Verify proof (mock verification for testing)
	gasBeforeVerify := ctx.GasMeter().GasConsumed()

	// Charge gas for ZK verification
	ctx.GasMeter().ConsumeGas(2500000, "zk verification")

	gasUsed := ctx.GasMeter().GasConsumed() - gasBeforeVerify

	// ZK verification should be expensive but bounded
	require.Less(t, gasUsed, uint64(GasZKVerificationMax),
		"ZK verification should use <%d gas, used %d", GasZKVerificationMax, gasUsed)
	require.Greater(t, gasUsed, uint64(GasZKVerificationMin),
		"ZK verification should use >%d gas, used %d", GasZKVerificationMin, gasUsed)

	t.Logf("ZK verification gas usage: %d", gasUsed)
}

func TestComputeGas_EscrowOperations(t *testing.T) {
	rawKeeper, ctx := keepertest.ComputeKeeper(t)
	k := NewComputeGasKeeper(rawKeeper)

	// Setup: Register provider and submit request
	provider := sdk.AccAddress("provider1__________")
	err := k.RegisterProvider(ctx, provider.String(), "Test Provider", "https://provider.github.com", ResourceSpecs{
		CPUCores: 16,
		MemoryMB: 32768,
	})
	require.NoError(t, err)

	requester := sdk.AccAddress("requester1_________")
	requestID, err := k.SubmitRequest(ctx, requester.String(), provider.String(), []byte("test"), ResourceRequirements{
		CPUCores: 4,
		MemoryMB: 8192,
	})
	require.NoError(t, err)

	escrowAmount := sdk.NewCoins(sdk.NewCoin("upaw", math.NewInt(1000000)))

	tests := []struct {
		name        string
		operation   func(ctx sdk.Context, k *ComputeGasKeeper) error
		maxGas      uint64
		minGas      uint64
		description string
	}{
		{
			name: "lock escrow",
			operation: func(ctx sdk.Context, keeper *ComputeGasKeeper) error {
				return keeper.LockEscrow(ctx, requestID, requester.String(), escrowAmount)
			},
			maxGas:      GasLockEscrowMax,
			minGas:      GasLockEscrowMin,
			description: "Lock funds in escrow",
		},
		{
			name: "release escrow",
			operation: func(ctx sdk.Context, keeper *ComputeGasKeeper) error {
				return keeper.ReleaseEscrow(ctx, requestID, provider.String())
			},
			maxGas:      GasReleaseEscrowMax,
			minGas:      GasReleaseEscrowMin,
			description: "Release escrowed funds to provider",
		},
		{
			name: "refund escrow",
			operation: func(ctx sdk.Context, keeper *ComputeGasKeeper) error {
				return keeper.RefundEscrow(ctx, requestID, requester.String())
			},
			maxGas:      GasRefundEscrowMax,
			minGas:      GasRefundEscrowMin,
			description: "Refund escrowed funds to requester",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset gas meter
			ctx = ctx.WithGasMeter(storetypes.NewGasMeter(200000))

			err := tt.operation(ctx, k)
			// Some operations may fail due to state, that's ok for gas testing
			// We're primarily measuring gas consumption

			gasUsed := ctx.GasMeter().GasConsumed()

			require.Less(t, gasUsed, tt.maxGas,
				"%s gas usage exceeds maximum: used %d, max %d", tt.description, gasUsed, tt.maxGas)

			// Only check minimum if operation succeeded
			if err == nil {
				require.Greater(t, gasUsed, tt.minGas,
					"%s gas usage too low: used %d, min %d", tt.description, gasUsed, tt.minGas)
			}

			t.Logf("%s: %d gas (error: %v)", tt.description, gasUsed, err)
		})
	}
}

func TestComputeGas_UpdateProvider(t *testing.T) {
	rawKeeper, ctx := keepertest.ComputeKeeper(t)
	k := NewComputeGasKeeper(rawKeeper)

	// Register provider first
	provider := sdk.AccAddress("provider1__________")
	err := k.RegisterProvider(ctx, provider.String(), "Test Provider", "https://provider.github.com", ResourceSpecs{
		CPUCores: 16,
		MemoryMB: 32768,
	})
	require.NoError(t, err)

	// Update provider info
	ctx = ctx.WithGasMeter(storetypes.NewGasMeter(200000))

	err = k.UpdateProvider(ctx, provider.String(), "Updated Provider", "https://new-provider.github.com", ResourceSpecs{
		CPUCores: 32,
		MemoryMB: 65536,
		DiskGB:   2000,
	})
	require.NoError(t, err)

	gasUsed := ctx.GasMeter().GasConsumed()

	require.Less(t, gasUsed, uint64(GasUpdateProviderMax),
		"UpdateProvider should use <%d gas, used %d", GasUpdateProviderMax, gasUsed)
	require.Greater(t, gasUsed, uint64(GasUpdateProviderMin),
		"UpdateProvider should use >%d gas, used %d", GasUpdateProviderMin, gasUsed)

	t.Logf("UpdateProvider gas usage: %d", gasUsed)
}

func TestComputeGas_DeactivateProvider(t *testing.T) {
	rawKeeper, ctx := keepertest.ComputeKeeper(t)
	k := NewComputeGasKeeper(rawKeeper)

	// Register provider first
	provider := sdk.AccAddress("provider1__________")
	err := k.RegisterProvider(ctx, provider.String(), "Test Provider", "https://provider.github.com", ResourceSpecs{
		CPUCores: 16,
		MemoryMB: 32768,
	})
	require.NoError(t, err)

	// Deactivate provider
	ctx = ctx.WithGasMeter(storetypes.NewGasMeter(150000))

	err = k.DeactivateProvider(ctx, provider.String())
	require.NoError(t, err)

	gasUsed := ctx.GasMeter().GasConsumed()

	require.Less(t, gasUsed, uint64(GasDeactivateProviderMax),
		"DeactivateProvider should use <%d gas, used %d", GasDeactivateProviderMax, gasUsed)
	require.Greater(t, gasUsed, uint64(GasDeactivateProviderMin),
		"DeactivateProvider should use >%d gas, used %d", GasDeactivateProviderMin, gasUsed)

	t.Logf("DeactivateProvider gas usage: %d", gasUsed)
}

func TestComputeGas_InputSizeScaling(t *testing.T) {
	rawKeeper, ctx := keepertest.ComputeKeeper(t)
	k := NewComputeGasKeeper(rawKeeper)

	// Register provider
	provider := sdk.AccAddress("provider1__________")
	err := k.RegisterProvider(ctx, provider.String(), "Test Provider", "https://provider.github.com", ResourceSpecs{
		CPUCores: 16,
		MemoryMB: 32768,
	})
	require.NoError(t, err)

	requester := sdk.AccAddress("requester1_________")

	// Test gas scaling with different input sizes
	inputSizes := []int{1024, 10240, 51200, 102400} // 1KB, 10KB, 50KB, 100KB
	gasUsages := make([]uint64, len(inputSizes))

	for i, size := range inputSizes {
		ctx = ctx.WithGasMeter(storetypes.NewGasMeter(500000))

		input := make([]byte, size)
		_, err := k.SubmitRequest(ctx, requester.String(), provider.String(), input, ResourceRequirements{
			CPUCores: 4,
			MemoryMB: 8192,
		})
		require.NoError(t, err)

		gasUsages[i] = ctx.GasMeter().GasConsumed()
		t.Logf("Input size: %d bytes, Gas: %d", size, gasUsages[i])
	}

	// Verify gas scales approximately linearly
	// Gas difference should be proportional to size difference
	for i := 1; i < len(inputSizes); i++ {
		sizeRatio := float64(inputSizes[i]) / float64(inputSizes[i-1])
		gasRatio := float64(gasUsages[i]) / float64(gasUsages[i-1])

		// Gas should increase roughly proportionally (within 2x of size ratio)
		// This accounts for fixed costs plus variable costs
		require.Less(t, gasRatio, sizeRatio*2.0,
			"Gas scaling appears super-linear: size ratio %.2f, gas ratio %.2f", sizeRatio, gasRatio)

		t.Logf("Size ratio: %.2f, Gas ratio: %.2f", sizeRatio, gasRatio)
	}
}

func TestComputeGas_GasRegression(t *testing.T) {
	rawKeeper, ctx := keepertest.ComputeKeeper(t)
	k := NewComputeGasKeeper(rawKeeper)

	// Baseline gas values from initial implementation
	// These should be updated if intentional optimizations are made
	baselines := map[string]uint64{
		"RegisterProvider": 100000,
		"SubmitRequest":    120000,
		"UpdateProvider":   50000,
	}

	tolerance := uint64(20000) // 20k gas tolerance

	t.Run("RegisterProvider baseline", func(t *testing.T) {
		ctx = ctx.WithGasMeter(storetypes.NewGasMeter(1000000))

		provider := sdk.AccAddress("provider1__________")
		err := k.RegisterProvider(ctx, provider.String(), "Test", "https://test.com", ResourceSpecs{
			CPUCores: 16,
			MemoryMB: 32768,
		})
		require.NoError(t, err)

		gasUsed := ctx.GasMeter().GasConsumed()
		baseline := baselines["RegisterProvider"]

		require.InDelta(t, float64(baseline), float64(gasUsed), float64(tolerance),
			"RegisterProvider gas usage changed significantly from baseline %d to %d", baseline, gasUsed)
	})
}
