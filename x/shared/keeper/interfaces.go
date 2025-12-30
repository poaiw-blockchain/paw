// Package keeper provides shared keeper interfaces for cross-module communication.
// ARCH-7: Versioned interfaces allow stable API contracts between modules.
package keeper

import (
	"context"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
)

// =============================================================================
// Oracle Keeper Interfaces (Versioned)
// =============================================================================

// OracleKeeperV1 defines the minimal oracle keeper interface for cross-module use.
// Version 1.0 - Initial release for testnet
// Modules should depend on this interface rather than the concrete keeper.
type OracleKeeperV1 interface {
	// GetPrice returns the aggregated price for an asset.
	// Returns the price and whether it exists.
	GetPrice(ctx context.Context, asset string) (sdkmath.LegacyDec, bool)

	// IsCircuitBreakerOpen checks if the oracle circuit breaker is active.
	IsCircuitBreakerOpen(ctx context.Context) bool
}

// OracleKeeperV1Extended extends V1 with additional query methods.
// Use this when you need more oracle functionality.
type OracleKeeperV1Extended interface {
	OracleKeeperV1

	// GetPriceWithTimestamp returns price with its block height.
	GetPriceWithTimestamp(ctx context.Context, asset string) (price sdkmath.LegacyDec, blockHeight int64, found bool)

	// GetAllPrices returns all aggregated prices.
	GetAllPrices(ctx context.Context) []PriceInfo
}

// PriceInfo holds price data returned by oracle queries.
type PriceInfo struct {
	Asset       string
	Price       sdkmath.LegacyDec
	BlockHeight int64
}

// =============================================================================
// DEX Keeper Interfaces (Versioned)
// =============================================================================

// DexKeeperV1 defines the minimal DEX keeper interface for cross-module use.
// Version 1.0 - Initial release for testnet
type DexKeeperV1 interface {
	// GetPool returns pool information by ID.
	GetPool(ctx context.Context, poolID uint64) (PoolInfo, bool)

	// IsCircuitBreakerOpen checks if the DEX circuit breaker is active.
	IsCircuitBreakerOpen(ctx context.Context) bool
}

// DexKeeperV1Extended extends V1 with swap simulation.
// Use this when you need to simulate swaps without executing.
type DexKeeperV1Extended interface {
	DexKeeperV1

	// SimulateSwap calculates expected output without executing.
	SimulateSwap(ctx context.Context, poolID uint64, tokenIn string, amountIn sdkmath.Int) (amountOut sdkmath.Int, err error)

	// GetPoolByDenoms finds a pool by token pair.
	GetPoolByDenoms(ctx context.Context, denomA, denomB string) (PoolInfo, bool)
}

// PoolInfo holds pool data returned by DEX queries.
type PoolInfo struct {
	PoolID     uint64
	TokenA     string
	TokenB     string
	ReserveA   sdkmath.Int
	ReserveB   sdkmath.Int
	TotalShares sdkmath.Int
}

// =============================================================================
// Compute Keeper Interfaces (Versioned)
// =============================================================================

// ComputeKeeperV1 defines the minimal compute keeper interface for cross-module use.
// Version 1.0 - Initial release for testnet
type ComputeKeeperV1 interface {
	// GetProvider returns provider information by address.
	GetProvider(ctx context.Context, addr sdk.AccAddress) (ProviderInfo, bool)

	// IsCircuitBreakerOpen checks if the compute circuit breaker is active.
	IsCircuitBreakerOpen(ctx context.Context) bool
}

// ComputeKeeperV1Extended extends V1 with request queries.
type ComputeKeeperV1Extended interface {
	ComputeKeeperV1

	// GetRequest returns request information by ID.
	GetRequest(ctx context.Context, requestID uint64) (RequestInfo, bool)

	// GetActiveProviderCount returns the number of active providers.
	GetActiveProviderCount(ctx context.Context) uint64
}

// ProviderInfo holds provider data returned by compute queries.
type ProviderInfo struct {
	Address    sdk.AccAddress
	Stake      sdkmath.Int
	Reputation uint64
	IsActive   bool
}

// RequestInfo holds request data returned by compute queries.
type RequestInfo struct {
	RequestID  uint64
	Requester  sdk.AccAddress
	Provider   sdk.AccAddress
	Status     string
	MaxPayment sdkmath.Int
}

// =============================================================================
// Version Constants
// =============================================================================

const (
	// OracleKeeperVersion is the current oracle keeper interface version.
	OracleKeeperVersion = "v1.0.0"

	// DexKeeperVersion is the current DEX keeper interface version.
	DexKeeperVersion = "v1.0.0"

	// ComputeKeeperVersion is the current compute keeper interface version.
	ComputeKeeperVersion = "v1.0.0"
)

// =============================================================================
// Interface Compatibility Notes
// =============================================================================

/*
API Versioning Guidelines:

1. MINOR VERSION BUMP (v1.0 -> v1.1):
   - Add new methods to Extended interfaces
   - Never remove or change existing method signatures
   - Existing code continues to work

2. MAJOR VERSION BUMP (v1 -> v2):
   - Create new interface (e.g., OracleKeeperV2)
   - May change method signatures
   - Old interfaces remain for backwards compatibility
   - Deprecate old versions with timeline

3. DEPRECATION:
   - Add "Deprecated: use XxxV2 instead" comment
   - Keep deprecated interfaces for at least 2 minor releases
   - Remove in next major version

4. EMBEDDING:
   - V2 can embed V1 to inherit methods
   - Example: type OracleKeeperV2 interface { OracleKeeperV1; NewMethod() }

5. ADAPTER PATTERN:
   - If keeper doesn't match interface exactly, create an adapter
   - Adapters live in the module using the interface
*/
