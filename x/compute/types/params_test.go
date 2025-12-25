package types

import (
	"testing"

	"cosmossdk.io/math"
)

func TestDefaultParams(t *testing.T) {
	params := DefaultParams()

	if !params.MinProviderStake.Equal(math.NewInt(1000000)) {
		t.Errorf("DefaultParams().MinProviderStake = %v, want %v", params.MinProviderStake, math.NewInt(1000000))
	}

	if params.VerificationTimeoutSeconds != 300 {
		t.Errorf("DefaultParams().VerificationTimeoutSeconds = %v, want 300", params.VerificationTimeoutSeconds)
	}

	if params.MaxRequestTimeoutSeconds != 3600 {
		t.Errorf("DefaultParams().MaxRequestTimeoutSeconds = %v, want 3600", params.MaxRequestTimeoutSeconds)
	}

	if params.ReputationSlashPercentage != 10 {
		t.Errorf("DefaultParams().ReputationSlashPercentage = %v, want 10", params.ReputationSlashPercentage)
	}

	if params.StakeSlashPercentage != 1 {
		t.Errorf("DefaultParams().StakeSlashPercentage = %v, want 1", params.StakeSlashPercentage)
	}

	if params.MinReputationScore != 50 {
		t.Errorf("DefaultParams().MinReputationScore = %v, want 50", params.MinReputationScore)
	}

	if params.EscrowReleaseDelaySeconds != 3600 {
		t.Errorf("DefaultParams().EscrowReleaseDelaySeconds = %v, want 3600", params.EscrowReleaseDelaySeconds)
	}

	if params.NonceRetentionBlocks != 17280 {
		t.Errorf("DefaultParams().NonceRetentionBlocks = %v, want 17280", params.NonceRetentionBlocks)
	}

	if params.ProviderCacheSize != 10 {
		t.Errorf("DefaultParams().ProviderCacheSize = %v, want 10", params.ProviderCacheSize)
	}

	if params.ProviderCacheRefreshInterval != 100 {
		t.Errorf("DefaultParams().ProviderCacheRefreshInterval = %v, want 100", params.ProviderCacheRefreshInterval)
	}

	if !params.UseProviderCache {
		t.Error("DefaultParams().UseProviderCache = false, want true")
	}

	if params.MaxRequestsPerAddressPerDay != 100 {
		t.Errorf("DefaultParams().MaxRequestsPerAddressPerDay = %v, want 100", params.MaxRequestsPerAddressPerDay)
	}

	if params.RequestCooldownBlocks != 10 {
		t.Errorf("DefaultParams().RequestCooldownBlocks = %v, want 10", params.RequestCooldownBlocks)
	}

	if params.AuthorizedChannels == nil {
		t.Error("DefaultParams().AuthorizedChannels should be initialized, got nil")
	}

	if params.CircuitParamHashes == nil {
		t.Error("DefaultParams().CircuitParamHashes should be initialized, got nil")
	}
}

func TestDefaultGovernanceParams(t *testing.T) {
	params := DefaultGovernanceParams()

	if !params.DisputeDeposit.Equal(math.NewInt(1_000_000)) {
		t.Errorf("DefaultGovernanceParams().DisputeDeposit = %v, want %v", params.DisputeDeposit, math.NewInt(1_000_000))
	}

	if params.EvidencePeriodSeconds != 86400 {
		t.Errorf("DefaultGovernanceParams().EvidencePeriodSeconds = %v, want 86400", params.EvidencePeriodSeconds)
	}

	if params.VotingPeriodSeconds != 86400 {
		t.Errorf("DefaultGovernanceParams().VotingPeriodSeconds = %v, want 86400", params.VotingPeriodSeconds)
	}

	expectedQuorum := math.LegacyMustNewDecFromStr("0.334")
	if !params.QuorumPercentage.Equal(expectedQuorum) {
		t.Errorf("DefaultGovernanceParams().QuorumPercentage = %v, want %v", params.QuorumPercentage, expectedQuorum)
	}

	expectedConsensus := math.LegacyMustNewDecFromStr("0.5")
	if !params.ConsensusThreshold.Equal(expectedConsensus) {
		t.Errorf("DefaultGovernanceParams().ConsensusThreshold = %v, want %v", params.ConsensusThreshold, expectedConsensus)
	}

	expectedSlash := math.LegacyMustNewDecFromStr("0.1")
	if !params.SlashPercentage.Equal(expectedSlash) {
		t.Errorf("DefaultGovernanceParams().SlashPercentage = %v, want %v", params.SlashPercentage, expectedSlash)
	}

	expectedAppeal := math.LegacyMustNewDecFromStr("0.05")
	if !params.AppealDepositPercentage.Equal(expectedAppeal) {
		t.Errorf("DefaultGovernanceParams().AppealDepositPercentage = %v, want %v", params.AppealDepositPercentage, expectedAppeal)
	}

	if params.MaxEvidenceSize != 10*1024*1024 {
		t.Errorf("DefaultGovernanceParams().MaxEvidenceSize = %v, want %v", params.MaxEvidenceSize, 10*1024*1024)
	}
}
