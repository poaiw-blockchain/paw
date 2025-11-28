package types

import (
	"cosmossdk.io/math"
)

// DefaultParams returns default compute parameters
func DefaultParams() Params {
	return Params{
		MinProviderStake:           math.NewInt(1000000), // 1 PAW
		VerificationTimeoutSeconds: 300,
		MaxRequestTimeoutSeconds:   3600,
		ReputationSlashPercentage:  10,
		StakeSlashPercentage:       1,
		MinReputationScore:         50,
		EscrowReleaseDelaySeconds:  3600,
	}
}
