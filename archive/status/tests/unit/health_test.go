package tests

import (
	"context"
	"testing"
	"time"

	"status/pkg/config"
	"status/pkg/health"

	"github.com/stretchr/testify/assert"
)

func TestNewMonitor(t *testing.T) {
	cfg := &config.Config{
		MonitorInterval:  30 * time.Second,
		BlockchainRPCURL: "http://localhost:26657",
		APIEndpoint:      "http://localhost:1317",
	}

	monitor := health.NewMonitor(cfg)
	assert.NotNil(t, monitor)
}

func TestGetStatus(t *testing.T) {
	cfg := &config.Config{
		MonitorInterval:   30 * time.Second,
		BlockchainRPCURL:  "http://localhost:26657",
		APIEndpoint:       "http://localhost:1317",
		WebSocketEndpoint: "ws://localhost:26657/websocket",
		ExplorerEndpoint:  "http://localhost:3000",
		FaucetEndpoint:    "http://localhost:8000",
	}

	monitor := health.NewMonitor(cfg)

	// Start monitor in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go monitor.Start(ctx)

	// Wait a bit for initialization
	time.Sleep(100 * time.Millisecond)

	status := monitor.GetStatus()
	assert.NotNil(t, status)
	assert.NotEmpty(t, status.Components)
	assert.Contains(t, []health.ComponentStatus{
		health.StatusOperational,
		health.StatusDegraded,
		health.StatusDown,
	}, status.Status)
}

func TestHealthCheck(t *testing.T) {
	cfg := &config.Config{
		MonitorInterval: 30 * time.Second,
	}

	monitor := health.NewMonitor(cfg)
	response := monitor.HealthCheck()

	assert.NotNil(t, response)
	assert.Equal(t, "healthy", response.Status)
	assert.NotEmpty(t, response.Version)
	assert.NotZero(t, response.Timestamp)
}

func TestGetComponent(t *testing.T) {
	cfg := &config.Config{
		MonitorInterval:   30 * time.Second,
		BlockchainRPCURL:  "http://localhost:26657",
		APIEndpoint:       "http://localhost:1317",
		WebSocketEndpoint: "ws://localhost:26657/websocket",
		ExplorerEndpoint:  "http://localhost:3000",
		FaucetEndpoint:    "http://localhost:8000",
	}

	monitor := health.NewMonitor(cfg)

	// Start monitor in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go monitor.Start(ctx)

	// Wait for initialization
	time.Sleep(100 * time.Millisecond)

	// Test getting existing component
	comp, err := monitor.GetComponent("Blockchain")
	assert.NoError(t, err)
	assert.NotNil(t, comp)
	assert.Equal(t, "Blockchain", comp.Name)

	// Test getting non-existent component
	_, err = monitor.GetComponent("NonExistent")
	assert.Error(t, err)
}

func TestGetUptimeHistory(t *testing.T) {
	cfg := &config.Config{
		MonitorInterval:  30 * time.Second,
		BlockchainRPCURL: "http://localhost:26657",
		APIEndpoint:      "http://localhost:1317",
	}

	monitor := health.NewMonitor(cfg)

	history := monitor.GetUptimeHistory(30)
	assert.NotNil(t, history)
	assert.Len(t, history, 30)

	// Verify each entry has required fields
	for _, entry := range history {
		assert.Contains(t, entry, "date")
		assert.Contains(t, entry, "status")
	}
}

func TestComponentStatusValues(t *testing.T) {
	assert.Equal(t, health.ComponentStatus("operational"), health.StatusOperational)
	assert.Equal(t, health.ComponentStatus("degraded"), health.StatusDegraded)
	assert.Equal(t, health.ComponentStatus("down"), health.StatusDown)
}
