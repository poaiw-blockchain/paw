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

// evaluatePattern evaluates pattern-based rules using time-series analysis
func (e *Evaluator) evaluatePattern(rule *alerting.AlertRule) (*alerting.EvaluationResult, error) {
	result := &alerting.EvaluationResult{
		RuleID:    rule.ID,
		Triggered: false,
		Timestamp: time.Now(),
		Metadata:  make(map[string]interface{}),
	}

	for _, condition := range rule.Conditions {
		// Get historical data for pattern analysis
		end := time.Now()
		duration := condition.Duration
		if duration == 0 {
			duration = 1 * time.Hour // Default window
		}
		start := end.Add(-duration)

		metrics, err := e.metricsProvider.GetMetricRange(condition.MetricName, rule.Labels, start, end)
		if err != nil {
			return nil, fmt.Errorf("failed to get metric range for pattern: %w", err)
		}

		if len(metrics) < 3 {
			result.Message = "Insufficient data points for pattern analysis (minimum 3 required)"
			return result, nil
		}

		// Extract values from metrics
		values := make([]float64, len(metrics))
		for i, m := range metrics {
			values[i] = m.Value
		}

		// Determine pattern type from condition operator
		patternType := e.getPatternType(condition.Operator)

		// Evaluate based on pattern type
		triggered, details := e.detectPattern(values, patternType, condition.Threshold)

		result.Metadata["pattern_type"] = patternType
		result.Metadata["data_points"] = len(values)
		for k, v := range details {
			result.Metadata[k] = v
		}

		if triggered {
			result.Triggered = true
			result.Value = details["current_value"].(float64)
			result.Threshold = condition.Threshold
			result.Message = fmt.Sprintf("Pattern '%s' detected in %s: %s",
				patternType, condition.MetricName, details["description"])
			break
		}
	}

	if !result.Triggered {
		result.Message = "No anomalous patterns detected"
	}

	return result, nil
}

// getPatternType determines pattern type from operator
func (e *Evaluator) getPatternType(op alerting.ComparisonOperator) string {
	switch op {
	case alerting.OpGreaterThan, alerting.OpGreaterThanOrEqual:
		return "spike"
	case alerting.OpLessThan, alerting.OpLessThanOrEqual:
		return "drop"
	default:
		return "anomaly"
	}
}

// detectPattern runs pattern detection algorithms
func (e *Evaluator) detectPattern(values []float64, patternType string, threshold float64) (bool, map[string]interface{}) {
	details := make(map[string]interface{})

	// Calculate statistics
	mean, stdDev := e.calculateMeanStdDev(values)
	currentValue := values[len(values)-1]

	details["mean"] = mean
	details["std_dev"] = stdDev
	details["current_value"] = currentValue

	// Z-score for current value
	zScore := 0.0
	if stdDev > 0 {
		zScore = (currentValue - mean) / stdDev
	}
	details["z_score"] = zScore

	// IQR-based outlier detection
	q1, q3 := e.calculateQuartiles(values)
	iqr := q3 - q1
	lowerBound := q1 - 1.5*iqr
	upperBound := q3 + 1.5*iqr
	details["iqr"] = iqr
	details["lower_bound"] = lowerBound
	details["upper_bound"] = upperBound

	// Moving average trend detection
	shortMA := e.movingAverage(values, 3)
	longMA := e.movingAverage(values, min(10, len(values)))
	trend := "stable"
	if len(shortMA) > 0 && len(longMA) > 0 {
		if shortMA[len(shortMA)-1] > longMA[len(longMA)-1]*1.05 {
			trend = "upward"
		} else if shortMA[len(shortMA)-1] < longMA[len(longMA)-1]*0.95 {
			trend = "downward"
		}
	}
	details["trend"] = trend

	// Pattern-specific detection
	switch patternType {
	case "spike":
		// Detect sudden spike: Z-score > threshold or above IQR upper bound
		if zScore > threshold {
			details["description"] = fmt.Sprintf("Z-score %.2f exceeds threshold %.2f", zScore, threshold)
			return true, details
		}
		if currentValue > upperBound && threshold <= 1.5 {
			details["description"] = fmt.Sprintf("Value %.2f exceeds IQR upper bound %.2f", currentValue, upperBound)
			return true, details
		}
		// Check for percentage spike from moving average
		if len(longMA) > 0 {
			pctChange := ((currentValue - longMA[len(longMA)-1]) / longMA[len(longMA)-1]) * 100
			details["pct_change_from_ma"] = pctChange
			if pctChange > threshold*10 { // threshold as multiplier
				details["description"] = fmt.Sprintf("Value spiked %.1f%% above moving average", pctChange)
				return true, details
			}
		}

	case "drop":
		// Detect sudden drop: Z-score < -threshold or below IQR lower bound
		if zScore < -threshold {
			details["description"] = fmt.Sprintf("Z-score %.2f below -%.2f threshold", zScore, threshold)
			return true, details
		}
		if currentValue < lowerBound && threshold <= 1.5 {
			details["description"] = fmt.Sprintf("Value %.2f below IQR lower bound %.2f", currentValue, lowerBound)
			return true, details
		}
		// Check for percentage drop from moving average
		if len(longMA) > 0 {
			pctChange := ((currentValue - longMA[len(longMA)-1]) / longMA[len(longMA)-1]) * 100
			details["pct_change_from_ma"] = pctChange
			if pctChange < -threshold*10 {
				details["description"] = fmt.Sprintf("Value dropped %.1f%% below moving average", -pctChange)
				return true, details
			}
		}

	case "anomaly":
		// General anomaly: |Z-score| > threshold
		if math.Abs(zScore) > threshold {
			direction := "above"
			if zScore < 0 {
				direction = "below"
			}
			details["description"] = fmt.Sprintf("Anomaly detected: value %.2f (%s mean by %.1f std devs)",
				currentValue, direction, math.Abs(zScore))
			return true, details
		}
		// Also check IQR bounds
		if currentValue < lowerBound || currentValue > upperBound {
			details["description"] = fmt.Sprintf("Value %.2f outside IQR bounds [%.2f, %.2f]",
				currentValue, lowerBound, upperBound)
			return true, details
		}
	}

	details["description"] = "No pattern detected"
	return false, details
}

// calculateMeanStdDev calculates mean and standard deviation
func (e *Evaluator) calculateMeanStdDev(values []float64) (mean, stdDev float64) {
	if len(values) == 0 {
		return 0, 0
	}

	// Calculate mean
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	mean = sum / float64(len(values))

	// Calculate standard deviation
	if len(values) < 2 {
		return mean, 0
	}
	sumSquares := 0.0
	for _, v := range values {
		diff := v - mean
		sumSquares += diff * diff
	}
	stdDev = math.Sqrt(sumSquares / float64(len(values)-1))

	return mean, stdDev
}

// calculateQuartiles calculates Q1 and Q3 for IQR
func (e *Evaluator) calculateQuartiles(values []float64) (q1, q3 float64) {
	if len(values) < 4 {
		// For small datasets, use simpler approximation
		sorted := make([]float64, len(values))
		copy(sorted, values)
		e.sortFloat64s(sorted)
		if len(sorted) == 0 {
			return 0, 0
		}
		q1 = sorted[len(sorted)/4]
		q3 = sorted[len(sorted)*3/4]
		return q1, q3
	}

	sorted := make([]float64, len(values))
	copy(sorted, values)
	e.sortFloat64s(sorted)

	n := len(sorted)
	q1Idx := n / 4
	q3Idx := (3 * n) / 4

	q1 = sorted[q1Idx]
	q3 = sorted[q3Idx]

	return q1, q3
}

// sortFloat64s sorts a slice of float64 in ascending order
func (e *Evaluator) sortFloat64s(values []float64) {
	for i := 0; i < len(values)-1; i++ {
		for j := i + 1; j < len(values); j++ {
			if values[j] < values[i] {
				values[i], values[j] = values[j], values[i]
			}
		}
	}
}

// movingAverage calculates simple moving average
func (e *Evaluator) movingAverage(values []float64, window int) []float64 {
	if len(values) < window || window < 1 {
		return nil
	}

	result := make([]float64, len(values)-window+1)
	sum := 0.0

	// Initial window sum
	for i := 0; i < window; i++ {
		sum += values[i]
	}
	result[0] = sum / float64(window)

	// Slide window
	for i := window; i < len(values); i++ {
		sum = sum - values[i-window] + values[i]
		result[i-window+1] = sum / float64(window)
	}

	return result
}

// min returns the minimum of two ints
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
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
