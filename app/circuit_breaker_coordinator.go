// Package app provides the PAW blockchain application implementation.
package app

import (
	"context"
	"fmt"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	computekeeper "github.com/paw-chain/paw/x/compute/keeper"
	computetypes "github.com/paw-chain/paw/x/compute/types"
	dexkeeper "github.com/paw-chain/paw/x/dex/keeper"
	dextypes "github.com/paw-chain/paw/x/dex/types"
	oraclekeeper "github.com/paw-chain/paw/x/oracle/keeper"
	oracletypes "github.com/paw-chain/paw/x/oracle/types"
)

// Circuit breaker coordination event types
const (
	EventTypeCircuitBreakerPropagated = "circuit_breaker_propagated"
	AttributeKeySourceModule          = "source_module"
	AttributeKeyTargetModule          = "target_module"
	AttributeKeyPropagationReason     = "propagation_reason"
)

// Module names for circuit breaker coordination
const (
	ModuleOracle  = "oracle"
	ModuleDEX     = "dex"
	ModuleCompute = "compute"
)

// CircuitBreakerStatus represents the unified status of all circuit breakers.
type CircuitBreakerStatus struct {
	OracleOpen    bool   `json:"oracle_open"`
	OracleReason  string `json:"oracle_reason,omitempty"`
	OracleActor   string `json:"oracle_actor,omitempty"`
	DEXOpen       bool   `json:"dex_open"`
	DEXReason     string `json:"dex_reason,omitempty"`
	DEXActor      string `json:"dex_actor,omitempty"`
	ComputeOpen   bool   `json:"compute_open"`
	ComputeReason string `json:"compute_reason,omitempty"`
	ComputeActor  string `json:"compute_actor,omitempty"`
	// AnyOpen is true if any circuit breaker is open
	AnyOpen bool `json:"any_open"`
}

// CircuitBreakerCoordinator coordinates circuit breaker state propagation across modules.
// ARCH-6: When one module triggers its circuit breaker, dependent modules are notified
// so they can take appropriate action (e.g., DEX pausing swaps when Oracle is down).
type CircuitBreakerCoordinator struct {
	oracleKeeper  *oraclekeeper.Keeper
	dexKeeper     *dexkeeper.Keeper
	computeKeeper *computekeeper.Keeper
}

// NewCircuitBreakerCoordinator creates a new circuit breaker coordinator.
func NewCircuitBreakerCoordinator(
	oracleKeeper *oraclekeeper.Keeper,
	dexKeeper *dexkeeper.Keeper,
	computeKeeper *computekeeper.Keeper,
) *CircuitBreakerCoordinator {
	return &CircuitBreakerCoordinator{
		oracleKeeper:  oracleKeeper,
		dexKeeper:     dexKeeper,
		computeKeeper: computeKeeper,
	}
}

// GetGlobalStatus returns the unified circuit breaker status across all modules.
func (c *CircuitBreakerCoordinator) GetGlobalStatus(ctx context.Context) CircuitBreakerStatus {
	status := CircuitBreakerStatus{}

	// Get Oracle status
	if c.oracleKeeper != nil {
		status.OracleOpen, status.OracleReason, status.OracleActor = c.oracleKeeper.GetCircuitBreakerState(ctx)
	}

	// Get DEX status
	if c.dexKeeper != nil {
		status.DEXOpen, status.DEXReason, status.DEXActor = c.dexKeeper.GetCircuitBreakerState(ctx)
	}

	// Get Compute status
	if c.computeKeeper != nil {
		status.ComputeOpen, status.ComputeReason, status.ComputeActor = c.computeKeeper.GetCircuitBreakerState(ctx)
	}

	status.AnyOpen = status.OracleOpen || status.DEXOpen || status.ComputeOpen
	return status
}

// emitPropagationEvent emits an event when a circuit breaker state is propagated.
func (c *CircuitBreakerCoordinator) emitPropagationEvent(ctx context.Context, source, target, reason string) {
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			EventTypeCircuitBreakerPropagated,
			sdk.NewAttribute(AttributeKeySourceModule, source),
			sdk.NewAttribute(AttributeKeyTargetModule, target),
			sdk.NewAttribute(AttributeKeyPropagationReason, reason),
		),
	)
}

// OracleHooksAdapter implements oracletypes.OracleHooks to receive Oracle circuit breaker events.
type OracleHooksAdapter struct {
	coordinator *CircuitBreakerCoordinator
}

// NewOracleHooksAdapter creates a new Oracle hooks adapter for the coordinator.
func NewOracleHooksAdapter(coordinator *CircuitBreakerCoordinator) *OracleHooksAdapter {
	return &OracleHooksAdapter{coordinator: coordinator}
}

// AfterPriceAggregated is called when a new price has been aggregated.
// No-op for circuit breaker coordination.
func (h *OracleHooksAdapter) AfterPriceAggregated(_ context.Context, _ string, _ sdkmath.LegacyDec, _ int64) error {
	return nil
}

// AfterPriceSubmitted is called when a validator submits a price.
// No-op for circuit breaker coordination.
func (h *OracleHooksAdapter) AfterPriceSubmitted(_ context.Context, _ string, _ string, _ sdkmath.LegacyDec) error {
	return nil
}

// Ensure OracleHooksAdapter implements oracletypes.OracleHooks
var _ oracletypes.OracleHooks = (*OracleHooksAdapter)(nil)

// OnCircuitBreakerTriggered is called when the Oracle circuit breaker activates.
// It notifies the DEX module since DEX swaps may rely on Oracle prices.
func (h *OracleHooksAdapter) OnCircuitBreakerTriggered(ctx context.Context, reason string) error {
	if h.coordinator == nil || h.coordinator.dexKeeper == nil {
		return nil
	}

	// Emit propagation event for observability
	h.coordinator.emitPropagationEvent(ctx, ModuleOracle, ModuleDEX, reason)

	// Note: We don't automatically open DEX circuit breaker here.
	// The DEX module can check Oracle status before allowing price-sensitive operations.
	// This is a notification pattern rather than automatic cascade.

	return nil
}

// DexHooksAdapter implements dextypes.DexHooks to receive DEX circuit breaker events.
type DexHooksAdapter struct {
	coordinator *CircuitBreakerCoordinator
}

// NewDexHooksAdapter creates a new DEX hooks adapter for the coordinator.
func NewDexHooksAdapter(coordinator *CircuitBreakerCoordinator) *DexHooksAdapter {
	return &DexHooksAdapter{coordinator: coordinator}
}

// AfterSwap is called after a successful swap operation.
// No-op for circuit breaker coordination.
func (h *DexHooksAdapter) AfterSwap(_ context.Context, _ uint64, _ string, _, _ string, _, _ sdkmath.Int) error {
	return nil
}

// AfterPoolCreated is called after a new liquidity pool is created.
// No-op for circuit breaker coordination.
func (h *DexHooksAdapter) AfterPoolCreated(_ context.Context, _ uint64, _, _, _ string) error {
	return nil
}

// AfterLiquidityChanged is called when liquidity is added or removed.
// No-op for circuit breaker coordination.
func (h *DexHooksAdapter) AfterLiquidityChanged(_ context.Context, _ uint64, _ string, _, _ sdkmath.Int, _ bool) error {
	return nil
}

// Ensure DexHooksAdapter implements dextypes.DexHooks
var _ dextypes.DexHooks = (*DexHooksAdapter)(nil)

// OnCircuitBreakerTriggered is called when the DEX circuit breaker activates.
// It notifies the Compute module since compute payments may go through DEX.
func (h *DexHooksAdapter) OnCircuitBreakerTriggered(ctx context.Context, reason string) error {
	if h.coordinator == nil || h.coordinator.computeKeeper == nil {
		return nil
	}

	// Emit propagation event for observability
	h.coordinator.emitPropagationEvent(ctx, ModuleDEX, ModuleCompute, reason)

	return nil
}

// ComputeHooksAdapter implements computetypes.ComputeHooks to receive Compute circuit breaker events.
type ComputeHooksAdapter struct {
	coordinator *CircuitBreakerCoordinator
}

// NewComputeHooksAdapter creates a new Compute hooks adapter for the coordinator.
func NewComputeHooksAdapter(coordinator *CircuitBreakerCoordinator) *ComputeHooksAdapter {
	return &ComputeHooksAdapter{coordinator: coordinator}
}

// AfterJobCompleted is called when a compute job finishes successfully.
// No-op for circuit breaker coordination.
func (h *ComputeHooksAdapter) AfterJobCompleted(_ context.Context, _ uint64, _ sdk.AccAddress, _ []byte) error {
	return nil
}

// AfterJobFailed is called when a compute job fails or times out.
// No-op for circuit breaker coordination.
func (h *ComputeHooksAdapter) AfterJobFailed(_ context.Context, _ uint64, _ string) error {
	return nil
}

// AfterProviderRegistered is called when a new compute provider registers.
// No-op for circuit breaker coordination.
func (h *ComputeHooksAdapter) AfterProviderRegistered(_ context.Context, _ sdk.AccAddress, _ sdkmath.Int) error {
	return nil
}

// AfterProviderSlashed is called when a provider is slashed for misbehavior.
// No-op for circuit breaker coordination.
func (h *ComputeHooksAdapter) AfterProviderSlashed(_ context.Context, _ sdk.AccAddress, _ sdkmath.Int, _ string) error {
	return nil
}

// Ensure ComputeHooksAdapter implements computetypes.ComputeHooks
var _ computetypes.ComputeHooks = (*ComputeHooksAdapter)(nil)

// OnCircuitBreakerTriggered is called when the Compute circuit breaker activates.
// Currently no downstream modules depend on Compute, so this is a no-op.
func (h *ComputeHooksAdapter) OnCircuitBreakerTriggered(ctx context.Context, reason string) error {
	// Emit event for observability even though no downstream modules
	if h.coordinator != nil {
		sdkCtx := sdk.UnwrapSDKContext(ctx)
		sdkCtx.EventManager().EmitEvent(
			sdk.NewEvent(
				EventTypeCircuitBreakerPropagated,
				sdk.NewAttribute(AttributeKeySourceModule, ModuleCompute),
				sdk.NewAttribute(AttributeKeyTargetModule, "none"),
				sdk.NewAttribute(AttributeKeyPropagationReason, reason),
			),
		)
	}
	return nil
}

// IsOracleAvailable checks if Oracle prices are available (circuit breaker closed).
// DEX can use this before executing price-sensitive operations.
func (c *CircuitBreakerCoordinator) IsOracleAvailable(ctx context.Context) bool {
	if c.oracleKeeper == nil {
		return true // If no oracle keeper, assume available
	}
	return !c.oracleKeeper.IsCircuitBreakerOpen(ctx)
}

// IsDEXAvailable checks if DEX operations are available (circuit breaker closed).
// Compute can use this before executing payment-related operations.
func (c *CircuitBreakerCoordinator) IsDEXAvailable(ctx context.Context) bool {
	if c.dexKeeper == nil {
		return true // If no DEX keeper, assume available
	}
	return !c.dexKeeper.IsCircuitBreakerOpen(ctx)
}

// IsComputeAvailable checks if Compute operations are available (circuit breaker closed).
func (c *CircuitBreakerCoordinator) IsComputeAvailable(ctx context.Context) bool {
	if c.computeKeeper == nil {
		return true // If no compute keeper, assume available
	}
	return !c.computeKeeper.IsCircuitBreakerOpen(ctx)
}

// CheckDependenciesForDEX verifies that all DEX dependencies are available.
// Returns an error if Oracle circuit breaker is open (DEX may need prices).
func (c *CircuitBreakerCoordinator) CheckDependenciesForDEX(ctx context.Context) error {
	if !c.IsOracleAvailable(ctx) {
		open, reason, _ := c.oracleKeeper.GetCircuitBreakerState(ctx)
		if open {
			return fmt.Errorf("DEX operations may be affected: Oracle circuit breaker open (%s)", reason)
		}
	}
	return nil
}

// CheckDependenciesForCompute verifies that all Compute dependencies are available.
// Returns an error if DEX circuit breaker is open (Compute may need payment routing).
func (c *CircuitBreakerCoordinator) CheckDependenciesForCompute(ctx context.Context) error {
	if !c.IsDEXAvailable(ctx) {
		open, reason, _ := c.dexKeeper.GetCircuitBreakerState(ctx)
		if open {
			return fmt.Errorf("Compute operations may be affected: DEX circuit breaker open (%s)", reason)
		}
	}
	return nil
}

// SetupHooks registers the coordinator's hook adapters with each module's keeper.
// This must be called after all keepers are initialized but before the chain starts.
func (c *CircuitBreakerCoordinator) SetupHooks() {
	if c.oracleKeeper != nil {
		// Create multi-hook to allow other hooks alongside coordinator
		oracleAdapter := NewOracleHooksAdapter(c)
		existingHooks := c.oracleKeeper.GetHooks()
		if existingHooks != nil {
			c.oracleKeeper.SetHooks(oracletypes.NewMultiOracleHooks(existingHooks, oracleAdapter))
		} else {
			c.oracleKeeper.SetHooks(oracleAdapter)
		}
	}

	if c.dexKeeper != nil {
		dexAdapter := NewDexHooksAdapter(c)
		existingHooks := c.dexKeeper.GetHooks()
		if existingHooks != nil {
			c.dexKeeper.SetHooks(dextypes.NewMultiDexHooks(existingHooks, dexAdapter))
		} else {
			c.dexKeeper.SetHooks(dexAdapter)
		}
	}

	if c.computeKeeper != nil {
		computeAdapter := NewComputeHooksAdapter(c)
		existingHooks := c.computeKeeper.GetHooks()
		if existingHooks != nil {
			c.computeKeeper.SetHooks(computetypes.NewMultiComputeHooks(existingHooks, computeAdapter))
		} else {
			c.computeKeeper.SetHooks(computeAdapter)
		}
	}
}
