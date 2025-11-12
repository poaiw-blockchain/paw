package types

import (
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "cosmossdk.io/errors"
)

var _ sdk.Msg = &MsgCreatePool{}

// MsgCreatePool defines a message to create a new liquidity pool
type MsgCreatePool struct {
	Creator string  `json:"creator"`
	TokenA  string  `json:"token_a"`
	TokenB  string  `json:"token_b"`
	AmountA sdk.Int `json:"amount_a"`
	AmountB sdk.Int `json:"amount_b"`
}

// NewMsgCreatePool creates a new MsgCreatePool instance
func NewMsgCreatePool(creator, tokenA, tokenB string, amountA, amountB sdk.Int) *MsgCreatePool {
	return &MsgCreatePool{
		Creator: creator,
		TokenA:  tokenA,
		TokenB:  tokenB,
		AmountA: amountA,
		AmountB: amountB,
	}
}

// Route implements the sdk.Msg interface
func (msg MsgCreatePool) Route() string {
	return RouterKey
}

// Type implements the sdk.Msg interface
func (msg MsgCreatePool) Type() string {
	return "create_pool"
}

// GetSigners implements the sdk.Msg interface
func (msg MsgCreatePool) GetSigners() []sdk.AccAddress {
	creator, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{creator}
}

// GetSignBytes implements the sdk.Msg interface
func (msg MsgCreatePool) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(&msg)
	return sdk.MustSortJSON(bz)
}

// ValidateBasic implements the sdk.Msg interface
func (msg MsgCreatePool) ValidateBasic() error {
	// Validate creator address
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(ErrInvalidAddress, "invalid creator address: %s", err)
	}

	// Validate token denoms
	if msg.TokenA == "" || msg.TokenB == "" {
		return sdkerrors.Wrap(ErrInvalidTokenDenom, "token denominations cannot be empty")
	}

	if msg.TokenA == msg.TokenB {
		return sdkerrors.Wrap(ErrSameToken, "token denominations must be different")
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
