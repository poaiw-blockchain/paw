package tests

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/paw-chain/paw/control-center/network-controls/circuit"
)

func TestCircuitBreakerRegistry(t *testing.T) {
	t.Parallel()

	t.Run("Register circuit breaker", func(t *testing.T) {
		registry := circuit.NewCircuitBreakerRegistry()

		err := registry.Register("dex", "")
		require.NoError(t, err)

		state, err := registry.GetState("dex", "")
		require.NoError(t, err)
		require.Equal(t, circuit.StatusClosed, state.Status)
	})

	t.Run("Duplicate registration fails", func(t *testing.T) {
		registry := circuit.NewCircuitBreakerRegistry()

		err := registry.Register("dex", "")
		require.NoError(t, err)

		err = registry.Register("dex", "")
		require.Error(t, err)
	})

	t.Run("Open circuit breaker", func(t *testing.T) {
		registry := circuit.NewCircuitBreakerRegistry()

		err := registry.Register("dex", "")
		require.NoError(t, err)

		err = registry.Open("dex", "", "admin", "testing", nil)
		require.NoError(t, err)

		state, err := registry.GetState("dex", "")
		require.NoError(t, err)
		require.Equal(t, circuit.StatusOpen, state.Status)
		require.Equal(t, "admin", state.PausedBy)
		require.Equal(t, "testing", state.Reason)
	})

	t.Run("Close circuit breaker", func(t *testing.T) {
		registry := circuit.NewCircuitBreakerRegistry()

		err := registry.Register("dex", "")
		require.NoError(t, err)

		err = registry.Open("dex", "", "admin", "testing", nil)
		require.NoError(t, err)

		err = registry.Close("dex", "", "admin", "test complete")
		require.NoError(t, err)

		state, err := registry.GetState("dex", "")
		require.NoError(t, err)
		require.Equal(t, circuit.StatusClosed, state.Status)
	})

	t.Run("Auto-resume timer", func(t *testing.T) {
		registry := circuit.NewCircuitBreakerRegistry()

		err := registry.Register("dex", "")
		require.NoError(t, err)

		autoResume := 1 * time.Second
		err = registry.Open("dex", "", "admin", "testing", &autoResume)
		require.NoError(t, err)

		// Wait for auto-resume
		time.Sleep(2 * time.Second)

		resumed := registry.CheckAutoResume()
		require.Len(t, resumed, 1)
		require.Equal(t, "dex", resumed[0])

		state, err := registry.GetState("dex", "")
		require.NoError(t, err)
		require.Equal(t, circuit.StatusClosed, state.Status)
	})

	t.Run("Transition history", func(t *testing.T) {
		registry := circuit.NewCircuitBreakerRegistry()

		err := registry.Register("dex", "")
		require.NoError(t, err)

		err = registry.Open("dex", "", "admin", "testing", nil)
		require.NoError(t, err)

		err = registry.Close("dex", "", "admin", "test complete")
		require.NoError(t, err)

		state, err := registry.GetState("dex", "")
		require.NoError(t, err)
		require.Len(t, state.TransitionHistory, 2)
		require.Equal(t, circuit.StatusClosed, state.TransitionHistory[0].From)
		require.Equal(t, circuit.StatusOpen, state.TransitionHistory[0].To)
		require.Equal(t, circuit.StatusOpen, state.TransitionHistory[1].From)
		require.Equal(t, circuit.StatusClosed, state.TransitionHistory[1].To)
	})

	t.Run("Metadata operations", func(t *testing.T) {
		registry := circuit.NewCircuitBreakerRegistry()

		err := registry.Register("dex", "")
		require.NoError(t, err)

		metadata := map[string]interface{}{
			"reason":   "manual pause",
			"severity": "high",
			"ticket":   "OPS-123",
		}

		err = registry.SetMetadata("dex", "", metadata)
		require.NoError(t, err)

		state, err := registry.GetState("dex", "")
		require.NoError(t, err)
		require.Equal(t, "manual pause", state.Metadata["reason"])
		require.Equal(t, "high", state.Metadata["severity"])
		require.Equal(t, "OPS-123", state.Metadata["ticket"])
	})

	t.Run("Submodule circuit breakers", func(t *testing.T) {
		registry := circuit.NewCircuitBreakerRegistry()

		err := registry.Register("dex", "pool:1")
		require.NoError(t, err)

		err = registry.Open("dex", "pool:1", "admin", "testing pool", nil)
		require.NoError(t, err)

		require.True(t, registry.IsOpen("dex", "pool:1"))
		require.False(t, registry.IsOpen("dex", ""))
	})
}

func TestCircuitBreakerManager(t *testing.T) {
	t.Parallel()

	t.Run("Start and stop manager", func(t *testing.T) {
		manager := circuit.NewManager()

		manager.Start()
		require.NoError(t, manager.HealthCheck())

		manager.Stop()
	})

	t.Run("Pause and resume operations", func(t *testing.T) {
		manager := circuit.NewManager()
		manager.Start()
		defer manager.Stop()

		err := manager.RegisterCircuitBreaker("dex", "")
		require.NoError(t, err)

		err = manager.PauseModule("dex", "", "admin", "testing", nil)
		require.NoError(t, err)

		require.True(t, manager.IsBlocked("dex", ""))

		err = manager.ResumeModule("dex", "", "admin", "test complete")
		require.NoError(t, err)

		require.False(t, manager.IsBlocked("dex", ""))
	})

	t.Run("Auto-resume callback", func(t *testing.T) {
		manager := circuit.NewManager()
		manager.Start()
		defer manager.Stop()

		var callbackCalled atomic.Bool
		manager.SetAutoResumeCallback(func(module, subModule string) error {
			callbackCalled.Store(true)
			return nil
		})

		err := manager.RegisterCircuitBreaker("dex", "")
		require.NoError(t, err)

		autoResume := 1 * time.Second
		err = manager.PauseModule("dex", "", "admin", "testing", &autoResume)
		require.NoError(t, err)

		// Wait for auto-resume
		time.Sleep(2 * time.Second)

		require.True(t, callbackCalled.Load())
	})

	t.Run("Export and import state", func(t *testing.T) {
		manager := circuit.NewManager()

		err := manager.RegisterCircuitBreaker("dex", "")
		require.NoError(t, err)

		err = manager.PauseModule("dex", "", "admin", "testing", nil)
		require.NoError(t, err)

		// Export state
		data, err := manager.ExportState()
		require.NoError(t, err)

		// Create new manager and import
		newManager := circuit.NewManager()
		err = newManager.ImportState(data)
		require.NoError(t, err)

		// Verify state was imported
		state, err := newManager.GetState("dex", "")
		require.NoError(t, err)
		require.Equal(t, circuit.StatusOpen, state.Status)
		require.Equal(t, "admin", state.PausedBy)
	})
}
