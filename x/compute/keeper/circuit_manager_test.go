package keeper

import (
	"errors"
	"testing"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark/constraint"
	"github.com/stretchr/testify/require"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/compute/types"
)

type mockKVProvider struct {
	sdk.Context
}

func (sp mockKVProvider) KVStore(key storetypes.StoreKey) storetypes.KVStore {
	return sp.Context.KVStore(key)
}

func TestInitializeComputeCircuitRegeneratesOnCorruption(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	cm := NewCircuitManager(k)

	// Seed circuit params with invalid verifying key bytes to force regeneration
	params := &types.CircuitParams{
		CircuitId: "compute-verification-v2",
		Enabled:   true,
		VerifyingKey: types.VerifyingKey{
			VkData: []byte{0x00, 0x01},
		},
	}
	require.NoError(t, k.SetCircuitParams(ctx, *params))

	// Override setup to return deterministic keys
	originalSetup := groth16Setup
	groth16Setup = func(ccs constraint.ConstraintSystem) (groth16.ProvingKey, groth16.VerifyingKey, error) {
		return groth16.NewProvingKey(ecc.BN254), groth16.NewVerifyingKey(ecc.BN254), nil
	}
	t.Cleanup(func() {
		groth16Setup = originalSetup
		// Reset circuit state for other tests
		circuitMu.Lock()
		circuitsInitialized = false
		circuitState = make(map[string]*circuitKeys)
		circuitMu.Unlock()
	})

	err := cm.Initialize(ctx)
	require.NoError(t, err)

	// Verify circuit was initialized
	circuitMu.RLock()
	keys := circuitState[computeCircuitDef.id]
	circuitMu.RUnlock()

	require.NotNil(t, keys)
	require.NotNil(t, keys.pk)
	require.NotNil(t, keys.vk)
}

func TestInitializeComputeCircuitSetupFailure(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	cm := NewCircuitManager(k)

	originalSetup := groth16Setup
	groth16Setup = func(ccs constraint.ConstraintSystem) (groth16.ProvingKey, groth16.VerifyingKey, error) {
		return nil, nil, errors.New("setup failed")
	}
	t.Cleanup(func() {
		groth16Setup = originalSetup
		// Reset circuit state for other tests
		circuitMu.Lock()
		circuitsInitialized = false
		circuitState = make(map[string]*circuitKeys)
		circuitMu.Unlock()
	})

	err := cm.Initialize(ctx)
	require.Error(t, err)
	require.Contains(t, err.Error(), "setup failed")
}

func TestGetStoreWithProviderContext(t *testing.T) {
	k, ctx := setupKeeperForTest(t)

	providerStore := k.getStore(ctx)
	require.NotNil(t, providerStore)

	sp := mockKVProvider{sdk.UnwrapSDKContext(ctx)}
	s := k.getStore(sp)
	require.NotNil(t, s)
}
