package keeper

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/paw-chain/paw/x/compute/types"
)

// TestSEC1_2_RateLimitUnderflowProtection tests that rate limiting properly
// handles the case when tokens are depleted (SEC-1.2 fix)
func TestSEC1_2_RateLimitUnderflowProtection(t *testing.T) {
	k, ctx := setupKeeperForTest(t)

	testAddr := createTestAddress("requester1")

	// Set default params first
	params := types.DefaultParams()
	require.NoError(t, k.SetParams(ctx, params))

	// Exhaust all tokens by making many requests
	// The default burst allowance is 20, so we need to exhaust those
	var lastErr error
	for i := 0; i < 25; i++ {
		err := k.CheckRateLimit(ctx, testAddr)
		if err != nil {
			// Expected to fail once tokens are exhausted
			lastErr = err
			require.Contains(t, err.Error(), "burst capacity depleted")
			break
		}
	}

	// Should have hit the rate limit
	require.NotNil(t, lastErr, "should have hit rate limit")
}

// TestSEC1_3_SafeArithmeticQuotaCalculations tests that quota calculations
// properly detect and prevent integer overflow (SEC-1.3 fix)
func TestSEC1_3_SafeArithmeticQuotaCalculations(t *testing.T) {
	testCases := []struct {
		name        string
		a           uint64
		b           uint64
		shouldError bool
	}{
		{
			name:        "normal addition",
			a:           100,
			b:           200,
			shouldError: false,
		},
		{
			name:        "near max uint64",
			a:           ^uint64(0) - 1, // MaxUint64 - 1
			b:           1,
			shouldError: false,
		},
		{
			name:        "overflow case",
			a:           ^uint64(0), // MaxUint64
			b:           1,
			shouldError: true,
		},
		{
			name:        "large values overflow",
			a:           ^uint64(0) / 2,
			b:           ^uint64(0)/2 + 2,
			shouldError: true,
		},
		{
			name:        "zero addition",
			a:           0,
			b:           0,
			shouldError: false,
		},
		{
			name:        "max plus zero",
			a:           ^uint64(0),
			b:           0,
			shouldError: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := SafeAddUint64ForTest(tc.a, tc.b)
			if tc.shouldError {
				require.Error(t, err)
				require.Contains(t, err.Error(), "overflow")
			} else {
				require.NoError(t, err)
				require.Equal(t, tc.a+tc.b, result)
			}
		})
	}
}

// TestSEC1_3_QuotaOverflowPrevention tests that CheckResourceQuota properly
// rejects requests that would cause overflow
func TestSEC1_3_QuotaOverflowPrevention(t *testing.T) {
	k, ctx := setupKeeperForTest(t)

	testAddr := createTestAddress("requester1")

	// Set default params first
	params := types.DefaultParams()
	require.NoError(t, k.SetParams(ctx, params))

	// Create a spec with near-max values that would overflow when added
	overflowSpec := types.ComputeSpec{
		CpuCores:       ^uint64(0),
		MemoryMb:       1024,
		StorageGb:      10,
		TimeoutSeconds: 3600,
	}

	// First, set up an existing quota with some usage
	quota := k.GetDefaultResourceQuota(testAddr.String())
	quota.CurrentCpu = 1 // Any value > 0 will cause overflow with MaxUint64
	err := k.SetResourceQuota(ctx, *quota)
	require.NoError(t, err)

	// Now check quota - should fail due to overflow
	err = k.CheckResourceQuota(ctx, testAddr, overflowSpec)
	require.Error(t, err)
	require.Contains(t, err.Error(), "overflow")
}

// TestSEC1_3_QuotaNormalOperations tests that normal quota operations
// still work correctly after the security fix
func TestSEC1_3_QuotaNormalOperations(t *testing.T) {
	k, ctx := setupKeeperForTest(t)

	testAddr := createTestAddress("requester1")

	// Set default params first
	params := types.DefaultParams()
	require.NoError(t, k.SetParams(ctx, params))

	// Normal spec that should pass
	normalSpec := types.ComputeSpec{
		CpuCores:       4,
		MemoryMb:       1024,
		StorageGb:      10,
		TimeoutSeconds: 3600,
	}

	// Should pass for first request
	err := k.CheckResourceQuota(ctx, testAddr, normalSpec)
	require.NoError(t, err)

	// Allocate and check again
	err = k.AllocateResources(ctx, testAddr, normalSpec)
	require.NoError(t, err)

	// Should still pass for second request
	err = k.CheckResourceQuota(ctx, testAddr, normalSpec)
	require.NoError(t, err)
}

// createTestAddress creates a test address from a string
func createTestAddress(name string) sdk.AccAddress {
	// Pad to consistent length for valid address
	padded := name
	for len(padded) < 20 {
		padded += "_"
	}
	return sdk.AccAddress([]byte(padded[:20]))
}
