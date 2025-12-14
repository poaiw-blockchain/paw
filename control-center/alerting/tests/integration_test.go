package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/paw/control-center/alerting"
	"github.com/paw/control-center/alerting/api"
	"github.com/paw/control-center/alerting/channels"
	"github.com/paw/control-center/alerting/engine"
	"github.com/paw/control-center/alerting/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupTestServer creates a test server with in-memory storage
func setupTestServer(t *testing.T) (*gin.Engine, *storage.PostgresStorage, func()) {
	// Use test database URL (should be set in environment or use Docker)
	dbURL := "postgres://postgres:postgres@localhost:5432/paw_alerting_test?sslmode=disable"
	redisURL := "redis://localhost:6379/1"

	store, err := storage.NewPostgresStorage(dbURL, redisURL)
	if err != nil {
		t.Skip("Skipping integration test: database not available")
	}

	provider := NewMockMetricsProvider()
	evaluator := engine.NewEvaluator(provider)

	config := &alerting.Config{
		EvaluationInterval:  10 * time.Second,
		DefaultForDuration:  1 * time.Minute,
		MaxRetries:          3,
		RetryBackoff:        5 * time.Second,
		EnableDeduplication: true,
		DeduplicationWindow: 5 * time.Minute,
	}

	rulesEngine := engine.NewRulesEngine(store, evaluator, config)
	notificationMgr := channels.NewManager(store, config)

	router := gin.Default()
	handler := api.NewHandler(store, rulesEngine, notificationMgr, config)
	handler.RegisterRoutes(router)

	cleanup := func() {
		store.Close()
	}

	return router, store, cleanup
}

func TestAPI_CreateAndGetRule(t *testing.T) {
	router, _, cleanup := setupTestServer(t)
	defer cleanup()

	// Create a rule
	rule := alerting.AlertRule{
		Name:        "Test Rule",
		Description: "Test rule for integration testing",
		Source:      alerting.SourcePerformance,
		Severity:    alerting.SeverityWarning,
		Enabled:     true,
		RuleType:    alerting.RuleTypeThreshold,
		Conditions: []alerting.Condition{
			{
				MetricName: "test_metric",
				Operator:   alerting.OpGreaterThan,
				Threshold:  100.0,
			},
		},
		EvaluationInterval: 30 * time.Second,
		ForDuration:        1 * time.Minute,
		Channels:           []string{"test-channel"},
	}

	jsonData, _ := json.Marshal(rule)

	// Create request
	req := httptest.NewRequest("POST", "/api/v1/alerts/rules/create", strings.NewReader(string(jsonData)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	// Parse response
	var createdRule alerting.AlertRule
	err := json.Unmarshal(w.Body.Bytes(), &createdRule)
	require.NoError(t, err)
	assert.NotEmpty(t, createdRule.ID)
	assert.Equal(t, "Test Rule", createdRule.Name)

	// Get the rule
	req2 := httptest.NewRequest("GET", "/api/v1/alerts/rules/"+createdRule.ID, nil)
	w2 := httptest.NewRecorder()

	router.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusOK, w2.Code)

	var fetchedRule alerting.AlertRule
	err = json.Unmarshal(w2.Body.Bytes(), &fetchedRule)
	require.NoError(t, err)
	assert.Equal(t, createdRule.ID, fetchedRule.ID)
	assert.Equal(t, "Test Rule", fetchedRule.Name)
}

func TestAPI_ListAlerts(t *testing.T) {
	router, store, cleanup := setupTestServer(t)
	defer cleanup()

	// Create some test alerts
	for i := 0; i < 5; i++ {
		alert := &alerting.Alert{
			ID:       string(rune('A' + i)),
			RuleID:   "test-rule",
			RuleName: "Test Alert",
			Source:   alerting.SourcePerformance,
			Severity: alerting.SeverityWarning,
			Status:   alerting.StatusActive,
			Message:  "Test message",
			Value:    float64(i * 10),
			Threshold: 50.0,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		}
		store.SaveAlert(alert)
	}

	// List alerts
	req := httptest.NewRequest("GET", "/api/v1/alerts?limit=10", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	require.NoError(t, err)

	alerts := response["alerts"].([]interface{})
	assert.GreaterOrEqual(t, len(alerts), 5)
}

func TestAPI_AcknowledgeAlert(t *testing.T) {
	router, store, cleanup := setupTestServer(t)
	defer cleanup()

	// Create a test alert
	alert := &alerting.Alert{
		ID:        "test-alert-ack",
		RuleID:    "test-rule",
		RuleName:  "Test Alert",
		Source:    alerting.SourcePerformance,
		Severity:  alerting.SeverityWarning,
		Status:    alerting.StatusActive,
		Message:   "Test message",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
	store.SaveAlert(alert)

	// Acknowledge the alert
	req := httptest.NewRequest("POST", "/api/v1/alerts/test-alert-ack/acknowledge", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var acknowledgedAlert alerting.Alert
	err := json.Unmarshal(w.Body.Bytes(), &acknowledgedAlert)
	require.NoError(t, err)
	assert.Equal(t, alerting.StatusAcknowledged, acknowledgedAlert.Status)
	assert.NotNil(t, acknowledgedAlert.AcknowledgedAt)
}

func TestAPI_CreateChannel(t *testing.T) {
	router, _, cleanup := setupTestServer(t)
	defer cleanup()

	// Create a webhook channel
	channel := alerting.Channel{
		Name:    "Test Webhook",
		Type:    alerting.ChannelTypeWebhook,
		Enabled: true,
		Config: map[string]interface{}{
			"url": "https://example.com/webhook",
		},
	}

	jsonData, _ := json.Marshal(channel)

	req := httptest.NewRequest("POST", "/api/v1/alerts/channels/create", strings.NewReader(string(jsonData)))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var createdChannel alerting.Channel
	err := json.Unmarshal(w.Body.Bytes(), &createdChannel)
	require.NoError(t, err)
	assert.NotEmpty(t, createdChannel.ID)
	assert.Equal(t, "Test Webhook", createdChannel.Name)
}

func TestAPI_GetAlertStats(t *testing.T) {
	router, store, cleanup := setupTestServer(t)
	defer cleanup()

	// Create alerts with different statuses
	alerts := []*alerting.Alert{
		{
			ID:        "stat-1",
			RuleID:    "test-rule",
			RuleName:  "Test",
			Source:    alerting.SourcePerformance,
			Severity:  alerting.SeverityCritical,
			Status:    alerting.StatusActive,
			Message:   "Test",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
		{
			ID:        "stat-2",
			RuleID:    "test-rule",
			RuleName:  "Test",
			Source:    alerting.SourcePerformance,
			Severity:  alerting.SeverityWarning,
			Status:    alerting.StatusAcknowledged,
			Message:   "Test",
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
		},
	}

	for _, alert := range alerts {
		store.SaveAlert(alert)
	}

	// Get stats
	req := httptest.NewRequest("GET", "/api/v1/alerts/stats", nil)
	w := httptest.NewRecorder()

	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var stats alerting.AlertStats
	err := json.Unmarshal(w.Body.Bytes(), &stats)
	require.NoError(t, err)
	assert.Greater(t, stats.TotalAlerts, 0)
	assert.NotEmpty(t, stats.BySeverity)
}

func BenchmarkAPI_ListAlerts(b *testing.B) {
	router, _, cleanup := setupTestServer(&testing.T{})
	defer cleanup()

	req := httptest.NewRequest("GET", "/api/v1/alerts?limit=100", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}
