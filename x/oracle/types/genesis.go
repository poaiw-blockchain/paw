package types

import (
	"fmt"

	sdkmath "cosmossdk.io/math"
)

// DefaultGenesis returns the default genesis state for the oracle module.
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params:           DefaultParams(),
		Prices:           []Price{},
		ValidatorPrices:  []ValidatorPrice{},
		ValidatorOracles: []ValidatorOracle{},
		PriceSnapshots:   []PriceSnapshot{},
	}
}

// Validate ensures the genesis state is well-formed.
func (gs GenesisState) Validate() error {
	p := gs.Params
	if p.VotePeriod == 0 {
		return fmt.Errorf("vote period must be positive")
	}
	if p.VoteThreshold.LTE(sdkmath.LegacyZeroDec()) || p.VoteThreshold.GT(sdkmath.LegacyOneDec()) {
		return fmt.Errorf("vote threshold must be in (0,1]")
	}
	if p.SlashFraction.LT(sdkmath.LegacyZeroDec()) || p.SlashFraction.GT(sdkmath.LegacyOneDec()) {
		return fmt.Errorf("slash fraction must be between 0 and 1")
	}
	if p.SlashWindow == 0 || p.MinValidPerWindow == 0 {
		return fmt.Errorf("slash window and min valid per window must be positive")
	}
	if p.TwapLookbackWindow == 0 {
		return fmt.Errorf("twap lookback window must be positive")
	}

	return nil
}
