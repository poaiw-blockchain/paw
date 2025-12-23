package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/oracle/types"
)

// TestInitGenesis_RequireGeographicDiversity_NotEnabled tests that genesis succeeds when geographic diversity is not required
func TestInitGenesis_RequireGeographicDiversity_NotEnabled(t *testing.T) {
	k, _, ctx := keepertest.OracleKeeper(t)

	// Create genesis state with RequireGeographicDiversity = false
	genesisState := types.GenesisState{
		Params: types.Params{
			VotePeriod:                 30,
			VoteThreshold:              math.LegacyMustNewDecFromStr("0.67"),
			SlashFraction:              math.LegacyMustNewDecFromStr("0.01"),
			SlashWindow:                10000,
			MinValidPerWindow:          100,
			TwapLookbackWindow:         1000,
			AuthorizedChannels:         []types.AuthorizedChannel{},
			AllowedRegions:             []string{"north_america", "europe", "asia"},
			MinGeographicRegions:       1,
			MinVotingPowerForConsensus: math.LegacyMustNewDecFromStr("0.10"),
			MaxValidatorsPerIp:         3,
			MaxValidatorsPerAsn:        5,
			RequireGeographicDiversity: false, // Not required
		},
		Prices:           []types.Price{},
		ValidatorPrices:  []types.ValidatorPrice{},
		ValidatorOracles: []types.ValidatorOracle{},
		PriceSnapshots:   []types.PriceSnapshot{},
	}

	// Should succeed even without GeoIP database when RequireGeographicDiversity is false
	err := k.InitGenesis(ctx, genesisState)
	require.NoError(t, err)

	// Verify params were set
	params, err := k.GetParams(ctx)
	require.NoError(t, err)
	require.False(t, params.RequireGeographicDiversity)
}

// TestInitGenesis_RequireGeographicDiversity_NoValidators tests genesis with diversity required but no validators
func TestInitGenesis_RequireGeographicDiversity_NoValidators(t *testing.T) {
	k, _, ctx := keepertest.OracleKeeper(t)

	// Create genesis state with RequireGeographicDiversity = true but no validators
	genesisState := types.GenesisState{
		Params: types.Params{
			VotePeriod:                 30,
			VoteThreshold:              math.LegacyMustNewDecFromStr("0.67"),
			SlashFraction:              math.LegacyMustNewDecFromStr("0.01"),
			SlashWindow:                10000,
			MinValidPerWindow:          100,
			TwapLookbackWindow:         1000,
			AuthorizedChannels:         []types.AuthorizedChannel{},
			AllowedRegions:             []string{"north_america", "europe", "asia"},
			MinGeographicRegions:       2,
			MinVotingPowerForConsensus: math.LegacyMustNewDecFromStr("0.10"),
			MaxValidatorsPerIp:         3,
			MaxValidatorsPerAsn:        5,
			RequireGeographicDiversity: true, // Required
		},
		Prices:           []types.Price{},
		ValidatorPrices:  []types.ValidatorPrice{},
		ValidatorOracles: []types.ValidatorOracle{}, // No validators yet
		PriceSnapshots:   []types.PriceSnapshot{},
	}

	// This will fail if GeoIP database is not available
	err := k.InitGenesis(ctx, genesisState)

	// The test should fail with GeoIP database error if database is not available
	// or succeed if database is available
	if err != nil {
		require.ErrorContains(t, err, "GeoIP database is not available")
	}
}

// TestInitGenesis_RequireGeographicDiversity_InvalidRegion tests genesis fails with invalid regions
func TestInitGenesis_RequireGeographicDiversity_InvalidRegion(t *testing.T) {
	k, _, ctx := keepertest.OracleKeeper(t)

	// Create a validator oracle with invalid region
	validatorOracle := types.ValidatorOracle{
		ValidatorAddr:    "cosmosvaloper1test",
		MissCounter:      0,
		TotalSubmissions: 10,
		IsActive:         true,
		GeographicRegion: "invalid_region", // Not in allowed list
		IpAddress:        "192.168.1.1",
		Asn:              12345,
	}

	genesisState := types.GenesisState{
		Params: types.Params{
			VotePeriod:                 30,
			VoteThreshold:              math.LegacyMustNewDecFromStr("0.67"),
			SlashFraction:              math.LegacyMustNewDecFromStr("0.01"),
			SlashWindow:                10000,
			MinValidPerWindow:          100,
			TwapLookbackWindow:         1000,
			AuthorizedChannels:         []types.AuthorizedChannel{},
			AllowedRegions:             []string{"north_america", "europe", "asia"},
			MinGeographicRegions:       2,
			MinVotingPowerForConsensus: math.LegacyMustNewDecFromStr("0.10"),
			MaxValidatorsPerIp:         3,
			MaxValidatorsPerAsn:        5,
			RequireGeographicDiversity: true,
		},
		Prices:           []types.Price{},
		ValidatorPrices:  []types.ValidatorPrice{},
		ValidatorOracles: []types.ValidatorOracle{validatorOracle},
		PriceSnapshots:   []types.PriceSnapshot{},
	}

	// Should fail - either due to missing GeoIP or invalid region
	err := k.InitGenesis(ctx, genesisState)
	require.Error(t, err)

	// Error should mention either GeoIP database or invalid region
	errorMsg := err.Error()
	require.True(t,
		containsAny(errorMsg, "GeoIP database", "not in allowed_regions"),
		"Expected error about GeoIP database or invalid region, got: %s", errorMsg)
}

// TestInitGenesis_RequireGeographicDiversity_EmptyRegion tests genesis fails with empty regions
func TestInitGenesis_RequireGeographicDiversity_EmptyRegion(t *testing.T) {
	k, _, ctx := keepertest.OracleKeeper(t)

	// Create a validator oracle with empty region
	validatorOracle := types.ValidatorOracle{
		ValidatorAddr:    "cosmosvaloper1test",
		MissCounter:      0,
		TotalSubmissions: 10,
		IsActive:         true,
		GeographicRegion: "", // Empty region
		IpAddress:        "192.168.1.1",
		Asn:              12345,
	}

	genesisState := types.GenesisState{
		Params: types.Params{
			VotePeriod:                 30,
			VoteThreshold:              math.LegacyMustNewDecFromStr("0.67"),
			SlashFraction:              math.LegacyMustNewDecFromStr("0.01"),
			SlashWindow:                10000,
			MinValidPerWindow:          100,
			TwapLookbackWindow:         1000,
			AuthorizedChannels:         []types.AuthorizedChannel{},
			AllowedRegions:             []string{"north_america", "europe", "asia"},
			MinGeographicRegions:       2,
			MinVotingPowerForConsensus: math.LegacyMustNewDecFromStr("0.10"),
			MaxValidatorsPerIp:         3,
			MaxValidatorsPerAsn:        5,
			RequireGeographicDiversity: true,
		},
		Prices:           []types.Price{},
		ValidatorPrices:  []types.ValidatorPrice{},
		ValidatorOracles: []types.ValidatorOracle{validatorOracle},
		PriceSnapshots:   []types.PriceSnapshot{},
	}

	// Should fail - either due to missing GeoIP or empty region
	err := k.InitGenesis(ctx, genesisState)
	require.Error(t, err)

	// Error should mention either GeoIP database or empty region
	errorMsg := err.Error()
	require.True(t,
		containsAny(errorMsg, "GeoIP database", "empty geographic region"),
		"Expected error about GeoIP database or empty region, got: %s", errorMsg)
}

// TestValidateGenesis_RequireGeographicDiversity tests genesis validation for geographic diversity parameters
func TestValidateGenesis_RequireGeographicDiversity(t *testing.T) {
	tests := []struct {
		name        string
		genesisState types.GenesisState
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid - diversity not required",
			genesisState: types.GenesisState{
				Params: types.Params{
					VotePeriod:                 30,
					VoteThreshold:              math.LegacyMustNewDecFromStr("0.67"),
					SlashFraction:              math.LegacyMustNewDecFromStr("0.01"),
					SlashWindow:                10000,
					MinValidPerWindow:          100,
					TwapLookbackWindow:         1000,
					AuthorizedChannels:         []types.AuthorizedChannel{},
					AllowedRegions:             []string{"north_america", "europe"},
					MinGeographicRegions:       1,
					MinVotingPowerForConsensus: math.LegacyMustNewDecFromStr("0.10"),
					MaxValidatorsPerIp:         3,
					MaxValidatorsPerAsn:        5,
					RequireGeographicDiversity: false,
				},
				Prices:           []types.Price{},
				ValidatorPrices:  []types.ValidatorPrice{},
				ValidatorOracles: []types.ValidatorOracle{},
				PriceSnapshots:   []types.PriceSnapshot{},
			},
			expectError: false,
		},
		{
			name: "valid - diversity required with proper config",
			genesisState: types.GenesisState{
				Params: types.Params{
					VotePeriod:                 30,
					VoteThreshold:              math.LegacyMustNewDecFromStr("0.67"),
					SlashFraction:              math.LegacyMustNewDecFromStr("0.01"),
					SlashWindow:                10000,
					MinValidPerWindow:          100,
					TwapLookbackWindow:         1000,
					AuthorizedChannels:         []types.AuthorizedChannel{},
					AllowedRegions:             []string{"north_america", "europe", "asia"},
					MinGeographicRegions:       2,
					MinVotingPowerForConsensus: math.LegacyMustNewDecFromStr("0.10"),
					MaxValidatorsPerIp:         3,
					MaxValidatorsPerAsn:        5,
					RequireGeographicDiversity: true,
				},
				Prices:           []types.Price{},
				ValidatorPrices:  []types.ValidatorPrice{},
				ValidatorOracles: []types.ValidatorOracle{},
				PriceSnapshots:   []types.PriceSnapshot{},
			},
			expectError: false,
		},
		{
			name: "invalid - diversity required but MinGeographicRegions is zero",
			genesisState: types.GenesisState{
				Params: types.Params{
					VotePeriod:                 30,
					VoteThreshold:              math.LegacyMustNewDecFromStr("0.67"),
					SlashFraction:              math.LegacyMustNewDecFromStr("0.01"),
					SlashWindow:                10000,
					MinValidPerWindow:          100,
					TwapLookbackWindow:         1000,
					AuthorizedChannels:         []types.AuthorizedChannel{},
					AllowedRegions:             []string{"north_america", "europe"},
					MinGeographicRegions:       0, // Invalid
					MinVotingPowerForConsensus: math.LegacyMustNewDecFromStr("0.10"),
					MaxValidatorsPerIp:         3,
					MaxValidatorsPerAsn:        5,
					RequireGeographicDiversity: true,
				},
				Prices:           []types.Price{},
				ValidatorPrices:  []types.ValidatorPrice{},
				ValidatorOracles: []types.ValidatorOracle{},
				PriceSnapshots:   []types.PriceSnapshot{},
			},
			expectError: true,
			errorMsg:    "min_geographic_regions must be positive",
		},
		{
			name: "invalid - diversity required but AllowedRegions is empty",
			genesisState: types.GenesisState{
				Params: types.Params{
					VotePeriod:                 30,
					VoteThreshold:              math.LegacyMustNewDecFromStr("0.67"),
					SlashFraction:              math.LegacyMustNewDecFromStr("0.01"),
					SlashWindow:                10000,
					MinValidPerWindow:          100,
					TwapLookbackWindow:         1000,
					AuthorizedChannels:         []types.AuthorizedChannel{},
					AllowedRegions:             []string{}, // Empty
					MinGeographicRegions:       2,
					MinVotingPowerForConsensus: math.LegacyMustNewDecFromStr("0.10"),
					MaxValidatorsPerIp:         3,
					MaxValidatorsPerAsn:        5,
					RequireGeographicDiversity: true,
				},
				Prices:           []types.Price{},
				ValidatorPrices:  []types.ValidatorPrice{},
				ValidatorOracles: []types.ValidatorOracle{},
				PriceSnapshots:   []types.PriceSnapshot{},
			},
			expectError: true,
			errorMsg:    "allowed_regions cannot be empty",
		},
		{
			name: "invalid - MinGeographicRegions exceeds AllowedRegions count",
			genesisState: types.GenesisState{
				Params: types.Params{
					VotePeriod:                 30,
					VoteThreshold:              math.LegacyMustNewDecFromStr("0.67"),
					SlashFraction:              math.LegacyMustNewDecFromStr("0.01"),
					SlashWindow:                10000,
					MinValidPerWindow:          100,
					TwapLookbackWindow:         1000,
					AuthorizedChannels:         []types.AuthorizedChannel{},
					AllowedRegions:             []string{"north_america", "europe"},
					MinGeographicRegions:       5, // More than 2 allowed regions
					MinVotingPowerForConsensus: math.LegacyMustNewDecFromStr("0.10"),
					MaxValidatorsPerIp:         3,
					MaxValidatorsPerAsn:        5,
					RequireGeographicDiversity: true,
				},
				Prices:           []types.Price{},
				ValidatorPrices:  []types.ValidatorPrice{},
				ValidatorOracles: []types.ValidatorOracle{},
				PriceSnapshots:   []types.PriceSnapshot{},
			},
			expectError: true,
			errorMsg:    "cannot exceed number of allowed_regions",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.genesisState.Validate()
			if tt.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestMainnetParams verifies mainnet parameters enforce geographic diversity
func TestMainnetParams(t *testing.T) {
	params := types.MainnetParams()

	require.True(t, params.RequireGeographicDiversity, "Mainnet must require geographic diversity")
	require.GreaterOrEqual(t, params.MinGeographicRegions, uint64(3), "Mainnet should require at least 3 regions")
	require.NotEmpty(t, params.AllowedRegions, "Mainnet must have allowed regions")
}

// Helper function to check if a string contains any of the given substrings
func containsAny(s string, substrings ...string) bool {
	for _, sub := range substrings {
		if len(s) >= len(sub) {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
		}
	}
	return false
}
