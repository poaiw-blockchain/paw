package keeper_test

import (
	"testing"

	sdkmath "cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	stakingkeeper "github.com/cosmos/cosmos-sdk/x/staking/keeper"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/oracle/keeper"
	"github.com/paw-chain/paw/x/oracle/types"
)

// testFixture encapsulates oracle keeper test environment
type testFixture struct {
	ctx           sdk.Context
	oracleKeeper  *keeper.Keeper
	stakingKeeper *stakingkeeper.Keeper
	msgServer     types.MsgServer
}

// TestCheckGeographicDiversityForNewValidator tests runtime diversity checking for new validators
func TestCheckGeographicDiversityForNewValidator(t *testing.T) {
	tests := []struct {
		name              string
		setupValidators   []validatorSetup
		newValidatorSetup validatorSetup
		enforceRuntime    bool
		expectError       bool
		errorContains     string
	}{
		{
			name: "allow new validator in different region - good diversity",
			setupValidators: []validatorSetup{
				{region: "north_america", power: 100},
				{region: "europe", power: 100},
				{region: "asia", power: 100},
			},
			newValidatorSetup: validatorSetup{region: "africa", power: 100},
			enforceRuntime:    true,
			expectError:       false,
		},
		{
			name: "reject new validator - would create regional concentration",
			setupValidators: []validatorSetup{
				{region: "north_america", power: 100},
				{region: "north_america", power: 100},
				{region: "europe", power: 100},
				{region: "asia", power: 100},
			},
			newValidatorSetup: validatorSetup{region: "north_america", power: 100},
			enforceRuntime:    true,
			expectError:       true,
			errorContains:     "would have",
		},
		{
			name: "allow with warning when enforcement disabled",
			setupValidators: []validatorSetup{
				{region: "north_america", power: 100},
				{region: "north_america", power: 100},
				{region: "europe", power: 100},
			},
			newValidatorSetup: validatorSetup{region: "north_america", power: 100},
			enforceRuntime:    false,
			expectError:       false,
		},
		{
			name: "reject when diversity score would drop below threshold",
			setupValidators: []validatorSetup{
				{region: "north_america", power: 100},
				{region: "north_america", power: 100},
				{region: "north_america", power: 100},
				{region: "europe", power: 100},
			},
			newValidatorSetup: validatorSetup{region: "north_america", power: 100},
			enforceRuntime:    true,
			expectError:       true,
			errorContains:     "diversity score",
		},
		{
			name: "allow when total validators < 5",
			setupValidators: []validatorSetup{
				{region: "north_america", power: 100},
				{region: "north_america", power: 100},
			},
			newValidatorSetup: validatorSetup{region: "north_america", power: 100},
			enforceRuntime:    true,
			expectError:       false,
		},
		{
			name: "allow when diversity not required",
			setupValidators: []validatorSetup{
				{region: "north_america", power: 100},
				{region: "north_america", power: 100},
				{region: "north_america", power: 100},
			},
			newValidatorSetup: validatorSetup{region: "north_america", power: 100},
			enforceRuntime:    true,
			expectError:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k, sk, ctx := keepertest.OracleKeeper(t)
			f := testFixture{
				ctx:           ctx,
				oracleKeeper:  k,
				stakingKeeper: sk,
				msgServer:     keeper.NewMsgServerImpl(*k),
			}

			// Set params with appropriate enforcement
			params := types.DefaultParams()
			params.RequireGeographicDiversity = tt.name != "allow when diversity not required"
			params.EnforceRuntimeDiversity = tt.enforceRuntime
			params.DiversityWarningThreshold = sdkmath.LegacyMustNewDecFromStr("0.40")
			err := f.oracleKeeper.SetParams(f.ctx, params)
			require.NoError(t, err)

			// Setup existing validators
			for i, vs := range tt.setupValidators {
				valAddr := createTestValidatorWithPower(t, f, vs.power)
				oracle := types.ValidatorOracle{
					ValidatorAddr:    valAddr.String(),
					GeographicRegion: vs.region,
					MissCounter:      0,
					TotalSubmissions: uint64(i + 1), // Ensure non-zero
					IsActive:         true,
				}
				err := f.oracleKeeper.SetValidatorOracle(f.ctx, oracle)
				require.NoError(t, err)
			}

			// Test checking diversity for new validator
			err = f.oracleKeeper.CheckGeographicDiversityForNewValidator(f.ctx, tt.newValidatorSetup.region)

			if tt.expectError {
				require.Error(t, err)
				if tt.errorContains != "" {
					require.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestMonitorGeographicDiversity tests periodic diversity monitoring in BeginBlocker
func TestMonitorGeographicDiversity(t *testing.T) {
	tests := []struct {
		name            string
		validators      []validatorSetup
		expectedEvents  int // Number of warning/critical events expected
		expectError     bool
	}{
		{
			name: "good diversity - no warnings",
			validators: []validatorSetup{
				{region: "north_america", power: 100},
				{region: "europe", power: 100},
				{region: "asia", power: 100},
				{region: "africa", power: 100},
			},
			expectedEvents: 0,
			expectError:    false,
		},
		{
			name: "low diversity score - emit warning",
			validators: []validatorSetup{
				{region: "north_america", power: 100},
				{region: "north_america", power: 100},
				{region: "north_america", power: 100},
				{region: "europe", power: 100},
			},
			expectedEvents: 1, // diversity_warning event
			expectError:    false,
		},
		{
			name: "insufficient regions - emit critical",
			validators: []validatorSetup{
				{region: "north_america", power: 100},
				{region: "north_america", power: 100},
			},
			expectedEvents: 1, // geographic_diversity_critical event
			expectError:    false,
		},
		{
			name: "regional concentration - emit concentration warning",
			validators: []validatorSetup{
				{region: "north_america", power: 100},
				{region: "north_america", power: 100},
				{region: "north_america", power: 100},
				{region: "europe", power: 100},
				{region: "asia", power: 100},
			},
			expectedEvents: 1, // geographic_concentration_warning event
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k, sk, ctx := keepertest.OracleKeeper(t)
			f := testFixture{
				ctx:           ctx,
				oracleKeeper:  k,
				stakingKeeper: sk,
				msgServer:     keeper.NewMsgServerImpl(*k),
			}

			// Set params requiring geographic diversity
			params := types.DefaultParams()
			params.RequireGeographicDiversity = true
			params.MinGeographicRegions = 3
			params.DiversityWarningThreshold = sdkmath.LegacyMustNewDecFromStr("0.40")
			err := f.oracleKeeper.SetParams(f.ctx, params)
			require.NoError(t, err)

			// Setup validators
			for i, vs := range tt.validators {
				valAddr := createTestValidatorWithPower(t, f, vs.power)
				oracle := types.ValidatorOracle{
					ValidatorAddr:    valAddr.String(),
					GeographicRegion: vs.region,
					MissCounter:      0,
					TotalSubmissions: uint64(i + 1),
					IsActive:         true,
				}
				err := f.oracleKeeper.SetValidatorOracle(f.ctx, oracle)
				require.NoError(t, err)
			}

			// Clear any events from setup
			f.ctx = f.ctx.WithEventManager(sdk.NewEventManager())

			// Run monitoring
			err = f.oracleKeeper.MonitorGeographicDiversity(f.ctx)

			if tt.expectError {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}

			// Check events
			events := f.ctx.EventManager().Events()
			warningEvents := 0
			for _, event := range events {
				if event.Type == "geographic_diversity_warning" ||
					event.Type == "geographic_diversity_critical" ||
					event.Type == "geographic_concentration_warning" {
					warningEvents++
				}
			}

			if tt.expectedEvents > 0 {
				require.GreaterOrEqual(t, warningEvents, tt.expectedEvents,
					"expected at least %d warning events, got %d", tt.expectedEvents, warningEvents)
			}

			// Always expect at least one status event
			statusEvents := 0
			for _, event := range events {
				if event.Type == "geographic_diversity_status" {
					statusEvents++
				}
			}
			require.Equal(t, 1, statusEvents, "expected exactly 1 status event")
		})
	}
}

// TestSubmitPriceWithGeographicDiversityCheck tests that SubmitPrice enforces diversity on first submission
func TestSubmitPriceWithGeographicDiversityCheck(t *testing.T) {
	tests := []struct {
		name              string
		existingValidators []validatorSetup
		newValidatorRegion string
		enforceRuntime     bool
		expectReject       bool
	}{
		{
			name: "accept first submission with good diversity",
			existingValidators: []validatorSetup{
				{region: "north_america", power: 100},
				{region: "europe", power: 100},
			},
			newValidatorRegion: "asia",
			enforceRuntime:     true,
			expectReject:       false,
		},
		{
			name: "reject first submission - would violate diversity",
			existingValidators: []validatorSetup{
				{region: "north_america", power: 100},
				{region: "north_america", power: 100},
				{region: "north_america", power: 100},
				{region: "europe", power: 100},
			},
			newValidatorRegion: "north_america",
			enforceRuntime:     true,
			expectReject:       true,
		},
		{
			name: "accept with warning when enforcement disabled",
			existingValidators: []validatorSetup{
				{region: "north_america", power: 100},
				{region: "north_america", power: 100},
			},
			newValidatorRegion: "north_america",
			enforceRuntime:     false,
			expectReject:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k, sk, ctx := keepertest.OracleKeeper(t)
			f := testFixture{
				ctx:           ctx,
				oracleKeeper:  k,
				stakingKeeper: sk,
				msgServer:     keeper.NewMsgServerImpl(*k),
			}

			// Set params
			params := types.DefaultParams()
			params.RequireGeographicDiversity = true
			params.EnforceRuntimeDiversity = tt.enforceRuntime
			params.DiversityWarningThreshold = sdkmath.LegacyMustNewDecFromStr("0.40")
			err := f.oracleKeeper.SetParams(f.ctx, params)
			require.NoError(t, err)

			// Setup existing validators
			for i, vs := range tt.existingValidators {
				valAddr := createTestValidatorWithPower(t, f, vs.power)
				oracle := types.ValidatorOracle{
					ValidatorAddr:    valAddr.String(),
					GeographicRegion: vs.region,
					MissCounter:      0,
					TotalSubmissions: uint64(i + 1),
					IsActive:         true,
				}
				err := f.oracleKeeper.SetValidatorOracle(f.ctx, oracle)
				require.NoError(t, err)
			}

			// Create new validator with region
			newValAddr := createTestValidatorWithPower(t, f, 100)
			newOracle := types.ValidatorOracle{
				ValidatorAddr:    newValAddr.String(),
				GeographicRegion: tt.newValidatorRegion,
				MissCounter:      0,
				TotalSubmissions: 0, // First submission
				IsActive:         true,
			}
			err = f.oracleKeeper.SetValidatorOracle(f.ctx, newOracle)
			require.NoError(t, err)

			// Create feeder account
			feederAddr := sdk.AccAddress(newValAddr)

			// Attempt price submission
			msg := &types.MsgSubmitPrice{
				Validator: newValAddr.String(),
				Feeder:    feederAddr.String(),
				Asset:     "BTC",
				Price:     sdkmath.LegacyNewDec(50000),
			}

			msgServer := f.msgServer
			_, err = msgServer.SubmitPrice(f.ctx, msg)

			if tt.expectReject {
				require.Error(t, err)
				require.Contains(t, err.Error(), "geographic diversity")
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// TestBeginBlockerDiversityCheck tests that BeginBlocker periodically checks diversity
func TestBeginBlockerDiversityCheck(t *testing.T) {
	k, sk, ctx := keepertest.OracleKeeper(t)
	f := testFixture{
		ctx:           ctx,
		oracleKeeper:  k,
		stakingKeeper: sk,
		msgServer:     keeper.NewMsgServerImpl(*k),
	}

	// Set params with check interval
	params := types.DefaultParams()
	params.RequireGeographicDiversity = true
	params.DiversityCheckInterval = 10 // Check every 10 blocks
	params.MinGeographicRegions = 3
	params.DiversityWarningThreshold = sdkmath.LegacyMustNewDecFromStr("0.40")
	err := f.oracleKeeper.SetParams(f.ctx, params)
	require.NoError(t, err)

	// Setup validators with poor diversity
	validators := []validatorSetup{
		{region: "north_america", power: 100},
		{region: "north_america", power: 100},
		{region: "north_america", power: 100},
		{region: "europe", power: 100},
	}
	for i, vs := range validators {
		valAddr := createTestValidatorWithPower(t, f, vs.power)
		oracle := types.ValidatorOracle{
			ValidatorAddr:    valAddr.String(),
			GeographicRegion: vs.region,
			MissCounter:      0,
			TotalSubmissions: uint64(i + 1),
			IsActive:         true,
		}
		err := f.oracleKeeper.SetValidatorOracle(f.ctx, oracle)
		require.NoError(t, err)
	}

	// Test multiple blocks
	testBlocks := []struct {
		height        int64
		expectCheck   bool
		expectWarning bool
	}{
		{height: 5, expectCheck: false, expectWarning: false},   // Not a check block
		{height: 10, expectCheck: true, expectWarning: true},    // Check block - should warn
		{height: 15, expectCheck: false, expectWarning: false},  // Not a check block
		{height: 20, expectCheck: true, expectWarning: true},    // Check block - should warn
	}

	for _, tb := range testBlocks {
		t.Run("block_"+string(rune(tb.height)), func(t *testing.T) {
			// Update block height
			f.ctx = f.ctx.WithBlockHeight(tb.height)
			f.ctx = f.ctx.WithEventManager(sdk.NewEventManager())

			// Run BeginBlocker
			err := f.oracleKeeper.BeginBlocker(f.ctx)
			require.NoError(t, err)

			// Check if diversity monitoring ran
			events := f.ctx.EventManager().Events()
			hasStatusEvent := false
			hasWarningEvent := false

			for _, event := range events {
				if event.Type == "geographic_diversity_status" {
					hasStatusEvent = true
				}
				if event.Type == "geographic_diversity_warning" ||
					event.Type == "geographic_diversity_critical" {
					hasWarningEvent = true
				}
			}

			if tb.expectCheck {
				require.True(t, hasStatusEvent, "expected diversity status event at block %d", tb.height)
			} else {
				require.False(t, hasStatusEvent, "unexpected diversity check at block %d", tb.height)
			}

			if tb.expectWarning {
				require.True(t, hasWarningEvent, "expected warning event at block %d", tb.height)
			}
		})
	}
}

// TestParamValidation tests validation of new diversity parameters
func TestParamValidation(t *testing.T) {
	tests := []struct {
		name          string
		modifyParams  func(*types.Params)
		expectError   bool
		errorContains string
	}{
		{
			name: "valid diversity params",
			modifyParams: func(p *types.Params) {
				p.DiversityCheckInterval = 100
				p.DiversityWarningThreshold = sdkmath.LegacyMustNewDecFromStr("0.40")
				p.EnforceRuntimeDiversity = true
			},
			expectError: false,
		},
		{
			name: "diversity check interval zero (disabled)",
			modifyParams: func(p *types.Params) {
				p.DiversityCheckInterval = 0
			},
			expectError: false,
		},
		{
			name: "invalid diversity threshold - negative",
			modifyParams: func(p *types.Params) {
				p.DiversityWarningThreshold = sdkmath.LegacyMustNewDecFromStr("-0.1")
			},
			expectError:   true,
			errorContains: "diversity warning threshold must be between 0 and 1",
		},
		{
			name: "invalid diversity threshold - > 1",
			modifyParams: func(p *types.Params) {
				p.DiversityWarningThreshold = sdkmath.LegacyMustNewDecFromStr("1.5")
			},
			expectError:   true,
			errorContains: "diversity warning threshold must be between 0 and 1",
		},
		{
			name: "valid edge case - threshold at 0",
			modifyParams: func(p *types.Params) {
				p.DiversityWarningThreshold = sdkmath.LegacyZeroDec()
			},
			expectError: false,
		},
		{
			name: "valid edge case - threshold at 1",
			modifyParams: func(p *types.Params) {
				p.DiversityWarningThreshold = sdkmath.LegacyOneDec()
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			k, sk, ctx := keepertest.OracleKeeper(t)
			f := testFixture{
				ctx:           ctx,
				oracleKeeper:  k,
				stakingKeeper: sk,
				msgServer:     keeper.NewMsgServerImpl(*k),
			}

			params := types.DefaultParams()
			tt.modifyParams(&params)

			msg := &types.MsgUpdateParams{
				Authority: f.oracleKeeper.GetAuthority(),
				Params:    params,
			}

			msgServer := f.msgServer
			_, err := msgServer.UpdateParams(f.ctx, msg)

			if tt.expectError {
				require.Error(t, err)
				if tt.errorContains != "" {
					require.Contains(t, err.Error(), tt.errorContains)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

// validatorSetup represents test validator configuration
type validatorSetup struct {
	region string
	power  int64
}

// validatorCounter is used to generate unique validator addresses
var validatorCounter int64

// createTestValidatorWithPower creates a test validator with the given power in the staking keeper
func createTestValidatorWithPower(t *testing.T, f testFixture, power int64) sdk.ValAddress {
	t.Helper()
	// Create unique validator address for testing
	validatorCounter++
	addrBytes := make([]byte, 20)
	copy(addrBytes, []byte("test_validator_"))
	addrBytes[15] = byte(validatorCounter % 256)
	addrBytes[16] = byte((validatorCounter / 256) % 256)
	addrBytes[17] = byte(power % 256)
	addrBytes[18] = byte((power / 256) % 256)
	valAddr := sdk.ValAddress(addrBytes)

	// Create the validator in the staking keeper
	err := keepertest.EnsureBondedValidatorWithKeeper(f.ctx, f.stakingKeeper, valAddr)
	require.NoError(t, err)

	return valAddr
}
