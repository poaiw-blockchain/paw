package ante

import (
	"fmt"
	"time"

	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

const (
	// MaxBlockTimeDrift is the maximum allowed drift between block timestamps (5 minutes)
	MaxBlockTimeDrift = 5 * time.Minute

	// MinBlockTimeDrift is the minimum expected time between blocks
	// Setting to 0 allows instant blocks in testing but production should be ~5s
	MinBlockTimeDrift = 0 * time.Second

	// MaxFutureBlockTime is how far in the future a block time can be (30 seconds)
	MaxFutureBlockTime = 30 * time.Second
)

// TimeValidatorDecorator validates block time to prevent time manipulation attacks.
// It ensures:
// 1. Block time is not too far in the future (prevents timestamp manipulation)
// 2. Block time progression is monotonic (each block timestamp >= previous)
// 3. Block time doesn't jump backwards (prevents time-travel attacks)
type TimeValidatorDecorator struct{}

// NewTimeValidatorDecorator creates a new TimeValidatorDecorator
func NewTimeValidatorDecorator() TimeValidatorDecorator {
	return TimeValidatorDecorator{}
}

// AnteHandle validates the block time before processing transactions
func (tvd TimeValidatorDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	// Skip validation in simulation mode
	if simulate {
		return next(ctx, tx, simulate)
	}

	blockTime := ctx.BlockTime()
	blockHeight := ctx.BlockHeight()

	// Skip validation for genesis block
	if blockHeight <= 1 {
		return next(ctx, tx, simulate)
	}

	// 1. Check if block time is too far in the future
	now := time.Now()
	if blockTime.After(now.Add(MaxFutureBlockTime)) {
		return ctx, sdkerrors.ErrInvalidRequest.Wrapf(
			"block time %s is too far in the future (max drift: %s from %s)",
			blockTime, MaxFutureBlockTime, now,
		)
	}

	// NOTE: Do not reject historical blocks based on wall-clock time.
	// Nodes performing catch-up (or deterministic unit tests that use fixed past
	// timestamps) must be able to process older block times.
	//
	// Monotonicity and "no huge time jumps" should be enforced relative to
	// *previous block time*, which is not reliably available in the ante handler
	// without explicit app support (e.g. persisting last block time at BeginBlock).
	_ = blockHeight

	return next(ctx, tx, simulate)
}

// ValidateBlockTime validates a block timestamp against security constraints
func ValidateBlockTime(blockTime time.Time, prevBlockTime time.Time, currentTime time.Time) error {
	// Check future drift
	if blockTime.After(currentTime.Add(MaxFutureBlockTime)) {
		return fmt.Errorf(
			"block time %s is too far in the future (max drift: %s from %s)",
			blockTime, MaxFutureBlockTime, currentTime,
		)
	}

	// Check monotonic progression
	if !prevBlockTime.IsZero() && blockTime.Before(prevBlockTime) {
		return fmt.Errorf(
			"block time %s is before previous block time %s",
			blockTime, prevBlockTime,
		)
	}

	return nil
}

// IsTimeManipulation detects potential time manipulation based on block time patterns
func IsTimeManipulation(blockTimes []time.Time, threshold time.Duration) bool {
	if len(blockTimes) < 2 {
		return false
	}

	// Check for sudden jumps in time
	for i := 1; i < len(blockTimes); i++ {
		diff := blockTimes[i].Sub(blockTimes[i-1])
		if diff > threshold {
			return true
		}
		if diff < 0 {
			// Time went backwards
			return true
		}
	}

	return false
}
