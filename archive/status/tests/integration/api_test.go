package tests

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"status/pkg/api"
	"status/pkg/config"
	"status/pkg/health"
	"status/pkg/incidents"
	"status/pkg/metrics"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
)

func setupTestServer() (*httptest.Server, *health.Monitor, *incidents.Manager, *metrics.Collector) {
	cfg := &config.Config{
		MonitorInterval:   30 * time.Second,
		MetricsRetention:  24 * time.Hour,
		IncidentRetention: 90 * 24 * time.Hour,
		BlockchainRPCURL:  "http://localhost:26657",
		APIEndpoint:       "http://localhost:1317",
		WebSocketEndpoint: "ws://localhost:26657/websocket",
		ExplorerEndpoint:  "http://localhost:3000",
		FaucetEndpoint:    "http://localhost:8000",
	}

	healthMonitor := health.NewMonitor(cfg)
	incidentManager := incidents.NewManager(cfg)
	metricsCollector := metrics.NewCollector(cfg)

	// Start background services
	ctx := context.Background()
	go healthMonitor.Start(ctx)
	go incidentManager.Start(ctx)
	go metricsCollector.Start(ctx)

	// Wait for initialization
	time.Sleep(100 * time.Millisecond)

	router := mux.NewRouter()
	apiHandler := api.NewHandler(healthMonitor, incidentManager, metricsCollector)
	apiRouter := router.PathPrefix("/api/v1").Subrouter()
	apiHandler.RegisterRoutes(apiRouter)

	server := httptest.NewServer(router)
	return server, healthMonitor, incidentManager, metricsCollector
}

func TestHealthEndpoint(t *testing.T) {
	server, _, _, _ := setupTestServer()
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/health")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Equal(t, "healthy", response["status"])
}

func TestStatusEndpoint(t *testing.T) {
	server, _, _, _ := setupTestServer()
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/status")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Contains(t, response, "overall_status")
	assert.Contains(t, response, "message")
	assert.Contains(t, response, "components")
}

func TestGetIncidentsEndpoint(t *testing.T) {
	server, _, _, _ := setupTestServer()
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/incidents")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Contains(t, response, "active")
	assert.Contains(t, response, "history")
}

func TestCreateIncidentEndpoint(t *testing.T) {
	server, _, _, _ := setupTestServer()
	defer server.Close()

	incidentData := map[string]interface{}{
		"title":       "Test Incident",
		"description": "This is a test incident",
		"severity":    "major",
		"components":  []string{"API", "Database"},
	}

	body, _ := json.Marshal(incidentData)
	resp, err := http.Post(
		server.URL+"/api/v1/incidents",
		"application/json",
		bytes.NewBuffer(body),
	)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusCreated, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Equal(t, "Test Incident", response["title"])
	assert.Equal(t, "major", response["severity"])
}

func TestUpdateIncidentEndpoint(t *testing.T) {
	server, _, manager, _ := setupTestServer()
	defer server.Close()

	// Create an incident first
	incident, _ := manager.CreateIncident(
		"Test Incident",
		"Test description",
		incidents.SeverityMinor,
		[]string{"API"},
	)

	// Update the incident
	updateData := map[string]interface{}{
		"message": "Issue has been identified",
		"status":  "identified",
	}

	body, _ := json.Marshal(updateData)
	url := fmt.Sprintf("%s/api/v1/incidents/%d/update", server.URL, incident.ID)
	req, _ := http.NewRequest(
		"POST",
		url,
		bytes.NewBuffer(body),
	)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
}

func TestGetMetricsEndpoint(t *testing.T) {
	server, _, _, _ := setupTestServer()
	defer server.Close()

	// Wait for metrics to be collected
	time.Sleep(1 * time.Second)

	resp, err := http.Get(server.URL + "/api/v1/metrics")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Contains(t, response, "tps")
	assert.Contains(t, response, "block_time")
	assert.Contains(t, response, "peers")
	assert.Contains(t, response, "response_time")
	assert.Contains(t, response, "network_stats")
}

func TestGetMetricsSummaryEndpoint(t *testing.T) {
	server, _, _, _ := setupTestServer()
	defer server.Close()

	// Wait for metrics
	time.Sleep(1 * time.Second)

	resp, err := http.Get(server.URL + "/api/v1/metrics/summary")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Contains(t, response, "network_stats")
}

func TestStatusHistoryEndpoint(t *testing.T) {
	server, _, _, _ := setupTestServer()
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/status/history?days=30")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err)
	assert.Contains(t, response, "days")
	assert.Contains(t, response, "history")
	assert.Equal(t, float64(30), response["days"])
}

func TestSubscribeEndpoint(t *testing.T) {
	server, _, _, _ := setupTestServer()
	defer server.Close()

	subscribeData := map[string]interface{}{
		"email": "user@domain.com",
		"preferences": map[string]bool{
			"incidents":   true,
			"maintenance": false,
		},
	}

	body, _ := json.Marshal(subscribeData)
	resp, err := http.Post(
		server.URL+"/api/v1/subscribe",
		"application/json",
		bytes.NewBuffer(body),
	)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err)
	assert.True(t, response["success"].(bool))
}

func TestUnsubscribeEndpoint(t *testing.T) {
	server, _, manager, _ := setupTestServer()
	defer server.Close()

	// Subscribe first
	manager.Subscribe("user@domain.com")

	unsubscribeData := map[string]interface{}{
		"email": "user@domain.com",
	}

	body, _ := json.Marshal(unsubscribeData)
	resp, err := http.Post(
		server.URL+"/api/v1/unsubscribe",
		"application/json",
		bytes.NewBuffer(body),
	)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	var response map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&response)
	assert.NoError(t, err)
	assert.True(t, response["success"].(bool))
}

func TestRSSFeedEndpoint(t *testing.T) {
	server, _, _, _ := setupTestServer()
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/status/rss")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Contains(t, resp.Header.Get("Content-Type"), "xml")
}

func TestInvalidEndpoints(t *testing.T) {
	server, _, _, _ := setupTestServer()
	defer server.Close()

	// Test invalid incident ID
	resp, err := http.Get(server.URL + "/api/v1/incidents/invalid")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	// Test non-existent incident
	resp, err = http.Get(server.URL + "/api/v1/incidents/99999")
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

func TestCreateIncidentValidation(t *testing.T) {
	server, _, _, _ := setupTestServer()
	defer server.Close()

	// Test missing title
	incidentData := map[string]interface{}{
		"description": "Test",
		"severity":    "major",
	}

	body, _ := json.Marshal(incidentData)
	resp, err := http.Post(
		server.URL+"/api/v1/incidents",
		"application/json",
		bytes.NewBuffer(body),
	)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	// Test invalid severity
	incidentData = map[string]interface{}{
		"title":       "Test",
		"description": "Test",
		"severity":    "invalid",
	}

	body, _ = json.Marshal(incidentData)
	resp, err = http.Post(
		server.URL+"/api/v1/incidents",
		"application/json",
		bytes.NewBuffer(body),
	)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestSubscribeValidation(t *testing.T) {
	server, _, _, _ := setupTestServer()
	defer server.Close()

	// Test missing email
	subscribeData := map[string]interface{}{
		"preferences": map[string]bool{
			"incidents": true,
		},
	}

	body, _ := json.Marshal(subscribeData)
	resp, err := http.Post(
		server.URL+"/api/v1/subscribe",
		"application/json",
		bytes.NewBuffer(body),
	)

	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}
