package keeper_test

import (
	"testing"

	"cosmossdk.io/math"
	sdk "github.com/cosmos/cosmos-sdk/types"
	"github.com/stretchr/testify/require"

	keepertest "github.com/paw-chain/paw/testutil/keeper"
	"github.com/paw-chain/paw/x/oracle/keeper"
	"github.com/paw-chain/paw/x/oracle/types"
)

// TestEmergencyPause tests the basic emergency pause functionality
func TestEmergencyPause(t *testing.T) {
	k, _, ctx := keepertest.OracleKeeper(t)

	// Initially oracle should not be paused
	require.False(t, k.IsOraclePaused(ctx))

	// Pause the oracle
	pausedBy := "paw1admin"
	reason := "Security incident detected"
	err := k.EmergencyPauseOracle(ctx, pausedBy, reason)
	require.NoError(t, err)

	// Now oracle should be paused
	require.True(t, k.IsOraclePaused(ctx))

	// Get pause state
	pauseState, err := k.GetEmergencyPauseState(ctx)
	require.NoError(t, err)
	require.True(t, pauseState.Paused)
	require.Equal(t, pausedBy, pauseState.PausedBy)
	require.Equal(t, reason, pauseState.PauseReason)
	require.Equal(t, ctx.BlockHeight(), pauseState.PausedAtHeight)

	// Try to pause again - should fail
	err = k.EmergencyPauseOracle(ctx, pausedBy, "Another reason")
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrOraclePaused)
}

// TestResumeOracle tests resuming oracle operations
func TestResumeOracle(t *testing.T) {
	k, _, ctx := keepertest.OracleKeeper(t)

	// Pause the oracle first
	pausedBy := "paw1admin"
	reason := "Testing"
	err := k.EmergencyPauseOracle(ctx, pausedBy, reason)
	require.NoError(t, err)
	require.True(t, k.IsOraclePaused(ctx))

	// Resume the oracle
	resumedBy := "paw1governance"
	resumeReason := "Issue resolved"
	err = k.ResumeOracle(ctx, resumedBy, resumeReason)
	require.NoError(t, err)

	// Oracle should no longer be paused
	require.False(t, k.IsOraclePaused(ctx))

	// Try to resume again - should fail
	err = k.ResumeOracle(ctx, resumedBy, resumeReason)
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrOracleNotPaused)
}

// TestCheckEmergencyPause tests the pause check function
func TestCheckEmergencyPause(t *testing.T) {
	k, _, ctx := keepertest.OracleKeeper(t)

	// Check should pass when not paused
	err := k.CheckEmergencyPause(ctx)
	require.NoError(t, err)

	// Pause the oracle
	err = k.EmergencyPauseOracle(ctx, "admin", "test")
	require.NoError(t, err)

	// Check should fail when paused
	err = k.CheckEmergencyPause(ctx)
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrOraclePaused)
}

// TestPriceSubmissionWhenPaused tests that price submissions are blocked when paused
func TestPriceSubmissionWhenPaused(t *testing.T) {
	k, _, ctx := keepertest.OracleKeeper(t)

	// Pause the oracle
	err := k.EmergencyPauseOracle(ctx, "admin", "emergency test")
	require.NoError(t, err)

	// Now price submission should be blocked via msg_server
	msgServer := keeper.NewMsgServerImpl(*k)

	// Create a dummy validator address for testing
	validatorAddr := sdk.ValAddress([]byte("validator____________"))
	msg := types.NewMsgSubmitPrice(
		validatorAddr.String(),
		sdk.AccAddress(validatorAddr).String(),
		"BTC",
		math.LegacyMustNewDecFromStr("51000.0"),
	)

	_, err = msgServer.SubmitPrice(ctx, msg)
	require.Error(t, err)
	require.Contains(t, err.Error(), "emergency pause check failed")
}

// TestAdminPauseAuthorization tests admin pause authorization
func TestAdminPauseAuthorization(t *testing.T) {
	k, _, ctx := keepertest.OracleKeeper(t)

	// Set emergency admin in params
	params, err := k.GetParams(ctx)
	require.NoError(t, err)

	adminAddr := sdk.AccAddress([]byte("admin_______________"))
	params.EmergencyAdmin = adminAddr.String()
	err = k.SetParams(ctx, params)
	require.NoError(t, err)

	// Create msg server
	msgServer := keeper.NewMsgServerImpl(*k)

	// Admin should be able to pause
	pauseMsg := types.NewMsgEmergencyPauseOracle(
		adminAddr.String(),
		"Admin triggered pause",
	)

	_, err = msgServer.EmergencyPauseOracle(ctx, pauseMsg)
	require.NoError(t, err)
	require.True(t, k.IsOraclePaused(ctx))
}

// TestGovernancePauseAuthorization tests governance pause authorization
func TestGovernancePauseAuthorization(t *testing.T) {
	k, _, ctx := keepertest.OracleKeeper(t)

	// Get the governance authority address
	authority := k.GetAuthority()

	// Create msg server
	msgServer := keeper.NewMsgServerImpl(*k)

	// Governance should be able to pause
	pauseMsg := types.NewMsgEmergencyPauseOracle(
		authority,
		"Governance triggered pause",
	)

	_, err := msgServer.EmergencyPauseOracle(ctx, pauseMsg)
	require.NoError(t, err)
	require.True(t, k.IsOraclePaused(ctx))
}

// TestUnauthorizedPause tests that unauthorized addresses cannot pause
func TestUnauthorizedPause(t *testing.T) {
	k, _, ctx := keepertest.OracleKeeper(t)

	// Create msg server
	msgServer := keeper.NewMsgServerImpl(*k)

	// Random address should not be able to pause
	randomAddr := sdk.AccAddress([]byte("random______________"))
	pauseMsg := types.NewMsgEmergencyPauseOracle(
		randomAddr.String(),
		"Unauthorized pause attempt",
	)

	_, err := msgServer.EmergencyPauseOracle(ctx, pauseMsg)
	require.Error(t, err)
	require.ErrorIs(t, err, types.ErrUnauthorizedPause)
	require.False(t, k.IsOraclePaused(ctx))
}

// TestOnlyGovernanceCanResume tests that only governance can resume
func TestOnlyGovernanceCanResume(t *testing.T) {
	k, _, ctx := keepertest.OracleKeeper(t)

	// Pause first
	err := k.EmergencyPauseOracle(ctx, "admin", "test")
	require.NoError(t, err)

	// Create msg server
	msgServer := keeper.NewMsgServerImpl(*k)

	// Random address should not be able to resume
	randomAddr := sdk.AccAddress([]byte("random______________"))
	resumeMsg := types.NewMsgResumeOracle(
		randomAddr.String(),
		"Unauthorized resume attempt",
	)

	_, err = msgServer.ResumeOracle(ctx, resumeMsg)
	require.Error(t, err)
	require.True(t, k.IsOraclePaused(ctx)) // Still paused

	// Governance should be able to resume
	authority := k.GetAuthority()
	resumeMsg = types.NewMsgResumeOracle(
		authority,
		"Governance approved resume",
	)

	_, err = msgServer.ResumeOracle(ctx, resumeMsg)
	require.NoError(t, err)
	require.False(t, k.IsOraclePaused(ctx)) // No longer paused
}

// TestPauseCycle tests multiple pause/resume cycles
func TestPauseCycle(t *testing.T) {
	k, _, ctx := keepertest.OracleKeeper(t)

	authority := k.GetAuthority()

	// Cycle 1
	err := k.EmergencyPauseOracle(ctx, authority, "cycle 1 pause")
	require.NoError(t, err)
	require.True(t, k.IsOraclePaused(ctx))

	err = k.ResumeOracle(ctx, authority, "cycle 1 resume")
	require.NoError(t, err)
	require.False(t, k.IsOraclePaused(ctx))

	// Cycle 2
	err = k.EmergencyPauseOracle(ctx, authority, "cycle 2 pause")
	require.NoError(t, err)
	require.True(t, k.IsOraclePaused(ctx))

	err = k.ResumeOracle(ctx, authority, "cycle 2 resume")
	require.NoError(t, err)
	require.False(t, k.IsOraclePaused(ctx))

	// Cycle 3
	err = k.EmergencyPauseOracle(ctx, authority, "cycle 3 pause")
	require.NoError(t, err)
	require.True(t, k.IsOraclePaused(ctx))

	err = k.ResumeOracle(ctx, authority, "cycle 3 resume")
	require.NoError(t, err)
	require.False(t, k.IsOraclePaused(ctx))
}

// TestPauseEvents tests that proper events are emitted
func TestPauseEvents(t *testing.T) {
	k, _, ctx := keepertest.OracleKeeper(t)

	authority := k.GetAuthority()

	// Pause
	err := k.EmergencyPauseOracle(ctx, authority, "test pause")
	require.NoError(t, err)

	// Check for pause event
	events := ctx.EventManager().Events()
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
	ctx = ctx.WithEventManager(sdk.NewEventManager())

	// Resume
	err = k.ResumeOracle(ctx, authority, "test resume")
	require.NoError(t, err)

	// Check for resume event
	events = ctx.EventManager().Events()
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
