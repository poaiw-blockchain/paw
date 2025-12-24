package types

import (
	"fmt"
	"strings"

	sdkmath "cosmossdk.io/math"
)

// DefaultGenesis returns the default genesis state for the DEX module.
func DefaultGenesis() *GenesisState {
	return &GenesisState{
		Params: Params{
			SwapFee:                              sdkmath.LegacyMustNewDecFromStr("0.003"),  // 0.30%
			LpFee:                                sdkmath.LegacyMustNewDecFromStr("0.0025"), // 0.25%
			ProtocolFee:                          sdkmath.LegacyMustNewDecFromStr("0.0005"), // 0.05%
			MinLiquidity:                         sdkmath.NewInt(1_000),
			MaxSlippagePercent:                   sdkmath.LegacyMustNewDecFromStr("0.50"), // 50% guardrail
			MaxPoolDrainPercent:                  sdkmath.LegacyMustNewDecFromStr("0.30"), // Production swap drain limit
			FlashLoanProtectionBlocks:            10,
			PoolCreationGas:                      1000,
			SwapValidationGas:                    1500,
			LiquidityGas:                         1200,
			AuthorizedChannels:                   []AuthorizedChannel{},
			UpgradePreserveCircuitBreakerState:   true,
			RecommendedMaxSlippage:               sdkmath.LegacyMustNewDecFromStr("0.03"),  // 3%
			EnableCommitReveal:                   false,                                     // Disabled for testnet
			CommitRevealDelay:                    10,                                        // 10 blocks (~60s)
			CommitTimeoutBlocks:                  100,                                       // 100 blocks (~10min)
		},
		Pools:           []Pool{},
		NextPoolId:      1,
		PoolTwapRecords: []PoolTWAP{},
		SwapCommits:     []SwapCommit{},
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
	if p.MaxPoolDrainPercent.IsNegative() || p.MaxPoolDrainPercent.GT(sdkmath.LegacyOneDec()) {
		return fmt.Errorf("max pool drain percent must be between 0 and 1")
	}
	if p.FlashLoanProtectionBlocks == 0 {
		return fmt.Errorf("flash loan protection blocks must be positive")
	}
	if p.PoolCreationGas == 0 {
		return fmt.Errorf("pool creation gas must be positive")
	}
	if p.SwapValidationGas == 0 {
		return fmt.Errorf("swap validation gas must be positive")
	}
	if p.LiquidityGas == 0 {
		return fmt.Errorf("liquidity gas must be positive")
	}
	for _, ch := range p.AuthorizedChannels {
		if strings.TrimSpace(ch.PortId) == "" {
			return fmt.Errorf("authorized channel port_id cannot be empty")
		}
		if strings.TrimSpace(ch.ChannelId) == "" {
			return fmt.Errorf("authorized channel channel_id cannot be empty")
		}
	}
	if p.RecommendedMaxSlippage.IsNegative() || p.RecommendedMaxSlippage.GT(sdkmath.LegacyOneDec()) {
		return fmt.Errorf("recommended max slippage must be between 0 and 1")
	}
	if p.EnableCommitReveal {
		if p.CommitRevealDelay == 0 {
			return fmt.Errorf("commit reveal delay must be positive when commit-reveal is enabled")
		}
		if p.CommitTimeoutBlocks == 0 {
			return fmt.Errorf("commit timeout blocks must be positive when commit-reveal is enabled")
		}
		if p.CommitTimeoutBlocks <= p.CommitRevealDelay {
			return fmt.Errorf("commit timeout blocks must be greater than commit reveal delay")
		}
	}
	if gs.NextPoolId == 0 {
		return fmt.Errorf("next pool id must be positive")
	}

	for _, twap := range gs.PoolTwapRecords {
		if twap.PoolId == 0 {
			return fmt.Errorf("pool TWAP record missing pool id")
		}
	}

	for _, commit := range gs.SwapCommits {
		if strings.TrimSpace(commit.Trader) == "" {
			return fmt.Errorf("swap commit missing trader address")
		}
		if strings.TrimSpace(commit.SwapHash) == "" {
			return fmt.Errorf("swap commit missing swap hash")
		}
		if commit.CommitHeight <= 0 {
			return fmt.Errorf("swap commit has invalid commit height")
		}
		if commit.ExpiryHeight <= commit.CommitHeight {
			return fmt.Errorf("swap commit expiry height must be greater than commit height")
		}
	}

	return nil
}
