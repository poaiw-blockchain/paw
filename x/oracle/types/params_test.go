package types

import (
	"testing"

	"cosmossdk.io/math"
)

func TestDefaultParams(t *testing.T) {
	params := DefaultParams()

	// Test basic fields
	if params.VotePeriod != 30 {
		t.Errorf("Expected VotePeriod 30, got %d", params.VotePeriod)
	}

	expectedVoteThreshold := math.LegacyMustNewDecFromStr("0.67")
	if !params.VoteThreshold.Equal(expectedVoteThreshold) {
		t.Errorf("Expected VoteThreshold %s, got %s", expectedVoteThreshold, params.VoteThreshold)
	}

	expectedSlashFraction := math.LegacyMustNewDecFromStr("0.05")
	if !params.SlashFraction.Equal(expectedSlashFraction) {
		t.Errorf("Expected SlashFraction %s, got %s", expectedSlashFraction, params.SlashFraction)
	}

	if params.SlashWindow != 10000 {
		t.Errorf("Expected SlashWindow 10000, got %d", params.SlashWindow)
	}

	if params.MinValidPerWindow != 100 {
		t.Errorf("Expected MinValidPerWindow 100, got %d", params.MinValidPerWindow)
	}

	if params.TwapLookbackWindow != 1000 {
		t.Errorf("Expected TwapLookbackWindow 1000, got %d", params.TwapLookbackWindow)
	}

	// Test geographic diversity fields
	if len(params.AllowedRegions) != 6 {
		t.Errorf("Expected 6 AllowedRegions, got %d", len(params.AllowedRegions))
	}

	expectedRegions := []string{"global", "na", "eu", "apac", "latam", "africa"}
	for i, expected := range expectedRegions {
		if params.AllowedRegions[i] != expected {
			t.Errorf("Expected AllowedRegions[%d] = %s, got %s", i, expected, params.AllowedRegions[i])
		}
	}

	if params.MinGeographicRegions != 1 {
		t.Errorf("Expected MinGeographicRegions 1, got %d", params.MinGeographicRegions)
	}

	expectedMinVotingPower := math.LegacyMustNewDecFromStr("0.10")
	if !params.MinVotingPowerForConsensus.Equal(expectedMinVotingPower) {
		t.Errorf("Expected MinVotingPowerForConsensus %s, got %s", expectedMinVotingPower, params.MinVotingPowerForConsensus)
	}

	if params.MaxValidatorsPerIp != 3 {
		t.Errorf("Expected MaxValidatorsPerIp 3, got %d", params.MaxValidatorsPerIp)
	}

	if params.MaxValidatorsPerAsn != 5 {
		t.Errorf("Expected MaxValidatorsPerAsn 5, got %d", params.MaxValidatorsPerAsn)
	}

	if params.RequireGeographicDiversity != false {
		t.Errorf("Expected RequireGeographicDiversity false, got %v", params.RequireGeographicDiversity)
	}

	if params.NonceTtlSeconds != 604800 {
		t.Errorf("Expected NonceTtlSeconds 604800, got %d", params.NonceTtlSeconds)
	}

	if params.DiversityCheckInterval != 100 {
		t.Errorf("Expected DiversityCheckInterval 100, got %d", params.DiversityCheckInterval)
	}

	expectedDiversityWarning := math.LegacyMustNewDecFromStr("0.40")
	if !params.DiversityWarningThreshold.Equal(expectedDiversityWarning) {
		t.Errorf("Expected DiversityWarningThreshold %s, got %s", expectedDiversityWarning, params.DiversityWarningThreshold)
	}

	if params.EnforceRuntimeDiversity != false {
		t.Errorf("Expected EnforceRuntimeDiversity false, got %v", params.EnforceRuntimeDiversity)
	}

	if params.EmergencyAdmin != "" {
		t.Errorf("Expected EmergencyAdmin empty, got %s", params.EmergencyAdmin)
	}

	if params.GeoipCacheTtlSeconds != 3600 {
		t.Errorf("Expected GeoipCacheTtlSeconds 3600, got %d", params.GeoipCacheTtlSeconds)
	}

	if params.GeoipCacheMaxEntries != 1000 {
		t.Errorf("Expected GeoipCacheMaxEntries 1000, got %d", params.GeoipCacheMaxEntries)
	}

	// Test authorized channels is empty
	if len(params.AuthorizedChannels) != 0 {
		t.Errorf("Expected empty AuthorizedChannels, got %d entries", len(params.AuthorizedChannels))
	}
}

func TestMainnetParams(t *testing.T) {
	params := MainnetParams()

	// Test mainnet-specific changes
	if !params.RequireGeographicDiversity {
		t.Error("Expected RequireGeographicDiversity true for mainnet")
	}

	if params.MinGeographicRegions != 3 {
		t.Errorf("Expected MinGeographicRegions 3 for mainnet, got %d", params.MinGeographicRegions)
	}

	if !params.EnforceRuntimeDiversity {
		t.Error("Expected EnforceRuntimeDiversity true for mainnet")
	}

	expectedSlashFraction := math.LegacyMustNewDecFromStr("0.075")
	if !params.SlashFraction.Equal(expectedSlashFraction) {
		t.Errorf("Expected mainnet SlashFraction %s, got %s", expectedSlashFraction, params.SlashFraction)
	}

	// Test that other params are inherited from DefaultParams
	if params.VotePeriod != 30 {
		t.Errorf("Expected VotePeriod 30 (inherited), got %d", params.VotePeriod)
	}

	expectedVoteThreshold := math.LegacyMustNewDecFromStr("0.67")
	if !params.VoteThreshold.Equal(expectedVoteThreshold) {
		t.Errorf("Expected VoteThreshold %s (inherited), got %s", expectedVoteThreshold, params.VoteThreshold)
	}

	if params.SlashWindow != 10000 {
		t.Errorf("Expected SlashWindow 10000 (inherited), got %d", params.SlashWindow)
	}
}

func TestMainnetParams_DifferencesFromDefault(t *testing.T) {
	defaultParams := DefaultParams()
	mainnetParams := MainnetParams()

	// Verify only specific fields are different
	tests := []struct {
		name         string
		defaultVal   interface{}
		mainnetVal   interface{}
		shouldDiffer bool
	}{
		{"VotePeriod", defaultParams.VotePeriod, mainnetParams.VotePeriod, false},
		{"RequireGeographicDiversity", defaultParams.RequireGeographicDiversity, mainnetParams.RequireGeographicDiversity, true},
		{"MinGeographicRegions", defaultParams.MinGeographicRegions, mainnetParams.MinGeographicRegions, true},
		{"EnforceRuntimeDiversity", defaultParams.EnforceRuntimeDiversity, mainnetParams.EnforceRuntimeDiversity, true},
		{"SlashWindow", defaultParams.SlashWindow, mainnetParams.SlashWindow, false},
		{"TwapLookbackWindow", defaultParams.TwapLookbackWindow, mainnetParams.TwapLookbackWindow, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			differs := tt.defaultVal != tt.mainnetVal
			if differs != tt.shouldDiffer {
				if tt.shouldDiffer {
					t.Errorf("Expected %s to differ between default and mainnet, but they are equal: %v", tt.name, tt.defaultVal)
				} else {
					t.Errorf("Expected %s to be equal between default and mainnet, but default=%v mainnet=%v", tt.name, tt.defaultVal, tt.mainnetVal)
				}
			}
		})
	}

	// Test SlashFraction differs
	if defaultParams.SlashFraction.Equal(mainnetParams.SlashFraction) {
		t.Error("Expected SlashFraction to differ between default and mainnet")
	}

	// Verify mainnet SlashFraction is higher
	if !mainnetParams.SlashFraction.GT(defaultParams.SlashFraction) {
		t.Error("Expected mainnet SlashFraction to be greater than default")
	}
}

func TestParams_VoteThresholdValidRange(t *testing.T) {
	tests := []struct {
		name   string
		params Params
	}{
		{"default params", DefaultParams()},
		{"mainnet params", MainnetParams()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.params.VoteThreshold.LTE(math.LegacyZeroDec()) {
				t.Error("VoteThreshold must be greater than 0")
			}
			if tt.params.VoteThreshold.GT(math.LegacyOneDec()) {
				t.Error("VoteThreshold must be less than or equal to 1")
			}
		})
	}
}

func TestParams_SlashFractionValidRange(t *testing.T) {
	tests := []struct {
		name   string
		params Params
	}{
		{"default params", DefaultParams()},
		{"mainnet params", MainnetParams()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.params.SlashFraction.LT(math.LegacyZeroDec()) {
				t.Error("SlashFraction must be >= 0")
			}
			if tt.params.SlashFraction.GT(math.LegacyOneDec()) {
				t.Error("SlashFraction must be <= 1")
			}
		})
	}
}

func TestParams_NonZeroPositiveFields(t *testing.T) {
	params := DefaultParams()

	if params.VotePeriod == 0 {
		t.Error("VotePeriod must be positive")
	}
	if params.SlashWindow == 0 {
		t.Error("SlashWindow must be positive")
	}
	if params.MinValidPerWindow == 0 {
		t.Error("MinValidPerWindow must be positive")
	}
	if params.TwapLookbackWindow == 0 {
		t.Error("TwapLookbackWindow must be positive")
	}
	if params.NonceTtlSeconds == 0 {
		t.Error("NonceTtlSeconds must be positive")
	}
	if params.DiversityCheckInterval == 0 {
		t.Error("DiversityCheckInterval must be positive")
	}
	if params.GeoipCacheTtlSeconds == 0 {
		t.Error("GeoipCacheTtlSeconds must be positive")
	}
	if params.GeoipCacheMaxEntries == 0 {
		t.Error("GeoipCacheMaxEntries must be positive")
	}
}

func TestParams_MaxValidatorsLimits(t *testing.T) {
	params := DefaultParams()

	if params.MaxValidatorsPerIp == 0 {
		t.Error("MaxValidatorsPerIp must be positive")
	}
	if params.MaxValidatorsPerAsn == 0 {
		t.Error("MaxValidatorsPerAsn must be positive")
	}
	if params.MaxValidatorsPerAsn < params.MaxValidatorsPerIp {
		t.Error("MaxValidatorsPerAsn should be >= MaxValidatorsPerIp (ASN is broader than IP)")
	}
}

func TestParams_DiversityMetrics(t *testing.T) {
	params := DefaultParams()

	if params.DiversityWarningThreshold.LT(math.LegacyZeroDec()) {
		t.Error("DiversityWarningThreshold must be >= 0")
	}
	if params.DiversityWarningThreshold.GT(math.LegacyOneDec()) {
		t.Error("DiversityWarningThreshold must be <= 1")
	}

	if params.MinVotingPowerForConsensus.LT(math.LegacyZeroDec()) {
		t.Error("MinVotingPowerForConsensus must be >= 0")
	}
	if params.MinVotingPowerForConsensus.GT(math.LegacyOneDec()) {
		t.Error("MinVotingPowerForConsensus must be <= 1")
	}
}

func TestParams_AllowedRegionsNotEmpty(t *testing.T) {
	params := DefaultParams()

	if len(params.AllowedRegions) == 0 {
		t.Error("AllowedRegions should not be empty by default")
	}

	// Check for duplicates
	seen := make(map[string]bool)
	for _, region := range params.AllowedRegions {
		if seen[region] {
			t.Errorf("Duplicate region found: %s", region)
		}
		seen[region] = true
	}
}

func TestParams_GeographicDiversityConsistency(t *testing.T) {
	tests := []struct {
		name   string
		params Params
	}{
		{"default params", DefaultParams()},
		{"mainnet params", MainnetParams()},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.params.RequireGeographicDiversity {
				if tt.params.MinGeographicRegions == 0 {
					t.Error("When RequireGeographicDiversity=true, MinGeographicRegions must be > 0")
				}
				if len(tt.params.AllowedRegions) == 0 {
					t.Error("When RequireGeographicDiversity=true, AllowedRegions must not be empty")
				}
				if tt.params.MinGeographicRegions > uint64(len(tt.params.AllowedRegions)) {
					t.Errorf("MinGeographicRegions (%d) cannot exceed AllowedRegions count (%d)",
						tt.params.MinGeographicRegions, len(tt.params.AllowedRegions))
				}
			}
		})
	}
}

func TestParams_MainnetStricterThanDefault(t *testing.T) {
	defaultParams := DefaultParams()
	mainnetParams := MainnetParams()

	// Mainnet should have stricter security
	if !mainnetParams.RequireGeographicDiversity && defaultParams.RequireGeographicDiversity {
		t.Error("Mainnet should require geographic diversity if default does")
	}

	if mainnetParams.MinGeographicRegions < defaultParams.MinGeographicRegions {
		t.Error("Mainnet should require at least as many regions as default")
	}

	if !mainnetParams.SlashFraction.GTE(defaultParams.SlashFraction) {
		t.Error("Mainnet slash fraction should be >= default (stricter)")
	}
}
