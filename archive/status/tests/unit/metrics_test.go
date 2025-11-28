package tests

import (
	"context"
	"testing"
	"time"

	"status/pkg/config"
	"status/pkg/metrics"

	"github.com/stretchr/testify/assert"
)

func TestNewCollector(t *testing.T) {
	cfg := &config.Config{
		MonitorInterval: 30 * time.Second,
	}

	collector := metrics.NewCollector(cfg)
	assert.NotNil(t, collector)
}

func TestGetMetrics(t *testing.T) {
	cfg := &config.Config{
		MonitorInterval:  1 * time.Second,
		MetricsRetention: 24 * time.Hour,
		BlockchainRPCURL: "http://localhost:26657",
		APIEndpoint:      "http://localhost:1317",
	}

	collector := metrics.NewCollector(cfg)

	// Start collector
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go collector.Start(ctx)

	// Wait for some metrics to be collected
	time.Sleep(2 * time.Second)

	m := collector.GetMetrics()
	assert.NotNil(t, m)
	assert.NotNil(t, m.TPS)
	assert.NotNil(t, m.BlockTime)
	assert.NotNil(t, m.Peers)
	assert.NotNil(t, m.ResponseTime)

	// Should have at least one data point
	assert.NotEmpty(t, m.TPS)
	assert.NotEmpty(t, m.BlockTime)
	assert.NotEmpty(t, m.Peers)
	assert.NotEmpty(t, m.ResponseTime)
}

func TestGetMetricsSummary(t *testing.T) {
	cfg := &config.Config{
		MonitorInterval:  1 * time.Second,
		MetricsRetention: 24 * time.Hour,
		BlockchainRPCURL: "http://localhost:26657",
		APIEndpoint:      "http://localhost:1317",
	}

	collector := metrics.NewCollector(cfg)

	// Start collector
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go collector.Start(ctx)

	// Wait for metrics
	time.Sleep(2 * time.Second)

	summary := collector.GetMetricsSummary()
	assert.NotNil(t, summary)
	assert.Contains(t, summary, "network_stats")
}

func TestNetworkStats(t *testing.T) {
	cfg := &config.Config{
		MonitorInterval:  1 * time.Second,
		BlockchainRPCURL: "http://localhost:26657",
		APIEndpoint:      "http://localhost:1317",
	}

	collector := metrics.NewCollector(cfg)

	// Start collector
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go collector.Start(ctx)

	// Wait for collection
	time.Sleep(2 * time.Second)

	m := collector.GetMetrics()
	assert.NotZero(t, m.NetworkStats.BlockHeight)
	assert.NotZero(t, m.NetworkStats.TotalValidators)
	assert.NotZero(t, m.NetworkStats.ActiveValidators)
	assert.NotEmpty(t, m.NetworkStats.HashRate)
}

func TestUptimeData(t *testing.T) {
	cfg := &config.Config{
		MonitorInterval: 1 * time.Second,
	}

	collector := metrics.NewCollector(cfg)

	// Start collector
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go collector.Start(ctx)

	// Wait for data
	time.Sleep(2 * time.Second)

	m := collector.GetMetrics()
	assert.NotNil(t, m.UptimeData)
	assert.Len(t, m.UptimeData, 30) // Should have 30 days

	// Verify each day has required fields
	for _, day := range m.UptimeData {
		assert.NotZero(t, day.Date)
		assert.NotEmpty(t, day.Status)
		assert.Contains(t, []string{"operational", "degraded", "down"}, day.Status)
	}
}

func TestMetricsRetention(t *testing.T) {
	cfg := &config.Config{
		MonitorInterval:  100 * time.Millisecond,
		MetricsRetention: 1 * time.Second, // Very short retention for testing
		BlockchainRPCURL: "http://localhost:26657",
		APIEndpoint:      "http://localhost:1317",
	}

	collector := metrics.NewCollector(cfg)

	// Start collector
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go collector.Start(ctx)

	// Wait for multiple collections
	time.Sleep(3 * time.Second)

	m := collector.GetMetrics()

	// Should not have more than a reasonable number of points
	// (100 points max as per implementation)
	assert.LessOrEqual(t, len(m.TPS), 100)
	assert.LessOrEqual(t, len(m.BlockTime), 100)
	assert.LessOrEqual(t, len(m.Peers), 100)
	assert.LessOrEqual(t, len(m.ResponseTime), 100)
}

func TestDataPointValues(t *testing.T) {
	cfg := &config.Config{
		MonitorInterval:  1 * time.Second,
		BlockchainRPCURL: "http://localhost:26657",
		APIEndpoint:      "http://localhost:1317",
	}

	collector := metrics.NewCollector(cfg)

	// Start collector
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go collector.Start(ctx)

	// Wait for data
	time.Sleep(2 * time.Second)

	m := collector.GetMetrics()

	// Verify TPS values are reasonable
	if len(m.TPS) > 0 {
		for _, dp := range m.TPS {
			assert.GreaterOrEqual(t, dp.Value, 0.0)
			assert.LessOrEqual(t, dp.Value, 1000.0) // Reasonable max
		}
	}

	// Verify block time values are reasonable
	if len(m.BlockTime) > 0 {
		for _, dp := range m.BlockTime {
			assert.GreaterOrEqual(t, dp.Value, 1.0)
			assert.LessOrEqual(t, dp.Value, 30.0) // Reasonable max
		}
	}

	// Verify peer values are reasonable
	if len(m.Peers) > 0 {
		for _, dp := range m.Peers {
			assert.GreaterOrEqual(t, dp.Value, 0.0)
			assert.LessOrEqual(t, dp.Value, 1000.0) // Reasonable max
		}
	}

	// Verify response time values are reasonable
	if len(m.ResponseTime) > 0 {
		for _, dp := range m.ResponseTime {
			assert.GreaterOrEqual(t, dp.Value, 0.0)
			assert.LessOrEqual(t, dp.Value, 10000.0) // 10 seconds max
		}
	}
}
