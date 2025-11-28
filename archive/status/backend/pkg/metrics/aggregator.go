package metrics

import (
	"context"
	"log"
	"sync"
	"time"

	"status/pkg/config"
)

// Aggregator aggregates metrics over time
type Aggregator struct {
	config  *config.Config
	manager *Manager
	history map[string][]TimeSeriesPoint
	mu      sync.RWMutex
}

// TimeSeriesPoint represents a single metric data point
type TimeSeriesPoint struct {
	Timestamp time.Time              `json:"timestamp"`
	Value     float64                `json:"value"`
	Labels    map[string]string      `json:"labels,omitempty"`
}

// AggregatedMetrics represents aggregated metric data
type AggregatedMetrics struct {
	MetricName string              `json:"metric_name"`
	StartTime  time.Time           `json:"start_time"`
	EndTime    time.Time           `json:"end_time"`
	Min        float64             `json:"min"`
	Max        float64             `json:"max"`
	Avg        float64             `json:"avg"`
	Count      int                 `json:"count"`
	Sum        float64             `json:"sum"`
	Percentile map[int]float64     `json:"percentile"` // 50th, 95th, 99th
	Points     []TimeSeriesPoint   `json:"points,omitempty"`
}

// NewAggregator creates a new metrics aggregator
func NewAggregator(cfg *config.Config, mgr *Manager) *Aggregator {
	return &Aggregator{
		config:  cfg,
		manager: mgr,
		history: make(map[string][]TimeSeriesPoint),
	}
}

// Start begins metric aggregation
func (a *Aggregator) Start(ctx context.Context) {
	log.Println("Starting metrics aggregator")

	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()

	cleanupTicker := time.NewTicker(time.Hour)
	defer cleanupTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Println("Metrics aggregator stopped")
			return
		case <-ticker.C:
			a.collectMetrics()
		case <-cleanupTicker.C:
			a.cleanupOldMetrics()
		}
	}
}

// collectMetrics collects current metrics and stores them
func (a *Aggregator) collectMetrics() {
	now := time.Now()

	// Collect system metrics
	systemMetrics := a.manager.GetSystemMetrics()

	a.recordMetric("api.uptime", now, systemMetrics.Uptime, nil)
	a.recordMetric("api.requests_per_second", now, systemMetrics.RequestsPerSecond, nil)
	a.recordMetric("api.error_rate", now, systemMetrics.ErrorRate, nil)
	a.recordMetric("api.latency", now, systemMetrics.AvgLatency, nil)
	a.recordMetric("api.active_connections", now, float64(systemMetrics.ActiveConnections), nil)

	// Collect component metrics
	components := a.manager.GetComponentStatus()
	for name, component := range components {
		labels := map[string]string{"component": name}

		uptimeValue := 0.0
		if component.Status == "operational" {
			uptimeValue = 100.0
		}

		a.recordMetric("component.uptime", now, uptimeValue, labels)
		a.recordMetric("component.latency", now, component.Latency, labels)
	}

	log.Println("Collected metrics snapshot")
}

// recordMetric records a metric data point
func (a *Aggregator) recordMetric(name string, timestamp time.Time, value float64, labels map[string]string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	point := TimeSeriesPoint{
		Timestamp: timestamp,
		Value:     value,
		Labels:    labels,
	}

	if _, exists := a.history[name]; !exists {
		a.history[name] = make([]TimeSeriesPoint, 0)
	}

	a.history[name] = append(a.history[name], point)
}

// cleanupOldMetrics removes metrics older than retention period
func (a *Aggregator) cleanupOldMetrics() {
	a.mu.Lock()
	defer a.mu.Unlock()

	retentionPeriod := 24 * time.Hour // Keep 24 hours of data
	cutoffTime := time.Now().Add(-retentionPeriod)

	for metricName, points := range a.history {
		// Filter out old points
		newPoints := make([]TimeSeriesPoint, 0)
		for _, point := range points {
			if point.Timestamp.After(cutoffTime) {
				newPoints = append(newPoints, point)
			}
		}

		if len(newPoints) > 0 {
			a.history[metricName] = newPoints
		} else {
			delete(a.history, metricName)
		}
	}

	log.Printf("Cleaned up old metrics. Current metrics: %d", len(a.history))
}

// GetAggregatedMetrics returns aggregated metrics for a time range
func (a *Aggregator) GetAggregatedMetrics(metricName string, start, end time.Time) *AggregatedMetrics {
	a.mu.RLock()
	defer a.mu.RUnlock()

	points, exists := a.history[metricName]
	if !exists {
		return nil
	}

	// Filter points by time range
	filteredPoints := make([]TimeSeriesPoint, 0)
	for _, point := range points {
		if point.Timestamp.After(start) && point.Timestamp.Before(end) {
			filteredPoints = append(filteredPoints, point)
		}
	}

	if len(filteredPoints) == 0 {
		return nil
	}

	// Calculate aggregations
	min := filteredPoints[0].Value
	max := filteredPoints[0].Value
	sum := 0.0

	for _, point := range filteredPoints {
		if point.Value < min {
			min = point.Value
		}
		if point.Value > max {
			max = point.Value
		}
		sum += point.Value
	}

	avg := sum / float64(len(filteredPoints))

	// Calculate percentiles
	percentiles := calculatePercentiles(filteredPoints)

	return &AggregatedMetrics{
		MetricName: metricName,
		StartTime:  start,
		EndTime:    end,
		Min:        min,
		Max:        max,
		Avg:        avg,
		Count:      len(filteredPoints),
		Sum:        sum,
		Percentile: percentiles,
		Points:     filteredPoints,
	}
}

// GetMetricHistory returns raw metric history
func (a *Aggregator) GetMetricHistory(metricName string, duration time.Duration) []TimeSeriesPoint {
	a.mu.RLock()
	defer a.mu.RUnlock()

	points, exists := a.history[metricName]
	if !exists {
		return nil
	}

	cutoffTime := time.Now().Add(-duration)
	result := make([]TimeSeriesPoint, 0)

	for _, point := range points {
		if point.Timestamp.After(cutoffTime) {
			result = append(result, point)
		}
	}

	return result
}

// GetAllMetrics returns all available metric names
func (a *Aggregator) GetAllMetrics() []string {
	a.mu.RLock()
	defer a.mu.RUnlock()

	names := make([]string, 0, len(a.history))
	for name := range a.history {
		names = append(names, name)
	}

	return names
}

// CalculateUptime calculates uptime percentage over a period
func (a *Aggregator) CalculateUptime(component string, duration time.Duration) float64 {
	metricName := "component.uptime"
	history := a.GetMetricHistory(metricName, duration)

	if len(history) == 0 {
		return 0
	}

	uptimeSum := 0.0
	count := 0

	for _, point := range history {
		if labels := point.Labels; labels != nil {
			if labels["component"] == component {
				uptimeSum += point.Value
				count++
			}
		}
	}

	if count == 0 {
		return 0
	}

	return uptimeSum / float64(count)
}

// CalculateAverageLatency calculates average latency over a period
func (a *Aggregator) CalculateAverageLatency(duration time.Duration) float64 {
	history := a.GetMetricHistory("api.latency", duration)

	if len(history) == 0 {
		return 0
	}

	sum := 0.0
	for _, point := range history {
		sum += point.Value
	}

	return sum / float64(len(history))
}

// GetMetricsReport generates a comprehensive metrics report
func (a *Aggregator) GetMetricsReport(duration time.Duration) map[string]interface{} {
	now := time.Now()
	start := now.Add(-duration)

	report := make(map[string]interface{})

	// API metrics
	apiUptime := a.GetAggregatedMetrics("api.uptime", start, now)
	apiLatency := a.GetAggregatedMetrics("api.latency", start, now)
	apiRPS := a.GetAggregatedMetrics("api.requests_per_second", start, now)
	apiErrorRate := a.GetAggregatedMetrics("api.error_rate", start, now)

	report["api"] = map[string]interface{}{
		"uptime":              apiUptime,
		"latency":             apiLatency,
		"requests_per_second": apiRPS,
		"error_rate":          apiErrorRate,
	}

	// Component metrics
	components := make(map[string]interface{})
	for _, comp := range []string{"API", "RPC", "Database", "Explorer"} {
		uptime := a.CalculateUptime(comp, duration)
		components[comp] = map[string]interface{}{
			"uptime": uptime,
		}
	}

	report["components"] = components
	report["period"] = map[string]interface{}{
		"start":    start,
		"end":      now,
		"duration": duration.String(),
	}

	return report
}

// calculatePercentiles calculates percentile values
func calculatePercentiles(points []TimeSeriesPoint) map[int]float64 {
	if len(points) == 0 {
		return nil
	}

	// Extract values and sort
	values := make([]float64, len(points))
	for i, point := range points {
		values[i] = point.Value
	}

	// Simple bubble sort (in production, use a proper sorting algorithm)
	for i := 0; i < len(values); i++ {
		for j := i + 1; j < len(values); j++ {
			if values[i] > values[j] {
				values[i], values[j] = values[j], values[i]
			}
		}
	}

	percentiles := make(map[int]float64)

	// Calculate percentiles
	percentiles[50] = getPercentile(values, 50)
	percentiles[95] = getPercentile(values, 95)
	percentiles[99] = getPercentile(values, 99)

	return percentiles
}

// getPercentile gets a specific percentile value
func getPercentile(sorted []float64, p int) float64 {
	if len(sorted) == 0 {
		return 0
	}

	index := (len(sorted) * p) / 100
	if index >= len(sorted) {
		index = len(sorted) - 1
	}

	return sorted[index]
}
