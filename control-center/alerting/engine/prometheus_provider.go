package engine

import (
	"context"
	"fmt"
	"time"

	"github.com/paw/control-center/alerting"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

// PrometheusProvider implements MetricsProvider using Prometheus
type PrometheusProvider struct {
	client v1.API
}

// NewPrometheusProvider creates a new Prometheus metrics provider
func NewPrometheusProvider(prometheusURL string) (*PrometheusProvider, error) {
	client, err := api.NewClient(api.Config{
		Address: prometheusURL,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create Prometheus client: %w", err)
	}

	return &PrometheusProvider{
		client: v1.NewAPI(client),
	}, nil
}

// GetMetric retrieves a single metric value from Prometheus
func (p *PrometheusProvider) GetMetric(name string, labels map[string]string) (*alerting.MetricValue, error) {
	// Build query with labels
	query := name
	if len(labels) > 0 {
		query += "{"
		first := true
		for k, v := range labels {
			if !first {
				query += ","
			}
			query += fmt.Sprintf(`%s="%s"`, k, v)
			first = false
		}
		query += "}"
	}

	// Execute query
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	result, warnings, err := p.client.Query(ctx, query, time.Now())
	if err != nil {
		return nil, fmt.Errorf("failed to query Prometheus: %w", err)
	}

	if len(warnings) > 0 {
		// Log warnings but continue
		for _, w := range warnings {
			fmt.Printf("Prometheus warning: %s\n", w)
		}
	}

	// Parse result
	switch v := result.(type) {
	case model.Vector:
		if len(v) == 0 {
			return nil, fmt.Errorf("no data found for metric %s", name)
		}

		// Use the first result
		sample := v[0]
		return &alerting.MetricValue{
			Name:      name,
			Value:     float64(sample.Value),
			Timestamp: sample.Timestamp.Time(),
			Labels:    convertLabels(sample.Metric),
		}, nil

	case *model.Scalar:
		return &alerting.MetricValue{
			Name:      name,
			Value:     float64(v.Value),
			Timestamp: v.Timestamp.Time(),
			Labels:    labels,
		}, nil

	default:
		return nil, fmt.Errorf("unexpected result type: %T", result)
	}
}

// GetMetricRange retrieves a range of metric values from Prometheus
func (p *PrometheusProvider) GetMetricRange(name string, labels map[string]string, start, end time.Time) ([]*alerting.MetricValue, error) {
	// Build query with labels
	query := name
	if len(labels) > 0 {
		query += "{"
		first := true
		for k, v := range labels {
			if !first {
				query += ","
			}
			query += fmt.Sprintf(`%s="%s"`, k, v)
			first = false
		}
		query += "}"
	}

	// Calculate appropriate step (resolution)
	duration := end.Sub(start)
	step := duration / 100 // 100 data points
	if step < 15*time.Second {
		step = 15 * time.Second // Minimum step
	}

	// Execute range query
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	r := v1.Range{
		Start: start,
		End:   end,
		Step:  step,
	}

	result, warnings, err := p.client.QueryRange(ctx, query, r)
	if err != nil {
		return nil, fmt.Errorf("failed to query Prometheus range: %w", err)
	}

	if len(warnings) > 0 {
		for _, w := range warnings {
			fmt.Printf("Prometheus warning: %s\n", w)
		}
	}

	// Parse result
	switch v := result.(type) {
	case model.Matrix:
		if len(v) == 0 {
			return nil, fmt.Errorf("no data found for metric %s", name)
		}

		// Convert to MetricValue slice
		stream := v[0]
		metrics := make([]*alerting.MetricValue, len(stream.Values))

		for i, pair := range stream.Values {
			metrics[i] = &alerting.MetricValue{
				Name:      name,
				Value:     float64(pair.Value),
				Timestamp: pair.Timestamp.Time(),
				Labels:    convertLabels(stream.Metric),
			}
		}

		return metrics, nil

	default:
		return nil, fmt.Errorf("unexpected result type for range query: %T", result)
	}
}

// convertLabels converts Prometheus labels to map
func convertLabels(metric model.Metric) map[string]string {
	labels := make(map[string]string)
	for k, v := range metric {
		if k != "__name__" { // Skip the metric name label
			labels[string(k)] = string(v)
		}
	}
	return labels
}
