package types

import (
	sdkerrors "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ sdk.Msg = &MsgSwap{}

// MsgSwap defines a message to swap tokens using AMM
type MsgSwap struct {
	Trader       string   `json:"trader"`
	PoolId       uint64   `json:"pool_id"`
	TokenIn      string   `json:"token_in"`
	TokenOut     string   `json:"token_out"`
	AmountIn     math.Int `json:"amount_in"`
	MinAmountOut math.Int `json:"min_amount_out"`
}

// NewMsgSwap creates a new MsgSwap instance
func NewMsgSwap(trader string, poolId uint64, tokenIn, tokenOut string, amountIn, minAmountOut math.Int) *MsgSwap {
	return &MsgSwap{
		Trader:       trader,
		PoolId:       poolId,
		TokenIn:      tokenIn,
		TokenOut:     tokenOut,
		AmountIn:     amountIn,
		MinAmountOut: minAmountOut,
	}
}

// Route implements the sdk.Msg interface
func (msg MsgSwap) Route() string {
	return RouterKey
}

// Type implements the sdk.Msg interface
func (msg MsgSwap) Type() string {
	return "swap"
}

// GetSigners implements the sdk.Msg interface
func (msg MsgSwap) GetSigners() []sdk.AccAddress {
	trader, err := sdk.AccAddressFromBech32(msg.Trader)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{trader}
}

// GetSignBytes implements the sdk.Msg interface
func (msg MsgSwap) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(&msg)
	return sdk.MustSortJSON(bz)
}

// ValidateBasic implements the sdk.Msg interface
func (msg MsgSwap) ValidateBasic() error {
	// Validate trader address
	_, err := sdk.AccAddressFromBech32(msg.Trader)
	if err != nil {
		return sdkerrors.Wrapf(ErrInvalidAddress, "invalid trader address: %s", err)
	}

	// Validate pool ID
	if msg.PoolId == 0 {
		return sdkerrors.Wrap(ErrInvalidPoolId, "pool id cannot be zero")
	}

	// Validate token denoms
	if msg.TokenIn == "" || msg.TokenOut == "" {
		return sdkerrors.Wrap(ErrInvalidTokenDenom, "token denominations cannot be empty")
	}

	if msg.TokenIn == msg.TokenOut {
		return sdkerrors.Wrap(ErrSameToken, "cannot swap same token")
	}

	// Validate amounts
	if msg.AmountIn.IsNil() || msg.AmountIn.LTE(sdk.ZeroInt()) {
		return sdkerrors.Wrap(ErrInvalidAmount, "amount in must be positive")
	}

	if msg.MinAmountOut.IsNil() || msg.MinAmountOut.IsNegative() {
		return sdkerrors.Wrap(ErrInvalidAmount, "min amount out cannot be negative")
	}

	return nil
}
