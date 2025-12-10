package keeper

import (
	"crypto/sha256"
	"testing"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestUpdateVerificationMetricsPersistsAndAccumulates(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	verifier := NewZKVerifier(k)

	require.NoError(t, verifier.updateVerificationMetrics(sdkCtx, true, 5*time.Millisecond, 100))

	metrics, err := k.GetZKMetrics(sdkCtx)
	require.NoError(t, err)
	require.Equal(t, uint64(1), metrics.TotalProofsVerified)
	require.Zero(t, metrics.TotalProofsFailed)
	require.Equal(t, uint64(5), metrics.AverageVerificationTimeMs)
	require.Equal(t, uint64(100), metrics.TotalGasConsumed)

	require.NoError(t, verifier.updateVerificationMetrics(sdkCtx, false, 15*time.Millisecond, 200))

	metrics, err = k.GetZKMetrics(sdkCtx)
	require.NoError(t, err)
	require.Equal(t, uint64(1), metrics.TotalProofsFailed)
	require.Equal(t, uint64(300), metrics.TotalGasConsumed)
	require.Equal(t, uint64(6), metrics.AverageVerificationTimeMs)
}

func TestHashComputationResultDeterministicOrdering(t *testing.T) {
	computation := []byte("result-bytes")
	metaA := map[string]interface{}{
		"timestamp":    int64(12345),
		"exit_code":    int32(0),
		"cpu_cycles":   uint64(11),
		"memory_bytes": uint64(22),
	}
	metaB := map[string]interface{}{
		"memory_bytes": uint64(22),
		"cpu_cycles":   uint64(11),
		"exit_code":    int32(0),
		"timestamp":    int64(12345),
	}

	hashA := HashComputationResult(computation, metaA)
	hashB := HashComputationResult(computation, metaB)

	require.Equal(t, hashA, hashB)
	require.Len(t, hashA, sha256.Size)

	metaB["exit_code"] = int32(1)
	hashC := HashComputationResult(computation, metaB)

	require.NotEqual(t, hashA, hashC)
}
