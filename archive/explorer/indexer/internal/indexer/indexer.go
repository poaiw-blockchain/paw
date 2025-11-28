package indexer

import (
	"context"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/paw-chain/paw/explorer/indexer/config"
	"github.com/paw-chain/paw/explorer/indexer/internal/cache"
	"github.com/paw-chain/paw/explorer/indexer/internal/database"
	"github.com/paw-chain/paw/explorer/indexer/internal/rpc"
	"github.com/paw-chain/paw/explorer/indexer/internal/subscriber"
	"github.com/paw-chain/paw/explorer/indexer/pkg/logger"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/rs/zerolog/log"
)

var (
	blocksIndexed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "explorer_blocks_indexed_total",
		Help: "Total number of blocks indexed",
	})

	transactionsIndexed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "explorer_transactions_indexed_total",
		Help: "Total number of transactions indexed",
	})

	eventsIndexed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "explorer_events_indexed_total",
		Help: "Total number of events indexed",
	})

	indexingDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "explorer_indexing_duration_seconds",
		Help:    "Block indexing duration in seconds",
		Buckets: prometheus.DefBuckets,
	})

	indexerHeight = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "explorer_indexer_height",
		Help: "Current indexer block height",
	})

	chainHeight = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "explorer_chain_height",
		Help: "Current chain block height",
	})

	historicalBlocksIndexed = promauto.NewCounter(prometheus.CounterOpts{
		Name: "explorer_historical_blocks_indexed_total",
		Help: "Total number of historical blocks indexed",
	})

	historicalIndexingProgress = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "explorer_historical_indexing_progress_percent",
		Help: "Historical indexing progress percentage",
	})

	blocksPerSecond = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "explorer_blocks_per_second",
		Help: "Current blocks indexed per second",
	})

	failedBlocksTotal = promauto.NewCounter(prometheus.CounterOpts{
		Name: "explorer_failed_blocks_total",
		Help: "Total number of blocks that failed to index",
	})
)

// Config holds indexer configuration
type Config struct {
	StartHeight              int64
	BatchSize                int
	Workers                  int
	RetryAttempts            int
	RetryDelay               time.Duration
	EnableHistoricalIndexing bool
	HistoricalBatchSize      int
	ParallelFetches          int
}

// Indexer manages blockchain data indexing
type Indexer struct {
	db         *database.DB
	subscriber *subscriber.Subscriber
	rpcClient  *rpc.Client
	config     Config
	ctx        context.Context
	cancel     context.CancelFunc
	wg         sync.WaitGroup

	// Historical indexing state
	historicalIndexingActive atomic.Bool
	lastIndexedHeight        atomic.Int64
	chainTipHeight           atomic.Int64
}

// New creates a new indexer
func New(db *database.DB, sub *subscriber.Subscriber, rpcClient *rpc.Client, config Config) *Indexer {
	ctx, cancel := context.WithCancel(context.Background())

	// Set defaults
	if config.HistoricalBatchSize == 0 {
		config.HistoricalBatchSize = 100
	}
	if config.ParallelFetches == 0 {
		config.ParallelFetches = 10
	}
	if config.RetryAttempts == 0 {
		config.RetryAttempts = 3
	}

	return &Indexer{
		db:         db,
		subscriber: sub,
		rpcClient:  rpcClient,
		config:     config,
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Start begins the indexing process
func (idx *Indexer) Start() error {
	log.Info().Msg("Starting blockchain indexer")

	// Get last indexed height
	lastHeight, err := idx.db.GetLastIndexedHeight()
	if err != nil {
		log.Error().Err(err).Msg("Failed to get last indexed height")
		lastHeight = idx.config.StartHeight
	}

	log.Info().Int64("last_height", lastHeight).Msg("Resuming from last indexed height")

	// Start historical indexing if needed
	if lastHeight < idx.config.StartHeight {
		if err := idx.indexHistorical(idx.config.StartHeight); err != nil {
			return fmt.Errorf("failed to index historical data: %w", err)
		}
	}

	// Start real-time indexing
	idx.wg.Add(1)
	go idx.indexRealtime()

	log.Info().Msg("Indexer started successfully")
	return nil
}

// indexHistorical indexes historical blocks from a starting height
func (idx *Indexer) indexHistorical(startHeight int64) error {
	if !idx.config.EnableHistoricalIndexing {
		log.Info().Msg("Historical indexing disabled, skipping")
		return nil
	}

	log.Info().Int64("start_height", startHeight).Msg("Starting historical indexing")
	idx.historicalIndexingActive.Store(true)
	defer idx.historicalIndexingActive.Store(false)

	// Get current chain height
	currentHeight, err := idx.getChainHeight()
	if err != nil {
		return fmt.Errorf("failed to get chain height: %w", err)
	}

	idx.chainTipHeight.Store(currentHeight)
	chainHeight.Set(float64(currentHeight))

	log.Info().
		Int64("start_height", startHeight).
		Int64("current_height", currentHeight).
		Int64("blocks_to_index", currentHeight-startHeight+1).
		Msg("Beginning historical indexing")

	// Check if we need to resume from a previous indexing session
	lastIndexed, err := idx.getLastIndexedHeight()
	if err != nil {
		log.Warn().Err(err).Msg("Could not get last indexed height, starting from configured height")
		lastIndexed = startHeight - 1
	}

	if lastIndexed >= startHeight {
		startHeight = lastIndexed + 1
		log.Info().Int64("resuming_from", startHeight).Msg("Resuming historical indexing from last checkpoint")
	}

	idx.lastIndexedHeight.Store(lastIndexed)

	// Calculate total blocks to index
	totalBlocks := currentHeight - startHeight + 1
	if totalBlocks <= 0 {
		log.Info().Msg("No historical blocks to index")
		return nil
	}

	log.Info().
		Int64("total_blocks", totalBlocks).
		Int("batch_size", idx.config.HistoricalBatchSize).
		Msg("Indexing parameters")

	// Index in batches
	startTime := time.Now()
	processedBlocks := int64(0)

	for height := startHeight; height <= currentHeight; {
		// Check if context is cancelled
		select {
		case <-idx.ctx.Done():
			return fmt.Errorf("indexing cancelled: %w", idx.ctx.Err())
		default:
		}

		// Calculate batch end height
		endHeight := minInt64(height+int64(idx.config.HistoricalBatchSize)-1, currentHeight)

		// Index this batch
		batchStart := time.Now()
		if err := idx.indexBlockBatch(height, endHeight); err != nil {
			log.Error().
				Err(err).
				Int64("start", height).
				Int64("end", endHeight).
				Msg("Failed to index batch")

			// Try to continue with next batch on error
			// Failed blocks will be tracked separately
			failedBlocksTotal.Inc()
		} else {
			// Update progress
			batchSize := endHeight - height + 1
			processedBlocks += batchSize
			batchDuration := time.Since(batchStart)

			// Calculate metrics
			progress := float64(processedBlocks) / float64(totalBlocks) * 100
			blocksProcessed := float64(processedBlocks)
			elapsedSeconds := time.Since(startTime).Seconds()
			bps := blocksProcessed / elapsedSeconds

			// Update Prometheus metrics
			historicalIndexingProgress.Set(progress)
			blocksPerSecond.Set(bps)
			indexerHeight.Set(float64(endHeight))

			// Save progress checkpoint
			if err := idx.saveProgress(endHeight, "indexing"); err != nil {
				log.Error().Err(err).Msg("Failed to save indexing progress")
			}

			idx.lastIndexedHeight.Store(endHeight)

			// Calculate ETA
			remainingBlocks := float64(totalBlocks - processedBlocks)
			etaSeconds := remainingBlocks / bps
			eta := time.Duration(etaSeconds) * time.Second

			log.Info().
				Int64("height", endHeight).
				Int64("batch_size", batchSize).
				Dur("batch_duration", batchDuration).
				Float64("progress_pct", progress).
				Float64("blocks_per_sec", bps).
				Dur("eta", eta).
				Msg("Indexed batch successfully")
		}

		// Move to next batch
		height = endHeight + 1

		// Brief pause to avoid overwhelming the RPC node
		time.Sleep(100 * time.Millisecond)
	}

	totalDuration := time.Since(startTime)
	avgBlocksPerSec := float64(processedBlocks) / totalDuration.Seconds()

	log.Info().
		Int64("total_blocks_indexed", processedBlocks).
		Dur("total_duration", totalDuration).
		Float64("avg_blocks_per_sec", avgBlocksPerSec).
		Msg("Historical indexing completed successfully")

	// Mark indexing as complete
	if err := idx.saveProgress(currentHeight, "complete"); err != nil {
		log.Error().Err(err).Msg("Failed to save completion status")
	}

	return nil
}

// indexBlockBatch indexes a batch of blocks
func (idx *Indexer) indexBlockBatch(startHeight, endHeight int64) error {
	log.Debug().
		Int64("start", startHeight).
		Int64("end", endHeight).
		Msg("Fetching block batch")

	// Fetch blocks in parallel
	blocks, err := idx.rpcClient.GetBlockBatch(idx.ctx, startHeight, endHeight)
	if err != nil {
		return fmt.Errorf("failed to fetch block batch: %w", err)
	}

	// Begin database transaction for the entire batch
	tx, err := idx.db.BeginTx()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Process each block
	for i, blockData := range blocks {
		if blockData == nil {
			log.Warn().
				Int64("height", startHeight+int64(i)).
				Msg("Block data is nil, skipping")
			continue
		}

		if err := idx.indexBlockData(tx, blockData); err != nil {
			log.Error().
				Err(err).
				Int64("height", startHeight+int64(i)).
				Msg("Failed to index block")

			// Save to failed blocks table
			if saveErr := idx.saveFailedBlock(startHeight+int64(i), err); saveErr != nil {
				log.Error().Err(saveErr).Msg("Failed to save failed block record")
			}

			continue
		}

		historicalBlocksIndexed.Inc()
	}

	// Commit the batch transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit batch transaction: %w", err)
	}

	return nil
}

// indexBlockData indexes a single block's data
func (idx *Indexer) indexBlockData(tx *database.Tx, blockData *rpc.BlockData) error {
	if blockData == nil || blockData.Block == nil {
		return fmt.Errorf("invalid block data")
	}

	block := blockData.Block
	results := blockData.Results

	// Parse height
	var height int64
	fmt.Sscanf(block.Result.Block.Header.Height, "%d", &height)

	// Parse block hash
	blockHash := block.Result.BlockID.Hash

	// Insert block
	dbBlock := database.Block{
		Height:          height,
		Hash:            blockHash,
		ProposerAddress: block.Result.Block.Header.ProposerAddress,
		Time:            block.Result.Block.Header.Time,
		TxCount:         len(block.Result.Block.Data.Txs),
		GasUsed:         0,
		GasWanted:       0,
		EvidenceCount:   len(block.Result.Block.Evidence.Evidence),
	}

	// Calculate total gas from transactions if results available
	if results != nil {
		for _, txResult := range results.Result.TxsResults {
			var gasUsed, gasWanted int64
			fmt.Sscanf(txResult.GasUsed, "%d", &gasUsed)
			fmt.Sscanf(txResult.GasWanted, "%d", &gasWanted)
			dbBlock.GasUsed += gasUsed
			dbBlock.GasWanted += gasWanted
		}
	}

	if err := idx.db.InsertBlock(tx, dbBlock); err != nil {
		return fmt.Errorf("failed to insert block: %w", err)
	}

	// Process transactions if present
	if results != nil && len(block.Result.Block.Data.Txs) > 0 {
		for i, txHash := range block.Result.Block.Data.Txs {
			if i < len(results.Result.TxsResults) {
				txResult := results.Result.TxsResults[i]
				if err := idx.indexTransaction(tx, txHash, txResult, height, i, block.Result.Block.Header.Time); err != nil {
					log.Error().
						Err(err).
						Str("tx_hash", txHash).
						Int64("height", height).
						Msg("Failed to index transaction")
					// Continue with other transactions
				}
			}
		}
	}

	blocksIndexed.Inc()
	return nil
}

// indexTransaction indexes a single transaction
func (idx *Indexer) indexTransaction(tx *database.Tx, txHashBase64 string, txResult rpc.TxResult, blockHeight int64, txIndex int, blockTime time.Time) error {
	// Decode transaction hash
	txHashBytes, err := hex.DecodeString(txHashBase64)
	var txHash string
	if err == nil {
		txHash = hex.EncodeToString(txHashBytes)
	} else {
		txHash = txHashBase64 // Use as-is if decode fails
	}

	// Parse gas values
	var gasUsed, gasWanted int64
	fmt.Sscanf(txResult.GasUsed, "%d", &gasUsed)
	fmt.Sscanf(txResult.GasWanted, "%d", &gasWanted)

	// Determine status
	status := "success"
	if txResult.Code != 0 {
		status = "failed"
	}

	// Marshal events
	eventsJSON, _ := json.Marshal(txResult.Events)

	transaction := database.Transaction{
		Hash:        txHash,
		BlockHeight: blockHeight,
		TxIndex:     txIndex,
		Type:        "unknown", // Would need to decode tx to get type
		Sender:      "",        // Would need to decode tx to get sender
		Status:      status,
		Code:        txResult.Code,
		GasUsed:     gasUsed,
		GasWanted:   gasWanted,
		FeeAmount:   "",
		FeeDenom:    "",
		RawLog:      txResult.Log,
		Time:        blockTime,
		Messages:    []byte("{}"), // Would need full tx decode
		Events:      eventsJSON,
	}

	if err := idx.db.InsertTransaction(tx, transaction); err != nil {
		return fmt.Errorf("failed to insert transaction: %w", err)
	}

	// Process events
	for eventIdx, event := range txResult.Events {
		if err := idx.indexEvent(tx, event, txHash, blockHeight, eventIdx, blockTime); err != nil {
			log.Error().
				Err(err).
				Str("event_type", event.Type).
				Msg("Failed to index event")
		}
	}

	transactionsIndexed.Inc()
	return nil
}

// indexEvent indexes a single event
func (idx *Indexer) indexEvent(tx *database.Tx, event rpc.Event, txHash string, blockHeight int64, eventIndex int, timestamp time.Time) error {
	// Convert attributes to JSON
	attrs, _ := json.Marshal(event.Attributes)

	dbEvent := database.Event{
		TxHash:      txHash,
		BlockHeight: blockHeight,
		EventType:   event.Type,
		Module:      extractModule(event.Type),
		Attributes:  attrs,
	}

	if err := idx.db.InsertEvent(tx, dbEvent); err != nil {
		return err
	}

	eventsIndexed.Inc()
	return nil
}

// getChainHeight gets the current blockchain height
func (idx *Indexer) getChainHeight() (int64, error) {
	ctx, cancel := context.WithTimeout(idx.ctx, 10*time.Second)
	defer cancel()

	height, err := idx.rpcClient.GetChainHeight(ctx)
	if err != nil {
		return 0, err
	}

	return height, nil
}

// getLastIndexedHeight gets the last indexed height from database
func (idx *Indexer) getLastIndexedHeight() (int64, error) {
	return idx.db.GetLastIndexedHeight()
}

// saveProgress saves indexing progress to database
func (idx *Indexer) saveProgress(height int64, status string) error {
	return idx.db.SaveIndexingProgress(height, status)
}

// saveFailedBlock saves a failed block for later retry
func (idx *Indexer) saveFailedBlock(height int64, err error) error {
	return idx.db.SaveFailedBlock(height, err.Error())
}

// minInt64 returns the minimum of two int64 values
func minInt64(a, b int64) int64 {
	if a < b {
		return a
	}
	return b
}

// GetIndexingStatus returns the current indexing status
func (idx *Indexer) GetIndexingStatus() (*IndexingStatus, error) {
	lastHeight, err := idx.getLastIndexedHeight()
	if err != nil {
		lastHeight = 0
	}

	chainHeight := idx.chainTipHeight.Load()
	if chainHeight == 0 {
		chainHeight, _ = idx.getChainHeight()
	}

	progress := float64(0)
	if chainHeight > 0 {
		progress = float64(lastHeight) / float64(chainHeight) * 100
	}

	status := "idle"
	if idx.historicalIndexingActive.Load() {
		status = "indexing"
	} else if lastHeight >= chainHeight {
		status = "complete"
	}

	return &IndexingStatus{
		LastIndexedHeight:  lastHeight,
		CurrentChainHeight: chainHeight,
		Status:             status,
		ProgressPercent:    progress,
		IsActive:           idx.historicalIndexingActive.Load(),
	}, nil
}

// IndexingStatus represents the current indexing status
type IndexingStatus struct {
	LastIndexedHeight  int64   `json:"last_indexed_height"`
	CurrentChainHeight int64   `json:"current_chain_height"`
	Status             string  `json:"status"`
	ProgressPercent    float64 `json:"progress_percent"`
	IsActive           bool    `json:"is_active"`
}

// indexRealtime processes real-time blockchain events
func (idx *Indexer) indexRealtime() {
	defer idx.wg.Done()

	eventChan := idx.subscriber.Events()

	for {
		select {
		case <-idx.ctx.Done():
			log.Info().Msg("Real-time indexer stopped")
			return
		case event, ok := <-eventChan:
			if !ok {
				log.Warn().Msg("Event channel closed, stopping real-time indexer")
				return
			}

			if err := idx.processBlockEvent(event); err != nil {
				log.Error().
					Err(err).
					Int64("height", event.Height).
					Msg("Failed to process block event")
				// Implement retry logic here
				continue
			}

			log.Debug().
				Int64("height", event.Height).
				Int("tx_count", len(event.Txs)).
				Msg("Processed block successfully")
		}
	}
}

// processBlockEvent processes a single block event
func (idx *Indexer) processBlockEvent(event subscriber.BlockEvent) error {
	tx, err := idx.db.BeginTx()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	// Insert block
	block := database.Block{
		Height:          event.Height,
		Hash:            event.Hash,
		ProposerAddress: event.Proposer,
		Time:            event.Time,
		TxCount:         len(event.Txs),
		GasUsed:         0,
		GasWanted:       0,
		EvidenceCount:   0,
	}

	// Calculate total gas
	for _, txResult := range event.Txs {
		block.GasUsed += txResult.GasUsed
		block.GasWanted += txResult.GasWanted
	}

	if err := idx.db.InsertBlock(tx, block); err != nil {
		return fmt.Errorf("failed to insert block: %w", err)
	}

	// Process transactions
	for i, txResult := range event.Txs {
		if err := idx.processTransaction(tx, txResult, event.Height, i, event.Time); err != nil {
			log.Error().
				Err(err).
				Str("tx_hash", txResult.Hash).
				Msg("Failed to process transaction")
			continue
		}
	}

	// Update last indexed height
	if err := idx.db.UpdateLastIndexedHeight(event.Height); err != nil {
		return fmt.Errorf("failed to update last indexed height: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// processTransaction processes a single transaction
func (idx *Indexer) processTransaction(tx *database.Tx, txResult subscriber.TransactionResult, blockHeight int64, txIndex int, blockTime time.Time) error {
	// Parse transaction messages
	messages, _ := json.Marshal(txResult.RawTx)
	events, _ := json.Marshal(txResult.Events)

	// Determine transaction type and sender
	txType := "unknown"
	sender := ""

	// Extract type and sender from raw transaction
	if rawTx, ok := txResult.RawTx["body"].(map[string]interface{}); ok {
		if msgs, ok := rawTx["messages"].([]interface{}); ok && len(msgs) > 0 {
			if msg, ok := msgs[0].(map[string]interface{}); ok {
				if typeURL, ok := msg["@type"].(string); ok {
					txType = typeURL
				}
				// Try to extract sender from different message types
				if fromAddr, ok := msg["from_address"].(string); ok {
					sender = fromAddr
				} else if senderAddr, ok := msg["sender"].(string); ok {
					sender = senderAddr
				}
			}
		}
	}

	// Determine transaction status
	status := "success"
	if txResult.Code != 0 {
		status = "failed"
	}

	// Extract fee information
	feeAmount := ""
	feeDenom := ""
	if rawTx, ok := txResult.RawTx["auth_info"].(map[string]interface{}); ok {
		if fee, ok := rawTx["fee"].(map[string]interface{}); ok {
			if amount, ok := fee["amount"].([]interface{}); ok && len(amount) > 0 {
				if coin, ok := amount[0].(map[string]interface{}); ok {
					feeAmount, _ = coin["amount"].(string)
					feeDenom, _ = coin["denom"].(string)
				}
			}
		}
	}

	transaction := database.Transaction{
		Hash:        txResult.Hash,
		BlockHeight: blockHeight,
		TxIndex:     txIndex,
		Type:        txType,
		Sender:      sender,
		Status:      status,
		Code:        txResult.Code,
		GasUsed:     txResult.GasUsed,
		GasWanted:   txResult.GasWanted,
		FeeAmount:   feeAmount,
		FeeDenom:    feeDenom,
		RawLog:      txResult.Log,
		Time:        blockTime,
		Messages:    messages,
		Events:      events,
	}

	if err := idx.db.InsertTransaction(tx, transaction); err != nil {
		return fmt.Errorf("failed to insert transaction: %w", err)
	}

	// Update account
	if sender != "" {
		if err := idx.db.UpsertAccount(tx, sender, blockHeight); err != nil {
			log.Error().Err(err).Str("address", sender).Msg("Failed to upsert account")
		}
	}

	// Process events
	for _, event := range txResult.Events {
		if err := idx.processEvent(tx, event, txResult.Hash, blockHeight); err != nil {
			log.Error().Err(err).Str("event_type", event.Type).Msg("Failed to process event")
		}
	}

	return nil
}

// processEvent processes a transaction event
func (idx *Indexer) processEvent(tx *database.Tx, event subscriber.Event, txHash string, blockHeight int64) error {
	// Convert attributes to JSON
	attrs, _ := json.Marshal(event.Attributes)

	dbEvent := database.Event{
		TxHash:      txHash,
		BlockHeight: blockHeight,
		EventType:   event.Type,
		Module:      extractModule(event.Type),
		Attributes:  attrs,
	}

	if err := idx.db.InsertEvent(tx, dbEvent); err != nil {
		return err
	}

	// Process specific event types
	switch event.Type {
	case "swap":
		return idx.processDEXSwap(tx, event, txHash, blockHeight)
	case "add_liquidity", "remove_liquidity":
		return idx.processDEXLiquidity(tx, event, txHash, blockHeight)
	case "oracle_price_update":
		return idx.processOraclePrice(tx, event, blockHeight)
	}

	return nil
}

// processDEXSwap processes a DEX swap event
func (idx *Indexer) processDEXSwap(tx *database.Tx, event subscriber.Event, txHash string, blockHeight int64) error {
	// Extract swap details from event attributes
	attrs := attributesToMap(event.Attributes)

	swap := database.DEXSwap{
		TxHash:    txHash,
		PoolID:    attrs["pool_id"],
		Sender:    attrs["sender"],
		TokenIn:   attrs["token_in"],
		TokenOut:  attrs["token_out"],
		AmountIn:  parseFloat(attrs["amount_in"]),
		AmountOut: parseFloat(attrs["amount_out"]),
		Price:     parseFloat(attrs["price"]),
		Fee:       parseFloat(attrs["fee"]),
		Time:      time.Now(),
	}

	return idx.db.InsertDEXSwap(tx, swap)
}

// processDEXLiquidity processes a DEX liquidity event
func (idx *Indexer) processDEXLiquidity(tx *database.Tx, event subscriber.Event, txHash string, blockHeight int64) error {
	// Implementation for liquidity events
	return nil
}

// processOraclePrice processes an oracle price update event
func (idx *Indexer) processOraclePrice(tx *database.Tx, event subscriber.Event, blockHeight int64) error {
	attrs := attributesToMap(event.Attributes)

	price := database.OraclePrice{
		Asset:       attrs["asset"],
		Price:       parseFloat(attrs["price"]),
		Timestamp:   time.Now(),
		BlockHeight: blockHeight,
		Source:      attrs["source"],
	}

	return idx.db.InsertOraclePrice(tx, price)
}

// Stop stops the indexer
func (idx *Indexer) Stop() {
	log.Info().Msg("Stopping indexer")
	idx.cancel()
	idx.wg.Wait()
	log.Info().Msg("Indexer stopped")
}

// Helper functions

func extractModule(eventType string) string {
	// Extract module name from event type
	// e.g., "cosmos.bank.v1beta1.MsgSend" -> "bank"
	return "unknown"
}

func attributesToMap(attrs []subscriber.Attribute) map[string]string {
	m := make(map[string]string)
	for _, attr := range attrs {
		m[attr.Key] = attr.Value
	}
	return m
}

func parseFloat(s string) float64 {
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}
