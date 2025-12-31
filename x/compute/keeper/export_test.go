package keeper

import (
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/compute/types"
)

// This file contains test helpers that are only accessible within the keeper package
// (for keeper_test package tests). Functions that need to be accessible from external
// test packages (like compute_test) should be in exports.go instead.

// SetSlashRecordForTest exposes slash record setter for tests.
// Only accessible within keeper package tests.
func (k Keeper) SetSlashRecordForTest(ctx sdk.Context, record types.SlashRecord) error {
	return k.setSlashRecord(ctx, record)
}

// SetAppealForTest exposes appeal setter for tests.
// Only accessible within keeper package tests.
func (k Keeper) SetAppealForTest(ctx sdk.Context, appeal types.Appeal) error {
	return k.setAppeal(ctx, appeal)
}

// VerifyIBCZKProofForTest exposes the internal verifier for testing.
// Only accessible within keeper package tests.
func (k Keeper) VerifyIBCZKProofForTest(ctx sdk.Context, proof []byte, publicInputs []byte) error {
	return k.verifyIBCZKProof(ctx, proof, publicInputs)
}

// VerifyAttestationsForTest exposes verification helper for regression tests.
// Only accessible within keeper package tests.
func (k Keeper) VerifyAttestationsForTest(ctx sdk.Context, attestations [][]byte, publicKeys [][]byte, message []byte) error {
	return k.verifyAttestations(ctx, attestations, publicKeys, message)
}

// GetValidatorPublicKeysForTest exposes validator key retrieval for tests.
// Only accessible within keeper package tests.
func (k Keeper) GetValidatorPublicKeysForTest(ctx sdk.Context, chainID string) ([][]byte, error) {
	return k.getValidatorPublicKeys(ctx, chainID)
}

// GetStoreKeyForTesting exports the store key for testing.
// Only accessible within keeper package tests.
func GetStoreKeyForTesting(k *Keeper) storetypes.StoreKey {
	return k.storeKey
}

// SafeAddUint64ForTest exposes the safeAddUint64 function for SEC-1.3 testing.
// Only accessible within keeper package tests.
func SafeAddUint64ForTest(a, b uint64) (uint64, error) {
	return safeAddUint64(a, b)
}
