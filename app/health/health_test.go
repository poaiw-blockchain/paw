package health

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"cosmossdk.io/log"
	"github.com/cosmos/cosmos-sdk/client"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type HealthCheckTestSuite struct {
	suite.Suite
	checker *Checker
	logger  log.Logger
}

func TestHealthCheckTestSuite(t *testing.T) {
	suite.Run(t, new(HealthCheckTestSuite))
}

func (suite *HealthCheckTestSuite) SetupTest() {
	suite.logger = log.NewNopLogger()
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	require.Equal(t, "http://localhost:26657", cfg.RPCURL)
	require.Equal(t, int64(10), cfg.MaxBlockLag)
	require.Equal(t, 5*time.Second, cfg.MaxResponseTime)
	require.Equal(t, 3, cfg.MinPeerCount)
	require.Equal(t, 5*time.Second, cfg.CacheDuration)
}

func TestNewChecker(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		config      Config
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid config",
			config: Config{
				RPCURL:          "http://localhost:26657",
				MaxBlockLag:     10,
				MaxResponseTime: 5 * time.Second,
				MinPeerCount:    3,
				CacheDuration:   5 * time.Second,
			},
			expectError: false,
		},
		{
			name: "missing RPC URL",
			config: Config{
				MaxBlockLag:     10,
				MaxResponseTime: 5 * time.Second,
				MinPeerCount:    3,
				CacheDuration:   5 * time.Second,
			},
			expectError: true,
			errorMsg:    "RPC URL is required",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			logger := log.NewNopLogger()
			clientCtx := client.Context{}

			checker, err := NewChecker(logger, tt.config, clientCtx)

			if tt.expectError {
				require.Error(t, err)
				require.Contains(t, err.Error(), tt.errorMsg)
				require.Nil(t, checker)
			} else {
				require.NoError(t, err)
				require.NotNil(t, checker)
				require.Equal(t, tt.config.MaxBlockLag, checker.maxBlockLag)
				require.Equal(t, tt.config.MaxResponseTime, checker.maxResponseTime)
				require.Equal(t, tt.config.MinPeerCount, checker.minPeerCount)
			}
		})
	}
}

func TestCalculateOverallStatus(t *testing.T) {
	t.Parallel()

	logger := log.NewNopLogger()
	cfg := DefaultConfig()
	clientCtx := client.Context{}

	checker, err := NewChecker(logger, cfg, clientCtx)
	require.NoError(t, err)

	tests := []struct {
		name       string
		components map[string]ComponentHealth
		expected   Status
	}{
		{
			name: "all healthy",
			components: map[string]ComponentHealth{
				"rpc":       {Status: StatusHealthy},
				"consensus": {Status: StatusHealthy},
				"network":   {Status: StatusHealthy},
			},
			expected: StatusHealthy,
		},
		{
			name: "one degraded",
			components: map[string]ComponentHealth{
				"rpc":       {Status: StatusHealthy},
				"consensus": {Status: StatusDegraded},
				"network":   {Status: StatusHealthy},
			},
			expected: StatusDegraded,
		},
		{
			name: "one unhealthy",
			components: map[string]ComponentHealth{
				"rpc":       {Status: StatusHealthy},
				"consensus": {Status: StatusUnhealthy},
				"network":   {Status: StatusHealthy},
			},
			expected: StatusUnhealthy,
		},
		{
			name: "unhealthy takes precedence over degraded",
			components: map[string]ComponentHealth{
				"rpc":       {Status: StatusDegraded},
				"consensus": {Status: StatusUnhealthy},
				"network":   {Status: StatusHealthy},
			},
			expected: StatusUnhealthy,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			status := checker.calculateOverallStatus(tt.components)
			require.Equal(t, tt.expected, status)
		})
	}
}

func TestShouldUseCached(t *testing.T) {
	t.Parallel()

	logger := log.NewNopLogger()
	cfg := DefaultConfig()
	cfg.CacheDuration = 1 * time.Second
	clientCtx := client.Context{}

	checker, err := NewChecker(logger, cfg, clientCtx)
	require.NoError(t, err)

	// No cache initially
	require.False(t, checker.shouldUseCached())

	// Set cache
	checker.mu.Lock()
	checker.cachedHealth = &HealthCheck{
		Status:     StatusHealthy,
		Timestamp:  time.Now(),
		Components: make(map[string]ComponentHealth),
	}
	checker.lastCheck = time.Now()
	checker.mu.Unlock()

	// Should use cache
	require.True(t, checker.shouldUseCached())

	// Wait for cache to expire
	time.Sleep(1100 * time.Millisecond)

	// Should not use cache
	require.False(t, checker.shouldUseCached())
}

func TestHandleHealth(t *testing.T) {
	t.Parallel()

	logger := log.NewNopLogger()
	cfg := DefaultConfig()
	clientCtx := client.Context{}

	checker, err := NewChecker(logger, cfg, clientCtx)
	require.NoError(t, err)

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	checker.handleHealth(w, req)

	require.Equal(t, http.StatusOK, w.Code)
	require.Equal(t, "application/json", w.Header().Get("Content-Type"))

	var response map[string]interface{}
	err = json.NewDecoder(w.Body).Decode(&response)
	require.NoError(t, err)
	require.Equal(t, "ok", response["status"])
	require.NotEmpty(t, response["timestamp"])
}

func TestRegisterRoutes(t *testing.T) {
	t.Parallel()

	logger := log.NewNopLogger()
	cfg := DefaultConfig()
	clientCtx := client.Context{}

	checker, err := NewChecker(logger, cfg, clientCtx)
	require.NoError(t, err)

	router := mux.NewRouter()
	checker.RegisterRoutes(router)

	// Test that routes are registered
	routes := []string{
		"/health",
		"/health/ready",
		"/health/detailed",
	}

	for _, route := range routes {
		req := httptest.NewRequest("GET", route, nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// Should get a response (not 404)
		require.NotEqual(t, http.StatusNotFound, w.Code, "Route %s should be registered", route)
	}
}

func TestComponentHealthJSONSerialization(t *testing.T) {
	t.Parallel()

	component := ComponentHealth{
		Status:    StatusHealthy,
		Message:   "All systems operational",
		Timestamp: time.Now(),
		Metrics: map[string]interface{}{
			"response_time_ms": 100,
			"peer_count":       5,
		},
	}

	data, err := json.Marshal(component)
	require.NoError(t, err)
	require.NotEmpty(t, data)

	var decoded ComponentHealth
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	require.Equal(t, component.Status, decoded.Status)
	require.Equal(t, component.Message, decoded.Message)
}

func TestHealthCheckJSONSerialization(t *testing.T) {
	t.Parallel()

	health := HealthCheck{
		Status:    StatusHealthy,
		Timestamp: time.Now(),
		Version:   "1.0.0",
		Components: map[string]ComponentHealth{
			"rpc": {
				Status:    StatusHealthy,
				Message:   "RPC healthy",
				Timestamp: time.Now(),
			},
		},
		Metrics: map[string]interface{}{
			"uptime_seconds": 3600,
		},
	}

	data, err := json.Marshal(health)
	require.NoError(t, err)
	require.NotEmpty(t, data)

	var decoded HealthCheck
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	require.Equal(t, health.Status, decoded.Status)
	require.Equal(t, health.Version, decoded.Version)
}

func TestStatusConstants(t *testing.T) {
	t.Parallel()

	require.Equal(t, Status("healthy"), StatusHealthy)
	require.Equal(t, Status("degraded"), StatusDegraded)
	require.Equal(t, Status("unhealthy"), StatusUnhealthy)
	require.Equal(t, Status("unknown"), StatusUnknown)
}

func TestConcurrentHealthChecks(t *testing.T) {
	t.Parallel()

	logger := log.NewNopLogger()
	cfg := DefaultConfig()
	cfg.CacheDuration = 100 * time.Millisecond
	clientCtx := client.Context{}

	checker, err := NewChecker(logger, cfg, clientCtx)
	require.NoError(t, err)

	// Simulate concurrent health check requests
	const numRequests = 10
	results := make(chan error, numRequests)

	for i := 0; i < numRequests; i++ {
		go func() {
			req := httptest.NewRequest("GET", "/health", nil)
			w := httptest.NewRecorder()
			checker.handleHealth(w, req)

			if w.Code != http.StatusOK {
				results <- fmt.Errorf("unexpected status %d", w.Code)
				return
			}

			results <- nil
		}()
	}

	// Collect results
	for i := 0; i < numRequests; i++ {
		err := <-results
		require.NoError(t, err, "Concurrent request %d failed", i)
	}
}

func BenchmarkHealthCheck(b *testing.B) {
	logger := log.NewNopLogger()
	cfg := DefaultConfig()
	clientCtx := client.Context{}

	checker, err := NewChecker(logger, cfg, clientCtx)
	require.NoError(b, err)

	req := httptest.NewRequest("GET", "/health", nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		checker.handleHealth(w, req)
	}
}

func BenchmarkCalculateOverallStatus(b *testing.B) {
	logger := log.NewNopLogger()
	cfg := DefaultConfig()
	clientCtx := client.Context{}

	checker, err := NewChecker(logger, cfg, clientCtx)
	require.NoError(b, err)

	components := map[string]ComponentHealth{
		"rpc":       {Status: StatusHealthy},
		"consensus": {Status: StatusHealthy},
		"network":   {Status: StatusDegraded},
		"database":  {Status: StatusHealthy},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = checker.calculateOverallStatus(components)
	}
}
