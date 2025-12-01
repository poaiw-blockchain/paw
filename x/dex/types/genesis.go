package types

import (
	"fmt"

	sdkmath "cosmossdk.io/math"
)

// DefaultGenesis returns the default genesis state for the DEX module.
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params: Params{
			SwapFee:            sdkmath.LegacyMustNewDecFromStr("0.003"),  // 0.30%
			LpFee:              sdkmath.LegacyMustNewDecFromStr("0.0025"), // 0.25%
			ProtocolFee:        sdkmath.LegacyMustNewDecFromStr("0.0005"), // 0.05%
			MinLiquidity:       sdkmath.NewInt(1_000),
			MaxSlippagePercent: sdkmath.LegacyMustNewDecFromStr("0.50"), // 50% guardrail
		},
		Pools:      []Pool{},
		NextPoolId: 1,
	}
}

// Validate ensures the genesis state is well-formed.
func (gs GenesisState) Validate() error {
	p := gs.Params

	if p.SwapFee.IsNegative() || p.SwapFee.GTE(sdkmath.LegacyOneDec()) {
		return fmt.Errorf("swap fee must be in [0,1)")
	}
	if p.LpFee.IsNegative() || p.ProtocolFee.IsNegative() {
		return fmt.Errorf("lp/protocol fees must be non-negative")
	}
	if p.LpFee.Add(p.ProtocolFee).GT(p.SwapFee) {
		return fmt.Errorf("lp fee plus protocol fee must not exceed swap fee")
	}
	if p.MinLiquidity.IsNegative() {
		return fmt.Errorf("min liquidity must be non-negative")
	}
	if p.MaxSlippagePercent.IsNegative() || p.MaxSlippagePercent.GT(sdkmath.LegacyOneDec()) {
		return fmt.Errorf("max slippage percent must be between 0 and 1")
	}
	if gs.NextPoolId == 0 {
		return fmt.Errorf("next pool id must be positive")
	}

	return nil
}
