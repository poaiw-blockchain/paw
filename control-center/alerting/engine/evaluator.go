package engine

import (
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/paw/control-center/alerting"
)

// Evaluator evaluates alert rules against metrics
type Evaluator struct {
	metricsProvider MetricsProvider
	stateStore      *StateStore
	mu              sync.RWMutex
}

// MetricsProvider interface for fetching metrics
type MetricsProvider interface {
	GetMetric(name string, labels map[string]string) (*alerting.MetricValue, error)
	GetMetricRange(name string, labels map[string]string, start, end time.Time) ([]*alerting.MetricValue, error)
}

// StateStore tracks alert state for deduplication and for_duration
type StateStore struct {
	activeAlerts map[string]*alertState
	mu           sync.RWMutex
}

type alertState struct {
	firstTriggeredAt time.Time
	lastEvaluatedAt  time.Time
	consecutiveTrue  int
	value            float64
}

// NewEvaluator creates a new rule evaluator
func NewEvaluator(provider MetricsProvider) *Evaluator {
	return &Evaluator{
		metricsProvider: provider,
		stateStore: &StateStore{
			activeAlerts: make(map[string]*alertState),
		},
	}
}

// Evaluate evaluates a rule and returns the result
func (e *Evaluator) Evaluate(rule *alerting.AlertRule) (*alerting.EvaluationResult, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	result := &alerting.EvaluationResult{
		RuleID:    rule.ID,
		Triggered: false,
		Timestamp: time.Now(),
		Metadata:  make(map[string]interface{}),
	}

	switch rule.RuleType {
	case alerting.RuleTypeThreshold:
		return e.evaluateThreshold(rule)
	case alerting.RuleTypeRateOfChange:
		return e.evaluateRateOfChange(rule)
	case alerting.RuleTypePattern:
		return e.evaluatePattern(rule)
	case alerting.RuleTypeComposite:
		return e.evaluateComposite(rule)
	default:
		return nil, fmt.Errorf("unsupported rule type: %s", rule.RuleType)
	}
}

// evaluateThreshold evaluates threshold-based rules
func (e *Evaluator) evaluateThreshold(rule *alerting.AlertRule) (*alerting.EvaluationResult, error) {
	result := &alerting.EvaluationResult{
		RuleID:    rule.ID,
		Triggered: false,
		Timestamp: time.Now(),
		Metadata:  make(map[string]interface{}),
	}

	// For threshold rules, all conditions must be met
	allMet := true
	var totalValue float64
	var totalThreshold float64

	for _, condition := range rule.Conditions {
		metric, err := e.metricsProvider.GetMetric(condition.MetricName, rule.Labels)
		if err != nil {
			return nil, fmt.Errorf("failed to get metric %s: %w", condition.MetricName, err)
		}

		conditionMet := e.evaluateCondition(metric.Value, condition.Operator, condition.Threshold)
		if !conditionMet {
			allMet = false
			break
		}

		totalValue += metric.Value
		totalThreshold += condition.Threshold
	}

	result.Triggered = allMet
	result.Value = totalValue
	result.Threshold = totalThreshold

	if allMet {
		// Check for_duration
		if rule.ForDuration > 0 {
			if e.shouldTriggerAfterDuration(rule.ID, result.Timestamp, rule.ForDuration) {
				result.Message = fmt.Sprintf("Alert %s triggered: value %.2f exceeds threshold %.2f for %s",
					rule.Name, totalValue, totalThreshold, rule.ForDuration)
			} else {
				result.Triggered = false
				result.Message = fmt.Sprintf("Condition met but waiting for duration: %.2f/%.2f",
					totalValue, totalThreshold)
			}
		} else {
			result.Message = fmt.Sprintf("Alert %s triggered: value %.2f exceeds threshold %.2f",
				rule.Name, totalValue, totalThreshold)
		}
	} else {
		e.clearAlertState(rule.ID)
		result.Message = "Condition not met"
	}

	return result, nil
}

// evaluateRateOfChange evaluates rate-of-change rules
func (e *Evaluator) evaluateRateOfChange(rule *alerting.AlertRule) (*alerting.EvaluationResult, error) {
	result := &alerting.EvaluationResult{
		RuleID:    rule.ID,
		Triggered: false,
		Timestamp: time.Now(),
		Metadata:  make(map[string]interface{}),
	}

	for _, condition := range rule.Conditions {
		// Get historical data
		end := time.Now()
		start := end.Add(-condition.Duration)

		metrics, err := e.metricsProvider.GetMetricRange(condition.MetricName, rule.Labels, start, end)
		if err != nil {
			return nil, fmt.Errorf("failed to get metric range: %w", err)
		}

		if len(metrics) < 2 {
			result.Message = "Insufficient data points for rate of change"
			return result, nil
		}

		// Calculate rate of change
		oldValue := metrics[0].Value
		newValue := metrics[len(metrics)-1].Value
		rateOfChange := ((newValue - oldValue) / oldValue) * 100 // Percentage change

		result.Value = rateOfChange
		result.Threshold = condition.Threshold

		conditionMet := e.evaluateCondition(rateOfChange, condition.Operator, condition.Threshold)
		if conditionMet {
			result.Triggered = true
			result.Message = fmt.Sprintf("Alert %s triggered: rate of change %.2f%% exceeds threshold %.2f%%",
				rule.Name, rateOfChange, condition.Threshold)
			result.Metadata["old_value"] = oldValue
			result.Metadata["new_value"] = newValue
			break
		}
	}

	if !result.Triggered {
		result.Message = "Rate of change within normal bounds"
	}

	return result, nil
}

// evaluatePattern evaluates pattern-based rules
func (e *Evaluator) evaluatePattern(rule *alerting.AlertRule) (*alerting.EvaluationResult, error) {
	result := &alerting.EvaluationResult{
		RuleID:    rule.ID,
		Triggered: false,
		Timestamp: time.Now(),
		Metadata:  make(map[string]interface{}),
		Message:   "Pattern matching not yet implemented",
	}

	// TODO: Implement pattern matching logic
	// This would involve time-series analysis, anomaly detection, etc.

	return result, nil
}

// evaluateComposite evaluates composite rules (multiple conditions with AND/OR)
func (e *Evaluator) evaluateComposite(rule *alerting.AlertRule) (*alerting.EvaluationResult, error) {
	result := &alerting.EvaluationResult{
		RuleID:    rule.ID,
		Triggered: false,
		Timestamp: time.Now(),
		Metadata:  make(map[string]interface{}),
	}

	conditionResults := make([]bool, len(rule.Conditions))
	messages := []string{}

	for i, condition := range rule.Conditions {
		metric, err := e.metricsProvider.GetMetric(condition.MetricName, rule.Labels)
		if err != nil {
			return nil, fmt.Errorf("failed to get metric %s: %w", condition.MetricName, err)
		}

		conditionMet := e.evaluateCondition(metric.Value, condition.Operator, condition.Threshold)
		conditionResults[i] = conditionMet

		if conditionMet {
			messages = append(messages, fmt.Sprintf("%s: %.2f %s %.2f",
				condition.MetricName, metric.Value, condition.Operator, condition.Threshold))
		}
	}

	// Apply composite operator
	if rule.CompositeOp == alerting.OpAND {
		result.Triggered = e.allTrue(conditionResults)
	} else if rule.CompositeOp == alerting.OpOR {
		result.Triggered = e.anyTrue(conditionResults)
	}

	if result.Triggered {
		result.Message = fmt.Sprintf("Composite alert %s triggered (%s): %v",
			rule.Name, rule.CompositeOp, messages)
	} else {
		result.Message = "Composite condition not met"
	}

	return result, nil
}

// evaluateCondition evaluates a single condition
func (e *Evaluator) evaluateCondition(value float64, operator alerting.ComparisonOperator, threshold float64) bool {
	switch operator {
	case alerting.OpGreaterThan:
		return value > threshold
	case alerting.OpGreaterThanOrEqual:
		return value >= threshold
	case alerting.OpLessThan:
		return value < threshold
	case alerting.OpLessThanOrEqual:
		return value <= threshold
	case alerting.OpEquals:
		return math.Abs(value-threshold) < 0.0001 // Float comparison with epsilon
	case alerting.OpNotEquals:
		return math.Abs(value-threshold) >= 0.0001
	default:
		return false
	}
}

// shouldTriggerAfterDuration checks if alert should trigger after for_duration
func (e *Evaluator) shouldTriggerAfterDuration(ruleID string, now time.Time, forDuration time.Duration) bool {
	e.stateStore.mu.Lock()
	defer e.stateStore.mu.Unlock()

	state, exists := e.stateStore.activeAlerts[ruleID]
	if !exists {
		// First time this alert is triggered
		e.stateStore.activeAlerts[ruleID] = &alertState{
			firstTriggeredAt: now,
			lastEvaluatedAt:  now,
			consecutiveTrue:  1,
		}
		return false
	}

	// Update state
	state.lastEvaluatedAt = now
	state.consecutiveTrue++

	// Check if duration has elapsed
	elapsed := now.Sub(state.firstTriggeredAt)
	return elapsed >= forDuration
}

// clearAlertState clears the state for a rule
func (e *Evaluator) clearAlertState(ruleID string) {
	e.stateStore.mu.Lock()
	defer e.stateStore.mu.Unlock()
	delete(e.stateStore.activeAlerts, ruleID)
}

// allTrue returns true if all booleans in slice are true
func (e *Evaluator) allTrue(values []bool) bool {
	for _, v := range values {
		if !v {
			return false
		}
	}
	return len(values) > 0
}

// anyTrue returns true if any boolean in slice is true
func (e *Evaluator) anyTrue(values []bool) bool {
	for _, v := range values {
		if v {
			return true
		}
	}
	return false
}
