package types

import (
	"cosmossdk.io/math"
)

// DefaultParams returns default oracle parameters
func DefaultParams() Params {
	return Params{
		VotePeriod:                 30, // 30 blocks
		VoteThreshold:              math.LegacyMustNewDecFromStr("0.67"),
		SlashFraction:              math.LegacyMustNewDecFromStr("0.01"),
		SlashWindow:                10000,
		MinValidPerWindow:          100,
		TwapLookbackWindow:         1000,
		AuthorizedChannels:         []AuthorizedChannel{},
		AllowedRegions:             []string{"global", "na", "eu", "apac", "latam", "africa"},
		MinGeographicRegions:       1,
		MinVotingPowerForConsensus: math.LegacyMustNewDecFromStr("0.10"), // 10% minimum voting power
		MaxValidatorsPerIp:         3,                                     // Max 3 validators per IP address
		MaxValidatorsPerAsn:        5,                                     // Max 5 validators per ASN
	}
}
