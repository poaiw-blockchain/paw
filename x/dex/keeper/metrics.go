package keeper

import (
	"context"
	"sync"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	once sync.Once

	// Swap metrics
	swapCount = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "paw_dex_swaps_total",
			Help: "Total number of swaps executed",
		},
		[]string{"pool_id", "token_in", "token_out"},
	)

	swapFailures = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "paw_dex_swap_failures_total",
			Help: "Total number of failed swaps",
		},
		[]string{"pool_id", "reason"},
	)

	swapVolume = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "paw_dex_swap_volume_24h",
			Help: "24-hour swap volume in USD",
		},
		[]string{"token"},
	)

	swapLatency = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "paw_dex_swap_latency_ms",
			Help:    "Swap execution latency in milliseconds",
			Buckets: prometheus.ExponentialBuckets(1, 2, 10), // 1ms to 512ms
		},
		[]string{"pool_id"},
	)

	// Pool metrics
	poolReserves = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "paw_dex_pool_reserves",
			Help: "Current pool reserve amounts",
		},
		[]string{"pool_id", "token"},
	)

	poolLiquidity = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "paw_dex_pool_liquidity_usd",
			Help: "Total pool liquidity in USD",
		},
		[]string{"pool_id"},
	)

	poolCount = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "paw_dex_pools_total",
			Help: "Total number of liquidity pools",
		},
	)

	poolAPY = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "paw_dex_pool_apy",
			Help: "Annualized pool APY percentage",
		},
		[]string{"pool_id"},
	)

	// Liquidity provider metrics
	lpTokensMinted = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "paw_dex_lp_tokens_minted_total",
			Help: "Total LP tokens minted",
		},
		[]string{"pool_id"},
	)

	lpTokensBurned = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "paw_dex_lp_tokens_burned_total",
			Help: "Total LP tokens burned",
		},
		[]string{"pool_id"},
	)

	liquidityAdded = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "paw_dex_liquidity_added_total",
			Help: "Total liquidity added to pools",
		},
		[]string{"pool_id", "token"},
	)

	liquidityRemoved = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "paw_dex_liquidity_removed_total",
			Help: "Total liquidity removed from pools",
		},
		[]string{"pool_id", "token"},
	)

	// Fee metrics
	feesCollected = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "paw_dex_fees_collected_total",
			Help: "Total fees collected from swaps",
		},
		[]string{"pool_id", "token"},
	)

	feesDistributed = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "paw_dex_fees_distributed_total",
			Help: "Total fees distributed to LPs",
		},
		[]string{"pool_id", "token"},
	)

	// Price metrics
	tokenPrice = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "paw_dex_token_price_usd",
			Help: "Current token price in USD",
		},
		[]string{"token"},
	)

	priceImpact = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "paw_dex_price_impact_percent",
			Help:    "Price impact of swaps in percentage",
			Buckets: []float64{0.1, 0.5, 1.0, 2.0, 5.0, 10.0, 20.0},
		},
		[]string{"pool_id"},
	)

	// Slippage metrics
	slippageActual = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "paw_dex_slippage_actual_percent",
			Help:    "Actual slippage experienced in swaps",
			Buckets: []float64{0.1, 0.5, 1.0, 2.0, 5.0, 10.0},
		},
		[]string{"pool_id"},
	)

	slippageExceeded = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "paw_dex_slippage_exceeded_total",
			Help: "Number of times slippage tolerance was exceeded",
		},
		[]string{"pool_id"},
	)

	// Arbitrage metrics
	arbitrageOpportunities = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "paw_dex_arbitrage_opportunities_total",
			Help: "Number of arbitrage opportunities detected",
		},
		[]string{"pool_pair"},
	)

	// Impermanent loss metrics
	impermanentLoss = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "paw_dex_impermanent_loss_percent",
			Help: "Estimated impermanent loss for LPs",
		},
		[]string{"pool_id"},
	)
)

// MetricsCollector provides methods to record DEX metrics
type MetricsCollector struct {
	ctx context.Context
}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector(ctx context.Context) *MetricsCollector {
	once.Do(func() {
		// Register custom collectors if needed
		// This ensures metrics are only registered once
	})

	return &MetricsCollector{
		ctx: ctx,
	}
}

// RecordSwap records a successful swap
func (mc *MetricsCollector) RecordSwap(poolID, tokenIn, tokenOut string, latencyMs float64) {
	swapCount.WithLabelValues(poolID, tokenIn, tokenOut).Inc()
	swapLatency.WithLabelValues(poolID).Observe(latencyMs)
}

// RecordSwapFailure records a failed swap
func (mc *MetricsCollector) RecordSwapFailure(poolID, reason string) {
	swapFailures.WithLabelValues(poolID, reason).Inc()
}

// RecordSwapVolume records swap volume for a token
func (mc *MetricsCollector) RecordSwapVolume(token string, volumeUSD float64) {
	swapVolume.WithLabelValues(token).Set(volumeUSD)
}

// RecordPoolReserves updates pool reserve metrics
func (mc *MetricsCollector) RecordPoolReserves(poolID, token string, amount float64) {
	poolReserves.WithLabelValues(poolID, token).Set(amount)
}

// RecordPoolLiquidity updates total pool liquidity
func (mc *MetricsCollector) RecordPoolLiquidity(poolID string, liquidityUSD float64) {
	poolLiquidity.WithLabelValues(poolID).Set(liquidityUSD)
}

// RecordPoolCount updates the total number of pools
func (mc *MetricsCollector) RecordPoolCount(count float64) {
	poolCount.Set(count)
}

// RecordPoolAPY records the APY for a pool
func (mc *MetricsCollector) RecordPoolAPY(poolID string, apy float64) {
	poolAPY.WithLabelValues(poolID).Set(apy)
}

// RecordLPTokensMinted records LP tokens minted
func (mc *MetricsCollector) RecordLPTokensMinted(poolID string, amount float64) {
	lpTokensMinted.WithLabelValues(poolID).Add(amount)
}

// RecordLPTokensBurned records LP tokens burned
func (mc *MetricsCollector) RecordLPTokensBurned(poolID string, amount float64) {
	lpTokensBurned.WithLabelValues(poolID).Add(amount)
}

// RecordLiquidityAdded records liquidity added to a pool
func (mc *MetricsCollector) RecordLiquidityAdded(poolID, token string, amount float64) {
	liquidityAdded.WithLabelValues(poolID, token).Add(amount)
}

// RecordLiquidityRemoved records liquidity removed from a pool
func (mc *MetricsCollector) RecordLiquidityRemoved(poolID, token string, amount float64) {
	liquidityRemoved.WithLabelValues(poolID, token).Add(amount)
}

// RecordFeesCollected records fees collected from swaps
func (mc *MetricsCollector) RecordFeesCollected(poolID, token string, amount float64) {
	feesCollected.WithLabelValues(poolID, token).Add(amount)
}

// RecordFeesDistributed records fees distributed to LPs
func (mc *MetricsCollector) RecordFeesDistributed(poolID, token string, amount float64) {
	feesDistributed.WithLabelValues(poolID, token).Add(amount)
}

// RecordTokenPrice records current token price
func (mc *MetricsCollector) RecordTokenPrice(token string, priceUSD float64) {
	tokenPrice.WithLabelValues(token).Set(priceUSD)
}

// RecordPriceImpact records price impact of a swap
func (mc *MetricsCollector) RecordPriceImpact(poolID string, impactPercent float64) {
	priceImpact.WithLabelValues(poolID).Observe(impactPercent)
}

// RecordSlippage records actual slippage
func (mc *MetricsCollector) RecordSlippage(poolID string, slippagePercent float64) {
	slippageActual.WithLabelValues(poolID).Observe(slippagePercent)
}

// RecordSlippageExceeded records when slippage tolerance is exceeded
func (mc *MetricsCollector) RecordSlippageExceeded(poolID string) {
	slippageExceeded.WithLabelValues(poolID).Inc()
}

// RecordArbitrageOpportunity records an arbitrage opportunity
func (mc *MetricsCollector) RecordArbitrageOpportunity(poolPair string) {
	arbitrageOpportunities.WithLabelValues(poolPair).Inc()
}

// RecordImpermanentLoss records estimated impermanent loss
func (mc *MetricsCollector) RecordImpermanentLoss(poolID string, lossPercent float64) {
	impermanentLoss.WithLabelValues(poolID).Set(lossPercent)
}
