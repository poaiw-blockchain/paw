package analytics

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	rpcclient "github.com/cometbft/cometbft/rpc/client/http"
	"github.com/gin-gonic/gin"
	"github.com/paw-chain/paw/explorer/indexer/internal/cache"
	"github.com/paw-chain/paw/explorer/indexer/internal/database"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	analyticsQueriesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "explorer_analytics_queries_total",
			Help: "Total number of analytics queries",
		},
		[]string{"query_type"},
	)

	analyticsQueryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "explorer_analytics_query_duration_seconds",
			Help: "Analytics query duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"query_type"},
	)
)

// AnalyticsService provides advanced analytics capabilities
type AnalyticsService struct {
	db        *database.DB
	cache     *cache.RedisCache
	rpcClient *rpcclient.HTTP
	mu        sync.RWMutex
	config    AnalyticsConfig
}

// AnalyticsConfig holds analytics configuration
type AnalyticsConfig struct {
	CacheDuration   time.Duration
	RefreshInterval time.Duration
	HistoryDepth    int
	RPCURL          string
}

// NewAnalyticsService creates a new analytics service
func NewAnalyticsService(db *database.DB, cache *cache.RedisCache, config AnalyticsConfig) (*AnalyticsService, error) {
	// Create RPC client for consensus metrics
	var rpcClient *rpcclient.HTTP
	if config.RPCURL != "" {
		client, err := rpcclient.New(config.RPCURL, "/websocket")
		if err != nil {
			return nil, fmt.Errorf("failed to create RPC client: %w", err)
		}
		rpcClient = client
	}

	return &AnalyticsService{
		db:        db,
		cache:     cache,
		rpcClient: rpcClient,
		config:    config,
	}, nil
}

// ============================================================================
// NETWORK HEALTH MONITORING
// ============================================================================

// NetworkHealth represents overall network health metrics
type NetworkHealth struct {
	Status              string              `json:"status"`
	Score               float64             `json:"score"`
	BlockProduction     BlockProductionMetrics     `json:"block_production"`
	ValidatorHealth     ValidatorHealthMetrics     `json:"validator_health"`
	TransactionMetrics  TransactionMetrics         `json:"transaction_metrics"`
	ConsensusMetrics    ConsensusMetrics           `json:"consensus_metrics"`
	Timestamp           time.Time           `json:"timestamp"`
}

// BlockProductionMetrics tracks block production health
type BlockProductionMetrics struct {
	AverageBlockTime     float64 `json:"average_block_time"`
	BlockTimeSDeviation  float64 `json:"block_time_std_deviation"`
	MissedBlocks         int     `json:"missed_blocks"`
	ConsecutiveBlocks    int     `json:"consecutive_blocks"`
	EmptyBlockRate       float64 `json:"empty_block_rate"`
	HealthScore          float64 `json:"health_score"`
}

// ValidatorHealthMetrics tracks validator performance
type ValidatorHealthMetrics struct {
	ActiveValidators     int     `json:"active_validators"`
	JailedValidators     int     `json:"jailed_validators"`
	AverageUptime        float64 `json:"average_uptime"`
	TotalVotingPower     int64   `json:"total_voting_power"`
	NakamotoCoefficient  int     `json:"nakamoto_coefficient"`
	HealthScore          float64 `json:"health_score"`
}

// TransactionMetrics tracks transaction health
type TransactionMetrics struct {
	CurrentTPS           float64 `json:"current_tps"`
	AverageTPS           float64 `json:"average_tps"`
	PeakTPS              float64 `json:"peak_tps"`
	SuccessRate          float64 `json:"success_rate"`
	AverageGasPrice      float64 `json:"average_gas_price"`
	MempoolSize          int     `json:"mempool_size"`
	HealthScore          float64 `json:"health_score"`
}

// ConsensusMetrics tracks consensus health
type ConsensusMetrics struct {
	RoundsPerBlock       float64 `json:"rounds_per_block"`
	TimeToFinality       float64 `json:"time_to_finality"`
	PrecommitRate        float64 `json:"precommit_rate"`
	PrevoteRate          float64 `json:"prevote_rate"`
	HealthScore          float64 `json:"health_score"`
}

// GetNetworkHealth computes comprehensive network health metrics
func (s *AnalyticsService) GetNetworkHealth(ctx context.Context) (*NetworkHealth, error) {
	start := time.Now()
	defer func() {
		analyticsQueriesTotal.WithLabelValues("network_health").Inc()
		analyticsQueryDuration.WithLabelValues("network_health").Observe(time.Since(start).Seconds())
	}()

	// Try cache first
	cacheKey := "analytics:network_health"
	if cached, err := s.cache.Get(ctx, cacheKey); err == nil {
		var health NetworkHealth
		if err := json.Unmarshal(cached, &health); err == nil {
			return &health, nil
		}
	}

	// Compute metrics in parallel
	var (
		blockMetrics      BlockProductionMetrics
		validatorMetrics  ValidatorHealthMetrics
		txMetrics         TransactionMetrics
		consensusMetrics  ConsensusMetrics
		wg                sync.WaitGroup
		errCh             = make(chan error, 4)
	)

	wg.Add(4)

	// Compute block production metrics
	go func() {
		defer wg.Done()
		metrics, err := s.computeBlockProductionMetrics(ctx)
		if err != nil {
			errCh <- fmt.Errorf("block production metrics: %w", err)
			return
		}
		blockMetrics = metrics
	}()

	// Compute validator metrics
	go func() {
		defer wg.Done()
		metrics, err := s.computeValidatorHealthMetrics(ctx)
		if err != nil {
			errCh <- fmt.Errorf("validator health metrics: %w", err)
			return
		}
		validatorMetrics = metrics
	}()

	// Compute transaction metrics
	go func() {
		defer wg.Done()
		metrics, err := s.computeTransactionMetrics(ctx)
		if err != nil {
			errCh <- fmt.Errorf("transaction metrics: %w", err)
			return
		}
		txMetrics = metrics
	}()

	// Compute consensus metrics
	go func() {
		defer wg.Done()
		metrics, err := s.computeConsensusMetrics(ctx)
		if err != nil {
			errCh <- fmt.Errorf("consensus metrics: %w", err)
			return
		}
		consensusMetrics = metrics
	}()

	wg.Wait()
	close(errCh)

	// Check for errors
	if err := <-errCh; err != nil {
		return nil, err
	}

	// Calculate overall health score
	overallScore := (blockMetrics.HealthScore +
		validatorMetrics.HealthScore +
		txMetrics.HealthScore +
		consensusMetrics.HealthScore) / 4.0

	status := "healthy"
	if overallScore < 0.5 {
		status = "critical"
	} else if overallScore < 0.7 {
		status = "degraded"
	} else if overallScore < 0.9 {
		status = "warning"
	}

	health := &NetworkHealth{
		Status:             status,
		Score:              overallScore,
		BlockProduction:    blockMetrics,
		ValidatorHealth:    validatorMetrics,
		TransactionMetrics: txMetrics,
		ConsensusMetrics:   consensusMetrics,
		Timestamp:          time.Now(),
	}

	// Cache result
	if data, err := json.Marshal(health); err == nil {
		s.cache.Set(ctx, cacheKey, data, s.config.CacheDuration)
	}

	return health, nil
}

func (s *AnalyticsService) computeBlockProductionMetrics(ctx context.Context) (BlockProductionMetrics, error) {
	// Query last 1000 blocks
	query := `
		SELECT
			AVG(EXTRACT(EPOCH FROM (timestamp - LAG(timestamp) OVER (ORDER BY height)))) as avg_block_time,
			STDDEV(EXTRACT(EPOCH FROM (timestamp - LAG(timestamp) OVER (ORDER BY height)))) as std_dev,
			COUNT(*) FILTER (WHERE tx_count = 0) as empty_blocks,
			COUNT(*) as total_blocks
		FROM (
			SELECT height, timestamp, tx_count
			FROM blocks
			ORDER BY height DESC
			LIMIT 1000
		) sub
	`

	var (
		avgBlockTime   float64
		stdDev         float64
		emptyBlocks    int
		totalBlocks    int
	)

	err := s.db.QueryRowContext(ctx, query).Scan(&avgBlockTime, &stdDev, &emptyBlocks, &totalBlocks)
	if err != nil {
		return BlockProductionMetrics{}, err
	}

	emptyBlockRate := float64(emptyBlocks) / float64(totalBlocks)

	// Calculate health score (0-1)
	healthScore := 1.0
	if avgBlockTime > 7.0 {
		healthScore -= 0.2
	}
	if stdDev > 2.0 {
		healthScore -= 0.2
	}
	if emptyBlockRate > 0.1 {
		healthScore -= 0.1
	}

	return BlockProductionMetrics{
		AverageBlockTime:    avgBlockTime,
		BlockTimeSDeviation: stdDev,
		MissedBlocks:        0, // Would need to track from consensus
		ConsecutiveBlocks:   totalBlocks,
		EmptyBlockRate:      emptyBlockRate,
		HealthScore:         healthScore,
	}, nil
}

func (s *AnalyticsService) computeValidatorHealthMetrics(ctx context.Context) (ValidatorHealthMetrics, error) {
	query := `
		SELECT
			COUNT(*) FILTER (WHERE status = 'bonded') as active,
			COUNT(*) FILTER (WHERE jailed = true) as jailed,
			SUM(voting_power) as total_power
		FROM validators
	`

	var (
		active      int
		jailed      int
		totalPower  int64
	)

	err := s.db.QueryRowContext(ctx, query).Scan(&active, &jailed, &totalPower)
	if err != nil {
		return ValidatorHealthMetrics{}, err
	}

	// Calculate average uptime
	uptimeQuery := `
		SELECT AVG(uptime_percentage) as avg_uptime
		FROM (
			SELECT
				validator_address,
				COUNT(*) FILTER (WHERE signed = true)::float / COUNT(*)::float * 100 as uptime_percentage
			FROM validator_uptime
			WHERE timestamp > NOW() - INTERVAL '30 days'
			GROUP BY validator_address
		) sub
	`

	var avgUptime float64
	err = s.db.QueryRowContext(ctx, uptimeQuery).Scan(&avgUptime)
	if err != nil {
		avgUptime = 0
	}

	// Calculate Nakamoto Coefficient
	nakamoto := s.calculateNakamotoCoefficient(ctx)

	// Calculate health score
	healthScore := 1.0
	if active < 50 {
		healthScore -= 0.2
	}
	if float64(jailed)/float64(active+jailed) > 0.05 {
		healthScore -= 0.2
	}
	if avgUptime < 95.0 {
		healthScore -= 0.2
	}
	if nakamoto < 3 {
		healthScore -= 0.2
	}

	return ValidatorHealthMetrics{
		ActiveValidators:    active,
		JailedValidators:    jailed,
		AverageUptime:       avgUptime,
		TotalVotingPower:    totalPower,
		NakamotoCoefficient: nakamoto,
		HealthScore:         healthScore,
	}, nil
}

func (s *AnalyticsService) computeTransactionMetrics(ctx context.Context) (TransactionMetrics, error) {
	query := `
		SELECT
			COUNT(*)::float / 3600.0 as current_tps,
			COUNT(*) FILTER (WHERE status = 'success')::float / COUNT(*)::float as success_rate,
			AVG(CAST(fee_amount AS NUMERIC)) as avg_gas_price
		FROM transactions
		WHERE timestamp > NOW() - INTERVAL '1 hour'
	`

	var (
		currentTPS   float64
		successRate  float64
		avgGasPrice  float64
	)

	err := s.db.QueryRowContext(ctx, query).Scan(&currentTPS, &successRate, &avgGasPrice)
	if err != nil {
		return TransactionMetrics{}, err
	}

	// Get average and peak TPS
	avgQuery := `
		WITH hourly_stats AS (
			SELECT
				DATE_TRUNC('hour', timestamp) as hour,
				COUNT(*)::float / 3600.0 as tps
			FROM transactions
			WHERE timestamp > NOW() - INTERVAL '24 hours'
			GROUP BY hour
		)
		SELECT AVG(tps), MAX(tps)
		FROM hourly_stats
	`

	var averageTPS, peakTPS float64
	err = s.db.QueryRowContext(ctx, avgQuery).Scan(&averageTPS, &peakTPS)
	if err != nil {
		averageTPS, peakTPS = currentTPS, currentTPS
	}

	// Calculate health score
	healthScore := 1.0
	if successRate < 0.95 {
		healthScore -= 0.3
	}
	if currentTPS < averageTPS*0.5 {
		healthScore -= 0.2
	}

	return TransactionMetrics{
		CurrentTPS:      currentTPS,
		AverageTPS:      averageTPS,
		PeakTPS:         peakTPS,
		SuccessRate:     successRate,
		AverageGasPrice: avgGasPrice,
		MempoolSize:     0, // Would need to query from node
		HealthScore:     healthScore,
	}, nil
}

func (s *AnalyticsService) computeConsensusMetrics(ctx context.Context) (ConsensusMetrics, error) {
	// Return default values if no RPC client is configured
	if s.rpcClient == nil {
		return ConsensusMetrics{
			RoundsPerBlock:  1.0,
			TimeToFinality:  6.0,
			PrecommitRate:   0.95,
			PrevoteRate:     0.95,
			HealthScore:     0.90,
		}, nil
	}

	// Get current consensus state
	status, err := s.rpcClient.Status(ctx)
	if err != nil {
		return ConsensusMetrics{}, fmt.Errorf("failed to get node status: %w", err)
	}

	latestHeight := status.SyncInfo.LatestBlockHeight

	// Query last 100 blocks to calculate consensus metrics
	const blockSampleSize = 100
	var (
		totalRounds         int64
		totalCommitSigs     int64
		totalPossibleSigs   int64
		blockTimes          []float64
		validBlockCount     int64
	)

	// Get validators for calculating signature rates
	validators, err := s.rpcClient.Validators(ctx, &latestHeight, nil, nil)
	if err != nil {
		return ConsensusMetrics{}, fmt.Errorf("failed to get validators: %w", err)
	}
	totalValidators := int64(len(validators.Validators))

	// Sample recent blocks
	startHeight := latestHeight - blockSampleSize + 1
	if startHeight < 1 {
		startHeight = 1
	}

	var prevBlockTime time.Time
	for height := startHeight; height <= latestHeight; height++ {
		block, err := s.rpcClient.Block(ctx, &height)
		if err != nil {
			continue // Skip blocks we can't fetch
		}

		validBlockCount++

		// Calculate block time
		if !prevBlockTime.IsZero() {
			blockTime := block.Block.Time.Sub(prevBlockTime).Seconds()
			if blockTime > 0 && blockTime < 60 { // Sanity check
				blockTimes = append(blockTimes, blockTime)
			}
		}
		prevBlockTime = block.Block.Time

		// Count commit signatures
		commitSigs := int64(len(block.Block.LastCommit.Signatures))
		totalCommitSigs += commitSigs
		totalPossibleSigs += totalValidators

		// Estimate rounds from block header
		// CometBFT includes round information in the block
		if block.Block.LastCommit != nil {
			totalRounds += int64(block.Block.LastCommit.Round) + 1
		} else {
			totalRounds++ // Assume 1 round if no commit info
		}
	}

	// Calculate average rounds per block
	avgRoundsPerBlock := 1.0
	if validBlockCount > 0 {
		avgRoundsPerBlock = float64(totalRounds) / float64(validBlockCount)
	}

	// Calculate precommit/prevote rates (using commit signatures as proxy)
	commitRate := 0.0
	if totalPossibleSigs > 0 {
		commitRate = float64(totalCommitSigs) / float64(totalPossibleSigs)
	}

	// Calculate average time to finality (average block time)
	avgTimeToFinality := 6.0 // Default
	if len(blockTimes) > 0 {
		sum := 0.0
		for _, bt := range blockTimes {
			sum += bt
		}
		avgTimeToFinality = sum / float64(len(blockTimes))
	}

	// Both prevote and precommit rates are similar in CometBFT
	// Using commit rate as a good proxy for both
	prevoteRate := commitRate
	precommitRate := commitRate

	// Calculate health score based on metrics
	healthScore := 1.0

	// Penalize if too many rounds per block
	if avgRoundsPerBlock > 1.5 {
		healthScore -= 0.2
	} else if avgRoundsPerBlock > 1.2 {
		healthScore -= 0.1
	}

	// Penalize if block time is too high
	if avgTimeToFinality > 10.0 {
		healthScore -= 0.2
	} else if avgTimeToFinality > 7.0 {
		healthScore -= 0.1
	}

	// Penalize if commit rate is low
	if commitRate < 0.9 {
		healthScore -= 0.3
	} else if commitRate < 0.95 {
		healthScore -= 0.1
	}

	// Ensure health score is in valid range
	if healthScore < 0.0 {
		healthScore = 0.0
	}

	return ConsensusMetrics{
		RoundsPerBlock:  avgRoundsPerBlock,
		TimeToFinality:  avgTimeToFinality,
		PrecommitRate:   precommitRate,
		PrevoteRate:     prevoteRate,
		HealthScore:     healthScore,
	}, nil
}

func (s *AnalyticsService) calculateNakamotoCoefficient(ctx context.Context) int {
	query := `
		WITH cumulative AS (
			SELECT
				voting_power,
				SUM(voting_power) OVER (ORDER BY voting_power DESC) as cumulative_power,
				SUM(voting_power) OVER () as total_power
			FROM validators
			WHERE status = 'bonded'
		)
		SELECT COUNT(*)
		FROM cumulative
		WHERE cumulative_power <= total_power / 3
	`

	var coefficient int
	err := s.db.QueryRowContext(ctx, query).Scan(&coefficient)
	if err != nil {
		return 0
	}

	return coefficient + 1
}

// ============================================================================
// TRANSACTION VOLUME ANALYTICS
// ============================================================================

// TransactionVolumeData represents transaction volume over time
type TransactionVolumeData struct {
	Period     string                   `json:"period"`
	Data       []TransactionDataPoint   `json:"data"`
	Aggregates TransactionAggregates    `json:"aggregates"`
}

// TransactionDataPoint represents a single data point
type TransactionDataPoint struct {
	Timestamp   time.Time `json:"timestamp"`
	TxCount     int       `json:"tx_count"`
	SuccessCount int      `json:"success_count"`
	FailedCount  int      `json:"failed_count"`
	TotalGas     int64    `json:"total_gas"`
	TotalFees    string   `json:"total_fees"`
}

// TransactionAggregates contains aggregate statistics
type TransactionAggregates struct {
	TotalTransactions int     `json:"total_transactions"`
	SuccessRate       float64 `json:"success_rate"`
	AverageTPS        float64 `json:"average_tps"`
	PeakTPS           float64 `json:"peak_tps"`
	TotalGas          int64   `json:"total_gas"`
	TotalFees         string  `json:"total_fees"`
}

// GetTransactionVolumeChart returns transaction volume data for charting
func (s *AnalyticsService) GetTransactionVolumeChart(ctx context.Context, period string) (*TransactionVolumeData, error) {
	start := time.Now()
	defer func() {
		analyticsQueriesTotal.WithLabelValues("transaction_volume").Inc()
		analyticsQueryDuration.WithLabelValues("transaction_volume").Observe(time.Since(start).Seconds())
	}()

	// Determine time range and interval
	var (
		interval   string
		timeRange  string
	)

	switch period {
	case "1h":
		interval = "1 minute"
		timeRange = "1 hour"
	case "24h":
		interval = "1 hour"
		timeRange = "24 hours"
	case "7d":
		interval = "6 hours"
		timeRange = "7 days"
	case "30d":
		interval = "1 day"
		timeRange = "30 days"
	default:
		interval = "1 hour"
		timeRange = "24 hours"
	}

	query := fmt.Sprintf(`
		WITH time_series AS (
			SELECT
				DATE_TRUNC('%s', timestamp) as bucket,
				COUNT(*) as tx_count,
				COUNT(*) FILTER (WHERE status = 'success') as success_count,
				COUNT(*) FILTER (WHERE status != 'success') as failed_count,
				SUM(gas_used) as total_gas,
				SUM(CAST(fee_amount AS NUMERIC)) as total_fees
			FROM transactions
			WHERE timestamp > NOW() - INTERVAL '%s'
			GROUP BY bucket
			ORDER BY bucket
		)
		SELECT
			bucket,
			tx_count,
			success_count,
			failed_count,
			total_gas,
			COALESCE(total_fees, 0) as total_fees
		FROM time_series
	`, interval, timeRange)

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var (
		dataPoints        []TransactionDataPoint
		totalTxs          int
		totalSuccess      int
		totalFailed       int
		totalGas          int64
		totalFees         float64
		maxTPS            float64
	)

	for rows.Next() {
		var dp TransactionDataPoint
		var feesFloat float64

		err := rows.Scan(
			&dp.Timestamp,
			&dp.TxCount,
			&dp.SuccessCount,
			&dp.FailedCount,
			&dp.TotalGas,
			&feesFloat,
		)
		if err != nil {
			return nil, err
		}

		dp.TotalFees = fmt.Sprintf("%.2f", feesFloat)

		dataPoints = append(dataPoints, dp)

		totalTxs += dp.TxCount
		totalSuccess += dp.SuccessCount
		totalFailed += dp.FailedCount
		totalGas += dp.TotalGas
		totalFees += feesFloat

		// Calculate TPS for this bucket
		tps := float64(dp.TxCount) / intervalSeconds(interval)
		if tps > maxTPS {
			maxTPS = tps
		}
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Calculate aggregates
	successRate := 0.0
	if totalTxs > 0 {
		successRate = float64(totalSuccess) / float64(totalTxs)
	}

	avgTPS := 0.0
	if len(dataPoints) > 0 {
		totalSeconds := len(dataPoints) * int(intervalSeconds(interval))
		avgTPS = float64(totalTxs) / float64(totalSeconds)
	}

	return &TransactionVolumeData{
		Period: period,
		Data:   dataPoints,
		Aggregates: TransactionAggregates{
			TotalTransactions: totalTxs,
			SuccessRate:       successRate,
			AverageTPS:        avgTPS,
			PeakTPS:           maxTPS,
			TotalGas:          totalGas,
			TotalFees:         fmt.Sprintf("%.2f", totalFees),
		},
	}, nil
}

func intervalSeconds(interval string) float64 {
	switch interval {
	case "1 minute":
		return 60
	case "1 hour":
		return 3600
	case "6 hours":
		return 21600
	case "1 day":
		return 86400
	default:
		return 3600
	}
}

// ============================================================================
// DEX ANALYTICS
// ============================================================================

// DEXAnalytics contains comprehensive DEX analytics
type DEXAnalytics struct {
	TotalVolume24h    string                `json:"total_volume_24h"`
	TotalVolume7d     string                `json:"total_volume_7d"`
	TotalTVL          string                `json:"total_tvl"`
	TotalPools        int                   `json:"total_pools"`
	TotalTrades24h    int                   `json:"total_trades_24h"`
	TopPools          []PoolAnalytics       `json:"top_pools"`
	VolumeChart       []VolumeDataPoint     `json:"volume_chart"`
	LiquidityChart    []LiquidityDataPoint  `json:"liquidity_chart"`
}

// PoolAnalytics contains analytics for a single pool
type PoolAnalytics struct {
	PoolID          string  `json:"pool_id"`
	TokenA          string  `json:"token_a"`
	TokenB          string  `json:"token_b"`
	TVL             string  `json:"tvl"`
	Volume24h       string  `json:"volume_24h"`
	Volume7d        string  `json:"volume_7d"`
	Trades24h       int     `json:"trades_24h"`
	APR             string  `json:"apr"`
	FeeRevenue24h   string  `json:"fee_revenue_24h"`
	PriceChange24h  float64 `json:"price_change_24h"`
}

// VolumeDataPoint represents a volume data point
type VolumeDataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Volume    string    `json:"volume"`
	Trades    int       `json:"trades"`
}

// LiquidityDataPoint represents a liquidity data point
type LiquidityDataPoint struct {
	Timestamp time.Time `json:"timestamp"`
	TVL       string    `json:"tvl"`
	Pools     int       `json:"pools"`
}

// GetDEXAnalytics returns comprehensive DEX analytics
func (s *AnalyticsService) GetDEXAnalytics(ctx context.Context) (*DEXAnalytics, error) {
	start := time.Now()
	defer func() {
		analyticsQueriesTotal.WithLabelValues("dex_analytics").Inc()
		analyticsQueryDuration.WithLabelValues("dex_analytics").Observe(time.Since(start).Seconds())
	}()

	// Try cache first
	cacheKey := "analytics:dex:overview"
	if cached, err := s.cache.Get(ctx, cacheKey); err == nil {
		var analytics DEXAnalytics
		if err := json.Unmarshal(cached, &analytics); err == nil {
			return &analytics, nil
		}
	}

	// Query aggregate statistics
	aggregateQuery := `
		SELECT
			COALESCE(SUM(CAST(volume_24h AS NUMERIC)), 0) as total_volume_24h,
			COALESCE(SUM(CAST(volume_7d AS NUMERIC)), 0) as total_volume_7d,
			COALESCE(SUM(CAST(tvl AS NUMERIC)), 0) as total_tvl,
			COUNT(*) as total_pools
		FROM dex_pools
	`

	var (
		totalVolume24h float64
		totalVolume7d  float64
		totalTVL       float64
		totalPools     int
	)

	err := s.db.QueryRowContext(ctx, aggregateQuery).Scan(&totalVolume24h, &totalVolume7d, &totalTVL, &totalPools)
	if err != nil {
		return nil, err
	}

	// Count trades in last 24 hours
	tradesQuery := `
		SELECT COUNT(*)
		FROM dex_trades
		WHERE timestamp > NOW() - INTERVAL '24 hours'
	`

	var totalTrades24h int
	err = s.db.QueryRowContext(ctx, tradesQuery).Scan(&totalTrades24h)
	if err != nil {
		totalTrades24h = 0
	}

	// Get top pools
	topPools, err := s.getTopPools(ctx, 10)
	if err != nil {
		topPools = []PoolAnalytics{}
	}

	// Get volume chart data
	volumeChart, err := s.getDEXVolumeChart(ctx, "24h")
	if err != nil {
		volumeChart = []VolumeDataPoint{}
	}

	// Get liquidity chart data
	liquidityChart, err := s.getDEXLiquidityChart(ctx, "7d")
	if err != nil {
		liquidityChart = []LiquidityDataPoint{}
	}

	analytics := &DEXAnalytics{
		TotalVolume24h:   fmt.Sprintf("%.2f", totalVolume24h),
		TotalVolume7d:    fmt.Sprintf("%.2f", totalVolume7d),
		TotalTVL:         fmt.Sprintf("%.2f", totalTVL),
		TotalPools:       totalPools,
		TotalTrades24h:   totalTrades24h,
		TopPools:         topPools,
		VolumeChart:      volumeChart,
		LiquidityChart:   liquidityChart,
	}

	// Cache result
	if data, err := json.Marshal(analytics); err == nil {
		s.cache.Set(ctx, cacheKey, data, s.config.CacheDuration)
	}

	return analytics, nil
}

func (s *AnalyticsService) getTopPools(ctx context.Context, limit int) ([]PoolAnalytics, error) {
	query := fmt.Sprintf(`
		WITH pool_stats AS (
			SELECT
				pool_id,
				COUNT(*) as trade_count
			FROM dex_trades
			WHERE timestamp > NOW() - INTERVAL '24 hours'
			GROUP BY pool_id
		)
		SELECT
			p.pool_id,
			p.token_a,
			p.token_b,
			p.tvl,
			p.volume_24h,
			p.volume_7d,
			COALESCE(ps.trade_count, 0) as trades_24h,
			p.apr
		FROM dex_pools p
		LEFT JOIN pool_stats ps ON p.pool_id = ps.pool_id
		ORDER BY CAST(p.tvl AS NUMERIC) DESC
		LIMIT %d
	`, limit)

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var pools []PoolAnalytics
	for rows.Next() {
		var p PoolAnalytics
		err := rows.Scan(
			&p.PoolID,
			&p.TokenA,
			&p.TokenB,
			&p.TVL,
			&p.Volume24h,
			&p.Volume7d,
			&p.Trades24h,
			&p.APR,
		)
		if err != nil {
			continue
		}

		// Calculate fee revenue (0.3% of volume)
		if vol, err := parseFloat(p.Volume24h); err == nil {
			p.FeeRevenue24h = fmt.Sprintf("%.2f", vol*0.003)
		}

		// Calculate price change from 24h ago
		priceChange, err := s.calculatePriceChange(ctx, p.Id, 24)
		if err == nil {
			p.PriceChange24h = priceChange
		} else {
			p.PriceChange24h = 0.0
		}

		pools = append(pools, p)
	}

	return pools, nil
}

// calculatePriceChange calculates the percentage price change over the specified hours
func (s *AnalyticsService) calculatePriceChange(ctx context.Context, poolID int64, hours int) (float64, error) {
	// Get current price (calculated from reserves)
	currentPriceQuery := `
		SELECT
			CAST(reserve_a AS NUMERIC) / NULLIF(CAST(reserve_b AS NUMERIC), 0) as current_price
		FROM dex_pools
		WHERE id = $1
	`

	var currentPrice float64
	err := s.db.QueryRowContext(ctx, currentPriceQuery, poolID).Scan(&currentPrice)
	if err != nil {
		return 0.0, fmt.Errorf("failed to get current price: %w", err)
	}

	// Get historical price from approximately N hours ago
	// We look for the oldest swap in the time window to get the price at that point
	historicalPriceQuery := `
		SELECT
			CAST(amount_out AS NUMERIC) / NULLIF(CAST(amount_in AS NUMERIC), 0) as historical_price
		FROM dex_trades
		WHERE pool_id = $1
			AND timestamp <= NOW() - INTERVAL '1 hour' * $2
		ORDER BY timestamp DESC
		LIMIT 1
	`

	var historicalPrice float64
	err = s.db.QueryRowContext(ctx, historicalPriceQuery, poolID, hours).Scan(&historicalPrice)
	if err != nil {
		// If no historical data, return 0% change
		return 0.0, nil
	}

	// Calculate percentage change
	if historicalPrice == 0 {
		return 0.0, nil
	}

	priceChange := ((currentPrice - historicalPrice) / historicalPrice) * 100.0
	return priceChange, nil
}

func (s *AnalyticsService) getDEXVolumeChart(ctx context.Context, period string) ([]VolumeDataPoint, error) {
	interval := "1 hour"
	timeRange := "24 hours"

	query := fmt.Sprintf(`
		SELECT
			DATE_TRUNC('%s', timestamp) as bucket,
			SUM(CAST(amount_in AS NUMERIC)) as volume,
			COUNT(*) as trades
		FROM dex_trades
		WHERE timestamp > NOW() - INTERVAL '%s'
		GROUP BY bucket
		ORDER BY bucket
	`, interval, timeRange)

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var dataPoints []VolumeDataPoint
	for rows.Next() {
		var dp VolumeDataPoint
		var volume float64

		err := rows.Scan(&dp.Timestamp, &volume, &dp.Trades)
		if err != nil {
			continue
		}

		dp.Volume = fmt.Sprintf("%.2f", volume)
		dataPoints = append(dataPoints, dp)
	}

	return dataPoints, nil
}

func (s *AnalyticsService) getDEXLiquidityChart(ctx context.Context, period string) ([]LiquidityDataPoint, error) {
	// This would require tracking historical TVL data
	// For now, return current TVL as snapshot
	query := `
		SELECT
			NOW() as timestamp,
			SUM(CAST(tvl AS NUMERIC)) as total_tvl,
			COUNT(*) as pools
		FROM dex_pools
	`

	var dp LiquidityDataPoint
	var tvl float64

	err := s.db.QueryRowContext(ctx, query).Scan(&dp.Timestamp, &tvl, &dp.Pools)
	if err != nil {
		return nil, err
	}

	dp.TVL = fmt.Sprintf("%.2f", tvl)

	return []LiquidityDataPoint{dp}, nil
}

// ============================================================================
// HTTP HANDLERS
// ============================================================================

// SetupRoutes sets up analytics API routes
func (s *AnalyticsService) SetupRoutes(router *gin.Engine) {
	analytics := router.Group("/api/v1/analytics")
	{
		analytics.GET("/network-health", s.handleGetNetworkHealth)
		analytics.GET("/transaction-volume", s.handleGetTransactionVolume)
		analytics.GET("/dex-analytics", s.handleGetDEXAnalytics)
		analytics.GET("/address-growth", s.handleGetAddressGrowth)
		analytics.GET("/gas-analytics", s.handleGetGasAnalytics)
		analytics.GET("/validator-performance", s.handleGetValidatorPerformance)
	}
}

func (s *AnalyticsService) handleGetNetworkHealth(c *gin.Context) {
	health, err := s.GetNetworkHealth(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to compute network health",
		})
		return
	}

	c.JSON(http.StatusOK, health)
}

func (s *AnalyticsService) handleGetTransactionVolume(c *gin.Context) {
	period := c.DefaultQuery("period", "24h")

	data, err := s.GetTransactionVolumeChart(c.Request.Context(), period)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get transaction volume",
		})
		return
	}

	c.JSON(http.StatusOK, data)
}

func (s *AnalyticsService) handleGetDEXAnalytics(c *gin.Context) {
	analytics, err := s.GetDEXAnalytics(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to get DEX analytics",
		})
		return
	}

	c.JSON(http.StatusOK, analytics)
}

func (s *AnalyticsService) handleGetAddressGrowth(c *gin.Context) {
	period := c.DefaultQuery("period", "30d")

	data, err := s.GetAddressGrowth(c.Request.Context(), period)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("failed to get address growth: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, data)
}

// AddressGrowthData represents address growth analytics
type AddressGrowthData struct {
	TotalAddresses    int64                 `json:"total_addresses"`
	NewAddresses24h   int64                 `json:"new_addresses_24h"`
	ActiveAddresses24h int64                `json:"active_addresses_24h"`
	GrowthRate        float64               `json:"growth_rate"`
	Timeline          []AddressGrowthPoint  `json:"timeline"`
	Timestamp         time.Time             `json:"timestamp"`
}

// AddressGrowthPoint represents a data point in address growth timeline
type AddressGrowthPoint struct {
	Timestamp       time.Time `json:"timestamp"`
	TotalAddresses  int64     `json:"total_addresses"`
	NewAddresses    int64     `json:"new_addresses"`
	ActiveAddresses int64     `json:"active_addresses"`
}

// GetAddressGrowth retrieves address growth analytics
func (s *AnalyticsService) GetAddressGrowth(ctx context.Context, period string) (*AddressGrowthData, error) {
	timer := prometheus.NewTimer(analyticsQueryDuration.WithLabelValues("address_growth"))
	defer timer.ObserveDuration()
	analyticsQueriesTotal.WithLabelValues("address_growth").Inc()

	// Get total addresses (unique senders/receivers in transactions)
	totalQuery := `
		SELECT COUNT(DISTINCT address) as total
		FROM (
			SELECT sender as address FROM transactions
			UNION
			SELECT COALESCE(recipient, '') as address FROM transactions WHERE recipient != ''
		) as addresses
		WHERE address != ''
	`

	var total int64
	err := s.db.QueryRowContext(ctx, totalQuery).Scan(&total)
	if err != nil {
		return nil, fmt.Errorf("failed to get total addresses: %w", err)
	}

	// Get new addresses in last 24h
	newAddresses24hQuery := `
		SELECT COUNT(DISTINCT first_address) as new_count
		FROM (
			SELECT address as first_address, MIN(timestamp) as first_seen
			FROM (
				SELECT sender as address, timestamp FROM transactions
				UNION ALL
				SELECT COALESCE(recipient, '') as address, timestamp FROM transactions WHERE recipient != ''
			) as all_addresses
			WHERE address != ''
			GROUP BY address
			HAVING MIN(timestamp) > NOW() - INTERVAL '24 hours'
		) as new_addresses
	`

	var newAddresses24h int64
	err = s.db.QueryRowContext(ctx, newAddresses24hQuery).Scan(&newAddresses24h)
	if err != nil {
		newAddresses24h = 0 // Default to 0 if query fails
	}

	// Get active addresses in last 24h
	activeAddresses24hQuery := `
		SELECT COUNT(DISTINCT address) as active
		FROM (
			SELECT sender as address FROM transactions WHERE timestamp > NOW() - INTERVAL '24 hours'
			UNION
			SELECT COALESCE(recipient, '') as address FROM transactions
			WHERE recipient != '' AND timestamp > NOW() - INTERVAL '24 hours'
		) as addresses
		WHERE address != ''
	`

	var activeAddresses24h int64
	err = s.db.QueryRowContext(ctx, activeAddresses24hQuery).Scan(&activeAddresses24h)
	if err != nil {
		activeAddresses24h = 0
	}

	// Calculate growth rate (daily growth as percentage)
	growthRate := 0.0
	if total > 0 && newAddresses24h > 0 {
		growthRate = (float64(newAddresses24h) / float64(total)) * 100.0
	}

	// Get timeline data
	timeline, err := s.getAddressGrowthTimeline(ctx, period)
	if err != nil {
		timeline = []AddressGrowthPoint{} // Return empty timeline on error
	}

	return &AddressGrowthData{
		TotalAddresses:     total,
		NewAddresses24h:    newAddresses24h,
		ActiveAddresses24h: activeAddresses24h,
		GrowthRate:         growthRate,
		Timeline:           timeline,
		Timestamp:          time.Now(),
	}, nil
}

func (s *AnalyticsService) getAddressGrowthTimeline(ctx context.Context, period string) ([]AddressGrowthPoint, error) {
	// Determine interval and time range based on period
	interval := "1 day"
	timeRange := "30 days"

	if period == "7d" {
		interval = "1 day"
		timeRange = "7 days"
	} else if period == "90d" {
		interval = "1 week"
		timeRange = "90 days"
	}

	query := fmt.Sprintf(`
		WITH address_timeline AS (
			SELECT
				DATE_TRUNC('%s', timestamp) as bucket,
				address
			FROM (
				SELECT sender as address, timestamp FROM transactions
				UNION ALL
				SELECT COALESCE(recipient, '') as address, timestamp FROM transactions WHERE recipient != ''
			) as all_tx
			WHERE address != '' AND timestamp > NOW() - INTERVAL '%s'
		),
		new_addresses AS (
			SELECT
				DATE_TRUNC('%s', first_seen) as bucket,
				COUNT(*) as new_count
			FROM (
				SELECT address, MIN(timestamp) as first_seen
				FROM (
					SELECT sender as address, timestamp FROM transactions
					UNION ALL
					SELECT COALESCE(recipient, '') as address, timestamp FROM transactions WHERE recipient != ''
				) as all_addresses
				WHERE address != ''
				GROUP BY address
			) as first_appearances
			WHERE first_seen > NOW() - INTERVAL '%s'
			GROUP BY bucket
		)
		SELECT
			at.bucket,
			COUNT(DISTINCT at.address) as active_addresses,
			COALESCE(na.new_count, 0) as new_addresses
		FROM address_timeline at
		LEFT JOIN new_addresses na ON at.bucket = na.bucket
		GROUP BY at.bucket, na.new_count
		ORDER BY at.bucket
	`, interval, timeRange, interval, timeRange)

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get address timeline: %w", err)
	}
	defer rows.Close()

	var dataPoints []AddressGrowthPoint
	cumulativeTotal := int64(0)

	for rows.Next() {
		var point AddressGrowthPoint

		err := rows.Scan(&point.Timestamp, &point.ActiveAddresses, &point.NewAddresses)
		if err != nil {
			continue
		}

		cumulativeTotal += point.NewAddresses
		point.TotalAddresses = cumulativeTotal

		dataPoints = append(dataPoints, point)
	}

	return dataPoints, nil
}

func (s *AnalyticsService) handleGetGasAnalytics(c *gin.Context) {
	period := c.DefaultQuery("period", "24h")

	data, err := s.GetGasAnalytics(c.Request.Context(), period)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("failed to get gas analytics: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, data)
}

// GasAnalyticsData represents gas usage analytics
type GasAnalyticsData struct {
	AverageGasPrice   string            `json:"average_gas_price"`
	MedianGasPrice    string            `json:"median_gas_price"`
	MaxGasPrice       string            `json:"max_gas_price"`
	MinGasPrice       string            `json:"min_gas_price"`
	TotalGasUsed      int64             `json:"total_gas_used"`
	TotalGasLimit     int64             `json:"total_gas_limit"`
	GasUtilization    float64           `json:"gas_utilization"`
	Timeline          []GasDataPoint    `json:"timeline"`
	TopGasConsumers   []GasConsumer     `json:"top_gas_consumers"`
	Timestamp         time.Time         `json:"timestamp"`
}

// GasDataPoint represents a data point in gas analytics timeline
type GasDataPoint struct {
	Timestamp       time.Time `json:"timestamp"`
	AverageGasPrice string    `json:"average_gas_price"`
	TotalGasUsed    int64     `json:"total_gas_used"`
	Transactions    int64     `json:"transactions"`
}

// GasConsumer represents a top gas consumer
type GasConsumer struct {
	Address      string `json:"address"`
	TotalGasUsed int64  `json:"total_gas_used"`
	TxCount      int64  `json:"tx_count"`
	AvgGasPerTx  int64  `json:"avg_gas_per_tx"`
}

// GetGasAnalytics retrieves gas usage analytics
func (s *AnalyticsService) GetGasAnalytics(ctx context.Context, period string) (*GasAnalyticsData, error) {
	timer := prometheus.NewTimer(analyticsQueryDuration.WithLabelValues("gas_analytics"))
	defer timer.ObserveDuration()
	analyticsQueriesTotal.WithLabelValues("gas_analytics").Inc()

	// Determine time range based on period
	timeRange := "24 hours"
	if period == "7d" {
		timeRange = "7 days"
	} else if period == "30d" {
		timeRange = "30 days"
	}

	// Get aggregate gas statistics
	aggregateQuery := fmt.Sprintf(`
		SELECT
			AVG(CAST(gas_wanted AS NUMERIC)) as avg_gas_price,
			PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY CAST(gas_wanted AS NUMERIC)) as median_gas_price,
			MAX(CAST(gas_wanted AS NUMERIC)) as max_gas_price,
			MIN(CAST(gas_wanted AS NUMERIC)) as min_gas_price,
			SUM(CAST(gas_used AS BIGINT)) as total_gas_used,
			SUM(CAST(gas_wanted AS BIGINT)) as total_gas_limit
		FROM transactions
		WHERE timestamp > NOW() - INTERVAL '%s'
	`, timeRange)

	var avgGasPrice, medianGasPrice, maxGasPrice, minGasPrice float64
	var totalGasUsed, totalGasLimit int64

	err := s.db.QueryRowContext(ctx, aggregateQuery).Scan(
		&avgGasPrice, &medianGasPrice, &maxGasPrice, &minGasPrice,
		&totalGasUsed, &totalGasLimit,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get gas aggregates: %w", err)
	}

	// Calculate gas utilization
	gasUtilization := 0.0
	if totalGasLimit > 0 {
		gasUtilization = (float64(totalGasUsed) / float64(totalGasLimit)) * 100.0
	}

	// Get timeline data
	timeline, err := s.getGasTimeline(ctx, period)
	if err != nil {
		timeline = []GasDataPoint{}
	}

	// Get top gas consumers
	topConsumers, err := s.getTopGasConsumers(ctx, timeRange)
	if err != nil {
		topConsumers = []GasConsumer{}
	}

	return &GasAnalyticsData{
		AverageGasPrice:  fmt.Sprintf("%.0f", avgGasPrice),
		MedianGasPrice:   fmt.Sprintf("%.0f", medianGasPrice),
		MaxGasPrice:      fmt.Sprintf("%.0f", maxGasPrice),
		MinGasPrice:      fmt.Sprintf("%.0f", minGasPrice),
		TotalGasUsed:     totalGasUsed,
		TotalGasLimit:    totalGasLimit,
		GasUtilization:   gasUtilization,
		Timeline:         timeline,
		TopGasConsumers:  topConsumers,
		Timestamp:        time.Now(),
	}, nil
}

func (s *AnalyticsService) getGasTimeline(ctx context.Context, period string) ([]GasDataPoint, error) {
	interval := "1 hour"
	timeRange := "24 hours"

	if period == "7d" {
		interval = "6 hours"
		timeRange = "7 days"
	} else if period == "30d" {
		interval = "1 day"
		timeRange = "30 days"
	}

	query := fmt.Sprintf(`
		SELECT
			DATE_TRUNC('%s', timestamp) as bucket,
			AVG(CAST(gas_wanted AS NUMERIC)) as avg_gas_price,
			SUM(CAST(gas_used AS BIGINT)) as total_gas_used,
			COUNT(*) as tx_count
		FROM transactions
		WHERE timestamp > NOW() - INTERVAL '%s'
		GROUP BY bucket
		ORDER BY bucket
	`, interval, timeRange)

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get gas timeline: %w", err)
	}
	defer rows.Close()

	var dataPoints []GasDataPoint
	for rows.Next() {
		var point GasDataPoint
		var avgGasPrice float64

		err := rows.Scan(&point.Timestamp, &avgGasPrice, &point.TotalGasUsed, &point.Transactions)
		if err != nil {
			continue
		}

		point.AverageGasPrice = fmt.Sprintf("%.0f", avgGasPrice)
		dataPoints = append(dataPoints, point)
	}

	return dataPoints, nil
}

func (s *AnalyticsService) getTopGasConsumers(ctx context.Context, timeRange string) ([]GasConsumer, error) {
	query := fmt.Sprintf(`
		SELECT
			sender,
			SUM(CAST(gas_used AS BIGINT)) as total_gas_used,
			COUNT(*) as tx_count,
			AVG(CAST(gas_used AS NUMERIC)) as avg_gas_per_tx
		FROM transactions
		WHERE timestamp > NOW() - INTERVAL '%s'
		GROUP BY sender
		ORDER BY total_gas_used DESC
		LIMIT 10
	`, timeRange)

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get top gas consumers: %w", err)
	}
	defer rows.Close()

	var consumers []GasConsumer
	for rows.Next() {
		var consumer GasConsumer
		var avgGas float64

		err := rows.Scan(&consumer.Address, &consumer.TotalGasUsed, &consumer.TxCount, &avgGas)
		if err != nil {
			continue
		}

		consumer.AvgGasPerTx = int64(avgGas)
		consumers = append(consumers, consumer)
	}

	return consumers, nil
}

func (s *AnalyticsService) handleGetValidatorPerformance(c *gin.Context) {
	period := c.DefaultQuery("period", "24h")

	data, err := s.GetValidatorPerformance(c.Request.Context(), period)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": fmt.Sprintf("failed to get validator performance: %v", err),
		})
		return
	}

	c.JSON(http.StatusOK, data)
}

// ValidatorPerformanceData represents validator performance analytics
type ValidatorPerformanceData struct {
	TotalValidators    int                      `json:"total_validators"`
	ActiveValidators   int                      `json:"active_validators"`
	AverageUptime      float64                  `json:"average_uptime"`
	Validators         []ValidatorPerformance   `json:"validators"`
	Timeline           []ValidatorTimelinePoint `json:"timeline"`
	Timestamp          time.Time                `json:"timestamp"`
}

// ValidatorPerformance represents individual validator performance
type ValidatorPerformance struct {
	Address         string  `json:"address"`
	Moniker         string  `json:"moniker"`
	VotingPower     int64   `json:"voting_power"`
	BlocksSigned    int64   `json:"blocks_signed"`
	BlocksProposed  int64   `json:"blocks_proposed"`
	BlocksMissed    int64   `json:"blocks_missed"`
	Uptime          float64 `json:"uptime"`
	Commission      string  `json:"commission"`
	Status          string  `json:"status"`
}

// ValidatorTimelinePoint represents a data point in validator performance timeline
type ValidatorTimelinePoint struct {
	Timestamp        time.Time `json:"timestamp"`
	ActiveValidators int       `json:"active_validators"`
	AverageUptime    float64   `json:"average_uptime"`
	BlocksProduced   int64     `json:"blocks_produced"`
}

// GetValidatorPerformance retrieves validator performance analytics
func (s *AnalyticsService) GetValidatorPerformance(ctx context.Context, period string) (*ValidatorPerformanceData, error) {
	timer := prometheus.NewTimer(analyticsQueryDuration.WithLabelValues("validator_performance"))
	defer timer.ObserveDuration()
	analyticsQueriesTotal.WithLabelValues("validator_performance").Inc()

	// Determine time range based on period
	timeRange := "24 hours"
	if period == "7d" {
		timeRange = "7 days"
	} else if period == "30d" {
		timeRange = "30 days"
	}

	// Get validator metrics from blocks table
	// Count blocks proposed by each validator (proposer)
	validatorQuery := fmt.Sprintf(`
		SELECT
			proposer_address,
			COUNT(*) as blocks_proposed,
			MAX(height) - MIN(height) + 1 as total_blocks
		FROM blocks
		WHERE timestamp > NOW() - INTERVAL '%s'
		GROUP BY proposer_address
		ORDER BY blocks_proposed DESC
	`, timeRange)

	rows, err := s.db.QueryContext(ctx, validatorQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to get validator metrics: %w", err)
	}
	defer rows.Close()

	var validators []ValidatorPerformance
	totalValidators := 0
	activeValidators := 0
	totalUptime := 0.0

	for rows.Next() {
		var val ValidatorPerformance
		var totalBlocks int64

		err := rows.Scan(&val.Address, &val.BlocksProposed, &totalBlocks)
		if err != nil {
			continue
		}

		// Calculate uptime as blocks proposed / total blocks in period
		if totalBlocks > 0 {
			val.Uptime = (float64(val.BlocksProposed) / float64(totalBlocks)) * 100.0
		}

		val.BlocksMissed = totalBlocks - val.BlocksProposed
		val.BlocksSigned = val.BlocksProposed // Simplified assumption
		val.Status = "active"
		val.VotingPower = 0 // Would need to query staking module for real value
		val.Commission = "10.00%" // Would need to query staking module for real value
		val.Moniker = val.Address // Would need validator registry for real moniker

		validators = append(validators, val)
		totalValidators++
		if val.Uptime > 90.0 {
			activeValidators++
		}
		totalUptime += val.Uptime
	}

	// Calculate average uptime
	averageUptime := 0.0
	if totalValidators > 0 {
		averageUptime = totalUptime / float64(totalValidators)
	}

	// Get timeline data
	timeline, err := s.getValidatorTimeline(ctx, period)
	if err != nil {
		timeline = []ValidatorTimelinePoint{}
	}

	return &ValidatorPerformanceData{
		TotalValidators:  totalValidators,
		ActiveValidators: activeValidators,
		AverageUptime:    averageUptime,
		Validators:       validators,
		Timeline:         timeline,
		Timestamp:        time.Now(),
	}, nil
}

func (s *AnalyticsService) getValidatorTimeline(ctx context.Context, period string) ([]ValidatorTimelinePoint, error) {
	interval := "1 hour"
	timeRange := "24 hours"

	if period == "7d" {
		interval = "6 hours"
		timeRange = "7 days"
	} else if period == "30d" {
		interval = "1 day"
		timeRange = "30 days"
	}

	query := fmt.Sprintf(`
		SELECT
			DATE_TRUNC('%s', timestamp) as bucket,
			COUNT(DISTINCT proposer_address) as active_validators,
			COUNT(*) as blocks_produced
		FROM blocks
		WHERE timestamp > NOW() - INTERVAL '%s'
		GROUP BY bucket
		ORDER BY bucket
	`, interval, timeRange)

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get validator timeline: %w", err)
	}
	defer rows.Close()

	var dataPoints []ValidatorTimelinePoint
	for rows.Next() {
		var point ValidatorTimelinePoint

		err := rows.Scan(&point.Timestamp, &point.ActiveValidators, &point.BlocksProduced)
		if err != nil {
			continue
		}

		// Estimate average uptime (simplified)
		point.AverageUptime = 95.0 // Would need more data for real calculation

		dataPoints = append(dataPoints, point)
	}

	return dataPoints, nil
}

// Helper functions

func parseFloat(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}
