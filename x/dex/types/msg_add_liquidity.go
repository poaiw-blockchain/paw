package types

import (
	sdkerrors "cosmossdk.io/errors"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ sdk.Msg = &MsgAddLiquidity{}

// MsgAddLiquidity defines a message to add liquidity to a pool
type MsgAddLiquidity struct {
	Provider string  `json:"provider"`
	PoolId   uint64  `json:"pool_id"`
	AmountA  sdk.Int `json:"amount_a"`
	AmountB  sdk.Int `json:"amount_b"`
}

// NewMsgAddLiquidity creates a new MsgAddLiquidity instance
func NewMsgAddLiquidity(provider string, poolId uint64, amountA, amountB sdk.Int) *MsgAddLiquidity {
	return &MsgAddLiquidity{
		Provider: provider,
		PoolId:   poolId,
		AmountA:  amountA,
		AmountB:  amountB,
	}
}

// Route implements the sdk.Msg interface
func (msg MsgAddLiquidity) Route() string {
	return RouterKey
}

// Type implements the sdk.Msg interface
func (msg MsgAddLiquidity) Type() string {
	return "add_liquidity"
}

// GetSigners implements the sdk.Msg interface
func (msg MsgAddLiquidity) GetSigners() []sdk.AccAddress {
	provider, err := sdk.AccAddressFromBech32(msg.Provider)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{provider}
}

// GetSignBytes implements the sdk.Msg interface
func (msg MsgAddLiquidity) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(&msg)
	return sdk.MustSortJSON(bz)
}

// ValidateBasic implements the sdk.Msg interface
func (msg MsgAddLiquidity) ValidateBasic() error {
	// Validate provider address
	_, err := sdk.AccAddressFromBech32(msg.Provider)
	if err != nil {
		return sdkerrors.Wrapf(ErrInvalidAddress, "invalid provider address: %s", err)
	}

	// Validate pool ID
	if msg.PoolId == 0 {
		return sdkerrors.Wrap(ErrInvalidPoolId, "pool id cannot be zero")
	}

	// Validate amounts
	if msg.AmountA.IsNil() || msg.AmountA.LTE(sdk.ZeroInt()) {
		return sdkerrors.Wrap(ErrInvalidAmount, "amount A must be positive")
	}

	if msg.AmountB.IsNil() || msg.AmountB.LTE(sdk.ZeroInt()) {
		return sdkerrors.Wrap(ErrInvalidAmount, "amount B must be positive")
	}

	return nil
}
