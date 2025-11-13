package types

import (
	sdkerrors "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

var _ sdk.Msg = &MsgRemoveLiquidity{}

// MsgRemoveLiquidity defines a message to remove liquidity from a pool
type MsgRemoveLiquidity struct {
	Provider string   `json:"provider"`
	PoolId   uint64   `json:"pool_id"`
	Shares   math.Int `json:"shares"`
}

// NewMsgRemoveLiquidity creates a new MsgRemoveLiquidity instance
func NewMsgRemoveLiquidity(provider string, poolId uint64, shares math.Int) *MsgRemoveLiquidity {
	return &MsgRemoveLiquidity{
		Provider: provider,
		PoolId:   poolId,
		Shares:   shares,
	}
}

// Route implements the sdk.Msg interface
func (msg MsgRemoveLiquidity) Route() string {
	return RouterKey
}

// Type implements the sdk.Msg interface
func (msg MsgRemoveLiquidity) Type() string {
	return "remove_liquidity"
}

// GetSigners implements the sdk.Msg interface
func (msg MsgRemoveLiquidity) GetSigners() []sdk.AccAddress {
	provider, err := sdk.AccAddressFromBech32(msg.Provider)
	if err != nil {
		panic(err)
	}
	return []sdk.AccAddress{provider}
}

// GetSignBytes implements the sdk.Msg interface
func (msg MsgRemoveLiquidity) GetSignBytes() []byte {
	bz := ModuleCdc.MustMarshalJSON(&msg)
	return sdk.MustSortJSON(bz)
}

// ValidateBasic implements the sdk.Msg interface
func (msg MsgRemoveLiquidity) ValidateBasic() error {
	// Validate provider address
	_, err := sdk.AccAddressFromBech32(msg.Provider)
	if err != nil {
		return sdkerrors.Wrapf(ErrInvalidAddress, "invalid provider address: %s", err)
	}

	// Validate pool ID
	if msg.PoolId == 0 {
		return sdkerrors.Wrap(ErrInvalidPoolId, "pool id cannot be zero")
	}

	// Validate shares
	if msg.Shares.IsNil() || msg.Shares.LTE(sdk.ZeroInt()) {
		return sdkerrors.Wrap(ErrInvalidShares, "shares must be positive")
	}

	return nil
}
