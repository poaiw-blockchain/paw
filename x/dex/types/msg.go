package types

import (
	"regexp"
	"strings"

	sdkerrors "cosmossdk.io/errors"
	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Security validation constants
const (
	maxTokenDenomLength = 128
)

// Dangerous patterns for injection attacks
var (
	// SQL injection patterns
	sqlInjectionPattern = regexp.MustCompile(`(?i)(--|;|'|\"|union|select|insert|update|delete|drop|create|alter|exec|execute|script|javascript|onclick|onerror|onload)`)
	// XSS patterns
	xssPattern = regexp.MustCompile(`(?i)(<script|<iframe|javascript:|onerror=|onload=|onclick=)`)
	// XML injection patterns
	xmlInjectionPattern = regexp.MustCompile(`(?i)(<!DOCTYPE|<!ENTITY|SYSTEM|file:///)`)
	// Valid denom pattern (alphanumeric, dash, underscore)
	validDenomPattern = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9/_\-.]*$`)
)

// validateTokenDenom checks if a token denomination is valid and safe
func validateTokenDenom(denom string) error {
	if denom == "" {
		return sdkerrors.Wrap(ErrInvalidTokenDenom, "token denomination cannot be empty")
	}

	if len(denom) > maxTokenDenomLength {
		return sdkerrors.Wrap(ErrInvalidTokenDenom, "token denomination too long")
	}

	// Check for SQL injection patterns
	if sqlInjectionPattern.MatchString(denom) {
		return sdkerrors.Wrap(ErrInvalidTokenDenom, "token denomination contains suspicious SQL pattern")
	}

	// Check for XSS patterns
	if xssPattern.MatchString(denom) {
		return sdkerrors.Wrap(ErrInvalidTokenDenom, "token denomination contains suspicious XSS pattern")
	}

	// Check for XML injection patterns
	if xmlInjectionPattern.MatchString(denom) {
		return sdkerrors.Wrap(ErrInvalidTokenDenom, "token denomination contains suspicious XML pattern")
	}

	// Check for dangerous shell characters
	dangerousChars := []string{";", "|", "&", "`", "$", "(", ")", "<", ">", "\n", "\r"}
	for _, char := range dangerousChars {
		if strings.Contains(denom, char) {
			return sdkerrors.Wrap(ErrInvalidTokenDenom, "token denomination contains dangerous character")
		}
	}

	// Check valid pattern
	if !validDenomPattern.MatchString(denom) {
		return sdkerrors.Wrap(ErrInvalidTokenDenom, "token denomination must start with a letter and contain only alphanumeric characters, dash, underscore, slash, or dot")
	}

	return nil
}

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

	// Validate token denominations for security
	if err := validateTokenDenom(msg.TokenA); err != nil {
		return sdkerrors.Wrap(err, "invalid token A")
	}
	if err := validateTokenDenom(msg.TokenB); err != nil {
		return sdkerrors.Wrap(err, "invalid token B")
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

	// Validate token denominations for security
	if err := validateTokenDenom(msg.TokenIn); err != nil {
		return sdkerrors.Wrap(err, "invalid token in")
	}
	if err := validateTokenDenom(msg.TokenOut); err != nil {
		return sdkerrors.Wrap(err, "invalid token out")
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
