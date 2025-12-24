package keeper

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"time"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/dex/types"
)

// Commit-Reveal Constants for Large Swap Protection
// These prevent sandwich attacks on large swaps by requiring a commit phase before reveal.
const (
	// LargeSwapThresholdPercent - swaps exceeding this % of pool reserves require commit-reveal
	LargeSwapThresholdPercent = "0.05" // 5% of pool reserves

	// RevealDelayBlocks - minimum blocks between commit and reveal
	RevealDelayBlocks = int64(2)

	// CommitExpiryBlocks - blocks after which uncommitted swaps expire
	CommitExpiryBlocks = int64(50)

	// CommitDepositAmount - deposit required for commit (returned on valid reveal)
	CommitDepositAmount = int64(1000000) // 1 UPAW (1M upaw)
)

// Storage key prefixes for commit-reveal
var (
	SwapCommitmentKeyPrefix       = []byte{0x16}
	SwapCommitmentByExpiryPrefix  = []byte{0x17}
	SwapCommitmentByTraderPrefix  = []byte{0x18}
)

// SwapCommitment stores a pending swap commitment
type SwapCommitment struct {
	CommitmentHash  []byte         `json:"commitment_hash"`
	Trader          string         `json:"trader"`
	PoolID          uint64         `json:"pool_id"`
	CommitBlock     int64          `json:"commit_block"`
	ExpiryBlock     int64          `json:"expiry_block"`
	DepositAmount   math.Int       `json:"deposit_amount"`
	DepositDenom    string         `json:"deposit_denom"`
}

// SwapCommitmentKey returns the store key for a swap commitment
func SwapCommitmentKey(commitmentHash []byte) []byte {
	return append(SwapCommitmentKeyPrefix, commitmentHash...)
}

// SwapCommitmentByExpiryKey returns an index key for expiry cleanup
func SwapCommitmentByExpiryKey(expiryBlock int64, commitmentHash []byte) []byte {
	blockBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(blockBytes, uint64(expiryBlock))
	key := append(SwapCommitmentByExpiryPrefix, blockBytes...)
	return append(key, commitmentHash...)
}

// SwapCommitmentByTraderKey returns an index key for trader's commitments
func SwapCommitmentByTraderKey(trader sdk.AccAddress, commitmentHash []byte) []byte {
	key := append(SwapCommitmentByTraderPrefix, trader.Bytes()...)
	return append(key, commitmentHash...)
}

// RequiresCommitReveal checks if a swap is large enough to require commit-reveal
func (k Keeper) RequiresCommitReveal(ctx context.Context, poolID uint64, amountIn math.Int) (bool, error) {
	pool, err := k.GetPool(ctx, poolID)
	if err != nil {
		return false, err
	}

	// Use the smaller reserve to calculate threshold
	smallerReserve := pool.ReserveA
	if pool.ReserveB.LT(pool.ReserveA) {
		smallerReserve = pool.ReserveB
	}

	threshold, err := math.LegacyNewDecFromStr(LargeSwapThresholdPercent)
	if err != nil {
		return false, err
	}

	thresholdAmount := math.LegacyNewDecFromInt(smallerReserve).Mul(threshold).TruncateInt()
	return amountIn.GT(thresholdAmount), nil
}

// ComputeSwapCommitmentHash computes the commitment hash for a swap
// Hash = SHA256(poolID || tokenIn || tokenOut || amountIn || minAmountOut || salt || trader)
func ComputeSwapCommitmentHash(
	poolID uint64,
	tokenIn, tokenOut string,
	amountIn, minAmountOut math.Int,
	salt []byte,
	trader sdk.AccAddress,
) []byte {
	h := sha256.New()

	// Pool ID
	poolBytes := make([]byte, 8)
	binary.BigEndian.PutUint64(poolBytes, poolID)
	h.Write(poolBytes)

	// Token pair
	h.Write([]byte(tokenIn))
	h.Write([]byte(tokenOut))

	// Amounts
	h.Write([]byte(amountIn.String()))
	h.Write([]byte(minAmountOut.String()))

	// Salt and trader
	h.Write(salt)
	h.Write(trader.Bytes())

	return h.Sum(nil)
}

// CommitSwap stores a swap commitment for later reveal
func (k Keeper) CommitSwap(
	ctx context.Context,
	trader sdk.AccAddress,
	poolID uint64,
	commitmentHash []byte,
) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := k.getStore(ctx)

	// Check if commitment already exists
	key := SwapCommitmentKey(commitmentHash)
	if store.Has(key) {
		return types.ErrDuplicateCommitment.Wrap("swap commitment already exists")
	}

	// Collect deposit
	depositCoin := sdk.NewCoin("upaw", math.NewInt(CommitDepositAmount))
	if err := k.bankKeeper.SendCoinsFromAccountToModule(
		sdkCtx, trader, types.ModuleName, sdk.NewCoins(depositCoin),
	); err != nil {
		return types.ErrInsufficientDeposit.Wrapf("failed to collect commitment deposit: %v", err)
	}

	// Create commitment
	commitment := SwapCommitment{
		CommitmentHash: commitmentHash,
		Trader:         trader.String(),
		PoolID:         poolID,
		CommitBlock:    sdkCtx.BlockHeight(),
		ExpiryBlock:    sdkCtx.BlockHeight() + CommitExpiryBlocks,
		DepositAmount:  depositCoin.Amount,
		DepositDenom:   depositCoin.Denom,
	}

	// Store commitment
	bz, err := json.Marshal(commitment)
	if err != nil {
		return err
	}
	store.Set(key, bz)

	// Store expiry index for cleanup
	expiryKey := SwapCommitmentByExpiryKey(commitment.ExpiryBlock, commitmentHash)
	store.Set(expiryKey, commitmentHash)

	// Store trader index
	traderKey := SwapCommitmentByTraderKey(trader, commitmentHash)
	store.Set(traderKey, commitmentHash)

	// Emit event
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"swap_committed",
			sdk.NewAttribute("trader", trader.String()),
			sdk.NewAttribute("pool_id", fmt.Sprintf("%d", poolID)),
			sdk.NewAttribute("commit_block", fmt.Sprintf("%d", commitment.CommitBlock)),
			sdk.NewAttribute("expiry_block", fmt.Sprintf("%d", commitment.ExpiryBlock)),
			sdk.NewAttribute("deposit", depositCoin.String()),
		),
	)

	return nil
}

// RevealAndExecuteSwap reveals a committed swap and executes it
func (k Keeper) RevealAndExecuteSwap(
	ctx context.Context,
	trader sdk.AccAddress,
	poolID uint64,
	tokenIn, tokenOut string,
	amountIn, minAmountOut math.Int,
	salt []byte,
) (math.Int, error) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := k.getStore(ctx)

	// Compute expected commitment hash
	expectedHash := ComputeSwapCommitmentHash(poolID, tokenIn, tokenOut, amountIn, minAmountOut, salt, trader)

	// Retrieve commitment
	key := SwapCommitmentKey(expectedHash)
	bz := store.Get(key)
	if bz == nil {
		return math.ZeroInt(), types.ErrCommitmentNotFound.Wrap("no matching commitment found")
	}

	var commitment SwapCommitment
	if err := json.Unmarshal(bz, &commitment); err != nil {
		return math.ZeroInt(), err
	}

	// Verify trader matches
	if commitment.Trader != trader.String() {
		return math.ZeroInt(), types.ErrUnauthorized.Wrap("trader mismatch")
	}

	// Verify pool ID matches
	if commitment.PoolID != poolID {
		return math.ZeroInt(), types.ErrInvalidPool.Wrap("pool ID mismatch")
	}

	// Check reveal delay has passed
	if sdkCtx.BlockHeight() < commitment.CommitBlock+RevealDelayBlocks {
		return math.ZeroInt(), types.ErrRevealTooEarly.Wrapf(
			"reveal available at block %d, current block %d",
			commitment.CommitBlock+RevealDelayBlocks,
			sdkCtx.BlockHeight(),
		)
	}

	// Check commitment hasn't expired
	if sdkCtx.BlockHeight() > commitment.ExpiryBlock {
		// Commitment expired - deposit is forfeited (already handled by cleanup)
		return math.ZeroInt(), types.ErrCommitmentExpired.Wrapf(
			"commitment expired at block %d",
			commitment.ExpiryBlock,
		)
	}

	// Return deposit to trader
	depositCoin := sdk.NewCoin(commitment.DepositDenom, commitment.DepositAmount)
	if err := k.bankKeeper.SendCoinsFromModuleToAccount(
		sdkCtx, types.ModuleName, trader, sdk.NewCoins(depositCoin),
	); err != nil {
		sdkCtx.Logger().Error("failed to return commitment deposit", "error", err)
		// Continue with swap execution even if deposit return fails
	}

	// Delete commitment and indexes
	k.deleteSwapCommitment(ctx, expectedHash, commitment)

	// Execute the swap using the secure path
	amountOut, err := k.ExecuteSwapSecure(ctx, trader, poolID, tokenIn, tokenOut, amountIn, minAmountOut)
	if err != nil {
		return math.ZeroInt(), err
	}

	// Emit reveal event
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"swap_revealed",
			sdk.NewAttribute("trader", trader.String()),
			sdk.NewAttribute("pool_id", fmt.Sprintf("%d", poolID)),
			sdk.NewAttribute("amount_in", amountIn.String()),
			sdk.NewAttribute("amount_out", amountOut.String()),
			sdk.NewAttribute("commit_block", fmt.Sprintf("%d", commitment.CommitBlock)),
			sdk.NewAttribute("reveal_block", fmt.Sprintf("%d", sdkCtx.BlockHeight())),
		),
	)

	return amountOut, nil
}

// deleteSwapCommitment removes a commitment and its indexes
func (k Keeper) deleteSwapCommitment(ctx context.Context, commitmentHash []byte, commitment SwapCommitment) {
	store := k.getStore(ctx)

	// Delete main commitment
	store.Delete(SwapCommitmentKey(commitmentHash))

	// Delete expiry index
	store.Delete(SwapCommitmentByExpiryKey(commitment.ExpiryBlock, commitmentHash))

	// Delete trader index
	trader, err := sdk.AccAddressFromBech32(commitment.Trader)
	if err == nil {
		store.Delete(SwapCommitmentByTraderKey(trader, commitmentHash))
	}
}

// CleanupExpiredCommitments removes expired commitments and forfeits their deposits
// Called from EndBlocker
func (k Keeper) CleanupExpiredCommitments(ctx context.Context) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := k.getStore(ctx)
	currentBlock := sdkCtx.BlockHeight()

	// Iterate through commitments expiring at or before current block
	// We check a range of blocks to catch any that were missed
	for checkBlock := currentBlock - 10; checkBlock <= currentBlock; checkBlock++ {
		if checkBlock < 0 {
			continue
		}

		prefix := SwapCommitmentByExpiryKey(checkBlock, nil)
		prefix = prefix[:len(SwapCommitmentByExpiryPrefix)+8] // Just the prefix + block height

		iterator := store.Iterator(prefix, storetypes.PrefixEndBytes(prefix))
		defer iterator.Close()

		var toDelete [][]byte
		for ; iterator.Valid(); iterator.Next() {
			commitmentHash := iterator.Value()
			toDelete = append(toDelete, commitmentHash)
		}

		// Process expired commitments
		for _, commitmentHash := range toDelete {
			bz := store.Get(SwapCommitmentKey(commitmentHash))
			if bz == nil {
				continue
			}

			var commitment SwapCommitment
			if err := json.Unmarshal(bz, &commitment); err != nil {
				continue
			}

			// Forfeit deposit to protocol treasury
			depositCoin := sdk.NewCoin(commitment.DepositDenom, commitment.DepositAmount)
			if err := k.bankKeeper.SendCoinsFromModuleToModule(
				sdkCtx, types.ModuleName, "fee_collector", sdk.NewCoins(depositCoin),
			); err != nil {
				sdkCtx.Logger().Error("failed to forfeit expired commitment deposit", "error", err)
			}

			// Delete the commitment
			k.deleteSwapCommitment(ctx, commitmentHash, commitment)

			// Emit expiry event
			sdkCtx.EventManager().EmitEvent(
				sdk.NewEvent(
					"swap_commitment_expired",
					sdk.NewAttribute("trader", commitment.Trader),
					sdk.NewAttribute("pool_id", fmt.Sprintf("%d", commitment.PoolID)),
					sdk.NewAttribute("commit_block", fmt.Sprintf("%d", commitment.CommitBlock)),
					sdk.NewAttribute("expiry_block", fmt.Sprintf("%d", commitment.ExpiryBlock)),
					sdk.NewAttribute("forfeited_deposit", depositCoin.String()),
				),
			)
		}
	}

	return nil
}

// GetSwapCommitment retrieves a swap commitment by hash
func (k Keeper) GetSwapCommitment(ctx context.Context, commitmentHash []byte) (*SwapCommitment, error) {
	store := k.getStore(ctx)
	bz := store.Get(SwapCommitmentKey(commitmentHash))
	if bz == nil {
		return nil, types.ErrCommitmentNotFound.Wrap("commitment not found")
	}

	var commitment SwapCommitment
	if err := json.Unmarshal(bz, &commitment); err != nil {
		return nil, err
	}

	return &commitment, nil
}

// GetTraderCommitments retrieves all active commitments for a trader
func (k Keeper) GetTraderCommitments(ctx context.Context, trader sdk.AccAddress) ([]SwapCommitment, error) {
	store := k.getStore(ctx)
	prefix := SwapCommitmentByTraderKey(trader, nil)
	prefix = prefix[:len(SwapCommitmentByTraderPrefix)+len(trader.Bytes())]

	iterator := store.Iterator(prefix, storetypes.PrefixEndBytes(prefix))
	defer iterator.Close()

	var commitments []SwapCommitment
	for ; iterator.Valid(); iterator.Next() {
		commitmentHash := iterator.Value()
		commitment, err := k.GetSwapCommitment(ctx, commitmentHash)
		if err != nil {
			continue
		}
		commitments = append(commitments, *commitment)
	}

	return commitments, nil
}

// CancelSwapCommitment allows a trader to cancel their commitment before reveal
// Returns deposit minus a cancellation fee
func (k Keeper) CancelSwapCommitment(
	ctx context.Context,
	trader sdk.AccAddress,
	commitmentHash []byte,
) error {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := k.getStore(ctx)

	// Get commitment
	bz := store.Get(SwapCommitmentKey(commitmentHash))
	if bz == nil {
		return types.ErrCommitmentNotFound.Wrap("commitment not found")
	}

	var commitment SwapCommitment
	if err := json.Unmarshal(bz, &commitment); err != nil {
		return err
	}

	// Verify trader
	if commitment.Trader != trader.String() {
		return types.ErrUnauthorized.Wrap("only commitment owner can cancel")
	}

	// Calculate refund (90% of deposit, 10% cancellation fee to protocol)
	refundAmount := commitment.DepositAmount.Mul(math.NewInt(90)).Quo(math.NewInt(100))
	feeAmount := commitment.DepositAmount.Sub(refundAmount)

	// Send refund to trader
	refundCoin := sdk.NewCoin(commitment.DepositDenom, refundAmount)
	if err := k.bankKeeper.SendCoinsFromModuleToAccount(
		sdkCtx, types.ModuleName, trader, sdk.NewCoins(refundCoin),
	); err != nil {
		return err
	}

	// Send fee to protocol
	feeCoin := sdk.NewCoin(commitment.DepositDenom, feeAmount)
	if err := k.bankKeeper.SendCoinsFromModuleToModule(
		sdkCtx, types.ModuleName, "fee_collector", sdk.NewCoins(feeCoin),
	); err != nil {
		sdkCtx.Logger().Error("failed to send cancellation fee", "error", err)
	}

	// Delete commitment
	k.deleteSwapCommitment(ctx, commitmentHash, commitment)

	// Emit event
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"swap_commitment_cancelled",
			sdk.NewAttribute("trader", trader.String()),
			sdk.NewAttribute("pool_id", fmt.Sprintf("%d", commitment.PoolID)),
			sdk.NewAttribute("refund", refundCoin.String()),
			sdk.NewAttribute("fee", feeCoin.String()),
		),
	)

	return nil
}

// SwapWithCommitReveal executes a swap, using commit-reveal for large swaps
// This is the main entry point that checks if commit-reveal is required
func (k Keeper) SwapWithCommitReveal(
	ctx context.Context,
	trader sdk.AccAddress,
	poolID uint64,
	tokenIn, tokenOut string,
	amountIn, minAmountOut math.Int,
	commitmentHash []byte, // nil for regular swaps, set for reveal phase
	salt []byte,           // only needed for reveal phase
) (math.Int, error) {
	// Check if this is a reveal operation
	if len(commitmentHash) > 0 && len(salt) > 0 {
		return k.RevealAndExecuteSwap(ctx, trader, poolID, tokenIn, tokenOut, amountIn, minAmountOut, salt)
	}

	// Check if swap requires commit-reveal
	requiresCommit, err := k.RequiresCommitReveal(ctx, poolID, amountIn)
	if err != nil {
		return math.ZeroInt(), err
	}

	if requiresCommit {
		return math.ZeroInt(), types.ErrCommitRequired.Wrapf(
			"swap of %s exceeds %s threshold, commit-reveal required",
			amountIn.String(),
			LargeSwapThresholdPercent,
		)
	}

	// Regular swap - execute directly
	return k.ExecuteSwapSecure(ctx, trader, poolID, tokenIn, tokenOut, amountIn, minAmountOut)
}

// UpdateCommitRevealMetrics records metrics for commit-reveal operations
func (k Keeper) UpdateCommitRevealMetrics(ctx context.Context) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	store := k.getStore(ctx)

	// Count active commitments
	var activeCount int
	iterator := store.Iterator(SwapCommitmentKeyPrefix, storetypes.PrefixEndBytes(SwapCommitmentKeyPrefix))
	defer iterator.Close()
	for ; iterator.Valid(); iterator.Next() {
		activeCount++
	}

	// Log metrics (in production, would emit to Prometheus)
	sdkCtx.Logger().Debug("commit-reveal metrics",
		"active_commitments", activeCount,
		"block_height", sdkCtx.BlockHeight(),
		"time", time.Now().UTC(),
	)
}
