package keeper

import (
	"testing"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark/constraint"
	"github.com/stretchr/testify/require"
)

// Covers GetCircuitInfo/GetAllCircuitInfo/ExportVerifyingKeys/GetCircuitStats
func TestCircuitManagerInfoAndKeys(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	cm := k.GetCircuitManager()

	info, err := cm.GetCircuitInfo(CircuitTypeCompute)
	require.NoError(t, err)
	require.Equal(t, string(CircuitTypeCompute), info.CircuitType)
	require.True(t, info.ConstraintCount > 0)
	require.True(t, info.PublicInputCount > 0)
	require.False(t, info.Initialized)

	all := cm.GetAllCircuitInfo()
	require.Len(t, all, 3)

	// Stub groth16 setup to avoid heavy key generation
	original := groth16Setup
	groth16Setup = func(ccs constraint.ConstraintSystem) (groth16.ProvingKey, groth16.VerifyingKey, error) {
		return groth16.NewProvingKey(ecc.BN254), groth16.NewVerifyingKey(ecc.BN254), nil
	}
	t.Cleanup(func() { groth16Setup = original })

	require.NoError(t, k.InitializeCircuits(ctx))

	keys, err := cm.ExportVerifyingKeys()
	require.NoError(t, err)
	require.Len(t, keys, 3)
	for _, bz := range keys {
		require.NotEmpty(t, bz)
	}

	stats, err := cm.GetCircuitStats(ctx)
	require.NoError(t, err)
	require.NotNil(t, stats)
}
