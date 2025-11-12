package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"time"

	"github.com/cosmos/cosmos-sdk/client"
	"github.com/cosmos/cosmos-sdk/client/tx"
	"github.com/cosmos/cosmos-sdk/codec"
	"github.com/cosmos/cosmos-sdk/types"
	banktypes "github.com/cosmos/cosmos-sdk/x/bank/types"
)

// LoadTestConfig holds the configuration for the load test
type LoadTestConfig struct {
	RPCEndpoint    string
	APIEndpoint    string
	ChainID        string
	Duration       time.Duration
	Concurrency    int
	TxRate         int // transactions per second
	TestType       string
	ReportInterval time.Duration
}

// LoadTestMetrics tracks performance metrics
type LoadTestMetrics struct {
	TotalTxSubmitted    uint64
	TotalTxSuccessful   uint64
	TotalTxFailed       uint64
	TotalQueries        uint64
	TotalQueryFailed    uint64
	MinLatency          int64 // nanoseconds
	MaxLatency          int64
	TotalLatency        int64
	StartTime           time.Time
	EndTime             time.Time
	LatencyHistogram    map[int]uint64 // bucket (ms) -> count
	ErrorsByType        map[string]uint64
	mutex               sync.RWMutex
}

// NewLoadTestMetrics creates a new metrics tracker
func NewLoadTestMetrics() *LoadTestMetrics {
	return &LoadTestMetrics{
		MinLatency:       int64(^uint64(0) >> 1), // max int64
		MaxLatency:       0,
		StartTime:        time.Now(),
		LatencyHistogram: make(map[int]uint64),
		ErrorsByType:     make(map[string]uint64),
	}
}

// RecordTransaction records a transaction result
func (m *LoadTestMetrics) RecordTransaction(success bool, latency time.Duration, errType string) {
	atomic.AddUint64(&m.TotalTxSubmitted, 1)

	if success {
		atomic.AddUint64(&m.TotalTxSuccessful, 1)
	} else {
		atomic.AddUint64(&m.TotalTxFailed, 1)
		m.mutex.Lock()
		m.ErrorsByType[errType]++
		m.mutex.Unlock()
	}

	// Update latency stats
	latencyNs := latency.Nanoseconds()
	atomic.AddInt64(&m.TotalLatency, latencyNs)

	// Update min/max latency
	for {
		current := atomic.LoadInt64(&m.MinLatency)
		if latencyNs >= current || atomic.CompareAndSwapInt64(&m.MinLatency, current, latencyNs) {
			break
		}
	}

	for {
		current := atomic.LoadInt64(&m.MaxLatency)
		if latencyNs <= current || atomic.CompareAndSwapInt64(&m.MaxLatency, current, latencyNs) {
			break
		}
	}

	// Update histogram
	bucket := int(latency.Milliseconds() / 10) // 10ms buckets
	m.mutex.Lock()
	m.LatencyHistogram[bucket]++
	m.mutex.Unlock()
}

// RecordQuery records a query result
func (m *LoadTestMetrics) RecordQuery(success bool, latency time.Duration) {
	atomic.AddUint64(&m.TotalQueries, 1)
	if !success {
		atomic.AddUint64(&m.TotalQueryFailed, 1)
	}

	latencyNs := latency.Nanoseconds()
	atomic.AddInt64(&m.TotalLatency, latencyNs)
}

// GetReport generates a performance report
func (m *LoadTestMetrics) GetReport() map[string]interface{} {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	duration := m.EndTime.Sub(m.StartTime).Seconds()
	avgLatency := time.Duration(0)
	if m.TotalTxSubmitted > 0 {
		avgLatency = time.Duration(m.TotalLatency / int64(m.TotalTxSubmitted))
	}

	tps := float64(m.TotalTxSuccessful) / duration

	return map[string]interface{}{
		"duration_seconds":     duration,
		"total_tx_submitted":   m.TotalTxSubmitted,
		"total_tx_successful":  m.TotalTxSuccessful,
		"total_tx_failed":      m.TotalTxFailed,
		"total_queries":        m.TotalQueries,
		"total_query_failed":   m.TotalQueryFailed,
		"transactions_per_sec": tps,
		"avg_latency_ms":       avgLatency.Milliseconds(),
		"min_latency_ms":       time.Duration(m.MinLatency).Milliseconds(),
		"max_latency_ms":       time.Duration(m.MaxLatency).Milliseconds(),
		"success_rate":         float64(m.TotalTxSuccessful) / float64(m.TotalTxSubmitted) * 100,
		"latency_histogram":    m.LatencyHistogram,
		"errors_by_type":       m.ErrorsByType,
	}
}

// LoadTester manages the load testing
type LoadTester struct {
	config  *LoadTestConfig
	metrics *LoadTestMetrics
	ctx     context.Context
	cancel  context.CancelFunc
}

// NewLoadTester creates a new load tester
func NewLoadTester(config *LoadTestConfig) *LoadTester {
	ctx, cancel := context.WithTimeout(context.Background(), config.Duration)

	return &LoadTester{
		config:  config,
		metrics: NewLoadTestMetrics(),
		ctx:     ctx,
		cancel:  cancel,
	}
}

// Run starts the load test
func (lt *LoadTester) Run() error {
	log.Printf("Starting load test: %s", lt.config.TestType)
	log.Printf("Duration: %v, Concurrency: %d, Target TPS: %d",
		lt.config.Duration, lt.config.Concurrency, lt.config.TxRate)

	var wg sync.WaitGroup

	// Start metrics reporter
	go lt.reportMetrics()

	// Start workers
	for i := 0; i < lt.config.Concurrency; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			lt.worker(workerID)
		}(i)
	}

	// Wait for completion
	wg.Wait()
	lt.metrics.EndTime = time.Now()

	return nil
}

// worker runs the actual load test operations
func (lt *LoadTester) worker(workerID int) {
	ticker := time.NewTicker(time.Second / time.Duration(lt.config.TxRate/lt.config.Concurrency))
	defer ticker.Stop()

	for {
		select {
		case <-lt.ctx.Done():
			return
		case <-ticker.C:
			switch lt.config.TestType {
			case "transactions":
				lt.sendTransaction(workerID)
			case "queries":
				lt.performQuery(workerID)
			case "mixed":
				if workerID%2 == 0 {
					lt.sendTransaction(workerID)
				} else {
					lt.performQuery(workerID)
				}
			case "dex":
				lt.performDEXOperation(workerID)
			}
		}
	}
}

// sendTransaction sends a test transaction
func (lt *LoadTester) sendTransaction(workerID int) {
	start := time.Now()

	// TODO: Implement actual transaction sending using Cosmos SDK client
	// For now, simulate with a sleep
	time.Sleep(time.Millisecond * 10)

	success := true // Replace with actual result
	errType := ""

	if !success {
		errType = "tx_broadcast_failed"
	}

	lt.metrics.RecordTransaction(success, time.Since(start), errType)
}

// performQuery performs a test query
func (lt *LoadTester) performQuery(workerID int) {
	start := time.Now()

	// TODO: Implement actual query using REST API
	time.Sleep(time.Millisecond * 5)

	success := true
	lt.metrics.RecordQuery(success, time.Since(start))
}

// performDEXOperation performs a DEX-specific operation
func (lt *LoadTester) performDEXOperation(workerID int) {
	start := time.Now()

	// TODO: Implement DEX swap simulation
	time.Sleep(time.Millisecond * 15)

	success := true
	errType := ""
	lt.metrics.RecordTransaction(success, time.Since(start), errType)
}

// reportMetrics periodically reports current metrics
func (lt *LoadTester) reportMetrics() {
	ticker := time.NewTicker(lt.config.ReportInterval)
	defer ticker.Stop()

	for {
		select {
		case <-lt.ctx.Done():
			return
		case <-ticker.C:
			elapsed := time.Since(lt.metrics.StartTime).Seconds()
			tps := float64(lt.metrics.TotalTxSuccessful) / elapsed
			qps := float64(lt.metrics.TotalQueries) / elapsed

			log.Printf("[Progress] TX: %d (%.2f tps), Queries: %d (%.2f qps), Errors: %d",
				lt.metrics.TotalTxSubmitted,
				tps,
				lt.metrics.TotalQueries,
				qps,
				lt.metrics.TotalTxFailed,
			)
		}
	}
}

// SaveReport saves the final report to a file
func (lt *LoadTester) SaveReport(filename string) error {
	report := lt.metrics.GetReport()

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal report: %w", err)
	}

	if err := os.WriteFile(filename, data, 0644); err != nil {
		return fmt.Errorf("failed to write report: %w", err)
	}

	log.Printf("Report saved to %s", filename)
	return nil
}

func main() {
	// Parse command-line flags
	rpcEndpoint := flag.String("rpc", "http://localhost:26657", "RPC endpoint")
	apiEndpoint := flag.String("api", "http://localhost:1317", "API endpoint")
	chainID := flag.String("chain-id", "paw-testnet-1", "Chain ID")
	duration := flag.Duration("duration", 5*time.Minute, "Test duration")
	concurrency := flag.Int("concurrency", 10, "Number of concurrent workers")
	txRate := flag.Int("rate", 100, "Target transactions per second")
	testType := flag.String("type", "transactions", "Test type: transactions, queries, mixed, dex")
	reportInterval := flag.Duration("report-interval", 10*time.Second, "Metrics reporting interval")
	outputFile := flag.String("output", "load-test-report.json", "Output file for report")

	flag.Parse()

	config := &LoadTestConfig{
		RPCEndpoint:    *rpcEndpoint,
		APIEndpoint:    *apiEndpoint,
		ChainID:        *chainID,
		Duration:       *duration,
		Concurrency:    *concurrency,
		TxRate:         *txRate,
		TestType:       *testType,
		ReportInterval: *reportInterval,
	}

	tester := NewLoadTester(config)

	log.Println("PAW Blockchain Load Tester")
	log.Println("===========================")

	if err := tester.Run(); err != nil {
		log.Fatalf("Load test failed: %v", err)
	}

	// Print final report
	report := tester.metrics.GetReport()
	fmt.Println("\n=== Load Test Results ===")
	fmt.Printf("Duration: %.2f seconds\n", report["duration_seconds"])
	fmt.Printf("Total Transactions: %d\n", report["total_tx_submitted"])
	fmt.Printf("Successful: %d\n", report["total_tx_successful"])
	fmt.Printf("Failed: %d\n", report["total_tx_failed"])
	fmt.Printf("TPS: %.2f\n", report["transactions_per_sec"])
	fmt.Printf("Success Rate: %.2f%%\n", report["success_rate"])
	fmt.Printf("Avg Latency: %d ms\n", report["avg_latency_ms"])
	fmt.Printf("Min Latency: %d ms\n", report["min_latency_ms"])
	fmt.Printf("Max Latency: %d ms\n", report["max_latency_ms"])

	// Save detailed report
	if err := tester.SaveReport(*outputFile); err != nil {
		log.Printf("Failed to save report: %v", err)
	}
}
