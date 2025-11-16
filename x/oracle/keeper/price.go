package keeper

import (
	"encoding/json"
	"fmt"
	"time"

	"cosmossdk.io/math"
	storetypes "cosmossdk.io/store/types"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/oracle/types"
)

// SetPriceFeed stores a price feed for an asset
func (k Keeper) SetPriceFeed(ctx sdk.Context, priceFeed types.PriceFeed) error {
	if err := priceFeed.Validate(); err != nil {
		return fmt.Errorf("invalid price feed: %w", err)
	}

	store := k.storeService.OpenKVStore(ctx)
	key := types.KeyPrefix(types.PriceFeedKeyPrefix + priceFeed.Asset)

	bz, err := k.cdc.Marshal(&priceFeed)
	if err != nil {
		return fmt.Errorf("failed to marshal price feed: %w", err)
	}

	if err := store.Set(key, bz); err != nil {
		return fmt.Errorf("failed to set price feed: %w", err)
	}

	// Emit event
	ctx.EventManager().EmitEvent(
		sdk.NewEvent(
			"price_feed_updated",
			sdk.NewAttribute("asset", priceFeed.Asset),
			sdk.NewAttribute("price", priceFeed.Price.String()),
			sdk.NewAttribute("timestamp", fmt.Sprintf("%d", priceFeed.Timestamp)),
			sdk.NewAttribute("validators", fmt.Sprintf("%d", len(priceFeed.Validators))),
		),
	)

	return nil
}

// GetPriceFeed retrieves a price feed for an asset
func (k Keeper) GetPriceFeed(ctx sdk.Context, asset string) (types.PriceFeed, bool) {
	store := k.storeService.OpenKVStore(ctx)
	key := types.KeyPrefix(types.PriceFeedKeyPrefix + asset)

	bz, err := store.Get(key)
	if err != nil || bz == nil {
		return types.PriceFeed{}, false
	}

	var priceFeed types.PriceFeed
	if err := k.cdc.Unmarshal(bz, &priceFeed); err != nil {
		return types.PriceFeed{}, false
	}

	return priceFeed, true
}

// GetAllPriceFeeds retrieves all price feeds
func (k Keeper) GetAllPriceFeeds(ctx sdk.Context) []types.PriceFeed {
	store := k.storeService.OpenKVStore(ctx)
	prefix := types.KeyPrefix(types.PriceFeedKeyPrefix)

	var priceFeeds []types.PriceFeed

	// Create iterator
	iterator, err := store.Iterator(prefix, storetypes.PrefixEndBytes(prefix))
	if err != nil {
		return priceFeeds
	}
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var priceFeed types.PriceFeed
		if err := k.cdc.Unmarshal(iterator.Value(), &priceFeed); err != nil {
			continue
		}
		priceFeeds = append(priceFeeds, priceFeed)
	}

	return priceFeeds
}

// DeletePriceFeed removes a price feed for an asset
func (k Keeper) DeletePriceFeed(ctx sdk.Context, asset string) error {
	store := k.storeService.OpenKVStore(ctx)
	key := types.KeyPrefix(types.PriceFeedKeyPrefix + asset)
	return store.Delete(key)
}

// IsPriceFeedStale checks if a price feed is stale based on expiry duration
func (k Keeper) IsPriceFeedStale(ctx sdk.Context, priceFeed types.PriceFeed) bool {
	params := k.GetParams(ctx)
	currentTime := ctx.BlockTime()
	priceTime := time.Unix(priceFeed.Timestamp, 0)

	return currentTime.Sub(priceTime) > time.Duration(params.ExpiryDuration)*time.Second
}

// GetValidPrice returns the price if it exists and is not stale
func (k Keeper) GetValidPrice(ctx sdk.Context, asset string) (math.LegacyDec, bool) {
	priceFeed, found := k.GetPriceFeed(ctx, asset)
	if !found {
		return math.LegacyDec{}, false
	}

	if k.IsPriceFeedStale(ctx, priceFeed) {
		return math.LegacyDec{}, false
	}

	return priceFeed.Price, true
}

// SetValidatorSubmission stores a validator's price submission
func (k Keeper) SetValidatorSubmission(ctx sdk.Context, submission types.ValidatorPriceSubmission) error {
	if err := submission.Validate(); err != nil {
		return fmt.Errorf("invalid validator submission: %w", err)
	}

	store := k.storeService.OpenKVStore(ctx)
	key := types.KeyPrefix(fmt.Sprintf("%s%s/%s", types.ValidatorKeyPrefix, submission.Asset, submission.Validator))

	bz, err := json.Marshal(&submission)
	if err != nil {
		return fmt.Errorf("failed to marshal validator submission: %w", err)
	}

	if err := store.Set(key, bz); err != nil {
		return fmt.Errorf("failed to set validator submission: %w", err)
	}

	return nil
}

// GetValidatorSubmission retrieves a validator's price submission for an asset
func (k Keeper) GetValidatorSubmission(ctx sdk.Context, asset, validator string) (types.ValidatorPriceSubmission, bool) {
	store := k.storeService.OpenKVStore(ctx)
	key := types.KeyPrefix(fmt.Sprintf("%s%s/%s", types.ValidatorKeyPrefix, asset, validator))

	bz, err := store.Get(key)
	if err != nil || bz == nil {
		return types.ValidatorPriceSubmission{}, false
	}

	var submission types.ValidatorPriceSubmission
	if err := json.Unmarshal(bz, &submission); err != nil {
		return types.ValidatorPriceSubmission{}, false
	}

	return submission, true
}

// GetValidatorSubmissions retrieves all validator submissions for an asset
func (k Keeper) GetValidatorSubmissions(ctx sdk.Context, asset string) []types.ValidatorPriceSubmission {
	store := k.storeService.OpenKVStore(ctx)
	prefix := types.KeyPrefix(types.ValidatorKeyPrefix + asset + "/")

	var submissions []types.ValidatorPriceSubmission

	iterator, err := store.Iterator(prefix, storetypes.PrefixEndBytes(prefix))
	if err != nil {
		return submissions
	}
	defer iterator.Close()

	for ; iterator.Valid(); iterator.Next() {
		var submission types.ValidatorPriceSubmission
		if err := json.Unmarshal(iterator.Value(), &submission); err != nil {
			continue
		}
		submissions = append(submissions, submission)
	}

	return submissions
}

// DeleteValidatorSubmission removes a validator's price submission
func (k Keeper) DeleteValidatorSubmission(ctx sdk.Context, asset, validator string) error {
	store := k.storeService.OpenKVStore(ctx)
	key := types.KeyPrefix(fmt.Sprintf("%s%s/%s", types.ValidatorKeyPrefix, asset, validator))
	return store.Delete(key)
}

// CleanupStaleSubmissions removes stale validator submissions
func (k Keeper) CleanupStaleSubmissions(ctx sdk.Context, asset string) {
	submissions := k.GetValidatorSubmissions(ctx, asset)
	params := k.GetParams(ctx)
	currentTime := ctx.BlockTime()

	for _, submission := range submissions {
		if submission.IsStale(currentTime, params.ExpiryDuration) {
			_ = k.DeleteValidatorSubmission(ctx, asset, submission.Validator)
		}
	}
}

// SetValidatorAccuracy stores validator accuracy tracking
func (k Keeper) SetValidatorAccuracy(ctx sdk.Context, accuracy types.ValidatorAccuracy) error {
	store := k.storeService.OpenKVStore(ctx)
	key := types.KeyPrefix("accuracy/" + accuracy.Validator)

	bz, err := json.Marshal(&accuracy)
	if err != nil {
		return fmt.Errorf("failed to marshal validator accuracy: %w", err)
	}

	return store.Set(key, bz)
}

// GetValidatorAccuracy retrieves validator accuracy tracking
func (k Keeper) GetValidatorAccuracy(ctx sdk.Context, validator string) types.ValidatorAccuracy {
	store := k.storeService.OpenKVStore(ctx)
	key := types.KeyPrefix("accuracy/" + validator)

	bz, err := store.Get(key)
	if err != nil || bz == nil {
		return types.NewValidatorAccuracy(validator)
	}

	var accuracy types.ValidatorAccuracy
	if err := json.Unmarshal(bz, &accuracy); err != nil {
		return types.NewValidatorAccuracy(validator)
	}

	return accuracy
}
