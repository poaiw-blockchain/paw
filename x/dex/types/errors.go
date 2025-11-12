package types

import (
	"cosmossdk.io/errors"
)

// DEX module sentinel errors
var (
	ErrInvalidPoolId         = errors.Register(ModuleName, 1, "invalid pool id")
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
)
