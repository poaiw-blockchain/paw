package types

import (
	"fmt"

	sdk "github.com/cosmos/cosmos-sdk/types"
)

// DefaultGenesis returns the default genesis state
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:     DefaultParams(),
		PriceFeeds: []PriceFeed{},
	}
}

// Validate performs basic genesis state validation returning an error upon any
// failure.
func (gs GenesisState) Validate() error {
	// Validate params
	if err := gs.Params.Validate(); err != nil {
		return fmt.Errorf("invalid params: %w", err)
	}

	// Track seen assets to detect duplicates
	seenAssets := make(map[string]bool)

	// Validate each price feed
	for i, priceFeed := range gs.PriceFeeds {
		// Validate asset
		if priceFeed.Asset == "" {
			return fmt.Errorf("price feed %d: asset cannot be empty", i)
		}

		// Check for duplicate assets
		if seenAssets[priceFeed.Asset] {
			return fmt.Errorf("price feed %d: duplicate asset %s", i, priceFeed.Asset)
		}
		seenAssets[priceFeed.Asset] = true

		// Validate price is positive
		if priceFeed.Price.IsNil() || !priceFeed.Price.IsPositive() {
			return fmt.Errorf("price feed %d (%s): price must be positive", i, priceFeed.Asset)
		}

		// Validate timestamp is non-negative
		if priceFeed.Timestamp < 0 {
			return fmt.Errorf("price feed %d (%s): timestamp cannot be negative", i, priceFeed.Asset)
		}

		// Validate validators
		if len(priceFeed.Validators) == 0 {
			return fmt.Errorf("price feed %d (%s): must have at least one validator", i, priceFeed.Asset)
		}

		// Validate each validator address
		seenValidators := make(map[string]bool)
		for j, valAddr := range priceFeed.Validators {
			if valAddr == "" {
				return fmt.Errorf("price feed %d (%s): validator %d address cannot be empty", i, priceFeed.Asset, j)
			}

			// Check for duplicate validators in this price feed
			if seenValidators[valAddr] {
				return fmt.Errorf("price feed %d (%s): duplicate validator %s", i, priceFeed.Asset, valAddr)
			}
			seenValidators[valAddr] = true

			// Validate address format
			if _, err := sdk.AccAddressFromBech32(valAddr); err != nil {
				return fmt.Errorf("price feed %d (%s): invalid validator address %s: %w", i, priceFeed.Asset, valAddr, err)
			}
		}
	}

	return nil
}
