package keeper

import (
	"context"
	"encoding/binary"
	"fmt"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/compute/types"
)

// RegisterProvider registers a new compute provider with the required stake
func (k Keeper) RegisterProvider(ctx context.Context, provider sdk.AccAddress, moniker, endpoint string, specs types.ComputeSpec, pricing types.Pricing, stake math.Int) error {
	// Validate parameters
	params, err := k.GetParams(ctx)
	if err != nil {
		return types.ErrOperationFailed.Wrapf("failed to get params: %v", err)
	}

	if stake.LT(params.MinProviderStake) {
		return types.ErrInsufficientStake.Wrapf("stake %s is less than minimum required %s", stake.String(), params.MinProviderStake.String())
	}

	// Check if provider already exists
	existing, err := k.GetProvider(ctx, provider)
	if err == nil && existing != nil {
		return types.ErrProviderAlreadyRegistered.Wrapf("provider %s", provider.String())
	}

	// SEC-20: Check maximum providers limit to prevent state bloat attacks
	currentCount := k.GetTotalProviderCount(ctx)
	if currentCount >= MaxProviders {
		return types.ErrMaxProvidersReached.Wrapf("limit is %d providers", MaxProviders)
	}

	// Validate specs
	specs, err = k.validateComputeSpec(specs, params, true)
	if err != nil {
		return fmt.Errorf("invalid compute specs: %w", err)
	}

	// Validate pricing
	if err := k.validatePricing(pricing); err != nil {
		return fmt.Errorf("invalid pricing: %w", err)
	}

	// Transfer stake from provider to module account
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	coins := sdk.NewCoins(sdk.NewCoin("upaw", stake))
	if err := k.bankKeeper.SendCoinsFromAccountToModule(sdkCtx, provider, types.ModuleName, coins); err != nil {
		return types.ErrInsufficientBalance.Wrapf("failed to escrow stake: %v", err)
	}

	// Create provider record
	now := sdkCtx.BlockTime()
	initialRep := k.initialReputation(stake, params.MinProviderStake, specs, params.MinReputationScore)

	providerRecord := types.Provider{
		Address:                provider.String(),
		Moniker:                moniker,
		Endpoint:               endpoint,
		AvailableSpecs:         specs,
		Pricing:                pricing,
		Stake:                  stake,
		Reputation:             initialRep,
		TotalRequestsCompleted: 0,
		TotalRequestsFailed:    0,
		Active:                 true,
		RegisteredAt:           now,
		LastActiveAt:           now,
	}

	// Store provider
	if err := k.SetProvider(ctx, providerRecord); err != nil {
		return types.ErrStorageFailed.Wrapf("failed to store provider: %v", err)
	}

	// Add to active providers index
	if err := k.setActiveProviderIndex(ctx, provider, true); err != nil {
		return types.ErrStorageFailed.Wrapf("failed to set active provider index: %v", err)
	}

	// Add to reputation index
	if err := k.setReputationIndex(ctx, provider, initialRep); err != nil {
		return types.ErrStorageFailed.Wrapf("failed to set reputation index: %v", err)
	}

	// SEC-20: Increment total provider count
	k.incrementTotalProviderCount(ctx)

	// Invalidate provider cache when new provider is registered
	if err := k.InvalidateProviderCache(ctx); err != nil {
		// Log error but don't fail registration
		sdkCtx.Logger().Error("failed to invalidate provider cache on registration", "error", err)
	}

	// Emit event
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"provider_registered",
			sdk.NewAttribute("provider", provider.String()),
			sdk.NewAttribute("moniker", moniker),
			sdk.NewAttribute("stake", stake.String()),
			sdk.NewAttribute("endpoint", endpoint),
		),
	)

	return nil
}

// UpdateProvider updates an existing provider's information
func (k Keeper) UpdateProvider(ctx context.Context, provider sdk.AccAddress, moniker, endpoint string, specs *types.ComputeSpec, pricing *types.Pricing) error {
	// Get existing provider
	existing, err := k.GetProvider(ctx, provider)
	if err != nil {
		return types.ErrProviderNotFound.Wrap(err.Error())
	}

	params, err := k.GetParams(ctx)
	if err != nil {
		return types.ErrOperationFailed.Wrapf("failed to get params: %v", err)
	}

	// Update fields if provided
	if moniker != "" {
		existing.Moniker = moniker
	}
	if endpoint != "" {
		existing.Endpoint = endpoint
	}
	if specs != nil {
		updatedSpecs, err := k.validateComputeSpec(*specs, params, true)
		if err != nil {
			return fmt.Errorf("invalid compute specs: %w", err)
		}
		existing.AvailableSpecs = updatedSpecs
	}
	if pricing != nil {
		if err := k.validatePricing(*pricing); err != nil {
			return fmt.Errorf("invalid pricing: %w", err)
		}
		existing.Pricing = *pricing
	}

	// Update last active timestamp
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	existing.LastActiveAt = sdkCtx.BlockTime()

	// Store updated provider
	if err := k.SetProvider(ctx, *existing); err != nil {
		return types.ErrStorageFailed.Wrapf("failed to update provider: %v", err)
	}

	// Emit event
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"provider_updated",
			sdk.NewAttribute("provider", provider.String()),
		),
	)

	return nil
}

// DeactivateProvider deactivates a provider and returns their stake
func (k Keeper) DeactivateProvider(ctx context.Context, provider sdk.AccAddress) error {
	// Get existing provider
	existing, err := k.GetProvider(ctx, provider)
	if err != nil {
		return types.ErrProviderNotFound.Wrap(err.Error())
	}

	if !existing.Active {
		return types.ErrProviderAlreadyInactive
	}

	// Mark as inactive
	existing.Active = false

	// Store updated provider
	if err := k.SetProvider(ctx, *existing); err != nil {
		return types.ErrStorageFailed.Wrapf("failed to deactivate provider: %v", err)
	}

	// Remove from active providers index
	if err := k.setActiveProviderIndex(ctx, provider, false); err != nil {
		return types.ErrStorageFailed.Wrapf("failed to update active provider index: %v", err)
	}

	// Remove from reputation index
	if err := k.deleteReputationIndex(ctx, provider, existing.Reputation); err != nil {
		return types.ErrStorageFailed.Wrapf("failed to delete reputation index: %v", err)
	}

	// Invalidate provider cache when provider is deactivated
	if err := k.InvalidateProviderCache(ctx); err != nil {
		// Log error but don't fail deactivation
		sdkCtx := sdk.UnwrapSDKContext(ctx)
		sdkCtx.Logger().Error("failed to invalidate provider cache on deactivation", "error", err)
	}

	// Return stake to provider
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	coins := sdk.NewCoins(sdk.NewCoin("upaw", existing.Stake))
	if err := k.bankKeeper.SendCoinsFromModuleToAccount(sdkCtx, types.ModuleName, provider, coins); err != nil {
		return types.ErrOperationFailed.Wrapf("failed to return stake: %v", err)
	}

	// Emit event
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			"provider_deactivated",
			sdk.NewAttribute("provider", provider.String()),
			sdk.NewAttribute("stake_returned", existing.Stake.String()),
		),
	)

	return nil
}

// GetProvider retrieves a provider by address
func (k Keeper) GetProvider(ctx context.Context, address sdk.AccAddress) (*types.Provider, error) {
	store := k.getStore(ctx)
	bz := store.Get(ProviderKey(address))

	if bz == nil {
		return nil, types.ErrProviderNotFound
	}

	var provider types.Provider
	if err := k.cdc.Unmarshal(bz, &provider); err != nil {
		return nil, types.ErrUnmarshalFailed.Wrapf("failed to unmarshal provider: %v", err)
	}

	return &provider, nil
}

// initialReputation derives a sensible starting reputation based on stake commitments.
func (k Keeper) initialReputation(stake, minStake math.Int, specs types.ComputeSpec, minReputation uint32) uint32 {
	base := minReputation
	if base < 40 {
		base = 40
	}
	stakeBonus := math.ZeroInt()
	if stake.GT(minStake) {
		extra := stake.Sub(minStake)
		stakeBonus = extra.MulRaw(30).Quo(minStake)
		if stakeBonus.GT(math.NewInt(30)) {
			stakeBonus = math.NewInt(30)
		}
	}

	qualityBonus := computeSpecQuality(specs)
	total := base + types.SaturateInt64ToUint32(stakeBonus.Int64()) + qualityBonus
	if total > 100 {
		return 100
	}

	return total
}

func computeSpecQuality(specs types.ComputeSpec) uint32 {
	score := specs.CpuCores/1000 + specs.MemoryMb/2048 + specs.StorageGb/200
	if specs.GpuCount > 0 {
		score += uint64(specs.GpuCount) * 5
	}

	if score > 30 {
		score = 30
	}

	return types.SaturateUint64ToUint32(score)
}

// SetProvider stores a provider record
func (k Keeper) SetProvider(ctx context.Context, provider types.Provider) error {
	store := k.getStore(ctx)
	bz, err := k.cdc.Marshal(&provider)
	if err != nil {
		return types.ErrMarshalFailed.Wrapf("failed to marshal provider: %v", err)
	}

	addr, err := sdk.AccAddressFromBech32(provider.Address)
	if err != nil {
		return types.ErrInvalidAddress.Wrapf("failed to parse address: %v", err)
	}

	store.Set(ProviderKey(addr), bz)
	return nil
}

// IterateProviders iterates over all providers
func (k Keeper) IterateProviders(ctx context.Context, cb func(provider types.Provider) (stop bool, err error)) error {
	store := k.getStore(ctx)
	iterator := storetypes.KVStorePrefixIterator(store, ProviderKeyPrefix)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var provider types.Provider
		if err := k.cdc.Unmarshal(iterator.Value(), &provider); err != nil {
			return types.ErrUnmarshalFailed.Wrapf("failed to unmarshal provider: %v", err)
		}

		stop, err := cb(provider)
		if err != nil {
			return types.ErrOperationFailed.Wrapf("callback error: %v", err)
		}
		if stop {
			break
		}
	}

	return nil
}

// IterateActiveProviders iterates over all active providers
func (k Keeper) IterateActiveProviders(ctx context.Context, cb func(provider types.Provider) (stop bool, err error)) error {
	store := k.getStore(ctx)
	iterator := storetypes.KVStorePrefixIterator(store, ActiveProvidersPrefix)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		// The value is the provider address
		address := sdk.AccAddress(iterator.Value())
		provider, err := k.GetProvider(ctx, address)
		if err != nil {
			continue // Skip if provider not found
		}

		stop, err := cb(*provider)
		if err != nil {
			return types.ErrOperationFailed.Wrapf("callback error: %v", err)
		}
		if stop {
			break
		}
	}

	return nil
}

// UpdateProviderReputation updates a provider's reputation score
func (k Keeper) UpdateProviderReputation(ctx context.Context, provider sdk.AccAddress, success bool) error {
	providerRecord, err := k.GetProvider(ctx, provider)
	if err != nil {
		return err
	}

	params, err := k.GetParams(ctx)
	if err != nil {
		return types.ErrOperationFailed.Wrapf("failed to get params: %v", err)
	}

	oldReputation := providerRecord.Reputation

	if success {
		providerRecord.TotalRequestsCompleted++
		// Gradually improve reputation towards 100
		if providerRecord.Reputation < 100 {
			providerRecord.Reputation++
		}
	} else {
		providerRecord.TotalRequestsFailed++
		// Slash reputation
		slashAmount := uint32(params.ReputationSlashPercentage)
		if providerRecord.Reputation > slashAmount {
			providerRecord.Reputation -= slashAmount
		} else {
			providerRecord.Reputation = 0
		}
	}

	// Update last active timestamp
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	providerRecord.LastActiveAt = sdkCtx.BlockTime()

	// Update reputation index if reputation changed and provider is active
	if oldReputation != providerRecord.Reputation && providerRecord.Active {
		// Remove old index entry
		if err := k.deleteReputationIndex(ctx, provider, oldReputation); err != nil {
			return types.ErrStorageFailed.Wrapf("failed to delete old reputation index: %v", err)
		}
		// Add new index entry
		if err := k.setReputationIndex(ctx, provider, providerRecord.Reputation); err != nil {
			return types.ErrStorageFailed.Wrapf("failed to set new reputation index: %v", err)
		}

		// Invalidate provider cache when reputation changes
		// This ensures the cache is refreshed on next BeginBlocker
		if err := k.InvalidateProviderCache(ctx); err != nil {
			// Log error but don't fail the reputation update
			sdkCtx := sdk.UnwrapSDKContext(ctx)
			sdkCtx.Logger().Error("failed to invalidate provider cache on reputation change", "error", err)
		}
	}

	return k.SetProvider(ctx, *providerRecord)
}

// setActiveProviderIndex sets or removes a provider from the active providers index
func (k Keeper) setActiveProviderIndex(ctx context.Context, provider sdk.AccAddress, active bool) error {
	store := k.getStore(ctx)
	key := ActiveProviderKey(provider)

	if active {
		store.Set(key, provider.Bytes())
	} else {
		store.Delete(key)
	}
	return nil
}

// setReputationIndex adds a provider to the reputation-sorted index
func (k Keeper) setReputationIndex(ctx context.Context, provider sdk.AccAddress, reputation uint32) error {
	store := k.getStore(ctx)
	key := ProviderByReputationKey(reputation, provider)
	store.Set(key, provider.Bytes())
	return nil
}

// deleteReputationIndex removes a provider from the reputation-sorted index
func (k Keeper) deleteReputationIndex(ctx context.Context, provider sdk.AccAddress, reputation uint32) error {
	store := k.getStore(ctx)
	key := ProviderByReputationKey(reputation, provider)
	store.Delete(key)
	return nil
}

// validateComputeSpec validates compute specifications using module parameters for bounds.
func (k Keeper) validateComputeSpec(spec types.ComputeSpec, params types.Params, applyDefaultTimeout bool) (types.ComputeSpec, error) {
	if spec.CpuCores == 0 {
		return spec, types.ErrInvalidResourceSpec.Wrap("cpu_cores must be greater than 0")
	}
	if spec.MemoryMb == 0 {
		return spec, types.ErrInvalidResourceSpec.Wrap("memory_mb must be greater than 0")
	}
	if spec.StorageGb == 0 {
		return spec, types.ErrInvalidResourceSpec.Wrap("storage_gb must be greater than 0")
	}
	if spec.TimeoutSeconds == 0 {
		if applyDefaultTimeout {
			spec.TimeoutSeconds = params.MaxRequestTimeoutSeconds
		} else {
			return spec, types.ErrInvalidResourceSpec.Wrap("timeout_seconds must be greater than 0")
		}
	}

	if spec.TimeoutSeconds > params.MaxRequestTimeoutSeconds {
		return spec, types.ErrInvalidResourceSpec.Wrapf("timeout_seconds %d exceeds maximum %d", spec.TimeoutSeconds, params.MaxRequestTimeoutSeconds)
	}

	return spec, nil
}

// validatePricing validates pricing structure
func (k Keeper) validatePricing(pricing types.Pricing) error {
	if !pricing.CpuPricePerMcoreHour.IsPositive() {
		return types.ErrInvalidParameters.Wrap("cpu_price_per_mcore_hour must be positive")
	}
	if !pricing.MemoryPricePerMbHour.IsPositive() {
		return types.ErrInvalidParameters.Wrap("memory_price_per_mb_hour must be positive")
	}
	if !pricing.GpuPricePerHour.IsPositive() {
		return types.ErrInvalidParameters.Wrap("gpu_price_per_hour must be positive")
	}
	if !pricing.StoragePricePerGbHour.IsPositive() {
		return types.ErrInvalidParameters.Wrap("storage_price_per_gb_hour must be positive")
	}

	return nil
}

// FindSuitableProvider finds a suitable provider for the given specs
// It first checks the cache for fast O(1) lookup, then falls back to full iteration
func (k Keeper) FindSuitableProvider(ctx context.Context, specs types.ComputeSpec, preferredProvider string) (sdk.AccAddress, error) {
	params, err := k.GetParams(ctx)
	if err != nil {
		return nil, types.ErrOperationFailed.Wrapf("failed to get params: %v", err)
	}

	// If preferred provider is specified and valid, try to use it
	if preferredProvider != "" {
		addr, err := sdk.AccAddressFromBech32(preferredProvider)
		if err == nil {
			provider, err := k.GetProvider(ctx, addr)
			if err == nil && provider.Active && provider.Reputation >= params.MinReputationScore {
				if k.canProviderHandleSpecs(*provider, specs) {
					return addr, nil
				}
			}
		}
	}

	// Try cache first if enabled
	if params.UseProviderCache {
		cached, err := k.FindSuitableProviderFromCache(ctx, specs)
		if err == nil && cached != nil {
			// Cache hit - return immediately
			return cached, nil
		}
		// Cache miss or error - fall through to full iteration
	}

	// Fallback: Use reputation index to find best provider (already sorted by reputation descending)
	store := k.getStore(ctx)
	iterator := storetypes.KVStorePrefixIterator(store, ProvidersByReputationPrefix)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		// The value is the provider address
		providerAddr := sdk.AccAddress(iterator.Value())
		provider, err := k.GetProvider(ctx, providerAddr)
		if err != nil {
			continue // Skip if provider not found
		}

		// Check if provider meets minimum reputation
		if provider.Reputation < params.MinReputationScore {
			// Since we're iterating in descending reputation order,
			// all remaining providers will have lower reputation
			break
		}

		// Check if provider is active
		if !provider.Active {
			continue
		}

		// Check if provider can handle the requested specs
		if k.canProviderHandleSpecs(*provider, specs) {
			return providerAddr, nil
		}
	}

	return nil, types.ErrProviderNotFound.Wrap("no suitable provider found for requested specs")
}

// canProviderHandleSpecs checks if a provider can handle the requested specs
func (k Keeper) canProviderHandleSpecs(provider types.Provider, specs types.ComputeSpec) bool {
	if provider.AvailableSpecs.CpuCores < specs.CpuCores {
		return false
	}
	if provider.AvailableSpecs.MemoryMb < specs.MemoryMb {
		return false
	}
	if specs.GpuCount > 0 {
		if provider.AvailableSpecs.GpuCount < specs.GpuCount {
			return false
		}
		if specs.GpuType != "" && provider.AvailableSpecs.GpuType != specs.GpuType {
			return false
		}
	}
	if provider.AvailableSpecs.StorageGb < specs.StorageGb {
		return false
	}

	return true
}

// EstimateCost estimates the cost of a compute request based on provider pricing
func (k Keeper) EstimateCost(ctx context.Context, providerAddr sdk.AccAddress, specs types.ComputeSpec) (math.Int, math.LegacyDec, error) {
	provider, err := k.GetProvider(ctx, providerAddr)
	if err != nil {
		return math.Int{}, math.LegacyDec{}, err
	}

	// Calculate cost per hour
	cpuCost := provider.Pricing.CpuPricePerMcoreHour.MulInt64(types.SaturateUint64ToInt64(specs.CpuCores))
	memoryCost := provider.Pricing.MemoryPricePerMbHour.MulInt64(types.SaturateUint64ToInt64(specs.MemoryMb))
	gpuCost := provider.Pricing.GpuPricePerHour.MulInt64(types.SaturateUint64ToInt64(uint64(specs.GpuCount)))
	storageCost := provider.Pricing.StoragePricePerGbHour.MulInt64(types.SaturateUint64ToInt64(specs.StorageGb))

	costPerHour := cpuCost.Add(memoryCost).Add(gpuCost).Add(storageCost)

	// Calculate total cost based on timeout
	hours := math.LegacyNewDec(types.SaturateUint64ToInt64(specs.TimeoutSeconds)).QuoInt64(3600)
	totalCost := costPerHour.Mul(hours)

	// Convert to integer (round up)
	totalCostInt := totalCost.Ceil().TruncateInt()

	return totalCostInt, costPerHour, nil
}

// SEC-20: MaxProviders is the maximum number of registered providers allowed
// This prevents state bloat attacks through unlimited registrations
const MaxProviders = 10000

// GetTotalProviderCount returns the total number of registered providers
// This is an O(1) operation using a counter key
func (k Keeper) GetTotalProviderCount(ctx context.Context) uint64 {
	store := k.getStore(ctx)
	bz := store.Get(TotalProvidersKey)
	if bz == nil {
		return 0
	}
	return binary.BigEndian.Uint64(bz)
}

// incrementTotalProviderCount increments the total provider count by 1
func (k Keeper) incrementTotalProviderCount(ctx context.Context) {
	store := k.getStore(ctx)
	count := k.GetTotalProviderCount(ctx)
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, count+1)
	store.Set(TotalProvidersKey, bz)
}

// decrementTotalProviderCount decrements the total provider count by 1
func (k Keeper) decrementTotalProviderCount(ctx context.Context) {
	store := k.getStore(ctx)
	count := k.GetTotalProviderCount(ctx)
	if count == 0 {
		return // Don't underflow
	}
	bz := make([]byte, 8)
	binary.BigEndian.PutUint64(bz, count-1)
	store.Set(TotalProvidersKey, bz)
}
