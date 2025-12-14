package tests

import (
	"testing"
	"time"

	"github.com/paw/control-center/alerting"
	"github.com/paw/control-center/alerting/engine"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockMetricsProvider implements MetricsProvider for testing
type MockMetricsProvider struct {
	metrics map[string]float64
}

func NewMockMetricsProvider() *MockMetricsProvider {
	return &MockMetricsProvider{
		metrics: make(map[string]float64),
	}
}

func (m *MockMetricsProvider) SetMetric(name string, value float64) {
	m.metrics[name] = value
}

func (m *MockMetricsProvider) GetMetric(name string, labels map[string]string) (*alerting.MetricValue, error) {
	value, ok := m.metrics[name]
	if !ok {
		value = 0
	}

	return &alerting.MetricValue{
		Name:      name,
		Value:     value,
		Timestamp: time.Now(),
		Labels:    labels,
	}, nil
}

func (m *MockMetricsProvider) GetMetricRange(name string, labels map[string]string, start, end time.Time) ([]*alerting.MetricValue, error) {
	// Return two values: old and new
	oldValue, ok := m.metrics[name+"_old"]
	if !ok {
		oldValue = 50.0
	}

	newValue, ok := m.metrics[name]
	if !ok {
		newValue = 100.0
	}

	return []*alerting.MetricValue{
		{
			Name:      name,
			Value:     oldValue,
			Timestamp: start,
			Labels:    labels,
		},
		{
			Name:      name,
			Value:     newValue,
			Timestamp: end,
			Labels:    labels,
		},
	}, nil
}

func TestEvaluator_ThresholdRule(t *testing.T) {
	provider := NewMockMetricsProvider()
	evaluator := engine.NewEvaluator(provider)

	// Test case: value exceeds threshold
	t.Run("ValueExceedsThreshold", func(t *testing.T) {
		provider.SetMetric("cpu_usage", 85.0)

		rule := &alerting.AlertRule{
			ID:       "test-rule-1",
			Name:     "High CPU Usage",
			RuleType: alerting.RuleTypeThreshold,
			Conditions: []alerting.Condition{
				{
					MetricName: "cpu_usage",
					Operator:   alerting.OpGreaterThan,
					Threshold:  80.0,
				},
			},
			EvaluationInterval: 10 * time.Second,
			ForDuration:        0, // No duration requirement
		}

		result, err := evaluator.Evaluate(rule)
		require.NoError(t, err)
		assert.True(t, result.Triggered)
		assert.Equal(t, 85.0, result.Value)
		assert.Equal(t, 80.0, result.Threshold)
	})

	// Test case: value below threshold
	t.Run("ValueBelowThreshold", func(t *testing.T) {
		provider.SetMetric("cpu_usage", 50.0)

		rule := &alerting.AlertRule{
			ID:       "test-rule-2",
			Name:     "High CPU Usage",
			RuleType: alerting.RuleTypeThreshold,
			Conditions: []alerting.Condition{
				{
					MetricName: "cpu_usage",
					Operator:   alerting.OpGreaterThan,
					Threshold:  80.0,
				},
			},
		}

		result, err := evaluator.Evaluate(rule)
		require.NoError(t, err)
		assert.False(t, result.Triggered)
	})
}

func TestEvaluator_RateOfChangeRule(t *testing.T) {
	provider := NewMockMetricsProvider()
	evaluator := engine.NewEvaluator(provider)

	t.Run("SignificantIncrease", func(t *testing.T) {
		provider.SetMetric("request_rate_old", 100.0)
		provider.SetMetric("request_rate", 200.0) // 100% increase

		rule := &alerting.AlertRule{
			ID:       "test-rule-3",
			Name:     "Request Rate Spike",
			RuleType: alerting.RuleTypeRateOfChange,
			Conditions: []alerting.Condition{
				{
					MetricName: "request_rate",
					Operator:   alerting.OpGreaterThan,
					Threshold:  50.0, // 50% increase threshold
					Duration:   5 * time.Minute,
				},
			},
		}

		result, err := evaluator.Evaluate(rule)
		require.NoError(t, err)
		assert.True(t, result.Triggered)
		assert.Greater(t, result.Value, 50.0) // Should be 100%
	})
}

func TestEvaluator_CompositeRule(t *testing.T) {
	provider := NewMockMetricsProvider()
	evaluator := engine.NewEvaluator(provider)

	t.Run("ANDOperator_BothTrue", func(t *testing.T) {
		provider.SetMetric("cpu_usage", 85.0)
		provider.SetMetric("memory_usage", 90.0)

		rule := &alerting.AlertRule{
			ID:          "test-rule-4",
			Name:        "High Resource Usage",
			RuleType:    alerting.RuleTypeComposite,
			CompositeOp: alerting.OpAND,
			Conditions: []alerting.Condition{
				{
					MetricName: "cpu_usage",
					Operator:   alerting.OpGreaterThan,
					Threshold:  80.0,
				},
				{
					MetricName: "memory_usage",
					Operator:   alerting.OpGreaterThan,
					Threshold:  85.0,
				},
			},
		}

		result, err := evaluator.Evaluate(rule)
		require.NoError(t, err)
		assert.True(t, result.Triggered)
	})

	t.Run("ANDOperator_OneFalse", func(t *testing.T) {
		provider.SetMetric("cpu_usage", 85.0)
		provider.SetMetric("memory_usage", 50.0) // Below threshold

		rule := &alerting.AlertRule{
			ID:          "test-rule-5",
			Name:        "High Resource Usage",
			RuleType:    alerting.RuleTypeComposite,
			CompositeOp: alerting.OpAND,
			Conditions: []alerting.Condition{
				{
					MetricName: "cpu_usage",
					Operator:   alerting.OpGreaterThan,
					Threshold:  80.0,
				},
				{
					MetricName: "memory_usage",
					Operator:   alerting.OpGreaterThan,
					Threshold:  85.0,
				},
			},
		}

		result, err := evaluator.Evaluate(rule)
		require.NoError(t, err)
		assert.False(t, result.Triggered)
	})

	t.Run("OROperator_OneTrue", func(t *testing.T) {
		provider.SetMetric("cpu_usage", 85.0)
		provider.SetMetric("memory_usage", 50.0)

		rule := &alerting.AlertRule{
			ID:          "test-rule-6",
			Name:        "Resource Alert",
			RuleType:    alerting.RuleTypeComposite,
			CompositeOp: alerting.OpOR,
			Conditions: []alerting.Condition{
				{
					MetricName: "cpu_usage",
					Operator:   alerting.OpGreaterThan,
					Threshold:  80.0,
				},
				{
					MetricName: "memory_usage",
					Operator:   alerting.OpGreaterThan,
					Threshold:  85.0,
				},
			},
		}

		result, err := evaluator.Evaluate(rule)
		require.NoError(t, err)
		assert.True(t, result.Triggered)
	})
}

func TestEvaluator_ComparisonOperators(t *testing.T) {
	provider := NewMockMetricsProvider()
	evaluator := engine.NewEvaluator(provider)

	tests := []struct {
		name      string
		value     float64
		operator  alerting.ComparisonOperator
		threshold float64
		expected  bool
	}{
		{"GreaterThan_True", 85.0, alerting.OpGreaterThan, 80.0, true},
		{"GreaterThan_False", 75.0, alerting.OpGreaterThan, 80.0, false},
		{"GreaterThanOrEqual_Equal", 80.0, alerting.OpGreaterThanOrEqual, 80.0, true},
		{"GreaterThanOrEqual_Greater", 85.0, alerting.OpGreaterThanOrEqual, 80.0, true},
		{"LessThan_True", 75.0, alerting.OpLessThan, 80.0, true},
		{"LessThan_False", 85.0, alerting.OpLessThan, 80.0, false},
		{"LessThanOrEqual_Equal", 80.0, alerting.OpLessThanOrEqual, 80.0, true},
		{"LessThanOrEqual_Less", 75.0, alerting.OpLessThanOrEqual, 80.0, true},
		{"Equals_True", 80.0, alerting.OpEquals, 80.0, true},
		{"Equals_False", 85.0, alerting.OpEquals, 80.0, false},
		{"NotEquals_True", 85.0, alerting.OpNotEquals, 80.0, true},
		{"NotEquals_False", 80.0, alerting.OpNotEquals, 80.0, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider.SetMetric("test_metric", tt.value)

			rule := &alerting.AlertRule{
				ID:       "test-rule",
				Name:     "Test Rule",
				RuleType: alerting.RuleTypeThreshold,
				Conditions: []alerting.Condition{
					{
						MetricName: "test_metric",
						Operator:   tt.operator,
						Threshold:  tt.threshold,
					},
				},
			}

			result, err := evaluator.Evaluate(rule)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result.Triggered,
				"Expected triggered=%v for value=%.1f %s %.1f",
				tt.expected, tt.value, tt.operator, tt.threshold)
		})
	}
}

func TestEvaluator_ForDuration(t *testing.T) {
	provider := NewMockMetricsProvider()
	evaluator := engine.NewEvaluator(provider)
	provider.SetMetric("cpu_usage", 85.0)

	rule := &alerting.AlertRule{
		ID:          "test-rule-duration",
		Name:        "High CPU for Duration",
		RuleType:    alerting.RuleTypeThreshold,
		ForDuration: 2 * time.Second,
		Conditions: []alerting.Condition{
			{
				MetricName: "cpu_usage",
				Operator:   alerting.OpGreaterThan,
				Threshold:  80.0,
			},
		},
	}

	// First evaluation - should not trigger (duration not met)
	result1, err := evaluator.Evaluate(rule)
	require.NoError(t, err)
	assert.False(t, result1.Triggered)

	// Wait for duration
	time.Sleep(2 * time.Second)

	// Second evaluation - should trigger (duration met)
	result2, err := evaluator.Evaluate(rule)
	require.NoError(t, err)
	assert.True(t, result2.Triggered)
}

func BenchmarkEvaluator_ThresholdRule(b *testing.B) {
	provider := NewMockMetricsProvider()
	evaluator := engine.NewEvaluator(provider)
	provider.SetMetric("cpu_usage", 85.0)

	rule := &alerting.AlertRule{
		ID:       "bench-rule",
		Name:     "Benchmark Rule",
		RuleType: alerting.RuleTypeThreshold,
		Conditions: []alerting.Condition{
			{
				MetricName: "cpu_usage",
				Operator:   alerting.OpGreaterThan,
				Threshold:  80.0,
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		evaluator.Evaluate(rule)
	}
}
