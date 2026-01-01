package types

import (
	"cosmossdk.io/math"
)

// DefaultParams returns default compute parameters
func DefaultParams() Params {
	return Params{
		MinProviderStake:                math.NewInt(1000000), // 1 PAW
		VerificationTimeoutSeconds:      300,
		MaxRequestTimeoutSeconds:        3600,
		ReputationSlashPercentage:       10,
		StakeSlashPercentage:            1,
		MinReputationScore:              50,
		EscrowReleaseDelaySeconds:       3600,
		AuthorizedChannels:              []AuthorizedChannel{},
		NonceRetentionBlocks:            17280,          // ~24 hours at 5 second block time
		CircuitParamHashes:              make(map[string][]byte), // Empty initially, populated on circuit initialization
		ProviderCacheSize:               10,             // Cache top 10 providers
		ProviderCacheRefreshInterval:    100,            // Refresh every 100 blocks (~8 minutes)
		UseProviderCache:                true,           // Enable cache by default
		MaxRequestsPerAddressPerDay:     100,            // Maximum compute requests per address per day
		RequestCooldownBlocks:           10,             // Minimum blocks between requests (~50 seconds at 5s blocks)
		ReputationDecayPercent:          100,            // 100 basis points = 1% decay per day
	}
}

// DefaultGovernanceParams returns default dispute/appeal governance parameters.
func DefaultGovernanceParams() GovernanceParams {
	return GovernanceParams{
		DisputeDeposit:          math.NewInt(1_000_000),
		EvidencePeriodSeconds:   86400,
		VotingPeriodSeconds:     86400,
		QuorumPercentage:        math.LegacyMustNewDecFromStr("0.334"),
		ConsensusThreshold:      math.LegacyMustNewDecFromStr("0.5"),
		SlashPercentage:         math.LegacyMustNewDecFromStr("0.1"),
		AppealDepositPercentage: math.LegacyMustNewDecFromStr("0.05"),
		MaxEvidenceSize:         10 * 1024 * 1024,
	}
}
