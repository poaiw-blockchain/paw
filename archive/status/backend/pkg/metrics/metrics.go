package metrics

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"status/pkg/config"
)

// DataPoint represents a single metric data point
type DataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

// NetworkStats represents blockchain network statistics
type NetworkStats struct {
	BlockHeight      int64  `json:"block_height"`
	TotalValidators  int    `json:"total_validators"`
	ActiveValidators int    `json:"active_validators"`
	HashRate         string `json:"hash_rate"`
}

// Metrics holds all collected metrics
type Metrics struct {
	TPS          []DataPoint  `json:"tps"`
	BlockTime    []DataPoint  `json:"block_time"`
	Peers        []DataPoint  `json:"peers"`
	ResponseTime []DataPoint  `json:"response_time"`
	NetworkStats NetworkStats `json:"network_stats"`
	UptimeData   []UptimeDay  `json:"uptime_data"`
}

// UptimeDay represents uptime status for a single day
type UptimeDay struct {
	Date   time.Time `json:"date"`
	Status string    `json:"status"`
}

// Collector collects and stores system metrics
type Collector struct {
	config     *config.Config
	metrics    *Metrics
	mutex      sync.RWMutex
	httpClient *http.Client
}

// NewCollector creates a new metrics collector
func NewCollector(cfg *config.Config) *Collector {
	return &Collector{
		config: cfg,
		metrics: &Metrics{
			TPS:          make([]DataPoint, 0),
			BlockTime:    make([]DataPoint, 0),
			Peers:        make([]DataPoint, 0),
			ResponseTime: make([]DataPoint, 0),
		},
		httpClient: &http.Client{Timeout: 10 * time.Second},
	}
}

// Start begins the metrics collection process
func (c *Collector) Start(ctx context.Context) {
	log.Println("Metrics collector started")

	ticker := time.NewTicker(c.config.MonitorInterval)
	defer ticker.Stop()

	// Initial collection
	c.collectMetrics()

	for {
		select {
		case <-ctx.Done():
			log.Println("Metrics collector stopped")
			return
		case <-ticker.C:
			c.collectMetrics()
		}
	}
}

// collectMetrics collects all metrics
func (c *Collector) collectMetrics() {
	c.mutex.Lock()
	defer c.mutex.Unlock()

	now := time.Now()

	// Collect TPS
	tps := c.collectTPS()
	c.addDataPoint(&c.metrics.TPS, DataPoint{Timestamp: now, Value: tps})

	// Collect Block Time
	blockTime := c.collectBlockTime()
	c.addDataPoint(&c.metrics.BlockTime, DataPoint{Timestamp: now, Value: blockTime})

	// Collect Peers
	peers := c.collectPeers()
	c.addDataPoint(&c.metrics.Peers, DataPoint{Timestamp: now, Value: float64(peers)})

	// Collect API Response Time
	responseTime := c.collectAPIResponseTime()
	c.addDataPoint(&c.metrics.ResponseTime, DataPoint{Timestamp: now, Value: responseTime})

	// Update network stats
	c.metrics.NetworkStats = c.collectNetworkStats()

	// Generate uptime data
	c.metrics.UptimeData = c.generateUptimeData(30)

	// Cleanup old data
	c.cleanupOldData()

	log.Printf("Metrics collected - TPS: %.2f, BlockTime: %.2fs, Peers: %d, ResponseTime: %.2fms",
		tps, blockTime, peers, responseTime)
}

// collectTPS collects transactions per second
func (c *Collector) collectTPS() float64 {
	// In production, query the blockchain for actual TPS
	// For now, generate realistic mock data
	baseValue := 150.0
	variance := 50.0
	return baseValue + (rand.Float64()-0.5)*variance
}

// collectBlockTime collects average block time
func (c *Collector) collectBlockTime() float64 {
	// In production, calculate from actual block timestamps
	// For now, generate realistic mock data around 6.5 seconds
	baseValue := 6.5
	variance := 1.0
	return baseValue + (rand.Float64()-0.5)*variance
}

// collectPeers collects number of connected peers
func (c *Collector) collectPeers() int {
	// In production, query the node for peer count
	// Try to get from RPC endpoint
	type NetInfoResponse struct {
		Result struct {
			NPeers string `json:"n_peers"`
		} `json:"result"`
	}

	url := c.config.BlockchainRPCURL + "/net_info"
	resp, err := c.httpClient.Get(url)
	if err == nil {
		defer resp.Body.Close()
		var netInfo NetInfoResponse
		if json.NewDecoder(resp.Body).Decode(&netInfo) == nil {
			var peers int
			fmt.Sscanf(netInfo.Result.NPeers, "%d", &peers)
			if peers > 0 {
				return peers
			}
		}
	}

	// Fallback to mock data
	baseValue := 42
	variance := 5
	return baseValue + rand.Intn(variance*2) - variance
}

// collectAPIResponseTime measures API response time
func (c *Collector) collectAPIResponseTime() float64 {
	start := time.Now()

	url := c.config.APIEndpoint + "/cosmos/base/tendermint/v1beta1/node_info"
	resp, err := c.httpClient.Get(url)
	if err == nil {
		defer resp.Body.Close()
		responseTime := time.Since(start)
		return float64(responseTime.Milliseconds())
	}

	// Fallback to mock data
	baseValue := 120.0
	variance := 30.0
	return baseValue + (rand.Float64()-0.5)*variance
}

// collectNetworkStats collects blockchain network statistics
func (c *Collector) collectNetworkStats() NetworkStats {
	// In production, query the blockchain for actual stats
	type StatusResponse struct {
		Result struct {
			SyncInfo struct {
				LatestBlockHeight string `json:"latest_block_height"`
			} `json:"sync_info"`
		} `json:"result"`
	}

	stats := NetworkStats{
		BlockHeight:      1234567,
		TotalValidators:  150,
		ActiveValidators: 125,
		HashRate:         "1.2 TH/s",
	}

	url := c.config.BlockchainRPCURL + "/status"
	resp, err := c.httpClient.Get(url)
	if err == nil {
		defer resp.Body.Close()
		var status StatusResponse
		if json.NewDecoder(resp.Body).Decode(&status) == nil {
			var height int64
			fmt.Sscanf(status.Result.SyncInfo.LatestBlockHeight, "%d", &height)
			if height > 0 {
				stats.BlockHeight = height
			}
		}
	}

	return stats
}

// generateUptimeData generates uptime history for N days
func (c *Collector) generateUptimeData(days int) []UptimeDay {
	uptime := make([]UptimeDay, days)
	now := time.Now()

	for i := 0; i < days; i++ {
		date := now.AddDate(0, 0, -i)

		// Simulate high uptime (98% operational)
		status := "operational"
		if rand.Float64() > 0.98 {
			status = "degraded"
		}

		uptime[days-1-i] = UptimeDay{
			Date:   date,
			Status: status,
		}
	}

	return uptime
}

// addDataPoint adds a new data point and maintains the retention limit
func (c *Collector) addDataPoint(series *[]DataPoint, point DataPoint) {
	*series = append(*series, point)

	// Keep only recent data points (e.g., last 100 points)
	maxPoints := 100
	if len(*series) > maxPoints {
		*series = (*series)[len(*series)-maxPoints:]
	}
}

// cleanupOldData removes data points older than retention period
func (c *Collector) cleanupOldData() {
	cutoff := time.Now().Add(-c.config.MetricsRetention)

	c.metrics.TPS = c.filterOldDataPoints(c.metrics.TPS, cutoff)
	c.metrics.BlockTime = c.filterOldDataPoints(c.metrics.BlockTime, cutoff)
	c.metrics.Peers = c.filterOldDataPoints(c.metrics.Peers, cutoff)
	c.metrics.ResponseTime = c.filterOldDataPoints(c.metrics.ResponseTime, cutoff)
}

// filterOldDataPoints removes data points older than cutoff time
func (c *Collector) filterOldDataPoints(points []DataPoint, cutoff time.Time) []DataPoint {
	filtered := make([]DataPoint, 0)
	for _, point := range points {
		if point.Timestamp.After(cutoff) {
			filtered = append(filtered, point)
		}
	}
	return filtered
}

// GetMetrics returns current metrics
func (c *Collector) GetMetrics() *Metrics {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	// Create a copy to avoid race conditions
	metrics := &Metrics{
		TPS:          make([]DataPoint, len(c.metrics.TPS)),
		BlockTime:    make([]DataPoint, len(c.metrics.BlockTime)),
		Peers:        make([]DataPoint, len(c.metrics.Peers)),
		ResponseTime: make([]DataPoint, len(c.metrics.ResponseTime)),
		NetworkStats: c.metrics.NetworkStats,
		UptimeData:   make([]UptimeDay, len(c.metrics.UptimeData)),
	}

	copy(metrics.TPS, c.metrics.TPS)
	copy(metrics.BlockTime, c.metrics.BlockTime)
	copy(metrics.Peers, c.metrics.Peers)
	copy(metrics.ResponseTime, c.metrics.ResponseTime)
	copy(metrics.UptimeData, c.metrics.UptimeData)

	return metrics
}

// GetMetricsSummary returns a summary of current metrics
func (c *Collector) GetMetricsSummary() map[string]interface{} {
	c.mutex.RLock()
	defer c.mutex.RUnlock()

	summary := make(map[string]interface{})

	if len(c.metrics.TPS) > 0 {
		summary["current_tps"] = c.metrics.TPS[len(c.metrics.TPS)-1].Value
		summary["avg_tps"] = c.calculateAverage(c.metrics.TPS)
	}

	if len(c.metrics.BlockTime) > 0 {
		summary["current_block_time"] = c.metrics.BlockTime[len(c.metrics.BlockTime)-1].Value
		summary["avg_block_time"] = c.calculateAverage(c.metrics.BlockTime)
	}

	if len(c.metrics.Peers) > 0 {
		summary["current_peers"] = int(c.metrics.Peers[len(c.metrics.Peers)-1].Value)
		summary["avg_peers"] = int(c.calculateAverage(c.metrics.Peers))
	}

	if len(c.metrics.ResponseTime) > 0 {
		summary["current_response_time"] = c.metrics.ResponseTime[len(c.metrics.ResponseTime)-1].Value
		summary["avg_response_time"] = c.calculateAverage(c.metrics.ResponseTime)
	}

	summary["network_stats"] = c.metrics.NetworkStats

	return summary
}

// calculateAverage calculates the average of data points
func (c *Collector) calculateAverage(points []DataPoint) float64 {
	if len(points) == 0 {
		return 0
	}

	sum := 0.0
	for _, point := range points {
		sum += point.Value
	}

	return math.Round(sum/float64(len(points))*100) / 100
}

// MarshalJSON custom JSON marshaling for DataPoint
func (d *DataPoint) MarshalJSON() ([]byte, error) {
	type Alias DataPoint
	return json.Marshal(&struct {
		*Alias
		Timestamp string `json:"timestamp"`
	}{
		Alias:     (*Alias)(d),
		Timestamp: d.Timestamp.Format(time.RFC3339),
	})
}

// MarshalJSON custom JSON marshaling for UptimeDay
func (u *UptimeDay) MarshalJSON() ([]byte, error) {
	type Alias UptimeDay
	return json.Marshal(&struct {
		*Alias
		Date string `json:"date"`
	}{
		Alias: (*Alias)(u),
		Date:  u.Date.Format(time.RFC3339),
	})
}
