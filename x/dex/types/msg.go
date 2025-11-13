package types

import (
	sdkerrors "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Ensure all message types implement the sdk.Msg interface
var (
	_ sdk.Msg = &MsgCreatePool{}
	_ sdk.Msg = &MsgAddLiquidity{}
	_ sdk.Msg = &MsgRemoveLiquidity{}
	_ sdk.Msg = &MsgSwap{}
)

// NewMsgCreatePool creates a new MsgCreatePool instance
func NewMsgCreatePool(creator, tokenA, tokenB string, amountA, amountB math.Int) *MsgCreatePool {
	return &MsgCreatePool{
		Creator: creator,
		TokenA:  tokenA,
		TokenB:  tokenB,
		AmountA: amountA,
		AmountB: amountB,
	}
}

// ValidateBasic implements the sdk.Msg interface for MsgCreatePool
func (msg MsgCreatePool) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Creator)
	if err != nil {
		return sdkerrors.Wrapf(ErrInvalidAddress, "invalid creator address: %s", err)
	}

	if msg.TokenA == "" || msg.TokenB == "" {
		return sdkerrors.Wrap(ErrInvalidTokenDenom, "token denominations cannot be empty")
	}

	if msg.TokenA == msg.TokenB {
		return sdkerrors.Wrap(ErrSameToken, "token denominations must be different")
	}

	if msg.AmountA.IsNil() || msg.AmountA.LTE(math.ZeroInt()) {
		return sdkerrors.Wrap(ErrInvalidAmount, "amount A must be positive")
	}

	if msg.AmountB.IsNil() || msg.AmountB.LTE(math.ZeroInt()) {
		return sdkerrors.Wrap(ErrInvalidAmount, "amount B must be positive")
	}

	return nil
}

// NewMsgAddLiquidity creates a new MsgAddLiquidity instance
func NewMsgAddLiquidity(provider string, poolID uint64, amountA, amountB math.Int) *MsgAddLiquidity {
	return &MsgAddLiquidity{
		Provider: provider,
		PoolId:   poolID,
		AmountA:  amountA,
		AmountB:  amountB,
	}
}

// ValidateBasic implements the sdk.Msg interface for MsgAddLiquidity
func (msg MsgAddLiquidity) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Provider)
	if err != nil {
		return sdkerrors.Wrapf(ErrInvalidAddress, "invalid provider address: %s", err)
	}

	if msg.PoolId == 0 {
		return sdkerrors.Wrap(ErrInvalidPoolID, "pool ID must be positive")
	}

	if msg.AmountA.IsNil() || msg.AmountA.LTE(math.ZeroInt()) {
		return sdkerrors.Wrap(ErrInvalidAmount, "amount A must be positive")
	}

	if msg.AmountB.IsNil() || msg.AmountB.LTE(math.ZeroInt()) {
		return sdkerrors.Wrap(ErrInvalidAmount, "amount B must be positive")
	}

	return nil
}

// NewMsgRemoveLiquidity creates a new MsgRemoveLiquidity instance
func NewMsgRemoveLiquidity(provider string, poolID uint64, shares math.Int) *MsgRemoveLiquidity {
	return &MsgRemoveLiquidity{
		Provider: provider,
		PoolId:   poolID,
		Shares:   shares,
	}
}

// ValidateBasic implements the sdk.Msg interface for MsgRemoveLiquidity
func (msg MsgRemoveLiquidity) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Provider)
	if err != nil {
		return sdkerrors.Wrapf(ErrInvalidAddress, "invalid provider address: %s", err)
	}

	if msg.PoolId == 0 {
		return sdkerrors.Wrap(ErrInvalidPoolID, "pool ID must be positive")
	}

	if msg.Shares.IsNil() || msg.Shares.LTE(math.ZeroInt()) {
		return sdkerrors.Wrap(ErrInsufficientShares, "shares must be positive")
	}

	return nil
}

// NewMsgSwap creates a new MsgSwap instance
func NewMsgSwap(trader string, poolID uint64, tokenIn, tokenOut string, amountIn, minAmountOut math.Int) *MsgSwap {
	return &MsgSwap{
		Trader:       trader,
		PoolId:       poolID,
		TokenIn:      tokenIn,
		TokenOut:     tokenOut,
		AmountIn:     amountIn,
		MinAmountOut: minAmountOut,
	}
}

// ValidateBasic implements the sdk.Msg interface for MsgSwap
func (msg MsgSwap) ValidateBasic() error {
	_, err := sdk.AccAddressFromBech32(msg.Trader)
	if err != nil {
		return sdkerrors.Wrapf(ErrInvalidAddress, "invalid trader address: %s", err)
	}

	if msg.PoolId == 0 {
		return sdkerrors.Wrap(ErrInvalidPoolID, "pool ID must be positive")
	}

	if msg.TokenIn == "" || msg.TokenOut == "" {
		return sdkerrors.Wrap(ErrInvalidTokenDenom, "token denominations cannot be empty")
	}

	if msg.TokenIn == msg.TokenOut {
		return sdkerrors.Wrap(ErrSameToken, "input and output tokens must be different")
	}

	if msg.AmountIn.IsNil() || msg.AmountIn.LTE(math.ZeroInt()) {
		return sdkerrors.Wrap(ErrInvalidAmount, "amount in must be positive")
	}

	if msg.MinAmountOut.IsNil() || msg.MinAmountOut.LT(math.ZeroInt()) {
		return sdkerrors.Wrap(ErrInvalidAmount, "min amount out cannot be negative")
	}

	return nil
}
