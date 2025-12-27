package keeper

import (
	"context"
	"fmt"
	"time"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"
	sdkerrors "github.com/cosmos/cosmos-sdk/types/errors"

	"github.com/paw-chain/paw/x/oracle/types"
)

// SetPrice sets the current aggregated price for an asset
func (k Keeper) SetPrice(ctx context.Context, price types.Price) error {
	store := k.getStore(ctx)
	bz, err := k.cdc.Marshal(&price)
	if err != nil {
		return err
	}
	store.Set(GetPriceKey(price.Asset), bz)

	// Emit event
	sdkCtx := sdk.UnwrapSDKContext(ctx)
	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeOraclePriceUpdate,
			sdk.NewAttribute(types.AttributeKeyAsset, price.Asset),
			sdk.NewAttribute(types.AttributeKeyPrice, price.Price.String()),
			sdk.NewAttribute(types.AttributeKeyBlockHeight, fmt.Sprintf("%d", price.BlockHeight)),
			sdk.NewAttribute(types.AttributeKeyNumValidators, fmt.Sprintf("%d", price.NumValidators)),
		),
	)

	return nil
}

// GetPrice retrieves the current price for an asset
func (k Keeper) GetPrice(ctx context.Context, asset string) (types.Price, error) {
	store := k.getStore(ctx)
	bz := store.Get(GetPriceKey(asset))
	if bz == nil {
		return types.Price{}, fmt.Errorf("price not found for asset: %s", asset)
	}

	var price types.Price
	if err := k.cdc.Unmarshal(bz, &price); err != nil {
		return types.Price{}, err
	}
	return price, nil
}

// DeletePrice removes a price from the store
func (k Keeper) DeletePrice(ctx context.Context, asset string) {
	store := k.getStore(ctx)
	store.Delete(GetPriceKey(asset))
}

// IteratePrices iterates over all prices in the store
func (k Keeper) IteratePrices(ctx context.Context, cb func(price types.Price) (stop bool)) error {
	store := k.getStore(ctx)
	iterator := storetypes.KVStorePrefixIterator(store, PriceKeyPrefix)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var price types.Price
		if err := k.cdc.Unmarshal(iterator.Value(), &price); err != nil {
			return err
		}
		if cb(price) {
			break
		}
	}
	return nil
}

// GetAllPrices returns all current prices
func (k Keeper) GetAllPrices(ctx context.Context) ([]types.Price, error) {
	// P3-PERF-3: Pre-size with estimated capacity (typically small number of assets)
	prices := make([]types.Price, 0, 50)
	err := k.IteratePrices(ctx, func(price types.Price) bool {
		prices = append(prices, price)
		return false
	})
	return prices, err
}

// SetValidatorPrice stores a validator's price submission for an asset.
// Creates both primary storage and secondary index for efficient asset-based queries.
func (k Keeper) SetValidatorPrice(ctx context.Context, validatorPrice types.ValidatorPrice) error {
	valAddr, err := sdk.ValAddressFromBech32(validatorPrice.ValidatorAddr)
	if err != nil {
		return err
	}

	store := k.getStore(ctx)
	bz, err := k.cdc.Marshal(&validatorPrice)
	if err != nil {
		return err
	}
	store.Set(GetValidatorPriceKey(valAddr, validatorPrice.Asset), bz)

	// PERF-6: Write secondary index for efficient asset-based iteration
	// Index stores empty value since actual data is in primary key
	store.Set(GetValidatorPriceByAssetKey(validatorPrice.Asset, valAddr), []byte{})

	return nil
}

// SubmitPrice records a validator price submission with full validation and triggers aggregation.
// Validates the submitter is an active bonded validator and the price is within acceptable bounds.
// Triggers price aggregation at vote period boundaries. Returns error if validator is not bonded.
func (k Keeper) SubmitPrice(ctx context.Context, validator sdk.ValAddress, asset string, price math.LegacyDec, feeders ...sdk.AccAddress) error {
	// Track price submission latency
	start := time.Now()
	defer func() {
		if k.metrics != nil {
			k.metrics.PriceSubmissionLatency.Observe(time.Since(start).Seconds())
		}
	}()

	sdkCtx := sdk.UnwrapSDKContext(ctx)

	if asset == "" {
		return sdkerrors.ErrInvalidRequest.Wrap("asset identifier cannot be empty")
	}

	if price.IsNil() || price.LTE(math.LegacyZeroDec()) {
		return sdkerrors.ErrInvalidRequest.Wrap("price must be positive")
	}

	isActive, err := k.IsActiveValidator(ctx, validator)
	if err != nil {
		return err
	}
	if !isActive {
		return fmt.Errorf("validator %s is not bonded", validator.String())
	}

	if err := k.ValidatePriceSubmission(ctx, validator, asset, price); err != nil {
		return err
	}

	votingPower, err := k.GetValidatorVotingPower(ctx, validator)
	if err != nil {
		return err
	}

	validatorPrice := types.ValidatorPrice{
		ValidatorAddr: validator.String(),
		Asset:         asset,
		Price:         price,
		BlockHeight:   sdkCtx.BlockHeight(),
		VotingPower:   votingPower,
	}

	if err := k.SetValidatorPrice(ctx, validatorPrice); err != nil {
		return err
	}

	if err := k.IncrementSubmissionCount(ctx, validator.String()); err != nil {
		return err
	}

	if err := k.ResetMissCounter(ctx, validator.String()); err != nil {
		return err
	}

	if err := k.RecordSubmission(ctx, validator.String()); err != nil {
		sdkCtx.Logger().Error("failed to record submission", "validator", validator.String(), "error", err)
	}

	// P3-PERF-3: Pre-size with known capacity (4-5 attributes)
	attrs := make([]sdk.Attribute, 4, 5)
	attrs[0] = sdk.NewAttribute(types.AttributeKeyValidator, validator.String())
	attrs[1] = sdk.NewAttribute(types.AttributeKeyAsset, asset)
	attrs[2] = sdk.NewAttribute(types.AttributeKeyPrice, price.String())
	attrs[3] = sdk.NewAttribute(types.AttributeKeyVotingPower, fmt.Sprintf("%d", votingPower))

	if len(feeders) > 0 && feeders[0] != nil {
		attrs = append(attrs, sdk.NewAttribute(types.AttributeKeyFeeder, feeders[0].String()))
	}

	sdkCtx.EventManager().EmitEvent(
		sdk.NewEvent(
			types.EventTypeOraclePriceSubmitted,
			attrs...,
		),
	)

	// Trigger aggregation at vote period boundaries
	if params, err := k.GetParams(ctx); err == nil && params.VotePeriod > 0 {
		if sdkCtx.BlockHeight()%int64(params.VotePeriod) == 0 {
			if err := k.AggregatePrices(ctx); err != nil {
				sdkCtx.Logger().Debug("price aggregation not ready", "asset", asset, "error", err.Error())
			}
		}
	}

	// Record successful price submission metrics
	if k.metrics != nil {
		k.metrics.PriceSubmissions.With(map[string]string{
			"validator": validator.String(),
			"asset":     asset,
		}).Inc()

		k.metrics.AggregatedPrice.With(map[string]string{
			"asset": asset,
		}).Set(price.MustFloat64())
	}

	return nil
}

// GetValidatorPrice retrieves a validator's price submission for an asset
func (k Keeper) GetValidatorPrice(ctx context.Context, validatorAddr sdk.ValAddress, asset string) (types.ValidatorPrice, error) {
	store := k.getStore(ctx)
	bz := store.Get(GetValidatorPriceKey(validatorAddr, asset))
	if bz == nil {
		return types.ValidatorPrice{}, fmt.Errorf("validator price not found")
	}

	var validatorPrice types.ValidatorPrice
	if err := k.cdc.Unmarshal(bz, &validatorPrice); err != nil {
		return types.ValidatorPrice{}, err
	}
	return validatorPrice, nil
}

// DeleteValidatorPrice removes a validator's price submission
func (k Keeper) DeleteValidatorPrice(ctx context.Context, validatorAddr sdk.ValAddress, asset string) {
	store := k.getStore(ctx)
	store.Delete(GetValidatorPriceKey(validatorAddr, asset))
	// PERF-6: Also delete secondary index
	store.Delete(GetValidatorPriceByAssetKey(asset, validatorAddr))
}

// IterateValidatorPrices iterates over all validator prices for an asset
// PERF-6: When asset is specified, uses secondary index for O(v) instead of O(V) iteration
func (k Keeper) IterateValidatorPrices(ctx context.Context, asset string, cb func(validatorPrice types.ValidatorPrice) (stop bool)) error {
	// PERF-6: Use asset-specific iteration when asset is specified
	if asset != "" {
		return k.IterateValidatorPricesByAsset(ctx, asset, cb)
	}

	// Iterate all prices when no asset filter
	store := k.getStore(ctx)
	iterator := storetypes.KVStorePrefixIterator(store, ValidatorPriceKeyPrefix)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var validatorPrice types.ValidatorPrice
		if err := k.cdc.Unmarshal(iterator.Value(), &validatorPrice); err != nil {
			return err
		}
		if cb(validatorPrice) {
			break
		}
	}
	return nil
}

// IterateValidatorPricesByAsset iterates over validator prices for a specific asset
// PERF-6: Uses secondary index (asset -> validator) for efficient O(v) iteration
// where v = validators who submitted for this asset, instead of O(V) for all validators
func (k Keeper) IterateValidatorPricesByAsset(ctx context.Context, asset string, cb func(validatorPrice types.ValidatorPrice) (stop bool)) error {
	store := k.getStore(ctx)

	// Use secondary index prefix to iterate only validators for this asset
	prefix := GetValidatorPricesByAssetPrefix(asset)
	iterator := storetypes.KVStorePrefixIterator(store, prefix)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		// Extract validator address from index key
		// Key format: prefix + asset + 0x00 + validator
		key := iterator.Key()
		if len(key) <= len(prefix) {
			continue
		}
		validatorStr := string(key[len(prefix):])
		valAddr, err := sdk.ValAddressFromBech32(validatorStr)
		if err != nil {
			continue // Skip malformed keys
		}

		// Fetch actual validator price from primary storage
		validatorPrice, err := k.GetValidatorPrice(ctx, valAddr, asset)
		if err != nil {
			continue // Index entry without corresponding price (should not happen)
		}

		if cb(validatorPrice) {
			break
		}
	}
	return nil
}

// GetValidatorPricesByAsset returns all validator price submissions for an asset
func (k Keeper) GetValidatorPricesByAsset(ctx context.Context, asset string) ([]types.ValidatorPrice, error) {
	// P3-PERF-3: Pre-size with estimated validator count
	validatorPrices := make([]types.ValidatorPrice, 0, 100)
	err := k.IterateValidatorPrices(ctx, asset, func(vp types.ValidatorPrice) bool {
		validatorPrices = append(validatorPrices, vp)
		return false
	})
	return validatorPrices, err
}

// GetAllValidatorPrices returns validator prices, optionally filtered by asset.
func (k Keeper) GetAllValidatorPrices(ctx context.Context, asset string) ([]types.ValidatorPrice, error) {
	// P3-PERF-3: Pre-size with estimated validator count
	validatorPrices := make([]types.ValidatorPrice, 0, 100)
	err := k.IterateValidatorPrices(ctx, asset, func(vp types.ValidatorPrice) bool {
		validatorPrices = append(validatorPrices, vp)
		return false
	})
	return validatorPrices, err
}

// SetPriceSnapshot stores a price snapshot for TWAP calculation
func (k Keeper) SetPriceSnapshot(ctx context.Context, snapshot types.PriceSnapshot) error {
	store := k.getStore(ctx)
	bz, err := k.cdc.Marshal(&snapshot)
	if err != nil {
		return err
	}
	store.Set(GetPriceSnapshotKey(snapshot.Asset, snapshot.BlockHeight), bz)
	return nil
}

// GetPriceSnapshot retrieves a specific price snapshot
func (k Keeper) GetPriceSnapshot(ctx context.Context, asset string, blockHeight int64) (types.PriceSnapshot, error) {
	store := k.getStore(ctx)
	bz := store.Get(GetPriceSnapshotKey(asset, blockHeight))
	if bz == nil {
		return types.PriceSnapshot{}, fmt.Errorf("price snapshot not found")
	}

	var snapshot types.PriceSnapshot
	if err := k.cdc.Unmarshal(bz, &snapshot); err != nil {
		return types.PriceSnapshot{}, err
	}
	return snapshot, nil
}

// IteratePriceSnapshots iterates over price snapshots for an asset
func (k Keeper) IteratePriceSnapshots(ctx context.Context, asset string, cb func(snapshot types.PriceSnapshot) (stop bool)) error {
	store := k.getStore(ctx)
	prefix := GetPriceSnapshotsByAssetKey(asset)
	iterator := storetypes.KVStorePrefixIterator(store, prefix)
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var snapshot types.PriceSnapshot
		if err := k.cdc.Unmarshal(iterator.Value(), &snapshot); err != nil {
			return err
		}
		if cb(snapshot) {
			break
		}
	}
	return nil
}

// DeleteOldSnapshots removes snapshots older than the lookback window
func (k Keeper) DeleteOldSnapshots(ctx context.Context, asset string, minBlockHeight int64) error {
	store := k.getStore(ctx)
	prefix := GetPriceSnapshotsByAssetKey(asset)
	iterator := storetypes.KVStorePrefixIterator(store, prefix)
	defer iterator.Close()

	// P3-PERF-3: Pre-size with estimated number of old snapshots to delete
	keysToDelete := make([][]byte, 0, 100)
	for ; iterator.Valid(); iterator.Next() {
		var snapshot types.PriceSnapshot
		if err := k.cdc.Unmarshal(iterator.Value(), &snapshot); err != nil {
			return err
		}
		if snapshot.BlockHeight < minBlockHeight {
			keysToDelete = append(keysToDelete, iterator.Key())
		}
	}

	// Delete outside iteration
	for _, key := range keysToDelete {
		store.Delete(key)
	}

	return nil
}

// SetFeederDelegation sets a feeder delegation for a validator
func (k Keeper) SetFeederDelegation(ctx context.Context, validatorAddr sdk.ValAddress, feederAddr sdk.AccAddress) error {
	store := k.getStore(ctx)
	store.Set(GetFeederDelegationKey(validatorAddr), feederAddr.Bytes())
	return nil
}

// GetFeederDelegation retrieves the feeder address for a validator
func (k Keeper) GetFeederDelegation(ctx context.Context, validatorAddr sdk.ValAddress) (sdk.AccAddress, error) {
	store := k.getStore(ctx)
	bz := store.Get(GetFeederDelegationKey(validatorAddr))
	if bz == nil {
		// No delegation, validator must submit themselves
		return nil, nil
	}
	return sdk.AccAddress(bz), nil
}

// DeleteFeederDelegation removes a feeder delegation
func (k Keeper) DeleteFeederDelegation(ctx context.Context, validatorAddr sdk.ValAddress) {
	store := k.getStore(ctx)
	store.Delete(GetFeederDelegationKey(validatorAddr))
}
