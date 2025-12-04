package keeper

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// OracleMetrics holds all Prometheus metrics for the Oracle module
type OracleMetrics struct {
	// Price submission metrics
	PriceSubmissions *prometheus.CounterVec
	AggregatedPrice  *prometheus.GaugeVec
	PriceDeviation   *prometheus.GaugeVec
	PriceAge         *prometheus.GaugeVec

	// Validator metrics
	ValidatorSubmissions *prometheus.CounterVec
	MissedVotes          *prometheus.CounterVec
	SlashingEvents       *prometheus.CounterVec
	ValidatorReputation  *prometheus.GaugeVec

	// Aggregation metrics
	PriceAggregations      *prometheus.CounterVec
	AggregationLatency     prometheus.Histogram
	ConsensusParticipation *prometheus.GaugeVec
	OutliersDetected       *prometheus.CounterVec

	// TWAP metrics
	TWAPValue        *prometheus.GaugeVec
	TWAPWindowSize   *prometheus.GaugeVec
	TWAPUpdates      prometheus.Counter
	ManipulationDetected *prometheus.CounterVec

	// Security metrics
	PriceRejections      *prometheus.CounterVec
	CircuitBreakerTriggers *prometheus.CounterVec
	AnomalousPatterns    *prometheus.CounterVec

	// IBC price feed metrics
	IBCPricesSent     *prometheus.CounterVec
	IBCPricesReceived *prometheus.CounterVec
	IBCTimeouts       *prometheus.CounterVec

	// ABCI metrics
	AssetsTracked prometheus.Gauge
	StaleDataCleanups prometheus.Counter
}

var (
	oracleMetricsOnce sync.Once
	oracleMetrics     *OracleMetrics
)

// NewOracleMetrics creates and registers Oracle metrics (singleton pattern)
func NewOracleMetrics() *OracleMetrics {
	oracleMetricsOnce.Do(func() {
		oracleMetrics = &OracleMetrics{
			// Price submission metrics
			PriceSubmissions: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: "paw",
					Subsystem: "oracle",
					Name:      "price_submissions_total",
					Help:      "Total price submissions by validator",
				},
				[]string{"asset", "validator"},
			),
			AggregatedPrice: promauto.NewGaugeVec(
				prometheus.GaugeOpts{
					Namespace: "paw",
					Subsystem: "oracle",
					Name:      "aggregated_price",
					Help:      "Current aggregated price for asset",
				},
				[]string{"asset"},
			),
			PriceDeviation: promauto.NewGaugeVec(
				prometheus.GaugeOpts{
					Namespace: "paw",
					Subsystem: "oracle",
					Name:      "price_deviation_percent",
					Help:      "Validator price deviation from median",
				},
				[]string{"asset", "validator"},
			),
			PriceAge: promauto.NewGaugeVec(
				prometheus.GaugeOpts{
					Namespace: "paw",
					Subsystem: "oracle",
					Name:      "price_age_seconds",
					Help:      "Seconds since last price update",
				},
				[]string{"asset"},
			),

			// Validator metrics
			ValidatorSubmissions: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: "paw",
					Subsystem: "oracle",
					Name:      "validator_submissions_total",
					Help:      "Total submissions per validator",
				},
				[]string{"validator", "asset"},
			),
			MissedVotes: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: "paw",
					Subsystem: "oracle",
					Name:      "missed_votes_total",
					Help:      "Missed oracle votes by validator",
				},
				[]string{"validator", "asset"},
			),
			SlashingEvents: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: "paw",
					Subsystem: "oracle",
					Name:      "slashing_events_total",
					Help:      "Validator slashing events",
				},
				[]string{"validator", "reason"},
			),
			ValidatorReputation: promauto.NewGaugeVec(
				prometheus.GaugeOpts{
					Namespace: "paw",
					Subsystem: "oracle",
					Name:      "validator_reputation_score",
					Help:      "Validator reputation score (0-100)",
				},
				[]string{"validator"},
			),

			// Aggregation metrics
			PriceAggregations: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: "paw",
					Subsystem: "oracle",
					Name:      "price_aggregations_total",
					Help:      "Total price aggregations performed",
				},
				[]string{"asset", "status"},
			),
			AggregationLatency: promauto.NewHistogram(
				prometheus.HistogramOpts{
					Namespace: "paw",
					Subsystem: "oracle",
					Name:      "aggregation_latency_seconds",
					Help:      "Price aggregation processing time",
					Buckets:   prometheus.DefBuckets,
				},
			),
			ConsensusParticipation: promauto.NewGaugeVec(
				prometheus.GaugeOpts{
					Namespace: "paw",
					Subsystem: "oracle",
					Name:      "consensus_participation_rate",
					Help:      "Percentage of validators participating in price consensus",
				},
				[]string{"asset"},
			),
			OutliersDetected: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: "paw",
					Subsystem: "oracle",
					Name:      "outliers_detected_total",
					Help:      "Outlier price submissions detected",
				},
				[]string{"asset", "severity"},
			),

			// TWAP metrics
			TWAPValue: promauto.NewGaugeVec(
				prometheus.GaugeOpts{
					Namespace: "paw",
					Subsystem: "oracle",
					Name:      "twap_price",
					Help:      "Time-weighted average price",
				},
				[]string{"asset"},
			),
			TWAPWindowSize: promauto.NewGaugeVec(
				prometheus.GaugeOpts{
					Namespace: "paw",
					Subsystem: "oracle",
					Name:      "twap_window_seconds",
					Help:      "TWAP calculation window size",
				},
				[]string{"asset"},
			),
			TWAPUpdates: promauto.NewCounter(
				prometheus.CounterOpts{
					Namespace: "paw",
					Subsystem: "oracle",
					Name:      "twap_updates_total",
					Help:      "Total TWAP update operations",
				},
			),
			ManipulationDetected: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: "paw",
					Subsystem: "oracle",
					Name:      "manipulation_detected_total",
					Help:      "Price manipulation attempts detected",
				},
				[]string{"asset", "detection_method"},
			),

			// Security metrics
			PriceRejections: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: "paw",
					Subsystem: "oracle",
					Name:      "price_rejections_total",
					Help:      "Price submissions rejected",
				},
				[]string{"asset", "reason"},
			),
			CircuitBreakerTriggers: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: "paw",
					Subsystem: "oracle",
					Name:      "circuit_breaker_triggers_total",
					Help:      "Oracle circuit breaker activations",
				},
				[]string{"asset", "reason"},
			),
			AnomalousPatterns: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: "paw",
					Subsystem: "oracle",
					Name:      "anomalous_patterns_detected_total",
					Help:      "Anomalous price patterns detected",
				},
				[]string{"asset", "pattern_type"},
			),

			// IBC price feed metrics
			IBCPricesSent: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: "paw",
					Subsystem: "oracle",
					Name:      "ibc_prices_sent_total",
					Help:      "Price updates sent to other chains",
				},
				[]string{"destination_chain", "asset"},
			),
			IBCPricesReceived: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: "paw",
					Subsystem: "oracle",
					Name:      "ibc_prices_received_total",
					Help:      "Price updates received from other chains",
				},
				[]string{"source_chain", "asset"},
			),
			IBCTimeouts: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: "paw",
					Subsystem: "oracle",
					Name:      "ibc_timeouts_total",
					Help:      "IBC price feed timeouts",
				},
				[]string{"chain", "asset"},
			),

			// ABCI metrics
			AssetsTracked: promauto.NewGauge(
				prometheus.GaugeOpts{
					Namespace: "paw",
					Subsystem: "oracle",
					Name:      "assets_tracked_total",
					Help:      "Total number of assets being tracked",
				},
			),
			StaleDataCleanups: promauto.NewCounter(
				prometheus.CounterOpts{
					Namespace: "paw",
					Subsystem: "oracle",
					Name:      "stale_data_cleanups_total",
					Help:      "Stale data cleanup operations",
				},
			),
		}
	})
	return oracleMetrics
}

// GetOracleMetrics returns the singleton Oracle metrics instance
func GetOracleMetrics() *OracleMetrics {
	if oracleMetrics == nil {
		return NewOracleMetrics()
	}
	return oracleMetrics
}
