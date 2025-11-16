package keeper

import (
	"encoding/json"
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/dex/types"
)

// FlashLoanActivity tracks flash loan activity for detection
type FlashLoanActivity struct {
	Address     string
	PoolID      uint64
	BorrowBlock int64
	BorrowTime  int64
	Amount      math.Int
	TokenDenom  string
	Repaid      bool
}

// TrackBorrow tracks a borrow operation that might be a flash loan
func (k Keeper) TrackBorrow(ctx sdk.Context, borrower string, poolId uint64, amount math.Int, denom string) {
	activity := FlashLoanActivity{
		Address:     borrower,
		PoolID:      poolId,
		BorrowBlock: ctx.BlockHeight(),
		BorrowTime:  ctx.BlockTime().Unix(),
		Amount:      amount,
		TokenDenom:  denom,
		Repaid:      false,
	}

	// Store flash loan activity
	k.SetFlashLoanActivity(ctx, borrower, poolId, activity)
}

// TrackRepayment tracks a repayment of a borrow
func (k Keeper) TrackRepayment(ctx sdk.Context, borrower string, poolId uint64, amount math.Int, denom string) {
	// Get flash loan activity
	activity, found := k.GetFlashLoanActivity(ctx, borrower, poolId)
	if !found {
		return
	}

	// Check if this is a same-block repayment (flash loan)
	if activity.BorrowBlock == ctx.BlockHeight() {
		// This is a flash loan!
		k.LogFlashLoan(ctx, borrower, poolId, activity.Amount, denom)

		// Check if flash loans are allowed
		if !k.AreFlashLoansAllowed(ctx) {
			// Flash loans are not allowed - this should have been prevented earlier
			ctx.EventManager().EmitEvent(
				sdk.NewEvent(
					"flash_loan_detected_violation",
					sdk.NewAttribute("borrower", borrower),
					sdk.NewAttribute("pool_id", fmt.Sprintf("%d", poolId)),
					sdk.NewAttribute("amount", amount.String()),
					sdk.NewAttribute("denom", denom),
					sdk.NewAttribute("block_height", fmt.Sprintf("%d", ctx.BlockHeight())),
				),
			)
		}
	}

	// Mark as repaid
	activity.Repaid = true
	k.SetFlashLoanActivity(ctx, borrower, poolId, activity)
}

// IsFlashLoanAttempt checks if a borrow-swap-repay pattern is being attempted in the same block
func (k Keeper) IsFlashLoanAttempt(ctx sdk.Context, address string, poolId uint64) bool {
	activity, found := k.GetFlashLoanActivity(ctx, address, poolId)
	if !found {
		return false
	}

	// Check if there's an active borrow in the current block
	return activity.BorrowBlock == ctx.BlockHeight() && !activity.Repaid
}

// DetectFlashLoanPattern detects suspicious flash loan patterns
// This checks for:
// 1. Large borrows relative to pool size
// 2. Multiple swaps in same block
// 3. Borrow-swap-repay in same block
func (k Keeper) DetectFlashLoanPattern(ctx sdk.Context, trader string, poolId uint64, swapAmount math.Int) (bool, string) {
	pool := k.GetPool(ctx, poolId)
	if pool == nil {
		return false, ""
	}

	// Check 1: Is there an active flash loan in this block?
	if k.IsFlashLoanAttempt(ctx, trader, poolId) {
		return true, "active_flash_loan_detected"
	}

	// Check 2: Is the swap amount > 10% of pool liquidity?
	totalLiquidity := pool.ReserveA.Add(pool.ReserveB)
	swapThreshold := totalLiquidity.Mul(math.NewInt(10)).Quo(math.NewInt(100)) // 10%

	if swapAmount.GT(swapThreshold) {
		// Large swap detected - potential flash loan
		largeSwapCount := k.GetLargeSwapCount(ctx, trader, ctx.BlockHeight())
		k.IncrementLargeSwapCount(ctx, trader, ctx.BlockHeight())

		if largeSwapCount > 0 {
			// Multiple large swaps in same block - highly suspicious
			return true, "multiple_large_swaps_same_block"
		}

		return true, "large_swap_detected"
	}

	// Check 3: Multiple swaps to same pool in same block
	swapCount := k.GetSwapCount(ctx, trader, poolId, ctx.BlockHeight())
	k.IncrementSwapCount(ctx, trader, poolId, ctx.BlockHeight())

	if swapCount >= 3 {
		// More than 3 swaps to same pool in same block
		return true, "excessive_swaps_same_block"
	}

	return false, ""
}

// LogFlashLoan logs a detected flash loan event
func (k Keeper) LogFlashLoan(ctx sdk.Context, borrower string, poolId uint64, amount math.Int, denom string) {
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"flash_loan_detected",
			sdk.NewAttribute("borrower", borrower),
			sdk.NewAttribute("pool_id", fmt.Sprintf("%d", poolId)),
			sdk.NewAttribute("amount", amount.String()),
			sdk.NewAttribute("denom", denom),
			sdk.NewAttribute("block_height", fmt.Sprintf("%d", ctx.BlockHeight())),
			sdk.NewAttribute("timestamp", fmt.Sprintf("%d", ctx.BlockTime().Unix())),
		),
	)
}

// AreFlashLoansAllowed checks if flash loans are allowed (governance parameter)
func (k Keeper) AreFlashLoansAllowed(ctx sdk.Context) bool {
	// For now, allow flash loans but monitor them
	// In production, this should be a governance parameter
	return true
}

// Storage functions

func (k Keeper) GetFlashLoanActivity(ctx sdk.Context, address string, poolId uint64) (FlashLoanActivity, bool) {
	store := ctx.KVStore(k.storeKey)
	key := types.GetFlashLoanKey(address, poolId)
	bz := store.Get(key)

	if bz == nil {
		return FlashLoanActivity{}, false
	}

	var activity FlashLoanActivity
	if err := json.Unmarshal(bz, &activity); err != nil {
		// If unmarshal fails, return empty activity
		return FlashLoanActivity{}, false
	}
	return activity, true
}

func (k Keeper) SetFlashLoanActivity(ctx sdk.Context, address string, poolId uint64, activity FlashLoanActivity) {
	store := ctx.KVStore(k.storeKey)
	key := types.GetFlashLoanKey(address, poolId)
	bz, err := json.Marshal(&activity)
	if err != nil {
		// Should never happen with simple struct
		panic(fmt.Sprintf("failed to marshal flash loan activity: %v", err))
	}
	store.Set(key, bz)
}

func (k Keeper) GetSwapCount(ctx sdk.Context, address string, poolId uint64, blockHeight int64) int {
	store := ctx.KVStore(k.storeKey)
	key := types.GetSwapCountKey(address, poolId, blockHeight)
	bz := store.Get(key)

	if bz == nil {
		return 0
	}

	return int(sdk.BigEndianToUint64(bz))
}

func (k Keeper) IncrementSwapCount(ctx sdk.Context, address string, poolId uint64, blockHeight int64) {
	count := k.GetSwapCount(ctx, address, poolId, blockHeight)
	store := ctx.KVStore(k.storeKey)
	key := types.GetSwapCountKey(address, poolId, blockHeight)
	store.Set(key, sdk.Uint64ToBigEndian(uint64(count+1)))
}

func (k Keeper) GetLargeSwapCount(ctx sdk.Context, address string, blockHeight int64) int {
	store := ctx.KVStore(k.storeKey)
	key := types.GetLargeSwapCountKey(address, blockHeight)
	bz := store.Get(key)

	if bz == nil {
		return 0
	}

	return int(sdk.BigEndianToUint64(bz))
}

func (k Keeper) IncrementLargeSwapCount(ctx sdk.Context, address string, blockHeight int64) {
	count := k.GetLargeSwapCount(ctx, address, blockHeight)
	store := ctx.KVStore(k.storeKey)
	key := types.GetLargeSwapCountKey(address, blockHeight)
	store.Set(key, sdk.Uint64ToBigEndian(uint64(count+1)))
}

// CleanupOldFlashLoanData cleans up flash loan tracking data older than N blocks
// This should be called in EndBlocker to prevent state bloat
func (k Keeper) CleanupOldFlashLoanData(ctx sdk.Context, blockRetention int64) {
	// Only run cleanup every 100 blocks to save gas
	if ctx.BlockHeight()%100 != 0 {
		return
	}

	cutoffBlock := ctx.BlockHeight() - blockRetention

	// In production, implement proper iteration and cleanup
	// For now, this is a placeholder to show the concept
	k.Logger(ctx).Info("Cleaning up old flash loan data", "cutoff_block", cutoffBlock)
}
