package ante

import (
	errorsmod "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"
)

// MemoLimitDecorator enforces a hard cap on memo size (bytes).
// This runs early in the ante chain to bound payload size before further processing.
type MemoLimitDecorator struct {
	maxBytes int
}

// NewMemoLimitDecorator returns a decorator that rejects memos exceeding maxBytes.
func NewMemoLimitDecorator(maxBytes int) MemoLimitDecorator {
	return MemoLimitDecorator{maxBytes: maxBytes}
}

// AnteHandle implements sdk.AnteDecorator.
func (d MemoLimitDecorator) AnteHandle(ctx sdk.Context, tx sdk.Tx, simulate bool, next sdk.AnteHandler) (sdk.Context, error) {
	if withMemo, ok := tx.(sdk.TxWithMemo); ok {
		memo := withMemo.GetMemo()
		if len(memo) > d.maxBytes {
			return ctx, errorsmod.Wrapf(sdkerrors.ErrInvalidRequest, "memo too large: %d bytes (max %d)", len(memo), d.maxBytes)
		}
	}

	return next(ctx, tx, simulate)
}
