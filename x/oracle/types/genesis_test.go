package types

import (
	"strings"
	"testing"

	"cosmossdk.io/math"
)

func TestDefaultGenesis(t *testing.T) {
	genesis := DefaultGenesis()

	if genesis == nil {
		t.Fatal("DefaultGenesis returned nil")
	}

	// Test that Params is set to DefaultParams
	expectedParams := DefaultParams()
	if genesis.Params.VotePeriod != expectedParams.VotePeriod {
		t.Errorf("Expected VotePeriod %d, got %d", expectedParams.VotePeriod, genesis.Params.VotePeriod)
	}

	// Test that slices are initialized but empty
	if genesis.Prices == nil {
		t.Error("Prices should be initialized, not nil")
	}
	if len(genesis.Prices) != 0 {
		t.Errorf("Expected empty Prices, got %d entries", len(genesis.Prices))
	}

	if genesis.ValidatorPrices == nil {
		t.Error("ValidatorPrices should be initialized, not nil")
	}
	if len(genesis.ValidatorPrices) != 0 {
		t.Errorf("Expected empty ValidatorPrices, got %d entries", len(genesis.ValidatorPrices))
	}

	if genesis.ValidatorOracles == nil {
		t.Error("ValidatorOracles should be initialized, not nil")
	}
	if len(genesis.ValidatorOracles) != 0 {
		t.Errorf("Expected empty ValidatorOracles, got %d entries", len(genesis.ValidatorOracles))
	}

	if genesis.PriceSnapshots == nil {
		t.Error("PriceSnapshots should be initialized, not nil")
	}
	if len(genesis.PriceSnapshots) != 0 {
		t.Errorf("Expected empty PriceSnapshots, got %d entries", len(genesis.PriceSnapshots))
	}
}

func TestGenesisState_Validate_ValidState(t *testing.T) {
	tests := []struct {
		name string
		gs   GenesisState
	}{
		{
			name: "default genesis",
			gs:   *DefaultGenesis(),
		},
		{
			name: "valid with custom params",
			gs: GenesisState{
				Params: Params{
					VotePeriod:                 60,
					VoteThreshold:              math.LegacyMustNewDecFromStr("0.75"),
					SlashFraction:              math.LegacyMustNewDecFromStr("0.02"),
					SlashWindow:                5000,
					MinValidPerWindow:          50,
					TwapLookbackWindow:         500,
					AuthorizedChannels:         []AuthorizedChannel{},
					AllowedRegions:             []string{"global", "na"},
					MinGeographicRegions:       1,
					MinVotingPowerForConsensus: math.LegacyMustNewDecFromStr("0.15"),
					MaxValidatorsPerIp:         2,
					MaxValidatorsPerAsn:        4,
					RequireGeographicDiversity: false,
					NonceTtlSeconds:            300000,
					DiversityCheckInterval:     50,
					DiversityWarningThreshold:  math.LegacyMustNewDecFromStr("0.30"),
					EnforceRuntimeDiversity:    false,
					EmergencyAdmin:             "",
					GeoipCacheTtlSeconds:       1800,
					GeoipCacheMaxEntries:       500,
				},
				Prices:           []Price{},
				ValidatorPrices:  []ValidatorPrice{},
				ValidatorOracles: []ValidatorOracle{},
				PriceSnapshots:   []PriceSnapshot{},
			},
		},
		{
			name: "valid with authorized channels",
			gs: GenesisState{
				Params: Params{
					VotePeriod:    30,
					VoteThreshold: math.LegacyMustNewDecFromStr("0.67"),
					SlashFraction: math.LegacyMustNewDecFromStr("0.05"),
					SlashWindow:   10000,
					MinValidPerWindow: 100,
					TwapLookbackWindow: 1000,
					AuthorizedChannels: []AuthorizedChannel{
						{PortId: "oracle", ChannelId: "channel-0"},
						{PortId: "oracle", ChannelId: "channel-1"},
					},
					AllowedRegions:             []string{"global"},
					MinGeographicRegions:       1,
					MinVotingPowerForConsensus: math.LegacyMustNewDecFromStr("0.10"),
					MaxValidatorsPerIp:         3,
					MaxValidatorsPerAsn:        5,
					RequireGeographicDiversity: false,
					NonceTtlSeconds:            604800,
					DiversityCheckInterval:     100,
					DiversityWarningThreshold:  math.LegacyMustNewDecFromStr("0.40"),
					EnforceRuntimeDiversity:    false,
					EmergencyAdmin:             "",
					GeoipCacheTtlSeconds:       3600,
					GeoipCacheMaxEntries:       1000,
				},
				Prices:           []Price{},
				ValidatorPrices:  []ValidatorPrice{},
				ValidatorOracles: []ValidatorOracle{},
				PriceSnapshots:   []PriceSnapshot{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.gs.Validate()
			if err != nil {
				t.Errorf("GenesisState.Validate() error = %v, want nil", err)
			}
		})
	}
}

func TestGenesisState_Validate_InvalidVotePeriod(t *testing.T) {
	gs := *DefaultGenesis()
	gs.Params.VotePeriod = 0

	err := gs.Validate()
	if err == nil {
		t.Error("Expected error for zero vote period")
	}
	if !strings.Contains(err.Error(), "vote period must be positive") {
		t.Errorf("Expected error about vote period, got: %v", err)
	}
}

func TestGenesisState_Validate_InvalidVoteThreshold(t *testing.T) {
	tests := []struct {
		name      string
		threshold math.LegacyDec
		wantErr   string
	}{
		{
			name:      "zero threshold",
			threshold: math.LegacyZeroDec(),
			wantErr:   "vote threshold must be in (0,1]",
		},
		{
			name:      "negative threshold",
			threshold: math.LegacyNewDec(-1),
			wantErr:   "vote threshold must be in (0,1]",
		},
		{
			name:      "threshold greater than 1",
			threshold: math.LegacyNewDecWithPrec(15, 1), // 1.5
			wantErr:   "vote threshold must be in (0,1]",
		},
		{
			name:      "threshold exactly 1 (valid)",
			threshold: math.LegacyOneDec(),
			wantErr:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gs := *DefaultGenesis()
			gs.Params.VoteThreshold = tt.threshold

			err := gs.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("Expected error containing %q, got nil", tt.wantErr)
				} else if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("Expected error containing %q, got: %v", tt.wantErr, err)
				}
			}
		})
	}
}

func TestGenesisState_Validate_InvalidSlashFraction(t *testing.T) {
	tests := []struct {
		name     string
		fraction math.LegacyDec
		wantErr  string
	}{
		{
			name:     "negative fraction",
			fraction: math.LegacyNewDec(-1),
			wantErr:  "slash fraction must be between 0 and 1",
		},
		{
			name:     "fraction greater than 1",
			fraction: math.LegacyNewDecWithPrec(15, 1), // 1.5
			wantErr:  "slash fraction must be between 0 and 1",
		},
		{
			name:     "fraction at zero (valid)",
			fraction: math.LegacyZeroDec(),
			wantErr:  "",
		},
		{
			name:     "fraction at 1 (valid)",
			fraction: math.LegacyOneDec(),
			wantErr:  "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gs := *DefaultGenesis()
			gs.Params.SlashFraction = tt.fraction

			err := gs.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("Expected error containing %q, got nil", tt.wantErr)
				} else if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("Expected error containing %q, got: %v", tt.wantErr, err)
				}
			}
		})
	}
}

func TestGenesisState_Validate_InvalidSlashWindow(t *testing.T) {
	tests := []struct {
		name            string
		slashWindow     uint64
		minValidPerWindow uint64
		wantErr         bool
	}{
		{
			name:              "zero slash window",
			slashWindow:       0,
			minValidPerWindow: 100,
			wantErr:           true,
		},
		{
			name:              "zero min valid per window",
			slashWindow:       10000,
			minValidPerWindow: 0,
			wantErr:           true,
		},
		{
			name:              "both zero",
			slashWindow:       0,
			minValidPerWindow: 0,
			wantErr:           true,
		},
		{
			name:              "both valid",
			slashWindow:       10000,
			minValidPerWindow: 100,
			wantErr:           false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gs := *DefaultGenesis()
			gs.Params.SlashWindow = tt.slashWindow
			gs.Params.MinValidPerWindow = tt.minValidPerWindow

			err := gs.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("GenesisState.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.wantErr && err != nil {
				if !strings.Contains(err.Error(), "slash window") && !strings.Contains(err.Error(), "min valid per window") {
					t.Errorf("Expected error about slash window/min valid, got: %v", err)
				}
			}
		})
	}
}

func TestGenesisState_Validate_InvalidTwapLookbackWindow(t *testing.T) {
	gs := *DefaultGenesis()
	gs.Params.TwapLookbackWindow = 0

	err := gs.Validate()
	if err == nil {
		t.Error("Expected error for zero TWAP lookback window")
	}
	if !strings.Contains(err.Error(), "twap lookback window must be positive") {
		t.Errorf("Expected error about TWAP lookback window, got: %v", err)
	}
}

func TestGenesisState_Validate_InvalidAuthorizedChannels(t *testing.T) {
	tests := []struct {
		name     string
		channels []AuthorizedChannel
		wantErr  string
	}{
		{
			name:     "empty port ID",
			channels: []AuthorizedChannel{{PortId: "", ChannelId: "channel-0"}},
			wantErr:  "port_id cannot be empty",
		},
		{
			name:     "whitespace only port ID",
			channels: []AuthorizedChannel{{PortId: "   ", ChannelId: "channel-0"}},
			wantErr:  "port_id cannot be empty",
		},
		{
			name:     "empty channel ID",
			channels: []AuthorizedChannel{{PortId: "oracle", ChannelId: ""}},
			wantErr:  "channel_id cannot be empty",
		},
		{
			name:     "whitespace only channel ID",
			channels: []AuthorizedChannel{{PortId: "oracle", ChannelId: "   "}},
			wantErr:  "channel_id cannot be empty",
		},
		{
			name: "valid channels",
			channels: []AuthorizedChannel{
				{PortId: "oracle", ChannelId: "channel-0"},
				{PortId: "oracle", ChannelId: "channel-1"},
			},
			wantErr: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gs := *DefaultGenesis()
			gs.Params.AuthorizedChannels = tt.channels

			err := gs.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("Expected error containing %q, got nil", tt.wantErr)
				} else if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("Expected error containing %q, got: %v", tt.wantErr, err)
				}
			}
		})
	}
}

func TestGenesisState_Validate_GeographicDiversity(t *testing.T) {
	tests := []struct {
		name                       string
		requireGeographicDiversity bool
		minGeographicRegions       uint64
		allowedRegions             []string
		wantErr                    string
	}{
		{
			name:                       "diversity required but min regions is zero",
			requireGeographicDiversity: true,
			minGeographicRegions:       0,
			allowedRegions:             []string{"na", "eu"},
			wantErr:                    "min_geographic_regions must be positive",
		},
		{
			name:                       "diversity required but allowed regions empty",
			requireGeographicDiversity: true,
			minGeographicRegions:       3,
			allowedRegions:             []string{},
			wantErr:                    "allowed_regions cannot be empty",
		},
		{
			name:                       "min regions exceeds allowed regions",
			requireGeographicDiversity: true,
			minGeographicRegions:       5,
			allowedRegions:             []string{"na", "eu", "apac"},
			wantErr:                    "cannot exceed number of allowed_regions",
		},
		{
			name:                       "valid diversity requirement",
			requireGeographicDiversity: true,
			minGeographicRegions:       3,
			allowedRegions:             []string{"na", "eu", "apac", "latam"},
			wantErr:                    "",
		},
		{
			name:                       "diversity not required - no validation",
			requireGeographicDiversity: false,
			minGeographicRegions:       0,
			allowedRegions:             []string{},
			wantErr:                    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gs := *DefaultGenesis()
			gs.Params.RequireGeographicDiversity = tt.requireGeographicDiversity
			gs.Params.MinGeographicRegions = tt.minGeographicRegions
			gs.Params.AllowedRegions = tt.allowedRegions

			err := gs.Validate()
			if tt.wantErr == "" {
				if err != nil {
					t.Errorf("Expected no error, got: %v", err)
				}
			} else {
				if err == nil {
					t.Errorf("Expected error containing %q, got nil", tt.wantErr)
				} else if !strings.Contains(err.Error(), tt.wantErr) {
					t.Errorf("Expected error containing %q, got: %v", tt.wantErr, err)
				}
			}
		})
	}
}

func TestGenesisState_Validate_BoundaryConditions(t *testing.T) {
	tests := []struct {
		name    string
		modify  func(*GenesisState)
		wantErr bool
	}{
		{
			name: "vote threshold exactly 1",
			modify: func(gs *GenesisState) {
				gs.Params.VoteThreshold = math.LegacyOneDec()
			},
			wantErr: false,
		},
		{
			name: "slash fraction at 0",
			modify: func(gs *GenesisState) {
				gs.Params.SlashFraction = math.LegacyZeroDec()
			},
			wantErr: false,
		},
		{
			name: "slash fraction at 1",
			modify: func(gs *GenesisState) {
				gs.Params.SlashFraction = math.LegacyOneDec()
			},
			wantErr: false,
		},
		{
			name: "min geographic regions equals allowed regions count",
			modify: func(gs *GenesisState) {
				gs.Params.RequireGeographicDiversity = true
				gs.Params.AllowedRegions = []string{"na", "eu", "apac"}
				gs.Params.MinGeographicRegions = 3
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gs := *DefaultGenesis()
			tt.modify(&gs)

			err := gs.Validate()
			if (err != nil) != tt.wantErr {
				t.Errorf("GenesisState.Validate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGenesisState_Validate_MultipleErrors(t *testing.T) {
	// Test that validation stops at first error
	gs := *DefaultGenesis()
	gs.Params.VotePeriod = 0
	gs.Params.VoteThreshold = math.LegacyZeroDec()
	gs.Params.TwapLookbackWindow = 0

	err := gs.Validate()
	if err == nil {
		t.Fatal("Expected error with multiple invalid params")
	}

	// Should fail on first check (vote period)
	if !strings.Contains(err.Error(), "vote period") {
		t.Errorf("Expected first error to be about vote period, got: %v", err)
	}
}
