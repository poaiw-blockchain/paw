// Package metrics provides Prometheus metrics for the PAW Faucet.
// This implementation follows the patterns from FAUCET_METRICS_PYTHON.md
// adapted for Go/Prometheus client_golang.
package metrics

import (
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// === Request Counters ===

	// RequestsTotal counts total faucet requests by status (success, failed, rate_limited)
	RequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "faucet_requests_total",
			Help: "Total faucet requests by status",
		},
		[]string{"status"},
	)

	// TokensDistributed counts total tokens distributed (in base denomination)
	TokensDistributed = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "faucet_tokens_distributed_total",
			Help: "Total tokens distributed (in base denomination)",
		},
	)

	// === Security Counters ===

	// RateLimitHits counts rate limit violations by type
	RateLimitHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "faucet_rate_limit_hits_total",
			Help: "Rate limit violations by type",
		},
		[]string{"type"}, // ip, address, global_daily, wallet_daily
	)

	// CaptchaAttempts counts CAPTCHA verification attempts by result
	CaptchaAttempts = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "faucet_captcha_attempts_total",
			Help: "CAPTCHA verification attempts by result",
		},
		[]string{"result"}, // pass, fail, skipped
	)

	// BlockedRequests counts blocked requests by reason
	BlockedRequests = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "faucet_blocked_requests_total",
			Help: "Blocked requests by reason",
		},
		[]string{"reason"}, // invalid_address, allowlist, balance_cap, mainnet_blocked
	)

	// PowAttempts counts proof-of-work verification attempts
	PowAttempts = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "faucet_pow_attempts_total",
			Help: "Proof-of-work verification attempts by result",
		},
		[]string{"result"}, // pass, fail, skipped
	)

	// === Operational Gauges ===

	// UniqueAddresses tracks total unique addresses served
	UniqueAddresses = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "faucet_unique_addresses_total",
			Help: "Total unique addresses served",
		},
	)

	// FaucetBalance tracks current faucet wallet balance
	FaucetBalance = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "faucet_balance",
			Help: "Current faucet wallet balance (in base denomination)",
		},
	)

	// RedisConnected tracks Redis connection status (1=connected, 0=disconnected)
	RedisConnected = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "faucet_redis_connected",
			Help: "Redis connection status (1=connected, 0=disconnected)",
		},
	)

	// DatabaseConnected tracks database connection status
	DatabaseConnected = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "faucet_database_connected",
			Help: "Database connection status (1=connected, 0=disconnected)",
		},
	)

	// NodeConnected tracks blockchain node connection status
	NodeConnected = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "faucet_node_connected",
			Help: "Blockchain node connection status (1=connected, 0=disconnected)",
		},
	)

	// NodeSyncing tracks whether the blockchain node is syncing
	NodeSyncing = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "faucet_node_syncing",
			Help: "Whether the blockchain node is syncing (1=syncing, 0=synced)",
		},
	)

	// BlockHeight tracks current blockchain height
	BlockHeight = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "faucet_block_height",
			Help: "Current blockchain block height",
		},
	)

	// UptimeSeconds tracks faucet uptime
	UptimeSeconds = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "faucet_uptime_seconds",
			Help: "Faucet uptime in seconds",
		},
	)

	// === Daily Statistics Gauges ===

	// RequestsLast24h tracks requests in the last 24 hours
	RequestsLast24h = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "faucet_requests_last_24h",
			Help: "Number of requests in the last 24 hours",
		},
	)

	// RequestsLastHour tracks requests in the last hour
	RequestsLastHour = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "faucet_requests_last_hour",
			Help: "Number of requests in the last hour",
		},
	)

	// === Histograms ===

	// RequestDuration tracks request processing duration
	RequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "faucet_request_duration_seconds",
			Help:    "Request processing duration in seconds",
			Buckets: []float64{0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0},
		},
		[]string{"endpoint"},
	)

	// NodeLatency tracks blockchain node API call latency
	NodeLatency = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "faucet_node_latency_seconds",
			Help:    "Blockchain node API call latency in seconds",
			Buckets: []float64{0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0},
		},
	)

	// TransactionLatency tracks transaction broadcast latency
	TransactionLatency = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "faucet_transaction_latency_seconds",
			Help:    "Transaction broadcast latency in seconds",
			Buckets: []float64{0.5, 1.0, 2.5, 5.0, 10.0, 30.0, 60.0},
		},
	)

	// === Info Gauge ===

	// FaucetInfo provides static faucet configuration information
	FaucetInfo = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "faucet_info",
			Help: "Faucet build and configuration information",
		},
		[]string{"version", "network", "chain_id", "denom", "amount_per_request"},
	)
)

// StartTime tracks when the faucet started
var StartTime time.Time

// Initialize sets up initial metric values
func Initialize(version, network, chainID, denom string, amountPerRequest int64) {
	StartTime = time.Now()

	// Set static info
	FaucetInfo.WithLabelValues(
		version,
		network,
		chainID,
		denom,
		formatAmount(amountPerRequest),
	).Set(1)
}

// RecordRequest records a faucet request with the given status
func RecordRequest(status string, amount int64) {
	RequestsTotal.WithLabelValues(status).Inc()
	if status == "success" && amount > 0 {
		TokensDistributed.Add(float64(amount))
	}
}

// RecordRateLimit records a rate limit violation
func RecordRateLimit(limitType string) {
	RateLimitHits.WithLabelValues(limitType).Inc()
}

// RecordCaptcha records a CAPTCHA verification result
func RecordCaptcha(result string) {
	CaptchaAttempts.WithLabelValues(result).Inc()
}

// RecordBlocked records a blocked request
func RecordBlocked(reason string) {
	BlockedRequests.WithLabelValues(reason).Inc()
}

// RecordPow records a proof-of-work verification result
func RecordPow(result string) {
	PowAttempts.WithLabelValues(result).Inc()
}

// UpdateBalance updates the faucet balance gauge
func UpdateBalance(balance int64) {
	FaucetBalance.Set(float64(balance))
}

// UpdateUniqueAddresses updates the unique addresses gauge
func UpdateUniqueAddresses(count int64) {
	UniqueAddresses.Set(float64(count))
}

// UpdateRedisStatus updates Redis connection status
func UpdateRedisStatus(connected bool) {
	if connected {
		RedisConnected.Set(1)
	} else {
		RedisConnected.Set(0)
	}
}

// UpdateDatabaseStatus updates database connection status
func UpdateDatabaseStatus(connected bool) {
	if connected {
		DatabaseConnected.Set(1)
	} else {
		DatabaseConnected.Set(0)
	}
}

// UpdateNodeStatus updates blockchain node connection status
func UpdateNodeStatus(connected bool, syncing bool, height int64) {
	if connected {
		NodeConnected.Set(1)
	} else {
		NodeConnected.Set(0)
	}

	if syncing {
		NodeSyncing.Set(1)
	} else {
		NodeSyncing.Set(0)
	}

	BlockHeight.Set(float64(height))
}

// UpdateUptime updates the uptime gauge
func UpdateUptime() {
	UptimeSeconds.Set(time.Since(StartTime).Seconds())
}

// UpdateDailyStats updates daily statistics gauges
func UpdateDailyStats(last24h, lastHour int64) {
	RequestsLast24h.Set(float64(last24h))
	RequestsLastHour.Set(float64(lastHour))
}

// ObserveRequestDuration records request duration for an endpoint
func ObserveRequestDuration(endpoint string, duration time.Duration) {
	RequestDuration.WithLabelValues(endpoint).Observe(duration.Seconds())
}

// ObserveNodeLatency records node API latency
func ObserveNodeLatency(duration time.Duration) {
	NodeLatency.Observe(duration.Seconds())
}

// ObserveTransactionLatency records transaction broadcast latency
func ObserveTransactionLatency(duration time.Duration) {
	TransactionLatency.Observe(duration.Seconds())
}

// Timer is a helper for timing operations
type Timer struct {
	start time.Time
}

// NewTimer creates a new timer
func NewTimer() *Timer {
	return &Timer{start: time.Now()}
}

// ObserveDuration observes the duration since the timer was created
func (t *Timer) ObserveDuration(histogram prometheus.Histogram) {
	histogram.Observe(time.Since(t.start).Seconds())
}

// Duration returns the duration since the timer was created
func (t *Timer) Duration() time.Duration {
	return time.Since(t.start)
}

// formatAmount formats an amount as a string
func formatAmount(amount int64) string {
	return fmt.Sprintf("%d", amount)
}
