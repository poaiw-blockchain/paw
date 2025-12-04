package keeper

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// DEXMetrics holds all Prometheus metrics for the DEX module
type DEXMetrics struct {
	// Swap metrics
	SwapsTotal       *prometheus.CounterVec
	SwapVolume       *prometheus.CounterVec
	SwapLatency      prometheus.Histogram
	SwapSlippage     prometheus.Histogram
	SwapFeesCollected *prometheus.CounterVec

	// Liquidity metrics
	LiquidityAdded    *prometheus.CounterVec
	LiquidityRemoved  *prometheus.CounterVec
	PoolReserves      *prometheus.GaugeVec
	LPTokenSupply     *prometheus.GaugeVec
	PoolTVL           *prometheus.GaugeVec

	// Pool metrics
	PoolsTotal           prometheus.Gauge
	PoolCreationRate     prometheus.Counter
	PoolImbalanceRatio   *prometheus.GaugeVec
	PoolFeeTier          *prometheus.GaugeVec

	// Circuit breaker metrics
	CircuitBreakerActive    *prometheus.GaugeVec
	CircuitBreakerTriggers  *prometheus.CounterVec
	CircuitBreakerRecoveries *prometheus.CounterVec

	// Security metrics
	MEVProtections       *prometheus.CounterVec
	RateLimitExceeds     *prometheus.CounterVec
	SuspiciousActivity   *prometheus.CounterVec

	// TWAP metrics
	TWAPUpdates       prometheus.Counter
	TWAPValue         *prometheus.GaugeVec

	// ABCI metrics
	ProtocolFeesDistributed *prometheus.CounterVec
	RateLimitCleanups       prometheus.Counter

	// IBC DEX metrics
	IBCSwapsSent      *prometheus.CounterVec
	IBCSwapsReceived  *prometheus.CounterVec
	IBCTimeouts       *prometheus.CounterVec
}

var (
	dexMetricsOnce sync.Once
	dexMetrics     *DEXMetrics
)

// NewDEXMetrics creates and registers DEX metrics (singleton pattern)
func NewDEXMetrics() *DEXMetrics {
	dexMetricsOnce.Do(func() {
		dexMetrics = &DEXMetrics{
			// Swap metrics
			SwapsTotal: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: "paw",
					Subsystem: "dex",
					Name:      "swaps_total",
					Help:      "Total number of swaps executed",
				},
				[]string{"pool_id", "token_in", "token_out", "status"},
			),
			SwapVolume: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: "paw",
					Subsystem: "dex",
					Name:      "swap_volume_total",
					Help:      "Total swap volume in base units",
				},
				[]string{"pool_id", "denom"},
			),
			SwapLatency: promauto.NewHistogram(
				prometheus.HistogramOpts{
					Namespace: "paw",
					Subsystem: "dex",
					Name:      "swap_latency_seconds",
					Help:      "Swap execution latency in seconds",
					Buckets:   prometheus.DefBuckets,
				},
			),
			SwapSlippage: promauto.NewHistogram(
				prometheus.HistogramOpts{
					Namespace: "paw",
					Subsystem: "dex",
					Name:      "swap_slippage_percent",
					Help:      "Swap slippage percentage",
					Buckets:   []float64{0.1, 0.5, 1.0, 2.0, 5.0, 10.0},
				},
			),
			SwapFeesCollected: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: "paw",
					Subsystem: "dex",
					Name:      "swap_fees_collected_total",
					Help:      "Total swap fees collected",
				},
				[]string{"pool_id", "denom"},
			),

			// Liquidity metrics
			LiquidityAdded: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: "paw",
					Subsystem: "dex",
					Name:      "liquidity_added_total",
					Help:      "Total liquidity added to pools",
				},
				[]string{"pool_id", "denom"},
			),
			LiquidityRemoved: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: "paw",
					Subsystem: "dex",
					Name:      "liquidity_removed_total",
					Help:      "Total liquidity removed from pools",
				},
				[]string{"pool_id", "denom"},
			),
			PoolReserves: promauto.NewGaugeVec(
				prometheus.GaugeOpts{
					Namespace: "paw",
					Subsystem: "dex",
					Name:      "pool_reserves",
					Help:      "Current pool reserves",
				},
				[]string{"pool_id", "denom"},
			),
			LPTokenSupply: promauto.NewGaugeVec(
				prometheus.GaugeOpts{
					Namespace: "paw",
					Subsystem: "dex",
					Name:      "lp_token_supply",
					Help:      "LP token supply per pool",
				},
				[]string{"pool_id"},
			),
			PoolTVL: promauto.NewGaugeVec(
				prometheus.GaugeOpts{
					Namespace: "paw",
					Subsystem: "dex",
					Name:      "pool_tvl_total",
					Help:      "Total Value Locked in pool",
				},
				[]string{"pool_id"},
			),

			// Pool metrics
			PoolsTotal: promauto.NewGauge(
				prometheus.GaugeOpts{
					Namespace: "paw",
					Subsystem: "dex",
					Name:      "pools_total",
					Help:      "Total number of liquidity pools",
				},
			),
			PoolCreationRate: promauto.NewCounter(
				prometheus.CounterOpts{
					Namespace: "paw",
					Subsystem: "dex",
					Name:      "pool_creations_total",
					Help:      "Total number of pools created",
				},
			),
			PoolImbalanceRatio: promauto.NewGaugeVec(
				prometheus.GaugeOpts{
					Namespace: "paw",
					Subsystem: "dex",
					Name:      "pool_imbalance_ratio",
					Help:      "Pool reserve ratio (reserve0/reserve1)",
				},
				[]string{"pool_id"},
			),
			PoolFeeTier: promauto.NewGaugeVec(
				prometheus.GaugeOpts{
					Namespace: "paw",
					Subsystem: "dex",
					Name:      "pool_fee_tier",
					Help:      "Pool fee tier in basis points",
				},
				[]string{"pool_id"},
			),

			// Circuit breaker metrics
			CircuitBreakerActive: promauto.NewGaugeVec(
				prometheus.GaugeOpts{
					Namespace: "paw",
					Subsystem: "dex",
					Name:      "circuit_breaker_active",
					Help:      "Circuit breaker activation status (0=inactive, 1=active)",
				},
				[]string{"pool_id"},
			),
			CircuitBreakerTriggers: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: "paw",
					Subsystem: "dex",
					Name:      "circuit_breaker_triggers_total",
					Help:      "Total circuit breaker trigger events",
				},
				[]string{"pool_id", "reason"},
			),
			CircuitBreakerRecoveries: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: "paw",
					Subsystem: "dex",
					Name:      "circuit_breaker_recoveries_total",
					Help:      "Total circuit breaker recovery events",
				},
				[]string{"pool_id"},
			),

			// Security metrics
			MEVProtections: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: "paw",
					Subsystem: "dex",
					Name:      "mev_protections_triggered_total",
					Help:      "MEV protection mechanisms triggered",
				},
				[]string{"pool_id", "protection_type"},
			),
			RateLimitExceeds: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: "paw",
					Subsystem: "dex",
					Name:      "rate_limits_exceeded_total",
					Help:      "Rate limit violations by operation",
				},
				[]string{"user", "operation"},
			),
			SuspiciousActivity: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: "paw",
					Subsystem: "dex",
					Name:      "suspicious_activity_detected_total",
					Help:      "Suspicious activity detections",
				},
				[]string{"type"},
			),

			// TWAP metrics
			TWAPUpdates: promauto.NewCounter(
				prometheus.CounterOpts{
					Namespace: "paw",
					Subsystem: "dex",
					Name:      "twap_updates_total",
					Help:      "Total TWAP update operations",
				},
			),
			TWAPValue: promauto.NewGaugeVec(
				prometheus.GaugeOpts{
					Namespace: "paw",
					Subsystem: "dex",
					Name:      "twap_price",
					Help:      "Time-weighted average price",
				},
				[]string{"pool_id"},
			),

			// ABCI metrics
			ProtocolFeesDistributed: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: "paw",
					Subsystem: "dex",
					Name:      "protocol_fees_distributed_total",
					Help:      "Protocol fees distributed",
				},
				[]string{"denom"},
			),
			RateLimitCleanups: promauto.NewCounter(
				prometheus.CounterOpts{
					Namespace: "paw",
					Subsystem: "dex",
					Name:      "rate_limit_cleanups_total",
					Help:      "Rate limit data cleanup operations",
				},
			),

			// IBC DEX metrics
			IBCSwapsSent: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: "paw",
					Subsystem: "dex",
					Name:      "ibc_swaps_sent_total",
					Help:      "Cross-chain swaps sent to other chains",
				},
				[]string{"destination_chain"},
			),
			IBCSwapsReceived: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: "paw",
					Subsystem: "dex",
					Name:      "ibc_swaps_received_total",
					Help:      "Cross-chain swaps received from other chains",
				},
				[]string{"source_chain"},
			),
			IBCTimeouts: promauto.NewCounterVec(
				prometheus.CounterOpts{
					Namespace: "paw",
					Subsystem: "dex",
					Name:      "ibc_timeouts_total",
					Help:      "IBC packet timeouts",
				},
				[]string{"chain", "packet_type"},
			),
		}
	})
	return dexMetrics
}

// GetDEXMetrics returns the singleton DEX metrics instance
func GetDEXMetrics() *DEXMetrics {
	if dexMetrics == nil {
		return NewDEXMetrics()
	}
	return dexMetrics
}
