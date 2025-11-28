package keeper

import (
	"context"
	"fmt"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"

	"github.com/paw-chain/paw/x/oracle/types"
)

// InitGenesis initializes the oracle module's state from a genesis state
func (k Keeper) InitGenesis(ctx context.Context, data types.GenesisState) error {
	// Set parameters
	if err := k.SetParams(ctx, data.Params); err != nil {
		return fmt.Errorf("failed to set params: %w", err)
	}

	// Set prices
	for _, price := range data.Prices {
		if err := k.SetPrice(ctx, price); err != nil {
			return fmt.Errorf("failed to set price for %s: %w", price.Asset, err)
		}
	}

	// Set validator prices
	for _, validatorPrice := range data.ValidatorPrices {
		if err := k.SetValidatorPrice(ctx, validatorPrice); err != nil {
			return fmt.Errorf("failed to set validator price: %w", err)
		}
	}

	// Set validator oracles
	for _, validatorOracle := range data.ValidatorOracles {
		if err := k.SetValidatorOracle(ctx, validatorOracle); err != nil {
			return fmt.Errorf("failed to set validator oracle: %w", err)
		}
	}

	// Set price snapshots
	for _, snapshot := range data.PriceSnapshots {
		if err := k.SetPriceSnapshot(ctx, snapshot); err != nil {
			return fmt.Errorf("failed to set price snapshot: %w", err)
		}
	}



	return nil
}

// ExportGenesis exports the oracle module's state to a genesis state
func (k Keeper) ExportGenesis(ctx context.Context) (*types.GenesisState, error) {
	// Get parameters
	params, err := k.GetParams(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get params: %w", err)
	}

	// Get all prices
	prices, err := k.GetAllPrices(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get prices: %w", err)
	}

	// Get all validator prices
	validatorPrices := []types.ValidatorPrice{}
	if err := k.IterateValidatorPrices(ctx, "", func(vp types.ValidatorPrice) bool {
		validatorPrices = append(validatorPrices, vp)
		return false
	}); err != nil {
		return nil, fmt.Errorf("failed to get validator prices: %w", err)
	}

	// Get all validator oracles
	validatorOracles, err := k.GetAllValidatorOracles(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get validator oracles: %w", err)
	}

	// Get all price snapshots
	priceSnapshots := []types.PriceSnapshot{}
	for _, price := range prices {
		if err := k.IteratePriceSnapshots(ctx, price.Asset, func(snapshot types.PriceSnapshot) bool {
			priceSnapshots = append(priceSnapshots, snapshot)
			return false
		}); err != nil {
			return nil, fmt.Errorf("failed to get price snapshots: %w", err)
		}
	}

	return &types.GenesisState{
		Params:               params,
		Prices:               prices,
		ValidatorPrices:      validatorPrices,
		ValidatorOracles:     validatorOracles,
		PriceSnapshots:       priceSnapshots,
	}, nil
}

// DefaultGenesis returns the default genesis state for the oracle module
func DefaultGenesis() *types.GenesisState {
	return &types.GenesisState{
		Params: types.Params{
			VotePeriod:         10,                               // 10 blocks
			VoteThreshold:      math.LegacyNewDecWithPrec(67, 2), // 67%
			SlashFraction:      math.LegacyNewDecWithPrec(1, 4),  // 0.01%
			SlashWindow:        100,                              // 100 blocks
			MinValidPerWindow:  90,                               // must submit 90 out of 100
			TwapLookbackWindow: 3600,                             // ~1 hour at 1s blocks
		},
		Prices:           []types.Price{},
		ValidatorPrices:  []types.ValidatorPrice{},
		ValidatorOracles: []types.ValidatorOracle{},
		PriceSnapshots:   []types.PriceSnapshot{},
	}
}

// ValidateGenesis validates the oracle module's genesis state
func ValidateGenesis(data types.GenesisState) error {
	// Validate parameters
	if data.Params.VotePeriod == 0 {
		return fmt.Errorf("vote period must be positive")
	}

	if data.Params.VoteThreshold.IsNil() || data.Params.VoteThreshold.LTE(math.LegacyZeroDec()) || data.Params.VoteThreshold.GT(math.LegacyOneDec()) {
		return fmt.Errorf("vote threshold must be between 0 and 1")
	}

	if data.Params.SlashFraction.IsNil() || data.Params.SlashFraction.LT(math.LegacyZeroDec()) || data.Params.SlashFraction.GT(math.LegacyOneDec()) {
		return fmt.Errorf("slash fraction must be between 0 and 1")
	}

	if data.Params.SlashWindow == 0 {
		return fmt.Errorf("slash window must be positive")
	}

	if data.Params.MinValidPerWindow == 0 {
		return fmt.Errorf("min valid per window must be positive")
	}

	if data.Params.MinValidPerWindow > data.Params.SlashWindow {
		return fmt.Errorf("min valid per window cannot exceed slash window")
	}

	// Validate prices
	priceMap := make(map[string]bool)
	for _, price := range data.Prices {
		if price.Asset == "" {
			return fmt.Errorf("price asset cannot be empty")
		}
		if priceMap[price.Asset] {
			return fmt.Errorf("duplicate price for asset: %s", price.Asset)
		}
		priceMap[price.Asset] = true

		if price.Price.IsNil() || price.Price.LTE(math.LegacyZeroDec()) {
			return fmt.Errorf("price must be positive for asset: %s", price.Asset)
		}
	}

	// Validate validator prices
	for _, vp := range data.ValidatorPrices {
		if vp.ValidatorAddr == "" {
			return fmt.Errorf("validator price: validator address cannot be empty")
		}
		if vp.Asset == "" {
			return fmt.Errorf("validator price: asset cannot be empty")
		}
		if vp.Price.IsNil() || vp.Price.LTE(math.LegacyZeroDec()) {
			return fmt.Errorf("validator price must be positive")
		}
		if _, err := sdk.ValAddressFromBech32(vp.ValidatorAddr); err != nil {
			return fmt.Errorf("invalid validator address in validator price: %s", vp.ValidatorAddr)
		}
	}

	// Validate validator oracles
	validatorMap := make(map[string]bool)
	for _, vo := range data.ValidatorOracles {
		if vo.ValidatorAddr == "" {
			return fmt.Errorf("validator oracle: validator address cannot be empty")
		}
		if validatorMap[vo.ValidatorAddr] {
			return fmt.Errorf("duplicate validator oracle: %s", vo.ValidatorAddr)
		}
		validatorMap[vo.ValidatorAddr] = true

		if _, err := sdk.ValAddressFromBech32(vo.ValidatorAddr); err != nil {
			return fmt.Errorf("invalid validator address in validator oracle: %s", vo.ValidatorAddr)
		}
	}

	// Validate price snapshots
	for _, snapshot := range data.PriceSnapshots {
		if snapshot.Asset == "" {
			return fmt.Errorf("price snapshot: asset cannot be empty")
		}
		if snapshot.Price.IsNil() || snapshot.Price.LTE(math.LegacyZeroDec()) {
			return fmt.Errorf("price snapshot: price must be positive")
		}
	}

	return nil
}
