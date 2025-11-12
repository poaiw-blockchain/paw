package health

import (
	"context"
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// HealthStatus represents the overall health status
type HealthStatus string

const (
	StatusHealthy   HealthStatus = "healthy"
	StatusDegraded  HealthStatus = "degraded"
	StatusUnhealthy HealthStatus = "unhealthy"
)

// HealthResponse represents the health check response
type HealthResponse struct {
	Status    HealthStatus           `json:"status"`
	Timestamp time.Time              `json:"timestamp"`
	Version   string                 `json:"version"`
	Checks    map[string]CheckResult `json:"checks"`
}

// CheckResult represents the result of an individual health check
type CheckResult struct {
	Status  HealthStatus `json:"status"`
	Message string       `json:"message,omitempty"`
	Latency string       `json:"latency,omitempty"`
}

// HealthChecker manages health checks
type HealthChecker struct {
	version        string
	checks         map[string]CheckFunc
	mu             sync.RWMutex
	checkTimeout   time.Duration
	cacheTimeout   time.Duration
	cachedResponse *HealthResponse
	lastCheck      time.Time
}

// CheckFunc is a function that performs a health check
type CheckFunc func(ctx context.Context) CheckResult

// NewHealthChecker creates a new health checker
func NewHealthChecker(version string) *HealthChecker {
	return &HealthChecker{
		version:      version,
		checks:       make(map[string]CheckFunc),
		checkTimeout: 5 * time.Second,
		cacheTimeout: 10 * time.Second,
	}
}

// RegisterCheck registers a new health check
func (hc *HealthChecker) RegisterCheck(name string, check CheckFunc) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	hc.checks[name] = check
}

// PerformChecks runs all registered health checks
func (hc *HealthChecker) PerformChecks(ctx context.Context) *HealthResponse {
	hc.mu.RLock()
	// Check if we can use cached response
	if hc.cachedResponse != nil && time.Since(hc.lastCheck) < hc.cacheTimeout {
		hc.mu.RUnlock()
		return hc.cachedResponse
	}
	hc.mu.RUnlock()

	// Run checks
	results := make(map[string]CheckResult)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for name, check := range hc.checks {
		wg.Add(1)
		go func(n string, c CheckFunc) {
			defer wg.Done()

			// Create timeout context for this check
			checkCtx, cancel := context.WithTimeout(ctx, hc.checkTimeout)
			defer cancel()

			result := c(checkCtx)

			mu.Lock()
			results[n] = result
			mu.Unlock()
		}(name, check)
	}

	wg.Wait()

	// Determine overall status
	overallStatus := StatusHealthy
	for _, result := range results {
		if result.Status == StatusUnhealthy {
			overallStatus = StatusUnhealthy
			break
		}
		if result.Status == StatusDegraded && overallStatus == StatusHealthy {
			overallStatus = StatusDegraded
		}
	}

	response := &HealthResponse{
		Status:    overallStatus,
		Timestamp: time.Now(),
		Version:   hc.version,
		Checks:    results,
	}

	// Cache the response
	hc.mu.Lock()
	hc.cachedResponse = response
	hc.lastCheck = time.Now()
	hc.mu.Unlock()

	return response
}

// HTTPHandlers returns HTTP handlers for health endpoints
func (hc *HealthChecker) HTTPHandlers() map[string]http.HandlerFunc {
	return map[string]http.HandlerFunc{
		"/health":       hc.HealthHandler,
		"/health/live":  hc.LivenessHandler,
		"/health/ready": hc.ReadinessHandler,
		"/metrics":      promhttp.Handler().ServeHTTP,
	}
}

// HealthHandler returns overall health status
func (hc *HealthChecker) HealthHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	response := hc.PerformChecks(ctx)

	w.Header().Set("Content-Type", "application/json")

	statusCode := http.StatusOK
	if response.Status == StatusUnhealthy {
		statusCode = http.StatusServiceUnavailable
	} else if response.Status == StatusDegraded {
		statusCode = http.StatusOK // Still OK but degraded
	}

	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// LivenessHandler is a simple liveness probe
func (hc *HealthChecker) LivenessHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "alive",
		"timestamp": time.Now(),
	})
}

// ReadinessHandler checks if the service is ready to accept traffic
func (hc *HealthChecker) ReadinessHandler(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	response := hc.PerformChecks(ctx)

	w.Header().Set("Content-Type", "application/json")

	if response.Status == StatusHealthy {
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "ready",
			"timestamp": time.Now(),
		})
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status":    "not_ready",
			"reason":    response.Status,
			"timestamp": time.Now(),
		})
	}
}

// Common health check functions

// DatabaseCheck creates a health check for database connectivity
func DatabaseCheck(pingFunc func(context.Context) error) CheckFunc {
	return func(ctx context.Context) CheckResult {
		start := time.Now()
		err := pingFunc(ctx)
		latency := time.Since(start)

		if err != nil {
			return CheckResult{
				Status:  StatusUnhealthy,
				Message: "Database connection failed: " + err.Error(),
				Latency: latency.String(),
			}
		}

		return CheckResult{
			Status:  StatusHealthy,
			Message: "Database connection OK",
			Latency: latency.String(),
		}
	}
}

// RPCCheck creates a health check for RPC connectivity
func RPCCheck(statusFunc func(context.Context) error) CheckFunc {
	return func(ctx context.Context) CheckResult {
		start := time.Now()
		err := statusFunc(ctx)
		latency := time.Since(start)

		if err != nil {
			return CheckResult{
				Status:  StatusUnhealthy,
				Message: "RPC connection failed: " + err.Error(),
				Latency: latency.String(),
			}
		}

		return CheckResult{
			Status:  StatusHealthy,
			Message: "RPC connection OK",
			Latency: latency.String(),
		}
	}
}

// ConsensusCheck creates a health check for consensus participation
func ConsensusCheck(isValidatorFunc func() bool, heightFunc func() int64) CheckFunc {
	return func(ctx context.Context) CheckResult {
		start := time.Now()

		height := heightFunc()
		if height == 0 {
			return CheckResult{
				Status:  StatusUnhealthy,
				Message: "No blocks produced",
				Latency: time.Since(start).String(),
			}
		}

		isValidator := isValidatorFunc()
		message := "Node is syncing"
		if isValidator {
			message = "Node is validating"
		}

		return CheckResult{
			Status:  StatusHealthy,
			Message: message,
			Latency: time.Since(start).String(),
		}
	}
}

// SyncCheck creates a health check for sync status
func SyncCheck(syncStatusFunc func(context.Context) (bool, int64, error)) CheckFunc {
	return func(ctx context.Context) CheckResult {
		start := time.Now()

		catching_up, latest_block_height, err := syncStatusFunc(ctx)
		latency := time.Since(start)

		if err != nil {
			return CheckResult{
				Status:  StatusUnhealthy,
				Message: "Failed to get sync status: " + err.Error(),
				Latency: latency.String(),
			}
		}

		if catching_up {
			return CheckResult{
				Status:  StatusDegraded,
				Message: "Node is catching up",
				Latency: latency.String(),
			}
		}

		return CheckResult{
			Status:  StatusHealthy,
			Message: "Node is synced",
			Latency: latency.String(),
		}
	}
}
