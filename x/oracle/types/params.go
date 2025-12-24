package types

import (
	"cosmossdk.io/math"
)

// DefaultParams returns default oracle parameters
func DefaultParams() Params {
	return Params{
		VotePeriod:    30, // 30 blocks
		VoteThreshold: math.LegacyMustNewDecFromStr("0.67"),
		// SlashFraction: 5% slash for oracle violations (increased from 1% for security)
		// Rationale: A validator could profit from price manipulation that yields >1% gain,
		// making a 1% slash insufficient deterrent. 5% ensures manipulation is unprofitable
		// even with significant potential gains, while remaining reasonable for honest mistakes.
		SlashFraction:              math.LegacyMustNewDecFromStr("0.05"),
		SlashWindow:                10000,
		MinValidPerWindow:          100,
		TwapLookbackWindow:         1000,
		AuthorizedChannels:         []AuthorizedChannel{},
		AllowedRegions:             []string{"global", "na", "eu", "apac", "latam", "africa"},
		MinGeographicRegions:       1,
		MinVotingPowerForConsensus: math.LegacyMustNewDecFromStr("0.10"), // 10% minimum voting power
		MaxValidatorsPerIp:         3,                                     // Max 3 validators per IP address
		MaxValidatorsPerAsn:        5,                                     // Max 5 validators per ASN
		RequireGeographicDiversity: false,                                 // Default to optional for testnet
		NonceTtlSeconds:            604800,                                // 7 days (604800 seconds)
		DiversityCheckInterval:     100,                                   // Check diversity every 100 blocks
		DiversityWarningThreshold:  math.LegacyMustNewDecFromStr("0.40"), // Warn below 40% diversity
		EnforceRuntimeDiversity:    false,                                 // Default to warning-only for testnet
		EmergencyAdmin:             "",                                    // No emergency admin by default
		GeoipCacheTtlSeconds:       3600,                                  // 1 hour cache TTL
		GeoipCacheMaxEntries:       1000,                                  // 1000 max cached IPs
	}
}

// MainnetParams returns oracle parameters suitable for mainnet deployment
func MainnetParams() Params {
	params := DefaultParams()
	params.RequireGeographicDiversity = true // Mandatory for mainnet
	params.MinGeographicRegions = 3          // Require at least 3 distinct regions
	params.EnforceRuntimeDiversity = true    // Enforce diversity at runtime
	// Increase slash fraction to 7.5% for mainnet (higher stakes require stronger deterrent)
	params.SlashFraction = math.LegacyMustNewDecFromStr("0.075")
	return params
}
