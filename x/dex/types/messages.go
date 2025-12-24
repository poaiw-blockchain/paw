package types

import (
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// Message type URLs
const (
	TypeMsgCreatePool      = "create_pool"
	TypeMsgAddLiquidity    = "add_liquidity"
	TypeMsgRemoveLiquidity = "remove_liquidity"
	TypeMsgSwap            = "swap"
)

var (
	_ sdk.Msg = &MsgCreatePool{}
	_ sdk.Msg = &MsgAddLiquidity{}
	_ sdk.Msg = &MsgRemoveLiquidity{}
	_ sdk.Msg = &MsgSwap{}
	_ sdk.Msg = &MsgCommitSwap{}
	_ sdk.Msg = &MsgRevealSwap{}
)

// MsgCreatePool implementations

// ValidateBasic performs basic validation of MsgCreatePool
func (m *MsgCreatePool) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Creator); err != nil {
		return fmt.Errorf("invalid creator address: %w", err)
	}

	if m.TokenA == "" {
		return fmt.Errorf("token_a cannot be empty")
	}

	if m.TokenB == "" {
		return fmt.Errorf("token_b cannot be empty")
	}

	if m.TokenA == m.TokenB {
		return fmt.Errorf("tokens must be different")
	}

	if err := sdk.ValidateDenom(m.TokenA); err != nil {
		return fmt.Errorf("invalid denom for token_a: %w", err)
	}

	if err := sdk.ValidateDenom(m.TokenB); err != nil {
		return fmt.Errorf("invalid denom for token_b: %w", err)
	}

	if m.AmountA.IsZero() || m.AmountA.IsNegative() {
		return fmt.Errorf("amount_a must be positive")
	}

	if m.AmountB.IsZero() || m.AmountB.IsNegative() {
		return fmt.Errorf("amount_b must be positive")
	}

	return nil
}

// GetSigners returns the expected signers for MsgCreatePool
// Assumes address is valid (validated in ValidateBasic)
func (m *MsgCreatePool) GetSigners() []sdk.AccAddress {
	creator, _ := sdk.AccAddressFromBech32(m.Creator)
	return []sdk.AccAddress{creator}
}

// MsgAddLiquidity implementations

// ValidateBasic performs basic validation of MsgAddLiquidity
func (m *MsgAddLiquidity) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Provider); err != nil {
		return fmt.Errorf("invalid provider address: %w", err)
	}

	if m.PoolId == 0 {
		return fmt.Errorf("pool_id must be positive")
	}

	if m.AmountA.IsZero() || m.AmountA.IsNegative() {
		return fmt.Errorf("amount_a must be positive")
	}

	if m.AmountB.IsZero() || m.AmountB.IsNegative() {
		return fmt.Errorf("amount_b must be positive")
	}

	return nil
}

// GetSigners returns the expected signers for MsgAddLiquidity
// Assumes address is valid (validated in ValidateBasic)
func (m *MsgAddLiquidity) GetSigners() []sdk.AccAddress {
	provider, _ := sdk.AccAddressFromBech32(m.Provider)
	return []sdk.AccAddress{provider}
}

// MsgRemoveLiquidity implementations

// ValidateBasic performs basic validation of MsgRemoveLiquidity
func (m *MsgRemoveLiquidity) ValidateBasic() error {
	if _, err := sdk.AccAddressFromBech32(m.Provider); err != nil {
		return fmt.Errorf("invalid provider address: %w", err)
	}

	if m.PoolId == 0 {
		return fmt.Errorf("pool_id must be positive")
	}

	if m.Shares.IsZero() || m.Shares.IsNegative() {
		return fmt.Errorf("shares must be positive")
	}

	return nil
}

// GetSigners returns the expected signers for MsgRemoveLiquidity
// Assumes address is valid (validated in ValidateBasic)
func (m *MsgRemoveLiquidity) GetSigners() []sdk.AccAddress {
	provider, _ := sdk.AccAddressFromBech32(m.Provider)
	return []sdk.AccAddress{provider}
}

// MsgSwap implementations

// ValidateBasic performs basic validation of MsgSwap
func (m *MsgSwap) ValidateBasic() error {
	if m.Trader == "" {
		return fmt.Errorf("trader address cannot be empty")
	}

	if _, err := sdk.AccAddressFromBech32(m.Trader); err != nil {
		return fmt.Errorf("invalid trader address: %w", err)
	}

	if m.PoolId == 0 {
		return fmt.Errorf("pool_id must be positive")
	}

	if m.TokenIn == "" {
		return fmt.Errorf("token_in cannot be empty")
	}

	if m.TokenOut == "" {
		return fmt.Errorf("token_out cannot be empty")
	}

	if err := sdk.ValidateDenom(m.TokenIn); err != nil {
		return fmt.Errorf("invalid denom for token_in: %w", err)
	}

	if err := sdk.ValidateDenom(m.TokenOut); err != nil {
		return fmt.Errorf("invalid denom for token_out: %w", err)
	}

	if m.TokenIn == m.TokenOut {
		return fmt.Errorf("cannot swap the same token denomination")
	}

	if m.AmountIn.IsZero() || m.AmountIn.IsNegative() {
		return fmt.Errorf("invalid amount: amount_in must be positive")
	}

	if m.MinAmountOut.IsNegative() {
		return fmt.Errorf("min_amount_out cannot be negative")
	}

	// Slippage protection: MinAmountOut must be set (non-zero) to protect against excessive slippage
	// This forces users to explicitly specify their slippage tolerance
	if m.MinAmountOut.IsZero() {
		return fmt.Errorf("min_amount_out must be positive for slippage protection")
	}

	// Sanity check: MinAmountOut should not exceed AmountIn (accounting for different decimals is done in keeper)
	// This prevents obviously invalid swap parameters
	if m.MinAmountOut.GT(m.AmountIn.Mul(math.NewInt(1000))) {
		return fmt.Errorf("min_amount_out unreasonably high compared to amount_in")
	}

	// Deadline validation: deadline must be set to protect against stale transactions
	if m.Deadline == 0 {
		return fmt.Errorf("deadline must be set for time-sensitive swap operations")
	}

	// Deadline must be in the future (basic sanity check - actual time check happens in keeper)
	// This prevents obviously past deadlines (e.g., someone using milliseconds instead of seconds)
	if m.Deadline < 0 {
		return fmt.Errorf("deadline must be a positive unix timestamp")
	}

	return nil
}

// GetSigners returns the expected signers for MsgSwap
// Assumes address is valid (validated in ValidateBasic)
func (m *MsgSwap) GetSigners() []sdk.AccAddress {
	trader, _ := sdk.AccAddressFromBech32(m.Trader)
	return []sdk.AccAddress{trader}
}

// MsgCommitSwap implementations

// ValidateBasic performs basic validation of MsgCommitSwap
func (m *MsgCommitSwap) ValidateBasic() error {
	if m.Trader == "" {
		return fmt.Errorf("trader address cannot be empty")
	}

	if _, err := sdk.AccAddressFromBech32(m.Trader); err != nil {
		return fmt.Errorf("invalid trader address: %w", err)
	}

	if m.SwapHash == "" {
		return fmt.Errorf("swap_hash cannot be empty")
	}

	// Basic length check - hash should be hex-encoded (64 characters for 32 bytes)
	if len(m.SwapHash) != 64 {
		return fmt.Errorf("swap_hash must be 64 hex characters (32 bytes)")
	}

	// Verify it's valid hex
	for _, c := range m.SwapHash {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return fmt.Errorf("swap_hash must be valid hexadecimal")
		}
	}

	return nil
}

// GetSigners returns the expected signers for MsgCommitSwap
func (m *MsgCommitSwap) GetSigners() []sdk.AccAddress {
	trader, _ := sdk.AccAddressFromBech32(m.Trader)
	return []sdk.AccAddress{trader}
}

// MsgRevealSwap implementations

// ValidateBasic performs basic validation of MsgRevealSwap
func (m *MsgRevealSwap) ValidateBasic() error {
	if m.Trader == "" {
		return fmt.Errorf("trader address cannot be empty")
	}

	if _, err := sdk.AccAddressFromBech32(m.Trader); err != nil {
		return fmt.Errorf("invalid trader address: %w", err)
	}

	if m.PoolId == 0 {
		return fmt.Errorf("pool_id must be positive")
	}

	if m.TokenIn == "" {
		return fmt.Errorf("token_in cannot be empty")
	}

	if m.TokenOut == "" {
		return fmt.Errorf("token_out cannot be empty")
	}

	if err := sdk.ValidateDenom(m.TokenIn); err != nil {
		return fmt.Errorf("invalid denom for token_in: %w", err)
	}

	if err := sdk.ValidateDenom(m.TokenOut); err != nil {
		return fmt.Errorf("invalid denom for token_out: %w", err)
	}

	if m.TokenIn == m.TokenOut {
		return fmt.Errorf("cannot swap the same token denomination")
	}

	if m.AmountIn.IsZero() || m.AmountIn.IsNegative() {
		return fmt.Errorf("invalid amount: amount_in must be positive")
	}

	if m.MinAmountOut.IsNegative() {
		return fmt.Errorf("min_amount_out cannot be negative")
	}

	if m.MinAmountOut.IsZero() {
		return fmt.Errorf("min_amount_out must be positive for slippage protection")
	}

	if m.Deadline == 0 {
		return fmt.Errorf("deadline must be set for time-sensitive swap operations")
	}

	if m.Deadline < 0 {
		return fmt.Errorf("deadline must be a positive unix timestamp")
	}

	if m.Nonce == "" {
		return fmt.Errorf("nonce cannot be empty")
	}

	// Nonce should be at least 16 characters for security
	if len(m.Nonce) < 16 {
		return fmt.Errorf("nonce must be at least 16 characters for security")
	}

	return nil
}

// GetSigners returns the expected signers for MsgRevealSwap
func (m *MsgRevealSwap) GetSigners() []sdk.AccAddress {
	trader, _ := sdk.AccAddressFromBech32(m.Trader)
	return []sdk.AccAddress{trader}
}
