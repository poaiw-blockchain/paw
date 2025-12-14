package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"runtime"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	startTime = time.Now()

	healthCheckTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "paw_health_check_total",
			Help: "Total number of health check requests",
		},
		[]string{"endpoint", "status"},
	)

	healthCheckDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "paw_health_check_duration_seconds",
			Help:    "Health check request duration in seconds",
			Buckets: []float64{0.01, 0.05, 0.1, 0.5, 1.0, 5.0},
		},
		[]string{"endpoint"},
	)

	serviceHealthy = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "paw_service_healthy",
			Help: "1 if service is healthy, 0 if unhealthy",
		},
		[]string{"service"},
	)
)

// HealthCheck represents the health check server
type HealthCheck struct {
	server      *http.Server
	nodeChecker NodeHealthChecker
	cache       *healthCache
	mu          sync.RWMutex
}

// NodeHealthChecker interface for checking node health
type NodeHealthChecker interface {
	CheckRPC() error
	CheckSync() (bool, int64, error)
	CheckConsensus() error
	GetPeerCount() (int, error)
	GetBlockHeight() (int64, error)
}

// healthCache caches health check results to avoid overload
type healthCache struct {
	mu          sync.RWMutex
	result      *DetailedHealthResponse
	lastChecked time.Time
	ttl         time.Duration
}

func newHealthCache(ttl time.Duration) *healthCache {
	return &healthCache{
		ttl: ttl,
	}
}

func (c *healthCache) get() (*DetailedHealthResponse, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.result == nil || time.Since(c.lastChecked) > c.ttl {
		return nil, false
	}

	return c.result, true
}

func (c *healthCache) set(result *DetailedHealthResponse) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.result = result
	c.lastChecked = time.Now()
}

// BasicHealthResponse is the response for /health
type BasicHealthResponse struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
}

// ReadinessResponse is the response for /health/ready
type ReadinessResponse struct {
	Status string                    `json:"status"`
	Checks map[string]CheckResult    `json:"checks"`
}

// DetailedHealthResponse is the response for /health/detailed
type DetailedHealthResponse struct {
	Status        string                    `json:"status"`
	UptimeSeconds int64                     `json:"uptime_seconds"`
	Version       string                    `json:"version"`
	Checks        map[string]CheckResult    `json:"checks"`
	Modules       map[string]ModuleHealth   `json:"modules"`
	System        SystemHealth              `json:"system"`
}

// CheckResult represents a single health check result
type CheckResult struct {
	Status  string `json:"status"`
	Message string `json:"message,omitempty"`
}

// ModuleHealth represents module-specific health information
type ModuleHealth struct {
	Status string                 `json:"status"`
	Metrics map[string]interface{} `json:"metrics,omitempty"`
}

// SystemHealth represents system-level health metrics
type SystemHealth struct {
	MemoryMB   uint64 `json:"memory_mb"`
	Goroutines int    `json:"goroutines"`
	Peers      int    `json:"peers"`
	BlockHeight int64  `json:"block_height"`
}

// StartHealthCheckServer starts the health check HTTP server
func StartHealthCheckServer(port int, nodeChecker NodeHealthChecker) *HealthCheck {
	hc := &HealthCheck{
		nodeChecker: nodeChecker,
		cache:       newHealthCache(5 * time.Second),
	}

	mux := http.NewServeMux()

	// Use middleware wrapper to avoid counting health checks in regular metrics
	mux.HandleFunc("/health", hc.withHealthMetrics("health", hc.handleBasicHealth))
	mux.HandleFunc("/health/ready", hc.withHealthMetrics("ready", hc.handleReadiness))
	mux.HandleFunc("/health/detailed", hc.withHealthMetrics("detailed", hc.handleDetailed))
	mux.HandleFunc("/health/startup", hc.withHealthMetrics("startup", hc.handleStartup))

	hc.server = &http.Server{
		Addr:              fmt.Sprintf(":%d", port),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
	}

	go func() {
		if err := hc.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("health check server error: %v\n", err)
		}
	}()

	return hc
}

// withHealthMetrics wraps health check handlers with metrics
func (hc *HealthCheck) withHealthMetrics(endpoint string, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Use a custom response writer to capture status code
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		handler(rw, r)

		duration := time.Since(start)
		status := fmt.Sprintf("%d", rw.statusCode)

		healthCheckTotal.WithLabelValues(endpoint, status).Inc()
		healthCheckDuration.WithLabelValues(endpoint).Observe(duration.Seconds())
	}
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// handleBasicHealth handles GET /health - always returns 200 if process is alive
func (hc *HealthCheck) handleBasicHealth(w http.ResponseWriter, r *http.Request) {
	response := BasicHealthResponse{
		Status:    "ok",
		Timestamp: time.Now().Format(time.RFC3339),
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// handleReadiness handles GET /health/ready - checks if service can handle traffic
func (hc *HealthCheck) handleReadiness(w http.ResponseWriter, r *http.Request) {
	checks := make(map[string]CheckResult)
	allHealthy := true

	// Check RPC connectivity
	if hc.nodeChecker != nil {
		if err := hc.nodeChecker.CheckRPC(); err != nil {
			checks["rpc"] = CheckResult{Status: "unhealthy", Message: err.Error()}
			allHealthy = false
			serviceHealthy.WithLabelValues("rpc").Set(0)
		} else {
			checks["rpc"] = CheckResult{Status: "ok"}
			serviceHealthy.WithLabelValues("rpc").Set(1)
		}

		// Check sync status
		syncing, height, err := hc.nodeChecker.CheckSync()
		if err != nil {
			checks["sync"] = CheckResult{Status: "unhealthy", Message: err.Error()}
			allHealthy = false
			serviceHealthy.WithLabelValues("sync").Set(0)
		} else if syncing {
			checks["sync"] = CheckResult{Status: "syncing", Message: fmt.Sprintf("catching up at height %d", height)}
			allHealthy = false
			serviceHealthy.WithLabelValues("sync").Set(0)
		} else {
			checks["sync"] = CheckResult{Status: "ok"}
			serviceHealthy.WithLabelValues("sync").Set(1)
		}

		// Check consensus participation
		if err := hc.nodeChecker.CheckConsensus(); err != nil {
			checks["consensus"] = CheckResult{Status: "degraded", Message: err.Error()}
			// Don't mark as unhealthy for non-validators
			serviceHealthy.WithLabelValues("consensus").Set(0.5)
		} else {
			checks["consensus"] = CheckResult{Status: "ok"}
			serviceHealthy.WithLabelValues("consensus").Set(1)
		}
	}

	status := "ready"
	statusCode := http.StatusOK
	if !allHealthy {
		status = "not_ready"
		statusCode = http.StatusServiceUnavailable
	}

	response := ReadinessResponse{
		Status: status,
		Checks: checks,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(response)
}

// handleDetailed handles GET /health/detailed - comprehensive health information
func (hc *HealthCheck) handleDetailed(w http.ResponseWriter, r *http.Request) {
	// Check cache first
	if cached, ok := hc.cache.get(); ok {
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("X-Cache", "HIT")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(cached)
		return
	}

	checks := make(map[string]CheckResult)
	modules := make(map[string]ModuleHealth)

	// Basic checks
	if hc.nodeChecker != nil {
		// RPC check
		if err := hc.nodeChecker.CheckRPC(); err != nil {
			checks["rpc"] = CheckResult{Status: "unhealthy", Message: err.Error()}
		} else {
			checks["rpc"] = CheckResult{Status: "ok"}
		}

		// Sync check
		syncing, height, err := hc.nodeChecker.CheckSync()
		if err != nil {
			checks["sync"] = CheckResult{Status: "unhealthy", Message: err.Error()}
		} else if syncing {
			checks["sync"] = CheckResult{Status: "syncing", Message: fmt.Sprintf("at height %d", height)}
		} else {
			checks["sync"] = CheckResult{Status: "ok"}
		}

		// Consensus check
		if err := hc.nodeChecker.CheckConsensus(); err != nil {
			checks["consensus"] = CheckResult{Status: "degraded", Message: err.Error()}
		} else {
			checks["consensus"] = CheckResult{Status: "ok"}
		}
	}

	// Module health (placeholder - would be populated by actual module queries)
	modules["dex"] = ModuleHealth{
		Status: "ok",
		Metrics: map[string]interface{}{
			"pools": 0,
			"volume_24h": 0,
		},
	}
	modules["oracle"] = ModuleHealth{
		Status: "ok",
		Metrics: map[string]interface{}{
			"active_validators": 0,
			"price_pairs": 0,
		},
	}
	modules["compute"] = ModuleHealth{
		Status: "ok",
		Metrics: map[string]interface{}{
			"active_providers": 0,
			"pending_requests": 0,
		},
	}

	// System metrics
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	peers := 0
	blockHeight := int64(0)
	if hc.nodeChecker != nil {
		peers, _ = hc.nodeChecker.GetPeerCount()
		blockHeight, _ = hc.nodeChecker.GetBlockHeight()
	}

	system := SystemHealth{
		MemoryMB:    m.Alloc / 1024 / 1024,
		Goroutines:  runtime.NumGoroutine(),
		Peers:       peers,
		BlockHeight: blockHeight,
	}

	// Determine overall status
	status := "healthy"
	for _, check := range checks {
		if check.Status == "unhealthy" {
			status = "unhealthy"
			break
		} else if check.Status == "degraded" && status == "healthy" {
			status = "degraded"
		}
	}

	response := &DetailedHealthResponse{
		Status:        status,
		UptimeSeconds: int64(time.Since(startTime).Seconds()),
		Version:       getVersion(),
		Checks:        checks,
		Modules:       modules,
		System:        system,
	}

	// Cache the result
	hc.cache.set(response)

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("X-Cache", "MISS")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

// handleStartup handles GET /health/startup - for Kubernetes startup probes
func (hc *HealthCheck) handleStartup(w http.ResponseWriter, r *http.Request) {
	// Give the application time to initialize (30 seconds grace period)
	if time.Since(startTime) < 30*time.Second {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"status": "starting",
			"message": "application is initializing",
		})
		return
	}

	// After grace period, use readiness check
	hc.handleReadiness(w, r)
}

// Shutdown gracefully shuts down the health check server
func (hc *HealthCheck) Shutdown(ctx context.Context) error {
	if hc.server != nil {
		return hc.server.Shutdown(ctx)
	}
	return nil
}

func getVersion() string {
	// Try to read version from environment or build info
	if version := os.Getenv("PAW_VERSION"); version != "" {
		return version
	}
	return "dev"
}
