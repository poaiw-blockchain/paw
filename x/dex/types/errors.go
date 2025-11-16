package types

import (
	"cosmossdk.io/errors"
)

// DEX module sentinel errors
var (
	ErrInvalidPoolId         = errors.Register(ModuleName, 1, "invalid pool id")
	ErrInvalidPoolID         = ErrInvalidPoolId // Alias for consistency
	ErrPoolNotFound          = errors.Register(ModuleName, 2, "pool not found")
	ErrPoolAlreadyExists     = errors.Register(ModuleName, 3, "pool already exists")
	ErrInvalidTokenDenom     = errors.Register(ModuleName, 4, "invalid token denomination")
	ErrInsufficientFunds     = errors.Register(ModuleName, 5, "insufficient funds")
	ErrInsufficientLiquidity = errors.Register(ModuleName, 6, "insufficient liquidity in pool")
	ErrInvalidAmount         = errors.Register(ModuleName, 7, "invalid amount")
	ErrInvalidSlippage       = errors.Register(ModuleName, 8, "slippage exceeded maximum")
	ErrInvalidShares         = errors.Register(ModuleName, 9, "invalid shares amount")
	ErrZeroAmount            = errors.Register(ModuleName, 10, "amount cannot be zero")
	ErrSameToken             = errors.Register(ModuleName, 11, "cannot swap same token")
	ErrMinAmountOut          = errors.Register(ModuleName, 12, "output amount less than minimum required")
	ErrInvalidAddress        = errors.Register(ModuleName, 13, "invalid address")
	ErrInsufficientShares    = errors.Register(ModuleName, 14, "insufficient liquidity shares")
	ErrInvalidParams         = errors.Register(ModuleName, 15, "invalid parameters")
	ErrInvalidGenesis        = errors.Register(ModuleName, 16, "invalid genesis state")
	ErrModulePaused          = errors.Register(ModuleName, 17, "module is paused")
	ErrCircuitBreakerTripped = errors.Register(ModuleName, 18, "circuit breaker tripped")

	// MEV Protection errors
	ErrSandwichAttackDetected       = errors.Register(ModuleName, 19, "sandwich attack detected")
	ErrFrontRunningDetected         = errors.Register(ModuleName, 20, "front-running detected")
	ErrPriceImpactExceeded          = errors.Register(ModuleName, 21, "price impact exceeds maximum allowed")
	ErrInvalidTransactionTimestamp  = errors.Register(ModuleName, 22, "invalid transaction timestamp")
	ErrTransactionOrderingViolation = errors.Register(ModuleName, 23, "transaction ordering violation")
	ErrInvalidMEVConfig             = errors.Register(ModuleName, 24, "invalid MEV protection configuration")
	ErrMEVAttackBlocked             = errors.Register(ModuleName, 25, "MEV attack blocked")
)
