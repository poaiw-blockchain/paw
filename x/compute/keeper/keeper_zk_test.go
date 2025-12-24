package keeper

import (
	"bytes"
	"testing"

	"github.com/consensys/gnark-crypto/ecc"
	"github.com/consensys/gnark/backend"
	"github.com/consensys/gnark/backend/groth16"
	"github.com/consensys/gnark/backend/witness"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"
)

func TestVerifyComputeProofWithCircuitManager_FailsWithoutCircuit(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Setup circuits initialized but without compute circuit keys
	circuitMu.Lock()
	circuitsInitialized = true
	circuitState = make(map[string]*circuitKeys) // No compute circuit
	circuitMu.Unlock()

	defer func() {
		circuitMu.Lock()
		circuitsInitialized = false
		circuitState = make(map[string]*circuitKeys)
		circuitMu.Unlock()
	}()

	ok, err := k.VerifyComputeProofWithCircuitManager(
		sdkCtx,
		[]byte{}, 1, "result", "provider", "resource",
	)
	require.False(t, ok)
	require.Error(t, err)
}

func TestVerifyEscrowProofWithCircuitManager_FailsWithoutCircuit(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Setup circuits initialized but without escrow circuit keys
	circuitMu.Lock()
	circuitsInitialized = true
	circuitState = make(map[string]*circuitKeys) // No escrow circuit
	circuitMu.Unlock()

	defer func() {
		circuitMu.Lock()
		circuitsInitialized = false
		circuitState = make(map[string]*circuitKeys)
		circuitMu.Unlock()
	}()

	ok, err := k.VerifyEscrowProofWithCircuitManager(
		sdkCtx,
		[]byte{}, 1, 10, "requester", "provider", "completion",
	)
	require.False(t, ok)
	require.Error(t, err)
}

func TestVerifyResultProofWithCircuitManager_FailsWithoutCircuit(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	// Setup circuits initialized but without result circuit keys
	circuitMu.Lock()
	circuitsInitialized = true
	circuitState = make(map[string]*circuitKeys) // No result circuit
	circuitMu.Unlock()

	defer func() {
		circuitMu.Lock()
		circuitsInitialized = false
		circuitState = make(map[string]*circuitKeys)
		circuitMu.Unlock()
	}()

	ok, err := k.VerifyResultProofWithCircuitManager(
		sdkCtx,
		[]byte{}, 1, "resultRoot", "inputRoot", "programHash",
	)
	require.False(t, ok)
	require.Error(t, err)
}

func TestVerifyComputeProofWithCircuitManager_SucceedsWithInjectedVerifier(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	proofBytes := newEmptyProofBytes(t)
	stubGroth16Verifier(t)

	// Setup circuit state with compute circuit keys
	circuitMu.Lock()
	circuitsInitialized = true
	circuitState = map[string]*circuitKeys{
		computeCircuitDef.id: {
			vk: groth16.NewVerifyingKey(ecc.BN254),
		},
	}
	circuitMu.Unlock()

	defer func() {
		circuitMu.Lock()
		circuitsInitialized = false
		circuitState = make(map[string]*circuitKeys)
		circuitMu.Unlock()
	}()

	ok, err := k.VerifyComputeProofWithCircuitManager(
		sdkCtx,
		proofBytes,
		42,
		uint64(101),
		uint64(202),
		uint64(303),
	)
	require.NoError(t, err)
	require.True(t, ok)
}

func TestVerifyEscrowProofWithCircuitManager_SucceedsWithInjectedVerifier(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	proofBytes := newEmptyProofBytes(t)
	stubGroth16Verifier(t)

	// Setup circuit state with escrow circuit keys
	circuitMu.Lock()
	circuitsInitialized = true
	circuitState = map[string]*circuitKeys{
		escrowCircuitDef.id: {
			vk: groth16.NewVerifyingKey(ecc.BN254),
		},
	}
	circuitMu.Unlock()

	defer func() {
		circuitMu.Lock()
		circuitsInitialized = false
		circuitState = make(map[string]*circuitKeys)
		circuitMu.Unlock()
	}()

	ok, err := k.VerifyEscrowProofWithCircuitManager(
		sdkCtx,
		proofBytes,
		7,
		500,
		uint64(11),
		uint64(22),
		uint64(33),
	)
	require.NoError(t, err)
	require.True(t, ok)
}

func TestVerifyResultProofWithCircuitManager_SucceedsWithInjectedVerifier(t *testing.T) {
	k, ctx := setupKeeperForTest(t)
	sdkCtx := sdk.UnwrapSDKContext(ctx)

	proofBytes := newEmptyProofBytes(t)
	stubGroth16Verifier(t)

	// Setup circuit state with result circuit keys
	circuitMu.Lock()
	circuitsInitialized = true
	circuitState = map[string]*circuitKeys{
		resultCircuitDef.id: {
			vk: groth16.NewVerifyingKey(ecc.BN254),
		},
	}
	circuitMu.Unlock()

	defer func() {
		circuitMu.Lock()
		circuitsInitialized = false
		circuitState = make(map[string]*circuitKeys)
		circuitMu.Unlock()
	}()

	ok, err := k.VerifyResultProofWithCircuitManager(
		sdkCtx,
		proofBytes,
		9,
		uint64(1),
		uint64(2),
		uint64(3),
	)
	require.NoError(t, err)
	require.True(t, ok)
}

func newEmptyProofBytes(t *testing.T) []byte {
	t.Helper()

	var buf bytes.Buffer
	proof := groth16.NewProof(ecc.BN254)
	_, err := proof.WriteTo(&buf)
	require.NoError(t, err)

	return buf.Bytes()
}

func stubGroth16Verifier(t *testing.T) {
	t.Helper()

	original := groth16Verify
	groth16Verify = func(_ groth16.Proof, _ groth16.VerifyingKey, _ witness.Witness, _ ...backend.VerifierOption) error {
		return nil
	}

	t.Cleanup(func() {
		groth16Verify = original
	})
}
