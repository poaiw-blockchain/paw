package keeper_test

import (
	"testing"

	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	"github.com/paw-chain/paw/x/oracle/types"
)

// TestEmergencyPause tests the basic emergency pause functionality
func TestEmergencyPause(t *testing.T) {
	f := SetupTest(t)

	// Initially oracle should not be paused
	require.False(t, f.oracleKeeper.IsOraclePaused(f.ctx))

	// Pause the oracle
	pausedBy := "paw1admin"
	reason := "Security incident detected"
	err := f.oracleKeeper.EmergencyPauseOracle(f.ctx, pausedBy, reason)
	require.NoError(t, err)

	// Now oracle should be paused
	require.True(t, f.oracleKeeper.IsOraclePaused(f.ctx))

	// Get pause state
	pauseState, err := f.oracleKeeper.GetEmergencyPauseState(f.ctx)
	require.NoError(t, err)
	require.True(t, pauseState.Paused)
	require.Equal(t, pausedBy, pauseState.PausedBy)
	require.Equal(t, reason, pauseState.PauseReason)
	require.Equal(t, f.ctx.BlockHeight(), pauseState.PausedAtHeight)

	// Try to pause again - should fail
	err = f.oracleKeeper.EmergencyPauseOracle(f.ctx, pausedBy, "Another reason")
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrOraclePaused)
}

// TestResumeOracle tests resuming oracle operations
func TestResumeOracle(t *testing.T) {
	f := SetupTest(t)

	// Pause the oracle first
	pausedBy := "paw1admin"
	reason := "Testing"
	err := f.oracleKeeper.EmergencyPauseOracle(f.ctx, pausedBy, reason)
	require.NoError(t, err)
	require.True(t, f.oracleKeeper.IsOraclePaused(f.ctx))

	// Resume the oracle
	resumedBy := "paw1governance"
	resumeReason := "Issue resolved"
	err = f.oracleKeeper.ResumeOracle(f.ctx, resumedBy, resumeReason)
	require.NoError(t, err)

	// Oracle should no longer be paused
	require.False(t, f.oracleKeeper.IsOraclePaused(f.ctx))

	// Try to resume again - should fail
	err = f.oracleKeeper.ResumeOracle(f.ctx, resumedBy, resumeReason)
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrOracleNotPaused)
}

// TestCheckEmergencyPause tests the pause check function
func TestCheckEmergencyPause(t *testing.T) {
	f := SetupTest(t)

	// Check should pass when not paused
	err := f.oracleKeeper.CheckEmergencyPause(f.ctx)
	require.NoError(t, err)

	// Pause the oracle
	err = f.oracleKeeper.EmergencyPauseOracle(f.ctx, "admin", "test")
	require.NoError(t, err)

	// Check should fail when paused
	err = f.oracleKeeper.CheckEmergencyPause(f.ctx)
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrOraclePaused)
}

// TestPriceSubmissionWhenPaused tests that price submissions are blocked when paused
func TestPriceSubmissionWhenPaused(t *testing.T) {
	f := SetupTest(t)

	// Setup validator
	validator := f.createValidator(t, 1)
	validatorAddr := sdk.ValAddress(validator.GetOperator())

	// Set geographic region for validator
	err := f.oracleKeeper.SetValidatorGeographicRegion(f.ctx, validatorAddr, "na", "1.2.3.4", 12345)
	require.NoError(t, err)

	// Initially price submission should work
	err = f.oracleKeeper.SubmitPrice(
		f.ctx,
		validatorAddr,
		"BTC",
		sdk.MustNewDecFromStr("50000.0"),
		sdk.AccAddress(validatorAddr),
	)
	require.NoError(t, err)

	// Pause the oracle
	err = f.oracleKeeper.EmergencyPauseOracle(f.ctx, "admin", "emergency test")
	require.NoError(t, err)

	// Now price submission should be blocked via msg_server
	msgServer := NewMsgServerImpl(f.oracleKeeper)
	msg := types.NewMsgSubmitPrice(
		validatorAddr.String(),
		sdk.AccAddress(validatorAddr).String(),
		"BTC",
		sdk.MustNewDecFromStr("51000.0"),
	)

	_, err = msgServer.SubmitPrice(f.ctx, msg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "emergency pause check failed")
}

// TestAdminPauseAuthorization tests admin pause authorization
func TestAdminPauseAuthorization(t *testing.T) {
	f := SetupTest(t)

	// Set emergency admin in params
	params, err := f.oracleKeeper.GetParams(f.ctx)
	require.NoError(t, err)

	adminAddr := sdk.AccAddress([]byte("admin_______________"))
	params.EmergencyAdmin = adminAddr.String()
	err = f.oracleKeeper.SetParams(f.ctx, params)
	require.NoError(t, err)

	// Create msg server
	msgServer := NewMsgServerImpl(f.oracleKeeper)

	// Admin should be able to pause
	pauseMsg := types.NewMsgEmergencyPauseOracle(
		adminAddr.String(),
		"Admin triggered pause",
	)

	_, err = msgServer.EmergencyPauseOracle(f.ctx, pauseMsg)
	require.NoError(t, err)
	require.True(t, f.oracleKeeper.IsOraclePaused(f.ctx))
}

// TestGovernancePauseAuthorization tests governance pause authorization
func TestGovernancePauseAuthorization(t *testing.T) {
	f := SetupTest(t)

	// Get the governance authority address
	authority := f.oracleKeeper.GetAuthority()

	// Create msg server
	msgServer := NewMsgServerImpl(f.oracleKeeper)

	// Governance should be able to pause
	pauseMsg := types.NewMsgEmergencyPauseOracle(
		authority,
		"Governance triggered pause",
	)

	_, err := msgServer.EmergencyPauseOracle(f.ctx, pauseMsg)
	require.NoError(t, err)
	require.True(t, f.oracleKeeper.IsOraclePaused(f.ctx))
}

// TestUnauthorizedPause tests that unauthorized addresses cannot pause
func TestUnauthorizedPause(t *testing.T) {
	f := SetupTest(t)

	// Create msg server
	msgServer := NewMsgServerImpl(f.oracleKeeper)

	// Random address should not be able to pause
	randomAddr := sdk.AccAddress([]byte("random______________"))
	pauseMsg := types.NewMsgEmergencyPauseOracle(
		randomAddr.String(),
		"Unauthorized pause attempt",
	)

	_, err := msgServer.EmergencyPauseOracle(f.ctx, pauseMsg)
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrUnauthorizedPause)
	require.False(t, f.oracleKeeper.IsOraclePaused(f.ctx))
}

// TestOnlyGovernanceCanResume tests that only governance can resume
func TestOnlyGovernanceCanResume(t *testing.T) {
	f := SetupTest(t)

	// Pause first
	err := f.oracleKeeper.EmergencyPauseOracle(f.ctx, "admin", "test")
	require.NoError(t, err)

	// Create msg server
	msgServer := NewMsgServerImpl(f.oracleKeeper)

	// Random address should not be able to resume
	randomAddr := sdk.AccAddress([]byte("random______________"))
	resumeMsg := types.NewMsgResumeOracle(
		randomAddr.String(),
		"Unauthorized resume attempt",
	)

	_, err = msgServer.ResumeOracle(f.ctx, resumeMsg)
	require.Error(t, err)
	require.True(t, f.oracleKeeper.IsOraclePaused(f.ctx)) // Still paused

	// Governance should be able to resume
	authority := f.oracleKeeper.GetAuthority()
	resumeMsg = types.NewMsgResumeOracle(
		authority,
		"Governance approved resume",
	)

	_, err = msgServer.ResumeOracle(f.ctx, resumeMsg)
	require.NoError(t, err)
	require.False(t, f.oracleKeeper.IsOraclePaused(f.ctx)) // No longer paused
}

// TestPauseCycle tests multiple pause/resume cycles
func TestPauseCycle(t *testing.T) {
	f := SetupTest(t)

	authority := f.oracleKeeper.GetAuthority()

	// Cycle 1
	err := f.oracleKeeper.EmergencyPauseOracle(f.ctx, authority, "cycle 1 pause")
	require.NoError(t, err)
	require.True(t, f.oracleKeeper.IsOraclePaused(f.ctx))

	err = f.oracleKeeper.ResumeOracle(f.ctx, authority, "cycle 1 resume")
	require.NoError(t, err)
	require.False(t, f.oracleKeeper.IsOraclePaused(f.ctx))

	// Cycle 2
	err = f.oracleKeeper.EmergencyPauseOracle(f.ctx, authority, "cycle 2 pause")
	require.NoError(t, err)
	require.True(t, f.oracleKeeper.IsOraclePaused(f.ctx))

	err = f.oracleKeeper.ResumeOracle(f.ctx, authority, "cycle 2 resume")
	require.NoError(t, err)
	require.False(t, f.oracleKeeper.IsOraclePaused(f.ctx))

	// Cycle 3
	err = f.oracleKeeper.EmergencyPauseOracle(f.ctx, authority, "cycle 3 pause")
	require.NoError(t, err)
	require.True(t, f.oracleKeeper.IsOraclePaused(f.ctx))

	err = f.oracleKeeper.ResumeOracle(f.ctx, authority, "cycle 3 resume")
	require.NoError(t, err)
	require.False(t, f.oracleKeeper.IsOraclePaused(f.ctx))
}

// TestPauseEvents tests that proper events are emitted
func TestPauseEvents(t *testing.T) {
	f := SetupTest(t)

	authority := f.oracleKeeper.GetAuthority()

	// Pause
	err := f.oracleKeeper.EmergencyPauseOracle(f.ctx, authority, "test pause")
	require.NoError(t, err)

	// Check for pause event
	events := f.ctx.EventManager().Events()
	found := false
	for _, event := range events {
		if event.Type == types.EventTypeEmergencyPause {
			found = true
			// Verify attributes
			hasActor := false
			hasReason := false
			for _, attr := range event.Attributes {
				if attr.Key == types.AttributeKeyPausedBy && attr.Value == authority {
					hasActor = true
				}
				if attr.Key == types.AttributeKeyPauseReason && attr.Value == "test pause" {
					hasReason = true
				}
			}
			require.True(t, hasActor, "event should have paused_by attribute")
			require.True(t, hasReason, "event should have pause_reason attribute")
			break
		}
	}
	require.True(t, found, "pause event should be emitted")

	// Reset event manager
	f.ctx = f.ctx.WithEventManager(sdk.NewEventManager())

	// Resume
	err = f.oracleKeeper.ResumeOracle(f.ctx, authority, "test resume")
	require.NoError(t, err)

	// Check for resume event
	events = f.ctx.EventManager().Events()
	found = false
	for _, event := range events {
		if event.Type == types.EventTypeEmergencyResume {
			found = true
			// Verify attributes
			hasActor := false
			hasReason := false
			for _, attr := range event.Attributes {
				if attr.Key == types.AttributeKeyActor && attr.Value == authority {
					hasActor = true
				}
				if attr.Key == types.AttributeKeyResumeReason && attr.Value == "test resume" {
					hasReason = true
				}
			}
			require.True(t, hasActor, "event should have actor attribute")
			require.True(t, hasReason, "event should have resume_reason attribute")
			break
		}
	}
	require.True(t, found, "resume event should be emitted")
}
