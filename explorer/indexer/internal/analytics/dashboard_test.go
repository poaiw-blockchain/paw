package analytics

import (
	"context"
	"testing"
	"time"
)

// TestComputeConsensusMetrics_NoRPCClient tests fallback when RPC client is not configured
func TestComputeConsensusMetrics_NoRPCClient(t *testing.T) {
	service := &AnalyticsService{
		rpcClient: nil,
		config: AnalyticsConfig{
			RPCURL: "",
		},
	}

	ctx := context.Background()
	metrics, err := service.computeConsensusMetrics(ctx)
	if err != nil {
		t.Fatalf("Expected no error with nil RPC client, got: %v", err)
	}

	// Verify default values are returned
	if metrics.RoundsPerBlock != 1.0 {
		t.Errorf("Expected RoundsPerBlock=1.0, got %f", metrics.RoundsPerBlock)
	}
	if metrics.TimeToFinality != 6.0 {
		t.Errorf("Expected TimeToFinality=6.0, got %f", metrics.TimeToFinality)
	}
	if metrics.PrecommitRate != 0.95 {
		t.Errorf("Expected PrecommitRate=0.95, got %f", metrics.PrecommitRate)
	}
	if metrics.PrevoteRate != 0.95 {
		t.Errorf("Expected PrevoteRate=0.95, got %f", metrics.PrevoteRate)
	}
	if metrics.HealthScore != 0.90 {
		t.Errorf("Expected HealthScore=0.90, got %f", metrics.HealthScore)
	}
}

// TestAnalyticsConfig tests the analytics configuration structure
func TestAnalyticsConfig(t *testing.T) {
	config := AnalyticsConfig{
		CacheDuration:   5 * time.Minute,
		RefreshInterval: 30 * time.Second,
		HistoryDepth:    1000,
		RPCURL:          "http://localhost:26657",
	}

	if config.RPCURL != "http://localhost:26657" {
		t.Errorf("Expected RPCURL to be set, got %s", config.RPCURL)
	}
	if config.CacheDuration != 5*time.Minute {
		t.Errorf("Expected CacheDuration=5m, got %v", config.CacheDuration)
	}
}

// TestConsensusMetricsStructure tests the consensus metrics structure
func TestConsensusMetricsStructure(t *testing.T) {
	metrics := ConsensusMetrics{
		RoundsPerBlock:  1.05,
		TimeToFinality:  6.5,
		PrecommitRate:   0.99,
		PrevoteRate:     0.99,
		HealthScore:     0.98,
	}

	if metrics.RoundsPerBlock <= 0 {
		t.Error("RoundsPerBlock should be positive")
	}
	if metrics.HealthScore < 0 || metrics.HealthScore > 1 {
		t.Error("HealthScore should be between 0 and 1")
	}
	if metrics.PrecommitRate < 0 || metrics.PrecommitRate > 1 {
		t.Error("PrecommitRate should be between 0 and 1")
	}
	if metrics.PrevoteRate < 0 || metrics.PrevoteRate > 1 {
		t.Error("PrevoteRate should be between 0 and 1")
	}
}
