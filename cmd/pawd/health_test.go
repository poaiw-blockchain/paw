package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"testing"
	"time"
)

// mockNodeChecker implements NodeHealthChecker for testing
type mockNodeChecker struct {
	rpcErr       error
	syncing      bool
	height       int64
	syncErr      error
	consensusErr error
	peerCount    int
	peerErr      error
}

func (m *mockNodeChecker) CheckRPC() error {
	return m.rpcErr
}

func (m *mockNodeChecker) CheckSync() (bool, int64, error) {
	return m.syncing, m.height, m.syncErr
}

func (m *mockNodeChecker) CheckConsensus() error {
	return m.consensusErr
}

func (m *mockNodeChecker) GetPeerCount() (int, error) {
	return m.peerCount, m.peerErr
}

func (m *mockNodeChecker) GetBlockHeight() (int64, error) {
	return m.height, nil
}

func TestHealthCheckBasic(t *testing.T) {
	checker := &mockNodeChecker{
		height:    12345,
		peerCount: 5,
	}

	hc := StartHealthCheckServer(38661, checker)
	defer hc.Shutdown(context.Background())

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Test basic health endpoint
	resp, err := http.Get("http://localhost:38661/health")
	if err != nil {
		t.Fatalf("Failed to get health: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result BasicHealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result.Status != "ok" {
		t.Errorf("Expected status 'ok', got '%s'", result.Status)
	}

	if result.Timestamp == "" {
		t.Error("Expected timestamp, got empty string")
	}
}

func TestHealthCheckReady(t *testing.T) {
	checker := &mockNodeChecker{
		height:    12345,
		peerCount: 5,
	}

	hc := StartHealthCheckServer(38662, checker)
	defer hc.Shutdown(context.Background())

	time.Sleep(100 * time.Millisecond)

	// Test readiness endpoint when healthy
	resp, err := http.Get("http://localhost:38662/health/ready")
	if err != nil {
		t.Fatalf("Failed to get health/ready: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result ReadinessResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result.Status != "ready" {
		t.Errorf("Expected status 'ready', got '%s'", result.Status)
	}
}

func TestHealthCheckReadyWhenSyncing(t *testing.T) {
	checker := &mockNodeChecker{
		syncing:   true,
		height:    12345,
		peerCount: 5,
	}

	hc := StartHealthCheckServer(38663, checker)
	defer hc.Shutdown(context.Background())

	time.Sleep(100 * time.Millisecond)

	// Test readiness endpoint when syncing
	resp, err := http.Get("http://localhost:38663/health/ready")
	if err != nil {
		t.Fatalf("Failed to get health/ready: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d", resp.StatusCode)
	}

	var result ReadinessResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result.Status != "not_ready" {
		t.Errorf("Expected status 'not_ready', got '%s'", result.Status)
	}

	if result.Checks["sync"].Status != "syncing" {
		t.Errorf("Expected sync status 'syncing', got '%s'", result.Checks["sync"].Status)
	}
}

func TestHealthCheckReadyWhenRPCFails(t *testing.T) {
	checker := &mockNodeChecker{
		rpcErr:    fmt.Errorf("connection refused"),
		height:    12345,
		peerCount: 5,
	}

	hc := StartHealthCheckServer(38664, checker)
	defer hc.Shutdown(context.Background())

	time.Sleep(100 * time.Millisecond)

	// Test readiness endpoint when RPC fails
	resp, err := http.Get("http://localhost:38664/health/ready")
	if err != nil {
		t.Fatalf("Failed to get health/ready: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503, got %d", resp.StatusCode)
	}

	var result ReadinessResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result.Checks["rpc"].Status != "unhealthy" {
		t.Errorf("Expected rpc status 'unhealthy', got '%s'", result.Checks["rpc"].Status)
	}
}

func TestHealthCheckDetailed(t *testing.T) {
	checker := &mockNodeChecker{
		height:    12345,
		peerCount: 5,
	}

	hc := StartHealthCheckServer(38665, checker)
	defer hc.Shutdown(context.Background())

	time.Sleep(100 * time.Millisecond)

	// Test detailed health endpoint
	resp, err := http.Get("http://localhost:38665/health/detailed")
	if err != nil {
		t.Fatalf("Failed to get health/detailed: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200, got %d", resp.StatusCode)
	}

	var result DetailedHealthResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result.Status != "healthy" {
		t.Errorf("Expected status 'healthy', got '%s'", result.Status)
	}

	if result.System.Peers != 5 {
		t.Errorf("Expected 5 peers, got %d", result.System.Peers)
	}

	if result.System.BlockHeight != 12345 {
		t.Errorf("Expected block height 12345, got %d", result.System.BlockHeight)
	}

	// Check that modules are present
	if _, ok := result.Modules["dex"]; !ok {
		t.Error("Expected DEX module in response")
	}
	if _, ok := result.Modules["oracle"]; !ok {
		t.Error("Expected Oracle module in response")
	}
	if _, ok := result.Modules["compute"]; !ok {
		t.Error("Expected Compute module in response")
	}
}

func TestHealthCheckCache(t *testing.T) {
	checker := &mockNodeChecker{
		height:    12345,
		peerCount: 5,
	}

	hc := StartHealthCheckServer(38666, checker)
	defer hc.Shutdown(context.Background())

	time.Sleep(100 * time.Millisecond)

	// First request should not be cached
	resp1, err := http.Get("http://localhost:38666/health/detailed")
	if err != nil {
		t.Fatalf("Failed to get health/detailed: %v", err)
	}
	defer resp1.Body.Close()

	cache1 := resp1.Header.Get("X-Cache")
	if cache1 != "MISS" {
		t.Errorf("Expected cache MISS, got %s", cache1)
	}

	// Second request should be cached
	resp2, err := http.Get("http://localhost:38666/health/detailed")
	if err != nil {
		t.Fatalf("Failed to get health/detailed: %v", err)
	}
	defer resp2.Body.Close()

	cache2 := resp2.Header.Get("X-Cache")
	if cache2 != "HIT" {
		t.Errorf("Expected cache HIT, got %s", cache2)
	}
}

func TestHealthCheckStartup(t *testing.T) {
	// Reset startTime to simulate fresh startup
	originalStart := startTime
	startTime = time.Now()
	defer func() { startTime = originalStart }()

	checker := &mockNodeChecker{
		height:    12345,
		peerCount: 5,
	}

	hc := StartHealthCheckServer(38667, checker)
	defer hc.Shutdown(context.Background())

	time.Sleep(100 * time.Millisecond)

	// Should return 503 during grace period
	resp, err := http.Get("http://localhost:38667/health/startup")
	if err != nil {
		t.Fatalf("Failed to get health/startup: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusServiceUnavailable {
		t.Errorf("Expected status 503 during startup, got %d", resp.StatusCode)
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if result["status"] != "starting" {
		t.Errorf("Expected status 'starting', got '%s'", result["status"])
	}
}
