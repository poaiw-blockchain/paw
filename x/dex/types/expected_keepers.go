package types

import (
	"context"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	sharedkeeper "github.com/paw-chain/paw/x/shared/keeper"
)

// AccountKeeper defines the expected account keeper used for simulations.
type AccountKeeper interface {
	GetAccount(ctx context.Context, addr sdk.AccAddress) sdk.AccountI
}

// BankKeeper defines the expected bank keeper used for simulations.
type BankKeeper interface {
	SpendableCoins(ctx context.Context, addr sdk.AccAddress) sdk.Coins
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
}

// =============================================================================
// Cross-Module Keeper Interfaces (ARCH-7: Versioned APIs)
// =============================================================================

// OracleKeeper defines the oracle keeper interface used by DEX for price queries.
// ARCH-7: This is an alias to the versioned interface for backwards compatibility.
// New code should use sharedkeeper.OracleKeeperV1 directly.
type OracleKeeper = sharedkeeper.OracleKeeperV1

// OracleKeeperExtended provides additional oracle functionality.
// Use this when you need price timestamps or bulk queries.
type OracleKeeperExtended = sharedkeeper.OracleKeeperV1Extended

// ComputeKeeper defines the compute keeper interface for DEX integration.
// Used for validating compute-based payment routing.
type ComputeKeeper = sharedkeeper.ComputeKeeperV1

// =============================================================================
// Local DEX Interfaces for External Consumers
// =============================================================================

// DexKeeperV1 is the versioned interface for external modules to use.
// Export our keeper interface for other modules.
type DexKeeperV1 interface {
	// GetPool returns pool information by ID.
	GetPool(ctx context.Context, poolID uint64) (Pool, bool)

	// GetPoolByDenoms finds a pool by token pair.
	GetPoolByDenoms(ctx context.Context, denomA, denomB string) (Pool, bool)

	// SimulateSwap calculates expected output without executing.
	SimulateSwap(ctx context.Context, poolID uint64, tokenIn string, amountIn sdkmath.Int) (amountOut sdkmath.Int, err error)

	// IsCircuitBreakerOpen checks if the DEX circuit breaker is active.
	IsCircuitBreakerOpen(ctx context.Context) bool

	// GetCircuitBreakerState returns full circuit breaker status.
	GetCircuitBreakerState(ctx context.Context) (open bool, reason string, actor string)
}
