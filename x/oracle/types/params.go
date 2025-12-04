package types

import (
	"cosmossdk.io/math"
)

// DefaultParams returns default oracle parameters
func DefaultParams() Params {
	return Params{
		VotePeriod:         30, // 30 blocks
		VoteThreshold:      math.LegacyMustNewDecFromStr("0.67"),
		SlashFraction:      math.LegacyMustNewDecFromStr("0.01"),
		SlashWindow:        10000,
		MinValidPerWindow:  100,
		TwapLookbackWindow: 1000,
		AuthorizedChannels: []AuthorizedChannel{},
	}
}
