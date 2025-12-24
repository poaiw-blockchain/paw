package keeper

import (
	"context"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/compute/types"
)

// SetSlashRecordForTest exposes slash record setter for tests.
func (k Keeper) SetSlashRecordForTest(ctx sdk.Context, record types.SlashRecord) error {
	return k.setSlashRecord(ctx, record)
}

// SetAppealForTest exposes appeal setter for tests.
func (k Keeper) SetAppealForTest(ctx sdk.Context, appeal types.Appeal) error {
	return k.setAppeal(ctx, appeal)
}

// VerifyIBCZKProofForTest exposes the internal verifier for testing.
func (k Keeper) VerifyIBCZKProofForTest(ctx sdk.Context, proof []byte, publicInputs []byte) error {
	return k.verifyIBCZKProof(ctx, proof, publicInputs)
}

// VerifyAttestationsForTest exposes verification helper for regression tests.
func (k Keeper) VerifyAttestationsForTest(ctx sdk.Context, attestations [][]byte, publicKeys [][]byte, message []byte) error {
	return k.verifyAttestations(ctx, attestations, publicKeys, message)
}

// GetValidatorPublicKeysForTest exposes validator key retrieval for tests.
func (k Keeper) GetValidatorPublicKeysForTest(ctx sdk.Context, chainID string) ([][]byte, error) {
	return k.getValidatorPublicKeys(ctx, chainID)
}

// TrackPendingOperationForTest exposes trackPendingOperation for white-box tests.
func TrackPendingOperationForTest(k *Keeper, ctx sdk.Context, op ChannelOperation) {
	k.trackPendingOperation(ctx, op)
}

// RecordNonceUsageForTesting exports recordNonceUsage for testing (accessible from external test packages)
func (k Keeper) RecordNonceUsageForTesting(ctx context.Context, provider sdk.AccAddress, nonce uint64) {
	k.recordNonceUsage(ctx, provider, nonce)
}

// CheckReplayAttackForTesting exports checkReplayAttack for testing (accessible from external test packages)
func (k Keeper) CheckReplayAttackForTesting(ctx context.Context, provider sdk.AccAddress, nonce uint64) bool {
	return k.checkReplayAttack(ctx, provider, nonce)
}

// GetAuthority returns the authority address for testing
func (k Keeper) GetAuthority() string {
	return k.authority
}

// GetDisputeForTesting exports getDispute for testing
func (k Keeper) GetDisputeForTesting(ctx context.Context, id uint64) (*types.Dispute, error) {
	return k.getDispute(ctx, id)
}

// GetBankKeeper exports bankKeeper for testing
func (k Keeper) GetBankKeeper() types.BankKeeper {
	return k.bankKeeper
}

// GetStoreKeyForTesting exports the store key for testing
func GetStoreKeyForTesting(k *Keeper) storetypes.StoreKey {
	return k.storeKey
}
