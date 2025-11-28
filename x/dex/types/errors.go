package types

import (
	"errors"

	sdkerrors "cosmossdk.io/errors"
)

// Sentinel errors for the DEX module
var (
	// ErrInvalidPoolState is returned when pool state is invalid
	ErrInvalidPoolState = sdkerrors.Register(ModuleName, 2, "invalid pool state")

	// ErrInsufficientLiquidity is returned when pool has insufficient liquidity
	ErrInsufficientLiquidity = sdkerrors.Register(ModuleName, 3, "insufficient liquidity")

	// ErrInvalidInput is returned when input parameters are invalid
	ErrInvalidInput = sdkerrors.Register(ModuleName, 4, "invalid input")

	// ErrReentrancy is returned when a reentrancy attack is detected
	ErrReentrancy = sdkerrors.Register(ModuleName, 5, "reentrancy detected")

	// ErrInvariantViolation is returned when pool invariants are violated
	ErrInvariantViolation = sdkerrors.Register(ModuleName, 6, "invariant violation")

	// ErrCircuitBreakerTriggered is returned when circuit breaker is active
	ErrCircuitBreakerTriggered = sdkerrors.Register(ModuleName, 7, "circuit breaker triggered")

	// ErrSwapTooLarge is returned when swap size exceeds limits
	ErrSwapTooLarge = sdkerrors.Register(ModuleName, 8, "swap too large")

	// ErrPriceImpactTooHigh is returned when price impact exceeds limits
	ErrPriceImpactTooHigh = sdkerrors.Register(ModuleName, 9, "price impact too high")

	// ErrFlashLoanDetected is returned when flash loan attack is detected
	ErrFlashLoanDetected = sdkerrors.Register(ModuleName, 10, "flash loan attack detected")

	// ErrOverflow is returned when arithmetic overflow occurs
	ErrOverflow = sdkerrors.Register(ModuleName, 11, "arithmetic overflow")

	// ErrUnderflow is returned when arithmetic underflow occurs
	ErrUnderflow = sdkerrors.Register(ModuleName, 12, "arithmetic underflow")

	// ErrDivisionByZero is returned when division by zero is attempted
	ErrDivisionByZero = sdkerrors.Register(ModuleName, 13, "division by zero")

	// ErrPoolNotFound is returned when pool doesn't exist
	ErrPoolNotFound = sdkerrors.Register(ModuleName, 14, "pool not found")

	// ErrPoolAlreadyExists is returned when attempting to create duplicate pool
	ErrPoolAlreadyExists = sdkerrors.Register(ModuleName, 15, "pool already exists")

	// ErrRateLimitExceeded is returned when rate limit is exceeded
	ErrRateLimitExceeded = sdkerrors.Register(ModuleName, 19, "rate limit exceeded")

	// ErrJITLiquidityDetected is returned when JIT liquidity is detected
	ErrJITLiquidityDetected = sdkerrors.Register(ModuleName, 20, "JIT liquidity detected")

	// ErrInvalidState is returned when state is invalid
	ErrInvalidState = sdkerrors.Register(ModuleName, 21, "invalid state")

	// ErrInsufficientShares is returned when user has insufficient LP shares
	ErrInsufficientShares = sdkerrors.Register(ModuleName, 16, "insufficient shares")

	// ErrSlippageTooHigh is returned when slippage exceeds tolerance
	ErrSlippageTooHigh = sdkerrors.Register(ModuleName, 17, "slippage too high")

	// ErrInvalidTokenPair is returned when token pair is invalid
	ErrInvalidTokenPair = sdkerrors.Register(ModuleName, 18, "invalid token pair")

	// ErrMaxPoolsReached is returned when maximum number of pools is reached
	ErrMaxPoolsReached = sdkerrors.Register(ModuleName, 19, "maximum number of pools reached")

	// ErrUnauthorized is returned when caller is not authorized
	ErrUnauthorized = sdkerrors.Register(ModuleName, 20, "unauthorized")

	// ErrDeadlineExceeded is returned when transaction deadline has passed
	ErrDeadlineExceeded = sdkerrors.Register(ModuleName, 21, "transaction deadline exceeded")

	// ErrInvalidSwapAmount is returned when swap amount is invalid
	ErrInvalidSwapAmount = sdkerrors.Register(ModuleName, 22, "invalid swap amount")

	// ErrInvalidLiquidityAmount is returned when liquidity amount is invalid
	ErrInvalidLiquidityAmount = sdkerrors.Register(ModuleName, 23, "invalid liquidity amount")

	// ErrStateCorruption is returned when state corruption is detected
	ErrStateCorruption = sdkerrors.Register(ModuleName, 24, "state corruption detected")
)

// ErrorWithRecovery wraps an error with recovery suggestions
type ErrorWithRecovery struct {
	Err      error
	Recovery string
}

func (e *ErrorWithRecovery) Error() string {
	return e.Err.Error()
}

func (e *ErrorWithRecovery) Unwrap() error {
	return e.Err
}

// RecoverySuggestions provides actionable recovery steps for each error type
var RecoverySuggestions = map[error]string{
	ErrInvalidPoolState: "Pool reserves are corrupted or invalid. Query pool state using REST/gRPC. Consider creating a backup checkpoint. Contact validators to investigate state corruption.",
	ErrInsufficientLiquidity: "Pool has insufficient liquidity for this swap. Reduce swap amount. Check pool reserves using query. Try alternative pools or add liquidity first.",
	ErrInvalidInput: "Check input parameters: amounts must be positive, addresses must be valid bech32. Verify token denoms exist. Review transaction parameters.",
	ErrReentrancy: "CRITICAL: Reentrancy attack detected and blocked. Transaction rolled back. Report to security team. Do not retry. Review transaction origin.",
	ErrInvariantViolation: "Pool invariant (x * y = k) violated. This indicates a serious bug. Transaction rolled back. Pool may need emergency recovery. Contact validators immediately.",

	ErrCircuitBreakerTriggered: "Circuit breaker active due to anomaly (large price swing, high volume, suspicious activity). Wait for automatic recovery (typically 1 hour) or check status page. No action needed.",
	ErrSwapTooLarge: "Swap exceeds maximum size limit. Split into multiple smaller swaps. Check max swap size in params. Use price impact calculator to find optimal size.",
	ErrPriceImpactTooHigh: "Swap would cause excessive price impact (>5%). Reduce swap amount. Split across multiple pools. Wait for liquidity to increase. Use limit orders if available.",
	ErrFlashLoanDetected: "SECURITY: Flash loan attack pattern detected. Multiple operations in same block. Transaction blocked. This is expected behavior for security. Do not retry.",

	ErrOverflow: "CRITICAL: Arithmetic overflow detected. This indicates amounts too large for safe math. Contact developers. This should never happen in production.",
	ErrUnderflow: "CRITICAL: Arithmetic underflow detected. This indicates negative result from unsigned math. Contact developers. This should never happen in production.",
	ErrDivisionByZero: "CRITICAL: Division by zero attempted. Pool reserves may be zero. Do not add liquidity to empty pools. Contact developers if this occurs.",

	ErrPoolNotFound: "Pool does not exist for this token pair. Query available pools using REST/gRPC. Create pool using MsgCreatePool if you're the first liquidity provider.",
	ErrPoolAlreadyExists: "Pool already exists for this token pair (order-independent). Query existing pool ID. Use MsgAddLiquidity to add to existing pool instead.",
	ErrInsufficientShares: "Your LP share balance is insufficient for this withdrawal. Query your share balance. Reduce withdrawal amount. You may have already withdrawn.",

	ErrSlippageTooHigh: "Price slipped beyond your tolerance. Increase slippage tolerance if you accept higher slippage. Wait for price to stabilize. Use smaller swap amounts.",
	ErrInvalidTokenPair: "Invalid token pair: tokens must be different, denoms must be valid, tokens must be sorted lexicographically. Check token denoms and order.",
	ErrMaxPoolsReached: "Maximum pool limit reached (check params). Wait for pool cleanup or parameter update. This is rare and indicates system-wide limit.",

	ErrUnauthorized: "Operation not permitted. Verify you're the transaction signer. Check if you own the LP shares. Review message permissions.",
	ErrDeadlineExceeded: "Transaction deadline passed before execution. Increase deadline in transaction. Account for network latency. Try during low congestion periods.",

	ErrInvalidSwapAmount: "Swap amount must be positive and within min/max limits. Check params for limits. Verify token amount is greater than minimum swap threshold.",
	ErrInvalidLiquidityAmount: "Liquidity amounts must be positive and balanced. For initial liquidity, both amounts must be provided. For additional liquidity, amounts must maintain pool ratio.",

	ErrStateCorruption: "CRITICAL: State corruption detected. Automatic recovery initiated. Pool may be temporarily disabled. Backup checkpoint will be restored. Contact validators.",
}

// WrapWithRecovery wraps an error with recovery suggestion
func WrapWithRecovery(err error, msg string, args ...interface{}) error {
	wrapped := sdkerrors.Wrapf(err, msg, args...)

	if suggestion, ok := RecoverySuggestions[err]; ok {
		return &ErrorWithRecovery{
			Err:      wrapped,
			Recovery: suggestion,
		}
	}

	return wrapped
}

// GetRecoverySuggestion returns the recovery suggestion for an error
func GetRecoverySuggestion(err error) string {
	// Unwrap to find the root error
	rootErr := err
	for {
		if unwrapped := errors.Unwrap(rootErr); unwrapped != nil {
			rootErr = unwrapped
		} else {
			break
		}
	}

	if suggestion, ok := RecoverySuggestions[rootErr]; ok {
		return suggestion
	}

	return "No recovery suggestion available. Check error message for details."
}
