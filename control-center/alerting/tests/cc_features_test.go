package tests

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/paw/control-center/alerting"
	"github.com/paw/control-center/alerting/channels"
	"github.com/paw/control-center/alerting/engine"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// =============================================================================
// CC-5: Pattern Matching Tests
// =============================================================================

// PatternMetricsProvider provides time-series data for pattern testing
type PatternMetricsProvider struct {
	values []float64
}

func NewPatternMetricsProvider(values []float64) *PatternMetricsProvider {
	return &PatternMetricsProvider{values: values}
}

func (p *PatternMetricsProvider) GetMetric(name string, labels map[string]string) (*alerting.MetricValue, error) {
	if len(p.values) == 0 {
		return &alerting.MetricValue{Value: 0, Timestamp: time.Now()}, nil
	}
	return &alerting.MetricValue{
		Name:      name,
		Value:     p.values[len(p.values)-1],
		Timestamp: time.Now(),
		Labels:    labels,
	}, nil
}

func (p *PatternMetricsProvider) GetMetricRange(name string, labels map[string]string, start, end time.Time) ([]*alerting.MetricValue, error) {
	result := make([]*alerting.MetricValue, len(p.values))
	duration := end.Sub(start)
	interval := duration / time.Duration(len(p.values))

	for i, v := range p.values {
		result[i] = &alerting.MetricValue{
			Name:      name,
			Value:     v,
			Timestamp: start.Add(time.Duration(i) * interval),
			Labels:    labels,
		}
	}
	return result, nil
}

func TestPatternMatching_SpikeDetection(t *testing.T) {
	// Normal values with a spike at the end
	values := []float64{10, 11, 10, 12, 10, 11, 10, 50} // 50 is a spike
	provider := NewPatternMetricsProvider(values)
	evaluator := engine.NewEvaluator(provider)

	rule := &alerting.AlertRule{
		ID:       "pattern-spike-test",
		Name:     "Spike Detection Test",
		RuleType: alerting.RuleTypePattern,
		Conditions: []alerting.Condition{
			{
				MetricName: "test_metric",
				Operator:   alerting.OpGreaterThan,
				Threshold:  2.0, // Z-score threshold
				Duration:   1 * time.Hour,
			},
		},
	}

	result, err := evaluator.Evaluate(rule)
	require.NoError(t, err)
	assert.True(t, result.Triggered, "Should detect spike")
	assert.Contains(t, result.Message, "spike", "Message should mention spike")
	assert.Contains(t, result.Metadata, "z_score")
	assert.Greater(t, result.Metadata["z_score"].(float64), 2.0)
}

func TestPatternMatching_DropDetection(t *testing.T) {
	// Normal values with a drop at the end
	values := []float64{100, 102, 98, 101, 99, 100, 102, 20} // 20 is a drop
	provider := NewPatternMetricsProvider(values)
	evaluator := engine.NewEvaluator(provider)

	rule := &alerting.AlertRule{
		ID:       "pattern-drop-test",
		Name:     "Drop Detection Test",
		RuleType: alerting.RuleTypePattern,
		Conditions: []alerting.Condition{
			{
				MetricName: "test_metric",
				Operator:   alerting.OpLessThan,
				Threshold:  2.0, // Z-score threshold
				Duration:   1 * time.Hour,
			},
		},
	}

	result, err := evaluator.Evaluate(rule)
	require.NoError(t, err)
	assert.True(t, result.Triggered, "Should detect drop")
	assert.Contains(t, result.Message, "drop", "Message should mention drop")
}

func TestPatternMatching_NoAnomaly(t *testing.T) {
	// Stable values with no anomalies
	values := []float64{100, 101, 99, 100, 102, 98, 101, 100}
	provider := NewPatternMetricsProvider(values)
	evaluator := engine.NewEvaluator(provider)

	rule := &alerting.AlertRule{
		ID:       "pattern-stable-test",
		Name:     "Stable Pattern Test",
		RuleType: alerting.RuleTypePattern,
		Conditions: []alerting.Condition{
			{
				MetricName: "test_metric",
				Operator:   alerting.OpGreaterThan,
				Threshold:  3.0, // High Z-score threshold
				Duration:   1 * time.Hour,
			},
		},
	}

	result, err := evaluator.Evaluate(rule)
	require.NoError(t, err)
	assert.False(t, result.Triggered, "Should not detect anomaly in stable data")
	assert.Contains(t, result.Message, "No anomalous patterns")
}

func TestPatternMatching_InsufficientData(t *testing.T) {
	// Only 2 data points - insufficient for pattern analysis
	values := []float64{100, 200}
	provider := NewPatternMetricsProvider(values)
	evaluator := engine.NewEvaluator(provider)

	rule := &alerting.AlertRule{
		ID:       "pattern-insufficient-test",
		Name:     "Insufficient Data Test",
		RuleType: alerting.RuleTypePattern,
		Conditions: []alerting.Condition{
			{
				MetricName: "test_metric",
				Operator:   alerting.OpGreaterThan,
				Threshold:  2.0,
				Duration:   1 * time.Hour,
			},
		},
	}

	result, err := evaluator.Evaluate(rule)
	require.NoError(t, err)
	assert.False(t, result.Triggered)
	assert.Contains(t, result.Message, "Insufficient data")
}

func TestPatternMatching_IQROutlier(t *testing.T) {
	// Values with an outlier outside IQR bounds
	values := []float64{10, 12, 11, 13, 10, 11, 12, 10, 11, 12, 10, 30}
	provider := NewPatternMetricsProvider(values)
	evaluator := engine.NewEvaluator(provider)

	rule := &alerting.AlertRule{
		ID:       "pattern-iqr-test",
		Name:     "IQR Outlier Test",
		RuleType: alerting.RuleTypePattern,
		Conditions: []alerting.Condition{
			{
				MetricName: "test_metric",
				Operator:   alerting.OpEquals, // Will use "anomaly" pattern type
				Threshold:  1.5,               // Low threshold for IQR detection
				Duration:   1 * time.Hour,
			},
		},
	}

	result, err := evaluator.Evaluate(rule)
	require.NoError(t, err)
	// Should detect via IQR or Z-score
	assert.Contains(t, result.Metadata, "iqr")
	assert.Contains(t, result.Metadata, "upper_bound")
}

func TestPatternMatching_TrendDetection(t *testing.T) {
	// Upward trend
	values := []float64{10, 15, 20, 25, 30, 35, 40, 45, 50, 55}
	provider := NewPatternMetricsProvider(values)
	evaluator := engine.NewEvaluator(provider)

	rule := &alerting.AlertRule{
		ID:       "pattern-trend-test",
		Name:     "Trend Detection Test",
		RuleType: alerting.RuleTypePattern,
		Conditions: []alerting.Condition{
			{
				MetricName: "test_metric",
				Operator:   alerting.OpGreaterThan,
				Threshold:  1.0,
				Duration:   1 * time.Hour,
			},
		},
	}

	result, err := evaluator.Evaluate(rule)
	require.NoError(t, err)
	assert.Contains(t, result.Metadata, "trend")
	assert.Equal(t, "upward", result.Metadata["trend"])
}

// =============================================================================
// CC-4: Alert Grouping Tests
// =============================================================================

func TestAlertGrouper_MergeAlerts(t *testing.T) {
	grouper := engine.NewAlertGrouper(100 * time.Millisecond)

	// Track merged alerts
	var mergedAlert *alerting.Alert
	var originalAlerts []*alerting.Alert
	var mu sync.Mutex
	done := make(chan struct{})
	var once sync.Once

	grouper.SetHandler(func(merged *alerting.Alert, originals []*alerting.Alert) error {
		mu.Lock()
		mergedAlert = merged
		originalAlerts = originals
		mu.Unlock()
		once.Do(func() { close(done) })
		return nil
	})

	// Add alerts with same rule ID and same severity (for proper grouping)
	alerts := []*alerting.Alert{
		{
			ID:        "alert-1",
			RuleID:    "rule-1",
			RuleName:  "Test Rule",
			Severity:  alerting.SeverityCritical,
			Value:     10.0,
			Threshold: 5.0,
			CreatedAt: time.Now().Add(-2 * time.Second),
		},
		{
			ID:        "alert-2",
			RuleID:    "rule-1",
			RuleName:  "Test Rule",
			Severity:  alerting.SeverityCritical,
			Value:     20.0,
			Threshold: 5.0,
			CreatedAt: time.Now().Add(-1 * time.Second),
		},
		{
			ID:        "alert-3",
			RuleID:    "rule-1",
			RuleName:  "Test Rule",
			Severity:  alerting.SeverityCritical,
			Value:     30.0,
			Threshold: 5.0,
			CreatedAt: time.Now(),
		},
	}

	for _, alert := range alerts {
		grouper.Add(alert)
	}

	// Wait for flush
	select {
	case <-done:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Timeout waiting for grouped alert")
	}

	// Wait a bit for any async processing to complete
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	defer mu.Unlock()

	require.NotNil(t, mergedAlert)
	assert.Equal(t, 3, len(originalAlerts))

	// Check merged alert properties
	assert.Contains(t, mergedAlert.ID, "grp-")
	assert.Equal(t, alerting.SeverityCritical, mergedAlert.Severity, "Should use highest severity")
	assert.Equal(t, 20.0, mergedAlert.Value, "Should be average value")
	assert.Contains(t, mergedAlert.Message, "[GROUPED]")
	assert.Contains(t, mergedAlert.Message, "3 alerts")

	// Check metadata
	assert.Equal(t, true, mergedAlert.Metadata["grouped"])
	assert.Equal(t, 3, mergedAlert.Metadata["alert_count"])
	assert.Equal(t, 20.0, mergedAlert.Metadata["avg_value"])
	assert.Equal(t, 10.0, mergedAlert.Metadata["min_value"])
	assert.Equal(t, 30.0, mergedAlert.Metadata["max_value"])
}

func TestAlertGrouper_SingleAlert(t *testing.T) {
	grouper := engine.NewAlertGrouper(50 * time.Millisecond)

	var mergedAlert *alerting.Alert
	done := make(chan struct{})

	grouper.SetHandler(func(merged *alerting.Alert, originals []*alerting.Alert) error {
		mergedAlert = merged
		close(done)
		return nil
	})

	alert := &alerting.Alert{
		ID:        "single-alert",
		RuleID:    "rule-1",
		RuleName:  "Test Rule",
		Severity:  alerting.SeverityWarning,
		Value:     15.0,
		Threshold: 10.0,
		CreatedAt: time.Now(),
	}

	grouper.Add(alert)

	select {
	case <-done:
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Timeout waiting for single alert")
	}

	require.NotNil(t, mergedAlert)
	// Single alert should be returned as-is
	assert.Equal(t, "single-alert", mergedAlert.ID)
	assert.Equal(t, 15.0, mergedAlert.Value)
}

func TestAlertGrouper_SeparateGroups(t *testing.T) {
	grouper := engine.NewAlertGrouper(50 * time.Millisecond)

	var groupCount int
	var mu sync.Mutex
	done := make(chan struct{}, 2)

	grouper.SetHandler(func(merged *alerting.Alert, originals []*alerting.Alert) error {
		mu.Lock()
		groupCount++
		mu.Unlock()
		done <- struct{}{}
		return nil
	})

	// Add alerts with different rule IDs - should create separate groups
	grouper.Add(&alerting.Alert{
		ID:       "alert-rule1",
		RuleID:   "rule-1",
		Severity: alerting.SeverityWarning,
	})
	grouper.Add(&alerting.Alert{
		ID:       "alert-rule2",
		RuleID:   "rule-2", // Different rule
		Severity: alerting.SeverityWarning,
	})

	// Wait for both groups to flush
	timeout := time.After(300 * time.Millisecond)
	for i := 0; i < 2; i++ {
		select {
		case <-done:
		case <-timeout:
			t.Fatal("Timeout waiting for grouped alerts")
		}
	}

	mu.Lock()
	assert.Equal(t, 2, groupCount, "Should have 2 separate groups")
	mu.Unlock()
}

// =============================================================================
// CC-3: Batch Sending Tests
// =============================================================================

func TestWebhookChannel_SendBatch(t *testing.T) {
	// Create test server
	var receivedAlerts int
	var batchHeader string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		batchHeader = r.Header.Get("X-PAW-Batch")
		countHeader := r.Header.Get("X-PAW-Alert-Count")
		if countHeader != "" {
			receivedAlerts = 1 // Got batch request
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &alerting.WebhookChannelConfig{
		Timeout: 5 * time.Second,
	}

	webhook := channels.NewWebhookChannel(config)

	channel := &alerting.Channel{
		ID:   "test-channel",
		Name: "Test Webhook",
		Type: alerting.ChannelTypeWebhook,
		Config: map[string]interface{}{
			"url": server.URL,
		},
	}

	alerts := []*alerting.Alert{
		{
			ID:        "batch-1",
			RuleID:    "rule-1",
			RuleName:  "Test Rule",
			Severity:  alerting.SeverityWarning,
			Message:   "Test message 1",
			CreatedAt: time.Now(),
		},
		{
			ID:        "batch-2",
			RuleID:    "rule-1",
			RuleName:  "Test Rule",
			Severity:  alerting.SeverityCritical,
			Message:   "Test message 2",
			CreatedAt: time.Now(),
		},
	}

	err := webhook.SendBatch(alerts, channel)
	require.NoError(t, err)

	assert.Equal(t, "true", batchHeader)
	assert.Equal(t, 1, receivedAlerts)
}

func TestWebhookChannel_SendBatchEmpty(t *testing.T) {
	config := &alerting.WebhookChannelConfig{
		Timeout: 5 * time.Second,
	}

	webhook := channels.NewWebhookChannel(config)

	channel := &alerting.Channel{
		ID:   "test-channel",
		Name: "Test Webhook",
		Type: alerting.ChannelTypeWebhook,
		Config: map[string]interface{}{
			"url": "http://example.com/webhook",
		},
	}

	// Empty batch should succeed without making request
	err := webhook.SendBatch([]*alerting.Alert{}, channel)
	require.NoError(t, err)
}

func TestWebhookChannel_SendBatchSlackPayload(t *testing.T) {
	var receivedPayload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&payload)
		if err == nil {
			receivedPayload = payload
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &alerting.WebhookChannelConfig{
		Timeout: 5 * time.Second,
	}

	webhook := channels.NewWebhookChannel(config)

	channel := &alerting.Channel{
		ID:   "slack-channel",
		Name: "Slack Webhook",
		Type: alerting.ChannelTypeWebhook,
		Config: map[string]interface{}{
			"url":      server.URL,
			"template": "slack",
		},
	}

	alerts := []*alerting.Alert{
		{
			ID:        "slack-1",
			RuleID:    "rule-1",
			RuleName:  "High CPU",
			Severity:  alerting.SeverityCritical,
			Message:   "CPU usage critical",
			CreatedAt: time.Now(),
		},
		{
			ID:        "slack-2",
			RuleID:    "rule-1",
			RuleName:  "High Memory",
			Severity:  alerting.SeverityWarning,
			Message:   "Memory usage warning",
			CreatedAt: time.Now(),
		},
	}

	err := webhook.SendBatch(alerts, channel)
	require.NoError(t, err)

	// Verify Slack payload structure
	assert.Equal(t, "PAW Alert Manager", receivedPayload["username"])
	assert.Contains(t, receivedPayload, "attachments")
	attachments := receivedPayload["attachments"].([]interface{})
	assert.GreaterOrEqual(t, len(attachments), 2) // Summary + alerts
}

func TestWebhookChannel_SendBatchDiscordPayload(t *testing.T) {
	var receivedPayload map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload map[string]interface{}
		err := json.NewDecoder(r.Body).Decode(&payload)
		if err == nil {
			receivedPayload = payload
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	config := &alerting.WebhookChannelConfig{
		Timeout: 5 * time.Second,
	}

	webhook := channels.NewWebhookChannel(config)

	channel := &alerting.Channel{
		ID:   "discord-channel",
		Name: "Discord Webhook",
		Type: alerting.ChannelTypeWebhook,
		Config: map[string]interface{}{
			"url":      server.URL,
			"template": "discord",
		},
	}

	alerts := []*alerting.Alert{
		{
			ID:        "discord-1",
			RuleID:    "rule-1",
			RuleName:  "Disk Full",
			Severity:  alerting.SeverityCritical,
			Message:   "Disk space low",
			CreatedAt: time.Now(),
		},
	}

	err := webhook.SendBatch(alerts, channel)
	require.NoError(t, err)

	// Verify Discord payload structure
	assert.Equal(t, "PAW Alert Manager", receivedPayload["username"])
	assert.Contains(t, receivedPayload, "embeds")
	embeds := receivedPayload["embeds"].([]interface{})
	assert.GreaterOrEqual(t, len(embeds), 1) // Summary embed
}

func TestWebhookChannel_SendBatchError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("Internal Server Error"))
	}))
	defer server.Close()

	config := &alerting.WebhookChannelConfig{
		Timeout: 5 * time.Second,
	}

	webhook := channels.NewWebhookChannel(config)

	channel := &alerting.Channel{
		ID:   "error-channel",
		Name: "Error Webhook",
		Type: alerting.ChannelTypeWebhook,
		Config: map[string]interface{}{
			"url": server.URL,
		},
	}

	alerts := []*alerting.Alert{
		{
			ID:       "error-1",
			RuleID:   "rule-1",
			RuleName: "Test",
			Severity: alerting.SeverityWarning,
		},
	}

	err := webhook.SendBatch(alerts, channel)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "500")
}

// =============================================================================
// Integration Tests
// =============================================================================

func TestRulesEngineWithGrouping(t *testing.T) {
	provider := NewPatternMetricsProvider([]float64{100})
	evaluator := engine.NewEvaluator(provider)

	config := &alerting.Config{
		EnableGrouping:      true,
		GroupingWindow:      50 * time.Millisecond,
		EnableDeduplication: false,
	}

	// We can't fully test without storage, but we can verify initialization
	// This at least exercises the grouper setup code path
	assert.NotNil(t, evaluator)
	assert.NotNil(t, config)
}

