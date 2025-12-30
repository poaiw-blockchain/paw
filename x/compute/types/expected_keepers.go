package types

import (
	"context"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	sharedkeeper "github.com/paw-chain/paw/x/shared/keeper"
)

// AccountKeeper defines the expected account keeper used for simulations (and module)
type AccountKeeper interface {
	GetAccount(ctx context.Context, addr sdk.AccAddress) sdk.AccountI
}

// BankKeeper defines the expected bank keeper used for simulations (and module)
type BankKeeper interface {
	GetBalance(ctx context.Context, addr sdk.AccAddress, denom string) sdk.Coin
	SpendableCoins(ctx context.Context, addr sdk.AccAddress) sdk.Coins
	SendCoins(ctx context.Context, fromAddr sdk.AccAddress, toAddr sdk.AccAddress, amt sdk.Coins) error
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
	MintCoins(ctx context.Context, moduleName string, amt sdk.Coins) error
	BurnCoins(ctx context.Context, moduleName string, amt sdk.Coins) error
}

// =============================================================================
// Cross-Module Keeper Interfaces (ARCH-7: Versioned APIs)
// =============================================================================

// OracleKeeper defines the oracle keeper interface used by Compute for price lookups.
// ARCH-7: This is an alias to the versioned interface for backwards compatibility.
type OracleKeeper = sharedkeeper.OracleKeeperV1

// DexKeeper defines the DEX keeper interface used by Compute for payment routing.
// Used when compute payments need to go through DEX swaps.
type DexKeeper = sharedkeeper.DexKeeperV1

// =============================================================================
// Local Compute Interfaces for External Consumers
// =============================================================================

// ComputeKeeperV1 is the versioned interface for external modules to use.
// Export our keeper interface for other modules.
type ComputeKeeperV1 interface {
	// GetProvider returns provider information by address.
	GetProvider(ctx context.Context, addr sdk.AccAddress) (Provider, error)

	// GetRequest returns request information by ID.
	GetRequest(ctx context.Context, requestID uint64) (Request, error)

	// GetActiveProviderCount returns the number of active providers.
	GetActiveProviderCount(ctx context.Context) uint64

	// IsCircuitBreakerOpen checks if the compute circuit breaker is active.
	IsCircuitBreakerOpen(ctx context.Context) bool

	// GetCircuitBreakerState returns full circuit breaker status.
	GetCircuitBreakerState(ctx context.Context) (open bool, reason string, actor string)

	// SimulateRequest estimates cost and gas for a compute request.
	SimulateRequest(ctx context.Context, specs *ComputeSpec) (estimatedGas uint64, estimatedCost sdkmath.Int, err error)
}
