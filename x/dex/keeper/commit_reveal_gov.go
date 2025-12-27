package keeper

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/dex/types"
)

// Storage key prefixes for governance-based commit-reveal
// Uses the SAME prefix as SwapCommitmentKeyPrefix in commit_reveal.go
// to ensure consistent storage for swap commits across both implementations.
// DEX module namespace prefix (0x02) is used for consistency.
var (
	SwapCommitKeyPrefix = []byte{0x02, 0x1D} // Unified prefix for swap commits
)

// SwapCommitKey returns the store key for a swap commit by hash
func SwapCommitKey(swapHash string) []byte {
	return append(SwapCommitKeyPrefix, []byte(swapHash)...)
}

// SetSwapCommit stores a swap commitment
func (k Keeper) SetSwapCommit(ctx context.Context, commit types.SwapCommit) error {
	store := k.getStore(ctx)
	bz, err := k.cdc.Marshal(&commit)
	if err != nil {
		return err
	}
	store.Set(SwapCommitKey(commit.SwapHash), bz)
	return nil
}

// GetSwapCommitByHash retrieves a swap commit by its hash
func (k Keeper) GetSwapCommitByHash(ctx context.Context, swapHash string) (*types.SwapCommit, error) {
	store := k.getStore(ctx)
	bz := store.Get(SwapCommitKey(swapHash))
	if bz == nil {
		return nil, types.ErrCommitmentNotFound
	}

	var commit types.SwapCommit
	if err := k.cdc.Unmarshal(bz, &commit); err != nil {
		return nil, err
	}

	return &commit, nil
}

// DeleteSwapCommit removes a swap commit from storage
func (k Keeper) DeleteSwapCommit(ctx context.Context, swapHash string) error {
	store := k.getStore(ctx)
	store.Delete(SwapCommitKey(swapHash))
	return nil
}

// GetAllSwapCommits returns all active swap commits
func (k Keeper) GetAllSwapCommits(ctx context.Context) []types.SwapCommit {
	store := k.getStore(ctx)
	iterator := store.Iterator(SwapCommitKeyPrefix, storetypes.PrefixEndBytes(SwapCommitKeyPrefix))
	defer iterator.Close()

	var commits []types.SwapCommit
	for ; iterator.Valid(); iterator.Next() {
		var commit types.SwapCommit
		if err := k.cdc.Unmarshal(iterator.Value(), &commit); err != nil {
			continue
		}
		commits = append(commits, commit)
	}

	return commits
}

// ComputeRevealHash computes the hash for a reveal message to match against committed hash
// Hash = keccak256(trader, pool_id, token_in, token_out, amount_in, min_amount_out, deadline, nonce)
func (k Keeper) ComputeRevealHash(msg *types.MsgRevealSwap) string {
	h := sha256.New()

	// All parameters in canonical form
	h.Write([]byte(msg.Trader))
	h.Write([]byte(fmt.Sprintf("%d", msg.PoolId)))
	h.Write([]byte(msg.TokenIn))
	h.Write([]byte(msg.TokenOut))
	h.Write([]byte(msg.AmountIn.String()))
	h.Write([]byte(msg.MinAmountOut.String()))
	h.Write([]byte(fmt.Sprintf("%d", msg.Deadline)))
	h.Write([]byte(msg.Nonce))

	return hex.EncodeToString(h.Sum(nil))
}

// CleanupExpiredSwapCommits removes expired swap commits
// Should be called from EndBlocker
func (k Keeper) CleanupExpiredSwapCommits(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	currentHeight := sdkCtx.BlockHeight()

	commits := k.GetAllSwapCommits(ctx)

	for _, commit := range commits {
		if currentHeight >= commit.ExpiryHeight {
			// Delete expired commitment
			if err := k.DeleteSwapCommit(ctx, commit.SwapHash); err != nil {
				sdkCtx.Logger().Error("failed to delete expired commit",
					"trader", commit.Trader,
					"swap_hash", commit.SwapHash,
					"error", err,
				)
				continue
			}

			// Emit expiry event
			sdkCtx.EventManager().EmitEvent(
				sdk.NewEvent(
					"swap_commitment_expired",
					sdk.NewAttribute("trader", commit.Trader),
					sdk.NewAttribute("swap_hash", commit.SwapHash),
					sdk.NewAttribute("commit_height", fmt.Sprintf("%d", commit.CommitHeight)),
					sdk.NewAttribute("expiry_height", fmt.Sprintf("%d", commit.ExpiryHeight)),
					sdk.NewAttribute("current_height", fmt.Sprintf("%d", currentHeight)),
				),
			)
		}
	}

	return nil
}
