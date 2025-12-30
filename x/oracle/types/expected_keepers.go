package types

import (
	"context"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingtypes "github.com/cosmos/cosmos-sdk/x/staking/types"

	sharedkeeper "github.com/paw-chain/paw/x/shared/keeper"
)

// AccountKeeper defines the minimal functions needed from the account keeper for simulations.
type AccountKeeper interface {
	GetAccount(ctx context.Context, addr sdk.AccAddress) sdk.AccountI
}

// BankKeeper defines the expected bank keeper methods for simulations.
type BankKeeper interface {
	SpendableCoins(ctx context.Context, addr sdk.AccAddress) sdk.Coins
	SendCoinsFromAccountToModule(ctx context.Context, senderAddr sdk.AccAddress, recipientModule string, amt sdk.Coins) error
	SendCoinsFromModuleToAccount(ctx context.Context, senderModule string, recipientAddr sdk.AccAddress, amt sdk.Coins) error
}

// StakingKeeper defines the staking keeper methods needed for simulations.
type StakingKeeper interface {
	GetAllValidators(ctx context.Context) ([]stakingtypes.Validator, error)
}

// =============================================================================
// Cross-Module Keeper Interfaces (ARCH-7: Versioned APIs)
// =============================================================================

// DexKeeper defines the DEX keeper interface used by Oracle for TWAP validation.
// Used when oracle needs to cross-reference DEX prices for manipulation detection.
type DexKeeper = sharedkeeper.DexKeeperV1

// ComputeKeeper defines the compute keeper interface for Oracle integration.
// Used for compute-based oracle data aggregation.
type ComputeKeeper = sharedkeeper.ComputeKeeperV1

// =============================================================================
// Local Oracle Interfaces for External Consumers
// =============================================================================

// OracleKeeperV1 is the versioned interface for external modules to use.
// Export our keeper interface for other modules.
type OracleKeeperV1 interface {
	// GetPrice returns the aggregated price for an asset.
	GetPrice(ctx context.Context, asset string) (Price, error)

	// GetPriceWithTimestamp returns price with its block height.
	GetPriceWithTimestamp(ctx context.Context, asset string) (price sdkmath.LegacyDec, blockHeight int64, found bool)

	// GetAllPrices returns all aggregated prices.
	GetAllPrices(ctx context.Context) []Price

	// IsCircuitBreakerOpen checks if the oracle circuit breaker is active.
	IsCircuitBreakerOpen(ctx context.Context) bool

	// GetCircuitBreakerState returns full circuit breaker status.
	GetCircuitBreakerState(ctx context.Context) (open bool, reason string, actor string)

	// CalculateTWAP calculates time-weighted average price over a period.
	CalculateTWAP(ctx context.Context, asset string, periodBlocks int64) (sdkmath.LegacyDec, error)
}
